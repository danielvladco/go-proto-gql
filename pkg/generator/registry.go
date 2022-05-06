package generator

import (
	"sync"

	"github.com/jhump/protoreflect/desc"
	"github.com/vektah/gqlparser/v2/ast"
)

type Registry interface {
	FindMethodByName(op ast.Operation, name string) *desc.MethodDescriptor
	FindObjectByName(name string) *desc.MessageDescriptor

	// FindObjectByFullyQualifiedName TODO maybe find a better way to get ast definition
	FindObjectByFullyQualifiedName(fqn string) (*desc.MessageDescriptor, *ast.Definition)

	// FindFieldByName given the proto Descriptor and the graphql field name get the the proto FieldDescriptor
	FindFieldByName(msg desc.Descriptor, name string) *desc.FieldDescriptor

	// FindUnionFieldByMessageFQNAndName given the proto Descriptor and the graphql field name get the the proto FieldDescriptor
	FindUnionFieldByMessageFQNAndName(fqn, name string) *desc.FieldDescriptor

	FindGraphqlFieldByProtoField(msg *ast.Definition, name string) *ast.FieldDefinition
}

func NewRegistry(files SchemaDescriptorList) Registry {
	v := &repository{
		mu:                       &sync.RWMutex{},
		methodsByName:            map[ast.Operation]map[string]*desc.MethodDescriptor{},
		objectsByName:            map[string]*desc.MessageDescriptor{},
		objectsByFQN:             map[string]*ObjectDescriptor{},
		graphqlFieldsByName:      map[desc.Descriptor]map[string]*desc.FieldDescriptor{},
		graphqlUnionFieldsByName: map[string]map[string]*desc.FieldDescriptor{},
		protoFieldsByName:        map[*ast.Definition]map[string]*ast.FieldDefinition{},
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
			switch m.Kind {
			case ast.Union:
				fqn := m.Descriptor.GetParent().GetFullyQualifiedName()
				if _, ok := v.graphqlUnionFieldsByName[fqn]; !ok {
					v.graphqlUnionFieldsByName[fqn] = map[string]*desc.FieldDescriptor{}
				}
				for _, tt := range m.GetTypes() {
					for _, f := range tt.GetFields() {
						v.graphqlUnionFieldsByName[fqn][f.Name] = f.FieldDescriptor
					}
				}
			case ast.Object:
				v.protoFieldsByName[m.Definition] = map[string]*ast.FieldDefinition{}
				for _, f := range m.GetFields() {
					if f.FieldDescriptor != nil {
						v.protoFieldsByName[m.Definition][f.FieldDescriptor.GetName()] = f.FieldDefinition
					}
				}
			case ast.InputObject:
				v.graphqlFieldsByName[m.Descriptor] = map[string]*desc.FieldDescriptor{}
				for _, f := range m.GetFields() {
					v.graphqlFieldsByName[m.Descriptor][f.Name] = f.FieldDescriptor
				}
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
	mu    *sync.RWMutex

	methodsByName            map[ast.Operation]map[string]*desc.MethodDescriptor
	objectsByName            map[string]*desc.MessageDescriptor
	objectsByFQN             map[string]*ObjectDescriptor
	graphqlFieldsByName      map[desc.Descriptor]map[string]*desc.FieldDescriptor
	protoFieldsByName        map[*ast.Definition]map[string]*ast.FieldDefinition
	graphqlUnionFieldsByName map[string]map[string]*desc.FieldDescriptor
}

func (r repository) FindMethodByName(op ast.Operation, name string) *desc.MethodDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, _ := r.methodsByName[op][name]
	return m
}

func (r repository) FindObjectByName(name string) *desc.MessageDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	o, _ := r.objectsByName[name]
	return o
}

func (r repository) FindObjectByFullyQualifiedName(fqn string) (*desc.MessageDescriptor, *ast.Definition) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	o, _ := r.objectsByFQN[fqn]
	msg, _ := o.Descriptor.(*desc.MessageDescriptor)
	return msg, o.Definition
}

func (r repository) FindFieldByName(msg desc.Descriptor, name string) *desc.FieldDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	f, _ := r.graphqlFieldsByName[msg][name]
	return f
}

func (r repository) FindUnionFieldByMessageFQNAndName(fqn, name string) *desc.FieldDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	f, _ := r.graphqlUnionFieldsByName[fqn][name]
	return f
}

func (r repository) FindGraphqlFieldByProtoField(msg *ast.Definition, name string) *ast.FieldDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	f, _ := r.protoFieldsByName[msg][name]
	return f
}
