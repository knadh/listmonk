package mailbox

import (
	"encoding/json"
	"regexp"
	"time"

	"github.com/emersion/go-message"
	_ "github.com/emersion/go-message/charset"
	"github.com/knadh/go-pop3"
	"github.com/knadh/listmonk/models"
)

// POP represents a POP mailbox.
type POP struct {
	opt    Opt
	client *pop3.Client
}

var (
	reCampUUID = regexp.MustCompile(`(?m)(?m:^` + models.EmailHeaderCampaignUUID + `:\s+?)([a-z0-9\-]{36})`)
	reSubUUID  = regexp.MustCompile(`(?m)(?m:^` + models.EmailHeaderSubscriberUUID + `:\s+?)([a-z0-9\-]{36})`)
)

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
		// Retrieve the raw bytes of the message.
		b, err := c.RetrRaw(id)
		if err != nil {
			return err
		}

		// Parse the message.
		m, err := message.Read(b)
		if err != nil {
			return err
		}

		// Check if the identifiers are available in the parsed message.
		var (
			campUUID = m.Header.Get(models.EmailHeaderCampaignUUID)
			subUUID  = m.Header.Get(models.EmailHeaderSubscriberUUID)
		)

		// If they are not, try to extract them from the message body.
		if campUUID == "" {
			if u := reCampUUID.FindSubmatch(b.Bytes()); len(u) == 2 {
				campUUID = string(u[1])
			}
		}
		if subUUID == "" {
			if u := reSubUUID.FindSubmatch(b.Bytes()); len(u) == 2 {
				subUUID = string(u[1])
			}
		}

		if campUUID == "" || subUUID == "" {
			continue
		}

		date, _ := time.Parse("Mon, 02 Jan 2006 15:04:05 -0700", m.Header.Get("Date"))
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
			CampaignUUID:   campUUID,
			SubscriberUUID: subUUID,
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
