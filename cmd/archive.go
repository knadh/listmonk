package main

import (
	"bytes"
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"

	"github.com/gorilla/feeds"
	"github.com/knadh/listmonk/internal/manager"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	null "gopkg.in/volatiletech/null.v6"
)

type campArchive struct {
	UUID      string    `json:"uuid"`
	Subject   string    `json:"subject"`
	Content   string    `json:"content"`
	CreatedAt null.Time `json:"created_at"`
	SendAt    null.Time `json:"send_at"`
	URL       string    `json:"url"`
}

// GetCampaignArchives renders the public campaign archives page.
func (a *App) GetCampaignArchives(c echo.Context) error {
	// Get archives from the DB.
	pg := a.pg.NewFromURL(c.Request().URL.Query())
	camps, total, err := a.getCampaignArchives(pg.Offset, pg.Limit, false)
	if err != nil {
		return err
	}

	if len(camps) == 0 {
		return c.JSON(http.StatusOK, okResp{models.PageResults{
			Results: []campArchive{},
		}})
	}

	// Meta.
	out := models.PageResults{
		Results: camps,
		Total:   total,
		Page:    pg.Page,
		PerPage: pg.PerPage,
	}

	return c.JSON(200, okResp{out})
}

// GetCampaignArchivesFeed renders the public campaign archives RSS feed.
func (a *App) GetCampaignArchivesFeed(c echo.Context) error {
	var (
		pg              = a.pg.NewFromURL(c.Request().URL.Query())
		showFullContent = a.cfg.EnablePublicArchiveRSSContent
	)

	// Get archives from the DB.
	camps, _, err := a.getCampaignArchives(pg.Offset, pg.Limit, showFullContent)
	if err != nil {
		return err
	}

	// Format output for the feed.
	out := make([]*feeds.Item, 0, len(camps))
	for _, c := range camps {
		pubDate := c.CreatedAt.Time

		if c.SendAt.Valid {
			pubDate = c.SendAt.Time
		}

		out = append(out, &feeds.Item{
			Title:   c.Subject,
			Link:    &feeds.Link{Href: c.URL},
			Content: c.Content,
			Created: pubDate,
		})
	}

	// Generate the feed.
	feed := &feeds.Feed{
		Title:       a.cfg.SiteName,
		Link:        &feeds.Link{Href: a.urlCfg.RootURL},
		Description: a.i18n.T("public.archiveTitle"),
		Items:       out,
	}

	if err := feed.WriteRss(c.Response().Writer); err != nil {
		a.log.Printf("error generating archive RSS feed: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("public.errorProcessingRequest"))
	}

	return nil
}

// CampaignArchivesPage renders the public campaign archives page.
func (a *App) CampaignArchivesPage(c echo.Context) error {
	// Get archives from the DB.
	pg := a.pg.NewFromURL(c.Request().URL.Query())
	out, total, err := a.getCampaignArchives(pg.Offset, pg.Limit, false)
	if err != nil {
		return err
	}
	pg.SetTotal(total)

	title := a.i18n.T("public.archiveTitle")
	return c.Render(http.StatusOK, "archive", struct {
		Title       string
		Description string
		Campaigns   []campArchive
		TotalPages  int
		Pagination  template.HTML
	}{title, title, out, pg.TotalPages, template.HTML(pg.HTML("?page=%d"))})
}

// CampaignArchivePage renders the public campaign archives page.
func (a *App) CampaignArchivePage(c echo.Context) error {
	// ID can be the UUID or slug.
	var (
		idStr      = c.Param("id")
		uuid, slug string
	)
	if reUUID.MatchString(idStr) {
		uuid = idStr
	} else {
		slug = idStr
	}

	// Get the campaign from the DB.
	pubCamp, err := a.core.GetArchivedCampaign(0, uuid, slug)
	if err != nil || pubCamp.Type != models.CampaignTypeRegular {
		notFound := false

		// Camppaig doesn't exist.
		if er, ok := err.(*echo.HTTPError); ok {
			if er.Code == http.StatusBadRequest {
				notFound = true
			}
		} else if pubCamp.Type != models.CampaignTypeRegular {
			// Campaign isn't of regular type.
			notFound = true
		}

		// 404.
		if notFound {
			return c.Render(http.StatusNotFound, tplMessage,
				makeMsgTpl(a.i18n.T("public.notFoundTitle"), "", a.i18n.T("public.campaignNotFound")))
		}

		// Some other internal error.
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(a.i18n.T("public.errorTitle"), "", a.i18n.Ts("public.errorFetchingCampaign")))
	}

	// "Compile" the campaign template with appropriate data.
	out, err := a.compileArchiveCampaigns([]models.Campaign{pubCamp})
	if err != nil {
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(a.i18n.T("public.errorTitle"), "", a.i18n.Ts("public.errorFetchingCampaign")))
	}

	// Render the campaign body.
	camp := out[0].Campaign
	msg, err := a.manager.NewCampaignMessage(camp, out[0].Subscriber)
	if err != nil {
		a.log.Printf("error rendering campaign: %v", err)
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(a.i18n.T("public.errorTitle"), "", a.i18n.Ts("public.errorFetchingCampaign")))
	}

	return c.HTML(http.StatusOK, string(msg.Body()))
}

