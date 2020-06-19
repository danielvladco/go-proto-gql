package generator

import (
	"reflect"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/jhump/protoreflect/desc"
	"github.com/vektah/gqlparser/v2/ast"
)

type ObjectDefinition struct {
	*ast.Definition
	desc.Descriptor

	fields     []*FieldDescriptor
	fieldNames map[string]*FieldDescriptor
}

func (o *ObjectDefinition) AsGraphql() *ast.Definition {
	return o.Definition
}

func (o *ObjectDefinition) uniqueName(f *desc.FieldDescriptor) string {
	return strings.Title(f.GetName())
}

func (o *ObjectDefinition) IsInput() bool {
	return o.Kind == ast.InputObject
}

func (o *ObjectDefinition) GetFields() []*FieldDescriptor {
	return o.fields
}

func (o *ObjectDefinition) IsMessage() bool {
	_, ok := o.Descriptor.(*desc.MessageDescriptor)
	return ok
}

// same isEmpty but for mortals
func (o *ObjectDefinition) IsEmpty() bool { return o.isEmpty(make(map[*ObjectDefinition]struct{})) }

// make sure objects are fulled with all objects
func (o *ObjectDefinition) isEmpty(callstack map[*ObjectDefinition]struct{}) bool {
	callstack[o] = struct{}{} // don't forget to free stack on calling return when needed
	t, ok := o.Descriptor.(*desc.MessageDescriptor)
	if !ok {
		delete(callstack, o)
		return false
	}
	if len(t.GetFields()) == 0 {
		return true
	}
	for _, f := range o.GetFields() {
		objType := f.GetType()
		if objType == nil {
			delete(callstack, o)
			return false
		}
		if !objType.IsMessage() {
			delete(callstack, o)
			return false
		}

		// check if the call stack already contains a reference to this type and prevent it from calling itself again
		if _, ok := callstack[objType]; ok {
			return true
		}
		if !objType.isEmpty(callstack) {
			delete(callstack, o)
			return false
		}
	}

	delete(callstack, o)
	return true
}

func (o *ObjectDefinition) IsAny() bool {
	messageType := proto.MessageType(o.GetFullyQualifiedName())
	if messageType == nil {
		return false
	}
	_, ok := reflect.New(messageType).Elem().Interface().(*any.Any)
	return ok
}
