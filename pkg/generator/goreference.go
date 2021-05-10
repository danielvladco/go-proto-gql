package generator

import (
	"google.golang.org/protobuf/compiler/protogen"
)

type GoRef interface {
	FindGoField(field string) *protogen.Field
}

func NewGoRef(p *protogen.Plugin) (GoRef, error) {
	return goRef{p}, nil
}

type goRef struct {
	*protogen.Plugin
}

func (g goRef) FindGoField(field string) *protogen.Field {
	for _, file := range g.Files {
		for _, msg := range file.Messages {
			for _, f := range msg.Fields {
				if string(f.Desc.FullName()) == field {
					return f
				}
			}
		}
	}
	return nil
}
