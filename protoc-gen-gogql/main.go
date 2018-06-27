package main

import (
	"io/ioutil"
	"os"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"

	"github.com/danielvladco/go-proto-gql/protoc-gen-gogql/plugin"
	"log"

	gogoplugin "github.com/gogo/protobuf/protoc-gen-gogo/plugin"
	"github.com/vektah/gqlgen/codegen"
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

	// Match parsing algorithm from Generator.CommandLineParameters
	keyvalparams := make(map[string]string)
	for _, parameter := range strings.Split(gen.Request.GetParameter(), ",") {
		kvp := strings.SplitN(parameter, "=", 2)
		// We only care about key-value pairs where the key is "gogoimport"
		lkvp := len(kvp)
		if lkvp < 2 {
			continue
		}
		for i := 0; i < lkvp; i += 2 {
			keyvalparams[kvp[i]] = kvp[i+1]
		}
	}

	p := plugin.NewPlugin()

	gen.CommandLineParameters(gen.Request.GetParameter())
	gen.WrapTypes()
	gen.SetPackageNames()
	gen.BuildTypeNameMap()
	gen.GeneratePlugin(p)

	schema := p.GetSchema()
	typemap := p.GetTypesMap()

	ioutil.WriteFile("schema.graphql", []byte(schema), 0644)
	for i := 0; i < len(gen.Response.File); i++ {
		err := codegen.Generate(codegen.Config{
			SchemaStr:        schema,
			Typemap:          typemap,
			ModelFilename:    strings.Replace(*gen.Response.File[i].Name, ".pb.go", ".gql_models.pb.go", -1),
			ExecFilename:     strings.Replace(*gen.Response.File[i].Name, ".pb.go", ".gql_server.pb.go", -1),
			ExecPackageName:  keyvalparams["package"],
			ModelPackageName: keyvalparams["modelpackage"],
		})
		if err != nil {
			log.Fatal(err.Error(), "unable to generate data")
		}
	}

	// Send back empty response. File generation is already handled by gqlgen library.
	data, err = proto.Marshal(&gogoplugin.CodeGeneratorResponse{})
	if err != nil {
		gen.Error(err, "failed to marshal output proto")
	}
	_, err = os.Stdout.Write(data)
	if err != nil {
		gen.Error(err, "failed to write output proto")
	}
}
