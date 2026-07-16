package utils

import (
	"reflect"
	"testing"
)

func TestNormalizeDomains(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "trims whitespace and converts domains to lowercase",
			input:    []string{" Example.COM ", "TEST.org"},
			expected: []string{"example.com", "test.org"},
		},
		{
			name:     "removes empty entries",
			input:    []string{"example.com", "", "   ", "test.org"},
			expected: []string{"example.com", "test.org"},
		},
		{
			name:     "preserves domain order",
			input:    []string{"THIRD.com", "first.com", "SECOND.com"},
			expected: []string{"third.com", "first.com", "second.com"},
		},
		{
			name:     "handles an empty list",
			input:    []string{},
			expected: []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := NormalizeDomains(test.input)

			if !reflect.DeepEqual(result, test.expected) {
				t.Errorf(
					"NormalizeDomains(%v) = %v; expected %v",
					test.input,
					result,
					test.expected,
				)
			}
		})
	}
}
