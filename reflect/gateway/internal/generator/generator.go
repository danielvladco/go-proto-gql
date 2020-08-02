package generator

import (
	"fmt"
	gql "github.com/danielvladco/go-proto-gql/pb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/jhump/protoreflect/desc"
	"github.com/vektah/gqlparser/v2/ast"
	"strconv"
	"strings"
)

func NewSchemas(files []*desc.FileDescriptor, mergeSchemas bool) (SchemaDescriptorList, error) {
	schema := &SchemaDescriptor{
		Schema: &ast.Schema{
			Directives: map[string]*ast.DirectiveDefinition{},
			Types:      map[string]*ast.Definition{},
		},
		reservedNames: graphqlReservedNames,
	}

	if mergeSchemas {
		for _, file := range files {
			err := generateFile(file, schema)
			if err != nil {
				return nil, err
			}
		}

		return []*SchemaDescriptor{schema}, nil
	}

	var schemas []*SchemaDescriptor
	for _, file := range files {
		schema := *schema
		err := generateFile(file, &schema)
		if err != nil {
			return nil, err
		}

		schemas = append(schemas, &schema)
	}

	return schemas, nil
}

func generateFile(file *desc.FileDescriptor, schema *SchemaDescriptor) error {
	for _, svc := range file.GetServices() {
		for _, rpc := range svc.GetMethods() {
			in, err := schema.CreateObjects(rpc.GetInputType(), true)
			if err != nil {
				return err
			}

			out, err := schema.CreateObjects(rpc.GetOutputType(), false)
			if err != nil {
				return err
			}

			if rpc.IsClientStreaming() && rpc.IsServerStreaming() {
				schema.GetMutation().addMethod(schema.createMethod(svc, rpc, in, out))
			}

			if rpc.IsServerStreaming() {
				schema.GetSubscription().addMethod(schema.createMethod(svc, rpc, in, out))
			} else {
				switch getMethodType(rpc) {
				case gql.Type_DEFAULT:
					switch ttt := getServiceType(svc); ttt {
					case gql.Type_DEFAULT, gql.Type_MUTATION:
						schema.GetMutation().addMethod(schema.createMethod(svc, rpc, in, out))
					case gql.Type_QUERY:
						schema.GetQuery().addMethod(schema.createMethod(svc, rpc, in, out))
					}
				case gql.Type_MUTATION:
					schema.GetMutation().addMethod(schema.createMethod(svc, rpc, in, out))
				case gql.Type_QUERY:
					schema.GetQuery().addMethod(schema.createMethod(svc, rpc, in, out))
				}
			}
		}
	}

	return nil
}

type SchemaDescriptorList []*SchemaDescriptor

func (s SchemaDescriptorList) AsGraphql() (astSchema []*ast.Schema) {
	for _, ss := range s {
		queryDef := ss.GetQuery().Definition
		mutationDef := ss.GetMutation().Definition
		subscriptionsDef := ss.GetSubscription().Definition
		ss.Schema.Query = queryDef
		ss.Schema.Types["Query"] = queryDef
		if ss.query.methods == nil {
			ss.Schema.Query.Fields = append(ss.Schema.Query.Fields, &ast.FieldDefinition{
				Name: "dummy",
				Type: ast.NamedType("Boolean", &ast.Position{}),
			})
		} else {
			for _, m := range ss.query.methods {
				ss.Schema.Query.Fields = append(ss.Schema.Query.Fields, m.AsGraphql())
			}
		}
		if ss.mutation.methods != nil {
			ss.Schema.Mutation = mutationDef
			ss.Schema.Types["Mutation"] = mutationDef

			for _, m := range ss.mutation.methods {
				ss.Schema.Mutation.Fields = append(ss.Schema.Mutation.Fields, m.AsGraphql())
			}
		}
		if ss.subscription.methods != nil {
			ss.Schema.Subscription = subscriptionsDef
			ss.Schema.Types["Subscription"] = subscriptionsDef
			for _, m := range ss.subscription.methods {
				ss.Schema.Subscription.Fields = append(ss.Schema.Subscription.Fields, m.AsGraphql())
			}
		}

		for _, o := range ss.objects {
			def := o.AsGraphql()
			ss.Schema.Types[def.Name] = def
		}

		astSchema = append(astSchema, ss.Schema)
	}
	return
}

