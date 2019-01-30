package gqltypes

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/99designs/gqlgen/graphql"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
)

func MarshalBytes(b []byte) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		_, _ = fmt.Fprintf(w, "%q", string(b))
	})
}

func UnmarshalBytes(v interface{}) ([]byte, error) {
	switch v := v.(type) {
	case string:
		return []byte(v), nil
	case *string:
		return []byte(*v), nil
	case []byte:
		return v, nil
	default:
		return nil, fmt.Errorf("%T is not []byte", v)
	}
}

func MarshalAny(any *any.Any) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		d := &ptypes.DynamicAny{}
		if err := ptypes.UnmarshalAny(any, d); err != nil {
			panic("unable to unmarshal any: " + err.Error())
			return
		}

		if err := json.NewEncoder(w).Encode(d.Message); err != nil {
			panic("unable to encode json: " + err.Error())
		}
	})
}

func UnmarshalAny(v interface{}) (*any.Any, error) {
	switch v := v.(type) {
	case []byte:
		return nil, nil //TODO add an unmarshal mechanism
	case json.RawMessage:
		return nil, nil
	default:
		return nil, fmt.Errorf("%T is not json.RawMessage", v)
	}
}
