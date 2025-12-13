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

func (e *Engine) PlaceOrder(userID string, pair Pair, side orderbook.Side, price, amount float64) (*orderbook.Order, []orderbook.Match, error) {

	// 1. Basic validation
	if !pair.IsValid() {
		return nil, nil, ErrInvalidPair
	}

	// Normalize and validate price
	price = utils.FloorToTick(price, PriceTick)
	if !utils.IsValidTick(price, PriceTick) {
		return nil, nil, ErrInvalidPriceTick
	}

	// Normalize and validate amount
	amount = utils.FloorToTick(amount, AmountTick)
	if !utils.IsValidTick(amount, AmountTick) {
		return nil, nil, ErrInvalidAmountTick
	}

	// 2. Create order
	order, err := orderbook.NewOrder(userID, side, price, amount)
	if err != nil {
		return nil, nil, err
	}

	// 3. Decide which asset and how much to lock
	var lockAsset string
	var lockAmount float64

	if side == orderbook.Bid {
		// BUY: lock quote currency (BRL)
		lockAsset = pair.Quote
		lockAmount = order.Price * order.Amount
	} else {
		// SELL: lock base currency (BTC)
		lockAsset = pair.Base
		lockAmount = order.Amount
	}

	// Lock funds
	if err := e.accounts.Lock(userID, lockAsset, lockAmount); err != nil {
		return nil, nil, err
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	ob := e.getOrCreateOrderbook(pair)

	// Place order and try to match
	matches := ob.PlaceLimitOrder(order)

	// 5. Execute balance transfers for each match
	for _, match := range matches {
		if err := e.executeTransfer(pair, match); err != nil {
			// Best-effort: unlock the initial lock so user won't get stuck
			_ = e.accounts.Unlock(userID, lockAsset, lockAmount)
			return nil, nil, fmt.Errorf("transfer failed: %w", err)
		}
	}

	// 6. Refund price improvement for BUY orders
	if err := e.refundBidDifference(userID, pair, order, matches); err != nil {
		// Best-effort: unlock the initial lock so user won't get stuck
		_ = e.accounts.Unlock(userID, lockAsset, lockAmount)
		return nil, nil, fmt.Errorf("refund failed: %w", err)
	}

	return order, matches, nil
}

// CancelOrder cancels an order and unlocks the reserved balance
func (e *Engine) CancelOrder(userID string, pair Pair, orderID int64) (*orderbook.Order, error) {
	if !pair.IsValid() {
		return nil, ErrInvalidPair
	}

	// Use a single critical section to avoid races with PlaceOrder/matching.
	e.mu.Lock()
	defer e.mu.Unlock()

	ob, exists := e.orderbooks[pair.String()]
	if !exists {
		return nil, ErrOrderNotFound
	}

	// Get order to check owner
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
		unlockAsset = pair.Quote
		unlockAmount = cancelledOrder.RemainingAmount() * cancelledOrder.Price
	} else {
		unlockAsset = pair.Base
		unlockAmount = cancelledOrder.RemainingAmount()
	}

	if unlockAmount > 0 {
		if err := e.accounts.Unlock(userID, unlockAsset, unlockAmount); err != nil {
			// For the challenge: fail-fast so we don't hide inconsistencies
			return nil, err
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

func (e *Engine) refundBidDifference(userID string, pair Pair, order *orderbook.Order, matches []orderbook.Match) error {
	// Refund applies only to BUY orders (BID)
	// and only when at least one match happened
	if order.Side != orderbook.Bid || len(matches) == 0 {
		return nil
	}

	// 1. Calculate how much money was really spent
	executedQuote := 0.0
	for _, m := range matches {
		executedQuote += m.SizeFilled * m.Price
	}

	// 2. Amount locked when the order was created
	initialLock := order.Price * order.Amount

	// 3. Amount that must stay locked for the remaining order
	stillLocked := order.Price * order.RemainingAmount()

	// 4. Money that must be returned to the user
	refund := initialLock - executedQuote - stillLocked

	// 5. Avoid unlocking very small values caused by float errors.

	const minRefundBRL = 0.01

	if refund >= minRefundBRL {
		if err := e.accounts.Unlock(userID, pair.Quote, refund); err != nil {
			return err
		}
	}

	return nil
}
