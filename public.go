package main

import (
	"bytes"
	"html/template"
	"image"
	"image/png"
	"io"
	"net/http"
	"regexp"
	"strconv"

	"github.com/labstack/echo"
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
	Unsubscribe bool
	Blacklist   bool
}

type msgTpl struct {
	publicTpl
	MessageTitle string
	Message      string
}

var (
	regexValidUUID = regexp.MustCompile("^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$")
	pixelPNG       = drawTransparentImage(3, 14)
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

// handleUnsubscribePage unsubscribes a subscriber and renders a view.
func handleUnsubscribePage(c echo.Context) error {
	var (
		app          = c.Get("app").(*App)
		campUUID     = c.Param("campUUID")
		subUUID      = c.Param("subUUID")
		unsub, _     = strconv.ParseBool(c.FormValue("unsubscribe"))
		blacklist, _ = strconv.ParseBool(c.FormValue("blacklist"))

		out = unsubTpl{}
	)
	out.Unsubscribe = unsub
	out.Blacklist = blacklist
	out.Title = "Unsubscribe from mailing list"

	if !regexValidUUID.MatchString(campUUID) ||
		!regexValidUUID.MatchString(subUUID) {
		return c.Render(http.StatusBadRequest, "message",
			makeMsgTpl("Invalid request", "",
				`The unsubscription request contains invalid IDs.
				Please follow the correct link.`))
	}

	// Unsubscribe.
	if unsub {
		res, err := app.Queries.Unsubscribe.Exec(campUUID, subUUID, blacklist)
		if err != nil {
			app.Logger.Printf("Error unsubscribing : %v", err)
			return echo.NewHTTPError(http.StatusBadRequest,
				"There was an internal error while unsubscribing you.")
		}

		if !blacklist {
			num, _ := res.RowsAffected()
			if num == 0 {
				return c.Render(http.StatusBadRequest, "message",
					makeMsgTpl("Already unsubscribed", "",
						`You are not subscribed to this mailing list.
						You may have already unsubscribed.`))
			}
		}
	}

	return c.Render(http.StatusOK, "unsubscribe", out)
}

// handleLinkRedirect handles link UUID to real link redirection.
func handleLinkRedirect(c echo.Context) error {
	var (
		app      = c.Get("app").(*App)
		linkUUID = c.Param("linkUUID")
		campUUID = c.Param("campUUID")
		subUUID  = c.Param("subUUID")
	)
	if !regexValidUUID.MatchString(linkUUID) ||
		!regexValidUUID.MatchString(campUUID) ||
		!regexValidUUID.MatchString(subUUID) {
		return c.Render(http.StatusBadRequest, "message",
			makeMsgTpl("Invalid link", "", "The link you clicked is invalid."))
	}

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
	if regexValidUUID.MatchString(campUUID) &&
		regexValidUUID.MatchString(subUUID) {
		if _, err := app.Queries.RegisterCampaignView.Exec(campUUID, subUUID); err != nil {
			app.Logger.Printf("error registering campaign view: %s", err)
		}
	}

	c.Response().Header().Set("Cache-Control", "no-cache")
	return c.Blob(http.StatusOK, "image/png", pixelPNG)
}

// drawTransparentImage draws a transparent PNG of given dimensions
// and returns the PNG bytes.
func drawTransparentImage(h, w int) []byte {
	var (
		img = image.NewRGBA(image.Rect(0, 0, w, h))
		out = &bytes.Buffer{}
	)
	png.Encode(out, img)
	return out.Bytes()
}
