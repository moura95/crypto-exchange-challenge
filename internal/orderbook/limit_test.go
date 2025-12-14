package orderbook

import (
	"testing"
	"time"
)

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

func TestLimit_Fill_PartialMatch_ExistingLarger(t *testing.T) {
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

func TestLimit_Price(t *testing.T) {
	limit := NewLimit(priceToTicks(50_000))

	price := limit.Price(priceTick)
	assertFloat(t, 50_000, price, "Limit price should match")
}
