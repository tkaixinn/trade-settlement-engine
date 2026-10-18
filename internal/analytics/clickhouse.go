package analytics

import (
	"context"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/tkaixinn/trade-settlement-engine/internal/ledger"
)

func NewClickHouseConn(ctx context.Context) (clickhouse.Conn, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{"localhost:9000"},
		Auth: clickhouse.Auth{
			Database: "analytics",
			Username: "default",
			Password: "clickhouse",
		},
	})
	if err != nil {
		return nil, err
	}
	if err := conn.Ping(ctx); err != nil {
		return nil, err
	}
	return conn, nil
}

func InsertFillEvent(ctx context.Context, conn clickhouse.Conn, fill ledger.FillEvent) error {
	return conn.Exec(ctx,
		"INSERT INTO fill_events_analytics (event_id, trader_id, clearing_id, amount) VALUES (?, ?, ?, ?)",
		fill.EventID, fill.TraderID, fill.ClearingID, fill.Amount)
}