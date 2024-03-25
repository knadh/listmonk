package main

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/knadh/listmonk/models"
)

const (
	notifTplImport       = "import-status"
	notifTplCampaign     = "campaign-status"
	notifSubscriberOptin = "subscriber-optin"
	notifSubscriberData  = "subscriber-data"
)

var (
	reTitle = regexp.MustCompile(`(?s)<title\s*data-i18n\s*>(.+?)</title>`)
)

// notifData represents params commonly used across different notification
// templates.
type notifData struct {
	RootURL string
	LogoURL string
}

// sendNotification sends out an e-mail notification to admins.
func (app *App) sendNotification(toEmails []string, subject, tplName string, data interface{}) error {
	if len(toEmails) == 0 {
		return nil
	}

	var buf bytes.Buffer
	if err := app.notifTpls.tpls.ExecuteTemplate(&buf, tplName, data); err != nil {
		app.log.Printf("error compiling notification template '%s': %v", tplName, err)
		return err
	}
	body := buf.Bytes()

	subject, body = getTplSubject(subject, body)

	m := models.Message{}
	m.ContentType = app.notifTpls.contentType
	m.From = app.constants.FromEmail
	m.To = toEmails
	m.Subject = subject
	m.Body = body
	m.Messenger = emailMsgr
	if err := app.manager.PushMessage(m); err != nil {
		app.log.Printf("error sending admin notification (%s): %v", subject, err)
		return err
	}
	return nil
}

// getTplSubject extrcts any custom i18n subject rendered in the given rendered
// template body. If it's not found, the incoming subject and body are returned.
func getTplSubject(subject string, body []byte) (string, []byte) {
	m := reTitle.FindSubmatch(body)
	if len(m) != 2 {
		return subject, body
	}

	return strings.TrimSpace(string(m[1])), reTitle.ReplaceAll(body, []byte(""))
}
