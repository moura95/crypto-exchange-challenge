package orderbook

import (
	"testing"
	"time"

	"github.com/moura95/crypto-exchange-challenge/pkg/utils"
)

// =============================================================================
// CONFIG (same as Orderbook.NewOrderbook())
// =============================================================================

const priceTick = 0.01

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

func priceToTicks(price float64) int64 {
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
	assertEqual(t, OrderPartiallyFilled, order.State, "Order should be partially_filled")

	order.FilledAmount = 1.0
	order.State = OrderFilled
	assertTrue(t, order.IsFilled(), "Fully filled order should be filled")
	assertEqual(t, OrderFilled, order.State, "Order should be filled")
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

	askOrder, err := NewOrder("1", Ask, 50_000, 1.0)
	assertNoError(t, err)
	limit.AddOrder(askOrder)

	bidOrder, err := NewOrder("2", Bid, 50_000, 1.0)
	assertNoError(t, err)

	matches := limit.Fill(bidOrder, priceTick)

	assertEqual(t, 1, len(matches), "Should have 1 match")
	assertFloat(t, 1.0, matches[0].SizeFilled, "Match size")
	assertFloat(t, 50_000.0, matches[0].Price, "Match price")

	assertTrue(t, askOrder.IsFilled(), "Ask order should be filled")
	assertTrue(t, bidOrder.IsFilled(), "Bid order should be filled")
	assertEqual(t, OrderFilled, askOrder.State, "Ask state")
	assertEqual(t, OrderFilled, bidOrder.State, "Bid state")

	assertEqual(t, 0, len(limit.Orders), "Limit should be empty")
	assertFloat(t, 0.0, limit.TotalVolume, "TotalVolume should be 0")
}

func TestLimit_Fill_PartialMatch_IncomingLarger(t *testing.T) {
	limit := NewLimit(priceToTicks(50_000))

	askOrder, err := NewOrder("1", Ask, 50_000, 1.0)
	assertNoError(t, err)
	limit.AddOrder(askOrder)

	bidOrder, err := NewOrder("2", Bid, 50_000, 2.0)
	assertNoError(t, err)

	matches := limit.Fill(bidOrder, priceTick)

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

	askOrder, err := NewOrder("1", Ask, 50_000, 2.0)
	assertNoError(t, err)
	limit.AddOrder(askOrder)

	bidOrder, err := NewOrder("2", Bid, 50_000, 1.0)
	assertNoError(t, err)

	matches := limit.Fill(bidOrder, priceTick)

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

	ask1, err := NewOrder("1", Ask, 50_000, 1.0)
	assertNoError(t, err)

	time.Sleep(1 * time.Millisecond)

	ask2, err := NewOrder("2", Ask, 50_000, 1.0)
	assertNoError(t, err)

	limit.AddOrder(ask1)
	limit.AddOrder(ask2)

	bidOrder, err := NewOrder("3", Bid, 50_000, 1.5)
	assertNoError(t, err)

	matches := limit.Fill(bidOrder, priceTick)

	assertEqual(t, 2, len(matches), "Should have 2 matches")
	assertFloat(t, 1.0, matches[0].SizeFilled, "First match size")
	assertEqual(t, "1", matches[0].Ask.UserID, "First match should be with user 1 (FIFO)")

	assertFloat(t, 0.5, matches[1].SizeFilled, "Second match size")
	assertEqual(t, "2", matches[1].Ask.UserID, "Second match should be with user 2")

	assertTrue(t, ask1.IsFilled(), "Ask1 should be filled")
	assertFalse(t, ask2.IsFilled(), "Ask2 should NOT be filled")
	assertFloat(t, 0.5, ask2.RemainingAmount(), "Ask2 remaining")
}

