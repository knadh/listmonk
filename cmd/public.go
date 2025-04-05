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
	AssetVersion        string
	EnablePublicSubPage bool
	EnablePublicArchive bool
	IndividualTracking  bool
}

// tplData is the data container that is injected
// into public templates for accessing data.
type tplData struct {
	SiteName            string
	RootURL             string
	LogoURL             string
	FaviconURL          string
	AssetVersion        string
	EnablePublicSubPage bool
	EnablePublicArchive bool
	IndividualTracking  bool
	Data                any
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

type optinReq struct {
	SubUUID   string
	ListUUIDs []string      `query:"l" form:"l"`
	Lists     []models.List `query:"-" form:"-"`
}

type optinTpl struct {
	publicTpl
	optinReq
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
func (t *tplRenderer) Render(w io.Writer, name string, data any, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, tplData{
		SiteName:            t.SiteName,
		RootURL:             t.RootURL,
		LogoURL:             t.LogoURL,
		FaviconURL:          t.FaviconURL,
		AssetVersion:        t.AssetVersion,
		EnablePublicSubPage: t.EnablePublicSubPage,
		EnablePublicArchive: t.EnablePublicArchive,
		IndividualTracking:  t.IndividualTracking,
		Data:                data,
		L:                   c.Get("app").(*App).i18n,
	})
}

// GetPublicLists returns the list of public lists with minimal fields
// required to submit a subscription.
func (h *Handlers) GetPublicLists(c echo.Context) error {
	// Get all public lists.
	lists, err := h.app.core.GetLists(models.ListTypePublic, true, nil)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("public.errorFetchingLists"))
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

// ViewCampaignMessage renders the HTML view of a campaign message.
// This is the view the {{ MessageURL }} template tag links to in e-mail campaigns.
func (h *Handlers) ViewCampaignMessage(c echo.Context) error {
	// Get the campaign.
	campUUID := c.Param("campUUID")
	camp, err := h.app.core.GetCampaign(0, campUUID, "")
	if err != nil {
		if er, ok := err.(*echo.HTTPError); ok {
			if er.Code == http.StatusBadRequest {
				return c.Render(http.StatusNotFound, tplMessage,
					makeMsgTpl(h.app.i18n.T("public.notFoundTitle"), "", h.app.i18n.T("public.campaignNotFound")))
			}
		}

		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(h.app.i18n.T("public.errorTitle"), "", h.app.i18n.Ts("public.errorFetchingCampaign")))
	}

	// Get the subscriber.
	subUUID := c.Param("subUUID")
	sub, err := h.app.core.GetSubscriber(0, subUUID, "")
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Render(http.StatusNotFound, tplMessage,
				makeMsgTpl(h.app.i18n.T("public.notFoundTitle"), "", h.app.i18n.T("public.errorFetchingEmail")))
		}

		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(h.app.i18n.T("public.errorTitle"), "", h.app.i18n.Ts("public.errorFetchingCampaign")))
	}

	// Compile the template.
	if err := camp.CompileTemplate(h.app.manager.TemplateFuncs(&camp)); err != nil {
		h.app.log.Printf("error compiling template: %v", err)
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(h.app.i18n.T("public.errorTitle"), "", h.app.i18n.Ts("public.errorFetchingCampaign")))
	}

	// Render the message body.
	msg, err := h.app.manager.NewCampaignMessage(&camp, sub)
	if err != nil {
		h.app.log.Printf("error rendering message: %v", err)
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(h.app.i18n.T("public.errorTitle"), "", h.app.i18n.Ts("public.errorFetchingCampaign")))
	}

	return c.HTML(http.StatusOK, string(msg.Body()))
}

