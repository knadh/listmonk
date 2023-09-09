package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"html/template"
	"image"
	"image/png"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/knadh/listmonk/internal/i18n"
	"github.com/knadh/listmonk/internal/manager"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
)

const (
	tplMessage = "message"
)

// tplRenderer wraps a template.tplRenderer for echo.
type tplRenderer struct {
	templates           *template.Template
	SiteName            string
	RootURL             string
	LogoURL             string
	FaviconURL          string
	EnablePublicSubPage bool
	EnablePublicArchive bool
}

// tplData is the data container that is injected
// into public templates for accessing data.
type tplData struct {
	SiteName            string
	RootURL             string
	LogoURL             string
	FaviconURL          string
	EnablePublicSubPage bool
	EnablePublicArchive bool
	Data                interface{}
	L                   *i18n.I18n
}

type publicTpl struct {
	Title       string
	Description string
}

type unsubTpl struct {
	publicTpl
	Subscriber       models.Subscriber
	Subscriptions    []models.Subscription
	SubUUID          string
	AllowBlocklist   bool
	AllowExport      bool
	AllowWipe        bool
	AllowPreferences bool
	ShowManage       bool
}

type optinTpl struct {
	publicTpl
	SubUUID   string
	ListUUIDs []string      `query:"l" form:"l"`
	Lists     []models.List `query:"-" form:"-"`
}

type msgTpl struct {
	publicTpl
	MessageTitle string
	Message      string
}

type subFormTpl struct {
	publicTpl
	Lists      []models.List
	CaptchaKey string
}

var (
	pixelPNG = drawTransparentImage(3, 14)
)

// Render executes and renders a template for echo.
func (t *tplRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, tplData{
		SiteName:            t.SiteName,
		RootURL:             t.RootURL,
		LogoURL:             t.LogoURL,
		FaviconURL:          t.FaviconURL,
		EnablePublicSubPage: t.EnablePublicSubPage,
		EnablePublicArchive: t.EnablePublicArchive,
		Data:                data,
		L:                   c.Get("app").(*App).i18n,
	})
}

// handleGetPublicLists returns the list of public lists with minimal fields
// required to submit a subscription.
func handleGetPublicLists(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
	)

	// Get all public lists.
	lists, err := app.core.GetLists(models.ListTypePublic)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("public.errorFetchingLists"))
	}

	type list struct {
		UUID string `json:"uuid"`
		Name string `json:"name"`
	}

	out := make([]list, 0, len(lists))
	for _, l := range lists {
		out = append(out, list{
			UUID: l.UUID,
			Name: l.Name,
		})
	}

	return c.JSON(http.StatusOK, out)
}

// handleViewCampaignMessage renders the HTML view of a campaign message.
// This is the view the {{ MessageURL }} template tag links to in e-mail campaigns.
func handleViewCampaignMessage(c echo.Context) error {
	var (
		app      = c.Get("app").(*App)
		campUUID = c.Param("campUUID")
		subUUID  = c.Param("subUUID")
	)

	// Get the campaign.
	camp, err := app.core.GetCampaign(0, campUUID)
	if err != nil {
		if er, ok := err.(*echo.HTTPError); ok {
			if er.Code == http.StatusBadRequest {
				return c.Render(http.StatusNotFound, tplMessage,
					makeMsgTpl(app.i18n.T("public.notFoundTitle"), "", app.i18n.T("public.campaignNotFound")))
			}
		}

		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(app.i18n.T("public.errorTitle"), "", app.i18n.Ts("public.errorFetchingCampaign")))
	}

	// Get the subscriber.
	sub, err := app.core.GetSubscriber(0, subUUID, "")
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Render(http.StatusNotFound, tplMessage,
				makeMsgTpl(app.i18n.T("public.notFoundTitle"), "", app.i18n.T("public.errorFetchingEmail")))
		}

		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(app.i18n.T("public.errorTitle"), "", app.i18n.Ts("public.errorFetchingCampaign")))
	}

	// Compile the template.
	if err := camp.CompileTemplate(app.manager.TemplateFuncs(&camp)); err != nil {
		app.log.Printf("error compiling template: %v", err)
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(app.i18n.T("public.errorTitle"), "", app.i18n.Ts("public.errorFetchingCampaign")))
	}

	// Render the message body.
	msg, err := app.manager.NewCampaignMessage(&camp, sub)
	if err != nil {
		app.log.Printf("error rendering message: %v", err)
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(app.i18n.T("public.errorTitle"), "", app.i18n.Ts("public.errorFetchingCampaign")))
	}

	return c.HTML(http.StatusOK, string(msg.Body()))
}