type SchemaDescriptor struct {
	*ast.Schema
	files []*desc.FileDescriptor

	query        *RootDefinition
	mutation     *RootDefinition
	subscription *RootDefinition

	objects []*ObjectDescriptor

	reservedNames map[string]desc.Descriptor
}

func (s *SchemaDescriptor) Objects() []*ObjectDescriptor {
	return s.objects
}

func (s *SchemaDescriptor) GetMutation() *RootDefinition {
	if s.mutation == nil {
		s.mutation = NewRootDefinition(Mutation)
	}
	return s.mutation
}

func (s *SchemaDescriptor) GetSubscription() *RootDefinition {
	if s.subscription == nil {
		s.subscription = NewRootDefinition(Subscription)
	}
	return s.subscription
}

func (s *SchemaDescriptor) GetQuery() *RootDefinition {
	if s.query == nil {
		s.query = NewRootDefinition(Query)
	}

	return s.query
}

// make name be unique
// just create a map and register every name
func (s *SchemaDescriptor) uniqueName(d desc.Descriptor, input bool) (name string) {
	if input {
		name = strings.Title(CamelCaseSlice(strings.Split(strings.TrimPrefix(d.GetFullyQualifiedName(), d.GetFile().GetPackage()+"."), ".")) + "Input")
	} else {
		name = strings.Title(CamelCaseSlice(strings.Split(strings.TrimPrefix(d.GetFullyQualifiedName(), d.GetFile().GetPackage()+"."), ".")))
	}

	d2, ok := s.reservedNames[name]
	if ok {
		if d == d2 {
			return name //fixme
		}
	}

	originalName := name
	for uniqueSuffix := 0; ; uniqueSuffix++ {
		_, ok := s.reservedNames[name]
		if !ok {
			break
		}
		if uniqueSuffix == 0 {
			name = CamelCaseSlice(strings.Split(d.GetFile().GetPackage(), ".")) + "_" + originalName
			continue
		}
		name = CamelCaseSlice(strings.Split(d.GetFile().GetPackage(), ".")) + "_" + originalName + strconv.Itoa(uniqueSuffix)
	}

	s.reservedNames[name] = d
	return
}

func (s *SchemaDescriptor) CreateObjects(msg *desc.MessageDescriptor, input bool) (*ObjectDescriptor, error) {
	return s.createObjects(msg, input, NewCallstack())
}

