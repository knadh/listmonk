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

	"github.com/knadh/listmonk/internal/captcha"
	"github.com/knadh/listmonk/internal/i18n"
	"github.com/knadh/listmonk/internal/manager"
	"github.com/knadh/listmonk/internal/notifs"
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
	Lists   []models.List
	Captcha struct {
		Enabled    bool
		Provider   string
		Key        string
		Complexity int
	}
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
func (a *App) GetPublicLists(c echo.Context) error {
	// Get all public lists.
	lists, err := a.core.GetLists(models.ListTypePublic, models.ListStatusActive, true, nil)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("public.errorFetchingLists"))
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
func (a *App) ViewCampaignMessage(c echo.Context) error {
	// Get the campaign.
	campUUID := c.Param("campUUID")
	camp, err := a.core.GetCampaign(0, campUUID, "")
	if err != nil {
		if er, ok := err.(*echo.HTTPError); ok {
			if er.Code == http.StatusBadRequest {
				return c.Render(http.StatusNotFound, tplMessage,
					makeMsgTpl(a.i18n.T("public.notFoundTitle"), "", a.i18n.T("public.campaignNotFound")))
			}
		}

		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(a.i18n.T("public.errorTitle"), "", a.i18n.Ts("public.errorFetchingCampaign")))
	}

	// Get the subscriber.
	subUUID := c.Param("subUUID")
	sub, err := a.core.GetSubscriber(0, subUUID, "")
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Render(http.StatusNotFound, tplMessage,
				makeMsgTpl(a.i18n.T("public.notFoundTitle"), "", a.i18n.T("public.errorFetchingEmail")))
		}

		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(a.i18n.T("public.errorTitle"), "", a.i18n.Ts("public.errorFetchingCampaign")))
	}

	// Compile the template.
	if err := camp.CompileTemplate(a.manager.TemplateFuncs(&camp)); err != nil {
		a.log.Printf("error compiling template: %v", err)
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(a.i18n.T("public.errorTitle"), "", a.i18n.Ts("public.errorFetchingCampaign")))
	}

	// Render the message body.
	msg, err := a.manager.NewCampaignMessage(&camp, sub)
	if err != nil {
		a.log.Printf("error rendering message: %v", err)
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(a.i18n.T("public.errorTitle"), "", a.i18n.Ts("public.errorFetchingCampaign")))
	}

	return c.HTML(http.StatusOK, string(msg.Body()))
}

