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

type sendgridNotif struct {
	Email                string `json:"email"`
	Timestamp            int64  `json:"timestamp"`
	Event                string `json:"event"`
	BounceClassification string `json:"bounce_classification"`
	EventID              string `json:"sg_event_id"`
	MessageID            string `json:"sg_message_id"`
	Response             string `json:"response"`
	Reason               string `json:"reason"`
	Status               string `json:"status"`
	Attempt              any    `json:"attempt"`

	// SendGrid flattens all X-headers and adds them to the bounce
	// event notification.
	CampaignUUID   string `json:"XListmonkCampaign"`
	SubscriberUUID string `json:"XListmonkSubscriber"`
}

// Sendgrid handles Sendgrid/SNS webhook notifications including confirming SNS topic subscription
// requests and bounce notifications.
type Sendgrid struct {
	pubKey *ecdsa.PublicKey
}

// NewSendgrid returns a new Sendgrid instance.
func NewSendgrid(key string) (*Sendgrid, error) {
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

// ProcessEvents verifies and normalizes SendGrid delivery lifecycle events.
func (s *Sendgrid) ProcessEvents(sig, timestamp string, b []byte) ([]models.CampaignDeliveryEvent, int, error) {
	if err := s.verifyNotif(sig, timestamp, b); err != nil {
		return nil, 0, err
	}

	var notifs []sendgridNotif
	if err := json.Unmarshal(b, &notifs); err != nil {
		return nil, 0, fmt.Errorf("error unmarshalling Sendgrid notification: %v", err)
	}

	out := make([]models.CampaignDeliveryEvent, 0, len(notifs))
	unsupported := 0
	for _, n := range notifs {
		eventType := strings.ToLower(n.Event)
		switch eventType {
		case models.DeliveryEventProcessed,
			models.DeliveryEventDeferred,
			models.DeliveryEventDelivered,
			models.DeliveryEventBounce,
			models.DeliveryEventDropped:
		default:
			unsupported++
			continue
		}

		bounceType := ""
		if eventType == models.DeliveryEventBounce {
			bounceType = models.BounceTypeHard
			if n.BounceClassification == "technical" || n.BounceClassification == "content" {
				bounceType = models.BounceTypeSoft
			}
		}

		meta, err := json.Marshal(map[string]any{
			"attempt":               n.Attempt,
			"bounce_classification": n.BounceClassification,
			"reason":                n.Reason,
			"response":              n.Response,
			"status":                n.Status,
		})
		if err != nil {
			return nil, 0, fmt.Errorf("error encoding Sendgrid event metadata: %v", err)
		}

		out = append(out, models.CampaignDeliveryEvent{
			Provider:          "sendgrid",
			ProviderEventID:   n.EventID,
			ProviderMessageID: n.MessageID,
			EventType:         eventType,
			CampaignUUID:      n.CampaignUUID,
			SubscriberUUID:    n.SubscriberUUID,
			Email:             strings.ToLower(strings.TrimSpace(n.Email)),
			BounceType:        bounceType,
			Meta:              meta,
			OccurredAt:        time.Unix(n.Timestamp, 0),
		})
	}

	return out, unsupported, nil
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
