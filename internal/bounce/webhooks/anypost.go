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

// Events are kept as raw JSON so each bounce's meta carries the event verbatim.
type anypostBatch struct {
	Events []json.RawMessage `json:"events"`
}

type anypostEvent struct {
	Type       string `json:"type"`
	OccurredAt string `json:"occurred_at"`
	Data       struct {
		Recipient string            `json:"recipient"`
		Headers   map[string]string `json:"headers"`
		Bounce    *struct {
			Type string `json:"type"`
		} `json:"bounce"`
		Suppression *struct {
			Reason string `json:"reason"`
		} `json:"suppression"`
	} `json:"data"`
}

// Anypost echoes customer X-headers with names lowercased and hyphens as
// underscores, so the campaign header arrives under this transformed key.
var anypostCampaignHeaderKey = strings.ReplaceAll(strings.ToLower(models.EmailHeaderCampaignUUID), "-", "_")

// Anypost handles bounce webhook notifications from Anypost.
type Anypost struct {
	hmacKey []byte
}

// NewAnypost returns a new Anypost webhook handler.
func NewAnypost(key []byte) *Anypost {
	return &Anypost{hmacKey: key}
}

// ProcessBounce processes an incoming Anypost webhook payload and returns
// bounce objects. A payload is a batch that may carry multiple events.
func (a *Anypost) ProcessBounce(sig string, body []byte) ([]models.Bounce, error) {
	if len(a.hmacKey) == 0 {
		return nil, fmt.Errorf("webhook key is not configured")
	}

	// Parse the signature header: t={timestamp},v1={hex}. One v1 per active
	// signing secret (two mid-rotation); a match on any one passes.
	ts, sigs, err := parseAnypostSignature(sig)
	if err != nil {
		return nil, err
	}

	// Verify timestamp tolerance (300 seconds).
	if math.Abs(float64(time.Now().Unix()-ts)) > 300 {
		return nil, fmt.Errorf("signature timestamp expired")
	}

	// Compute HMAC-SHA256 of "{timestamp}.{body}" and compare against each
	// v1 component.
	mac := hmac.New(sha256.New, a.hmacKey)
	mac.Write([]byte(fmt.Sprintf("%d.%s", ts, body)))
	expected := mac.Sum(nil)

	matched := false
	for _, s := range sigs {
		sigB, err := hex.DecodeString(strings.TrimSpace(s))
		if err != nil {
			continue
		}
		if hmac.Equal(expected, sigB) {
			matched = true
			break
		}
	}
	if !matched {
		return nil, fmt.Errorf("invalid signature")
	}

	var batch anypostBatch
	if err := json.Unmarshal(body, &batch); err != nil {
		return nil, fmt.Errorf("error unmarshalling Anypost notification: %v", err)
	}

	out := make([]models.Bounce, 0, len(batch.Events))
	for _, raw := range batch.Events {
		var ev anypostEvent
		if err := json.Unmarshal(raw, &ev); err != nil {
			// Signature already verified, so a parse failure is contract
			// drift: fail the batch (logged + retried) rather than ack and
			// silently drop the event.
			return nil, fmt.Errorf("error unmarshalling Anypost event: %v", err)
		}

		// Map event to bounce type.
		var typ string
		switch ev.Type {
		case "email.bounced":
			typ = models.BounceTypeHard
			if ev.Data.Bounce != nil && ev.Data.Bounce.Type == "transient" {
				typ = models.BounceTypeSoft
			}
		case "email.complained":
			typ = models.BounceTypeComplaint
		case "email.suppressed":
			// Anypost dropped this send because the address was already
			// suppressed. Map only the bounce-like reasons; unsubscribed and
			// manual are opt-outs, not bounces, so skip them.
			if ev.Data.Suppression == nil {
				continue
			}
			switch ev.Data.Suppression.Reason {
			case "permanent_bounce":
				typ = models.BounceTypeHard
			case "complaint":
				typ = models.BounceTypeComplaint
			default:
				continue
			}
		default:
			// Ignore irrelevant events (deliveries, opens, clicks etc.).
			continue
		}

		if ev.Data.Recipient == "" {
			continue
		}

		campUUID := ev.Data.Headers[anypostCampaignHeaderKey]

		t, _ := time.Parse(time.RFC3339, ev.OccurredAt)
		if t.IsZero() {
			t = time.Now()
		}

		out = append(out, models.Bounce{
			Email:        strings.ToLower(ev.Data.Recipient),
			CampaignUUID: campUUID,
			Type:         typ,
			Source:       "anypost",
			Meta:         raw,
			CreatedAt:    t,
		})
	}

	return out, nil
}

// parseAnypostSignature parses a signature header of the form
// "t={timestamp},v1={hex}[,v1={hex}]" and returns the timestamp and all
// v1 components.
func parseAnypostSignature(sig string) (int64, []string, error) {
	var (
		ts     int64
		hashes []string
	)

	for _, part := range strings.Split(sig, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			if _, err := fmt.Sscanf(kv[1], "%d", &ts); err != nil {
				return 0, nil, fmt.Errorf("invalid timestamp in signature: %v", err)
			}
		case "v1":
			hashes = append(hashes, kv[1])
		}
	}

	if ts == 0 || len(hashes) == 0 {
		return 0, nil, fmt.Errorf("invalid signature format")
	}

	return ts, hashes, nil
}
