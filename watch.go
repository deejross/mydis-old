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
	"strings"
	"sync"

	"github.com/coreos/etcd/etcdserver"
	"github.com/coreos/etcd/mvcc"
	"github.com/deejross/mydis/pb"
)

// WatchController object.
type WatchController struct {
	closeCh  chan struct{}
	server   *etcdserver.EtcdServer
	watchers map[int64]*Watcher
	nextID   int64
	lock     sync.RWMutex
}

// NewWatchController returns a new WatchController object.
func NewWatchController(server *etcdserver.EtcdServer) *WatchController {
	wc := &WatchController{
		closeCh:  make(chan struct{}),
		server:   server,
		watchers: map[int64]*Watcher{},
	}
	return wc
}

// NewWatcher returns a new Watcher object.
func (w *WatchController) NewWatcher() *Watcher {
	watcher := &Watcher{
		closeCh:    make(chan struct{}),
		reqCh:      make(chan *pb.WatchRequest),
		resCh:      make(chan struct{}),
		eventCh:    make(chan *pb.Event),
		controller: w,
		watching:   map[string]*pb.WatchRequest{},
	}

	w.lock.Lock()
	w.watchers[w.nextID] = watcher

	go func(id int64) {
		watcher.backgroundProcess()
		w.lock.Lock()
		delete(w.watchers, id)
		w.lock.Unlock()
	}(w.nextID)

	w.nextID++
	w.lock.Unlock()

	return watcher
}

// Close the WatchController.
func (w *WatchController) Close() {
	watchers := []*Watcher{}

	// get list of watchers then loop through and Close to avoid deadlock.
	w.lock.RLock()
	for _, watcher := range w.watchers {
		watchers = append(watchers, watcher)
	}
	w.lock.RUnlock()

	for _, watcher := range watchers {
		watcher.Close()
	}
}

// Watcher object.
type Watcher struct {
	controller *WatchController
	closeCh    chan struct{}
	reqCh      chan *pb.WatchRequest
	resCh      chan struct{}
	eventCh    chan *pb.Event
	server     *etcdserver.EtcdServer
	watching   map[string]*pb.WatchRequest
}

// RequestID gets the string ID from the WatchRequest.
func (w *Watcher) RequestID(r *pb.WatchRequest) string {
	id := r.Key
	if r.Prefix {
		id += suffixForKeysUsingPrefix
	}
	return id
}

// Close the Watcher.
func (w *Watcher) Close() {
	w.closeCh <- struct{}{}
}

func (w *Watcher) backgroundProcess() {
	stream := w.controller.server.Watchable().NewWatchStream()
	streamCh := stream.Chan()

	defer func() {
		stream.Close()
		close(w.closeCh)
		close(w.reqCh)
		close(w.resCh)
		close(w.eventCh)
	}()

	for {
		select {
		case <-w.closeCh:
			return
		case r := <-w.reqCh:
			hash := w.RequestID(r)

			if wr, ok := w.watching[hash]; ok && r.Cancel {
				// process cancelation.
				stream.Cancel(mvcc.WatchID(wr.Id))
				delete(w.watching, hash)

				w.resCh <- struct{}{}
				continue
			} else if ok {
				// prevent duplcates.
				w.resCh <- struct{}{}
				continue
			}

			bkey := StringToBytes(r.Key)
			end := []byte{}
			if r.Prefix {
				end = getPrefix(r.Key)
			}

			r.Id = int64(stream.Watch(bkey, end, r.Rev))
			w.watching[hash] = r
			w.resCh <- struct{}{}
		case r := <-streamCh:
			for _, e := range r.Events {
				ev := &pb.Event{
					Type:     pb.Event_EventType(e.Type),
					Current:  &pb.ByteValue{},
					Previous: &pb.ByteValue{},
				}
				if e.Kv != nil {
					ev.Current = &pb.ByteValue{Key: BytesToString(e.Kv.Key), Value: e.Kv.Value}
				}
				if e.PrevKv != nil {
					ev.Previous = &pb.ByteValue{Key: BytesToString(e.PrevKv.Key), Value: e.PrevKv.Value}
				}

				for _, r := range w.watching {
					key := BytesToString(e.Kv.Key)
					if r.Key == key || (r.Prefix && strings.HasPrefix(key, r.Key)) {
						w.eventCh <- ev
					}
				}
			}
		}
	}
}

// Watch a key for changes.
func (s *Server) Watch(stream pb.Mydis_WatchServer) error {
	watcher := s.wc.NewWatcher()
	defer watcher.Close()

	// sender
	go func() {
		for {
			ev := <-watcher.eventCh
			stream.Send(ev)
		}
	}()

	// receiver
	for {
		r, err := stream.Recv()
		if err != nil {
			return err
		}

		watcher.reqCh <- r
		<-watcher.resCh
	}
}
