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

// // notifData represents params commonly used across different notification
// // templates.
// type notifData struct {
// 	RootURL string
// 	LogoURL string
// }

// sendNotification sends out an e-mail notification to admins.
func (app *App) sendNotification(toEmails []string, subject, tplName string, data interface{}) error {
	var b bytes.Buffer
	if err := app.notifTpls.ExecuteTemplate(&b, tplName, data); err != nil {
		app.log.Printf("error compiling notification template '%s': %v", tplName, err)
		return err
	}

	err := app.manager.PushMessage(manager.Message{
		From:      app.constants.FromEmail,
		To:        toEmails,
		Subject:   subject,
		Body:      b.Bytes(),
		Messenger: "email",
	})

	if err != nil {
		app.log.Printf("error sending admin notification (%s): %v", subject, err)
		return err
	}

	return nil
}
