package plugin

import (
	"bytes"
	"fmt"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/pkg/errors"
	"log"
	"os"
	"reflect"
	"strconv"

	"strings"
	"unicode"

	"github.com/danielvladco/go-proto-gql"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
	golangproto "github.com/golang/protobuf/proto"
	"github.com/mwitkow/go-proto-validators"
)

type ModelDescriptor struct {
	BuiltIn     bool
	PackageDir  string
	TypeName    string
	UsedAsInput bool
	Index       int
	OneofTypes  map[string]struct{}

	packageName     string
	originalPackage string
}

type Method struct {
	Name         string
	InputType    string
	OutputType   string
	Phony        bool
	ServiceIndex int
	Index        int
}

type Methods []*Method

func (ms Methods) NameExists(name string) bool {
	for _, m := range ms {
		if m.Name == name {
			return true
		}
	}
	return false
}

type Type struct {
	*descriptor.DescriptorProto
	*descriptor.EnumDescriptorProto
	ModelDescriptor
}

func NewPlugin() *plugin {
	return &plugin{
		Schema:    NewBytes(),
		types:     make(map[string]*Type),
		inputs:    make(map[string]*Type),
		enums:     make(map[string]*Type),
		maps:      make(map[string]*Type),
		oneofs:    make(map[string]*Type),
		oneofsRef: make(map[OneofRef]string),
		scalars: map[string]*Type{
			"__Directive":  {ModelDescriptor: ModelDescriptor{BuiltIn: true}},
			"__Type":       {ModelDescriptor: ModelDescriptor{BuiltIn: true}},
			"__Field":      {ModelDescriptor: ModelDescriptor{BuiltIn: true}},
			"__EnumValue":  {ModelDescriptor: ModelDescriptor{BuiltIn: true}},
			"__InputValue": {ModelDescriptor: ModelDescriptor{BuiltIn: true}},
			"__Schema":     {ModelDescriptor: ModelDescriptor{BuiltIn: true}},
			"Int":          {ModelDescriptor: ModelDescriptor{BuiltIn: true}},
			"Float":        {ModelDescriptor: ModelDescriptor{BuiltIn: true}},
			"String":       {ModelDescriptor: ModelDescriptor{BuiltIn: true}},
			"Boolean":      {ModelDescriptor: ModelDescriptor{BuiltIn: true}},
			"ID":           {ModelDescriptor: ModelDescriptor{BuiltIn: true}},
		},
		gqlModelNames: make(map[*Type]string),
		logger:        log.New(os.Stderr, "protoc-gen-gogql: ", log.Lshortfile),
	}
}

type OneofRef struct {
	parent *Type
	index  int32
}

type plugin struct {
	*Schema
	*generator.Generator
	generator.PluginImports

	// info for the current schema
	mutations     Methods
	queries       Methods
	subscriptions Methods
	inputs        map[string]*Type // map containing string representation of proto message that is used as gql input
	types         map[string]*Type // map containing string representation of proto message that is used as gql type
	enums         map[string]*Type // map containing string representation of proto enum that is used as gql enum
	scalars       map[string]*Type
	maps          map[string]*Type
	oneofs        map[string]*Type
	oneofsRef     map[OneofRef]string // map containing reference to gql oneof names, we need this since oneofs cant fe found by name
	gqlModelNames map[*Type]string    // map containing reference to gql type names

	logger *log.Logger
}

func (p *plugin) IsAny(typeName string) (ok bool) {
	if len(typeName) > 0 && typeName[0] == '.' {
		messageType := proto.MessageType(strings.TrimPrefix(typeName, "."))
		if messageType == nil {
			messageType = golangproto.MessageType(strings.TrimPrefix(typeName, "."))
			if messageType == nil {
				return false
			}

			_, ok = reflect.New(messageType).Elem().Interface().(*any.Any)
			return ok
		}

		_, ok = reflect.New(messageType).Elem().Interface().(*any.Any)
		return ok
	}

	return false
}