// SubscriptionPage renders the subscription management page and handles unsubscriptions.
// This is the view that {{ UnsubscribeURL }} in campaigns link to.
func (h *Handlers) SubscriptionPage(c echo.Context) error {
	var (
		subUUID       = c.Param("subUUID")
		showManage, _ = strconv.ParseBool(c.FormValue("manage"))
	)

	// Get the subscriber from the DB.
	s, err := h.app.core.GetSubscriber(0, subUUID, "")
	if err != nil {
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(h.app.i18n.T("public.errorTitle"), "", h.app.i18n.Ts("public.errorProcessingRequest")))
	}

	// Prepare the public template.
	out := unsubTpl{
		Subscriber:       s,
		SubUUID:          subUUID,
		publicTpl:        publicTpl{Title: h.app.i18n.T("public.unsubscribeTitle")},
		AllowBlocklist:   h.app.constants.Privacy.AllowBlocklist,
		AllowExport:      h.app.constants.Privacy.AllowExport,
		AllowWipe:        h.app.constants.Privacy.AllowWipe,
		AllowPreferences: h.app.constants.Privacy.AllowPreferences,
	}

	// If the subscriber is blocklisted, throw an error.
	if s.Status == models.SubscriberStatusBlockListed {
		return c.Render(http.StatusOK, tplMessage, makeMsgTpl(h.app.i18n.T("public.noSubTitle"), "", h.app.i18n.Ts("public.blocklisted")))
	}

	// Only show preference management if it's enabled in settings.
	if h.app.constants.Privacy.AllowPreferences {
		out.ShowManage = showManage

		// Get the subscriber's lists from the DB to render in the template.
		subs, err := h.app.core.GetSubscriptions(0, subUUID, false)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("public.errorFetchingLists"))
		}

		out.Subscriptions = make([]models.Subscription, 0, len(subs))
		for _, s := range subs {
			// Private lists shouldn't be rendered in the template.
			if s.Type == models.ListTypePrivate {
				continue
			}

			out.Subscriptions = append(out.Subscriptions, s)
		}
	}

	return c.Render(http.StatusOK, "subscription", out)
}

// SubscriptionPrefs renders the subscription management page and
// s unsubscriptions. This is the view that {{ UnsubscribeURL }} in
// campaigns link to.
func (h *Handlers) SubscriptionPrefs(c echo.Context) error {
	// Read the form.
	var req struct {
		Name      string   `form:"name" json:"name"`
		ListUUIDs []string `form:"l" json:"list_uuids"`
		Blocklist bool     `form:"blocklist" json:"blocklist"`
		Manage    bool     `form:"manage" json:"manage"`
	}
	if err := c.Bind(&req); err != nil {
		return c.Render(http.StatusBadRequest, tplMessage,
			makeMsgTpl(h.app.i18n.T("public.errorTitle"), "", h.app.i18n.T("globals.messages.invalidData")))
	}

	// Simple unsubscribe.
	var (
		campUUID  = c.Param("campUUID")
		subUUID   = c.Param("subUUID")
		blocklist = h.app.constants.Privacy.AllowBlocklist && req.Blocklist
	)
	if !req.Manage || blocklist {
		if err := h.app.core.UnsubscribeByCampaign(subUUID, campUUID, blocklist); err != nil {
			return c.Render(http.StatusInternalServerError, tplMessage,
				makeMsgTpl(h.app.i18n.T("public.errorTitle"), "", h.app.i18n.T("public.errorProcessingRequest")))
		}

		return c.Render(http.StatusOK, tplMessage,
			makeMsgTpl(h.app.i18n.T("public.unsubbedTitle"), "", h.app.i18n.T("public.unsubbedInfo")))
	}

	// Is preference management enabled?
	if !h.app.constants.Privacy.AllowPreferences {
		return c.Render(http.StatusBadRequest, tplMessage,
			makeMsgTpl(h.app.i18n.T("public.errorTitle"), "", h.app.i18n.T("public.invalidFeature")))
	}

	// Manage preferences.
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || len(req.Name) > 256 {
		return c.Render(http.StatusBadRequest, tplMessage,
			makeMsgTpl(h.app.i18n.T("public.errorTitle"), "", h.app.i18n.T("subscribers.invalidName")))
	}

	// Get the subscriber from the DB.
	sub, err := h.app.core.GetSubscriber(0, subUUID, "")
	if err != nil {
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(h.app.i18n.T("public.errorTitle"), "", h.app.i18n.Ts("globals.messages.pFound",
				"name", h.app.i18n.T("globals.terms.subscriber"))))
	}
	sub.Name = req.Name

	// Update the subscriber properties in the DB.
	if _, err := h.app.core.UpdateSubscriber(sub.ID, sub); err != nil {
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(h.app.i18n.T("public.errorTitle"), "", h.app.i18n.T("public.errorProcessingRequest")))
	}

	// Get the subscriber's lists and whatever is not sent in the request (unchecked),
	// unsubscribe them.
	reqUUIDs := make(map[string]struct{})
	for _, u := range req.ListUUIDs {
		reqUUIDs[u] = struct{}{}
	}

	// Get subscription from teh DB.
	subs, err := h.app.core.GetSubscriptions(0, subUUID, false)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("public.errorFetchingLists"))
	}

	// Filter the lists in the request against the subscriptions in the DB.
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
	if err := h.app.core.UnsubscribeLists([]int{sub.ID}, nil, unsubUUIDs); err != nil {
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(h.app.i18n.T("public.errorTitle"), "", h.app.i18n.T("public.errorProcessingRequest")))

	}

	return c.Render(http.StatusOK, tplMessage,
		makeMsgTpl(h.app.i18n.T("globals.messages.done"), "", h.app.i18n.T("public.prefsSaved")))
}

