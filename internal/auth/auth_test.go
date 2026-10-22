package auth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const (
	oidcTestClientID = "listmonk-test"
	oidcTestNonce    = "testnonce"
	oidcTestSubject  = "user-123"
)

// fakeOIDCProvider is a hermetic OIDC IdP serving discovery, JWKS, token, and
// userinfo endpoints from a single httptest server. ID tokens are signed with
// a real RSA key so the oidc verifier path is exercised end to end.
type fakeOIDCProvider struct {
	srv          *httptest.Server
	key          *rsa.PrivateKey
	userinfoHits int32 // atomic.
}

func b64url(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

// signIDToken returns an RS256-signed JWT for the given claims.
func (p *fakeOIDCProvider) signIDToken(claims map[string]any) string {
	header, _ := json.Marshal(map[string]any{"alg": "RS256", "kid": "testkey", "typ": "JWT"})
	payload, _ := json.Marshal(claims)
	unsigned := b64url(header) + "." + b64url(payload)
	sum := sha256.Sum256([]byte(unsigned))
	sig, _ := rsa.SignPKCS1v15(rand.Reader, p.key, crypto.SHA256, sum[:])
	return unsigned + "." + b64url(sig)
}

// newFakeOIDCProvider starts a fake IdP whose ID token carries idTokenClaims
// (on top of the required iss/sub/aud/exp/iat/nonce claims) and whose userinfo
// endpoint returns userInfoClaims.
func newFakeOIDCProvider(t *testing.T, idTokenClaims, userInfoClaims map[string]any) *fakeOIDCProvider {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating RSA key: %v", err)
	}
	p := &fakeOIDCProvider{key: key}

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"issuer":                 p.srv.URL,
			"authorization_endpoint": p.srv.URL + "/authorize",
			"token_endpoint":         p.srv.URL + "/token",
			"userinfo_endpoint":      p.srv.URL + "/userinfo",
			"jwks_uri":               p.srv.URL + "/keys",
		})
	})
	mux.HandleFunc("/keys", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]any{{
				"kty": "RSA",
				"kid": "testkey",
				"use": "sig",
				"alg": "RS256",
				"n":   b64url(key.N.Bytes()),
				"e":   b64url(big.NewInt(int64(key.E)).Bytes()),
			}},
		})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		claims := map[string]any{
			"iss":   p.srv.URL,
			"sub":   oidcTestSubject,
			"aud":   oidcTestClientID,
			"exp":   time.Now().Add(time.Hour).Unix(),
			"iat":   time.Now().Unix(),
			"nonce": oidcTestNonce,
		}
		for k, v := range idTokenClaims {
			claims[k] = v
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "test-access-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
			"id_token":     p.signIDToken(claims),
		})
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&p.userinfoHits, 1)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(userInfoClaims)
	})

	// Assign the server before it starts so the handlers (spawned by Start)
	// observe it.
	p.srv = httptest.NewUnstartedServer(mux)
	p.srv.Start()
	t.Cleanup(p.srv.Close)

	return p
}

// auth returns an Auth wired to the fake provider, skipping initOIDC.
func (p *fakeOIDCProvider) auth(t *testing.T) *Auth {
	t.Helper()

	provider, err := oidc.NewProvider(context.Background(), p.srv.URL)
	if err != nil {
		t.Fatalf("creating OIDC provider: %v", err)
	}

	return &Auth{
		oauthCfg: oauth2.Config{
			ClientID:    oidcTestClientID,
			Endpoint:    provider.Endpoint(),
			RedirectURL: "http://localhost/callback",
			Scopes:      []string{oidc.ScopeOpenID, "profile", "email"},
		},
		verifier: provider.Verifier(&oidc.Config{ClientID: oidcTestClientID}),
		provider: provider,
		log:      log.New(io.Discard, "", 0),
	}
}

func (p *fakeOIDCProvider) userinfoHitCount() int32 {
	return atomic.LoadInt32(&p.userinfoHits)
}

// TestExchangeOIDCTokenEmailVerified covers the OIDC email verification gate.
// `email_verified` is OPTIONAL per OIDC Core 5.1; when the IdP makes no
// assertion in the ID token, verification must be resolved from the userinfo
// endpoint instead of reading the zero-value false as a rejection.
func TestExchangeOIDCTokenEmailVerified(t *testing.T) {
	tests := []struct {
		name             string
		idTokenClaims    map[string]any
		userInfoClaims   map[string]any
		wantErr          string
		wantEmail        string
		wantVerified     bool
		wantUserinfoHits int32
	}{
		{
			// Regression test for #3297: providers like Okta org auth
			// servers put `email` in the ID token but only return
			// `email_verified` from the userinfo endpoint.
			name:             "absent email_verified falls back to userinfo",
			idTokenClaims:    map[string]any{"email": "user@example.com"},
			userInfoClaims:   map[string]any{"sub": oidcTestSubject, "email": "user@example.com", "email_verified": true},
			wantEmail:        "user@example.com",
			wantVerified:     true,
			wantUserinfoHits: 1,
		},
		{
			name:             "explicit email_verified false is rejected",
			idTokenClaims:    map[string]any{"email": "user@example.com", "email_verified": false},
			userInfoClaims:   map[string]any{"sub": oidcTestSubject},
			wantErr:          "OIDC email is not verified",
			wantUserinfoHits: 0,
		},
		{
			// The userinfo endpoint must not be consulted, let alone be
			// able to override, an explicit negative from the ID token.
			name:             "explicit email_verified false is not overridden by userinfo",
			idTokenClaims:    map[string]any{"email": "user@example.com", "email_verified": false},
			userInfoClaims:   map[string]any{"sub": oidcTestSubject, "email_verified": true},
			wantErr:          "OIDC email is not verified",
			wantUserinfoHits: 0,
		},
		{
			name:             "userinfo subject mismatch is rejected",
			idTokenClaims:    map[string]any{},
			userInfoClaims:   map[string]any{"sub": "somebody-else", "email": "user@example.com", "email_verified": true},
			wantErr:          "subject did not match",
			wantUserinfoHits: 1,
		},
		{
			// A missing e-mail still triggers the userinfo fallback even
			// when `email_verified` is present on the ID token.
			name:             "missing email still falls back when email_verified is present",
			idTokenClaims:    map[string]any{"email_verified": false},
			userInfoClaims:   map[string]any{"sub": oidcTestSubject, "email": "user@example.com", "email_verified": true},
			wantEmail:        "user@example.com",
			wantVerified:     true,
			wantUserinfoHits: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newFakeOIDCProvider(t, tt.idTokenClaims, tt.userInfoClaims)
			a := p.auth(t)

			rawIDTk, claims, err := a.ExchangeOIDCToken("test-code", oidcTestNonce, "")
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if rawIDTk == "" {
					t.Fatalf("expected raw ID token, got empty")
				}
				if claims.Email != tt.wantEmail {
					t.Fatalf("claims.Email = %q, want %q", claims.Email, tt.wantEmail)
				}
				if claims.EmailVerified != tt.wantVerified {
					t.Fatalf("claims.EmailVerified = %v, want %v", claims.EmailVerified, tt.wantVerified)
				}
			}
			if got := p.userinfoHitCount(); got != tt.wantUserinfoHits {
				t.Fatalf("userinfo hits = %d, want %d", got, tt.wantUserinfoHits)
			}
		})
	}
}
