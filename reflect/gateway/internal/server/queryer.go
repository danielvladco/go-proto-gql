package server

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"reflect"
	"time"

	graphql99 "github.com/99designs/gqlgen/graphql"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/ptypes"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/nautilus/graphql"
	"github.com/vektah/gqlparser/v2/ast"
	"google.golang.org/protobuf/types/known/anypb"
)

type any = map[string]interface{}

func NewQueryer(pm *GQLProtoMapping, p GQLDescriptors, caller Caller) graphql.Queryer {
	return &queryer{pm: pm, p: p, c: caller}
}

type queryer struct {
	pm *GQLProtoMapping
	p  GQLDescriptors

	c Caller
}
type QueryerLogger struct {
	Next graphql.Queryer
}

func (q QueryerLogger) Query(ctx context.Context, input *graphql.QueryInput, i interface{}) (err error) {
	startTime := time.Now()
	err = q.Next.Query(ctx, input, i)
	log.Printf("[INFO] gql to grpc call took: %fms", float64(time.Since(startTime))/float64(time.Millisecond))
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
			// TODO add parallel execution
			err = q.resolveSelection(ctx, selection, res, input.Variables)

		case ast.Mutation:
			err = q.resolveSelection(ctx, selection, res, input.Variables)

		case ast.Subscription:
			panic("subscription not implemented")
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
	return err
}

func (q *queryer) resolveSelection(ctx context.Context, selection ast.SelectionSet, res any, vars map[string]interface{}) (err error) {
	for _, ss := range selection {
		field, ok := ss.(*ast.Field)
		if !ok {
			continue
		}
		if field.Name == "__typename" {
			res[nameOrAlias(field)] = field.ObjectDefinition.Name
			continue
		}
		res[nameOrAlias(field)], err = q.resolveCall(ctx, field, vars)
		if err != nil {
			return err
		}
	}
	return
}

func (q *queryer) resolveCall(ctx context.Context, field *ast.Field, vars map[string]interface{}) (interface{}, error) {
	method, err := q.pm.GetMethod(field.Name)
	if err != nil {
		return nil, err
	}

	inputMsg, err := q.pbEncode(field, vars)
	if err != nil {
		return nil, err
	}

	msg, err := q.c.Call(ctx, method.ServiceDescriptorProto, method.MethodDescriptorProto, inputMsg)
	if err != nil {
		return nil, err
	}

	return q.pbDecode(field, msg)
}

