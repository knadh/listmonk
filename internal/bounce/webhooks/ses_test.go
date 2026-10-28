package webhooks

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/knadh/listmonk/models"
)

const testSigningCertURL = "https://sns.ap-southeast-1.amazonaws.com/SimpleNotificationService-010a507c1833636cd94bdb98bd93083a.pem"

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// newTestSES returns an SES instance whose certificate fetches are served a
// freshly generated test certificate through a stubbed http.DefaultTransport,
// along with the private key used to sign test notifications. The full
// signature verification path (cert URL regex, PEM fetch, X.509 parse,
// RSA-SHA1 check) still runs against the real code.
func newTestSES(t *testing.T) (*SES, *rsa.PrivateKey) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("error generating RSA key: %v", err)
	}

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "sns.ap-southeast-1.amazonaws.com"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("error creating test certificate: %v", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})

	orig := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != testSigningCertURL {
			return nil, fmt.Errorf("unexpected HTTP request in test: %s", r.URL)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{},
			Body:       io.NopCloser(strings.NewReader(string(certPEM))),
			Request:    r,
		}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = orig })

	return NewSES(), key
}

// signedSESNotification wraps message (the SES payload JSON) in a signed SNS
// Notification envelope and returns the marshalled webhook request body.
func signedSESNotification(t *testing.T, s *SES, key *rsa.PrivateKey, message string) []byte {
	t.Helper()

	n := sesNotif{
		Message:          message,
		MessageId:        "16505c47-4d38-5d39-9fe2-4f9d77269f9b",
		SignatureVersion: "1",
		SigningCertURL:   testSigningCertURL,
		Subject:          "Amazon SES Email Event Notification",
		Timestamp:        "2026-10-28T11:54:33.000Z",
		TopicArn:         "arn:aws:sns:ap-southeast-1:123456789012:ses-events",
		Type:             "Notification",
	}

	sum := sha1.Sum(s.buildSignature(n))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA1, sum[:])
	if err != nil {
		t.Fatalf("error signing test notification: %v", err)
	}
	n.Signature = base64.StdEncoding.EncodeToString(sig)

	b, err := json.Marshal(n)
	if err != nil {
		t.Fatalf("error marshalling SNS notification: %v", err)
	}
	return b
}

func TestSESProcessBounce(t *testing.T) {
	// Complaint delivered via SES configuration-set event publishing
	// (eventType instead of notificationType).
	eventComplaint := `{
		"eventType": "Complaint",
		"complaint": {
			"complainedRecipients": [{"emailAddress": "complaint@simulator.amazonses.com"}],
			"complaintFeedbackType": "abuse",
			"timestamp": "2026-10-28T11:54:32.438Z"
		},
		"mail": {
			"timestamp": "2026-10-28T11:54:30.007Z",
			"destination": ["complaint@simulator.amazonses.com"],
			"headersTruncated": false
		}
	}`

	eventBounce := `{
		"eventType": "Bounce",
		"bounce": {
			"bounceType": "Permanent",
			"bouncedRecipients": [{"emailAddress": "bounce@simulator.amazonses.com", "status": "5.1.1"}]
		},
		"mail": {
			"timestamp": "2026-10-28T11:54:31.007Z",
			"destination": ["bounce@simulator.amazonses.com"],
			"headersTruncated": false
		}
	}`

	notifComplaint := `{
		"notificationType": "Complaint",
		"complaint": {
			"complainedRecipients": [{"emailAddress": "complaint@simulator.amazonses.com"}],
			"complaintFeedbackType": "abuse",
			"timestamp": "2026-10-28T11:54:32.438Z"
		},
		"mail": {
			"timestamp": "2026-10-28T11:54:30.007Z",
			"destination": ["complaint@simulator.amazonses.com"],
			"headersTruncated": false
		}
	}`

	eventDelivery := `{
		"eventType": "Delivery",
		"delivery": {"timestamp": "2026-10-28T11:54:32.438Z"},
		"mail": {
			"timestamp": "2026-10-28T11:54:30.007Z",
			"destination": ["delivery@simulator.amazonses.com"],
			"headersTruncated": false
		}
	}`

	tests := []struct {
		name      string
		message   string
		wantType  string
		wantEmail string
		wantErr   string
	}{
		{
			name:      "configuration-set complaint event",
			message:   eventComplaint,
			wantType:  models.BounceTypeComplaint,
			wantEmail: "complaint@simulator.amazonses.com",
		},
		{
			name:      "configuration-set bounce event",
			message:   eventBounce,
			wantType:  models.BounceTypeHard,
			wantEmail: "bounce@simulator.amazonses.com",
		},
		{
			name:      "identity complaint notification",
			message:   notifComplaint,
			wantType:  models.BounceTypeComplaint,
			wantEmail: "complaint@simulator.amazonses.com",
		},
		{
			name:    "unrelated event type rejected",
			message: eventDelivery,
			wantErr: "notification type is not bounce",
		},
	}

	s, key := newTestSES(t)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b, err := s.ProcessBounce(signedSESNotification(t, s, key, tc.message))
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error %q, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ProcessBounce returned error: %v", err)
			}
			if b.Type != tc.wantType {
				t.Fatalf("expected bounce type %q, got %q", tc.wantType, b.Type)
			}
			if b.Email != tc.wantEmail {
				t.Fatalf("expected email %q, got %q", tc.wantEmail, b.Email)
			}
			if b.Source != "ses" {
				t.Fatalf("expected source %q, got %q", "ses", b.Source)
			}
		})
	}
}
