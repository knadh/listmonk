package main

import (
	"html/template"
	"io"
	"net/http"
	"regexp"
	"strconv"

	"github.com/labstack/echo"
)

type Template struct {
	templates *template.Template
}

type publicTpl struct {
	Title       string
	Description string
}

type unsubTpl struct {
	publicTpl
	Blacklisted bool
}

type errorTpl struct {
	publicTpl

	ErrorTitle   string
	ErrorMessage string
}

var regexValidUUID = regexp.MustCompile("^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$")

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	// fmt.Println(t.templates.ExecuteTemplate(os.Stdout, name, nil))
	return t.templates.ExecuteTemplate(w, name, data)
}

// handleUnsubscribePage unsubscribes a subscriber and renders a view.
func handleUnsubscribePage(c echo.Context) error {
	var (
		app          = c.Get("app").(*App)
		campUUID     = c.Param("campUUID")
		subUUID      = c.Param("subUUID")
		blacklist, _ = strconv.ParseBool(c.FormValue("blacklist"))

		out = unsubTpl{}
	)
	out.Blacklisted = blacklist
	out.Title = "Unsubscribe from mailing list"

	if !regexValidUUID.MatchString(campUUID) ||
		!regexValidUUID.MatchString(subUUID) {
		err := errorTpl{}
		err.Title = "Invalid request"
		err.ErrorTitle = err.Title
		err.ErrorMessage = "The unsubscription request contains invalid IDs. Please make sure to follow the correct link."
		return c.Render(http.StatusBadRequest, "error", err)
	}

	// Unsubscribe.
	res, err := app.Queries.Unsubscribe.Exec(campUUID, subUUID, blacklist)
	if err != nil {
		app.Logger.Printf("Error unsubscribing : %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "Subscription doesn't exist")
	}

	num, err := res.RowsAffected()
	if num == 0 {
		err := errorTpl{}
		err.Title = "Invalid subscription"
		err.ErrorTitle = err.Title
		err.ErrorMessage = "Looks like you are not subscribed to this mailing list."
		return c.Render(http.StatusBadRequest, "error", err)
	}

	return c.Render(http.StatusOK, "unsubscribe", out)
}
