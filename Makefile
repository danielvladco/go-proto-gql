# Go tools dependencies
GO_TOOLS := \
	google.golang.org/grpc/cmd/protoc-gen-go-grpc \
	google.golang.org/protobuf/cmd/protoc-gen-go \
	github.com/99designs/gqlgen \
	./protoc-gen-gql \
	./protoc-gen-gogql

GOPATH := $(shell go env GOPATH)

.PHONY: all
all: all-tools pb/graphql.pb.go

.PHONY: all-tools
all-tools: ${GOPATH}/bin/protoc go-tools

.PHONY: go-tools
go-tools: $(foreach l, ${GO_TOOLS}, ${GOPATH}/bin/$(notdir $(l)))

define LIST_RULE
${GOPATH}/bin/$(notdir $(1)): go.mod
	go install $(1)
endef

$(foreach l, $(GO_TOOLS), $(eval $(call LIST_RULE, $(l) )))

pb/graphql.pb.go: ./pb/graphql.proto all-tools
	protoc --go_out=paths=source_relative:. ./pb/graphql.proto

${GOPATH}/bin/protoc:
	./scripts/install-protoc.sh
