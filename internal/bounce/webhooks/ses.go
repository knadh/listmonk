package webhooks

import (
	"bytes"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/knadh/listmonk/models"
)

// AWS signature/validation logic borrowed from @cavnit's contrib:
// https://gist.github.com/cavnit/f4d63ba52b3aa05406c07dcbca2ca6cf

// https://sns.ap-southeast-1.amazonaws.com/SimpleNotificationService-010a507c1833636cd94bdb98bd93083a.pem
var sesRegCertURL = regexp.MustCompile(`(?i)^https://sns\.[a-z0-9\-]+\.amazonaws\.com(\.cn)?/SimpleNotificationService\-[a-z0-9]+\.pem$`)

// sesNotif is an individual notification wrapper posted by SNS.
type sesNotif struct {
	// Message may be a plaintext message or a stringified JSON payload based on the message type.
	// Four SES messages, this is the actual payload.
	Message string `json:"Message"`

	MessageId        string `json:"MessageId"`
	Signature        string `json:"Signature"`
	SignatureVersion string `json:"SignatureVersion"`
	SigningCertURL   string `json:"SigningCertURL"`
	Subject          string `json:"Subject"`
	Timestamp        string `json:"Timestamp"`
	Token            string `json:"Token"`
	TopicArn         string `json:"TopicArn"`
	Type             string `json:"Type"`
	SubscribeURL     string `json:"SubscribeURL"`
	UnsubscribeURL   string `json:"UnsubscribeURL"`
}

type sesTimestamp time.Time

type sesMail struct {
	EventType string `json:"eventType"`
	NotifType string `json:"notificationType"`
	Bounce    struct {
		BounceType        string `json:"bounceType"`
		BouncedRecipients []struct {
			Status string `json:"status"`
		} `json:"bouncedRecipients"`
	} `json:"bounce"`
	Mail struct {
		Timestamp        sesTimestamp        `json:"timestamp"`
		HeadersTruncated bool                `json:"headersTruncated"`
		Destination      []string            `json:"destination"`
		Headers          []map[string]string `json:"headers"`
	} `json:"mail"`
}

// SES handles SES/SNS webhook notifications including confirming SNS topic subscription
// requests and bounce notifications.
type SES struct {
	certs map[string]*x509.Certificate
}

// NewSES returns a new SES instance.
func NewSES() *SES {
	return &SES{
		certs: make(map[string]*x509.Certificate),
	}
}

