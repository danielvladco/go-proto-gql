package main

import (
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
	"io/ioutil"
	"os"
	"strings"

	"github.com/danielvladco/go-proto-gql/protoc-gen-gql/plugin"
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

	p := plugin.NewPlugin()

	gen.CommandLineParameters(gen.Request.GetParameter())
	gen.WrapTypes()
	gen.SetPackageNames()
	gen.BuildTypeNameMap()
	gen.GeneratePlugin(p)

	fileLen := len(gen.Response.GetFile())
	for i := 0; i < fileLen; i++ {
		gen.Response.File[i].Name = proto.String(strings.Replace(gen.Response.File[i].GetName(), ".pb.go", ".pb.graphqls", -1))
		gen.Response.File[i].Content = proto.String(p.GetSchemaByIndex(i).String())
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
