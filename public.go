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

// Template wraps a template.Template for echo.
type Template struct {
	templates *template.Template
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

type errorTpl struct {
	publicTpl

	ErrorTitle   string
	ErrorMessage string
}

var (
	regexValidUUID = regexp.MustCompile("^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$")
	pixelPNG       = drawTransparentImage(3, 14)
)

// Render executes and renders a template for echo.
func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
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
		return c.Render(http.StatusBadRequest, "error",
			makeErrorTpl("Invalid request", "",
				`The unsubscription request contains invalid IDs.
				Please click on the correct link.`))
	}

	// Unsubscribe.
	if unsub {
		res, err := app.Queries.Unsubscribe.Exec(campUUID, subUUID, blacklist)
		if err != nil {
			app.Logger.Printf("Error unsubscribing : %v", err)
			return echo.NewHTTPError(http.StatusBadRequest, "There was an internal error while unsubscribing you.")
		}

		if !blacklist {
			num, _ := res.RowsAffected()
			if num == 0 {
				return c.Render(http.StatusBadRequest, "error",
					makeErrorTpl("Already unsubscribed", "",
						`Looks like you are not subscribed to this mailing list.
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
		return c.Render(http.StatusBadRequest, "error",
			makeErrorTpl("Invalid link", "", "The link you clicked is invalid."))
	}

	var url string
	if err := app.Queries.RegisterLinkClick.Get(&url, linkUUID, campUUID, subUUID); err != nil {
		app.Logger.Printf("error fetching redirect link: %s", err)
		return c.Render(http.StatusInternalServerError, "error",
			makeErrorTpl("Error opening link", "", "There was an error opening the link. Please try later."))
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