func (s *SchemaDescriptor) createObjects(d desc.Descriptor, input bool, callstack Callstack) (obj *ObjectDescriptor, err error) {
	// the case if trying to resolve a primitive as a object. In this case we just return nil
	if d == nil {
		return
	}
	callstack.Push(d) // if add a return statement dont forget to clean stack
	defer callstack.Pop(d)
	switch dd := d.(type) {
	case *desc.MessageDescriptor:
		if IsEmpty(dd) {
			return nil, nil
		}
		if IsAny(dd) {
			any := s.createScalar("Any", "Any is any json type")
			s.objects = append(s.objects, any)
			return any, nil
		}

		kind := ast.Object
		if input {
			kind = ast.InputObject
		}
		fields := FieldDescriptorList{}
		for _, df := range dd.GetFields() {
			var fieldDirective []*ast.Directive
			if callstack.Has(df.GetMessageType()) {
				continue
			}
			if oneof := df.GetOneOf(); oneof != nil {
				if !input {
					field, err := s.createUnion(oneof, callstack)
					if err != nil {
						return nil, err
					}
					fields = append(fields, field)
					continue
				}

				// create oneofs as directives for input objects
				directive := &ast.DirectiveDefinition{
					Description: getDescription(oneof),
					Name:        s.uniqueName(oneof, input),
					Locations:   []ast.DirectiveLocation{ast.LocationInputFieldDefinition},
					Position:    &ast.Position{Src: &ast.Source{}},
				}
				s.Schema.Directives[directive.Name] = directive
				fieldDirective = append(fieldDirective, &ast.Directive{
					Name:     directive.Name,
					Position: &ast.Position{Src: &ast.Source{}},
					//ParentDefinition: obj.Definition, TODO
					Definition: directive,
					Location:   ast.LocationInputFieldDefinition,
				})
			}

			obj, err = s.createObjects(resolveFieldType(df), input, callstack)
			if err != nil {
				return nil, err
			}

			f, err := s.createField(df, obj)
			if err != nil {
				return nil, err
			}
			f.Directives = fieldDirective
			fields = append(fields, f)
		}

		obj = &ObjectDescriptor{
			Definition: &ast.Definition{
				Kind:        kind,
				Description: getDescription(d),
				Name:        s.uniqueName(d, input),
				Fields:      fields.AsGraphql(),
				Position:    &ast.Position{},
			},
			Descriptor: d,
			fields:     fields,
		}
	case *desc.EnumDescriptor:
		obj = &ObjectDescriptor{
			Definition: &ast.Definition{
				Kind:        ast.Enum,
				Description: getDescription(d),
				Name:        s.uniqueName(d, input),
				EnumValues:  enumValues(dd.GetValues()),
				Position:    &ast.Position{},
			},
			Descriptor: d,
		}
	default:
		panic(fmt.Sprintf("received unexpected value %v of type %T", dd, dd))
	}

	//if obj.IsEmpty() {
	//	return obj, nil
	//}
	//if obj.IsAny() {
	//	any := s.createScalar("Any", "Any is any json type")
	//	s.objects = append(s.objects, any)
	//	return any, nil
	//}
	s.objects = append(s.objects, obj)
	return obj, nil
}

func resolveFieldType(field *desc.FieldDescriptor) desc.Descriptor {
	msgType := field.GetMessageType()
	enumType := field.GetEnumType()
	if msgType != nil {
		return msgType
	}
	if enumType != nil {
		return enumType
	}
	return nil
}

func enumValues(evals []*desc.EnumValueDescriptor) (vlist ast.EnumValueList) {
	for _, eval := range evals {
		vlist = append(vlist, &ast.EnumValueDefinition{
			Description: getDescription(eval),
			Name:        eval.GetName(),
			Position:    &ast.Position{},
		})
	}

	return vlist
}

type FieldDescriptorList []*FieldDescriptor

func (fl FieldDescriptorList) AsGraphql() (dl []*ast.FieldDefinition) {
	for _, f := range fl {
		dl = append(dl, f.FieldDefinition)
	}
	return dl
}

type FieldDescriptor struct {
	*ast.FieldDefinition
	*desc.FieldDescriptor

	typ *ObjectDescriptor
}

func (f *FieldDescriptor) GetType() *ObjectDescriptor {
	return f.typ
}

type MethodDescriptor struct {
	*desc.ServiceDescriptor
	*desc.MethodDescriptor

	*ast.FieldDefinition

	input  *ObjectDescriptor
	output *ObjectDescriptor
}

func (m *MethodDescriptor) AsGraphql() *ast.FieldDefinition {
	return m.FieldDefinition
}

func (m *MethodDescriptor) GetInput() *ObjectDescriptor {
	return m.input
}

func (m *MethodDescriptor) GetOutput() *ObjectDescriptor {
	return m.output
}

type RootDefinition struct {
	*ast.Definition

	methods     []*MethodDescriptor
	methodNames map[string]*MethodDescriptor
}

func (r *RootDefinition) Methods() []*MethodDescriptor {
	return r.methods
}

func (r *RootDefinition) addMethod(method *MethodDescriptor) {
	r.methods = append(r.methods, method)
	// TODO maybe not do it here?
	r.Definition.Fields = append(r.Definition.Fields, method.FieldDefinition)
}

type rootName string

const (
	Mutation     rootName = "Mutation"
	Query        rootName = "Query"
	Subscription rootName = "Subscription"
)

func NewRootDefinition(name rootName) *RootDefinition {
	return &RootDefinition{Definition: &ast.Definition{
		Kind:     ast.Object,
		Name:     string(name),
		Position: &ast.Position{},
	}}
}

