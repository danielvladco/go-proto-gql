package test

import (
	"bytes"
	_ "embed"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/formatter"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/danielvladco/go-proto-gql/pkg/generator"
	"github.com/danielvladco/go-proto-gql/pkg/protoparser"
)

var (
	//go:embed testdata/constructs-expect.graphql
	constructsExpectedGraphql []byte

	//go:embed testdata/options-expect.graphql
	optionsExpectedGraphql []byte
)

func Test_Generator(t *testing.T) {
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
			apiPath := filepath.Join(absDir, "..")
			fmt.Println(apiPath)
			descs, err := protoparser.Parse([]string{apiPath, absDir}, []string{tc.inputFile}, protoparser.WithSourceCodeInfo(false))
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
			expectedFormattedSchema, _ := gqlparser.LoadSchema(&ast.Source{Input: string(tc.expect)})
			//if err != nil {
			//	t.Fatal(err)
			//}
			compareGraphql(t, gqlDesc[0].AsGraphql(), expectedFormattedSchema)
		})
	}
}

func compareGraphql(t *testing.T, got, expect *ast.Schema) {
	expectedGraphql := &bytes.Buffer{}
	actualGraphql := &bytes.Buffer{}
	formatter.NewFormatter(actualGraphql).FormatSchema(got)
	formatter.NewFormatter(expectedGraphql).FormatSchema(expect)
	if actualGraphql.String() != expectedGraphql.String() {
		t.Errorf("Generated graphql file does not match expectations")

		edits := myers.ComputeEdits("file://constructs-expect.graphql", expectedGraphql.String(), actualGraphql.String())
		t.Errorf("%s", gotextdiff.ToUnified("expect", "actual", expectedGraphql.String(), edits))
	}
}
