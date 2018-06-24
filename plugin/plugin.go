package plugin

import (
	"reflect"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
	"log"
	"github.com/elysio-co/go-proto-validators"
	"github.com/danielvladco/go-proto-gql"
	"fmt"
)

type plugin struct {
	*generator.Generator
	generator.PluginImports
	//regexPkg      generator.Single
	//fmtPkg        generator.Single
	//protoPkg      generator.Single
	//validatorPkg  generator.Single
	messages      map[string]*generator.Descriptor
	useGogoImport bool
}

func NewPlugin(useGogoImport bool) generator.Plugin {
	return &plugin{useGogoImport: useGogoImport, messages: make(map[string]*generator.Descriptor)}
}

func (p *plugin) Name() string {
	return "gql"
}

func (p *plugin) Init(g *generator.Generator) {
	p.Generator = g
}

func (p *plugin) Generate(file *generator.FileDescriptor) {
	p.PluginImports = generator.NewPluginImports(p.Generator)
	for _, message := range file.Messages() {
		//if len(message.DescriptorProto.Field) == 0 {
		//	continue
		//}

		if message.DescriptorProto.GetOptions().GetMapEntry() {
			//TODO
			log.Println("implement maps")
			continue
		}

		nested := ""
		ln := len(message.GetNestedType())
		for i, n := range message.GetNestedType() {
			nested += n.GetName()
			if ln != i {
				nested += "."
			}
		}
		message.GetNestedType()

		p.messages["."+file.PackageName()+"."+nested+"."+message.GetName()] = message
	}

	//if !p.useGogoImport {
	//	vanity.TurnOffGogoImport(file.FileDescriptorProto)
	//}
	//p.regexPkg = p.NewImport("regexp")
	//p.fmtPkg = p.NewImport("fmt")
	//p.validatorPkg = p.NewImport("github.com/mwitkow/go-proto-validators")

	//for _, msg := range file.Messages() {
	//	if msg.DescriptorProto.GetOptions().GetMapEntry() {
	//		continue
	//	}
	//	if gogoproto.IsProto3(file.FileDescriptorProto) {
	//		p.generateProto3Message(file, msg)
	//	} else {
	//		p.generateProto2Message(file, msg)
	//	}
	//}

	//mutations := make(chan string, 100)
	//queries := make(chan string, 100)
	//end := make(chan struct{})
	p.P("var schema = `")
	p.P("type schema {\n\tquery: Query\n\tmutation: Mutation\n}")
	//go func() {
	//	buff := &bytes.Buffer{}
	//	for {
	//		select {
	//		case rpc := <-mutations:
	//			buff.WriteString(rpc)
	//			buff.WriteString("\n\t")
	//		case <-end:
	//			break
	//		}
	//	}
	//	p.P("type Mutation {\n\t", buff.String(), "}")
	//}()
	//
	//go func() {
	//	print(11112)
	//	buff := &bytes.Buffer{}
	//	for {
	//		select {
	//		case rpc := <-queries:
	//			buff.WriteString(rpc)
	//		case <-end:
	//			break
	//		}
	//	}
	//
	//	p.P("type Mutation {\n\t", buff.String(), "\n}")
	//}()

	for _, svc := range file.Service {
		for _, rpc := range svc.Method {
			//str := fmt.Sprintln()
			switch getMethodType(rpc) {
			case gql.Type_MUTATION:

				//mutations <- str
			case gql.Type_QUERY:
				//queries <- str
			case gql.Type_SUBSCRIPTION:
				//not sure yet
			}
			p.P("type Mutation {")
			p.In()
			_, ok := p.messages[rpc.GetInputType()]
			p.P("# please ", ok)
			p.P(file.Package, "_", svc.Name, "_", rpc.Name, "(input: ", rpc.GetInputType(), "!): ", rpc.GetOutputType(), "_Out")
			p.Out()
			p.P("}")
		}
	}
	m := file.GetMessage("New.Type.E")
	p.P("#11 ", m.GetName(), "#22 ",file.PackageName())
	for n, m := range p.messages {

		p.P("#", n, " > ", strings.Join(m.GetReservedName(), "."))
	}
	//fmt.Printf("%+v", p.messages)

	//end <- struct {}{}
	p.P("`")

	//go func() {
	//	buff := &bytes.Buffer{}
	//	for {
	//		select {
	//		case input := <-inputs:
	//			file.GetMessage(*input)
	//		case time.After(2 * time.Second):
	//			return
	//		}
	//	}
	//	p.P(buff.String())
	//}()
	//
	//for _, msg := range file.Messages() {
	//	switch {
	//	case contains(inputs, msg.Name):
	//		for _, field := range msg.Field {
	//			if field.Type != nil && *field.Type == descriptor.FieldDescriptorProto_TYPE_MESSAGE {
	//
	//			}
	//		}
	//	case contains(outputs, msg.Name):
	//
	//	default:
	//		p.P("# unused message ", file.Name, "_", msg.Name)
	//	}
	//}
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

func (p *plugin) isSupportedInt(field *descriptor.FieldDescriptorProto) bool {
	switch *(field.Type) {
	case descriptor.FieldDescriptorProto_TYPE_INT32, descriptor.FieldDescriptorProto_TYPE_INT64:
		return true
	case descriptor.FieldDescriptorProto_TYPE_UINT32, descriptor.FieldDescriptorProto_TYPE_UINT64:
		return true
	case descriptor.FieldDescriptorProto_TYPE_SINT32, descriptor.FieldDescriptorProto_TYPE_SINT64:
		return true
	}
	return false
}

func (p *plugin) isSupportedFloat(field *descriptor.FieldDescriptorProto) bool {
	switch *(field.Type) {
	case descriptor.FieldDescriptorProto_TYPE_FLOAT, descriptor.FieldDescriptorProto_TYPE_DOUBLE:
		return true
	case descriptor.FieldDescriptorProto_TYPE_FIXED32, descriptor.FieldDescriptorProto_TYPE_FIXED64:
		return true
	case descriptor.FieldDescriptorProto_TYPE_SFIXED32, descriptor.FieldDescriptorProto_TYPE_SFIXED64:
		return true
	}
	return false
}

func (p *plugin) generateProto2Message(file *generator.FileDescriptor, message *generator.Descriptor) {
	log.Fatal("proto2 syntax is not supported")
}

func (p *plugin) generateProto3Message(file *generator.FileDescriptor, message *generator.Descriptor) {
	ccTypeName := generator.CamelCaseSlice(message.TypeName())
	p.P(`func (this *`, ccTypeName, `) Validate() error {`)
	p.In()
	for _, field := range message.Field {
		//fieldName := p.GetOneOfFieldName(message, field)
		//variableName := "this." + fieldName
		repeated := field.IsRepeated()
		// Golang's proto3 has no concept of unset primitive fields
		//nullable := (gogoproto.IsNullable(field) || !gogoproto.ImportsGoGoProto(file.FileDescriptorProto)) && field.IsMessage()
		switch {
		case p.fieldIsProto3Map(file, message, field):
			p.P(`// graphql does not support maps`)
			continue
		case field.OneofIndex != nil:

		case repeated:

		case field.IsString():
		case p.isSupportedInt(field):
		case p.isSupportedFloat(field):
		case field.IsBytes():
		case field.IsMessage():
		case field.IsMessage():

		}
	}
	p.P(`return nil`)
	p.Out()
	p.P(`}`)
}

func (p *plugin) fieldIsProto3Map(file *generator.FileDescriptor, message *generator.Descriptor, field *descriptor.FieldDescriptorProto) bool {
	// Context from descriptor.proto
	// Whether the message is an automatically generated map entry type for the
	// maps field.
	//
	// For maps fields:
	//     map<KeyType, ValueType> map_field = 1;
	// The parsed descriptor looks like:
	//     message MapFieldEntry {
	//         option map_entry = true;
	//         optional KeyType key = 1;
	//         optional ValueType value = 2;
	//     }
	//     repeated MapFieldEntry map_field = 1;
	//
	// Implementations may choose not to generate the map_entry=true message, but
	// use a native map in the target language to hold the keys and values.
	// The reflection APIs in such implementions still need to work as
	// if the field is a repeated message field.
	//
	// NOTE: Do not set the option in .proto files. Always use the maps syntax
	// instead. The option should only be implicitly set by the proto compiler
	// parser.
	if field.GetType() != descriptor.FieldDescriptorProto_TYPE_MESSAGE || !field.IsRepeated() {
		return false
	}
	typeName := field.GetTypeName()
	var msg *descriptor.DescriptorProto
	if strings.HasPrefix(typeName, ".") {
		// Fully qualified case, look up in global map, must work or fail badly.
		msg = p.ObjectNamed(field.GetTypeName()).(*generator.Descriptor).DescriptorProto
	} else {
		// Nested, relative case.
		msg = file.GetNestedMessage(message.DescriptorProto, field.GetTypeName())
	}
	return msg.GetOptions().GetMapEntry()
}

func (p *plugin) validatorWithAnyConstraint(fv *validator.FieldValidator) bool {
	if fv == nil {
		return false
	}

	// Need to use reflection in order to be future-proof for new types of constraints.
	v := reflect.ValueOf(fv)
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).Interface() != nil {
			return true
		}
	}
	return false
}

