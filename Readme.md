Protoc plugins for generating graphql schema

If you use micro-service architecture with grpc for backend and graphql api gateway for frontend you will find yourself
repeating a lot of code for translating from one transport layer to another (which many times may be a source of bugs)

This repository aims to simplify working with grpc trough protocol buffers and graphql by generating code.

There are 2 plugins: `gql` that generates graphql schema 
and `gogqlgen` that generates methods for implementing
`github.com/99designs/gqlgen/graphql.Marshaler` and `github.com/99designs/gqlgen/graphql.Unmarshaler` interfaces. 

Install:
-

```sh
go get github.com/danielvladco/go-proto-gql
go install github.com/danielvladco/go-proto-gql/protoc-gen-gql
go install github.com/danielvladco/go-proto-gql/protoc-gen-gogqlgen
```

Usage Examples:
-
The protoc compiler expects to find plugins named `proto-gen-<PLUGIN_NAME>` on the execution `$PATH`. So first:

```sh
export PATH=${PATH}:${GOPATH}/bin
```

This script will generate graphql files with extension .graphqls 
rather than go code which means it can be further used for any other language or framework.
It uses https://github.com/mwitkow/go-proto-validators annotations do determine if the field is required or not. (see ./example/options.proto)

Example: 
```sh
protoc --gql_out=paths=source_relative:. \
	-I=${GOPATH}/pkg/mod/ -I=. -I=./examples/account/ \
	./examples/account/*.proto
```

If you still want to generate go source code instead of graphql then use 
http://github.com/99designs/gqlgen plugin.

The plugin will work if you did not use enums. If you want enums then use the next plugin. 

Add `--gogqlgen_out` to your protoc code generation command
run to generate enum marshal/unmarshal interfaces

NOTE: to generate with gogo import add `gogoimport=true` as a parameter

Example:
```sh
protoc --gogqlgen_out=gogoimport=false:. \
	-I=${GOPATH}/pkg/mod/ -I=. -I=./examples/account/ \
	./examples/account/*.proto
``` 

See example folder for more examples of proto input and generated output.

For more documentation inspect code ;)
   
TODO:
-
- There are some edge-cases where the generated gql will be invalid (ex. do not use this names as fields: reset, string, protoMessage, descriptor). They are pretty rare and can be ignored 
- Add comments from proto to gql for documentation purposes

## Community:
I am one on this. Will be very glad for any contributions so feel free to create issues and forks.

## License:

`go-proto-gql` is released under the Apache 2.0 license. See the LICENSE file for details.
