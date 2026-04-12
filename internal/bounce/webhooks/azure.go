package webhooks

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/knadh/listmonk/models"
)

var reSMTPStatus = regexp.MustCompile(`\b([245]\.\d\.\d)\b`)

type azureEvent struct {
	EventType string          `json:"eventType"`
	Data      azureEventData  `json:"data"`
	RawData   json.RawMessage `json:"-"`
}

type azureEventData struct {
	Recipient                string                     `json:"recipient"`
	MessageID                string                     `json:"messageId"`
	InternetMessageID        string                     `json:"internetMessageId"`
	Status                   string                     `json:"status"`
	DeliveryStatusDetails    azureDeliveryStatus        `json:"deliveryStatusDetails"`
	DeliveryAttemptTimestamp *time.Time                 `json:"deliveryAttemptTimestamp"`
	DeliveryAttemptTimeStamp *time.Time                 `json:"deliveryAttemptTimeStamp"`
	AdditionalData           map[string]json.RawMessage `json:"-"`
}

type azureDeliveryStatus struct {
	StatusMessage string `json:"statusMessage"`
}

// Azure handles webhook notifications from Azure Communication Services through Event Grid.
type Azure struct {
	sharedSecret       string
	sharedSecretHeader string
}

// NewAzure returns a new Azure webhook handler.
func NewAzure(sharedSecret, sharedSecretHeader string) *Azure {
	return &Azure{
		sharedSecret:       strings.TrimSpace(sharedSecret),
		sharedSecretHeader: strings.TrimSpace(sharedSecretHeader),
	}
}

// ProcessSubscription processes Event Grid subscription validation requests and
// returns the response payload that should be written to HTTP response body.
func (a *Azure) ProcessSubscription(b []byte) (json.RawMessage, error) {
	events, err := parseAzureEvents(b)
	if err != nil {
		return nil, err
	}
	if len(events) == 0 {
		return nil, errors.New("empty event payload")
	}

	// Validation code arrives in the first event for subscription validation flow.
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(events[0].RawData, &payload); err != nil {
		return nil, fmt.Errorf("error reading validation payload: %v", err)
	}

	rawData, ok := payload["data"]
	if !ok {
		return nil, errors.New("missing event data")
	}

	var data map[string]string
	if err := json.Unmarshal(rawData, &data); err != nil {
		return nil, fmt.Errorf("error reading validation data: %v", err)
	}

	code := strings.TrimSpace(data["validationCode"])
	if code == "" {
		return nil, errors.New("missing validationCode in subscription payload")
	}

	res, _ := json.Marshal(map[string]string{
		"validationResponse": code,
	})
	return json.RawMessage(res), nil
}

// ProcessBounce parses Azure Event Grid email delivery events and returns bounce entries.
func (a *Azure) ProcessBounce(req *http.Request, b []byte) ([]models.Bounce, error) {
	if err := a.verifyAuth(req); err != nil {
		return nil, err
	}

	events, err := parseAzureEvents(b)
	if err != nil {
		return nil, err
	}

	out := make([]models.Bounce, 0, len(events))
	for _, ev := range events {
		// Ignore non-delivery report events.
		if ev.EventType != "Microsoft.Communication.EmailDeliveryReportReceived" {
			continue
		}

		typ, ok := mapAzureStatus(ev.Data.Status, ev.Data.DeliveryStatusDetails.StatusMessage)
		if !ok {
			continue
		}

		email := strings.ToLower(strings.TrimSpace(ev.Data.Recipient))
		if email == "" {
			continue
		}

		campUUID, subUUID, _ := parseAzureMessageContext(ev.Data)

		tstamp := ev.Data.DeliveryAttemptTimestamp
		if tstamp == nil {
			tstamp = ev.Data.DeliveryAttemptTimeStamp
		}

		createdAt := time.Now()
		if tstamp != nil && !tstamp.IsZero() {
			createdAt = *tstamp
		}

		out = append(out, models.Bounce{
			CampaignUUID:   campUUID,
			SubscriberUUID: subUUID,
			Email:          email,
			Type:           typ,
			Meta:           json.RawMessage(ev.RawData),
			Source:         "azure",
			CreatedAt:      createdAt,
		})
	}

	return out, nil
}

