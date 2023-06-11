package webhooks

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/knadh/listmonk/models"
)

type msg91Notif struct {
	ID              int           `json:"id"`
	UserID          int           `json:"user_id"`
	DomainID        int           `json:"domain_id"`
	RecipientID     int           `json:"recipient_id"`
	OutboundEmailID int           `json:"outbound_email_id"`
	EventID         int           `json:"event_id"`
	OutBoundEmail   outboundEmail `json:"outbound_email"`
	Event           event         `json:"event"`
}

type event struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}
type outboundEmail struct {
	ID        int        `json:"id"`
	UserID    int        `json:"user_id"`
	DomainID  int        `json:"domain_id"`
	ThreadID  int        `json:"thread_id"`
	To        []reciever `json:"to"`
	MessageID string     `json:"message_id"`
	CreatedAt time.Time  `json:"created_at"`
}
type reciever struct {
	Name  string `json:"name"`
	EMail string `json:"email"`
}

// Msg91 handles msg91/SNS webhook notifications including bounces
// requests and bounce notifications.
type Msg91 struct{}

// NewMSG91 returns a new Msg91 instance.
func NewMSG91() (*Msg91, error) {
	// Get the certificate from the key.
	sigB, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return nil, err
	}

	pubKey, err := x509.ParsePKIXPublicKey(sigB)
	if err != nil {
		return nil, err
	}

	return &Sendgrid{pubKey: pubKey.(*ecdsa.PublicKey)}, nil
}

// ProcessBounce processes Sendgrid bounce notifications and returns one or more Bounce objects.
func (s *Sendgrid) ProcessBounce(sig, timestamp string, b []byte) ([]models.Bounce, error) {
	if err := s.verifyNotif(sig, timestamp, b); err != nil {
		return nil, err
	}

	var notifs []sendgridNotif
	if err := json.Unmarshal(b, &notifs); err != nil {
		return nil, fmt.Errorf("error unmarshalling Sendgrid notification: %v", err)
	}

	out := make([]models.Bounce, 0, len(notifs))
	for _, n := range notifs {
		if n.Event != "bounce" {
			continue
		}

		typ := models.BounceTypeHard
		if n.BounceClassification == "technical" || n.BounceClassification == "content" {
			typ = models.BounceTypeSoft
		}

		tstamp := time.Unix(n.Timestamp, 0)
		bn := models.Bounce{
			CampaignUUID: n.CampaignUUID,
			Email:        strings.ToLower(n.Email),
			Type:         typ,
			Meta:         json.RawMessage(b),
			Source:       "sendgrid",
			CreatedAt:    tstamp,
		}
		out = append(out, bn)
	}

	return out, nil
}

// verifyNotif verifies the signature on a notification payload.
func (s *Sendgrid) verifyNotif(sig, timestamp string, b []byte) error {
	sigB, err := base64.StdEncoding.DecodeString(sig)
	if err != nil {
		return err
	}

	ecdsaSig := struct {
		R *big.Int
		S *big.Int
	}{}

	if _, err := asn1.Unmarshal(sigB, &ecdsaSig); err != nil {
		return fmt.Errorf("error asn1 unmarshal of signature: %v", err)
	}

	h := sha256.New()
	h.Write([]byte(timestamp))
	h.Write(b)
	hash := h.Sum(nil)

	if !ecdsa.Verify(s.pubKey, hash, ecdsaSig.R, ecdsaSig.S) {
		return errors.New("invalid signature")
	}

	return nil
}
