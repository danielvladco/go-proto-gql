package test

import (
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"github.com/mitchellh/mapstructure"
	"log"
	"net"
	"net/http"
	"reflect"
	"testing"

	"github.com/golang/protobuf/proto"
	any "github.com/golang/protobuf/ptypes/any"
	empty "github.com/golang/protobuf/ptypes/empty"
	"github.com/nautilus/graphql"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/danielvladco/go-proto-gql/pkg/server"
	pb "github.com/danielvladco/go-proto-gql/test/testdata"
)

var (
	//go:embed testdata/gateway-expect.graphql
	testGatewayExpectedSchema []byte
)

func Test_Gateway(t *testing.T) {
	var (
		optionsServerCh    = make(chan string)
		constructsServerCh = make(chan string)
		gatewayServerCh    = make(chan net.Addr)
	)
	go func() {
		l, err := net.Listen("tcp", "")
		if err != nil {
			log.Fatal(err)
		}
		optionsServerCh <- l.Addr().String()
		s := grpc.NewServer()
		pb.RegisterServiceServer(s, &optionsServiceMock{})
		reflection.Register(s)
		s.Serve(l)
	}()
	go func() {
		l, err := net.Listen("tcp", "")
		if err != nil {
			log.Fatal(err)
		}
		constructsServerCh <- l.Addr().String()
		s := grpc.NewServer()
		pb.RegisterConstructsServer(s, constructsServiceMock{})
		reflection.Register(s)
		s.Serve(l)
	}()
	go func() {
		optionsServer := <-optionsServerCh
		constructsServer := <-constructsServerCh
		handler, err := server.Server(&server.Config{
			Grpc: &server.Grpc{
				Services: []*server.Service{
					{Address: optionsServer, Reflection: true},
					{Address: constructsServer, Reflection: true},
				},
			},
			Playground: proto.Bool(true),
		})
		if err != nil {
			log.Fatal(err)
		}
		l, err := net.Listen("tcp", "")
		if err != nil {
			log.Fatal(err)
		}
		gatewayServerCh <- l.Addr()
		http.Serve(l, handler)
	}()
	gatewayServer := <-gatewayServerCh
	gatewayUrl := fmt.Sprintf("http://127.0.0.1:%d/query", gatewayServer.(*net.TCPAddr).Port)
	schema, err := graphql.IntrospectRemoteSchema(gatewayUrl)
	if err != nil {
		t.Fatal(err)
	}
	querier := graphql.NewSingleRequestQueryer(gatewayUrl)
	expectedFormattedSchema, _ := gqlparser.LoadSchema(&ast.Source{Input: string(testGatewayExpectedSchema)})
	t.Run("schema is correct", func(t *testing.T) {
		compareGraphql(t, schema.Schema, expectedFormattedSchema)
	})
	type j = map[string]interface{}

	//log.Printf("http://127.0.0.1:%d/playground\n", gatewayServer.(*net.TCPAddr).Port)
	//<-make(chan struct{})

	tests := []struct {
		name          string
		query         string
		operationName string
		vars          j
		wantResponse  proto.Message
	}{{
		name:          "Mutation constructs scalars",
		query:         constructsScalarsQuery,
		operationName: "constructsScalars_",
		wantResponse: &pb.Scalars{
			Double: 1.1, Float: 2.2, Int32: 3, Int64: -4, Uint32: 5,
			Uint64: 6, Sint32: 7, Sint64: 8, Fixed32: 9, Fixed64: 10, Sfixed32: 11,
			Sfixed64: 12, Bool: true, StringX: "test", Bytes: []byte("test"),
		},
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			recv := map[string]interface{}{}
			if err := querier.Query(context.Background(), &graphql.QueryInput{
				Query:         tc.query,
				OperationName: tc.operationName,
				//Variables:     tc.vars,
			}, &recv); err != nil {
				t.Fatal(err)
			}
			svcRecv, _ := reflect.New(reflect.TypeOf(tc.wantResponse).Elem()).Interface().(proto.Message)
			if decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
				DecodeHook: bytesDecoder, Result: svcRecv, TagName: "json",
			}); err != nil {
				t.Fatal(err)
			} else if err := decoder.Decode(recv[tc.operationName]); err != nil {
				t.Fatal(err)
			}
			if !proto.Equal(svcRecv, tc.wantResponse) {
				t.Errorf("mutation constructsScalars_ failed: expected: %s got: %s", tc.wantResponse, svcRecv)
			}
		})
	}
}

