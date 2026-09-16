package main

import (
	"log"
	"net"

	"google.golang.org/grpc"

	"github.com/tkaixinn/trade-settlement-engine/internal/matching"
	pb "github.com/tkaixinn/trade-settlement-engine/proto/matching"
)

func main() {
	engine := matching.NewEngine()
	publisher := matching.NewPublisher()

	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterMatchingServiceServer(grpcServer, &matching.Server{
		Engine:    engine,
		Publisher: publisher,
	})

	log.Println("matching gRPC server listening on :50052")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}