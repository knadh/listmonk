package utils

import (
	"reflect"
	"testing"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		email    string
		expected bool
	}{
		{"test@example.com", true},
		{"invalid-email", false},
		{"<test@example.com>", false},
		{"test@example", true},
		{"", false},
	}

	for _, test := range tests {
		if result := ValidateEmail(test.email); result != test.expected {
			t.Errorf("ValidateEmail(%q) = %v; want %v", test.email, result, test.expected)
		}
	}
}

func TestGenerateRandomString(t *testing.T) {
	n := 16
	s1, err := GenerateRandomString(n)
	if err != nil {
		t.Fatalf("GenerateRandomString failed: %v", err)
	}
	if len(s1) != n {
		t.Errorf("expected length %d, got %d", n, len(s1))
	}

	s2, _ := GenerateRandomString(n)
	if s1 == s2 {
		t.Errorf("expected different random strings, got same: %s", s1)
	}
}

func TestSanitizeURI(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/admin", "/admin"},
		{"  /admin  ", "/admin"},
		{"http://example.com/admin", "/admin"},
		{"//example.com/admin", "/admin"},
		{"", "/"},
		{"/path/../forbidden", "/"},
	}

	for _, test := range tests {
		if result := SanitizeURI(test.input); result != test.expected {
			t.Errorf("SanitizeURI(%q) = %q; want %q", test.input, result, test.expected)
		}
	}
}

func TestGetTplSubject(t *testing.T) {
	tests := []struct {
		subject  string
		body     []byte
		expected string
		newBody  []byte
	}{
		{
			"Default Subject",
			[]byte("<html><head><title data-i18n>Custom Subject</title></head><body>Content</body></html>"),
			"Custom Subject",
			[]byte("<html><head></head><body>Content</body></html>"),
		},
		{
			"Default Subject",
			[]byte("<html><body>No custom subject</body></html>"),
			"Default Subject",
			[]byte("<html><body>No custom subject</body></html>"),
		},
		{
			"Default Subject",
			[]byte("Multiple titles: <title data-i18n>Title 1</title> <title data-i18n>Title 2</title>"),
			"Title 1",
			[]byte("Multiple titles:  "), // reTitle.ReplaceAll replaces all matches
		},
	}

	for _, test := range tests {
		resSub, resBody := GetTplSubject(test.subject, test.body)
		if resSub != test.expected {
			t.Errorf("GetTplSubject subject = %q; want %q", resSub, test.expected)
		}
		if !reflect.DeepEqual(resBody, test.newBody) {
			t.Errorf("GetTplSubject body = %q; want %q", string(resBody), string(test.newBody))
		}
	}
}
