package pb

//go:generate bash -c "protoc --go_out=paths=source_relative,plugins=grpc:../../../../ ./*.proto -I=../../../../pb -I=."
