package matching

import (
	"sync"
	"testing"
)

func TestEngine_ConcurrentSubmission(t *testing.T) {
	engine := NewEngine()

	var wg sync.WaitGroup
	numOrders := 100

	for i := 0; i < numOrders; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			order := &Order{
				ID:       "sell-order",
				Side:     Sell,
				Type:     Limit,
				Price:    100,
				Quantity: 1,
			}
			engine.Submit(order)
		}(i)
	}

	wg.Wait()

	buyOrder := &Order{
		ID:       "buy-check",
		Side:     Buy,
		Type:     Market,
		Quantity: numOrders,
	}
	fills := engine.Submit(buyOrder)

	totalFilled := 0
	for _, f := range fills {
		totalFilled += f.Quantity
	}

	if totalFilled != numOrders {
		t.Errorf("expected %d units filled, got %d", numOrders, totalFilled)
	}
}