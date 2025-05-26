package webhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	// "os" // Uncomment for file-based tests
	// "path/filepath" // Uncomment for file-based tests
	"testing"
	"time"

	"github.com/knadh/listmonk/models"
	"github.com/stretchr/testify/assert"
	// "github.com/stretchr/testify/require" // Uncomment for file-based tests
)

const (
	testMailgunWebhookKey = "test-mailgun-key"
)

// Helper to create a valid Mailgun webhook payload string
func makeMailgunPayload(eventData mailgunEventData, signature mailgunSignature) string {
	payload := mailgunWebhookPayload{
		EventData: eventData,
		Signature: signature,
	}
	b, _ := json.Marshal(payload)
	return string(b)
}

// Helper to generate a valid signature for given timestamp and token
func generateTestMailgunSignature(timestamp, token string) string {
	// Using direct HMAC logic as VerifySignature might not be easily callable or might have side effects.
	mac := hmac.New(sha256.New, []byte(testMailgunWebhookKey))
	mac.Write([]byte(timestamp))
	mac.Write([]byte(token))
	return hex.EncodeToString(mac.Sum(nil))
}

func TestMailgun_ProcessBounce(t *testing.T) {
	mg := NewMailgun(testMailgunWebhookKey)

	campaignUUID := "d2c1c2e0-b0f6-4ba0-a189-f795f7c77060"
	recipientEmail := "subscriber@example.com"

	// Use a fixed time for "now" across tests for reproducible CreatedAt, but parse it to avoid timezone issues in assertion.
	baseTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	unixNow := float64(baseTime.Unix())
	tsString := fmt.Sprintf("%d", baseTime.Unix()) // Timestamp as string for signature
	token := "testtoken123"

	// Common signature for many valid tests
	validSig := mailgunSignature{
		Timestamp: tsString,
		Token:     token,
		Signature: generateTestMailgunSignature(tsString, token),
	}

	tests := []struct {
		name         string
		payload      string
		wantErr      bool
		wantBounces  []models.Bounce
		skipSigCheck bool // To test scenarios where signature might be empty or verification skipped
		mgInstance   *Mailgun // To allow using different Mailgun instances (e.g., one with no key)
	}{
		{
			name: "valid permanent failure event",
			payload: makeMailgunPayload(
				mailgunEventData{
					Event:     mailgunEventFailed,
					Severity:  "permanent",
					Recipient: recipientEmail,
					Timestamp: unixNow,
					UserVariables: map[string]string{
						models.EmailHeaderCampaignUUID: campaignUUID,
					},
					Message: mailgunMessage{Headers: mailgunMessageHeaders{MessageID: "test-msg-id"}},
				},
				validSig,
			),
			wantErr: false,
			wantBounces: []models.Bounce{{
				Email:        recipientEmail,
				CampaignUUID: campaignUUID,
				Type:         models.BounceTypeHard,
				Source:       "mailgun",
				CreatedAt:    baseTime,
			}},
		},
		{
			name: "valid temporary failure event",
			payload: makeMailgunPayload(
				mailgunEventData{
					Event:     mailgunEventFailed,
					Severity:  "temporary",
					Recipient: recipientEmail,
					Timestamp: unixNow,
					UserVariables: map[string]string{
						models.EmailHeaderCampaignUUID: campaignUUID,
					},
				},
				validSig,
			),
			wantErr: false,
			wantBounces: []models.Bounce{{
				Email:        recipientEmail,
				CampaignUUID: campaignUUID,
				Type:         models.BounceTypeSoft,
				Source:       "mailgun",
				CreatedAt:    baseTime,
			}},
		},
		{
			name: "valid complaint event",
			payload: makeMailgunPayload(
				mailgunEventData{
					Event:     mailgunEventComplained,
					Recipient: recipientEmail,
					Timestamp: unixNow,
					UserVariables: map[string]string{
						models.EmailHeaderCampaignUUID: campaignUUID,
					},
				},
				validSig,
			),
			wantErr: false,
			wantBounces: []models.Bounce{{
				Email:        recipientEmail,
				CampaignUUID: campaignUUID,
				Type:         models.BounceTypeComplaint,
				Source:       "mailgun",
				CreatedAt:    baseTime,
			}},
		},
		{
			name: "valid legacy bounced event (hard)",
			payload: makeMailgunPayload(
				mailgunEventData{
					Event:     mailgunEventBounced,
					Recipient: recipientEmail,
					Code:      550, // Example hard bounce code
					Timestamp: unixNow,
					UserVariables: map[string]string{
						models.EmailHeaderCampaignUUID: campaignUUID,
					},
				},
				validSig,
			),
			wantErr: false,
			wantBounces: []models.Bounce{{
				Email:        recipientEmail,
				CampaignUUID: campaignUUID,
				Type:         models.BounceTypeHard,
				Source:       "mailgun",
				CreatedAt:    baseTime,
			}},
		},
		{
			name: "valid legacy bounced event (soft)",
			payload: makeMailgunPayload(
				mailgunEventData{
					Event:     mailgunEventBounced,
					Recipient: recipientEmail,
					Code:      450, // Example soft bounce code
					Timestamp: unixNow,
					UserVariables: map[string]string{
						models.EmailHeaderCampaignUUID: campaignUUID,
					},
				},
				validSig,
			),
			wantErr: false,
			wantBounces: []models.Bounce{{
				Email:        recipientEmail,
				CampaignUUID: campaignUUID,
				Type:         models.BounceTypeSoft,
				Source:       "mailgun",
				CreatedAt:    baseTime,
			}},
		},
		{
			name: "invalid signature",
			payload: makeMailgunPayload(
				mailgunEventData{Event: mailgunEventFailed, Severity: "permanent", Recipient: recipientEmail, Timestamp: unixNow},
				mailgunSignature{Timestamp: tsString, Token: token, Signature: "invalid-signature"},
			),
			wantErr: true,
		},
		{
			name:    "malformed json",
			payload: "{not a json",
			wantErr: true,
		},
		{
			name: "ignored event type (delivered)",
			payload: makeMailgunPayload(
				mailgunEventData{Event: mailgunEventDelivered, Recipient: recipientEmail, Timestamp: unixNow},
				validSig,
			),
			wantErr:     false,
			wantBounces: nil,
		},
		{
			name: "missing recipient",
			payload: makeMailgunPayload(
				mailgunEventData{
					Event:     mailgunEventFailed,
					Severity:  "permanent",
					// Recipient: recipientEmail, // Missing
					Timestamp: unixNow,
				},
				validSig,
			),
			wantErr: false, // Should process but email will be empty
			wantBounces: []models.Bounce{{
				Email:        "", // Expect empty email
				CampaignUUID: "", // No campaign UUID if no user-vars
				Type:         models.BounceTypeHard,
				Source:       "mailgun",
				CreatedAt:    baseTime,
			}},
		},
		{
			name: "missing campaign UUID from user-variables",
			payload: makeMailgunPayload(
				mailgunEventData{
					Event:     mailgunEventFailed,
					Severity:  "permanent",
					Recipient: recipientEmail,
					Timestamp: unixNow,
					// UserVariables: map[string]string{models.EmailHeaderCampaignUUID: campaignUUID}, // Missing
				},
				validSig,
			),
			wantErr: false,
			wantBounces: []models.Bounce{{
				Email:        recipientEmail,
				CampaignUUID: "", // Expect empty campaign UUID
				Type:         models.BounceTypeHard,
				Source:       "mailgun",
				CreatedAt:    baseTime,
			}},
		},
		{
			name: "signature verification skipped (no key on Mailgun instance)",
			mgInstance: NewMailgun(""), // Instance with no key
			payload: makeMailgunPayload( // Payload has a signature, but it won't be checked
				mailgunEventData{
					Event:     mailgunEventFailed,
					Severity:  "permanent",
					Recipient: recipientEmail,
					Timestamp: unixNow,
				},
				mailgunSignature{Timestamp: tsString, Token: token, Signature: "any-signature-as-it-wont-be-checked"},
			),
			wantErr: false,
			wantBounces: []models.Bounce{{
				Email:        recipientEmail,
				CampaignUUID: "",
				Type:         models.BounceTypeHard,
				Source:       "mailgun",
				CreatedAt:    baseTime,
			}},
		},
		{
			name: "missing signature block in payload",
			payload: `{"event-data": {"event": "failed", "recipient": "test@example.com"}}`, // No "signature" field
			wantErr: true, // Signature verification should fail due to missing components
		},
		{
			name: "missing event-data block in payload",
			payload: `{"signature": {"timestamp": "123", "token": "abc", "signature": "def"}}`, // No "event-data" field
			wantErr: true, // JSON unmarshalling of event-data will fail or lead to nil pointer
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			currentMg := mg
			if tt.mgInstance != nil { // Allow overriding the Mailgun instance for specific tests
				currentMg = tt.mgInstance
			}

			bounces, err := currentMg.ProcessBounce([]byte(tt.payload))
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// For Meta, just check if it's present as it contains the raw payload
				// For CreatedAt, if wantBounces has zero time, copy from actual for comparison if actual is non-zero
				for i := range bounces {
					assert.NotNil(t, bounces[i].Meta, "Meta should not be nil for bounce %d", i)
					bounces[i].Meta = nil // Nil out meta before deep comparison
					if tt.wantBounces != nil && i < len(tt.wantBounces) {
						// Handle CreatedAt comparison carefully due to potential time.Time nuances
						// If expected CreatedAt is specifically set (non-zero), compare it.
						// Otherwise, if it's zero, we might not want to enforce a specific time if the test doesn't care.
						// The baseTime approach should make this consistent.
						// No, we always expect baseTime if a bounce is generated.
						// If tt.wantBounces[i].CreatedAt.IsZero() && !bounces[i].CreatedAt.IsZero() {
						//     tt.wantBounces[i].CreatedAt = bounces[i].CreatedAt
						// }
					}
				}
				assert.Equal(t, tt.wantBounces, bounces)
			}
		})
	}
}

