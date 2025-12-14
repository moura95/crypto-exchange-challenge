package engine

import (
	"sync"
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
	// comparação direta funciona aqui porque você usa ticks fixos (0.01 / 1e-8) e valores “redondos”
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

func assertFalse(t *testing.T, condition bool, msg string) {
	t.Helper()
	if condition {
		t.Errorf("%s: expected false", msg)
	}
}

func btcBrl() Pair {
	return Pair{Base: "BTC", Quote: "BRL"}
}

func setupEngine() *Engine {
	e := NewEngine()
	// Give users some balance
	_ = e.accounts.Credit("1", "BRL", 100_000)
	_ = e.accounts.Credit("1", "BTC", 10)
	_ = e.accounts.Credit("2", "BRL", 100_000)
	_ = e.accounts.Credit("2", "BTC", 10)
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
	assertFalse(t, invalid1.IsValid(), "Invalid pair (empty base)")

	invalid2 := Pair{Base: "BTC", Quote: ""}
	assertFalse(t, invalid2.IsValid(), "Invalid pair (empty quote)")

	invalid3 := Pair{Base: "BTC", Quote: "USD"}
	assertFalse(t, invalid3.IsValid(), "Invalid pair (quote must be BRL)")
}

// =============================================================================
// ENGINE BASIC TESTS
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

	_ = e.accounts.Credit("1", "BTC", 10)
	err := e.accounts.Debit("1", "BTC", 3)
	assertNoError(t, err)

	balance := e.accounts.GetBalance("1", "BTC")
	assertFloat(t, 7, balance.Available, "Balance after debit")
}

func TestEngine_GetAllBalances(t *testing.T) {
	e := NewEngine()

	_ = e.accounts.Credit("1", "BTC", 10)
	_ = e.accounts.Credit("1", "BRL", 50_000)

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
	// Buyer: paid 50000 BRL, received 1 BTC
	userId1BRL := e.accounts.GetBalance("1", "BRL")
	userID1BTC := e.accounts.GetBalance("1", "BTC")
	assertFloat(t, 50_000, userId1BRL.Available, "UserId:1 BRL after match")
	assertFloat(t, 11, userID1BTC.Available, "UserId:1 BTC after match")

	// Seller: received 50000 BRL, sold 1 BTC
	userID2BRL := e.accounts.GetBalance("2", "BRL")
	userID2BTC := e.accounts.GetBalance("2", "BTC")
	assertFloat(t, 150_000, userID2BRL.Available, "UserId:2 BRL after match")
	assertFloat(t, 9, userID2BTC.Available, "UserId:2 BTC after match")

	// locked deve estar limpo para ambos nesse cenário
	assertFloat(t, 0, userId1BRL.Locked, "Buyer BRL locked should be 0 after full fill")
	assertFloat(t, 0, userID2BTC.Locked, "Seller BTC locked should be 0 after full fill")
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

	// UserId:1 BRL: locked 100000 initially; spent 50000; should remain 50000 locked for remaining 1 BTC @ 50000
	userID1BRL := e.accounts.GetBalance("1", "BRL")
	assertFloat(t, 0, userID1BRL.Available, "UserId:1 BRL available")
	assertFloat(t, 50_000, userID1BRL.Locked, "UserId:1 BRL locked for remaining order")
}

func TestEngine_PlaceOrder_InsufficientBalance(t *testing.T) {
	e := NewEngine()
	_ = e.accounts.Credit("1", "BRL", 1_000)

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
	assertFloat(t, 50_000, balance.Available, "Available after cancel")
	assertFloat(t, 0, balance.Locked, "Locked after cancel")
}

// =============================================================================
// PRICE/TIME PRIORITY (FIFO)
// =============================================================================

