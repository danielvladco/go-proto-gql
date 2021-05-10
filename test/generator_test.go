package test

import (
	"bytes"
	_ "embed"
	"fmt"
	"github.com/danielvladco/go-proto-gql/pkg/protoparser"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"go/build"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/danielvladco/go-proto-gql/pkg/generator"
	"github.com/vektah/gqlparser/v2/formatter"
)

var (
	//go:embed testdata/constructs-expect.graphql
	constructsExpectedGraphql []byte

	//go:embed testdata/options-expect.graphql
	optionsExpectedGraphql []byte
)

func Test_test(t *testing.T) {
	tests := []struct {
		name      string
		inputFile string
		expect    []byte
	}{{
		name:      "Constructs",
		inputFile: "testdata/constructs-input.proto",
		expect:    constructsExpectedGraphql,
	}, {
		name:      "Options",
		inputFile: "testdata/options-input.proto",
		expect:    optionsExpectedGraphql,
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, file, _, _ := runtime.Caller(0)
			absFile, _ := filepath.Abs(file)
			absDir, _ := filepath.Split(absFile)
			googlePaths := filepath.Join(build.Default.GOPATH, "include")
			projectPath := filepath.Join(absDir, "..")
			fmt.Println(googlePaths)
			descs, err :=  protoparser.Parse([]string{projectPath, googlePaths, absDir}, []string{tc.inputFile}, protoparser.WithSourceCodeInfo(false))
			if err != nil {
				t.Fatal(err)
			}
			p, err := protogen.Options{}.New(&pluginpb.CodeGeneratorRequest{
				FileToGenerate: []string{tc.inputFile},
				ProtoFile:      generator.ResolveProtoFilesRecursively(descs).AsFileDescriptorProto(),
			})
			if err != nil {
				t.Fatal(err)
			}
			gqlDesc, err := generator.NewSchemas(descs, false, true, p)
			if err != nil {
				t.Fatal(err)
			}
			expectedGraphql := &bytes.Buffer{}
			actualGraphql := &bytes.Buffer{}
			expectedFormattedSchema, err := gqlparser.LoadSchema(&ast.Source{
				Name:  "testdata/constructs-expect.graphql",
				Input: string(tc.expect),
			})
			formatter.NewFormatter(actualGraphql).FormatSchema(gqlDesc[0].AsGraphql())
			formatter.NewFormatter(expectedGraphql).FormatSchema(expectedFormattedSchema)
			if actualGraphql.String() != expectedGraphql.String() {
				t.Errorf("Generated graphql file does not match expectations")

				edits := myers.ComputeEdits("file://constructs-expect.graphql", expectedGraphql.String(), actualGraphql.String())
				t.Errorf("%s", gotextdiff.ToUnified("expect", "actual", expectedGraphql.String(), edits))
			}
		})
	}
}
