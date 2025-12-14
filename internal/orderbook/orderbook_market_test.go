package orderbook

import "testing"

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

func TestOrderbook_PlaceMarketOrder_Sell_PartialFill_WhenNotEnoughLiquidity(t *testing.T) {
	ob := NewOrderbook()

	// Only 0.3 BTC worth of bids available
	bid, err := NewOrder("buyer", Bid, 50_000, 0.3)
	assertNoError(t, err)
	ob.PlaceLimitOrder(bid)

	// Market SELL tries to sell 1.0 BTC, but only 0.3 can be matched
	sell, err := NewMarketOrder("seller", Ask, 1.0)
	assertNoError(t, err)

	matches := ob.PlaceMarketOrder(sell)

	assertEqual(t, 1, len(matches), "Should have 1 match")
	assertFloat(t, 0.3, matches[0].SizeFilled, "Filled size should be the available liquidity")
	assertFloat(t, 50_000, matches[0].Price, "Match price")

	assertEqual(t, OrderPartiallyFilled, sell.State, "Market sell should be partially filled")
	assertFloat(t, 0.3, sell.FilledAmount, "Filled amount")
	assertFloat(t, 0.7, sell.RemainingAmount(), "Remaining amount")

	// Market orders must NOT be stored in the orderbook
	_, exists := ob.GetOrder(sell.ID)
	assertFalse(t, exists, "Market order should not be stored in Orders map")

	// Bid should be fully consumed
	assertEqual(t, 0, len(ob.Bids()), "Bids should be empty after consuming liquidity")
	assertFloat(t, 0.0, ob.BidTotalVolume(), "Bid total volume should be 0")
}

func TestOrderbook_PlaceMarketOrder_EmptyBook(t *testing.T) {
	ob := NewOrderbook()

	// No asks available
	buy, err := NewMarketOrder("buyer", Bid, 1.0)
	assertNoError(t, err)

	matches := ob.PlaceMarketOrder(buy)

	assertEqual(t, 0, len(matches), "Should have no matches")
	assertEqual(t, OrderOpen, buy.State, "Market order with no matches stays open")
	assertFloat(t, 0.0, buy.FilledAmount, "No filled amount")

	// Market orders must NOT be stored in the orderbook
	_, exists := ob.GetOrder(buy.ID)
	assertFalse(t, exists, "Market order should not be stored in Orders map")
}
