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
	"path"
	"unsafe"

	"time"

	"net"

	"github.com/coreos/etcd/embed"
	pb "github.com/coreos/etcd/etcdserver/etcdserverpb"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/coreos/etcd/pkg/transport"
	"github.com/ghodss/yaml"
	"github.com/golang/protobuf/proto"
)

var null = &Null{}
var suffixForKeysUsingPrefix = "*_MYDIS_WITHPREFIX"
var suffixForLocks = "*_MYDIS_LOCK"

// ZeroByte represents a single zero byte in a byte slice.
var ZeroByte = []byte{0}

// VERSION of Mydis
var VERSION = "0.5.0"

// Value object.
type Value struct {
	err error
	b   []byte
}

// NewValue returns a new Value object.
func NewValue(t interface{}) Value {
	switch t := t.(type) {
	case error:
		return Value{err: t}
	case Value:
		return t
	case []byte:
		return Value{b: t}
	case string:
		return Value{b: StringToBytes(t)}
	case bool:
		if t {
			return Value{b: []byte{1}}
		}
		return Value{b: ZeroByte}
	case proto.Message:
		b, _ := proto.Marshal(t)
		return Value{b: b}
	case int:
		b, _ := proto.Marshal(&IntValue{Value: int64(t)})
		return Value{b: b}
	case int8:
		b, _ := proto.Marshal(&IntValue{Value: int64(t)})
		return Value{b: b}
	case int16:
		b, _ := proto.Marshal(&IntValue{Value: int64(t)})
		return Value{b: b}
	case int32:
		b, _ := proto.Marshal(&IntValue{Value: int64(t)})
		return Value{b: b}
	case int64:
		b, _ := proto.Marshal(&IntValue{Value: t})
		return Value{b: b}
	case float32:
		b, _ := proto.Marshal(&FloatValue{Value: float64(t)})
		return Value{b: b}
	case float64:
		b, _ := proto.Marshal(&FloatValue{Value: t})
		return Value{b: b}
	case time.Time:
		b, _ := proto.Marshal(&IntValue{Value: int64(t.Nanosecond())})
		return Value{b: b}
	case time.Duration:
		b, _ := proto.Marshal(&IntValue{Value: t.Nanoseconds()})
		return Value{b: b}
	case [][]byte:
		b, _ := proto.Marshal(&List{Value: t})
		return Value{b: b}
	case []string:
		b, _ := proto.Marshal(&List{Value: ListStringToBytes(t)})
		return Value{b: b}
	case []Value:
		b, _ := proto.Marshal(&List{Value: ListValueToBytes(t)})
		return Value{b: b}
	case map[string]string:
		b, _ := proto.Marshal(&Hash{Value: MapStringToMapBytes(t)})
		return Value{b: b}
	case map[string][]byte:
		b, _ := proto.Marshal(&Hash{Value: t})
		return Value{b: b}
	case map[string]bool:
		b, _ := proto.Marshal(&Hash{Value: MapBoolToMapBytes(t)})
		return Value{b: b}
	case map[string]int64:
		b, _ := proto.Marshal(&Hash{Value: MapIntToMapBytes(t)})
		return Value{b: b}
	case map[string]float64:
		b, _ := proto.Marshal(&Hash{Value: MapFloatToMapBytes(t)})
		return Value{b: b}
	case map[string]Value:
		b, _ := proto.Marshal(&Hash{Value: MapValueToMapBytes(t)})
		return Value{b: b}
	default:
		return Value{err: errors.New("Unknown type, recommend marshalling to byte slice")}
	}
}

// Error returns the error associated with this Value, if one exists.
func (v Value) Error() error {
	return v.err
}

// Bytes returns the Value as a byte slice.
func (v Value) Bytes() ([]byte, error) {
	if v.err != nil {
		return nil, v.err
	}
	return v.b, nil
}

// String returns the Value as a string.
func (v Value) String() (string, error) {
	if v.err != nil {
		return "", v.err
	}
	return BytesToString(v.b), nil
}

