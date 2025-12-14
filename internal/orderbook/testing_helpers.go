package orderbook

import (
	"testing"

	"github.com/moura95/crypto-exchange-challenge/pkg/utils"
)

const priceTick = 0.01

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
