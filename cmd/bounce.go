package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

// GetBounces handles retrieval of bounce records.
func (h *Handlers) GetBounces(c echo.Context) error {
	// Fetch one bounce from the DB.
	id, _ := strconv.Atoi(c.Param("id"))
	if id > 0 {
		out, err := h.app.core.GetBounce(id)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, okResp{out})
	}

	// Query and fetch bounces from the DB.
	var (
		pg        = h.app.paginator.NewFromURL(c.Request().URL.Query())
		campID, _ = strconv.Atoi(c.QueryParam("campaign_id"))
		source    = c.FormValue("source")
		orderBy   = c.FormValue("order_by")
		order     = c.FormValue("order")
	)
	res, total, err := h.app.core.QueryBounces(campID, 0, source, orderBy, order, pg.Offset, pg.Limit)
	if err != nil {
		return err
	}

	// No results.
	if len(res) == 0 {
		return c.JSON(http.StatusOK, okResp{models.PageResults{Results: []models.Bounce{}}})
	}

	out := models.PageResults{
		Results: res,
		Total:   total,
		Page:    pg.Page,
		PerPage: pg.PerPage,
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// GetSubscriberBounces retrieves a subscriber's bounce records.
func (h *Handlers) GetSubscriberBounces(c echo.Context) error {
	subID, _ := strconv.Atoi(c.Param("id"))
	if subID < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidID"))
	}

	// Query and fetch bounces from the DB.
	out, _, err := h.app.core.QueryBounces(0, subID, "", "", "", 0, 1000)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// DeleteBounces handles bounce deletion, either a single one (ID in the URI), or a list.
func (h *Handlers) DeleteBounces(c echo.Context) error {
	// Is it an /:id call?
	var (
		all, _ = strconv.ParseBool(c.QueryParam("all"))
		idStr  = c.Param("id")

		ids = []int{}
	)
	if idStr != "" {
		id, _ := strconv.Atoi(idStr)
		if id < 1 {
			return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidID"))
		}
		ids = append(ids, id)
	} else if !all {
		// There are multiple IDs in the query string.
		i, err := parseStringIDs(c.Request().URL.Query()["id"])
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.Ts("globals.messages.invalidID", "error", err.Error()))
		}

		if len(i) == 0 {
			return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.Ts("globals.messages.invalidID"))
		}
		ids = i
	}

	// Delete bounces from the DB.
	if err := h.app.core.DeleteBounces(ids); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// BounceWebhook renders the HTML preview of a template.
func (h *Handlers) BounceWebhook(c echo.Context) error {
	// Read the request body instead of using c.Bind() to read to save the entire raw request as meta.
	rawReq, err := io.ReadAll(c.Request().Body)
	if err != nil {
		h.app.log.Printf("error reading ses notification body: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.Ts("globals.messages.internalError"))
	}

	var (
		service = c.Param("service")

		bounces []models.Bounce
	)
	switch true {
	// Native internal webhook.
	case service == "":
		var b models.Bounce
		if err := json.Unmarshal(rawReq, &b); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.Ts("globals.messages.invalidData")+":"+err.Error())
		}

		if bv, err := h.validateBounceFields(b); err != nil {
			return err
		} else {
			b = bv
		}

		if len(b.Meta) == 0 {
			b.Meta = json.RawMessage("{}")
		}

		if b.CreatedAt.Year() == 0 {
			b.CreatedAt = time.Now()
		}

		bounces = append(bounces, b)

	// Amazon SES.
	case service == "ses" && h.app.constants.BounceSESEnabled:
		switch c.Request().Header.Get("X-Amz-Sns-Message-Type") {
		// SNS webhook registration confirmation. Only after these are processed will the endpoint
		// start getting bounce notifications.
		case "SubscriptionConfirmation", "UnsubscribeConfirmation":
			if err := h.app.bounce.SES.ProcessSubscription(rawReq); err != nil {
				h.app.log.Printf("error processing SNS (SES) subscription: %v", err)
				return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidData"))
			}

		// Bounce notification.
		case "Notification":
			b, err := h.app.bounce.SES.ProcessBounce(rawReq)
			if err != nil {
				h.app.log.Printf("error processing SES notification: %v", err)
				return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidData"))
			}
			bounces = append(bounces, b)

		default:
			return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidData"))
		}

	// SendGrid.
	case service == "sendgrid" && h.app.constants.BounceSendgridEnabled:
		var (
			sig = c.Request().Header.Get("X-Twilio-Email-Event-Webhook-Signature")
			ts  = c.Request().Header.Get("X-Twilio-Email-Event-Webhook-Timestamp")
		)

		// Sendgrid sends multiple bounces.
		bs, err := h.app.bounce.Sendgrid.ProcessBounce(sig, ts, rawReq)
		if err != nil {
			h.app.log.Printf("error processing sendgrid notification: %v", err)
			return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidData"))
		}
		bounces = append(bounces, bs...)

	// Postmark.
	case service == "postmark" && h.app.constants.BouncePostmarkEnabled:
		bs, err := h.app.bounce.Postmark.ProcessBounce(rawReq, c)
		if err != nil {
			h.app.log.Printf("error processing postmark notification: %v", err)
			if _, ok := err.(*echo.HTTPError); ok {
				return err
			}

			return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidData"))
		}
		bounces = append(bounces, bs...)

	// ForwardEmail.
	case service == "forwardemail" && h.app.constants.BounceForwardemailEnabled:
		var (
			sig = c.Request().Header.Get("X-Webhook-Signature")
		)

		bs, err := h.app.bounce.Forwardemail.ProcessBounce(sig, rawReq)
		if err != nil {
			h.app.log.Printf("error processing forwardemail notification: %v", err)
			if _, ok := err.(*echo.HTTPError); ok {
				return err
			}

			return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidData"))
		}
		bounces = append(bounces, bs...)

	default:
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.Ts("bounces.unknownService"))
	}

	// Insert bounces into the DB.
	for _, b := range bounces {
		if err := h.app.bounce.Record(b); err != nil {
			h.app.log.Printf("error recording bounce: %v", err)
		}
	}

	return c.JSON(http.StatusOK, okResp{true})
}

func (h *Handlers) validateBounceFields(b models.Bounce) (models.Bounce, error) {
	if b.Email == "" && b.SubscriberUUID == "" {
		return b, echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.Ts("globals.messages.invalidFields", "name", "email / subscriber_uuid"))
	}

	if b.SubscriberUUID != "" && !reUUID.MatchString(b.SubscriberUUID) {
		return b, echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.Ts("globals.messages.invalidFields", "name", "subscriber_uuid"))
	}

	if b.Email != "" {
		em, err := h.app.importer.SanitizeEmail(b.Email)
		if err != nil {
			return b, echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		b.Email = em
	}

	if b.Type != models.BounceTypeHard && b.Type != models.BounceTypeSoft && b.Type != models.BounceTypeComplaint {
		return b, echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.Ts("globals.messages.invalidFields", "name", "type"))
	}

	return b, nil
}
