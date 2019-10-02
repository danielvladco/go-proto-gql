package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	gqlplugin "github.com/danielvladco/go-proto-gql/plugin"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/generator"
)

func main() {
	gen := generator.New()

	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		gen.Error(err, "reading input")
	}

	if err := proto.Unmarshal(data, gen.Request); err != nil {
		gen.Error(err, "parsing input proto")
	}

	if len(gen.Request.FileToGenerate) == 0 {
		gen.Fail("no files to generate")
	}

	gen.CommandLineParameters(gen.Request.GetParameter())
	gen.WrapTypes()
	gen.SetPackageNames()
	gen.BuildTypeNameMap()
	generator.RegisterPlugin(&plugin{
		enums:    make(map[string]struct{}),
		messages: make(map[string]struct{}),
		Plugin:   gqlplugin.NewPlugin(),
	})
	gen.GenerateAllFiles()

	for i := 0; i < len(gen.Response.File); i++ {
		gen.Response.File[i].Name = proto.String(strings.Replace(*gen.Response.File[i].Name, ".pb.go", ".gqlgen.pb.go", -1))
	}

	data, err = proto.Marshal(gen.Response)
	if err != nil {
		gen.Error(err, "failed to marshal output proto")
	}
	_, err = os.Stdout.Write(data)
	if err != nil {
		gen.Error(err, "failed to write output proto")
	}
}

type plugin struct {
	*gqlplugin.Plugin

	enums    map[string]struct{}
	messages map[string]struct{}
}

func (p *plugin) Name() string                                   { return "gogqlgen" }
func (p *plugin) Init(g *generator.Generator)                    { p.Generator = g }
func (p *plugin) GenerateImports(file *generator.FileDescriptor) {}
func (p *plugin) Generate(file *generator.FileDescriptor) {
	ioPkg := p.AddImport("io")
	graphqlPkg := p.AddImport("github.com/danielvladco/go-proto-gql/pb")
	jsonPkg := p.AddImport("encoding/json")
	contextPkg := p.AddImport("context")

	p.Scalars()
	for _, r := range p.Request.FileToGenerate {
		if r == file.GetName() {
			p.InitFile(file)
			for _, svc := range file.GetService() {
				p.Generator.P(`type `, svc.GetName(), `GQLServer struct { Service `, svc.GetName(), `Server }`)
				for _, rpc := range svc.GetMethod() {
					if rpc.GetClientStreaming() || rpc.GetServerStreaming() {
						continue
					}
					methodName := strings.Replace(generator.CamelCase(svc.GetName()+rpc.GetName()), "_", "", -1)
					methodNameSplit := gqlplugin.SplitCamelCase(methodName)
					var methodNameSplitNew []string
					for _, m := range methodNameSplit {
						if m == "id" || m == "Id" {
							m = "ID"
						}
						methodNameSplitNew = append(methodNameSplitNew, m)
					}
					methodName = strings.Join(methodNameSplitNew, "")

					p.RecordTypeUse(rpc.GetInputType())
					typeInObj := p.ObjectNamed(rpc.GetInputType())
					p.AddImport(typeInObj.GoImportPath())

					p.RecordTypeUse(rpc.GetOutputType())
					typeOutObj := p.ObjectNamed(rpc.GetOutputType())
					typeIn := generator.CamelCaseSlice(typeInObj.TypeName())
					typeOut := generator.CamelCaseSlice(typeOutObj.TypeName())
					if p.DefaultPackageName(typeInObj) != "" {
						typeIn = string(p.AddImport(typeInObj.GoImportPath())) + "." + typeIn
					}
					if p.DefaultPackageName(typeOutObj) != "" {
						typeOut = string(p.AddImport(typeOutObj.GoImportPath())) + "." + typeOut
					}
					in, inref := ", in *"+typeIn, ", in"
					if p.IsEmpty(p.Inputs()[rpc.GetInputType()]) {
						in, inref = "", ", &"+typeIn+"{}"
					}
					if p.IsEmpty(p.Types()[rpc.GetOutputType()]) {
						p.Generator.P("func (s *", svc.GetName(), "GQLServer) ", methodName, "(ctx ", contextPkg, ".Context", in, ") (*bool, error) { _, err := s.Service.", generator.CamelCase(rpc.GetName()), "(ctx", inref, ")\n return nil, err }")
					} else {
						p.Generator.P("func (s *", svc.GetName(), "GQLServer) ", methodName, "(ctx ", contextPkg, ".Context", in, ") (*", typeOut, ", error) { return s.Service.", generator.CamelCase(rpc.GetName()), "(ctx", inref, ") }")
					}
				}
			}
			for _, msg := range file.GetMessageType() {
				if msg.GetOptions().GetMapEntry() {
					println("found a map!")
					key, value := msg.GetField()[0], msg.GetField()[1]

					mapName := generator.CamelCaseSlice(msg.GetReservedName())
					keyType, _ := p.GoType(&generator.Descriptor{DescriptorProto: msg}, key)
					valType, _ := p.GoType(&generator.Descriptor{DescriptorProto: msg}, value)

					mapType := fmt.Sprintf("map[%s]%s", keyType, valType)
					p.Generator.P(`
func Marshal`, mapName, `(mp `, mapType, `) `, graphqlPkg, `.Marshaler {
	return `, graphqlPkg, `.WriterFunc(func(w `, ioPkg, `.Writer) {
		err := `, jsonPkg, `.NewEncoder(w).Encode(mp)
		if err != nil {
			panic("this map type is not supported")
		}
	})
}

func Unmarshal`, mapName, `(v interface{}) (mp `, mapType, `, err error) {
	switch vv := v.(type) {
	case []byte:
		err = `, jsonPkg, `.Unmarshal(vv, &mp)
		return mp, err
	case `, jsonPkg, `.RawMessage:
		err = `, jsonPkg, `.Unmarshal(vv, &mp)
		return mp, err
	default:
		return nil, fmt.Errorf("%T is not `, mapType, `", v)
	}
}
`)
				}
				for _, oneof := range msg.OneofDecl {
					oneofName := append(msg.GetReservedName(), oneof.GetName())
					p.Generator.P(`type Is`, generator.CamelCaseSlice(oneofName),
						" interface{\n\tis", generator.CamelCaseSlice(oneofName), "()\n}")
				}
			}
			for _, enum := range file.GetEnumType() {
				//log.Fatal(enum, enum.GetReservedName())
				enumType := generator.CamelCase(enum.GetName())
				p.Generator.P(`
func (c *`, enumType, `) UnmarshalGQL(v interface{}) error {
	code, ok := v.(string)
	if ok {
		*c = `, enumType, `(`, enumType, `_value[code])
		return nil
	}
	return fmt.Errorf("cannot unmarshal `, enumType, ` enum")
}

func (c `, enumType, `) MarshalGQL(w `, ioPkg, `.Writer) {
	fmt.Fprintf(w, "%q", c.String())
}
`)
			}

		}
	}
}
