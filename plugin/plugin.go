package plugin

import (
	"strings"

	"fmt"
	"github.com/danielvladco/go-proto-gql"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
	"github.com/mwitkow/go-proto-validators"
	"log"
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
	enums   map[string]*descriptor.EnumDescriptorProto
	types   []*descriptor.DescriptorProto
	ins     []*descriptor.DescriptorProto

	useGogoImport bool
}

func NewPlugin(useGogoImport bool) generator.Plugin {
	return &plugin{useGogoImport: useGogoImport, outputs: make(map[string]*Message), inputs: make(map[string]*Message), enums: make(map[string]*descriptor.EnumDescriptorProto)}
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
	p.P(`
type Error {
    code: String
    message: String
    details: [Details]
}
union Details = Description | ValidationErr
type Description {
    descriptionL String!
}
type Validation {
    field: String
    description: String
}`)
	p.P("schema {\n\tquery: Query\n\tmutation: Mutation\n\tsubscription: Subscription\n}")
	printRPC := func(t gql.Type) {
		for _, svc := range file.Service {
			for _, rpc := range svc.Method {
				tt := getMethodType(rpc)
				mutation := gql.Type_MUTATION
				if tt == nil {
					tt = &mutation
				}
				if *tt == t {
					p.P(Field(file.GetPackage()), svc.Name, rpc.Name, "(req: ", format(rpc.GetInputType()), "): ", format(rpc.GetOutputType(), "error"))
					p.inputs[rpc.GetInputType()] = &Message{DescriptorProto: &descriptor.DescriptorProto{}, io: true}
					p.outputs[rpc.GetOutputType()] = &Message{DescriptorProto: &descriptor.DescriptorProto{}, io: true}
				}
			}
		}
	}
	p.P("type Mutation {")
	p.In()
	printRPC(gql.Type_MUTATION)
	p.Out()
	p.P("}\ntype Query {")
	p.In()
	printRPC(gql.Type_QUERY)
	p.Out()
	p.P("}\ntype Subscription {")
	p.In()
	printRPC(gql.Type_SUBSCRIPTION)
	p.Out()
	p.P("}")

	for name := range p.outputs {
		p.P("union ", format(name, "error"), " = ", format(name), " | ", "Error")
	}

	// get all input messages
	for _, m := range file.Messages() {
		if _, ok := p.inputs["."+file.GetPackage()+"."+m.GetName()]; ok {
			for _, f := range m.GetField() {
				if f.GetType() == descriptor.FieldDescriptorProto_TYPE_MESSAGE {
					p.inputs[f.GetTypeName()] = &Message{DescriptorProto: &descriptor.DescriptorProto{}, io: true}
				}
				if f.GetType() == descriptor.FieldDescriptorProto_TYPE_ENUM {
					p.enums[f.GetTypeName()] = &descriptor.EnumDescriptorProto{}
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
				if f.GetType() == descriptor.FieldDescriptorProto_TYPE_ENUM {
					p.enums[f.GetTypeName()] = &descriptor.EnumDescriptorProto{}
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

	for e := range p.enums {
		mm := strings.Split(e, ".")
		if len(mm) < 3 {
			panic("unable to resolve type")
		}

		if file.GetPackage() == mm[1] {
			msg := file.GetMessage(strings.Join(mm[2:len(mm)-1], "."))
			for _, tt := range msg.GetEnumType() {
				if tt.GetName() == mm[len(mm)-1] {
					p.enums[e] = tt
				}
			}
		}
	}

	for name, m := range p.inputs {
		var tail []string
		if !m.io {
			tail = []string{"in"}
		}
		p.P("input ", format(name, tail...), " {")
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
		p.P("type ", format(name, tail...), " {")
		p.In()
		for _, f := range m.GetField() {
			p.P(ToLowerFirst(generator.CamelCase(f.GetName())), ": ", p.graphQLType(m.DescriptorProto, f, name, resolveRequired(f), tail...))
		}
		p.Out()
		p.P("}")
	}

	for name, e := range p.enums {
		p.P("enum ", format(name), " {")
		p.In()
		for _, v := range e.Value {
			p.P(v.GetName())
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

func format(s string, tail ...string) string {
	ss := strings.Split(s, ".")
	if len(ss) < 3 {
		panic("unable to resolve type")
	}
	for i := range ss {
		ss[i] = ToLowerFirst(ss[i])
	}
	return generator.CamelCase(strings.Replace(strings.Trim(strings.Join(append(ss, tail...), "."), "."), ".", "_", -1))
}

func Field(s string) string {
	return ToLowerFirst(generator.CamelCase(s))
}
func Type(s string) string {
	return strings.Title(generator.CamelCase(s))
}
func ToLowerFirst(s string) string {
	if len(s) > 0 {
		return string(unicode.ToLower(rune(s[0]))) + s[1:]
	}
	return ""
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

func getMethodType(rpc *descriptor.MethodDescriptorProto) *gql.Type {
	if rpc.Options != nil {
		v, err := proto.GetExtension(rpc.Options, gql.E_Type)
		if err == nil {
			return v.(*gql.Type)
		} else {
			log.Println(err.Error())
		}
	}
	return nil
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
	case descriptor.FieldDescriptorProto_TYPE_ENUM, descriptor.FieldDescriptorProto_TYPE_MESSAGE:
		gqltype = format(field.GetTypeName(), tail...)
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