// Test with a more complete example payload, potentially loaded from a file
func TestMailgun_ProcessBounce_ExampleFile(t *testing.T) {
	// This is a placeholder for a test that would load a realistic JSON payload from a file
	// For example, create a file `testdata/mailgun_bounce_permanent.json`
	// payloadBytes, err := os.ReadFile(filepath.Join("testdata", "mailgun_bounce_permanent.json"))
	// require.NoError(t, err)
	//
	// var payload mailgunWebhookPayload
	// err = json.Unmarshal(payloadBytes, &payload)
	// require.NoError(t, err)
	//
	// mg := NewMailgun(testMailgunWebhookKey)
	//
	// // Manually set/override signature for testing as file content is static
	// ts := payload.Signature.Timestamp
	// token := payload.Signature.Token
	// payload.Signature.Signature = generateTestMailgunSignature(ts, token)
	//
	// modifiedPayloadBytes, err := json.Marshal(payload)
	// require.NoError(t, err)
	//
	// bounces, err := mg.ProcessBounce(modifiedPayloadBytes)
	// require.NoError(t, err)
	// require.Len(t, bounces, 1)
	//
	// assert.Equal(t, models.BounceTypeHard, bounces[0].Type)
	// assert.Equal(t, "recipient@example.com", bounces[0].Email) // update with actual email from file
	// assert.Equal(t, "expected-campaign-uuid", bounces[0].CampaignUUID) // update
}