// OptinPage renders the double opt-in confirmation page that subscribers
// see when they click on the "Confirm subscription" button in double-optin
// notifications.
func (h *Handlers) OptinPage(c echo.Context) error {
	var (
		subUUID    = c.Param("subUUID")
		confirm, _ = strconv.ParseBool(c.FormValue("confirm"))
		req        optinReq
	)
	if err := c.Bind(&req); err != nil {
		return err
	}

	// Validate list UUIDs if there are incoming UUIDs in the request.
	if len(req.ListUUIDs) > 0 {
		for _, l := range req.ListUUIDs {
			if !reUUID.MatchString(l) {
				return c.Render(http.StatusBadRequest, tplMessage,
					makeMsgTpl(h.app.i18n.T("public.errorTitle"), "", h.app.i18n.T("globals.messages.invalidUUID")))
			}
		}
	}

	// Get the list of subscription lists where the subscriber hasn't confirmed.
	lists, err := h.app.core.GetSubscriberLists(0, subUUID, nil, req.ListUUIDs, models.SubscriptionStatusUnconfirmed, "")
	if err != nil {
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(h.app.i18n.T("public.errorTitle"), "", h.app.i18n.Ts("public.errorFetchingLists")))
	}

	// There are no lists to confirm.
	if len(lists) == 0 {
		return c.Render(http.StatusOK, tplMessage,
			makeMsgTpl(h.app.i18n.T("public.noSubTitle"), "", h.app.i18n.Ts("public.noSubInfo")))
	}

	// Confirm.
	if confirm {
		meta := models.JSON{}
		if h.app.constants.Privacy.RecordOptinIP {
			if h := c.Request().Header.Get("X-Forwarded-For"); h != "" {
				meta["optin_ip"] = h
			} else if h := c.Request().RemoteAddr; h != "" {
				meta["optin_ip"] = strings.Split(h, ":")[0]
			}
		}

		// Confirm subscriptions in the DB.
		if err := h.app.core.ConfirmOptionSubscription(subUUID, req.ListUUIDs, meta); err != nil {
			h.app.log.Printf("error unsubscribing: %v", err)
			return c.Render(http.StatusInternalServerError, tplMessage,
				makeMsgTpl(h.app.i18n.T("public.errorTitle"), "", h.app.i18n.Ts("public.errorProcessingRequest")))
		}

		return c.Render(http.StatusOK, tplMessage,
			makeMsgTpl(h.app.i18n.T("public.subConfirmedTitle"), "", h.app.i18n.Ts("public.subConfirmed")))
	}

	var out optinTpl
	out.Lists = lists
	out.SubUUID = subUUID
	out.Title = h.app.i18n.T("public.confirmOptinSubTitle")

	return c.Render(http.StatusOK, "optin", out)
}

