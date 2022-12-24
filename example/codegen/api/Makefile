.PHONY: generate
generate: gen-gqlgen-constructs gen-gqlgen-options

.PHONY: gen-gogql-%
gen-gogql-%:
	protoc --gogql_out=paths=source_relative:. -I ../../../api -I . ./pb/$*.proto --experimental_allow_proto3_optional

.PHONY: gen-pb-grpc-%
gen-pb-grpc-%:
	protoc --go-grpc_out=paths=source_relative:. -I ../../../api -I . ./pb/$*.proto --experimental_allow_proto3_optional

.PHONY: gen-pb-%
gen-pb-%:
	protoc --go_out=paths=source_relative:. -I ../../../api -I . ./pb/$*.proto --experimental_allow_proto3_optional

.PHONY: gen-gql-%
gen-gql-%:
	protoc --gql_out=svc=true:. -I ../../../api -I . ./pb/$*.proto --experimental_allow_proto3_optional

.PHONY: gen-gqlgen-%
gen-gqlgen-%: gen-gql-% gen-gogql-% gen-pb-% gen-pb-grpc-%
	gqlgen --config ./gqlgen-$*.yaml

.PHONY: clean-%
clean-%:
	rm -rf pb/$*.gqlgen.pb.go pb/$*_grpc.pb.go pb/$*.pb.go pb/$*.graphql graphql/$*/generated.go

.PHONY: clean
clean: clean-constructs clean-options
