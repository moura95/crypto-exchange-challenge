package orderbook

import (
	"time"

	"github.com/moura95/crypto-exchange-challenge/pkg/utils"
)

type Limit struct {
	PriceTicks  int64
	Orders      []*Order
	TotalVolume float64
}

func NewLimit(priceTicks int64) *Limit {
	return &Limit{
		PriceTicks: priceTicks,
		Orders:     []*Order{},
	}
}

// Price returns the float price for display/DTO. Source of truth is PriceTicks.
func (l *Limit) Price(priceTick float64) float64 {
	return utils.TicksToPrice(l.PriceTicks, priceTick)
}

func (l *Limit) AddOrder(o *Order) {
	o.Limit = l
	l.Orders = append(l.Orders, o)
	l.TotalVolume += o.RemainingAmount()
}

func (l *Limit) DeleteOrder(o *Order) {
	for i := 0; i < len(l.Orders); i++ {
		if l.Orders[i].ID == o.ID {
			l.Orders = append(l.Orders[:i], l.Orders[i+1:]...)
			l.TotalVolume -= o.RemainingAmount()
			o.Limit = nil
			return
		}
	}
}

// Fill fills incomingOrder against this price level.
// Self-trade prevention: skip resting orders from same user.
func (l *Limit) Fill(incomingOrder *Order, priceTick float64) []Match {
	var matches []Match
	var ordersToDelete []*Order

	levelPrice := utils.TicksToPrice(l.PriceTicks, priceTick)

	for _, existingOrder := range l.Orders {
		if incomingOrder.IsFilled() {
			break
		}
		if existingOrder.UserID == incomingOrder.UserID {
			continue
		}

		fillSize := min(incomingOrder.RemainingAmount(), existingOrder.RemainingAmount())

		incomingOrder.FilledAmount += fillSize
		existingOrder.FilledAmount += fillSize

		switch {
		case incomingOrder.IsFilled():
			incomingOrder.State = OrderFilled
		case incomingOrder.FilledAmount > 0:
			incomingOrder.State = OrderPartiallyFilled
		}

		switch {
		case existingOrder.IsFilled():
			existingOrder.State = OrderFilled
			ordersToDelete = append(ordersToDelete, existingOrder)
		case existingOrder.FilledAmount > 0:
			existingOrder.State = OrderPartiallyFilled
		}

		l.TotalVolume -= fillSize

		var bid, ask *Order
		if incomingOrder.Side == Bid {
			bid = incomingOrder
			ask = existingOrder
		} else {
			bid = existingOrder
			ask = incomingOrder
		}

		match := Match{
			Bid:        bid,
			Ask:        ask,
			Price:      levelPrice,
			SizeFilled: fillSize,
			Timestamp:  time.Now(),
		}

		matches = append(matches, match)
	}

	for _, order := range ordersToDelete {
		l.DeleteOrder(order)
	}

	return matches
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
