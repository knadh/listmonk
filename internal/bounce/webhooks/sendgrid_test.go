package webhooks

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/knadh/listmonk/models"
)

func newSignedSendGrid(t *testing.T, payload []byte) (*Sendgrid, string, string) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	publicKey, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}
	sg, err := NewSendgrid(base64.StdEncoding.EncodeToString(publicKey))
	if err != nil {
		t.Fatalf("create SendGrid verifier: %v", err)
	}

	timestamp := "1787040000"
	hash := sha256.Sum256(append([]byte(timestamp), payload...))
	r, s, err := ecdsa.Sign(rand.Reader, key, hash[:])
	if err != nil {
		t.Fatalf("sign payload: %v", err)
	}
	signature, err := asn1.Marshal(struct {
		R *big.Int
		S *big.Int
	}{R: r, S: s})
	if err != nil {
		t.Fatalf("marshal signature: %v", err)
	}

	return sg, base64.StdEncoding.EncodeToString(signature), timestamp
}

func TestSendGridProcessEvents(t *testing.T) {
	const campaignUUID = "d6da0074-1084-4aa1-9c62-65fde375d33c"
	const subscriberUUID = "07f04382-c46a-4f36-839c-9a8cb907ff22"
	payload := []byte(`[
		{"email":"User@Example.com","timestamp":1787040001,"event":"processed","sg_event_id":"event-1","sg_message_id":"message-1","XListmonkCampaign":"` + campaignUUID + `","XListmonkSubscriber":"` + subscriberUUID + `"},
		{"email":"user@example.com","timestamp":1787040002,"event":"deferred","sg_event_id":"event-2","sg_message_id":"message-1","attempt":"1","response":"temporarily unavailable","XListmonkCampaign":"` + campaignUUID + `","XListmonkSubscriber":"` + subscriberUUID + `"},
		{"email":"user@example.com","timestamp":1787040003,"event":"delivered","sg_event_id":"event-3","sg_message_id":"message-1","status":"250","XListmonkCampaign":"` + campaignUUID + `","XListmonkSubscriber":"` + subscriberUUID + `"},
		{"email":"user@example.com","timestamp":1787040004,"event":"bounce","sg_event_id":"event-4","sg_message_id":"message-2","bounce_classification":"technical","reason":"mailbox unavailable","XListmonkCampaign":"` + campaignUUID + `","XListmonkSubscriber":"` + subscriberUUID + `"},
		{"email":"user@example.com","timestamp":1787040005,"event":"dropped","sg_event_id":"event-5","sg_message_id":"message-3","reason":"suppressed","XListmonkCampaign":"` + campaignUUID + `","XListmonkSubscriber":"` + subscriberUUID + `"},
		{"email":"user@example.com","timestamp":1787040006,"event":"open","sg_event_id":"event-6","sg_message_id":"message-1","XListmonkCampaign":"` + campaignUUID + `"}
	]`)

	sg, signature, timestamp := newSignedSendGrid(t, payload)
	events, unsupported, err := sg.ProcessEvents(signature, timestamp, payload)
	if err != nil {
		t.Fatalf("ProcessEvents() error = %v", err)
	}
	if len(events) != 5 {
		t.Fatalf("ProcessEvents() returned %d events, want 5", len(events))
	}
	if unsupported != 1 {
		t.Fatalf("ProcessEvents() reported %d unsupported events, want 1", unsupported)
	}

	wantTypes := []string{
		models.DeliveryEventProcessed,
		models.DeliveryEventDeferred,
		models.DeliveryEventDelivered,
		models.DeliveryEventBounce,
		models.DeliveryEventDropped,
	}
	for i, event := range events {
		if event.EventType != wantTypes[i] {
			t.Errorf("event %d type = %q, want %q", i, event.EventType, wantTypes[i])
		}
		if event.CampaignUUID != campaignUUID || event.SubscriberUUID != subscriberUUID {
			t.Errorf("event %d correlation metadata was not preserved", i)
		}
		if event.Email != "user@example.com" {
			t.Errorf("event %d email = %q, want normalized address", i, event.Email)
		}
		if string(event.Meta) == "" || !json.Valid(event.Meta) {
			t.Errorf("event %d metadata is invalid JSON", i)
		}
	}
	if events[3].BounceType != models.BounceTypeSoft {
		t.Errorf("technical bounce type = %q, want soft", events[3].BounceType)
	}
}

func TestSendGridProcessEventsRejectsInvalidSignature(t *testing.T) {
	payload := []byte(`[{"event":"delivered"}]`)
	sg, _, timestamp := newSignedSendGrid(t, payload)

	if _, _, err := sg.ProcessEvents(base64.StdEncoding.EncodeToString([]byte("invalid")), timestamp, payload); err == nil {
		t.Fatal("ProcessEvents() expected invalid signature error")
	}
}

func TestSendGridProcessEventsRejectsMalformedJSON(t *testing.T) {
	payload := []byte(`not-json`)
	sg, signature, timestamp := newSignedSendGrid(t, payload)

	if _, _, err := sg.ProcessEvents(signature, timestamp, payload); err == nil {
		t.Fatal("ProcessEvents() expected JSON error")
	}
}

func TestSendGridProcessEventsAllowsMissingCorrelation(t *testing.T) {
	payload := []byte(`[
		{"email":"user@example.com","timestamp":1787040001,"event":"delivered","sg_event_id":"event-1"},
		{"email":"Bounced@Example.com","timestamp":1787040002,"event":"bounce","sg_event_id":"event-2","bounce_classification":"invalid"}
	]`)
	sg, signature, timestamp := newSignedSendGrid(t, payload)

	events, unsupported, err := sg.ProcessEvents(signature, timestamp, payload)
	if err != nil {
		t.Fatalf("ProcessEvents() error = %v", err)
	}
	if len(events) != 2 || events[0].CampaignUUID != "" || events[0].SubscriberUUID != "" {
		t.Fatalf("missing correlation metadata was not preserved as empty: %#v", events)
	}
	if events[1].EventType != models.DeliveryEventBounce || events[1].Email != "bounced@example.com" {
		t.Fatalf("uncorrelated bounce was not preserved for email fallback: %#v", events[1])
	}
	if events[1].BounceType != models.BounceTypeHard {
		t.Fatalf("uncorrelated bounce type = %q, want hard", events[1].BounceType)
	}
	if unsupported != 0 {
		t.Fatalf("ProcessEvents() reported %d unsupported events, want 0", unsupported)
	}
}
