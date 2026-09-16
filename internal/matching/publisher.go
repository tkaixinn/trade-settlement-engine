package matching

import (
	"context"
	"encoding/json"
	"log"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

type FillEvent struct {
	EventID    string  `json:"event_id"`
	TraderID   string  `json:"trader_id"`
	ClearingID string  `json:"clearing_id"`
	Amount     float64 `json:"amount"`
}

type Publisher struct {
	writer *kafka.Writer
}

func NewPublisher() *Publisher {
	return &Publisher{
		writer: &kafka.Writer{
			Addr:     kafka.TCP("localhost:9092"),
			Topic:    "fill-events",
			Balancer: &kafka.LeastBytes{},
		},
	}
}

func (p *Publisher) PublishFill(ctx context.Context, f Fill) error {
	amount := f.Price * float64(f.Quantity)

	event := FillEvent{
		EventID:    uuid.New().String(),
		TraderID:   f.BuyAccountID,
		ClearingID: f.SellAccountID,
		Amount:     amount,
	}

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	if err := p.writer.WriteMessages(ctx, kafka.Message{Value: data}); err != nil {
		log.Printf("failed to publish fill event: %v", err)
		return err
	}

	log.Printf("published fill event %s", event.EventID)
	return nil
}