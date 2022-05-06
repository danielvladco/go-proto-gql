package server

import (
	"net/http"

	"github.com/nautilus/gateway"
	"github.com/nautilus/graphql"
	"github.com/rs/cors"

	"github.com/danielvladco/go-proto-gql/pkg/generator"
)

func Server(cfg *Config) (http.Handler, error) {
	if cfg.Playground == nil {
		plg := true
		cfg.Playground = &plg
	}

	caller, descs, err := NewReflectCaller(cfg.Grpc)
	if err != nil {
		return nil, err
	}

	gqlDesc, err := generator.NewSchemas(descs, true, true, nil)
	if err != nil {
		return nil, err
	}

	repo := generator.NewRegistry(gqlDesc)

	queryFactory := gateway.QueryerFactory(func(ctx *gateway.PlanningContext, url string) graphql.Queryer {
		return QueryerLogger{NewQueryer(repo, caller)}
	})
	sources := []*graphql.RemoteSchema{{URL: "url1"}}
	sources[0].Schema = gqlDesc.AsGraphql()[0]

	g, err := gateway.New(sources, gateway.WithQueryerFactory(&queryFactory))
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/query", g.GraphQLHandler)
	if cfg.Playground != nil && *cfg.Playground {
		mux.HandleFunc("/playground", g.PlaygroundHandler)
	}
	var handler http.Handler = mux
	if cfg.Cors == nil {
		return cors.Default().Handler(handler), nil
	}
	return cors.New(*cfg.Cors).Handler(handler), nil
}
