package orderbook

import (
	"errors"
	"fmt"
	"time"
)

type Order struct {
	ID           int64
	UserID       string
	Side         Side
	Type         OrderType
	Price        float64
	Amount       float64
	FilledAmount float64
	State        OrderState
	Timestamp    time.Time
	Limit        *Limit
}

func NewOrder(userID string, side Side, price, amount float64) (*Order, error) {
	if userID == "" {
		return nil, errors.New("userID cannot be empty")
	}
	if side != Bid && side != Ask {
		return nil, ErrInvalidSide
	}
	if price <= 0 {
		return nil, ErrInvalidPrice
	}
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}

	return &Order{
		ID:           nextOrderID(),
		UserID:       userID,
		Side:         side,
		Type:         OrderTypeLimit,
		Price:        price,
		Amount:       amount,
		FilledAmount: 0,
		State:        OrderOpen,
		Timestamp:    time.Now(),
	}, nil
}

func NewMarketOrder(userID string, side Side, amount float64) (*Order, error) {
	if userID == "" {
		return nil, errors.New("userID cannot be empty")
	}
	if side != Bid && side != Ask {
		return nil, ErrInvalidSide
	}
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}

	return &Order{
		ID:           nextOrderID(),
		UserID:       userID,
		Side:         side,
		Type:         OrderTypeMarket,
		Price:        0,
		Amount:       amount,
		FilledAmount: 0,
		State:        OrderOpen,
		Timestamp:    time.Now(),
	}, nil
}

func (o *Order) IsFilled() bool {
	return o.FilledAmount >= o.Amount
}

func (o *Order) RemainingAmount() float64 {
	return o.Amount - o.FilledAmount
}

func (o *Order) String() string {
	if o.Type == OrderTypeMarket {
		return fmt.Sprintf("[ID:%d User:%s %s MARKET %.8f filled:%.8f state:%s]",
			o.ID, o.UserID, o.Side, o.Amount, o.FilledAmount, o.State)
	}

	return fmt.Sprintf("[ID:%d User:%s %s LIMIT %.8f@%.2f filled:%.8f state:%s]",
		o.ID, o.UserID, o.Side, o.Amount, o.Price, o.FilledAmount, o.State)
}
