package main

import (
	"github.com/danielvladco/go-proto-gql/reflect/gateway/internal/server"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"google.golang.org/protobuf/types/pluginpb"
	"log"
	"os"
)

func main() {
	_, ss, fs, err := server.NewReflectCaller([]string{"localhost:8081", "localhost:8082"})
	if err != nil {
		log.Fatal(err)
	}

	//proto.Marshal()
	sss := []*descriptor.FileDescriptorProto{}
	for _, s := range ss {
		sss = append(sss, s.AsFileDescriptorProto())
	}
	generator := &pluginpb.CodeGeneratorRequest{
		FileToGenerate:  fs,
		ProtoFile:       sss,
		CompilerVersion: nil,
	}

	b, err := proto.Marshal(generator)
	if err != nil {
		log.Fatal(err)
	}
	f, err := os.Create("testing_file.bytes")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	_, err = f.Write(b)
	if err != nil {
		log.Fatal(err)
	}
}
