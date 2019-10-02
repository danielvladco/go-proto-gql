package main

import (
	"github.com/danielvladco/go-proto-gql/plugin"
	"github.com/golang/protobuf/protoc-gen-go/generator"
	goplugin "github.com/golang/protobuf/protoc-gen-go/plugin"
	"github.com/graphql-go/graphql"
	"google.golang.org/grpc"
)

func main() {
	cc, err := grpc.Dial("localhost:8081")
	die(err)
	client := NewClient(cc)

	desc, err := client.ListPackages()
	die(err)

	gen := generator.New()

	gen.Request = &goplugin.CodeGeneratorRequest{
		ProtoFile: desc,
	}

	p := &service{plugin.NewPlugin()}

	gen.WrapTypes()
	gen.SetPackageNames()
	gen.BuildTypeNameMap()
	generator.RegisterPlugin(p)
	gen.GenerateAllFiles()
}

type service struct {
	*plugin.Plugin

	fields []*graphql.Field
}

func (p *service) GenerateImports(file *generator.FileDescriptor) {}
func (p *service) Name() string                                   { return "gqlddd" }
func (p *service) Init(g *generator.Generator)                    { p.Generator = g }
func (p *service) Generate(file *generator.FileDescriptor) {
	p.InitFile(file)

	types := p.Types()
	for _, tt := range types {
		fields := graphql.Fields{}
		for _, f := range tt.GetField() {
			fields[f.GetName()] = &graphql.Field{
				Type: p.GraphQLType(f, types),
			}
		}

		graphql.NewObject(graphql.ObjectConfig{
			Name:   tt.TypeName,
			Fields: fields,
		})
	}
}

func die(err error) {
	if err != nil {
		panic(err)
	}
}
