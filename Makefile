.PHONY: install test

install:
	protoc --gogo_out=paths=source_relative,Mgoogle/protobuf/descriptor.proto=github.com/gogo/protobuf/protoc-gen-gogo/descriptor:. \
    	-I=${GOPATH}/pkg/mod/ -I=. ./gql.proto
	go install ./protoc-gen-gogql
	go install ./protoc-gen-gogqlenum

example: protogen gqlgen server

protogen:
	protoc --go_out=\
	Mgithub.com/mwitkow/go-proto-validators@v0.0.0-20180403085117-0950a7990007/validator.proto=github.com/mwitkow/go-proto-validators,\
	plugins=grpc,paths=source_relative:. \
	--gogql_out=paths=source_relative,gqlgen=./examples/gqlgen.yml:. \
	--gogqlenum_out=\
	Mgithub.com/mwitkow/go-proto-validators@v0.0.0-20180403085117-0950a7990007/validator.proto=github.com/mwitkow/go-proto-validators,\
	paths=source_relative,\
	gogoimport=false:. \
	-I=${GOPATH}/pkg/mod/ -I=. -I=./examples/account/ \
	./examples/account/*.proto

gqlgen:
	go run ./examples/cmd/main.go -c ./examples/gqlgen.yml

server:
	go run ./examples/main.go
