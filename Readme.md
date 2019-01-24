Protoc plugins for generating graphql schema

If you use micro-service architecture with grpc for backend and graphql api gateway for frontend you will find yourself
repeating a lot of code for translating from one transport layer to another (which many times may be a source of bugs)

This repository aims to simplify working with grpc trough protocol buffers and graphql by generating code.

There are 2 plugins: `gogql` that generates graphql schema 
and `gogqlenum` that generates methods for implementing
`github.com/99designs/gqlgen/graphql.Marshaler` and `github.com/99designs/gqlgen/graphql.Unmarshaler` interfaces. 

Install:
-

```sh
go get github.com/danielvladco/go-proto-gql
go install github.com/danielvladco/go-proto-gql/protoc-gen-gogql
go install github.com/danielvladco/go-proto-gql/protoc-gen-gogqlenum
```

Usage Examples:
-
The protoc compiler expects to find plugins named `proto-gen-<PLUGIN_NAME>` on the execution `$PATH`. So first:

```sh
export PATH=${PATH}:${GOPATH}/bin
```

Import all the proto files for which you want to generate gql scheams.
Add `gqlgen=${PATH_TO_GQLGEN_YML}` parameter if you want to create a gqlgen configuration yaml file.   

NOTE: this script will generate graphql files with extension .graphqls 
rather than go code which means it can be further used for any other language or framework
Example: 
```sh
protoc --gogql_out=paths=source_relative,gqlgen=./examples/gqlgen.yml:. \
	-I=${GOPATH}/pkg/mod/ -I=. -I=./examples/account/ \
	./examples/account/*.proto
```

If you still want to generate go source code instead of graphql then use 
http://github.com/99designs/gqlgen plugin.

The plugin will work if you did not use enums. If you want enums then use the next plugin. 

Add `--gogqlenum_out` to your protoc code generation command
run to generate enum marshal/unmarshal interfaces

NOTE: to generate with gogo import add `gogoimport=true` as a parameter

Example:
```sh
protoc --go_out=Mgithub.com/mwitkow/go-proto-validators@v0.0.0-20180403085117-0950a7990007/validator.proto=github.com/mwitkow/go-proto-validators,plugins=grpc,paths=source_relative:. \
	--gogqlenum_out=Mgithub.com/mwitkow/go-proto-validators@v0.0.0-20180403085117-0950a7990007/validator.proto=github.com/mwitkow/go-proto-validators,paths=source_relative,gogoimport=false:. \
	-I=${GOPATH}/pkg/mod/ -I=. -I=./examples/account/ \
	./examples/account/*.proto
``` 

See examples folder.

For more documentation inspect code ;)
   
TODO:
-
- Add tests (needs to be tested all numbers: float, int, uint, double etc.)
- Add gql `union` from proto `oneof`
- Add gql interfaces (is it a good idea?)
- Implement maps as scalars with name containing key type and value type:
  
  e. i.
  ```proto
  message Foo { 
    map<string, string> bar = 1; 
    map<AccountID, Account> baz = 2; 
  }
  ```
  will be represented as: 
  ```graphql schema
  # map with key: "String" and value: "String"
  scalar Map_String_String
  # map with key: "AccountID" and value: "Account"
  scalar Map_AccountID_Account
  
  type Foo {
    bar: Map_String_String
    baz: Map_AccountID_Account
  }
  ```
- Add comments from proto to gql for documentation purposes
- Add an additional endpoint that will make requests and act like streaming from the client side
  
  i. e. A proto rpc like this...
  ```proto 
  service Service { rpc MessagesLoop(stream MessageReq) returns (stream MessageRes); }
  ```
  Will look in gql like this...
  ```graphql schema 
  type Query        { messagesLoopSend(in: MessageReq): Boolean! }
  type Subscription { messagesLoop: MessageRes }
  ```
## Community:
I am one on this. Will be very glad for any contributions so feel free to create issues and forks.

## License:

`go-proto-gql` is released under the Apache 2.0 license. See the LICENSE file for details.
