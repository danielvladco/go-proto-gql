package main

import (
	"fmt"
	"strconv"
	"strings"

	gqlpb "github.com/danielvladco/go-proto-gql/pb"
	"github.com/golang/protobuf/proto"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/danielvladco/go-proto-gql/pkg/generator"
)

func main() {
	var svc = proto.Bool(false)
	var merge = proto.Bool(false)
	protogen.Options{ParamFunc: func(name, value string) error {
		switch name {
		case "svc":
			if b, err := strconv.ParseBool(value); err != nil {
				*svc = b
			}
		case "merge":
			if b, err := strconv.ParseBool(value); err != nil {
				*merge = b
			}
		}
		return nil
	}}.Run(Generate(merge, svc))
}

var (
	ioPkg      = protogen.GoImportPath("io")
	fmtPkg     = protogen.GoImportPath("fmt")
	graphqlPkg = protogen.GoImportPath("github.com/99designs/gqlgen/graphql")
	contextPkg = protogen.GoImportPath("context")
)

func Generate(merge, svc *bool) func(*protogen.Plugin) error {
	return func(p *protogen.Plugin) error {
		goref, _ := generator.NewGoRef(p.Request)
		descs, _ := generator.CreateDescriptorsFromProto(p.Request)
		schemas, err := generator.NewSchemas(descs, *merge, *svc, goref)
		if err != nil {
			return err
		}
		for _, file := range p.Files {
			if !file.Generate {
				continue
			}
			schema := schemas.GetForDescriptor(file)
			g := p.NewGeneratedFile(file.GeneratedFilenamePrefix+".gqlgen.pb.go", file.GoImportPath)
			g.P("package ", file.GoPackageName)
			//InitFile(file)
			for svcIndex, svc := range file.Services {
				svcOpts := generator.GraphqlServiceOptions(svc.Desc.Options())
				if svcOpts != nil && svcOpts.Ignore != nil && *svcOpts.Ignore {
					continue
				}
				g.P(`type `, svc.GoName, `Resolvers struct { Service `, svc.GoName, `Server }`)
				for rpcIndex, rpc := range svc.Methods {
					rpcOpts := generator.GraphqlMethodOptions(rpc.Desc.Options())
					if rpcOpts != nil && rpcOpts.Ignore != nil && *rpcOpts.Ignore {
						continue
					}
					// TODO handle streaming
					if rpc.Desc.IsStreamingClient() || rpc.Desc.IsStreamingServer() {
						continue
					}

					methodName := ""
					switch generator.GetRequestType(rpcOpts, svcOpts) {
					case gqlpb.Type_QUERY:
						methodName = schema.GetMutation().UniqueName(file.Proto.Service[svcIndex], file.Proto.Service[svcIndex].Method[rpcIndex])
					default:
						methodName = schema.GetMutation().UniqueName(file.Proto.Service[svcIndex], file.Proto.Service[svcIndex].Method[rpcIndex])
					}
					methodName = goMethodName(methodName)

					typeIn := g.QualifiedGoIdent(rpc.Input.GoIdent)
					typeOut := g.QualifiedGoIdent(rpc.Output.GoIdent)
					in, inref := ", in *"+typeIn, ", in"
					if IsEmpty(rpc.Input) {
						in, inref = "", ", &"+typeIn+"{}"
					}
					if IsEmpty(rpc.Output) {
						g.P("func (s *", svc.GoName, "Resolvers) ", methodName, "(ctx ", contextPkg.Ident("Context"), in, ") (*bool, error) { _, err := s.Service.", rpc.GoName, "(ctx", inref, ")\n return nil, err }")
					} else {
						g.P("func (s *", svc.GoName, "Resolvers) ", methodName, "(ctx ", contextPkg.Ident("Context"), in, ") (*", typeOut, ", error) { return s.Service.", rpc.GoName, "(ctx", inref, ") }")
					}
				}
			}

			generateMapsAndOneofs(g, file.Messages)
			generateEnums(g, file.Enums)
		}
		return nil
	}
}

func goMethodName(name string) string {
	methodNameSplit := generator.SplitCamelCase(name)
	var methodNameSplitNew []string
	for _, m := range methodNameSplit {
		if m == "id" || m == "Id" {
			m = "ID"
		}
		methodNameSplitNew = append(methodNameSplitNew, m)
	}
	return strings.ReplaceAll(strings.Title(strings.Join(methodNameSplitNew, "")), "_", "")
}

// TODO logic for generation in case the package is different than that of generated protobufs
// This is basically working code

func generateEnums(g *protogen.GeneratedFile, enums []*protogen.Enum) {
	for _, enum := range enums {
		enumType := enum.GoIdent.GoName
		g.P(`
func Marshal`, enumType, `(x `, enumType, `) `, graphqlPkg.Ident("Marshaler"), ` {
	return `, graphqlPkg.Ident("WriterFunc"), `(func(w `, ioPkg.Ident("Writer"), `) {
		_, _ = `, fmtPkg.Ident("Fprintf"), `(w, "%q", x.String())
	})
}

func Unmarshal`, enumType, ` (v interface{}) (`, enumType, `, error) {
	code, ok := v.(string)
	if ok {
		return `, enumType, `(`, enumType, `_value[code]), nil
	}
	return 0, `, fmtPkg.Ident("Errorf"), `("cannot unmarshal `, enumType, ` enum")
}
`)
	}
}

