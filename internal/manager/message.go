package manager

import (
	"bytes"
	"fmt"

	"github.com/knadh/listmonk/models"
)

// NewCampaignMessage creates and returns a CampaignMessage that is made available
// to message templates while they're compiled. It represents a message from
// a campaign that's bound to a single Subscriber.
func (m *Manager) NewCampaignMessage(c *models.Campaign, s models.Subscriber) (CampaignMessage, error) {
	msg := CampaignMessage{
		Campaign:   c,
		Subscriber: s,

		subject:  c.Subject,
		from:     c.FromEmail,
		to:       s.Email,
		unsubURL: fmt.Sprintf(m.cfg.UnsubURL, c.UUID, s.UUID),
	}

	if err := msg.render(); err != nil {
		return msg, err
	}

	return msg, nil
}

// render takes a Message, executes its pre-compiled Campaign.Tpl
// and applies the resultant bytes to Message.body to be used in messages.
func (m *CampaignMessage) render() error {
	out := bytes.Buffer{}

	// Render the subject if it's a template.
	if m.Campaign.SubjectTpl != nil {
		if err := m.Campaign.SubjectTpl.ExecuteTemplate(&out, models.ContentTpl, m); err != nil {
			return err
		}
		m.subject = out.String()
		out.Reset()
	}

	// Compile the main template.
	if err := m.Campaign.Tpl.ExecuteTemplate(&out, models.BaseTpl, m); err != nil {
		return err
	}
	m.body = out.Bytes()

	// Is there an alt body?
	if m.Campaign.ContentType != models.CampaignContentTypePlain && m.Campaign.AltBody.Valid {
		if m.Campaign.AltBodyTpl != nil {
			b := bytes.Buffer{}
			if err := m.Campaign.AltBodyTpl.ExecuteTemplate(&b, models.ContentTpl, m); err != nil {
				return err
			}
			m.altBody = b.Bytes()
		} else {
			m.altBody = []byte(m.Campaign.AltBody.String)
		}
	}

	return nil
}

// Subject returns a copy of the message subject
func (m *CampaignMessage) Subject() string {
	return m.subject
}

// Body returns a copy of the message body.
func (m *CampaignMessage) Body() []byte {
	out := make([]byte, len(m.body))
	copy(out, m.body)
	return out
}

// AltBody returns a copy of the message's alt body.
func (m *CampaignMessage) AltBody() []byte {
	out := make([]byte, len(m.altBody))
	copy(out, m.altBody)
	return out
}