// ProcessSubscription processes an SNS topic subscribe / unsubscribe notification
// by parsing and verifying the payload and calling the subscribe / unsubscribe URL.
func (s *SES) ProcessSubscription(b []byte) error {
	var n sesNotif
	if err := json.Unmarshal(b, &n); err != nil {
		return fmt.Errorf("error unmarshalling SNS notification: %v", err)
	}
	if err := s.verifyNotif(n); err != nil {
		return err
	}

	// Make an HTTP request to the sub/unsub URL.
	u := n.SubscribeURL
	if n.Type == "UnsubscriptionConfirmation" {
		u = n.UnsubscribeURL
	}

	resp, err := http.Get(u)
	if err != nil {
		return fmt.Errorf("error requesting subscription URL: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("non 200 response on subscription URL: %v", resp.StatusCode)
	}

	return nil
}

// ProcessBounce processes an SES bounce notification and returns a Bounce object.
func (s *SES) ProcessBounce(b []byte) (models.Bounce, error) {
	var (
		bounce models.Bounce
		n      sesNotif
	)
	if err := json.Unmarshal(b, &n); err != nil {
		return bounce, fmt.Errorf("error unmarshalling SES notification: %v", err)
	}
	if err := s.verifyNotif(n); err != nil {
		return bounce, err
	}

	var m sesMail
	if err := json.Unmarshal([]byte(n.Message), &m); err != nil {
		return bounce, fmt.Errorf("error unmarshalling SES notification: %v", err)
	}

	if (m.EventType != "" && m.EventType != "Bounce") ||
		(m.NotifType != "" && (m.NotifType != "Bounce" && m.NotifType != "Complaint")) {
		return bounce, errors.New("notification type is not bounce")
	}

	if len(m.Mail.Destination) == 0 {
		return bounce, errors.New("no destination e-mails found in SES notification")
	}

	typ := models.BounceTypeSoft
	if m.Bounce.BounceType == "Permanent" {
		typ = models.BounceTypeHard
	}
	if m.Bounce.BounceType == "Transient" && len(m.Bounce.BouncedRecipients) > 0 {
		// "Invalid domain" bounce.
		if m.Bounce.BouncedRecipients[0].Status == "5.4.4" {
			typ = models.BounceTypeHard
		}
	}
	if m.NotifType == "Complaint" {
		typ = models.BounceTypeComplaint
	}

	// Look for the campaign ID in headers.
	campUUID := ""
	if !m.Mail.HeadersTruncated {
		for _, h := range m.Mail.Headers {
			key, ok := h["name"]
			if !ok || key != models.EmailHeaderCampaignUUID {
				continue
			}

			campUUID, ok = h["value"]
			if !ok {
				continue
			}
			break
		}
	}

	return models.Bounce{
		Email:        strings.ToLower(m.Mail.Destination[0]),
		CampaignUUID: campUUID,
		Type:         typ,
		Source:       "ses",
		Meta:         json.RawMessage(n.Message),
		CreatedAt:    time.Time(m.Mail.Timestamp),
	}, nil
}

func (s *SES) buildSignature(n sesNotif) []byte {
	var b bytes.Buffer
	b.WriteString("Message" + "\n" + n.Message + "\n")
	b.WriteString("MessageId" + "\n" + n.MessageId + "\n")

	if n.Subject != "" {
		b.WriteString("Subject" + "\n" + n.Subject + "\n")
	}
	if n.SubscribeURL != "" {
		b.WriteString("SubscribeURL" + "\n" + n.SubscribeURL + "\n")
	}

	b.WriteString("Timestamp" + "\n" + n.Timestamp + "\n")

	if n.Token != "" {
		b.WriteString("Token" + "\n" + n.Token + "\n")
	}
	b.WriteString("TopicArn" + "\n" + n.TopicArn + "\n")
	b.WriteString("Type" + "\n" + n.Type + "\n")

	return b.Bytes()
}

// verifyNotif verifies the signature on a notification payload.
func (s *SES) verifyNotif(n sesNotif) error {
	// Get the message signing certificate.
	cert, err := s.getCert(n.SigningCertURL)
	if err != nil {
		return fmt.Errorf("error getting SNS cert: %v", err)
	}

	sign, err := base64.StdEncoding.DecodeString(n.Signature)
	if err != nil {
		return err
	}

	return cert.CheckSignature(x509.SHA1WithRSA, s.buildSignature(n), sign)
}

// getCert takes the SNS certificate URL and fetches it and caches it for the first time,
// and returns the cached cert for subsequent calls.
func (s *SES) getCert(certURL string) (*x509.Certificate, error) {
	// Ensure that the cert URL is Amazon's.
	u, err := url.Parse(certURL)
	if err != nil {
		return nil, err
	}
	if !sesRegCertURL.MatchString(certURL) {
		return nil, fmt.Errorf("invalid SNS certificate URL: %v", u.Host)
	}

	// Return if it's cached.
	if c, ok := s.certs[u.Path]; ok {
		return c, nil
	}

	// Fetch the certificate.
	resp, err := http.Get(certURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid SNS certificate URL: %v", u.Host)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	p, _ := pem.Decode(body)
	if p == nil {
		return nil, errors.New("invalid PEM")
	}

	cert, err := x509.ParseCertificate(p.Bytes)

	// Cache the cert in-memory.
	s.certs[u.Path] = cert

	return cert, err
}

func (st *sesTimestamp) UnmarshalJSON(b []byte) error {
	t, err := time.Parse("2006-01-02T15:04:05.999999999Z", strings.Trim(string(b), `"`))
	if err != nil {
		return err
	}
	*st = sesTimestamp(t)
	return nil
}
