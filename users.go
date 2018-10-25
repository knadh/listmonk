package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/asaskevich/govalidator"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo"
)

// handleGetUsers handles retrieval of users.
func handleGetUsers(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		out []models.User

		id, _  = strconv.Atoi(c.Param("id"))
		single = false
	)

	// Fetch one list.
	if id > 0 {
		single = true
	}

	err := app.Queries.GetUsers.Select(&out, id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error fetching users: %s", pqErrMsg(err)))
	} else if single && len(out) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "User not found.")
	} else if len(out) == 0 {
		return c.JSON(http.StatusOK, okResp{[]struct{}{}})
	}

	if single {
		return c.JSON(http.StatusOK, okResp{out[0]})
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleCreateUser handles user creation.
func handleCreateUser(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		o   = models.User{}
	)

	if err := c.Bind(&o); err != nil {
		return err
	}

	if !govalidator.IsEmail(o.Email) {
		return errors.New("invalid `email`")
	}
	if !govalidator.IsByteLength(o.Name, 1, stdInputMaxLen) {
		return errors.New("invalid length for `name`")
	}

	// Insert and read ID.
	var newID int
	if err := app.Queries.CreateUser.Get(&newID,
		o.Email,
		o.Name,
		o.Password,
		o.Type,
		o.Status); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error creating user: %v", pqErrMsg(err)))
	}

	// Hand over to the GET handler to return the last insertion.
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", newID))
	return c.JSON(http.StatusOK, handleGetLists(c))
}

// handleUpdateUser handles user modification.
func handleUpdateUser(c echo.Context) error {
	var (
		app   = c.Get("app").(*App)
		id, _ = strconv.Atoi(c.Param("id"))
	)

	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid ID.")
	} else if id == 1 {
		return echo.NewHTTPError(http.StatusBadRequest,
			"The primordial super admin cannot be deleted.")
	}

	var o models.User
	if err := c.Bind(&o); err != nil {
		return err
	}

	if !govalidator.IsEmail(o.Email) {
		return errors.New("invalid `email`")
	}
	if !govalidator.IsByteLength(o.Name, 1, stdInputMaxLen) {
		return errors.New("invalid length for `name`")
	}

	// TODO: PASSWORD HASHING.
	res, err := app.Queries.UpdateUser.Exec(o.ID,
		o.Email,
		o.Name,
		o.Password,
		o.Type,
		o.Status)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error updating user: %s", pqErrMsg(err)))
	}

	if n, _ := res.RowsAffected(); n == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "User not found.")
	}

	return handleGetUsers(c)
}

// handleDeleteUser handles user deletion.
func handleDeleteUser(c echo.Context) error {
	var (
		id, _ = strconv.Atoi(c.Param("id"))
	)

	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid ID.")
	}

	return c.JSON(http.StatusOK, okResp{true})
}
