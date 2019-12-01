package main

import (
	"bytes"
	"html/template"

	"github.com/knadh/stuffbin"
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
	if err := app.NotifTpls.ExecuteTemplate(&b, tplName, data); err != nil {
		app.Logger.Printf("error compiling notification template '%s': %v", tplName, err)
		return err
	}

	err := app.Messenger.Push(app.Constants.FromEmail,
		toEmails,
		subject,
		b.Bytes(),
		nil)
	if err != nil {
		app.Logger.Printf("error sending admin notification (%s): %v", subject, err)
		return err
	}
	return nil
}

// compileNotifTpls compiles and returns e-mail notification templates that are
// used for sending ad-hoc notifications to admins and subscribers.
func compileNotifTpls(path string, fs stuffbin.FileSystem, app *App) (*template.Template, error) {
	// Register utility functions that the e-mail templates can use.
	funcs := template.FuncMap{
		"RootURL": func() string {
			return app.Constants.RootURL
		},
		"LogoURL": func() string {
			return app.Constants.LogoURL
		}}

	tpl, err := stuffbin.ParseTemplatesGlob(funcs, fs, "/email-templates/*.html")
	if err != nil {
		return nil, err
	}

	return tpl, err
}
