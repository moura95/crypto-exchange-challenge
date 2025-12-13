package account

import (
	"testing"
)

func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func assertError(t *testing.T, expected, actual error) {
	t.Helper()
	if expected != actual {
		t.Errorf("expected error %v, got %v", expected, actual)
	}
}

func assertFloat(t *testing.T, expected, actual float64, msg string) {
	t.Helper()
	if expected != actual {
		t.Errorf("%s: expected %.4f, got %.4f", msg, expected, actual)
	}
}

func assertNil(t *testing.T, actual interface{}, msg string) {
	t.Helper()
	if actual != nil {
		t.Errorf("%s: expected nil, got %v", msg, actual)
	}
}

// =============================================================================
// BALANCE TESTS
// =============================================================================

func TestBalance_Total(t *testing.T) {
	balance := &Balance{Available: 100, Locked: 50}
	assertFloat(t, 150, balance.Total(), "Total")
}

// =============================================================================
// MANAGER TESTS
// =============================================================================

func TestNewManager(t *testing.T) {
	m := NewManager()
	if m == nil {
		t.Fatal("NewManager returned nil")
	}
	if m.accounts == nil {
		t.Error("accounts map should be initialized")
	}
}

func TestManager_Credit(t *testing.T) {
	m := NewManager()

	// Credit new account
	err := m.Credit("1", "BTC", 10.0)
	assertNoError(t, err)

	balance := m.GetBalance("1", "BTC")
	assertFloat(t, 10.0, balance.Available, "Available credit")
	assertFloat(t, 0.0, balance.Locked, "Locked should be 0")

	// Credit existing account
	err = m.Credit("1", "BTC", 5.0)
	assertNoError(t, err)

	balance = m.GetBalance("1", "BTC")
	assertFloat(t, 15.0, balance.Available, "Available after second credit")
}

func TestManager_Credit_InvalidInputs(t *testing.T) {
	m := NewManager()

	err := m.Credit("", "BTC", 10.0)
	assertError(t, ErrInvalidUserID, err)

	err = m.Credit("1", "", 10.0)
	assertError(t, ErrInvalidAsset, err)

	err = m.Credit("1", "BTC", 0)
	assertError(t, ErrInvalidAmount, err)

	err = m.Credit("1", "BTC", -10)
	assertError(t, ErrInvalidAmount, err)
}

func TestManager_Debit(t *testing.T) {
	m := NewManager()

	// Setup
	m.Credit("1", "BTC", 100.0)

	// Debit
	err := m.Debit("1", "BTC", 30.0)
	assertNoError(t, err)

	balance := m.GetBalance("1", "BTC")
	assertFloat(t, 70.0, balance.Available, "Available after debit")
}

func TestManager_Debit_InsufficientBalance(t *testing.T) {
	m := NewManager()

	m.Credit("1", "BTC", 50.0)

	err := m.Debit("1", "BTC", 100.0)
	assertError(t, ErrInsufficientBalance, err)

	// Balance should not change
	balance := m.GetBalance("1", "BTC")
	assertFloat(t, 50.0, balance.Available, "Available should not change")
}

func TestManager_Debit_InvalidInputs(t *testing.T) {
	m := NewManager()

	err := m.Debit("", "BTC", 10.0)
	assertError(t, ErrInvalidUserID, err)

	err = m.Debit("1", "", 10.0)
	assertError(t, ErrInvalidAsset, err)

	err = m.Debit("1", "BTC", 0)
	assertError(t, ErrInvalidAmount, err)
}

func TestManager_Lock(t *testing.T) {
	m := NewManager()

	m.Credit("1", "BRL", 100000.0)

	// Lock for order
	err := m.Lock("1", "BRL", 50000.0)
	assertNoError(t, err)

	balance := m.GetBalance("1", "BRL")
	assertFloat(t, 50000.0, balance.Available, "Available after lock")
	assertFloat(t, 50000.0, balance.Locked, "Locked after lock")
	assertFloat(t, 100000.0, balance.Total(), "Total should not change")
}

func TestManager_Lock_InsufficientBalance(t *testing.T) {
	m := NewManager()

	m.Credit("1", "BRL", 50000.0)

	err := m.Lock("1", "BRL", 100000.0)
	assertError(t, ErrInsufficientBalance, err)

	// Balance should not change
	balance := m.GetBalance("1", "BRL")
	assertFloat(t, 50000.0, balance.Available, "Available should not change")
	assertFloat(t, 0.0, balance.Locked, "Locked should not change")
}

func TestManager_Lock_InvalidInputs(t *testing.T) {
	m := NewManager()

	err := m.Lock("", "BTC", 10.0)
	assertError(t, ErrInvalidUserID, err)

	err = m.Lock("1", "", 10.0)
	assertError(t, ErrInvalidAsset, err)

	err = m.Lock("1", "BTC", 0)
	assertError(t, ErrInvalidAmount, err)
}

func TestManager_Unlock(t *testing.T) {
	m := NewManager()

	m.Credit("1", "BRL", 100000.0)
	m.Lock("1", "BRL", 60000.0)

	// Unlock (cancel order)
	err := m.Unlock("1", "BRL", 30000.0)
	assertNoError(t, err)

	balance := m.GetBalance("1", "BRL")
	assertFloat(t, 70000.0, balance.Available, "Available after unlock")
	assertFloat(t, 30000.0, balance.Locked, "Locked after unlock")
}

