package mailbox

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"regexp"
	"strings"
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
	lo     *log.Logger
}

type bounceHeaders struct {
	Header string
	Regexp *regexp.Regexp
}

type bounceMeta struct {
	From           string   `json:"from"`
	Subject        string   `json:"subject"`
	MessageID      string   `json:"message_id"`
	DeliveredTo    string   `json:"delivered_to"`
	Received       []string `json:"received"`
	ClassifyReason string   `json:"classify_reason"`
}

var (
	// List of header to look for in the e-mail body, regexp to fall back to if the header is empty.
	headerLookups = []bounceHeaders{
		{models.EmailHeaderCampaignUUID, regexp.MustCompile(`(?m)(?:^` + models.EmailHeaderCampaignUUID + `:\s+?)([a-z0-9\-]{36})`)},
		{models.EmailHeaderSubscriberUUID, regexp.MustCompile(`(?m)(?:^` + models.EmailHeaderSubscriberUUID + `:\s+?)([a-z0-9\-]{36})`)},
		{models.EmailHeaderDate, regexp.MustCompile(`(?m)(?:^` + models.EmailHeaderDate + `:\s+?)([\w,\,\ ,:,+,-]*(?:\(?:\w*\))?)`)},
		{models.EmailHeaderFrom, regexp.MustCompile(`(?m)(?:^` + models.EmailHeaderFrom + `:\s+?)(.*)`)},
		{models.EmailHeaderSubject, regexp.MustCompile(`(?m)(?:^` + models.EmailHeaderSubject + `:\s+?)(.*)`)},
		{models.EmailHeaderMessageId, regexp.MustCompile(`(?m)(?:^` + models.EmailHeaderMessageId + `:\s+?)(.*)`)},
		{models.EmailHeaderDeliveredTo, regexp.MustCompile(`(?m)(?:^` + models.EmailHeaderDeliveredTo + `:\s+?)(.*)`)},
	}

	reHdrReceived = regexp.MustCompile(`(?m)(?:^` + models.EmailHeaderReceived + `:\s+?)(.*)`)

	// SMTP status code (5.x.x or 4.x.x) to classify hard/soft bounces.
	reSMTPStatus = regexp.MustCompile(`(?m)(?i)^(?:Status:\s*)?(?:\d{3}\s+)?([45]\.\d+\.\d+)`)

	// List of (conventional) strings to guess hard bounces.
	reHardBounce = regexp.MustCompile(`(?i)(NXDOMAIN|user unknown|address not found|mailbox not found|address.*reject|does not exist|` +
		`invalid recipient|no such user|recipient.*invalid|undeliverable|permanent.*failure|permanent.*error|` +
		`bad.*address|unknown.*user|account.*disabled|address.*disabled)`)
)

// NewPOP returns a new instance of the POP mailbox client.
func NewPOP(opt Opt, lo *log.Logger) *POP {
	return &POP{
		opt: opt,
		client: pop3.New(pop3.Opt{
			Host:          opt.Host,
			Port:          opt.Port,
			TLSEnabled:    opt.TLSEnabled,
			TLSSkipVerify: opt.TLSSkipVerify,
		}),
		lo: lo,
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
			p.lo.Printf("error retrieving bounce message %d: %v", id, err)
			continue
		}

		// Parse the message.
		m, err := message.Read(b)
		if err != nil {
			p.lo.Printf("error parsing bounce message %d: %v", id, err)
			continue
		}

		h := m

		// If this is a multipart message, find the last part.
		if mr := m.MultipartReader(); mr != nil {
			for {
				part, err := mr.NextPart()
				if err == io.EOF {
					break
				} else if err != nil {
					p.lo.Printf("error reading multipart bounce message %d: %v", id, err)
					continue
				}
				h = part
			}
		}

		// Reset the "unread portion" pointer of the message buffer.
		// If you don't do this, you can't read the entire body because the pointer will not point to the beginning.
		b, _ = c.RetrRaw(id)

		// Lookup headers in the e-mail. If a header isn't found, fall back to regexp lookups.
		hdr := make(map[string]string, 7)
		for _, l := range headerLookups {
			v := h.Header.Get(l.Header)

			// Not in the header. Try regexp.
			if v == "" {
				if m := l.Regexp.FindAllSubmatch(b.Bytes(), -1); m != nil {
					v = string(m[len(m)-1][1])
				}
			}

			hdr[l.Header] = strings.TrimSpace(v)
		}

		// Received is a []string header.
		msgReceived := h.Header.Map()[models.EmailHeaderReceived]
		if len(msgReceived) == 0 {
			if u := reHdrReceived.FindAllSubmatch(b.Bytes(), -1); u != nil {
				for i := range u {
					msgReceived = append(msgReceived, string(u[i][1]))
				}
			}
		}

		date, _ := time.Parse("Mon, 02 Jan 2006 15:04:05 -0700", hdr[models.EmailHeaderDate])
		if date.IsZero() {
			date = time.Now()
		}

		// Classify the bounce type based on message content.
		bounceType, bounceReason := classifyBounce(b.Bytes())

		// Additional bounce e-mail metadata.
		meta, _ := json.Marshal(bounceMeta{
			From:           hdr[models.EmailHeaderFrom],
			Subject:        hdr[models.EmailHeaderSubject],
			MessageID:      hdr[models.EmailHeaderMessageId],
			DeliveredTo:    hdr[models.EmailHeaderDeliveredTo],
			Received:       msgReceived,
			ClassifyReason: bounceReason,
		})

		select {
		case ch <- models.Bounce{
			Type:           bounceType,
			CampaignUUID:   hdr[models.EmailHeaderCampaignUUID],
			SubscriberUUID: hdr[models.EmailHeaderSubscriberUUID],
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

// classifyBounce analyzes the bounce message content and determines if it's a hard or soft bounce.
// It checks SMTP status codes, diagnostic headers, and bounce keywords (using string heuristics).
// soft is the default preference.
// Returns the bounce type and a classification reason containing context about what matched.
func classifyBounce(b []byte) (string, string) {
	if matches := reSMTPStatus.FindAllSubmatch(b, -1); matches != nil {
		for _, m := range matches {
			if len(m) >= 2 && len(m[0]) > 1 {
				// Full status code (e.g., "5.1.1").
				status := m[1]

				// 5.x.x is hard bounce.
				if status[0] == '5' {
					return models.BounceTypeHard, fmt.Sprintf("smtp_status=%s", status)
				}

				// 4.x.x  is soft bounce.
				if status[0] == '4' {
					return models.BounceTypeSoft, fmt.Sprintf("smtp_status=%s", status)
				}
			}
		}
	}

	// Check for explicit hard bounce keywords.
	if match := reHardBounce.FindSubmatch(b); match != nil {
		return models.BounceTypeHard, fmt.Sprintf("body_match=%s", match[1])
	}

	return models.BounceTypeSoft, "default"
}
