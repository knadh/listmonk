package mailbox

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/emersion/go-message"
	"github.com/emersion/go-message/textproto"
	"github.com/knadh/listmonk/models"
)

var (
	// List of header to look for in the e-mail body, regexp to fall back to if the header is empty.
	headerLookups = []bounceHeaders{
		{models.EmailHeaderCampaignUUID, regexp.MustCompile(`(?m)(?i)(?:^` + models.EmailHeaderCampaignUUID + `:\s+?)([a-z0-9\-]{36})`)},
		{models.EmailHeaderSubscriberUUID, regexp.MustCompile(`(?m)(?i)(?:^` + models.EmailHeaderSubscriberUUID + `:\s+?)([a-z0-9\-]{36})`)},
		{models.EmailHeaderDate, regexp.MustCompile(`(?m)(?:^` + models.EmailHeaderDate + `:\s+?)([\w,\,\ ,:,+,-]*(?:\(?:\w*\))?)`)},
		{models.EmailHeaderFrom, regexp.MustCompile(`(?m)(?:^` + models.EmailHeaderFrom + `:\s+?)(.*)`)},
		{models.EmailHeaderSubject, regexp.MustCompile(`(?m)(?:^` + models.EmailHeaderSubject + `:\s+?)(.*)`)},
		{models.EmailHeaderMessageId, regexp.MustCompile(`(?m)(?i)(?:^` + models.EmailHeaderMessageId + `:\s+?)(.*)`)},
		{models.EmailHeaderDeliveredTo, regexp.MustCompile(`(?m)(?i)(?:^` + models.EmailHeaderDeliveredTo + `:\s+?)(.*)`)},
	}

	reHdrReceived = regexp.MustCompile(`(?m)(?:^` + models.EmailHeaderReceived + `:\s+?)((?:[^\r\n]|\r?\n[ \t])+)`)
	reUnfold      = regexp.MustCompile(`\r?\n[ \t]+`)

	// SMTP status code (5.x.x or 4.x.x) to classify hard/soft bounces.
	reSMTPStatus = regexp.MustCompile(`(?m)(?i)^(?:Status(?:\s*code)?:\s*)?(?:\d{3}\s+)?([45]\.\d+\.\d+)`)

	// List of (conventional) strings to guess hard bounces.
	reHardBounce = regexp.MustCompile(`(?i)(NXDOMAIN|user unknown|address not found|mailbox not found|address.*reject|does not exist|` +
		`invalid recipient|no such user|recipient.*invalid|undeliverable|permanent.*failure|permanent.*error|` +
		`bad.*address|unknown.*user|account.*disabled|address.*disabled)`)

	errorNotMultipartReport = fmt.Errorf("not a multipart/report message")
	utf8BOM                 = []byte{0xEF, 0xBB, 0xBF}
)

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
	BounceDetails  []string `json:"bounce_details,omitempty"`
}


// parseTime attempt to parse a timestamp in the RFC822 and RFC822Z format then
// tries other commonly found non-standard compliant formats used in email
// messages.
func parseTime(timeStr string) (time.Time, error) {
	layouts := []string{
		"02 Jan 2006 15:04:05 -0700",
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"02 Jan 2006 15:04:05 MST",
		"Mon, 2 Jan 2006 15:04:05 -0700 (MST)",
		"Mon, 2 Jan 2006 15:04:05 MST",
		"2 Jan 06 15:04 MST",
		"2 Jan 06 15:04 -0700",
		"Mon, 2 Jan 06 15:04 -0700",
		"Mon, 2 Jan 06 15:04 -0700 (MST)",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, timeStr); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse")
}

