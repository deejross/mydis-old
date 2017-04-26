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
	"context"
	"log"
	"net/url"
	"os"
	"testing"

	"strings"

	"sync"

	"github.com/coreos/etcd/embed"
	myc "github.com/deejross/mydis/client"
	"github.com/deejross/mydis/pb"
	"google.golang.org/grpc/metadata"
)

var server *Server
var peer1 *Server
var peer2 *Server
var peer3 *Server
var client *myc.Client
var ctx = context.Background()

func TestMain(m *testing.M) {
	log.Println("Setting up for testing...")

	// define single server
	serverStore := "testing.etcd"

	// define cluster
	peer1Store := "testing.peer1.etcd"
	peer1Name := "peer1"
	peer1URL, _ := url.Parse("https://127.0.0.1:2381")
	peer2Store := "testing.peer2.etcd"
	peer2Name := "peer2"
	peer2URL, _ := url.Parse("https://127.0.0.1:2382")
	peer3Store := "testing.peer3.etcd"
	peer3Name := "peer3"
	peer3URL, _ := url.Parse("https://127.0.0.1:2383")
	clusterName := strings.Join([]string{
		peer1Name + "=" + peer1URL.String(),
		peer2Name + "=" + peer2URL.String(),
		peer3Name + "=" + peer3URL.String(),
	}, ",")

	// setup single server
	serverConfig := embed.NewConfig()
	serverConfig.LCUrls = []url.URL{}
	serverConfig.ACUrls = []url.URL{}
	serverConfig.ClientAutoTLS = true
	// set TickMs and ElectionMs very low for lease expiration testing.
	serverConfig.TickMs = 2
	serverConfig.ElectionMs = 10
	serverConfig.Dir = serverStore
	server = NewServer(serverConfig)

	// setup default peer config
	defaultPeerConfig := embed.NewConfig()
	defaultPeerConfig.ACUrls = []url.URL{}
	defaultPeerConfig.LCUrls = []url.URL{}
	defaultPeerConfig.ClientAutoTLS = true
	defaultPeerConfig.PeerAutoTLS = true
	defaultPeerConfig.InitialClusterToken = "unit-test-cluster"
	defaultPeerConfig.InitialCluster = clusterName

	// setup the first node in the cluster
	peer1Config := CopyConfig(defaultPeerConfig)
	peer1Config.Dir = peer1Store
	peer1Config.APUrls = []url.URL{*peer1URL}
	peer1Config.LPUrls = []url.URL{*peer1URL}
	peer1Config.Name = peer1Name
	peer1 = NewServer(peer1Config)

	// setup the second node in the cluster.
	peer2Config := CopyConfig(defaultPeerConfig)
	peer2Config.Dir = peer2Store
	peer2Config.APUrls = []url.URL{*peer2URL}
	peer2Config.LPUrls = []url.URL{*peer2URL}
	peer2Config.Name = peer2Name
	peer2 = NewServer(peer2Config)

	// setup the third node in the cluster.
	peer3Config := CopyConfig(defaultPeerConfig)
	peer3Config.Dir = peer3Store
	peer3Config.APUrls = []url.URL{*peer3URL}
	peer3Config.LPUrls = []url.URL{*peer3URL}
	peer3Config.Name = peer3Name
	peer3 = NewServer(peer3Config)

	// cleanup in case last test got canceled or paniced
	os.RemoveAll(serverStore)
	os.RemoveAll(peer1Store)
	os.RemoveAll(peer2Store)
	os.RemoveAll(peer3Store)

	// create context for direct tests
	md := metadata.New(map[string]string{
		"maxlockwait": "1",
	})
	ctx = metadata.NewContext(ctx, md)

	// start the single server
	if err := server.Start(":8000", ":8383"); err != nil {
		log.Println(err)
		os.Exit(1)
	}

	// start the cluster
	wg := sync.WaitGroup{}
	go func() {
		wg.Add(1)
		defer wg.Done()

		if err := peer1.Start(":8001", ":8384"); err != nil {
			log.Println(err)
			os.Exit(1)
		}
	}()

	go func() {
		wg.Add(1)
		defer wg.Done()

		if err := peer2.Start(":8002", ":8385"); err != nil {
			log.Println(err)
			os.Exit(1)
		}
	}()
	go func() {
		wg.Add(1)
		defer wg.Done()

		if err := peer3.Start(":8003", ":8386"); err != nil {
			log.Println(err)
			os.Exit(1)
		}
	}()
	wg.Wait()

	var err error
	client, err = myc.NewClient(myc.NewClientConfig("localhost:8383"))
	client.SetLockTimeout(1)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	log.Println("Running tests...")
	code := m.Run()
	log.Println("Testing complete, shutting down servers and cleaning up...")

	client.Close()
	server.Close()
	peer1.Close()
	peer2.Close()
	peer3.Close()

	// cleanup
	os.RemoveAll(serverStore)
	os.RemoveAll(peer1Store)
	os.RemoveAll(peer2Store)
	os.RemoveAll(peer3Store)

	log.Println("Cleanup complete, exiting")

	os.Exit(code)
}

func TestCluster(t *testing.T) {
	var client1, client2, client3 *myc.Client
	var err error
	client1, err = myc.NewClient(myc.NewClientConfig("localhost:8384"))
	client1.SetLockTimeout(1)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	client2, err = myc.NewClient(myc.NewClientConfig("localhost:8385"))
	client2.SetLockTimeout(1)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	client3, err = myc.NewClient(myc.NewClientConfig("localhost:8386"))
	client3.SetLockTimeout(1)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	peer1.Set(ctx, &pb.ByteValue{Key: "key1", Value: []byte("val1")})

	if s, err := client1.Get("key1").String(); err != nil {
		t.Error(err)
	} else if s != "val1" {
		t.Error("Unexpected value:", s)
	}

	if s, err := client2.Get("key1").String(); err != nil {
		t.Error(err)
	} else if s != "val1" {
		t.Error("Unexpected value:", s)
	}

	if s, err := client3.Get("key1").String(); err != nil {
		t.Error(err)
	} else if s != "val1" {
		t.Error("Unexpected value:", s)
	}

	if err := client2.Set("key2", "val2"); err != nil {
		t.Error(err)
	}

	if s, err := client1.Get("key2").String(); err != nil {
		t.Error(err)
	} else if s != "val2" {
		t.Error("Unexpected value:", s)
	}

	if s, err := client2.Get("key2").String(); err != nil {
		t.Error(err)
	} else if s != "val2" {
		t.Error("Unexpected value:", s)
	}

	if s, err := client3.Get("key2").String(); err != nil {
		t.Error(err)
	} else if s != "val2" {
		t.Error("Unexpected value:", s)
	}

	client1.Close()
	client2.Close()
	client3.Close()
}

func testReset() {
	server.Clear(ctx, null)
	testAddKey1()
}

func testAddKey1() {
	server.Set(ctx, &pb.ByteValue{Key: "key1", Value: []byte("val1")})
}
