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
)

func TestSetList(t *testing.T) {
	testReset()

	if _, err := server.SetList(ctx, &List{
		Key: "list1",
		Value: [][]byte{
			[]byte("val1"),
			[]byte("val2"),
			[]byte("val3"),
		},
	}); err != nil {
		t.Error(err)
	}
}

func TestGetList(t *testing.T) {
	if lst, err := server.GetList(ctx, &Key{Key: "list1"}); err != nil {
		t.Error(err)
	} else if len(lst.Value) != 3 {
		t.Error("Unexpected response:", lst.Value)
	} else if !bytes.Equal(lst.Value[2], []byte("val3")) {
		t.Error("Unexpected response:", lst.Value)
	}
}

func TestGetListItem(t *testing.T) {
	if bv, err := server.GetListItem(ctx, &ListItem{Key: "list1", Index: 1}); err != nil {
		t.Error(err)
	} else if !bytes.Equal(bv.Value, []byte("val2")) {
		t.Error("Unexpected value:", bv.Value)
	}

	if bv, err := server.GetListItem(ctx, &ListItem{Key: "list1", Index: 100}); err != nil {
		t.Error(err)
	} else if !bytes.Equal(bv.Value, []byte("val3")) {
		t.Error("Unexpected value:", bv.Value)
	}

	if bv, err := server.GetListItem(ctx, &ListItem{Key: "list1", Index: -1}); err != nil {
		t.Error(err)
	} else if !bytes.Equal(bv.Value, []byte("val3")) {
		t.Error("Unexpected value:", bv.Value)
	}

	if bv, err := server.GetListItem(ctx, &ListItem{Key: "list1", Index: -3}); err != nil {
		t.Error(err)
	} else if !bytes.Equal(bv.Value, []byte("val1")) {
		t.Error("Unexpected value:", bv.Value)
	}

	if bv, err := server.GetListItem(ctx, &ListItem{Key: "list1", Index: -100}); err != nil {
		t.Error(err)
	} else if !bytes.Equal(bv.Value, []byte("val1")) {
		t.Error("Unexpected value:", bv.Value)
	}
}

func TestSetListItem(t *testing.T) {
	if _, err := server.SetListItem(ctx, &ListItem{Key: "list1", Index: 2, Value: []byte("end")}); err != nil {
		t.Error(err)
	}
	if bv, err := server.GetListItem(ctx, &ListItem{Key: "list1", Index: 2}); err != nil {
		t.Error(err)
	} else if !bytes.Equal(bv.Value, []byte("end")) {
		t.Error("Unexpected value:", bv.Value)
	}
	if _, err := server.SetListItem(ctx, &ListItem{Key: "list1", Index: 10, Value: []byte("fail")}); err != ErrListIndexOutOfRange {
		t.Error("Expected ErrListIndexOutOfRange")
	}
}

func TestListLength(t *testing.T) {
	if iv, err := server.ListLength(ctx, &Key{Key: "list1"}); err != nil {
		t.Error(err)
	} else if iv.Value != 3 {
		t.Error("Unexpected value:", iv.Value)
	}
}

func TestListInsert(t *testing.T) {
	if _, err := server.ListInsert(ctx, &ListItem{Key: "list1", Index: 0, Value: []byte("start")}); err != nil {
		t.Error(err)
	}

	if lst, err := server.GetList(ctx, &Key{Key: "list1"}); err != nil {
		t.Error(err)
	} else if len(lst.Value) != 4 {
		t.Error("Unexpected response:", lst.Value)
	} else if !bytes.Equal(lst.Value[0], []byte("start")) {
		t.Error("Unexpected response:", lst.Value)
	}
}

func TestListAppend(t *testing.T) {
	if _, err := server.ListAppend(ctx, &ListItem{Key: "list1", Value: []byte("end again")}); err != nil {
		t.Error(err)
	}

	if lst, err := server.GetList(ctx, &Key{Key: "list1"}); err != nil {
		t.Error(err)
	} else if len(lst.Value) != 5 {
		t.Error("Unexpected response:", lst.Value)
	} else if !bytes.Equal(lst.Value[4], []byte("end again")) {
		t.Error("Unexpected response:", lst.Value)
	}
}

func TestListLimit(t *testing.T) {
	if _, err := server.ListLimit(ctx, &ListItem{Key: "list1", Index: 5}); err != nil {
		t.Error(err)
	}
	if _, err := server.ListAppend(ctx, &ListItem{Key: "list1", Value: []byte("end again2")}); err != nil {
		t.Error(err)
	}

	if lst, err := server.GetList(ctx, &Key{Key: "list1"}); err != nil {
		t.Error(err)
	} else if len(lst.Value) != 5 {
		t.Error("Unexpected response:", lst.Value)
	} else if !bytes.Equal(lst.Value[0], []byte("val1")) {
		t.Error("Unexpected response:", lst.Value)
	} else if !bytes.Equal(lst.Value[4], []byte("end again2")) {
		t.Error("Unexpected response:", lst.Value)
	}
}

