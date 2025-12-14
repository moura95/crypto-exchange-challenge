package orderbook

import (
	"testing"
	"time"
)

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
