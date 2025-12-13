package orderbook

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/moura95/crypto-exchange-challenge/pkg/utils"
)

// =============================================================================
// ERRORS
// =============================================================================

var (
	ErrOrderNotFound = errors.New("order not found")
	ErrInvalidPrice  = errors.New("price must be greater than 0")
	ErrInvalidAmount = errors.New("amount must be greater than 0")
	ErrInvalidSide   = errors.New("invalid side")
)

// =============================================================================
// TYPES
// =============================================================================

type Side string

const (
	Bid Side = "bid" // Buy
	Ask Side = "ask" // Sell
)

func (s Side) String() string {
	return string(s)
}

type OrderState string

const (
	OrderOpen            OrderState = "open"
	OrderPartiallyFilled OrderState = "partially_filled"
	OrderFilled          OrderState = "filled"
	OrderCancelled       OrderState = "cancelled"
)

// =============================================================================
// ID GENERATOR
// =============================================================================

var orderIDCounter int64

func nextOrderID() int64 {
	return atomic.AddInt64(&orderIDCounter, 1)
}

// =============================================================================
// ORDER
// =============================================================================

type Order struct {
	ID           int64
	UserID       string
	Side         Side
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
		Price:        price,
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
	return fmt.Sprintf("[ID:%d User:%s %s %.4f@%.2f filled:%.4f state:%s]",
		o.ID, o.UserID, o.Side, o.Amount, o.Price, o.FilledAmount, o.State)
}

// =============================================================================
// MATCH
// =============================================================================

type Match struct {
	Bid        *Order
	Ask        *Order
	Price      float64
	SizeFilled float64
	Timestamp  time.Time
}

func (m Match) String() string {
	return fmt.Sprintf("[Match: %.4f @ %.2f | Buyer:%s Seller:%s]",
		m.SizeFilled, m.Price, m.Bid.UserID, m.Ask.UserID)
}

// =============================================================================
// LIMIT (Price Level)
// =============================================================================

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

