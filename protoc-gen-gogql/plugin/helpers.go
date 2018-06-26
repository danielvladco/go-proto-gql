package plugin

import (
	"bytes"
	"fmt"
)

type Bytes struct {
	*bytes.Buffer
	indent string
}

func NewBytes() *Bytes {
	return &Bytes{Buffer: &bytes.Buffer{}}
}

// In Indents the output one tab stop.
func (g *Bytes) In() { g.indent += "\t" }

// Out unindents the output one tab stop.
func (g *Bytes) Out() {
	if len(g.indent) > 0 {
		g.indent = g.indent[1:]
	}
}
func (g *Bytes) P(str ...interface{}) {
	fmt.Fprint(g, g.indent)
	for _, v := range str {
		switch s := v.(type) {
		case string:
			fmt.Fprint(g, s)
		case *string:
			fmt.Fprint(g, *s)
		case bool:
			fmt.Fprintf(g, "%t", s)
		case *bool:
			fmt.Fprintf(g, "%t", *s)
		case int:
			fmt.Fprintf(g, "%d", s)
		case *int32:
			fmt.Fprintf(g, "%d", *s)
		case *int64:
			fmt.Fprintf(g, "%d", *s)
		case float64:
			fmt.Fprintf(g, "%g", s)
		case *float64:
			fmt.Fprintf(g, "%g", *s)
		default:
			panic(fmt.Sprintf("unknown type in printer: %T", v))
		}
	}
	g.WriteByte('\n')
}
