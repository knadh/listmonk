package webhooks

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/knadh/listmonk/models"
)

const testMailgunKey = "test-signing-key"

type mailgunTestPayload struct {
	Signature struct {
		Timestamp string `json:"timestamp"`
		Token     string `json:"token"`
		Signature string `json:"signature"`
	} `json:"signature"`
	EventData struct {
		Event         string            `json:"event"`
		Severity      string            `json:"severity,omitempty"`
		Timestamp     float64           `json:"timestamp"`
		Recipient     string            `json:"recipient"`
		Message       mailgunMessage    `json:"message"`
		UserVariables map[string]string `json:"user-variables,omitempty"`
	} `json:"event-data"`
}

func signMailgun(t *testing.T, key, timestamp, token string) string {
	t.Helper()

	mac := hmac.New(sha256.New, []byte(key))
	if _, err := mac.Write([]byte(timestamp + token)); err != nil {
		t.Fatalf("writing signature payload: %v", err)
	}

	return hex.EncodeToString(mac.Sum(nil))
}

func buildMailgunPayload(t *testing.T, key string, signedAt, eventAt time.Time, event, severity, recipient string, userVars, headers map[string]string) []byte {
	t.Helper()

	ts := strconv.FormatInt(signedAt.Unix(), 10)
	token := "test-token"

	var p mailgunTestPayload
	p.Signature.Timestamp = ts
	p.Signature.Token = token
	p.Signature.Signature = signMailgun(t, key, ts, token)
	p.EventData.Event = event
	p.EventData.Severity = severity
	p.EventData.Timestamp = float64(eventAt.Unix())
	p.EventData.Recipient = recipient
	p.EventData.Message = mailgunMessage{Headers: headers}
	p.EventData.UserVariables = userVars

	out, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	return out
}

