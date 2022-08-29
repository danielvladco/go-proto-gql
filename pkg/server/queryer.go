package server

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/ptypes"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/nautilus/graphql"
	"github.com/vektah/gqlparser/v2/ast"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/catalystsquad/go-proto-gql/pkg/generator"
)

type any = map[string]interface{}

func NewQueryer(pm generator.Registry, caller Caller) graphql.Queryer {
	return &queryer{pm: pm, c: caller}
}

type queryer struct {
	pm generator.Registry

	c Caller
}
type QueryerLogger struct {
	Next graphql.Queryer
}

func (q QueryerLogger) Query(ctx context.Context, input *graphql.QueryInput, i interface{}) (err error) {
	startTime := time.Now()
	err = q.Next.Query(ctx, input, i)
	log.Printf("[INFO] graphql call took: %fms", float64(time.Since(startTime))/float64(time.Millisecond))
	return err
}

func (q *queryer) Query(ctx context.Context, input *graphql.QueryInput, result interface{}) error {
	var res = map[string]interface{}{}
	var err error
	var selection ast.SelectionSet
	for _, op := range input.QueryDocument.Operations {
		selection, err = graphql.ApplyFragments(op.SelectionSet, input.QueryDocument.Fragments)
		if err != nil {
			return err
		}
		switch op.Operation {
		case ast.Query:
			err = q.resolveQuery(ctx, selection, res, input.Variables)

		case ast.Mutation:
			err = q.resolveMutation(ctx, selection, res, input.Variables)

		case ast.Subscription:
			return &graphql.Error{
				Extensions: map[string]interface{}{"code": "UNIMPLEMENTED"},
				Message:    "subscription is not supported",
			}
		}
	}
	if err != nil {
		return &graphql.Error{
			Extensions: map[string]interface{}{"code": "UNKNOWN_ERROR"},
			Message:    err.Error(),
		}
	}

	val := reflect.ValueOf(result)
	if val.Kind() != reflect.Ptr {
		return errors.New("result must be a pointer")
	}
	val = val.Elem()
	if !val.CanAddr() {
		return errors.New("result must be addressable (a pointer)")
	}
	val.Set(reflect.ValueOf(res))
	return nil
}

func (q *queryer) resolveMutation(ctx context.Context, selection ast.SelectionSet, res any, vars map[string]interface{}) (err error) {
	for _, ss := range selection {
		field, ok := ss.(*ast.Field)
		if !ok {
			continue
		}
		if field.Name == "__typename" {
			res[nameOrAlias(field)] = field.ObjectDefinition.Name
			continue
		}
		res[nameOrAlias(field)], err = q.resolveCall(ctx, ast.Mutation, field, vars)
		if err != nil {
			return err
		}
	}
	return
}

