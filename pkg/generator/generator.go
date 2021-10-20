package generator

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jhump/protoreflect/desc"
	"github.com/vektah/gqlparser/v2/ast"
	"google.golang.org/protobuf/compiler/protogen"
	descriptor "google.golang.org/protobuf/types/descriptorpb"

	gqlpb "github.com/danielvladco/go-proto-gql/pb"
)

const (
	fieldPrefix        = "Field"
	inputSuffix        = "Input"
	typeSep            = "_"
	packageSep         = "."
	anyTypeDescription = "Any is any json type"
	scalarBytes        = "Bytes"
	goFieldDirective   = "goField"

	DefaultExtension = "graphql"
)

func NewSchemas(files []*desc.FileDescriptor, mergeSchemas, genServiceDesc bool, goref GoRef) (schemas SchemaDescriptorList, _ error) {
	if mergeSchemas {
		schema := NewSchemaDescriptor(genServiceDesc, goref)
		for _, file := range files {
			err := generateFile(file, schema)
			if err != nil {
				return nil, err
			}
		}

		return []*SchemaDescriptor{schema}, nil
	}

	for _, file := range files {
		schema := NewSchemaDescriptor(genServiceDesc, goref)
		err := generateFile(file, schema)
		if err != nil {
			return nil, err
		}

		schemas = append(schemas, schema)
	}

	return
}

func generateFile(file *desc.FileDescriptor, schema *SchemaDescriptor) error {
	schema.FileDescriptors = append(schema.FileDescriptors, file)

	for _, svc := range file.GetServices() {
		svcOpts := GraphqlServiceOptions(svc.AsServiceDescriptorProto().GetOptions())
		if svcOpts != nil && svcOpts.Ignore != nil && *svcOpts.Ignore {
			continue
		}
		for _, rpc := range svc.GetMethods() {
			rpcOpts := GraphqlMethodOptions(rpc.AsMethodDescriptorProto().GetOptions())
			if rpcOpts != nil && rpcOpts.Ignore != nil && *rpcOpts.Ignore {
				continue
			}
			in, err := schema.CreateObjects(rpc.GetInputType(), true)
			if err != nil {
				return err
			}

			out, err := schema.CreateObjects(rpc.GetOutputType(), false)
			if err != nil {
				return err
			}

			if rpc.IsServerStreaming() && rpc.IsClientStreaming() {
				schema.GetMutation().addMethod(svc, rpc, in, out)
			}

			if rpc.IsServerStreaming() {
				schema.GetSubscription().addMethod(svc, rpc, in, out)
			} else {
				switch GetRequestType(rpcOpts, svcOpts) {
				case gqlpb.Type_QUERY:
					schema.GetQuery().addMethod(svc, rpc, in, out)
				default:
					schema.GetMutation().addMethod(svc, rpc, in, out)
				}
			}
		}
	}

	return nil
}

type SchemaDescriptorList []*SchemaDescriptor

func (s SchemaDescriptorList) AsGraphql() (astSchema []*ast.Schema) {
	for _, ss := range s {
		astSchema = append(astSchema, ss.AsGraphql())
	}
	return
}

func (s SchemaDescriptorList) GetForDescriptor(file *protogen.File) *SchemaDescriptor {
	for _, schema := range s {
		for _, d := range schema.FileDescriptors {
			if d.AsFileDescriptorProto() == file.Proto {
				return schema
			}
		}
	}
	return nil
}

func NewSchemaDescriptor(genServiceDesc bool, goref GoRef) *SchemaDescriptor {
	return &SchemaDescriptor{
		Schema: &ast.Schema{
			Directives: map[string]*ast.DirectiveDefinition{},
			Types:      map[string]*ast.Definition{},
		},
		reservedNames:              graphqlReservedNames,
		createdObjects:             map[createdObjectKey]*ObjectDescriptor{},
		generateServiceDescriptors: genServiceDesc,
		goRef:                      goref,
	}
}

type SchemaDescriptor struct {
	*ast.Schema

	FileDescriptors []*desc.FileDescriptor

	files []*desc.FileDescriptor

	query        *RootDefinition
	mutation     *RootDefinition
	subscription *RootDefinition

	objects []*ObjectDescriptor

	reservedNames  map[string]desc.Descriptor
	createdObjects map[createdObjectKey]*ObjectDescriptor

	generateServiceDescriptors bool

	goRef GoRef
}

