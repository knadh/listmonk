package main

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
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
			app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.templates}", "error", pqErrMsg(err)))
	}
	if single && len(out) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.template}"))
	}

	if len(out) == 0 {
		return c.JSON(http.StatusOK, okResp{[]struct{}{}})
	} else if single {
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
				app.i18n.Ts("templates.placeholderHelp", "placeholder", tplTag))
		}
	} else {
		if id < 1 {
			return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
		}

		err := app.queries.GetTemplates.Select(&tpls, id, false)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError,
				app.i18n.Ts("globals.messages.errorFetching",
					"name", "{globals.terms.templates}", "error", pqErrMsg(err)))
		}

		if len(tpls) == 0 {
			return echo.NewHTTPError(http.StatusBadRequest,
				app.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.template}"))
		}
		body = tpls[0].Body
	}

	// Compile the template.
	camp := models.Campaign{
		UUID:         dummyUUID,
		Name:         app.i18n.T("templates.dummyName"),
		Subject:      app.i18n.T("templates.dummySubject"),
		FromEmail:    "dummy-campaign@listmonk.app",
		TemplateBody: body,
		Body:         dummyTpl,
	}

	if err := camp.CompileTemplate(app.manager.TemplateFuncs(&camp)); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("templates.errorCompiling", "error", err.Error()))
	}

	// Render the message body.
	msg, err := app.manager.NewCampaignMessage(&camp, dummySubscriber)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("templates.errorRendering", "error", err.Error()))
	}

	return c.HTML(http.StatusOK, string(msg.Body()))
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

	if err := validateTemplate(o, app); err != nil {
		return err
	}

	// Insert and read ID.
	var newID int
	if err := app.queries.CreateTemplate.Get(&newID,
		o.Name,
		o.Body); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorCreating",
				"name", "{globals.terms.template}", "error", pqErrMsg(err)))
	}

	// Hand over to the GET handler to return the last insertion.
	return handleGetTemplates(copyEchoCtx(c, map[string]string{
		"id": fmt.Sprintf("%d", newID),
	}))
}

// handleUpdateTemplate handles template modification.
func handleUpdateTemplate(c echo.Context) error {
	var (
		app   = c.Get("app").(*App)
		id, _ = strconv.Atoi(c.Param("id"))
	)

	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
	}

	var o models.Template
	if err := c.Bind(&o); err != nil {
		return err
	}

	if err := validateTemplate(o, app); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	res, err := app.queries.UpdateTemplate.Exec(id, o.Name, o.Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorUpdating",
				"name", "{globals.terms.template}", "error", pqErrMsg(err)))
	}

	if n, _ := res.RowsAffected(); n == 0 {
		return echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.template}"))
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
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
	}

	_, err := app.queries.SetDefaultTemplate.Exec(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorUpdating",
				"name", "{globals.terms.template}", "error", pqErrMsg(err)))
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
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
	}

	var delID int
	err := app.queries.DeleteTemplate.Get(&delID, id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorDeleting",
				"name", "{globals.terms.template}", "error", pqErrMsg(err)))
	}
	if delID == 0 {
		return echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.T("templates.cantDeleteDefault"))
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// validateTemplate validates template fields.
func validateTemplate(o models.Template, app *App) error {
	if !strHasLen(o.Name, 1, stdInputMaxLen) {
		return errors.New(app.i18n.T("campaigns.fieldInvalidName"))
	}

	if !regexpTplTag.MatchString(o.Body) {
		return echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("templates.placeholderHelp", "placeholder", tplTag))
	}

	return nil
}
