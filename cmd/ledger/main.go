package main

import (
	"context"
	"log"
	"net"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"

	"github.com/tkaixinn/trade-settlement-engine/internal/ledger"
	pb "github.com/tkaixinn/trade-settlement-engine/proto"
)

func main() {
	ctx := context.Background()

	connString := "postgres://ledger:ledger@localhost:5433/ledger"
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		log.Fatalf("unable to create connection pool: %v", err)
	}
	defer pool.Close()

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterLedgerServiceServer(grpcServer, &ledger.Server{Pool: pool})

	log.Println("ledger gRPC server listening on :50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}