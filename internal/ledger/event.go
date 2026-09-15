package ledger

type FillEvent struct {
	EventID  string  `json:"event_id"`
	TraderID string  `json:"trader_id"`
	ClearingID string `json:"clearing_id"`
	Amount   float64 `json:"amount"`
}