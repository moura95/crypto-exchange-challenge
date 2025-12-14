package v1

type LimitLevel struct {
	Price       float64 `json:"price"`
	TotalVolume float64 `json:"total_volume"`
	OrderCount  int     `json:"order_count"`
}

type OrderbookResponse struct {
	Pair           string       `json:"pair"`
	Bids           []LimitLevel `json:"bids"`
	Asks           []LimitLevel `json:"asks"`
	Spread         float64      `json:"spread"`
	BidTotalVolume float64      `json:"bid_total_volume"`
	AskTotalVolume float64      `json:"ask_total_volume"`
}
