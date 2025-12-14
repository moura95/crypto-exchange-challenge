package engine

import "errors"

var (
	ErrInvalidPair       = errors.New("invalid pair")
	ErrInvalidPriceTick  = errors.New("price not aligned to tick")
	ErrInvalidAmountTick = errors.New("amount not aligned to tick")
	ErrOrderNotFound     = errors.New("order not found")
	ErrUnauthorized      = errors.New("unauthorized: order belongs to another user")
)
