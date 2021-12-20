module github.com/danielvladco/go-proto-gql

go 1.16

require (
	github.com/99designs/gqlgen v0.13.0
	github.com/golang/protobuf v1.5.2
	github.com/jhump/protoreflect v1.8.2
	github.com/mitchellh/mapstructure v1.3.0
	github.com/nautilus/gateway v0.1.4
	github.com/nautilus/graphql v0.0.9
	github.com/vektah/gqlparser/v2 v2.1.0
	golang.org/x/net v0.0.0-20210410081132-afb366fc7cd1 // indirect
	golang.org/x/sys v0.0.0-20210403161142-5e06dd20ab57 // indirect
	golang.org/x/tools v0.1.0 // indirect
	google.golang.org/grpc v1.43.0
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.1.0
	google.golang.org/protobuf v1.27.1
)

replace github.com/99designs/gqlgen v0.13.0 => github.com/danielvladco/gqlgen v0.13.0-inputs
