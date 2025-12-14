package v1

type CreditDebitRequest struct {
	UserID string  `json:"user_id" example:"1"`
	Asset  string  `json:"asset" example:"BTC"`
	Amount float64 `json:"amount" example:"1"`
}

type BalanceItem struct {
	Asset     string  `json:"asset"`
	Available float64 `json:"available"`
	Locked    float64 `json:"locked"`
	Total     float64 `json:"total"`
}

type BalanceResponse struct {
	UserID   string        `json:"user_id"`
	Balances []BalanceItem `json:"balances"`
}
