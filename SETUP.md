Development Setup
=================

Introduction
------------
Mydis is built using many technologies such as Go, gRPC and others. To use these technologies, you'll need to setup a development environment. The instructions were written for macOS, but should be adaptable to other platforms. If you are able to successfully setup a development environment on another platform, please update this documentation with the steps required and submit a pull request.

Install Go
----------
To install Go and related tools on macOS, it is recommended to use [Homebrew](https://brew.sh/). You can find installation instructions on their website. Once it's installed, you can continue to follow these steps:
- `brew install make go glide protobuf go-delve/delve/delve`

If you already have these packages installed, please update them to the latest version: `brew upgrade make go glide protobuf go-delve/delve/delve`.

Next you'll need to setup a workspace. The first thing to do is setup your `$GOPATH`. Find a suitable location for the top-level folder for your Go projects (example: `~/Code/Go`). Create the $GOPATH environment variable in `.bash_profile`:
- `echo export GOPATH=~/Code/Go >> ~/.bash_profile` (you can replace `~/Code/Go` with a different path if you prefer)
- Create the `$GOPATH` directory, if it doesn't already exist
- Inside `$GOPATH`, create these subdirectories: `bin`, `pkg`, and `src`
- Add `$GOPATH/bin` to your `PATH`:
  - `echo export PATH=$PATH:$GOPATH/bin >> ~/.bash_profile`

**NOTE**: You will need to restart any existing Terminal sessions for these changes to take effect.

Install gRPC
------------
The Protocol Buffer compiler was installed in the previous section, so now we need to install the gRPC and gRPC Gateway packages along with their `protoc` compiler plugins:
```bash
go get -u github.com/golang/protobuf/{proto,protoc-gen-go}
go get -u google.golang.org/grpc
go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
```

When using `go get` on these packages, it will pull from the master branch and doing so will cause a lot of things to break since the latest versions of all three libraries are currently incompatible with one another. It is best to force the use of the version that Mydis currently uses. After running the `go get` commands above, run the following:
```bash
cd $GOPATH/src/google.golang.org/grpc
git checkout v1.0.4
cd $GOPATH/src/github.com/grpc-ecosystem/grpc-gateway
git checkout 84398b94e188ee336f307779b57b3aa91af7063c
cd $GOPATH/src/github.com/golang/protobuf
git checkout 4bd1920723d7b7c925de087aa32e2187708897f7
```

Clone This Repo
---------------
The recommended destination for this repo is `$GOPATH/src/github.com/deejross/mydis` you can clone using the Git CLI:
```bash
mkdir -p $GOPATH/src/github.com/deejross
cd $GOPATH/src/github.com/deejross
git clone https://github.com/deejross/mydis
```

Download Vendored Packages
--------------------------
You will need to download dependencies using `glide` within the `mydis` directory:
- `glide install`

**NOTE**: Some users may get some errors using the above command. If this happens, try running it again before submitting a new issue.

Protocol Buffer/gRPC Changes
----------------------------
After changing anything in the `mydis.proto` file, run `make` to generate new `mydis.pb.go` and `mydis.pb.gw.go` files.

Done
----
Go develop! Don't forget to submit a pull request when you're ready.
