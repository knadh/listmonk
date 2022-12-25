package main

import (
	"fmt"
	"net/http"
	"net/textproto"
	"strings"

	"github.com/knadh/listmonk/internal/manager"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

// handleSendTxMessage handles the sending of a transactional message.
func handleSendTxMessage(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		m   models.TxMessage
	)

	if err := c.Bind(&m); err != nil {
		return err
	}

	// Validate input.
	if r, err := validateTxMessage(m, app); err != nil {
		return err
	} else {
		m = r
	}

	// Get the cached tx template.
	tpl, err := app.manager.GetTpl(m.TemplateID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("globals.messages.notFound", "name", fmt.Sprintf("template %d", m.TemplateID)))
	}

	var (
		num      = len(m.SubscriberEmails)
		isEmails = true
	)
	if len(m.SubscriberIDs) > 0 {
		num = len(m.SubscriberIDs)
		isEmails = false
	}

	notFound := []string{}
	for n := 0; n < num; n++ {
		var (
			subID    int
			subEmail string
		)

		if !isEmails {
			subID = m.SubscriberIDs[n]
		} else {
			subEmail = m.SubscriberEmails[n]
		}

		// Get the subscriber.
		sub, err := app.core.GetSubscriber(subID, "", subEmail)
		if err != nil {
			// If the subscriber is not found, log that error and move on without halting on the list.
			if er, ok := err.(*echo.HTTPError); ok && er.Code == http.StatusBadRequest {
				notFound = append(notFound, fmt.Sprintf("%v", er.Message))
				continue
			}

			return err
		}

		// Render the message.
		if err := m.Render(sub, tpl); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest,
				app.i18n.Ts("globals.messages.errorFetching", "name"))
		}

		// Prepare the final message.
		msg := manager.Message{}
		msg.Subscriber = sub
		msg.To = []string{sub.Email}
		msg.From = m.FromEmail
		msg.Subject = m.Subject
		msg.ContentType = m.ContentType
		msg.Messenger = m.Messenger
		msg.Body = m.Body

		// Optional headers.
		if len(m.Headers) != 0 {
			msg.Headers = make(textproto.MIMEHeader, len(m.Headers))
			for _, set := range m.Headers {
				for hdr, val := range set {
					msg.Headers.Add(hdr, val)
				}
			}
		}

		if err := app.manager.PushMessage(msg); err != nil {
			app.log.Printf("error sending message (%s): %v", msg.Subject, err)
			return err
		}
	}

	if len(notFound) > 0 {
		return echo.NewHTTPError(http.StatusBadRequest, strings.Join(notFound, "; "))
	}

	return c.JSON(http.StatusOK, okResp{true})
}

func validateTxMessage(m models.TxMessage, app *App) (models.TxMessage, error) {
	if len(m.SubscriberEmails) > 0 && m.SubscriberEmail != "" {
		return m, echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("globals.messages.invalidFields", "name", "do not send `subscriber_email`"))
	}
	if len(m.SubscriberIDs) > 0 && m.SubscriberID != 0 {
		return m, echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("globals.messages.invalidFields", "name", "do not send `subscriber_id`"))
	}

	if m.SubscriberEmail != "" {
		m.SubscriberEmails = append(m.SubscriberEmails, m.SubscriberEmail)
	}

	if m.SubscriberID != 0 {
		m.SubscriberIDs = append(m.SubscriberIDs, m.SubscriberID)
	}

	if (len(m.SubscriberEmails) == 0 && len(m.SubscriberIDs) == 0) || (len(m.SubscriberEmails) > 0 && len(m.SubscriberIDs) > 0) {
		return m, echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("globals.messages.invalidFields", "name", "send subscriber_emails OR subscriber_ids"))
	}

	for n, email := range m.SubscriberEmails {
		if m.SubscriberEmail != "" {
			em, err := app.importer.SanitizeEmail(email)
			if err != nil {
				return m, echo.NewHTTPError(http.StatusBadRequest, err.Error())
			}
			m.SubscriberEmails[n] = em
		}
	}

	if m.FromEmail == "" {
		m.FromEmail = app.constants.FromEmail
	}

	if m.Messenger == "" {
		m.Messenger = emailMsgr
	} else if !app.manager.HasMessenger(m.Messenger) {
		return m, echo.NewHTTPError(http.StatusBadRequest, app.i18n.Ts("campaigns.fieldInvalidMessenger", "name", m.Messenger))
	}

	return m, nil
}
