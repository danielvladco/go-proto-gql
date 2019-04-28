package plugin

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/danielvladco/go-proto-gql/pb"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
	"github.com/gogo/protobuf/types"
	golangproto "github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
)

const (
	ScalarBytes      = "Bytes"
	ScalarFloat32    = "Float32"
	ScalarInt64      = "Int64"
	ScalarInt32      = "Int32"
	ScalarUint32     = "Uint32"
	ScalarUint64     = "Uint64"
	ScalarAny        = "Any"
	ScalarDirective  = "__Directive"
	ScalarType       = "__Type"
	ScalarField      = "__Field"
	ScalarEnumValue  = "__EnumValue"
	ScalarInputValue = "__InputValue"
	ScalarSchema     = "__Schema"
	ScalarInt        = "Int"
	ScalarFloat      = "Float"
	ScalarString     = "String"
	ScalarBoolean    = "Boolean"
	ScalarID         = "ID"
)

func NewPlugin() *Plugin {
	return &Plugin{
		fileIndex: -1,
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
		logger: log.New(os.Stderr, "protoc-gen-gogql: ", log.Lshortfile),
	}
}

type Plugin struct {
	*generator.Generator

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

	fileIndex     int
	schemas       []*schema
	gqlModelNames []map[*Type]string // map containing reference to gql type names

	logger *log.Logger
}

func (p *Plugin) GetOneof(ref OneofRef) *Type              { return p.oneofs[p.oneofsRef[ref]] }
func (p *Plugin) Oneofs() map[string]*Type                 { return p.oneofs }
func (p *Plugin) Maps() map[string]*Type                   { return p.maps }
func (p *Plugin) Scalars() map[string]*Type                { return p.scalars }
func (p *Plugin) Enums() map[string]*Type                  { return p.enums }
func (p *Plugin) Types() map[string]*Type                  { return p.types }
func (p *Plugin) Inputs() map[string]*Type                 { return p.inputs }
func (p *Plugin) Subscriptions() Methods                   { return p.subscriptions }
func (p *Plugin) Queries() Methods                         { return p.queries }
func (p *Plugin) Mutations() Methods                       { return p.mutations }
func (p *Plugin) GetSchemaByIndex(index int) *bytes.Buffer { return p.schemas[index].buffer }

func (p *Plugin) IsAny(typeName string) (ok bool) {
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

		_, ok = reflect.New(messageType).Elem().Interface().(*types.Any)
		return ok
	}

	return false
}

// same isEmpty but for mortals
func (p *Plugin) IsEmpty(t *Type) bool { return p.isEmpty(t, make(map[*Type]struct{})) }