// SubscriptionFormPage handles subscription requests coming from public
// HTML subscription forms.
func (h *Handlers) SubscriptionFormPage(c echo.Context) error {
	if !h.app.constants.EnablePublicSubPage {
		return c.Render(http.StatusNotFound, tplMessage,
			makeMsgTpl(h.app.i18n.T("public.errorTitle"), "", h.app.i18n.Ts("public.invalidFeature")))
	}

	// Get all public lists from the DB.
	lists, err := h.app.core.GetLists(models.ListTypePublic, true, nil)
	if err != nil {
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(h.app.i18n.T("public.errorTitle"), "", h.app.i18n.Ts("public.errorFetchingLists")))
	}

	// There are no public lists available for subscription.
	if len(lists) == 0 {
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(h.app.i18n.T("public.errorTitle"), "", h.app.i18n.Ts("public.noListsAvailable")))
	}

	out := subFormTpl{}
	out.Title = h.app.i18n.T("public.sub")
	out.Lists = lists

	// Captcha is enabled. Set the key for the template to render.
	if h.app.constants.Security.EnableCaptcha {
		out.CaptchaKey = h.app.constants.Security.CaptchaKey
	}

	return c.Render(http.StatusOK, "subscription-form", out)
}

// SubscriptionForm handles subscription requests coming from public
// HTML subscription forms.
func (h *Handlers) SubscriptionForm(c echo.Context) error {
	// If there's a nonce value, a bot could've filled the form.
	if c.FormValue("nonce") != "" {
		return echo.NewHTTPError(http.StatusBadGateway, h.app.i18n.T("public.invalidFeature"))
	}

	// Process CAPTCHA.
	if h.app.constants.Security.EnableCaptcha {
		err, ok := h.app.captcha.Verify(c.FormValue("h-captcha-response"))
		if err != nil {
			h.app.log.Printf("Captcha request failed: %v", err)
		}

		if !ok {
			return c.Render(http.StatusBadRequest, tplMessage,
				makeMsgTpl(h.app.i18n.T("public.errorTitle"), "", h.app.i18n.T("public.invalidCaptcha")))
		}
	}

	hasOptin, err := h.processSubForm(c)
	if err != nil {
		e, ok := err.(*echo.HTTPError)
		if !ok {
			return e
		}

		return c.Render(e.Code, tplMessage, makeMsgTpl(h.app.i18n.T("public.errorTitle"), "", fmt.Sprintf("%s", e.Message)))
	}

	// If there were double optin lists, show the opt-in pending message instead of
	// the subscription confirmation message.
	msg := "public.subConfirmed"
	if hasOptin {
		msg = "public.subOptinPending"
	}

	return c.Render(http.StatusOK, tplMessage, makeMsgTpl(h.app.i18n.T("public.subTitle"), "", h.app.i18n.Ts(msg)))
}

// PublicSubscription handles subscription requests coming from public
// API calls.
func (h *Handlers) PublicSubscription(c echo.Context) error {
	if !h.app.constants.EnablePublicSubPage {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("public.invalidFeature"))
	}

	hasOptin, err := h.processSubForm(c)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{struct {
		HasOptin bool `json:"has_optin"`
	}{hasOptin}})
}