func TestEngine_PriceTimePriority(t *testing.T) {
	e := setupEngine()
	_ = e.accounts.Credit("3", "BTC", 10)

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

func TestEngine_PlaceOrder_BuyPriceImprovement_ShouldRefundDifference(t *testing.T) {
	e := setupEngine()

	// Seller places ask @ 49k
	_, _, err := e.PlaceOrder("2", btcBrl(), orderbook.Ask, 49_000, 1)
	assertNoError(t, err)

	// Buyer places bid @ 50k (should execute at 49k and refund 1k)
	order, matches, err := e.PlaceOrder("1", btcBrl(), orderbook.Bid, 50_000, 1)
	assertNoError(t, err)

	assertEqual(t, 1, len(matches), "Should have 1 match")
	assertEqual(t, orderbook.OrderFilled, order.State, "Order should be filled")
	assertFloat(t, 49_000, matches[0].Price, "Execution price should be best ask (price improvement)")

	// Buyer started with 100k BRL.
	// If executed at 49k: Available should be 51k, Locked should be 0.
	buyerBRL := e.accounts.GetBalance("1", "BRL")
	assertFloat(t, 51_000, buyerBRL.Available, "Buyer BRL available after price improvement trade")
	assertFloat(t, 0, buyerBRL.Locked, "Buyer BRL locked should be 0 after fully filled")

	// Buyer BTC should increase by 1 (started with 10)
	buyerBTC := e.accounts.GetBalance("1", "BTC")
	assertFloat(t, 11, buyerBTC.Available, "Buyer BTC after trade")

	// Seller receives 49k BRL, and loses 1 BTC
	sellerBRL := e.accounts.GetBalance("2", "BRL")
	sellerBTC := e.accounts.GetBalance("2", "BTC")
	assertFloat(t, 149_000, sellerBRL.Available, "Seller BRL after trade")
	assertFloat(t, 9, sellerBTC.Available, "Seller BTC after trade")
}

func TestEngine_CancelOrder_Twice_ShouldReturnNotFound(t *testing.T) {
	e := setupEngine()

	// Place an order that stays open
	order, _, err := e.PlaceOrder("1", btcBrl(), orderbook.Bid, 50_000, 1)
	assertNoError(t, err)

	// First cancel -> ok
	_, err = e.CancelOrder("1", btcBrl(), order.ID)
	assertNoError(t, err)

	// Second cancel -> must be not found
	_, err = e.CancelOrder("1", btcBrl(), order.ID)
	assertEqual(t, ErrOrderNotFound, err, "Second cancel should return not found")
}

func TestEngine_ConcurrentPlaceOrders(t *testing.T) {
	e := setupEngine()

	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			user := "1"
			if id%2 == 0 {
				user = "2"
			}
			_, _, _ = e.PlaceOrder(user, btcBrl(), orderbook.Bid, 50_000, 0.01)
		}(i)
	}

	wg.Wait()
}

func TestEngine_PlaceOrder_BuyPartialFill_WithPriceImprovement_ShouldRefundAndKeepCorrectLocked(t *testing.T) {
	e := setupEngine()

	// User 2 places ASK: 0.5 BTC @ 49,000
	_, _, err := e.PlaceOrder("2", btcBrl(), orderbook.Ask, 49_000, 0.5)
	assertNoError(t, err)

	// User 1 places BID limit: 1 BTC @ 50,000
	order, matches, err := e.PlaceOrder("1", btcBrl(), orderbook.Bid, 50_000, 1.0)
	assertNoError(t, err)

	// Should match only 0.5 BTC (because only 0.5 is available)
	assertEqual(t, 1, len(matches), "Should have 1 match")
	assertFloat(t, 0.5, matches[0].SizeFilled, "Filled size")
	assertFloat(t, 49_000, matches[0].Price, "Executed price (price improvement)")

	// Order should be partially filled
	assertEqual(t, orderbook.OrderPartiallyFilled, order.State, "Order state")
	assertFloat(t, 0.5, order.RemainingAmount(), "Remaining amount")

	// Buyer balances:
	// Initial BRL: 100,000
	// Lock at start: 50,000 => Available 50,000 / Locked 50,000
	// Executed quote: 0.5 * 49,000 = 24,500 (debited from locked)
	// Still locked needed: 0.5 * 50,000 = 25,000
	// Refund: 50,000 - 24,500 - 25,000 = 500
	// Final: Available 50,500 / Locked 25,000
	buyerBRL := e.accounts.GetBalance("1", "BRL")
	assertFloat(t, 50_500, buyerBRL.Available, "Buyer BRL available after refund")
	assertFloat(t, 25_000, buyerBRL.Locked, "Buyer BRL locked for remaining order")

	buyerBTC := e.accounts.GetBalance("1", "BTC")
	assertFloat(t, 10.5, buyerBTC.Available, "Buyer BTC after partial fill")

	// Seller balances:
	// Seller sold 0.5 BTC and received 24,500 BRL
	sellerBRL := e.accounts.GetBalance("2", "BRL")
	assertFloat(t, 124_500, sellerBRL.Available, "Seller BRL after trade")

	sellerBTC := e.accounts.GetBalance("2", "BTC")
	assertFloat(t, 9.5, sellerBTC.Available, "Seller BTC after trade")
}

