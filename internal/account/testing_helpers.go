package account

import "testing"

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
