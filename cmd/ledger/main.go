package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"google.golang.org/grpc"

	"github.com/tkaixinn/trade-settlement-engine/internal/analytics"
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

	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()

	chConn, err := analytics.NewClickHouseConn(ctx)
	if err != nil {
		log.Fatalf("unable to connect to clickhouse: %v", err)
	}
	defer chConn.Close()

	go startGRPCServer(pool)
	go startMetricsServer()

	consumeKafka(ctx, pool, rdb, chConn)
}

func startGRPCServer(pool *pgxpool.Pool) {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterLedgerServiceServer(grpcServer, &ledger.Server{Pool: pool})

	log.Println("ledger gRPC server listening on :50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve grpc: %v", err)
	}
}

func startMetricsServer() {
	http.Handle("/metrics", promhttp.Handler())
	log.Println("metrics server listening on :9101")
	if err := http.ListenAndServe(":9101", nil); err != nil {
		log.Fatalf("failed to serve metrics: %v", err)
	}
}

func consumeKafka(ctx context.Context, pool *pgxpool.Pool, rdb *redis.Client, chConn clickhouse.Conn) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "fill-events",
		GroupID: "ledger-service",
	})
	defer reader.Close()

	log.Println("kafka consumer listening on fill-events")

	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			log.Printf("error reading message: %v", err)
			continue
		}

		var fill ledger.FillEvent
		if err := json.Unmarshal(msg.Value, &fill); err != nil {
			log.Printf("error decoding message: %v", err)
			continue
		}

		eventID, err := uuid.Parse(fill.EventID)
		if err != nil {
			log.Printf("invalid event id: %v", err)
			continue
		}
		traderID, err := uuid.Parse(fill.TraderID)
		if err != nil {
			log.Printf("invalid trader id: %v", err)
			continue
		}
		clearingID, err := uuid.Parse(fill.ClearingID)
		if err != nil {
			log.Printf("invalid clearing id: %v", err)
			continue
		}

		entries := []ledger.Entry{
			{AccountID: traderID, Amount: fill.Amount},
			{AccountID: clearingID, Amount: -fill.Amount},
		}

		if err := ledger.RecordEvent(ctx, pool, rdb, eventID, entries); err != nil {
			log.Printf("failed to record event: %v", err)
			continue
		}

		log.Printf("recorded event %s", eventID)

		if err := analytics.InsertFillEvent(ctx, chConn, fill); err != nil {
			log.Printf("failed to insert into clickhouse: %v", err)
		}
	}
}