// handleSubscriptionPage renders the subscription management page and
// handles unsubscriptions. This is the view that {{ UnsubscribeURL }} in
// campaigns link to.
func handleSubscriptionPage(c echo.Context) error {
	var (
		app           = c.Get("app").(*App)
		subUUID       = c.Param("subUUID")
		showManage, _ = strconv.ParseBool(c.FormValue("manage"))
		out           = unsubTpl{}
	)
	out.SubUUID = subUUID
	out.Title = app.i18n.T("public.unsubscribeTitle")
	out.AllowBlocklist = app.constants.Privacy.AllowBlocklist
	out.AllowExport = app.constants.Privacy.AllowExport
	out.AllowWipe = app.constants.Privacy.AllowWipe
	out.AllowPreferences = app.constants.Privacy.AllowPreferences

	if app.constants.Privacy.AllowPreferences {
		out.ShowManage = showManage
	}

	// Get the subscriber's lists.
	subs, err := app.core.GetSubscriptions(0, subUUID, false)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("public.errorFetchingLists"))
	}

	s, err := app.core.GetSubscriber(0, subUUID, "")
	if err != nil {
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(app.i18n.T("public.errorTitle"), "", app.i18n.Ts("public.errorProcessingRequest")))
	}
	out.Subscriber = s

	if s.Status == models.SubscriberStatusBlockListed {
		return c.Render(http.StatusOK, tplMessage,
			makeMsgTpl(app.i18n.T("public.noSubTitle"), "", app.i18n.Ts("public.blocklisted")))
	}

	// Filter out unrelated private lists.
	if showManage {
		out.Subscriptions = make([]models.Subscription, 0, len(subs))
		for _, s := range subs {
			if s.Type == models.ListTypePrivate {
				continue
			}

			out.Subscriptions = append(out.Subscriptions, s)
		}
	}

	return c.Render(http.StatusOK, "subscription", out)
}

