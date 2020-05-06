package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"

	gqlplugin "github.com/danielvladco/go-proto-gql/plugin"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
	"github.com/gogo/protobuf/vanity"
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

	useGogoImport := false
	if gogoimport, ok := gqlplugin.Params(gen)["gogoimport"]; ok {
		useGogoImport, err = strconv.ParseBool(gogoimport)
		if err != nil {
			gen.Error(err, "parsing gogoimport option")
		}
	}

	gen.CommandLineParameters(gen.Request.GetParameter())
	gen.WrapTypes()
	gen.SetPackageNames()
	gen.BuildTypeNameMap()
	gen.GeneratePlugin(&plugin{
		useGogoImport: useGogoImport,
		enums:         make(map[string]struct{}),
		messages:      make(map[string]struct{}),
		Plugin:        gqlplugin.NewPlugin(),
	})

	for i := 0; i < len(gen.Response.File); i++ {
		gen.Response.File[i].Name = proto.String(strings.Replace(*gen.Response.File[i].Name, ".pb.go", ".gqlgen.pb.go", -1))
	}

	// Send back the results.
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
	generator.PluginImports
	ioPkg      generator.Single
	fmtPkg     generator.Single
	graphqlPkg generator.Single
	jsonPkg    generator.Single
	contextPkg generator.Single
	reflectPkg generator.Single

	enums    map[string]struct{}
	messages map[string]struct{}

	useGogoImport bool
}

