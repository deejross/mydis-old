// Copyright 2017 Ross Peoples
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mydis

import (
	"errors"
	"io/ioutil"
	"log"
	"sync"

	"strings"

	"strconv"

	"crypto/tls"

	"os"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/metadata"
)

var (
	// ErrKeyNotFound is the client side representation of etcdserver.ErrKeyNotFound.
	ErrKeyNotFound = errors.New("Key not found")
)

var knownErrors = map[string]error{
	"etcdserver: key not found":      ErrKeyNotFound,
	"Key is locked":                  ErrKeyLocked,
	"List is empty":                  ErrListEmpty,
	"etcdserver: Index out of range": ErrListIndexOutOfRange,
	"Hash field does not exist":      ErrHashFieldNotFound,
	"Type mismatch":                  ErrTypeMismatch,
}

func normalizeError(err error) error {
	if err == nil {
		return nil
	}
	errDesc := grpc.ErrorDesc(err)
	if newErr, ok := knownErrors[errDesc]; ok {
		return newErr
	}
	return err
}

// ClientConfig object.
type ClientConfig struct {
	Address      string
	TLS          *tls.Config
	AutoTLS      bool
	autoTLSDir   string
	creds        *credentials.TransportCredentials
	transportOpt grpc.DialOption
}

// NewClientConfig returns a new ClientConfig with default values.
func NewClientConfig(address string) ClientConfig {
	return ClientConfig{
		Address: address,
		AutoTLS: true,
	}
}

// Client object.
type Client struct {
	authToken string
	config    ClientConfig
	ctx       context.Context
	closeCh   chan struct{}
	closing   bool
	reqCh     chan *WatchRequest
	resCh     chan struct{}
	socket    *grpc.ClientConn
	stream    Mydis_WatchClient
	mc        MydisClient
	lock      sync.RWMutex
	newID     int64
	watching  map[string]struct{}
	watchers  map[int64]chan *Event
}

// NewClient returns a new Client object.
func NewClient(config ClientConfig) (*Client, error) {
	if config.AutoTLS {
		dir, err := ioutil.TempDir("", "mydis-client")
		if err != nil {
			return nil, err
		}

		config.autoTLSDir = dir
		tlsInfo, err := generateCert(dir)
		if err != nil {
			os.RemoveAll(dir)
			return nil, err
		}
		tlsConfig, err := tlsInfo.ClientConfig()
		if err != nil {
			os.RemoveAll(dir)
			return nil, err
		}
		config.TLS = tlsConfig
	}

	if config.TLS != nil {
		creds := credentials.NewTLS(config.TLS)
		config.creds = &creds
		config.transportOpt = grpc.WithTransportCredentials(creds)
	} else {
		config.transportOpt = grpc.WithInsecure()
	}

	socket, err := grpc.Dial(config.Address, config.transportOpt)
	if err != nil {
		return nil, err
	}

	grpclog.SetLogger(log.New(ioutil.Discard, "", log.LstdFlags))

	client := &Client{
		config:   config,
		ctx:      context.Background(),
		closeCh:  make(chan struct{}),
		reqCh:    make(chan *WatchRequest),
		resCh:    make(chan struct{}),
		socket:   socket,
		mc:       NewMydisClient(socket),
		watching: map[string]struct{}{},
		watchers: map[int64]chan *Event{},
	}

	stream, err := client.mc.Watch(client.ctx)
	if err != nil {
		return nil, err
	}
	client.stream = stream

	go client.backgroundProcess()

	return client, err
}

// Close the connection to the server.
func (c *Client) Close() {
	c.lock.Lock()
	c.closing = true
	c.lock.Unlock()

	c.closeCh <- struct{}{}

	if c.config.autoTLSDir != "" {
		os.RemoveAll(c.config.autoTLSDir)
	}
}

// Keys returns a list of valid keys.
func (c *Client) Keys() ([]string, error) {
	res, err := c.mc.Keys(c.ctx, null)
	if err != nil {
		err = normalizeError(err)
		return nil, err
	}
	return res.Keys, nil
}

