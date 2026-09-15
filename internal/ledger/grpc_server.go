package ledger

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	pb "github.com/tkaixinn/trade-settlement-engine/proto"
)

type Server struct {
	pb.UnimplementedLedgerServiceServer
	Pool *pgxpool.Pool
}

func (s *Server) GetBalance(ctx context.Context, req *pb.GetBalanceRequest) (*pb.GetBalanceResponse, error) {
	accountID, err := uuid.Parse(req.AccountId)
	if err != nil {
		return nil, err
	}

	var balance float64
	err = s.Pool.QueryRow(ctx,
		"SELECT COALESCE(SUM(amount), 0) FROM ledger_entries WHERE account_id = $1",
		accountID).Scan(&balance)
	if err != nil {
		return nil, err
	}

	return &pb.GetBalanceResponse{
		AccountId: req.AccountId,
		Balance:   balance,
	}, nil
}

func (s *Server) GetLedgerHistory(ctx context.Context, req *pb.GetLedgerHistoryRequest) (*pb.GetLedgerHistoryResponse, error) {
	accountID, err := uuid.Parse(req.AccountId)
	if err != nil {
		return nil, err
	}

	rows, err := s.Pool.Query(ctx,
		"SELECT id, event_id, amount, created_at FROM ledger_entries WHERE account_id = $1 ORDER BY created_at",
		accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*pb.LedgerEntry
	for rows.Next() {
		var id, eventID uuid.UUID
		var amount float64
		var createdAt string
		if err := rows.Scan(&id, &eventID, &amount, &createdAt); err != nil {
			return nil, err
		}
		entries = append(entries, &pb.LedgerEntry{
			Id:        id.String(),
			EventId:   eventID.String(),
			Amount:    amount,
			CreatedAt: createdAt,
		})
	}

	return &pb.GetLedgerHistoryResponse{Entries: entries}, nil
}