// SubscriptionPage renders the subscription management page and handles unsubscriptions.
// This is the view that {{ UnsubscribeURL }} in campaigns link to.
func (a *App) SubscriptionPage(c echo.Context) error {
	var (
		subUUID       = c.Param("subUUID")
		showManage, _ = strconv.ParseBool(c.FormValue("manage"))
	)

	// Get the subscriber from the DB.
	s, err := a.core.GetSubscriber(0, subUUID, "")
	if err != nil {
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(a.i18n.T("public.errorTitle"), "", a.i18n.Ts("public.errorProcessingRequest")))
	}

	// Prepare the public template.
	out := unsubTpl{
		Subscriber:       s,
		SubUUID:          subUUID,
		publicTpl:        publicTpl{Title: a.i18n.T("public.unsubscribeTitle")},
		AllowBlocklist:   a.cfg.Privacy.AllowBlocklist,
		AllowExport:      a.cfg.Privacy.AllowExport,
		AllowWipe:        a.cfg.Privacy.AllowWipe,
		AllowPreferences: a.cfg.Privacy.AllowPreferences,
	}

	// If the subscriber is blocklisted, throw an error.
	if s.Status == models.SubscriberStatusBlockListed {
		return c.Render(http.StatusOK, tplMessage, makeMsgTpl(a.i18n.T("public.noSubTitle"), "", a.i18n.Ts("public.blocklisted")))
	}

	// Only show preference management if it's enabled in settings.
	if a.cfg.Privacy.AllowPreferences {
		out.ShowManage = showManage

		// Get the subscriber's lists from the DB to render in the template.
		subs, err := a.core.GetSubscriptions(0, subUUID, false)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("public.errorFetchingLists"))
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
func (a *App) SubscriptionPrefs(c echo.Context) error {
	// Read the form.
	var req struct {
		Name      string   `form:"name" json:"name"`
		ListUUIDs []string `form:"l" json:"list_uuids"`
		Blocklist bool     `form:"blocklist" json:"blocklist"`
		Manage    bool     `form:"manage" json:"manage"`
	}
	if err := c.Bind(&req); err != nil {
		return c.Render(http.StatusBadRequest, tplMessage,
			makeMsgTpl(a.i18n.T("public.errorTitle"), "", a.i18n.T("globals.messages.invalidData")))
	}

	// Simple unsubscribe.
	var (
		campUUID  = c.Param("campUUID")
		subUUID   = c.Param("subUUID")
		blocklist = a.cfg.Privacy.AllowBlocklist && req.Blocklist
	)
	if !req.Manage || blocklist {
		if err := a.core.UnsubscribeByCampaign(subUUID, campUUID, blocklist); err != nil {
			return c.Render(http.StatusInternalServerError, tplMessage,
				makeMsgTpl(a.i18n.T("public.errorTitle"), "", a.i18n.T("public.errorProcessingRequest")))
		}

		return c.Render(http.StatusOK, tplMessage,
			makeMsgTpl(a.i18n.T("public.unsubbedTitle"), "", a.i18n.T("public.unsubbedInfo")))
	}

	// Is preference management enabled?
	if !a.cfg.Privacy.AllowPreferences {
		return c.Render(http.StatusBadRequest, tplMessage,
			makeMsgTpl(a.i18n.T("public.errorTitle"), "", a.i18n.T("public.invalidFeature")))
	}

	// Manage preferences.
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || len(req.Name) > 256 {
		return c.Render(http.StatusBadRequest, tplMessage,
			makeMsgTpl(a.i18n.T("public.errorTitle"), "", a.i18n.T("subscribers.invalidName")))
	}

	// Get the subscriber from the DB.
	sub, err := a.core.GetSubscriber(0, subUUID, "")
	if err != nil {
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(a.i18n.T("public.errorTitle"), "", a.i18n.Ts("globals.messages.pFound",
				"name", a.i18n.T("globals.terms.subscriber"))))
	}
	sub.Name = req.Name

	// Update the subscriber properties in the DB.
	if _, err := a.core.UpdateSubscriber(sub.ID, sub); err != nil {
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(a.i18n.T("public.errorTitle"), "", a.i18n.T("public.errorProcessingRequest")))
	}

	// Get the subscriber's lists and whatever is not sent in the request (unchecked),
	// unsubscribe them.
	reqUUIDs := make(map[string]struct{})
	for _, u := range req.ListUUIDs {
		reqUUIDs[u] = struct{}{}
	}

	// Get subscription from teh DB.
	subs, err := a.core.GetSubscriptions(0, subUUID, false)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("public.errorFetchingLists"))
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
	if err := a.core.UnsubscribeLists([]int{sub.ID}, nil, unsubUUIDs); err != nil {
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(a.i18n.T("public.errorTitle"), "", a.i18n.T("public.errorProcessingRequest")))

	}

	return c.Render(http.StatusOK, tplMessage,
		makeMsgTpl(a.i18n.T("globals.messages.done"), "", a.i18n.T("public.prefsSaved")))
}

// OptinPage renders the double opt-in confirmation page that subscribers
// see when they click on the "Confirm subscription" button in double-optin
// notifications.
func (a *App) OptinPage(c echo.Context) error {
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
					makeMsgTpl(a.i18n.T("public.errorTitle"), "", a.i18n.T("globals.messages.invalidUUID")))
			}
		}
	}

	// Get the list of subscription lists where the subscriber hasn't confirmed.
	lists, err := a.core.GetSubscriberLists(0, subUUID, nil, req.ListUUIDs, models.SubscriptionStatusUnconfirmed, "")
	if err != nil {
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(a.i18n.T("public.errorTitle"), "", a.i18n.Ts("public.errorFetchingLists")))
	}

	// There are no lists to confirm.
	if len(lists) == 0 {
		return c.Render(http.StatusOK, tplMessage,
			makeMsgTpl(a.i18n.T("public.noSubTitle"), "", a.i18n.Ts("public.noSubInfo")))
	}

	// Confirm.
	if confirm {
		meta := models.JSON{}
		if a.cfg.Privacy.RecordOptinIP {
			if h := c.Request().Header.Get("X-Forwarded-For"); h != "" {
				meta["optin_ip"] = h
			} else if h := c.Request().RemoteAddr; h != "" {
				meta["optin_ip"] = strings.Split(h, ":")[0]
			}
		}

		// Confirm subscriptions in the DB.
		if err := a.core.ConfirmOptionSubscription(subUUID, req.ListUUIDs, meta); err != nil {
			a.log.Printf("error unsubscribing: %v", err)
			return c.Render(http.StatusInternalServerError, tplMessage,
				makeMsgTpl(a.i18n.T("public.errorTitle"), "", a.i18n.Ts("public.errorProcessingRequest")))
		}

		return c.Render(http.StatusOK, tplMessage,
			makeMsgTpl(a.i18n.T("public.subConfirmedTitle"), "", a.i18n.Ts("public.subConfirmed")))
	}

	var out optinTpl
	out.Lists = lists
	out.SubUUID = subUUID
	out.Title = a.i18n.T("public.confirmOptinSubTitle")

	return c.Render(http.StatusOK, "optin", out)
}