// LinkRedirect redirects a link UUID to its original underlying link
// after recording the link click for a particular subscriber in the particular
// campaign. These links are generated by {{ TrackLink }} tags in campaigns.
func (h *Handlers) LinkRedirect(c echo.Context) error {
	// If individual tracking is disabled, do not record the subscriber ID.
	subUUID := c.Param("subUUID")
	if !h.app.constants.Privacy.IndividualTracking {
		subUUID = ""
	}

	// Inser the link click in the DB.
	var (
		linkUUID = c.Param("linkUUID")
		campUUID = c.Param("campUUID")
	)
	url, err := h.app.core.RegisterCampaignLinkClick(linkUUID, campUUID, subUUID)
	if err != nil {
		e := err.(*echo.HTTPError)
		return c.Render(e.Code, tplMessage, makeMsgTpl(h.app.i18n.T("public.errorTitle"), "", e.Error()))
	}

	return c.Redirect(http.StatusTemporaryRedirect, url)
}

// RegisterCampaignView registers a campaign view which comes in
// the form of an pixel image request. Regardless of errors, this handler
// should always render the pixel image bytes. The pixel URL is generated by
// the {{ TrackView }} template tag in campaigns.
func (h *Handlers) RegisterCampaignView(c echo.Context) error {
	// If individual tracking is disabled, do not record the subscriber ID.
	subUUID := c.Param("subUUID")
	if !h.app.constants.Privacy.IndividualTracking {
		subUUID = ""
	}

	// Exclude dummy hits from template previews.
	campUUID := c.Param("campUUID")
	if campUUID != dummyUUID && subUUID != dummyUUID {
		if err := h.app.core.RegisterCampaignView(campUUID, subUUID); err != nil {
			h.app.log.Printf("error registering campaign view: %s", err)
		}
	}

	c.Response().Header().Set("Cache-Control", "no-cache")
	return c.Blob(http.StatusOK, "image/png", pixelPNG)
}

// SelfExportSubscriberData pulls the subscriber's profile, list subscriptions,
// campaign views and clicks and produces a JSON report that is then e-mailed
// to the subscriber. This is a privacy feature and the data that's exported
// is dependent on the configuration.
func (h *Handlers) SelfExportSubscriberData(c echo.Context) error {
	// Is export allowed?
	if !h.app.constants.Privacy.AllowExport {
		return c.Render(http.StatusBadRequest, tplMessage,
			makeMsgTpl(h.app.i18n.T("public.errorTitle"), "", h.app.i18n.Ts("public.invalidFeature")))
	}

	// Get the subscriber's data. A single query that gets the profile,
	// list subscriptions, campaign views, and link clicks. Names of
	// private lists are replaced with "Private list".
	subUUID := c.Param("subUUID")
	data, b, err := h.exportSubscriberData(0, subUUID, h.app.constants.Privacy.Exportable)
	if err != nil {
		h.app.log.Printf("error exporting subscriber data: %s", err)
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(h.app.i18n.T("public.errorTitle"), "", h.app.i18n.Ts("public.errorProcessingRequest")))
	}

	// Prepare the attachment e-mail.
	var msg bytes.Buffer
	if err := h.app.notifTpls.tpls.ExecuteTemplate(&msg, notifSubscriberData, data); err != nil {
		h.app.log.Printf("error compiling notification template '%s': %v", notifSubscriberData, err)
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(h.app.i18n.T("public.errorTitle"), "", h.app.i18n.Ts("public.errorProcessingRequest")))
	}

	var (
		subject = h.app.i18n.Ts("email.data.title")
		body    = msg.Bytes()
	)
	subject, body = getTplSubject(subject, body)

	// E-mail the data as a JSON attachment to the subscriber.
	const fname = "data.json"
	if err := h.app.emailMessenger.Push(models.Message{
		ContentType: h.app.notifTpls.contentType,
		From:        h.app.constants.FromEmail,
		To:          []string{data.Email},
		Subject:     subject,
		Body:        body,
		Attachments: []models.Attachment{
			{
				Name:    fname,
				Content: b,
				Header:  manager.MakeAttachmentHeader(fname, "base64", "application/json"),
			},
		},
	}); err != nil {
		h.app.log.Printf("error e-mailing subscriber profile: %s", err)
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(h.app.i18n.T("public.errorTitle"), "", h.app.i18n.Ts("public.errorProcessingRequest")))
	}

	return c.Render(http.StatusOK, tplMessage,
		makeMsgTpl(h.app.i18n.T("public.dataSentTitle"), "", h.app.i18n.T("public.dataSent")))
}

