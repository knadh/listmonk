package core

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/knadh/listmonk/internal/auth"
	"github.com/knadh/listmonk/internal/utils"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
	"gopkg.in/volatiletech/null.v6"
)

func (c *Core) GetUsers() ([]auth.User, error) {
	out := []auth.User{}
	if err := c.q.GetUsers.Select(&out); err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.users}", "error", pqErrMsg(err)))
	}

	return c.setupUserFields(out), nil
}

// GetUser retrieves a specific user based on any one given identifier.
func (c *Core) GetUser(id int, username, email string) (auth.User, error) {
	var out auth.User
	if err := c.q.GetUser.Get(&out, id, username, email); err != nil {
		if err == sql.ErrNoRows {
			return out, echo.NewHTTPError(http.StatusNotFound,
				c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.user}"))

		}

		return out, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.users}", "error", pqErrMsg(err)))
	}

	return c.setupUserFields([]auth.User{out})[0], nil
}

// CreateUser creates a new user.
func (c *Core) CreateUser(u auth.User) (auth.User, error) {
	var id int

	// If it's an API user, generate a random token for password
	// and set the e-mail to default.
	if u.Type == auth.UserTypeAPI {
		// Generate a random admin password.
		tk, err := utils.GenerateRandomString(32)
		if err != nil {
			return auth.User{}, err
		}

		u.Email = null.String{String: u.Username + "@api", Valid: true}
		u.PasswordLogin = false
		u.Password = null.String{String: tk, Valid: true}
	}

	if err := c.q.CreateUser.Get(&id, u.Username, u.PasswordLogin, u.Password, u.Email, u.Name, u.Type, u.UserRoleID, u.ListRoleID, u.Status); err != nil {
		return auth.User{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.user}", "error", pqErrMsg(err)))
	}

	// Hide the password field in the response except for when the user type is an API token,
	// where the frontend shows the token on the UI just once.
	if u.Type != auth.UserTypeAPI {
		u.Password = null.String{Valid: false}
	}

	out, err := c.GetUser(id, "", "")
	return out, err
}

// UpdateUser updates a given user.
func (c *Core) UpdateUser(id int, u auth.User) (auth.User, error) {
	listRoleID := 0
	if u.ListRoleID == nil {
		listRoleID = -1
	} else {
		listRoleID = *u.ListRoleID
	}

	res, err := c.q.UpdateUser.Exec(id, u.Username, u.PasswordLogin, u.Password, u.Email, u.Name, u.Type, u.UserRoleID, listRoleID, u.Status)
	if err != nil {
		return auth.User{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.user}", "error", pqErrMsg(err)))
	}

	if n, _ := res.RowsAffected(); n == 0 {
		return auth.User{}, echo.NewHTTPError(http.StatusBadRequest, c.i18n.T("users.needSuper"))
	}

	out, err := c.GetUser(id, "", "")

	return out, err
}

// UpdateUserProfile updates the basic fields of a given uesr (name, email, password).
func (c *Core) UpdateUserProfile(id int, u auth.User) (auth.User, error) {
	res, err := c.q.UpdateUserProfile.Exec(id, u.Name, u.Email, u.PasswordLogin, u.Password)
	if err != nil {
		return auth.User{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.user}", "error", pqErrMsg(err)))
	}

	if n, _ := res.RowsAffected(); n == 0 {
		return auth.User{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.user}"))
	}

	return c.GetUser(id, "", "")
}

// UpdateUserLogin updates a user's record post-login.
func (c *Core) UpdateUserLogin(id int, avatar string) error {
	if _, err := c.q.UpdateUserLogin.Exec(id, avatar); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.user}", "error", pqErrMsg(err)))
	}

	return nil
}

// DeleteUsers deletes a given user.
func (c *Core) DeleteUsers(ids []int) error {
	res, err := c.q.DeleteUsers.Exec(pq.Array(ids))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.user}", "error", pqErrMsg(err)))
	}
	if num, err := res.RowsAffected(); err != nil || num == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, c.i18n.T("users.needSuper"))
	}

	return nil
}

// LoginUser attempts to log the given user_id in by matching the password.
func (c *Core) LoginUser(username, password string) (auth.User, error) {
	var out auth.User
	if err := c.q.LoginUser.Get(&out, username, password); err != nil {
		if err == sql.ErrNoRows {
			return out, echo.NewHTTPError(http.StatusForbidden, c.i18n.T("users.invalidLogin"))
		}

		return out, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.users}", "error", pqErrMsg(err)))
	}

	return out, nil
}

// setupUserFields prepares and sets up various user fields.
func (c *Core) setupUserFields(users []auth.User) []auth.User {
	for n, u := range users {
		u := u

		if u.Password.String != "" {
			u.HasPassword = true
			u.PasswordLogin = true
		}

		if u.Type == auth.UserTypeAPI {
			u.Email = null.String{}
		}

		u.UserRole.ID = u.UserRoleID
		u.UserRole.Name = u.UserRoleName
		u.UserRole.Permissions = u.UserRolePerms
		u.UserRoleID = 0

		// Prepare lookup maps.
		u.ListPermissionsMap = make(map[int]map[string]struct{})
		u.PermissionsMap = make(map[string]struct{})
		for _, p := range u.UserRolePerms {
			u.PermissionsMap[p] = struct{}{}
		}

		if u.ListRoleID != nil {
			// Unmarshall the raw list perms map.
			var listPerms []auth.ListPermission
			if u.ListsPermsRaw != nil {
				if err := json.Unmarshal(*u.ListsPermsRaw, &listPerms); err != nil {
					c.log.Printf("error unmarshalling list permissions for role %d: %v", u.ID, err)
				}
			}

			u.ListRole = &auth.ListRolePermissions{ID: *u.ListRoleID, Name: u.ListRoleName.String, Lists: listPerms}

			// Iterate each list in the list permissions and setup get/manage list IDs.
			for _, p := range listPerms {
				u.ListPermissionsMap[p.ID] = make(map[string]struct{})

				for _, perm := range p.Permissions {
					u.ListPermissionsMap[p.ID][perm] = struct{}{}

					// List IDs with get / manage permissions.
					if perm == auth.PermListGet {
						u.GetListIDs = append(u.GetListIDs, p.ID)
					}
					if perm == auth.PermListManage {
						u.ManageListIDs = append(u.ManageListIDs, p.ID)
					}
				}
			}
		}

		users[n] = u
	}

	return users
}
