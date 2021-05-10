package server

import (
	"context"
	"github.com/danielvladco/go-proto-gql/pkg/generator"
	"github.com/danielvladco/go-proto-gql/pkg/protoparser"
	"google.golang.org/grpc/credentials"
	"log"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic/grpcdynamic"
	"google.golang.org/grpc"

	"github.com/danielvladco/go-proto-gql/pkg/reflection"
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

func NewReflectCaller(config *Grpc) (Caller, []*desc.FileDescriptor, error) {
	var descs []*desc.FileDescriptor
	c := &caller{serviceStub: map[*desc.ServiceDescriptor]grpcdynamic.Stub{}}
	descsconn := map[*desc.FileDescriptor]*grpc.ClientConn{}
	for _, e := range config.Services {
		var options []grpc.DialOption
		if e.Authentication != nil {
			if e.Authentication.Insecure != nil && *e.Authentication.Insecure {
				options = append(options, grpc.WithInsecure())
			}
			if e.Authentication.Tls != nil {
				cred, err := credentials.NewServerTLSFromFile(e.Authentication.Tls.Certificate, e.Authentication.Tls.PrivateKey)
				if err != nil {
					return nil, nil, err
				}
				options = append(options, grpc.WithTransportCredentials(cred))
			}
		}

		conn, err := grpc.Dial(e.Address, options...)
		if err != nil {
			return nil, nil, err
		}
		if e.Reflection {
			descs, err = reflection.NewClient(conn).ListPackages()
			if err != nil {
				return nil, nil, err
			}
		}
		if e.ProtoFiles != nil {
			descs, err = protoparser.Parse(config.ImportPaths, e.ProtoFiles)
			if err != nil {
				return nil, nil, err
			}
		}
		for _, d := range descs {
			descsconn[d] = conn
			for _, svc := range d.GetServices() {
				c.serviceStub[svc] = grpcdynamic.NewStub(descsconn[d])
			}
		}
	}

	return c, generator.ResolveProtoFilesRecursively(descs), nil
}

func getDeps(file *desc.FileDescriptor) []*desc.FileDescriptor {
	mp := map[*desc.FileDescriptor]struct{}{}
	getAllDependencies(file, mp)
	deps := make([]*desc.FileDescriptor, len(mp))
	i := 0
	for dp := range mp {
		deps[i] = dp
		i++
	}
	return deps
}

func getAllDependencies(file *desc.FileDescriptor, files map[*desc.FileDescriptor]struct{}) {
	deps := file.GetDependencies()
	for _, d := range deps {
		files[d] = struct{}{}
		getAllDependencies(d, files)
	}
}

func (c caller) Call(ctx context.Context, rpc *desc.MethodDescriptor, message proto.Message) (proto.Message, error) {
	startTime := time.Now()
	res, err := c.serviceStub[rpc.GetService()].InvokeRpc(ctx, rpc, message)
	rpc.GetService()
	log.Printf("[INFO] grpc call %q took: %fms", rpc.GetFullyQualifiedName(), float64(time.Since(startTime))/float64(time.Millisecond))
	return res, err
}
