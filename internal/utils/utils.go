package utils

import (
	"crypto/rand"
	"errors"
	"net/mail"
	"net/url"
	"path"
	"strings"
)

// ErrInvalidEmail is returned by SanitizeEmail for malformed input.
var ErrInvalidEmail = errors.New("invalid e-mail address")

// ValidateEmail reports whether s is a correctly formed bare e-mail address
// (no display name component).
func ValidateEmail(s string) bool {
	_, err := SanitizeEmail(s)
	return err == nil
}

// SanitizeEmail trims, lowercases, and validates s as a bare e-mail address
// (no display name) and returns the canonical form. Returns ErrInvalidEmail
// for anything `mail.ParseAddress` rejects or for input with a display name.
func SanitizeEmail(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	em, err := mail.ParseAddress(s)
	if err != nil || em.Address != s {
		return "", ErrInvalidEmail
	}
	return em.Address, nil
}

// ParseEmailAddress extracts the lowercased bare address from an RFC 5322
// "From"-style header value, accepting both bare addresses ("a@b.com") and
// the display-name form ("Name <a@b.com>"). Returns "" if unparseable.
func ParseEmailAddress(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	em, err := mail.ParseAddress(s)
	if err != nil {
		return ""
	}
	return strings.ToLower(em.Address)
}

// GenerateRandomString generates a cryptographically random, alphanumeric string of length n.
func GenerateRandomString(n int) (string, error) {
	const dictionary = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	var bytes = make([]byte, n)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	for k, v := range bytes {
		bytes[k] = dictionary[v%byte(len(dictionary))]
	}

	return string(bytes), nil
}

// SanitizeURI takes a URL or URI, removes the domain from it, returns only the URI.
// This is used for cleaning "next" redirect URLs/URIs to prevent open redirects.
func SanitizeURI(u string) string {
	u = strings.TrimSpace(u)
	if u == "" {
		return "/"
	}

	p, err := url.Parse(u)
	if err != nil || strings.Contains(p.Path, "..") {
		return "/"
	}

	return path.Clean(p.Path)
}
