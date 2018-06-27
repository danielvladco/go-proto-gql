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

type Method struct {
	Name       string
	InputType  string
	OutputType string
	HideInput  bool
	HideOutput bool
}

type plugin struct {
	*generator.Generator
	generator.PluginImports
	inputs   map[string]*Message
	outputs  map[string]*Message
	imports  map[string]string
	typesMap map[string]string
	types    []*descriptor.DescriptorProto
	ins      []*descriptor.DescriptorProto

	mutations     []*Method
	queries       []*Method
	subscriptions []*Method
	schema        *Bytes
	//mutation     *Bytes
	//subscription *Bytes
	//query        *Bytes
}

func NewPlugin() *plugin {
	schema := NewBytes()
	schema.P(`
type Error {
    code: String
    message: String
    details: [Details]
}
type Description {
    description: String!
}
type Validation {
    field: String
    description: String
}
union Details = Description | Validation
schema {
	query: Query
	mutation: Mutation
	subscription: Subscription
}
`)
	return &plugin{
		outputs:  make(map[string]*Message),
		inputs:   make(map[string]*Message),
		typesMap: make(map[string]string),
		schema:   schema,
	}
}

func (p *plugin) GetSchema() string {
	errorOut := make(map[string]struct{})
	collectResError := func(mm []*Method) {
		for _, m := range mm {
			if !m.HideOutput {
				errorOut[fmt.Sprint("union ", format(m.OutputType, "error"), " = ", format(m.OutputType)+" | Error")] = struct{}{}
			}
		}
	}
	collectResError(p.mutations)
	collectResError(p.queries)
	collectResError(p.subscriptions)
	for m := range errorOut {
		p.schema.P(m)
	}
	printMethods := func(mm []*Method) {
		for _, m := range mm {
			in := "(req: " + format(m.InputType) + ")"
			if m.HideInput {
				in = ""
			}
			out := format(m.OutputType)
			if m.HideOutput {
				out = ""
			}
			p.schema.P(m.Name, in, ": ", out+"Error")
		}
	}
	p.schema.P("type Mutation {")
	p.schema.In()
	printMethods(p.mutations)
	p.schema.Out()
	p.schema.P("}\ntype Query {")
	p.schema.In()
	printMethods(p.queries)
	p.schema.Out()
	p.schema.P("}\ntype Subscription {")
	p.schema.In()
	printMethods(p.subscriptions)
	p.schema.Out()
	p.schema.P("}")
	return p.schema.String()
}
func (p *plugin) GetTypesMap() map[string]string { return p.typesMap }
func (p *plugin) Name() string                   { return "gql" }
func (p *plugin) Init(g *generator.Generator)    { p.Generator = g }

