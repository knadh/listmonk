package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

	m, err := parseTxMessage(c, app)
	if err != nil {
		return err
	}

	// Validate input.
	if r, err := validateTxMessage(m, app); err != nil {
		return err
	} else {
		m = r
	}

	// Get the template
	tpl, err := getTemplate(app, m.TemplateID)
	if err != nil {
		return err
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
		msg := models.CreateMailMessage(sub, m)

		if err := sendEmail(app, msg); err != nil {
			return err
		}
	}

	if len(notFound) > 0 {
		return echo.NewHTTPError(http.StatusBadRequest, strings.Join(notFound, "; "))
	}

	return c.JSON(http.StatusOK, okResp{true})
}

func parseTxMessage(c echo.Context, app *App) (models.TxMessage, error) {
	m := models.TxMessage{}
	// If it's a multipart form, there may be file attachments.
	if strings.HasPrefix(c.Request().Header.Get("Content-Type"), "multipart/form-data") {
		if data, attachments, err := parseMultiPartMessageDetails(c, app); err != nil {
			return models.TxMessage{}, err
		} else {
			// Parse the JSON data.
			if err := json.Unmarshal([]byte(data[0]), &m); err != nil {
				return models.TxMessage{}, echo.NewHTTPError(http.StatusBadRequest,
					app.i18n.Ts("globals.messages.invalidFields", "name", fmt.Sprintf("data: %s", err.Error())))
			}

			m.Attachments = append(m.Attachments, attachments...)
		}
	} else if err := c.Bind(&m); err != nil {
		return models.TxMessage{}, err
	}
	return m, nil
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

// handleSendExternalTxMessage handles the sending of a transactional message to an external recipient.
func handleSendExternalTxMessage(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		m   models.ExternalTxMessage
	)

	m, err := parseExternalTxMessage(c, app)
	if err != nil {
		return err
	}

	// Validate input.
	if r, err := validateExternalTxMessage(m, app); err != nil {
		return err
	} else {
		m = r
	}

	// Get the template
	tpl, err := getTemplate(app, m.TemplateID)
	if err != nil {
		return err
	}

	txMessage := m.MapToTxMessage()
	notFound := []string{}
	for n := 0; n < len(txMessage.SubscriberEmails); n++ {
		// Render the message.
		if err := txMessage.Render(models.Subscriber{}, tpl); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest,
				app.i18n.Ts("globals.messages.errorFetching", "name"))
		}

		// Prepare the final message.
		msg := models.CreateMailMessage(models.Subscriber{Email: txMessage.SubscriberEmails[n]}, txMessage)

		if err := sendEmail(app, msg); err != nil {
			return err
		}
	}

	if len(notFound) > 0 {
		return echo.NewHTTPError(http.StatusBadRequest, strings.Join(notFound, "; "))
	}

	return c.JSON(http.StatusOK, okResp{true})
}

func parseExternalTxMessage(c echo.Context, app *App) (models.ExternalTxMessage, error) {
	m := models.ExternalTxMessage{}
	// If it's a multipart form, there may be file attachments.
	if strings.HasPrefix(c.Request().Header.Get("Content-Type"), "multipart/form-data") {
		if data, attachments, err := parseMultiPartMessageDetails(c, app); err != nil {
			return models.ExternalTxMessage{}, err
		} else {
			// Parse the JSON data.
			if err := json.Unmarshal([]byte(data[0]), &m); err != nil {
				return models.ExternalTxMessage{}, echo.NewHTTPError(http.StatusBadRequest,
					app.i18n.Ts("globals.messages.invalidFields", "name", fmt.Sprintf("data: %s", err.Error())))
			}

			m.Attachments = append(m.Attachments, attachments...)
		}
	} else if err := c.Bind(&m); err != nil {
		return models.ExternalTxMessage{}, err
	}
	return m, nil
}

func validateExternalTxMessage(m models.ExternalTxMessage, app *App) (models.ExternalTxMessage, error) {
	if len(m.RecipientEmails) > 0 && m.RecipientEmail != "" {
		return m, echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("globals.messages.invalidFields", "name", "do not send `subscriber_email`"))
	}

	if m.RecipientEmail != "" {
		m.RecipientEmails = append(m.RecipientEmails, m.RecipientEmail)
	}

	for n, email := range m.RecipientEmails {
		if m.RecipientEmail != "" {
			em, err := app.importer.SanitizeEmail(email)
			if err != nil {
				return m, echo.NewHTTPError(http.StatusBadRequest, err.Error())
			}
			m.RecipientEmails[n] = em
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

func parseMultiPartMessageDetails(c echo.Context, app *App) ([]string, []models.Attachment, error) {
	form, err := c.MultipartForm()
	if err != nil {
		return nil, nil, echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("globals.messages.invalidFields", "name", err.Error()))
	}

	data, ok := form.Value["data"]
	if !ok || len(data) != 1 {
		return nil, nil, echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("globals.messages.invalidFields", "name", "data"))
	}

	attachments := []models.Attachment{}
	// Attach files.
	for _, f := range form.File["file"] {
		file, err := f.Open()
		if err != nil {
			return nil, nil, echo.NewHTTPError(http.StatusInternalServerError,
				app.i18n.Ts("globals.messages.invalidFields", "name", fmt.Sprintf("file: %s", err.Error())))
		}
		defer file.Close()

		b, err := io.ReadAll(file)
		if err != nil {
			return nil, nil, echo.NewHTTPError(http.StatusInternalServerError,
				app.i18n.Ts("globals.messages.invalidFields", "name", fmt.Sprintf("file: %s", err.Error())))
		}

		attachments = append(attachments, models.Attachment{
			Name:    f.Filename,
			Header:  manager.MakeAttachmentHeader(f.Filename, "base64", f.Header.Get("Content-Type")),
			Content: b,
		})
	}

	return data, attachments, nil
}

func getTemplate(app *App, templateId int) (*models.Template, error) {
	// Get the cached tx template.
	tpl, err := app.manager.GetTpl(templateId)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("globals.messages.notFound", "name", fmt.Sprintf("template %d", templateId)))
	}
	return tpl, nil
}

func sendEmail(app *App, msg models.Message) error {
	if err := app.manager.PushMessage(msg); err != nil {
		app.log.Printf("error sending message (%s): %v", msg.Subject, err)
		return err
	}
	return nil
}
