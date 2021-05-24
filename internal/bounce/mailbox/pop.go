package mailbox

import (
	"encoding/json"
	"time"

	"github.com/knadh/go-pop3"
	"github.com/knadh/listmonk/models"
)

// POP represents a POP mailbox.
type POP struct {
	opt    Opt
	client *pop3.Client
}

// NewPOP returns a new instance of the POP mailbox client.
func NewPOP(opt Opt) *POP {
	return &POP{
		opt: opt,
		client: pop3.New(pop3.Opt{
			Host:          opt.Host,
			Port:          opt.Port,
			TLSEnabled:    opt.TLSEnabled,
			TLSSkipVerify: opt.TLSSkipVerify,
		}),
	}
}

// Scan scans the mailbox and pushes the downloaded messages into the given channel.
// The messages that are downloaded are deleted from the server. If limit > 0,
// all messages on the server are downloaded and deleted.
func (p *POP) Scan(limit int, ch chan models.Bounce) error {
	c, err := p.client.NewConn()
	if err != nil {
		return err
	}
	defer c.Quit()

	// Authenticate.
	if p.opt.AuthProtocol != "none" {
		if err := c.Auth(p.opt.Username, p.opt.Password); err != nil {
			return err
		}
	}

	// Get the total number of messages on the server.
	count, _, err := c.Stat()
	if err != nil {
		return err
	}

	// No messages.
	if count == 0 {
		return nil
	}

	if limit > 0 && count > limit {
		count = limit
	}

	// Download messages.
	for id := 1; id <= count; id++ {
		// Download just one line of the body as the body is not required at all.
		m, err := c.Top(id, 1)
		if err != nil {
			return err
		}

		var (
			campUUID = m.Header.Get(models.EmailHeaderCampaignUUID)
			subUUID  = m.Header.Get(models.EmailHeaderSubscriberUUID)
			date, _  = time.Parse("Mon, 02 Jan 2006 15:04:05 -0700", m.Header.Get("Date"))
		)

		if campUUID == "" || subUUID == "" {
			continue
		}
		if date.IsZero() {
			date = time.Now()
		}

		// Additional bounce e-mail metadata.
		meta, _ := json.Marshal(struct {
			From        string   `json:"from"`
			Subject     string   `json:"subject"`
			MessageID   string   `json:"message_id"`
			DeliveredTo string   `json:"delivered_to"`
			Received    []string `json:"received"`
		}{
			From:        m.Header.Get("From"),
			Subject:     m.Header.Get("Subject"),
			MessageID:   m.Header.Get("Message-Id"),
			DeliveredTo: m.Header.Get("Delivered-To"),
			Received:    m.Header.Map()["Received"],
		})

		select {
		case ch <- models.Bounce{
			Type:           "hard",
			CampaignUUID:   m.Header.Get(models.EmailHeaderCampaignUUID),
			SubscriberUUID: m.Header.Get(models.EmailHeaderSubscriberUUID),
			Source:         p.opt.Host,
			CreatedAt:      date,
			Meta:           json.RawMessage(meta),
		}:
		default:
		}
	}

	// Delete the downloaded messages.
	for id := 1; id <= count; id++ {
		if err := c.Dele(id); err != nil {
			return err
		}
	}

	return nil
}
