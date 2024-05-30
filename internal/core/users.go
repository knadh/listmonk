package core

import (
	"database/sql"
	"net/http"

	"github.com/knadh/listmonk/internal/utils"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
	"gopkg.in/volatiletech/null.v6"
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
			// u.Password = null.String{}

			out[n] = u
		}

		if u.Type == models.UserTypeAPI {
			out[n].Email = null.String{}
		}
	}

	return out, nil
}

// GetUser retrieves a specific user based on any one given identifier.
func (c *Core) GetUser(id int, username, email string) (models.User, error) {
	var out models.User
	if err := c.q.GetUser.Get(&out, id, username, email); err != nil {
		return out, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.users}", "error", pqErrMsg(err)))
	}
	if out.Password.String != "" {
		out.HasPassword = true
		out.PasswordLogin = true
	}

	return out, nil
}

// CreateUser creates a new user.
func (c *Core) CreateUser(u models.User) (models.User, error) {
	var out models.User

	// If it's an API user, generate a random token for password
	// and set the e-mail to default.
	if u.Type == models.UserTypeAPI {
		// Generate a random admin password.
		tk, err := utils.GenerateRandomString(32)
		if err != nil {
			return out, err
		}

		u.Email = null.String{String: u.Username + "@api", Valid: true}
		u.PasswordLogin = false
		u.Password = null.String{String: tk, Valid: true}
	}

	if err := c.q.CreateUser.Get(&out, u.Username, u.PasswordLogin, u.Password, u.Email, u.Name, u.Type, u.Status); err != nil {
		return models.User{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.user}", "error", pqErrMsg(err)))
	}

	// Hide the password field in the response except for when the user type is an API token,
	// where the frontend shows the token on the UI just once.
	if u.Type != models.UserTypeAPI {
		u.Password = null.String{Valid: false}
	}

	return out, nil
}

// UpdateUser updates a given user.
func (c *Core) UpdateUser(id int, u models.User) (models.User, error) {
	res, err := c.q.UpdateUser.Exec(id, u.Username, u.PasswordLogin, u.Password, u.Email, u.Name, u.Type, u.Status)
	if err != nil {
		return models.User{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.user}", "error", pqErrMsg(err)))
	}

	if n, _ := res.RowsAffected(); n == 0 {
		return models.User{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.user}"))
	}

	return c.GetUser(id, "", "")
}

// UpdateUserProfile updates the basic fields of a given uesr (name, email, password).
func (c *Core) UpdateUserProfile(id int, u models.User) (models.User, error) {
	res, err := c.q.UpdateUserProfile.Exec(id, u.Name, u.Email, u.PasswordLogin, u.Password)
	if err != nil {
		return models.User{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.user}", "error", pqErrMsg(err)))
	}

	if n, _ := res.RowsAffected(); n == 0 {
		return models.User{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.user}"))
	}

	return c.GetUser(id, "", "")
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
