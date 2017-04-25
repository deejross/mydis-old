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
	"errors"
	"time"

	etcdpb "github.com/coreos/etcd/etcdserver/etcdserverpb"
	"github.com/deejross/mydis/pb"
	"golang.org/x/net/context"
)

var (
	// ErrInvalidKey signals that the given key name is invalid.
	ErrInvalidKey = errors.New("Invalid key name")
	// ErrTypeMismatch signals that the type of value being requested is unexpected.
	ErrTypeMismatch = errors.New("Type mismatch")
	// EtcdKeyNotFound is the error message Etcd uses to denote key was not found.
	EtcdKeyNotFound = "Key not found"
)

// Get a byte array from the cache.
func (s *Server) Get(ctx context.Context, key *pb.Key) (*pb.ByteValue, error) {
	res, err := s.cache.Server.Range(ctx, getRangeRequestFromKey(key))

	if err != nil && err.Error() == EtcdKeyNotFound {
		return &pb.ByteValue{}, ErrKeyNotFound
	} else if err != nil {
		return &pb.ByteValue{}, err
	} else if res.Kvs == nil && key.Block {
		waited := time.Duration(0)
		sleep := 10 * time.Millisecond

		for {
			key.Block = false
			if res, err := s.Get(ctx, key); err == ErrKeyNotFound {
				time.Sleep(sleep)
				waited += sleep
				if waited.Seconds() >= float64(key.BlockTimeout) && key.BlockTimeout > 0 {
					return res, err
				}
			} else {
				return res, err
			}
		}
	} else if res.Count > 0 {
		return &pb.ByteValue{Value: res.Kvs[0].Value}, nil
	}

	return &pb.ByteValue{}, ErrKeyNotFound
}

// GetMany gets a list of values from the cache.
func (s *Server) GetMany(ctx context.Context, keys *pb.KeysList) (*pb.Hash, error) {
	if len(keys.Keys) == 0 {
		return &pb.Hash{}, nil
	}

	req := &etcdpb.TxnRequest{
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
		Failure: []*etcdpb.RequestOp{},
	}

	for _, key := range keys.Keys {
		op := &etcdpb.RequestOp{
			Request: &etcdpb.RequestOp_RequestRange{
				RequestRange: &etcdpb.RangeRequest{
					Key: StringToBytes(key),
				},
			},
		}
		req.Failure = append(req.Failure, op)
	}

	res, err := s.cache.Server.Txn(ctx, req)
	if err != nil {
		return nil, err
	}

	h := &pb.Hash{Value: map[string][]byte{}}
	for i, op := range res.Responses {
		key := keys.Keys[i]
		kvs := op.GetResponseRange().Kvs
		if kvs != nil && len(kvs) > 0 {
			h.Value[key] = op.GetResponseRange().Kvs[0].Value
		}
	}
	return h, nil
}

// GetWithPrefix gets all byte arrays with the given prefix.
func (s *Server) GetWithPrefix(ctx context.Context, key *pb.Key) (*pb.Hash, error) {
	req := getRangeRequestFromKey(key)
	req.RangeEnd = getPrefix(key.Key)
	res, err := s.cache.Server.Range(ctx, req)
	if err != nil {
		return nil, err
	}

	h := &pb.Hash{Value: map[string][]byte{}}
	for _, kv := range res.Kvs {
		h.Value[BytesToString(kv.Key)] = kv.Value
	}
	return h, nil
}

// Set a byte array in the cache.
func (s *Server) Set(ctx context.Context, val *pb.ByteValue) (*pb.Null, error) {
	bkey := StringToBytes(val.Key)
	if len(bkey) == 0 || bytes.Equal(bkey, ZeroByte) {
		return null, ErrInvalidKey
	}

	maxW := s.getMaxWait(ctx)
	maxWait := time.Now().Add(time.Duration(maxW) * time.Second)
	keyLock := getLockName(val.Key)

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
							Key:   StringToBytes(val.Key),
							Value: val.Value,
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

// SetNX sets a value only if the key doesn't exist, returns true if changed.
func (s *Server) SetNX(ctx context.Context, val *pb.ByteValue) (*pb.Bool, error) {
	key := &pb.Key{Key: val.Key}
	if _, err := s.Lock(ctx, key); err != nil {
		return nil, err
	}
	if b, err := s.Has(ctx, key); err != nil {
		return nil, err
	} else if b.Value {
		return &pb.Bool{Value: false}, nil
	}

	if _, err := s.UnlockThenSet(ctx, val); err != nil {
		return nil, err
	}
	return &pb.Bool{Value: true}, nil
}

// SetMany sets multiple byte arrays. Returns a map[key]errorText of any errors encountered.
func (s *Server) SetMany(ctx context.Context, h *pb.Hash) (*pb.ErrorHash, error) {
	errors := &pb.ErrorHash{Errors: map[string]string{}}

	for key, val := range h.Value {
		bv := &pb.ByteValue{Key: key, Value: val}
		_, err := s.Set(ctx, bv)
		if err != nil {
			errors.Errors[key] = err.Error()
		}
	}
	return errors, nil
}

// Length returns the length of the value for the given key.
func (s *Server) Length(ctx context.Context, key *pb.Key) (*pb.IntValue, error) {
	bv, err := s.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	return &pb.IntValue{Value: int64(len(bv.Value))}, nil
}
