package orderbook

import (
	"testing"
	"time"

	"github.com/moura95/crypto-exchange-challenge/pkg/utils"
)

// =============================================================================
// TEST HELPERS
// =============================================================================

func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
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
		t.Errorf("%s: expected true, got false", msg)
	}
}

func assertFalse(t *testing.T, condition bool, msg string) {
	t.Helper()
	if condition {
		t.Errorf("%s: expected false, got true", msg)
	}
}

// Helper to convert price to ticks (same logic as engine)
func priceToTicks(price float64) int64 {
	const priceTick = 0.01
	return utils.PriceToTicks(price, priceTick)
}

// =============================================================================
// ORDER TESTS
// =============================================================================

func TestNewOrder_Valid(t *testing.T) {
	order, err := NewOrder("1", Bid, 50_000, 1.0)
	assertNoError(t, err)

	assertEqual(t, "1", order.UserID, "UserID")
	assertEqual(t, Bid, order.Side, "Side")
	assertFloat(t, 50_000, order.Price, "Price")
	assertFloat(t, 1.0, order.Amount, "Amount")
	assertFloat(t, 0.0, order.FilledAmount, "FilledAmount")
	assertEqual(t, OrderOpen, order.State, "State")
	assertTrue(t, order.ID > 0, "ID should be positive")
}

func TestNewOrder_InvalidPrice(t *testing.T) {
	_, err := NewOrder("1", Bid, 0, 1.0)
	if err != ErrInvalidPrice {
		t.Errorf("expected ErrInvalidPrice, got %v", err)
	}

	_, err = NewOrder("1", Bid, -100, 1.0)
	if err != ErrInvalidPrice {
		t.Errorf("expected ErrInvalidPrice, got %v", err)
	}
}

func TestNewOrder_InvalidAmount(t *testing.T) {
	_, err := NewOrder("1", Bid, 50_000, 0)
	if err != ErrInvalidAmount {
		t.Errorf("expected ErrInvalidAmount, got %v", err)
	}

	_, err = NewOrder("1", Bid, 50_000, -1.0)
	if err != ErrInvalidAmount {
		t.Errorf("expected ErrInvalidAmount, got %v", err)
	}
}

func TestNewOrder_InvalidUserID(t *testing.T) {
	_, err := NewOrder("", Bid, 50_000, 1.0)
	if err == nil {
		t.Error("expected error for empty userID")
	}
}

func TestNewOrder_InvalidSide(t *testing.T) {
	_, err := NewOrder("1", Side("invalid"), 50_000, 1.0)
	if err != ErrInvalidSide {
		t.Errorf("expected ErrInvalidSide, got %v", err)
	}
}

func TestOrder_IsFilled(t *testing.T) {
	order, err := NewOrder("1", Bid, 50_000, 1.0)
	assertNoError(t, err)

	assertFalse(t, order.IsFilled(), "New order should not be filled")

	order.FilledAmount = 0.5
	order.State = OrderPartiallyFilled
	assertFalse(t, order.IsFilled(), "Partially filled order should not be filled")
	assertEqual(t, order.State, OrderPartiallyFilled, "Order should be partially_filled")

	order.FilledAmount = 1.0
	order.State = OrderFilled
	assertTrue(t, order.IsFilled(), "Fully filled order should be filled")
	assertEqual(t, order.State, OrderFilled, "Order should be filled")
}

func TestOrder_RemainingAmount(t *testing.T) {
	order, err := NewOrder("1", Bid, 50_000, 2.0)
	assertNoError(t, err)

	assertFloat(t, 2.0, order.RemainingAmount(), "Initial remaining")

	order.FilledAmount = 0.5
	assertFloat(t, 1.5, order.RemainingAmount(), "After partial fill")

	order.FilledAmount = 2.0
	assertFloat(t, 0.0, order.RemainingAmount(), "After full fill")
}

// =============================================================================
// LIMIT TESTS
// =============================================================================

func TestLimit_AddOrder(t *testing.T) {
	// Create a limit at price level 50000 (5000000 ticks)
	limit := NewLimit(priceToTicks(50_000))

	order1, err := NewOrder("1", Bid, 50_000, 1.0)
	assertNoError(t, err)

	order2, err := NewOrder("2", Bid, 50_000, 2.0)
	assertNoError(t, err)

	limit.AddOrder(order1)
	assertEqual(t, 1, len(limit.Orders), "Orders count after first add")
	assertFloat(t, 1.0, limit.TotalVolume, "TotalVolume after first add")
	assertEqual(t, limit, order1.Limit, "Order.Limit should reference the limit")

	limit.AddOrder(order2)
	assertEqual(t, 2, len(limit.Orders), "Orders count after second add")
	assertFloat(t, 3.0, limit.TotalVolume, "TotalVolume after second add")
}

