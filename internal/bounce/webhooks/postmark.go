package webhooks

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type postmarkNotif struct {
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
	authHandler echo.HandlerFunc
}

func NewPostmark(username, password string) *Postmark {
	return &Postmark{
		authHandler: middleware.BasicAuth(makePostmarkAuthHandler(username, password))(func(c echo.Context) error {
			return nil
		}),
	}
}

// ProcessBounce processes Postmark bounce notifications and returns one object.
func (p *Postmark) ProcessBounce(b []byte, c echo.Context) ([]models.Bounce, error) {
	// Do basicauth.
	if err := p.authHandler(c); err != nil {
		return nil, err
	}

	var n postmarkNotif
	if err := json.Unmarshal(b, &n); err != nil {
		return nil, fmt.Errorf("error unmarshalling postmark notification: %v", err)
	}

	// Ignore non-bounce messages.
	if n.RecordType != "Bounce" {
		return nil, nil
	}

	supportedBounceType := true
	typ := models.BounceTypeHard
	switch n.Type {
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
		return nil, fmt.Errorf("unsupported bounce type: %v", n.Type)
	}

	// Look for the campaign ID in headers.
	campUUID := ""
	if v, ok := n.Metadata["X-Listmonk-Campaign"]; ok {
		campUUID = v
	}

	return []models.Bounce{{
		Email:        strings.ToLower(n.Email),
		CampaignUUID: campUUID,
		Type:         typ,
		Source:       "postmark",
		Meta:         json.RawMessage(b),
		CreatedAt:    n.BouncedAt,
	}}, nil
}

func makePostmarkAuthHandler(cfgUser, cfgPassword string) func(username, password string, c echo.Context) (bool, error) {
	var (
		u = []byte(cfgUser)
		p = []byte(cfgPassword)
	)

	return func(username, password string, c echo.Context) (bool, error) {
		if len(u) == 0 || len(p) == 0 {
			return true, nil
		}

		if subtle.ConstantTimeCompare([]byte(username), u) == 1 && subtle.ConstantTimeCompare([]byte(password), p) == 1 {
			return true, nil
		}

		return false, nil
	}
}
