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

	"github.com/deejross/mydis/pb"
)

func TestLock(t *testing.T) {
	testReset()

	if _, err := server.Lock(ctx, &pb.Key{Key: "key1"}); err != nil {
		t.Error(err)
	}

	t.Log("INFO: This test will take about 2 seconds to complete")
	if _, err := server.LockWithTimeout(ctx, &pb.Expiration{Key: "key1", Exp: 1}); err != ErrKeyLocked {
		t.Error("Unexpected or no error:", err)
	}

	if _, err := server.Set(ctx, &pb.ByteValue{Key: "key1", Value: []byte("val1")}); err != ErrKeyLocked {
		t.Error("Unexpected or no error:", err)
	}

	if _, err := server.Unlock(ctx, &pb.Key{Key: "key1"}); err != nil {
		t.Error(err)
	}

	if _, err := server.Set(ctx, &pb.ByteValue{Key: "key1", Value: []byte("val1")}); err != nil {
		t.Error(err)
	}
}
