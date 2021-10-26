package generator

import (
	"strings"

	"github.com/jhump/protoreflect/desc"
	"github.com/vektah/gqlparser/v2/ast"
	any "google.golang.org/protobuf/types/known/anypb"
)

type ObjectDescriptor struct {
	*ast.Definition
	desc.Descriptor

	types      []*ObjectDescriptor
	fields     []*FieldDescriptor
	fieldNames map[string]*FieldDescriptor
}

func (o *ObjectDescriptor) AsGraphql() *ast.Definition {
	return o.Definition
}

func (o *ObjectDescriptor) uniqueName(f *desc.FieldDescriptor) string {
	return strings.Title(f.GetName())
}

func (o *ObjectDescriptor) IsInput() bool {
	return o.Kind == ast.InputObject
}

func (o *ObjectDescriptor) GetFields() []*FieldDescriptor {
	return o.fields
}

func (o *ObjectDescriptor) GetTypes() []*ObjectDescriptor {
	return o.types
}

func (o *ObjectDescriptor) IsMessage() bool {
	_, ok := o.Descriptor.(*desc.MessageDescriptor)
	return ok
}

// same isEmpty but for mortals
func IsEmpty(o *desc.MessageDescriptor) bool { return isEmpty(o, NewCallstack()) }

// make sure objects are fulled with all objects
func isEmpty(o *desc.MessageDescriptor, callstack Callstack) bool {
	callstack.Push(o)
	defer callstack.Pop(o)

	if len(o.GetFields()) == 0 {
		return true
	}
	for _, f := range o.GetFields() {
		objType := f.GetMessageType()
		if objType == nil {
			return false
		}

		// check if the call stack already contains a reference to this type and prevent it from calling itself again
		if callstack.Has(objType) {
			return true
		}
		if !isEmpty(objType, callstack) {
			return false
		}
	}

	return true
}

// TODO maybe not compare by strings
func IsAny(o *desc.MessageDescriptor) bool {
	return string((&any.Any{}).ProtoReflect().Descriptor().FullName()) == o.GetFullyQualifiedName()
}