func getDescription(descs ...desc.Descriptor) string {
	var description []string
	for _, d := range descs {
		info := d.GetSourceInfo()
		if info == nil {
			continue
		}
		if info.LeadingComments != nil {
			description = append(description, *info.LeadingComments)
		}
		if info.TrailingComments != nil {
			description = append(description, *info.TrailingComments)
		}
	}

	return strings.Join(description, "\n")
}

func (s *SchemaDescriptor) createMethod(svc *desc.ServiceDescriptor, rpc *desc.MethodDescriptor, in, out *ObjectDescriptor) *MethodDescriptor {
	var args ast.ArgumentDefinitionList
	if in != nil && in.Descriptor != nil && !IsEmpty(in.Descriptor.(*desc.MessageDescriptor)) {
		args = append(args, &ast.ArgumentDefinition{
			Description:  "",
			DefaultValue: nil,
			Name:         "in",
			Type:         ast.NamedType(in.Name, &ast.Position{}),
			Directives:   nil,
			Position:     &ast.Position{},
		})
	}
	objType := ast.NamedType("Boolean", &ast.Position{})
	if out != nil && out.Descriptor != nil && !IsEmpty(out.Descriptor.(*desc.MessageDescriptor)) {
		objType = ast.NamedType(out.Name, &ast.Position{})
	}

	svcDir := &ast.DirectiveDefinition{
		Description: getDescription(svc),
		Name:        svc.GetName(),
		Locations:   []ast.DirectiveLocation{ast.LocationFieldDefinition},
		Position:    &ast.Position{Src: &ast.Source{}},
	}
	s.Schema.Directives[svcDir.Name] = svcDir

	m := &MethodDescriptor{
		ServiceDescriptor: svc,
		MethodDescriptor:  rpc,
		FieldDefinition: &ast.FieldDefinition{
			Description: getDescription(rpc),
			Name:        ToLowerFirst(svc.GetName()) + strings.Title(rpc.GetName()),
			Arguments:   args,
			Type:        objType,
			Directives: []*ast.Directive{{
				Name:       svcDir.Name,
				Position:   &ast.Position{},
				Definition: svcDir,
				Location:   svcDir.Locations[0],
			}},
			Position: &ast.Position{},
		},
		input:  in,
		output: out,
	}

	return m
}

func (s *SchemaDescriptor) createField(field *desc.FieldDescriptor, obj *ObjectDescriptor) (_ *FieldDescriptor, err error) {
	fieldAst := &ast.FieldDefinition{
		Description: getDescription(field),
		Name:        ToLowerFirst(CamelCase(field.GetName())),
		Type:        &ast.Type{Position: &ast.Position{}},
		Position:    &ast.Position{},
	}
	switch field.GetType() {
	case descriptor.FieldDescriptorProto_TYPE_DOUBLE,
		descriptor.FieldDescriptorProto_TYPE_FLOAT:
		fieldAst.Type.NamedType = ScalarFloat

	case descriptor.FieldDescriptorProto_TYPE_BYTES:
		scalar := s.createScalar("Bytes", "")
		fieldAst.Type.NamedType = scalar.Name

	case descriptor.FieldDescriptorProto_TYPE_INT64,
		descriptor.FieldDescriptorProto_TYPE_SINT64,
		descriptor.FieldDescriptorProto_TYPE_SFIXED64,
		descriptor.FieldDescriptorProto_TYPE_INT32,
		descriptor.FieldDescriptorProto_TYPE_SINT32,
		descriptor.FieldDescriptorProto_TYPE_SFIXED32,
		descriptor.FieldDescriptorProto_TYPE_UINT32,
		descriptor.FieldDescriptorProto_TYPE_FIXED32,
		descriptor.FieldDescriptorProto_TYPE_UINT64,
		descriptor.FieldDescriptorProto_TYPE_FIXED64:
		fieldAst.Type.NamedType = ScalarInt

	case descriptor.FieldDescriptorProto_TYPE_BOOL:
		fieldAst.Type.NamedType = ScalarBoolean

	case descriptor.FieldDescriptorProto_TYPE_STRING:
		fieldAst.Type.NamedType = ScalarString

	case descriptor.FieldDescriptorProto_TYPE_GROUP:
		return nil, fmt.Errorf("proto2 groups are not supported please use proto3 syntax")

	case descriptor.FieldDescriptorProto_TYPE_ENUM:
		fieldAst.Type.NamedType = obj.Name

	case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
		fieldAst.Type.NamedType = obj.Name

	default:
		panic("unknown proto field type")
	}

	if isRepeated(field) {
		fieldAst.Type = ast.ListType(fieldAst.Type, &ast.Position{})
		fieldAst.Type.Elem.NonNull = true
	}
	if isRequired(field) {
		fieldAst.Type.NonNull = true
	}

	return &FieldDescriptor{
		FieldDefinition: fieldAst,
		FieldDescriptor: field,
		typ:             obj,
	}, nil
}

