package main

import (
	"context"
	"log"

	"github.com/google/uuid"

	"github.com/tkaixinn/trade-settlement-engine/internal/matching"
)

func main() {
	ctx := context.Background()

	engine := matching.NewEngine()
	publisher := matching.NewPublisher()

	sellOrder := &matching.Order{
		ID:        uuid.New().String(),
		AccountID: "05aeff6e-4cdc-4119-b127-7312bf5ae9a6",
		Side:      matching.Sell,
		Type:      matching.Limit,
		Price:     100,
		Quantity:  10,
	}
	engine.Submit(sellOrder)
	log.Println("resting sell order placed")

	buyOrder := &matching.Order{
		ID:        uuid.New().String(),
		AccountID: "ebedb0d4-8b95-4f66-b047-77d308dea1c3",
		Side:      matching.Buy,
		Type:      matching.Market,
		Quantity:  10,
	}
	fills := engine.Submit(buyOrder)
	log.Printf("buy order matched, %d fills", len(fills))

	for _, f := range fills {
		if err := publisher.PublishFill(ctx, f); err != nil {
			log.Printf("failed to publish fill: %v", err)
		}
	}
}