package gqltypes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/99designs/gqlgen/graphql"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
)

type DummyResolver struct{}

func (r *DummyResolver) Dummy(ctx context.Context) (*bool, error) { return nil, nil }

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

func MarshalAny(any any.Any) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		d := &ptypes.DynamicAny{}
		if err := ptypes.UnmarshalAny(&any, d); err != nil {
			panic("unable to unmarshal any: " + err.Error())
			return
		}

		if err := json.NewEncoder(w).Encode(d.Message); err != nil {
			panic("unable to encode json: " + err.Error())
		}
	})
}

func UnmarshalAny(v interface{}) (any.Any, error) {
	switch v := v.(type) {
	case []byte:
		return any.Any{}, nil //TODO add an unmarshal mechanism
	case json.RawMessage:
		return any.Any{}, nil
	default:
		return any.Any{}, fmt.Errorf("%T is not json.RawMessage", v)
	}
}

func MarshalInt32(any int32) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
	})
}

func UnmarshalInt32(v interface{}) (int32, error) {
	switch v := v.(type) {
	case int32:
		return 0, nil //TODO add an unmarshal mechanism
	case json.Number:
		return 0, nil
	default:
		return 0, fmt.Errorf("%T is not int32", v)
	}
}

func MarshalInt64(any int64) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
	})
}

func UnmarshalInt64(v interface{}) (int64, error) {
	switch v := v.(type) {
	case int64:
		return 0, nil //TODO add an unmarshal mechanism
	case json.Number:
		return 0, nil
	default:
		return 0, fmt.Errorf("%T is not int64", v)
	}
}

func MarshalUint32(any uint32) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
	})
}

func UnmarshalUint32(v interface{}) (uint32, error) {
	switch v := v.(type) {
	case int32:
		return 0, nil //TODO add an unmarshal mechanism
	case json.Number:
		return 0, nil
	default:
		return 0, fmt.Errorf("%T is not int32", v)
	}
}

func MarshalUint64(any uint64) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
	})
}

func UnmarshalUint64(v interface{}) (uint64, error) {
	switch v := v.(type) {
	case uint64:
		return 0, nil //TODO add an unmarshal mechanism
	case json.Number:
		return 0, nil
	default:
		return 0, fmt.Errorf("%T is not int64", v)
	}
}

func MarshalFloat32(any float32) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
	})
}

func UnmarshalFloat32(v interface{}) (float32, error) {
	switch v := v.(type) {
	case float32:
		return 0, nil //TODO add an unmarshal mechanism
	case json.Number:
		return 0, nil
	default:
		return 0, fmt.Errorf("%T is not float32", v)
	}
}