// SubscriptionFormPage handles subscription requests coming from public
// HTML subscription forms.
func (a *App) SubscriptionFormPage(c echo.Context) error {
	if !a.cfg.EnablePublicSubPage {
		return c.Render(http.StatusNotFound, tplMessage,
			makeMsgTpl(a.i18n.T("public.errorTitle"), "", a.i18n.Ts("public.invalidFeature")))
	}

	// Get all public lists from the DB.
	lists, err := a.core.GetLists(models.ListTypePublic, models.ListStatusActive, true, nil)
	if err != nil {
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(a.i18n.T("public.errorTitle"), "", a.i18n.Ts("public.errorFetchingLists")))
	}

	// There are no public lists available for subscription.
	if len(lists) == 0 {
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(a.i18n.T("public.errorTitle"), "", a.i18n.Ts("public.noListsAvailable")))
	}

	out := subFormTpl{}
	out.Title = a.i18n.T("public.sub")
	out.Lists = lists

	// Captcha configuration for template rendering.
	if a.cfg.Security.Captcha.Altcha.Enabled {
		out.Captcha.Enabled = true
		out.Captcha.Provider = "altcha"
		out.Captcha.Complexity = a.cfg.Security.Captcha.Altcha.Complexity
	} else if a.cfg.Security.Captcha.HCaptcha.Enabled {
		out.Captcha.Enabled = true
		out.Captcha.Provider = "hcaptcha"
		out.Captcha.Key = a.cfg.Security.Captcha.HCaptcha.Key
	}

	return c.Render(http.StatusOK, "subscription-form", out)
}

// SubscriptionForm handles subscription requests coming from public
// HTML subscription forms.
func (a *App) SubscriptionForm(c echo.Context) error {
	if !a.cfg.EnablePublicSubPage {
		return echo.NewHTTPError(http.StatusNotFound, a.i18n.T("public.invalidFeature"))

	}

	// If there's a nonce value, a bot could've filled the form.
	if c.FormValue("nonce") != "" {
		return echo.NewHTTPError(http.StatusBadGateway, a.i18n.T("public.invalidFeature"))
	}

	// Process CAPTCHA.
	if a.captcha.IsEnabled() {
		var val string

		// Get the appropriate captcha response field based on provider.
		switch a.captcha.GetProvider() {
		case captcha.ProviderHCaptcha:
			val = c.FormValue("h-captcha-response")
		case captcha.ProviderAltcha:
			val = c.FormValue("altcha")
		default:
			return c.Render(http.StatusBadRequest, tplMessage,
				makeMsgTpl(a.i18n.T("public.errorTitle"), "", a.i18n.T("public.invalidCaptcha")))
		}

		if val == "" {
			return c.Render(http.StatusBadRequest, tplMessage,
				makeMsgTpl(a.i18n.T("public.errorTitle"), "", a.i18n.T("public.invalidCaptcha")))
		}

		err, ok := a.captcha.Verify(val)
		if err != nil {
			a.log.Printf("captcha request failed: %v", err)
		}

		if !ok {
			return c.Render(http.StatusBadRequest, tplMessage,
				makeMsgTpl(a.i18n.T("public.errorTitle"), "", a.i18n.T("public.invalidCaptcha")))
		}
	}

	hasOptin, err := a.processSubForm(c)
	if err != nil {
		e, ok := err.(*echo.HTTPError)
		if !ok {
			return err
		}

		return c.Render(e.Code, tplMessage, makeMsgTpl(a.i18n.T("public.errorTitle"), "", fmt.Sprintf("%s", e.Message)))
	}

	// If there were double optin lists, show the opt-in pending message instead of
	// the subscription confirmation message.
	msg := "public.subConfirmed"
	if hasOptin {
		msg = "public.subOptinPending"
	}

	return c.Render(http.StatusOK, tplMessage, makeMsgTpl(a.i18n.T("public.subTitle"), "", a.i18n.Ts(msg)))
}

