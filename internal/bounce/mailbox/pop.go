package mailbox

import (
	"encoding/json"
	"io"
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
	reCampUUID           = regexp.MustCompile(`(?m)(?:^` + models.EmailHeaderCampaignUUID + `:\s+?)([a-z0-9\-]{36})`)
	reSubUUID            = regexp.MustCompile(`(?m)(?:^` + models.EmailHeaderSubscriberUUID + `:\s+?)([a-z0-9\-]{36})`)
	reMessageDate        = regexp.MustCompile(`(?m)(?:^` + models.EmailHeaderDate + `:\s+?)([\w,\,\ ,:,+,-]*(?:\(?:\w*\))?)`)
	reMessageFrom        = regexp.MustCompile(`(?m)(?:^` + models.EmailHeaderFrom + `:\s+?)(.*)`)
	reMessageSubject     = regexp.MustCompile(`(?m)(?:^` + models.EmailHeaderSubject + `:\s+?)(.*)`)
	reMessageID          = regexp.MustCompile(`(?m)(?:^` + models.EmailHeaderMessageId + `:\s+?)(.*)`)
	reMessageDeliveredTo = regexp.MustCompile(`(?m)(?:^` + models.EmailHeaderDeliveredTo + `:\s+?)(.*)`)
	reMessageReceived    = regexp.MustCompile(`(?m)(?:^` + models.EmailHeaderReceived + `:\s+?)(.*)`)
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

		h := m

		// If this is a multipart message, find the last part.
		if mr := m.MultipartReader(); mr != nil {
			for {
				part, err := mr.NextPart()
				if err == io.EOF {
					break
				} else if err != nil {
					return err
				}
				h = part
			}
		}

		// Check if the identifiers are available in the parsed header.
		campUUID := h.Header.Get(models.EmailHeaderCampaignUUID)
		subUUID := h.Header.Get(models.EmailHeaderSubscriberUUID)
		messageDate := h.Header.Get(models.EmailHeaderDate)
		messageFrom := h.Header.Get(models.EmailHeaderFrom)
		messageSubject := h.Header.Get(models.EmailHeaderSubject)
		messageID := h.Header.Get(models.EmailHeaderMessageId)
		messageDeliveredTo := h.Header.Get(models.EmailHeaderDeliveredTo)
		messageReceived := h.Header.Map()[models.EmailHeaderReceived]

		// Reset the "unread portion" pointer of the message buffer.
		// If you don't do this, you can't read the entire body because the pointer will not point to the beginning.
		b, _ = c.RetrRaw(id)

		// If they are not, try to extract them from the message body.
		if campUUID == "" {
			if u := reCampUUID.FindAllSubmatch(b.Bytes(), -1); u != nil {
				campUUID = string(u[len(u)-1][1])
			} else {
				continue
			}
		}
		if subUUID == "" {
			if u := reSubUUID.FindAllSubmatch(b.Bytes(), -1); u != nil {
				subUUID = string(u[len(u)-1][1])
			} else {
				continue
			}
		}
		if messageDate == "" {
			if u := reMessageDate.FindAllSubmatch(b.Bytes(), -1); u != nil {
				messageDate = string(u[len(u)-1][1])
			}
		}
		if messageFrom == "" {
			if u := reMessageFrom.FindAllSubmatch(b.Bytes(), -1); u != nil {
				messageFrom = string(u[len(u)-1][1])
			}
		}
		if messageSubject == "" {
			if u := reMessageSubject.FindAllSubmatch(b.Bytes(), -1); u != nil {
				messageSubject = string(u[len(u)-1][1])
			}
		}
		if messageID == "" {
			if u := reMessageID.FindAllSubmatch(b.Bytes(), -1); u != nil {
				messageID = string(u[len(u)-1][1])
			}
		}
		if messageDeliveredTo == "" {
			if u := reMessageDeliveredTo.FindAllSubmatch(b.Bytes(), -1); u != nil {
				messageDeliveredTo = string(u[len(u)-1][1])
			}
		}
		if len(messageReceived) == 0 {
			if u := reMessageReceived.FindAllSubmatch(b.Bytes(), -1); u != nil {
				for i := 0; i < len(u); i++ {
					messageReceived = append(messageReceived, string(u[i][1]))
				}
			}
		}

		date, _ := time.Parse("Mon, 02 Jan 2006 15:04:05 -0700", messageDate)
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
			From:        messageFrom,
			Subject:     messageSubject,
			MessageID:   messageID,
			DeliveredTo: messageDeliveredTo,
			Received:    messageReceived,
		})

		select {
		case ch <- models.Bounce{
			Type:           "hard",
			CampaignUUID:   campUUID,
			SubscriberUUID: subUUID,
			Source:         p.opt.Host,
			CreatedAt:      date,
			Meta:           meta,
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
