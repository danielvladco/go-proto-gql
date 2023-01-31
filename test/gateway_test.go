package test

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"net"
	"net/http"
	"reflect"
	"testing"

	"github.com/golang/protobuf/proto"
	any "github.com/golang/protobuf/ptypes/any"
	empty "github.com/golang/protobuf/ptypes/empty"
	"github.com/nautilus/graphql"
	"github.com/tailor-inc/go-proto-gql/pkg/server"
	pb "github.com/tailor-inc/go-proto-gql/test/testdata"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	//go:embed testdata/gateway-expect.graphql
	testGatewayExpectedSchema []byte
)

type j = map[string]interface{}
type l = []interface{}

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
		l, err := net.Listen("tcp", ":9090")
		if err != nil {
			log.Fatal(err)
		}
		gatewayServerCh <- l.Addr()
		http.Serve(l, handler)
	}()
	gatewayServer := <-gatewayServerCh
	gatewayUrl := fmt.Sprintf("http://127.0.0.1:%d/query", gatewayServer.(*net.TCPAddr).Port)

	querier := graphql.NewSingleRequestQueryer(gatewayUrl)
	t.Run("schema is correct", func(t *testing.T) {
		schema, err := graphql.IntrospectRemoteSchema(gatewayUrl)
		if err != nil {
			t.Fatal(err)
		}
		expectedFormattedSchema, _ := gqlparser.LoadSchema(&ast.Source{Input: string(testGatewayExpectedSchema)})
		compareGraphql(t, schema.Schema, expectedFormattedSchema)
	})

	tests := []struct {
		name         string
		query        string
		vars         j
		wantResponse j
	}{{
		name:         "Mutation constructs scalars",
		query:        constructsScalarsQuery,
		wantResponse: constructsScalarsResponse,
	}, {
		name:         "Mutation constructs any",
		query:        constructsAnyQuery,
		wantResponse: constructsAnyResponse,
	}, {
		name:         "Mutation constructs maps",
		query:        constructsMapsQuery,
		wantResponse: constructsMapsResponse,
	}, {
		name:         "Mutation constructs repeated",
		query:        constructsRepeatedQuery,
		wantResponse: constructsRepeatedResponse,
	}, {
		name:         "Mutation constructs oneofs", //TODO check oneof input params not to allow inputting data for the same oneof input
		query:        constructsOneofsQuery,
		wantResponse: constructsOneofsResponse,
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			recv := map[string]interface{}{}
			if err := querier.Query(context.Background(), &graphql.QueryInput{
				Query:     tc.query,
				Variables: tc.vars,
			}, &recv); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(recv, tc.wantResponse) {
				t.Errorf("mutation failed: expected: %s got: %s", tc.wantResponse, recv)
			}
		})
	}
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
func (c constructsServiceMock) Any_(ctx context.Context, any *any.Any) (*any.Any, error) {
	return any, nil
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
	return oneof, nil
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
    stringX
    bytes
    __typename
  }
}
`

var constructsScalarsResponse = j{
	"constructsScalars_": j{
		"__typename": "Scalars",
		"bool":       true,
		"bytes":      "dGVzdA==",
		"stringX":    "test",
		"double":     1.1,
		"fixed32":    float64(9),
		"fixed64":    float64(10),
		"float":      2.2,
		"int32":      float64(3),
		"int64":      float64(-4),
		"sfixed32":   float64(11),
		"sfixed64":   float64(12),
		"sint32":     float64(7),
		"sint64":     float64(8),
		"uint32":     float64(5),
		"uint64":     float64(6),
	},
}

const constructsAnyQuery = `
mutation {
  constructsAny_(in: {__typename: "Ref", localTime2: {time: "1234"}, fileEnum: BAR2, local: {
    bar1: {param1: "param1"}, en1: A1, externalTime1: {seconds:1123, nanos: 123}
  }})
}
`

var constructsAnyResponse = j{
	"constructsAny_": j{
		"__typename": "Ref",
		"en1":        "A0",
		"en2":        "A0",
		"external":   nil,
		"file":       nil,
		"fileEnum":   "BAR2",
		"fileMsg":    nil,
		"foreign":    nil,
		"gz":         nil,
		"local": j{
			"__typename": "Ref_Foo",
			"bar1": j{
				"__typename": "Ref_Foo_Bar",
				"param1":     "param1",
			},
			"bar2": nil,
			"en1":  "A1",
			"en2":  "A0",
			"externalTime1": j{
				"seconds": float64(1123),
				"nanos":   float64(123),
			},
			"localTime2": nil,
		},
		"localTime": nil,
		"localTime2": j{
			"__typename": "Timestamp",
			"time":       "1234",
		},
	},
}

const constructsMapsQuery = `
mutation {
 constructsMaps_(
    in: {
      int32Int32: { key: 1, value: 1 }
      int64Int64: { key: 2, value: 2 }
      uint32Uint32: { key: 3, value: 3 }
      uint64Uint64: { key: 4, value: 4 }
      sint32Sint32: { key: 5, value: 5 }
      sint64Sint64: { key: 6, value: 5 }
      fixed32Fixed32: { key: 7, value: 7 }
      fixed64Fixed64: { key: 8, value: 8 }
      sfixed32Sfixed32: { key: 9, value: 9 }
      sfixed64Sfixed64: { key: 10, value: 10 }
      boolBool: { key: true, value: true }
      stringString: { key: "test1", value: "test1" }
      stringBytes: { key: "test2", value: "dGVzdA==" }
      stringFloat: { key: "test1", value: 11.1 }
      stringDouble: { key: "test1", value: 12.2 }
      stringFoo: { key: "test1", value: { param1: "param1", param2: "param2" } }
      stringBar: { key: "test1", value: BAR3 }
    }
  ) {
    int32Int32 { key value }
    int64Int64 { key value}
    uint32Uint32 { key value }
	uint64Uint64 { key value}
    sint32Sint32 { key value}
    sint64Sint64 { key value}
    fixed32Fixed32 { key value}
    fixed64Fixed64 { key value}
    sfixed32Sfixed32 { key value}
    sfixed64Sfixed64 { key value}
    boolBool { key value}
    stringString { key value}
    stringBytes { key value}
    stringFloat { key value}
    stringDouble { key value}
    stringFoo { key value { param1 param2 }}
    stringBar { key value}
  }
}
`

var constructsMapsResponse = j{
	"constructsMaps_": j{
		"boolBool":         l{j{"key": true, "value": true}},
		"stringBar":        l{j{"key": "test1", "value": "BAR3"}},
		"stringBytes":      l{j{"key": "test2", "value": "dGVzdA=="}},
		"stringDouble":     l{j{"key": "test1", "value": 12.2}},
		"stringFloat":      l{j{"key": "test1", "value": 11.1}},
		"stringFoo":        l{j{"key": "test1", "value": j{"param1": "param1", "param2": "param2"}}},
		"stringString":     l{j{"key": "test1", "value": "test1"}},
		"fixed32Fixed32":   l{j{"key": float64(7), "value": float64(7)}},
		"fixed64Fixed64":   l{j{"key": float64(8), "value": float64(8)}},
		"int32Int32":       l{j{"key": float64(1), "value": float64(1)}},
		"int64Int64":       l{j{"key": float64(2), "value": float64(2)}},
		"sfixed32Sfixed32": l{j{"key": float64(9), "value": float64(9)}},
		"sfixed64Sfixed64": l{j{"key": float64(10), "value": float64(10)}},
		"sint32Sint32":     l{j{"key": float64(5), "value": float64(5)}},
		"sint64Sint64":     l{j{"key": float64(6), "value": float64(5)}},
		"uint32Uint32":     l{j{"key": float64(3), "value": float64(3)}},
		"uint64Uint64":     l{j{"key": float64(4), "value": float64(4)}},
	},
}

const constructsRepeatedQuery = `
mutation {
 constructsRepeated_(
    in: {
      double: [1.1]
      float: [2.2]
      int32: [3]
      int64: [4]
      uint32: [7]
      uint64: [8]
      sint32: [9]
      sint64: [10]
      fixed32: [11]
      fixed64: [12]
      sfixed32: [13]
      sfixed64: [14]
      bool: [true]
      stringX: ["test"]
      bytes: ["dGVzdA=="]
      foo: { param1: "param1", param2: "param2" }
      bar: BAR3
    }
  ) {
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
    stringX
    bytes
    foo {param1 param2}
    bar
  }
}
`

var constructsRepeatedResponse = j{
	"constructsRepeated_": j{
		"bar":      l{"BAR3"},
		"bool":     l{true},
		"stringX":  l{"test"},
		"bytes":    l{"dGVzdA=="},
		"foo":      l{j{"param1": "param1", "param2": "param2"}},
		"double":   l{1.1},
		"float":    l{2.2},
		"fixed32":  l{float64(11)},
		"fixed64":  l{float64(12)},
		"int32":    l{float64(3)},
		"int64":    l{float64(4)},
		"sfixed32": l{float64(13)},
		"sfixed64": l{float64(14)},
		"sint32":   l{float64(9)},
		"sint64":   l{float64(10)},
		"uint32":   l{float64(7)},
		"uint64":   l{float64(8)},
	},
}

var constructsOneofsResponse = j{
	"constructsOneof_": j{
		"oneof1": j{
			"__typename": "Oneof_Oneof1",
			"param2":     "", //FIXME remove unrequested field
			"param3":     "3",
		},
		"oneof2": j{
			"__typename": "Oneof_Oneof2",
			"param4":     "", //FIXME remove unrequested field
			"param5":     "5",
		},
		"oneof3": j{
			"__typename": "Oneof_Oneof3",
			"param6":     "6",
		},
		"param1": "2",
	},
}

const constructsOneofsQuery = `
mutation {
  constructsOneof_(
    in: {param1: "2", param3:"3", param5: "5", param6: "6"}
  ){
    param1
    oneof1 {
      __typename
      ... on Oneof_Param2 {
        param2
      }
      ... on Oneof_Param3 {
        param3
      }
    }
    oneof2 {
      __typename
      ... on Oneof_Param4 {
        param4
      }
      ... on Oneof_Param5 {
        param5
      }
    }
    oneof3 {
      __typename
      ... on Oneof_Param6 {
        param6
      }
    }
  }
}
`
