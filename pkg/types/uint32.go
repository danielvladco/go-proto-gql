package types

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/99designs/gqlgen/graphql"
)

func MarshalUint32(i uint32) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		io.WriteString(w, strconv.FormatUint(uint64(i), 10))
	})
}

func UnmarshalUint32(v interface{}) (uint32, error) {
	switch v := v.(type) {
	case string:
		ui64, err := strconv.ParseUint(v, 10, 32)
		return uint32(ui64), err
	case int:
		return uint32(v), nil
	case uint:
		return uint32(v), nil
	case int32:
		return uint32(v), nil
	case uint32:
		return v, nil
	case json.Number:
		ui64, err := strconv.ParseUint(string(v), 10, 32)
		return uint32(ui64), err
	default:
		return 0, fmt.Errorf("%T is not an int", v)
	}
}