func (l *Limit) Price() float64 {
	const priceTick = 0.01
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

// Fill fill incoming order with orders from this price level
// Implement Self-Trade Prevention: skip orders from same user
// Return list of matches executed
func (l *Limit) Fill(incomingOrder *Order) []Match {
	var matches []Match
	var ordersToDelete []*Order

	for _, existingOrder := range l.Orders {
		if incomingOrder.IsFilled() {
			break
		}

		// Self-Trade Prevention: skip orders from same user
		if existingOrder.UserID == incomingOrder.UserID {
			continue
		}

		fillSize := min(incomingOrder.RemainingAmount(), existingOrder.RemainingAmount())

		incomingOrder.FilledAmount += fillSize
		existingOrder.FilledAmount += fillSize

		if incomingOrder.IsFilled() {
			incomingOrder.State = OrderFilled
		} else if incomingOrder.FilledAmount > 0 {
			incomingOrder.State = OrderPartiallyFilled
		}

		if existingOrder.IsFilled() {
			existingOrder.State = OrderFilled
			ordersToDelete = append(ordersToDelete, existingOrder)
		} else if existingOrder.FilledAmount > 0 {
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
			Price:      existingOrder.Price,
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

// =============================================================================
// ORDERBOOK
// =============================================================================

type Orderbook struct {
	bids []*Limit
	asks []*Limit

	BidLimits map[int64]*Limit
	AskLimits map[int64]*Limit
	Orders    map[int64]*Order

	mu sync.RWMutex
}

func NewOrderbook() *Orderbook {
	return &Orderbook{
		bids:      []*Limit{},
		asks:      []*Limit{},
		BidLimits: make(map[int64]*Limit),
		AskLimits: make(map[int64]*Limit),
		Orders:    make(map[int64]*Order),
	}
}

// PlaceLimitOrder places order in orderbook and tries to match
func (ob *Orderbook) PlaceLimitOrder(order *Order, priceTicks int64) []Match {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	var matches []Match

	if order.Side == Bid {
		for _, askLimit := range ob.asks {
			if askLimit.PriceTicks > priceTicks {
				break
			}

			if order.IsFilled() {
				break
			}

			limitMatches := askLimit.Fill(order)
			matches = append(matches, limitMatches...)

			if len(askLimit.Orders) == 0 {
				ob.clearLimit(false, askLimit)
			}
		}
	} else {
		for _, bidLimit := range ob.bids {
			if bidLimit.PriceTicks < priceTicks {
				break
			}

			if order.IsFilled() {
				break
			}

			limitMatches := bidLimit.Fill(order)
			matches = append(matches, limitMatches...)

			if len(bidLimit.Orders) == 0 {
				ob.clearLimit(true, bidLimit)
			}
		}
	}

	if !order.IsFilled() {
		ob.addOrderToBook(order, priceTicks)
	}

	return matches
}

func (ob *Orderbook) addOrderToBook(order *Order, priceTicks int64) {
	var limit *Limit

	if order.Side == Bid {
		limit = ob.BidLimits[priceTicks]
	} else {
		limit = ob.AskLimits[priceTicks]
	}

	if limit == nil {
		limit = NewLimit(priceTicks)

		if order.Side == Bid {
			ob.bids = append(ob.bids, limit)
			ob.BidLimits[priceTicks] = limit
			sort.Slice(ob.bids, func(i, j int) bool {
				return ob.bids[i].PriceTicks > ob.bids[j].PriceTicks
			})
		} else {
			ob.asks = append(ob.asks, limit)
			ob.AskLimits[priceTicks] = limit
			sort.Slice(ob.asks, func(i, j int) bool {
				return ob.asks[i].PriceTicks < ob.asks[j].PriceTicks
			})
		}
	}

	limit.AddOrder(order)
	ob.Orders[order.ID] = order
}

func (ob *Orderbook) CancelOrder(orderID int64) (*Order, error) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	order, exists := ob.Orders[orderID]
	if !exists {
		return nil, ErrOrderNotFound
	}

	limit := order.Limit

	if limit != nil {
		limit.DeleteOrder(order)

		if len(limit.Orders) == 0 {
			ob.clearLimit(order.Side == Bid, limit)
		}
	}

	delete(ob.Orders, orderID)
	order.State = OrderCancelled

	return order, nil
}

func (ob *Orderbook) clearLimit(isBid bool, limit *Limit) {
	if isBid {
		delete(ob.BidLimits, limit.PriceTicks) // ← use PriceTicks
		for i := 0; i < len(ob.bids); i++ {
			if ob.bids[i].PriceTicks == limit.PriceTicks {
				ob.bids = append(ob.bids[:i], ob.bids[i+1:]...)
				break
			}
		}
	} else {
		delete(ob.AskLimits, limit.PriceTicks) // ← use PriceTicks
		for i := 0; i < len(ob.asks); i++ {
			if ob.asks[i].PriceTicks == limit.PriceTicks {
				ob.asks = append(ob.asks[:i], ob.asks[i+1:]...)
				break
			}
		}
	}
}

func (ob *Orderbook) Bids() []*Limit {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	return ob.bids
}

func (ob *Orderbook) Asks() []*Limit {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	return ob.asks
}

func (ob *Orderbook) BestBid() (*Limit, bool) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	if len(ob.bids) == 0 {
		return nil, false
	}

	return ob.bids[0], true
}

func (ob *Orderbook) BestAsk() (*Limit, bool) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	if len(ob.asks) == 0 {
		return nil, false
	}

	return ob.asks[0], true
}

func (ob *Orderbook) Spread() float64 {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	if len(ob.bids) == 0 || len(ob.asks) == 0 {
		return 0
	}

	// Calculate spread in ticks
	spreadTicks := ob.asks[0].PriceTicks - ob.bids[0].PriceTicks

	// Convert back to price using utility function
	const priceTick = 0.01
	return utils.TicksToPrice(spreadTicks, priceTick)
}

func (ob *Orderbook) BidTotalVolume() float64 {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	total := 0.0
	for _, limit := range ob.bids {
		total += limit.TotalVolume
	}
	return total
}

func (ob *Orderbook) AskTotalVolume() float64 {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	total := 0.0
	for _, limit := range ob.asks {
		total += limit.TotalVolume
	}
	return total
}

func (ob *Orderbook) GetOrder(orderID int64) (*Order, bool) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	order, exists := ob.Orders[orderID]
	return order, exists
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
