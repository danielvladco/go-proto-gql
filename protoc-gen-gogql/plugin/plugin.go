package plugin

import (
	"bytes"
	"fmt"
	"github.com/99designs/gqlgen/codegen"
	"log"
	"os"

	"strings"
	"unicode"

	"github.com/danielvladco/go-proto-gql"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
	"github.com/mwitkow/go-proto-validators"
)

type Type struct {
	*descriptor.DescriptorProto
	*descriptor.EnumDescriptorProto
	packageDir  string
	packageName string
	typeName    string
	in          bool
}

func (t *Type) isEmpty() bool {
	if t.DescriptorProto != nil {
		return len(t.DescriptorProto.GetField()) == 0
	}
	return false
}

type Method struct {
	Name       string
	InputType  string
	OutputType string
}

type plugin struct {
	*Schema
	*generator.Generator
	generator.PluginImports

	// info for the current schema
	mutations     []*Method
	queries       []*Method
	subscriptions []*Method
	inputs        map[string]*Type // map containing string representation of proto message that is used as gql input
	types         map[string]*Type // map containing string representation of proto message that is used as gql type
	enums         map[string]*Type // map containing string representation of proto enum that is used as gql enum
	gqlModelNames map[*Type]string // map containing reference to gql type names

	gqlModels codegen.TypeMap // 99designs typemap entries

	logger *log.Logger
}

func NewPlugin() *plugin {
	return &plugin{
		Schema:        NewBytes(),
		types:         make(map[string]*Type),
		inputs:        make(map[string]*Type),
		enums:         make(map[string]*Type),
		gqlModelNames: make(map[*Type]string),
		gqlModels:     make(codegen.TypeMap),
		logger:        log.New(os.Stderr, "protoc-gen-gogql: ", log.Lshortfile),
	}
}

func (p *plugin) Error(err error, msgs ...string) {
	s := strings.Join(msgs, " ") + ":" + err.Error()
	_ = p.logger.Output(2, s)
	os.Exit(1)
}

func (p *plugin) Warn(msgs ...interface{}) {
	_ = p.logger.Output(2, fmt.Sprintln(append([]interface{}{"WARN: "}, msgs...)...))
}

func (p *plugin) GetSchemaByIndex(index int) *bytes.Buffer { return p.Schema.GetSchemaByIndex(index) }
func (p *plugin) GetTypesMap() codegen.TypeMap             { return p.gqlModels }
func (p *plugin) Name() string                             { return "gql" }
func (p *plugin) Init(g *generator.Generator)              { p.Generator = g }

func (p *plugin) GenerateImports(file *generator.FileDescriptor) {
	p.PluginImports = generator.NewPluginImports(p.Generator)
	p.PluginImports.GenerateImports(file)
}

func (p *plugin) Generate(file *generator.FileDescriptor) {
	for _, fileName := range p.Request.FileToGenerate {
		if fileName == file.GetName() {
			p.NextSchema()
			// purge data for the current schema
			p.types, p.inputs, p.enums, p.gqlModelNames, p.subscriptions, p.mutations, p.queries =
				make(map[string]*Type), make(map[string]*Type), make(map[string]*Type), make(map[*Type]string), nil, nil, nil
			p.generate(file)
		}
	}
}

func (p *plugin) getImportPath(file *descriptor.FileDescriptorProto) string {
	fileName := strings.Split(file.GetName(), "/")
	gopackageOpt := strings.Split(file.GetOptions().GetGoPackage(), ";")

	if len(gopackageOpt) == 2 {
		return gopackageOpt[0]
	}
	if len(fileName) > 1 {
		return strings.Join(fileName[:len(fileName)-1], "/")
	}

	p.Error(fmt.Errorf("go import path for %s could not be resolved correctly: use option go_package with full package name in this file", file.GetName()))
	return "."
}

func (p *plugin) getMessageType(file *descriptor.FileDescriptorProto, typeName string) *Type {
	packagePrefix := "." + file.GetPackage() + "."
	if strings.HasPrefix(typeName, packagePrefix) {
		typeWithoutPrefix := strings.TrimPrefix(typeName, packagePrefix)
		if m := file.GetMessage(typeWithoutPrefix); m != nil {
			return &Type{
				DescriptorProto: m,
				typeName:        generator.CamelCaseSlice(strings.Split(typeWithoutPrefix, ".")),
				packageDir:      p.getImportPath(file),
				packageName:     strings.Replace(file.GetPackage(), ".", "_", -1) + "_",
				in:              false,
			}
		}
		if e := getEnum(file, typeWithoutPrefix); e != nil {
			return &Type{
				EnumDescriptorProto: e,
				typeName:            generator.CamelCaseSlice(strings.Split(typeWithoutPrefix, ".")),
				packageDir:          p.getImportPath(file),
				packageName:         strings.Replace(file.GetPackage(), ".", "_", -1) + "_",
			}
		}
	}

	return nil
}

