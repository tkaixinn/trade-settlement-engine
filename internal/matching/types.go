package matching

type Side int

const (
	Buy Side = iota
	Sell
)

type OrderType int

const (
	Limit OrderType = iota
	Market
)

type Order struct {
	ID        string
	AccountID string
	Side      Side
	Type      OrderType
	Price     float64
	Quantity  int
}

type Fill struct {
	BuyOrderID    string
	SellOrderID   string
	BuyAccountID  string
	SellAccountID string
	Price         float64
	Quantity      int
}