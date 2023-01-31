package test

import (
	"bytes"
	_ "embed"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/formatter"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/tailor-inc/go-proto-gql/pkg/generator"
	"github.com/tailor-inc/go-proto-gql/pkg/protoparser"
)

func Test_Generator(t *testing.T) {
	tests := []struct {
		name       string
		inputFile  string
		expectFile string
	}{{
		name:       "Constructs",
		inputFile:  "testdata/constructs-input.proto",
		expectFile: "testdata/constructs-expect.graphql",
	}, {
		name:       "Options",
		inputFile:  "testdata/options-input.proto",
		expectFile: "testdata/options-expect.graphql",
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, file, _, _ := runtime.Caller(0)
			absFile, _ := filepath.Abs(file)
			absDir, _ := filepath.Split(absFile)
			apiPath := filepath.Join(absDir, "..")
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
			f, err := os.Open(relativeFile(tc.expectFile))
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			bb, err := io.ReadAll(f)
			if err != nil {
				t.Fatal(err)
			}
			expectedFormattedSchema, _ := gqlparser.LoadSchema(&ast.Source{Input: string(bb)})
			compareGraphql(t, gqlDesc[0].AsGraphql(), expectedFormattedSchema)
		})
	}
}

func relativeFile(filename string) string {
	_, file, _, _ := runtime.Caller(0)
	dir, _ := filepath.Split(file)

	return filepath.Join(dir, filename)
}

func compareGraphql(t *testing.T, got, expect *ast.Schema) {
	t.Helper()

	expectedGraphql := &bytes.Buffer{}
	actualGraphql := &bytes.Buffer{}
	formatter.NewFormatter(actualGraphql).FormatSchema(got)
	formatter.NewFormatter(expectedGraphql).FormatSchema(expect)

	if actualGraphql.String() != expectedGraphql.String() {
		diff := difflib.UnifiedDiff{
			A:        difflib.SplitLines(expectedGraphql.String()),
			B:        difflib.SplitLines(actualGraphql.String()),
			FromFile: "expect",
			ToFile:   "got",
			Context:  3,
		}
		t.Errorf("Generated graphql file does not match expectations")
		str, _ := difflib.GetUnifiedDiffString(diff)
		t.Errorf("%s", str)
	}
}
