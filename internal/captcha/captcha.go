package captcha

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/altcha-org/altcha-lib-go"
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
		challenge, err := altcha.CreateChallenge(altcha.ChallengeOptions{
			Algorithm:  altcha.SHA256,
			MaxNumber:  int64(c.altcha.Complexity),
			SaltLength: 12,
			HMACKey:    c.altcha.HMACKey,
		})
		if err != nil {
			return "", fmt.Errorf("failed to create Altcha challenge: %w", err)
		}

		challengeJSON, err := json.Marshal(challenge)
		if err != nil {
			return "", fmt.Errorf("failed to marshal Altcha challenge: %w", err)
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
	valid, err := altcha.VerifySolution(payload, c.altcha.HMACKey, false)
	if err != nil {
		return fmt.Errorf("failed to verify captcha solution: %w", err), false
	}

	if !valid {
		return fmt.Errorf("captcha verification failed"), false
	}

	return nil, true
}
