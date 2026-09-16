package matching

import "sort"

type Book struct {
	buys       map[float64][]*Order
	sells      map[float64][]*Order
	buyPrices  []float64
	sellPrices []float64
}

func NewBook() *Book {
	return &Book{
		buys:  make(map[float64][]*Order),
		sells: make(map[float64][]*Order),
	}
}

func (b *Book) addBuyPrice(price float64) {
	i := sort.Search(len(b.buyPrices), func(i int) bool { return b.buyPrices[i] <= price })
	b.buyPrices = append(b.buyPrices, 0)
	copy(b.buyPrices[i+1:], b.buyPrices[i:])
	b.buyPrices[i] = price
}

func (b *Book) removeBuyPrice(price float64) {
	for i, p := range b.buyPrices {
		if p == price {
			b.buyPrices = append(b.buyPrices[:i], b.buyPrices[i+1:]...)
			return
		}
	}
}

func (b *Book) addSellPrice(price float64) {
	i := sort.Search(len(b.sellPrices), func(i int) bool { return b.sellPrices[i] >= price })
	b.sellPrices = append(b.sellPrices, 0)
	copy(b.sellPrices[i+1:], b.sellPrices[i:])
	b.sellPrices[i] = price
}

func (b *Book) removeSellPrice(price float64) {
	for i, p := range b.sellPrices {
		if p == price {
			b.sellPrices = append(b.sellPrices[:i], b.sellPrices[i+1:]...)
			return
		}
	}
}

func (b *Book) bestBuyPrice() (float64, bool) {
	if len(b.buyPrices) == 0 {
		return 0, false
	}
	return b.buyPrices[0], true
}

func (b *Book) bestSellPrice() (float64, bool) {
	if len(b.sellPrices) == 0 {
		return 0, false
	}
	return b.sellPrices[0], true
}

func (b *Book) AddOrder(o *Order) []Fill {
	if o.Side == Buy {
		return b.matchBuy(o)
	}
	return b.matchSell(o)
}

func (b *Book) matchBuy(incoming *Order) []Fill {
	var fills []Fill

	for incoming.Quantity > 0 {
		sellPrice, found := b.bestSellPrice()
		if !found {
			break
		}
		if incoming.Type == Limit && sellPrice > incoming.Price {
			break
		}

		queue := b.sells[sellPrice]
		resting := queue[0]

		qty := min(incoming.Quantity, resting.Quantity)
		fills = append(fills, Fill{
			BuyOrderID:  incoming.ID,
			SellOrderID: resting.ID,
			Price:       sellPrice,
			Quantity:    qty,
		})

		incoming.Quantity -= qty
		resting.Quantity -= qty

		if resting.Quantity == 0 {
			queue = queue[1:]
			b.sells[sellPrice] = queue
			if len(queue) == 0 {
				delete(b.sells, sellPrice)
				b.removeSellPrice(sellPrice)
			}
		}
	}

	if incoming.Quantity > 0 && incoming.Type == Limit {
		if _, exists := b.buys[incoming.Price]; !exists {
			b.addBuyPrice(incoming.Price)
		}
		b.buys[incoming.Price] = append(b.buys[incoming.Price], incoming)
	}

	return fills
}

func (b *Book) matchSell(incoming *Order) []Fill {
	var fills []Fill

	for incoming.Quantity > 0 {
		buyPrice, found := b.bestBuyPrice()
		if !found {
			break
		}
		if incoming.Type == Limit && buyPrice < incoming.Price {
			break
		}

		queue := b.buys[buyPrice]
		resting := queue[0]

		qty := min(incoming.Quantity, resting.Quantity)
		fills = append(fills, Fill{
			BuyOrderID:  resting.ID,
			SellOrderID: incoming.ID,
			Price:       buyPrice,
			Quantity:    qty,
		})

		incoming.Quantity -= qty
		resting.Quantity -= qty

		if resting.Quantity == 0 {
			queue = queue[1:]
			b.buys[buyPrice] = queue
			if len(queue) == 0 {
				delete(b.buys, buyPrice)
				b.removeBuyPrice(buyPrice)
			}
		}
	}

	if incoming.Quantity > 0 && incoming.Type == Limit {
		if _, exists := b.sells[incoming.Price]; !exists {
			b.addSellPrice(incoming.Price)
		}
		b.sells[incoming.Price] = append(b.sells[incoming.Price], incoming)
	}

	return fills
}