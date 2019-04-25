package gql

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
)

type DummyResolver struct{}

func (r *DummyResolver) Dummy(ctx context.Context) (*bool, error) { return nil, nil }

func MarshalBytes(b []byte) Marshaler {
	return WriterFunc(func(w io.Writer) {
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
	case json.RawMessage:
		return []byte(v), nil
	default:
		return nil, fmt.Errorf("%T is not []byte", v)
	}
}

func MarshalAny(any any.Any) Marshaler {
	return WriterFunc(func(w io.Writer) {
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

func MarshalInt32(any int32) Marshaler {
	return WriterFunc(func(w io.Writer) {
		_, _ = w.Write([]byte(strconv.Itoa(int(any))))
	})
}

func UnmarshalInt32(v interface{}) (int32, error) {
	switch v := v.(type) {
	case int:
		return int32(v), nil
	case int32:
		return v, nil
	case json.Number:
		i, err := v.Int64()
		return int32(i), err
	default:
		return 0, fmt.Errorf("%T is not int32", v)
	}
}

func MarshalInt64(any int64) Marshaler {
	return WriterFunc(func(w io.Writer) {
		_, _ = w.Write([]byte(strconv.Itoa(int(any))))
	})
}

func UnmarshalInt64(v interface{}) (int64, error) {
	switch v := v.(type) {
	case int:
		return int64(v), nil
	case int64:
		return v, nil
	case json.Number:
		i, err := v.Int64()
		return i, err
	default:
		return 0, fmt.Errorf("%T is not int32", v)
	}
}

func MarshalUint32(any uint32) Marshaler {
	return WriterFunc(func(w io.Writer) {
		_, _ = w.Write([]byte(strconv.Itoa(int(any))))
	})
}

func UnmarshalUint32(v interface{}) (uint32, error) {
	switch v := v.(type) {
	case int:
		return uint32(v), nil
	case uint32:
		return v, nil
	case json.Number:
		i, err := v.Int64()
		return uint32(i), err
	default:
		return 0, fmt.Errorf("%T is not int32", v)
	}
}

func MarshalUint64(any uint64) Marshaler {
	return WriterFunc(func(w io.Writer) {
		_, _ = w.Write([]byte(strconv.Itoa(int(any))))
	})
}

func UnmarshalUint64(v interface{}) (uint64, error) {
	switch v := v.(type) {
	case int:
		return uint64(v), nil
	case uint64:
		return v, nil //TODO add an unmarshal mechanism
	case json.Number:
		i, err := v.Int64()
		return uint64(i), err
	default:
		return 0, fmt.Errorf("%T is not uint64", v)
	}
}

func MarshalFloat32(any float32) Marshaler {
	return WriterFunc(func(w io.Writer) {
		_, _ = w.Write([]byte(strconv.Itoa(int(any))))
	})
}

func UnmarshalFloat32(v interface{}) (float32, error) {
	switch v := v.(type) {
	case int:
		return float32(v), nil
	case float32:
		return v, nil
	case json.Number:
		f, err := v.Float64()
		return float32(f), err
	default:
		return 0, fmt.Errorf("%T is not float32", v)
	}
}

type Marshaler interface {
	MarshalGQL(w io.Writer)
}

type WriterFunc func(writer io.Writer)

func (f WriterFunc) MarshalGQL(w io.Writer) {
	f(w)
}
