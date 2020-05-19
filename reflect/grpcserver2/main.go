package main

import (
	"context"
	pb "github.com/danielvladco/go-proto-gql/reflect/grpcserver2/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
)

func main() {
	l, err := net.Listen("tcp", ":8081")
	if err != nil {
		log.Fatal(err)
	}

	s := grpc.NewServer()
	pb.RegisterAuthServiceServer(s, &service{})
	reflection.Register(s)
	log.Fatal(s.Serve(l))
}

type service struct {
}

func (s service) SingIn(ctx context.Context, req *pb.SingInReq) (*pb.SingInRes, error) {
	log.Println(req.Id)
	log.Println(req.User)
	return &pb.SingInRes{
		Id:   req.Id,
		User: req.User,
	}, nil
}

func (s service) SignUp(ctx context.Context, req *pb.SignUpReq) (*pb.SignUpRes, error) {
	log.Println(req.Id)
	log.Println(req.User)
	return &pb.SignUpRes{
		Id:   req.Id,
		User: req.User,
	}, nil
}

func (s service) Logout(ctx context.Context, req *pb.LogoutReq) (*pb.LogoutRes, error) {
	log.Println(req.Id)
	log.Println(req.User)
	return &pb.LogoutRes{
		Id:   req.Id,
		User: req.User,
	}, nil
}
