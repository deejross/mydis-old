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

	"github.com/coreos/etcd/etcdserver"
	"github.com/gogo/protobuf/proto"
	"golang.org/x/net/context"
)

// GetInt gets an integer value for the given key.
func (s *Server) GetInt(ctx context.Context, key *Key) (*IntValue, error) {
	bv, err := s.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	iv := &IntValue{}
	if err := proto.Unmarshal(bv.Value, iv); strings.HasPrefix(err.Error(), "proto: can't skip unknown wire type") {
		return nil, ErrTypeMismatch
	} else if err != nil {
		return nil, err
	}
	return iv, nil
}

// SetInt sets an integer.
func (s *Server) SetInt(ctx context.Context, iv *IntValue) (*Null, error) {
	b, err := proto.Marshal(iv)
	if err != nil {
		return null, err
	}
	return s.Set(ctx, &ByteValue{Key: iv.Key, Value: b})
}

// IncrementInt increments an integer stored at the given key by the number and returns the new value.
func (s *Server) IncrementInt(ctx context.Context, iv *IntValue) (*IntValue, error) {
	key := &Key{Key: iv.Key}
	if _, err := s.Lock(ctx, key); err != nil {
		return nil, err
	}

	oldiv, err := s.GetInt(ctx, key)
	if err == etcdserver.ErrKeyNotFound {
		oldiv = &IntValue{Value: 0}
	} else if err != nil {
		s.Unlock(ctx, key)
		return nil, err
	}

	newval := &IntValue{Value: oldiv.Value + iv.Value}
	b, err := proto.Marshal(newval)
	if err != nil {
		s.Unlock(ctx, key)
		return nil, err
	}

	if _, err := s.UnlockThenSet(ctx, &ByteValue{Key: iv.Key, Value: b}); err != nil {
		return nil, err
	}
	return newval, nil
}

// DecrementInt decrements an integer stored at the given key by the number and returns the new value.
func (s *Server) DecrementInt(ctx context.Context, iv *IntValue) (*IntValue, error) {
	iv.Value *= -1
	return s.IncrementInt(ctx, iv)
}