// same isEmpty but for mortals
func (p *plugin) IsEmpty(t *Type) bool { return p.isEmpty(t, make(map[*Type]struct{})) }

// make sure objects are fulled with all types
func (p *plugin) isEmpty(t *Type, callstack map[*Type]struct{}) bool {
	callstack[t] = struct{}{} // don't forget to free stack on calling return when needed

	if t.DescriptorProto != nil {
		if len(t.DescriptorProto.GetField()) == 0 {
			return true
		} else {
			for _, f := range t.GetField() {
				if !f.IsMessage() {
					delete(callstack, t)
					return false
				}
				fieldType, ok := p.inputs[f.GetTypeName()]
				if !ok {
					if fieldType, ok = p.types[f.GetTypeName()]; !ok {
						if fieldType, ok = p.enums[f.GetTypeName()]; !ok {
							if fieldType, ok = p.scalars[f.GetTypeName()]; !ok {
								if fieldType, ok = p.maps[f.GetTypeName()]; !ok {
									continue
								}
							}
						}
					}
				}

				// check if the call stack already contains a reference to this type and prevent it from calling itself again
				if _, ok := callstack[fieldType]; ok {
					return true
				}
				if !p.isEmpty(fieldType, callstack) {
					delete(callstack, t)
					return false
				}
			}
			delete(callstack, t)
			return true
		}
	}
	delete(callstack, t)
	return false
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
func (p *plugin) GetTypesMap() map[*Type]string            { return p.gqlModelNames }
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
			p.types, p.inputs, p.enums, p.maps, p.scalars, p.oneofs, p.oneofsRef, p.gqlModelNames,
				p.subscriptions, p.mutations, p.queries =
				make(map[string]*Type), make(map[string]*Type), make(map[string]*Type),
				make(map[string]*Type), make(map[string]*Type), make(map[string]*Type),
				make(map[OneofRef]string), make(map[*Type]string), nil, nil, nil
			p.generate(file)
		}
	}
}

func (p *plugin) getImportPath(file *descriptor.FileDescriptorProto) string {
	fileName := strings.Split(file.GetName(), "/")
	gopackageOpt := strings.Split(file.GetOptions().GetGoPackage(), ";")

	if path, ok := p.ImportMap[file.GetName()]; ok {
		return path
	}

	if len(gopackageOpt) == 1 && strings.Contains(gopackageOpt[0], "/") { // check if it is a path or if it is just the package name
		return gopackageOpt[0]
	}
	if len(gopackageOpt) == 2 {
		return gopackageOpt[0]
	}
	if len(fileName) > 1 {
		return strings.Join(fileName[:len(fileName)-1], "/")
	}

	p.Error(fmt.Errorf("go import path for %s could not be resolved correctly: use option go_package with full package name in this file", file.GetName()))
	return "."
}

func (p *plugin) getEnumType(file *descriptor.FileDescriptorProto, typeName string) *Type {
	packagePrefix := "." + file.GetPackage() + "."
	if strings.HasPrefix(typeName, packagePrefix) {
		typeWithoutPrefix := strings.TrimPrefix(typeName, packagePrefix)
		if e := getEnum(file, typeWithoutPrefix); e != nil {
			return &Type{EnumDescriptorProto: e, ModelDescriptor: ModelDescriptor{
				TypeName:        generator.CamelCaseSlice(strings.Split(typeWithoutPrefix, ".")),
				PackageDir:      p.getImportPath(file),
				originalPackage: file.GetPackage(),
				packageName:     strings.Replace(file.GetPackage(), ".", "_", -1) + "_",
			}}
		}
	}
	return nil
}

// recursively passes trough all fields of a all types from a given map and completes it with missing types that ar user as fields
// i. e. if message Type1 { Type2: field1 = 1; } exists in the map this function will look up Type2 and add Type2 to the map as well
func (p *plugin) FillTypeMap(typeName string, messages map[string]*Type, inputField bool) {
	p.fillTypeMap(typeName, messages, inputField, NewCallstack())
}

