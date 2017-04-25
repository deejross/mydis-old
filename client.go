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

	"github.com/deejross/mydis/pb"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/naming"
)

var (
	// ErrKeyNotFound indicates the given key was not found.
	ErrKeyNotFound = errors.New("Key not found")
)

var knownErrors = map[string]error{
	EtcdKeyNotFound:                  ErrKeyNotFound,
	"Key is locked":                  ErrKeyLocked,
	"List is empty":                  ErrListEmpty,
	"etcdserver: Index out of range": ErrListIndexOutOfRange,
	"Hash field does not exist":      ErrHashFieldNotFound,
	"Type mismatch":                  ErrTypeMismatch,
	"Invalid key name":               ErrInvalidKey,
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

// staticResolver implements both the naming.Resolver and naming.Watcher
// interfaces. It always returns a single static list then blocks forever
type staticResolver struct {
	addresses []*naming.Update
}

func newStaticResolver(addresses []string) *staticResolver {
	sr := &staticResolver{}
	for _, a := range addresses {
		sr.addresses = append(sr.addresses, &naming.Update{
			Op:   naming.Add,
			Addr: a,
		})
	}
	return sr
}

// Resolve just returns the staticResolver it was called from as it satisfies
// both the naming.Resolver and naming.Watcher interfaces
func (sr *staticResolver) Resolve(target string) (naming.Watcher, error) {
	return sr, nil
}

// Next is called in a loop by grpc.RoundRobin expecting updates to which addresses are
// appropriate. Since we just want to return a static list once return a list on the first
// call then block forever on the second instead of sitting in a tight loop
func (sr *staticResolver) Next() ([]*naming.Update, error) {
	if sr.addresses != nil {
		addrs := sr.addresses
		sr.addresses = nil
		return addrs, nil
	}
	// Since staticResolver.Next is called in a tight loop block forever
	// after returning the initial set of addresses
	forever := make(chan struct{})
	<-forever
	return nil, nil
}

// Close does nothing
func (sr *staticResolver) Close() {}

// ClientConfig object.
type ClientConfig struct {
	Addresses    []string
	TLS          *tls.Config
	AutoTLS      bool
	creds        *credentials.TransportCredentials
	transportOpt grpc.DialOption
}

// NewClientConfig returns a new ClientConfig with default values.
func NewClientConfig(address string) ClientConfig {
	return ClientConfig{
		Addresses: []string{address},
		AutoTLS:   true,
	}
}

// NewClientConfigAddresses returns a new ClientConfig multiple node addresses.
func NewClientConfigAddresses(addresses []string) ClientConfig {
	return ClientConfig{
		Addresses: addresses,
		AutoTLS:   true,
	}
}

// Client object.
type Client struct {
	authToken string
	cancel    context.CancelFunc
	config    ClientConfig
	ctx       context.Context
	closeCh   chan struct{}
	closing   bool
	reqCh     chan *pb.WatchRequest
	resCh     chan struct{}
	socket    *grpc.ClientConn
	stream    pb.Mydis_WatchClient
	mc        pb.MydisClient
	lock      sync.RWMutex
	newID     int64
	watching  map[string]struct{}
	watchers  map[int64]chan *pb.Event
}

// NewClient returns a new Client object.
func NewClient(config ClientConfig) (*Client, error) {
	if config.AutoTLS {
		var err error
		config.TLS, err = NewSelfCerts("Mydis")
		if err != nil {
			return nil, err
		}
	}

	if config.TLS != nil {
		creds := credentials.NewTLS(config.TLS)
		config.creds = &creds
		config.transportOpt = grpc.WithTransportCredentials(creds)
	} else {
		config.transportOpt = grpc.WithInsecure()
	}

	balancer := grpc.RoundRobin(newStaticResolver(config.Addresses))
	socket, err := grpc.Dial(config.Addresses[0], config.transportOpt, grpc.WithBalancer(balancer))
	if err != nil {
		return nil, err
	}

	grpclog.SetLogger(log.New(ioutil.Discard, "", log.LstdFlags))

	ctx, cancel := context.WithCancel(context.Background())
	client := &Client{
		cancel:   cancel,
		config:   config,
		ctx:      ctx,
		closeCh:  make(chan struct{}),
		reqCh:    make(chan *pb.WatchRequest),
		resCh:    make(chan struct{}),
		socket:   socket,
		mc:       pb.NewMydisClient(socket),
		watching: map[string]struct{}{},
		watchers: map[int64]chan *pb.Event{},
	}

	go client.backgroundProcess()

	return client, err
}

// Close the connection to the server.
func (c *Client) Close() {
	c.lock.Lock()
	c.closing = true
	c.lock.Unlock()

	c.closeCh <- struct{}{}
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
	res, err := c.mc.KeysWithPrefix(c.ctx, &pb.Key{Key: prefix})
	if err != nil {
		err = normalizeError(err)
		return nil, err
	}
	return res.Keys, nil
}

// Has checks if the cache has the given key.
func (c *Client) Has(key string) (bool, error) {
	res, err := c.mc.Has(c.ctx, &pb.Key{Key: key})
	if err != nil {
		err = normalizeError(err)
		return false, err
	}
	return res.Value, nil
}

// SetExpire sets the expiration on a key in seconds.
func (c *Client) SetExpire(key string, seconds int64) error {
	_, err := c.mc.SetExpire(c.ctx, &pb.Expiration{Key: key, Exp: seconds})
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
	_, err := c.mc.Lock(c.ctx, &pb.Key{Key: key})
	err = normalizeError(err)
	return err
}

// LockWithTimeout locks a key, waiting for the given number of seconds if already locked before returning an error.
func (c *Client) LockWithTimeout(key string, seconds int64) error {
	_, err := c.mc.LockWithTimeout(c.ctx, &pb.Expiration{Key: key, Exp: seconds})
	err = normalizeError(err)
	return err
}

// Unlock a key for modification.
func (c *Client) Unlock(key string) error {
	_, err := c.mc.Unlock(c.ctx, &pb.Key{Key: key})
	err = normalizeError(err)
	return err
}

// UnlockThenSet unlocks a key, then immediately sets its value.
func (c *Client) UnlockThenSet(key string, v Value) error {
	_, err := c.mc.UnlockThenSet(c.ctx, &pb.ByteValue{Key: key, Value: v.b})
	err = normalizeError(err)
	return err
}

// Delete removes a key from the cache.
func (c *Client) Delete(key string) error {
	_, err := c.mc.Delete(c.ctx, &pb.Key{Key: key})
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
	bv, err := c.mc.Get(c.ctx, &pb.Key{Key: key})
	if err != nil {
		err = normalizeError(err)
		return NewValue(err)
	}
	return NewValue(bv.Value)
}

// GetMany gets multiple values from the cache.
func (c *Client) GetMany(keys []string) (map[string]Value, error) {
	h, err := c.mc.GetMany(c.ctx, &pb.KeysList{Keys: keys})
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
	h, err := c.mc.GetWithPrefix(c.ctx, &pb.Key{Key: prefix})
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

	bv := &pb.ByteValue{Key: key, Value: b}
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

	bool, err := c.mc.SetNX(c.ctx, &pb.ByteValue{Key: key, Value: b})
	if err != nil {
		err = normalizeError(err)
		return false, err
	}
	return bool.Value, nil
}

// SetMany values, returning a map[key]errorText for any errors.
func (c *Client) SetMany(vals map[string]Value) (map[string]string, error) {
	h := &pb.Hash{Value: MapValueToMapBytes(vals)}
	m, err := c.mc.SetMany(c.ctx, h)
	if err != nil {
		err = normalizeError(err)
		return nil, err
	}
	return m.Errors, nil
}

// Length returns the byte length of the value for the given key.
func (c *Client) Length(key string) (int64, error) {
	iv, err := c.mc.Length(c.ctx, &pb.Key{Key: key})
	if err != nil {
		err = normalizeError(err)
		return 0, err
	}
	return iv.Value, nil
}

// IncrementInt increments an integer stored at the given key by the given number and returns new value.
func (c *Client) IncrementInt(key string, by int64) (int64, error) {
	iv, err := c.mc.IncrementInt(c.ctx, &pb.IntValue{Key: key, Value: by})
	if err != nil {
		err = normalizeError(err)
		return 0, err
	}
	return iv.Value, nil
}

// DecrementInt decrements an integer stored at the given key by the given number and returns new value.
func (c *Client) DecrementInt(key string, by int64) (int64, error) {
	iv, err := c.mc.DecrementInt(c.ctx, &pb.IntValue{Key: key, Value: by})
	if err != nil {
		err = normalizeError(err)
		return 0, err
	}
	return iv.Value, nil
}

// IncrementFloat increments a float stored at the given key by the given number and returns new value.
func (c *Client) IncrementFloat(key string, by float64) (float64, error) {
	fv, err := c.mc.IncrementFloat(c.ctx, &pb.FloatValue{Key: key, Value: by})
	if err != nil {
		err = normalizeError(err)
		return 0, err
	}
	return fv.Value, nil
}

// DecrementFloat decrements a float stored at the given key by the given number and returns new value.
func (c *Client) DecrementFloat(key string, by float64) (float64, error) {
	fv, err := c.mc.DecrementFloat(c.ctx, &pb.FloatValue{Key: key, Value: by})
	if err != nil {
		err = normalizeError(err)
		return 0, err
	}
	return fv.Value, nil
}

// GetListItem gets a single item from a list by index, supports negative indexing.
func (c *Client) GetListItem(key string, index int64) Value {
	bv, err := c.mc.GetListItem(c.ctx, &pb.ListItem{Key: key, Index: index})
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

	_, err = c.mc.SetListItem(c.ctx, &pb.ListItem{Key: key, Index: index, Value: b})
	err = normalizeError(err)
	return err
}

// ListLength gets the number of items in a list.
func (c *Client) ListLength(key string) (int64, error) {
	iv, err := c.mc.ListLength(c.ctx, &pb.Key{Key: key})
	if err != nil {
		err = normalizeError(err)
		return 0, err
	}
	return iv.Value, nil
}

// ListLimit sets the maximum length of a list, removing items from the top once limit is reached.
func (c *Client) ListLimit(key string, limit int64) error {
	_, err := c.mc.ListLimit(c.ctx, &pb.ListItem{Key: key, Index: limit})
	err = normalizeError(err)
	return err
}

// ListInsert inserts a new item at the given index in the list.
func (c *Client) ListInsert(key string, index int64, v interface{}) error {
	b, err := NewValue(v).Bytes()
	if err != nil {
		return err
	}

	_, err = c.mc.ListInsert(c.ctx, &pb.ListItem{Key: key, Index: index, Value: b})
	err = normalizeError(err)
	return err
}

// ListAppend inserts a new item at the end of the list.
func (c *Client) ListAppend(key string, v interface{}) error {
	b, err := NewValue(v).Bytes()
	if err != nil {
		return err
	}

	_, err = c.mc.ListAppend(c.ctx, &pb.ListItem{Key: key, Value: b})
	err = normalizeError(err)
	return err
}

// ListPopLeft returns and removes the first item in a list.
func (c *Client) ListPopLeft(key string) Value {
	bv, err := c.mc.ListPopLeft(c.ctx, &pb.Key{Key: key})
	if err != nil {
		err = normalizeError(err)
		return NewValue(err)
	}
	return NewValue(bv.Value)
}

// ListPopLeftBlock returns and removes the first item in a list, or waits the given number of seconds for a value, timeout of zero waits forever.
func (c *Client) ListPopLeftBlock(key string, timeout int64) Value {
	bv, err := c.mc.ListPopLeft(c.ctx, &pb.Key{Key: key, Block: true, BlockTimeout: timeout})
	if err != nil {
		err = normalizeError(err)
		return NewValue(err)
	}
	return NewValue(bv.Value)
}

// ListPopRight returns and removes the last item in a list.
func (c *Client) ListPopRight(key string) Value {
	bv, err := c.mc.ListPopRight(c.ctx, &pb.Key{Key: key})
	if err != nil {
		err = normalizeError(err)
		return NewValue(err)
	}
	return NewValue(bv.Value)
}

// ListPopRightBlock returns and removes the last item in a list, or waits the given number of seconds for a value, timeout of zero waits forever.
func (c *Client) ListPopRightBlock(key string, timeout int64) Value {
	bv, err := c.mc.ListPopRight(c.ctx, &pb.Key{Key: key, Block: true, BlockTimeout: timeout})
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

	iv, err := c.mc.ListHas(c.ctx, &pb.ListItem{Key: key, Value: b})
	if err != nil {
		err = normalizeError(err)
		return 0, err
	}
	return iv.Value, nil
}

// ListDelete removes an item from a list by index.
func (c *Client) ListDelete(key string, index int64) error {
	_, err := c.mc.ListDelete(c.ctx, &pb.ListItem{Key: key, Index: index})
	err = normalizeError(err)
	return err
}

// ListDeleteItem removes the first occurrence of value from a list, returns index or -1 if not found.
func (c *Client) ListDeleteItem(key string, v interface{}) (int64, error) {
	b, err := NewValue(v).Bytes()
	if err != nil {
		return 0, err
	}

	iv, err := c.mc.ListDeleteItem(c.ctx, &pb.ListItem{Key: key, Value: b})
	if err != nil {
		err = normalizeError(err)
		return 0, err
	}
	return iv.Value, nil
}

// GetHashField gets a single value in a hash.
func (c *Client) GetHashField(key, field string) Value {
	bv, err := c.mc.GetHashField(c.ctx, &pb.HashField{Key: key, Field: field})
	if err != nil {
		err = normalizeError(err)
		return NewValue(err)
	}
	return NewValue(bv.Value)
}

// GetHashFields gets multiple hash values.
func (c *Client) GetHashFields(key string, fields []string) (map[string]Value, error) {
	h, err := c.mc.GetHashFields(c.ctx, &pb.HashFieldSet{Key: key, Field: fields})
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
	b, err := c.mc.HashHas(c.ctx, &pb.HashField{Key: key, Field: field})
	if err != nil {
		err = normalizeError(err)
		return false, err
	}
	return b.Value, nil
}

// HashLength returns the number of fields in a hash.
func (c *Client) HashLength(key string) (int64, error) {
	iv, err := c.mc.HashLength(c.ctx, &pb.Key{Key: key})
	if err != nil {
		err = normalizeError(err)
		return 0, err
	}
	return iv.Value, nil
}

// HashFields gets a list of the fields in a hash.
func (c *Client) HashFields(key string) ([]string, error) {
	lst, err := c.mc.HashFields(c.ctx, &pb.Key{Key: key})
	if err != nil {
		err = normalizeError(err)
		return nil, err
	}
	return lst.Keys, nil
}

// HashValues gets a list of the values in a hash.
func (c *Client) HashValues(key string) ([]Value, error) {
	lst, err := c.mc.HashValues(c.ctx, &pb.Key{Key: key})
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

	_, err = c.mc.SetHashField(c.ctx, &pb.HashField{Key: key, Field: field, Value: b})
	return err
}

// SetHashFields sets multiple values in a hash.
func (c *Client) SetHashFields(key string, vals map[string]Value) error {
	m := MapValueToMapBytes(vals)
	_, err := c.mc.SetHashFields(c.ctx, &pb.Hash{Key: key, Value: m})
	err = normalizeError(err)
	return err
}

// DelHashField deletes a field from a hash.
func (c *Client) DelHashField(key, field string) error {
	_, err := c.mc.DelHashField(c.ctx, &pb.HashField{Key: key, Field: field})
	err = normalizeError(err)
	return err
}

// NewEventChannel returns a new Event channel.
func (c *Client) NewEventChannel() (ch chan *pb.Event, id int64) {
	id = c.newID
	c.newID++

	ch = make(chan *pb.Event, 100)
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
	r := &pb.WatchRequest{
		Key:    key,
		Prefix: prefix,
	}

	c.reqCh <- r
	<-c.resCh
}

// Unwatch stops watching for a key change.
func (c *Client) Unwatch(key string, prefix bool) {
	r := &pb.WatchRequest{
		Key:    key,
		Prefix: prefix,
		Cancel: true,
	}

	c.reqCh <- r
	<-c.resCh
}

// AuthEnable enables authentication.
func (c *Client) AuthEnable() error {
	_, err := c.mc.AuthEnable(c.ctx, &pb.AuthEnableRequest{})
	return err
}

// AuthDisable disables authentication.
func (c *Client) AuthDisable() error {
	_, err := c.mc.AuthDisable(c.ctx, &pb.AuthDisableRequest{})
	return err
}

// Authenticate processes an authenticate request.
func (c *Client) Authenticate(username, password string) (string, error) {
	resp, err := c.mc.Authenticate(c.ctx, &pb.AuthenticateRequest{Name: username, Password: password})
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
	_, err := c.mc.UserAdd(c.ctx, &pb.AuthUserAddRequest{Name: username, Password: password})
	return err
}

// UserGet gets detailed user information.
func (c *Client) UserGet(username string) ([]string, error) {
	resp, err := c.mc.UserGet(c.ctx, &pb.AuthUserGetRequest{Name: username})
	if err != nil {
		return nil, err
	}
	return resp.Roles, err
}

// UserList gets a list of all users.
func (c *Client) UserList() ([]string, error) {
	resp, err := c.mc.UserList(c.ctx, &pb.AuthUserListRequest{})
	if err != nil {
		return nil, err
	}
	return resp.Users, err
}

// UserDelete deletes a specified user.
func (c *Client) UserDelete(username string) error {
	_, err := c.mc.UserDelete(c.ctx, &pb.AuthUserDeleteRequest{Name: username})
	return err
}

// UserChangePassword changes the password of the specified user.
func (c *Client) UserChangePassword(username, password string) error {
	_, err := c.mc.UserChangePassword(c.ctx, &pb.AuthUserChangePasswordRequest{Name: username, Password: password})
	return err
}

// UserGrantRole grants a role to a specified user.
func (c *Client) UserGrantRole(username, role string) error {
	_, err := c.mc.UserGrantRole(c.ctx, &pb.AuthUserGrantRoleRequest{User: username, Role: role})
	return err
}

// UserRevokeRole revokes a role from a specified user.
func (c *Client) UserRevokeRole(username, role string) error {
	_, err := c.mc.UserRevokeRole(c.ctx, &pb.AuthUserRevokeRoleRequest{Name: username, Role: role})
	return err
}

// RoleAdd adds a new role.
func (c *Client) RoleAdd(role string) error {
	_, err := c.mc.RoleAdd(c.ctx, &pb.AuthRoleAddRequest{Name: role})
	return err
}

// RoleGet gets detailed role information.
func (c *Client) RoleGet(role string) ([]*pb.Permission, error) {
	resp, err := c.mc.RoleGet(c.ctx, &pb.AuthRoleGetRequest{Role: role})
	if err != nil {
		return nil, err
	}
	return resp.Perm, err
}

// RoleList gets a list of all roles.
func (c *Client) RoleList() ([]string, error) {
	resp, err := c.mc.RoleList(c.ctx, &pb.AuthRoleListRequest{})
	if err != nil {
		return nil, err
	}
	return resp.Roles, err
}

// RoleDelete deletes a specified role.
func (c *Client) RoleDelete(role string) error {
	_, err := c.mc.RoleDelete(c.ctx, &pb.AuthRoleDeleteRequest{Role: role})
	return err
}

// RoleGrantPermission grants a permission of a specified key or range to a specified role.
func (c *Client) RoleGrantPermission(role string, perm *pb.Permission) error {
	_, err := c.mc.RoleGrantPermission(c.ctx, &pb.AuthRoleGrantPermissionRequest{Name: role, Perm: perm})
	return err
}

// RoleRevokePermission revokes a key or range permission of a specified key.
func (c *Client) RoleRevokePermission(role string, perm *pb.Permission) error {
	_, err := c.mc.RoleRevokePermission(c.ctx, &pb.AuthRoleRevokePermissionRequest{Key: string(perm.Key), RangeEnd: string(perm.RangeEnd), Role: role})
	return err
}

func (c *Client) backgroundProcess() {
	defer func() {
		c.lock.RLock()
		defer c.lock.RUnlock()
		defer func() {
			if c.closing {
				c.stream.CloseSend()
				close(c.reqCh)
				close(c.resCh)
				c.socket.Close()
			}
		}()

		// handle reconnects.
		if !c.closing {
			go c.backgroundProcess()

		}
	}()

	stream, err := c.mc.Watch(c.ctx)
	if err != nil {
		return
	}
	c.stream = stream

	// re-watch keys after a reconnect
	go func() {
		for key := range c.watching {
			prefix := false
			if strings.HasSuffix(key, suffixForKeysUsingPrefix) {
				prefix = true
			}
			c.Watch(key, prefix)
		}
	}()

	// receiver
	go func() {
		for {
			ev, err := c.stream.Recv()
			if err != nil {
				c.closeCh <- struct{}{}
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
		case <-c.ctx.Done():
			return
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
