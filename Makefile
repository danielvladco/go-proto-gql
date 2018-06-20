.PHONY: clean install test

install:
	cd ./protoc-gen-gogql && go install

test:
	protoc --gogql_out=. --proto_path=${GOPATH}/src/github.com/gogo/protobuf/protobuf \
		--proto_path=${GOPATH}/src/ --proto_path=. test.proto

clean:
	rm test.gql.pb.go