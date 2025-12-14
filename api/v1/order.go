package v1

import "time"

type PlaceOrderRequest struct {
	UserID string  `json:"user_id"`
	Pair   string  `json:"pair"`  // "BTC/BRL"
	Side   string  `json:"side"`  // "bid" ou "ask"
	Type   string  `json:"type"`  // "limit" ou "market"
	Price  float64 `json:"price"` // 0 para market orders
	Amount float64 `json:"amount"`
}

type OrderResponse struct {
	ID           int64     `json:"id"`
	UserID       string    `json:"user_id"`
	Pair         string    `json:"pair"`
	Side         string    `json:"side"`
	Type         string    `json:"type"`
	Price        float64   `json:"price"`
	Amount       float64   `json:"amount"`
	FilledAmount float64   `json:"filled_amount"`
	State        string    `json:"state"`
	Timestamp    time.Time `json:"timestamp"`
}

type MatchResponse struct {
	BidOrderID int64     `json:"bid_order_id"`
	AskOrderID int64     `json:"ask_order_id"`
	Price      float64   `json:"price"`
	SizeFilled float64   `json:"size_filled"`
	Timestamp  time.Time `json:"timestamp"`
}

type PlaceOrderResponse struct {
	Order   OrderResponse   `json:"order"`
	Matches []MatchResponse `json:"matches"`
}

type CancelOrderRequest struct {
	UserID  string `json:"user_id"`
	Pair    string `json:"pair"`
	OrderID int64  `json:"order_id"`
}
