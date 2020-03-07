package main

import (
	"bytes"
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
func sendNotification(toEmails []string, subject, tplName string, data interface{}, app *App) error {
	var b bytes.Buffer
	if err := app.notifTpls.ExecuteTemplate(&b, tplName, data); err != nil {
		app.log.Printf("error compiling notification template '%s': %v", tplName, err)
		return err
	}

	err := app.messenger.Push(app.constants.FromEmail,
		toEmails,
		subject,
		b.Bytes(),
		nil)
	if err != nil {
		app.log.Printf("error sending admin notification (%s): %v", subject, err)
		return err
	}
	return nil
}