func TestEngine_CancelOrder_AfterBuyPartialFill_SamePrice_ShouldUnlockRemaining(t *testing.T) {
	e := setupEngine()

	// User 2 places ASK: 0.5 BTC @ 50,000
	_, _, err := e.PlaceOrder("2", btcBrl(), orderbook.Ask, 50_000, 0.5)
	assertNoError(t, err)

	// User 1 places BID: 1 BTC @ 50,000 (will partially fill)
	order, matches, err := e.PlaceOrder("1", btcBrl(), orderbook.Bid, 50_000, 1.0)
	assertNoError(t, err)

	// One partial match
	assertEqual(t, 1, len(matches), "Should have 1 match")
	assertEqual(t, orderbook.OrderPartiallyFilled, order.State, "Order should be partially filled")
	assertFloat(t, 0.5, order.RemainingAmount(), "Remaining amount should be 0.5")

	// Locked before cancel should be exactly for remaining amount
	buyerBRLBeforeCancel := e.accounts.GetBalance("1", "BRL")
	assertFloat(t, 25_000, buyerBRLBeforeCancel.Locked, "Buyer BRL locked before cancel")

	// Cancel remaining order
	cancelled, err := e.CancelOrder("1", btcBrl(), order.ID)
	assertNoError(t, err)

	assertEqual(t, orderbook.OrderCancelled, cancelled.State, "Order should be cancelled")
	assertFloat(t, 0.5, cancelled.FilledAmount, "Filled amount should be preserved")

	// After cancel:
	// Initial BRL: 100,000
	// Spent: 25,000
	// Remaining 25,000 must be unlocked
	buyerBRLAfterCancel := e.accounts.GetBalance("1", "BRL")
	assertFloat(t, 75_000, buyerBRLAfterCancel.Available, "Buyer BRL available after cancel")
	assertFloat(t, 0, buyerBRLAfterCancel.Locked, "Buyer BRL locked should be 0")

	// Buyer BTC should have +0.5 from the trade
	buyerBTC := e.accounts.GetBalance("1", "BTC")
	assertFloat(t, 10.5, buyerBTC.Available, "Buyer BTC after partial fill and cancel")
}

// =============================================================================
// MARKET ORDER TESTS
// =============================================================================

func TestEngine_PlaceMarketOrder_Buy_FullFill(t *testing.T) {
	e := setupEngine()

	// Setup: User 2 places two ASK orders
	_, _, err := e.PlaceOrder("2", btcBrl(), orderbook.Ask, 50_000, 0.5)
	assertNoError(t, err)

	_, _, err = e.PlaceOrder("2", btcBrl(), orderbook.Ask, 50_100, 0.5)
	assertNoError(t, err)

	// User 1 places MARKET BUY for 1 BTC (should consume both asks)
	order, matches, err := e.PlaceMarketOrder("1", btcBrl(), orderbook.Bid, 1.0)
	assertNoError(t, err)

	// Should have 2 matches
	assertEqual(t, 2, len(matches), "Should have 2 matches")
	assertEqual(t, orderbook.OrderFilled, order.State, "Order should be filled")
	assertFloat(t, 1.0, order.FilledAmount, "Filled amount")

	// First match @ 50,000 (best price)
	assertFloat(t, 50_000, matches[0].Price, "First match price")
	assertFloat(t, 0.5, matches[0].SizeFilled, "First match size")

	// Second match @ 50,100
	assertFloat(t, 50_100, matches[1].Price, "Second match price")
	assertFloat(t, 0.5, matches[1].SizeFilled, "Second match size")

	// Buyer balances: spent (0.5*50k + 0.5*50.1k) = 50,050
	buyerBRL := e.accounts.GetBalance("1", "BRL")
	buyerBTC := e.accounts.GetBalance("1", "BTC")
	assertFloat(t, 49_950, buyerBRL.Available, "Buyer BRL after trade") // 100k - 50,050
	assertFloat(t, 0, buyerBRL.Locked, "Buyer BRL locked should be 0")
	assertFloat(t, 11, buyerBTC.Available, "Buyer BTC after trade") // 10 + 1

	// Seller balances
	sellerBRL := e.accounts.GetBalance("2", "BRL")
	sellerBTC := e.accounts.GetBalance("2", "BTC")
	assertFloat(t, 150_050, sellerBRL.Available, "Seller BRL after trade") // 100k + 50,050
	assertFloat(t, 9, sellerBTC.Available, "Seller BTC after trade")       // 10 - 1
}

