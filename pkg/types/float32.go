package types

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/99designs/gqlgen/graphql"
)

func MarshalFloat32(any float32) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
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