func TestLimit_DeleteOrder(t *testing.T) {
	limit := NewLimit(priceToTicks(50_000))

	order1, err := NewOrder("1", Bid, 50_000, 1.0)
	assertNoError(t, err)

	order2, err := NewOrder("2", Bid, 50_000, 2.0)
	assertNoError(t, err)

	limit.AddOrder(order1)
	limit.AddOrder(order2)

	limit.DeleteOrder(order1)
	assertEqual(t, 1, len(limit.Orders), "Orders count after delete")
	assertFloat(t, 2.0, limit.TotalVolume, "TotalVolume after delete")
	assertEqual(t, order2.ID, limit.Orders[0].ID, "Remaining order should be order2")
}

func TestLimit_Fill_FullMatch(t *testing.T) {
	limit := NewLimit(priceToTicks(50_000))

	// Seller puts an ask order
	askOrder, err := NewOrder("1", Ask, 50_000, 1.0)
	assertNoError(t, err)
	limit.AddOrder(askOrder)

	// Buyer comes in with matching bid
	bidOrder, err := NewOrder("2", Bid, 50_000, 1.0)
	assertNoError(t, err)

	matches := limit.Fill(bidOrder)

	assertEqual(t, 1, len(matches), "Should have 1 match")
	assertFloat(t, 1.0, matches[0].SizeFilled, "Match size")
	assertFloat(t, 50_000.0, matches[0].Price, "Match price")

	assertTrue(t, askOrder.IsFilled(), "Ask order should be filled")
	assertTrue(t, bidOrder.IsFilled(), "Bid order should be filled")
	assertEqual(t, OrderFilled, askOrder.State, "Ask state")
	assertEqual(t, OrderFilled, bidOrder.State, "Bid state")

	// Limit should be empty now - both orders fully matched
	assertEqual(t, 0, len(limit.Orders), "Limit should be empty")
	assertFloat(t, 0.0, limit.TotalVolume, "TotalVolume should be 0")
}

func TestLimit_Fill_PartialMatch_IncomingLarger(t *testing.T) {
	limit := NewLimit(priceToTicks(50_000))

	// Small ask available
	askOrder, err := NewOrder("1", Ask, 50_000, 1.0)
	assertNoError(t, err)
	limit.AddOrder(askOrder)

	// Large bid comes in - can only fill partially
	bidOrder, err := NewOrder("2", Bid, 50_000, 2.0)
	assertNoError(t, err)

	matches := limit.Fill(bidOrder)

	assertEqual(t, 1, len(matches), "Should have 1 match")
	assertFloat(t, 1.0, matches[0].SizeFilled, "Match size")

	assertTrue(t, askOrder.IsFilled(), "Ask order should be filled")
	assertFalse(t, bidOrder.IsFilled(), "Bid order should NOT be filled")
	assertFloat(t, 1.0, bidOrder.RemainingAmount(), "Bid remaining")
	assertEqual(t, OrderPartiallyFilled, bidOrder.State, "Bid state")

	assertEqual(t, 0, len(limit.Orders), "Limit should be empty (ask removed)")
}

func TestLimit_Fill_PartialMatch(t *testing.T) {
	limit := NewLimit(priceToTicks(50_000))

	// Large ask available
	askOrder, err := NewOrder("1", Ask, 50_000, 2.0)
	assertNoError(t, err)
	limit.AddOrder(askOrder)

	// Small bid comes in
	bidOrder, err := NewOrder("2", Bid, 50_000, 1.0)
	assertNoError(t, err)

	matches := limit.Fill(bidOrder)

	assertEqual(t, 1, len(matches), "Should have 1 match")
	assertFloat(t, 1.0, matches[0].SizeFilled, "Match size")

	assertFalse(t, askOrder.IsFilled(), "Ask order should NOT be filled")
	assertTrue(t, bidOrder.IsFilled(), "Bid order should be filled")
	assertFloat(t, 1.0, askOrder.RemainingAmount(), "Ask remaining")
	assertEqual(t, OrderPartiallyFilled, askOrder.State, "Ask state")

	assertEqual(t, 1, len(limit.Orders), "Limit should still have ask")
	assertFloat(t, 1.0, limit.TotalVolume, "TotalVolume should be 1")
}

