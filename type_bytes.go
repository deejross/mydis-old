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

	"github.com/coreos/etcd/etcdserver"
	pb "github.com/coreos/etcd/etcdserver/etcdserverpb"
	"golang.org/x/net/context"
)

var (
	// ErrInvalidKey signals that the given key name is invalid.
	ErrInvalidKey = errors.New("Invalid key name")
)

// Get a byte array from the cache.
func (s *Server) Get(ctx context.Context, key *Key) (*ByteValue, error) {
	res, err := s.cache.Server.Range(ctx, getRangeRequestFromKey(key))
	if err != nil {
		return &ByteValue{}, err
	}
	if res.Count > 0 {
		return &ByteValue{Value: res.Kvs[0].Value}, nil
	}
	return &ByteValue{}, etcdserver.ErrKeyNotFound
}

// GetMany gets a list of values from the cache.
func (s *Server) GetMany(ctx context.Context, keys *KeysList) (*Hash, error) {
	if len(keys.Keys) == 0 {
		return &Hash{}, nil
	}

	req := &pb.TxnRequest{
		Compare: []*pb.Compare{
			{
				Key:    ZeroByte,
				Target: pb.Compare_VALUE,
				Result: pb.Compare_EQUAL,
				TargetUnion: &pb.Compare_Value{
					Value: ZeroByte,
				},
			},
		},
		Failure: []*pb.RequestOp{},
	}

	for _, key := range keys.Keys {
		op := &pb.RequestOp{
			Request: &pb.RequestOp_RequestRange{
				RequestRange: &pb.RangeRequest{
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

	h := &Hash{Value: map[string][]byte{}}
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
func (s *Server) GetWithPrefix(ctx context.Context, key *Key) (*Hash, error) {
	req := getRangeRequestFromKey(key)
	req.RangeEnd = getPrefix(key.Key)
	res, err := s.cache.Server.Range(ctx, req)
	if err != nil {
		return nil, err
	}

	h := &Hash{Value: map[string][]byte{}}
	for _, kv := range res.Kvs {
		h.Value[BytesToString(kv.Key)] = kv.Value
	}
	return h, nil
}

// Set a byte array in the cache.
func (s *Server) Set(ctx context.Context, val *ByteValue) (*Null, error) {
	bkey := StringToBytes(val.Key)
	if len(bkey) == 0 || bytes.Equal(bkey, ZeroByte) {
		return null, ErrInvalidKey
	}

	maxW := s.getMaxWait(ctx)
	maxWait := time.Now().Add(time.Duration(maxW) * time.Second)
	keyLock := getLockName(val.Key)

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
					Request: &pb.RequestOp_RequestPut{
						RequestPut: &pb.PutRequest{
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
func (s *Server) SetNX(ctx context.Context, val *ByteValue) (*Bool, error) {
	key := &Key{Key: val.Key}
	if _, err := s.Lock(ctx, key); err != nil {
		return nil, err
	}
	if b, err := s.Has(ctx, key); err != nil {
		return nil, err
	} else if b.Value {
		return &Bool{Value: false}, nil
	}

	if _, err := s.UnlockThenSet(ctx, val); err != nil {
		return nil, err
	}
	return &Bool{Value: true}, nil
}

// SetMany sets multiple byte arrays. Returns a map[key]errorText of any errors encountered.
func (s *Server) SetMany(ctx context.Context, h *Hash) (*ErrorHash, error) {
	errors := &ErrorHash{Errors: map[string]string{}}

	for key, val := range h.Value {
		bv := &ByteValue{Key: key, Value: val}
		_, err := s.Set(ctx, bv)
		if err != nil {
			errors.Errors[key] = err.Error()
		}
	}
	return errors, nil
}

// Length returns the length of the value for the given key.
func (s *Server) Length(ctx context.Context, key *Key) (*IntValue, error) {
	bv, err := s.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	return &IntValue{Value: int64(len(bv.Value))}, nil
}
