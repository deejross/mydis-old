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

import "testing"

func TestSetInt(t *testing.T) {
	if _, err := server.SetInt(ctx, &IntValue{Key: "int1", Value: 42}); err != nil {
		t.Error(err)
	}
}

func TestGetInt(t *testing.T) {
	if iv, err := server.GetInt(ctx, &Key{Key: "int1"}); err != nil {
		t.Error(err)
	} else if iv.Value != 42 {
		t.Error("Unexpected value:", iv.Value)
	}

	if _, err := server.GetInt(ctx, &Key{Key: "key1"}); err == nil {
		t.Error("Expected error but got nothing")
	}
}

func TestIncrementInt(t *testing.T) {
	testReset()

	if _, err := server.SetInt(ctx, &IntValue{Key: "int1", Value: 42}); err != nil {
		t.Error(err)
	}

	if iv, err := server.IncrementInt(ctx, &IntValue{Key: "int1", Value: 10}); err != nil {
		t.Error(err)
	} else if iv.Value != 52 {
		t.Error("Unexpected value:", iv.Value)
	}
	if iv, err := server.GetInt(ctx, &Key{Key: "int1"}); err != nil {
		t.Error(err)
	} else if iv.Value != 52 {
		t.Error("New value not commited")
	}

	if iv, err := server.IncrementInt(ctx, &IntValue{Key: "newInt", Value: 42}); err != nil {
		t.Error(err)
	} else if iv.Value != 42 {
		t.Error("Unexpected value:", iv.Value)
	}

	if iv, err := server.GetInt(ctx, &Key{Key: "newInt"}); err != nil {
		t.Error(err)
	} else if iv.Value != 42 {
		t.Error("New value not commited, got:", iv.Value)
	}
}

func TestDecrementInt(t *testing.T) {
	testReset()

	if _, err := server.SetInt(ctx, &IntValue{Key: "int1", Value: 42}); err != nil {
		t.Error(err)
	}

	if iv, err := server.DecrementInt(ctx, &IntValue{Key: "int1", Value: 10}); err != nil {
		t.Error(err)
	} else if iv.Value != 32 {
		t.Error("Unexpected value:", iv.Value)
	}
	if iv, err := server.GetInt(ctx, &Key{Key: "int1"}); err != nil {
		t.Error(err)
	} else if iv.Value != 32 {
		t.Error("New value not commited")
	}

	if iv, err := server.DecrementInt(ctx, &IntValue{Key: "newInt", Value: 42}); err != nil {
		t.Error(err)
	} else if iv.Value != -42 {
		t.Error("Unexpected value:", iv.Value)
	}

	if iv, err := server.GetInt(ctx, &Key{Key: "newInt"}); err != nil {
		t.Error(err)
	} else if iv.Value != -42 {
		t.Error("New value not commited, got:", iv.Value)
	}
}
