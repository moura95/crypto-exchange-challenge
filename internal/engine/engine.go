package engine

import (
	"errors"
	"fmt"
	"sync"

	"github.com/moura95/crypto-exchange-challenge/internal/account"
	"github.com/moura95/crypto-exchange-challenge/internal/orderbook"
	"github.com/moura95/crypto-exchange-challenge/pkg/utils"
)

// =============================================================================
// ERRORS
// =============================================================================

var (
	ErrInvalidPair       = errors.New("invalid pair")
	ErrInvalidPriceTick  = errors.New("price not aligned to tick")
	ErrInvalidAmountTick = errors.New("amount not aligned to tick")
	ErrOrderNotFound     = errors.New("order not found")
	ErrUnauthorized      = errors.New("unauthorized: order belongs to another user")
)

// =============================================================================
// CONSTANTS - Ticks do mercado BTC/BRL
// =============================================================================

const (
	PriceTick  = 0.01
	AmountTick = 0.00000001
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
	return p.Base != "" && p.Quote != "" && p.Quote == "BRL"
}

// =============================================================================
// ENGINE
// =============================================================================

type Engine struct {
	orderbooks map[string]*orderbook.Orderbook
	accounts   *account.Manager
	mu         sync.RWMutex
}

func NewEngine() *Engine {
	return &Engine{
		orderbooks: make(map[string]*orderbook.Orderbook),
		accounts:   account.NewManager(),
	}
}

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

	// 1. Normalize Price
	price = utils.FloorToTick(price, PriceTick)

	// 2. Validate
	if !utils.IsValidTick(price, PriceTick) {
		return nil, nil, ErrInvalidPriceTick
	}

	// 3. Normalize amount
	amount = utils.FloorToTick(amount, AmountTick)

	// 4. Validate amount
	if !utils.IsValidTick(amount, AmountTick) {
		return nil, nil, ErrInvalidAmountTick
	}

	// 5. Convert Price to tick
	priceTicks := utils.PriceToTicks(price, PriceTick)

	order, err := orderbook.NewOrder(userID, side, price, amount)
	if err != nil {
		return nil, nil, err
	}

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

	err = e.accounts.Lock(userID, lockAsset, lockAmount)
	if err != nil {
		return nil, nil, err
	}

	e.mu.Lock()
	ob := e.getOrCreateOrderbook(pair)

	// Place order in orderbook
	matches := ob.PlaceLimitOrder(order, priceTicks)

	// Execute transfers for each match
	for _, match := range matches {
		if err := e.executeTransfer(pair, match); err != nil {
			e.mu.Unlock()
			// Rollback
			return nil, nil, fmt.Errorf("transfer failed: %w", err)
		}
	}

	// If order was partially filled, unlock the unused portion
	if len(matches) > 0 && !order.IsFilled() {
		if side == orderbook.Bid {
			executedQuote := 0.0
			for _, m := range matches {
				executedQuote += m.SizeFilled * m.Price
			}

			initialLock := price * order.Amount

			stillLocked := price * order.RemainingAmount()

			refund := initialLock - executedQuote - stillLocked

			if refund > 0.000001 {
				e.accounts.Unlock(userID, pair.Quote, refund)
			}
		}
	}

	e.mu.Unlock()

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

	// Get order to check owner
	order, exists := ob.GetOrder(orderID)
	if !exists {
		return nil, ErrOrderNotFound
	}

	// Check if user is owner the order
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
		unlockAsset = pair.Quote
		unlockAmount = cancelledOrder.RemainingAmount() * cancelledOrder.Price
	} else {
		unlockAsset = pair.Base
		unlockAmount = cancelledOrder.RemainingAmount()
	}

	if unlockAmount > 0 {
		err = e.accounts.Unlock(userID, unlockAsset, unlockAmount)
		if err != nil {
			// Log error but don't fail - order is already cancelled
		}
	}

	return cancelledOrder, nil
}

// executeTransfer executes the balance transfer after a match
func (e *Engine) executeTransfer(pair Pair, match orderbook.Match) error {
	buyer := match.Bid.UserID
	seller := match.Ask.UserID
	baseAmount := match.SizeFilled
	quoteAmount := match.SizeFilled * match.Price

	// Seller: debit locked base (BTC), credit quote (BRL)
	if err := e.accounts.DebitLocked(seller, pair.Base, baseAmount); err != nil {
		return fmt.Errorf("seller debit locked failed: %w", err)
	}
	if err := e.accounts.Credit(seller, pair.Quote, quoteAmount); err != nil {
		return fmt.Errorf("seller credit failed: %w", err)
	}

	// Buyer: debit locked quote (BRL), credit base (BTC)
	if err := e.accounts.DebitLocked(buyer, pair.Quote, quoteAmount); err != nil {
		return fmt.Errorf("buyer debit locked failed: %w", err)
	}
	if err := e.accounts.Credit(buyer, pair.Base, baseAmount); err != nil {
		return fmt.Errorf("buyer credit failed: %w", err)
	}

	return nil
}

// =============================================================================
// ORDERBOOK OPERATIONS
// =============================================================================

func (e *Engine) GetOrderbook(pair Pair) *orderbook.Orderbook {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.orderbooks[pair.String()]
}
