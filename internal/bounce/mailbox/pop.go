package mailbox

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/emersion/go-message"
	_ "github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/textproto"
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
	DiagnosticCode string   `json:"diagnostic_code,omitempty"`
}

var (
	// List of header to look for in the e-mail body, regexp to fall back to if the header is empty.
	headerLookups = []bounceHeaders{
		{models.EmailHeaderCampaignUUID, regexp.MustCompile(`(?mi)(?:^` + models.EmailHeaderCampaignUUID + `:\s+?)([a-z0-9\-]{36})`)},
		{models.EmailHeaderSubscriberUUID, regexp.MustCompile(`(?mi)(?:^` + models.EmailHeaderSubscriberUUID + `:\s+?)([a-z0-9\-]{36})`)},
		{models.EmailHeaderDate, regexp.MustCompile(`(?mi)(?:^` + models.EmailHeaderDate + `:\s+?)([\w,\,\ ,:,+,-]*(?:\(?:\w*\))?)`)},
		{models.EmailHeaderFrom, regexp.MustCompile(`(?mi)(?:^` + models.EmailHeaderFrom + `:\s+?)(.*)`)},
		{models.EmailHeaderSubject, regexp.MustCompile(`(?mi)(?:^` + models.EmailHeaderSubject + `:\s+?)(.*)`)},
		{models.EmailHeaderMessageId, regexp.MustCompile(`(?mi)(?:^` + models.EmailHeaderMessageId + `:\s+?)(.*)`)},
		{models.EmailHeaderDeliveredTo, regexp.MustCompile(`(?mi)(?:^` + models.EmailHeaderDeliveredTo + `:\s+?)(.*)`)},
	}

	reHdrReceived = regexp.MustCompile(`(?mi)(?:^` + models.EmailHeaderReceived + `:[ \t]*)([^\r\n]*(?:\r?\n[ \t]+[^\r\n]*)*)`)

	// SMTP status code (5.x.x or 4.x.x) to classify hard/soft bounces.
	reSMTPStatus = regexp.MustCompile(`(?m)(?i)^(?:Status(?:\s+code)?:\s*)?(?:\d{3}\s+)?([45]\.\d+\.\d+)`)

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
// The scanned messages are deleted from the server. A positive limit caps the
// number of messages scanned.
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

		bounces, err := parseBounce(b.Bytes())
		if err != nil {
			p.lo.Printf("error parsing bounce message %d: %v", id, err)
			continue
		}
		for _, bounce := range bounces {
			bounce.Source = p.opt.Host
			ch <- bounce
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

// parseBounce uses MIME headers and delivery status fields to extract bounce messages.
// When standard headers aren't available, regex heuristics are used to try and extract info.
func parseBounce(raw []byte) ([]models.Bounce, error) {
	m, err := message.Read(bytes.NewReader(bytes.TrimPrefix(raw, []byte{0xef, 0xbb, 0xbf})))
	if err != nil && !message.IsUnknownCharset(err) {
		// Even if the message has an unknown charset, headers can still be parsed.
		return nil, err
	}

	h := m.Header
	var (
		original   bool
		dsn        bool
		feedback   *textproto.Header
		recipients []textproto.Header
	)

	// Walk through nested multipart messages.
	if err := m.Walk(func(_ []int, part *message.Entity, err error) error {
		if err != nil && !message.IsUnknownCharset(err) {
			return err
		}

		if !original {
			h = part.Header
		}

		ct, _, _ := part.Header.ContentType()
		switch ct {
		case "message/feedback-report":
			hdr, err := textproto.ReadHeader(bufio.NewReader(part.Body))
			if err != nil && err != io.EOF {
				return err
			}

			feedback = &hdr
		case "message/rfc822", "message/global", "text/rfc822-headers":
			hdr, err := textproto.ReadHeader(bufio.NewReader(part.Body))
			if err != nil && err != io.EOF {
				return err
			}

			h = message.Header{Header: hdr}
			original = true
		case "message/delivery-status", "message/global-delivery-status":
			dsn = true
			r := bufio.NewReader(part.Body)

			// Skip the first block that describes the reporting server and not the subscriber / email recipient.
			if _, err := textproto.ReadHeader(r); err != nil {
				return err
			}

			for {
				hdr, err := textproto.ReadHeader(r)
				if err != nil && err != io.EOF {
					return err
				}

				if hdr.Len() == 0 {
					break
				}

				// Found recipients.
				recipients = append(recipients, hdr)
				if err == io.EOF {
					break
				}
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	// Try regex heuristics.
	hdr := make(map[string]string, len(headerLookups))
	for _, l := range headerLookups {
		v := h.Get(l.Header)
		if v == "" {
			if matches := l.Regexp.FindAllSubmatch(raw, -1); len(matches) > 0 {
				v = string(matches[len(matches)-1][1])
			}
		}

		hdr[l.Header] = strings.TrimSpace(v)
	}

	var received []string
	for fields := h.FieldsByKey(models.EmailHeaderReceived); fields.Next(); {
		received = append(received, fields.Value())
	}

	if len(received) == 0 {
		for _, match := range reHdrReceived.FindAllSubmatch(raw, -1) {
			received = append(received, strings.Join(strings.Fields(string(match[1])), " "))
		}
	}

	date, _ := mail.ParseDate(hdr[models.EmailHeaderDate])
	if date.IsZero() {
		date = time.Now()
	}

	bounceType, reason := classifyBounce(raw)
	base := models.Bounce{
		Type:           bounceType,
		CampaignUUID:   hdr[models.EmailHeaderCampaignUUID],
		SubscriberUUID: hdr[models.EmailHeaderSubscriberUUID],
		CreatedAt:      date,
	}

	meta := bounceMeta{
		From:           hdr[models.EmailHeaderFrom],
		Subject:        hdr[models.EmailHeaderSubject],
		MessageID:      hdr[models.EmailHeaderMessageId],
		DeliveredTo:    hdr[models.EmailHeaderDeliveredTo],
		Received:       received,
		ClassifyReason: reason,
	}

	// There are ARF/feedback headers.
	if feedback != nil {
		if !strings.EqualFold(strings.TrimSpace(feedback.Get("Feedback-Type")), "abuse") {
			return nil, nil
		}

		base.Type = models.BounceTypeComplaint
		meta.ClassifyReason = "feedback_type=abuse"
		base.Meta, _ = json.Marshal(meta)
		if base.SubscriberUUID != "" {
			return []models.Bounce{base}, nil
		}

		var out []models.Bounce
		for fields := feedback.FieldsByKey("Original-Rcpt-To"); fields.Next(); {
			a, err := mail.ParseAddress(strings.TrimSpace(fields.Value()))
			if err != nil {
				return nil, fmt.Errorf("invalid original recipient: %w", err)
			}

			b := base
			b.Email = strings.ToLower(a.Address)
			out = append(out, b)
		}
		if len(out) == 0 {
			return nil, fmt.Errorf("missing subscriber UUID or original recipient")
		}

		return out, nil
	}

	if !dsn {
		base.Meta, _ = json.Marshal(meta)
		if base.SubscriberUUID == "" {
			return nil, fmt.Errorf("missing subscriber UUID")
		}

		return []models.Bounce{base}, nil
	}

	var out []models.Bounce
	for _, recipient := range recipients {
		b := base
		status := strings.TrimSpace(recipient.Get("Status"))
		// Some servers append an explanatory comment after the status code.
		if fields := strings.Fields(status); len(fields) > 0 {
			status = fields[0]
		}
		action := strings.ToLower(strings.TrimSpace(recipient.Get("Action")))

		switch {
		case strings.HasPrefix(status, "2."), action == "delivered", action == "relayed", action == "expanded":
			continue
		case strings.HasPrefix(status, "5."):
			b.Type = models.BounceTypeHard
		case strings.HasPrefix(status, "4."):
			b.Type = models.BounceTypeSoft
		default:
			return nil, fmt.Errorf("invalid delivery status %q", status)
		}

		if len(recipients) > 1 {
			b.SubscriberUUID = ""
		}

		// No sub UUID, try to extract the email address from the final recipient header.
		if b.SubscriberUUID == "" {
			kind, address, _ := strings.Cut(recipient.Get("Final-Recipient"), ";")
			if !strings.EqualFold(strings.TrimSpace(kind), "rfc822") && !strings.EqualFold(strings.TrimSpace(kind), "utf-8") {
				return nil, fmt.Errorf("invalid final recipient")
			}

			a, err := mail.ParseAddress(strings.TrimSpace(address))
			if err != nil {
				return nil, fmt.Errorf("invalid final recipient: %w", err)
			}

			b.Email = strings.ToLower(a.Address)
		}

		meta.ClassifyReason = "smtp_status=" + status
		meta.DiagnosticCode = recipient.Get("Diagnostic-Code")
		b.Meta, _ = json.Marshal(meta)

		out = append(out, b)
	}

	return out, nil
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
