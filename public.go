package main

import (
	"bytes"
	"html/template"
	"image"
	"image/png"
	"io"
	"net/http"
	"strconv"

	"github.com/knadh/listmonk/messenger"
	"github.com/labstack/echo"
	"github.com/lib/pq"
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

type msgTpl struct {
	publicTpl
	MessageTitle string
	Message      string
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
	out.AllowBlacklist = app.Constants.Privacy.AllowBlacklist
	out.AllowExport = app.Constants.Privacy.AllowExport
	out.AllowWipe = app.Constants.Privacy.AllowWipe

	// Unsubscribe.
	if unsub {
		// Is blacklisting allowed?
		if !app.Constants.Privacy.AllowBlacklist {
			blacklist = false
		}

		if _, err := app.Queries.Unsubscribe.Exec(campUUID, subUUID, blacklist); err != nil {
			app.Logger.Printf("error unsubscribing: %v", err)
			return c.Render(http.StatusInternalServerError, "message",
				makeMsgTpl("Error", "",
					`Error processing request. Please retry.`))
		}
		return c.Render(http.StatusOK, "message",
			makeMsgTpl("Unsubscribed", "",
				`You have been successfully unsubscribed.`))
	}

	return c.Render(http.StatusOK, "subscription", out)
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
	if err := app.Queries.RegisterLinkClick.Get(&url, linkUUID, campUUID, subUUID); err != nil {
		app.Logger.Printf("error fetching redirect link: %s", err)
		return c.Render(http.StatusInternalServerError, "message",
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
	if _, err := app.Queries.RegisterCampaignView.Exec(campUUID, subUUID); err != nil {
		app.Logger.Printf("error registering campaign view: %s", err)
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
	if !app.Constants.Privacy.AllowExport {
		return c.Render(http.StatusBadRequest, "message",
			makeMsgTpl("Invalid request", "",
				"The feature is not available."))
	}

	// Get the subscriber's data. A single query that gets the profile,
	// list subscriptions, campaign views, and link clicks. Names of
	// private lists are replaced with "Private list".
	data, b, err := exportSubscriberData(0, subUUID, app.Constants.Privacy.Exportable, app)
	if err != nil {
		app.Logger.Printf("error exporting subscriber data: %s", err)
		return c.Render(http.StatusInternalServerError, "message",
			makeMsgTpl("Error processing request", "",
				"There was an error processing your request. Please try later."))
	}

	// Send the data out to the subscriber as an atachment.
	msg, err := getNotificationTemplate("subscriber-data", nil, app)
	if err != nil {
		app.Logger.Printf("error preparing subscriber data e-mail template: %s", err)
		return c.Render(http.StatusInternalServerError, "message",
			makeMsgTpl("Error preparing data", "",
				"There was an error preparing your data. Please try later."))
	}

	const fname = "profile.json"
	if err := app.Messenger.Push(app.Constants.FromEmail,
		[]string{data.Email},
		"Your profile data",
		msg,
		[]*messenger.Attachment{
			&messenger.Attachment{
				Name:    fname,
				Content: b,
				Header:  messenger.MakeAttachmentHeader(fname, "base64"),
			},
		},
	); err != nil {
		app.Logger.Printf("error e-mailing subscriber profile: %s", err)
		return c.Render(http.StatusInternalServerError, "message",
			makeMsgTpl("Error e-mailing data", "",
				"There was an error e-mailing your data. Please try later."))
	}
	return c.Render(http.StatusOK, "message",
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
	if !app.Constants.Privacy.AllowExport {
		return c.Render(http.StatusBadRequest, "message",
			makeMsgTpl("Invalid request", "",
				"The feature is not available."))
	}

	if _, err := app.Queries.DeleteSubscribers.Exec(nil, pq.StringArray{subUUID}); err != nil {
		app.Logger.Printf("error wiping subscriber data: %s", err)
		return c.Render(http.StatusInternalServerError, "message",
			makeMsgTpl("Error processing request", "",
				"There was an error processing your request. Please try later."))
	}

	return c.Render(http.StatusOK, "message",
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