// make sure objects are fulled with all types
func (p *Plugin) isEmpty(t *Type, callstack map[*Type]struct{}) bool {
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

func (p *Plugin) Error(err error, msgs ...string) {
	s := strings.Join(msgs, " ") + ":" + err.Error()
	_ = p.logger.Output(2, s)
	os.Exit(1)
}

func (p *Plugin) Warn(msgs ...interface{}) {
	_ = p.logger.Output(2, fmt.Sprintln(append([]interface{}{"WARN: "}, msgs...)...))
}

func (p *Plugin) getImportPath(file *descriptor.FileDescriptorProto) string {
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

	p.Warn(fmt.Errorf("go import path for %s could not be resolved correctly: use option go_package with full package name in this file", file.GetName()))
	return file.GetName()
}

func (p *Plugin) getEnumType(file *descriptor.FileDescriptorProto, typeName string) *Type {
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

// recursively goes trough all fields of the types from messages map and fills it with child types.
// i. e. if message Type1 { Type2: field1 = 1; } exists in the map this function will look up Type2 and add Type2 to the map as well
func (p *Plugin) FillTypeMap(typeName string, messages map[string]*Type, inputField bool) {
	p.fillTypeMap(typeName, messages, inputField, NewCallstack())
}

// creates oneof type with full path name
func (p *Plugin) createOneofFromParent(name string, parent *Type, callstack Callstack, inputField bool, field *descriptor.FieldDescriptorProto, file *descriptor.FileDescriptorProto) {
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
	p.oneofsRef[OneofRef{Parent: parent, Index: field.GetOneofIndex()}] = uniqueOneofName
}

func (p *Plugin) getMessageType(file *descriptor.FileDescriptorProto, typeName string) *Type {
	packagePrefix := "." + file.GetPackage() + "."
	if strings.HasPrefix(typeName, packagePrefix) {
		typeWithoutPrefix := strings.TrimPrefix(typeName, packagePrefix)
		if m := file.GetMessage(typeWithoutPrefix); m != nil {

			// Any is considered to be scalar
			if p.IsAny(typeName) {
				p.scalars[typeName] = &Type{ModelDescriptor: ModelDescriptor{
					PackageDir: "github.com/danielvladco/go-proto-gql/pb", //TODO generate gqlgen.yml
					TypeName:   ScalarAny,
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

func (p *Plugin) fillTypeMap(typeName string, objects map[string]*Type, inputField bool, callstack Callstack) {
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

				// defines scalars for unsupported graphql types
				switch *field.Type {
				case descriptor.FieldDescriptorProto_TYPE_BYTES:
					p.scalars[ScalarBytes] = &Type{ModelDescriptor: ModelDescriptor{PackageDir: "github.com/danielvladco/go-proto-gql/pb", TypeName: ScalarBytes}}

				case descriptor.FieldDescriptorProto_TYPE_FLOAT:
					p.scalars[ScalarFloat32] = &Type{ModelDescriptor: ModelDescriptor{PackageDir: "github.com/danielvladco/go-proto-gql/pb", TypeName: ScalarFloat32}}

				case descriptor.FieldDescriptorProto_TYPE_INT64,
					descriptor.FieldDescriptorProto_TYPE_SINT64,
					descriptor.FieldDescriptorProto_TYPE_SFIXED64:
					p.scalars[ScalarInt64] = &Type{ModelDescriptor: ModelDescriptor{PackageDir: "github.com/danielvladco/go-proto-gql/pb", TypeName: ScalarInt64}}

				case descriptor.FieldDescriptorProto_TYPE_INT32,
					descriptor.FieldDescriptorProto_TYPE_SINT32,
					descriptor.FieldDescriptorProto_TYPE_SFIXED32:
					p.scalars[ScalarInt32] = &Type{ModelDescriptor: ModelDescriptor{PackageDir: "github.com/danielvladco/go-proto-gql/pb", TypeName: ScalarInt32}}

				case descriptor.FieldDescriptorProto_TYPE_UINT32,
					descriptor.FieldDescriptorProto_TYPE_FIXED32:
					p.scalars[ScalarUint32] = &Type{ModelDescriptor: ModelDescriptor{PackageDir: "github.com/danielvladco/go-proto-gql/pb", TypeName: ScalarUint32}}

				case descriptor.FieldDescriptorProto_TYPE_UINT64,
					descriptor.FieldDescriptorProto_TYPE_FIXED64:
					p.scalars[ScalarUint64] = &Type{ModelDescriptor: ModelDescriptor{PackageDir: "github.com/danielvladco/go-proto-gql/pb", TypeName: ScalarUint64}}
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

func (p *Plugin) defineGqlTypes(entries map[string]*Type, suffix ...string) {
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
			p.gqlModelNames[p.fileIndex][sameModel] = sameModel.packageName + entryName
			gqlModels[sameModel.packageName+entryName] = sameModel

			entryName = entry.packageName + entryName
		}

		// add entries to the type map
		p.gqlModelNames[p.fileIndex][entry] = entryName
		gqlModels[entryName] = entry
	}
}

func (p *Plugin) getMethodType(rpc *descriptor.MethodDescriptorProto) gql.Type {
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

func (p *Plugin) getServiceType(svc *descriptor.ServiceDescriptorProto) gql.Type {
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

func (p *Plugin) PrintComments(path ...int) bool {
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
		p.schemas[p.fileIndex].P(nl, "# ", strings.TrimPrefix(line, " "))
		nl = "\n"
	}
	return true
}

func (p *Plugin) checkInitialized() {
	if p.fileIndex == -1 {
		panic("file is not initialized! please use the InitFile(file) method for the file you want to generate code")
	}
}

func (p *Plugin) GraphQLType(field *descriptor.FieldDescriptorProto, messagesIn map[string]*Type) string {
	p.checkInitialized()
	var gqltype string
	switch field.GetType() {
	case descriptor.FieldDescriptorProto_TYPE_DOUBLE:
		gqltype = ScalarFloat

	case descriptor.FieldDescriptorProto_TYPE_BYTES:
		gqltype = ScalarBytes

	case descriptor.FieldDescriptorProto_TYPE_FLOAT:
		gqltype = ScalarFloat32

	case descriptor.FieldDescriptorProto_TYPE_INT64,
		descriptor.FieldDescriptorProto_TYPE_SINT64,
		descriptor.FieldDescriptorProto_TYPE_SFIXED64:
		gqltype = ScalarInt64

	case descriptor.FieldDescriptorProto_TYPE_INT32,
		descriptor.FieldDescriptorProto_TYPE_SINT32,
		descriptor.FieldDescriptorProto_TYPE_SFIXED32:
		gqltype = ScalarInt32

	case descriptor.FieldDescriptorProto_TYPE_UINT32,
		descriptor.FieldDescriptorProto_TYPE_FIXED32:
		gqltype = ScalarUint32

	case descriptor.FieldDescriptorProto_TYPE_UINT64,
		descriptor.FieldDescriptorProto_TYPE_FIXED64:
		gqltype = ScalarUint64

	case descriptor.FieldDescriptorProto_TYPE_BOOL:
		gqltype = ScalarBoolean

	case descriptor.FieldDescriptorProto_TYPE_STRING:
		gqltype = ScalarString

	case descriptor.FieldDescriptorProto_TYPE_GROUP:
		p.Error(errors.New("proto2 groups are not supported please use proto3 syntax"))

	case descriptor.FieldDescriptorProto_TYPE_ENUM:
		gqltype = p.gqlModelNames[p.fileIndex][p.enums[field.GetTypeName()]]

	case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
		if msg, ok := messagesIn[field.GetTypeName()]; ok {
			gqltype = p.gqlModelNames[p.fileIndex][msg]
		} else if scalar, ok := p.scalars[field.GetTypeName()]; ok {
			gqltype = p.gqlModelNames[p.fileIndex][scalar]
		} else if mp, ok := p.maps[field.GetTypeName()]; ok {
			return p.gqlModelNames[p.fileIndex][mp]
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

func (p *Plugin) In() {
	p.checkInitialized()
	p.schemas[p.fileIndex].In()
}

func (p *Plugin) Out() {
	p.checkInitialized()
	p.schemas[p.fileIndex].Out()
}

func (p *Plugin) P(str ...interface{}) {
	p.checkInitialized()
	p.schemas[p.fileIndex].P(str...)
}
func (p *Plugin) GqlModelNames() map[*Type]string {
	p.checkInitialized()
	return p.gqlModelNames[p.fileIndex]
}

func (p *Plugin) defineMethods(file *generator.FileDescriptor) {
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
}

func (p *Plugin) InitFile(file *generator.FileDescriptor) {
	p.fileIndex++
	p.schemas = append(p.schemas, &schema{buffer: &bytes.Buffer{}})
	p.gqlModelNames = append(p.gqlModelNames, make(map[*Type]string))
	// purge data for the current schema
	p.types, p.inputs, p.enums, p.maps, p.scalars, p.oneofs, p.oneofsRef,
		p.subscriptions, p.mutations, p.queries =
		make(map[string]*Type), make(map[string]*Type), make(map[string]*Type),
		make(map[string]*Type), make(map[string]*Type), make(map[string]*Type),
		make(map[OneofRef]string), nil, nil, nil

	p.defineMethods(file)
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
}