// handleSubscriptionPrefs renders the subscription management page and
// handles unsubscriptions. This is the view that {{ UnsubscribeURL }} in
// campaigns link to.
func handleSubscriptionPrefs(c echo.Context) error {
	var (
		app      = c.Get("app").(*App)
		campUUID = c.Param("campUUID")
		subUUID  = c.Param("subUUID")

		req struct {
			Name      string   `form:"name" json:"name"`
			ListUUIDs []string `form:"l" json:"list_uuids"`
			Blocklist bool     `form:"blocklist" json:"blocklist"`
			Manage    bool     `form:"manage" json:"manage"`
		}
	)

	// Read the form.
	if err := c.Bind(&req); err != nil {
		return c.Render(http.StatusBadRequest, tplMessage,
			makeMsgTpl(app.i18n.T("public.errorTitle"), "", app.i18n.T("globals.messages.invalidData")))
	}

	// Simple unsubscribe.
	blocklist := app.constants.Privacy.AllowBlocklist && req.Blocklist
	if !req.Manage || blocklist {
		if err := app.core.UnsubscribeByCampaign(subUUID, campUUID, blocklist); err != nil {
			return c.Render(http.StatusInternalServerError, tplMessage,
				makeMsgTpl(app.i18n.T("public.errorTitle"), "", app.i18n.T("public.errorProcessingRequest")))
		}

		return c.Render(http.StatusOK, tplMessage,
			makeMsgTpl(app.i18n.T("public.unsubbedTitle"), "", app.i18n.T("public.unsubbedInfo")))
	}

	// Is preference management enabled?
	if !app.constants.Privacy.AllowPreferences {
		return c.Render(http.StatusBadRequest, tplMessage,
			makeMsgTpl(app.i18n.T("public.errorTitle"), "", app.i18n.T("public.invalidFeature")))
	}

	// Manage preferences.
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || len(req.Name) > 256 {
		return c.Render(http.StatusBadRequest, tplMessage,
			makeMsgTpl(app.i18n.T("public.errorTitle"), "", app.i18n.T("subscribers.invalidName")))
	}

	// Get the subscriber from the DB.
	sub, err := app.core.GetSubscriber(0, subUUID, "")
	if err != nil {
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(app.i18n.T("public.errorTitle"), "", app.i18n.Ts("globals.messages.pFound",
				"name", app.i18n.T("globals.terms.subscriber"))))
	}
	sub.Name = req.Name

	// Update name.
	if _, err := app.core.UpdateSubscriber(sub.ID, sub); err != nil {
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(app.i18n.T("public.errorTitle"), "", app.i18n.T("public.errorProcessingRequest")))
	}

	// Get the subscriber's lists and whatever is not sent in the request (unchecked),
	// unsubscribe them.
	reqUUIDs := make(map[string]struct{})
	for _, u := range req.ListUUIDs {
		reqUUIDs[u] = struct{}{}
	}

	subs, err := app.core.GetSubscriptions(0, subUUID, false)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("public.errorFetchingLists"))
	}

	unsubUUIDs := make([]string, 0, len(req.ListUUIDs))
	for _, s := range subs {
		if s.Type == models.ListTypePrivate {
			continue
		}
		if _, ok := reqUUIDs[s.UUID]; !ok {
			unsubUUIDs = append(unsubUUIDs, s.UUID)
		}
	}

	// Unsubscribe from lists.
	if err := app.core.UnsubscribeLists([]int{sub.ID}, nil, unsubUUIDs); err != nil {
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(app.i18n.T("public.errorTitle"), "", app.i18n.T("public.errorProcessingRequest")))

	}

	return c.Render(http.StatusOK, tplMessage,
		makeMsgTpl(app.i18n.T("globals.messages.done"), "", app.i18n.T("public.prefsSaved")))
}

// handleOptinPage renders the double opt-in confirmation page that subscribers
// see when they click on the "Confirm subscription" button in double-optin
// notifications.
func handleOptinPage(c echo.Context) error {
	var (
		app        = c.Get("app").(*App)
		subUUID    = c.Param("subUUID")
		confirm, _ = strconv.ParseBool(c.FormValue("confirm"))
		out        = optinTpl{}
	)
	out.SubUUID = subUUID
	out.Title = app.i18n.T("public.confirmOptinSubTitle")
	out.SubUUID = subUUID

	// Get and validate fields.
	if err := c.Bind(&out); err != nil {
		return err
	}

	// Validate list UUIDs if there are incoming UUIDs in the request.
	if len(out.ListUUIDs) > 0 {
		for _, l := range out.ListUUIDs {
			if !reUUID.MatchString(l) {
				return c.Render(http.StatusBadRequest, tplMessage,
					makeMsgTpl(app.i18n.T("public.errorTitle"), "", app.i18n.T("globals.messages.invalidUUID")))
			}
		}
	}

	// Get the list of subscription lists where the subscriber hasn't confirmed.
	lists, err := app.core.GetSubscriberLists(0, subUUID, nil, out.ListUUIDs, models.SubscriptionStatusUnconfirmed, "")
	if err != nil {
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(app.i18n.T("public.errorTitle"), "", app.i18n.Ts("public.errorFetchingLists")))
	}

	// There are no lists to confirm.
	if len(lists) == 0 {
		return c.Render(http.StatusOK, tplMessage,
			makeMsgTpl(app.i18n.T("public.noSubTitle"), "", app.i18n.Ts("public.noSubInfo")))
	}
	out.Lists = lists

	// Confirm.
	if confirm {
		meta := models.JSON{}
		if app.constants.Privacy.RecordOptinIP {
			if h := c.Request().Header.Get("X-Forwarded-For"); h != "" {
				meta["optin_ip"] = h
			} else if h := c.Request().RemoteAddr; h != "" {
				meta["optin_ip"] = strings.Split(h, ":")[0]
			}
		}

		if err := app.core.ConfirmOptionSubscription(subUUID, out.ListUUIDs, meta); err != nil {
			app.log.Printf("error unsubscribing: %v", err)
			return c.Render(http.StatusInternalServerError, tplMessage,
				makeMsgTpl(app.i18n.T("public.errorTitle"), "", app.i18n.Ts("public.errorProcessingRequest")))
		}

		return c.Render(http.StatusOK, tplMessage,
			makeMsgTpl(app.i18n.T("public.subConfirmedTitle"), "", app.i18n.Ts("public.subConfirmed")))
	}

	return c.Render(http.StatusOK, "optin", out)
}

