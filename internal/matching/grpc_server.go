package matching

import (
	"context"
	"errors"

	pb "github.com/tkaixinn/trade-settlement-engine/proto/matching"
)

type Server struct {
	pb.UnimplementedMatchingServiceServer
	Engine    *Engine
	Publisher *Publisher
}

func (s *Server) SubmitOrder(ctx context.Context, req *pb.SubmitOrderRequest) (*pb.SubmitOrderResponse, error) {
	side, err := parseSide(req.Side)
	if err != nil {
		return nil, err
	}

	orderType, err := parseOrderType(req.OrderType)
	if err != nil {
		return nil, err
	}

	order := &Order{
		ID:        req.AccountId + "-order",
		AccountID: req.AccountId,
		Side:      side,
		Type:      orderType,
		Price:     req.Price,
		Quantity:  int(req.Quantity),
	}

	fills := s.Engine.Submit(order)

	pbFills := make([]*pb.Fill, 0, len(fills))
	for _, f := range fills {
		pbFills = append(pbFills, &pb.Fill{
			BuyAccountId:  f.BuyAccountID,
			SellAccountId: f.SellAccountID,
			Price:         f.Price,
			Quantity:      int32(f.Quantity),
		})

		if s.Publisher != nil {
			if err := s.Publisher.PublishFill(ctx, f); err != nil {
				return nil, err
			}
		}
	}

	return &pb.SubmitOrderResponse{Fills: pbFills}, nil
}

func parseSide(s string) (Side, error) {
	switch s {
	case "buy":
		return Buy, nil
	case "sell":
		return Sell, nil
	default:
		return 0, errors.New("side must be 'buy' or 'sell'")
	}
}

func parseOrderType(s string) (OrderType, error) {
	switch s {
	case "limit":
		return Limit, nil
	case "market":
		return Market, nil
	default:
		return 0, errors.New("order_type must be 'limit' or 'market'")
	}
}