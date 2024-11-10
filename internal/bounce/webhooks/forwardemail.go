package webhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/knadh/listmonk/models"
)

type BounceDetails struct {
	Action   string `json:"action"`
	Message  string `json:"message"`
	Category string `json:"category"`
	Code     int    `json:"code"`
	Status   string `json:"status"`
	Line     int    `json:"line"`
}

type forwardemailNotif struct {
	EmailID         string            `json:"email_id"`
	ListID          string            `json:"list_id"`
	ListUnsubscribe string            `json:"list_unsubscribe"`
	FeedbackID      string            `json:"feedback_id"`
	Recipient       string            `json:"recipient"`
	Message         string            `json:"message"`
	Response        string            `json:"response"`
	ResponseCode    int               `json:"response_code"`
	TruthSource     string            `json:"truth_source"`
	Headers         map[string]string `json:"headers"`
	Bounce          BounceDetails     `json:"bounce"`
	BouncedAt       time.Time         `json:"bounced_at"`
}

// Forwardemail handles webhook notifications (mainly bounce notifications).
type Forwardemail struct {
	hmacKey []byte
}

func NewForwardemail(key []byte) *Forwardemail {
	return &Forwardemail{hmacKey: key}
}

// ProcessBounce processes Forward Email bounce notifications and returns one object.
func (p *Forwardemail) ProcessBounce(sig, b []byte) ([]models.Bounce, error) {
	key := []byte(p.hmacKey)

	mac := hmac.New(sha256.New, key)

	mac.Write(b)

	signature := mac.Sum(nil)

	if subtle.ConstantTimeCompare(signature, []byte(sig)) != 1 {
		return nil, fmt.Errorf("invalid signature")
	}

	var n forwardemailNotif
	if err := json.Unmarshal(b, &n); err != nil {
		return nil, fmt.Errorf("error unmarshalling Forwardemail notification: %v", err)
	}

	typ := models.BounceTypeSoft
	// TODO: support `typ = models.BounceTypeComplaint` in future
	switch n.Bounce.Category {
	case "block", "recipient", "virus", "spam":
		typ = models.BounceTypeHard
	}

	campUUID := ""
	if v, ok := n.Headers["X-Listmonk-Campaign"]; ok {
		campUUID = v
	}

	return []models.Bounce{{
		Email:        strings.ToLower(n.Recipient),
		CampaignUUID: campUUID,
		Type:         typ,
		Source:       "forwardemail",
		Meta:         json.RawMessage(b),
		CreatedAt:    n.BouncedAt,
	}}, nil
}
