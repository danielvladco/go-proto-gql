package protoparser

import (
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
)

type Option func(p *protoparse.Parser)

func WithSourceCodeInfo(v bool) Option {
	return func(p *protoparse.Parser) {
		p.IncludeSourceCodeInfo = v
	}
}

func Parse(importPaths, fileNames []string, options ...Option) ([]*desc.FileDescriptor, error) {
	newFileNames, err := protoparse.ResolveFilenames(importPaths, fileNames...)
	if err != nil {
		return nil, err
	}
	parser := &protoparse.Parser{ImportPaths: importPaths, IncludeSourceCodeInfo: true}
	for _, o := range options {
		o(parser)
	}
	descs, err := parser.ParseFiles(newFileNames...)
	if err != nil {
		return nil, err
	}
	return descs, nil
}
