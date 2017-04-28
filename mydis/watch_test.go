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

	"github.com/deejross/mydis/pb"
)

func TestWatch(t *testing.T) {
	testReset()

	ch, id := client.NewEventChannel()
	client.Watch("watch1", false)
	time.Sleep(100 * time.Millisecond)

	if err := client.Set("watch1", "value"); err != nil {
		t.Error(err)
	}
	time.Sleep(100 * time.Millisecond)

	select {
	case ev := <-ch:
		if ev.Type != pb.Event_PUT || ev.Current.Key != "watch1" || !bytes.Equal(ev.Current.Value, []byte("value")) {
			t.Error("Unexpected event:", ev)
		}
	case <-time.After(1 * time.Second):
		t.Error("Never got event")
	}

	if err := client.Set("watch1", "value2"); err != nil {
		t.Error(err)
	}
	time.Sleep(100 * time.Millisecond)

	select {
	case ev := <-ch:
		if ev.Type != pb.Event_PUT || ev.Current.Key != "watch1" || !bytes.Equal(ev.Current.Value, []byte("value2")) {
			t.Error("Unexpected event:", ev)
		}
	case <-time.After(1 * time.Second):
		t.Error("Never got second event")
	}

	client.Unwatch("watch1", false)
	time.Sleep(100 * time.Millisecond)

	if err := client.Set("watch1", "value3"); err != nil {
		t.Error(err)
	}

	select {
	case ev := <-ch:
		t.Error("Unexpected event:", ev)
	case <-time.After(10 * time.Millisecond):
		// good
	}

	client.CloseEventChannel(id)
}