func TestEngine_PlaceMarketOrder_Sell_FullFill(t *testing.T) {
	e := setupEngine()

	// Setup: User 1 places two BID orders
	_, _, err := e.PlaceOrder("1", btcBrl(), orderbook.Bid, 50_200, 0.6)
	assertNoError(t, err)

	_, _, err = e.PlaceOrder("1", btcBrl(), orderbook.Bid, 50_100, 0.4)
	assertNoError(t, err)

	// User 2 places MARKET SELL for 1 BTC (should consume both bids)
	order, matches, err := e.PlaceMarketOrder("2", btcBrl(), orderbook.Ask, 1.0)
	assertNoError(t, err)

	// Should have 2 matches
	assertEqual(t, 2, len(matches), "Should have 2 matches")
	assertEqual(t, orderbook.OrderFilled, order.State, "Order should be filled")
	assertFloat(t, 1.0, order.FilledAmount, "Filled amount")

	// First match @ 50,200 (best bid)
	assertFloat(t, 50_200, matches[0].Price, "First match price")
	assertFloat(t, 0.6, matches[0].SizeFilled, "First match size")

	// Second match @ 50,100
	assertFloat(t, 50_100, matches[1].Price, "Second match price")
	assertFloat(t, 0.4, matches[1].SizeFilled, "Second match size")

	// Seller balances: received (0.6*50,200 + 0.4*50,100) = 50,160
	sellerBRL := e.accounts.GetBalance("2", "BRL")
	sellerBTC := e.accounts.GetBalance("2", "BTC")
	assertFloat(t, 150_160, sellerBRL.Available, "Seller BRL after trade") // 100k + 50,160
	assertFloat(t, 0, sellerBRL.Locked, "Seller BRL locked should be 0")
	assertFloat(t, 9, sellerBTC.Available, "Seller BTC after trade") // 10 - 1

	// Buyer balances
	buyerBRL := e.accounts.GetBalance("1", "BRL")
	buyerBTC := e.accounts.GetBalance("1", "BTC")
	assertFloat(t, 49_840, buyerBRL.Available, "Buyer BRL after trade") // 100k - 50,160
	assertFloat(t, 11, buyerBTC.Available, "Buyer BTC after trade")     // 10 + 1
}

func TestEngine_PlaceMarketOrder_Buy_PartialFill_InsufficientLiquidity(t *testing.T) {
	e := setupEngine()

	// Setup: Only 0.5 BTC available on asks
	_, _, err := e.PlaceOrder("2", btcBrl(), orderbook.Ask, 50_000, 0.5)
	assertNoError(t, err)

	// User 1 tries to buy 2 BTC (market) - not enough liquidity
	_, _, err = e.PlaceMarketOrder("1", btcBrl(), orderbook.Bid, 2.0)
	assertError(t, err)

	// Error should be about insufficient liquidity
	assertEqual(t, "insufficient liquidity for market order", err.Error(), "Error message")

	// Balance should not change (order rejected before locking)
	buyerBRL := e.accounts.GetBalance("1", "BRL")
	assertFloat(t, 100_000, buyerBRL.Available, "Buyer BRL should not change")
	assertFloat(t, 0, buyerBRL.Locked, "Buyer BRL locked should be 0")
}

