// Package grpcreflection provides gRPC reflection client.
// Currently, gRPC reflection depends on Protocol Buffers, so we split this package from grpc package.
package reflection

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jhump/protoreflect/desc"
	gr "github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/grpc/status"
)

const reflectionServiceName = "grpc.reflection.v1alpha.ServerReflection"

var ErrTLSHandshakeFailed = errors.New("TLS handshake failed")

// Client defines gRPC reflection client.
type Client interface {
	// ListPackages lists file descriptors from the gRPC reflection server.
	// ListPackages returns these errors:
	//   - ErrTLSHandshakeFailed: TLS misconfig.
	ListPackages() ([]*desc.FileDescriptor, error)
	// Reset clears internal states of Client.
	Reset()
}

type client struct {
	client *gr.Client
}

// NewClient returns an instance of gRPC reflection client for gRPC protocol.
func NewClient(conn grpc.ClientConnInterface) Client {
	return &client{
		client: gr.NewClient(context.Background(), grpc_reflection_v1alpha.NewServerReflectionClient(conn)),
	}
}

// NewWebClient returns an instance of gRPC reflection client for gRPC-Web protocol.
//func NewWebClient(conn *grpcweb.ClientConn) Client {
//	return &client{
//		client: gr.NewClient(context.Background(), grpcweb_reflection_v1alpha.NewServerReflectionClient(conn)),
//	}
//}

func (c *client) ListPackages() ([]*desc.FileDescriptor, error) {
	//c.client.FileContainingExtension()
	ssvcs, err := c.client.ListServices()
	if err != nil {
		msg := status.Convert(err).Message()
		// Check whether the error message contains TLS related error.
		// If the server didn't enable TLS, the error message contains the first string.
		// If Evans didn't enable TLS against to the TLS enabled server, the error message contains
		// the second string.
		if strings.Contains(msg, "tls: first record does not look like a TLS handshake") ||
			strings.Contains(msg, "latest connection error: <nil>") {
			return nil, ErrTLSHandshakeFailed
		}
		return nil, fmt.Errorf("failed to list services from reflecton enabled gRPC server: %w", err)
	}

	var fds []*desc.FileDescriptor
	for _, s := range ssvcs {
		if s == reflectionServiceName {
			continue
		}
		svc, err := c.client.ResolveService(s)
		if err != nil {
			return nil, err
		}

		fd := svc.GetFile() //.AsFileDescriptorProto()
		fds = append(fds, fd)
	}
	return fds, nil
}

func (c *client) Reset() {
	c.client.Reset()
}