func TestLimit_Fill_SelfTradePrevention(t *testing.T) {
	limit := NewLimit(priceToTicks(50_000))

	askOrder, err := NewOrder("1", Ask, 50_000, 1.0)
	assertNoError(t, err)
	limit.AddOrder(askOrder)

	bidOrder, err := NewOrder("1", Bid, 50_000, 1.0)
	assertNoError(t, err)

	matches := limit.Fill(bidOrder, priceTick)

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

	matches := ob.PlaceLimitOrder(bidOrder)

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

	askOrder, err := NewOrder("1", Ask, 50_000, 1.0)
	assertNoError(t, err)
	ob.PlaceLimitOrder(askOrder)

	bidOrder, err := NewOrder("2", Bid, 50_000, 1.0)
	assertNoError(t, err)
	matches := ob.PlaceLimitOrder(bidOrder)

	assertEqual(t, 1, len(matches), "Should have 1 match")
	assertFloat(t, 1.0, matches[0].SizeFilled, "Match size")
	assertFloat(t, 50_000.0, matches[0].Price, "Match price")
	assertEqual(t, "2", matches[0].Bid.UserID, "Buyer")
	assertEqual(t, "1", matches[0].Ask.UserID, "Seller")

	assertTrue(t, askOrder.IsFilled(), "Ask should be filled")
	assertTrue(t, bidOrder.IsFilled(), "Bid should be filled")

	assertEqual(t, 0, len(ob.Bids()), "Bids should be empty")
	assertEqual(t, 0, len(ob.Asks()), "Asks should be empty")
	assertFloat(t, 0.0, ob.BidTotalVolume(), "Bid volume should be 0")
	assertFloat(t, 0.0, ob.AskTotalVolume(), "Ask volume should be 0")
}

func TestOrderbook_PlaceLimitOrder_PartialMatch(t *testing.T) {
	ob := NewOrderbook()

	askOrder, err := NewOrder("1", Ask, 50_000, 2.0)
	assertNoError(t, err)
	ob.PlaceLimitOrder(askOrder)

	bidOrder, err := NewOrder("2", Bid, 50_000, 1.0)
	assertNoError(t, err)
	matches := ob.PlaceLimitOrder(bidOrder)

	assertEqual(t, 1, len(matches), "Should have 1 match")
	assertFloat(t, 1.0, matches[0].SizeFilled, "Match size")

	assertTrue(t, bidOrder.IsFilled(), "Bid should be filled")
	assertFalse(t, askOrder.IsFilled(), "Ask order should NOT be filled")
	assertFloat(t, 1.0, askOrder.RemainingAmount(), "Ask remaining")

	assertEqual(t, 1, len(ob.Asks()), "Should have 1 ask remaining")
	assertFloat(t, 1.0, ob.AskTotalVolume(), "Ask volume should be 1")
}

func TestOrderbook_PlaceLimitOrder_PriceNoMatch(t *testing.T) {
	ob := NewOrderbook()

	askOrder, err := NewOrder("1", Ask, 51_000, 1.0)
	assertNoError(t, err)
	ob.PlaceLimitOrder(askOrder)

	bidOrder, err := NewOrder("2", Bid, 50_000, 1.0)
	assertNoError(t, err)
	matches := ob.PlaceLimitOrder(bidOrder)

	assertEqual(t, 0, len(matches), "Should have no matches")
	assertEqual(t, 1, len(ob.Asks()), "Should have 1 ask")
	assertEqual(t, 1, len(ob.Bids()), "Should have 1 bid")
}

func TestOrderbook_PlaceLimitOrder_MultipleMatches(t *testing.T) {
	ob := NewOrderbook()

	ask1, err := NewOrder("1", Ask, 50_000, 1.0)
	assertNoError(t, err)
	ob.PlaceLimitOrder(ask1)

	ask2, err := NewOrder("2", Ask, 50_100, 1.0)
	assertNoError(t, err)
	ob.PlaceLimitOrder(ask2)

	bidOrder, err := NewOrder("3", Bid, 50_100, 2.0)
	assertNoError(t, err)
	matches := ob.PlaceLimitOrder(bidOrder)

	assertEqual(t, 2, len(matches), "Should have 2 matches")

	assertFloat(t, 50_000.0, matches[0].Price, "First match price (best)")
	assertFloat(t, 1.0, matches[0].SizeFilled, "First match size")
	assertEqual(t, "1", matches[0].Ask.UserID, "First seller")

	assertFloat(t, 50_100.0, matches[1].Price, "Second match price")
	assertFloat(t, 1.0, matches[1].SizeFilled, "Second match size")
	assertEqual(t, "2", matches[1].Ask.UserID, "Second seller")

	assertEqual(t, 0, len(ob.Asks()), "Asks should be empty")
}

