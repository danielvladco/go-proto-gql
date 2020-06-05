package server

import (
	"github.com/jhump/protoreflect/desc"
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/danielvladco/go-proto-gql/plugin"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
)

func GenerateGQLComponents(filesToGenerate []string, descs []*desc.FileDescriptor) GQLDescriptors {
	p := plugin.NewPlugin(descs)
	fnames := map[string]*desc.FileDescriptor{}
	for _, f := range descs {
		fnames[f.GetName()] = f
	}
	for _, fileName := range filesToGenerate {
		if f, ok := fnames[fileName]; ok {
			p.InitFile(f)
		}
	}
	return p
}

type GQLDescriptors interface {
	GetOneof(ref plugin.OneofRef) *plugin.Type
	Oneofs() map[string]*plugin.Type
	Scalars() map[string]*plugin.Type
	Enums() map[string]*plugin.Type
	Types() map[string]*plugin.Type
	Inputs() map[string]*plugin.Type
	Subscriptions() plugin.Methods
	Queries() plugin.Methods
	Mutations() plugin.Methods
	Fields() map[plugin.FieldKey]*descriptor.FieldDescriptorProto
	FieldBack(t *descriptor.DescriptorProto, name string) *descriptor.FieldDescriptorProto
	GraphQLField(t *descriptor.DescriptorProto, f *descriptor.FieldDescriptorProto) string
	GraphQLType(field *descriptor.FieldDescriptorProto, input bool) *ast.Type
	GqlModelNames() map[*plugin.Type]string
	IsEmpty(t *plugin.Type) bool
	IsAny(typeName string) bool
}