func TestLimit_Fill_MultipleOrders_FIFO(t *testing.T) {
	limit := NewLimit(priceToTicks(50_000))

	// First seller arrives
	ask1, err := NewOrder("1", Ask, 50_000, 1.0)
	assertNoError(t, err)

	time.Sleep(1 * time.Millisecond)

	// Second seller arrives at same price
	ask2, err := NewOrder("2", Ask, 50_000, 1.0)
	assertNoError(t, err)

	limit.AddOrder(ask1)
	limit.AddOrder(ask2)

	// Buyer wants 1.5 BTC - should match with ask1 first (FIFO)
	bidOrder, err := NewOrder("3", Bid, 50_000, 1.5)
	assertNoError(t, err)

	matches := limit.Fill(bidOrder)

	assertEqual(t, 2, len(matches), "Should have 2 matches")

	// First match should be with ask1 (FIFO - first in, first out)
	assertFloat(t, 1.0, matches[0].SizeFilled, "First match size")
	assertEqual(t, "1", matches[0].Ask.UserID, "First match should be with user 1 (FIFO)")

	// Second match with ask2
	assertFloat(t, 0.5, matches[1].SizeFilled, "Second match size")
	assertEqual(t, "2", matches[1].Ask.UserID, "Second match should be with user 2")

	assertTrue(t, ask1.IsFilled(), "Ask1 should be filled")
	assertFalse(t, ask2.IsFilled(), "Ask2 should NOT be filled")
	assertFloat(t, 0.5, ask2.RemainingAmount(), "Ask2 remaining")
}

func TestLimit_Fill_SelfTradePrevention(t *testing.T) {
	limit := NewLimit(priceToTicks(50_000))

	// User 1 places a sell order
	askOrder, err := NewOrder("1", Ask, 50_000, 1.0)
	assertNoError(t, err)
	limit.AddOrder(askOrder)

	// Same user tries to buy - should be prevented (self-trade protection)
	bidOrder, err := NewOrder("1", Bid, 50_000, 1.0)
	assertNoError(t, err)

	matches := limit.Fill(bidOrder)

	// No matches should occur - we protect users from trading with themselves
	assertEqual(t, 0, len(matches), "Should have no matches (self-trade prevention)")
	assertFalse(t, askOrder.IsFilled(), "Ask order should NOT be filled")
	assertFalse(t, bidOrder.IsFilled(), "Bid order should NOT be filled")
	assertEqual(t, 1, len(limit.Orders), "Ask should still be in limit")
}

// =============================================================================
// ORDERBOOK TESTS
// =============================================================================

func TestNewOrderbook(t *testing.T) {
	ob := NewOrderbook()

	assertEqual(t, 0, len(ob.bids), "Bids should be empty")
	assertEqual(t, 0, len(ob.asks), "Asks should be empty")
	assertEqual(t, 0, len(ob.Orders), "Orders should be empty")
}

func TestOrderbook_PlaceLimitOrder_NoMatch(t *testing.T) {
	ob := NewOrderbook()

	bidOrder, err := NewOrder("1", Bid, 50_000, 1.0)
	assertNoError(t, err)

	// Calculate ticks for the price
	priceTicks := priceToTicks(50_000)
	matches := ob.PlaceLimitOrder(bidOrder, priceTicks)

	assertEqual(t, 0, len(matches), "Should have no matches")
	assertEqual(t, 1, len(ob.Bids()), "Should have 1 bid")
	assertEqual(t, OrderOpen, bidOrder.State, "Order state should be open")
	assertFloat(t, 1.0, ob.BidTotalVolume(), "Bid total volume")

	order, exists := ob.GetOrder(bidOrder.ID)
	assertTrue(t, exists, "Order should exist in orderbook")
	assertEqual(t, bidOrder.ID, order.ID, "Stored order ID")
}

