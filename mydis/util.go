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
	"strings"

	"crypto/tls"

	"github.com/coreos/etcd/embed"
	etcdpb "github.com/coreos/etcd/etcdserver/etcdserverpb"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/deejross/mydis/pb"
	"github.com/deejross/mydis/util"
	"github.com/ghodss/yaml"
)

var null = &pb.Null{}
var suffixForKeysUsingPrefix = "*_MYDIS_WITHPREFIX"
var suffixForLocks = "*_MYDIS_LOCK"

// ZeroByte represents a single zero byte in a byte slice.
var ZeroByte = []byte{0}

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
	return util.StringToBytes(key + suffixForLocks)
}

func kvsToList(kvs []*mvccpb.KeyValue) *pb.KeysList {
	lst := &pb.KeysList{Keys: []string{}}
	for _, kv := range kvs {
		lst.Keys = append(lst.Keys, util.BytesToString(kv.Key))
	}
	return lst
}

func getRangeRequestFromKey(key *pb.Key) *etcdpb.RangeRequest {
	req := &etcdpb.RangeRequest{
		Key: util.StringToBytes(key.Key),
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
	bkey := util.StringToBytes(key)
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

// GetKeyPrefix returns the actual rangeStart and rangeEnd for keys that end with '*'.
func GetKeyPrefix(key string) (bkey []byte, rangEnd []byte) {
	if strings.HasSuffix(key, "*") {
		newKey := strings.TrimSuffix(key, "*")
		rangEnd = getPrefix(newKey)
		bkey = util.StringToBytes(newKey)
		return
	}
	return util.StringToBytes(key), ZeroByte
}

// GetPermission returns a new Permission object for the given information or nil if permName unrecognized.
func GetPermission(key string, permName string) *pb.Permission {
	bkey, rangeEnd := GetKeyPrefix(key)
	permType, ok := pb.Permission_Type_value[strings.ToUpper(strings.TrimSpace(permName))]
	if !ok {
		return nil
	}

	return &pb.Permission{
		Key:      bkey,
		RangeEnd: rangeEnd,
		PermType: pb.Permission_Type(permType),
	}
}

func enforceListLimit(lst *pb.List) {
	if lst.Limit == 0 {
		return
	}

	size := int64(len(lst.Value))
	if size <= lst.Limit {
		return
	}

	lst.Value = lst.Value[size-lst.Limit:]
}

func generateTLSInfo(config *embed.Config) (*tls.Config, error) {
	if config.ClientAutoTLS && config.ClientTLSInfo.Empty() {
		return util.NewSelfCerts("Mydis")
	}

	return nil, nil
}
