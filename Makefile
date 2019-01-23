.PHONY: install test

install:
	protoc --gogo_out=paths=source_relative,Mgoogle/protobuf/descriptor.proto=github.com/gogo/protobuf/protoc-gen-gogo/descriptor:. \
    	-I=${GOPATH}/pkg/mod/ -I=. ./gql.proto
	go install ./protoc-gen-gogql
	go install ./protoc-gen-gogqlenum

test:
	protoc --go_out=paths=source_relative:. \
		--gogql_out=paths=source_relative,gqlgen=./examples/gqlgen.yml:. \
		--gogqlenum_out=paths=source_relative,gogoimport=false:. \
		-I=${GOPATH}/pkg/mod/ -I=. -I=./examples/account/ \
		./examples/account/*.proto

