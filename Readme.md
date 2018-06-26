protoc plugin for generating gql handlers 


Install:
-
    go get github.com/danielvladco/go-proto-gql
    cd ${GOPATH}github.com/danielvladco/go-proto-gql/protoc-gen-gogql && go install
    cd ${GOPATH}github.com/danielvladco/go-proto-gql/protoc-gen-gogqlenum && go install

Examples:
-
install and add --gogqlenum_out=. plugin to your protoc code generation comand
run to generate enum marshal/unmarshal interfaces
NOTE: genetate code with --gogo_out not --go_out

    protoc --gogqlenum_out=. --gogo_out=. --proto_path=${GOPATH}/src/github.com/gogo/protobuf/protobuf --proto_path=${GOPATH}/src/ --proto_path=. account.proto 
    
import all the proto files for witch you want to generate gql handlers and use call the command like this

    protoc --gogql_out=. --proto_path=${GOPATH}/src/github.com/gogo/protobuf/protobuf --proto_path=${GOPATH}/src/ --proto_path=. test.proto


for more documentation inspect code ;)

dependencies: 

    github.com/vektah/gqlgen 
    github.com/mwitkow/go-proto-validators 
    
TODO:
-
- tests
- implement more gql features via proto options

BUGS:
-
- generates output code to the location where you are calling command but not in the out directory as expected 
(as workaround use this command in the path you want to generate code)
