package main

import (
	"io/ioutil"
	"os"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
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
	gen.GeneratePlugin(&plugin{enums: make(map[string]struct{}), messages: make(map[string]struct{})})

	for i := 0; i < len(gen.Response.File); i++ {
		gen.Response.File[i].Name = proto.String(strings.Replace(*gen.Response.File[i].Name, ".pb.go", ".gql_enum.pb.go", -1))
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
	*generator.Generator
	generator.PluginImports
	ioPkg    generator.Single
	fmtPkg   generator.Single
	enums    map[string]struct{}
	messages map[string]struct{}
}

func (p *plugin) Name() string                { return "gql" }
func (p *plugin) Init(g *generator.Generator) { p.Generator = g }
func (p *plugin) Generate(file *generator.FileDescriptor) {
	p.PluginImports = generator.NewPluginImports(p.Generator)
	p.fmtPkg = p.NewImport("fmt")
	p.ioPkg = p.NewImport("io")
	for _, svc := range file.Service {
		for _, rpc := range svc.Method {
			p.messages[rpc.GetInputType()] = struct{}{}
			p.messages[rpc.GetOutputType()] = struct{}{}
		}
	}
	for msgname := range p.messages {
		msg := &descriptor.DescriptorProto{}
		mm := strings.Split(msgname, ".")
		if len(mm) < 3 {
			panic("unable to resolve type")
		}

		if file.GetPackage() == mm[1] {
			msg = file.GetMessage(strings.Join(mm[2:], "."))
			for _, f := range msg.GetField() {
				if f.IsEnum() {
					p.enums[f.GetTypeName()] = struct{}{}
				}
			}
		}
	}

	for name := range p.enums {
		nn := strings.Split(name, ".")
		if len(nn) < 3 {
			panic("unexpected enum type")
		}
		enumType := generator.CamelCase(strings.Join(nn[2:], "_"))
		p.P(`
func (c *`, enumType, `) UnmarshalGQL(v interface{}) error {
	code, ok := v.(string)
	if ok {
		*c = `, enumType, `(`, enumType, `_value[code])
		return nil
	}
	return fmt.Errorf("cannot unmarshal `, enumType, ` enum")
}

func (c `, enumType, `) MarshalGQL(w `, p.ioPkg.Use(), `.Writer) {
	`, p.fmtPkg.Use(), `.Fprint(w, c.String())
}
`)
	}
}
