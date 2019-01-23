package plugin

import (
	"bytes"
	"fmt"
)

type Schema struct {
	indent  string
	index   int
	schemas []*bytes.Buffer
}

func NewBytes() *Schema {
	return &Schema{}
}

func (s *Schema) GetSchemaByIndex(index int) *bytes.Buffer {
	return s.schemas[index]
}

func (s *Schema) NextSchema() {
	s.indent = ""
	s.schemas = append(s.schemas, new(bytes.Buffer))
	s.index = len(s.schemas) - 1
}

// In Indents the output one tab stop.
func (s *Schema) In() { s.indent += "\t" }

// Out unindents the output one tab stop.
func (s *Schema) Out() {
	if len(s.indent) > 0 {
		s.indent = s.indent[1:]
	}
}
func (s *Schema) P(str ...interface{}) {
	g := s.schemas[s.index]
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
