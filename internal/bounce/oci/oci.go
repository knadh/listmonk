// Package oci implements a "cron-based" bounce source that polls Oracle Cloud
// Infrastructure's email-delivery suppression list. OCI does not push bounces
// via HTTP webhooks; instead, the suppression list must be fetched (and
// optionally cleared) with signed REST calls. The signing scheme is OCI's
// custom HTTP Signature: RSA-SHA256 over a canonical "(request-target) date
// host" string, with keyId = "<tenancy>/<user>/<fingerprint>".
//
// The poller's Scan() method is meant to be invoked from a sleep-loop
// goroutine in the bounce manager, the same way the mailbox scanner is.
package oci

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/knadh/listmonk/models"
)

const (
	apiBasePath = "/20170907/suppressions"
	httpTimeout = 30 * time.Second
	sourceName  = "oci"
	reasonHard  = "HARDBOUNCE"
)

// Opt is the OCI poller configuration. Field names match the JSON keys stored
// under the `bounce.oci` settings row.
type Opt struct {
	Enabled           bool          `json:"enabled"`
	Host              string        `json:"host"`
	TenancyOCID       string        `json:"tenancy_ocid"`
	UserOCID          string        `json:"user_ocid"`
	Fingerprint       string        `json:"fingerprint"`
	PrivateKey        string        `json:"private_key"`
	CompartmentID     string        `json:"compartment_id"`
	DeleteAfterRecord bool          `json:"delete_after_record"`
	ScanInterval      time.Duration `json:"scan_interval"`
}

// ExistsFn reports whether a suppression entry with the given OCID has
// already been recorded locally. It is consulted only when DeleteAfterRecord
// is false — otherwise OCI itself is the source of truth and entries
// disappear as soon as we record them.
type ExistsFn func(ocid string) (bool, error)

// OCI is an Oracle Cloud Infrastructure suppression-list poller.
type OCI struct {
	opt    Opt
	key    *rsa.PrivateKey
	keyID  string
	client *http.Client
	exists ExistsFn
	log    *log.Logger
}

// suppression is one entry returned by the OCI suppression-list endpoint.
// Only the fields we use are decoded.
type suppression struct {
	ID           string `json:"id"`
	EmailAddress string `json:"emailAddress"`
	Reason       string `json:"reason"`
	TimeCreated  string `json:"timeCreated"`
}

// New validates the configuration, parses the PEM private key once, and
// returns a ready-to-use poller.
func New(opt Opt, exists ExistsFn, lo *log.Logger) (*OCI, error) {
	if opt.Host == "" || opt.TenancyOCID == "" || opt.UserOCID == "" ||
		opt.Fingerprint == "" || opt.CompartmentID == "" || opt.PrivateKey == "" {
		return nil, errors.New("oci: missing required configuration")
	}

	key, err := parsePrivateKey(opt.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("oci: parse private key: %w", err)
	}

	return &OCI{
		opt:    opt,
		key:    key,
		keyID:  opt.TenancyOCID + "/" + opt.UserOCID + "/" + opt.Fingerprint,
		client: &http.Client{Timeout: httpTimeout},
		exists: exists,
		log:    lo,
	}, nil
}

// Scan fetches the current suppression list from OCI, enqueues each entry as
// a bounce, and (if configured) deletes each entry from OCI.
func (o *OCI) Scan(ch chan<- models.Bounce) error {
	list, err := o.fetchSuppressions()
	if err != nil {
		return err
	}

	for _, s := range list {
		if s.EmailAddress == "" {
			continue
		}

		// When entries aren't deleted from OCI after recording, the same
		// suppression will resurface on every poll. Skip ones we've already
		// recorded so we don't keep re-inserting duplicate bounce rows.
		if !o.opt.DeleteAfterRecord && o.exists != nil {
			seen, err := o.exists(s.ID)
			if err != nil {
				o.log.Printf("oci: dedup lookup for %s failed: %v", s.ID, err)
			} else if seen {
				continue
			}
		}

		ch <- o.toBounce(s)

		if o.opt.DeleteAfterRecord {
			if err := o.deleteSuppression(s.ID); err != nil {
				o.log.Printf("oci: failed to delete suppression %s: %v", s.ID, err)
			}
		}
	}
	return nil
}