// handleSubscriptionFormPage handles subscription requests coming from public
// HTML subscription forms.
func handleSubscriptionFormPage(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
	)

	if !app.constants.EnablePublicSubPage {
		return c.Render(http.StatusNotFound, tplMessage,
			makeMsgTpl(app.i18n.T("public.errorTitle"), "", app.i18n.Ts("public.invalidFeature")))
	}

	// Get all public lists.
	lists, err := app.core.GetLists(models.ListTypePublic)
	if err != nil {
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(app.i18n.T("public.errorTitle"), "", app.i18n.Ts("public.errorFetchingLists")))
	}

	if len(lists) == 0 {
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(app.i18n.T("public.errorTitle"), "", app.i18n.Ts("public.noListsAvailable")))
	}

	out := subFormTpl{}
	out.Title = app.i18n.T("public.sub")
	out.Lists = lists

	if app.constants.Security.EnableCaptcha {
		out.CaptchaKey = app.constants.Security.CaptchaKey
	}

	return c.Render(http.StatusOK, "subscription-form", out)
}

// handleSubscriptionForm handles subscription requests coming from public
// HTML subscription forms.
func handleSubscriptionForm(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
	)

	// If there's a nonce value, a bot could've filled the form.
	if c.FormValue("nonce") != "" {
		return echo.NewHTTPError(http.StatusBadGateway, app.i18n.T("public.invalidFeature"))
	}

	// Process CAPTCHA.
	if app.constants.Security.EnableCaptcha {
		err, ok := app.captcha.Verify(c.FormValue("h-captcha-response"))
		if err != nil {
			app.log.Printf("Captcha request failed: %v", err)
		}

		if !ok {
			return c.Render(http.StatusBadRequest, tplMessage,
				makeMsgTpl(app.i18n.T("public.errorTitle"), "", app.i18n.T("public.invalidCaptcha")))
		}
	}

	hasOptin, err := processSubForm(c)
	if err != nil {
		e, ok := err.(*echo.HTTPError)
		if !ok {
			return e
		}

		return c.Render(e.Code, tplMessage,
			makeMsgTpl(app.i18n.T("public.errorTitle"), "", fmt.Sprintf("%s", e.Message)))
	}

	msg := "public.subConfirmed"
	if hasOptin {
		msg = "public.subOptinPending"
	}

	return c.Render(http.StatusOK, tplMessage, makeMsgTpl(app.i18n.T("public.subTitle"), "", app.i18n.Ts(msg)))
}

// handlePublicSubscription handles subscription requests coming from public
// API calls.
func handlePublicSubscription(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
	)

	if !app.constants.EnablePublicSubPage {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("public.invalidFeature"))
	}

	hasOptin, err := processSubForm(c)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{struct {
		HasOptin bool `json:"has_optin"`
	}{hasOptin}})
}

