# Go tools dependencies
GO_TOOLS := \
	google.golang.org/grpc/cmd/protoc-gen-go-grpc \
	google.golang.org/protobuf/cmd/protoc-gen-go \
	github.com/99designs/gqlgen \
	./protoc-gen-gql \
	./protoc-gen-gogql

GOPATH := $(shell go env GOPATH)

.PHONY: all
all: tools generate-proto

.PHONY: tools
tools: install-protoc go-tools

define LIST_RULE
.PHONY: install-$(notdir $(l))
install-$(notdir $(l)):
	go install $(l)
endef

$(foreach l, $(GO_TOOLS), $(eval $(call LIST_RULE, $(l) )))

.PHONY: go-tools
go-tools: $(foreach l, ${GO_TOOLS}, install-$(notdir $(l)))

.PHONY: clean
clean:
	$(foreach l, ${GO_TOOLS},rm -f ${GOPATH}/bin/$(notdir $(l)))

.PHONY: generate-proto
generate-proto:
	export PATH="$$PATH:$(GOPATH)/bin" && protoc -I ./api/danielvladco/protobuf -I ./api  --go_out=paths=source_relative:./pkg/graphqlpb ./api/danielvladco/protobuf/*.proto

.PHONY: install-protoc
install-protoc:
	./scripts/install-protoc.sh
