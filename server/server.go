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

package main

import (
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/coreos/etcd/embed"
	"github.com/deejross/mydis/mydis"
)

var (
	defaultConfigFile   = "/etc/mydis/mydis.conf"
	defaultAddressHTTP1 = "0.0.0.0:8000"
	defaultAddressHTTP2 = "0.0.0.0:8383"
)

func main() {
	cfg := loadConfig()
	server := mydis.NewServer(cfg)
	err := server.Start(getAddressHTTP1(), getAddressHTTP2())
	if err != nil {
		fmt.Println(err)
		log.Fatalln(err)
	}

	select {} // block forever
}

func getAddressHTTP1() string {
	if port := os.Getenv("PORT"); port != "" {
		return "0.0.0.0:" + port
	}
	return defaultAddressHTTP1
}

func getAddressHTTP2() string {
	if address := os.Getenv("MYDIS_ADDRESS"); address != "" {
		return address
	}
	return defaultAddressHTTP2
}

func loadConfig() *embed.Config {
	args := os.Args
	if len(args) > 1 {
		if cfg, err := embed.ConfigFromFile(args[1]); err != nil {
			return cfg
		}
	}

	if cfg, err := embed.ConfigFromFile(defaultConfigFile); err == nil {
		return cfg
	}
	if cfg, err := embed.ConfigFromFile("mydis.conf"); err == nil {
		return cfg
	}

	log.Println("Unable to open config file, using default settings")
	return NewServerConfig()
}

// NewServerConfig returns a new server Config object with defaults set.
func NewServerConfig() *embed.Config {
	cfg := embed.NewConfig()
	cfg.LCUrls = []url.URL{}
	cfg.LPUrls = []url.URL{}
	cfg.Dir = "default.etcd"
	cfg.ClientAutoTLS = true
	return cfg
}
