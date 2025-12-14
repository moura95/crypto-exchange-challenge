package orderbook

import "sync/atomic"

type Side string

const (
	Bid Side = "bid" // Buy
	Ask Side = "ask" // Sell
)

func (s Side) String() string { return string(s) }

type OrderState string

const (
	OrderOpen            OrderState = "open"
	OrderPartiallyFilled OrderState = "partially_filled"
	OrderFilled          OrderState = "filled"
	OrderCancelled       OrderState = "cancelled"
)

type OrderType string

const (
	OrderTypeLimit  OrderType = "limit"
	OrderTypeMarket OrderType = "market"
)

var orderIDCounter int64

func nextOrderID() int64 {
	return atomic.AddInt64(&orderIDCounter, 1)
}
