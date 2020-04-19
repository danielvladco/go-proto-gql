.PHONY: install example

install:
	go install github.com/gogo/protobuf/protoc-gen-gogo
	protoc --gogo_out=paths=source_relative,Mgoogle/protobuf/descriptor.proto=github.com/gogo/protobuf/protoc-gen-gogo/descriptor:./pb \
	-I=${GOPATH}/pkg/mod/ -I=/usr/local/include -I=./pb ./pb/*.proto
	go install ./protoc-gen-gql
	go install ./protoc-gen-gogqlgen
	go install ./protoc-gen-gqlgencfg

example:
	protoc \
	--go_out=plugins=grpc,paths=source_relative:. \
	--gqlgencfg_out=paths=source_relative:. \
	--gql_out=svcdir=true,paths=source_relative:. \
	--gogqlgen_out=paths=source_relative,gogoimport=false:. \
	-I=. -I=/usr/local/include -I=./example/ ./example/*.proto
