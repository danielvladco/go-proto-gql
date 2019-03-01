package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

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
	// Match parsing algorithm from Generator.CommandLineParameters
	for _, parameter := range strings.Split(gen.Request.GetParameter(), ",") {
		kvp := strings.SplitN(parameter, "=", 2)
		// We only care about key-value pairs where the key is "gogoimport"
		if len(kvp) != 2 || kvp[0] != "gogoimport" {
			continue
		}
		useGogoImport, err = strconv.ParseBool(kvp[1])
		if err != nil {
			gen.Error(err, "parsing gogoimport option")
		}
	}

	gen.CommandLineParameters(gen.Request.GetParameter())
	gen.WrapTypes()
	gen.SetPackageNames()
	gen.BuildTypeNameMap()
	gen.GeneratePlugin(&plugin{useGogoImport: useGogoImport, enums: make(map[string]struct{}), messages: make(map[string]struct{})})

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
	*generator.Generator
	generator.PluginImports
	ioPkg      generator.Single
	fmtPkg     generator.Single
	graphqlPkg generator.Single
	jsonPkg    generator.Single

	enums    map[string]struct{}
	messages map[string]struct{}

	useGogoImport bool
}

func (p *plugin) Name() string                { return "gqlenum" }
func (p *plugin) Init(g *generator.Generator) { p.Generator = g }
func (p *plugin) Generate(file *generator.FileDescriptor) {
	if !p.useGogoImport {
		vanity.TurnOffGogoImport(file.FileDescriptorProto)
	}

	p.PluginImports = generator.NewPluginImports(p.Generator)
	p.fmtPkg = p.NewImport("fmt")
	p.ioPkg = p.NewImport("io")
	p.graphqlPkg = p.NewImport("github.com/99designs/gqlgen/graphql")
	p.jsonPkg = p.NewImport("encoding/json")

	for _, r := range p.Request.FileToGenerate {
		if r == file.GetName() {
			for _, msg := range file.Messages() {
				if msg.GetOptions().GetMapEntry() {
					key, value := msg.GetMapFields()

					mapName := generator.CamelCaseSlice(msg.TypeName())
					keyType, _ := p.GoType(msg, key)
					valType, _ := p.GoType(msg, value)

					mapType := fmt.Sprintf("map[%s]%s", keyType, valType)
					p.P(`
func Marshal`, mapName, `(mp `, mapType, `) `, p.graphqlPkg.Use(), `.Marshaler {
	return graphql.WriterFunc(func(w `, p.ioPkg.Use(), `.Writer) {
		err := `, p.jsonPkg.Use(), `.NewEncoder(w).Encode(mp)
		if err != nil {
			panic("stupid map")
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
			}
			for _, enum := range file.Enums() {
				enumType := generator.CamelCaseSlice(enum.TypeName())
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
	`, p.fmtPkg.Use(), `.Fprintf(w, "%q", c.String())
}
`)
			}
		}
	}
}
