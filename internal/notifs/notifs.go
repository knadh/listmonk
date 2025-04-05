package notifs

import (
	"bytes"
	"html/template"
	"log"
	"net/textproto"
	"regexp"
	"strings"

	"github.com/knadh/listmonk/models"
)

type FuncPush func(msg models.Message) error
type FuncNotif func(toEmails []string, subject, tplName string, data any, headers textproto.MIMEHeader) error
type FuncNotifSystem func(subject, tplName string, data any, headers textproto.MIMEHeader) error

type Opt struct {
	FromEmail    string
	SystemEmails []string
	ContentType  string
}

var (
	reTitle = regexp.MustCompile(`(?s)<title\s*data-i18n\s*>(.+?)</title>`)
)

type Notifs struct {
	Tpls   *template.Template
	pushFn FuncPush
	lo     *log.Logger

	opt Opt
}

// NewNotifs returns a new Notifs instance.
func NewNotifs(opt Opt, tpls *template.Template, pushFn FuncPush, lo *log.Logger) *Notifs {
	return &Notifs{
		opt:    opt,
		Tpls:   tpls,
		pushFn: pushFn,
		lo:     lo,
	}
}

// NotifySystem sends out an e-mail notification to the admin emails.
func (n *Notifs) NotifySystem(subject, tplName string, data any, hdr textproto.MIMEHeader) error {
	return n.Notify(n.opt.SystemEmails, subject, tplName, data, hdr)
}

// Notify sends out an e-mail notification.
func (n *Notifs) Notify(toEmails []string, subject, tplName string, data any, hdr textproto.MIMEHeader) error {
	if len(toEmails) == 0 {
		return nil
	}

	var buf bytes.Buffer
	if err := n.Tpls.ExecuteTemplate(&buf, tplName, data); err != nil {
		n.lo.Printf("error compiling notification template '%s': %v", tplName, err)
		return err
	}
	body := buf.Bytes()

	subject, body = getTplSubject(subject, body)

	m := models.Message{
		Messenger:   "email",
		ContentType: n.opt.ContentType,
		From:        n.opt.FromEmail,
		To:          toEmails,
		Subject:     subject,
		Body:        body,
		Headers:     hdr,
	}

	// Send the message.
	if err := n.pushFn(m); err != nil {
		n.lo.Printf("error sending admin notification (%s): %v", subject, err)
		return err
	}

	return nil
}

// getTplSubject extracts any custom i18n subject rendered in the given rendered
// template body. If it's not found, the incoming subject and body are returned.
func getTplSubject(subject string, body []byte) (string, []byte) {
	m := reTitle.FindSubmatch(body)
	if len(m) != 2 {
		return subject, body
	}

	return strings.TrimSpace(string(m[1])), reTitle.ReplaceAll(body, []byte(""))
}
