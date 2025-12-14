package orderbook

import (
	"sort"
	"sync"

	"github.com/moura95/crypto-exchange-challenge/pkg/utils"
)

type Orderbook struct {
	bids []*Limit
	asks []*Limit

	BidLimits map[int64]*Limit
	AskLimits map[int64]*Limit
	Orders    map[int64]*Order

	mu sync.RWMutex

	priceTick float64
}

func NewOrderbook() *Orderbook {
	return &Orderbook{
		bids:      []*Limit{},
		asks:      []*Limit{},
		BidLimits: make(map[int64]*Limit),
		AskLimits: make(map[int64]*Limit),
		Orders:    make(map[int64]*Order),
		priceTick: 0.01,
	}
}

// PlaceLimitOrder places order in orderbook and tries to match
func (ob *Orderbook) PlaceLimitOrder(order *Order) []Match {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	orderPriceTicks := utils.PriceToTicks(order.Price, ob.priceTick)

	var matches []Match

	if order.Side == Bid {
		for _, askLimit := range ob.asks {
			if askLimit.PriceTicks > orderPriceTicks {
				break
			}
			if order.IsFilled() {
				break
			}

			limitMatches := askLimit.Fill(order, ob.priceTick)
			matches = append(matches, limitMatches...)

			if len(askLimit.Orders) == 0 {
				ob.clearLimit(false, askLimit)
			}
		}
	} else {
		for _, bidLimit := range ob.bids {
			if bidLimit.PriceTicks < orderPriceTicks {
				break
			}
			if order.IsFilled() {
				break
			}

			limitMatches := bidLimit.Fill(order, ob.priceTick)
			matches = append(matches, limitMatches...)

			if len(bidLimit.Orders) == 0 {
				ob.clearLimit(true, bidLimit)
			}
		}
	}

	if !order.IsFilled() {
		ob.addOrderToBook(order, orderPriceTicks)
	}

	return matches
}

// PlaceMarketOrder executes immediately against the top of book.
func (ob *Orderbook) PlaceMarketOrder(order *Order) []Match {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	var matches []Match

	if order.Side == Bid {
		// BUY market: consume asks from best price (lowest)
		for _, askLimit := range ob.asks {
			if order.IsFilled() {
				break
			}

			limitMatches := askLimit.Fill(order, ob.priceTick)
			matches = append(matches, limitMatches...)

			if len(askLimit.Orders) == 0 {
				ob.clearLimit(false, askLimit)
			}
		}
	} else {
		// SELL market: consume bids from best price (highest)
		for _, bidLimit := range ob.bids {
			if order.IsFilled() {
				break
			}

			limitMatches := bidLimit.Fill(order, ob.priceTick)
			matches = append(matches, limitMatches...)

			if len(bidLimit.Orders) == 0 {
				ob.clearLimit(true, bidLimit)
			}
		}
	}

	// Market order never goes to the book
	if order.IsFilled() {
		order.State = OrderFilled
	} else if order.FilledAmount > 0 {
		order.State = OrderPartiallyFilled
	} else {
		// IOC behavior: executed 0 and finishes here
		order.State = OrderOpen
	}

	return matches
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

	spreadTicks := ob.asks[0].PriceTicks - ob.bids[0].PriceTicks
	return utils.TicksToPrice(spreadTicks, ob.priceTick)
}

func (ob *Orderbook) BidTotalVolume() float64 {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	total := 0.0
	for _, l := range ob.bids {
		total += l.TotalVolume
	}
	return total
}

func (ob *Orderbook) AskTotalVolume() float64 {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	total := 0.0
	for _, l := range ob.asks {
		total += l.TotalVolume
	}
	return total
}

func (ob *Orderbook) GetOrder(orderID int64) (*Order, bool) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	o, ok := ob.Orders[orderID]
	return o, ok
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

func (ob *Orderbook) clearLimit(isBid bool, limit *Limit) {
	if isBid {
		delete(ob.BidLimits, limit.PriceTicks)
		for i := 0; i < len(ob.bids); i++ {
			if ob.bids[i].PriceTicks == limit.PriceTicks {
				ob.bids = append(ob.bids[:i], ob.bids[i+1:]...)
				break
			}
		}
	} else {
		delete(ob.AskLimits, limit.PriceTicks)
		for i := 0; i < len(ob.asks); i++ {
			if ob.asks[i].PriceTicks == limit.PriceTicks {
				ob.asks = append(ob.asks[:i], ob.asks[i+1:]...)
				break
			}
		}
	}
}