func TestProcessBounce_HardBounce(t *testing.T) {
	eventAt := time.Unix(1700000000, 0)
	body := buildMailgunPayload(t, testMailgunKey, time.Now(), eventAt, "failed", "permanent", "Foo@Bar.COM", nil, nil)

	bs, err := NewMailgun([]byte(testMailgunKey)).ProcessBounce(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(bs) != 1 {
		t.Fatalf("expected one bounce, got %d", len(bs))
	}

	b := bs[0]
	if b.Type != models.BounceTypeHard {
		t.Fatalf("expected type %q, got %q", models.BounceTypeHard, b.Type)
	}
	if b.Source != "mailgun" {
		t.Fatalf("expected source mailgun, got %q", b.Source)
	}
	if b.Email != "foo@bar.com" {
		t.Fatalf("expected lowercased email, got %q", b.Email)
	}
	if b.CreatedAt.Unix() != eventAt.Unix() {
		t.Fatalf("expected created_at from event timestamp %v, got %v", eventAt, b.CreatedAt)
	}
	if !bytes.Equal(b.Meta, body) {
		t.Fatalf("expected meta to equal raw body")
	}
}

func TestProcessBounce_SoftBounce(t *testing.T) {
	body := buildMailgunPayload(t, testMailgunKey, time.Now(), time.Unix(1700000100, 0), "failed", "temporary", "test@example.com", nil, nil)

	bs, err := NewMailgun([]byte(testMailgunKey)).ProcessBounce(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(bs) != 1 {
		t.Fatalf("expected one bounce, got %d", len(bs))
	}

	b := bs[0]
	if b.Type != models.BounceTypeSoft {
		t.Fatalf("expected type %q, got %q", models.BounceTypeSoft, b.Type)
	}
	if b.Source != "mailgun" {
		t.Fatalf("expected source mailgun, got %q", b.Source)
	}
	if b.Email != "test@example.com" {
		t.Fatalf("unexpected email: %q", b.Email)
	}
	if !bytes.Equal(b.Meta, body) {
		t.Fatalf("expected meta to equal raw body")
	}
}

func TestProcessBounce_Complaint(t *testing.T) {
	body := buildMailgunPayload(t, testMailgunKey, time.Now(), time.Unix(1700000200, 0), "complained", "", "test@example.com", nil, nil)

	bs, err := NewMailgun([]byte(testMailgunKey)).ProcessBounce(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(bs) != 1 {
		t.Fatalf("expected one bounce, got %d", len(bs))
	}

	b := bs[0]
	if b.Type != models.BounceTypeComplaint {
		t.Fatalf("expected type %q, got %q", models.BounceTypeComplaint, b.Type)
	}
	if b.Source != "mailgun" {
		t.Fatalf("expected source mailgun, got %q", b.Source)
	}
	if b.Email != "test@example.com" {
		t.Fatalf("unexpected email: %q", b.Email)
	}
	if !bytes.Equal(b.Meta, body) {
		t.Fatalf("expected meta to equal raw body")
	}
}

func TestProcessBounce_IgnoredEvents(t *testing.T) {
	tests := []string{"delivered", "opened", "clicked", "unsubscribed", "accepted"}

	for _, event := range tests {
		t.Run(event, func(t *testing.T) {
			body := buildMailgunPayload(t, testMailgunKey, time.Now(), time.Unix(1700000300, 0), event, "", "test@example.com", nil, nil)

			bs, err := NewMailgun([]byte(testMailgunKey)).ProcessBounce(body)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(bs) != 0 {
				t.Fatalf("expected ignored event to return no bounces, got %d", len(bs))
			}
		})
	}
}

func TestProcessBounce_InvalidSignature(t *testing.T) {
	body := buildMailgunPayload(t, testMailgunKey, time.Now(), time.Now(), "failed", "permanent", "test@example.com", nil, nil)

	var p mailgunNotif
	if err := json.Unmarshal(body, &p); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	p.Signature.Signature = strings.Repeat("0", 64)

	tampered, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal tampered payload: %v", err)
	}

	bs, err := NewMailgun([]byte(testMailgunKey)).ProcessBounce(tampered)
	if err == nil {
		t.Fatalf("expected invalid signature error")
	}
	if len(bs) != 0 {
		t.Fatalf("expected no bounces on signature failure, got %d", len(bs))
	}
}

func TestProcessBounce_ExpiredTimestamp(t *testing.T) {
	body := buildMailgunPayload(t, testMailgunKey, time.Now().Add(-10*time.Minute), time.Now(), "failed", "permanent", "test@example.com", nil, nil)

	bs, err := NewMailgun([]byte(testMailgunKey)).ProcessBounce(body)
	if err == nil {
		t.Fatalf("expected expired timestamp error")
	}
	if len(bs) != 0 {
		t.Fatalf("expected no bounces on expired timestamp, got %d", len(bs))
	}
}

func TestProcessBounce_MalformedJSON(t *testing.T) {
	bs, err := NewMailgun([]byte(testMailgunKey)).ProcessBounce([]byte("{not json"))
	if err == nil {
		t.Fatalf("expected malformed json error")
	}
	if len(bs) != 0 {
		t.Fatalf("expected no bounces on malformed json, got %d", len(bs))
	}
}

func TestProcessBounce_MissingKey(t *testing.T) {
	body := buildMailgunPayload(t, testMailgunKey, time.Now(), time.Now(), "failed", "permanent", "test@example.com", nil, nil)

	tests := []struct {
		name string
		key  []byte
	}{
		{name: "nil", key: nil},
		{name: "empty", key: []byte{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bs, err := NewMailgun(tc.key).ProcessBounce(body)
			if err == nil {
				t.Fatalf("expected missing key error")
			}
			if len(bs) != 0 {
				t.Fatalf("expected no bounces when key is missing, got %d", len(bs))
			}
		})
	}
}

func TestProcessBounce_CampaignUUID(t *testing.T) {
	tests := []struct {
		name     string
		userVars map[string]string
		headers  map[string]string
		want     string
	}{
		{
			name:     "user-variables only",
			userVars: map[string]string{"X-Listmonk-Campaign": "camp-user"},
			want:     "camp-user",
		},
		{
			name:    "headers only",
			headers: map[string]string{"X-Listmonk-Campaign": "camp-header"},
			want:    "camp-header",
		},
		{
			name:     "user-variables take precedence",
			userVars: map[string]string{"X-Listmonk-Campaign": "camp-user"},
			headers:  map[string]string{"X-Listmonk-Campaign": "camp-header"},
			want:     "camp-user",
		},
		{
			name: "not present",
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := buildMailgunPayload(t, testMailgunKey, time.Now(), time.Now(), "failed", "permanent", "test@example.com", tc.userVars, tc.headers)

			bs, err := NewMailgun([]byte(testMailgunKey)).ProcessBounce(body)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(bs) != 1 {
				t.Fatalf("expected one bounce, got %d", len(bs))
			}
			if bs[0].CampaignUUID != tc.want {
				t.Fatalf("expected campaign uuid %q, got %q", tc.want, bs[0].CampaignUUID)
			}
		})
	}
}

func TestProcessBounce_EmailLowercased(t *testing.T) {
	body := buildMailgunPayload(t, testMailgunKey, time.Now(), time.Now(), "failed", "permanent", "Foo@Bar.COM", nil, nil)

	bs, err := NewMailgun([]byte(testMailgunKey)).ProcessBounce(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(bs) != 1 {
		t.Fatalf("expected one bounce, got %d", len(bs))
	}
	if bs[0].Email != "foo@bar.com" {
		t.Fatalf("expected lowercased email, got %q", bs[0].Email)
	}
}
