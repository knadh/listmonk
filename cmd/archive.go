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

// handleGetCampaignArchives renders the public campaign archives page.
func (h *Handler) handleGetCampaignArchives(c echo.Context) error {
	var (
		pg = h.app.paginator.NewFromURL(c.Request().URL.Query())
	)

	camps, total, err := getCampaignArchives(pg.Offset, pg.Limit, false, h.app)
	if err != nil {
		return err
	}

	var out models.PageResults
	if len(camps) == 0 {
		out.Results = []campArchive{}
		return c.JSON(http.StatusOK, okResp{out})
	}

	// Meta.
	out.Results = camps
	out.Total = total
	out.Page = pg.Page
	out.PerPage = pg.PerPage

	return c.JSON(200, okResp{out})
}

// handleGetCampaignArchivesFeed renders the public campaign archives RSS feed.
func (h *Handler) handleGetCampaignArchivesFeed(c echo.Context) error {
	var (
		pg              = h.app.paginator.NewFromURL(c.Request().URL.Query())
		showFullContent = h.app.constants.EnablePublicArchiveRSSContent
	)

	camps, _, err := getCampaignArchives(pg.Offset, pg.Limit, showFullContent, h.app)
	if err != nil {
		return err
	}

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

	feed := &feeds.Feed{
		Title:       h.app.constants.SiteName,
		Link:        &feeds.Link{Href: h.app.constants.RootURL},
		Description: h.app.i18n.T("public.archiveTitle"),
		Items:       out,
	}

	if err := feed.WriteRss(c.Response().Writer); err != nil {
		h.app.log.Printf("error generating archive RSS feed: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, h.app.i18n.T("public.errorProcessingRequest"))
	}

	return nil
}

// handleCampaignArchivesPage renders the public campaign archives page.
func (h *Handler) handleCampaignArchivesPage(c echo.Context) error {
	var (
		pg = h.app.paginator.NewFromURL(c.Request().URL.Query())
	)

	out, total, err := getCampaignArchives(pg.Offset, pg.Limit, false, h.app)
	if err != nil {
		return err
	}
	pg.SetTotal(total)

	title := h.app.i18n.T("public.archiveTitle")
	return c.Render(http.StatusOK, "archive", struct {
		Title       string
		Description string
		Campaigns   []campArchive
		TotalPages  int
		Pagination  template.HTML
	}{title, title, out, pg.TotalPages, template.HTML(pg.HTML("?page=%d"))})
}

// handleCampaignArchivePage renders the public campaign archives page.
func (h *Handler) handleCampaignArchivePage(c echo.Context) error {
	var (
		id   = c.Param("id")
		uuid = ""
		slug = ""
	)

	// ID can be the UUID or slug.
	if reUUID.MatchString(id) {
		uuid = id
	} else {
		slug = id
	}

	pubCamp, err := h.app.core.GetArchivedCampaign(0, uuid, slug)
	if err != nil || pubCamp.Type != models.CampaignTypeRegular {
		notFound := false
		if er, ok := err.(*echo.HTTPError); ok {
			if er.Code == http.StatusBadRequest {
				notFound = true
			}
		} else if pubCamp.Type != models.CampaignTypeRegular {
			notFound = true
		}

		if notFound {
			return c.Render(http.StatusNotFound, tplMessage,
				makeMsgTpl(h.app.i18n.T("public.notFoundTitle"), "", h.app.i18n.T("public.campaignNotFound")))
		}

		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(h.app.i18n.T("public.errorTitle"), "", h.app.i18n.Ts("public.errorFetchingCampaign")))
	}

	out, err := compileArchiveCampaigns([]models.Campaign{pubCamp}, h.app)
	if err != nil {
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(h.app.i18n.T("public.errorTitle"), "", h.app.i18n.Ts("public.errorFetchingCampaign")))
	}

	// Render the message body.
	camp := out[0].Campaign
	msg, err := h.app.manager.NewCampaignMessage(camp, out[0].Subscriber)
	if err != nil {
		h.app.log.Printf("error rendering message: %v", err)
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(h.app.i18n.T("public.errorTitle"), "", h.app.i18n.Ts("public.errorFetchingCampaign")))
	}

	return c.HTML(http.StatusOK, string(msg.Body()))
}

// handleCampaignArchivePageLatest renders the latest public campaign.
func (h *Handler) handleCampaignArchivePageLatest(c echo.Context) error {
	camps, _, err := getCampaignArchives(0, 1, true, h.app)
	if err != nil {
		return err
	}

	if len(camps) == 0 {
		return c.Render(http.StatusNotFound, tplMessage,
			makeMsgTpl(h.app.i18n.T("public.notFoundTitle"), "", h.app.i18n.T("public.campaignNotFound")))
	}

	camp := camps[0]

	return c.HTML(http.StatusOK, camp.Content)
}

func getCampaignArchives(offset, limit int, renderBody bool, app *App) ([]campArchive, int, error) {
	pubCamps, total, err := app.core.GetArchivedCampaigns(offset, limit)
	if err != nil {
		return []campArchive{}, total, echo.NewHTTPError(http.StatusInternalServerError, app.i18n.T("public.errorFetchingCampaign"))
	}

	msgs, err := compileArchiveCampaigns(pubCamps, app)
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

		if camp.ArchiveSlug.Valid {
			archive.URL, _ = url.JoinPath(app.constants.ArchiveURL, camp.ArchiveSlug.String)
		} else {
			archive.URL, _ = url.JoinPath(app.constants.ArchiveURL, camp.UUID)
		}

		if renderBody {
			msg, err := app.manager.NewCampaignMessage(camp, m.Subscriber)
			if err != nil {
				return []campArchive{}, total, err
			}
			archive.Content = string(msg.Body())
		}

		out = append(out, archive)
	}

	return out, total, nil
}

func compileArchiveCampaigns(camps []models.Campaign, app *App) ([]manager.CampaignMessage, error) {
	var (
		b = bytes.Buffer{}
	)

	out := make([]manager.CampaignMessage, 0, len(camps))
	for _, c := range camps {
		camp := c
		if err := camp.CompileTemplate(app.manager.TemplateFuncs(&camp)); err != nil {
			app.log.Printf("error compiling template: %v", err)
			return nil, echo.NewHTTPError(http.StatusInternalServerError, app.i18n.T("public.errorFetchingCampaign"))
		}

		// Load the dummy subscriber meta.
		var sub models.Subscriber
		if err := json.Unmarshal([]byte(camp.ArchiveMeta), &sub); err != nil {
			app.log.Printf("error unmarshalling campaign archive meta: %v", err)
			return nil, echo.NewHTTPError(http.StatusInternalServerError, app.i18n.T("public.errorFetchingCampaign"))
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
