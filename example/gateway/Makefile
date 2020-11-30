.PHONY: generate start stop

start: generate
	docker-compose up

stop:
	docker-compose down

generate: optionsserver/pb/options_grpc.pb.go constructserver/pb/constructs_grpc.pb.go

%_grpc.pb.go: %.proto %.pb.go
	protoc --go-grpc_out=paths=source_relative:. -I . -I ../../ $*.proto

%.pb.go: %.proto
	protoc --go_out=paths=source_relative:. -I . -I ../../ $*.proto
