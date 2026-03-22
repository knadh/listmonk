package core

import (
	"reflect"
	"testing"
)

func TestStrSliceContains(t *testing.T) {
	tests := []struct {
		str      string
		sl       []string
		expected bool
	}{
		{"apple", []string{"apple", "banana", "cherry"}, true},
		{"grape", []string{"apple", "banana", "cherry"}, false},
		{"", []string{"apple", "banana", "cherry"}, false},
		{"apple", []string{}, false},
	}

	for _, test := range tests {
		if result := strSliceContains(test.str, test.sl); result != test.expected {
			t.Errorf("strSliceContains(%q, %v) = %v; want %v", test.str, test.sl, result, test.expected)
		}
	}
}

func TestNormalizeTags(t *testing.T) {
	tests := []struct {
		tags     []string
		expected []string
	}{
		{[]string{"tag1", "tag 2", "  tag3  "}, []string{"tag1", "tag-2", "tag3"}},
		{[]string{"  ", ""}, nil},
		{[]string{"TAG1", "Tag2"}, []string{"TAG1", "Tag2"}}, // current implementation doesn't lower case
	}

	for _, test := range tests {
		result := normalizeTags(test.tags)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("normalizeTags(%v) = %v; want %v", test.tags, result, test.expected)
		}
	}
}

func TestStrHasLen(t *testing.T) {
	tests := []struct {
		str      string
		min, max int
		expected bool
	}{
		{"abc", 1, 5, true},
		{"abc", 3, 3, true},
		{"abc", 4, 5, false},
		{"", 0, 0, true},
	}

	for _, test := range tests {
		if result := strHasLen(test.str, test.min, test.max); result != test.expected {
			t.Errorf("strHasLen(%q, %d, %d) = %v; want %v", test.str, test.min, test.max, result, test.expected)
		}
	}
}