type createdObjectKey struct {
	desc  desc.Descriptor
	input bool
}

func (s *SchemaDescriptor) AsGraphql() *ast.Schema {
	queryDef := s.GetQuery().Definition
	mutationDef := s.GetMutation().Definition
	subscriptionsDef := s.GetSubscription().Definition
	s.Schema.Query = queryDef
	s.Schema.Types["Query"] = queryDef
	if s.query.methods == nil {
		s.Schema.Query.Fields = append(s.Schema.Query.Fields, &ast.FieldDefinition{
			Name: "dummy",
			Type: ast.NamedType("Boolean", &ast.Position{}),
		})
	}
	if s.mutation.methods != nil {
		s.Schema.Mutation = mutationDef
		s.Schema.Types["Mutation"] = mutationDef
	}
	if s.subscription.methods != nil {
		s.Schema.Subscription = subscriptionsDef
		s.Schema.Types["Subscription"] = subscriptionsDef
	}

	for _, o := range s.objects {
		def := o.AsGraphql()
		s.Schema.Types[def.Name] = def
	}
	return s.Schema
}

func (s *SchemaDescriptor) Objects() []*ObjectDescriptor {
	return s.objects
}

func (s *SchemaDescriptor) GetMutation() *RootDefinition {
	if s.mutation == nil {
		s.mutation = NewRootDefinition(Mutation, s)
	}
	return s.mutation
}

func (s *SchemaDescriptor) GetSubscription() *RootDefinition {
	if s.subscription == nil {
		s.subscription = NewRootDefinition(Subscription, s)
	}
	return s.subscription
}

func (s *SchemaDescriptor) GetQuery() *RootDefinition {
	if s.query == nil {
		s.query = NewRootDefinition(Query, s)
	}

	return s.query
}

// make name be unique
// just create a map and register every name
func (s *SchemaDescriptor) uniqueName(d desc.Descriptor, input bool) (name string) {
	var collisionPrefix string
	var suffix string
	if _, ok := d.(*desc.MessageDescriptor); input && ok {
		suffix = inputSuffix
	}
	name = strings.Title(CamelCaseSlice(strings.Split(strings.TrimPrefix(d.GetFullyQualifiedName(), d.GetFile().GetPackage()+packageSep), packageSep)) + suffix)

	if _, ok := d.(*desc.FieldDescriptor); ok {
		collisionPrefix = fieldPrefix
		name = CamelCaseSlice(strings.Split(strings.Trim(d.GetParent().GetName()+packageSep+strings.Title(d.GetName()), packageSep), packageSep))
	} else {
		collisionPrefix = CamelCaseSlice(strings.Split(d.GetFile().GetPackage(), packageSep))
	}

	originalName := name
	for uniqueSuffix := 0; ; uniqueSuffix++ {
		d2, ok := s.reservedNames[name]
		if !ok {
			break
		}
		if d2 == d {
			return name
		}
		if uniqueSuffix == 0 {
			name = collisionPrefix + typeSep + originalName
			continue
		}
		name = collisionPrefix + typeSep + originalName + strconv.Itoa(uniqueSuffix)
	}

	s.reservedNames[name] = d
	return
}

