package ledger

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Entry struct {
	AccountID uuid.UUID
	Amount    float64
}

func RecordEvent(ctx context.Context, pool *pgxpool.Pool, eventID uuid.UUID, entries []Entry) error {
	var total float64
	for _, e := range entries {
		total += e.Amount
	}
	if total != 0 {
		return errors.New("ledger entries do not balance to zero")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var exists bool
	err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM processed_events WHERE event_id = $1)", eventID).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	for _, e := range entries {
		_, err = tx.Exec(ctx,
			"INSERT INTO ledger_entries (event_id, account_id, amount) VALUES ($1, $2, $3)",
			eventID, e.AccountID, e.Amount)
		if err != nil {
			return err
		}
	}

	_, err = tx.Exec(ctx, "INSERT INTO processed_events (event_id) VALUES ($1)", eventID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}