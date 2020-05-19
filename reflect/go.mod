module github.com/danielvladco/go-proto-gql/reflect

go 1.14

require (
	github.com/99designs/gqlgen v0.11.3
	github.com/danielvladco/go-proto-gql v0.7.3
	github.com/davecgh/go-spew v1.1.1
	github.com/gogo/protobuf v1.3.1
	github.com/golang/protobuf v1.4.2
	github.com/graphql-go/graphql v0.7.9
	github.com/jhump/protoreflect v1.6.1
	github.com/mitchellh/mapstructure v1.3.0
	github.com/nautilus/gateway v0.1.4
	github.com/nautilus/graphql v0.0.9
	github.com/sirupsen/logrus v1.6.0 // indirect
	github.com/vektah/gqlparser/v2 v2.0.1
	golang.org/x/net v0.0.0-20200513185701-a91f0712d120 // indirect
	golang.org/x/sys v0.0.0-20200515095857-1151b9dac4a9 // indirect
	golang.org/x/text v0.3.2 // indirect
	google.golang.org/genproto v0.0.0-20200514193133-8feb7f20f2a2 // indirect
	google.golang.org/grpc v1.29.1
	google.golang.org/protobuf v1.23.0
)

replace github.com/danielvladco/go-proto-gql v0.7.3 => ../
