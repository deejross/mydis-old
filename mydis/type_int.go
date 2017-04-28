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

	"github.com/deejross/mydis/pb"
	"github.com/deejross/mydis/util"
	"github.com/gogo/protobuf/proto"
	"golang.org/x/net/context"
)

// GetInt gets an integer value for the given key.
func (s *Server) GetInt(ctx context.Context, key *pb.Key) (*pb.IntValue, error) {
	bv, err := s.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	iv := &pb.IntValue{}
	if err := proto.Unmarshal(bv.Value, iv); err != nil && strings.HasPrefix(err.Error(), "proto: can't skip unknown wire type") {
		return nil, util.ErrTypeMismatch
	} else if err != nil {
		return nil, err
	}
	return iv, nil
}

// SetInt sets an integer.
func (s *Server) SetInt(ctx context.Context, iv *pb.IntValue) (*pb.Null, error) {
	b, err := proto.Marshal(iv)
	if err != nil {
		return null, err
	}
	return s.Set(ctx, &pb.ByteValue{Key: iv.Key, Value: b})
}

// IncrementInt increments an integer stored at the given key by the number and returns the new value.
func (s *Server) IncrementInt(ctx context.Context, iv *pb.IntValue) (*pb.IntValue, error) {
	key := &pb.Key{Key: iv.Key}
	if _, err := s.Lock(ctx, key); err != nil {
		return nil, err
	}

	oldiv, err := s.GetInt(ctx, key)
	if err == util.ErrKeyNotFound {
		oldiv = &pb.IntValue{Value: 0}
	} else if err != nil {
		s.Unlock(ctx, key)
		return nil, err
	}

	newval := &pb.IntValue{Value: oldiv.Value + iv.Value}
	b, err := proto.Marshal(newval)
	if err != nil {
		s.Unlock(ctx, key)
		return nil, err
	}

	if _, err := s.UnlockThenSet(ctx, &pb.ByteValue{Key: iv.Key, Value: b}); err != nil {
		return nil, err
	}
	return newval, nil
}

// DecrementInt decrements an integer stored at the given key by the number and returns the new value.
func (s *Server) DecrementInt(ctx context.Context, iv *pb.IntValue) (*pb.IntValue, error) {
	iv.Value *= -1
	return s.IncrementInt(ctx, iv)
}
