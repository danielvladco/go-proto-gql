package plugin

import (
	"bytes"
	"fmt"

	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
)

type schema struct {
	indent string
	buffer *bytes.Buffer
	*generator.Generator
}

// In Indents the output one tab stop.
func (s *schema) In() { s.indent += "\t" }

// Out unindents the output one tab stop.
func (s *schema) Out() {
	if len(s.indent) > 0 {
		s.indent = s.indent[1:]
	}
}
func (s *schema) P(str ...interface{}) {
	g := s.buffer
	_, _ = fmt.Fprint(g, s.indent)
	for _, v := range str {
		switch s := v.(type) {
		case string:
			_, _ = fmt.Fprint(g, s)
		case *string:
			_, _ = fmt.Fprint(g, *s)
		case bool:
			_, _ = fmt.Fprintf(g, "%t", s)
		case *bool:
			_, _ = fmt.Fprintf(g, "%t", *s)
		case int:
			_, _ = fmt.Fprintf(g, "%d", s)
		case *int32:
			_, _ = fmt.Fprintf(g, "%d", *s)
		case *int64:
			_, _ = fmt.Fprintf(g, "%d", *s)
		case float64:
			_, _ = fmt.Fprintf(g, "%g", s)
		case *float64:
			_, _ = fmt.Fprintf(g, "%g", *s)
		default:
			panic(fmt.Sprintf("unknown type in printer: %T", v))
		}
	}
	g.WriteByte('\n')
}