// KeysWithPrefix returns a list of keys with the given prefix.
func (c *Client) KeysWithPrefix(prefix string) ([]string, error) {
	res, err := c.mc.KeysWithPrefix(c.ctx, &Key{Key: prefix})
	if err != nil {
		err = normalizeError(err)
		return nil, err
	}
	return res.Keys, nil
}

// Has checks if the cache has the given key.
func (c *Client) Has(key string) (bool, error) {
	res, err := c.mc.Has(c.ctx, &Key{Key: key})
	if err != nil {
		err = normalizeError(err)
		return false, err
	}
	return res.Value, nil
}

// SetExpire sets the expiration on a key in seconds.
func (c *Client) SetExpire(key string, seconds int64) error {
	_, err := c.mc.SetExpire(c.ctx, &Expiration{Key: key, Exp: seconds})
	err = normalizeError(err)
	return err
}

// SetLockTimeout sets the default timeout in seconds if already locked.
func (c *Client) SetLockTimeout(seconds int64) {
	if seconds < 1 {
		seconds = 1
	}
	md := metadata.New(map[string]string{
		"maxlockwait": strconv.FormatInt(seconds, 10),
	})
	c.ctx = metadata.NewContext(c.ctx, md)
}

// Lock a key from being modified.
func (c *Client) Lock(key string) error {
	_, err := c.mc.Lock(c.ctx, &Key{Key: key})
	err = normalizeError(err)
	return err
}

// LockWithTimeout locks a key, waiting for the given number of seconds if already locked before returning an error.
func (c *Client) LockWithTimeout(key string, seconds int64) error {
	_, err := c.mc.LockWithTimeout(c.ctx, &Expiration{Key: key, Exp: seconds})
	err = normalizeError(err)
	return err
}

// Unlock a key for modification.
func (c *Client) Unlock(key string) error {
	_, err := c.mc.Unlock(c.ctx, &Key{Key: key})
	err = normalizeError(err)
	return err
}

// UnlockThenSet unlocks a key, then immediately sets its value.
func (c *Client) UnlockThenSet(key string, v Value) error {
	_, err := c.mc.UnlockThenSet(c.ctx, &ByteValue{Key: key, Value: v.b})
	err = normalizeError(err)
	return err
}

// Delete removes a key from the cache.
func (c *Client) Delete(key string) error {
	_, err := c.mc.Delete(c.ctx, &Key{Key: key})
	err = normalizeError(err)
	return err
}

// Clear the cache.
func (c *Client) Clear() error {
	_, err := c.mc.Clear(c.ctx, null)
	err = normalizeError(err)
	return err
}

// Get a value from the cache.
func (c *Client) Get(key string) Value {
	bv, err := c.mc.Get(c.ctx, &Key{Key: key})
	if err != nil {
		err = normalizeError(err)
		return NewValue(err)
	}
	return NewValue(bv.Value)
}

// GetMany gets multiple values from the cache.
func (c *Client) GetMany(keys []string) (map[string]Value, error) {
	h, err := c.mc.GetMany(c.ctx, &KeysList{Keys: keys})
	if err != nil {
		err = normalizeError(err)
		return nil, err
	}

	m := map[string]Value{}
	for k, v := range h.Value {
		m[k] = NewValue(v)
	}
	return m, nil
}

// GetWithPrefix gets the keys with the given prefix.
func (c *Client) GetWithPrefix(prefix string) (map[string]Value, error) {
	h, err := c.mc.GetWithPrefix(c.ctx, &Key{Key: prefix})
	if err != nil {
		err = normalizeError(err)
		return nil, err
	}

	m := map[string]Value{}
	for k, v := range h.Value {
		m[k] = NewValue(v)
	}
	return m, nil
}

// Set a value in the cache.
func (c *Client) Set(key string, v interface{}) error {
	b, err := NewValue(v).Bytes()
	if err != nil {
		return err
	}

	bv := &ByteValue{Key: key, Value: b}
	if _, err := c.mc.Set(c.ctx, bv); err != nil {
		err = normalizeError(err)
		return err
	}
	return nil
}