func TestOrderbook_PlaceLimitOrder_FullMatch(t *testing.T) {
	ob := NewOrderbook()

	// Seller places ask
	askOrder, err := NewOrder("1", Ask, 50_000, 1.0)
	assertNoError(t, err)
	ob.PlaceLimitOrder(askOrder, priceToTicks(50_000))

	// Buyer comes in with matching bid - should execute immediately
	bidOrder, err := NewOrder("2", Bid, 50_000, 1.0)
	assertNoError(t, err)
	matches := ob.PlaceLimitOrder(bidOrder, priceToTicks(50_000))

	assertEqual(t, 1, len(matches), "Should have 1 match")
	assertFloat(t, 1.0, matches[0].SizeFilled, "Match size")
	assertFloat(t, 50_000.0, matches[0].Price, "Match price")
	assertEqual(t, "2", matches[0].Bid.UserID, "Buyer")
	assertEqual(t, "1", matches[0].Ask.UserID, "Seller")

	assertTrue(t, askOrder.IsFilled(), "Ask should be filled")
	assertTrue(t, bidOrder.IsFilled(), "Bid should be filled")

	// Both orders matched - book should be empty
	assertEqual(t, 0, len(ob.Bids()), "Bids should be empty")
	assertEqual(t, 0, len(ob.Asks()), "Asks should be empty")
	assertFloat(t, 0.0, ob.BidTotalVolume(), "Bid volume should be 0")
	assertFloat(t, 0.0, ob.AskTotalVolume(), "Ask volume should be 0")
}

func TestOrderbook_PlaceLimitOrder_PartialMatch(t *testing.T) {
	ob := NewOrderbook()

	// Large ask available
	askOrder, err := NewOrder("1", Ask, 50_000, 2.0)
	assertNoError(t, err)
	ob.PlaceLimitOrder(askOrder, priceToTicks(50_000))

	// Small bid comes in - partial fill
	bidOrder, err := NewOrder("2", Bid, 50_000, 1.0)
	assertNoError(t, err)
	matches := ob.PlaceLimitOrder(bidOrder, priceToTicks(50_000))

	assertEqual(t, 1, len(matches), "Should have 1 match")
	assertFloat(t, 1.0, matches[0].SizeFilled, "Match size")

	assertTrue(t, bidOrder.IsFilled(), "Bid should be filled")
	assertFalse(t, askOrder.IsFilled(), "Ask should NOT be filled")
	assertFloat(t, 1.0, askOrder.RemainingAmount(), "Ask remaining")

	// One ask should remain in the book
	assertEqual(t, 1, len(ob.Asks()), "Should have 1 ask remaining")
	assertFloat(t, 1.0, ob.AskTotalVolume(), "Ask volume should be 1")
}

func TestOrderbook_PlaceLimitOrder_PriceNoMatch(t *testing.T) {
	ob := NewOrderbook()

	// Ask at higher price
	askOrder, err := NewOrder("1", Ask, 51_000, 1.0)
	assertNoError(t, err)
	ob.PlaceLimitOrder(askOrder, priceToTicks(51_000))

	// Bid at lower price - no match should occur (spread too wide)
	bidOrder, err := NewOrder("2", Bid, 50_000, 1.0)
	assertNoError(t, err)
	matches := ob.PlaceLimitOrder(bidOrder, priceToTicks(50_000))

	assertEqual(t, 0, len(matches), "Should have no matches")
	assertEqual(t, 1, len(ob.Asks()), "Should have 1 ask")
	assertEqual(t, 1, len(ob.Bids()), "Should have 1 bid")
}

func TestOrderbook_PlaceLimitOrder_MultipleMatches(t *testing.T) {
	ob := NewOrderbook()

	// Two sellers at different prices
	ask1, err := NewOrder("1", Ask, 50_000, 1.0)
	assertNoError(t, err)
	ob.PlaceLimitOrder(ask1, priceToTicks(50_000))

	ask2, err := NewOrder("2", Ask, 50_100, 1.0)
	assertNoError(t, err)
	ob.PlaceLimitOrder(ask2, priceToTicks(50_100))

	// Buyer willing to pay up to 50100 for 2 BTC - should match both
	bidOrder, err := NewOrder("3", Bid, 50_100, 2.0)
	assertNoError(t, err)
	matches := ob.PlaceLimitOrder(bidOrder, priceToTicks(50_100))

	assertEqual(t, 2, len(matches), "Should have 2 matches")

	// First match at best price (price improvement for buyer!)
	assertFloat(t, 50_000.0, matches[0].Price, "First match price (best)")
	assertFloat(t, 1.0, matches[0].SizeFilled, "First match size")
	assertEqual(t, "1", matches[0].Ask.UserID, "First seller")

	// Second match at worse price
	assertFloat(t, 50_100.0, matches[1].Price, "Second match price")
	assertFloat(t, 1.0, matches[1].SizeFilled, "Second match size")
	assertEqual(t, "2", matches[1].Ask.UserID, "Second seller")

	assertEqual(t, 0, len(ob.Asks()), "Asks should be empty")
}

