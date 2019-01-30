.PHONY: install example

install:
	protoc --gogo_out=paths=source_relative,Mgoogle/protobuf/descriptor.proto=github.com/gogo/protobuf/protoc-gen-gogo/descriptor:. \
    	-I=${GOPATH}/pkg/mod/ -I=. ./gql.proto
	go install ./protoc-gen-gql
	go install ./protoc-gen-gogqlgen
	go install ./protoc-gen-gqlgencfg

example:
	protoc --go_out=\
	Mgithub.com/mwitkow/go-proto-validators@v0.0.0-20180403085117-0950a7990007/validator.proto=github.com/mwitkow/go-proto-validators,\
	plugins=grpc,paths=source_relative:. \
	--gqlgencfg_out=paths=source_relative:. \
	--gql_out=paths=source_relative:. \
	--gogqlgen_out=\
	Mgithub.com/mwitkow/go-proto-validators@v0.0.0-20180403085117-0950a7990007/validator.proto=github.com/mwitkow/go-proto-validators,\
	paths=source_relative,\
	gogoimport=false:. \
	-I=${GOPATH}/pkg/mod/ -I=. -I=./example/ \
	./example/*.proto
