package server

import (
	"fmt"
	"github.com/danielvladco/go-proto-gql/plugin"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/jhump/protoreflect/desc"
)

func NewMapping(gqlDesc GQLDescriptors, descs []*desc.FileDescriptor) *GQLProtoMapping {
	m := &GQLProtoMapping{
		inputs:  make(map[*desc.MessageDescriptor]*plugin.Type),
		objects: make(map[*desc.MessageDescriptor]*plugin.Type),

		messages:      make(map[*descriptor.DescriptorProto]*desc.MessageDescriptor),
		enums:         make(map[*descriptor.EnumDescriptorProto]*desc.EnumDescriptor),
		oneofs:        make(map[OneofKey]*desc.OneOfDescriptor),
		fields:        make(map[*descriptor.FieldDescriptorProto]*desc.FieldDescriptor),
		rpcs:          make(map[string]*plugin.Method),
		gqltypeToDesc: make(map[string]*desc.MessageDescriptor),
		pbtypeToDesc:  make(map[string]*plugin.Type),
		gqlDesc:       gqlDesc,
	}

	// TODO use maps from the start
	for _, q := range gqlDesc.Queries() {
		m.rpcs[q.Name] = q
	}

	for _, q := range gqlDesc.Mutations() {
		m.rpcs[q.Name] = q
	}

	for _, d := range descs {
		m.getEnums(d.GetEnumTypes())
	}

	for _, d := range descs {
		m.getMessages(d.GetMessageTypes())
	}

	for _, in := range gqlDesc.Inputs() {
		dd := m.messages[in.DescriptorProto]
		m.inputs[dd] = in
	}

	for _, obj := range gqlDesc.Types() {
		m.objects[m.messages[obj.DescriptorProto]] = obj
	}

	for _, tt := range gqlDesc.Types() {
		d, ok := m.messages[tt.DescriptorProto]
		if !ok {
			continue
		}

		m.pbtypeToDesc[d.GetFullyQualifiedName()] = tt
	}
	for _, tt := range gqlDesc.Inputs() {
		d, ok := m.messages[tt.DescriptorProto]
		if !ok {
			continue
		}
		m.gqltypeToDesc[gqlDesc.GqlModelNames()[tt]] = d
	}
	return m
}

type OneofKey struct {
	desc         *descriptor.DescriptorProto
	gqlFieldName string
}

type GQLProtoMapping struct {
	inputs  map[*desc.MessageDescriptor]*plugin.Type
	objects map[*desc.MessageDescriptor]*plugin.Type

	messages map[*descriptor.DescriptorProto]*desc.MessageDescriptor
	enums    map[*descriptor.EnumDescriptorProto]*desc.EnumDescriptor
	oneofs   map[OneofKey]*desc.OneOfDescriptor
	fields   map[*descriptor.FieldDescriptorProto]*desc.FieldDescriptor

	rpcs map[string]*plugin.Method

	gqlDesc GQLDescriptors

	gqltypeToDesc map[string]*desc.MessageDescriptor
	pbtypeToDesc  map[string]*plugin.Type
}

func (c GQLProtoMapping) getEnums(enums []*desc.EnumDescriptor) {
	for _, e := range enums {
		c.enums[e.AsEnumDescriptorProto()] = e
	}
}

func (c GQLProtoMapping) getMessages(messages []*desc.MessageDescriptor) {
	for _, e := range messages {
		msg := e.AsDescriptorProto()

		for i, oo := range e.GetOneOfs() {
			c.oneofs[OneofKey{
				desc:         msg,
				gqlFieldName: plugin.ToLowerFirst(generator.CamelCase(msg.OneofDecl[i].GetName())),
			}] = oo
		}

		c.getMessages(e.GetNestedMessageTypes())

		c.getEnums(e.GetNestedEnumTypes())

		for _, f := range e.GetFields() {
			c.fields[f.AsFieldDescriptorProto()] = f
		}

		c.messages[msg] = e
	}
}

func (c GQLProtoMapping) InputType(methodName string) (in *plugin.Type, err error) {
	query, ok := c.rpcs[methodName]
	if !ok {
		return nil, fmt.Errorf("no item: %s", methodName)
	}

	if input, ok := c.gqlDesc.Inputs()[query.InputType]; ok {
		in = input
	} else if scalar, ok := c.gqlDesc.Scalars()[query.InputType]; ok {
		in = scalar
	}
	return in, err
}

func (c GQLProtoMapping) GetMethod(methodName string) (in *plugin.Method, err error) {
	m, ok := c.rpcs[methodName]
	if !ok {
		return nil, fmt.Errorf("no item: %s", methodName)
	}

	return m, err
}