func (a *Azure) verifyAuth(req *http.Request) error {
	const (
		defaultSecretHeader = "X-Listmonk-Webhook-Secret"
		querySecretParam    = "code"
	)

	// If no local credential is configured, allow webhook payloads.
	if a.sharedSecret == "" {
		return nil
	}

	if req == nil {
		return errors.New("missing azure event grid request context")
	}

	querySecret := strings.TrimSpace(req.URL.Query().Get(querySecretParam))
	if secretsEqual(a.sharedSecret, querySecret) {
		return nil
	}

	headerName := a.sharedSecretHeader
	if headerName == "" {
		headerName = defaultSecretHeader
	}
	headerSecret := strings.TrimSpace(req.Header.Get(headerName))
	if secretsEqual(a.sharedSecret, headerSecret) {
		return nil
	}

	return errors.New("invalid azure event grid shared secret")
}

func secretsEqual(expected, given string) bool {
	if expected == "" || given == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(given)) == 1
}

func parseAzureEvents(b []byte) ([]azureEvent, error) {
	var raws []json.RawMessage
	if err := json.Unmarshal(b, &raws); err != nil {
		return nil, fmt.Errorf("error unmarshalling azure notification array: %v", err)
	}

	events := make([]azureEvent, 0, len(raws))
	for _, raw := range raws {
		ev := azureEvent{RawData: raw}
		if err := json.Unmarshal(raw, &ev); err != nil {
			return nil, fmt.Errorf("error unmarshalling azure event: %v", err)
		}
		events = append(events, ev)
	}

	return events, nil
}

func parseAzureMessageContext(data azureEventData) (string, string, bool) {
	// In ACS delivery reports, listmonk's SMTP Message-ID is typically exposed
	// as internetMessageId. Keep messageId as a compatibility fallback.
	for _, rawMsgID := range []string{data.InternetMessageID, data.MessageID} {
		if campUUID, subUUID, ok := models.ParseListmonkMessageID(rawMsgID); ok {
			return campUUID, subUUID, true
		}
	}
	return "", "", false
}

func mapAzureStatus(status, details string) (string, bool) {
	s := strings.ToLower(strings.TrimSpace(status))
	d := strings.ToLower(strings.TrimSpace(details))

	switch s {
	case "bounced", "suppressed":
		return models.BounceTypeHard, true
	case "failed":
		// Infer severity from SMTP-enhanced status if available.
		if m := reSMTPStatus.FindStringSubmatch(d); len(m) > 1 {
			if strings.HasPrefix(m[1], "5.") {
				return models.BounceTypeHard, true
			}
			if strings.HasPrefix(m[1], "4.") {
				return models.BounceTypeSoft, true
			}
		}

		if strings.Contains(d, "mailbox full") ||
			strings.Contains(d, "temporar") ||
			strings.Contains(d, "timeout") ||
			strings.Contains(d, "rate limit") ||
			strings.Contains(d, "throttle") {
			return models.BounceTypeSoft, true
		}
		if strings.Contains(d, "invalid") ||
			strings.Contains(d, "does not exist") ||
			strings.Contains(d, "unknown user") ||
			strings.Contains(d, "domain") {
			return models.BounceTypeHard, true
		}

		return models.BounceTypeSoft, true
	case "filteredspam", "quarantined":
		if strings.Contains(d, "spam complaint") ||
			strings.Contains(d, "complaint") ||
			strings.Contains(d, "abuse") {
			return models.BounceTypeComplaint, true
		}
		return models.BounceTypeSoft, true
	default:
		return "", false
	}
}

