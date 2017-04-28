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
	"strconv"
	"testing"

	"github.com/deejross/mydis/pb"
)

func BenchmarkSet(b *testing.B) {
	for i := 0; i < b.N; i++ {
		si := strconv.Itoa(i)
		server.Set(ctx, &pb.ByteValue{
			Key:   "key" + si,
			Value: []byte("val" + si),
		})
	}
}

func BenchmarkGet(b *testing.B) {
	for i := 0; i < b.N; i++ {
		server.Get(ctx, &pb.Key{Key: "key" + strconv.Itoa(i)})
	}
}

func BenchmarkSetMany(b *testing.B) {
	vals := &pb.Hash{Value: map[string][]byte{
		"key1": []byte("val1"),
		"key2": []byte("val2"),
		"key3": []byte("val3"),
		"key4": []byte("val4"),
		"key5": []byte("val5"),
	}}
	for i := 0; i < b.N; i++ {
		server.SetMany(ctx, vals)
	}
}

func BenchmarkGetMany(b *testing.B) {
	keys := &pb.KeysList{Keys: []string{"key1", "key2", "key3", "key4", "key5"}}
	for i := 0; i < b.N; i++ {
		server.GetMany(ctx, keys)
	}
}