// parseDSN attempts to parse a message that follows the rfc3464 (Delivery
// Status Notifications) or rfc5965 (Abuse Reporting Format) into a
// `models.Bounce`. If the message doesn't follow the DSN format `regexParser`
// is called with the message and `errorNotMultipartReport` is emitted with the
// result.
func parseDSN(b io.Reader) (bounce models.Bounce, err error) {
	var meta bounceMeta

	// Remove utf8 BOM in present as causes errors parsing message
	bBuff := bufio.NewReader(b)
	bomCheck, err := bBuff.Peek(3)
	if err != nil {
		return bounce, fmt.Errorf("error checking for UTF8 BOM: %v", err)
	}
	if bytes.Equal(utf8BOM, bomCheck) {
		d, err := bBuff.Discard(3)
		if err != nil {
			return bounce, fmt.Errorf("error removing UTF8 BOM: %v", err)
		}
		if d != 3 {
			return bounce, fmt.Errorf("error removing UTF8 BOM: expected discard 3 bytes discarded %d instead", d)
		}
	}

	// Parse the message.
	m, err := message.Read(bBuff)
	if err != nil {
		return bounce, fmt.Errorf("error parsing: %v", err)
	}

	// Bounced messages should be a MIME message with a top-level content-type
	// of multipart/report as defined in rfc3464 paragraph 2 "Format of a
	// Delivery Status Notification".
	ct, _, err := m.Header.ContentType()
	if err != nil {
		return bounce, fmt.Errorf("reading content-type: %v", err)
	}
	if ct != "multipart/report" {
		// Not a DSN so very likely not a bounce message. Attempt to use
		// `regexParser` to identify subscriber and bounce classification.
		bounce, err = regexParser(m)
		if err == nil {
			return bounce, errorNotMultipartReport
		}
		return bounce, err
	}

	mr := m.MultipartReader()
	if mr == nil {
		// Mallformed DSN that isn't a MIME multipart message
		return bounce, fmt.Errorf("malformed Delivery Status Notification")
	}

	// Delivery Status Notifications will usually reference the original
	// Message-Id
	var originalMessageId string
	references := strings.Fields(m.Header.Get("References"))
	if len(references) > 0 {
		originalMessageId = references[0]
	}
	reportMsgDate, _ := parseTime(m.Header.Get(models.EmailHeaderDate))

	// Find the message/delivery-status part and the text/rfc822-headers or
	// message/rfc822 parts. Also message/global-delivery-status and
	// message/global because Postfix doesn't follow standards. Limited support
	// for rfc5965 abuse report message/feedback-report parts.
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			return bounce, fmt.Errorf("reading multipart: %v", err)
		}

		partCt, _, err := part.Header.ContentType()
		if err != nil {
			return bounce, fmt.Errorf("reading multipart content-type: %v", err)
		}
		switch partCt {
		case "message/delivery-status", "message/global-delivery-status":
			partBody := bufio.NewReader(part.Body)

			dsnMessageFields, err := textproto.ReadHeader(partBody)
			if err != nil {
				return bounce, fmt.Errorf("parsing message/delivery-status message fields: %v", err)
			}
			dsnFirstRecipientFields, err := textproto.ReadHeader(partBody)
			if err != nil {
				return bounce, fmt.Errorf("parsing message/delivery-status per-recipient fields: %v", err)
			}

			arrivalTime, _ := parseTime(dsnMessageFields.Get("Arrival-Date"))
			bounce.CreatedAt, _ = parseTime(dsnMessageFields.Get("Last-Attempt-Date"))
			if bounce.CreatedAt.IsZero() {
				bounce.CreatedAt = arrivalTime
			}

			finalRecipient := strings.Split(dsnFirstRecipientFields.Get("Final-Recipient"), ";")
			if len(finalRecipient) > 0 && strings.ToLower(finalRecipient[0]) == "rfc822" {
				bounce.Email = strings.TrimSpace(finalRecipient[1])
			}

			status := strings.Fields(dsnFirstRecipientFields.Get("Status"))[0]
			switch status[0] {
			case '5':
				bounce.Type = models.BounceTypeHard
				meta.ClassifyReason = fmt.Sprintf("smtp_status=%s", status)
			case '4':
				bounce.Type = models.BounceTypeHard
				meta.ClassifyReason = fmt.Sprintf("smtp_status=%s", status)
			}

			// Status is a mandatory field in message/delivery-status so we
			// shouldn't get here...
			if bounce.Type == "" {
				action := strings.ToLower(dsnFirstRecipientFields.Get("Action"))
				if action == "failed" {
					bounce.Type = models.BounceTypeHard
					meta.ClassifyReason = fmt.Sprintf("smtp_action=%s", action)
				}
			}

			// Set to default soft bounce if both Status and Action missing (or
			// Status is not a 5.x or 4.x code e.g a success delivery report
			// with 2.x)
			if bounce.Type == "" {
				bounce.Type = models.BounceTypeSoft
				meta.ClassifyReason = "default"
			}

			meta.BounceDetails = []string{dsnFirstRecipientFields.Get("Diagnostic-Code")}
			remoteMTA := strings.Replace(strings.ToLower(dsnFirstRecipientFields.Get("Remote-MTA")), "dns;", "", 1)
			if (remoteMTA) != "" {
				meta.BounceDetails = append(meta.BounceDetails, fmt.Sprintf("Remote-MTA: %s", strings.TrimSpace(remoteMTA)))
			}

		case "text/rfc822-headers", "message/rfc822", "message/global":
			partBody := bufio.NewReader(part.Body)

			originalHeaders, err := textproto.ReadHeader(partBody)
			if err != nil {
				return bounce, fmt.Errorf("parsing text/rfc822-headers or message/rfc822 headers: %v", err)
			}

			bounce.SubscriberUUID = originalHeaders.Get(models.EmailHeaderSubscriberUUID)
			bounce.CampaignUUID = originalHeaders.Get(models.EmailHeaderCampaignUUID)

			meta.From = originalHeaders.Get(models.EmailHeaderFrom)
			meta.Subject = originalHeaders.Get(models.EmailHeaderSubject)
			meta.MessageID = originalHeaders.Get(models.EmailHeaderMessageId)
			meta.DeliveredTo = originalHeaders.Get(models.EmailHeaderDeliveredTo)

			receivedHeaders := originalHeaders.FieldsByKey(models.EmailHeaderReceived)
			for receivedHeaders.Next() {
				meta.Received = append(meta.Received, receivedHeaders.Value())
			}

			if bounce.CreatedAt.IsZero() {
				bounce.CreatedAt, _ = parseTime(originalHeaders.Get(models.EmailHeaderDate))
			}

		case "message/feedback-report":
			partBody := bufio.NewReader(part.Body)

			arfMessageFields, err := textproto.ReadHeader(partBody)
			if err != nil {
				return bounce, fmt.Errorf("parsing message/feedback-report message fields: %v", err)
			}

			// Check we are a rfc5965 abuse report as message/feedback-report
			// must contain "Feedback-Type: abuse" field
			feedbackType := arfMessageFields.Get("Feedback-Type")
			if strings.ToLower(feedbackType) != "abuse" {
				return bounce, fmt.Errorf("message/feedback-report does not contain \"Feedback-Type: abuse\"")
			}

			originalRecipient := arfMessageFields.Get("Original-Rcpt-to")
			if originalRecipient != "" {
				bounce.Email = strings.TrimSpace(originalRecipient)
			}

			// Abuse report = Hard bounce
			bounce.Type = models.BounceTypeHard
			meta.ClassifyReason = "rfc5965 abuse report"
			meta.BounceDetails = []string{fmt.Sprintf("User-Agent: %s", arfMessageFields.Get("User-Agent"))}
		}
	}

	// Clear subscriber email if we managed to find their UUID. Set it in
	// meta.DeliveredTo if empty instead.
	if bounce.SubscriberUUID != "" {
		if meta.DeliveredTo == "" {
			meta.DeliveredTo = bounce.Email
		}
		bounce.Email = ""
	}

	if meta.MessageID == "" {
		meta.MessageID = originalMessageId
	}

	if bounce.CreatedAt.IsZero() {
		bounce.CreatedAt = reportMsgDate
	}

	if bounce.Type == "" {
		bounce.Type = models.BounceTypeSoft
		meta.ClassifyReason = "default"
	}

	bounce.Meta, _ = json.Marshal(meta)
	return bounce, nil
}