// CampaignArchivePageLatest renders the latest public campaign.
func (a *App) CampaignArchivePageLatest(c echo.Context) error {
	// Get the latest campaign from the DB.
	camps, _, err := a.getCampaignArchives(0, 1, true)
	if err != nil {
		return err
	}

	if len(camps) == 0 {
		return c.Render(http.StatusNotFound, tplMessage,
			makeMsgTpl(a.i18n.T("public.notFoundTitle"), "", a.i18n.T("public.campaignNotFound")))
	}
	camp := camps[0]

	return c.HTML(http.StatusOK, camp.Content)
}

// getCampaignArchives fetches the public campaign archives from the DB.
func (a *App) getCampaignArchives(offset, limit int, renderBody bool) ([]campArchive, int, error) {
	pubCamps, total, err := a.core.GetArchivedCampaigns(offset, limit)
	if err != nil {
		return []campArchive{}, total, echo.NewHTTPError(http.StatusInternalServerError, a.i18n.T("public.errorFetchingCampaign"))
	}

	msgs, err := a.compileArchiveCampaigns(pubCamps)
	if err != nil {
		return []campArchive{}, total, err
	}

	out := make([]campArchive, 0, len(msgs))
	for _, m := range msgs {
		camp := m.Campaign

		archive := campArchive{
			UUID:      camp.UUID,
			Subject:   camp.Subject,
			CreatedAt: camp.CreatedAt,
			SendAt:    camp.SendAt,
		}

		// The campaign may have a custom slug.
		if camp.ArchiveSlug.Valid {
			archive.URL, _ = url.JoinPath(a.urlCfg.ArchiveURL, camp.ArchiveSlug.String)
		} else {
			archive.URL, _ = url.JoinPath(a.urlCfg.ArchiveURL, camp.UUID)
		}

		// Render the full template body if requested.
		if renderBody {
			msg, err := a.manager.NewCampaignMessage(camp, m.Subscriber)
			if err != nil {
				return []campArchive{}, total, err
			}
			archive.Content = string(msg.Body())
		}

		out = append(out, archive)
	}

	return out, total, nil
}

// compileArchiveCampaigns compiles the campaign template with the subscriber data.
func (a *App) compileArchiveCampaigns(camps []models.Campaign) ([]manager.CampaignMessage, error) {

	var (
		b   = bytes.Buffer{}
		out = make([]manager.CampaignMessage, 0, len(camps))
	)
	for _, c := range camps {
		camp := c
		if err := camp.CompileTemplate(a.manager.TemplateFuncs(&camp)); err != nil {
			a.log.Printf("error compiling template: %v", err)
			return nil, echo.NewHTTPError(http.StatusInternalServerError, a.i18n.T("public.errorFetchingCampaign"))
		}

		// Load the dummy subscriber meta.
		var sub models.Subscriber
		if err := json.Unmarshal([]byte(camp.ArchiveMeta), &sub); err != nil {
			a.log.Printf("error unmarshalling campaign archive meta: %v", err)
			return nil, echo.NewHTTPError(http.StatusInternalServerError, a.i18n.T("public.errorFetchingCampaign"))
		}

		m := manager.CampaignMessage{
			Campaign:   &camp,
			Subscriber: sub,
		}

		// Render the subject if it's a template.
		if camp.SubjectTpl != nil {
			if err := camp.SubjectTpl.ExecuteTemplate(&b, models.ContentTpl, m); err != nil {
				return nil, err
			}
			camp.Subject = b.String()
			b.Reset()
		}

		out = append(out, m)
	}

	return out, nil
}
