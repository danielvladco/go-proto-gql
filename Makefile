.PHONY: install test

install:
	protoc --gogo_out=Mgoogle/protobuf/descriptor.proto=github.com/gogo/protobuf/protoc-gen-gogo/descriptor:. \
        		--proto_path=${GOPATH}/src/ \
        		--proto_path=${GOPATH}/src/github.com/gogo/protobuf/protobuf/ \
        		--proto_path=. \
        		gql.proto
	cd ./protoc-gen-gogql && go install
	cd ./protoc-gen-gogqlenum && go install

test:
	protoc --gogo_out=./examples/account/ \
		--gogqlenum_out=./examples/account/ \
		--proto_path=${GOPATH}/src/github.com/gogo/protobuf/protobuf \
		--proto_path=${GOPATH}/src/ \
		--proto_path=./examples/account/ \
		./examples/account/account.proto
