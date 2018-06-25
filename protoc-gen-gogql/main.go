package main

import (
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"

	"github.com/danielvladco/go-proto-gql/plugin"
	"log"
	"path"
)

//go:generate protoc --go_out=../ ../*.proto
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
	dir, _ := os.Getwd()
	gopath := os.Getenv("GOPATH")
	if strings.Trim(gopath, " ") == "" {
		log.Fatal("unable to locate $GOPATH environment variable")
	}

	ss := strings.SplitN(dir, path.Join(gopath, "src/"), 2)
	if len(ss) < 2 {
		log.Fatal("plugin wa used outside of $GOPATH scope (to use this plugin you must call it from within a $GOPATH)")
	}

	log.Println(strings.Trim(ss[1], "/"))
	for _, m := range gen.Response.File {
		log.Println(m.Name)
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
	gen.GeneratePlugin(plugin.NewPlugin(useGogoImport))
	for _, m := range gen.Response.File {
		log.Println(m.GetName(), "Xxxx")
	}
	for i := 0; i < len(gen.Response.File); i++ {
		gen.Response.File[i].Name = proto.String(strings.Replace(*gen.Response.File[i].Name, ".pb.go", ".gql.pb.go", -1))
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
