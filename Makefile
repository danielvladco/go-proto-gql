.PHONY: install example

install:
	go install github.com/golang/protobuf/protoc-gen-go
	protoc --go_out=paths=source_relative:./pb -I=./pb ./pb/*.proto
	go install ./protoc-gen-gql
	go install ./protoc-gen-gogqlgen
	go install ./protoc-gen-gqlgencfg

example:
	protoc \
	--go_out=plugins=grpc,paths=source_relative:. \
	--gqlgencfg_out=paths=source_relative:. \
	--gql_out=svcdir=true,paths=source_relative:. \
	--gogqlgen_out=paths=source_relative,gogoimport=false:. \
	-I=. -I=./example/ ./example/*.proto