// SetNX sets a value only if the key doesn't exist, returns true if changed.
func (c *Client) SetNX(key string, v interface{}) (bool, error) {
	b, err := NewValue(v).Bytes()
	if err != nil {
		return false, err
	}

	bool, err := c.mc.SetNX(c.ctx, &ByteValue{Key: key, Value: b})
	if err != nil {
		err = normalizeError(err)
		return false, err
	}
	return bool.Value, nil
}

// SetMany values, returning a map[key]errorText for any errors.
func (c *Client) SetMany(vals map[string]Value) (map[string]string, error) {
	h := &Hash{Value: MapValueToMapBytes(vals)}
	m, err := c.mc.SetMany(c.ctx, h)
	if err != nil {
		err = normalizeError(err)
		return nil, err
	}
	return m.Errors, nil
}

// Length returns the byte length of the value for the given key.
func (c *Client) Length(key string) (int64, error) {
	iv, err := c.mc.Length(c.ctx, &Key{Key: key})
	if err != nil {
		err = normalizeError(err)
		return 0, err
	}
	return iv.Value, nil
}

// IncrementInt increments an integer stored at the given key by the given number and returns new value.
func (c *Client) IncrementInt(key string, by int64) (int64, error) {
	iv, err := c.mc.IncrementInt(c.ctx, &IntValue{Key: key, Value: by})
	if err != nil {
		err = normalizeError(err)
		return 0, err
	}
	return iv.Value, nil
}

// DecrementInt decrements an integer stored at the given key by the given number and returns new value.
func (c *Client) DecrementInt(key string, by int64) (int64, error) {
	iv, err := c.mc.DecrementInt(c.ctx, &IntValue{Key: key, Value: by})
	if err != nil {
		err = normalizeError(err)
		return 0, err
	}
	return iv.Value, nil
}

// IncrementFloat increments a float stored at the given key by the given number and returns new value.
func (c *Client) IncrementFloat(key string, by float64) (float64, error) {
	fv, err := c.mc.IncrementFloat(c.ctx, &FloatValue{Key: key, Value: by})
	if err != nil {
		err = normalizeError(err)
		return 0, err
	}
	return fv.Value, nil
}

// DecrementFloat decrements a float stored at the given key by the given number and returns new value.
func (c *Client) DecrementFloat(key string, by float64) (float64, error) {
	fv, err := c.mc.DecrementFloat(c.ctx, &FloatValue{Key: key, Value: by})
	if err != nil {
		err = normalizeError(err)
		return 0, err
	}
	return fv.Value, nil
}

// GetListItem gets a single item from a list by index, supports negative indexing.
func (c *Client) GetListItem(key string, index int64) Value {
	bv, err := c.mc.GetListItem(c.ctx, &ListItem{Key: key, Index: index})
	if err != nil {
		err = normalizeError(err)
		return NewValue(err)
	}
	return NewValue(bv.Value)
}

// SetListItem sets a single item in a list by index.
func (c *Client) SetListItem(key string, index int64, v interface{}) error {
	b, err := NewValue(v).Bytes()
	if err != nil {
		return err
	}

	_, err = c.mc.SetListItem(c.ctx, &ListItem{Key: key, Index: index, Value: b})
	err = normalizeError(err)
	return err
}

// ListLength gets the number of items in a list.
func (c *Client) ListLength(key string) (int64, error) {
	iv, err := c.mc.ListLength(c.ctx, &Key{Key: key})
	if err != nil {
		err = normalizeError(err)
		return 0, err
	}
	return iv.Value, nil
}

// ListLimit sets the maximum length of a list, removing items from the top once limit is reached.
func (c *Client) ListLimit(key string, limit int64) error {
	_, err := c.mc.ListLimit(c.ctx, &ListItem{Key: key, Index: limit})
	err = normalizeError(err)
	return err
}

// ListInsert inserts a new item at the given index in the list.
func (c *Client) ListInsert(key string, index int64, v interface{}) error {
	b, err := NewValue(v).Bytes()
	if err != nil {
		return err
	}

	_, err = c.mc.ListInsert(c.ctx, &ListItem{Key: key, Index: index, Value: b})
	err = normalizeError(err)
	return err
}

