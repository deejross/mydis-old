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
	"bytes"
	"time"

	"strconv"

	etcdpb "github.com/coreos/etcd/etcdserver/etcdserverpb"
	"github.com/deejross/mydis/pb"
	"github.com/gogo/protobuf/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc/metadata"
)

func (s *Server) getMaxWait(ctx context.Context) int64 {
	defaultMaxWait := int64(5)
	md, ok := metadata.FromContext(ctx)
	if !ok {
		return defaultMaxWait
	}
	if maxWait, ok := md["maxlockwait"]; ok {
		if len(maxWait) == 0 {
			return defaultMaxWait
		}
		i, err := strconv.ParseInt(maxWait[0], 10, 64)
		if err != nil {
			return defaultMaxWait
		}
		return i
	}
	return defaultMaxWait
}

// Lock a key from being modified. If a lock has already been placed on the key,
// code will block until lock is released, or until 5 seconds has passed. If
// 5 second timeout is reached, ErrKeyLocked is returned.
func (s *Server) Lock(ctx context.Context, key *pb.Key) (*pb.Null, error) {
	maxWait := s.getMaxWait(ctx)
	return s.LockWithTimeout(ctx, &pb.Expiration{Key: key.Key, Exp: maxWait})
}

// LockWithTimeout works the same as Lock, but allows the lock timeout to be specified
// instead of using the default of 5 seconds in the case that a lock has already been
// placed on the key. Setting expiration to zero will timeout immediately. If expiration
// is less than zero, timeout will be set to forever.
func (s *Server) LockWithTimeout(ctx context.Context, ex *pb.Expiration) (*pb.Null, error) {
	maxWait := time.Now().Add(time.Duration(ex.Exp) * time.Second)
	keyLock := getLockName(ex.Key)

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
					Request: &etcdpb.RequestOp_RequestPut{
						RequestPut: &etcdpb.PutRequest{
							Key:   keyLock,
							Value: ZeroByte,
						},
					},
				},
			},
		}); err != nil {
			return null, err
		} else if res.Succeeded == false {
			break
		}

		if ex.Exp == 0 {
			return null, ErrKeyLocked
		}

		time.Sleep(delay)
		if time.Now().After(maxWait) && ex.Exp >= 0 {
			return null, ErrKeyLocked
		}
	}
	return null, nil
}

// Unlock a key for modifications.
func (s *Server) Unlock(ctx context.Context, key *pb.Key) (*pb.Null, error) {
	return s.Delete(ctx, &pb.Key{Key: BytesToString(getLockName(key.Key))})
}

// UnlockThenSet unlocks a key, then immediately sets a new value for it.
func (s *Server) UnlockThenSet(ctx context.Context, val *pb.ByteValue) (*pb.Null, error) {
	bkey := StringToBytes(val.Key)
	if len(bkey) == 0 || bytes.Equal(bkey, ZeroByte) {
		return null, ErrInvalidKey
	}
	keyLock := getLockName(val.Key)
	_, err := s.cache.Server.Txn(ctx, &etcdpb.TxnRequest{
		Compare: []*etcdpb.Compare{
			{
				Key:    ZeroByte,
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
						Key: keyLock,
					},
				},
			},
			{
				Request: &etcdpb.RequestOp_RequestPut{
					RequestPut: &etcdpb.PutRequest{
						Key:   StringToBytes(val.Key),
						Value: val.Value,
					},
				},
			},
		},
	})
	return null, err
}

// UnlockThenSetList unlocks a key, then immediately sets a list value for it.
func (s *Server) UnlockThenSetList(ctx context.Context, val *pb.List) (*pb.Null, error) {
	key := val.Key
	val.Key = ""
	b, err := proto.Marshal(val)
	if err != nil {
		return null, err
	}
	return s.UnlockThenSet(ctx, &pb.ByteValue{Key: key, Value: b})
}

// UnlockThenSetHash unlocks a key, then immediately sets a hash value for it.
func (s *Server) UnlockThenSetHash(ctx context.Context, val *pb.Hash) (*pb.Null, error) {
	key := val.Key
	val.Key = ""
	b, err := proto.Marshal(val)
	if err != nil {
		return null, err
	}
	return s.UnlockThenSet(ctx, &pb.ByteValue{Key: key, Value: b})
}
