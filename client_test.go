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

func TestClientSet(t *testing.T) {
	testReset()

	if err := client.Set("key2", "val2"); err != nil {
		t.Error(err)
	}
	if s, err := client.Get("key2").String(); err != nil {
		t.Error(err)
	} else if s != "val2" {
		t.Error("Unexpected value:", s)
	}
}

func TestClientGet(t *testing.T) {
	if s, err := client.Get("key2").String(); err != nil {
		t.Error(err)
	} else if s != "val2" {
		t.Error("Unexpected value:", s)
	}
}

func TestClientKeys(t *testing.T) {
	if keys, err := client.Keys(); err != nil {
		t.Error(err)
	} else if len(keys) != 2 || keys[0] != "key1" {
		t.Error("Unexpected value:", keys)
	}
}

func TestClientKeysWithPrefix(t *testing.T) {
	if keys, err := client.KeysWithPrefix("key"); err != nil {
		t.Error(err)
	} else if len(keys) != 2 || keys[1] != "key2" {
		t.Error("Unexpected value", keys)
	}
}

func TestClientHas(t *testing.T) {
	if b, err := client.Has("key1"); err != nil {
		t.Error(err)
	} else if b == false {
		t.Error("Expected key does not exist")
	}
}

func TestClientSetExpire(t *testing.T) {
	if err := client.SetExpire("key1", 1); err != nil {
		t.Error(err)
	}
	if b, err := client.Has("key1"); err != nil {
		t.Error(err)
	} else if b == false {
		t.Error("Expected key not found")
	}

	t.Log("INFO: Waiting two seconds for key expiration")
	time.Sleep(2000 * time.Millisecond)
	if b, err := client.Has("key1"); err != nil {
		t.Error(err)
	} else if b {
		t.Error("Unexpected key found")
		s, err := client.Get("key1").String()
		t.Log("Key value:", s, err)
	}
}

func TestClientLock(t *testing.T) {
	testReset()

	if err := client.Lock("key1"); err != nil {
		t.Error(err)
	}

	t.Log("INFO: This test will take about 2 seconds to complete")
	if err := client.LockWithTimeout("key1", 1); err != ErrKeyLocked {
		t.Error("Unexpected or no error:", err.Error())
	}

	if err := client.Set("key1", "val1"); err != ErrKeyLocked {
		t.Error("Unexpected or no error:", err.Error())
	}

	if err := client.Unlock("key1"); err != nil {
		t.Error(err)
	}

	if err := client.Set("key1", "val1"); err != nil {
		t.Error(err)
	}
}

func TestClientDelete(t *testing.T) {
	if err := client.Delete("key1"); err != nil {
		t.Error(err)
	}

	if s, err := client.Get("key1").String(); err != ErrKeyNotFound {
		t.Error("Unexpected value:", s, err)
	}
}

func TestClientClear(t *testing.T) {
	if err := client.Clear(); err != nil {
		t.Error(err)
	}
	if keys, err := client.Keys(); err != nil {
		t.Error(err)
	} else if len(keys) != 0 {
		t.Error("Unexpected value:", keys)
	}
}

func TestClientSetMany(t *testing.T) {
	if m, err := client.SetMany(map[string]Value{
		"key2": NewValue("val2"),
		"key3": NewValue("val3"),
	}); err != nil {
		t.Error(err)
	} else if len(m) != 0 {
		t.Error("Unexpected errors:", m)
	}

	if s, err := client.Get("key3").String(); err != nil {
		t.Error(err)
	} else if s != "val3" {
		t.Error("Unexpected value:", s)
	}
}

func TestClientGetMany(t *testing.T) {
	if m, err := client.GetMany([]string{"key2", "key3"}); err != nil {
		t.Error(err)
	} else if len(m) != 2 {
		t.Error("Unexpected value:", m)
	}
}

func TestClientGetWithPrefix(t *testing.T) {
	if m, err := client.GetWithPrefix("key"); err != nil {
		t.Error(err)
	} else if len(m) != 2 {
		t.Error("Unexpected value:", m)
	}
}

