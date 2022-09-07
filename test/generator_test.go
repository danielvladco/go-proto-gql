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
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/formatter"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/catalystsquad/go-proto-gql/pkg/generator"
	"github.com/catalystsquad/go-proto-gql/pkg/protoparser"
)

func Test_Generator(t *testing.T) {
	tests := []struct {
		name          string
		inputFile     string
		expectFile    string
		useFieldNames bool
		useBigInt     bool
		ignoreProtos  []string
	}{{
		name:       "Constructs",
		inputFile:  "testdata/constructs-input.proto",
		expectFile: "testdata/constructs-expect.graphql",
	}, {
		name:       "Options",
		inputFile:  "testdata/options-input.proto",
		expectFile: "testdata/options-expect.graphql",
	}, {
		name:          "Options",
		inputFile:     "testdata/options-input.proto",
		expectFile:    "testdata/options-expect-fieldnames.graphql",
		useFieldNames: true,
		useBigInt:     true,
		ignoreProtos:  []string{"google.protobuf.Value", "google.protobuf.Struct"},
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
			gqlDesc, err := generator.NewSchemas(descs, false, true, tc.useFieldNames, tc.useBigInt, tc.ignoreProtos, p)
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
			compareGraphqlUsingFile(t, gqlDesc[0].AsGraphql(), string(bb))
		})
	}
}

func relativeFile(filename string) string {
	_, file, _, _ := runtime.Caller(0)
	dir, _ := filepath.Split(file)

	return filepath.Join(dir, filename)
}

func compareGraphql(t *testing.T, got, expect *ast.Schema) {
	expectedGraphql := &bytes.Buffer{}
	actualGraphql := &bytes.Buffer{}
	formatter.NewFormatter(actualGraphql, formatter.WithIndent("  ")).FormatSchema(got)
	formatter.NewFormatter(expectedGraphql, formatter.WithIndent("  ")).FormatSchema(expect)
	os.WriteFile("actual", []byte(actualGraphql.String()), 0755)
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

func compareGraphqlUsingFile(t *testing.T, got *ast.Schema, expect string) {
	actualGraphql := &bytes.Buffer{}
	formatter.NewFormatter(actualGraphql, formatter.WithIndent("  ")).FormatSchema(got)
	os.WriteFile("actual", []byte(actualGraphql.String()), 0755)
	if actualGraphql.String() != expect {
		diff := difflib.UnifiedDiff{
			A:        difflib.SplitLines(expect),
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
