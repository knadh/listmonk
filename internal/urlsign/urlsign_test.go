package urlsign

import (
	"encoding/base64"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestSignVerify(t *testing.T) {
	s := New([]byte(strings.Repeat("k", 32)), true, 0, "")
	raw := "https://example.com/news/subscription/optin/subscriber?l=list1&l=list2"
	signed := s.Sign(raw, s.GetExpiry())

	f := func(raw string, want bool) {
		t.Helper()

		u, err := url.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}

		if _, got := s.Verify(u); got != want {
			t.Fatalf("Verify(%q) = %v, want %v", raw, got, want)
		}
	}

	f(signed, true)
	f(raw, false)

	f(strings.Replace(signed, "subscriber", "other", 1), false)
	f(strings.Replace(signed, "/optin/", "/wipe/", 1), false)
	f(strings.Replace(signed, "list2", "list3", 1), false)
	f(strings.Replace(signed, "l=list1&", "", 1), false)
	f(signed+"&l=list3", false)
	f(signed+"&manage=true", false)
	f(signed+"&utm_source=newsletter&custom=value", true)
	f(signed+"&confirm=true", true)

	f(signed+"&sig=bad", false)
	f(signed+"&bad=%zz", false)
	f(signed+"&bad=x;y", false)

	// Paths must be retained even if proxies rewrite the URLs.
	f(strings.Replace(signed, "https://example.com", "http://localhost:9000", 1), true)
	f(strings.Replace(signed, "/news/", "/", 1), false)

	u, _ := url.Parse(signed)
	if _, ok := New([]byte(strings.Repeat("x", 32)), true, 0, "").Verify(u); ok {
		t.Fatal("wrong key accepted")
	}

	if _, ok := New([]byte(strings.Repeat("k", 32)), true, 0, "").Verify(u); !ok {
		t.Fatal("same key did not survive reconstruction")
	}

	if s.Sign(signed, s.GetExpiry()) != signed {
		t.Fatal("signing is not idempotent")
	}
}

func TestUnrelatedQueryParams(t *testing.T) {
	s := New([]byte(strings.Repeat("k", 32)), true, time.Hour, "")
	raw := "https://example.com/subscription/campaign/subscriber?manage=true"
	expiry := s.GetExpiry()
	base, _ := url.Parse(s.Sign(raw, expiry))
	u, _ := url.Parse(s.Sign(raw+"&utm_source=email&custom=original", expiry))
	if u.Query().Get("sig") != base.Query().Get("sig") || u.Query().Get("custom") != "original" {
		t.Fatal("unrelated parameters affected signing or were lost")
	}

	q := u.Query()
	q.Set("utm_source", "forwarded")
	q.Del("custom")
	q.Add("utm_source", "another")
	u.RawQuery = q.Encode()
	if _, ok := s.Verify(u); !ok {
		t.Fatal("changing unrelated parameters invalidated the link")
	}

	for _, values := range [][]string{nil, {"false"}, {"true", "false"}} {
		q["manage"] = values
		u.RawQuery = q.Encode()
		if _, ok := s.Verify(u); ok {
			t.Fatalf("accepted changed manage parameter: %v", values)
		}
	}
}

func TestDisabled(t *testing.T) {
	s := New(nil, false, 0, "")

	f := func(raw string) {
		t.Helper()

		if got := s.Sign(raw, s.GetExpiry()); got != raw {
			t.Fatalf("disabled signing changed %q to %q", raw, got)
		}

		u, err := url.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}

		if expiry, ok := s.Verify(u); !ok || !expiry.IsZero() {
			t.Fatalf("disabled verification rejected %q", raw)
		}
	}

	f("https://example.com/subscription/campaign/subscriber")
	f("https://example.com/subscription/optin/subscriber?l=list1&l=list2")
	f("https://example.com/subscription/campaign/subscriber?sig=old-signature")
}

func TestExpiry(t *testing.T) {
	now := time.Unix(2000000000, 0)
	s := New([]byte(strings.Repeat("k", 32)), true, time.Minute, "")
	s.now = func() time.Time { return now }

	raw := "https://example.com/subscription/campaign/subscriber"
	parent, _ := url.Parse(s.Sign(raw, s.GetExpiry()))
	expiry := now.Unix() + 60

	if parent.Query().Get("exp") != strconv.FormatInt(expiry, 10) {
		t.Fatal("incorrect expiry")
	}

	f := func(delta int64, want bool) {
		t.Helper()

		now = time.Unix(expiry+delta, 0)
		if _, got := s.Verify(parent); got != want {
			t.Fatalf("expiry boundary %d: got %v, want %v", delta, got, want)
		}
	}

	verifiedExpiry, ok := s.Verify(parent)
	if !ok || !verifiedExpiry.Equal(time.Unix(expiry, 0)) {
		t.Fatalf("incorrect verified expiry: %v, %v", verifiedExpiry, ok)
	}

	f(-1, true)
	f(0, false)
	f(1, false)

	// Derived links keep the original deadline.
	s.ttl = 24 * time.Hour
	child, _ := url.Parse(s.Sign("https://example.com/subscription/wipe/subscriber", verifiedExpiry))
	if _, ok := s.Verify(child); child.Query().Get("exp") != parent.Query().Get("exp") || ok {
		t.Fatal("child renewed expiry")
	}

	s.ttl = 0
	if _, ok := s.Verify(parent); ok {
		t.Fatal("TTL change revived an expired link")
	}

	forever, _ := url.Parse(s.Sign(raw, s.GetExpiry()))
	s.ttl = time.Minute
	now = now.Add(365 * 24 * time.Hour)

	if expiry, ok := s.Verify(forever); !ok || !expiry.IsZero() || forever.Query().Get("exp") != "0" {
		t.Fatal("non-expiring link expired")
	}
}

