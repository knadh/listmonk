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
	"github.com/knadh/listmonk/internal/messenger"
	"github.com/knadh/listmonk/internal/subimporter"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
)

const (
	tplMessage = "message"
)

// tplRenderer wraps a template.tplRenderer for echo.
type tplRenderer struct {
	templates  *template.Template
	RootURL    string
	LogoURL    string
	FaviconURL string
}

// tplData is the data container that is injected
// into public templates for accessing data.
type tplData struct {
	RootURL    string
	LogoURL    string
	FaviconURL string
	Data       interface{}
	L          *i18n.I18n
}

type publicTpl struct {
	Title       string
	Description string
}

type unsubTpl struct {
	publicTpl
	SubUUID        string
	AllowBlocklist bool
	AllowExport    bool
	AllowWipe      bool
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
	Lists []models.List
}

type subForm struct {
	subimporter.SubReq
	SubListUUIDs []string `form:"l"`
}

var (
	pixelPNG = drawTransparentImage(3, 14)
)

// Render executes and renders a template for echo.
func (t *tplRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, tplData{
		RootURL:    t.RootURL,
		LogoURL:    t.LogoURL,
		FaviconURL: t.FaviconURL,
		Data:       data,
		L:          c.Get("app").(*App).i18n,
	})
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
	var camp models.Campaign
	if err := app.queries.GetCampaign.Get(&camp, 0, campUUID); err != nil {
		if err == sql.ErrNoRows {
			return c.Render(http.StatusNotFound, tplMessage,
				makeMsgTpl(app.i18n.T("public.notFoundTitle"), "",
					app.i18n.T("public.campaignNotFound")))
		}

		app.log.Printf("error fetching campaign: %v", err)
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(app.i18n.T("public.errorTitle"), "",
				app.i18n.Ts("public.errorFetchingCampaign")))
	}

	// Get the subscriber.
	sub, err := getSubscriber(0, subUUID, "", app)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Render(http.StatusNotFound, tplMessage,
				makeMsgTpl(app.i18n.T("public.notFoundTitle"), "",
					app.i18n.T("public.errorFetchingEmail")))
		}

		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(app.i18n.T("public.errorTitle"), "",
				app.i18n.Ts("public.errorFetchingCampaign")))
	}

	// Compile the template.
	if err := camp.CompileTemplate(app.manager.TemplateFuncs(&camp)); err != nil {
		app.log.Printf("error compiling template: %v", err)
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(app.i18n.T("public.errorTitle"), "",
				app.i18n.Ts("public.errorFetchingCampaign")))
	}

	// Render the message body.
	msg, err := app.manager.NewCampaignMessage(&camp, sub)
	if err != nil {
		app.log.Printf("error rendering message: %v", err)
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(app.i18n.T("public.errorTitle"), "",
				app.i18n.Ts("public.errorFetchingCampaign")))
	}

	return c.HTML(http.StatusOK, string(msg.Body()))
}

