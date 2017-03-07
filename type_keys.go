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
	pb "github.com/coreos/etcd/etcdserver/etcdserverpb"
	"golang.org/x/net/context"
)

// Keys returns a list of valid keys.
func (s *Server) Keys(ctx context.Context, null *Null) (*KeysList, error) {
	res, err := s.cache.Server.Range(ctx, &pb.RangeRequest{
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
func (s *Server) KeysWithPrefix(ctx context.Context, key *Key) (*KeysList, error) {
	res, err := s.cache.Server.Range(ctx, &pb.RangeRequest{
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
func (s *Server) Has(ctx context.Context, key *Key) (*Bool, error) {
	res, err := s.cache.Server.Range(ctx, &pb.RangeRequest{
		Key:      StringToBytes(key.Key),
		KeysOnly: true,
	})
	if err == etcdserver.ErrKeyNotFound {
		return &Bool{Value: false}, nil
	} else if err != nil {
		return nil, err
	}

	for _, kv := range res.Kvs {
		if key.Key == BytesToString(kv.Key) {
			return &Bool{Value: true}, nil
		}
	}
	return &Bool{Value: false}, nil
}

// SetExpire sets the expiration in seconds on a key.
func (s *Server) SetExpire(ctx context.Context, ex *Expiration) (*Null, error) {
	res, err := s.cache.Server.LeaseGrant(ctx, &pb.LeaseGrantRequest{
		TTL: ex.Exp,
	})
	if err != nil {
		return null, err
	}

	_, err = s.cache.Server.Put(ctx, &pb.PutRequest{
		Key:         StringToBytes(ex.Key),
		IgnoreValue: true,
		Lease:       res.ID,
	})

	return null, err
}

// Delete a key from the cache.
func (s *Server) Delete(ctx context.Context, key *Key) (*Null, error) {
	maxW := time.Duration(s.getMaxWait(ctx))
	maxWait := time.Now().Add(maxW * time.Second)
	keyLock := getLockName(key.Key)

	for {
		if res, err := s.cache.Server.Txn(ctx, &pb.TxnRequest{
			Compare: []*pb.Compare{
				{
					Key:    keyLock,
					Target: pb.Compare_VALUE,
					Result: pb.Compare_EQUAL,
					TargetUnion: &pb.Compare_Value{
						Value: ZeroByte,
					},
				},
			},
			Failure: []*pb.RequestOp{
				{
					Request: &pb.RequestOp_RequestDeleteRange{
						RequestDeleteRange: &pb.DeleteRangeRequest{
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
func (s *Server) Clear(ctx context.Context, null *Null) (*Null, error) {
	_, err := s.cache.Server.DeleteRange(ctx, &pb.DeleteRangeRequest{
		Key:      ZeroByte,
		RangeEnd: ZeroByte,
	})
	return null, err
}