func TestListPopLeft(t *testing.T) {
	if bv, err := server.ListPopLeft(ctx, &Key{Key: "list1"}); err != nil {
		t.Error(err)
	} else if !bytes.Equal(bv.Value, []byte("val1")) {
		t.Error("Unexpected value:", bv.Value)
	}
	if bv, err := server.GetListItem(ctx, &ListItem{Key: "list1", Index: 0}); err != nil {
		t.Error(err)
	} else if !bytes.Equal(bv.Value, []byte("val2")) {
		t.Error("Unexpected value:", bv.Value)
	}
}

func TestListPopRight(t *testing.T) {
	if bv, err := server.ListPopRight(ctx, &Key{Key: "list1"}); err != nil {
		t.Error(err)
	} else if !bytes.Equal(bv.Value, []byte("end again2")) {
		t.Error("Unexpected value:", bv.Value)
	}
	if bv, err := server.GetListItem(ctx, &ListItem{Key: "list1", Index: -1}); err != nil {
		t.Error(err)
	} else if !bytes.Equal(bv.Value, []byte("end again")) {
		t.Error("Unexpected value:", bv.Value)
	}
}

func TestListHas(t *testing.T) {
	if iv, err := server.ListHas(ctx, &ListItem{Key: "list1", Value: []byte("end")}); err != nil {
		t.Error(err)
	} else if iv.Value != 1 {
		t.Error("Unexpected value:", iv.Value)
	}

	if iv, err := server.ListHas(ctx, &ListItem{Key: "list1", Value: []byte("nothing")}); err != nil {
		t.Error(err)
	} else if iv.Value != -1 {
		t.Error("Unexpected value:", iv.Value)
	}

	if iv, err := server.ListHas(ctx, &ListItem{Key: "nothing", Value: []byte("nothing")}); err != nil {
		t.Error(err)
	} else if iv.Value != -1 {
		t.Error("Unexpected value:", iv.Value)
	}
}

func TestListDelete(t *testing.T) {
	count := int64(0)
	if iv, err := server.ListLength(ctx, &Key{Key: "list1"}); err != nil {
		t.Error(err)
	} else {
		count = iv.Value
	}

	if _, err := server.ListDelete(ctx, &ListItem{Key: "list1", Index: 0}); err != nil {
		t.Error(err)
	}
	if _, err := server.ListDelete(ctx, &ListItem{Key: "list1", Index: 10}); err != ErrListIndexOutOfRange {
		t.Error("Expected ErrListIndexOutOfRange")
	}

	if iv, err := server.ListLength(ctx, &Key{Key: "list1"}); err != nil {
		t.Error(err)
	} else if iv.Value != count-1 {
		t.Error("Unexpected value:", iv.Value)
	}
}

func TestListDeleteItem(t *testing.T) {
	if iv, err := server.ListDeleteItem(ctx, &ListItem{Key: "list1", Value: []byte("end")}); err != nil {
		t.Error(err)
	} else if iv.Value != 0 {
		t.Error("Unexpected value:", iv.Value)
	}

	if iv, err := server.ListHas(ctx, &ListItem{Key: "list1", Value: []byte("end")}); err != nil {
		t.Error(err)
	} else if iv.Value != -1 {
		t.Error("Unexpected value:", iv.Value)
	}

	if iv, err := server.ListDeleteItem(ctx, &ListItem{Key: "list1", Value: []byte("nothing")}); err != nil {
		t.Error(err)
	} else if iv.Value != -1 {
		t.Error("Unexpected value:", iv.Value)
	}

	if iv, err := server.ListDeleteItem(ctx, &ListItem{Key: "nothing", Value: []byte("nothing")}); err != nil {
		t.Error(err)
	} else if iv.Value != -1 {
		t.Error("Unexpected value:", iv.Value)
	}
}

func TestListPopLeftBlocking(t *testing.T) {
	if _, err := server.Delete(ctx, &Key{Key: "listBlock"}); err != nil {
		t.Error(err)
	}

	go func() {
		time.Sleep(100 * time.Millisecond)
		if _, err := server.ListAppend(ctx, &ListItem{Key: "listBlock", Value: []byte("test")}); err != nil {
			t.Error(err)
		}
	}()

	if bv, err := server.ListPopLeft(ctx, &Key{Key: "listBlock", Block: true, BlockTimeout: 1}); err != nil {
		t.Error(err)
	} else if len(bv.Value) != 4 {
		t.Error("Unexpected value:", bv.Value)
	}
}

func TestListPopRightBlocking(t *testing.T) {
	if _, err := server.Delete(ctx, &Key{Key: "listBlock"}); err != nil {
		t.Error(err)
	}

	go func() {
		time.Sleep(100 * time.Millisecond)
		if _, err := server.ListAppend(ctx, &ListItem{Key: "listBlock", Value: []byte("test")}); err != nil {
			t.Error(err)
		}
	}()

	if bv, err := server.ListPopRight(ctx, &Key{Key: "listBlock", Block: true, BlockTimeout: 1}); err != nil {
		t.Error(err)
	} else if len(bv.Value) != 4 {
		t.Error("Unexpected value:", bv.Value)
	}
}
