// Package urlsign cryptographically signs subscriber-facing URLs.
package urlsign

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var signedParams = [...]string{"auth", "exp", "l", "manage"}

// Signer signs a given URL.
type Signer struct {
	key      []byte
	enabled  bool
	ttl      time.Duration
	basePath string
	now      func() time.Time
}

// New returns a new Signer instance. basePath is the root URL's path prefix
// (eg: /newsletter) that is stripped before signing so that signatures only
// cover listmonk's canonical paths.
func New(key []byte, enabled bool, ttl time.Duration, basePath string) *Signer {
	return &Signer{key: append([]byte(nil), key...), enabled: enabled, ttl: ttl,
		basePath: strings.TrimSuffix(basePath, "/"), now: time.Now}
}

// Enabled reports whether links require signatures.
func (s *Signer) Enabled() bool {
	return s.enabled
}

// Sign adds an expiry and signature. Pass the verified parent link's expiry
// when generating follow-on links, so browsing cannot extend their lifetime.
func (s *Signer) Sign(raw string, expiry time.Time) string {
	if !s.enabled {
		return raw
	}

	return s.signURL(raw, expiry, false)
}

// SignAuth signs a link that authorizes the subscriber management page.
// Unlike Sign(), it signs even when signing is disabled and requires an expiry.
func (s *Signer) SignAuth(raw string, expiry time.Time) string {
	if expiry.IsZero() {
		return ""
	}

	return s.signURL(raw, expiry, true)
}

// Verify checks a URL and returns its validated expiry. The zero time means
// that there's no expiry (even when signing is disabled).
func (s *Signer) Verify(u *url.URL) (time.Time, bool) {
	if !s.enabled {
		return time.Time{}, true
	}

	exp, _, ok := s.verify(u)
	return exp, ok
}

// VerifyAuth checks a link signed with SignAuth(), even when signing is disabled,
// and returns its expiry.
func (s *Signer) VerifyAuth(u *url.URL) (time.Time, bool) {
	exp, q, ok := s.verify(u)
	if !ok || exp.IsZero() || q.Get("auth") != "1" {
		return time.Time{}, false
	}

	return exp, true
}

// GetExpiry returns the deadline for a new message. Zero time means no expiry.
func (s *Signer) GetExpiry() time.Time {
	if s.ttl == 0 {
		return time.Time{}
	}

	return s.now().Add(s.ttl)
}

// signURL adds the expiry, optional auth flag, and signature to the given URL.
func (s *Signer) signURL(raw string, expiry time.Time, auth bool) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}

	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return ""
	}

	var expires int64
	if !expiry.IsZero() {
		expires = expiry.Unix()
		if expires <= 0 {
			return ""
		}
	}

	// Only SignAuth() can grant auth.
	q.Del("auth")
	if auth {
		q.Set("auth", "1")
	}
	q.Set("exp", strconv.FormatInt(expires, 10))
	q.Set("sig", base64.RawURLEncoding.EncodeToString(s.sign(strings.TrimPrefix(u.EscapedPath(), s.basePath), q)))
	u.RawQuery = q.Encode()

	return u.String()
}

// verify checks the signature and expiry of a URL and returns its expiry and query.
func (s *Signer) verify(u *url.URL) (time.Time, url.Values, bool) {
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil || len(q["sig"]) != 1 || len(q["exp"]) != 1 {
		return time.Time{}, nil, false
	}

	expires, err := strconv.ParseInt(q.Get("exp"), 10, 64)
	if err != nil || expires < 0 || (expires != 0 && s.now().Unix() >= expires) {
		return time.Time{}, nil, false
	}

	sig, err := base64.RawURLEncoding.DecodeString(q.Get("sig"))
	if err != nil || !hmac.Equal(sig, s.sign(u.EscapedPath(), q)) {
		return time.Time{}, nil, false
	}
	if expires == 0 {
		return time.Time{}, q, true
	}

	return time.Unix(expires, 0), q, true
}

// sign signs the canonical listmonk path (without the root URL prefix) and known listmonk parameters.
func (s *Signer) sign(path string, q url.Values) []byte {
	signed := make(url.Values, len(signedParams))
	for _, key := range signedParams {
		signed[key] = q[key]
	}

	m := hmac.New(sha256.New, s.key)
	m.Write([]byte("listmonk-url-v2\n" + path + "?" + signed.Encode()))

	return m.Sum(nil)
}
