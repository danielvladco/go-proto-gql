package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/danielvladco/go-proto-gql/plugin"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/dynamic/grpcdynamic"
	"github.com/nautilus/graphql"
	"github.com/vektah/gqlparser/v2/ast"
	"google.golang.org/grpc"
	"reflect"
)

type json = map[string]interface{}

func NewQueryer(
	queries map[string]*plugin.Method,
	origMessages map[string]*desc.MessageDescriptor,
	origServices map[SvcKey]SvcVal,
	p *plugin.Plugin) graphql.Queryer {
	return &queryer{
		queries:      queries,
		origMessages: origMessages,
		origServices: origServices,
		p:            p,
	}
}

type SvcKey [2]string

type SvcVal struct {
	*grpc.ClientConn
	*desc.MethodDescriptor
}

type queryer struct {
	queries      map[string]*plugin.Method
	origMessages map[string]*desc.MessageDescriptor
	origServices map[SvcKey]SvcVal
	p            *plugin.Plugin
}

func (q *queryer) Query(ctx context.Context, input *graphql.QueryInput, result interface{}) error {
	var res = map[string]interface{}{}
	var err error

	for _, op := range input.QueryDocument.Operations {
		selection, err := graphql.ApplyFragments(op.SelectionSet, input.QueryDocument.Fragments)
		if err != nil {
			return err
		}
		switch op.Operation {
		case ast.Query:
			err = q.resolveSelection(selection, res)

		case ast.Mutation:
			err = q.resolveSelection(selection, res)

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

func (q *queryer) resolveSelection(selection ast.SelectionSet, res json) (err error) {
	for _, ss := range selection {
		switch rpc := ss.(type) {
		case *ast.Field:
			res[rpc.Name], err = q.resolveCall(rpc)
			if err != nil {
				return err
			}
		}
	}
	return
}

func (q *queryer) resolveCall(rpc *ast.Field) (json, error) {
	query, ok := q.queries[rpc.Name]
	if !ok {
		return nil, fmt.Errorf("no item: %s", rpc.Name)
	}

	data := q.origServices[SvcKey{query.ServiceDescriptorProto.GetName(), query.MethodDescriptorProto.GetName()}]

	res := map[string]interface{}{}
	origRPC := data.MethodDescriptor
	conn := data.ClientConn
	var in *plugin.Type
	var inputMsg *dynamic.Message
	var inputFields = map[string]*desc.FieldDescriptor{}
	if scalar, ok := q.p.Scalars()[query.InputType]; ok {
		if !q.p.IsEmpty(scalar) {
			in = scalar
		}
	} else if input, ok := q.p.Inputs()[query.InputType]; ok {
		if !q.p.IsEmpty(input) {
			in = input
		}
	}
	inputDesc := q.origMessages[in.DescriptorProto.GetName()] // TODO
	for _, f := range inputDesc.GetFields() {
		inputFields[f.GetName()] = f
	}

	inputMsg = dynamic.NewMessage(inputDesc)

	var outputFields = map[string]*desc.FieldDescriptor{}
	if scalar, ok := q.p.Scalars()[query.InputType]; ok {
		if !q.p.IsEmpty(scalar) {
			in = scalar
		}
	} else if input, ok := q.p.Types()[query.OutputType]; ok {
		if !q.p.IsEmpty(input) {
			in = input
		}
	}

	// TODO
	outputDesc := q.origMessages[in.DescriptorProto.GetName()]
	for _, f := range outputDesc.GetFields() {
		outputFields[f.GetName()] = f
	}

	for _, arg := range rpc.Arguments.ForName("in").Value.Children {
		fname := q.p.FieldBack(in, arg.Name).GetName()
		inputMsg.SetField(inputFields[fname], arg.Value.Raw)
	}

	msg, err := grpcdynamic.NewStub(conn).InvokeRpc(context.Background(), origRPC, inputMsg)
	if err != nil {
		return res, err
	}
	dynamicMsg, ok := msg.(*dynamic.Message)
	if !ok {
		return res, err
	}
	for _, f := range rpc.SelectionSet {
		switch out := f.(type) {
		case *ast.Field:
			res[out.Name] = dynamicMsg.GetField(outputFields[q.p.FieldBack(in, out.Name).GetName()])
		}
	}
	return res, nil
}
