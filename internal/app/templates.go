package app

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo"
)

const (
	// tplTag is the template tag that should be present in a template
	// as the placeholder for campaign bodies.
	tplTag = `{{ template "content" . }}`

	dummyTpl = `
		<p>Hi there</p>
		<p>Lorem ipsum dolor sit amet, consectetur adipiscing elit. Duis et elit ac elit sollicitudin condimentum non a magna. Sed tempor mauris in facilisis vehicula. Aenean nisl urna, accumsan ac tincidunt vitae, interdum cursus massa. Interdum et malesuada fames ac ante ipsum primis in faucibus. Aliquam varius turpis et turpis lacinia placerat. Aenean id ligula a orci lacinia blandit at eu felis. Phasellus vel lobortis lacus. Suspendisse leo elit, luctus sed erat ut, venenatis fermentum ipsum. Donec bibendum neque quis.</p>

		<h3>Sub heading</h3>
		<p>Nam luctus dui non placerat mattis. Morbi non accumsan orci, vel interdum urna. Duis faucibus id nunc ut euismod. Curabitur et eros id erat feugiat fringilla in eget neque. Aliquam accumsan cursus eros sed faucibus.</p>

		<p>Here is a link to <a href="https://listmonk.app" target="_blank">listmonk</a>.</p>`
)

var (
	regexpTplTag = regexp.MustCompile(`{{(\s+)?template\s+?"content"(\s+)?\.(\s+)?}}`)
)

// handleGetTemplates handles retrieval of templates.
func handleGetTemplates(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		out []models.Template

		id, _     = strconv.Atoi(c.Param("id"))
		single    = false
		noBody, _ = strconv.ParseBool(c.QueryParam("no_body"))
	)

	// Fetch one list.
	if id > 0 {
		single = true
	}

	err := app.queries.GetTemplates.Select(&out, id, noBody)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error fetching templates: %s", pqErrMsg(err)))
	}
	if single && len(out) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Template not found.")
	}

	if len(out) == 0 {
		return c.JSON(http.StatusOK, okResp{[]struct{}{}})
	}
	if single {
		return c.JSON(http.StatusOK, okResp{out[0]})
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handlePreviewTemplate renders the HTML preview of a template.
func handlePreviewTemplate(c echo.Context) error {
	var (
		app   = c.Get("app").(*App)
		id, _ = strconv.Atoi(c.Param("id"))
		body  = c.FormValue("body")

		tpls []models.Template
	)

	if body != "" {
		if !regexpTplTag.MatchString(body) {
			return echo.NewHTTPError(http.StatusBadRequest,
				fmt.Sprintf("Template body should contain the %s placeholder exactly once", tplTag))
		}
	} else {
		if id < 1 {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid ID.")
		}

		err := app.queries.GetTemplates.Select(&tpls, id, false)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError,
				fmt.Sprintf("Error fetching templates: %s", pqErrMsg(err)))
		}

		if len(tpls) == 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "Template not found.")
		}
		body = tpls[0].Body
	}

	// Compile the template.
	camp := models.Campaign{
		UUID:         dummyUUID,
		Name:         "Dummy Campaign",
		Subject:      "Dummy Campaign Subject",
		FromEmail:    "dummy-campaign@listmonk.app",
		TemplateBody: body,
		Body:         dummyTpl,
	}

	if err := camp.CompileTemplate(app.manager.TemplateFuncs(&camp)); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Error compiling template: %v", err))
	}

	// Render the message body.
	m := app.manager.NewCampaignMessage(&camp, dummySubscriber)
	if err := m.Render(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			fmt.Sprintf("Error rendering message: %v", err))
	}

	return c.HTML(http.StatusOK, string(m.Body()))
}

// handleCreateTemplate handles template creation.
func handleCreateTemplate(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		o   = models.Template{}
	)

	if err := c.Bind(&o); err != nil {
		return err
	}

	if err := validateTemplate(o); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Insert and read ID.
	var newID int
	if err := app.queries.CreateTemplate.Get(&newID,
		o.Name,
		o.Body); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error template user: %v", pqErrMsg(err)))
	}

	// Hand over to the GET handler to return the last insertion.
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", newID))
	return handleGetTemplates(c)
}

// handleUpdateTemplate handles template modification.
func handleUpdateTemplate(c echo.Context) error {
	var (
		app   = c.Get("app").(*App)
		id, _ = strconv.Atoi(c.Param("id"))
	)

	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid ID.")
	}

	var o models.Template
	if err := c.Bind(&o); err != nil {
		return err
	}

	if err := validateTemplate(o); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// TODO: PASSWORD HASHING.
	res, err := app.queries.UpdateTemplate.Exec(o.ID, o.Name, o.Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error updating template: %s", pqErrMsg(err)))
	}

	if n, _ := res.RowsAffected(); n == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Template not found.")
	}

	return handleGetTemplates(c)
}

// handleTemplateSetDefault handles template modification.
func handleTemplateSetDefault(c echo.Context) error {
	var (
		app   = c.Get("app").(*App)
		id, _ = strconv.Atoi(c.Param("id"))
	)

	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid ID.")
	}

	_, err := app.queries.SetDefaultTemplate.Exec(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error updating template: %s", pqErrMsg(err)))
	}

	return handleGetTemplates(c)
}

// handleDeleteTemplate handles template deletion.
func handleDeleteTemplate(c echo.Context) error {
	var (
		app   = c.Get("app").(*App)
		id, _ = strconv.Atoi(c.Param("id"))
	)

	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid ID.")
	} else if id == 1 {
		return echo.NewHTTPError(http.StatusBadRequest, "Cannot delete the primordial template.")
	}

	var delID int
	err := app.queries.DeleteTemplate.Get(&delID, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusOK, okResp{true})
		}

		return echo.NewHTTPError(http.StatusBadRequest,
			fmt.Sprintf("Error deleting template: %v", err))
	}

	if delID == 0 {
		return echo.NewHTTPError(http.StatusBadRequest,
			"Cannot delete the last, default, or non-existent template.")
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// validateTemplate validates template fields.
func validateTemplate(o models.Template) error {
	if !strHasLen(o.Name, 1, stdInputMaxLen) {
		return errors.New("invalid length for `name`")
	}

	if !regexpTplTag.MatchString(o.Body) {
		return fmt.Errorf("template body should contain the %s placeholder exactly once", tplTag)
	}

	return nil
}
