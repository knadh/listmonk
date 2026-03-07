package webhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/knadh/listmonk/models"
)

type lettermintNotif struct {
	ID        string `json:"id"`
	Event     string `json:"event"`
	Timestamp string `json:"timestamp"`
	Data      struct {
		MessageID string `json:"message_id"`
		Subject   string `json:"subject"`
		Recipient string `json:"recipient"`
		Response  *struct {
			StatusCode     int    `json:"status_code"`
			EnhancedStatus string `json:"enhanced_status_code"`
			Content        string `json:"content"`
		} `json:"response"`
		Metadata json.RawMessage `json:"metadata"`
		Tag      string          `json:"tag"`
	} `json:"data"`
}

// Lettermint handles bounce webhook notifications from Lettermint.
type Lettermint struct {
	hmacKey []byte
}

// NewLettermint returns a new Lettermint webhook handler.
func NewLettermint(key []byte) *Lettermint {
	return &Lettermint{hmacKey: key}
}

// ProcessBounce processes an incoming Lettermint webhook payload and returns a bounce object.
func (l *Lettermint) ProcessBounce(sig string, body []byte) ([]models.Bounce, error) {
	// Parse the signature header: t={timestamp},v1={hex_signature}.
	ts, sigHex, err := parseLettermintSignature(sig)
	if err != nil {
		return nil, err
	}

	// Verify timestamp tolerance (300 seconds).
	if math.Abs(float64(time.Now().Unix()-ts)) > 300 {
		return nil, fmt.Errorf("signature timestamp expired")
	}

	// Compute HMAC-SHA256 of "{timestamp}.{body}" and compare.
	mac := hmac.New(sha256.New, l.hmacKey)
	mac.Write([]byte(fmt.Sprintf("%d.%s", ts, body)))
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(sigHex)) {
		return nil, fmt.Errorf("invalid signature")
	}

	var n lettermintNotif
	if err := json.Unmarshal(body, &n); err != nil {
		return nil, fmt.Errorf("error unmarshalling Lettermint notification: %v", err)
	}

	// Map event to bounce type.
	var typ string
	switch n.Event {
	case "message.hard_bounced":
		typ = models.BounceTypeHard
	case "message.soft_bounced":
		typ = models.BounceTypeSoft
	case "message.spam_complaint":
		typ = models.BounceTypeComplaint
	default:
		// Ignore irrelevant events (e.g. webhook.test).
		return nil, nil
	}

	campUUID := ""
	if len(n.Data.Metadata) > 0 {
		var meta map[string]string
		if err := json.Unmarshal(n.Data.Metadata, &meta); err == nil {
			if v, ok := meta["X-Listmonk-Campaign"]; ok {
				campUUID = v
			}
		}
	}

	t, _ := time.Parse(time.RFC3339, n.Timestamp)
	if t.IsZero() {
		t = time.Now()
	}

	return []models.Bounce{{
		Email:        strings.ToLower(n.Data.Recipient),
		CampaignUUID: campUUID,
		Type:         typ,
		Source:       "lettermint",
		Meta:         json.RawMessage(body),
		CreatedAt:    t,
	}}, nil
}

// parseLettermintSignature parses a signature header of the form "t={timestamp},v1={hex}".
func parseLettermintSignature(sig string) (int64, string, error) {
	var (
		ts   int64
		hash string
	)

	for _, part := range strings.Split(sig, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			if _, err := fmt.Sscanf(kv[1], "%d", &ts); err != nil {
				return 0, "", fmt.Errorf("invalid timestamp in signature: %v", err)
			}
		case "v1":
			hash = kv[1]
		}
	}

	if ts == 0 || hash == "" {
		return 0, "", fmt.Errorf("invalid signature format")
	}

	return ts, hash, nil
}