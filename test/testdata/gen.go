package pb

//go:generate ${GOPATH}/bin/protoc -I . -I ../../  --go_out=paths=source_relative:. constructs-input.proto
//go:generate ${GOPATH}/bin/protoc -I . -I ../../  --go-grpc_out=paths=source_relative:. constructs-input.proto
//go:generate ${GOPATH}/bin/protoc -I . -I ../../  --go_out=paths=source_relative:. options-input.proto
//go:generate ${GOPATH}/bin/protoc -I . -I ../../  --go-grpc_out=paths=source_relative:. options-input.proto
