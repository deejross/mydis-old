#!/bin/bash
VERSION=v$(go run cli/cli.go --version)
GOOS=darwin GOARCH=amd64 go build -o mydis-server-$VERSION-darwin-amd64 server/server.go
GOOS=linux GOARCH=amd64 go build -o mydis-server-$VERSION-linux-amd64 server/server.go
GOOS=windows GOARCH=amd64 go build -o mydis-server-$VERSION-windows-amd64.exe server/server.go

GOOS=darwin GOARCH=amd64 go build -o mydis-cli-$VERSION-darwin-amd64 cli/cli.go
GOOS=linux GOARCH=amd64 go build -o mydis-cli-$VERSION-linux-amd64 cli/cli.go
GOOS=windows GOARCH=amd64 go build -o mydis-cli-$VERSION-windows-amd64.exe cli/cli.go