func TestManager_Unlock_InsufficientLocked(t *testing.T) {
	m := NewManager()

	m.Credit("1", "BRL", 100000.0)
	m.Lock("1", "BRL", 30000.0)

	err := m.Unlock("1", "BRL", 50000.0)
	assertError(t, ErrInsufficientLocked, err)
}

func TestManager_Unlock_InvalidInputs(t *testing.T) {
	m := NewManager()

	err := m.Unlock("", "BTC", 10.0)
	assertError(t, ErrInvalidUserID, err)

	err = m.Unlock("1", "", 10.0)
	assertError(t, ErrInvalidAsset, err)

	err = m.Unlock("1", "BTC", 0)
	assertError(t, ErrInvalidAmount, err)
}

func TestManager_DebitLocked(t *testing.T) {
	m := NewManager()

	m.Credit("1", "BRL", 100000.0)
	m.Lock("1", "BRL", 50000.0)

	// After match, debit from locked
	err := m.DebitLocked("1", "BRL", 50000.0)
	assertNoError(t, err)

	balance := m.GetBalance("1", "BRL")
	assertFloat(t, 50000.0, balance.Available, "Available should not change")
	assertFloat(t, 0.0, balance.Locked, "Locked after debit")
	assertFloat(t, 50000.0, balance.Total(), "Total after debit")
}

func TestManager_DebitLocked_InsufficientLocked(t *testing.T) {
	m := NewManager()

	m.Credit("1", "BRL", 100000.0)
	m.Lock("1", "BRL", 30000.0)

	err := m.DebitLocked("1", "BRL", 50000.0)
	assertError(t, ErrInsufficientLocked, err)
}

func TestManager_DebitLocked_InvalidInputs(t *testing.T) {
	m := NewManager()

	err := m.DebitLocked("", "BTC", 10.0)
	assertError(t, ErrInvalidUserID, err)

	err = m.DebitLocked("1", "", 10.0)
	assertError(t, ErrInvalidAsset, err)

	err = m.DebitLocked("1", "BTC", 0)
	assertError(t, ErrInvalidAmount, err)
}

func TestManager_GetAllBalances(t *testing.T) {
	m := NewManager()

	m.Credit("1", "BTC", 10.0)
	m.Credit("1", "BRL", 100000.0)
	m.Credit("1", "ETH", 50.0)

	balances := m.GetAllBalances("1")

	if len(balances) != 3 {
		t.Errorf("expected 3 balances, got %d", len(balances))
	}

	assertFloat(t, 10.0, balances["BTC"].Available, "BTC balance")
	assertFloat(t, 100000.0, balances["BRL"].Available, "BRL balance")
	assertFloat(t, 50.0, balances["ETH"].Available, "ETH balance")
}

func TestManager_FullOrder_Buy(t *testing.T) {
	m := NewManager()

	// UserID:1 wants to buy 1 BTC @ 50000 BRL
	// 1. Credit BRL
	m.Credit("1", "BRL", 100000.0)

	// 2. Lock BRL for order
	err := m.Lock("1", "BRL", 50000.0)
	assertNoError(t, err)

	balance := m.GetBalance("1", "BRL")
	assertFloat(t, 50000.0, balance.Available, "Available after lock")
	assertFloat(t, 50000.0, balance.Locked, "Locked after lock")

	// 3. Match happens - debit locked BRL, credit BTC
	err = m.DebitLocked("1", "BRL", 50000.0)
	assertNoError(t, err)

	err = m.Credit("1", "BTC", 1.0)
	assertNoError(t, err)

	// Verify final state
	brlBalance := m.GetBalance("1", "BRL")
	assertFloat(t, 50000.0, brlBalance.Available, "BRL Available after match")
	assertFloat(t, 0.0, brlBalance.Locked, "BRL Locked after match")

	btcBalance := m.GetBalance("1", "BTC")
	assertFloat(t, 1.0, btcBalance.Available, "BTC Available after match")
}

func TestManager_FullOrder_Sell(t *testing.T) {
	m := NewManager()

	// UserId:2 wants to sell 1 BTC @ 50000 BRL
	// 1. Credit BTC
	m.Credit("2", "BTC", 2.0)

	// 2. Lock BTC for order
	err := m.Lock("2", "BTC", 1.0)
	assertNoError(t, err)

	balance := m.GetBalance("2", "BTC")
	assertFloat(t, 1.0, balance.Available, "Available after lock")
	assertFloat(t, 1.0, balance.Locked, "Locked after lock")

	// 3. Match happens - debit locked BTC, credit BRL
	err = m.DebitLocked("2", "BTC", 1.0)
	assertNoError(t, err)

	err = m.Credit("2", "BRL", 50000.0)
	assertNoError(t, err)

	// Verify final state
	btcBalance := m.GetBalance("2", "BTC")
	assertFloat(t, 1.0, btcBalance.Available, "BTC Available after match")
	assertFloat(t, 0.0, btcBalance.Locked, "BTC Locked after match")

	brlBalance := m.GetBalance("2", "BRL")
	assertFloat(t, 50000.0, brlBalance.Available, "BRL Available after match")
}

func TestManager_FullOrder_Cancel(t *testing.T) {
	m := NewManager()

	// OrderID:3 creates order then cancels
	m.Credit("3", "BRL", 100000.0)
	m.Lock("3", "BRL", 50000.0)

	// Cancel order - unlock
	err := m.Unlock("3", "BRL", 50000.0)
	assertNoError(t, err)

	balance := m.GetBalance("3", "BRL")
	assertFloat(t, 100000.0, balance.Available, "Available after cancel")
	assertFloat(t, 0.0, balance.Locked, "Locked after cancel")
}