// creates oneof type with full path name
func (p *plugin) createOneofFromParent(name string, parent *Type, callstack Callstack, inputField bool, field *descriptor.FieldDescriptorProto, file *descriptor.FileDescriptorProto) {
	objects := make(map[string]*Type)
	if inputField {
		objects = p.inputs
	} else {
		objects = p.types
	}

	oneofName := ""
	for _, typeName := range callstack.List() {
		oneofName += "."
		oneofName += objects[typeName].DescriptorProto.GetName()
	}

	// the name of the generated type
	fieldType := "." + strings.Trim(parent.originalPackage, ".") + oneofName + "_" + strings.Title(generator.CamelCase(field.GetName()))

	uniqueOneofName := "." + strings.Trim(parent.originalPackage, ".") + oneofName + "." + name

	// create types for each field
	if !inputField {
		objects[fieldType] = &Type{
			DescriptorProto: &descriptor.DescriptorProto{
				Field: []*descriptor.FieldDescriptorProto{{TypeName: field.TypeName, Name: field.Name, Type: field.Type, Options: field.Options}},
			},
			ModelDescriptor: ModelDescriptor{
				TypeName:        generator.CamelCaseSlice(strings.Split(strings.Trim(oneofName+"."+strings.Title(field.GetName()), "."), ".")),
				PackageDir:      parent.PackageDir,
				Index:           0,
				packageName:     parent.packageName,
				originalPackage: parent.originalPackage,
				UsedAsInput:     inputField,
			},
		}
	}

	// create type for a Oneof construct
	if _, ok := p.oneofs[uniqueOneofName]; !ok {
		p.oneofs[uniqueOneofName] = &Type{
			ModelDescriptor: ModelDescriptor{
				OneofTypes:      map[string]struct{}{fieldType: {}},
				originalPackage: parent.originalPackage,
				packageName:     parent.packageName,
				UsedAsInput:     inputField,
				Index:           0,
				PackageDir:      parent.PackageDir,
				TypeName:        generator.CamelCaseSlice(strings.Split("Is"+strings.Trim(oneofName+"."+name, "."), ".")),
			},
		}
	} else {
		p.oneofs[uniqueOneofName].OneofTypes[fieldType] = struct{}{}
	}
	p.oneofsRef[OneofRef{parent: parent, index: field.GetOneofIndex()}] = uniqueOneofName
}

func (p *plugin) getMessageType(file *descriptor.FileDescriptorProto, typeName string) *Type {
	packagePrefix := "." + file.GetPackage() + "."
	if strings.HasPrefix(typeName, packagePrefix) {
		typeWithoutPrefix := strings.TrimPrefix(typeName, packagePrefix)
		if m := file.GetMessage(typeWithoutPrefix); m != nil {

			// Any is considered to be scalar
			if p.IsAny(typeName) {
				p.scalars[typeName] = &Type{ModelDescriptor: ModelDescriptor{
					PackageDir: "github.com/danielvladco/go-proto-gql/gqltypes", //TODO generate gqlgen.yml
					TypeName:   "Any",
				}}
				return nil
			}

			return &Type{DescriptorProto: m, ModelDescriptor: ModelDescriptor{
				TypeName:        generator.CamelCaseSlice(strings.Split(typeWithoutPrefix, ".")),
				PackageDir:      p.getImportPath(file),
				originalPackage: file.GetPackage(),
				packageName:     strings.Replace(file.GetPackage(), ".", "_", -1) + "_",
			}}
		}
	}
	return nil
}

