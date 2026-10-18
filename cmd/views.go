package main

import (
	"errors"
	"net/http"
	"strings"

	"github.com/knadh/listmonk/internal/auth"
	"github.com/labstack/echo/v5"
)

type adminView struct {
	Title       string
	Description string
	PageID      string
	Profile     auth.User
}

func newAdminView(c *echo.Context, title, description, pageID string) adminView {
	// Admin views like error pages can be rendereded inside/outside auth.
	profile, _ := c.Get(auth.UserHTTPCtxKey).(auth.User)

	return adminView{
		Title:       title,
		Description: description,
		PageID:      pageID,
		Profile:     profile,
	}
}

func (v adminView) Can(perms ...string) bool {
	return can(v.Profile, perms...)
}

func (v adminView) CanManageList(id int) bool {
	return canManageList(v.Profile, id)
}

func can(user auth.User, perms ...string) bool {
	for _, perm := range perms {
		if before, ok := strings.CutSuffix(perm, "*"); ok {
			prefix := before
			for _, p := range user.UserRole.Permissions {
				if strings.HasPrefix(p, prefix) {
					return true
				}
			}
			continue
		}
		if user.HasPerm(perm) {
			return true
		}
	}
	return false
}

func canManageList(user auth.User, id int) bool {
	return user.HasListPerm(auth.PermTypeManage, id) == nil
}

// renderAdminMessage renders a message page.
func (a *App) renderAdminMessage(c *echo.Context, status int, title, description string) error {
	return c.Render(status, "admin-message", newAdminView(c, title, description, ""))
}

// adminPage captures errors from admin HTML handlers and converts
// error messages into HTML renders.
func (a *App) adminPage(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		if err := next(c); err != nil {
			return a.renderAdminError(c, err)
		}

		return nil
	}
}

// renderAdminError renders an error as an admin message page.
func (a *App) renderAdminError(c *echo.Context, err error) error {
	response, unwrapErr := echo.UnwrapResponse(c.Response())
	if unwrapErr != nil {
		return unwrapErr
	}
	if response.Committed {
		return nil
	}

	var (
		status = http.StatusInternalServerError
		msg    = http.StatusText(status)
	)
	if code := echo.StatusCode(err); code != 0 {
		status = code
		msg = http.StatusText(status)
		if httpErr, ok := errors.AsType[*echo.HTTPError](err); ok && httpErr.Message != "" {
			msg = httpErr.Message
		}
	} else {
		a.log.Println(err.Error())
	}

	if c.Request().Method == http.MethodHead {
		return c.NoContent(status)
	}

	if err := a.renderAdminMessage(c, status, "Error", msg); err != nil {
		a.log.Println(err.Error())
		// The template renderer itself failed!
		if !response.Committed {
			return c.HTML(http.StatusInternalServerError, "Internal Server Error")
		}
	}
	return nil
}