func TestOrderbook_PlaceLimitOrder_PriceTimePriority(t *testing.T) {
	ob := NewOrderbook()

	// First seller arrives
	ask1, err := NewOrder("1", Ask, 50_000, 1.0)
	assertNoError(t, err)

	time.Sleep(1 * time.Millisecond)

	// Second seller at same price
	ask2, err := NewOrder("2", Ask, 50_000, 1.0)
	assertNoError(t, err)

	ob.PlaceLimitOrder(ask1, priceToTicks(50_000))
	ob.PlaceLimitOrder(ask2, priceToTicks(50_000))

	// Buyer comes in - should match with ask1 first (FIFO)
	bidOrder, err := NewOrder("3", Bid, 50_000, 1.0)
	assertNoError(t, err)
	matches := ob.PlaceLimitOrder(bidOrder, priceToTicks(50_000))

	assertEqual(t, 1, len(matches), "Should have 1 match")
	assertEqual(t, "1", matches[0].Ask.UserID, "Should match with user 1 first (FIFO)")

	// Ask2 should remain in book
	assertEqual(t, 1, len(ob.Asks()), "Should have 1 ask remaining")
	assertEqual(t, "2", ob.Asks()[0].Orders[0].UserID, "Remaining order should be user 2")
}

func TestOrderbook_PlaceLimitOrder_SelfTradePrevention(t *testing.T) {
	ob := NewOrderbook()

	// User 1 places sell order
	askOrder, err := NewOrder("1", Ask, 50_000, 1.0)
	assertNoError(t, err)
	ob.PlaceLimitOrder(askOrder, priceToTicks(50_000))

	// Same user tries to buy - should not match with own order
	bidOrder, err := NewOrder("1", Bid, 50_000, 1.0)
	assertNoError(t, err)
	matches := ob.PlaceLimitOrder(bidOrder, priceToTicks(50_000))

	// No match should occur (we protect users from self-trading)
	assertEqual(t, 0, len(matches), "Should have no matches (self-trade prevention)")

	// Both orders should be in the book waiting for external counterparty
	assertEqual(t, 1, len(ob.Asks()), "Should have 1 ask")
	assertEqual(t, 1, len(ob.Bids()), "Should have 1 bid")
}

func TestOrderbook_CancelOrder(t *testing.T) {
	ob := NewOrderbook()

	order, err := NewOrder("1", Bid, 50_000, 1.0)
	assertNoError(t, err)
	ob.PlaceLimitOrder(order, priceToTicks(50_000))

	assertEqual(t, 1, len(ob.Bids()), "Should have 1 bid")

	// User decides to cancel the order
	cancelledOrder, err := ob.CancelOrder(order.ID)
	assertNoError(t, err)

	assertEqual(t, order.ID, cancelledOrder.ID, "Cancelled order ID")
	assertEqual(t, OrderCancelled, cancelledOrder.State, "State should be cancelled")

	// Order removed from book
	assertEqual(t, 0, len(ob.Bids()), "Bids should be empty after cancel")
	assertFloat(t, 0.0, ob.BidTotalVolume(), "Bid volume should be 0")

	_, exists := ob.GetOrder(order.ID)
	assertFalse(t, exists, "Order should not exist after cancel")
}

func TestOrderbook_CancelOrder_NotFound(t *testing.T) {
	ob := NewOrderbook()

	// Try to cancel non-existent order
	_, err := ob.CancelOrder(99999)
	if err != ErrOrderNotFound {
		t.Errorf("expected ErrOrderNotFound, got %v", err)
	}
}

