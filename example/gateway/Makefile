.PHONY: generate start stop

generate: gen-pb-options gen-grpc-options gen-pb-constructs gen-grpc-constructs

gen-grpc-%:
	protoc --go-grpc_out=paths=source_relative:. -I . -I ../../api ./$*server/pb/$*.proto

gen-pb-%:
	protoc --go_out=paths=source_relative:. -I . -I ../../api ./$*server/pb/$*.proto

clean: clean-options clean-constructs

clean-%:
	rm -f ./$*server/pb/$*_grpc.pb.go && rm -f ./$*server/pb/$*.pb.go