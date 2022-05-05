package main

import (
	"bytes"

	"github.com/knadh/listmonk/internal/manager"
)

const (
	notifTplImport       = "import-status"
	notifTplCampaign     = "campaign-status"
	notifSubscriberOptin = "subscriber-optin"
	notifSubscriberData  = "subscriber-data"
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

	var b bytes.Buffer
	if err := app.notifTpls.tpls.ExecuteTemplate(&b, tplName, data); err != nil {
		app.log.Printf("error compiling notification template '%s': %v", tplName, err)
		return err
	}

	m := manager.Message{}
	m.ContentType = app.notifTpls.contentType
	m.From = app.constants.FromEmail
	m.To = toEmails
	m.Subject = subject
	m.Body = b.Bytes()
	m.Messenger = emailMsgr
	if err := app.manager.PushMessage(m); err != nil {
		app.log.Printf("error sending admin notification (%s): %v", subject, err)
		return err
	}
	return nil
}