// WipeSubscriberData allows a subscriber to delete their data. The
// profile and subscriptions are deleted, while the campaign_views and link
// clicks remain as orphan data unconnected to any subscriber.
func (h *Handlers) WipeSubscriberData(c echo.Context) error {
	// Is wiping allowed?
	if !h.app.constants.Privacy.AllowWipe {
		return c.Render(http.StatusBadRequest, tplMessage,
			makeMsgTpl(h.app.i18n.T("public.errorTitle"), "", h.app.i18n.Ts("public.invalidFeature")))
	}

	subUUID := c.Param("subUUID")
	if err := h.app.core.DeleteSubscribers(nil, []string{subUUID}); err != nil {
		h.app.log.Printf("error wiping subscriber data: %s", err)
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(h.app.i18n.T("public.errorTitle"), "", h.app.i18n.Ts("public.errorProcessingRequest")))
	}

	return c.Render(http.StatusOK, tplMessage,
		makeMsgTpl(h.app.i18n.T("public.dataRemovedTitle"), "", h.app.i18n.T("public.dataRemoved")))
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
func (h *Handlers) processSubForm(c echo.Context) (bool, error) {
	// Get and validate fields.
	var req struct {
		Name          string   `form:"name" json:"name"`
		Email         string   `form:"email" json:"email"`
		FormListUUIDs []string `form:"l" json:"list_uuids"`
	}
	if err := c.Bind(&req); err != nil {
		return false, err
	}

	if len(req.FormListUUIDs) == 0 {
		return false, echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("public.noListsSelected"))
	}

	// If there's no name, use the name bit from the e-mail.
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		req.Name = strings.Split(req.Email, "@")[0]
	}

	// Validate fields.
	if len(req.Email) > 1000 {
		return false, echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("subscribers.invalidEmail"))
	}

	em, err := h.app.importer.SanitizeEmail(req.Email)
	if err != nil {
		return false, echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	req.Email = em

	req.Name = strings.TrimSpace(req.Name)
	if len(req.Name) == 0 || len(req.Name) > stdInputMaxLen {
		return false, echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("subscribers.invalidName"))
	}

	listUUIDs := pq.StringArray(req.FormListUUIDs)

	// Fetch the list types and ensure that they are not private.
	listTypes, err := h.app.core.GetListTypes(nil, req.FormListUUIDs)
	if err != nil {
		return false, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("%s", err.(*echo.HTTPError).Message))
	}

	for _, t := range listTypes {
		if t == models.ListTypePrivate {
			return false, echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidUUID"))
		}
	}

	// Insert the subscriber into the DB.
	_, hasOptin, err := h.app.core.InsertSubscriber(models.Subscriber{
		Name:   req.Name,
		Email:  req.Email,
		Status: models.SubscriberStatusEnabled,
	}, nil, listUUIDs, false)
	if err != nil {
		// Subscriber already exists. Update subscriptions in the DB.
		if e, ok := err.(*echo.HTTPError); ok && e.Code == http.StatusConflict {
			// Get the subscriber from the DB by their email.
			sub, err := h.app.core.GetSubscriber(0, "", req.Email)
			if err != nil {
				return false, err
			}

			// Update the subscriber's subscriptions in the DB.
			_, hasOptin, err := h.app.core.UpdateSubscriberWithLists(sub.ID, sub, nil, listUUIDs, false, false)
			if err != nil {
				return false, err
			}

			return hasOptin, nil
		}

		return false, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("%s", err.(*echo.HTTPError).Message))
	}

	return hasOptin, nil
}
