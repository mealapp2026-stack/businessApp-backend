package httpapi

import "testing"

func TestNormalizePhone(t *testing.T) {
	tests := map[string]string{
		"+234 801 234 5678": "+2348012345678",
		"0801-234-5678":     "08012345678",
		"(0801) 234.5678":   "08012345678",
	}
	for input, expected := range tests {
		if actual := normalizePhone(input); actual != expected {
			t.Errorf("normalizePhone(%q) = %q, want %q", input, actual, expected)
		}
	}
}

func TestSupportedCurrencies(t *testing.T) {
	for _, code := range []string{"NGN", "USD", "EUR", "XOF"} {
		if _, ok := supportedCurrencies[code]; !ok {
			t.Errorf("expected %s to be supported", code)
		}
	}
	if _, ok := supportedCurrencies["INVALID"]; ok {
		t.Error("unexpected invalid currency support")
	}
}
