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
	"time"

	"github.com/coreos/etcd/etcdserver"
	etcdpb "github.com/coreos/etcd/etcdserver/etcdserverpb"
	"github.com/deejross/mydis/pb"
	"golang.org/x/net/context"
)

// Keys returns a list of valid keys.
func (s *Server) Keys(ctx context.Context, null *pb.Null) (*pb.KeysList, error) {
	res, err := s.cache.Server.Range(ctx, &etcdpb.RangeRequest{
		Key:      ZeroByte,
		RangeEnd: ZeroByte,
		KeysOnly: true,
	})
	if err != nil {
		return nil, err
	}
	return kvsToList(res.Kvs), nil
}

// KeysWithPrefix returns a list of keys with the given prefix.
func (s *Server) KeysWithPrefix(ctx context.Context, key *pb.Key) (*pb.KeysList, error) {
	res, err := s.cache.Server.Range(ctx, &etcdpb.RangeRequest{
		Key:      StringToBytes(key.Key),
		RangeEnd: getPrefix(key.Key),
		KeysOnly: true,
	})
	if err != nil {
		return nil, err
	}
	return kvsToList(res.Kvs), nil
}

// Has determines if the given key exists.
func (s *Server) Has(ctx context.Context, key *pb.Key) (*pb.Bool, error) {
	res, err := s.cache.Server.Range(ctx, &etcdpb.RangeRequest{
		Key:      StringToBytes(key.Key),
		KeysOnly: true,
	})
	if err == etcdserver.ErrKeyNotFound {
		return &pb.Bool{Value: false}, nil
	} else if err != nil {
		return nil, err
	}

	for _, kv := range res.Kvs {
		if key.Key == BytesToString(kv.Key) {
			return &pb.Bool{Value: true}, nil
		}
	}
	return &pb.Bool{Value: false}, nil
}

// SetExpire sets the expiration in seconds on a key.
func (s *Server) SetExpire(ctx context.Context, ex *pb.Expiration) (*pb.Null, error) {
	res, err := s.cache.Server.LeaseGrant(ctx, &etcdpb.LeaseGrantRequest{
		TTL: ex.Exp,
	})
	if err != nil {
		return null, err
	}

	_, err = s.cache.Server.Put(ctx, &etcdpb.PutRequest{
		Key:         StringToBytes(ex.Key),
		IgnoreValue: true,
		Lease:       res.ID,
	})

	return null, err
}

// Delete a key from the cache.
func (s *Server) Delete(ctx context.Context, key *pb.Key) (*pb.Null, error) {
	maxW := time.Duration(s.getMaxWait(ctx))
	maxWait := time.Now().Add(maxW * time.Second)
	keyLock := getLockName(key.Key)

	for {
		if res, err := s.cache.Server.Txn(ctx, &etcdpb.TxnRequest{
			Compare: []*etcdpb.Compare{
				{
					Key:    keyLock,
					Target: etcdpb.Compare_VALUE,
					Result: etcdpb.Compare_EQUAL,
					TargetUnion: &etcdpb.Compare_Value{
						Value: ZeroByte,
					},
				},
			},
			Failure: []*etcdpb.RequestOp{
				{
					Request: &etcdpb.RequestOp_RequestDeleteRange{
						RequestDeleteRange: &etcdpb.DeleteRangeRequest{
							Key: StringToBytes(key.Key),
						},
					},
				},
			},
		}); err != nil {
			return null, err
		} else if res.Succeeded == false {
			break
		}

		time.Sleep(delay)
		if time.Now().After(maxWait) {
			return null, ErrKeyLocked
		}
	}
	return null, nil
}

// Clear all keys in the cache.
func (s *Server) Clear(ctx context.Context, null *pb.Null) (*pb.Null, error) {
	_, err := s.cache.Server.DeleteRange(ctx, &etcdpb.DeleteRangeRequest{
		Key:      ZeroByte,
		RangeEnd: ZeroByte,
	})
	return null, err
}