// handleLinkRedirect redirects a link UUID to its original underlying link
// after recording the link click for a particular subscriber in the particular
// campaign. These links are generated by {{ TrackLink }} tags in campaigns.
func handleLinkRedirect(c echo.Context) error {
	var (
		app      = c.Get("app").(*App)
		linkUUID = c.Param("linkUUID")
		campUUID = c.Param("campUUID")
		subUUID  = c.Param("subUUID")
	)

	// If individual tracking is disabled, do not record the subscriber ID.
	if !app.constants.Privacy.IndividualTracking {
		subUUID = ""
	}

	url, err := app.core.RegisterCampaignLinkClick(linkUUID, campUUID, subUUID)
	if err != nil {
		e := err.(*echo.HTTPError)
		return c.Render(e.Code, tplMessage, makeMsgTpl(app.i18n.T("public.errorTitle"), "", e.Error()))
	}

	return c.Redirect(http.StatusTemporaryRedirect, url)
}

// handleRegisterCampaignView registers a campaign view which comes in
// the form of an pixel image request. Regardless of errors, this handler
// should always render the pixel image bytes. The pixel URL is is generated by
// the {{ TrackView }} template tag in campaigns.
func handleRegisterCampaignView(c echo.Context) error {
	var (
		app      = c.Get("app").(*App)
		campUUID = c.Param("campUUID")
		subUUID  = c.Param("subUUID")
	)

	// If individual tracking is disabled, do not record the subscriber ID.
	if !app.constants.Privacy.IndividualTracking {
		subUUID = ""
	}

	// Exclude dummy hits from template previews.
	if campUUID != dummyUUID && subUUID != dummyUUID {
		if err := app.core.RegisterCampaignView(campUUID, subUUID); err != nil {
			app.log.Printf("error registering campaign view: %s", err)
		}
	}

	c.Response().Header().Set("Cache-Control", "no-cache")
	return c.Blob(http.StatusOK, "image/png", pixelPNG)
}

// handleSelfExportSubscriberData pulls the subscriber's profile, list subscriptions,
// campaign views and clicks and produces a JSON report that is then e-mailed
// to the subscriber. This is a privacy feature and the data that's exported
// is dependent on the configuration.
func handleSelfExportSubscriberData(c echo.Context) error {
	var (
		app     = c.Get("app").(*App)
		subUUID = c.Param("subUUID")
	)
	// Is export allowed?
	if !app.constants.Privacy.AllowExport {
		return c.Render(http.StatusBadRequest, tplMessage,
			makeMsgTpl(app.i18n.T("public.errorTitle"), "", app.i18n.Ts("public.invalidFeature")))
	}

	// Get the subscriber's data. A single query that gets the profile,
	// list subscriptions, campaign views, and link clicks. Names of
	// private lists are replaced with "Private list".
	data, b, err := exportSubscriberData(0, subUUID, app.constants.Privacy.Exportable, app)
	if err != nil {
		app.log.Printf("error exporting subscriber data: %s", err)
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(app.i18n.T("public.errorTitle"), "", app.i18n.Ts("public.errorProcessingRequest")))
	}

	// Prepare the attachment e-mail.
	var msg bytes.Buffer
	if err := app.notifTpls.tpls.ExecuteTemplate(&msg, notifSubscriberData, data); err != nil {
		app.log.Printf("error compiling notification template '%s': %v", notifSubscriberData, err)
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(app.i18n.T("public.errorTitle"), "", app.i18n.Ts("public.errorProcessingRequest")))
	}

	// Send the data as a JSON attachment to the subscriber.
	const fname = "data.json"
	if err := app.messengers[emailMsgr].Push(models.Message{
		ContentType: app.notifTpls.contentType,
		From:        app.constants.FromEmail,
		To:          []string{data.Email},
		Subject:     app.i18n.Ts("email.data.title"),
		Body:        msg.Bytes(),
		Attachments: []models.Attachment{
			{
				Name:    fname,
				Content: b,
				Header:  manager.MakeAttachmentHeader(fname, "base64", "application/json"),
			},
		},
	}); err != nil {
		app.log.Printf("error e-mailing subscriber profile: %s", err)
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(app.i18n.T("public.errorTitle"), "", app.i18n.Ts("public.errorProcessingRequest")))
	}

	return c.Render(http.StatusOK, tplMessage,
		makeMsgTpl(app.i18n.T("public.dataSentTitle"), "", app.i18n.T("public.dataSent")))
}

