package webhooks

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/knadh/listmonk/models"
)

type postmarkBounceNotif struct {
	RecordType    string            `json:"RecordType"`
	MessageStream string            `json:"MessageStream"`
	ID            int               `json:"ID"`
	Type          string            `json:"Type"`
	TypeCode      int               `json:"TypeCode"`
	Name          string            `json:"Name"`
	Tag           string            `json:"Tag"`
	MessageID     string            `json:"MessageID"`
	Metadata      map[string]string `json:"Metadata"`
	ServerID      int               `json:"ServerID"`
	Description   string            `json:"Description"`
	Details       string            `json:"Details"`
	Email         string            `json:"Email"`
	From          string            `json:"From"`
	BouncedAt     time.Time         `json:"BouncedAt"` // "2019-11-05T16:33:54.9070259Z"
	DumpAvailable bool              `json:"DumpAvailable"`
	Inactive      bool              `json:"Inactive"`
	CanActivate   bool              `json:"CanActivate"`
	Subject       string            `json:"Subject"`
	Content       string            `json:"Content"`
}

// Postmark handles webhook notifications (mainly bounce notifications).
type Postmark struct {
}

// NewPostmark returns a new Postmark instance.
func NewPostmark() *Postmark {
	return &Postmark{}
}

// ProcessBounce processes Postmark bounce notifications and returns one object.
func (s *Postmark) ProcessBounce(b []byte) (models.Bounce, error) {
	var (
		bounce models.Bounce
		m      postmarkBounceNotif
	)
	if err := json.Unmarshal(b, &m); err != nil {
		return bounce, fmt.Errorf("error unmarshalling postmark notification: %v", err)
	}

	if m.RecordType != "Bounce" {
		return bounce, errors.New("notification type is not bounce")
	}

	supportedBounceType := true
	typ := models.BounceTypeHard
	switch m.Type {
	case "HardBounce", "BadEmailAddress", "ManuallyDeactivated":
		typ = models.BounceTypeHard
	case "SoftBounce", "Transient", "DnsError", "SpamNotification", "VirusNotification", "DMARCPolicy":
		typ = models.BounceTypeSoft
	case "SpamComplaint":
		typ = models.BounceTypeComplaint
	default:
		supportedBounceType = false
	}

	if !supportedBounceType {
		return bounce, fmt.Errorf("unsupported bounce type: %v", m.Type)
	}

	// Look for the campaign ID in headers.
	campUUID := ""
	for k, v := range m.Metadata {
		if k == "X-Listmonk-Campaign" {
			campUUID = v
			break
		}
	}
	return models.Bounce{
		Email:        strings.ToLower(m.Email),
		CampaignUUID: campUUID,
		Type:         typ,
		Source:       "postmark",
		Meta:         json.RawMessage(b),
		CreatedAt:    m.BouncedAt,
	}, nil
}
