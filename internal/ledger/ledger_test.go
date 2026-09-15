package ledger

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func setupTest(t *testing.T) (*pgxpool.Pool, *redis.Client) {
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, "postgres://ledger:ledger@localhost:5433/ledger")
	if err != nil {
		t.Fatalf("failed to connect to postgres: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})

	return pool, rdb
}

func TestRecordEvent_DuplicateEventIsIgnored(t *testing.T) {
	ctx := context.Background()
	pool, rdb := setupTest(t)
	defer pool.Close()
	defer rdb.Close()

	accountA := uuid.New()
	accountB := uuid.New()

	_, err := pool.Exec(ctx, "INSERT INTO accounts (id, name, account_type) VALUES ($1, 'Test A', 'trader')", accountA)
	if err != nil {
		t.Fatalf("failed to insert account A: %v", err)
	}
	_, err = pool.Exec(ctx, "INSERT INTO accounts (id, name, account_type) VALUES ($1, 'Test B', 'clearing')", accountB)
	if err != nil {
		t.Fatalf("failed to insert account B: %v", err)
	}

	eventID := uuid.New()
	entries := []Entry{
		{AccountID: accountA, Amount: 75},
		{AccountID: accountB, Amount: -75},
	}

	if err := RecordEvent(ctx, pool, rdb, eventID, entries); err != nil {
		t.Fatalf("first RecordEvent call failed: %v", err)
	}
	if err := RecordEvent(ctx, pool, rdb, eventID, entries); err != nil {
		t.Fatalf("second RecordEvent call failed: %v", err)
	}

	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM ledger_entries WHERE event_id = $1", eventID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to count ledger entries: %v", err)
	}

	if count != 2 {
		t.Errorf("expected exactly 2 ledger entries after duplicate call, got %d", count)
	}
}

func TestRecordEvent_UnbalancedEntriesRejected(t *testing.T) {
	ctx := context.Background()
	pool, rdb := setupTest(t)
	defer pool.Close()
	defer rdb.Close()

	accountA := uuid.New()
	eventID := uuid.New()

	entries := []Entry{
		{AccountID: accountA, Amount: 100},
	}

	err := RecordEvent(ctx, pool, rdb, eventID, entries)
	if err == nil {
		t.Error("expected an error for unbalanced entries, got nil")
	}
}

func TestRecordEvent_ValidEventIsRecorded(t *testing.T) {
	ctx := context.Background()
	pool, rdb := setupTest(t)
	defer pool.Close()
	defer rdb.Close()

	accountA := uuid.New()
	accountB := uuid.New()

	_, err := pool.Exec(ctx, "INSERT INTO accounts (id, name, account_type) VALUES ($1, 'Test C', 'trader')", accountA)
	if err != nil {
		t.Fatalf("failed to insert account A: %v", err)
	}
	_, err = pool.Exec(ctx, "INSERT INTO accounts (id, name, account_type) VALUES ($1, 'Test D', 'clearing')", accountB)
	if err != nil {
		t.Fatalf("failed to insert account B: %v", err)
	}

	eventID := uuid.New()
	entries := []Entry{
		{AccountID: accountA, Amount: 30},
		{AccountID: accountB, Amount: -30},
	}

	if err := RecordEvent(ctx, pool, rdb, eventID, entries); err != nil {
		t.Fatalf("expected valid event to succeed, got error: %v", err)
	}

	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM ledger_entries WHERE event_id = $1", eventID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to count ledger entries: %v", err)
	}
	if count != 2 {
		t.Errorf("expected exactly 2 ledger entries for a new valid event, got %d", count)
	}
}