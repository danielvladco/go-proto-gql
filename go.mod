module github.com/danielvladco/go-proto-gql

go 1.14

require (
	github.com/danielvladco/go-proto-gql/pb v0.6.0
	github.com/davecgh/go-spew v1.1.1
	github.com/golang/protobuf v1.4.2
	github.com/graphql-go/graphql v0.7.8
	github.com/graphql-go/handler v0.2.3
	github.com/jhump/goprotoc v0.2.0 // indirect
	github.com/jhump/protoreflect v1.5.0
	github.com/lyft/protoc-gen-star v0.4.15 // indirect
	github.com/spf13/afero v1.2.2 // indirect
	github.com/vektah/gqlparser/v2 v2.0.1
	golang.org/x/net v0.0.0-20190813141303-74dc4d7220e7 // indirect
	golang.org/x/sys v0.0.0-20190813064441-fde4db37ae7a // indirect
	golang.org/x/text v0.3.2 // indirect
	google.golang.org/genproto v0.0.0-20190801165951-fa694d86fc64 // indirect
	google.golang.org/grpc v1.23.0
	google.golang.org/protobuf v1.23.0
	gopkg.in/yaml.v2 v2.2.4
)

replace github.com/danielvladco/go-proto-gql/pb => ./pb
