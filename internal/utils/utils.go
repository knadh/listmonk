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

// NormalizeDomains trims whitespace, converts domains to lowercase,
// and removes empty entries while preserving their original order.
func NormalizeDomains(domains []string) []string {
	normalized := make([]string, 0, len(domains))

	for _, domain := range domains {
		domain = strings.TrimSpace(strings.ToLower(domain))
		if domain != "" {
			normalized = append(normalized, domain)
		}
	}

	return normalized
}

// NormalizeFileExtensions trims whitespace, removes a leading dot,
// and converts file extensions to lowercase.
func NormalizeFileExtensions(extensions []string) []string {
	normalized := make([]string, len(extensions))

	for i, extension := range extensions {
		normalized[i] = strings.ToLower(
			strings.TrimPrefix(strings.TrimSpace(extension), "."),
		)
	}

	return normalized
}

// NormalizeTrustedURLs trims whitespace, removes empty entries,
// and validates that each URL uses HTTP or HTTPS.
// The wildcard "*" is accepted as a trusted URL entry.
func NormalizeTrustedURLs(trustedURLs []string) ([]string, error) {
	normalized := make([]string, 0, len(trustedURLs))

	for _, trustedURL := range trustedURLs {
		trustedURL = strings.TrimSpace(trustedURL)
		if trustedURL == "" {
			continue
		}

		if trustedURL == "*" {
			normalized = append(normalized, trustedURL)
			continue
		}

		parsedURL, err := url.Parse(trustedURL)
		if err != nil ||
			(parsedURL.Scheme != "http" && parsedURL.Scheme != "https") ||
			parsedURL.Host == "" {
			return nil, errors.New("invalid trusted URL: " + trustedURL)
		}

		normalized = append(normalized, trustedURL)
	}

	return normalized, nil
}
