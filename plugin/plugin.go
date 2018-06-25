package plugin

import (
	"strings"

	"fmt"
	"github.com/danielvladco/go-proto-gql"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
	"github.com/mwitkow/go-proto-validators"
	"unicode"
)

type Message struct {
	*descriptor.DescriptorProto
	io bool
}

type plugin struct {
	*generator.Generator
	generator.PluginImports
	inputs  map[string]*Message
	outputs map[string]*Message
	types   []*descriptor.DescriptorProto
	ins     []*descriptor.DescriptorProto

	useGogoImport bool
}

func NewPlugin(useGogoImport bool) generator.Plugin {
	return &plugin{useGogoImport: useGogoImport, outputs: make(map[string]*Message), inputs: make(map[string]*Message)}
}

func (p *plugin) Name() string {
	return "gql"
}

func (p *plugin) Init(g *generator.Generator) {
	p.Generator = g
}

func (p *plugin) Generate(file *generator.FileDescriptor) {
	p.PluginImports = generator.NewPluginImports(p.Generator)
	p.P("var schema = `")
	p.P("schema {\n\tquery: Query\n\tmutation: Mutation\n}")
	p.P("type Mutation {")
	p.P("type Error {\n\tcode: String\n\tmessage: String\n\tdetails: [Details!]\n}")
	p.P("union Details = String | Int | Boolean | Float | ValidationErr")
	p.In()
	for _, svc := range file.Service {
		for _, rpc := range svc.Method {
			switch getMethodType(rpc) {
			case gql.Type_MUTATION:
				p.P(Field(file.GetPackage()), svc.Name, rpc.Name, "(req: ", formatSchema(rpc.GetInputType()), "): ", formatSchema(rpc.GetOutputType(), "error"))
				p.inputs[rpc.GetInputType()] = &Message{DescriptorProto: &descriptor.DescriptorProto{}, io: true}
				p.outputs[rpc.GetOutputType()] = &Message{DescriptorProto: &descriptor.DescriptorProto{}, io: true}
			}
		}
	}
	p.Out()
	p.P("}\ntype Query {")
	p.In()
	for _, svc := range file.Service {
		for _, rpc := range svc.Method {
			switch getMethodType(rpc) {
			case gql.Type_QUERY:
				p.P(Field(file.GetPackage()), svc.Name, rpc.Name, "(req: ", formatSchema(rpc.GetInputType()), "): ", formatSchema(rpc.GetOutputType(), "error"))
				p.inputs[rpc.GetInputType()] = &Message{DescriptorProto: &descriptor.DescriptorProto{}, io: true}
				p.outputs[rpc.GetOutputType()] = &Message{DescriptorProto: &descriptor.DescriptorProto{}, io: true}
			}
		}
	}
	p.Out()
	p.P("}")

	for name := range p.outputs {
		p.P("union ", formatSchema(name, "error"), " = ", formatSchema(name), " | ", "Error")
	}

	// get all input messages
	for _, m := range file.Messages() {
		if _, ok := p.inputs["."+file.GetPackage()+"."+m.GetName()]; ok {
			for _, f := range m.GetField() {
				if f.GetType() == descriptor.FieldDescriptorProto_TYPE_MESSAGE {
					p.inputs[f.GetTypeName()] = &Message{DescriptorProto: &descriptor.DescriptorProto{}, io: true}
				}
			}
		}
	}

	// get all type messages
	for _, m := range file.Messages() {
		if _, ok := p.outputs["."+file.GetPackage()+"."+m.GetName()]; ok {
			for _, f := range m.GetField() {
				if f.GetType() == descriptor.FieldDescriptorProto_TYPE_MESSAGE {
					p.outputs[f.GetTypeName()] = &Message{DescriptorProto: &descriptor.DescriptorProto{}, io: true}
				}
			}
		}
	}

	for m := range p.inputs {
		mm := strings.Split(m, ".")
		if len(mm) < 3 {
			panic("unable to resolve type")
		}
		if file.GetPackage() == mm[1] {
			p.inputs[m].DescriptorProto = file.GetMessage(strings.Join(mm[2:], "."))
		}
	}

	for m := range p.outputs {
		mm := strings.Split(m, ".")
		if len(mm) < 3 {
			panic("unable to resolve type")
		}
		if file.GetPackage() == mm[1] {
			p.outputs[m].DescriptorProto = file.GetMessage(strings.Join(mm[2:], "."))
		}
	}

	for name, m := range p.inputs {
		var tail []string
		if !m.io {
			tail = []string{"in"}
		}
		p.P("input ", formatSchema(name, tail...), " {")
		p.In()
		for _, f := range m.GetField() {

			p.P(ToLowerFirst(generator.CamelCase(f.GetName())), ": ", p.graphQLType(m.DescriptorProto, f, name, resolveRequired(f), tail...))
		}
		p.Out()
		p.P("}")
	}
	for name, m := range p.outputs {
		var tail []string
		if !m.io {
			tail = []string{"out"}
		}
		p.P("type ", formatSchema(name, tail...), " {")
		p.In()
		for _, f := range m.GetField() {
			p.P(ToLowerFirst(generator.CamelCase(f.GetName())), ": ", p.graphQLType(m.DescriptorProto, f, name, resolveRequired(f), tail...))
		}
		p.Out()
		p.P("}")
	}
	p.P("`")
}

