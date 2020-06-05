package server

import (
	"github.com/danielvladco/go-proto-gql/plugin"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
	"github.com/vektah/gqlparser/v2/ast"
)

func GenerateSchema(p GQLDescriptors) (*ast.Schema, error) {
	queries := p.Queries()
	mutations := p.Mutations()
	subscriptions := p.Subscriptions()

	queryDef := &ast.Definition{Kind: ast.Object, Name: "Query", Position: &ast.Position{}}
	mutationDef := &ast.Definition{Kind: ast.Object, Name: "Mutation", Position: &ast.Position{}}
	subscriptionsDef := &ast.Definition{Kind: ast.Object, Name: "Subscription", Position: &ast.Position{}}
	schema := &ast.Schema{
		Query:      queryDef,
		Types:      map[string]*ast.Definition{"Query": queryDef},
		Directives: map[string]*ast.DirectiveDefinition{},
	}
	if len(mutations) > 0 {
		schema.Mutation = mutationDef
		schema.Types[mutationDef.Name] = mutationDef
	}
	if len(queries) == 0 {
		//id := ast.NonNullNamedType("ID", &ast.Position{})
		//node := &ast.Definition{
		//	Kind:   ast.Interface,
		//	Name:   "Node",
		//	Fields: []*ast.FieldDefinition{{Name: "id", Type: id}},
		//}
		//schema.Types[node.Name] = node
		schema.Query.Fields = append(schema.Query.Fields, &ast.FieldDefinition{
			Name: "dummy",
			//Arguments: []*ast.ArgumentDefinition{{
			//	Description: "",
			//	Name:        "id",
			//	Type:        id,
			//}},
			Type: ast.NamedType("Boolean", &ast.Position{}),
		})
	}
	if len(subscriptions) > 0 {
		schema.Subscription = subscriptionsDef
		schema.Types[subscriptionsDef.Name] = subscriptionsDef
	}

	oneofs := p.Oneofs()
	for _, oo := range oneofs {
		var types []string
		if oo.TypeName == plugin.ScalarAny {
			for _, t := range p.Types() {
				types = append(types, p.GqlModelNames()[t])
			}
		} else {
			if len(oo.OneofTypes) == 0 {
				continue
			}

			types = oo.OneofTypes.Strings()
			for i, v := range types {
				typ := p.Types()[v]
				if p.IsEmpty(typ) {
					continue
				}
				types[i] = p.GqlModelNames()[typ]
			}

			// if ti is not the special type Any, register it as a new directive
			schema.Directives[p.GqlModelNames()[oo]] = &ast.DirectiveDefinition{
				Name:      p.GqlModelNames()[oo],
				Locations: []ast.DirectiveLocation{ast.LocationInputFieldDefinition},
				Position:  &ast.Position{Src: &ast.Source{}},
			}
		}
		schema.Types[p.GqlModelNames()[oo]] = &ast.Definition{
			Kind:     ast.Union,
			Name:     p.GqlModelNames()[oo],
			Types:    types,
			Position: &ast.Position{},
		}
	}
	scalars := p.Scalars()
	for _, tt := range scalars {
		if p.GqlModelNames()[tt] == "" {
			continue
		}
		schema.Types[p.GqlModelNames()[tt]] = &ast.Definition{
			Kind:     ast.Scalar,
			Name:     p.GqlModelNames()[tt],
			Position: &ast.Position{},
		}
	}

	types := p.Types()
	for _, tt := range types {
		if p.IsEmpty(tt) {
			continue
		}

		oneofs := make(map[plugin.OneofRef]struct{})
		var fields ast.FieldList
		for _, f := range tt.GetField() {
			field := &ast.FieldDefinition{
				Position: &ast.Position{},
			}
			if f.OneofIndex != nil {
				oneofRef := plugin.OneofRef{Parent: tt, Index: *f.OneofIndex}
				if _, ok := oneofs[oneofRef]; !ok {
					//TODO find a better way
					// these are fake types
					//generator.CamelCase(m.OneofDecl[*f.OneofIndex].GetName()))
					// use this just as workaround
					p.GraphQLField(tt.DescriptorProto, f)
					field.Name = plugin.ToLowerFirst(generator.CamelCase(tt.OneofDecl[*f.OneofIndex].GetName()))
					field.Type = &ast.Type{NamedType: p.GqlModelNames()[p.GetOneof(oneofRef)], Position: &ast.Position{}}
				} else {
					oneofs[oneofRef] = struct{}{}
				}
			} else {
				field.Name = p.GraphQLField(tt.DescriptorProto, f)
				field.Type = p.GraphQLType(f, false)
			}

			fields = append(fields, field)
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
		parent := &ast.Definition{
			Kind: ast.InputObject,
			Name: p.GqlModelNames()[tt],
			//Fields:   fields,
			Position: &ast.Position{},
		}
		var fields ast.FieldList
		for _, f := range tt.GetField() {
			var dirs []*ast.Directive
			if f.OneofIndex != nil {
				oneofRef := plugin.OneofRef{Parent: tt, Index: *f.OneofIndex}
				directiveName := p.GqlModelNames()[p.GetOneof(oneofRef)]
				directiveDef := schema.Directives[directiveName]
				dirs = []*ast.Directive{{Name: directiveName, ParentDefinition: parent, Definition: directiveDef, Location: directiveDef.Locations[0], Position: &ast.Position{}}}
			}
			fields = append(fields, &ast.FieldDefinition{
				Name:       p.GraphQLField(tt.DescriptorProto, f),
				Type:       p.GraphQLType(f, true),
				Directives: dirs,
				Position:   &ast.Position{},
			})
		}

		if p.IsEmpty(tt) {
			continue
		}

		parent.Fields = fields
		schema.Types[p.GqlModelNames()[tt]] = parent
	}
	// TODO do not repeat this
	enums := p.Enums()
	for _, ee := range enums {
		var enumValues ast.EnumValueList
		for _, v := range ee.GetValue() {
			enumValues = append(enumValues, &ast.EnumValueDefinition{
				Name:     v.GetName(),
				Position: &ast.Position{},
			})
		}

		schema.Types[p.GqlModelNames()[ee]] = &ast.Definition{
			Kind:       ast.Enum,
			Name:       p.GqlModelNames()[ee],
			EnumValues: enumValues,
			Position:   &ast.Position{},
		}
	}

	if err := rangeMethods(queries, &schema.Query.Fields, p); err != nil {
		return nil, err
	}
	if err := rangeMethods(mutations, &schema.Mutation.Fields, p); err != nil {
		return nil, err
	}
	if err := rangeMethods(subscriptions, &schema.Subscription.Fields, p); err != nil {
		return nil, err
	}

	return schema, nil
}

func rangeMethods(methods plugin.Methods, res *ast.FieldList, p GQLDescriptors) error {
	for _, m := range methods {
		var in *ast.Type
		var out = &ast.Type{NamedType: "Boolean", Position: &ast.Position{}}

		// some scalars can be used as inputs such as 'Any'
		if !m.MethodDescriptorProto.GetClientStreaming() || m.Phony {
			if scalar, ok := p.Scalars()[m.InputType]; ok {
				if !p.IsEmpty(scalar) {
					in = &ast.Type{NamedType: p.GqlModelNames()[scalar], Position: &ast.Position{}}
				}

			} else if input, ok := p.Inputs()[m.InputType]; ok {
				if !p.IsEmpty(input) {
					in = &ast.Type{NamedType: p.GqlModelNames()[input], Position: &ast.Position{}}
				}
			}
		}

		if oneof, ok := p.Oneofs()[m.OutputType]; ok {
			if !p.IsEmpty(oneof) {
				out = &ast.Type{NamedType: p.GqlModelNames()[oneof], Position: &ast.Position{}}
			}
		} else if scalar, ok := p.Scalars()[m.OutputType]; ok {
			if !p.IsEmpty(scalar) {
				out = &ast.Type{NamedType: p.GqlModelNames()[scalar], Position: &ast.Position{}}
			}
		} else if typ, ok := p.Types()[m.OutputType]; ok {
			if !p.IsEmpty(typ) {
				out = &ast.Type{NamedType: p.GqlModelNames()[typ], Position: &ast.Position{}}
			}
		}

		var args ast.ArgumentDefinitionList
		if in != nil {
			args = append(args, &ast.ArgumentDefinition{
				Description:  "",
				DefaultValue: nil,
				Name:         "in",
				Type:         in,
				Directives:   nil,
				Position:     &ast.Position{},
			})
		}
		*res = append(*res, &ast.FieldDefinition{
			Description: "",
			Name:        m.Name,
			Arguments:   args,
			Type:        out,
		})
	}
	return nil
}
