package main

import (
	context "context"
	pb "github.com/danielvladco/go-proto-gql/reflect/grpcserver1/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
)

func main() {
	l, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}

	s := grpc.NewServer()
	pb.RegisterServerServiceServer(s, &service{})
	reflection.Register(s)
	log.Fatal(s.Serve(l))
}

type service struct {
}

func (s service) Test(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	log.Println(req.Id)
	log.Println(req.Name)
	return &pb.Response{
		Id:    req.Id,
		Name:  req.Name,
		Tel:   "some tel",
		Email: "some email",
	}, nil
}
