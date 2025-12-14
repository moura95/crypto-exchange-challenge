package orderbook

import "testing"

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

func TestNewMarketOrder_Valid(t *testing.T) {
	order, err := NewMarketOrder("1", Bid, 1.0)
	assertNoError(t, err)

	assertEqual(t, "1", order.UserID, "UserID")
	assertEqual(t, Bid, order.Side, "Side")
	assertEqual(t, OrderTypeMarket, order.Type, "Type should be market")
	assertFloat(t, 0.0, order.Price, "Price should be 0 for market orders")
	assertFloat(t, 1.0, order.Amount, "Amount")
	assertEqual(t, OrderOpen, order.State, "State")
}

func TestNewMarketOrder_InvalidAmount(t *testing.T) {
	_, err := NewMarketOrder("1", Bid, 0)
	if err != ErrInvalidAmount {
		t.Errorf("expected ErrInvalidAmount, got %v", err)
	}

	_, err = NewMarketOrder("1", Bid, -1.0)
	if err != ErrInvalidAmount {
		t.Errorf("expected ErrInvalidAmount, got %v", err)
	}
}

func TestNewMarketOrder_InvalidUserID(t *testing.T) {
	_, err := NewMarketOrder("", Bid, 1.0)
	if err == nil {
		t.Error("expected error for empty userID")
	}
}

func TestNewMarketOrder_InvalidSide(t *testing.T) {
	_, err := NewMarketOrder("1", Side("invalid"), 1.0)
	if err != ErrInvalidSide {
		t.Errorf("expected ErrInvalidSide, got %v", err)
	}
}
