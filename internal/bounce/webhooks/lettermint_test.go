package webhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/knadh/listmonk/models"
)

func signLettermintTest(body []byte, key []byte) string {
	ts := time.Now().Unix()
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(fmt.Sprintf("%d.%s", ts, body)))
	return fmt.Sprintf("t=%d,v1=%s", ts, hex.EncodeToString(mac.Sum(nil)))
}

func TestLettermintCampaignHeaders(t *testing.T) {
	key := []byte("test-webhook-key")
	tests := []struct {
		name     string
		data     string
		campaign string
	}{
		{"mixed case", `"headers":[{"name":"x-LiStMoNk-CaMpAiGn","value":"campaign-header"}],"metadata":{}`, "campaign-header"},
		{"first nonempty repeated value", `"headers":[{"name":"X-Listmonk-Campaign","value":" "},{"name":"x-listmonk-campaign","value":" campaign-first "},{"name":"X-Listmonk-Campaign","value":"campaign-second"}],"metadata":{}`, "campaign-first"},
		{"header takes precedence", `"headers":[{"name":"X-Listmonk-Campaign","value":"campaign-header"}],"metadata":{"X-Listmonk-Campaign":"campaign-metadata"}`, "campaign-header"},
		{"legacy metadata", `"metadata":{"X-Listmonk-Campaign":"campaign-legacy"}`, "campaign-legacy"},
		{"empty header fallback", `"headers":[{"name":"X-Listmonk-Campaign","value":""}],"metadata":{"X-Listmonk-Campaign":"campaign-fallback"}`, "campaign-fallback"},
		{"unrelated headers", `"headers":[{"name":"X-Listmonk-Subscriber","value":"subscriber"}],"metadata":{}`, ""},
		{"null legacy metadata", `"metadata":null`, ""},
		{"missing snapshot", `"headers":[],"metadata":{}`, ""},
		{"invalid metadata does not hide header", `"headers":[{"name":"X-Listmonk-Campaign","value":"campaign-header"}],"metadata":{"other":1}`, "campaign-header"},
	}
	events := []struct {
		event string
		typ   string
	}{
		{"message.hard_bounced", models.BounceTypeHard},
		{"message.soft_bounced", models.BounceTypeSoft},
		{"message.spam_complaint", models.BounceTypeComplaint},
	}
	for _, event := range events {
		for _, tt := range tests {
			t.Run(event.event+"/"+tt.name, func(t *testing.T) {
				body := []byte(fmt.Sprintf(`{"event":%q,"timestamp":"2026-10-04T10:00:00Z","data":{"recipient":"USER@example.test",%s}}`, event.event, tt.data))
				bounces, err := NewLettermint(key).ProcessBounce(signLettermintTest(body, key), body)
				if err != nil {
					t.Fatal(err)
				}
				if len(bounces) != 1 {
					t.Fatalf("got %d bounces", len(bounces))
				}
				got := bounces[0]
				if got.CampaignUUID != tt.campaign || got.Email != "user@example.test" || got.Type != event.typ || got.Source != "lettermint" {
					t.Fatalf("unexpected bounce: %+v", got)
				}
				if string(got.Meta) != string(body) {
					t.Fatal("raw payload changed")
				}
			})
		}
	}
}

func TestLettermintCampaignHeaderRequiresValidSignature(t *testing.T) {
	key := []byte("test-webhook-key")
	body := []byte(`{"event":"message.hard_bounced","data":{"recipient":"user@example.test","headers":[{"name":"X-Listmonk-Campaign","value":"campaign"}]}}`)
	signature := signLettermintTest(body, key)
	body = append(body, ' ')
	if _, err := NewLettermint(key).ProcessBounce(signature, body); err == nil {
		t.Fatal("accepted modified payload")
	}
}