// recursively passes trough all fields of a all types from a given map and completes it with missing types that ar user as fields
// i. e. if message Type1 { Type2: field1 = 1; } exists in the map this function will look up Type2 and add Type2 to the map as well
func (p *plugin) fillTypeMap(typeName string, messages map[string]*Type) {
	for _, ff := range p.AllFiles().GetFile() {
		if message := p.getMessageType(ff, typeName); message != nil {
			if _, ok := messages[typeName]; ok {
				message.in = true
			}

			if message.EnumDescriptorProto != nil {
				p.enums[typeName] = message
				continue
			}

			messages[typeName] = message

			for _, field := range message.Field {
				if field.IsMessage() || field.IsEnum() {
					p.fillTypeMap(field.GetTypeName(), messages)
				}
			}
		}
	}
}

func (p *plugin) fillGqlModels(entries map[string]*Type, in bool) {
	for _, m := range entries {
		entryName := m.typeName

		// add In suffix to input types for clarity
		if in && !m.in {
			entryName += "In"
		}

		// concatenate package name in case if two types with the same name are found
		if _, ok := p.gqlModels[entryName]; ok {
			entryName = m.packageName + entryName
		}

		// add entries to the type map
		p.gqlModels[entryName] = codegen.TypeMapEntry{Model: m.packageDir + "." + m.typeName}
		p.gqlModelNames[m] = entryName
	}
}

