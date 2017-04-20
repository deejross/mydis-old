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
	"testing"
	"time"

	"github.com/coreos/etcd/etcdserver"
	"github.com/deejross/mydis/pb"
)

func TestSet(t *testing.T) {
	testReset()

	if _, err := server.Set(ctx, &pb.ByteValue{Key: "key1", Value: []byte("val1")}); err != nil {
		t.Error(err)
	}
}

func TestSetNX(t *testing.T) {
	testReset()

	if b, err := server.SetNX(ctx, &pb.ByteValue{Key: "key1", Value: []byte("val1")}); err != nil {
		t.Error(err)
	} else if b.Value {
		t.Error("Should not have set value as it already exists")
	}

	if b, err := server.SetNX(ctx, &pb.ByteValue{Key: "key2", Value: []byte("val2")}); err != nil {
		t.Error(err)
	} else if !b.Value {
		t.Error("Should have set value as it doesn't already exist")
	}
}

func TestSetMany(t *testing.T) {
	testReset()

	if errors, err := server.SetMany(ctx, &pb.Hash{Value: map[string][]byte{
		"key2": []byte("val2"),
		"key3": []byte("val3"),
	}}); err != nil {
		t.Error(err)
	} else if len(errors.Errors) > 0 {
		t.Error(errors.Errors)
	}
}

func TestGet(t *testing.T) {
	if bv, err := server.Get(ctx, &pb.Key{Key: "key1"}); err != nil {
		t.Error(err)
	} else if !bytes.Equal(bv.Value, []byte("val1")) {
		t.Error("Unexpected value:", bv.Value)
	}
}

func TestGetMany(t *testing.T) {
	if h, err := server.GetMany(ctx, &pb.KeysList{Keys: []string{"key1", "key2"}}); err != nil {
		t.Error(err)
	} else if len(h.Value) != 2 {
		t.Error("Unexpected response:", h)
	} else if !bytes.Equal([]byte("val2"), h.Value["key2"]) {
		t.Error("Unexpected response:", h)
	}
}

func TestGetWithPrefix(t *testing.T) {
	if bh, err := server.GetWithPrefix(ctx, &pb.Key{Key: "key"}); err != nil {
		t.Error(err)
	} else if len(bh.Value) < 3 {
		t.Error("Unexpected response:", bh.Value)
	}
}

func TestGetBlocking(t *testing.T) {
	if _, err := server.Get(ctx, &pb.Key{Key: "key1", Block: true, BlockTimeout: 1}); err != nil {
		t.Error(err)
	}

	go func() {
		time.Sleep(100 * time.Millisecond)
		if _, err := server.Set(ctx, &pb.ByteValue{Key: "keyBlock", Value: []byte("val")}); err != nil {
			t.Error(err)
		}
	}()
	if bv, err := server.Get(ctx, &pb.Key{Key: "keyBlock", Block: true, BlockTimeout: 1}); err != nil {
		t.Error(err)
	} else if len(bv.Value) != 3 {
		t.Error("Unexpected response:", bv.Value)
	}
}

func TestGetBlockingTimeout(t *testing.T) {
	server.Delete(ctx, &pb.Key{Key: "keyBlock"})
	if _, err := server.Get(ctx, &pb.Key{Key: "keyBlock", Block: true, BlockTimeout: 1}); err != etcdserver.ErrKeyNotFound {
		t.Error("Unexpected response:", err)
	}
}

func TestLength(t *testing.T) {
	if iv, err := server.Length(ctx, &pb.Key{Key: "key1"}); err != nil {
		t.Error(err)
	} else if iv.Value != 4 {
		t.Error("Unexpected value:", iv.Value)
	}
}