// regexParser uses regex to attempt to extract key headers expected in a bounce
// message to populate `models.Bounce`.
func regexParser(m *message.Entity) (bounce models.Bounce, err error) {
	inputBytes, err := io.ReadAll(m.Body)
	if err != nil {
		return bounce, fmt.Errorf("error reading regexParser input: %v", err)
	}

	// Use regexp to attempt to find headers in raw message body.
	hdr := make(map[string]string, 7)
	for _, l := range headerLookups {
		if m := l.Regexp.FindAllSubmatch(inputBytes, -1); m != nil {
			hdr[l.Header] = html.UnescapeString(strings.TrimSpace(string(m[len(m)-1][1])))
		}
	}

	var msgReceived []string
	if u := reHdrReceived.FindAllSubmatch(inputBytes, -1); u != nil {
		for _, m := range u {
			msgReceived = append(
				msgReceived,
				strings.Join(strings.Fields(reUnfold.ReplaceAllString(string(m[1]), " ")), " "))
		}
	}

	date, _ := parseTime(m.Header.Get(models.EmailHeaderDate))
	if date.IsZero() {
		date, _ = parseTime(hdr[models.EmailHeaderDate])
	}
	if date.IsZero() {
		date = time.Now()
	}

	// Classify the bounce type based on message content.
	bounceType, bounceReason := classifyBounce(inputBytes)

	// Additional bounce e-mail metadata.
	meta, _ := json.Marshal(bounceMeta{
		From:           hdr[models.EmailHeaderFrom],
		Subject:        hdr[models.EmailHeaderSubject],
		MessageID:      hdr[models.EmailHeaderMessageId],
		DeliveredTo:    hdr[models.EmailHeaderDeliveredTo],
		Received:       msgReceived,
		ClassifyReason: bounceReason,
	})

	return models.Bounce{
		Type:           bounceType,
		CampaignUUID:   hdr[models.EmailHeaderCampaignUUID],
		SubscriberUUID: hdr[models.EmailHeaderSubscriberUUID],
		CreatedAt:      date,
		Meta:           meta,
	}, nil
}

// classifyBounce analyzes the bounce message content and determines if it's a
// hard or soft bounce. It checks SMTP status codes, diagnostic headers, and
// bounce keywords (using string heuristics). soft is the default preference.
// Returns the bounce type and a classification reason containing context about
// what matched.
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
