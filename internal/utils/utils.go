package utils

import (
	"crypto/rand"
	"net/mail"
	"net/url"
	"path"
	"strings"
)

// ValidateEmail validates whether the given string is a correctly formed e-mail address.
func ValidateEmail(email string) bool {
	// Since `mail.ParseAddress` parses an email address which can also contain an optional name component,
	// here we check if incoming email string is same as the parsed email.Address. So this eliminates
	// any valid email address with name and also valid address with empty name like `<abc@example.com>`.
	em, err := mail.ParseAddress(email)
	if err != nil || em.Address != email {
		return false
	}

	return true
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
