package engine

import (
	"testing"

	"github.com/moura95/crypto-exchange-challenge/internal/orderbook"
)

// =============================================================================
// HELPERS
// =============================================================================

func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func assertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func assertEqual(t *testing.T, expected, actual interface{}, msg string) {
	t.Helper()
	if expected != actual {
		t.Errorf("%s: expected %v, got %v", msg, expected, actual)
	}
}

func assertFloat(t *testing.T, expected, actual float64, msg string) {
	t.Helper()
	if expected != actual {
		t.Errorf("%s: expected %.4f, got %.4f", msg, expected, actual)
	}
}

func assertTrue(t *testing.T, condition bool, msg string) {
	t.Helper()
	if !condition {
		t.Errorf("%s: expected true", msg)
	}
}

func assertNil(t *testing.T, actual interface{}, msg string) {
	t.Helper()
	if actual != nil {
		t.Errorf("%s: expected nil, got %v", msg, actual)
	}
}

func btcBrl() Pair {
	return Pair{Base: "BTC", Quote: "BRL"}
}

func setupEngine() *Engine {
	e := NewEngine()
	// Give users some balance
	e.accounts.Credit("1", "BRL", 100_000)
	e.accounts.Credit("1", "BTC", 10)
	e.accounts.Credit("2", "BRL", 100_000)
	e.accounts.Credit("2", "BTC", 10)
	return e
}

// =============================================================================
// PAIR TESTS
// =============================================================================

func TestPair_String(t *testing.T) {
	pair := Pair{Base: "BTC", Quote: "BRL"}
	assertEqual(t, "BTC/BRL", pair.String(), "Pair string")
}

func TestPair_IsValid(t *testing.T) {
	valid := Pair{Base: "BTC", Quote: "BRL"}
	assertTrue(t, valid.IsValid(), "Valid pair")

	invalid1 := Pair{Base: "", Quote: "BRL"}
	assertTrue(t, !invalid1.IsValid(), "Invalid pair (empty base)")

	invalid2 := Pair{Base: "BTC", Quote: ""}
	assertTrue(t, !invalid2.IsValid(), "Invalid pair (empty quote)")
}

// =============================================================================
// ENGINE TESTS
// =============================================================================

func TestNewEngine(t *testing.T) {
	e := NewEngine()
	if e == nil {
		t.Fatal("NewEngine returned nil")
	}
	if e.orderbooks == nil {
		t.Error("orderbooks map should be initialized")
	}
	if e.accounts == nil {
		t.Error("accounts should be initialized")
	}
}

func TestEngine_Credit(t *testing.T) {
	e := NewEngine()

	err := e.accounts.Credit("1", "BTC", 10)
	assertNoError(t, err)

	balance := e.accounts.GetBalance("1", "BTC")
	assertFloat(t, 10, balance.Available, "Balance after credit")
}

func TestEngine_Debit(t *testing.T) {
	e := NewEngine()

	e.accounts.Credit("1", "BTC", 10)
	err := e.accounts.Debit("1", "BTC", 3)
	assertNoError(t, err)

	balance := e.accounts.GetBalance("1", "BTC")
	assertFloat(t, 7, balance.Available, "Balance after debit")
}

func TestEngine_GetAllBalances(t *testing.T) {
	e := NewEngine()

	e.accounts.Credit("1", "BTC", 10)
	e.accounts.Credit("1", "BRL", 50_000)

	balances := e.accounts.GetAllBalances("1")
	assertEqual(t, 2, len(balances), "Number of balances")
	assertFloat(t, 10, balances["BTC"].Available, "BTC balance")
	assertFloat(t, 50_000, balances["BRL"].Available, "BRL balance")
}

// =============================================================================
// PLACE ORDER TESTS
// =============================================================================

func TestEngine_PlaceOrder_NoMatch(t *testing.T) {
	e := setupEngine()

	// UserID:1 places buy order, no sellers
	order, matches, err := e.PlaceOrder("1", btcBrl(), orderbook.Bid, 50_000, 1)
	assertNoError(t, err)

	assertEqual(t, 0, len(matches), "Should have no matches")
	assertEqual(t, orderbook.OrderOpen, order.State, "Order should be open")

	// Balance should be locked
	balance := e.accounts.GetBalance("1", "BRL")
	assertFloat(t, 50_000, balance.Available, "Available after lock")
	assertFloat(t, 50_000, balance.Locked, "Locked after order")
}