func (p *plugin) fillTypeMap(typeName string, objects map[string]*Type, inputField bool, callstack Callstack) {
	callstack.Push(typeName) // if add a return statement dont forget to clean stack
	defer callstack.Pop(typeName)

	for _, file := range p.AllFiles().GetFile() {
		if enum := p.getEnumType(file, typeName); enum != nil {
			if enum.EnumDescriptorProto != nil {
				p.enums[typeName] = enum
				continue
			}
		}

		if message := p.getMessageType(file, typeName); message != nil {
			if message.DescriptorProto.GetOptions().GetMapEntry() { // if the message is a map
				message.UsedAsInput = inputField
				p.maps[typeName] = message
			} else {
				objects[typeName] = message
			}

			for _, field := range message.GetField() {
				if field.OneofIndex != nil {
					p.createOneofFromParent(message.OneofDecl[field.GetOneofIndex()].GetName(), message, callstack, inputField, field, file)
				}
				if field.IsBytes() {
					p.scalars["Bytes"] = &Type{ModelDescriptor: ModelDescriptor{
						PackageDir: "github.com/danielvladco/go-proto-gql/gqltypes",
						TypeName:   "Bytes",
					}} // we will need to add a new scalar for this purpose
				}
				if !field.IsMessage() && !field.IsEnum() {
					continue
				}

				// check if already called and do nothing
				if callstack.Has(field.GetTypeName()) {
					continue
				}

				p.fillTypeMap(field.GetTypeName(), objects, inputField, callstack)
			}

			return // we know that it is not possible for multiple messages to have the same name
		}
	}
}

func (p *plugin) defineGqlTypes(entries map[string]*Type, suffix ...string) {
	gqlModels := make(map[string]*Type)
	scalars := make(map[string]struct{})
	for _, scalar := range p.scalars {
		scalars[scalar.TypeName] = struct{}{}
	}

	for _, entry := range entries {
		entryName := entry.ModelDescriptor.TypeName

		// concatenate package name in case if the message has a scalar name, Any is considered a scalar
		if _, ok := scalars[entryName]; ok {
			entryName = entry.packageName + entryName
		} else {

			// add suffix to type if needed but only if it is not a scalar type
			for _, sf := range suffix {
				entryName += sf
			}
		}

		// concatenate package name in case if two types with the same name are found
		if sameModel, ok := gqlModels[entryName]; ok {
			// redesign old type to have package namespace as well
			p.gqlModelNames[sameModel] = sameModel.packageName + entryName
			gqlModels[sameModel.packageName+entryName] = sameModel

			entryName = entry.packageName + entryName
		}

		// add entries to the type map
		p.gqlModelNames[entry] = entryName
		gqlModels[entryName] = entry
	}
}

