package webhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/knadh/listmonk/models"
)

const mailgunSignatureToleranceSeconds = 300

type mailgunNotif struct {
	Signature struct {
		Timestamp string `json:"timestamp"`
		Token     string `json:"token"`
		Signature string `json:"signature"`
	} `json:"signature"`
	EventData struct {
		Event         string            `json:"event"`
		Severity      string            `json:"severity"`
		Timestamp     float64           `json:"timestamp"`
		Recipient     string            `json:"recipient"`
		Message       mailgunMessage    `json:"message"`
		UserVariables map[string]string `json:"user-variables"`
	} `json:"event-data"`
}

type mailgunMessage struct {
	Headers map[string]string `json:"headers"`
}

// Mailgun handles bounce webhook notifications from Mailgun.
type Mailgun struct {
	hmacKey []byte
}

// NewMailgun returns a new Mailgun webhook handler.
func NewMailgun(key []byte) *Mailgun {
	return &Mailgun{hmacKey: key}
}

// ProcessBounce processes an incoming Mailgun webhook payload and returns bounce objects.
func (m *Mailgun) ProcessBounce(body []byte) ([]models.Bounce, error) {
	if len(m.hmacKey) == 0 {
		return nil, fmt.Errorf("webhook key is not configured")
	}

	var n mailgunNotif
	if err := json.Unmarshal(body, &n); err != nil {
		return nil, fmt.Errorf("error unmarshalling Mailgun notification: %v", err)
	}

	sigTS, err := strconv.ParseInt(strings.TrimSpace(n.Signature.Timestamp), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid signature timestamp: %v", err)
	}

	if math.Abs(float64(time.Now().Unix()-sigTS)) > mailgunSignatureToleranceSeconds {
		return nil, fmt.Errorf("signature timestamp expired")
	}

	providedSig, err := hex.DecodeString(strings.TrimSpace(n.Signature.Signature))
	if err != nil {
		return nil, fmt.Errorf("invalid signature encoding: %v", err)
	}

	mac := hmac.New(sha256.New, m.hmacKey)
	mac.Write([]byte(n.Signature.Timestamp + n.Signature.Token))
	if !hmac.Equal(mac.Sum(nil), providedSig) {
		return nil, fmt.Errorf("invalid signature")
	}

	var typ string
	switch n.EventData.Event {
	case "failed":
		switch n.EventData.Severity {
		case "permanent":
			typ = models.BounceTypeHard
		case "temporary":
			typ = models.BounceTypeSoft
		default:
			return nil, nil
		}
	case "complained":
		typ = models.BounceTypeComplaint
	default:
		return nil, nil
	}

	campUUID := ""
	if v, ok := n.EventData.UserVariables["X-Listmonk-Campaign"]; ok {
		campUUID = v
	} else if v, ok := n.EventData.Message.Headers["X-Listmonk-Campaign"]; ok {
		campUUID = v
	}

	sec, dec := math.Modf(n.EventData.Timestamp)
	createdAt := time.Unix(int64(sec), int64(dec*float64(time.Second)))
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	return []models.Bounce{{
		Email:        strings.ToLower(n.EventData.Recipient),
		CampaignUUID: campUUID,
		Type:         typ,
		Source:       "mailgun",
		Meta:         json.RawMessage(body),
		CreatedAt:    createdAt,
	}}, nil
}