func bytesDecoder(_ reflect.Type, dest reflect.Type, val interface{}) (interface{}, error) {
	if dest.Kind() == reflect.Slice && dest.Elem().Kind() == reflect.Uint8 {
		strVal, ok := val.(string)
		if ok {
			return base64.StdEncoding.DecodeString(strVal)
		}
	}
	return val, nil
}

type constructsServiceMock struct {
	pb.ConstructsServer
}

func (c constructsServiceMock) Scalars_(ctx context.Context, scalars *pb.Scalars) (*pb.Scalars, error) {
	return scalars, nil
}
func (c constructsServiceMock) Repeated_(ctx context.Context, repeated *pb.Repeated) (*pb.Repeated, error) {
	return repeated, nil
}
func (c constructsServiceMock) Maps_(ctx context.Context, maps *pb.Maps) (*pb.Maps, error) {
	return maps, nil
}
func (c constructsServiceMock) Any_(ctx context.Context, any *any.Any) (*pb.Any, error) {
	return &pb.Any{}, nil
}
func (c constructsServiceMock) Empty_(ctx context.Context, empty *empty.Empty) (*pb.Empty, error) {
	return &pb.Empty{}, nil
}
func (c constructsServiceMock) Empty2_(ctx context.Context, recursive *pb.EmptyRecursive) (*pb.EmptyNested, error) {
	return &pb.EmptyNested{}, nil
}
func (c constructsServiceMock) Empty3_(ctx context.Context, empty3 *pb.Empty3) (*pb.Empty3, error) {
	return &pb.Empty3{}, nil
}
func (c constructsServiceMock) Ref_(ctx context.Context, ref *pb.Ref) (*pb.Ref, error) {
	return &pb.Ref{}, nil
}
func (c constructsServiceMock) Oneof_(ctx context.Context, oneof *pb.Oneof) (*pb.Oneof, error) {
	return &pb.Oneof{}, nil
}
func (c constructsServiceMock) CallWithId(ctx context.Context, empty *pb.Empty) (*pb.Empty, error) {
	return &pb.Empty{}, nil
}

type optionsServiceMock struct {
	pb.ServiceServer
}

func (o *optionsServiceMock) Mutate1(ctx context.Context, data *pb.Data) (*pb.Data, error) {
	return data, nil
}
func (o *optionsServiceMock) Mutate2(ctx context.Context, data *pb.Data) (*pb.Data, error) {
	return data, nil
}
func (o *optionsServiceMock) Query1(ctx context.Context, data *pb.Data) (*pb.Data, error) {
	return data, nil
}
func (o *optionsServiceMock) Ignore(ctx context.Context, data *pb.Data) (*pb.Data, error) {
	return data, nil
}
func (o *optionsServiceMock) Name(ctx context.Context, data *pb.Data) (*pb.Data, error) {
	return data, nil
}

const constructsScalarsQuery = `
mutation {
  constructsScalars_(in: {
    double: 1.1, float: 2.2, int32: 3, int64: -4, 
    uint32: 5, uint64: 6, sint32: 7, sint64: 8, 
    fixed32: 9, fixed64: 10, sfixed32: 11, sfixed64: 12,
    bool: true, stringX: "test", bytes: "dGVzdA=="
  }) {
    double
    float
    int32
    int64
    uint32
    uint64
    sint32
    sint64
    fixed32
    fixed64
    sfixed32
    sfixed64
    bool
    string_x: stringX
    bytes
    __typename
  }
}
`