func (o *OCI) toBounce(s suppression) models.Bounce {
	typ := models.BounceTypeSoft
	if s.Reason == reasonHard {
		typ = models.BounceTypeHard
	}

	createdAt, err := time.Parse(time.RFC3339, s.TimeCreated)
	if err != nil {
		createdAt = time.Now()
	}

	meta, _ := json.Marshal(map[string]string{
		"ocid":        s.ID,
		"reason":      s.Reason,
		"timeCreated": s.TimeCreated,
	})

	return models.Bounce{
		Email:     strings.ToLower(s.EmailAddress),
		Type:      typ,
		Source:    sourceName,
		Meta:      meta,
		CreatedAt: createdAt,
	}
}

func (o *OCI) fetchSuppressions() ([]suppression, error) {
	path := apiBasePath + "?compartmentId=" + url.QueryEscape(o.opt.CompartmentID)

	resp, err := o.do(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("oci: GET %s returned %d: %s", path, resp.StatusCode, string(body))
	}

	// OCI returns either a raw array or, on some endpoints/versions, an object
	// wrapping `items`. Probe by peeking at the first non-whitespace byte.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	trimmed := bytesTrimSpace(body)
	if len(trimmed) > 0 && trimmed[0] == '{' {
		var wrap struct {
			Items []suppression `json:"items"`
		}
		if err := json.Unmarshal(body, &wrap); err != nil {
			return nil, fmt.Errorf("oci: decode suppressions object: %w", err)
		}
		return wrap.Items, nil
	}

	var list []suppression
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, fmt.Errorf("oci: decode suppressions array: %w", err)
	}
	return list, nil
}

func (o *OCI) deleteSuppression(id string) error {
	path := apiBasePath + "/" + url.PathEscape(id) + "?compartmentId=" + url.QueryEscape(o.opt.CompartmentID)

	resp, err := o.do(http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// do builds, signs, and dispatches a request. Body is nil for GET/DELETE,
// which is all the suppression-list endpoint requires.
func (o *OCI) do(method, path string, body io.Reader) (*http.Response, error) {
	u := "https://" + o.opt.Host + path

	req, err := http.NewRequest(method, u, body)
	if err != nil {
		return nil, err
	}

	date := time.Now().UTC().Format(http.TimeFormat)
	req.Header.Set("Date", date)
	req.Header.Set("Host", o.opt.Host)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", o.buildAuth(method, path, date))

	return o.client.Do(req)
}

// buildAuth signs the canonical "(request-target) date host" string and
// returns the value for the Authorization header.
func (o *OCI) buildAuth(method, path, date string) string {
	signingString := "(request-target): " + strings.ToLower(method) + " " + path + "\n" +
		"date: " + date + "\n" +
		"host: " + o.opt.Host

	digest := sha256.Sum256([]byte(signingString))
	sig, err := rsa.SignPKCS1v15(rand.Reader, o.key, crypto.SHA256, digest[:])
	if err != nil {
		// Signing only fails on programmer errors (wrong hash, broken key).
		// Returning an empty header makes the request 401 and surfaces the
		// problem in logs rather than silently retrying.
		o.log.Printf("oci: sign error: %v", err)
		return ""
	}

	return fmt.Sprintf(
		`Signature version="1",keyId="%s",algorithm="rsa-sha256",headers="(request-target) date host",signature="%s"`,
		o.keyID,
		base64.StdEncoding.EncodeToString(sig),
	)
}

func parsePrivateKey(pemStr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, errors.New("no PEM block found")
	}

	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	rsaKey, ok := k.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("not an RSA private key")
	}
	return rsaKey, nil
}

func bytesTrimSpace(b []byte) []byte {
	i := 0
	for i < len(b) && (b[i] == ' ' || b[i] == '\t' || b[i] == '\n' || b[i] == '\r') {
		i++
	}
	return b[i:]
}
