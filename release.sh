#!/bin/bash
GOOS=darwin GOARCH=amd64 go build -o mydis-server-darwin-amd64 server/server.go
GOOS=linux GOARCH=amd64 go build -o mydis-server-linux-amd64 server/server.go
GOOS=windows GOARCH=amd64 go build -o mydis-server-windows-amd64.exe server/server.go

GOOS=darwin GOARCH=amd64 go build -o mydis-cli-darwin-amd64 cli/cli.go
GOOS=linux GOARCH=amd64 go build -o mydis-cli-linux-amd64 cli/cli.go
GOOS=windows GOARCH=amd64 go build -o mydis-cli-windows-amd64.exe cli/cli.go