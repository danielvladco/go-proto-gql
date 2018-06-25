package main

import (
	"context"
	"github.com/danielvladco/go-proto-gql/gqlgen"
	"github.com/danielvladco/go-proto-gql/gqlgen/types"
	"github.com/vektah/gqlgen/handler"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.Handle("/", handler.Playground("Jobinn GraphQL playground", "/query"))
	mux.Handle("/query", handler.GraphQL(gqlgen.MakeExecutableSchema(&res{})))
	http.ListenAndServe(":9999", mux)
}

type res struct {
}

func (r *res) Mutation_PostTest(ctx context.Context, input *types.Test) (*types.Test, error) {
	return input, nil
}

func (r *res) Query_GetTest(ctx context.Context, input *types.Test) (*types.Test, error) {
	return input, nil
}
