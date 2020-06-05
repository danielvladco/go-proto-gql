package server

import (
	"context"
	"log"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic/grpcdynamic"
	"google.golang.org/grpc"

	"github.com/danielvladco/go-proto-gql/reflect/gateway/internal/reflection"
)

type Caller interface {
	Call(ctx context.Context, svc *descriptor.ServiceDescriptorProto, rpc *descriptor.MethodDescriptorProto, message proto.Message) (proto.Message, error)
}

type caller struct {
	origServices map[SvcKey]SvcVal
}

func NewReflectCaller(endpoints []string) (*caller, []*desc.FileDescriptor, []string, error) {
	var descs []*desc.FileDescriptor

	descsconn := map[*desc.FileDescriptor]*grpc.ClientConn{}
	for _, e := range endpoints {
		conn, err := grpc.Dial(e, grpc.WithInsecure())
		if err != nil {
			return nil, nil, nil, err
		}
		client := reflection.NewClient(conn)

		tempdesc, err := client.ListPackages()
		if err != nil {
			return nil, nil, nil, err
		}

		for _, d := range tempdesc {
			descsconn[d] = conn
			descs = append(descs, d)
		}
	}

	origServices := map[SvcKey]SvcVal{}
	for _, d := range descs {
		for _, svc := range d.GetServices() {
			for _, rpc := range svc.GetMethods() {
				origServices[SvcKey{
					svc.AsServiceDescriptorProto(),
					rpc.AsMethodDescriptorProto(),
				}] = SvcVal{
					grpcdynamic.NewStub(descsconn[d]),
					rpc,
				}
			}
		}
	}

	var filesToGenerate []string
	//var protoFiles []*descriptor.FileDescriptorProto
	for _, d := range descs {
		//p := d.AsFileDescriptorProto()
		//n := fmt.Sprintf("_%d_%s", i, p.GetName())
		//p.Name = &n
		//gopkg := "github.com/danielvladco/go-proto-gql/reflect/grpcserver1/pb;pb"
		//p.Options.GoPackage = &gopkg
		filesToGenerate = append(filesToGenerate, d.AsFileDescriptorProto().GetName())
		//protoFiles = append(protoFiles, p)
		for _, dp := range getDeps(d) {
			//	protoFiles = append(protoFiles, dp.AsFileDescriptorProto())
			descs = append(descs, dp)
		}
	}

	return &caller{
		origServices: origServices,
	}, descs, filesToGenerate, nil
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

func (c caller) Call(ctx context.Context, svc *descriptor.ServiceDescriptorProto, rpc *descriptor.MethodDescriptorProto, message proto.Message) (proto.Message, error) {
	svcMapping := c.origServices[SvcKey{svc, rpc}]
	startTime := time.Now()
	res, err := svcMapping.Stub.InvokeRpc(ctx, svcMapping.MethodDescriptor, message)
	log.Printf("[INFO] grpc request took: %fms", float64(time.Since(startTime))/float64(time.Millisecond))
	return res, err
}
