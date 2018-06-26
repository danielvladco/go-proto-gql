Go protoc plugin generating graphql schema and graphql resolvers code

The repository aims to simplify working with grpc trough protocol buffers and graphql by generating code.
Document here about how to use the generated code: https://github.com/vektah/gqlgen

Given plugins generates boilerplate code that is needed to connect graphql resolvers with grpc services.
If you use micro-service architecture with grpc and with graphql api gateway for frontend you will find yourself
repeating a lot of code for translating from one transport layer to another (which many times may be a source of bugs)

Install:
-
    go get github.com/danielvladco/go-proto-gql
    cd ${GOPATH}github.com/danielvladco/go-proto-gql/protoc-gen-gogql && go install
    cd ${GOPATH}github.com/danielvladco/go-proto-gql/protoc-gen-gogqlenum && go install

Usage Examples:
-
The protoc compiler expects to find plugins named proto-gen-<PLUGIN_NAME> on the execution $PATH. So first:

    export PATH=${PATH}:${GOPATH}/bin

Add --gogqlenum_out=. plugin to your protoc code generation comamnd
run to generate enum marshal/unmarshal interfaces

NOTE: genetate code with --gogo_out not --go_out

    protoc --gogqlenum_out=. \
        --gogo_out=. \
        --proto_path=${GOPATH}/src/github.com/gogo/protobuf/protobuf \
        --proto_path=${GOPATH}/src/ \
        --proto_path=. \
        account.proto
    
Import all the proto files for witch you want to generate gql handlers and use call the command like this

    protoc --gogql_out=. \
    --proto_path=${GOPATH}/src/github.com/gogo/protobuf/protobuf \
    --proto_path=${GOPATH}/src/ \
    --proto_path=. \
    test.proto

--gogqlenum_out and --gogql_out triggers the protoc-gen-gogqlenum and protoc-gen-gogql plugins to generate
required files. This is all the magic here.

See examples folder.
For more documentation inspect code ;)

dependencies: 

    github.com/vektah/gqlgen 
    github.com/mwitkow/go-proto-validators
    github.com/gogo/protobuf
    
TODO:
-
- fix known bugs
- comments in code and refactor
- tests
- gql unions (oneof)
- gql interfaces (?)
- gql directives (maybe with proto options)
- gql input for type field (maybe with proto options)

BUGS:
-
- generates output code to the location where you are calling command but not in the out directory as expected 
(as workaround use this command in the path you want to generate code)

###License

go-proto-gql is released under the Apache 2.0 license. See the LICENSE file for details.