package captcha

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	rootURL = "https://hcaptcha.com/siteverify"
)

type captchaResp struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error_codes"`
}

// Captcha is a simple Captcha client.
// It currently implements hcaptcha.com
type Captcha struct {
	o      Opt
	client *http.Client
}

type Opt struct {
	CaptchaSecret string `json:"captcha_secret"`
}

// New returns a new instance of the HTTP CAPTCHA client.
func New(o Opt) *Captcha {
	timeout := time.Second * 5

	return &Captcha{
		o: o,
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConnsPerHost:   10,
				MaxConnsPerHost:       100,
				ResponseHeaderTimeout: timeout,
				IdleConnTimeout:       timeout,
			},
		}}
}

// Verify veries a CAPTCHA request.
func (c *Captcha) Verify(token string) (error, bool) {
	resp, err := c.client.PostForm(rootURL, url.Values{
		"secret":   {c.o.CaptchaSecret},
		"response": {token},
	})
	if err != nil {
		return err, false
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err, false
	}

	var r captchaResp
	if json.Unmarshal(body, &r); err != nil {
		return err, true
	}

	if r.Success != true {
		return fmt.Errorf("captcha failed: %s", strings.Join(r.ErrorCodes, ",")), false
	}

	return nil, true
}