func (s *SchemaDescriptor) createScalar(name string, description string) *ObjectDescriptor {
	obj := &ObjectDescriptor{
		Definition: &ast.Definition{
			Kind:        ast.Scalar,
			Description: description,
			Name:        name,
			Position:    &ast.Position{},
		},
	}
	s.objects = append(s.objects, obj)
	return obj
}

func (s *SchemaDescriptor) createUnion(oneof *desc.OneOfDescriptor, callstack Callstack) (*FieldDescriptor, error) {
	var types []string
	for _, choice := range oneof.GetChoices() {
		obj, err := s.createObjects(resolveFieldType(choice), false, callstack)
		if err != nil {
			return nil, err
		}
		f, err := s.createField(choice, obj)
		if err != nil {
			return nil, err
		}

		obj = &ObjectDescriptor{
			Definition: &ast.Definition{
				Kind:        ast.Object,
				Description: getDescription(f),
				Name:        CamelCaseSlice(strings.Split(strings.Trim(oneof.GetName()+"."+strings.Title(f.GetName()), "."), ".")),
				Fields:      ast.FieldList{f.FieldDefinition},
				Position:    &ast.Position{},
			},
			Descriptor: f,
			fields:     []*FieldDescriptor{f},
			fieldNames: map[string]*FieldDescriptor{},
		}
		s.objects = append(s.objects, obj)
		types = append(types, obj.Name)
	}
	obj := &ObjectDescriptor{
		Definition: &ast.Definition{
			Kind:        ast.Union,
			Description: getDescription(oneof),
			Name:        s.uniqueName(oneof, false),
			Types:       types,
			Position:    &ast.Position{},
		},
		Descriptor: oneof,
	}
	s.objects = append(s.objects, obj)

	return &FieldDescriptor{
		FieldDefinition: &ast.FieldDefinition{
			Description: getDescription(oneof),
			Name:        ToLowerFirst(CamelCase(oneof.GetName())),
			Type:        ast.NamedType(obj.Name, &ast.Position{}),
			Position:    &ast.Position{},
		},
		FieldDescriptor: nil,
		typ:             obj,
	}, nil
}

func isRepeated(field *desc.FieldDescriptor) bool {
	return field.GetLabel() == descriptor.FieldDescriptorProto_LABEL_REPEATED
}

func isRequired(field *desc.FieldDescriptor) bool {
	if v := graphqlFieldOptions(field); v != nil {
		return v.GetRequired()
	}
	return false
}

func graphqlFieldOptions(field *desc.FieldDescriptor) *gql.Field {
	if field.GetOptions() != nil {
		v, err := proto.GetExtension(field.GetOptions(), gql.E_Field)
		if err == nil && v.(*gql.Field) != nil {
			return v.(*gql.Field)
		}
	}
	return nil
}

const (
	ScalarInt     = "Int"
	ScalarFloat   = "Float"
	ScalarString  = "String"
	ScalarBoolean = "Boolean"
	ScalarID      = "ID"
)

var graphqlReservedNames = map[string]desc.Descriptor{
	"__Directive":  nil,
	"__Type":       nil,
	"__Field":      nil,
	"__EnumValue":  nil,
	"__InputValue": nil,
	"__Schema":     nil,
	"Int":          nil,
	"Float":        nil,
	"String":       nil,
	"Boolean":      nil,
	"ID":           nil,
}
