package main

import (
	"context"
	"github.com/danielvladco/go-proto-gql/reflect/examples/optionsserver/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
)

func main() {
	l, err := net.Listen("tcp", ":8082")
	if err != nil {
		log.Fatal(err)
	}

	s := grpc.NewServer()
	pb.RegisterServiceServer(s, &service{})
	reflection.Register(s)
	log.Fatal(s.Serve(l))
}

type service struct{}

func (s service) Mutate1(ctx context.Context, data *pb.Data) (*pb.Data, error) {
	return data, nil
}

func (s service) Mutate2(ctx context.Context, data *pb.Data) (*pb.Data, error) {
	return data, nil
}

func (s service) Query1(ctx context.Context, data *pb.Data) (*pb.Data, error) {
	return data, nil
}

func (s service) Publish(server pb.Service_PublishServer) error {
	panic("implement me")
}

func (s service) Subscribe(data *pb.Data, server pb.Service_SubscribeServer) error {
	panic("implement me")
}

func (s service) PubSub1(server pb.Service_PubSub1Server) error {
	panic("implement me")
}

func (s service) InvalidSubscribe1(server pb.Service_InvalidSubscribe1Server) error {
	panic("implement me")
}

func (s service) InvalidSubscribe2(data *pb.Data, server pb.Service_InvalidSubscribe2Server) error {
	panic("implement me")
}

func (s service) InvalidSubscribe3(server pb.Service_InvalidSubscribe3Server) error {
	panic("implement me")
}

func (s service) PubSub2(server pb.Service_PubSub2Server) error {
	panic("implement me")
}
