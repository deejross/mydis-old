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
	"github.com/coreos/etcd/etcdserver"
	"github.com/gogo/protobuf/proto"
	"golang.org/x/net/context"
)

// GetFloat gets a float value for the given key.
func (s *Server) GetFloat(ctx context.Context, key *Key) (*FloatValue, error) {
	bv, err := s.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	fv := &FloatValue{}
	if err := proto.Unmarshal(bv.Value, fv); err != nil {
		return nil, err
	}
	return fv, nil
}

// SetFloat sets a float.
func (s *Server) SetFloat(ctx context.Context, fv *FloatValue) (*Null, error) {
	b, err := proto.Marshal(fv)
	if err != nil {
		return nil, err
	}
	return s.Set(ctx, &ByteValue{Key: fv.Key, Value: b})
}

// IncrementFloat increments a float stored at the given key by the number and returns the new value.
func (s *Server) IncrementFloat(ctx context.Context, fv *FloatValue) (*FloatValue, error) {
	key := &Key{Key: fv.Key}
	if _, err := s.Lock(ctx, key); err != nil {
		return nil, err
	}

	oldfv, err := s.GetFloat(ctx, key)
	if err == etcdserver.ErrKeyNotFound {
		oldfv = &FloatValue{Value: 0}
	} else if err != nil {
		return nil, err
	}

	newval := &FloatValue{Value: oldfv.Value + fv.Value}
	b, err := proto.Marshal(newval)
	if err != nil {
		return nil, err
	}

	if _, err := s.UnlockThenSet(ctx, &ByteValue{Key: fv.Key, Value: b}); err != nil {
		return nil, err
	}
	return newval, nil
}

// DecrementFloat decrements a float stored at the given key by the number and returns the new value.
func (s *Server) DecrementFloat(ctx context.Context, fv *FloatValue) (*FloatValue, error) {
	fv.Value *= -1
	return s.IncrementFloat(ctx, fv)
}