func (p *plugin) generate(file *generator.FileDescriptor) {
	for svci, svc := range file.GetService() {
		for rpci, rpc := range svc.GetMethod() {
			m := &Method{
				Name:         ToLowerFirst(svc.GetName()) + strings.Title(rpc.GetName()),
				InputType:    rpc.GetInputType(),
				OutputType:   rpc.GetOutputType(),
				ServiceIndex: svci, Index: rpci,
			}

			if rpc.GetClientStreaming() && rpc.GetServerStreaming() {
				p.mutations = append(p.mutations, &Method{
					Name:       ToLowerFirst(svc.GetName()) + strings.Title(rpc.GetName()),
					InputType:  rpc.GetInputType(),
					OutputType: "", Phony: true, ServiceIndex: svci, Index: rpci,
				})
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
		p.FillTypeMap(inputName, p.inputs, true)
	}

	// get all type messages
	for inputName := range p.types {
		p.FillTypeMap(inputName, p.types, false)
	}

	//p.mapToScalar()
	p.defineGqlTypes(p.inputs, "Input")
	p.defineGqlTypes(p.types)
	p.defineGqlTypes(p.enums)
	p.defineGqlTypes(p.maps)
	p.defineGqlTypes(p.scalars)
	p.defineGqlTypes(p.oneofs)

	// Start rendering
	p.Schema.P("# Code generated by protoc-gen-gogql. DO NOT EDIT")
	p.Schema.P("# source: ", file.GetName(), "\n")
	p.PrintComments(2)
	// TODO: add comments from proto to gql for documentation purposes to all elements. Currently are supported only a few

	// scalars
	for _, scalar := range p.scalars {
		if !scalar.BuiltIn {
			p.Schema.P("scalar ", p.gqlModelNames[scalar], "\n")
		}
	}
	// maps
	for _, m := range p.maps {
		key, val := m.GetMapFields()

		messagesIn := make(map[string]*Type)
		if m.UsedAsInput {
			messagesIn = p.inputs
		} else {
			messagesIn = p.types
		}
		p.Schema.P("# map with key: '", p.graphQLType(key, nil), "' and value: '", p.graphQLType(val, messagesIn), "'")
		p.Schema.P("scalar ", p.gqlModelNames[m], "\n")
	}

	// render gql inputs
	for protoTypeName, m := range p.inputs {
		if p.IsEmpty(m) || p.IsAny(protoTypeName) {
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
				if p.IsEmpty(typ) {
					continue
				}
			}
			oneof := ""
			if f.OneofIndex != nil {
				oneof = fmt.Sprintf("@Oneof(name: %q)", p.gqlModelNames[p.oneofs[p.oneofsRef[OneofRef{parent: m, index: f.GetOneofIndex()}]]])
			}
			p.Schema.P(ToLowerFirst(generator.CamelCase(f.GetName())), ": ", p.graphQLType(f, p.inputs), " ", oneof, " ", opts.GetDirs())
		}
		p.Schema.Out()
		p.Schema.P("}\n")
	}

	// render gql types
	for protoTypeName, m := range p.types {
		if p.IsEmpty(m) || p.IsAny(protoTypeName) {
			continue
		}
		p.Schema.P("type ", p.gqlModelNames[m], " {")
		p.Schema.In()
		oneofs := make(map[OneofRef]struct{})
		for _, f := range m.GetField() {
			if f.OneofIndex != nil {
				oneofRef := OneofRef{parent: m, index: *f.OneofIndex}
				if _, ok := oneofs[oneofRef]; !ok {
					p.Schema.P(ToLowerFirst(generator.CamelCase(m.OneofDecl[*f.OneofIndex].GetName())), ": ", p.gqlModelNames[p.oneofs[p.oneofsRef[oneofRef]]])
				}
				oneofs[oneofRef] = struct{}{}
				continue
			}
			opts := p.getGqlFieldOptions(f)
			if opts != nil {
				if opts.Params != nil {
					*opts.Params = "(" + strings.Trim(*opts.Params, " ") + ")"
				}
				if opts.Dirs != nil {
					*opts.Dirs = strings.Trim(*opts.Dirs, " ")
				}
			}

			if typ, ok := p.types[f.GetTypeName()]; ok {
				if p.IsEmpty(typ) {
					continue
				}
			}

			p.Schema.P(ToLowerFirst(generator.CamelCase(f.GetName())), opts.GetParams(), ": ", p.graphQLType(f, p.types), " ", opts.GetDirs())
		}
		p.Schema.Out()
		p.Schema.P("}\n")
	}

	for _, oneof := range p.oneofs {
		p.Schema.P("union ", p.gqlModelNames[oneof], " = ", strings.Join(func() (ss []string) {
			for typeName := range oneof.OneofTypes {
				ss = append(ss, p.gqlModelNames[p.types[typeName]])
			}
			return
		}(), " | "), "\n")
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
		for _, m := range methods {
			in, out := "", "Boolean!"

			// some scalars can be used as inputs such as 'Any'
			if scalar, ok := p.scalars[m.InputType]; ok {
				if !p.IsEmpty(scalar) {
					in = "(in: " + p.gqlModelNames[scalar] + ")"
				}
			} else if input, ok := p.inputs[m.InputType]; ok {
				if !p.IsEmpty(input) {
					in = "(in: " + p.gqlModelNames[input] + ")"
				}
			}
			if scalar, ok := p.scalars[m.OutputType]; ok {
				if !p.IsEmpty(scalar) {
					in = "(in: " + p.gqlModelNames[scalar] + ")"
				}
			} else if typ, ok := p.types[m.OutputType]; ok {
				if !p.IsEmpty(typ) {
					out = p.gqlModelNames[typ]
				}
			}

			p.PrintComments(6, m.ServiceIndex, 2, m.Index)
			if m.Phony {
				p.Schema.P("# See the Subscription with the same name")
			}
			p.Schema.P(m.Name, in, ": ", out)
		}
	}

	if len(p.mutations) > 0 {
		p.Schema.P("type Mutation {")
		p.Schema.In()
		renderMethod(p.mutations)
		p.Schema.Out()
		p.Schema.P("}\n")
	}
	if len(p.queries) > 0 {
		p.Schema.P("type Query {")
		p.Schema.In()
		renderMethod(p.queries)
		p.Schema.Out()
		p.Schema.P("}\n")
	}
	if len(p.subscriptions) > 0 {
		p.Schema.P("type Subscription {")
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

func (p *plugin) PrintComments(path ...int) bool {
	nl := ""
	comments := p.Comments(strings.Join(func() (pp []string) {
		for p := range path {
			pp = append(pp, strconv.Itoa(p))
		}
		return
	}(), ","))
	if comments == "" {
		return false
	}
	for _, line := range strings.Split(comments, "\n") {
		p.Schema.P(nl, "# ", strings.TrimPrefix(line, " "))
		nl = "\n"
	}
	return true
}

func (p *plugin) graphQLType(field *descriptor.FieldDescriptorProto, messagesIn map[string]*Type) string {
	var gqltype string
	switch field.GetType() {
	case descriptor.FieldDescriptorProto_TYPE_DOUBLE, descriptor.FieldDescriptorProto_TYPE_FLOAT:
		gqltype = "Float"
	case descriptor.FieldDescriptorProto_TYPE_INT64, descriptor.FieldDescriptorProto_TYPE_UINT64,
		descriptor.FieldDescriptorProto_TYPE_INT32, descriptor.FieldDescriptorProto_TYPE_FIXED64,
		descriptor.FieldDescriptorProto_TYPE_FIXED32,
		descriptor.FieldDescriptorProto_TYPE_UINT32,
		descriptor.FieldDescriptorProto_TYPE_SFIXED32,
		descriptor.FieldDescriptorProto_TYPE_SFIXED64,
		descriptor.FieldDescriptorProto_TYPE_SINT32,
		descriptor.FieldDescriptorProto_TYPE_SINT64:
		gqltype = "Int"
	case descriptor.FieldDescriptorProto_TYPE_BOOL:
		gqltype = "Boolean"
	case descriptor.FieldDescriptorProto_TYPE_STRING:
		gqltype = "String"
	case descriptor.FieldDescriptorProto_TYPE_GROUP:
		p.Error(errors.New("proto2 groups are not supported please use proto3 syntax"))
	case descriptor.FieldDescriptorProto_TYPE_BYTES:
		gqltype = "Bytes"
	case descriptor.FieldDescriptorProto_TYPE_ENUM:
		gqltype = p.gqlModelNames[p.enums[field.GetTypeName()]]
	case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
		if msg, ok := messagesIn[field.GetTypeName()]; ok {
			gqltype = p.gqlModelNames[msg]
		} else if scalar, ok := p.scalars[field.GetTypeName()]; ok {
			gqltype = p.gqlModelNames[scalar]
		} else if mp, ok := p.maps[field.GetTypeName()]; ok {
			return p.gqlModelNames[mp]
		} else {
			panic("unknown proto field type")
		}
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

func getValidatorType(field *descriptor.FieldDescriptorProto) *validator.FieldValidator {
	if field.Options != nil {
		v, err := proto.GetExtension(field.Options, validator.E_Field)
		if err == nil && v.(*validator.FieldValidator) != nil {
			return v.(*validator.FieldValidator)
		}
	}
	return nil
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
