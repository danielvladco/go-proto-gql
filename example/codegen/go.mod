module github.com/danielvladco/go-proto-gql/example/codegen

go 1.17

replace (
	github.com/danielvladco/go-proto-gql v0.9.0 => ../..
	github.com/danielvladco/go-proto-gql/example/codegen/api v0.0.0 => ./api
)

require (
	github.com/99designs/gqlgen v0.17.22
	github.com/danielvladco/go-proto-gql/example/codegen/api v0.0.0
)

require (
	github.com/agnivade/levenshtein v1.1.1 // indirect
	github.com/danielvladco/go-proto-gql v0.9.0 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/vektah/gqlparser/v2 v2.5.1 // indirect
	golang.org/x/net v0.0.0-20220722155237-a158d28d115b // indirect
	golang.org/x/sys v0.0.0-20220811171246-fbc7d0a398ab // indirect
	golang.org/x/text v0.4.0 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
	google.golang.org/grpc v1.51.0 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
)
