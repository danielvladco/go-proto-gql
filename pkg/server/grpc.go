package server

import (
	"context"
	"github.com/catalystsquad/go-proto-gql/pkg/protoparser"
	"google.golang.org/grpc/credentials"
	"log"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic/grpcdynamic"
	"google.golang.org/grpc"

	"github.com/catalystsquad/go-proto-gql/pkg/reflection"
)

type Grpc struct {
	Services    []*Service
	ImportPaths []string
}

type Caller interface {
	Call(ctx context.Context, rpc *desc.MethodDescriptor, message proto.Message) (proto.Message, error)
}

type caller struct {
	serviceStub map[*desc.ServiceDescriptor]grpcdynamic.Stub
}

func NewReflectCaller(config *Grpc) (_ Caller, descs []*desc.FileDescriptor, err error) {
	c := &caller{serviceStub: map[*desc.ServiceDescriptor]grpcdynamic.Stub{}}
	descsconn := map[*desc.FileDescriptor]*grpc.ClientConn{}
	for _, e := range config.Services {
		var options []grpc.DialOption
		if e.Authentication != nil && e.Authentication.Tls != nil {
			cred, err := credentials.NewServerTLSFromFile(e.Authentication.Tls.Certificate, e.Authentication.Tls.PrivateKey)
			if err != nil {
				return nil, nil, err
			}
			options = append(options, grpc.WithTransportCredentials(cred))
		} else {
			options = append(options, grpc.WithInsecure())
		}
		conn, err := grpc.Dial(e.Address, options...)
		if err != nil {
			return nil, nil, err
		}
		if e.Reflection {
			newDescs, err := reflection.NewClientWithImportsResolver(conn).ListPackages()
			if err != nil {
				return nil, nil, err
			}
			descs = append(descs, newDescs...)
		}
		if e.ProtoFiles != nil {
			newDescs, err := protoparser.Parse(config.ImportPaths, e.ProtoFiles)
			if err != nil {
				return nil, nil, err
			}
			descs = append(descs, newDescs...)
		}
		for _, d := range descs {
			descsconn[d] = conn
			for _, svc := range d.GetServices() {
				c.serviceStub[svc] = grpcdynamic.NewStub(descsconn[d])
			}
		}
	}

	return c, descs, nil
}

func (c caller) Call(ctx context.Context, rpc *desc.MethodDescriptor, message proto.Message) (proto.Message, error) {
	startTime := time.Now()
	res, err := c.serviceStub[rpc.GetService()].InvokeRpc(ctx, rpc, message)
	log.Printf("[INFO] grpc call %q took: %fms", rpc.GetFullyQualifiedName(), float64(time.Since(startTime))/float64(time.Millisecond))
	return res, err
}
