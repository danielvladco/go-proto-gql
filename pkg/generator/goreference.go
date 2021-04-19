package generator

import (
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"
)

type GoRef interface {
	FindGoField(field string) *protogen.Field
}

func NewGoRef(req *pluginpb.CodeGeneratorRequest) (GoRef, error) {
	p, err := protogen.Options{}.New(req)
	if err != nil {
		return nil, err
	}
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
