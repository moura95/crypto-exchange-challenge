package engine

import (
	"testing"
)

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