func (p *plugin) graphQLType(message *generator.Descriptor, field *descriptor.FieldDescriptorProto) string {
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
		//TODO
		//panic("mapping a proto group type to graphql is unimplemented")
	case descriptor.FieldDescriptorProto_TYPE_ENUM:
		//TODO
		//panic("mapping a proto enum type to graphql is unimplemented")
	case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
		// TODO: fix this to be more robust about imported objects
		//mobj := p.ObjectNamed(field.GetTypeName())
		//// fmt.Fprint(os.Stderr, mobj.PackageName())
		//if strings.HasPrefix(mobj.PackageName(), opseeTypes) {
		//	gqltype = fmt.Sprint(schemaPkgName.Use(), ".", generator.CamelCaseSlice(mobj.TypeName()))
		//	break
		//}
		//
		//gqltype = p.graphQLTypeVarName(mobj)
	case descriptor.FieldDescriptorProto_TYPE_BYTES:
		// TODO send bytes
		//gqltype = fmt.Sprint(schemaPkgName.Use(), ".", "ByteString")
	default:
		panic("unknown proto field type")
	}

	//if field.IsRepeated() && !p.IsMap(field) {
	//	gqltype = fmt.Sprint(pkgName.Use(), ".NewList(", gqltype, ")")
	//}
	//
	//if field.IsRequired() {
	//	gqltype = fmt.Sprint(pkgName.Use(), ".NewNonNull(", gqltype, ")")
	//}

	return gqltype
}