// Bool returns the Value as bool.
func (v Value) Bool() (bool, error) {
	if v.err != nil {
		return false, v.err
	}
	if len(v.b) > 0 && v.b[0] != 0 {
		return true, nil
	}
	return false, nil
}

// Proto returns the given ProtoMessage.
func (v Value) Proto(pb proto.Message) error {
	if v.err != nil {
		return v.err
	}
	if pb == nil {
		return nil
	}
	err := proto.Unmarshal(v.b, pb)
	if err != nil {
		return err
	}
	return nil
}

// Int returns the Value as an int64.
func (v Value) Int() (int64, error) {
	iv := &IntValue{}
	if err := v.Proto(iv); err != nil {
		return 0, err
	}
	return iv.Value, nil
}

// Float returns the Value as a flat64.
func (v Value) Float() (float64, error) {
	fv := &FloatValue{}
	if err := v.Proto(fv); err != nil {
		return 0, err
	}
	return fv.Value, nil
}

// Time returns the Value as a Time.
func (v Value) Time() (time.Time, error) {
	iv := &IntValue{}
	if err := v.Proto(iv); err != nil {
		return time.Time{}, err
	}
	return time.Unix(0, iv.Value), nil
}

// Duration returns the Value as a Duration.
func (v Value) Duration() (time.Duration, error) {
	iv := &IntValue{}
	if err := v.Proto(iv); err != nil {
		return 0, err
	}
	return time.Duration(iv.Value), nil
}

// List returns the Value as a List.
func (v Value) List() ([]Value, error) {
	lst := &List{}
	if err := v.Proto(lst); err != nil {
		return nil, err
	}
	return ListToValues(lst.Value), nil
}

// Map returns the Value as a Map.
func (v Value) Map() (map[string]Value, error) {
	h := &Hash{}
	if err := v.Proto(h); err != nil {
		return nil, err
	}
	return MapBytesToValues(h.Value), nil
}

// ListToValues converts a [][]byte to []Value.
func ListToValues(lst [][]byte) []Value {
	if lst == nil {
		return []Value{}
	}

	newLst := make([]Value, len(lst))
	for i := 0; i < len(lst); i++ {
		newLst[i] = NewValue(lst[i])
	}
	return newLst
}

// ListStringToValues convers a []string to []Value.
func ListStringToValues(lst []string) []Value {
	if lst == nil {
		return []Value{}
	}

	newLst := make([]Value, len(lst))
	for i := 0; i < len(lst); i++ {
		newLst[i] = NewValue(lst[i])
	}
	return newLst
}

// ListStringToBytes converts a []string to [][]byte.
func ListStringToBytes(lst []string) [][]byte {
	if lst == nil {
		return [][]byte{}
	}

	newLst := make([][]byte, len(lst))
	for i := 0; i < len(lst); i++ {
		newLst[i] = NewValue(lst[i]).b
	}
	return newLst
}

// ListValueToBytes converts a []Value to [][]byte.
func ListValueToBytes(lst []Value) [][]byte {
	if lst == nil {
		return [][]byte{}
	}

	newLst := make([][]byte, len(lst))
	for i := 0; i < len(lst); i++ {
		newLst[i] = NewValue(lst[i]).b
	}
	return newLst
}

// MapBytesToValues converts a map[string][]byte to map[string]Value.
func MapBytesToValues(h map[string][]byte) map[string]Value {
	m := map[string]Value{}
	if h == nil {
		return m
	}
	for k, v := range h {
		m[k] = NewValue(v)
	}
	return m
}

// MapStringToMapBytes converts a map[string]string to map[string][]byte.
func MapStringToMapBytes(h map[string]string) map[string][]byte {
	m := map[string][]byte{}
	if h == nil {
		return m
	}
	for k, v := range h {
		m[k] = StringToBytes(v)
	}
	return m
}

// MapBoolToMapBytes converts a map[string]string to map[string][]byte.
func MapBoolToMapBytes(h map[string]bool) map[string][]byte {
	m := map[string][]byte{}
	if h == nil {
		return m
	}
	for k, v := range h {
		m[k] = NewValue(v).b
	}
	return m
}

