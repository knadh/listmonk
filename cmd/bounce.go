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

// GetBounce handles retrieval of a specific bounce record by ID.
func (a *App) GetBounce(c echo.Context) error {
	// Fetch one bounce from the DB.
	id := getID(c)
	out, err := a.core.GetBounce(id)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// GetBounces handles retrieval of bounce records.
func (a *App) GetBounces(c echo.Context) error {
	var (
		campID, _ = strconv.Atoi(c.QueryParam("campaign_id"))
		source    = c.FormValue("source")
		orderBy   = c.FormValue("order_by")
		order     = c.FormValue("order")

		pg = a.pg.NewFromURL(c.Request().URL.Query())
	)

	// Query and fetch bounces from the DB.
	res, total, err := a.core.QueryBounces(campID, 0, source, orderBy, order, pg.Offset, pg.Limit)
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
func (a *App) GetSubscriberBounces(c echo.Context) error {
	// Query and fetch bounces from the DB.
	subID := getID(c)
	out, _, err := a.core.QueryBounces(0, subID, "", "", "", 0, 1000)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// DeleteBounces handles bounce deletion of a list.
func (a *App) DeleteBounces(c echo.Context) error {
	all, _ := strconv.ParseBool(c.QueryParam("all"))

	var ids []int
	if !all {
		// There are multiple IDs in the query string.
		res, err := parseStringIDs(c.Request().URL.Query()["id"])
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidID", "error", err.Error()))
		}
		if len(res) == 0 {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidID"))
		}

		ids = res
	}

	// Delete bounces from the DB.
	if err := a.core.DeleteBounces(ids, all); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// DeleteBounce handles bounce deletion of a single bounce record.
func (a *App) DeleteBounce(c echo.Context) error {
	// Delete bounces from the DB.
	id := getID(c)
	if err := a.core.DeleteBounces([]int{id}, false); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// BlocklistBouncedSubscribers handles blocklisting of all bounced subscribers.
func (a *App) BlocklistBouncedSubscribers(c echo.Context) error {
	if err := a.core.BlocklistBouncedSubscribers(); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// BounceWebhook renders the HTML preview of a template.
func (a *App) BounceWebhook(c echo.Context) error {
	// Read the request body instead of using c.Bind() to read to save the entire raw request as meta.
	rawReq, err := io.ReadAll(c.Request().Body)
	if err != nil {
		a.log.Printf("error reading ses notification body: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.internalError"))
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
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidData")+":"+err.Error())
		}

		if bv, err := a.validateBounceFields(b); err != nil {
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
	case service == "ses" && a.cfg.BounceSESEnabled:
		switch c.Request().Header.Get("X-Amz-Sns-Message-Type") {
		// SNS webhook registration confirmation. Only after these are processed will the endpoint
		// start getting bounce notifications.
		case "SubscriptionConfirmation", "UnsubscribeConfirmation":
			if err := a.bounce.SES.ProcessSubscription(rawReq); err != nil {
				a.log.Printf("error processing SNS (SES) subscription: %v", err)
				return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidData"))
			}

		// Bounce notification.
		case "Notification":
			b, err := a.bounce.SES.ProcessBounce(rawReq)
			if err != nil {
				a.log.Printf("error processing SES notification: %v", err)
				return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidData"))
			}
			bounces = append(bounces, b)

		default:
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidData"))
		}

	// SendGrid.
	case service == "sendgrid" && a.cfg.BounceSendgridEnabled:
		var (
			sig = c.Request().Header.Get("X-Twilio-Email-Event-Webhook-Signature")
			ts  = c.Request().Header.Get("X-Twilio-Email-Event-Webhook-Timestamp")
		)

		// Sendgrid sends multiple bounces.
		bs, err := a.bounce.Sendgrid.ProcessBounce(sig, ts, rawReq)
		if err != nil {
			a.log.Printf("error processing sendgrid notification: %v", err)
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidData"))
		}
		bounces = append(bounces, bs...)

	// Postmark.
	case service == "postmark" && a.cfg.BouncePostmarkEnabled:
		bs, err := a.bounce.Postmark.ProcessBounce(rawReq, c)
		if err != nil {
			a.log.Printf("error processing postmark notification: %v", err)
			if _, ok := err.(*echo.HTTPError); ok {
				return err
			}

			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidData"))
		}
		bounces = append(bounces, bs...)

	// ForwardEmail.
	case service == "forwardemail" && a.cfg.BounceForwardemailEnabled:
		var (
			sig = c.Request().Header.Get("X-Webhook-Signature")
		)

		bs, err := a.bounce.Forwardemail.ProcessBounce(sig, rawReq)
		if err != nil {
			a.log.Printf("error processing forwardemail notification: %v", err)
			if _, ok := err.(*echo.HTTPError); ok {
				return err
			}

			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidData"))
		}
		bounces = append(bounces, bs...)

	default:
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("bounces.unknownService"))
	}

	// Insert bounces into the DB.
	for _, b := range bounces {
		if err := a.bounce.Record(b); err != nil {
			a.log.Printf("error recording bounce: %v", err)
		}
	}

	return c.JSON(http.StatusOK, okResp{true})
}

func (a *App) validateBounceFields(b models.Bounce) (models.Bounce, error) {
	if b.Email == "" && b.SubscriberUUID == "" {
		return b, echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "email / subscriber_uuid"))
	}

	if b.SubscriberUUID != "" && !reUUID.MatchString(b.SubscriberUUID) {
		return b, echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "subscriber_uuid"))
	}

	if b.Email != "" {
		em, err := a.importer.SanitizeEmail(b.Email)
		if err != nil {
			return b, echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		b.Email = em
	}

	if b.Type != models.BounceTypeHard && b.Type != models.BounceTypeSoft && b.Type != models.BounceTypeComplaint {
		return b, echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "type"))
	}

	return b, nil
}
