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
	"testing"
	"time"

	"github.com/deejross/mydis/pb"
)

func TestKeys(t *testing.T) {
	testReset()

	if lst, err := server.Keys(ctx, null); err != nil {
		t.Error(err)
	} else if len(lst.Keys) == 0 {
		t.Error("No keys returned")
	} else if lst.Keys[0] != "key1" {
		t.Error("Unexpected key:", lst.Keys[0])
	}
}

func TestKeysWithPrefix(t *testing.T) {
	testReset()

	if lst, err := server.KeysWithPrefix(ctx, &pb.Key{Key: "key"}); err != nil {
		t.Error(err)
	} else if len(lst.Keys) == 0 {
		t.Error("No keys returned")
	} else if lst.Keys[0] != "key1" {
		t.Error("Unexpected key:", lst.Keys[0])
	}
}

func TestHas(t *testing.T) {
	testReset()

	if b, err := server.Has(ctx, &pb.Key{Key: "key1"}); err != nil {
		t.Error(err)
	} else if b.Value == false {
		t.Error("Expected key not found")
	}
}

func TestSetExpire(t *testing.T) {
	testReset()

	if _, err := server.SetExpire(ctx, &pb.Expiration{Key: "key1", Exp: 1}); err != nil {
		t.Error(err)
	}
	if b, err := server.Has(ctx, &pb.Key{Key: "key1"}); err != nil {
		t.Error(err)
	} else if b.Value == false {
		t.Error("Expected key not found")
	}

	t.Log("INFO: Waiting two seconds for key expiration")
	time.Sleep(2000 * time.Millisecond)
	if b, err := server.Has(ctx, &pb.Key{Key: "key1"}); err != nil {
		t.Error(err)
	} else if b.Value {
		t.Error("Unexpected key found")
		s, err := server.Get(ctx, &pb.Key{Key: "key1"})
		t.Log("Key value:", s.Value, err)
	}
}

func TestDelete(t *testing.T) {
	testReset()

	if _, err := server.Delete(ctx, &pb.Key{Key: "key1"}); err != nil {
		t.Error(err)
	}
	if b, err := server.Has(ctx, &pb.Key{Key: "key1"}); err != nil {
		t.Error(err)
	} else if b.Value {
		t.Error("Unexpected key found")
	}
}

func TestClear(t *testing.T) {
	testReset()

	if _, err := server.Clear(ctx, null); err != nil {
		t.Error(err)
	}
	if lst, err := server.Keys(ctx, null); err != nil {
		t.Error(err)
	} else if len(lst.Keys) > 0 {
		t.Error("Expected empty cache")
	}
}