// handleSubscriptionPage renders the subscription management page and
// handles unsubscriptions. This is the view that {{ UnsubscribeURL }} in
// campaigns link to.
func handleSubscriptionPage(c echo.Context) error {
	var (
		app          = c.Get("app").(*App)
		campUUID     = c.Param("campUUID")
		subUUID      = c.Param("subUUID")
		unsub        = c.Request().Method == http.MethodPost
		blocklist, _ = strconv.ParseBool(c.FormValue("blocklist"))
		out          = unsubTpl{}
	)
	out.SubUUID = subUUID
	out.Title = app.i18n.T("public.unsubscribeTitle")
	out.AllowBlocklist = app.constants.Privacy.AllowBlocklist
	out.AllowExport = app.constants.Privacy.AllowExport
	out.AllowWipe = app.constants.Privacy.AllowWipe

	// Unsubscribe.
	if unsub {
		// Is blocklisting allowed?
		if !app.constants.Privacy.AllowBlocklist {
			blocklist = false
		}

		if _, err := app.queries.Unsubscribe.Exec(campUUID, subUUID, blocklist); err != nil {
			app.log.Printf("error unsubscribing: %v", err)
			return c.Render(http.StatusInternalServerError, tplMessage,
				makeMsgTpl(app.i18n.T("public.errorTitle"), "",
					app.i18n.Ts("public.errorProcessingRequest")))
		}

		return c.Render(http.StatusOK, tplMessage,
			makeMsgTpl(app.i18n.T("public.unsubbedTitle"), "",
				app.i18n.T("public.unsubbedInfo")))
	}

	return c.Render(http.StatusOK, "subscription", out)
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
					makeMsgTpl(app.i18n.T("public.errorTitle"), "",
						app.i18n.T("globals.messages.invalidUUID")))
			}
		}
	}

	// Get the list of subscription lists where the subscriber hasn't confirmed.
	if err := app.queries.GetSubscriberLists.Select(&out.Lists, 0, subUUID,
		nil, pq.StringArray(out.ListUUIDs), models.SubscriptionStatusUnconfirmed, nil); err != nil {
		app.log.Printf("error fetching lists for opt-in: %s", pqErrMsg(err))

		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(app.i18n.T("public.errorTitle"), "",
				app.i18n.Ts("public.errorFetchingLists")))
	}

	// There are no lists to confirm.
	if len(out.Lists) == 0 {
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(app.i18n.T("public.noSubTitle"), "",
				app.i18n.Ts("public.noSubInfo")))
	}

	// Confirm.
	if confirm {
		if _, err := app.queries.ConfirmSubscriptionOptin.Exec(subUUID, pq.StringArray(out.ListUUIDs)); err != nil {
			app.log.Printf("error unsubscribing: %v", err)
			return c.Render(http.StatusInternalServerError, tplMessage,
				makeMsgTpl(app.i18n.T("public.errorTitle"), "",
					app.i18n.Ts("public.errorProcessingRequest")))
		}

		return c.Render(http.StatusOK, tplMessage,
			makeMsgTpl(app.i18n.T("public.subConfirmedTitle"), "",
				app.i18n.Ts("public.subConfirmed")))
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
			makeMsgTpl(app.i18n.T("public.errorTitle"), "",
				app.i18n.Ts("public.invalidFeature")))
	}

	// Get all public lists.
	var lists []models.List
	if err := app.queries.GetLists.Select(&lists, models.ListTypePublic, "name"); err != nil {
		app.log.Printf("error fetching public lists for form: %s", pqErrMsg(err))
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(app.i18n.T("public.errorTitle"), "",
				app.i18n.Ts("public.errorFetchingLists")))
	}

	if len(lists) == 0 {
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(app.i18n.T("public.errorTitle"), "",
				app.i18n.Ts("public.noListsAvailable")))
	}

	out := subFormTpl{}
	out.Title = app.i18n.T("public.sub")
	out.Lists = lists

	return c.Render(http.StatusOK, "subscription-form", out)
}