// ListAppend inserts a new item at the end of the list.
func (c *Client) ListAppend(key string, v interface{}) error {
	b, err := NewValue(v).Bytes()
	if err != nil {
		return err
	}

	_, err = c.mc.ListAppend(c.ctx, &ListItem{Key: key, Value: b})
	err = normalizeError(err)
	return err
}

// ListPopLeft returns and removes the first item in a list.
func (c *Client) ListPopLeft(key string) Value {
	bv, err := c.mc.ListPopLeft(c.ctx, &Key{Key: key})
	if err != nil {
		err = normalizeError(err)
		return NewValue(err)
	}
	return NewValue(bv.Value)
}

// ListPopRight returns and removes the last item in a list.
func (c *Client) ListPopRight(key string) Value {
	bv, err := c.mc.ListPopRight(c.ctx, &Key{Key: key})
	if err != nil {
		err = normalizeError(err)
		return NewValue(err)
	}
	return NewValue(bv.Value)
}

// ListHas determines if a list contains an item, returns index or -1 if not found.
func (c *Client) ListHas(key string, v interface{}) (int64, error) {
	b, err := NewValue(v).Bytes()
	if err != nil {
		return 0, err
	}

	iv, err := c.mc.ListHas(c.ctx, &ListItem{Key: key, Value: b})
	if err != nil {
		err = normalizeError(err)
		return 0, err
	}
	return iv.Value, nil
}

// ListDelete removes an item from a list by index.
func (c *Client) ListDelete(key string, index int64) error {
	_, err := c.mc.ListDelete(c.ctx, &ListItem{Key: key, Index: index})
	err = normalizeError(err)
	return err
}

// ListDeleteItem removes the first occurrence of value from a list, returns index or -1 if not found.
func (c *Client) ListDeleteItem(key string, v interface{}) (int64, error) {
	b, err := NewValue(v).Bytes()
	if err != nil {
		return 0, err
	}

	iv, err := c.mc.ListDeleteItem(c.ctx, &ListItem{Key: key, Value: b})
	if err != nil {
		err = normalizeError(err)
		return 0, err
	}
	return iv.Value, nil
}

// GetHashField gets a single value in a hash.
func (c *Client) GetHashField(key, field string) Value {
	bv, err := c.mc.GetHashField(c.ctx, &HashField{Key: key, Field: field})
	if err != nil {
		err = normalizeError(err)
		return NewValue(err)
	}
	return NewValue(bv.Value)
}

// GetHashFields gets multiple hash values.
func (c *Client) GetHashFields(key string, fields []string) (map[string]Value, error) {
	h, err := c.mc.GetHashFields(c.ctx, &HashFieldSet{Key: key, Field: fields})
	if err != nil {
		err = normalizeError(err)
		return nil, err
	}
	m := map[string]Value{}
	for k, v := range h.Value {
		m[k] = NewValue(v)
	}
	return m, nil
}

// HashHas determines if a hash has a given field.
func (c *Client) HashHas(key, field string) (bool, error) {
	b, err := c.mc.HashHas(c.ctx, &HashField{Key: key, Field: field})
	if err != nil {
		err = normalizeError(err)
		return false, err
	}
	return b.Value, nil
}

// HashLength returns the number of fields in a hash.
func (c *Client) HashLength(key string) (int64, error) {
	iv, err := c.mc.HashLength(c.ctx, &Key{Key: key})
	if err != nil {
		err = normalizeError(err)
		return 0, err
	}
	return iv.Value, nil
}

// HashFields gets a list of the fields in a hash.
func (c *Client) HashFields(key string) ([]string, error) {
	lst, err := c.mc.HashFields(c.ctx, &Key{Key: key})
	if err != nil {
		err = normalizeError(err)
		return nil, err
	}
	return lst.Keys, nil
}

// HashValues gets a list of the values in a hash.
func (c *Client) HashValues(key string) ([]Value, error) {
	lst, err := c.mc.HashValues(c.ctx, &Key{Key: key})
	if err != nil {
		err = normalizeError(err)
		return nil, err
	}
	return ListToValues(lst.Value), nil
}

