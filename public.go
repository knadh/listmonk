package main

import (
	"bytes"
	"database/sql"
	"html/template"
	"image"
	"image/png"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/knadh/listmonk/internal/messenger"
	"github.com/knadh/listmonk/internal/subimporter"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo"
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
}

type publicTpl struct {
	Title       string
	Description string
}

type unsubTpl struct {
	publicTpl
	SubUUID        string
	AllowBlacklist bool
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
	})
}

// handleSubscriptionPage renders the subscription management page and
// handles unsubscriptions.
func handleSubscriptionPage(c echo.Context) error {
	var (
		app          = c.Get("app").(*App)
		campUUID     = c.Param("campUUID")
		subUUID      = c.Param("subUUID")
		unsub, _     = strconv.ParseBool(c.FormValue("unsubscribe"))
		blacklist, _ = strconv.ParseBool(c.FormValue("blacklist"))
		out          = unsubTpl{}
	)
	out.SubUUID = subUUID
	out.Title = "Unsubscribe from mailing list"
	out.AllowBlacklist = app.constants.Privacy.AllowBlacklist
	out.AllowExport = app.constants.Privacy.AllowExport
	out.AllowWipe = app.constants.Privacy.AllowWipe

	// Unsubscribe.
	if unsub {
		// Is blacklisting allowed?
		if !app.constants.Privacy.AllowBlacklist {
			blacklist = false
		}

		if _, err := app.queries.Unsubscribe.Exec(campUUID, subUUID, blacklist); err != nil {
			app.log.Printf("error unsubscribing: %v", err)
			return c.Render(http.StatusInternalServerError, tplMessage,
				makeMsgTpl("Error", "",
					`Error processing request. Please retry.`))
		}

		return c.Render(http.StatusOK, tplMessage,
			makeMsgTpl("Unsubscribed", "",
				`You have been successfully unsubscribed.`))
	}

	return c.Render(http.StatusOK, "subscription", out)
}

// handleOptinPage handles a double opt-in confirmation from subscribers.
func handleOptinPage(c echo.Context) error {
	var (
		app        = c.Get("app").(*App)
		subUUID    = c.Param("subUUID")
		confirm, _ = strconv.ParseBool(c.FormValue("confirm"))
		out        = optinTpl{}
	)
	out.SubUUID = subUUID
	out.Title = "Confirm subscriptions"
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
					makeMsgTpl("Invalid request", "",
						`One or more UUIDs in the request are invalid.`))
			}
		}
	}

	// Get the list of subscription lists where the subscriber hasn't confirmed.
	if err := app.queries.GetSubscriberLists.Select(&out.Lists, 0, subUUID,
		nil, pq.StringArray(out.ListUUIDs), models.SubscriptionStatusUnconfirmed, nil); err != nil {
		app.log.Printf("error fetching lists for opt-in: %s", pqErrMsg(err))
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl("Error", "", `Error fetching lists. Please retry.`))
	}

	// There are no lists to confirm.
	if len(out.Lists) == 0 {
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl("No subscriptions", "",
				`There are no subscriptions to confirm.`))
	}

	// Confirm.
	if confirm {
		if _, err := app.queries.ConfirmSubscriptionOptin.Exec(subUUID, pq.StringArray(out.ListUUIDs)); err != nil {
			app.log.Printf("error unsubscribing: %v", err)
			return c.Render(http.StatusInternalServerError, tplMessage,
				makeMsgTpl("Error", "",
					`Error processing request. Please retry.`))
		}
		return c.Render(http.StatusOK, tplMessage,
			makeMsgTpl("Confirmed", "",
				`Your subscriptions have been confirmed.`))
	}

	return c.Render(http.StatusOK, "optin", out)
}

// handleOptinPage handles a double opt-in confirmation from subscribers.
func handleSubscriptionForm(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		req subForm
	)

	// Get and validate fields.
	if err := c.Bind(&req); err != nil {
		return err
	}

	if len(req.SubListUUIDs) == 0 {
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl("Error", "",
				`No lists to subscribe to.`))
	}

	// If there's no name, use the name bit from the e-mail.
	req.Email = strings.ToLower(req.Email)
	if req.Name == "" {
		req.Name = strings.Split(req.Email, "@")[0]
	}

	// Validate fields.
	if err := subimporter.ValidateFields(req.SubReq); err != nil {
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl("Error", "", err.Error()))
	}

	// Insert the subscriber into the DB.
	req.Status = models.SubscriberStatusEnabled
	req.ListUUIDs = pq.StringArray(req.SubListUUIDs)
	if _, err := insertSubscriber(req.SubReq, app); err != nil {
		return err
	}

	return c.Render(http.StatusInternalServerError, tplMessage,
		makeMsgTpl("Done", "", `Subscribed successfully.`))
}

