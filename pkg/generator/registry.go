package generator

import (
	"github.com/jhump/protoreflect/desc"
	"github.com/vektah/gqlparser/v2/ast"
)

type Registry interface {
	FindMethodByName(op ast.Operation, name string) *desc.MethodDescriptor
	FindObjectByName(name string) *desc.MessageDescriptor

	// Todo maybe find a better way to get ast definition
	FindObjectByFullyQualifiedName(fqn string) (*desc.MessageDescriptor, *ast.Definition)
	FindFieldByName(msg desc.Descriptor, name string) *desc.FieldDescriptor
}

func NewRegistry(files SchemaDescriptorList) Registry {
	v := &repository{
		methodsByName:   map[ast.Operation]map[string]*desc.MethodDescriptor{},
		objectsByName:   map[string]*desc.MessageDescriptor{},
		objectsByFQN:    map[string]*ObjectDescriptor{},
		gqlFieldsByName: map[desc.Descriptor]map[string]*desc.FieldDescriptor{},
	}
	for _, f := range files {
		v.methodsByName[ast.Mutation] = map[string]*desc.MethodDescriptor{}
		for _, m := range f.GetMutation().Methods() {
			v.methodsByName[ast.Mutation][m.Name] = m.MethodDescriptor
		}
		v.methodsByName[ast.Query] = map[string]*desc.MethodDescriptor{}
		for _, m := range f.GetQuery().Methods() {
			v.methodsByName[ast.Query][m.Name] = m.MethodDescriptor
		}
		v.methodsByName[ast.Subscription] = map[string]*desc.MethodDescriptor{}
		for _, m := range f.GetSubscription().Methods() {
			v.methodsByName[ast.Subscription][m.Name] = m.MethodDescriptor
		}
	}
	for _, f := range files {
		for _, m := range f.Objects() {
			v.gqlFieldsByName[m.Descriptor] = map[string]*desc.FieldDescriptor{}
			for _, f := range m.GetFields() {
				v.gqlFieldsByName[m.Descriptor][f.Name] = f.FieldDescriptor
			}

			switch msgDesc := m.Descriptor.(type) {
			case *desc.MessageDescriptor:
				v.objectsByFQN[m.GetFullyQualifiedName()] = m
				v.objectsByName[m.Name] = msgDesc
			}
		}
	}
	return v
}

type repository struct {
	files SchemaDescriptorList

	methodsByName   map[ast.Operation]map[string]*desc.MethodDescriptor
	objectsByName   map[string]*desc.MessageDescriptor
	objectsByFQN    map[string]*ObjectDescriptor
	gqlFieldsByName map[desc.Descriptor]map[string]*desc.FieldDescriptor
}

func (r repository) FindMethodByName(op ast.Operation, name string) *desc.MethodDescriptor {
	m, _ := r.methodsByName[op][name]
	return m
}

func (r repository) FindObjectByName(name string) *desc.MessageDescriptor {
	o, _ := r.objectsByName[name]
	return o
}

func (r repository) FindObjectByFullyQualifiedName(fqn string) (*desc.MessageDescriptor, *ast.Definition) {
	o, _ := r.objectsByFQN[fqn]
	msg, _ := o.Descriptor.(*desc.MessageDescriptor)
	return msg, o.Definition
}

func (r repository) FindFieldByName(msg desc.Descriptor, name string) *desc.FieldDescriptor {
	f, _ := r.gqlFieldsByName[msg][name]
	return f
}
