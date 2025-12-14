package orderbook

import "errors"

var (
	ErrOrderNotFound = errors.New("order not found")
	ErrInvalidPrice  = errors.New("price must be greater than 0")
	ErrInvalidAmount = errors.New("amount must be greater than 0")
	ErrInvalidSide   = errors.New("invalid side")
)
