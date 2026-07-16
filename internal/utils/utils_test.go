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

func TestNormalizeFileExtensions(t *testing.T) {
	input := []string{" .JPG ", "PNG", ".Pdf", ""}
	expected := []string{"jpg", "png", "pdf", ""}

	result := NormalizeFileExtensions(input)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf(
			"NormalizeFileExtensions(%v) = %v; expected %v",
			input,
			result,
			expected,
		)
	}
}

func TestNormalizeTrustedURLs(t *testing.T) {
	tests := []struct {
		name        string
		input       []string
		expected    []string
		expectError bool
	}{
		{
			name:     "trims URLs and removes empty entries",
			input:    []string{" https://example.com ", "", "   ", "http://test.org"},
			expected: []string{"https://example.com", "http://test.org"},
		},
		{
			name:     "accepts wildcard",
			input:    []string{"*"},
			expected: []string{"*"},
		},
		{
			name:        "rejects URL without scheme",
			input:       []string{"example.com"},
			expectError: true,
		},
		{
			name:        "rejects unsupported scheme",
			input:       []string{"ftp://example.com"},
			expectError: true,
		},
		{
			name:        "rejects URL without host",
			input:       []string{"https://"},
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := NormalizeTrustedURLs(test.input)

			if test.expectError {
				if err == nil {
					t.Fatalf("NormalizeTrustedURLs(%v) expected an error", test.input)
				}
				return
			}

			if err != nil {
				t.Fatalf("NormalizeTrustedURLs(%v) returned error: %v", test.input, err)
			}

			if !reflect.DeepEqual(result, test.expected) {
				t.Errorf(
					"NormalizeTrustedURLs(%v) = %v; expected %v",
					test.input,
					result,
					test.expected,
				)
			}
		})
	}
}
