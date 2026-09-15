package main

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/tkaixinn/trade-settlement-engine/internal/ledger"
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

	traderA := uuid.MustParse("ebedb0d4-8b95-4f66-b047-77d308dea1c3")
	clearing := uuid.MustParse("05aeff6e-4cdc-4119-b127-7312bf5ae9a6")
	eventID := uuid.New()

	entries := []ledger.Entry{
		{AccountID: traderA, Amount: 100},
		{AccountID: clearing, Amount: -100},
	}

	err = ledger.RecordEvent(ctx, pool, rdb, eventID, entries)
	if err != nil {
		log.Fatalf("failed to record event: %v", err)
	}

	fmt.Println("event recorded successfully:", eventID)
}