package main

import (
	"github.com/99designs/gqlgen/codegen"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"syscall"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"

	"github.com/danielvladco/go-proto-gql/protoc-gen-gogql/plugin"
)

func main() {
	gen := generator.New()

	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		gen.Error(err, "reading input")
	}
	//ioutil.WriteFile("data.txt", []byte(data), 0644)
	if err := proto.Unmarshal(data, gen.Request); err != nil {
		gen.Error(err, "parsing input proto")
	}

	p := plugin.NewPlugin()

	gen.CommandLineParameters(gen.Request.GetParameter())
	gen.WrapTypes()
	gen.SetPackageNames()
	gen.BuildTypeNameMap()
	gen.GeneratePlugin(p)

	configGqlgen := ""
	// Match parsing algorithm from Generator.CommandLineParameters
	for _, parameter := range strings.Split(gen.Request.GetParameter(), ",") {
		kvp := strings.SplitN(parameter, "=", 2)
		// We only care about key-value pairs where the key is "gqlgen"
		if len(kvp) != 2 || kvp[0] != "gqlgen" {
			continue
		}
		configGqlgen = kvp[1]
	}
	if configGqlgen != "" {
		filePath, _ := path.Split(configGqlgen)
		if err = os.MkdirAll(filePath, os.ModePerm); err != nil {
			gen.Error(err)
		}
		f, err := os.OpenFile(configGqlgen, os.O_RDWR|os.O_CREATE, os.ModePerm)
		if err != nil {
			gen.Error(err)
		}
		defer f.Close()

		cfg := &codegen.Config{}
		if err := yaml.NewDecoder(f).Decode(cfg); err != nil && err != io.EOF {
			gen.Error(err)
		}
		if err = f.Truncate(0); err != nil {
			gen.Error(err)
		}
		if _, err := f.Seek(0, 0); err != nil {
			gen.Error(err)
		}
		// edits the graphql.yml config file
		typeMaps := p.GetTypesMap()
		if cfg.Models == nil {
			cfg.Models = make(codegen.TypeMap)
		}
		for key := range typeMaps {
			if m, ok := cfg.Models[key]; ok {
				m.Model = typeMaps[key].Model
				cfg.Models[key] = m
			} else {
				cfg.Models[key] = codegen.TypeMapEntry{Model: typeMaps[key].Model}
			}
		}

		e := yaml.NewEncoder(f)
		if err := e.Encode(cfg); err != nil {
			_ = syscall.Unlink(configGqlgen)
			gen.Error(err)
		}
		defer e.Close()
	}

	// Send back empty response. File generation is already handled by gqlgen library.
	for i := 0; i < len(gen.Response.GetFile()); i++ {
		gen.Response.File[i].Name = proto.String(strings.Replace(gen.Response.File[i].GetName(), ".pb.go", ".pbgen.graphqls", -1))
		schema := p.GetSchemaByIndex(i).String()
		gen.Response.File[i].Content = &schema
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
