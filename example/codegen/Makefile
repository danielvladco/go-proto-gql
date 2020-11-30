.PHONY: generate start

start: generate
	go run ./main.go

generate: gql/constructs/generated.go gql/options/generated.go

pb/%.gqlgen.pb.go: pb/%.proto pb/%_grpc.pb.go pb/%.pb.go
	protoc --gogql_out=paths=source_relative:. -I . -I ../../ ./pb/$*.proto

pb/%_grpc.pb.go: pb/%.proto pb/%.pb.go
	protoc --go-grpc_out=paths=source_relative:. -I . -I ../../ ./pb/$*.proto

pb/%.pb.go: pb/%.proto
	protoc --go_out=paths=source_relative:. -I . -I ../../ ./pb/$*.proto

pb/%.graphqls: pb/%.proto
	protoc --gql_out=svc=true:. -I . -I ../../ ./pb/$*.proto

gql/%/generated.go: pb/%.graphqls pb/%.gqlgen.pb.go
	gqlgen --config ./gqlgen-$*.yaml