func (p *plugin) generate(file *generator.FileDescriptor) {
	for _, svc := range file.GetService() {
		for _, rpc := range svc.GetMethod() {
			m := &Method{
				Name:       ToLowerFirst(svc.GetName()) + strings.Title(rpc.GetName()),
				InputType:  rpc.GetInputType(),
				OutputType: rpc.GetOutputType(),
			}

			if rpc.GetClientStreaming() {
				p.Warn("client streaming not supported with gql")
				//TODO: add an additional endpoint that will make requests and act like streaming from the client side
				// i. e. A proto rpc like this...
				// `service Service { rpc MessagesLoop(stream MessageReq) returns (stream MessageRes); }`
				// Will look in gql like this...
				// `type Query        { messagesLoopSend(in: MessageReq): Boolean! }
				// type Subscription { messagesLoop: MessageRes }`
			}

			if rpc.GetServerStreaming() {
				p.subscriptions = append(p.subscriptions, m)
			} else {
				switch p.getMethodType(rpc) {
				case gql.Type_DEFAULT:
					switch ttt := p.getServiceType(svc); ttt {
					case gql.Type_DEFAULT, gql.Type_MUTATION:
						p.mutations = append(p.mutations, m)
					case gql.Type_QUERY:
						p.queries = append(p.queries, m)
					}
				case gql.Type_MUTATION:
					p.mutations = append(p.mutations, m)
				case gql.Type_QUERY:
					p.queries = append(p.queries, m)
				}
			}

			p.inputs[rpc.GetInputType()] = &Type{}
			p.types[rpc.GetOutputType()] = &Type{}
		}
	}

	// get all input messages
	for inputName := range p.inputs {
		p.fillTypeMap(inputName, p.inputs)
	}

	// get all type messages
	for inputName := range p.types {
		p.fillTypeMap(inputName, p.types)
	}

	p.fillGqlModels(p.inputs, true)
	p.fillGqlModels(p.types, false)
	p.fillGqlModels(p.enums, false)

	// Start rendering
	p.Schema.P("# Code generated by protoc-gen-gogql. DO NOT EDIT")
	p.Schema.P("# source: ", file.GetName(), "\n")
	// TODO: add comments from proto to gql for documentation purposes

	// render gql inputs
	for _, m := range p.inputs {
		if m.isEmpty() {
			continue
		}
		p.Schema.P("input ", p.gqlModelNames[m], " {")
		p.Schema.In()
		for _, f := range m.GetField() {
			opts := p.getGqlFieldOptions(f)
			if opts != nil {
				if opts.Dirs != nil {
					*opts.Dirs = strings.Trim(*opts.Dirs, " ")
				}
			}

			if typ, ok := p.inputs[f.GetTypeName()]; ok {
				if typ.isEmpty() {
					continue
				}
			}

			p.Schema.P(ToLowerFirst(generator.CamelCase(f.GetName())), ": ", p.graphQLType(f, true), " ", opts.GetDirs())
		}
		p.Schema.Out()
		p.Schema.P("}\n")
	}

	// render gql types
	for _, m := range p.types {
		if m.isEmpty() {
			continue
		}
		p.Schema.P("type ", p.gqlModelNames[m], " {")
		p.Schema.In()
		for _, f := range m.GetField() {
			opts := p.getGqlFieldOptions(f)
			if opts != nil {
				if opts.Params != nil {
					*opts.Params = "(" + strings.Trim(*opts.Params, " ") + ")"
				}
				if opts.Dirs != nil {
					*opts.Dirs = strings.Trim(*opts.Dirs, " ")
				}
			}

			if typ, ok := p.inputs[f.GetTypeName()]; ok {
				if typ.isEmpty() {
					continue
				}
			}

			p.Schema.P(ToLowerFirst(generator.CamelCase(f.GetName())), opts.GetParams(), ": ", p.graphQLType(f, false), " ", opts.GetDirs())
		}
		p.Schema.Out()
		p.Schema.P("}\n")
	}

	// render enums
	for _, e := range p.enums {
		p.Schema.P("enum ", p.gqlModelNames[e], " {")
		p.Schema.In()
		for _, v := range e.GetValue() {
			p.Schema.P(v.GetName())
		}
		p.Schema.Out()
		p.Schema.P("}\n")
	}

	renderMethod := func(methods []*Method) {
		in, out := "", "Boolean!"
		for _, m := range methods {
			if typ, ok := p.inputs[m.InputType]; ok {
				if !typ.isEmpty() {
					in = "(in: " + p.gqlModelNames[p.inputs[m.InputType]] + ")"
				}
			}
			if typ, ok := p.types[m.OutputType]; ok {
				if !typ.isEmpty() {
					out = p.gqlModelNames[p.types[m.OutputType]]
				}
			}

			p.Schema.P(m.Name, in, ": ", out)
		}
	}

	if len(p.mutations) > 0 {
		p.Schema.P("extend type Mutation {")
		p.Schema.In()
		renderMethod(p.mutations)
		p.Schema.Out()
		p.Schema.P("}\n")
	}
	if len(p.queries) > 0 {
		p.Schema.P("extend type Query {")
		p.Schema.In()
		renderMethod(p.queries)
		p.Schema.Out()
		p.Schema.P("}\n")
	}
	if len(p.subscriptions) > 0 {
		p.Schema.P("extend type Subscription {")
		p.Schema.In()
		renderMethod(p.subscriptions)
		p.Schema.Out()
		p.Schema.P("}\n")
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

func ToLowerFirst(s string) string {
	if len(s) > 0 {
		return string(unicode.ToLower(rune(s[0]))) + s[1:]
	}
	return ""
}

func (p *plugin) getMethodType(rpc *descriptor.MethodDescriptorProto) gql.Type {
	if rpc.Options != nil {
		v, err := proto.GetExtension(rpc.Options, gql.E_RpcType)
		if err == nil {
			tt := v.(*gql.Type)
			if tt != nil {
				return *tt
			}
		} else {
			p.Error(err)
		}
	}

	return gql.Type_DEFAULT
}

func (p *plugin) getServiceType(svc *descriptor.ServiceDescriptorProto) gql.Type {
	if svc.Options != nil {
		v, err := proto.GetExtension(svc.Options, gql.E_SvcType)
		if err == nil {
			tt := v.(*gql.Type)
			if tt != nil {
				return *tt
			}
		} else {
			p.Error(err)
		}
	}

	return gql.Type_DEFAULT
}

func (p *plugin) getGqlFieldOptions(field *descriptor.FieldDescriptorProto) *gql.Field {
	if field.Options != nil {
		v, err := proto.GetExtension(field.Options, gql.E_Field)
		if err == nil && v.(*gql.Field) != nil {
			return v.(*gql.Field)
		}
	}
	return nil
}

func getValidatorType(field *descriptor.FieldDescriptorProto) *validator.FieldValidator {
	if field.Options != nil {
		v, err := proto.GetExtension(field.Options, validator.E_Field)
		if err == nil && v.(*validator.FieldValidator) != nil {
			return v.(*validator.FieldValidator)
		}
	}
	return nil
}

func (p *plugin) graphQLType(field *descriptor.FieldDescriptorProto, inputContext bool) string {
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
		p.Warn("mapping a proto group type to graphql is unimplemented")
		//TODO: maybe implement maps as scalars with name containing key type and value type
		// e. i. `map<string, string>` will be represented as: `scalar Map_String_String # map with key: "String" and value: "String"`
		// e. i. `map<AccountID, Account>` will be represented as: `scalar Map_AccountID_Account # map with key: "AccountID" and value: "Account"`
		gqltype = fmt.Sprint("Map")
	case descriptor.FieldDescriptorProto_TYPE_ENUM:
		gqltype = p.gqlModelNames[p.enums[field.GetTypeName()]]
	case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
		if inputContext {
			gqltype = p.gqlModelNames[p.inputs[field.GetTypeName()]]
		} else {
			gqltype = p.gqlModelNames[p.types[field.GetTypeName()]]
		}
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
	if resolveRequired(field) {
		suffix += "!"
	}

	return prefix + gqltype + suffix
}

func getEnum(file *descriptor.FileDescriptorProto, typeName string) *descriptor.EnumDescriptorProto {
	for _, enum := range file.GetEnumType() {
		if enum.GetName() == typeName {
			return enum
		}
	}

	for _, msg := range file.GetMessageType() {
		if nes := getNestedEnum(file, msg, strings.TrimPrefix(typeName, msg.GetName()+".")); nes != nil {
			return nes
		}
	}
	return nil
}

func getNestedEnum(file *descriptor.FileDescriptorProto, msg *descriptor.DescriptorProto, typeName string) *descriptor.EnumDescriptorProto {
	for _, enum := range msg.GetEnumType() {
		if enum.GetName() == typeName {
			return enum
		}
	}

	for _, nes := range msg.GetNestedType() {
		if res := getNestedEnum(file, nes, strings.TrimPrefix(typeName, nes.GetName()+".")); res != nil {
			return res
		}
	}
	return nil
}