// PublicSubscription handles subscription requests coming from public
// API calls.
func (a *App) PublicSubscription(c echo.Context) error {
	if !a.cfg.EnablePublicSubPage {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("public.invalidFeature"))
	}

	hasOptin, err := a.processSubForm(c)
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
func (a *App) LinkRedirect(c echo.Context) error {
	var (
		linkUUID = c.Param("linkUUID")
		campUUID = c.Param("campUUID")
	)

	// If tracking is globally disabled, resolve the URL without recording a click.
	if a.cfg.Privacy.DisableTracking {
		url, err := a.core.GetLinkURL(linkUUID)
		if err != nil {
			e := err.(*echo.HTTPError)
			return c.Render(e.Code, tplMessage, makeMsgTpl(a.i18n.T("public.errorTitle"), "", e.Error()))
		}
		return c.Redirect(http.StatusTemporaryRedirect, url)
	}

	// If individual tracking is disabled, do not record the subscriber ID.
	subUUID := c.Param("subUUID")
	if !a.cfg.Privacy.IndividualTracking {
		subUUID = ""
	}

	url, err := a.core.RegisterCampaignLinkClick(linkUUID, campUUID, subUUID)
	if err != nil {
		e := err.(*echo.HTTPError)
		return c.Render(e.Code, tplMessage, makeMsgTpl(a.i18n.T("public.errorTitle"), "", e.Error()))
	}

	return c.Redirect(http.StatusTemporaryRedirect, url)
}

// RegisterCampaignView registers a campaign view which comes in
// the form of an pixel image request. Regardless of errors, this handler
// should always render the pixel image bytes. The pixel URL is generated by
// the {{ TrackView }} template tag in campaigns.
func (a *App) RegisterCampaignView(c echo.Context) error {
	// If tracking is globally disabled, return the pixel without recording.
	if a.cfg.Privacy.DisableTracking {
		c.Response().Header().Set("Cache-Control", "no-cache")
		return c.Blob(http.StatusOK, "image/png", pixelPNG)
	}

	// If individual tracking is disabled, do not record the subscriber ID.
	subUUID := c.Param("subUUID")
	if !a.cfg.Privacy.IndividualTracking {
		subUUID = ""
	}

	// Exclude dummy hits from template previews.
	campUUID := c.Param("campUUID")
	if campUUID != dummyUUID && subUUID != dummyUUID {
		if err := a.core.RegisterCampaignView(campUUID, subUUID); err != nil {
			a.log.Printf("error registering campaign view: %s", err)
		}
	}

	c.Response().Header().Set("Cache-Control", "no-cache")
	return c.Blob(http.StatusOK, "image/png", pixelPNG)
}

// SelfExportSubscriberData pulls the subscriber's profile, list subscriptions,
// campaign views and clicks and produces a JSON report that is then e-mailed
// to the subscriber. This is a privacy feature and the data that's exported
// is dependent on the configuration.
func (a *App) SelfExportSubscriberData(c echo.Context) error {
	// Is export allowed?
	if !a.cfg.Privacy.AllowExport {
		return c.Render(http.StatusBadRequest, tplMessage,
			makeMsgTpl(a.i18n.T("public.errorTitle"), "", a.i18n.Ts("public.invalidFeature")))
	}

	// Get the subscriber's data. A single query that gets the profile,
	// list subscriptions, campaign views, and link clicks. Names of
	// private lists are replaced with "Private list".
	subUUID := c.Param("subUUID")
	data, b, err := a.exportSubscriberData(0, subUUID, a.cfg.Privacy.Exportable)
	if err != nil {
		a.log.Printf("error exporting subscriber data: %s", err)
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(a.i18n.T("public.errorTitle"), "", a.i18n.Ts("public.errorProcessingRequest")))
	}

	// Prepare the attachment e-mail.
	var msg bytes.Buffer
	if err := notifs.Tpls.ExecuteTemplate(&msg, notifs.TplSubscriberData, data); err != nil {
		a.log.Printf("error compiling notification template '%s': %v", notifs.TplSubscriberData, err)
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(a.i18n.T("public.errorTitle"), "", a.i18n.Ts("public.errorProcessingRequest")))
	}

	// TODO: GetTplSubject should be moved to a utils package.
	subject, body := notifs.GetTplSubject(a.i18n.Ts("email.data.title"), msg.Bytes())

	// E-mail the data as a JSON attachment to the subscriber.
	const fname = "data.json"
	if err := a.emailMsgr.Push(models.Message{
		From:    a.cfg.FromEmail,
		To:      []string{data.Email},
		Subject: subject,
		Body:    body,
		Attachments: []models.Attachment{
			{
				Name:    fname,
				Content: b,
				Header:  manager.MakeAttachmentHeader(fname, "base64", "application/json"),
			},
		},
	}); err != nil {
		a.log.Printf("error e-mailing subscriber profile: %s", err)
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(a.i18n.T("public.errorTitle"), "", a.i18n.Ts("public.errorProcessingRequest")))
	}

	return c.Render(http.StatusOK, tplMessage,
		makeMsgTpl(a.i18n.T("public.dataSentTitle"), "", a.i18n.T("public.dataSent")))
}