func (p *plugin) Name() string                { return "gogqlgen" }
func (p *plugin) Init(g *generator.Generator) { p.Generator = g }
func (p *plugin) Generate(file *generator.FileDescriptor) {
	if !p.useGogoImport {
		vanity.TurnOffGogoImport(file.FileDescriptorProto)
	}

	p.PluginImports = generator.NewPluginImports(p.Generator)
	p.fmtPkg = p.NewImport("fmt")
	p.ioPkg = p.NewImport("io")
	p.graphqlPkg = p.NewImport("github.com/danielvladco/go-proto-gql/pb")
	p.jsonPkg = p.NewImport("encoding/json")
	p.contextPkg = p.NewImport("context")
	p.reflectPkg = p.NewImport("reflect")

	p.Scalars()
	for _, r := range p.Request.FileToGenerate {
		if r == file.GetName() {
			p.InitFile(file)
			msgRef := make(map[string]*generator.Descriptor)
			for _, msg := range file.Messages() {
				msgRef[msg.GetName()] = msg
			}
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

					typeInObj := p.TypeNameByObject(rpc.GetInputType())
					p.NewImport(string(typeInObj.GoImportPath()))
					typeOutObj := p.TypeNameByObject(rpc.GetOutputType())
					typeIn := generator.CamelCaseSlice(typeInObj.TypeName())
					typeOut := generator.CamelCaseSlice(typeOutObj.TypeName())
					if p.DefaultPackageName(typeInObj) != "" {
						typeIn = p.NewImport(string(typeInObj.GoImportPath())).Use() + "." + typeIn
					}
					if p.DefaultPackageName(typeOutObj) != "" {
						typeOut = p.NewImport(string(typeOutObj.GoImportPath())).Use() + "." + typeOut
					}
					in, inref := ", in *"+typeIn, ", in"
					if p.IsEmpty(p.Inputs()[rpc.GetInputType()]) {
						in, inref = "", ", &"+typeIn+"{}"
					}
					if p.IsEmpty(p.Types()[rpc.GetOutputType()]) {
						p.Generator.P("func (s *", svc.GetName(), "GQLServer) ", methodName, "(ctx ", p.contextPkg.Use(), ".Context", in, ") (*bool, error) { _, err := s.Service.", generator.CamelCase(rpc.GetName()), "(ctx", inref, ")\n return nil, err }")
					} else {
						p.Generator.P("func (s *", svc.GetName(), "GQLServer) ", methodName, "(ctx ", p.contextPkg.Use(), ".Context", in, ") (*", typeOut, ", error) { return s.Service.", generator.CamelCase(rpc.GetName()), "(ctx", inref, ") }")
					}
				}
			}
			for _, msg := range file.Messages() {
				if msg.GetOptions().GetMapEntry() {
					key, value := msg.GetMapFields()

					mapName := generator.CamelCaseSlice(msg.TypeName())
					keyType, _ := p.GoType(msg, key)
					valType, _ := p.GoType(msg, value)

					mapType := fmt.Sprintf("map[%s]%s", keyType, valType)
					p.Generator.P(`
func Marshal`, mapName, `(mp `, mapType, `) `, p.graphqlPkg.Use(), `.Marshaler {
	return `, p.graphqlPkg.Use(), `.WriterFunc(func(w `, p.ioPkg.Use(), `.Writer) {
		err := `, p.jsonPkg.Use(), `.NewEncoder(w).Encode(mp)
		if err != nil {
			panic("this map type is not supported")
		}
	})
}

func Unmarshal`, mapName, `(v interface{}) (mp `, mapType, `, err error) {
	switch vv := v.(type) {
	case []byte:
		err = `, p.jsonPkg.Use(), `.Unmarshal(vv, &mp)
		return mp, err
	case `, p.jsonPkg.Use(), `.RawMessage:
		err = `, p.jsonPkg.Use(), `.Unmarshal(vv, &mp)
		return mp, err
	default:
		return nil, `, p.fmtPkg.Use(), `.Errorf("%T is not `, mapType, `", v)
	}
}
`)
				}
				for _, oneof := range msg.OneofDecl {
					oneofName := append(msg.TypeName(), oneof.GetName())
					method := "is" + generator.CamelCaseSlice(msg.TypeName()) + "_" + generator.CamelCase(msg.GetName())
					p.Generator.P(`type Is`, generator.CamelCaseSlice(oneofName),
						" interface{\n\t", method, "()\n}")

					// render Oneof_ConcreteTypes()
					parentName := generator.CamelCaseSlice(msg.TypeName())
					ss := oneofConcretes(msg, msgRef)
					if len(ss) != 0 {
						p.Generator.P(`
func (p *`, generator.CamelCase(parentName), `) Oneof_ConcreteTypes() []`, p.reflectPkg.Use(), `.Type {
	return []reflect.Type{`, strings.Join(ss, ","), `}
}`)
					}
				}
			}
			for _, enum := range file.Enums() {
				enumType := generator.CamelCaseSlice(enum.TypeName())
				p.Generator.P(`
func (c *`, enumType, `) UnmarshalGQL(v interface{}) error {
	code, ok := v.(string)
	if ok {
		*c = `, enumType, `(`, enumType, `_value[code])
		return nil
	}
	return fmt.Errorf("cannot unmarshal `, enumType, ` enum")
}

func (c `, enumType, `) MarshalGQL(w `, p.ioPkg.Use(), `.Writer) {
	`, p.fmtPkg.Use(), `.Fprintf(w, "%q", c.String())
}
`)
			}

		}
	}
}

func oneofConcretes(msg *generator.Descriptor, msgRef map[string]*generator.Descriptor) []string {
	ss := make([]string, 0)
	for _, field := range msg.GetField() {
		fieldName := strings.Title(generator.CamelCase(field.GetName()))
		var shortTypeName string
		tnParts := strings.Split(field.GetTypeName(), ".")
		if len(tnParts) != 0 {
			shortTypeName = tnParts[len(tnParts)-1]
		}
		if m, ok := msgRef[shortTypeName]; ok && len(m.GetField()) != 0 {
			name := generator.CamelCaseSlice(msg.TypeName()) + "_" + fieldName
			rName := `reflect.TypeOf((*` + name + `)(nil)).Elem()`
			ss = append(ss, rName)
		}
	}
	sort.Sort(sort.StringSlice(ss))
	return ss
}