func (q *queryer) resolveQuery(ctx context.Context, selection ast.SelectionSet, res any, vars map[string]interface{}) (err error) {
	type mapEntry struct {
		key string
		val interface{}
	}
	errCh := make(chan error, 4)
	resCh := make(chan mapEntry, 4)
	for _, ss := range selection {
		field, ok := ss.(*ast.Field)
		if !ok {
			continue
		}
		go func(field *ast.Field) {
			if field.Name == "__typename" {
				resCh <- mapEntry{
					key: nameOrAlias(field),
					val: field.ObjectDefinition.Name,
				}
				return
			}
			resolvedValue, err := q.resolveCall(ctx, ast.Query, field, vars)
			if err != nil {
				errCh <- err
				return
			}
			resCh <- mapEntry{
				key: nameOrAlias(field),
				val: resolvedValue,
			}
		}(field)
	}
	var errs graphql.ErrorList
	for i := 0; i < len(selection); i++ {
		select {
		case r := <-resCh:
			res[r.key] = r.val
		case err := <-errCh:
			errs = append(errs, err)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return
}

func (q *queryer) resolveCall(ctx context.Context, op ast.Operation, field *ast.Field, vars map[string]interface{}) (interface{}, error) {
	method := q.pm.FindMethodByName(op, field.Name)
	if method == nil {
		return nil, errors.New("method not found")
	}

	inputMsg, err := q.pbEncode(method.GetInputType(), field, vars)
	if err != nil {
		return nil, err
	}

	msg, err := q.c.Call(ctx, method, inputMsg)
	if err != nil {
		return nil, err
	}

	return q.pbDecode(field, msg)
}

func (q *queryer) pbEncode(in *desc.MessageDescriptor, field *ast.Field, vars map[string]interface{}) (proto.Message, error) {
	inputMsg := dynamic.NewMessage(in)
	inArg := field.Arguments.ForName("in")
	if inArg == nil {
		return inputMsg, nil
	}

	var anyObj *desc.MessageDescriptor
	if generator.IsAny(in) {
		if len(inArg.Value.Children) == 0 {
			return nil, errors.New("no '__typename' provided")
		}
		typename := inArg.Value.Children.ForName("__typename")
		if typename == nil {
			return nil, errors.New("no '__typename' provided")
		}

		vv, err := typename.Value(vars)
		if err != nil {
			return nil, errors.New("no '__typename' provided")
		}
		vvv, ok := vv.(string)
		if !ok {
			return nil, errors.New("no '__typename' provided")
		}

		obj := q.pm.FindObjectByName(vvv)
		if obj == nil {
			return nil, errors.New("__typename should be a valid typename")
		}
		anyObj = obj
		inputMsg = dynamic.NewMessage(anyObj)
	}
	for _, arg := range inArg.Value.Children {
		val, err := arg.Value.Value(vars)
		if err != nil {
			return nil, err
		}
		if arg.Name == "__typename" {
			continue
		}

		var reqDesc *desc.FieldDescriptor
		if anyObj != nil {
			reqDesc = q.pm.FindFieldByName(anyObj, arg.Name)
		} else {
			reqDesc = q.pm.FindFieldByName(in, arg.Name)
		}

		if val, err = q.pbValue(val, reqDesc); err != nil {
			return nil, err
		}

		if reqDesc.IsRepeated() && reflect.TypeOf(val).Kind() != reflect.Slice {
			inputMsg.AddRepeatedField(reqDesc, val)
		} else {
			inputMsg.SetField(reqDesc, val)
		}
	}

	if anyObj != nil {
		//anyMsgDesc := q.pm.messages[in.DescriptorProto]
		//anyMsg := dynamic.NewMessage(q.pm.messages[in.DescriptorProto])
		//typeUrl := anyMsgDesc.FindFieldByName("type_url")
		//value := anyMsgDesc.FindFieldByName("value")
		//anyMsg.SetField(typeUrl, anyObj.GetFullyQualifiedName())
		return ptypes.MarshalAny(inputMsg)
	}
	return inputMsg, nil
}

func (q *queryer) pbValue(val interface{}, reqDesc *desc.FieldDescriptor) (_ interface{}, err error) {
	msgDesc := reqDesc.GetMessageType()

	switch v := val.(type) {
	case float64:
		if reqDesc.GetType() == descriptor.FieldDescriptorProto_TYPE_FLOAT {
			return float32(v), nil
		}
	case int64:
		switch reqDesc.GetType() {
		case descriptor.FieldDescriptorProto_TYPE_INT32,
			descriptor.FieldDescriptorProto_TYPE_SINT32,
			descriptor.FieldDescriptorProto_TYPE_SFIXED32:
			return int32(v), nil

		case descriptor.FieldDescriptorProto_TYPE_UINT32,
			descriptor.FieldDescriptorProto_TYPE_FIXED32:
			return uint32(v), nil

		case descriptor.FieldDescriptorProto_TYPE_UINT64,
			descriptor.FieldDescriptorProto_TYPE_FIXED64:
			return uint64(v), nil
		case descriptor.FieldDescriptorProto_TYPE_FLOAT:
			return float32(v), nil
		case descriptor.FieldDescriptorProto_TYPE_DOUBLE:
			return float64(v), nil
		}
	case string:
		switch reqDesc.GetType() {
		case descriptor.FieldDescriptorProto_TYPE_ENUM:
			// TODO predefine this
			enumDesc := reqDesc.GetEnumType()
			values := map[string]int32{}
			for _, v := range enumDesc.GetValues() {
				values[v.GetName()] = v.GetNumber()
			}
			return values[v], nil
		case descriptor.FieldDescriptorProto_TYPE_BYTES:
			bytes, err := base64.StdEncoding.DecodeString(v)
			if err != nil {
				return nil, fmt.Errorf("bytes should be a base64 encoded string")
			}
			return bytes, nil
		}
	case []interface{}:
		v2 := make([]interface{}, len(v))
		for i, vv := range v {
			v2[i], err = q.pbValue(vv, reqDesc)
			if err != nil {
				return nil, err
			}
		}
		return v2, nil
	case map[string]interface{}:
		var anyTypeDescriptor *desc.MessageDescriptor
		var vvv string
		var ok bool
		for kk, vv := range v {
			if kk == "__typename" {
				vvv, ok = vv.(string)
				if !ok {
					return nil, errors.New("'__typename' must be a string")
				}
				delete(v, "__typename")
				break
			}
		}
		var msg *dynamic.Message
		protoDesc := msgDesc
		if generator.IsAny(protoDesc) {
			anyTypeDescriptor = q.pm.FindObjectByName(vvv)
			if anyTypeDescriptor == nil {
				return nil, errors.New("'__typename' must be a valid INPUT_OBJECT")
			}
			msg = dynamic.NewMessage(anyTypeDescriptor)
		} else {
			msg = dynamic.NewMessage(protoDesc)
		}
		oneofValidate := map[*desc.OneOfDescriptor]struct{}{}
		for kk, vv := range v {

			if anyTypeDescriptor != nil {
				msgDesc = anyTypeDescriptor
			}
			//plugType := q.pm.inputs[msgDesc]
			//fieldDesc := q.pm.fields[q.p.FieldBack(plugType.DescriptorProto, kk)]
			fieldDesc := q.pm.FindFieldByName(msgDesc, kk)
			oneof := fieldDesc.GetOneOf()
			if oneof != nil {
				_, ok := oneofValidate[oneof]
				if ok {
					return nil, fmt.Errorf("field with name %q on Object %q can't be set", kk, msgDesc.GetName())
				}
				oneofValidate[oneof] = struct{}{}
			}

			vv2, err := q.pbValue(vv, fieldDesc)
			if err != nil {
				return nil, err
			}
			msg.SetField(fieldDesc, vv2)
		}
		if anyTypeDescriptor != nil {
			return ptypes.MarshalAny(msg)
		}
		return msg, nil
	}

	return val, nil
}

func (q *queryer) pbDecodeOneofField(desc *desc.MessageDescriptor, dynamicMsg *dynamic.Message, selection ast.SelectionSet) (oneof any, err error) {
	oneof = any{}
	for _, f := range selection {
		out, ok := f.(*ast.Field)
		if !ok {
			continue
		}
		if out.Name == "__typename" {
			oneof[nameOrAlias(out)] = out.ObjectDefinition.Name
			continue
		}

		fieldDesc := q.pm.FindUnionFieldByMessageFQNAndName(desc.GetFullyQualifiedName(), out.Name)
		protoVal := dynamicMsg.GetField(fieldDesc)
		oneof[nameOrAlias(out)], err = q.gqlValue(protoVal, fieldDesc.GetMessageType(), fieldDesc.GetEnumType(), out)
		if err != nil {
			return nil, err
		}
	}
	return oneof, nil
}

func (q *queryer) pbDecode(field *ast.Field, msg proto.Message) (res interface{}, err error) {
	switch dynamicMsg := msg.(type) {
	case *dynamic.Message:
		return q.gqlValue(dynamicMsg, dynamicMsg.GetMessageDescriptor(), nil, field)
	case *anypb.Any:
		return q.gqlValue(dynamicMsg, nil, nil, field)
	default:
		return nil, fmt.Errorf("expected proto message of type *dynamic.Message or *anypb.Any but received: %T", msg)
	}
}

// FIXME take care of recursive calls
func (q *queryer) gqlValue(val interface{}, msgDesc *desc.MessageDescriptor, enumDesc *desc.EnumDescriptor, field *ast.Field) (_ interface{}, err error) {
	switch v := val.(type) {
	case int32:
		// int32 enum
		if enumDesc != nil {
			values := map[int32]string{}
			for _, v := range enumDesc.GetValues() {
				values[v.GetNumber()] = v.GetName()
			}

			return values[v], nil
		}

	case map[interface{}]interface{}:
		res := make([]interface{}, len(v))
		i := 0
		for kk, vv := range v {
			vals := any{}
			for _, f := range field.SelectionSet {
				out, ok := f.(*ast.Field)
				if !ok {
					continue
				}
				switch out.Name {
				case "value":
					valueField := msgDesc.FindFieldByName("value")
					if vals[nameOrAlias(out)], err = q.gqlValue(vv, valueField.GetMessageType(), valueField.GetEnumType(), out); err != nil {
						return nil, err
					}
				case "key":
					vals[nameOrAlias(out)] = kk
				case "__typename":
					vals[nameOrAlias(out)] = out.ObjectDefinition.Name
				}
			}

			res[i] = vals
			i++
		}
		return res, nil

	case []interface{}:
		v2 := make([]interface{}, len(v))
		for i, vv := range v {
			v2[i], err = q.gqlValue(vv, msgDesc, enumDesc, field)
			if err != nil {
				return nil, err
			}
		}
		return v2, nil

	case *dynamic.Message:
		if v == nil {
			return nil, nil
		}
		fields := v.GetKnownFields()
		vals := make(map[string]interface{}, len(fields))
		//gqlFields := map[string]string{}
		for _, s := range field.SelectionSet {
			out, ok := s.(*ast.Field)
			if !ok {
				continue
			}

			if out.Name == "__typename" {
				vals[nameOrAlias(out)] = out.ObjectDefinition.Name
				continue
			}

			descMsg := v.GetMessageDescriptor()
			fieldDesc := q.pm.FindFieldByName(descMsg, out.Name)
			if fieldDesc == nil {
				vals[nameOrAlias(out)], err = q.pbDecodeOneofField(descMsg, v, out.SelectionSet)
				if err != nil {
					return nil, err
				}
				continue
			}

			vals[nameOrAlias(out)], err = q.gqlValue(v.GetField(fieldDesc), fieldDesc.GetMessageType(), fieldDesc.GetEnumType(), out)
			if err != nil {
				return nil, err
			}
		}
		return vals, nil
	case *anypb.Any:
		vals, err := q.anyMessageToMap(v)
		if err != nil {
			return nil, err
		}
		return vals, nil

	case []byte:
		return base64.StdEncoding.EncodeToString(v), nil
	}

	return val, nil
}

func (q *queryer) anyMessageToMap(v *anypb.Any) (map[string]interface{}, error) {
	fqn, err := ptypes.AnyMessageName(v)
	if err != nil {
		return nil, err
	}
	grpcType, definition := q.pm.FindObjectByFullyQualifiedName(fqn)
	outputMsg := dynamic.NewMessage(grpcType)
	if err = outputMsg.Unmarshal(v.Value); err != nil {
		return nil, err
	}
	return q.protoMessageToMap(outputMsg, definition)
}

func (q *queryer) protoMessageToMap(outputMsg *dynamic.Message, definition *ast.Definition) (map[string]interface{}, error) {
	fields := outputMsg.GetKnownFields()
	vals := make(map[string]interface{}, len(fields))
	vals["__typename"] = definition.Name
	for _, field := range fields {
		fieldDef := q.pm.FindGraphqlFieldByProtoField(definition, field.GetName())
		// the field is probably invalid or ignored
		if fieldDef == nil {
			continue
			//return nil, fmt.Errorf("proto field %q doesn't have a graphql counterpart on type %q", field.GetName(), definition.Name)
		}
		val := outputMsg.GetField(field)
		switch vv := val.(type) {
		case int32:
			if field.GetEnumType() != nil {
				values := map[int32]string{}
				for _, v := range field.GetEnumType().GetValues() {
					values[v.GetNumber()] = v.GetName()
				}

				vals[fieldDef.Name] = values[vv]
			}

		case *dynamic.Message:
			_, definition := q.pm.FindObjectByFullyQualifiedName(vv.GetMessageDescriptor().GetFullyQualifiedName())
			val, err := q.protoMessageToMap(vv, definition)
			if err != nil {
				return nil, err
			}
			vals[fieldDef.Name] = val
		case *anypb.Any:
			val, err := q.anyMessageToMap(vv)
			if err != nil {
				return nil, err
			}
			vals[fieldDef.Name] = val
		case []interface{}:
			var arrayVals []interface{}
			for _, val := range vv {
				switch vv := val.(type) {
				case int32:
					if field.GetEnumType() != nil {
						values := map[int32]string{}
						for _, v := range field.GetEnumType().GetValues() {
							values[v.GetNumber()] = v.GetName()
						}

						arrayVals = append(arrayVals, values[vv])
					}

				case *dynamic.Message:
					_, definition := q.pm.FindObjectByFullyQualifiedName(vv.GetMessageDescriptor().GetFullyQualifiedName())
					val, err := q.protoMessageToMap(vv, definition)
					if err != nil {
						return nil, err
					}
					arrayVals = append(arrayVals, val)
				case *anypb.Any:
					val, err := q.anyMessageToMap(vv)
					if err != nil {
						return nil, err
					}
					arrayVals = append(arrayVals, val)
				default:
					arrayVals = append(arrayVals, vv)
				}
			}
		default:
			vals[fieldDef.Name] = vv
		}
	}
	return vals, nil
}

func nameOrAlias(field *ast.Field) string {
	if field.Alias != "" {
		return field.Alias
	}

	return field.Name
}
