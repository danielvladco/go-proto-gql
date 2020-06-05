package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/danielvladco/go-proto-gql/reflect/gateway/internal/server"
	"github.com/mitchellh/mapstructure"
	"github.com/nautilus/gateway"
	"github.com/nautilus/graphql"
)

type Config struct {
	Endpoints  []string `json:"endpoints"`
	Playground *bool    `json:"playground"`
	Address    string   `json:"address"`
}

var (
	configFile = flag.String("cfg", "/opt/config.json", "")
)

func main() {
	flag.Parse()

	f, err := os.Open(*configFile)
	fatalOnErr(err)
	cfg := &Config{}
	err = json.NewDecoder(f).Decode(cfg)
	fatalOnErr(err)
	if cfg.Address == "" {
		cfg.Address = ":8080"
	}
	if cfg.Playground == nil {
		plg := true
		cfg.Playground = &plg
	}

	caller, descs, res, err := server.NewReflectCaller(cfg.Endpoints)
	if err != nil {
		log.Fatal(err)
	}

	gqlDesc := server.GenerateGQLComponents(res, descs)

	queryFactory := gateway.QueryerFactory(func(ctx *gateway.PlanningContext, url string) graphql.Queryer {
		return server.QueryerLogger{server.NewQueryer(server.NewMapping(gqlDesc, descs), gqlDesc, caller)}
	})

	sources := []*graphql.RemoteSchema{{URL: "url1"}}
	sources[0].Schema, err = server.GenerateSchema(gqlDesc)
	if err != nil {
		log.Fatal(err)
	}

	//sc, _ := os.Create("schema.graphql")
	//defer sc.Close()
	//formatter.NewFormatter(sc).FormatSchema(sources[0].Schema)

	g, err := gateway.New(sources, gateway.WithQueryerFactory(&queryFactory))
	if err != nil {
		log.Fatal(err)
	}
	result := &graphql.IntrospectionQueryResult{}
	err = schemaTestLoadQuery(g, graphql.IntrospectionQuery, result, map[string]interface{}{})

	//in, _ := os.Create("introspection.json")
	//defer in.Close()
	//_ = json.NewEncoder(in).Encode(result)

	mux := http.NewServeMux()
	mux.HandleFunc("/query", g.GraphQLHandler)
	if *cfg.Playground {
		mux.HandleFunc("/playground", g.PlaygroundHandler)
	}
	log.Fatal(http.ListenAndServe(cfg.Address, mux))
}

func fatalOnErr(err error) {
	if err != nil {
		panic(err)
	}
}

func schemaTestLoadQuery(qw *gateway.Gateway, query string, target interface{}, variables map[string]interface{}) error {
	reqCtx := &gateway.RequestContext{
		Context:   context.Background(),
		Query:     query,
		Variables: variables,
	}
	plan, err := qw.GetPlans(reqCtx)
	if err != nil {
		return err
	}

	// executing the introspection query should return a full description of the schema
	response, err := qw.Execute(reqCtx, plan)
	if err != nil {
		return err
	}

	// massage the map into the structure
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName: "json",
		Result:  target,
	})
	if err != nil {
		return err
	}
	err = decoder.Decode(response)
	if err != nil {
		return err
	}

	return nil
}
