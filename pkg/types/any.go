package types

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/99designs/gqlgen/graphql"
	"google.golang.org/protobuf/types/known/anypb"
)

// TODO create mapping for proto to graphql types to unmarshal any
func MarshalAny(any *anypb.Any) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		d, err := any.UnmarshalNew()
		if err != nil {
			panic("unable to unmarshal any: " + err.Error())
			return
		}

		if err := json.NewEncoder(w).Encode(d); err != nil {
			panic("unable to encode json: " + err.Error())
		}
	})
}

func UnmarshalAny(v interface{}) (*anypb.Any, error) {
	switch v := v.(type) {
	case []byte:
		return &anypb.Any{}, nil //TODO add an unmarshal mechanism
	case json.RawMessage:
		return &anypb.Any{}, nil
	default:
		return &anypb.Any{}, fmt.Errorf("%T is not json.RawMessage", v)
	}
}
