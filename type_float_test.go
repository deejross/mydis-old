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

func TestSetFloat(t *testing.T) {
	if _, err := server.SetFloat(ctx, &FloatValue{Key: "float1", Value: 42.5}); err != nil {
		t.Error(err)
	}
}

func TestGetFloat(t *testing.T) {
	if fv, err := server.GetFloat(ctx, &Key{Key: "float1"}); err != nil {
		t.Error(err)
	} else if fv.Value != 42.5 {
		t.Error("Unexpected value:", fv.Value)
	}

	if _, err := server.GetFloat(ctx, &Key{Key: "key1"}); err == nil {
		t.Error("Expected error but got nothing")
	}
}

func TestIncrementFloat(t *testing.T) {
	testReset()

	if _, err := server.SetFloat(ctx, &FloatValue{Key: "float1", Value: 42.5}); err != nil {
		t.Error(err)
	}

	if fv, err := server.IncrementFloat(ctx, &FloatValue{Key: "float1", Value: 10.1}); err != nil {
		t.Error(err)
	} else if fv.Value != 52.6 {
		t.Error("Unexpected value:", fv.Value)
	}
	if fv, err := server.GetFloat(ctx, &Key{Key: "float1"}); err != nil {
		t.Error(err)
	} else if fv.Value != 52.6 {
		t.Error("New value not committed")
	}

	if fv, err := server.IncrementFloat(ctx, &FloatValue{Key: "newFloat", Value: 42.5}); err != nil {
		t.Error(err)
	} else if fv.Value != 42.5 {
		t.Error("Unexpected value:", fv.Value)
	}

	if fv, err := server.GetFloat(ctx, &Key{Key: "newFloat"}); err != nil {
		t.Error(err)
	} else if fv.Value != 42.5 {
		t.Error("New value not committed, got:", fv.Value)
	}
}

func TestDecrementFloat(t *testing.T) {
	testReset()

	if _, err := server.SetFloat(ctx, &FloatValue{Key: "float1", Value: 42.5}); err != nil {
		t.Error(err)
	}

	if fv, err := server.DecrementFloat(ctx, &FloatValue{Key: "float1", Value: 10.1}); err != nil {
		t.Error(err)
	} else if fv.Value != 32.4 {
		t.Error("Unexpected value:", fv.Value)
	}
	if fv, err := server.GetFloat(ctx, &Key{Key: "float1"}); err != nil {
		t.Error(err)
	} else if fv.Value != 32.4 {
		t.Error("New value not committed")
	}

	if fv, err := server.DecrementFloat(ctx, &FloatValue{Key: "newFloat", Value: 42.5}); err != nil {
		t.Error(err)
	} else if fv.Value != -42.5 {
		t.Error("Unexpected value:", fv.Value)
	}

	if fv, err := server.GetFloat(ctx, &Key{Key: "newFloat"}); err != nil {
		t.Error(err)
	} else if fv.Value != -42.5 {
		t.Error("New value not committed, got:", fv.Value)
	}
}
