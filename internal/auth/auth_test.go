package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// newTestOIDCProvider starts a server that serves an OIDC discovery document,
// optionally advertising PKCE support.
func newTestOIDCProvider(t *testing.T, pkce bool) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		doc := map[string]any{
			"issuer":                                srv.URL,
			"authorization_endpoint":                srv.URL + "/authorize",
			"token_endpoint":                        srv.URL + "/token",
			"jwks_uri":                              srv.URL + "/jwks.json",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		}
		if pkce {
			doc["code_challenge_methods_supported"] = []string{"S256"}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(doc)
	})

	return srv
}

func newTestAuth(providerURL string) *Auth {
	return &Auth{
		cfg: Config{OIDC: OIDCConfig{
			Enabled:     true,
			ProviderURL: providerURL,
			ClientID:    "listmonk",
			RedirectURL: "http://localhost:9000/auth/oidc",
		}},
		log: log.New(io.Discard, "", 0),
	}
}

// TestGetOIDCAuthURLWithPKCE tests that a PKCE challenge is sent to a provider that
// supports it, and that the returned verifier is the one the challenge was derived from.
func TestGetOIDCAuthURLWithPKCE(t *testing.T) {
	srv := newTestOIDCProvider(t, true)

	authURL, verifier := newTestAuth(srv.URL).GetOIDCAuthURL("state123", "nonce123")
	if verifier == "" {
		t.Fatal("expected a code verifier for a provider that supports PKCE")
	}

	u, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("error parsing auth URL: %v", err)
	}
	q := u.Query()

	if got := q.Get("code_challenge_method"); got != "S256" {
		t.Errorf("code_challenge_method = %q, want S256", got)
	}

	sum := sha256.Sum256([]byte(verifier))
	if want := base64.RawURLEncoding.EncodeToString(sum[:]); q.Get("code_challenge") != want {
		t.Errorf("code_challenge = %q, want %q (S256 of the verifier)", q.Get("code_challenge"), want)
	}

	// The existing params should be untouched.
	if got := q.Get("nonce"); got != "nonce123" {
		t.Errorf("nonce = %q, want nonce123", got)
	}
	if got := q.Get("state"); got != "state123" {
		t.Errorf("state = %q, want state123", got)
	}
}

// TestGetOIDCAuthURLWithoutPKCE tests that nothing is sent to a provider that doesn't
// advertise PKCE support.
func TestGetOIDCAuthURLWithoutPKCE(t *testing.T) {
	srv := newTestOIDCProvider(t, false)

	authURL, verifier := newTestAuth(srv.URL).GetOIDCAuthURL("state123", "nonce123")
	if verifier != "" {
		t.Errorf("got a code verifier %q for a provider that doesn't support PKCE", verifier)
	}

	u, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("error parsing auth URL: %v", err)
	}

	for _, p := range []string{"code_challenge", "code_challenge_method"} {
		if got := u.Query().Get(p); got != "" {
			t.Errorf("%s = %q, want it to be absent", p, got)
		}
	}
}