// WipeSubscriberData allows a subscriber to delete their data. The
// profile and subscriptions are deleted, while the campaign_views and link
// clicks remain as orphan data unconnected to any subscriber.
func (a *App) WipeSubscriberData(c echo.Context) error {
	// Is wiping allowed?
	if !a.cfg.Privacy.AllowWipe {
		return c.Render(http.StatusBadRequest, tplMessage,
			makeMsgTpl(a.i18n.T("public.errorTitle"), "", a.i18n.Ts("public.invalidFeature")))
	}

	subUUID := c.Param("subUUID")
	if err := a.core.DeleteSubscribers(nil, []string{subUUID}); err != nil {
		a.log.Printf("error wiping subscriber data: %s", err)
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(a.i18n.T("public.errorTitle"), "", a.i18n.Ts("public.errorProcessingRequest")))
	}

	return c.Render(http.StatusOK, tplMessage,
		makeMsgTpl(a.i18n.T("public.dataRemovedTitle"), "", a.i18n.T("public.dataRemoved")))
}

// AltchaChallenge generates a challenge for Altcha captcha.
func (a *App) AltchaChallenge(c echo.Context) error {
	// Check if Altcha is enabled.
	if !a.captcha.IsEnabled() || a.captcha.GetProvider() != captcha.ProviderAltcha {
		return echo.NewHTTPError(http.StatusNotFound, "captcha not enabled")
	}

	// Generate challenge.
	out, err := a.captcha.GenerateChallenge()
	if err != nil {
		a.log.Printf("error generating altcha challenge: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Error generating challenge")
	}

	// Return the challenge as JSON.
	c.Response().Header().Set("Content-Type", "application/json")
	return c.String(http.StatusOK, out)
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
func (a *App) processSubForm(c echo.Context) (bool, error) {
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
		return false, echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("public.noListsSelected"))
	}

	// Validate fields.
	if len(req.Email) > 1000 {
		return false, echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("subscribers.invalidEmail"))
	}

	em, err := a.importer.SanitizeEmail(req.Email)
	if err != nil {
		return false, echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	req.Email = em

	req.Name = strings.TrimSpace(req.Name)
	if len(req.Name) == 0 {
		// If there's no name, use the name bit from the e-mail.
		req.Name = strings.Split(req.Email, "@")[0]
	} else if len(req.Name) > stdInputMaxLen {
		return false, echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("subscribers.invalidName"))
	}

	listUUIDs := pq.StringArray(req.FormListUUIDs)

	// Fetch the list types and ensure that they are not private.
	listTypes, err := a.core.GetListTypes(nil, req.FormListUUIDs)
	if err != nil {
		return false, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("%s", err.(*echo.HTTPError).Message))
	}

	for _, t := range listTypes {
		if t == models.ListTypePrivate {
			return false, echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidUUID"))
		}
	}

	// Insert the subscriber into the DB.
	_, hasOptin, err := a.core.InsertSubscriber(models.Subscriber{
		Name:   req.Name,
		Email:  req.Email,
		Status: models.SubscriberStatusEnabled,
	}, nil, listUUIDs, false, true)
	if err == nil {
		return hasOptin, nil
	}

	// Insert returned an error. Examine it.
	var lastErr = err

	// Subscriber already exists. Update subscriptions in the DB.
	if e, ok := err.(*echo.HTTPError); ok && e.Code == http.StatusConflict {
		// Get the subscriber from the DB by their email.
		sub, err := a.core.GetSubscriber(0, "", req.Email)
		if err != nil {
			return false, err
		}

		// Update the subscriber's subscriptions in the DB.
		_, hasOptin, err := a.core.UpdateSubscriberWithLists(sub.ID, sub, nil, listUUIDs, false, false, true)
		if err == nil {
			return hasOptin, nil
		}
		lastErr = err
	}

	// Something else went wrong.
	if e, ok := lastErr.(*echo.HTTPError); ok {
		return false, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("%s", e.Message))
	}
	return false, echo.NewHTTPError(http.StatusInternalServerError, a.i18n.T("public.errorProcessingRequest"))
}
