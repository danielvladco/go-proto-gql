package server

import (
	"github.com/danielvladco/go-proto-gql/plugin"
	"github.com/vektah/gqlparser/v2/ast"
)

func GenerateSchema(p *plugin.Plugin) (*ast.Schema, error) {
	queries := p.Queries()
	mutations := p.Mutations()
	queryDef := &ast.Definition{Kind: ast.Object, Name: "Query", Position: &ast.Position{}}
	mutationDef := &ast.Definition{Kind: ast.Object, Name: "Mutation", Position: &ast.Position{}}
	schema := &ast.Schema{
		Query: queryDef,
		Types: map[string]*ast.Definition{"Query": queryDef},
	}
	if len(mutations) > 0 {
		schema.Mutation = mutationDef
		schema.Types[mutationDef.Name] = mutationDef
	}
	if len(queries) == 0 {
		id := ast.NonNullNamedType("ID", &ast.Position{})
		node := &ast.Definition{
			Kind:   ast.Interface,
			Name:   "Node",
			Fields: []*ast.FieldDefinition{{Name: "id", Type: id}},
		}
		schema.Types[node.Name] = node
		schema.Query.Fields = append(schema.Query.Fields, &ast.FieldDefinition{
			Name: "node",
			Arguments: []*ast.ArgumentDefinition{{
				Description: "",
				Name:        "id",
				Type:        id,
			}},
			Type: ast.NamedType(node.Name, &ast.Position{}),
		})
	}

	types := p.Types()
	for _, tt := range types {
		var fields ast.FieldList
		for _, f := range tt.GetField() {
			fields = append(fields, &ast.FieldDefinition{
				Name: p.GraphQLField(tt, f),
				Type: &ast.Type{
					NamedType: p.GraphQLType(f, types),
				},
				Position: &ast.Position{},
			})
		}

		schema.Types[p.GqlModelNames()[tt]] = &ast.Definition{
			Kind:     ast.Object,
			Name:     p.GqlModelNames()[tt],
			Fields:   fields,
			Position: &ast.Position{},
		}
	}

	// TODO do not repeat this
	inputs := p.Inputs()
	for _, tt := range inputs {
		var fields ast.FieldList
		for _, f := range tt.GetField() {
			fields = append(fields, &ast.FieldDefinition{
				Name:     p.GraphQLField(tt, f),
				Type:     &ast.Type{NamedType: p.GraphQLType(f, inputs)},
				Position: &ast.Position{},
			})
		}

		schema.Types[p.GqlModelNames()[tt]] = &ast.Definition{
			Kind:     ast.InputObject,
			Name:     p.GqlModelNames()[tt],
			Fields:   fields,
			Position: &ast.Position{},
		}
	}

	if err := rangeMethods(p.Queries(), &schema.Query.Fields, p); err != nil {
		return nil, err
	}
	if err := rangeMethods(p.Mutations(), &schema.Mutation.Fields, p); err != nil {
		return nil, err
	}
	return schema, nil
}

func rangeMethods(methods plugin.Methods, res *ast.FieldList, p *plugin.Plugin) error {
	for _, m := range methods {
		var in *ast.Type
		var out = &ast.Type{NamedType: "Boolean", Position: &ast.Position{}}

		// some scalars can be used as inputs such as 'Any'
		if scalar, ok := p.Scalars()[m.InputType]; ok {
			if !p.IsEmpty(scalar) {
				in = &ast.Type{NamedType: p.GqlModelNames()[scalar], Position: &ast.Position{}}
			}
		} else if input, ok := p.Inputs()[m.InputType]; ok {
			if !p.IsEmpty(input) {
				in = &ast.Type{NamedType: p.GqlModelNames()[input], Position: &ast.Position{}}
			}
		}
		if scalar, ok := p.Scalars()[m.OutputType]; ok {
			if !p.IsEmpty(scalar) {
				out = &ast.Type{NamedType: p.GqlModelNames()[scalar], Position: &ast.Position{}}
			}
		} else if typ, ok := p.Types()[m.OutputType]; ok {
			if !p.IsEmpty(typ) {
				out = &ast.Type{NamedType: p.GqlModelNames()[typ], Position: &ast.Position{}}
			}
		}

		//tttp := p.Types()[m.OutputType]
		*res = append(*res, &ast.FieldDefinition{
			Description: "",
			Name:        m.Name,
			Arguments: ast.ArgumentDefinitionList{{
				Description:  "",
				Name:         "in",
				DefaultValue: nil,
				Type:         in,
				Directives:   nil,
				Position:     &ast.Position{},
			}},
			Type: out,
		})
	}
	return nil
}
