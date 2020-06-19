package main

import (
	"github.com/danielvladco/go-proto-gql/reflect/gateway/internal/generator"
	"github.com/danielvladco/go-proto-gql/reflect/gateway/internal/server"
	"github.com/vektah/gqlparser/v2/formatter"
	"log"
	"os"
)

func main() {
	//fs, err := protoparse.Parser{}.ParseFiles("../example/constructs.proto")
	//if err != nil {
	//	log.Fatal(err)
	//}

	// TODO
	// 1. use `desc.CreateFileDescriptor()` instead of reflect call
	// 2. resolve unique names
	// 3. investigate why we don't receive source code info (required for comments) (guess it might be related to server refection)

	_, ss, _, err := server.NewReflectCaller([]string{"localhost:8081", "localhost:8082"})
	if err != nil {
		log.Fatal(err)
	}

	//proto.Marshal()
	sch, err := generator.NewSchemas(ss, true)
	if err != nil {
		log.Fatal(err)
	}

	src := sch.AsGraphql()[0]

	//desc.CreateFileDescriptor()

	f, err := os.Create("test.graphql")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	formatter.NewFormatter(f).FormatSchema(src)

}