func TestOrderbook_PlaceLimitOrder_PriceTimePriority(t *testing.T) {
	ob := NewOrderbook()

	ask1, err := NewOrder("1", Ask, 50_000, 1.0)
	assertNoError(t, err)

	time.Sleep(1 * time.Millisecond)

	ask2, err := NewOrder("2", Ask, 50_000, 1.0)
	assertNoError(t, err)

	ob.PlaceLimitOrder(ask1)
	ob.PlaceLimitOrder(ask2)

	bidOrder, err := NewOrder("3", Bid, 50_000, 1.0)
	assertNoError(t, err)
	matches := ob.PlaceLimitOrder(bidOrder)

	assertEqual(t, 1, len(matches), "Should have 1 match")
	assertEqual(t, "1", matches[0].Ask.UserID, "Should match with user 1 first (FIFO)")

	assertEqual(t, 1, len(ob.Asks()), "Should have 1 ask remaining")
	assertEqual(t, "2", ob.Asks()[0].Orders[0].UserID, "Remaining order should be user 2")
}

func TestOrderbook_PlaceLimitOrder_SelfTradePrevention(t *testing.T) {
	ob := NewOrderbook()

	askOrder, err := NewOrder("1", Ask, 50_000, 1.0)
	assertNoError(t, err)
	ob.PlaceLimitOrder(askOrder)

	bidOrder, err := NewOrder("1", Bid, 50_000, 1.0)
	assertNoError(t, err)
	matches := ob.PlaceLimitOrder(bidOrder)

	assertEqual(t, 0, len(matches), "Should have no matches (self-trade prevention)")
	assertEqual(t, 1, len(ob.Asks()), "Should have 1 ask")
	assertEqual(t, 1, len(ob.Bids()), "Should have 1 bid")
}

func TestOrderbook_CancelOrder(t *testing.T) {
	ob := NewOrderbook()

	order, err := NewOrder("1", Bid, 50_000, 1.0)
	assertNoError(t, err)
	ob.PlaceLimitOrder(order)

	assertEqual(t, 1, len(ob.Bids()), "Should have 1 bid")

	cancelledOrder, err := ob.CancelOrder(order.ID)
	assertNoError(t, err)

	assertEqual(t, order.ID, cancelledOrder.ID, "Cancelled order ID")
	assertEqual(t, OrderCancelled, cancelledOrder.State, "State should be cancelled")

	assertEqual(t, 0, len(ob.Bids()), "Bids should be empty after cancel")
	assertFloat(t, 0.0, ob.BidTotalVolume(), "Bid volume should be 0")

	_, exists := ob.GetOrder(order.ID)
	assertFalse(t, exists, "Order should not exist after cancel")
}

func TestOrderbook_CancelOrder_NotFound(t *testing.T) {
	ob := NewOrderbook()

	_, err := ob.CancelOrder(99999)
	if err != ErrOrderNotFound {
		t.Errorf("expected ErrOrderNotFound, got %v", err)
	}
}

func TestOrderbook_CancelOrder_PartiallyFilled(t *testing.T) {
	ob := NewOrderbook()

	askOrder, err := NewOrder("1", Ask, 50_000, 1.0)
	assertNoError(t, err)
	ob.PlaceLimitOrder(askOrder)

	bidOrder, err := NewOrder("2", Bid, 50_000, 2.0)
	assertNoError(t, err)
	matches := ob.PlaceLimitOrder(bidOrder)

	assertEqual(t, 1, len(matches), "Should have 1 match")
	assertFloat(t, 1.0, bidOrder.FilledAmount, "Bid filled amount")
	assertFloat(t, 1.0, bidOrder.RemainingAmount(), "Bid remaining")

	cancelledOrder, err := ob.CancelOrder(bidOrder.ID)
	assertNoError(t, err)

	assertEqual(t, OrderCancelled, cancelledOrder.State, "State should be cancelled")
	assertFloat(t, 1.0, cancelledOrder.FilledAmount, "Filled amount should be preserved")

	assertEqual(t, 0, len(ob.Bids()), "Bids should be empty")
}

