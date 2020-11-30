package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jhump/protoreflect/desc"
	"github.com/vektah/gqlparser/v2/formatter"
	"google.golang.org/protobuf/proto"
	descriptor "google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/danielvladco/go-proto-gql/pkg/generator"
)

func main() {
	in, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}
	req := &pluginpb.CodeGeneratorRequest{}
	if err := proto.Unmarshal(in, req); err != nil {
		log.Fatal(err)
	}

	files, err := generate(req)
	res := &pluginpb.CodeGeneratorResponse{File: files}
	if err != nil {
		res.Error = proto.String(err.Error())
	}

	out, err := proto.Marshal(res)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := os.Stdout.Write(out); err != nil {
		log.Fatal(err)
	}
}

func generate(req *pluginpb.CodeGeneratorRequest) (outFiles []*pluginpb.CodeGeneratorResponse_File, err error) {
	var genServiceDesc bool
	var merge bool
	for _, param := range strings.Split(req.GetParameter(), ",") {
		var value string
		if i := strings.Index(param, "="); i >= 0 {
			value, param = param[i+1:], param[0:i]
		}
		switch param {
		case "svc":
			if genServiceDesc, err = strconv.ParseBool(value); err != nil {
				return nil, err
			}
		case "merge":
			if merge, err = strconv.ParseBool(value); err != nil {
				return nil, err
			}
		}
	}
	dd, err := desc.CreateFileDescriptorsFromSet(&descriptor.FileDescriptorSet{
		File: req.GetProtoFile(),
	})
	if err != nil {
		return nil, err
	}
	var descs []*desc.FileDescriptor
	for _, d := range dd {
		for _, filename := range req.FileToGenerate {
			if filename == d.GetName() {
				descs = append(descs, d)
			}
		}
	}

	gqlDesc, err := generator.NewSchemas(descs, merge, genServiceDesc)
	if err != nil {
		return nil, err
	}
	for _, schema := range gqlDesc {
		buff := &bytes.Buffer{}
		formatter.NewFormatter(buff).FormatSchema(schema.AsGraphql())
		protoFileName := schema.FileDescriptors[0].GetName()

		outFiles = append(outFiles, &pluginpb.CodeGeneratorResponse_File{
			Name:    proto.String(resolveGraphqlFilename(protoFileName, merge)),
			Content: proto.String(buff.String()),
		})
	}

	return
}

func resolveGraphqlFilename(protoFileName string, merge bool) string {
	if merge {
		gqlFileName := "schema.graphqls"
		absProtoFileName, err := filepath.Abs(protoFileName)
		if err == nil {
			protoDirSlice := strings.Split(filepath.Dir(absProtoFileName), string(filepath.Separator))
			if len(protoDirSlice) > 0 {
				gqlFileName = protoDirSlice[len(protoDirSlice)-1] + ".graphqls"
			}
		}
		protoDir, _ := path.Split(protoFileName)
		return path.Join(protoDir, gqlFileName)
	}

	return strings.TrimSuffix(protoFileName, path.Ext(protoFileName)) + ".graphqls"
}