func (s *SchemaDescriptor) CreateObjects(d desc.Descriptor, input bool) (obj *ObjectDescriptor, err error) {
	// the case if trying to resolve a primitive as a object. In this case we just return nil
	if d == nil {
		return
	}
	if obj, ok := s.createdObjects[createdObjectKey{d, input}]; ok {
		return obj, nil
	}

	obj = &ObjectDescriptor{
		Definition: &ast.Definition{
			Description: getDescription(d),
			Name:        s.uniqueName(d, input),
			Position:    &ast.Position{},
		},
		Descriptor: d,
	}

	s.createdObjects[createdObjectKey{d, input}] = obj

	switch dd := d.(type) {
	case *desc.MessageDescriptor:
		if IsEmpty(dd) {
			return obj, nil
		}
		if IsAny(dd) {
			//TODO find a better way to handle any types
			delete(s.createdObjects, createdObjectKey{d, input})
			any := s.createScalar(s.uniqueName(dd, false), anyTypeDescription)
			return any, nil
		}

		kind := ast.Object
		if input {
			kind = ast.InputObject
		}
		fields := FieldDescriptorList{}
		outputOneofRegistrar := map[*desc.OneOfDescriptor]struct{}{}

		for _, df := range dd.GetFields() {
			fieldOpts := GraphqlFieldOptions(df.AsFieldDescriptorProto().GetOptions())
			if fieldOpts != nil && fieldOpts.Ignore != nil && *fieldOpts.Ignore {
				continue
			}
			var fieldDirective []*ast.Directive
			if df.GetType() == descriptor.FieldDescriptorProto_TYPE_MESSAGE && IsEmpty(df.GetMessageType()) {
				continue
			}

			if oneof := df.GetOneOf(); oneof != nil {
				opts := GraphqlOneofOptions(oneof.AsOneofDescriptorProto().GetOptions())
				if opts.GetIgnore() {
					continue
				}
				if !input {
					if _, ok := outputOneofRegistrar[oneof]; ok {
						continue
					}
					outputOneofRegistrar[oneof] = struct{}{}
					field, err := s.createUnion(oneof)
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

			fieldObj, err := s.CreateObjects(resolveFieldType(df), input)
			if err != nil {
				return nil, err
			}
			if fieldObj == nil && df.GetMessageType() != nil {
				continue
			}
			f, err := s.createField(df, fieldObj)
			if err != nil {
				return nil, err
			}
			f.Directives = append(f.Directives, fieldDirective...)
			fields = append(fields, f)
		}

		obj.Definition.Fields = fields.AsGraphql()
		obj.Definition.Kind = kind
		obj.fields = fields
	case *desc.EnumDescriptor:
		obj.Definition.Kind = ast.Enum
		obj.Definition.EnumValues = enumValues(dd.GetValues())
	default:
		panic(fmt.Sprintf("received unexpected value %v of type %T", dd, dd))
	}

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

	Parent *SchemaDescriptor

	methods       []*MethodDescriptor
	reservedNames map[string]ServiceAndMethod
}

type ServiceAndMethod struct {
	svc *descriptor.ServiceDescriptorProto
	rpc *descriptor.MethodDescriptorProto
}

func (r *RootDefinition) UniqueName(svc *descriptor.ServiceDescriptorProto, rpc *descriptor.MethodDescriptorProto) (name string) {
	rpcOpts := GraphqlMethodOptions(rpc.GetOptions())
	svcOpts := GraphqlServiceOptions(svc.GetOptions())
	if rpcOpts != nil && rpcOpts.Name != nil {
		name = *rpcOpts.Name
	} else if svcOpts != nil && svcOpts.Name != nil {
		if *svcOpts.Name == "" {
			name = ToLowerFirst(rpc.GetName())
		} else {
			name = *svcOpts.Name + strings.Title(rpc.GetName())
		}
	} else {
		name = ToLowerFirst(svc.GetName()) + strings.Title(rpc.GetName())
	}

	originalName := name
	for uniqueSuffix := 0; ; uniqueSuffix++ {
		snm, ok := r.reservedNames[name]
		if !ok {
			break
		}
		if svc == snm.svc && snm.rpc == rpc {
			return name
		}
		name = originalName + strconv.Itoa(uniqueSuffix)
	}

	r.reservedNames[name] = ServiceAndMethod{svc, rpc}
	return
}

func (r *RootDefinition) Methods() []*MethodDescriptor {
	return r.methods
}

func (r *RootDefinition) addMethod(svc *desc.ServiceDescriptor, rpc *desc.MethodDescriptor, in, out *ObjectDescriptor) {
	var args ast.ArgumentDefinitionList

	if in != nil && (in.Descriptor != nil && !IsEmpty(in.Descriptor.(*desc.MessageDescriptor)) || in.Definition.Kind == ast.Scalar) {
		args = append(args, &ast.ArgumentDefinition{
			Name:     "in",
			Type:     ast.NamedType(in.Name, &ast.Position{}),
			Position: &ast.Position{},
		})
	}
	objType := ast.NamedType("Boolean", &ast.Position{})
	if out != nil && (out.Descriptor != nil && !IsEmpty(out.Descriptor.(*desc.MessageDescriptor)) || in.Definition.Kind == ast.Scalar) {
		objType = ast.NamedType(out.Name, &ast.Position{})
	}

	svcDir := &ast.DirectiveDefinition{
		Description: getDescription(svc),
		Name:        svc.GetName(),
		Locations:   []ast.DirectiveLocation{ast.LocationFieldDefinition},
		Position:    &ast.Position{Src: &ast.Source{}},
	}
	r.Parent.Schema.Directives[svcDir.Name] = svcDir

	m := &MethodDescriptor{
		ServiceDescriptor: svc,
		MethodDescriptor:  rpc,
		FieldDefinition: &ast.FieldDefinition{
			Description: getDescription(rpc),
			Name:        r.UniqueName(svc.AsServiceDescriptorProto(), rpc.AsMethodDescriptorProto()),
			Arguments:   args,
			Type:        objType,
			Position:    &ast.Position{},
		},
		input:  in,
		output: out,
	}
	if r.Parent.generateServiceDescriptors {
		m.Directives = []*ast.Directive{{
			Name:       svcDir.Name,
			Position:   &ast.Position{},
			Definition: svcDir,
			Location:   svcDir.Locations[0],
		}}
	}

	r.methods = append(r.methods, m)
	// TODO maybe not do it here?
	r.Definition.Fields = append(r.Definition.Fields, m.FieldDefinition)
}

type rootName string

const (
	Mutation     rootName = "Mutation"
	Query        rootName = "Query"
	Subscription rootName = "Subscription"
)

func NewRootDefinition(name rootName, parent *SchemaDescriptor) *RootDefinition {
	return &RootDefinition{Definition: &ast.Definition{
		Kind:     ast.Object,
		Name:     string(name),
		Position: &ast.Position{},
	}, Parent: parent, reservedNames: map[string]ServiceAndMethod{}}
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

func (s *SchemaDescriptor) createField(field *desc.FieldDescriptor, obj *ObjectDescriptor) (_ *FieldDescriptor, err error) {
	fieldAst := &ast.FieldDefinition{
		Description: getDescription(field),
		Name:        ToLowerFirst(CamelCase(field.GetName())),
		Type:        &ast.Type{Position: &ast.Position{}},
		Position:    &ast.Position{},
	}
	fieldOpts := GraphqlFieldOptions(field.AsFieldDescriptorProto().GetOptions())
	if fieldOpts != nil && fieldOpts.Name != nil {
		fieldAst.Name = *fieldOpts.Name
		directive := &ast.DirectiveDefinition{
			Name: goFieldDirective,
			Arguments: []*ast.ArgumentDefinition{{
				Name:     "forceResolver",
				Type:     ast.NamedType("Boolean", &ast.Position{}),
				Position: &ast.Position{},
			}, {
				Name:     "name",
				Type:     ast.NamedType("String", &ast.Position{}),
				Position: &ast.Position{},
			}},
			Locations: []ast.DirectiveLocation{ast.LocationInputFieldDefinition, ast.LocationFieldDefinition},
			Position:  &ast.Position{Src: &ast.Source{}},
		}
		s.Schema.Directives[directive.Name] = directive
		if s.goRef != nil {
			fieldAst.Directives = []*ast.Directive{{
				Name: directive.Name,
				Arguments: []*ast.Argument{{
					Name: "name",
					Value: &ast.Value{
						Raw:      s.goRef.FindGoField(field.GetFullyQualifiedName()).GoName,
						Kind:     ast.StringValue,
						Position: &ast.Position{},
					},
					Position: &ast.Position{},
				}},
				Position: &ast.Position{},
				//ParentDefinition: nil, TODO
				Definition: directive,
			}}
		}
	}
	switch field.GetType() {
	case descriptor.FieldDescriptorProto_TYPE_DOUBLE,
		descriptor.FieldDescriptorProto_TYPE_FLOAT:
		fieldAst.Type.NamedType = ScalarFloat

	case descriptor.FieldDescriptorProto_TYPE_BYTES:
		scalar := s.createScalar(scalarBytes, "")
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

func (s *SchemaDescriptor) createUnion(oneof *desc.OneOfDescriptor) (*FieldDescriptor, error) {
	var types []string
	for _, choice := range oneof.GetChoices() {
		obj, err := s.CreateObjects(resolveFieldType(choice), false)
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
				Name:        s.uniqueName(choice, false),
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
	name := ToLowerFirst(CamelCase(oneof.GetName()))
	opts := GraphqlOneofOptions(oneof.AsOneofDescriptorProto().GetOptions())
	if opts.GetName() != "" {
		name = opts.GetName()
	}
	return &FieldDescriptor{
		FieldDefinition: &ast.FieldDefinition{
			Description: getDescription(oneof),
			Name:        name,
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
	if v := GraphqlFieldOptions(field.AsFieldDescriptorProto().GetOptions()); v != nil {
		return v.GetRequired()
	}
	return false
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
