package server

import (
	"context"
	"github.com/mitchellh/mapstructure"
	"github.com/nautilus/gateway"
	"github.com/nautilus/graphql"
	"testing"
)

func TestGenerateSchema(t *testing.T) {
	schema, err := GenerateSchema(nil) // FIXME add test
	if err != nil {
		t.Fatal(err)
	}
	g, err := gateway.New([]*graphql.RemoteSchema{{Schema: schema, URL: ""}})

	result := &graphql.IntrospectionQueryResult{}
	err = schemaTestLoadQuery(g, graphql.IntrospectionQuery, result, map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
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
