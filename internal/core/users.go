package core

import (
	"database/sql"
	"net/http"

	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
)

// GetUsers retrieves all users.
func (c *Core) GetUsers() ([]models.User, error) {
	out := []models.User{}
	if err := c.q.GetUsers.Select(&out, 0); err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.users}", "error", pqErrMsg(err)))
	}

	for n, u := range out {
		if u.Password.String != "" {
			u.HasPassword = true
			u.PasswordLogin = true
			u.Password.String = ""
			u.Password.Valid = false
			out[n] = u
		}
	}

	return out, nil
}

// GetUser retrieves a specific user.
func (c *Core) GetUser(id int) (models.User, error) {
	var out models.User
	if err := c.q.GetUsers.Get(&out, id); err != nil {
		return out, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.users}", "error", pqErrMsg(err)))
	}
	if out.Password.String != "" {
		out.HasPassword = true
		out.PasswordLogin = true
		out.Password.String = ""
		out.Password.Valid = false
	}

	return out, nil
}

// CreateUser creates a new user.
func (c *Core) CreateUser(u models.User) (models.User, error) {
	var out models.User
	if err := c.q.CreateUser.Get(&out, u.Username, u.PasswordLogin, u.Password, u.Email, u.Name, u.Status); err != nil {
		return models.User{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.user}", "error", pqErrMsg(err)))
	}

	return out, nil
}

// UpdateUser updates a given user.
func (c *Core) UpdateUser(id int, u models.User) (models.User, error) {
	res, err := c.q.UpdateUser.Exec(id, u.Username, u.PasswordLogin, u.Password, u.Email, u.Name, u.Status)
	if err != nil {
		return models.User{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.user}", "error", pqErrMsg(err)))
	}

	if n, _ := res.RowsAffected(); n == 0 {
		return models.User{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.user}"))
	}

	return c.GetUser(id)
}

// DeleteUsers deletes a given user.
func (c *Core) DeleteUsers(ids []int) error {
	res, err := c.q.DeleteUsers.Exec(pq.Array(ids))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.user}", "error", pqErrMsg(err)))
	}
	if num, err := res.RowsAffected(); err != nil || num == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, c.i18n.T("users.cantDelete"))
	}

	return nil
}

// LoginUser attempts to log the given user_id in by matching the password.
func (c *Core) LoginUser(username, password string) (models.User, error) {
	var out models.User
	if err := c.q.LoginUser.Get(&out, username, password); err != nil {
		if err == sql.ErrNoRows {
			return out, echo.NewHTTPError(http.StatusForbidden,
				c.i18n.T("users.invalidLogin"))
		}

		return out, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.users}", "error", pqErrMsg(err)))
	}

	return out, nil
}