// SetHashField sets a single value in a hash.
func (c *Client) SetHashField(key, field string, v interface{}) error {
	b, err := NewValue(v).Bytes()
	if err != nil {
		err = normalizeError(err)
		return err
	}

	_, err = c.mc.SetHashField(c.ctx, &HashField{Key: key, Field: field, Value: b})
	return err
}

// SetHashFields sets multiple values in a hash.
func (c *Client) SetHashFields(key string, vals map[string]Value) error {
	m := MapValueToMapBytes(vals)
	_, err := c.mc.SetHashFields(c.ctx, &Hash{Key: key, Value: m})
	err = normalizeError(err)
	return err
}

// DelHashField deletes a field from a hash.
func (c *Client) DelHashField(key, field string) error {
	_, err := c.mc.DelHashField(c.ctx, &HashField{Key: key, Field: field})
	err = normalizeError(err)
	return err
}

// NewEventChannel returns a new Event channel.
func (c *Client) NewEventChannel() (ch chan *Event, id int64) {
	id = c.newID
	c.newID++

	ch = make(chan *Event, 100)
	c.lock.Lock()
	c.watchers[id] = ch
	c.lock.Unlock()
	return
}

// CloseEventChannel closes the Event channel.
func (c *Client) CloseEventChannel(id int64) {
	c.lock.Lock()
	if ch, ok := c.watchers[id]; ok {
		close(ch)
		delete(c.watchers, id)
	}
	c.lock.Unlock()
}

// Watch for a key change.
func (c *Client) Watch(key string, prefix bool) {
	r := &WatchRequest{
		Key:    key,
		Prefix: prefix,
	}

	c.reqCh <- r
	<-c.resCh
}

// Unwatch stops watching for a key change.
func (c *Client) Unwatch(key string, prefix bool) {
	r := &WatchRequest{
		Key:    key,
		Prefix: prefix,
		Cancel: true,
	}

	c.reqCh <- r
	<-c.resCh
}

// AuthEnable enables authentication.
func (c *Client) AuthEnable() error {
	_, err := c.mc.AuthEnable(c.ctx, &AuthEnableRequest{})
	return err
}

// AuthDisable disables authentication.
func (c *Client) AuthDisable() error {
	_, err := c.mc.AuthDisable(c.ctx, &AuthDisableRequest{})
	return err
}

// Authenticate processes an authenticate request.
func (c *Client) Authenticate(username, password string) (string, error) {
	resp, err := c.mc.Authenticate(c.ctx, &AuthenticateRequest{Name: username, Password: password})
	if err != nil {
		return "", err
	}
	c.authToken = resp.Token
	md, ok := metadata.FromContext(c.ctx)
	if !ok {
		md = metadata.MD{}
	}
	md["token"] = []string{c.authToken}
	c.ctx = metadata.NewContext(c.ctx, md)
	return resp.Token, err
}

// LogOut removes the cached authentication token, reverting the client to pre-Authenticate state.
func (c *Client) LogOut() {
	md, ok := metadata.FromContext(c.ctx)
	if !ok {
		md = metadata.MD{}
	}
	delete(md, "token")
	c.ctx = metadata.NewContext(c.ctx, md)
}

// UserAdd adds a new user.
func (c *Client) UserAdd(username, password string) error {
	_, err := c.mc.UserAdd(c.ctx, &AuthUserAddRequest{Name: username, Password: password})
	return err
}

// UserGet gets detailed user information.
func (c *Client) UserGet(username string) ([]string, error) {
	resp, err := c.mc.UserGet(c.ctx, &AuthUserGetRequest{Name: username})
	if err != nil {
		return nil, err
	}
	return resp.Roles, err
}

// UserList gets a list of all users.
func (c *Client) UserList() ([]string, error) {
	resp, err := c.mc.UserList(c.ctx, &AuthUserListRequest{})
	if err != nil {
		return nil, err
	}
	return resp.Users, err
}