// handleWipeSubscriberData allows a subscriber to delete their data. The
// profile and subscriptions are deleted, while the campaign_views and link
// clicks remain as orphan data unconnected to any subscriber.
func handleWipeSubscriberData(c echo.Context) error {
	var (
		app     = c.Get("app").(*App)
		subUUID = c.Param("subUUID")
	)

	// Is wiping allowed?
	if !app.constants.Privacy.AllowWipe {
		return c.Render(http.StatusBadRequest, tplMessage,
			makeMsgTpl(app.i18n.T("public.errorTitle"), "", app.i18n.Ts("public.invalidFeature")))
	}

	if err := app.core.DeleteSubscribers(nil, []string{subUUID}); err != nil {
		app.log.Printf("error wiping subscriber data: %s", err)
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(app.i18n.T("public.errorTitle"), "", app.i18n.Ts("public.errorProcessingRequest")))
	}

	return c.Render(http.StatusOK, tplMessage,
		makeMsgTpl(app.i18n.T("public.dataRemovedTitle"), "", app.i18n.T("public.dataRemoved")))
}

// drawTransparentImage draws a transparent PNG of given dimensions
// and returns the PNG bytes.
func drawTransparentImage(h, w int) []byte {
	var (
		img = image.NewRGBA(image.Rect(0, 0, w, h))
		out = &bytes.Buffer{}
	)
	_ = png.Encode(out, img)
	return out.Bytes()
}

// processSubForm processes an incoming form/public API subscription request.
// The bool indicates whether there was subscription to an optin list so that
// an appropriate message can be shown.
func processSubForm(c echo.Context) (bool, error) {
	var (
		app = c.Get("app").(*App)
		req struct {
			Name          string   `form:"name" json:"name"`
			Email         string   `form:"email" json:"email"`
			FormListUUIDs []string `form:"l" json:"list_uuids"`
		}
	)

	// Get and validate fields.
	if err := c.Bind(&req); err != nil {
		return false, err
	}

	if len(req.FormListUUIDs) == 0 {
		return false, echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("public.noListsSelected"))
	}

	// If there's no name, use the name bit from the e-mail.
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		req.Name = strings.Split(req.Email, "@")[0]
	}

	// Validate fields.
	if len(req.Email) > 1000 {
		return false, echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("subscribers.invalidEmail"))
	}

	em, err := app.importer.SanitizeEmail(req.Email)
	if err != nil {
		return false, echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	req.Email = em

	req.Name = strings.TrimSpace(req.Name)
	if len(req.Name) == 0 || len(req.Name) > stdInputMaxLen {
		return false, echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("subscribers.invalidName"))
	}

	listUUIDs := pq.StringArray(req.FormListUUIDs)

	// Insert the subscriber into the DB.
	_, hasOptin, err := app.core.InsertSubscriber(models.Subscriber{
		Name:   req.Name,
		Email:  req.Email,
		Status: models.SubscriberStatusEnabled,
	}, nil, listUUIDs, false)
	if err != nil {
		// Subscriber already exists. Update subscriptions.
		if e, ok := err.(*echo.HTTPError); ok && e.Code == http.StatusConflict {
			sub, err := app.core.GetSubscriber(0, "", req.Email)
			if err != nil {
				return false, err
			}

			if _, err := app.core.UpdateSubscriberWithLists(sub.ID, sub, nil, listUUIDs, false, false); err != nil {
				return false, err
			}

			return false, nil
		}

		return false, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("%s", err.(*echo.HTTPError).Message))
	}

	return hasOptin, nil
}