func TestOrderbook_BestBid_BestAsk(t *testing.T) {
	ob := NewOrderbook()

	order1, err := NewOrder("1", Bid, 49_000, 1.0)
	assertNoError(t, err)
	ob.PlaceLimitOrder(order1)

	order2, err := NewOrder("2", Bid, 50_000, 1.0)
	assertNoError(t, err)
	ob.PlaceLimitOrder(order2)

	order3, err := NewOrder("3", Bid, 48_000, 1.0)
	assertNoError(t, err)
	ob.PlaceLimitOrder(order3)

	order4, err := NewOrder("4", Ask, 51_000, 1.0)
	assertNoError(t, err)
	ob.PlaceLimitOrder(order4)

	order5, err := NewOrder("5", Ask, 52_000, 1.0)
	assertNoError(t, err)
	ob.PlaceLimitOrder(order5)

	order6, err := NewOrder("6", Ask, 50_500, 1.0)
	assertNoError(t, err)
	ob.PlaceLimitOrder(order6)

	bestBid, hasBid := ob.BestBid()
	assertTrue(t, hasBid, "Should have best bid")
	assertFloat(t, 50_000, bestBid.Price(priceTick), "Best bid price")

	bestAsk, hasAsk := ob.BestAsk()
	assertTrue(t, hasAsk, "Should have best ask")
	assertFloat(t, 50_500, bestAsk.Price(priceTick), "Best ask price")
}

func TestOrderbook_Spread(t *testing.T) {
	ob := NewOrderbook()

	order, err := NewOrder("1", Bid, 49_000, 1.0)
	assertNoError(t, err)
	ob.PlaceLimitOrder(order)

	order2, err := NewOrder("2", Ask, 51_000, 1.0)
	assertNoError(t, err)
	ob.PlaceLimitOrder(order2)

	spread := ob.Spread()
	assertFloat(t, 2000.0, spread, "Spread should be 51000 - 49000")
}

func TestOrderbook_Spread_EmptyBook(t *testing.T) {
	ob := NewOrderbook()

	spread := ob.Spread()
	assertFloat(t, 0.0, spread, "Spread should be 0 for empty book")
}

func TestOrderbook_TotalVolumes(t *testing.T) {
	ob := NewOrderbook()

	bid1, err := NewOrder("1", Bid, 50_000, 1.0)
	assertNoError(t, err)
	ob.PlaceLimitOrder(bid1)

	bid2, err := NewOrder("2", Bid, 49_000, 2.0)
	assertNoError(t, err)
	ob.PlaceLimitOrder(bid2)

	ask1, err := NewOrder("3", Ask, 51_000, 3.0)
	assertNoError(t, err)
	ob.PlaceLimitOrder(ask1)

	assertFloat(t, 3.0, ob.BidTotalVolume(), "Bid total volume")
	assertFloat(t, 3.0, ob.AskTotalVolume(), "Ask total volume")
}

func TestOrderbook_PlaceMarketOrder_Buy_FullFilled(t *testing.T) {
	ob := NewOrderbook()

	// Two asks in the book
	ask1, err := NewOrder("seller1", Ask, 50_000, 0.6)
	assertNoError(t, err)
	ob.PlaceLimitOrder(ask1)

	ask2, err := NewOrder("seller2", Ask, 50_100, 0.4)
	assertNoError(t, err)
	ob.PlaceLimitOrder(ask2)

	// Market BUY 1.0 BTC should consume 0.6 @ 50_000 + 0.4 @ 50_100
	buy, err := NewMarketOrder("buyer", Bid, 1.0)
	assertNoError(t, err)

	matches := ob.PlaceMarketOrder(buy)

	assertEqual(t, 2, len(matches), "Should have 2 matches")
	assertFloat(t, 50_000, matches[0].Price, "First match price should be best ask")
	assertFloat(t, 0.6, matches[0].SizeFilled, "First match size")
	assertEqual(t, "seller1", matches[0].Ask.UserID, "First match seller")

	assertFloat(t, 50_100, matches[1].Price, "Second match price")
	assertFloat(t, 0.4, matches[1].SizeFilled, "Second match size")
	assertEqual(t, "seller2", matches[1].Ask.UserID, "Second match seller")

	assertEqual(t, OrderFilled, buy.State, "Market buy should be filled")
	assertFloat(t, 1.0, buy.FilledAmount, "Filled amount")

	// Market orders must NOT be stored in the orderbook
	_, exists := ob.GetOrder(buy.ID)
	assertFalse(t, exists, "Market order should not be stored in Orders map")

	// Asks should be empty after consuming both
	assertEqual(t, 0, len(ob.Asks()), "Asks should be empty")
	assertFloat(t, 0.0, ob.AskTotalVolume(), "Ask total volume should be 0")
}