func TestEngine_PlaceMarketOrder_Sell_InsufficientLiquidity(t *testing.T) {
	e := setupEngine()

	// Setup: Only 0.5 BTC worth of bids available
	_, _, err := e.PlaceOrder("1", btcBrl(), orderbook.Bid, 50_000, 0.5)
	assertNoError(t, err)

	// User 2 tries to sell 2 BTC (market) - not enough liquidity
	_, _, err = e.PlaceMarketOrder("2", btcBrl(), orderbook.Ask, 2.0)
	assertError(t, err)

	assertEqual(t, "insufficient liquidity for market order", err.Error(), "Error message")

	// Balance should not change
	sellerBTC := e.accounts.GetBalance("2", "BTC")
	assertFloat(t, 10, sellerBTC.Available, "Seller BTC should not change")
	assertFloat(t, 0, sellerBTC.Locked, "Seller BTC locked should be 0")
}

func TestEngine_PlaceMarketOrder_Buy_InsufficientBalance(t *testing.T) {
	e := NewEngine()

	// User with only 1,000 BRL
	_ = e.accounts.Credit("1", "BRL", 1_000)

	// Setup: ASK @ 50,000 for 1 BTC
	_ = e.accounts.Credit("2", "BTC", 10)
	_, _, _ = e.PlaceOrder("2", btcBrl(), orderbook.Ask, 50_000, 1.0)

	// User 1 tries to market buy 1 BTC (needs ~50,250 but has only 1,000)
	_, _, err := e.PlaceMarketOrder("1", btcBrl(), orderbook.Bid, 1.0)
	assertError(t, err)

	// Should fail on Lock (insufficient balance)
	buyerBRL := e.accounts.GetBalance("1", "BRL")
	assertFloat(t, 1_000, buyerBRL.Available, "Balance should not change")
}

func TestEngine_PlaceMarketOrder_Sell_InsufficientBalance(t *testing.T) {
	e := NewEngine()

	// User with only 0.1 BTC
	_ = e.accounts.Credit("2", "BTC", 0.1)

	// Setup: BID @ 50,000 for 1 BTC
	_ = e.accounts.Credit("1", "BRL", 100_000)
	_, _, _ = e.PlaceOrder("1", btcBrl(), orderbook.Bid, 50_000, 1.0)

	// User 2 tries to market sell 1 BTC (has only 0.1)
	_, _, err := e.PlaceMarketOrder("2", btcBrl(), orderbook.Ask, 1.0)
	assertError(t, err)

	sellerBTC := e.accounts.GetBalance("2", "BTC")
	assertFloat(t, 0.1, sellerBTC.Available, "Balance should not change")
}

func TestEngine_PlaceMarketOrder_Buy_EmptyBook(t *testing.T) {
	e := setupEngine()

	// No asks in the book
	_, _, err := e.PlaceMarketOrder("1", btcBrl(), orderbook.Bid, 1.0)
	assertError(t, err)

	assertEqual(t, "insufficient liquidity for market order", err.Error(), "Error message")
}

func TestEngine_PlaceMarketOrder_Sell_EmptyBook(t *testing.T) {
	e := setupEngine()

	// No bids in the book
	_, _, err := e.PlaceMarketOrder("2", btcBrl(), orderbook.Ask, 1.0)
	assertError(t, err)

	assertEqual(t, "insufficient liquidity for market order", err.Error(), "Error message")
}

func TestEngine_PlaceMarketOrder_InvalidPair(t *testing.T) {
	e := setupEngine()

	_, _, err := e.PlaceMarketOrder("1", Pair{}, orderbook.Bid, 1.0)
	assertEqual(t, ErrInvalidPair, err, "Should return invalid pair error")
}

func TestEngine_PlaceMarketOrder_InvalidAmount(t *testing.T) {
	e := setupEngine()

	_, _, err := e.PlaceMarketOrder("1", btcBrl(), orderbook.Bid, 0)
	assertError(t, err)

	_, _, err = e.PlaceMarketOrder("1", btcBrl(), orderbook.Bid, -1)
	assertError(t, err)
}