// handleSubscriptionForm handles subscription requests coming from public
// HTML subscription forms.
func handleSubscriptionForm(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		req subForm
	)

	// Get and validate fields.
	if err := c.Bind(&req); err != nil {
		return err
	}

	// If there's a nonce value, a bot could've filled the form.
	if c.FormValue("nonce") != "" {
		return c.Render(http.StatusOK, tplMessage,
			makeMsgTpl(app.i18n.T("public.errorTitle"), "",
				app.i18n.T("public.invalidFeature")))

	}

	if len(req.SubListUUIDs) == 0 {
		return c.Render(http.StatusBadRequest, tplMessage,
			makeMsgTpl(app.i18n.T("public.errorTitle"), "",
				app.i18n.T("public.noListsSelected")))
	}

	// If there's no name, use the name bit from the e-mail.
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		req.Name = strings.Split(req.Email, "@")[0]
	}

	// Validate fields.
	if r, err := app.importer.ValidateFields(req.SubReq); err != nil {
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(app.i18n.T("public.errorTitle"), "", err.Error()))
	} else {
		req.SubReq = r
	}

	// Insert the subscriber into the DB.
	req.Status = models.SubscriberStatusEnabled
	req.ListUUIDs = pq.StringArray(req.SubListUUIDs)
	_, _, hasOptin, err := insertSubscriber(req.SubReq, app)
	if err != nil {
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(app.i18n.T("public.errorTitle"), "", fmt.Sprintf("%s", err.(*echo.HTTPError).Message)))
	}

	msg := "public.subConfirmed"
	if hasOptin {
		msg = "public.subOptinPending"
	}

	return c.Render(http.StatusOK, tplMessage, makeMsgTpl(app.i18n.T("public.subTitle"), "", app.i18n.Ts(msg)))
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

	var url string
	if err := app.queries.RegisterLinkClick.Get(&url, linkUUID, campUUID, subUUID); err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Column == "link_id" {
			return c.Render(http.StatusNotFound, tplMessage,
				makeMsgTpl(app.i18n.T("public.errorTitle"), "",
					app.i18n.Ts("public.invalidLink")))
		}

		app.log.Printf("error fetching redirect link: %s", err)
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(app.i18n.T("public.errorTitle"), "",
				app.i18n.Ts("public.errorProcessingRequest")))
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
		if _, err := app.queries.RegisterCampaignView.Exec(campUUID, subUUID); err != nil {
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
			makeMsgTpl(app.i18n.T("public.errorTitle"), "",
				app.i18n.Ts("public.invalidFeature")))
	}

	// Get the subscriber's data. A single query that gets the profile,
	// list subscriptions, campaign views, and link clicks. Names of
	// private lists are replaced with "Private list".
	data, b, err := exportSubscriberData(0, subUUID, app.constants.Privacy.Exportable, app)
	if err != nil {
		app.log.Printf("error exporting subscriber data: %s", err)
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(app.i18n.T("public.errorTitle"), "",
				app.i18n.Ts("public.errorProcessingRequest")))
	}

	// Prepare the attachment e-mail.
	var msg bytes.Buffer
	if err := app.notifTpls.tpls.ExecuteTemplate(&msg, notifSubscriberData, data); err != nil {
		app.log.Printf("error compiling notification template '%s': %v", notifSubscriberData, err)
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(app.i18n.T("public.errorTitle"), "",
				app.i18n.Ts("public.errorProcessingRequest")))
	}

	// Send the data as a JSON attachment to the subscriber.
	const fname = "data.json"
	if err := app.messengers[emailMsgr].Push(messenger.Message{
		ContentType: app.notifTpls.contentType,
		From:        app.constants.FromEmail,
		To:          []string{data.Email},
		Subject:     "Your data",
		Body:        msg.Bytes(),
		Attachments: []messenger.Attachment{
			{
				Name:    fname,
				Content: b,
				Header:  messenger.MakeAttachmentHeader(fname, "base64"),
			},
		},
	}); err != nil {
		app.log.Printf("error e-mailing subscriber profile: %s", err)
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(app.i18n.T("public.errorTitle"), "",
				app.i18n.Ts("public.errorProcessingRequest")))
	}

	return c.Render(http.StatusOK, tplMessage,
		makeMsgTpl(app.i18n.T("public.dataSentTitle"), "",
			app.i18n.T("public.dataSent")))
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
			makeMsgTpl(app.i18n.T("public.errorTitle"), "",
				app.i18n.Ts("public.invalidFeature")))
	}

	if _, err := app.queries.DeleteSubscribers.Exec(nil, pq.StringArray{subUUID}); err != nil {
		app.log.Printf("error wiping subscriber data: %s", err)
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(app.i18n.T("public.errorTitle"), "",
				app.i18n.Ts("public.errorProcessingRequest")))
	}

	return c.Render(http.StatusOK, tplMessage,
		makeMsgTpl(app.i18n.T("public.dataRemovedTitle"), "",
			app.i18n.T("public.dataRemoved")))
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
