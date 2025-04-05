package webhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
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
	Status   any    `json:"status"`
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
	TruthSource     bool              `json:"truth_source"`
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

func (p *Forwardemail) ProcessBounce(sigHex string, body []byte) ([]models.Bounce, error) {
	// Decode the hex-encoded signature from the webhook
	sig, err := hex.DecodeString(sigHex)
	if err != nil {
		return nil, fmt.Errorf("invalid signature encoding: %v", err)
	}

	// Generate HMAC using the request body and secret key
	mac := hmac.New(sha256.New, p.hmacKey)
	mac.Write(body)
	expectedSignature := mac.Sum(nil)

	// Compare the generated signature with the provided signature
	if !hmac.Equal(expectedSignature, sig) {
		return nil, errors.New("invalid signature")
	}

	// Parse the JSON payload
	var n forwardemailNotif
	if err := json.Unmarshal(body, &n); err != nil {
		return nil, fmt.Errorf("error unmarshalling Forwardemail notification: %v", err)
	}

	// Categorize the bounce type
	typ := models.BounceTypeSoft
	hardBounceCategories := []string{"block", "recipient", "virus", "spam"}
	for _, category := range hardBounceCategories {
		if n.Bounce.Category == category {
			typ = models.BounceTypeHard
			break
		}
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
		Meta:         json.RawMessage(body),
		CreatedAt:    n.BouncedAt,
	}}, nil
}
