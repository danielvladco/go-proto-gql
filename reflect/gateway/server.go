package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/danielvladco/go-proto-gql/reflect/gateway/internal/reflection"
	"github.com/danielvladco/go-proto-gql/reflect/gateway/internal/server"
	"github.com/mitchellh/mapstructure"
	"log"
	"net/http"
	"os"

	"github.com/danielvladco/go-proto-gql/plugin"
	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
	plugin_go "github.com/gogo/protobuf/protoc-gen-gogo/plugin"
	godescriptor "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/jhump/protoreflect/desc"
	"github.com/nautilus/gateway"
	"github.com/nautilus/graphql"
	"google.golang.org/grpc"
)

type Config struct {
	Endpoints  []string `json:"endpoints"`
	Playground *bool    `json:"playground"`
	Address    string   `json:"address"`
}

var (
	configFile = flag.String("cfg", "/opt/config.json", "")
)

func main() {
	flag.Parse()

	f, err := os.Open(*configFile)
	fatalOnErr(err)
	cfg := &Config{}
	err = json.NewDecoder(f).Decode(cfg)
	fatalOnErr(err)
	if cfg.Address == "" {
		cfg.Address = ":8082"
	}
	if cfg.Playground == nil {
		plg := true
		cfg.Playground = &plg
	}

	var descs []*desc.FileDescriptor
	descsconn := map[*desc.FileDescriptor]*grpc.ClientConn{}
	for _, e := range cfg.Endpoints {
		conn, err := grpc.Dial(e, grpc.WithInsecure())
		fatalOnErr(err)
		client := reflection.NewClient(conn)

		tempdesc, err := client.ListPackages()
		fatalOnErr(err)

		for _, d := range tempdesc {
			descsconn[d] = conn
		}
		descs = append(descs, tempdesc...)
	}

	// FIXME please! the worst workaround in all human history... it hurts to look at this
	ddd := []*godescriptor.FileDescriptorProto{}
	for i, d := range descs {

		_ = i
		p := d.AsFileDescriptorProto()
		n := fmt.Sprintf("_%d_%s", i, p.GetName())
		p.Name = &n
		//pkg := "pb"
		gopkg := "github.com/danielvladco/go-proto-gql/reflect/grpcserver1/pb;pb"
		//p.Package = &pkg
		p.Options.GoPackage = &gopkg
		ddd = append(ddd, p)
	}
	buff := &bytes.Buffer{}
	err = gob.NewEncoder(buff).Encode(ddd)
	fatalOnErr(err)
	var protoFiles []*descriptor.FileDescriptorProto
	err = gob.NewDecoder(buff).Decode(&protoFiles)
	fatalOnErr(err)

	fnames := []string{}
	for _, f := range protoFiles {
		fnames = append(fnames, f.GetName()) //strings.Replace(f.GetName(), ".proto", ".pb.go", -1))
	}
	p := gen(&plugin_go.CodeGeneratorRequest{
		FileToGenerate:  fnames,
		Parameter:       nil,
		ProtoFile:       protoFiles,
		CompilerVersion: nil,
	})

	// FIXME use queries as real queries instead of mutations
	queries := map[string]*plugin.Method{}
	for _, q := range p.Mutations() {
		queries[q.Name] = q
	}

	//TODO wtf
	origServices := map[server.SvcKey]server.SvcVal{}
	for _, d := range descs {
		for _, svc := range d.GetServices() {
			for _, rpc := range svc.GetMethods() {
				origServices[server.SvcKey{svc.GetName(), rpc.GetName()}] = server.SvcVal{descsconn[d], rpc}
			}
		}
	}

	//TODO wtf
	origMessages := map[string]*desc.MessageDescriptor{}
	for _, d := range descs {
		for _, m := range d.GetMessageTypes() {
			origMessages[m.GetName()] = m
		}
	}

	queryFactory := gateway.QueryerFactory(func(ctx *gateway.PlanningContext, url string) graphql.Queryer {
		return server.NewQueryer(queries, origMessages, origServices, p)
	})

	sources := []*graphql.RemoteSchema{{URL: "url1"}}
	sources[0].Schema, err = server.GenerateSchema(p)
	if err != nil {
		log.Fatal(err)
	}

	g, err := gateway.New(sources, gateway.WithQueryerFactory(&queryFactory))
	result := &graphql.IntrospectionQueryResult{}
	err = schemaTestLoadQuery(g, graphql.IntrospectionQuery, result, map[string]interface{}{})
	if err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/query", g.GraphQLHandler)
	if *cfg.Playground {
		mux.HandleFunc("/playground", g.PlaygroundHandler)
	}
	http.ListenAndServe(cfg.Address, mux)
}

func gen(request *plugin_go.CodeGeneratorRequest) *plugin.Plugin {
	gen := generator.New()
	gen.Request = request
	gen.CommandLineParameters(gen.Request.GetParameter())
	gen.WrapTypes()
	gen.SetPackageNames()
	gen.BuildTypeNameMap()
	p := &plug{Plugin: plugin.NewPlugin()}
	gen.GeneratePlugin(p)
	return p.Plugin
}

type plug struct {
	*plugin.Plugin
	generator.PluginImports
}

func (p plug) Name() string {
	return "gqlproxy"
}

func (p *plug) Init(g *generator.Generator) {
	p.Generator = g
	p.PluginImports = generator.NewPluginImports(p.Generator)
}

func (p plug) Generate(file *generator.FileDescriptor) {
	for _, fileName := range p.Request.FileToGenerate {
		if fileName == file.GetName() {
			p.InitFile(file.FileDescriptorProto)
		}
	}
}
func fatalOnErr(err error) {
	if err != nil {
		panic(err)
	}
}

func schemaTestLoadQuery(qw *gateway.Gateway, query string, target interface{}, variables map[string]interface{}) error {
	reqCtx := &gateway.RequestContext{
		Context:   context.Background(),
		Query:     query,
		Variables: variables,
	}
	plan, err := qw.GetPlans(reqCtx)
	if err != nil {
		return err
	}

	// executing the introspection query should return a full description of the schema
	response, err := qw.Execute(reqCtx, plan)
	if err != nil {
		return err
	}

	// massage the map into the structure
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName: "json",
		Result:  target,
	})
	if err != nil {
		return err
	}
	err = decoder.Decode(response)
	if err != nil {
		return err
	}

	return nil
}
