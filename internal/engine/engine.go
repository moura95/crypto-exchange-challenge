package engine

import (
	"errors"
	"sync"

	"github.com/moura95/crypto-exchange-challenge/internal/account"
	"github.com/moura95/crypto-exchange-challenge/internal/orderbook"
)

// =============================================================================
// ERRORS
// =============================================================================

var (
	ErrInvalidPair   = errors.New("invalid pair")
	ErrOrderNotFound = errors.New("order not found")
	ErrUnauthorized  = errors.New("unauthorized: order belongs to another user")
)

// =============================================================================
// PAIR
// =============================================================================

type Pair struct {
	Base  string // BTC
	Quote string // BRL
}

func (p Pair) String() string {
	return p.Base + "/" + p.Quote
}

func (p Pair) IsValid() bool {
	return p.Base != "" && p.Quote != ""
}

// =============================================================================
// ENGINE
// =============================================================================

type Engine struct {
	orderbooks map[string]*orderbook.Orderbook // orderbooks["BTC/BRL"]
	accounts   *account.Manager
	mu         sync.RWMutex
}

func NewEngine() *Engine {
	return &Engine{
		orderbooks: make(map[string]*orderbook.Orderbook),
		accounts:   account.NewManager(),
	}
}

// getOrCreateOrderbook returns existing orderbook or creates new one
func (e *Engine) getOrCreateOrderbook(pair Pair) *orderbook.Orderbook {
	key := pair.String()

	if ob, exists := e.orderbooks[key]; exists {
		return ob
	}

	ob := orderbook.NewOrderbook()
	e.orderbooks[key] = ob
	return ob
}

// =============================================================================
// ORDER OPERATIONS
// =============================================================================

// PlaceOrder creates a new order, locks balance, and tries to match
func (e *Engine) PlaceOrder(userID string, pair Pair, side orderbook.Side, price, amount float64) (*orderbook.Order, []orderbook.Match, error) {
	if !pair.IsValid() {
		return nil, nil, ErrInvalidPair
	}

	// Create order (validates price, amount, side)
	order, err := orderbook.NewOrder(userID, side, price, amount)
	if err != nil {
		return nil, nil, err
	}

	// Calculate how much to lock
	var lockAsset string
	var lockAmount float64

	if side == orderbook.Bid {
		// Buy: lock quote asset (BRL)
		lockAsset = pair.Quote
		lockAmount = price * amount
	} else {
		// Sell: lock base asset (BTC)
		lockAsset = pair.Base
		lockAmount = amount
	}

	// Lock balance
	err = e.accounts.Lock(userID, lockAsset, lockAmount)
	if err != nil {
		return nil, nil, err
	}

	e.mu.Lock()
	ob := e.getOrCreateOrderbook(pair)
	e.mu.Unlock()

	// Place order in orderbook
	matches := ob.PlaceLimitOrder(order)

	// Execute transfers for each match
	for _, match := range matches {
		e.executeTransfer(pair, match)
	}

	// If order was partially filled, unlock the unused portion
	if order.RemainingAmount() > 0 && order.FilledAmount > 0 {
		var unlockAmount float64
		if side == orderbook.Bid {
			// For partial buy: unlock unused quote (BRL)
			unlockAmount = order.RemainingAmount() * price
			// Note: remaining is still locked in the orderbook
		}
		// For sell, base asset remains locked for the remaining order
		_ = unlockAmount // Not unlocking here, order is still active
	}

	return order, matches, nil
}

// CancelOrder cancels an order and unlocks the reserved balance
func (e *Engine) CancelOrder(userID string, pair Pair, orderID int64) (*orderbook.Order, error) {
	if !pair.IsValid() {
		return nil, ErrInvalidPair
	}

	e.mu.RLock()
	ob, exists := e.orderbooks[pair.String()]
	e.mu.RUnlock()

	if !exists {
		return nil, ErrOrderNotFound
	}

	// Get order to check ownership
	order, exists := ob.GetOrder(orderID)
	if !exists {
		return nil, ErrOrderNotFound
	}

	// Check if user owns the order
	if order.UserID != userID {
		return nil, ErrUnauthorized
	}

	// Cancel order in orderbook
	cancelledOrder, err := ob.CancelOrder(orderID)
	if err != nil {
		return nil, err
	}

	// Unlock remaining balance
	var unlockAsset string
	var unlockAmount float64

	if cancelledOrder.Side == orderbook.Bid {
		// Buy order: unlock remaining quote (BRL)
		unlockAsset = pair.Quote
		unlockAmount = cancelledOrder.RemainingAmount() * cancelledOrder.Price
	} else {
		// Sell order: unlock remaining base (BTC)
		unlockAsset = pair.Base
		unlockAmount = cancelledOrder.RemainingAmount()
	}

	if unlockAmount > 0 {
		err = e.accounts.Unlock(userID, unlockAsset, unlockAmount)
		if err != nil {
			// Log error but don't fail - order is already cancelled
			// In production, this would need proper handling
		}
	}

	return cancelledOrder, nil
}

// executeTransfer executes the balance transfer after a match
func (e *Engine) executeTransfer(pair Pair, match orderbook.Match) {
	buyer := match.Bid.UserID
	seller := match.Ask.UserID
	baseAmount := match.SizeFilled                // BTC
	quoteAmount := match.SizeFilled * match.Price // BRL

	// Seller: debit locked base (BTC), credit quote (BRL)
	e.accounts.DebitLocked(seller, pair.Base, baseAmount)
	e.accounts.Credit(seller, pair.Quote, quoteAmount)

	// Buyer: debit locked quote (BRL), credit base (BTC)
	e.accounts.DebitLocked(buyer, pair.Quote, quoteAmount)
	e.accounts.Credit(buyer, pair.Base, baseAmount)
}

// =============================================================================
// ORDERBOOK OPERATIONS
// =============================================================================

// GetOrderbook returns the orderbook for a pair
func (e *Engine) GetOrderbook(pair Pair) *orderbook.Orderbook {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.orderbooks[pair.String()]
}