// handleLinkRedirect handles link UUID to real link redirection.
func handleLinkRedirect(c echo.Context) error {
	var (
		app      = c.Get("app").(*App)
		linkUUID = c.Param("linkUUID")
		campUUID = c.Param("campUUID")
		subUUID  = c.Param("subUUID")
	)

	var url string
	if err := app.queries.RegisterLinkClick.Get(&url, linkUUID, campUUID, subUUID); err != nil {
		if err != sql.ErrNoRows {
			app.log.Printf("error fetching redirect link: %s", err)
		}

		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl("Error opening link", "",
				"There was an error opening the link. Please try later."))
	}

	return c.Redirect(http.StatusTemporaryRedirect, url)
}

// handleRegisterCampaignView registers a campaign view which comes in
// the form of an pixel image request. Regardless of errors, this handler
// should always render the pixel image bytes.
func handleRegisterCampaignView(c echo.Context) error {
	var (
		app      = c.Get("app").(*App)
		campUUID = c.Param("campUUID")
		subUUID  = c.Param("subUUID")
	)

	// Exclude dummy hits from template previews.
	if campUUID != dummyUUID && subUUID != dummyUUID {
		if _, err := app.queries.RegisterCampaignView.Exec(campUUID, subUUID); err != nil {
			app.log.Printf("error registering campaign view: %s", err)
		}
	}

	c.Response().Header().Set("Cache-Control", "no-cache")
	return c.Blob(http.StatusOK, "image/png", pixelPNG)
}

// handleSelfExportSubscriberData pulls the subscriber's profile,
// list subscriptions, campaign views and clicks and produces
// a JSON report. This is a privacy feature and depends on the
// configuration in app.Constants.Privacy.
func handleSelfExportSubscriberData(c echo.Context) error {
	var (
		app     = c.Get("app").(*App)
		subUUID = c.Param("subUUID")
	)
	// Is export allowed?
	if !app.constants.Privacy.AllowExport {
		return c.Render(http.StatusBadRequest, tplMessage,
			makeMsgTpl("Invalid request", "", "The feature is not available."))
	}

	// Get the subscriber's data. A single query that gets the profile,
	// list subscriptions, campaign views, and link clicks. Names of
	// private lists are replaced with "Private list".
	data, b, err := exportSubscriberData(0, subUUID, app.constants.Privacy.Exportable, app)
	if err != nil {
		app.log.Printf("error exporting subscriber data: %s", err)
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl("Error processing request", "",
				"There was an error processing your request. Please try later."))
	}

	// Send the data out to the subscriber as an atachment.
	var msg bytes.Buffer
	if err := app.notifTpls.ExecuteTemplate(&msg, notifSubscriberData, data); err != nil {
		app.log.Printf("error compiling notification template '%s': %v",
			notifSubscriberData, err)
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl("Error preparing data", "",
				"There was an error preparing your data. Please try later."))
	}

	const fname = "profile.json"
	if err := app.messenger.Push(app.constants.FromEmail,
		[]string{data.Email},
		"Your profile data",
		msg.Bytes(),
		[]*messenger.Attachment{
			&messenger.Attachment{
				Name:    fname,
				Content: b,
				Header:  messenger.MakeAttachmentHeader(fname, "base64"),
			},
		},
	); err != nil {
		app.log.Printf("error e-mailing subscriber profile: %s", err)
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl("Error e-mailing data", "",
				"There was an error e-mailing your data. Please try later."))
	}
	return c.Render(http.StatusOK, tplMessage,
		makeMsgTpl("Data e-mailed", "",
			`Your data has been e-mailed to you as an attachment.`))
}

// handleWipeSubscriberData allows a subscriber to self-delete their data. The
// profile and subscriptions are deleted, while the campaign_views and link
// clicks remain as orphan data unconnected to any subscriber.
func handleWipeSubscriberData(c echo.Context) error {
	var (
		app     = c.Get("app").(*App)
		subUUID = c.Param("subUUID")
	)

	// Is wiping allowed?
	if !app.constants.Privacy.AllowExport {
		return c.Render(http.StatusBadRequest, tplMessage,
			makeMsgTpl("Invalid request", "",
				"The feature is not available."))
	}

	if _, err := app.queries.DeleteSubscribers.Exec(nil, pq.StringArray{subUUID}); err != nil {
		app.log.Printf("error wiping subscriber data: %s", err)
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl("Error processing request", "",
				"There was an error processing your request. Please try later."))
	}

	return c.Render(http.StatusOK, tplMessage,
		makeMsgTpl("Data removed", "",
			`Your subscriptions and all associated data has been removed.`))
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
