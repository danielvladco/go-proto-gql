package plugin

import (
	"errors"
	"fmt"
	"github.com/jhump/protoreflect/desc"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/danielvladco/go-proto-gql/pb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/vektah/gqlparser/v2/ast"
)

const (
	ScalarBytes      = "Bytes"
	ScalarFloat32    = "Float32"
	ScalarInt64      = "Int64"
	ScalarInt32      = "Int32"
	ScalarUint32     = "Uint32"
	ScalarUint64     = "Uint64"
	ScalarAnyInput   = "AnyInput"
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

func NewPlugin(protoFiles []*desc.FileDescriptor) *Plugin {
	p := &Plugin{
		fileIndex:  0,
		protoFiles: protoFiles,
		//schemas:       []*schema{{buffer: &bytes.Buffer{}}},
		gqlModelNames: []map[*Type]string{make(map[*Type]string)},
		fields:        make(map[FieldKey]*descriptor.FieldDescriptorProto),
		types:         make(map[string]*Type),
		inputs:        make(map[string]*Type),
		enums:         make(map[string]*Type),
		//maps:           make(map[string]*Type),
		oneofs:         make(map[string]*Type),
		oneofsRef:      make(map[OneofRef]string),
		oneofFakeTypes: make(map[string]struct{}),
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
	return p
}

type FieldKey struct {
	T *descriptor.DescriptorProto
	F string
}
type Plugin struct {
	protoFiles []*desc.FileDescriptor
	//*generator.Generator

	// info for the current schema
	mutations     Methods
	queries       Methods
	subscriptions Methods
	inputs        map[string]*Type // map containing string representation of proto message that is used as gql input
	types         map[string]*Type // map containing string representation of proto message that is used as gql type
	enums         map[string]*Type // map containing string representation of proto enum that is used as gql enum
	scalars       map[string]*Type
	//maps           map[string]*Type
	oneofs map[string]*Type

	oneofsRef      map[OneofRef]string // map containing reference to gql oneof names, we need this since oneofs cant fe found by name
	oneofFakeTypes map[string]struct{}

	fields map[FieldKey]*descriptor.FieldDescriptorProto

	fileIndex int
	//schemas       []*schema
	gqlModelNames []map[*Type]string // map containing reference to gql type names

	logger *log.Logger
}

func (p *Plugin) FieldBack(t *descriptor.DescriptorProto, name string) *descriptor.FieldDescriptorProto {
	return p.fields[FieldKey{t, name}]
}

func (p *Plugin) Fields() map[FieldKey]*descriptor.FieldDescriptorProto { return p.fields }
func (p *Plugin) GetOneof(ref OneofRef) *Type                           { return p.oneofs[p.oneofsRef[ref]] }
func (p *Plugin) Oneofs() map[string]*Type                              { return p.oneofs }

//func (p *Plugin) Maps() map[string]*Type                   { return p.maps }
func (p *Plugin) Scalars() map[string]*Type { return p.scalars }
func (p *Plugin) Enums() map[string]*Type   { return p.enums }
func (p *Plugin) Types() map[string]*Type   { return p.types }
func (p *Plugin) Inputs() map[string]*Type  { return p.inputs }
func (p *Plugin) Subscriptions() Methods    { return p.subscriptions }
func (p *Plugin) Queries() Methods          { return p.queries }
func (p *Plugin) Mutations() Methods        { return p.mutations }

//func (p *Plugin) GetSchemaByIndex(index int) *bytes.Buffer { return p.schemas[index].buffer }

func (p *Plugin) IsAny(typeName string) (ok bool) {
	messageType := proto.MessageType(strings.TrimPrefix(typeName, "."))
	if messageType == nil {
		return false
	}
	_, ok = reflect.New(messageType).Elem().Interface().(*any.Any)
	return ok
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
				if f.GetType() != descriptor.FieldDescriptorProto_TYPE_MESSAGE {
					delete(callstack, t)
					return false
				}
				fieldType, ok := p.inputs[f.GetTypeName()]
				if !ok {
					if fieldType, ok = p.types[f.GetTypeName()]; !ok {
						if fieldType, ok = p.enums[f.GetTypeName()]; !ok {
							if fieldType, ok = p.scalars[f.GetTypeName()]; !ok {
								if fieldType, ok = p.oneofs[f.GetTypeName()]; !ok {
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

// recursively goes trough all fields of the types from messages map and fills it with child types.
// i. e. if message Type1 { Type2: field1 = 1; } exists in the map this function will look up Type2 and add Type2 to the map as well
func (p *Plugin) FillTypeMap(typeName string, messages map[string]*Type, inputField bool) {
	p.fillTypeMap(typeName, messages, inputField, NewCallstack())
}
func (p *Plugin) fillTypeMap(typeName string, objects map[string]*Type, inputField bool, callstack Callstack) {
	callstack.Push(typeName) // if add a return statement dont forget to clean stack
	defer callstack.Pop(typeName)

	for _, file := range p.protoFiles {
		if enum := p.getEnumType(file.AsFileDescriptorProto(), typeName); enum != nil {
			if enum.EnumDescriptorProto != nil {
				p.enums[typeName] = enum
				continue
			}
		}

		// Any is considered to be scalar
		if p.IsAny(typeName) {
			if inputField {
				p.scalars[typeName] = &Type{
					//DescriptorProto:     m.DescriptorProto,
					ModelDescriptor: ModelDescriptor{
						//PackageDir: "github.com/danielvladco/go-proto-gql/pb", //TODO generate gqlgen.yml
						TypeName: ScalarAnyInput,
						FQN:      typeName,
					},
				}
			} else {
				p.oneofs[typeName] = &Type{
					//DescriptorProto:     m.DescriptorProto,
					ModelDescriptor: ModelDescriptor{
						//PackageDir: "github.com/danielvladco/go-proto-gql/pb", //TODO generate gqlgen.yml
						TypeName: ScalarAny,
						FQN:      typeName,
					},
				}
			}
			delete(objects, typeName)
			return
		}

		if message := p.getMessageType(typeName); message != nil {
			//if message.DescriptorProto.GetOptions().GetMapEntry() { // if the message is a map
			//	message.UsedAsInput = inputField
			//	p.maps[typeName] = message
			//} else {
			objects[typeName] = message
			//}

			for _, field := range message.GetField() {
				if field.OneofIndex != nil {
					p.createOneofFromParent(message.OneofDecl[field.GetOneofIndex()].GetName(), message, callstack, inputField, field, file.AsFileDescriptorProto(), objects)
					continue
				}

				// defines scalars for unsupported graphql types
				switch *field.Type {
				case descriptor.FieldDescriptorProto_TYPE_BYTES:
					p.scalars[ScalarBytes] = &Type{ModelDescriptor: ModelDescriptor{TypeName: ScalarBytes}}

				case descriptor.FieldDescriptorProto_TYPE_FLOAT:
					p.scalars[ScalarFloat32] = &Type{ModelDescriptor: ModelDescriptor{TypeName: ScalarFloat32}}

				case descriptor.FieldDescriptorProto_TYPE_INT64,
					descriptor.FieldDescriptorProto_TYPE_SINT64,
					descriptor.FieldDescriptorProto_TYPE_SFIXED64:
					p.scalars[ScalarInt64] = &Type{ModelDescriptor: ModelDescriptor{TypeName: ScalarInt64}}

				case descriptor.FieldDescriptorProto_TYPE_INT32,
					descriptor.FieldDescriptorProto_TYPE_SINT32,
					descriptor.FieldDescriptorProto_TYPE_SFIXED32:
					p.scalars[ScalarInt32] = &Type{ModelDescriptor: ModelDescriptor{TypeName: ScalarInt32}}

				case descriptor.FieldDescriptorProto_TYPE_UINT32,
					descriptor.FieldDescriptorProto_TYPE_FIXED32:
					p.scalars[ScalarUint32] = &Type{ModelDescriptor: ModelDescriptor{TypeName: ScalarUint32}}

				case descriptor.FieldDescriptorProto_TYPE_UINT64,
					descriptor.FieldDescriptorProto_TYPE_FIXED64:
					p.scalars[ScalarUint64] = &Type{ModelDescriptor: ModelDescriptor{TypeName: ScalarUint64}}
				}

				if field.GetType() != descriptor.FieldDescriptorProto_TYPE_MESSAGE && field.GetType() != descriptor.FieldDescriptorProto_TYPE_ENUM {
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

// creates oneof type with full path name
func (p *Plugin) createOneofFromParent(name string, parent *Type, callstack Callstack, inputField bool, field *descriptor.FieldDescriptorProto, file *descriptor.FileDescriptorProto, objects map[string]*Type) {
	oneofName := ""
	for _, typeName := range callstack.List() {
		oneofName += "."
		oneofName += objects[typeName].DescriptorProto.GetName()
	}

	// the name of the generated type
	fieldType := "." + strings.Trim(parent.originalPackage, ".") + oneofName + "_" + strings.Title(CamelCase(field.GetName()))
	uniqueOneofName := "." + strings.Trim(parent.originalPackage, ".") + oneofName + "." + name

	// create types for each field
	if !inputField {
		p.oneofFakeTypes[fieldType] = struct{}{}
		objects[fieldType] = &Type{
			DescriptorProto: &descriptor.DescriptorProto{
				Field: []*descriptor.FieldDescriptorProto{{TypeName: field.TypeName, Name: field.Name, Type: field.Type, Options: field.Options}},
			},
			ModelDescriptor: ModelDescriptor{
				Phony:    true,
				TypeName: CamelCaseSlice(strings.Split(strings.Trim(oneofName+"."+strings.Title(field.GetName()), "."), ".")),
				//PackageDir:      parent.PackageDir,
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
				Phony:           true,
				OneofTypes:      map[string]struct{}{fieldType: {}},
				originalPackage: parent.originalPackage,
				packageName:     parent.packageName,
				UsedAsInput:     inputField,
				Index:           0,
				//PackageDir:      parent.PackageDir,
				TypeName: CamelCaseSlice(strings.Split("Is"+strings.Trim(oneofName+"."+name, "."), ".")),
			},
		}
	} else {
		p.oneofs[uniqueOneofName].OneofTypes[fieldType] = struct{}{}
	}
	p.oneofsRef[OneofRef{Parent: parent, Index: field.GetOneofIndex()}] = uniqueOneofName
}

func (p *Plugin) ObjectNamed(typeName string) *desc.MessageDescriptor {
	typeName = strings.TrimPrefix(typeName, ".")
	for _, f := range p.protoFiles {
		msg := f.FindMessage(typeName)
		if msg != nil {
			return msg
		}
	}
	return nil
}

func (p *Plugin) getMessageType(typeName string) *Type {
	if _, ok := p.oneofFakeTypes[typeName]; ok {
		return nil
	}

	m := p.ObjectNamed(typeName)
	if m == nil {
		return nil
	}
	return &Type{DescriptorProto: m.AsDescriptorProto(), ModelDescriptor: ModelDescriptor{
		TypeName: CamelCaseSlice(strings.Split(strings.TrimPrefix(m.GetFullyQualifiedName(), m.GetFile().GetPackage()+"."), ".")),
		//PackageDir:      p.getImportPath(m.File().FileDescriptorProto),
		originalPackage: m.GetFile().GetPackage(),
		packageName:     strings.Replace(m.GetFile().GetPackage(), ".", "_", -1) + "_",
	}}
}

// FIXME
func (p *Plugin) getEnumType(file *descriptor.FileDescriptorProto, typeName string) *Type {
	packagePrefix := "." + file.GetPackage() + "."
	if strings.HasPrefix(typeName, packagePrefix) {
		typeWithoutPrefix := strings.TrimPrefix(typeName, packagePrefix)
		if e := getEnum(file, typeWithoutPrefix); e != nil {
			return &Type{EnumDescriptorProto: e, ModelDescriptor: ModelDescriptor{
				TypeName: CamelCaseSlice(strings.Split(typeWithoutPrefix, ".")),
				//PackageDir:      p.getImportPath(file),
				originalPackage: file.GetPackage(),
				packageName:     strings.Replace(file.GetPackage(), ".", "_", -1) + "_",
			}}
		}
	}
	return nil
}

func (p *Plugin) generateUniqueGraphqlSymbols(gqlModels map[string]*Type, entries map[string]*Type, suffix ...string) {
	for _, entry := range entries {

		entryName := entry.ModelDescriptor.TypeName
		if entryName == "" {
			continue
		}

		for _, sf := range suffix {
			entryName += sf
		}

		originalName := entryName
		for uniqueSuffix := 0; ; uniqueSuffix++ {
			_, ok := gqlModels[entryName]
			if !ok {
				break
			}
			if uniqueSuffix == 0 {
				entryName = entry.packageName + originalName
				continue
			}
			entryName = entry.packageName + originalName + strconv.Itoa(uniqueSuffix)
		}

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
		}
	}

	return gql.Type_DEFAULT
}

func (p *Plugin) checkInitialized() {
	if p.fileIndex == -1 {
		panic("file is not initialized! please use the InitFile(file) method for the file you want to generate code")
	}
}

func (p *Plugin) GraphQLField(t *descriptor.DescriptorProto, f *descriptor.FieldDescriptorProto) string {
	dd := ToLowerFirst(CamelCase(f.GetName()))
	p.fields[FieldKey{t, dd}] = f
	return dd
}

func (p *Plugin) GraphQLType(field *descriptor.FieldDescriptorProto, input bool) (gqltype *ast.Type) {
	p.checkInitialized()
	gqltype = &ast.Type{Position: &ast.Position{}}
	switch field.GetType() {
	case descriptor.FieldDescriptorProto_TYPE_DOUBLE,
		descriptor.FieldDescriptorProto_TYPE_FLOAT:
		gqltype.NamedType = ScalarFloat

	case descriptor.FieldDescriptorProto_TYPE_BYTES:
		gqltype.NamedType = ScalarBytes

	case descriptor.FieldDescriptorProto_TYPE_INT64,
		descriptor.FieldDescriptorProto_TYPE_SINT64,
		descriptor.FieldDescriptorProto_TYPE_SFIXED64,
		descriptor.FieldDescriptorProto_TYPE_INT32,
		descriptor.FieldDescriptorProto_TYPE_SINT32,
		descriptor.FieldDescriptorProto_TYPE_SFIXED32,
		descriptor.FieldDescriptorProto_TYPE_UINT32,
		descriptor.FieldDescriptorProto_TYPE_FIXED32,
		descriptor.FieldDescriptorProto_TYPE_UINT64,
		descriptor.FieldDescriptorProto_TYPE_FIXED64:
		gqltype.NamedType = ScalarInt

	case descriptor.FieldDescriptorProto_TYPE_BOOL:
		gqltype.NamedType = ScalarBoolean

	case descriptor.FieldDescriptorProto_TYPE_STRING:
		gqltype.NamedType = ScalarString

	case descriptor.FieldDescriptorProto_TYPE_GROUP:
		p.Error(errors.New("proto2 groups are not supported please use proto3 syntax"))

	case descriptor.FieldDescriptorProto_TYPE_ENUM:
		gqltype.NamedType = p.gqlModelNames[p.fileIndex][p.enums[field.GetTypeName()]]

	case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
		if input {
			if msg, ok := p.inputs[field.GetTypeName()]; ok {
				// TODO try to bring oneofs here
				//if field.OneofIndex != nil {
				//}

				gqltype.NamedType = p.gqlModelNames[p.fileIndex][msg]
			} else if scalar, ok := p.scalars[field.GetTypeName()]; ok {
				gqltype.NamedType = p.gqlModelNames[p.fileIndex][scalar]
				//} else if mp, ok := p.maps[field.GetTypeName()]; ok {
				//	gqltype.NamedType = p.gqlModelNames[p.fileIndex][mp]
				//	return
			} else {
				panic("unknown proto field type")
			}
		} else {
			if msg, ok := p.types[field.GetTypeName()]; ok {
				// TODO try to bring oneofs here
				//if field.OneofIndex != nil {
				//}

				gqltype.NamedType = p.gqlModelNames[p.fileIndex][msg]
			} else if oneof, ok := p.oneofs[field.GetTypeName()]; ok {
				gqltype.NamedType = p.gqlModelNames[p.fileIndex][oneof]
				//} else if mp, ok := p.maps[field.GetTypeName()]; ok {
				//	gqltype.NamedType = p.gqlModelNames[p.fileIndex][mp]
				//	return
			} else {
				panic("unknown proto field type")
			}
		}

	default:
		panic("unknown proto field type")
	}

	if field.GetLabel() == descriptor.FieldDescriptorProto_LABEL_REPEATED {
		gqltype = ast.ListType(gqltype, &ast.Position{})
		gqltype.Elem.NonNull = true
	}
	if resolveRequired(field) {
		gqltype.NonNull = true
	}

	return
}
func (p *Plugin) GqlModelNames() map[*Type]string {
	p.checkInitialized()
	return p.gqlModelNames[p.fileIndex]
}

func (p *Plugin) defineMethods(file *descriptor.FileDescriptorProto) {
	for svci, svc := range file.GetService() {
		for rpci, rpc := range svc.GetMethod() {
			m := &Method{
				Name:                   ToLowerFirst(svc.GetName()) + strings.Title(rpc.GetName()),
				InputType:              rpc.GetInputType(),
				OutputType:             rpc.GetOutputType(),
				ServiceIndex:           svci,
				Index:                  rpci,
				MethodDescriptorProto:  rpc,
				ServiceDescriptorProto: svc,
			}

			if rpc.GetClientStreaming() && rpc.GetServerStreaming() {
				p.mutations = append(p.mutations, &Method{
					Name:                   ToLowerFirst(svc.GetName()) + strings.Title(rpc.GetName()),
					InputType:              rpc.GetInputType(),
					OutputType:             "",
					Phony:                  true,
					ServiceIndex:           svci,
					Index:                  rpci,
					MethodDescriptorProto:  rpc,
					ServiceDescriptorProto: svc,
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

func (p *Plugin) InitFile(file *desc.FileDescriptor) {
	p.defineMethods(file.AsFileDescriptorProto())
	// get all input messages
	for inputName := range p.inputs {
		p.FillTypeMap(inputName, p.inputs, true)
	}

	// get all type messages
	for typeName := range p.types {
		p.FillTypeMap(typeName, p.types, false)
	}
	gqlTypes := make(map[string]*Type)

	p.generateUniqueGraphqlSymbols(gqlTypes, p.enums)
	p.generateUniqueGraphqlSymbols(gqlTypes, p.scalars)
	p.generateUniqueGraphqlSymbols(gqlTypes, p.oneofs)
	p.generateUniqueGraphqlSymbols(gqlTypes, p.inputs, "Input")
	p.generateUniqueGraphqlSymbols(gqlTypes, p.types)
}
