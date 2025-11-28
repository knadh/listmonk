package models

import (
	"bytes"
	"fmt"
	"html/template"
	"net/textproto"
	"strings"
	txttpl "text/template"
)

// Message is the message pushed to a Messenger.
type Message struct {
	From        string
	To          []string
	Subject     string
	ContentType string
	Body        []byte
	AltBody     []byte
	Headers     textproto.MIMEHeader
	Attachments []Attachment

	Subscriber Subscriber

	// Campaign is generally the same instance for a large number of subscribers.
	Campaign *Campaign

	// Messenger is the messenger backend to use: email|postback.
	Messenger string
}

// Attachment represents a file or blob attachment that can be
// sent along with a message by a Messenger.
type Attachment struct {
	Name    string
	Header  textproto.MIMEHeader
	Content []byte
}

// TxMessage subscriber modes.
const (
	TxSubModeDefault  = "default"
	TxSubModeFallback = "fallback"
	TxSubModeExternal = "external"
)

// TxMessage represents an e-mail campaign.
type TxMessage struct {
	SubscriberMode   string   `json:"subscriber_mode"`
	SubscriberEmails []string `json:"subscriber_emails"`
	SubscriberIDs    []int    `json:"subscriber_ids"`

	// Deprecated.
	SubscriberEmail string `json:"subscriber_email"`
	SubscriberID    int    `json:"subscriber_id"`

	TemplateID  int            `json:"template_id"`
	Data        map[string]any `json:"data"`
	FromEmail   string         `json:"from_email"`
	Headers     Headers        `json:"headers"`
	ContentType string         `json:"content_type"`
	Messenger   string         `json:"messenger"`
	Subject     string         `json:"subject"`

	// File attachments added from multi-part form data.
	Attachments []Attachment `json:"-"`

	Body       []byte             `json:"-"`
	Tpl        *template.Template `json:"-"`
	SubjectTpl *txttpl.Template   `json:"-"`
}

func (m *TxMessage) Render(sub Subscriber, tpl *Template) error {
	data := struct {
		Subscriber Subscriber
		Tx         *TxMessage
	}{sub, m}

	// Render the body.
	b := bytes.Buffer{}
	if err := tpl.Tpl.ExecuteTemplate(&b, BaseTpl, data); err != nil {
		return err
	}
	m.Body = make([]byte, b.Len())
	copy(m.Body, b.Bytes())
	b.Reset()

	// Was a subject provided in the message?
	var (
		subjTpl *txttpl.Template
		subject = m.Subject
	)
	if subject != "" {
		if strings.Contains(m.Subject, "{{") {
			// If the subject has a template string, render that.
			s, err := txttpl.New(BaseTpl).Funcs(txttpl.FuncMap(nil)).Parse(m.Subject)
			if err != nil {
				return fmt.Errorf("error compiling subject: %v", err)
			}
			subjTpl = s
		}
	} else {
		// Use the subject from the template.
		subject = tpl.Subject
		subjTpl = tpl.SubjectTpl
	}

	// If the subject is also a template, render that.
	if subjTpl != nil {
		if err := subjTpl.ExecuteTemplate(&b, BaseTpl, data); err != nil {
			return err
		}
		m.Subject = b.String()
		b.Reset()
	} else {
		m.Subject = subject
	}

	return nil
}