func TestClientLength(t *testing.T) {
	if i, err := client.Length("key2"); err != nil {
		t.Error(err)
	} else if i != 4 {
		t.Error("Unexpected value:", i)
	}
}

func TestClientGetSetInt(t *testing.T) {
	if err := client.Set("int1", 5); err != nil {
		t.Error(err)
	}
	if i, err := client.Get("int1").Int(); err != nil {
		t.Error(err)
	} else if i != 5 {
		t.Error("Unexpected value:", i)
	}
}

func TestClientGetSetFloat(t *testing.T) {
	if err := client.Set("float1", 5.5); err != nil {
		t.Error(err)
	}
	if f, err := client.Get("float1").Float(); err != nil {
		t.Error(err)
	} else if f != 5.5 {
		t.Error("Unexpected value:", f)
	}
}

func TestClientIncrementInt(t *testing.T) {
	if i, err := client.IncrementInt("int1", 5); err != nil {
		t.Error(err)
	} else if i != 10 {
		t.Error("Unexpected value:", i)
	}
}

func TestClientDecrementInt(t *testing.T) {
	if i, err := client.DecrementInt("int1", 5); err != nil {
		t.Error(err)
	} else if i != 5 {
		t.Error("Unexpected value:", i)
	}
}

func TestClientIncrementFloat(t *testing.T) {
	if f, err := client.IncrementFloat("float1", 5.1); err != nil {
		t.Error(err)
	} else if f != 10.6 {
		t.Error("Unexpected value:", f)
	}
}

func TestClientDecrementFloat(t *testing.T) {
	if f, err := client.DecrementFloat("float1", 5.1); err != nil {
		t.Error(err)
	} else if f != 5.5 {
		t.Error("Unexpected value:", f)
	}
}

func TestClientGetSetList(t *testing.T) {
	if err := client.Set("list1", []string{"item1", "item2", "item3"}); err != nil {
		t.Error(err)
	}
	if lst, err := client.Get("list1").List(); err != nil {
		t.Error(err)
	} else if len(lst) != 3 {
		t.Error("Unexpected value:", lst)
	}
}

func TestClientGetListItem(t *testing.T) {
	if s, err := client.GetListItem("list1", 1).String(); err != nil {
		t.Error(err)
	} else if s != "item2" {
		t.Error("Unexpected value:", s)
	}
}

func TestClientGetSetListItem(t *testing.T) {
	if err := client.SetListItem("list1", 0, "val1"); err != nil {
		t.Error(err)
	}
	if s, err := client.GetListItem("list1", 0).String(); err != nil {
		t.Error(err)
	} else if s != "val1" {
		t.Error("Unexpected value:", s)
	}

	if _, err := client.GetListItem("list1", 10).String(); err != nil {
		t.Error("Unexpected error", err)
	}
}

func TestClientListLength(t *testing.T) {
	if length, err := client.ListLength("list1"); err != nil {
		t.Error(err)
	} else if length != 3 {
		t.Error("Unexpected value:", length)
	}
}

func TestClientListLimit(t *testing.T) {
	if err := client.ListLimit("list1", 3); err != nil {
		t.Error(err)
	}
	if err := client.ListAppend("list1", "item4"); err != nil {
		t.Error(err)
	}

	if length, err := client.ListLength("list1"); err != nil {
		t.Error(err)
	} else if length != 3 {
		t.Error("Unexpected length:", length)
	}

	if s, err := client.GetListItem("list1", 0).String(); err != nil {
		t.Error(err)
	} else if s != "item2" {
		t.Error("Unexpected value:", s)
	}
}

func TestClientListInsert(t *testing.T) {
	if err := client.ListInsert("list1", 0, "newval"); err != nil {
		t.Error(err)
	}
	if s, err := client.GetListItem("list1", 0).String(); err != nil {
		t.Error(err)
	} else if s != "item2" {
		t.Error("Unexpected value:", s)
	}

	if err := client.ListInsert("list1", 1, "newval"); err != nil {
		t.Error(err)
	}
	if s, err := client.GetListItem("list1", 0).String(); err != nil {
		t.Error(err)
	} else if s != "newval" {
		t.Error("Unexpected value:", s)
	}
}

