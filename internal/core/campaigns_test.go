package core

import "testing"

func TestIsValidAnalyticsGranularity(t *testing.T) {
	tests := map[string]bool{
		"":      true,
		"hour":  true,
		"day":   true,
		"week":  true,
		"month": true,
		"raw":   false,
		"year":  false,
		"DAY":   false,
	}

	for granularity, want := range tests {
		if got := isValidAnalyticsGranularity(granularity); got != want {
			t.Errorf("isValidAnalyticsGranularity(%q) = %t, want %t", granularity, got, want)
		}
	}
}
