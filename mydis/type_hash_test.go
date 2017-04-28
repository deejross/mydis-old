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

	"github.com/deejross/mydis/pb"
)

func TestSetHash(t *testing.T) {
	testReset()

	if _, err := server.SetHash(ctx, &pb.Hash{Key: "hash1", Value: map[string][]byte{
		"f1": []byte("val1"),
		"f2": []byte("val2"),
		"f3": []byte("val3"),
		"f4": []byte("val4"),
	}}); err != nil {
		t.Error(err)
	}
}

func TestGetHash(t *testing.T) {
	if h, err := server.GetHash(ctx, &pb.Key{Key: "hash1"}); err != nil {
		t.Error(err)
	} else if len(h.Value) != 4 || !bytes.Equal(h.Value["f1"], []byte("val1")) {
		t.Error("Unexpected value:", h.Value)
	}
}

func TestGetHashField(t *testing.T) {
	if b, err := server.GetHashField(ctx, &pb.HashField{Key: "hash1", Field: "f2"}); err != nil {
		t.Error(err)
	} else if !bytes.Equal(b.Value, []byte("val2")) {
		t.Error("Unexpected value:", b.Value)
	}
}

func TestGetHashFields(t *testing.T) {
	if h, err := server.GetHashFields(ctx, &pb.HashFieldSet{Key: "hash1", Field: []string{"f3", "f4"}}); err != nil {
		t.Error(err)
	} else if len(h.Value) != 2 || !bytes.Equal(h.Value["f4"], []byte("val4")) {
		t.Error("Unexpected value:", h.Value)
	}
}

func TestHashHas(t *testing.T) {
	if b, err := server.HashHas(ctx, &pb.HashField{Key: "hash1", Field: "f1"}); err != nil {
		t.Error(err)
	} else if !b.Value {
		t.Error("Unexpected value:", b.Value)
	}
}

func TestHashLength(t *testing.T) {
	if iv, err := server.HashLength(ctx, &pb.Key{Key: "hash1"}); err != nil {
		t.Error(err)
	} else if iv.Value != 4 {
		t.Error("Unexpected value:", iv.Value)
	}
}

func TestHashFields(t *testing.T) {
	if keys, err := server.HashFields(ctx, &pb.Key{Key: "hash1"}); err != nil {
		t.Error(err)
	} else if len(keys.Keys) != 4 || keys.Keys[0] != "f1" {
		t.Error("Unexpected value:", keys.Keys)
	}
}

func TestHashValues(t *testing.T) {
	if lst, err := server.HashValues(ctx, &pb.Key{Key: "hash1"}); err != nil {
		t.Error(err)
	} else if len(lst.Value) != 4 || !bytes.Equal(lst.Value[0], []byte("val1")) {
		t.Error("Unexpected value:", lst.Value)
	}
}

func TestSetHashField(t *testing.T) {
	if _, err := server.SetHashField(ctx, &pb.HashField{Key: "hash1", Field: "f5", Value: []byte("val5")}); err != nil {
		t.Error(err)
	}
	if b, err := server.GetHashField(ctx, &pb.HashField{Key: "hash1", Field: "f5"}); err != nil {
		t.Error(err)
	} else if !bytes.Equal(b.Value, []byte("val5")) {
		t.Error("Unexpected value:", b)
	}

	if _, err := server.SetHashField(ctx, &pb.HashField{Key: "hash2", Field: "k1", Value: []byte("v1")}); err != nil {
		t.Error(err)
	}
	if b, err := server.GetHashField(ctx, &pb.HashField{Key: "hash2", Field: "k1"}); err != nil {
		t.Error(err)
	} else if !bytes.Equal(b.Value, []byte("v1")) {
		t.Error("Unexpected value:", b)
	}
}

func TestSetHashFields(t *testing.T) {
	if _, err := server.SetHashFields(ctx, &pb.Hash{Key: "hash1", Value: map[string][]byte{
		"f5": []byte("val5 reset"),
		"f6": []byte("val6"),
	}}); err != nil {
		t.Error(err)
	}
	if h, err := server.GetHashFields(ctx, &pb.HashFieldSet{Key: "hash1", Field: []string{"f4", "f5", "f6"}}); err != nil {
		t.Error(err)
	} else if !bytes.Equal(h.Value["f4"], []byte("val4")) || !bytes.Equal(h.Value["f5"], []byte("val5 reset")) {
		t.Error("Unexpected value:", h.Value)
	}
}

func TestDelHashField(t *testing.T) {
	if _, err := server.DelHashField(ctx, &pb.HashField{Key: "hash1", Field: "f6"}); err != nil {
		t.Error(err)
	}
	if h, err := server.GetHash(ctx, &pb.Key{Key: "hash1"}); err != nil {
		t.Error(err)
	} else if len(h.Value) != 5 || h.Value["f6"] != nil {
		t.Error("Unexpected value:", h.Value)
	}
}
