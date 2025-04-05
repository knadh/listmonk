package main

import (
	"errors"
	"html/template"
	"net/http"
	"regexp"
	"strconv"
	"strings"

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

// GetTemplates handles retrieval of templates.
func (h *Handlers) GetTemplates(c echo.Context) error {
	// Fetch one list.
	var (
		id, _     = strconv.Atoi(c.Param("id"))
		noBody, _ = strconv.ParseBool(c.QueryParam("no_body"))
	)
	if id > 0 {
		out, err := h.app.core.GetTemplate(id, noBody)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, okResp{out})
	}

	// Fetch templates from the DB (with or without body).
	out, err := h.app.core.GetTemplates("", noBody)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// PreviewTemplate renders the HTML preview of a template.
func (h *Handlers) PreviewTemplate(c echo.Context) error {
	tpl := models.Template{
		Type: c.FormValue("template_type"),
		Body: c.FormValue("body"),
	}

	// Body is posted with the request.
	if tpl.Body != "" {
		if tpl.Type == "" {
			tpl.Type = models.TemplateTypeCampaign
		}

		if tpl.Type == models.TemplateTypeCampaign && !regexpTplTag.MatchString(tpl.Body) {
			return echo.NewHTTPError(http.StatusBadRequest,
				h.app.i18n.Ts("templates.placeholderHelp", "placeholder", tplTag))
		}
	} else {
		// There is no body. Fetch the template from the DB.
		id, _ := strconv.Atoi(c.Param("id"))
		if id < 1 {
			return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidID"))
		}

		t, err := h.app.core.GetTemplate(id, false)
		if err != nil {
			return err
		}

		tpl = t
	}

	// Compile the campaign template.
	var out []byte
	if tpl.Type == models.TemplateTypeCampaign {
		camp := models.Campaign{
			UUID:         dummyUUID,
			Name:         h.app.i18n.T("templates.dummyName"),
			Subject:      h.app.i18n.T("templates.dummySubject"),
			FromEmail:    "dummy-campaign@listmonk.app",
			TemplateBody: tpl.Body,
			Body:         dummyTpl,
		}

		if err := camp.CompileTemplate(h.app.manager.TemplateFuncs(&camp)); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest,
				h.app.i18n.Ts("templates.errorCompiling", "error", err.Error()))
		}

		// Render the message body.
		msg, err := h.app.manager.NewCampaignMessage(&camp, dummySubscriber)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest,
				h.app.i18n.Ts("templates.errorRendering", "error", err.Error()))
		}
		out = msg.Body()
	} else {
		// Compile transactional template.
		if err := tpl.Compile(h.app.manager.GenericTemplateFuncs()); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		m := models.TxMessage{
			Subject: tpl.Subject,
		}

		// Render the message.
		if err := m.Render(dummySubscriber, &tpl); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		out = m.Body
	}

	return c.HTML(http.StatusOK, string(out))
}

// CreateTemplate handles template creation.
func (h *Handlers) CreateTemplate(c echo.Context) error {
	var o models.Template
	if err := c.Bind(&o); err != nil {
		return err
	}
	if err := h.validateTemplate(o); err != nil {
		return err
	}

	// Subject is only relevant for fixed tx templates. For campaigns,
	// the subject changes per campaign and is on models.Campaign.
	var funcs template.FuncMap
	if o.Type == models.TemplateTypeCampaign {
		o.Subject = ""
		funcs = h.app.manager.TemplateFuncs(nil)
	} else {
		funcs = h.app.manager.GenericTemplateFuncs()
	}

	// Compile the template and validate.
	if err := o.Compile(funcs); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Create the template the in the DB.
	out, err := h.app.core.CreateTemplate(o.Name, o.Type, o.Subject, []byte(o.Body))
	if err != nil {
		return err
	}

	// If it's a transactional template, cache it in the manager
	// to be used for arbitrary incoming tx message pushes.
	if o.Type == models.TemplateTypeTx {
		h.app.manager.CacheTpl(out.ID, &o)
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// UpdateTemplate handles template modification.
func (h *Handlers) UpdateTemplate(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidID"))
	}

	var o models.Template
	if err := c.Bind(&o); err != nil {
		return err
	}
	if err := h.validateTemplate(o); err != nil {
		return err
	}

	// Subject is only relevant for fixed tx templates. For campaigns,
	// the subject changes per campaign and is on models.Campaign.
	var funcs template.FuncMap
	if o.Type == models.TemplateTypeCampaign {
		o.Subject = ""
		funcs = h.app.manager.TemplateFuncs(nil)
	} else {
		funcs = h.app.manager.GenericTemplateFuncs()
	}

	// Compile the template and validate.
	if err := o.Compile(funcs); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Update the template in the DB.
	out, err := h.app.core.UpdateTemplate(id, o.Name, o.Subject, []byte(o.Body))
	if err != nil {
		return err
	}

	// If it's a transactional template, cache it.
	if out.Type == models.TemplateTypeTx {
		h.app.manager.CacheTpl(out.ID, &o)
	}

	return c.JSON(http.StatusOK, okResp{out})

}

// TemplateSetDefault handles template modification.
func (h *Handlers) TemplateSetDefault(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidID"))
	}

	// Update the template in the DB.
	if err := h.app.core.SetDefaultTemplate(id); err != nil {
		return err
	}

	return h.GetTemplates(c)
}

// DeleteTemplate handles template deletion.
func (h *Handlers) DeleteTemplate(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("globals.messages.invalidID"))
	}

	// Delete the template from the DB.
	if err := h.app.core.DeleteTemplate(id); err != nil {
		return err
	}

	// Delete cached template.
	h.app.manager.DeleteTpl(id)

	return c.JSON(http.StatusOK, okResp{true})
}

// compileTemplate validates template fields.
func (h *Handlers) validateTemplate(o models.Template) error {
	if !strHasLen(o.Name, 1, stdInputMaxLen) {
		return errors.New(h.app.i18n.T("campaigns.fieldInvalidName"))
	}

	if o.Type == models.TemplateTypeCampaign && !regexpTplTag.MatchString(o.Body) {
		return echo.NewHTTPError(http.StatusBadRequest,
			h.app.i18n.Ts("templates.placeholderHelp", "placeholder", tplTag))
	}

	if o.Type == models.TemplateTypeTx && strings.TrimSpace(o.Subject) == "" {
		return echo.NewHTTPError(http.StatusBadRequest,
			h.app.i18n.Ts("globals.messages.missingFields", "name", "subject"))
	}

	return nil
}
