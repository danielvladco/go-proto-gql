package main

import (
	"errors"
	"io/ioutil"
	"os"
	"path"
	"strings"

	. "github.com/danielvladco/go-proto-gql/plugin"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
	"gopkg.in/yaml.v2"
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

	p := &plugin{Plugin: NewPlugin()}
	gen.CommandLineParameters(gen.Request.GetParameter())
	gen.WrapTypes()
	gen.SetPackageNames()
	gen.BuildTypeNameMap()
	gen.GeneratePlugin(p)

	fileLen := len(gen.Response.GetFile())
	for i := 0; i < fileLen; i++ {
		gen.Response.File[i].Name = proto.String(strings.Replace(gen.Response.File[i].GetName(), ".pb.go", ".gqlgen.pb.yml", -1))
		_, file := path.Split(strings.Replace(gen.Response.File[i].GetName(), ".gqlgen.pb.yml", ".pb.graphqls", -1))
		p.config[i].SchemaFilename = []string{path.Join(".", file)}
		b, err := yaml.Marshal(p.config[i])
		if err != nil {
			p.Error(errors.New("cannot create codegen.yml"))
		}
		gen.Response.File[i].Content = proto.String(string(b))
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
	*Plugin

	config []*config
}

func (p *plugin) GenerateImports(file *generator.FileDescriptor) {}
func (p *plugin) Name() string                                   { return "gqlgencfg" }
func (p *plugin) Init(g *generator.Generator)                    { p.Generator = g }
func (p *plugin) Generate(file *generator.FileDescriptor) {
	for _, fileName := range p.Request.FileToGenerate {
		if fileName == file.GetName() {
			p.InitFile(file)

			cfg := &config{
				SchemaFilename: []string{},
				Exec:           packageConfig{Filename: "exec.go"},
				Model:          packageConfig{Filename: "models.go"},
				Models:         make(map[string]typeMapEntry),
			}

			for typ, key := range p.GqlModelNames() {
				if typ == nil || p.IsEmpty(typ) || key == "Input" || key == "" { // filter possible junk data
					continue
				}

				if m, ok := cfg.Models[key]; ok {
					m.Model = []string{typ.PackageDir + "." + typ.TypeName}
					cfg.Models[key] = m
				} else {
					cfg.Models[key] = typeMapEntry{Model: []string{typ.PackageDir + "." + typ.TypeName}}
				}
			}
			p.config = append(p.config, cfg)
		}
	}
}

type (
	config struct {
		SchemaFilename []string                `yaml:"schema,omitempty"`
		Exec           packageConfig           `yaml:"exec"`
		Model          packageConfig           `yaml:"model"`
		Resolver       packageConfig           `yaml:"resolver,omitempty"`
		Models         map[string]typeMapEntry `yaml:"models,omitempty"`
		StructTag      string                  `yaml:"struct_tag,omitempty"`
	}

	packageConfig struct {
		Filename string `yaml:"filename,omitempty"`
		Package  string `yaml:"package,omitempty"`
		Type     string `yaml:"type,omitempty"`
	}

	typeMapEntry struct {
		Model  []string                `yaml:"model"`
		Fields map[string]typeMapField `yaml:"fields,omitempty"`
	}
	typeMapField struct {
		Resolver  bool   `yaml:"resolver"`
		FieldName string `yaml:"fieldName"`
	}
)