func (p *plugin) Generate(file *generator.FileDescriptor) {
	p.PluginImports = generator.NewPluginImports(p.Generator)

	for _, svc := range file.Service {
		for _, rpc := range svc.Method {
			m := &Method{
				Name:       Field(file.GetPackage()) + strings.Title(svc.GetName()) + strings.Title(rpc.GetName()),
				InputType:  rpc.GetInputType(),
				OutputType: rpc.GetOutputType(),
			}
			tt := getMethodType(rpc)

			mutation := gql.Type_MUTATION
			if tt == nil {
				tt = &mutation
			}
			switch *tt {
			case gql.Type_MUTATION:
				p.mutations = append(p.mutations, m)
			case gql.Type_QUERY:
				p.queries = append(p.queries, m)
			case gql.Type_SUBSCRIPTION:
				p.subscriptions = append(p.subscriptions, m)
			}

			p.inputs[rpc.GetInputType()] = &Message{DescriptorProto: &descriptor.DescriptorProto{}, io: true}
			p.outputs[rpc.GetOutputType()] = &Message{DescriptorProto: &descriptor.DescriptorProto{}, io: true}
		}
	}

	// get all input messages
	for _, m := range file.Messages() {
		if _, ok := p.inputs["."+file.GetPackage()+"."+m.GetName()]; ok {
			for _, f := range m.GetField() {
				if f.IsMessage() {
					p.inputs[f.GetTypeName()] = &Message{DescriptorProto: &descriptor.DescriptorProto{}, io: true}
				}
			}
		}
	}

	// get all type messages
	for _, m := range file.Messages() {
		if _, ok := p.outputs["."+file.GetPackage()+"."+m.GetName()]; ok {
			for _, f := range m.GetField() {
				if f.IsMessage() {
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
		fns := strings.Split(file.GetName(), "/")
		if len(fns) < 2 {
			return
		}

		p.typesMap[format(name, tail...)] = strings.Join(fns[:len(fns)-1], "/") + "." + goType(name)
		p.schema.P("input ", format(name, tail...), " {")
		p.schema.In()
		for _, f := range m.GetField() {
			if f.IsBytes() {
				p.schema.P("#", ToLowerFirst(generator.CamelCase(f.GetName())), ": Bytes # Bytes type not implemented in gql!")
				continue
			}
			p.schema.P(ToLowerFirst(generator.CamelCase(f.GetName())), ": ", p.graphQLType(m.DescriptorProto, f, name, resolveRequired(f), tail...))
		}
		p.schema.Out()
		p.schema.P("}")
		if len(m.GetField()) == 0 {
			p.HideInputOutput(name)
		}
	}
	for name, m := range p.outputs {
		var tail []string
		if !m.io {
			tail = []string{"out"}
		}
		fns := strings.Split(file.GetName(), "/")
		if len(fns) < 2 {
			return
		}
		p.typesMap[format(name, tail...)] = strings.Join(fns[:len(fns)-1], "/") + "." + goType(name)
		p.schema.P("type ", format(name, tail...), " {")
		p.schema.In()
		for _, f := range m.GetField() {
			p.schema.P(ToLowerFirst(generator.CamelCase(f.GetName())), ": ", p.graphQLType(m.DescriptorProto, f, name, resolveRequired(f), tail...))
		}
		p.schema.Out()
		p.schema.P("}")
		if len(m.GetField()) == 0 {
			p.HideInputOutput(name)
		}
	}

	for _, e := range file.Enums() {
		fns := strings.Split(file.GetName(), "/")
		pkgName := e.File().GetPackage()
		if len(fns) < 2 {
			return
		}

		oldGQLName := strings.Join(append([]string{".", pkgName}, e.TypeName()...), ".")

		p.typesMap[format(oldGQLName)] = strings.Join(fns[:len(fns)-1], "/") + "." + generator.CamelCaseSlice(e.TypeName())
		p.schema.P("enum ", format(oldGQLName), " {")
		p.schema.In()
		for _, v := range e.GetValue() {
			p.schema.P(v.GetName())
		}
		p.schema.Out()
		p.schema.P("}")
	}
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

func goType(s string) string {
	ss := strings.Split(s, ".")
	if len(ss) < 3 {
		panic("unable to resolve type")
	}
	return generator.CamelCase(strings.Replace(strings.Trim(strings.Join(ss[2:], "."), "."), ".", "_", -1))
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

func (p *plugin) HideInputOutput(name string) {
	for _, n := range p.mutations {
		if name == n.InputType {
			n.HideInput = true
		}
		if name == n.OutputType {
			n.HideOutput = true
		}
	}
	for _, n := range p.queries {
		if name == n.InputType {
			n.HideInput = true
		}
		if name == n.OutputType {
			n.HideOutput = true
		}
	}
	for _, n := range p.subscriptions {
		if name == n.InputType {
			n.HideInput = true
		}
		if name == n.OutputType {
			n.HideOutput = true
		}
	}
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
		gqltype = fmt.Sprint("Bytes")
	default:
		panic("unknown proto field type")
	}

	suffix := ""
	prefix := ""
	if field.IsRepeated() {
		prefix += "["
		suffix += "!]"
	}
	if required {
		suffix += "!"
	}

	return prefix + gqltype + suffix
}