func generateMapsAndOneofs(g *protogen.GeneratedFile, messages []*protogen.Message) {
	//var resolvers []*protogen.Message
	for _, msg := range messages {
		if msg.Desc.IsMapEntry() {
			var (
				//mapName = msg.Fields
				//mapType = fieldGoType(g, protoreflect.MessageKind, msg, nil)
				keyType, _ = fieldGoType(g, msg.Fields[0], true)
				valType, _ = fieldGoType(g, msg.Fields[1], true)
			)

			g.P(`
type `, msg.GoIdent.GoName, `Input = `, msg.GoIdent.GoName, ` 
type `, msg.GoIdent.GoName, ` struct {
	Key `, keyType, ` 
	Value `, valType, `
}
`)
		} else {
			g.P("type ", msg.GoIdent.GoName, "Input = ", msg.GoIdent)
		}
		var mapResolver bool
		for _, f := range msg.Fields {
			if f.Message == nil || !f.Message.Desc.IsMapEntry() {
				continue
			}
			fieldOpts := generator.GraphqlFieldOptions(f.Desc.Options())
			if fieldOpts != nil && fieldOpts.Ignore != nil && *fieldOpts.Ignore {
				continue
			}
			if !mapResolver {
				g.P("type ", msg.GoIdent.GoName, "Resolvers struct{}")
				g.P("type ", msg.GoIdent.GoName, "InputResolvers struct{}")
			}
			mapResolver = true
			g.P(`

func (r `, msg.GoIdent.GoName, `Resolvers) `, f.GoName, `(_ `, contextPkg.Ident("Context"), `, obj *`, msg.GoIdent, `) (list []*`, f.Message.GoIdent, `, _ error) {
	for k,v := range obj.`, f.GoName, ` {
		list = append(list, &`, f.Message.GoIdent, `{
			Key:   k,
			Value: v,
		})
	}
	return
}

func (m `, msg.GoIdent.GoName, `InputResolvers) `, f.GoName, `(_ `, contextPkg.Ident("Context"), `, obj *`, msg.GoIdent, `, data []*`, f.Message.GoIdent, `) error {
	for _, v := range data {
		obj.`, f.GoName, `[v.Key] = v.Value
	}
	return nil
}
`)
		}

		var oneofResolver bool
		for _, oneof := range msg.Oneofs {
			if !oneofResolver {
				g.P("type ", msg.GoIdent.GoName, "Resolvers struct{}")
				g.P("type ", msg.GoIdent.GoName, "InputResolvers struct{}")
			}
			oneofResolver = true
			for _, f := range oneof.Fields {
				goFieldType, isPointer := fieldGoType(g, f, false)
				drawRef := ""
				if !isPointer {
					drawRef = "*"
				}
				g.P(`
func (o `, msg.GoIdent.GoName, `InputResolvers) `, f.GoName, `(_ `, contextPkg.Ident("Context"), `, obj *`, msg.GoIdent, `, data *`, goFieldType, `) error {
	obj.`, oneof.GoName, ` = &`, f.GoIdent.GoName, `{`, f.GoName, `: `, drawRef, `data}
	return nil
}
`)
			}
			g.P(`
func (o `, msg.GoIdent.GoName, `Resolvers) `, oneof.GoName, `(_ `, contextPkg.Ident("Context"), `, obj *`, msg.GoIdent, `) (`, oneof.GoIdent, `, error) {
	return obj.`, oneof.GoName, `, nil
}`)
			g.P(`type `, oneof.GoIdent, " interface{}")
		}
		generateEnums(g, msg.Enums)
		generateMapsAndOneofs(g, msg.Messages)
	}
}

func noUnderscore(s string) string {
	return strings.ReplaceAll(s, "_", "")
}

// same isEmpty but for mortals
func IsEmpty(o *protogen.Message) bool { return isEmpty(o, generator.NewCallstack()) }

// make sure objects are fulled with all objects
func isEmpty(o *protogen.Message, callstack generator.Callstack) bool {
	callstack.Push(o)
	defer callstack.Pop(o)

	if len(o.Fields) == 0 {
		return true
	}
	for _, f := range o.Fields {
		objType := f.Message
		if objType == nil {
			return false
		}

		// check if the call stack already contains a reference to this type and prevent it from calling itself again
		if callstack.Has(objType) {
			return true
		}
		if !isEmpty(objType, callstack) {
			return false
		}
	}

	return true
}

func fieldGoType(g *protogen.GeneratedFile, field *protogen.Field, includePointer bool) (goType string, pointer bool) {
	if field.Desc.IsWeak() {
		return "struct{}", false
	}

	//pointer = field.Desc.HasPresence()
	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		goType = "bool"
	case protoreflect.EnumKind:
		goType = g.QualifiedGoIdent(field.Enum.GoIdent)
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		goType = "int32"
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		goType = "uint32"
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		goType = "int64"
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		goType = "uint64"
	case protoreflect.FloatKind:
		goType = "float32"
	case protoreflect.DoubleKind:
		goType = "float64"
	case protoreflect.StringKind:
		goType = "string"
	case protoreflect.BytesKind:
		goType = "[]byte"
		pointer = false // rely on nullability of slices for presence
	case protoreflect.MessageKind, protoreflect.GroupKind:
		if includePointer {
			goType = "*"
		}
		goType += g.QualifiedGoIdent(field.Message.GoIdent)
		pointer = true
	}
	switch {
	case field.Desc.IsList():
		return "[]" + goType, false
	case field.Desc.IsMap():
		keyType, _ := fieldGoType(g, field.Message.Fields[0], includePointer)
		valType, _ := fieldGoType(g, field.Message.Fields[1], includePointer)
		return fmt.Sprintf("map[%v]%v", keyType, valType), false
	}
	return goType, pointer
}
