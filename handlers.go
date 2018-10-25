package main

import (
	"encoding/json"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/asaskevich/govalidator"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
)

const (
	// stdInputMaxLen is the maximum allowed length for a standard input field.
	stdInputMaxLen = 200

	// bodyMaxLen is the maximum allowed length for e-mail bodies.
	bodyMaxLen = 1000000

	// defaultPerPage is the default number of results returned in an GET call.
	defaultPerPage = 20

	// maxPerPage is the maximum number of allowed for paginated records.
	maxPerPage = 100
)

type okResp struct {
	Data interface{} `json:"data"`
}

// pagination represents a query's pagination (limit, offset) related values.
type pagination struct {
	PerPage int `json:"per_page"`
	Page    int `json:"page"`
	Offset  int `json:"offset"`
	Limit   int `json:"limit"`
}

// auth is a middleware that handles session authentication. If a session is not set,
// it creates one and redirects the user to the login page. If a session is set,
// it's authenticated before proceeding to the handler.
func authSession(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		sess, _ := session.Get("session", c)

		// It's a brand new session. Persist it.
		if sess.IsNew {
			sess.Options = &sessions.Options{
				Path:     "/",
				MaxAge:   86400 * 7,
				HttpOnly: true,
				// Secure:   true,
			}

			sess.Values["user_id"] = 1
			sess.Values["user"] = "kailash"
			sess.Values["role"] = "superadmin"
			sess.Values["user_email"] = "kailash@zerodha.com"

			sess.Save(c.Request(), c.Response())
		}

		return next(c)
	}
}

// handleIndex is the root handler that renders the login page if there's no
// authenticated session, or redirects to the dashboard, if there's one.
func handleIndexPage(c echo.Context) error {
	app := c.Get("app").(*App)
	return c.File(filepath.Join(app.Constants.AssetPath, "index.html"))
}

// makeAttribsBlob takes a list of keys and values and creates
// a JSON map out of them.
func makeAttribsBlob(keys []string, vals []string) ([]byte, bool) {
	attribs := make(map[string]interface{})
	for i, key := range keys {
		var (
			s   = vals[i]
			val interface{}
		)

		// Try to detect common JSON types.
		if govalidator.IsFloat(s) {
			val, _ = strconv.ParseFloat(s, 64)
		} else if govalidator.IsInt(s) {
			val, _ = strconv.ParseInt(s, 10, 64)
		} else {
			ls := strings.ToLower(s)
			if ls == "true" || ls == "false" {
				val, _ = strconv.ParseBool(ls)
			} else {
				// It's a string.
				val = s
			}
		}

		attribs[key] = val
	}

	if len(attribs) > 0 {
		j, _ := json.Marshal(attribs)
		return j, true
	}

	return nil, false
}

// getPagination takes form values and extracts pagination values from it.
func getPagination(q url.Values) pagination {
	var (
		perPage, _ = strconv.Atoi(q.Get("per_page"))
		page, _    = strconv.Atoi(q.Get("page"))
	)

	if perPage < 1 || perPage > maxPerPage {
		perPage = defaultPerPage
	}

	if page < 1 {
		page = 0
	} else {
		page--
	}

	return pagination{
		Page:    page + 1,
		PerPage: perPage,
		Offset:  page * perPage,
		Limit:   perPage,
	}
}