func TestOrderbook_PlaceMarketOrder_Sell_FullFill(t *testing.T) {
	ob := NewOrderbook()

	// Two bids in the book
	bid1, err := NewOrder("buyer1", Bid, 50_200, 0.7)
	assertNoError(t, err)
	ob.PlaceLimitOrder(bid1)

	bid2, err := NewOrder("buyer2", Bid, 50_100, 0.3)
	assertNoError(t, err)
	ob.PlaceLimitOrder(bid2)

	// Market SELL 1.0 BTC should consume 0.7 @ 50_200 + 0.3 @ 50_100
	sell, err := NewMarketOrder("seller", Ask, 1.0)
	assertNoError(t, err)

	matches := ob.PlaceMarketOrder(sell)

	assertEqual(t, 2, len(matches), "Should have 2 matches")
	assertFloat(t, 50_200, matches[0].Price, "First match price should be best bid")
	assertFloat(t, 0.7, matches[0].SizeFilled, "First match size")
	assertEqual(t, "buyer1", matches[0].Bid.UserID, "First match buyer")

	assertFloat(t, 50_100, matches[1].Price, "Second match price")
	assertFloat(t, 0.3, matches[1].SizeFilled, "Second match size")
	assertEqual(t, "buyer2", matches[1].Bid.UserID, "Second match buyer")

	assertEqual(t, OrderFilled, sell.State, "Market sell should be filled")
	assertFloat(t, 1.0, sell.FilledAmount, "Filled amount")

	// Market orders must NOT be stored in the orderbook
	_, exists := ob.GetOrder(sell.ID)
	assertFalse(t, exists, "Market order should not be stored in Orders map")

	// Bids should be empty after consuming both
	assertEqual(t, 0, len(ob.Bids()), "Bids should be empty")
	assertFloat(t, 0.0, ob.BidTotalVolume(), "Bid total volume should be 0")
}

func TestOrderbook_PlaceMarketOrder_Buy_PartialFill_WhenNotEnoughLiquidity(t *testing.T) {
	ob := NewOrderbook()

	// Only 0.4 BTC available on asks
	ask, err := NewOrder("seller", Ask, 50_000, 0.4)
	assertNoError(t, err)
	ob.PlaceLimitOrder(ask)

	// Market BUY tries to buy 1.0 BTC, but only 0.4 exists
	buy, err := NewMarketOrder("buyer", Bid, 1.0)
	assertNoError(t, err)

	matches := ob.PlaceMarketOrder(buy)

	assertEqual(t, 1, len(matches), "Should have 1 match")
	assertFloat(t, 0.4, matches[0].SizeFilled, "Filled size should be the available liquidity")
	assertFloat(t, 50_000, matches[0].Price, "Match price")

	assertEqual(t, OrderPartiallyFilled, buy.State, "Market buy should be partially filled")
	assertFloat(t, 0.4, buy.FilledAmount, "Filled amount")
	assertFloat(t, 0.6, buy.RemainingAmount(), "Remaining amount")

	// Market orders must NOT be stored in the orderbook
	_, exists := ob.GetOrder(buy.ID)
	assertFalse(t, exists, "Market order should not be stored in Orders map")

	// Ask should be fully consumed
	assertEqual(t, 0, len(ob.Asks()), "Asks should be empty after consuming liquidity")
	assertFloat(t, 0.0, ob.AskTotalVolume(), "Ask total volume should be 0")
}