func resolveRequired(field *descriptor.FieldDescriptorProto) bool {
	if v := getValidatorType(field); v != nil {
		switch {
		case v.GetMsgExists():
			return true
		case v.GetStringNotEmpty():
			return true
		case v.GetRepeatedCountMin() > 0:
			return true
		}
	}
	return false
}

func formatSchema(s string, tail ...string) string {
	ss := strings.Split(s, ".")
	if len(ss) < 3 {
		panic("unable to resolve type")
	}
	ss[2] = ToLowerFirst(ss[2])
	return generator.CamelCase(strings.Replace(strings.Trim(strings.Join(append(ss, tail...), "."), "."), ".", "_", -1))
}

func Field(s string) string {
	return ToLowerFirst(generator.CamelCase(s))
}
func Type(s string) string {
	return strings.Title(generator.CamelCase(s))
}
func ToLowerFirst(s string) string {
	return string(unicode.ToLower(rune(s[0]))) + s[1:]
}

func getMessage(file *descriptor.FileDescriptorProto, typeName string) *descriptor.DescriptorProto {
	typeName = strings.TrimPrefix(typeName, "."+*file.Package+".")
	for _, msg := range file.GetMessageType() {
		if msg.GetName() == typeName {
			return msg
		}
		nes := file.GetNestedMessage(msg, strings.TrimPrefix(typeName, msg.GetName()+"."))
		if nes != nil {
			return nes
		}
	}
	return nil
}

func getMethodType(rpc *descriptor.MethodDescriptorProto) gql.Type {
	if rpc.Options != nil {
		v, err := proto.GetExtension(rpc.Options, gql.E_Type)
		if err == nil {
			return v.(gql.Type)
		}
	}
	return gql.Type_MUTATION
}

func getValidatorType(field *descriptor.FieldDescriptorProto) *validator.FieldValidator {
	if field.Options != nil {
		v, err := proto.GetExtension(field.Options, validator.E_Field)
		if err == nil {
			return v.(*validator.FieldValidator)
		}
	}

	return nil
}

func (p *plugin) graphQLType(message *descriptor.DescriptorProto, field *descriptor.FieldDescriptorProto, fullName string, required bool, tail ...string) string {
	var gqltype string
	switch field.GetType() {
	case descriptor.FieldDescriptorProto_TYPE_DOUBLE, descriptor.FieldDescriptorProto_TYPE_FLOAT:
		gqltype = fmt.Sprint("Float")
	case descriptor.FieldDescriptorProto_TYPE_INT64, descriptor.FieldDescriptorProto_TYPE_UINT64,
		descriptor.FieldDescriptorProto_TYPE_INT32, descriptor.FieldDescriptorProto_TYPE_FIXED64,
		descriptor.FieldDescriptorProto_TYPE_FIXED32, descriptor.FieldDescriptorProto_TYPE_SFIXED32,
		descriptor.FieldDescriptorProto_TYPE_SFIXED64, descriptor.FieldDescriptorProto_TYPE_SINT32,
		descriptor.FieldDescriptorProto_TYPE_SINT64:
		gqltype = fmt.Sprint("Int")
	case descriptor.FieldDescriptorProto_TYPE_BOOL:
		gqltype = fmt.Sprint("Boolean")
	case descriptor.FieldDescriptorProto_TYPE_STRING:
		gqltype = fmt.Sprint("String")
	case descriptor.FieldDescriptorProto_TYPE_GROUP:
		panic("mapping a proto group type to graphql is unimplemented")
	case descriptor.FieldDescriptorProto_TYPE_ENUM:
		panic("mapping a proto enum type to graphql is unimplemented")
	case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
		gqltype = formatSchema(fullName, tail...)
	case descriptor.FieldDescriptorProto_TYPE_BYTES:
		//gqltype = fmt.Sprint(schemaPkgName.Use(), ".", "ByteString")
	default:
		panic("unknown proto field type")
	}

	suffix := ""
	prefix := ""
	if field.IsRepeated() {
		prefix += "["
		suffix += "]"
	}
	if required {
		suffix += "!"
	}

	return prefix + gqltype + suffix
}