func TestInvalidExpiry(t *testing.T) {
	s := New([]byte(strings.Repeat("k", 32)), true, time.Hour, "")
	u, _ := url.Parse(s.Sign("https://example.com/subscription/campaign/subscriber", s.GetExpiry()))

	f := func(values ...string) {
		t.Helper()

		q := u.Query()
		q.Del("sig")
		if values == nil {
			q.Del("exp")
		} else {
			q["exp"] = values
		}

		// Invalid expiry should fail despite signature being correct.
		q.Set("sig", base64.RawURLEncoding.EncodeToString(s.sign(u.EscapedPath(), q)))

		altered := *u
		altered.RawQuery = q.Encode()
		if expiry, ok := s.Verify(&altered); ok || !expiry.IsZero() {
			t.Fatalf("accepted invalid expiry: %v", values)
		}
	}

	f()
	f("")
	f("invalid")
	f("1.5")
	f("-1")
	f("9223372036854775808")
	f("0", "0")
}

func TestTamperedExpiry(t *testing.T) {
	s := New([]byte(strings.Repeat("k", 32)), true, time.Hour, "")
	u, _ := url.Parse(s.Sign("https://example.com/subscription/campaign/subscriber", s.GetExpiry()))

	f := func(expiry string) {
		t.Helper()

		altered := *u
		q := altered.Query()
		q.Set("exp", expiry)
		altered.RawQuery = q.Encode()

		if expiry, ok := s.Verify(&altered); ok || !expiry.IsZero() {
			t.Fatalf("accepted tampered expiry: %q", expiry)
		}
	}

	f("0")
	f(strconv.FormatInt(s.GetExpiry().Add(time.Hour).Unix(), 10))
}

// Only the zero time may encode the non-expiring sentinel, not the Unix epoch
// or a timestamp that truncates to it.
func TestSignTimeExpiry(t *testing.T) {
	s := New([]byte(strings.Repeat("k", 32)), true, time.Hour, "")
	raw := "https://example.com/subscription/campaign/subscriber"
	for _, expiry := range []time.Time{time.Unix(-1, 0), time.Unix(0, 0), time.Unix(0, 500000000)} {
		if got := s.Sign(raw, expiry); got != "" {
			t.Fatalf("unrepresentable expiry %v generated a link: %s", expiry, got)
		}
	}

	expiry := time.Now().Add(time.Hour).Truncate(time.Second).Add(500 * time.Millisecond)
	u, err := url.Parse(s.Sign(raw, expiry))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := s.Verify(u)
	if !ok || !got.Equal(expiry.Truncate(time.Second)) {
		t.Fatalf("expiry did not round-trip at URL precision: %v, %v", got, ok)
	}
}

// TestSignBasePath checks that the root URL prefix is excluded from signatures,
// so links verify against the canonical path listmonk receives.
func TestSignBasePath(t *testing.T) {
	s := New([]byte(strings.Repeat("k", 32)), true, time.Hour, "/newsletter/")
	signed, err := url.Parse(s.Sign("https://example.com/newsletter/subscription/campaign/subscriber", s.GetExpiry()))
	if err != nil {
		t.Fatal(err)
	}

	// Path as received by listmonk (prefix stripped by the proxy).
	u := *signed
	u.Path = "/subscription/campaign/subscriber"
	if _, ok := s.Verify(&u); !ok {
		t.Fatal("canonical path did not verify")
	}

	// The prefixed path is not a listmonk route and should not verify.
	if _, ok := s.Verify(signed); ok {
		t.Fatal("prefixed path verified")
	}

	// Links signed under a different root prefix still verify.
	other := New([]byte(strings.Repeat("k", 32)), true, time.Hour, "/old")
	o, _ := url.Parse(other.Sign("https://example.com/old/subscription/campaign/subscriber", other.GetExpiry()))
	o.Path = "/subscription/campaign/subscriber"
	if _, ok := s.Verify(o); !ok {
		t.Fatal("link signed under another root prefix did not verify")
	}
}

func TestAuth(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	raw := "https://example.com/subscription/campaign/subscriber?manage=true"

	for _, enabled := range []bool{true, false} {
		s := New([]byte(strings.Repeat("k", 32)), enabled, 0, "")
		s.now = func() time.Time { return now }

		f := func(raw string, want bool) {
			t.Helper()

			u, _ := url.Parse(raw)
			if _, got := s.VerifyAuth(u); got != want {
				t.Fatalf("enabled=%v VerifyAuth(%q) = %v, want %v", enabled, raw, got, want)
			}
		}

		signed := s.SignAuth(raw, now.Add(time.Hour))
		f(signed, true)
		f(raw, false)
		f(raw+"&auth=1", false)
		f(strings.Replace(signed, "auth=1&", "", 1), false)
		f(strings.Replace(signed, "subscriber", "other", 1), false)

		// Regular signed links don't shouldn't accept auth.
		f(New([]byte(strings.Repeat("k", 32)), true, 0, "").Sign(raw+"&auth=1", now.Add(time.Hour)), false)

		// Auth links require an expiry and expire.
		if s.SignAuth(raw, time.Time{}) != "" {
			t.Fatal("auth link signed without expiry")
		}

		s.now = func() time.Time { return now.Add(time.Hour) }
		f(signed, false)
	}
}
