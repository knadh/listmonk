package utils

import (
	"crypto/rand"
	"net/mail"
	"net/url"
	"path"
	"regexp"
	"strings"
)

var (
	reTitle = regexp.MustCompile(`(?s)<title\s*data-i18n\s*>(.+?)</title>`)
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

// GetTplSubject extracts any custom i18n subject rendered in the given rendered
// template body. If it's not found, the incoming subject and body are returned.
func GetTplSubject(subject string, body []byte) (string, []byte) {
	m := reTitle.FindSubmatch(body)
	if len(m) != 2 {
		return subject, body
	}

	return strings.TrimSpace(string(m[1])), reTitle.ReplaceAll(body, []byte(""))
}
