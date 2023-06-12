package webhooks

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/knadh/listmonk/internal/core"
	"github.com/knadh/listmonk/models"
)

type msg91Notif struct {
	Data data `json:"data"`
}
type data struct {
	ID              int           `json:"id"`
	UserID          int           `json:"user_id"`
	DomainID        int           `json:"domain_id"`
	RecipientID     int           `json:"recipient_id"`
	OutboundEmailID int           `json:"outbound_email_id"`
	EventID         int           `json:"event_id"`
	OutBoundEmail   outboundEmail `json:"outbound_email"`
	Event           event         `json:"event"`
}
type event struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}
type outboundEmail struct {
	ID        int        `json:"id"`
	UserID    int        `json:"user_id"`
	DomainID  int        `json:"domain_id"`
	ThreadID  int        `json:"thread_id"`
	To        []reciever `json:"to"`
	MessageID string     `json:"message_id"`
	CreatedAt time.Time  `json:"created_at"`
}
type reciever struct {
	Name  string `json:"name"`
	EMail string `json:"email"`
}

// Msg91 handles msg91/SNS webhook notifications including bounces
// requests and bounce notifications.
type Msg91 struct {
	core *core.Core
	log  *log.Logger
}

// NewMSG91 returns a new Msg91 instance.
func NewMSG91(c *core.Core, lo *log.Logger) *Msg91 {

	return &Msg91{
		core: c,
		log:  lo,
	}
}

// ProcessBounce processes msg91 bounce notifications and returns one or more Bounce objects.
func (s *Msg91) ProcessBounce(b []byte) ([]models.Bounce, error) {

	var notifs msg91Notif
	if err := json.Unmarshal(b, &notifs); err != nil {
		return nil, fmt.Errorf("error unmarshalling msg91 notification: %v", err)
	}

	out := make([]models.Bounce, 0, len(notifs.Data.OutBoundEmail.To))

	for _, e := range notifs.Data.OutBoundEmail.To {
		if strings.ToLower(notifs.Data.Event.Title) != "bounced" {
			s.log.Printf("level:INFO, msg: non bounce type delivery skipped , type:%v", notifs.Data.Event.Title)
			continue
		}
		sub, err := s.core.GetSubscriber(0, "", e.EMail)
		if err != nil {
			s.log.Printf("level:ERROR, msg: failed to get subscriber , email:%v", e.EMail)
			return nil, err
		}
		bn := models.Bounce{
			SubscriberUUID: sub.UUID,
			Email:          strings.ToLower(e.EMail),
			Type:           "hard",
			Meta:           json.RawMessage(b),
			Source:         "msg91",
			CreatedAt:      notifs.Data.Event.CreatedAt,
		}
		out = append(out, bn)
	}

	return out, nil
}
