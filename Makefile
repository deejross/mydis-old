.PHONY: all

all:
	@protoc -I/usr/local/include -Ipb \
		-I${GOPATH}/src \
		-I${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--go_out=plugins=grpc:pb \
		pb/mydis.proto
	@protoc -I/usr/local/include -Ipb \
		-I${GOPATH}/src \
		-I${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--grpc-gateway_out=logtostderr=true:pb \
		pb/mydis.proto
	@go generate .