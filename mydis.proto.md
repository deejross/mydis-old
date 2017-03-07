Protocol Buffers
----------------

Install the `protoc` compiler:
```bash
brew install protobuf
```

Install the Go plugin for the compiler:
```bash
go get -u github.com/golang/protobuf/protoc-gen-go
```

Add the Go/bin directory to the PATH by adding this line to `~/.bash_profile` after `$GOPATH` is defined:
```bash
export PATH=$PATH:$GOPATH/bin
```

Copy and paste the `export` line into any current Terminal sessions, or open a new Terminal session for it to take effect.

In the app directory, compile the protocol for the cache:
```bash
protoc --go_out=plugins=grpc:. mydis.proto
```

This generates the `mydis.pb.go` file.
