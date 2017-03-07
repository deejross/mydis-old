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
	"errors"
	"fmt"
	"log"
	"strings"

	"google.golang.org/grpc"

	"net"

	"time"

	"github.com/coreos/etcd/embed"
	"github.com/coreos/etcd/pkg/transport"
	"github.com/coreos/pkg/capnslog"
	"google.golang.org/grpc/credentials"
)

const (
	delay = time.Duration(1) * time.Millisecond
)

// ErrKeyLocked menas that the key cannot be modified, as it's locked by another process.
var ErrKeyLocked = errors.New("Key is locked")

// Server object.
type Server struct {
	config *embed.Config
	cache  *embed.Etcd
	socket net.Listener
	server *grpc.Server
	wc     *WatchController
}

// NewServer returns a new Server object.
func NewServer(config *embed.Config) *Server {
	if !config.Debug {
		capnslog.SetGlobalLogLevel(capnslog.ERROR)
	}

	s := &Server{
		config: config,
	}
	return s
}

// Start the server.
func (s *Server) Start(address string) error {
	e, err := embed.StartEtcd(s.config)
	if err != nil {
		return err
	}
	<-e.Server.ReadyNotify()
	s.cache = e
	s.wc = NewWatchController(s.cache.Server)

	socket, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	s.socket = socket

	if err := s.applyTLS(); err != nil {
		return err
	}

	RegisterMydisServer(s.server, s)
	fmt.Println("Server listening at", address)

	go func() {
		err = s.server.Serve(s.socket)
		if strings.HasSuffix(err.Error(), "use of closed network connection") {
			return
		}
		log.Println(err)
	}()

	return nil
}

// Close the server.
func (s *Server) Close() {
	s.server.GracefulStop()
	s.cache.Close()
}

func (s *Server) applyTLS() error {
	tlsInfo, err := generateTLSInfo(s.config)
	if err != nil {
		return err
	}

	if tlsInfo.Empty() {
		s.server = grpc.NewServer()
	} else {
		tlsConfig, err := tlsInfo.ServerConfig()
		if err != nil {
			return err
		}

		tlsCreds := transport.ShallowCopyTLSConfig(tlsConfig)
		tlsCreds.InsecureSkipVerify = true
		creds := grpc.Creds(credentials.NewTLS(tlsCreds))
		s.server = grpc.NewServer(creds)
	}

	return nil
}