func TestEngine_PlaceOrder_FullMatch(t *testing.T) {
	e := setupEngine()

	// UserId:2 places sell order
	_, _, err := e.PlaceOrder("2", btcBrl(), orderbook.Ask, 50_000, 1)
	assertNoError(t, err)

	// UserId:1 places buy order - should match
	order, matches, err := e.PlaceOrder("1", btcBrl(), orderbook.Bid, 50_000, 1)
	assertNoError(t, err)

	assertEqual(t, 1, len(matches), "Should have 1 match")
	assertFloat(t, 1, matches[0].SizeFilled, "Match size")
	assertEqual(t, orderbook.OrderFilled, order.State, "Order should be filled")

	// Check balances after match
	// Alice: paid 50000 BRL, received 1 BTC
	userId1BRL := e.accounts.GetBalance("1", "BRL")
	userID1BTC := e.accounts.GetBalance("1", "BTC")
	assertFloat(t, 50_000, userId1BRL.Available, "UserId:1 BRL after match")
	assertFloat(t, 11, userID1BTC.Available, "UserId:1 BTC after match")

	// Bob: received 50000 BRL, sold 1 BTC
	userID2BRL := e.accounts.GetBalance("2", "BRL")
	userID2BTC := e.accounts.GetBalance("2", "BTC")
	assertFloat(t, 150_000, userID2BRL.Available, "UserId:2 BRL after match")
	assertFloat(t, 9, userID2BTC.Available, "UserId:2 BTC after match")
}

func TestEngine_PlaceOrder_PartialMatch(t *testing.T) {
	e := setupEngine()

	// UserId:2 sells 1 BTC
	_, _, err := e.PlaceOrder("2", btcBrl(), orderbook.Ask, 50_000, 1)
	assertNoError(t, err)

	// UserId:1 wants to buy 2 BTC - only 1 available
	order, matches, err := e.PlaceOrder("1", btcBrl(), orderbook.Bid, 50_000, 2)
	assertNoError(t, err)

	assertEqual(t, 1, len(matches), "Should have 1 match")
	assertFloat(t, 1, matches[0].SizeFilled, "Match size")
	assertEqual(t, orderbook.OrderPartiallyFilled, order.State, "Order should be partially filled")
	assertFloat(t, 1, order.RemainingAmount(), "Remaining amount")

	// UserId:1 BRL: started 100000, locked 100000 for 2 BTC, got back 50000 from selling 1 BTC
	userID1BRL := e.accounts.GetBalance("1", "BRL")
	assertFloat(t, 0, userID1BRL.Available, "UserId:1 BRL available")
	assertFloat(t, 50000, userID1BRL.Locked, "UserId:1 BRL locked for remaining order")
}

func TestEngine_PlaceOrder_InsufficientBalance(t *testing.T) {
	e := NewEngine()
	e.accounts.Credit("1", "BRL", 1_000)

	// Try to buy 1 BTC @ 50000 (needs 50_000 BRL)
	_, _, err := e.PlaceOrder("1", btcBrl(), orderbook.Bid, 50_000, 1)
	assertError(t, err)
}

func TestEngine_PlaceOrder_InvalidPair(t *testing.T) {
	e := setupEngine()

	_, _, err := e.PlaceOrder("1", Pair{}, orderbook.Bid, 50_000, 1)
	assertEqual(t, ErrInvalidPair, err, "Should return invalid pair error")
}

func TestEngine_PlaceOrder_SelfTradePrevention(t *testing.T) {
	e := setupEngine()

	// UserId:1 places sell order
	_, _, err := e.PlaceOrder("1", btcBrl(), orderbook.Ask, 50_000, 1)
	assertNoError(t, err)

	// UserId:1 tries to buy - should NOT match (self-trade prevention)
	order, matches, err := e.PlaceOrder("1", btcBrl(), orderbook.Bid, 50_000, 1)
	assertNoError(t, err)

	assertEqual(t, 0, len(matches), "Should have no matches (self-trade)")
	assertEqual(t, orderbook.OrderOpen, order.State, "Order should be open")

	// Both orders should be in the book
	ob := e.GetOrderbook(btcBrl())
	assertEqual(t, 1, len(ob.Bids()), "Should have 1 bid")
	assertEqual(t, 1, len(ob.Asks()), "Should have 1 ask")
}

