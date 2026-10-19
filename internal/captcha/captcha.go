package captcha

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/altcha-org/altcha-lib-go/v2"
	"github.com/knadh/listmonk/internal/tmptokens"
)

const (
	hCaptchaURL = "https://hcaptcha.com/siteverify"
)

type hCaptchaResp struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error_codes"`
}

const (
	ProviderNone     = ""
	ProviderHCaptcha = "hcaptcha"
	ProviderAltcha   = "altcha"
)

// Captcha is a captcha client supporting multiple providers.
type Captcha struct {
	provider string
	hCaptcha hCaptchaOpt
	altcha   altchaOpt
	client   *http.Client
}

type Opt struct {
	HCaptcha struct {
		Enabled bool   `json:"enabled"`
		Key     string `json:"key"`
		Secret  string `json:"secret"`
	} `json:"hcaptcha"`
	Altcha struct {
		Enabled    bool `json:"enabled"`
		Complexity int  `json:"complexity"`
	} `json:"altcha"`
}

type hCaptchaOpt struct {
	Secret string
}

type altchaOpt struct {
	Complexity int
	HMACKey    string
}

// New returns a new instance of the CAPTCHA client.
func New(o Opt) *Captcha {
	timeout := time.Second * 5

	c := &Captcha{
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConnsPerHost:   10,
				MaxConnsPerHost:       100,
				ResponseHeaderTimeout: timeout,
				IdleConnTimeout:       timeout,
			},
		},
	}

	// Determine which provider is enabled
	if o.Altcha.Enabled {
		c.provider = ProviderAltcha

		// Generate an random HMAC key for Altcha.
		b := make([]byte, 24) // 24 bytes will give 32 characters when base64 encoded
		_, err := rand.Read(b)
		if err != nil {
			panic(fmt.Sprintf("error generating Altcha HMAC key: %v", err))
		}
		hmacKey := base64.URLEncoding.EncodeToString(b)[:32]

		c.altcha = altchaOpt{
			Complexity: o.Altcha.Complexity,
			HMACKey:    hmacKey,
		}
	} else if o.HCaptcha.Enabled {
		c.provider = ProviderHCaptcha
		c.hCaptcha = hCaptchaOpt{
			Secret: o.HCaptcha.Secret,
		}
	}

	return c
}

// IsEnabled returns true if any captcha provider is enabled.
func (c *Captcha) IsEnabled() bool {
	return c.provider != ProviderNone
}

// GetProvider returns the active captcha provider.
func (c *Captcha) GetProvider() string {
	return c.provider
}

// GenerateChallenge generates a challenge for the active provider.
// For hCaptcha, this returns empty string as challenges are generated client-side.
// For Altcha, this returns a JSON challenge.
func (c *Captcha) GenerateChallenge() (string, error) {
	switch c.provider {
	case ProviderAltcha:
		if c.altcha.Complexity < 0 {
			c.altcha.Complexity = 30000
		}

		// Keep complexity as the maximum number of SHA-256 attempts.
		// The complexity param from Altcha v1 is gone, so retrofit the
		// value in user settings to the next config.
		n, err := rand.Int(rand.Reader, big.NewInt(int64(c.altcha.Complexity)+1))
		if err != nil {
			return "", fmt.Errorf("error generating Altcha counter: %w", err)
		}

		counter := int(n.Int64())
		exp := time.Now().Add(5 * time.Minute)
		challenge, err := altcha.CreateChallenge(altcha.CreateChallengeOptions{
			Algorithm:           string(altcha.SHA256),
			Cost:                1,
			Counter:             &counter,
			DeriveKey:           altcha.DeriveKeySHA(),
			HMACSignatureSecret: c.altcha.HMACKey,
			ExpiresAt:           &exp,
		})
		if err != nil {
			return "", fmt.Errorf("error creating Altcha challenge: %w", err)
		}

		challengeJSON, err := json.Marshal(challenge)
		if err != nil {
			return "", fmt.Errorf("error marshalling Altcha challenge: %w", err)
		}

		return string(challengeJSON), nil
	case ProviderHCaptcha:
		// hCaptcha generates challenges client-side.
		return "", nil
	default:
		return "", fmt.Errorf("no captcha provider enabled")
	}
}

// Verify verifies a CAPTCHA response.
func (c *Captcha) Verify(token string) (error, bool) {
	switch c.provider {
	case ProviderAltcha:
		return c.verifyAltcha(token)
	case ProviderHCaptcha:
		return c.verifyHCaptcha(token)
	default:
		return fmt.Errorf("no captcha provider enabled"), false
	}
}

// verifyHCaptcha verifies an hCaptcha response.
func (c *Captcha) verifyHCaptcha(token string) (error, bool) {
	resp, err := c.client.PostForm(hCaptchaURL, url.Values{
		"secret":   {c.hCaptcha.Secret},
		"response": {token},
	})
	if err != nil {
		return err, false
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err, false
	}

	var r hCaptchaResp
	if err := json.Unmarshal(body, &r); err != nil {
		return err, true
	}

	if !r.Success {
		return fmt.Errorf("hCaptcha failed: %s", strings.Join(r.ErrorCodes, ",")), false
	}

	return nil, true
}

// verifyAltcha verifies an Altcha response.
func (c *Captcha) verifyAltcha(payload string) (error, bool) {
	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return fmt.Errorf("error decoding captcha solution: %w", err), false
	}

	var solution altcha.Payload
	if err := json.Unmarshal(data, &solution); err != nil {
		return fmt.Errorf("error parsing captcha solution: %w", err), false
	}

	result, err := altcha.VerifySolution(altcha.VerifySolutionOptions{
		Challenge:           solution.Challenge,
		Solution:            solution.Solution,
		DeriveKey:           altcha.DeriveKeySHA(),
		HMACSignatureSecret: c.altcha.HMACKey,
	})
	if err != nil {
		return fmt.Errorf("error verifying captcha solution: %w", err), false
	}

	if !result.Verified || result.Expired {
		return fmt.Errorf("captcha verification failed"), false
	}

	// Disallow token reuse.
	id := "altcha:" + solution.Challenge.Signature
	if _, err := tmptokens.Check(id); err == nil {
		return fmt.Errorf("captcha token already used"), false
	}
	tmptokens.Set(id, 5*time.Minute, nil)

	return nil, true
}