// MapIntToMapBytes converts a map[string]string to map[string][]byte.
func MapIntToMapBytes(h map[string]int64) map[string][]byte {
	m := map[string][]byte{}
	if h == nil {
		return m
	}
	for k, v := range h {
		m[k] = NewValue(v).b
	}
	return m
}

// MapFloatToMapBytes converts a map[string]string to map[string][]byte.
func MapFloatToMapBytes(h map[string]float64) map[string][]byte {
	m := map[string][]byte{}
	if h == nil {
		return m
	}
	for k, v := range h {
		m[k] = NewValue(v).b
	}
	return m
}

// MapValueToMapBytes converts a map[string]Value to map[string][]byte.
func MapValueToMapBytes(h map[string]Value) map[string][]byte {
	m := map[string][]byte{}
	if h == nil {
		return m
	}
	for k, v := range h {
		m[k] = v.b
	}
	return m
}

// BytesToString efficiently converts a byte slice to a string without allocating any additional memory.
func BytesToString(b []byte) string {
	p := unsafe.Pointer(&b)
	return *(*string)(p)
}

// StringToBytes efficiently converts a string to a byte slice without allocating any additional memory.
func StringToBytes(s string) []byte {
	p := unsafe.Pointer(&s)
	return *(*[]byte)(p)
}

// ConfigFromYAML creates a Config object from YAML.
func ConfigFromYAML(b []byte) (*embed.Config, error) {
	config := &embed.Config{}
	err := yaml.Unmarshal(b, config)
	if err != nil {
		return nil, err
	}
	return config, config.Validate()
}

// ConfigToYAML creates a YAML config from a Config object.
func ConfigToYAML(config *embed.Config) ([]byte, error) {
	return yaml.Marshal(config)
}

// CopyConfig creates a new copy of a Config object.
func CopyConfig(config *embed.Config) *embed.Config {
	b, _ := ConfigToYAML(config)
	c, _ := ConfigFromYAML(b)
	return c
}

func getLockName(key string) []byte {
	return StringToBytes(key + suffixForLocks)
}

func kvsToList(kvs []*mvccpb.KeyValue) *KeysList {
	lst := &KeysList{Keys: []string{}}
	for _, kv := range kvs {
		lst.Keys = append(lst.Keys, BytesToString(kv.Key))
	}
	return lst
}

func getRangeRequestFromKey(key *Key) *pb.RangeRequest {
	req := &pb.RangeRequest{
		Key: StringToBytes(key.Key),
	}
	if key.Limit > 0 {
		req.Limit = key.Limit
	}
	if key.Revision > 0 {
		req.Revision = key.Revision
	}
	return req
}

func getPrefix(key string) []byte {
	bkey := StringToBytes(key)
	end := make([]byte, len(bkey))
	copy(end, bkey)
	for i := len(end) - 1; i >= 0; i-- {
		if end[i] < 0xff {
			end[i]++
			end = end[:i+1]
			return end
		}
	}
	return ZeroByte
}

func enforceListLimit(lst *List) {
	if lst.Limit == 0 {
		return
	}

	size := int64(len(lst.Value))
	if size <= lst.Limit {
		return
	}

	lst.Value = lst.Value[size-lst.Limit:]
}

func generateTLSInfo(config *embed.Config) (transport.TLSInfo, error) {
	if config.ClientAutoTLS && config.ClientTLSInfo.Empty() {
		return generateCert(path.Join(config.Dir, "fixtures/mydis-client"))
	}

	return config.ClientTLSInfo, nil
}

func generateCert(path string) (transport.TLSInfo, error) {
	info := transport.TLSInfo{}

	interfaces, err := net.Interfaces()
	if err != nil {
		return info, err
	}
	hosts := []string{}
	for _, iface := range interfaces {
		addresses, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addresses {
			hosts = append(hosts, a.String())
		}
	}

	return transport.SelfCert(path, hosts)
}
