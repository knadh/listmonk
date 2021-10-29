package main

import (
	"bytes"
	"html/template"

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

	//check to see if we have any custom templates defined in the Admin Dashboard
	s, err := GetSettings(app)
	if err != nil {
		return err
	}

	header := []byte(s.AdminCustomTemplateHeader)
	footer := []byte(s.AdminCustomTemplateFooter)

	//duplicate the default template and replace any custom templates
	dupTpl, _ := app.notifTpls.Clone()
	if len(header) != 0 {
		header, _ := template.New("header").Parse(string(header))
		dupTpl.AddParseTree("header", header.Tree) 
	}

	if len(footer) != 0 {
		footer, _ := template.New("footer").Parse(string(footer))
		dupTpl.AddParseTree("footer", footer.Tree) 
	}

	var b bytes.Buffer
	if err := dupTpl.ExecuteTemplate(&b, tplName, data); err != nil {
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

func GetDefaultEmailTemplate(app *App, tplName string) []byte {
	var b bytes.Buffer
	var i interface{}
	if err := app.notifTpls.ExecuteTemplate(&b, tplName, i); err != nil {
		app.log.Printf("error retrieving template '%s': %v", tplName, err)
		return nil
	}

	return b.Bytes()
}
