.PHONY: build

build:
	protoc --gogql_out=. \
		--proto_path=${GOPATH}/src/github.com/gogo/protobuf/protobuf \
		--proto_path=${GOPATH}/src/ \
		--proto_path=. \
		test.proto