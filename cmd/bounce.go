package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/knadh/listmonk/internal/auth"
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
	subID := getID(c)

	// Check if the user has access to at least one of the lists on the subscriber.
	if err := a.hasSubPerm(auth.GetUser(c), []int{subID}); err != nil {
		return err
	}

	// Query and fetch bounces from the DB.
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

// bounceWebhookResult contains the outcome of processing a webhook request.
// Most providers return bounce records, while subscription validation endpoints
// may return an immediate response body instead.
type bounceWebhookResult struct {
	bounces     []models.Bounce
	response    []byte
	hasResponse bool
}

type bounceWebhookHandler func(echo.Context, []byte) (bounceWebhookResult, error)

// BounceWebhook handles incoming bounce webhook notifications from various providers.
func (a *App) BounceWebhook(c echo.Context) error {
	// If bounce processing is disabled, a.bounce will be nil.
	// Return early to prevent nil pointer dereference.
	if a.bounce == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable,
			a.i18n.Ts("globals.messages.internalError"))
	}

	// Read the request body instead of using c.Bind() to save the entire raw request as meta.
	rawReq, err := io.ReadAll(c.Request().Body)
	if err != nil {
		a.log.Printf("error reading bounce notification body: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.internalError"))
	}

	handler, ok := a.bounceWebhookHandlers()[c.Param("service")]
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("bounces.unknownService"))
	}

	result, err := handler(c, rawReq)
	if err != nil {
		return err
	}

	if result.hasResponse {
		return c.JSONBlob(http.StatusOK, result.response)
	}

	// Insert bounces into the DB.
	for _, b := range result.bounces {
		if err := a.bounce.Record(b); err != nil {
			a.log.Printf("error recording bounce: %v", err)
		}
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// bounceWebhookHandlers registers only the webhook providers that are enabled.
// Each handler encapsulates the processing rules for a single provider.
func (a *App) bounceWebhookHandlers() map[string]bounceWebhookHandler {
	handlers := map[string]bounceWebhookHandler{
		"": a.processNativeBounceWebhook,
	}

	if a.bounce.SES != nil {
		handlers["ses"] = a.processSESBounceWebhook
	}
	if a.bounce.Azure != nil {
		handlers["azure"] = a.processAzureBounceWebhook
	}
	if a.bounce.Sendgrid != nil {
		handlers["sendgrid"] = a.processSendgridBounceWebhook
	}
	if a.bounce.Postmark != nil {
		handlers["postmark"] = a.processPostmarkBounceWebhook
	}
	if a.bounce.Forwardemail != nil {
		handlers["forwardemail"] = a.processForwardEmailBounceWebhook
	}
	if a.bounce.Lettermint != nil {
		handlers["lettermint"] = a.processLettermintBounceWebhook
	}

	return handlers
}

func (a *App) processNativeBounceWebhook(_ echo.Context, rawReq []byte) (bounceWebhookResult, error) {
	var b models.Bounce
	if err := json.Unmarshal(rawReq, &b); err != nil {
		return bounceWebhookResult{}, echo.NewHTTPError(
			http.StatusBadRequest,
			a.i18n.Ts("globals.messages.invalidData")+":"+err.Error(),
		)
	}

	validatedBounce, err := a.validateBounceFields(b)
	if err != nil {
		return bounceWebhookResult{}, err
	}
	b = validatedBounce

	if len(b.Meta) == 0 {
		b.Meta = json.RawMessage("{}")
	}
	if b.CreatedAt.Year() == 0 {
		b.CreatedAt = time.Now()
	}

	return bounceWebhookResult{bounces: []models.Bounce{b}}, nil
}

func (a *App) processSESBounceWebhook(c echo.Context, rawReq []byte) (bounceWebhookResult, error) {
	switch c.Request().Header.Get("X-Amz-Sns-Message-Type") {
	case "SubscriptionConfirmation", "UnsubscribeConfirmation":
		if err := a.bounce.SES.ProcessSubscription(rawReq); err != nil {
			a.log.Printf("error processing SNS (SES) subscription: %v", err)
			return bounceWebhookResult{}, echo.NewHTTPError(
				http.StatusBadRequest,
				a.i18n.T("globals.messages.invalidData"),
			)
		}
		return bounceWebhookResult{}, nil

	case "Notification":
		b, err := a.bounce.SES.ProcessBounce(rawReq)
		if err != nil {
			a.log.Printf("error processing SES notification: %v", err)
			return bounceWebhookResult{}, echo.NewHTTPError(
				http.StatusBadRequest,
				a.i18n.T("globals.messages.invalidData"),
			)
		}
		return bounceWebhookResult{bounces: []models.Bounce{b}}, nil

	default:
		return bounceWebhookResult{}, echo.NewHTTPError(
			http.StatusBadRequest,
			a.i18n.T("globals.messages.invalidData"),
		)
	}
}

func (a *App) processAzureBounceWebhook(c echo.Context, rawReq []byte) (bounceWebhookResult, error) {
	switch c.Request().Header.Get("aeg-event-type") {
	case "SubscriptionValidation", "SubscriptionValidationEvent":
		res, err := a.bounce.Azure.ProcessSubscription(rawReq)
		if err != nil {
			a.log.Printf("error processing Azure Event Grid subscription validation: %v", err)
			return bounceWebhookResult{}, echo.NewHTTPError(
				http.StatusBadRequest,
				a.i18n.T("globals.messages.invalidData"),
			)
		}
		return bounceWebhookResult{response: res, hasResponse: true}, nil

	case "", "Notification":
		bounces, err := a.bounce.Azure.ProcessBounce(c.Request(), rawReq)
		if err != nil {
			a.log.Printf("error processing Azure Event Grid notification: %v", err)
			return bounceWebhookResult{}, echo.NewHTTPError(
				http.StatusBadRequest,
				a.i18n.T("globals.messages.invalidData"),
			)
		}
		return bounceWebhookResult{bounces: bounces}, nil

	default:
		return bounceWebhookResult{}, echo.NewHTTPError(
			http.StatusBadRequest,
			a.i18n.T("globals.messages.invalidData"),
		)
	}
}

func (a *App) processSendgridBounceWebhook(c echo.Context, rawReq []byte) (bounceWebhookResult, error) {
	signature := c.Request().Header.Get("X-Twilio-Email-Event-Webhook-Signature")
	timestamp := c.Request().Header.Get("X-Twilio-Email-Event-Webhook-Timestamp")

	bounces, err := a.bounce.Sendgrid.ProcessBounce(signature, timestamp, rawReq)
	if err != nil {
		a.log.Printf("error processing sendgrid notification: %v", err)
		return bounceWebhookResult{}, echo.NewHTTPError(
			http.StatusBadRequest,
			a.i18n.T("globals.messages.invalidData"),
		)
	}

	return bounceWebhookResult{bounces: bounces}, nil
}

func (a *App) processPostmarkBounceWebhook(c echo.Context, rawReq []byte) (bounceWebhookResult, error) {
	bounces, err := a.bounce.Postmark.ProcessBounce(rawReq, c)
	if err != nil {
		a.log.Printf("error processing postmark notification: %v", err)
		return bounceWebhookResult{}, a.normalizeBounceWebhookError(err)
	}

	return bounceWebhookResult{bounces: bounces}, nil
}

func (a *App) processForwardEmailBounceWebhook(c echo.Context, rawReq []byte) (bounceWebhookResult, error) {
	signature := c.Request().Header.Get("X-Webhook-Signature")
	bounces, err := a.bounce.Forwardemail.ProcessBounce(signature, rawReq)
	if err != nil {
		a.log.Printf("error processing forwardemail notification: %v", err)
		return bounceWebhookResult{}, a.normalizeBounceWebhookError(err)
	}

	return bounceWebhookResult{bounces: bounces}, nil
}

func (a *App) processLettermintBounceWebhook(c echo.Context, rawReq []byte) (bounceWebhookResult, error) {
	signature := c.Request().Header.Get("X-Lettermint-Signature")
	bounces, err := a.bounce.Lettermint.ProcessBounce(signature, rawReq)
	if err != nil {
		a.log.Printf("error processing lettermint notification: %v", err)
		return bounceWebhookResult{}, a.normalizeBounceWebhookError(err)
	}

	return bounceWebhookResult{bounces: bounces}, nil
}

func (a *App) normalizeBounceWebhookError(err error) error {
	if _, ok := err.(*echo.HTTPError); ok {
		return err
	}

	return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidData"))
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