func TestClientListAppend(t *testing.T) {
	if err := client.ListAppend("list1", "end"); err != nil {
		t.Error(err)
	}
	if s, err := client.GetListItem("list1", -1).String(); err != nil {
		t.Error(err)
	} else if s != "end" {
		t.Error("Unexpected value:", s)
	}
}

func TestClientPopLeft(t *testing.T) {
	if s, err := client.ListPopLeft("list1").String(); err != nil {
		t.Error(err)
	} else if s != "item3" {
		t.Error("Unexpected value:", s)
	}

	if i, err := client.ListLength("list1"); err != nil {
		t.Error(err)
	} else if i != 2 {
		t.Error("Unexpected value:", i)
	}
}

func TestClientListPopLeftBlock(t *testing.T) {
	go func() {
		time.Sleep(100 * time.Millisecond)
		if err := client.ListAppend("leftBlock", "test"); err != nil {
			t.Error(err)
		}
	}()

	if s, err := client.ListPopLeftBlock("leftBlock", 2).String(); err != nil {
		t.Error(err)
	} else if s != "test" {
		t.Error("Unexpected value:", s)
	}
}

func TestClientListPopRight(t *testing.T) {
	if s, err := client.ListPopRight("list1").String(); err != nil {
		t.Error(err)
	} else if s != "end" {
		t.Error("Unexpected value:", s)
	}

	if i, err := client.ListLength("list1"); err != nil {
		t.Error(err)
	} else if i != 1 {
		t.Error("Unexpected value:", i)
	}
}

func TestClientListPopRightBlock(t *testing.T) {
	go func() {
		time.Sleep(100 * time.Millisecond)
		if err := client.ListAppend("leftBlock", "test"); err != nil {
			t.Error(err)
		}
	}()

	if s, err := client.ListPopRightBlock("leftBlock", 2).String(); err != nil {
		t.Error(err)
	} else if s != "test" {
		t.Error("Unexpected value:", s)
	}
}

func TestClientListHas(t *testing.T) {
	if err := client.ListAppend("list1", "someval"); err != nil {
		t.Error(err)
	}
	if i, err := client.ListHas("list1", "someval"); err != nil {
		t.Error(err)
	} else if i != 1 {
		t.Error("Unexpected value:", i)
	}
}

func TestClientListDelete(t *testing.T) {
	if err := client.ListDelete("list1", 0); err != nil {
		t.Error(err)
	}
	if i, err := client.ListLength("list1"); err != nil {
		t.Error(err)
	} else if i != 1 {
		t.Error("Unexpected value:", i)
	}
	if _, err := client.ListPopLeft("list1").String(); err != nil {
		t.Error(err)
	}
}

func TestClientListDeleteItem(t *testing.T) {
	if err := client.ListAppend("list1", "newItem"); err != nil {
		t.Error(err)
	}
	if i, err := client.ListLength("list1"); err != nil {
		t.Error(err)
	} else if i != 1 {
		t.Error("Unexpected value:", i)
	}
	if i, err := client.ListDeleteItem("list1", "newItem"); err != nil {
		t.Error(err)
	} else if i != 0 {
		t.Error("Unexpected value:", i)
	}
}

func TestClientGetSetHash(t *testing.T) {
	if err := client.Set("hash1", map[string]string{
		"key1": "val1",
		"key2": "val2",
		"key3": "val3",
	}); err != nil {
		t.Error(err)
	}

	if h, err := client.Get("hash1").Map(); err != nil {
		t.Error(err)
	} else if len(h) != 3 {
		t.Error("Unexpected value:", h)
	}
}

func TestClientGetHashField(t *testing.T) {
	if s, err := client.GetHashField("hash1", "key2").String(); err != nil {
		t.Error(err)
	} else if s != "val2" {
		t.Error("Unexpected value:", s)
	}

	if _, err := client.GetHashField("hash1", "nofield").String(); err != ErrHashFieldNotFound {
		t.Error("Unexpected error:", err)
	}
}