func TestOrderbook_CancelOrder_PartiallyFilled(t *testing.T) {
	ob := NewOrderbook()

	// Small ask available
	askOrder, err := NewOrder("1", Ask, 50_000, 1.0)
	assertNoError(t, err)
	ob.PlaceLimitOrder(askOrder, priceToTicks(50_000))

	// Large bid - will be partially filled
	bidOrder, err := NewOrder("2", Bid, 50_000, 2.0)
	assertNoError(t, err)
	matches := ob.PlaceLimitOrder(bidOrder, priceToTicks(50_000))

	assertEqual(t, 1, len(matches), "Should have 1 match")
	assertFloat(t, 1.0, bidOrder.FilledAmount, "Bid filled amount")
	assertFloat(t, 1.0, bidOrder.RemainingAmount(), "Bid remaining")

	// User cancels the remaining portion
	cancelledOrder, err := ob.CancelOrder(bidOrder.ID)
	assertNoError(t, err)

	assertEqual(t, OrderCancelled, cancelledOrder.State, "State should be cancelled")
	assertFloat(t, 1.0, cancelledOrder.FilledAmount, "Filled amount should be preserved")

	assertEqual(t, 0, len(ob.Bids()), "Bids should be empty")
}

func TestOrderbook_BestBid_BestAsk(t *testing.T) {
	ob := NewOrderbook()

	// Add multiple bids at different prices
	order1, err := NewOrder("1", Bid, 49_000, 1.0)
	assertNoError(t, err)
	ob.PlaceLimitOrder(order1, priceToTicks(49_000))

	order2, err := NewOrder("2", Bid, 50_000, 1.0)
	assertNoError(t, err)
	ob.PlaceLimitOrder(order2, priceToTicks(50_000))

	order3, err := NewOrder("3", Bid, 48_000, 1.0)
	assertNoError(t, err)
	ob.PlaceLimitOrder(order3, priceToTicks(48_000))

	// Add multiple asks at different prices
	order4, err := NewOrder("4", Ask, 51_000, 1.0)
	assertNoError(t, err)
	ob.PlaceLimitOrder(order4, priceToTicks(51_000))

	order5, err := NewOrder("5", Ask, 52_000, 1.0)
	assertNoError(t, err)
	ob.PlaceLimitOrder(order5, priceToTicks(52_000))

	order6, err := NewOrder("6", Ask, 50_500, 1.0)
	assertNoError(t, err)
	ob.PlaceLimitOrder(order6, priceToTicks(50_500))

	// Best bid should be highest price (50000)
	bestBid, hasBid := ob.BestBid()
	assertTrue(t, hasBid, "Should have best bid")
	assertFloat(t, 50_000, bestBid.Price(), "Best bid price")

	// Best ask should be lowest price (50500)
	bestAsk, hasAsk := ob.BestAsk()
	assertTrue(t, hasAsk, "Should have best ask")
	assertFloat(t, 50_500, bestAsk.Price(), "Best ask price")
}

func TestOrderbook_Spread(t *testing.T) {
	ob := NewOrderbook()

	// Create a market with spread
	order, err := NewOrder("1", Bid, 49_000, 1.0)
	assertNoError(t, err)
	ob.PlaceLimitOrder(order, priceToTicks(49_000))

	order2, err := NewOrder("2", Ask, 51_000, 1.0)
	assertNoError(t, err)
	ob.PlaceLimitOrder(order2, priceToTicks(51_000))

	// Spread = difference between best ask and best bid
	spread := ob.Spread()
	assertFloat(t, 2000.0, spread, "Spread should be 51000 - 49000")
}

func TestOrderbook_Spread_EmptyBook(t *testing.T) {
	ob := NewOrderbook()

	// Empty book has no spread
	spread := ob.Spread()
	assertFloat(t, 0.0, spread, "Spread should be 0 for empty book")
}

func TestOrderbook_TotalVolumes(t *testing.T) {
	ob := NewOrderbook()

	// Add bids with different volumes
	bid1, err := NewOrder("1", Bid, 50_000, 1.0)
	assertNoError(t, err)
	ob.PlaceLimitOrder(bid1, priceToTicks(50_000))

	bid2, err := NewOrder("2", Bid, 49_000, 2.0)
	assertNoError(t, err)
	ob.PlaceLimitOrder(bid2, priceToTicks(49_000))

	// Add ask
	ask1, err := NewOrder("3", Ask, 51_000, 3.0)
	assertNoError(t, err)
	ob.PlaceLimitOrder(ask1, priceToTicks(51_000))

	// Total volume = sum of all orders
	assertFloat(t, 3.0, ob.BidTotalVolume(), "Bid total volume")
	assertFloat(t, 3.0, ob.AskTotalVolume(), "Ask total volume")
}