// =============================================================================
// CANCEL ORDER TESTS
// =============================================================================

func TestEngine_CancelOrder(t *testing.T) {
	e := setupEngine()

	// UserId:1 places order
	order, _, err := e.PlaceOrder("1", btcBrl(), orderbook.Bid, 50_000, 1)
	assertNoError(t, err)

	// Check balance is locked
	balanceBefore := e.accounts.GetBalance("1", "BRL")
	assertFloat(t, 50_000, balanceBefore.Locked, "Should be locked")

	// Cancel order
	cancelled, err := e.CancelOrder("1", btcBrl(), order.ID)
	assertNoError(t, err)

	assertEqual(t, orderbook.OrderCancelled, cancelled.State, "Should be cancelled")

	// Balance should be unlocked
	balanceAfter := e.accounts.GetBalance("1", "BRL")
	assertFloat(t, 100_000, balanceAfter.Available, "Available after cancel")
	assertFloat(t, 0, balanceAfter.Locked, "Locked after cancel")
}

func TestEngine_CancelOrder_NotFound(t *testing.T) {
	e := setupEngine()

	_, err := e.CancelOrder("1", btcBrl(), 99999)
	assertEqual(t, ErrOrderNotFound, err, "Should return not found error")
}

func TestEngine_CancelOrder_Unauthorized(t *testing.T) {
	e := setupEngine()

	// UserId:1 place order
	order, _, err := e.PlaceOrder("1", btcBrl(), orderbook.Bid, 50_000, 1)
	assertNoError(t, err)

	// UserId:2 try to cancel UserId:1 order
	_, err = e.CancelOrder("2", btcBrl(), order.ID)
	assertEqual(t, ErrUnauthorized, err, "Should return unauthorized error")
}

func TestEngine_CancelOrder_PartiallyFilled(t *testing.T) {
	e := setupEngine()

	// UserID:1 sell 1 BTC
	_, _, err := e.PlaceOrder("1", btcBrl(), orderbook.Ask, 50_000, 1)
	assertNoError(t, err)

	// UserID:2 buy 2 BTC - partial fill (1 matched, 1 remaining)
	order, _, err := e.PlaceOrder("2", btcBrl(), orderbook.Bid, 50_000, 2)
	assertNoError(t, err)

	// Cancel remaining order
	cancelled, err := e.CancelOrder("2", btcBrl(), order.ID)
	assertNoError(t, err)

	assertEqual(t, orderbook.OrderCancelled, cancelled.State, "Should be cancelled")
	assertFloat(t, 1, cancelled.FilledAmount, "Filled amount preserved")

	// Only the remaining locked amount should be unlocked
	balance := e.accounts.GetBalance("2", "BRL")
	assertFloat(t, 50000, balance.Available, "Available after cancel")
	assertFloat(t, 0, balance.Locked, "Locked after cancel")
}

func TestEngine_PriceTimePriority(t *testing.T) {
	e := setupEngine()
	e.accounts.Credit("3", "BTC", 10)

	// UserID:1 sells 1 BTC @ 50000 (first)
	_, _, err := e.PlaceOrder("1", btcBrl(), orderbook.Ask, 50_000, 1)
	assertNoError(t, err)

	// UserID:3 sells 1 BTC @ 50000 (second, same price)
	_, _, err = e.PlaceOrder("3", btcBrl(), orderbook.Ask, 50_000, 1)
	assertNoError(t, err)

	// UserID:2 buys 1 BTC - should match with UserID:1 (FIFO)
	_, matches, err := e.PlaceOrder("2", btcBrl(), orderbook.Bid, 50_000, 1)
	assertNoError(t, err)

	assertEqual(t, 1, len(matches), "Should have 1 match")
	assertEqual(t, "1", matches[0].Ask.UserID, "Should match with UserID:1 (FIFO)")
}