func TestClientGetHashFields(t *testing.T) {
	if h, err := client.GetHashFields("hash1", []string{"key3", "nofield"}); err != nil {
		t.Error(err)
	} else if len(h) != 1 {
		t.Error("Unexpected value:", h)
	}
}

func TestClientHashHas(t *testing.T) {
	if b, err := client.HashHas("hash1", "key2"); err != nil {
		t.Error(err)
	} else if !b {
		t.Error("Expected field not found")
	}

	if b, err := client.HashHas("hash1", "nofield"); err != nil {
		t.Error(err)
	} else if b {
		t.Error("Unexpected field found")
	}
}

func TestClientHashLength(t *testing.T) {
	if i, err := client.HashLength("hash1"); err != nil {
		t.Error(err)
	} else if i != 3 {
		t.Error("Unexpected value:", i)
	}

	if _, err := client.HashLength("none"); err != ErrKeyNotFound {
		t.Error("Unexpected error:", err)
	}
}

func TestClientHashFields(t *testing.T) {
	if keys, err := client.HashFields("hash1"); err != nil {
		t.Error(err)
	} else if len(keys) != 3 || keys[0] != "key1" {
		t.Error("Unexpected value:", keys)
	}
}

func TestClientHashValues(t *testing.T) {
	if vals, err := client.HashValues("hash1"); err != nil {
		t.Error(err)
	} else if len(vals) != 3 {
		t.Error("Unexpected value:", vals)
	}
}

func TestClientProto(t *testing.T) {
	if err := client.Set("proto1", &pb.Event{Current: &pb.ByteValue{Key: "test"}}); err != nil {
		t.Error(err)
	}

	ev := &pb.Event{}
	if err := client.Get("proto1").Proto(ev); err != nil {
		t.Error(err)
	} else if ev.Current.Key != "test" {
		t.Error("Unexpected value:", ev)
	}
}

func TestClientAuthEnable(t *testing.T) {
	client.UserAdd("root", "root")
	client.RoleAdd("root")
	client.UserGrantRole("root", "root")
	if err := client.AuthEnable(); err != nil {
		t.Error(err)
	}
}

func TestClientRoleAdd(t *testing.T) {
	client.Authenticate("root", "root")

	// create reader role and user
	readerPerms := GetPermission("key*", "READ")
	if err := client.RoleAdd("reader"); err != nil {
		t.Error(err)
	}
	if err := client.RoleGrantPermission("reader", readerPerms); err != nil {
		t.Error(err)
	}
	client.UserAdd("reader", "reader")
	if err := client.UserGrantRole("reader", "reader"); err != nil {
		t.Error(err)
	}

	// create writer role and user
	writerPerms := GetPermission("key*", "READWRITE")
	if err := client.RoleAdd("writer"); err != nil {
		t.Error(err)
	}
	if err := client.RoleGrantPermission("writer", writerPerms); err != nil {
		t.Error(err)
	}
	client.UserAdd("writer", "writer")
	if err := client.UserGrantRole("writer", "writer"); err != nil {
		t.Error(err)
	}
}

func TestClientAuthAccess(t *testing.T) {
	// test writer
	client.Authenticate("writer", "writer")
	if err := client.Set("key5", "val5"); err != nil {
		t.Error(err)
	}
	if err := client.Get("key5").Error(); err != nil {
		t.Error(err)
	}
	if err := client.Set("something else", "val"); err == nil {
		t.Error("Expected an auth error")
	}

	// test reader
	client.Authenticate("reader", "reader")
	if err := client.Get("key5").Error(); err != nil {
		t.Error(err)
	}
	if err := client.Get("something else").Error(); err == nil {
		t.Error("Expected an auth error")
	}
	if err := client.Set("key5", "val"); err == nil {
		t.Error("Expected an auth error")
	}
}

func TestClientAuthDisable(t *testing.T) {
	client.Authenticate("root", "root")

	client.UserDelete("reader")
	client.UserDelete("writer")
	client.RoleDelete("reader")
	client.RoleDelete("writer")
	if err := client.AuthDisable(); err != nil {
		t.Error(err)
	}
	client.UserDelete("root")
	client.RoleDelete("root")
	client.LogOut()
}