func (q *queryer) pbEncode(field *ast.Field, vars map[string]interface{}) (proto.Message, error) {
	in, err := q.pm.InputType(field.Name)
	if err != nil {
		return nil, err
	}
	if in == nil {
		return nil, errors.New("unexpected call without a request message")
	}

	inputMsg := dynamic.NewMessage(q.pm.messages[in.DescriptorProto])
	inArg := field.Arguments.ForName("in")
	if inArg == nil {
		return inputMsg, nil
	}

	var anyTypeDescriptor *desc.MessageDescriptor
	if q.p.IsAny(in.FQN) {
		_, _ = graphql99.UnmarshalAny(nil)
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

		anyTypeDescriptor, ok = q.pm.gqltypeToDesc[vvv]
		if !ok {
			return nil, errors.New("__typename should be a valid typename")
		}
		inputMsg = dynamic.NewMessage(anyTypeDescriptor)
	}
	oneofValidate := map[*desc.OneOfDescriptor]struct{}{}
	for _, arg := range inArg.Value.Children {
		val, err := arg.Value.Value(vars)
		if err != nil {
			return nil, err
		}
		if arg.Name == "__typename" {
			continue
		}

		var reqDesc *desc.FieldDescriptor
		if anyTypeDescriptor != nil {
			reqDesc = q.pm.fields[q.p.FieldBack(anyTypeDescriptor.AsDescriptorProto(), arg.Name)]
		} else {
			reqDesc = q.pm.fields[q.p.FieldBack(in.DescriptorProto, arg.Name)]
		}

		oneof := reqDesc.GetOneOf()
		if oneof != nil {
			_, ok := oneofValidate[oneof]
			if ok {
				return nil, fmt.Errorf("field with name %q on Object %q can't be set", arg.Name, q.p.GqlModelNames()[in])
			}
			oneofValidate[oneof] = struct{}{}
		}

		if val, err = q.pbValue(val, reqDesc); err != nil {
			return nil, err
		}

		inputMsg.SetField(reqDesc, val)
	}

	if anyTypeDescriptor != nil {
		//anyMsgDesc := q.pm.messages[in.DescriptorProto]
		//anyMsg := dynamic.NewMessage(q.pm.messages[in.DescriptorProto])
		//typeUrl := anyMsgDesc.FindFieldByName("type_url")
		//value := anyMsgDesc.FindFieldByName("value")
		//anyMsg.SetField(typeUrl, anyTypeDescriptor.GetFullyQualifiedName())
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
		if q.p.IsAny(msgDesc.GetFullyQualifiedName()) {
			anyTypeDescriptor, ok = q.pm.gqltypeToDesc[vvv]
			if !ok {
				return nil, errors.New("'__typename' must be a valid INPUT_OBJECT")
			}
			msg = dynamic.NewMessage(anyTypeDescriptor)
		} else {
			msg = dynamic.NewMessage(msgDesc)
		}
		oneofValidate := map[*desc.OneOfDescriptor]struct{}{}
		for kk, vv := range v {

			if anyTypeDescriptor != nil {
				msgDesc = anyTypeDescriptor
			}
			plugType := q.pm.inputs[msgDesc]
			fieldDesc := q.pm.fields[q.p.FieldBack(plugType.DescriptorProto, kk)]
			oneof := fieldDesc.GetOneOf()
			if oneof != nil {
				_, ok := oneofValidate[oneof]
				if ok {
					return nil, fmt.Errorf("field with name %q on Object %q can't be set", kk, q.p.GqlModelNames()[plugType])
				}
				oneofValidate[oneof] = struct{}{}
			}

			vv2, err := q.pbValue(vv, fieldDesc)
			if err != nil {
				return nil, err
			}
			msg.SetField(q.pm.fields[q.p.FieldBack(plugType.DescriptorProto, kk)], vv2)
		}
		if anyTypeDescriptor != nil {
			return ptypes.MarshalAny(msg)
		}
		return msg, nil
	}

	return val, nil
}

func (q *queryer) pbDecodeOneofField(desc *descriptor.DescriptorProto, dynamicMsg *dynamic.Message, selection ast.SelectionSet) (oneof any, err error) {
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
		fieldBack := q.p.FieldBack(desc, out.Name)
		fieldDesc := q.pm.fields[fieldBack]
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
			descMsg := v.GetMessageDescriptor().AsDescriptorProto()
			protoFieldDesc := q.p.FieldBack(v.GetMessageDescriptor().AsDescriptorProto(), out.Name)
			if protoFieldDesc == nil {
				vals[nameOrAlias(out)], err = q.pbDecodeOneofField(descMsg, v, out.SelectionSet)
				if err != nil {
					return nil, err
				}
				break
			}
			fieldDesc := q.pm.fields[protoFieldDesc]

			vals[nameOrAlias(out)], err = q.gqlValue(v.GetField(fieldDesc), fieldDesc.GetMessageType(), fieldDesc.GetEnumType(), out)
			if err != nil {
				return nil, err
			}
		}
		return vals, nil
	case *anypb.Any:
		fqn, err := ptypes.AnyMessageName(v)
		if err != nil {
			return nil, err
		}
		tp := q.pm.pbtypeToDesc[fqn]
		outputMsg := dynamic.NewMessage(q.pm.messages[tp.DescriptorProto])
		err = outputMsg.Unmarshal(v.Value)
		if err != nil {
			return nil, err
		}
		val, err := q.gqlValue(outputMsg, outputMsg.GetMessageDescriptor(), nil, field)
		if err != nil {
			return nil, err
		}

		vals := map[string]interface{}{}
		for _, f := range field.SelectionSet {
			out, ok := f.(*ast.Field)
			if !ok {
				continue
			}
			if out.Name == "__typename" {
				vals[nameOrAlias(out)] = q.p.GqlModelNames()[tp]
			}
		}

		vv, ok := val.(map[string]interface{})
		if !ok {
			return nil, errors.New("any type can be only of message type")
		}
		for k, v := range vv {
			vals[k] = v
		}
		return vals, nil

	case []byte:
		return base64.StdEncoding.EncodeToString(v), nil
	}

	return val, nil
}

func nameOrAlias(field *ast.Field) string {
	if field.Alias != "" {
		return field.Alias
	}

	return field.Name
}