// UserDelete deletes a specified user.
func (c *Client) UserDelete(username string) error {
	_, err := c.mc.UserDelete(c.ctx, &AuthUserDeleteRequest{Name: username})
	return err
}

// UserChangePassword changes the password of the specified user.
func (c *Client) UserChangePassword(username, password string) error {
	_, err := c.mc.UserChangePassword(c.ctx, &AuthUserChangePasswordRequest{Name: username, Password: password})
	return err
}

// UserGrantRole grants a role to a specified user.
func (c *Client) UserGrantRole(username, role string) error {
	_, err := c.mc.UserGrantRole(c.ctx, &AuthUserGrantRoleRequest{User: username, Role: role})
	return err
}

// UserRevokeRole revokes a role from a specified user.
func (c *Client) UserRevokeRole(username, role string) error {
	_, err := c.mc.UserRevokeRole(c.ctx, &AuthUserRevokeRoleRequest{Name: username, Role: role})
	return err
}

// RoleAdd adds a new role.
func (c *Client) RoleAdd(role string) error {
	_, err := c.mc.RoleAdd(c.ctx, &AuthRoleAddRequest{Name: role})
	return err
}

// RoleGet gets detailed role information.
func (c *Client) RoleGet(role string) ([]*Permission, error) {
	resp, err := c.mc.RoleGet(c.ctx, &AuthRoleGetRequest{Role: role})
	if err != nil {
		return nil, err
	}
	return resp.Perm, err
}

// RoleList gets a list of all roles.
func (c *Client) RoleList() ([]string, error) {
	resp, err := c.mc.RoleList(c.ctx, &AuthRoleListRequest{})
	if err != nil {
		return nil, err
	}
	return resp.Roles, err
}

// RoleDelete deletes a specified role.
func (c *Client) RoleDelete(role string) error {
	_, err := c.mc.RoleDelete(c.ctx, &AuthRoleDeleteRequest{Role: role})
	return err
}

// RoleGrantPermission grants a permission of a specified key or range to a specified role.
func (c *Client) RoleGrantPermission(role string, perm *Permission) error {
	_, err := c.mc.RoleGrantPermission(c.ctx, &AuthRoleGrantPermissionRequest{Name: role, Perm: perm})
	return err
}

// RoleRevokePermission revokes a key or range permission of a specified key.
func (c *Client) RoleRevokePermission(role string, perm *Permission) error {
	_, err := c.mc.RoleRevokePermission(c.ctx, &AuthRoleRevokePermissionRequest{Key: string(perm.Key), RangeEnd: string(perm.RangeEnd), Role: role})
	return err
}

func (c *Client) backgroundProcess() {
	defer func() {
		c.lock.RLock()

		c.stream.CloseSend()

		if c.closing {
			close(c.closeCh)
			close(c.reqCh)
			close(c.resCh)
			c.socket.Close()
		} else {
			// handle reconnects just in case gRPC doesn't do this automatically.
			stream, err := c.mc.Watch(c.ctx)
			if err != nil {
				return
			}
			c.stream = stream

			go c.backgroundProcess()

			for key := range c.watching {
				prefix := false
				if strings.HasSuffix(key, suffixForKeysUsingPrefix) {
					prefix = true
				}
				c.Watch(key, prefix)
			}
		}
		c.lock.RUnlock()
	}()

	// receiver
	go func() {
		for {
			ev, err := c.stream.Recv()
			if err != nil {
				return
			}

			c.lock.RLock()
			for _, ch := range c.watchers {
				ch <- ev
			}
			c.lock.RUnlock()
		}
	}()

	// sender
	for {
		select {
		case <-c.closeCh:
			return
		case r := <-c.reqCh:
			err := c.stream.Send(r)
			c.resCh <- struct{}{}
			if err != nil {
				return
			}

			key := r.Key
			if r.Prefix {
				key += suffixForKeysUsingPrefix
			}

			c.lock.Lock()
			if r.Cancel {
				delete(c.watching, key)
			} else {
				c.watching[key] = struct{}{}
			}
			c.lock.Unlock()
		}
	}
}
