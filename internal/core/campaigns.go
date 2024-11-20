package core

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/types"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
)

const (
	CampaignAnalyticsViews   = "views"
	CampaignAnalyticsClicks  = "clicks"
	CampaignAnalyticsBounces = "bounces"

	campaignTplDefault = "default"
	campaignTplArchive = "archive"
)

// QueryCampaigns retrieves paginated campaigns optionally filtering them by the given arbitrary
// query expression. It also returns the total number of records in the DB.
func (c *Core) QueryCampaigns(searchStr string, statuses, tags []string, orderBy, order string, offset, limit int, authid string) (models.Campaigns, int, error) {

	queryStr, stmt := makeSearchQuery(searchStr, orderBy, order, c.q.QueryCampaigns, campQuerySortFields, authid)

	if statuses == nil {
		statuses = []string{}
	}

	if tags == nil {
		tags = []string{}
	}
	// Unsafe to ignore scanning fields not present in models.Campaigns.
	var out models.Campaigns
	if err := c.db.Select(&out, stmt, 0, pq.StringArray(statuses), pq.StringArray(tags), queryStr, offset, limit, authid); err != nil {
		c.log.Printf("error fetching campaigns: %v", err)
		return nil, 0, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}

	for i := 0; i < len(out); i++ {
		// Replace null tags.
		if out[i].Tags == nil {
			out[i].Tags = []string{}
		}
	}

	// Lazy load stats.
	if err := out.LoadStats(c.q.GetCampaignStats); err != nil {
		c.log.Printf("error fetching campaign stats: %v", err)
		return nil, 0, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.campaigns}", "error", pqErrMsg(err)))
	}

	total := 0
	if len(out) > 0 {
		total = out[0].Total
	}

	return out, total, nil
}

// GetCampaign retrieves a campaign.
func (c *Core) GetCampaign(id int, uuid, archiveSlug string, authID string) (models.Campaign, error) {
	return c.getCampaign(id, uuid, archiveSlug, campaignTplDefault, authID)
}

// GetCampaign retrieves a campaign by authid.
func (c *Core) GetCampaignByAuthId(authid string) (models.Campaigns, error) {
	return c.getCampaignByAuthid(authid)
}

/*func (c *Core) GetArchivedCampaign(id int, uuid, archiveSlug string, authid string) (models.Campaign, error) {
	out, err := c.getCampaign(authid)
	if err != nil {
		return out, err
	}

	if !out.Archive {
		return models.Campaign{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.campaign}"))
	}

	return out, nil
}
*/

// GetArchivedCampaign retrieves a campaign with the archive template body.
func (c *Core) GetArchivedCampaign(id int, uuid, archiveSlug string, authID string) (models.Campaign, error) {
	out, err := c.getCampaign(id, uuid, archiveSlug, campaignTplArchive, authID)
	if err != nil {
		return out, err
	}

	if !out.Archive {
		return models.Campaign{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.campaign}"))
	}

	return out, nil
}

// getCampaign retrieves a campaign. If typlType=default, then the campaign's
// template body is returned as "template_body". If tplType="archive",
// the archive template is returned.
func (c *Core) getCampaign(id int, uuid, archiveSlug string, tplType string, authID string) (models.Campaign, error) {
	// Unsafe to ignore scanning fields not present in models.Campaigns.
	var uu interface{}
	if uuid != "" {
		uu = uuid
	}

	var out models.Campaigns
	if err := c.q.GetCampaign.Select(&out, id, uu, archiveSlug, tplType, authID); err != nil {
		// if err := c.db.Select(&out, stmt, 0, pq.Array([]string{}), queryStr, 0, 1); err != nil {
		c.log.Printf("error fetching campaign: %v", err)
		return models.Campaign{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}

	if len(out) == 0 {
		return models.Campaign{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.campaign}"))
	}

	for i := 0; i < len(out); i++ {
		// Replace null tags.
		if out[i].Tags == nil {
			out[i].Tags = []string{}
		}
	}
	// Lazy load stats.
	if err := out.LoadStats(c.q.GetCampaignStats); err != nil {
		c.log.Printf("error fetching campaign stats: %v", err)
		return models.Campaign{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}

	return out[0], nil
}

// getCampaignsByAuthid retrieves all campaigns associated with a given authid.
func (c *Core) getCampaignByAuthid(authid string) (models.Campaigns, error) {
	var out models.Campaigns
	if err := c.q.GetCampaignByAuthId.Select(&out, authid); err != nil {
		c.log.Printf("error fetching campaigns: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}

	c.log.Printf("Campaigns fetched successfully. Number of campaigns: %d", len(out))

	if len(out) == 0 {
		c.log.Println("No campaigns found for the provided authid.")
		return nil, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.campaign}"))
	}

	// Lazy load stats for each campaign
	if err := out.LoadStats(c.q.GetCampaignStats); err != nil {
		c.log.Printf("error fetching campaign stats: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}

	c.log.Println("Campaign stats loaded successfully.")
	return out, nil
}

/*
func (c *Core) getCampaignByAuthid(authid string) (models.Campaign, error) {

	var out models.Campaigns
	if err := c.q.GetCampaignByAuthId.Select(&out, authid); err != nil {
		c.log.Printf("error fetching campaign: %v", err)
		return models.Campaign{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}

	c.log.Printf("Campaign fetched successfully. Number of campaigns: %d", len(out))

	if len(out) == 0 {
		c.log.Println("No campaigns found.")
		return models.Campaign{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.campaign}"))
	}

	if err := out.LoadStats(c.q.GetCampaignStats); err != nil {
		c.log.Printf("error fetching campaign stats: %v", err)
		return models.Campaign{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}

	c.log.Println("Campaign stats loaded successfully.")
	return out[0], nil
}
*/

// GetCampaignForPreview retrieves a campaign with a template body.
func (c *Core) GetCampaignForPreview(id, tplID int, authID string) (models.Campaign, error) {
	var out models.Campaign
	out.AuthID = authID
	if err := c.q.GetCampaignForPreview.Get(&out, id, tplID, out.AuthID); err != nil {
		if err == sql.ErrNoRows {
			return models.Campaign{}, echo.NewHTTPError(http.StatusBadRequest,
				c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.campaign}"))
		}

		c.log.Printf("error fetching campaign: %v", err)
		return models.Campaign{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}

	return out, nil
}

// GetArchivedCampaigns retrieves campaigns with a template body.
func (c *Core) GetArchivedCampaigns(offset, limit int, authID string) (models.Campaigns, int, error) {
	var out models.Campaigns
	if err := c.q.GetArchivedCampaigns.Select(&out, offset, limit, campaignTplArchive, authID); err != nil {
		c.log.Printf("error fetching public campaigns: %v", err)
		return models.Campaigns{}, 0, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}

	total := 0
	if len(out) > 0 {
		total = out[0].Total
	}

	return out, total, nil
}

// CreateCampaign creates a new campaign with conditions based on messenger type (email, sms, voice).

// CreateCampaign creates a new campaign, ensuring required fields like template_id and list_ids are provided.
func (c *Core) CreateCampaign(o models.Campaign, listIDs []int, mediaIDs []int, authID string, voiceOption string) (models.Campaign, error) {
	// Generate a UUID for the campaign
	uu, err := uuid.NewV4()
	if err != nil {
		c.log.Printf("error generating UUID: %v", err)
		return models.Campaign{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUUID", "error", err.Error()))
	}

	// Set AuthID
	o.AuthID = authID

	if len(listIDs) == 0 {
		return models.Campaign{}, echo.NewHTTPError(http.StatusBadRequest, "List ID is required.")
	}

	// Apply default values based on campaign type and voice option
	switch o.Messenger {
	case "voice":
		switch voiceOption {
		case "template":
			// Template-based voice campaign requires template ID and list ID
			// Template ID and list IDs are already validated as required, so no defaults here
		case "music":
			// Music-based voice campaign requires music ID
			if o.MusicID == "" {
				return models.Campaign{}, echo.NewHTTPError(http.StatusBadRequest, "Music ID is required for music-based voice campaigns.")
			}
		case "text-to-speech":
			// Text-to-speech requires body, vendor, loop, voice, and language
			if o.Body == "" {
				return models.Campaign{}, echo.NewHTTPError(http.StatusBadRequest, "Body is required for TTS campaigns.")
			}
			if o.Vendor == "" {
				o.Vendor = "aws"
			}
			if o.Loop == 0 {
				o.Loop = 1 // Default loop count
			}
			if o.Voice == "" {
				o.Voice = "woman"
			}
			if o.Language == "" {
				o.Language = "en-US"
			}
		}
	case "email":
		// Email campaign requires subject, body, from_email
		if o.Subject == "" {
			return models.Campaign{}, echo.NewHTTPError(http.StatusBadRequest, "Subject is required for email campaigns.")
		}
		if o.Body == "" {
			return models.Campaign{}, echo.NewHTTPError(http.StatusBadRequest, "Body is required for email campaigns.")
		}
		if o.FromEmail == "" {
			return models.Campaign{}, echo.NewHTTPError(http.StatusBadRequest, "From email is required for email campaigns.")
		}
	case "sms":
		// SMS campaign requires from, body
		if o.FromPhone == "" {
			return models.Campaign{}, echo.NewHTTPError(http.StatusBadRequest, "From email is required for SMS campaigns.")
		}
		if o.Body == "" {
			return models.Campaign{}, echo.NewHTTPError(http.StatusBadRequest, "Body is required for SMS campaigns.")
		}
	}

	var out1 types.JSONText
	if err := c.q.CheckInsertCampaignValidData.Get(&out1, o.Name, o.AuthID, o.TemplateID, pq.Array(mediaIDs), pq.Array(listIDs)); err != nil {
		return models.Campaign{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "dashboard stats", "error", pqErrMsg(err)))
	}
	var validationData map[string]interface{}
	if err := json.Unmarshal(out1, &validationData); err != nil {
		return models.Campaign{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.T("globals.messages.errorParsingResponse"))
	}

	if validationData["duplicateCount"].(float64) > 0 {
		return models.Campaign{}, echo.NewHTTPError(http.StatusBadRequest, c.i18n.Ts("globals.messages.invalidFields", "name", "Name"))
	}
	if o.TemplateID != 0 && validationData["templateCount"].(float64) < 1 {
		return models.Campaign{}, echo.NewHTTPError(http.StatusBadRequest, c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.template}"))
	}
	if len(mediaIDs) > 0 && validationData["mediaCount"].(float64) < 1 {
		return models.Campaign{}, echo.NewHTTPError(http.StatusBadRequest, c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.media}"))
	}
	if validationData["listCount"].(float64) < 1 {
		return models.Campaign{}, echo.NewHTTPError(http.StatusBadRequest, c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.list}"))
	}

	// Insert the campaign into the database
	var newID int
	if err := c.q.CreateCampaign.Get(&newID,
		uu,                                    // campaign uuid
		o.Type,                                // campaign type
		o.Name,                                // campaign name
		o.Subject,                             // campaign subject
		o.FromEmail,                           // from email
		o.Body,                                // body (text for TTS or email campaigns)
		o.AltBody,                             // alternative body
		o.ContentType,                         // content type (email campaigns)
		o.SendAt,                              // send at date/time
		o.Headers,                             // custom headers
		pq.StringArray(normalizeTags(o.Tags)), // campaign tags
		o.Messenger,                           // messenger type (voice, email, sms)
		o.TemplateID,                          // template id (required)
		pq.Array(listIDs),                     // list of IDs (required)
		o.Archive,                             // archive flag
		o.ArchiveSlug,                         // archive slug
		o.ArchiveTemplateID,                   // archive template id
		o.ArchiveMeta,                         // archive metadata
		pq.Array(mediaIDs),                    // media ids for campaign
		o.AuthID,                              // auth id
		o.MusicID,                             // music id for voice campaigns
		o.Vendor,                              // vendor for TTS campaigns
		o.Loop,                                // loop count for TTS campaigns
		o.Voice,                               // voice for TTS campaigns
		o.Language,
		o.FromPhone); // language for TTS campaigns
	err != nil {
		if err == sql.ErrNoRows {
			return models.Campaign{}, echo.NewHTTPError(http.StatusBadRequest, c.i18n.T("campaigns.notFound"))
		}

		c.log.Printf("error creating campaign: %v", err)
		return models.Campaign{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}

	// Fetch the newly created campaign
	out, err := c.GetCampaign(newID, "", "", authID)
	if err != nil {
		return models.Campaign{}, err
	}

	return out, nil
}

// UpdateCampaign updates a campaign.
func (c *Core) UpdateCampaign(id int, o models.Campaign, listIDs []int, mediaIDs []int, sendLater bool, authid string) (models.Campaign, error) {

	o.AuthID = authid

	var out1 types.JSONText
	if err := c.q.CheckUpdateCampaignValidData.Get(&out1, o.Name, o.AuthID, o.TemplateID, pq.Array(mediaIDs), pq.Array(listIDs), id); err != nil {
		return models.Campaign{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "dashboard stats", "error", pqErrMsg(err)))
	}
	var validationData map[string]interface{}
	if err := json.Unmarshal(out1, &validationData); err != nil {
		return models.Campaign{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.T("globals.messages.errorParsingResponse"))
	}

	if validationData["duplicateCount"].(float64) > 0 {
		return models.Campaign{}, echo.NewHTTPError(http.StatusBadRequest, c.i18n.Ts("globals.messages.invalidFields", "name", "Name"))
	}
	if validationData["templateCount"].(float64) < 1 {
		return models.Campaign{}, echo.NewHTTPError(http.StatusBadRequest, c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.template}"))
	}
	if len(mediaIDs) > 0 && validationData["mediaCount"].(float64) < 1 {
		return models.Campaign{}, echo.NewHTTPError(http.StatusBadRequest, c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.media}"))
	}
	if validationData["listCount"].(float64) < 1 {
		return models.Campaign{}, echo.NewHTTPError(http.StatusBadRequest, c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.list}"))
	}

	_, err := c.q.UpdateCampaign.Exec(id,
		o.Name,
		o.Subject,
		o.FromEmail,
		o.Body,
		o.AltBody,
		o.ContentType,
		o.SendAt,
		sendLater,
		o.Headers,
		pq.StringArray(normalizeTags(o.Tags)),
		o.Messenger,
		o.TemplateID,
		pq.Array(listIDs),
		o.Archive,
		o.ArchiveSlug,
		o.ArchiveTemplateID,
		o.ArchiveMeta,
		pq.Array(mediaIDs),
		o.AuthID)
	if err != nil {
		c.log.Printf("error updating campaign: %v", err)
		return models.Campaign{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}

	out, err := c.GetCampaign(o.ID, "", "", o.AuthID)
	if err != nil {
		return models.Campaign{}, err
	}

	return out, nil
}

// UpdateCampaignStatus updates a campaign's status, eg: draft to running.
func (c *Core) UpdateCampaignStatus(id int, status string, authID string) (models.Campaign, error) {
	cm, err := c.GetCampaign(id, "", "", authID)
	if err != nil {
		return models.Campaign{}, err
	}

	errMsg := ""
	switch status {
	case models.CampaignStatusDraft:
		if cm.Status != models.CampaignStatusScheduled {
			errMsg = c.i18n.T("campaigns.onlyScheduledAsDraft")
		}
	case models.CampaignStatusScheduled:
		if cm.Status != models.CampaignStatusDraft {
			errMsg = c.i18n.T("campaigns.onlyDraftAsScheduled")
		}
		if !cm.SendAt.Valid {
			errMsg = c.i18n.T("campaigns.needsSendAt")
		}

	case models.CampaignStatusRunning:
		if cm.Status != models.CampaignStatusPaused && cm.Status != models.CampaignStatusDraft {
			errMsg = c.i18n.T("campaigns.onlyPausedDraft")
		}
	case models.CampaignStatusPaused:
		if cm.Status != models.CampaignStatusRunning {
			errMsg = c.i18n.T("campaigns.onlyActivePause")
		}
	case models.CampaignStatusCancelled:
		if cm.Status != models.CampaignStatusRunning && cm.Status != models.CampaignStatusPaused {
			errMsg = c.i18n.T("campaigns.onlyActiveCancel")
		}
	}

	if len(errMsg) > 0 {
		return models.Campaign{}, echo.NewHTTPError(http.StatusBadRequest, errMsg)
	}

	res, err := c.q.UpdateCampaignStatus.Exec(cm.ID, status, authID)
	if err != nil {
		c.log.Printf("error updating campaign status: %v", err)

		return models.Campaign{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}

	if n, _ := res.RowsAffected(); n == 0 {
		return models.Campaign{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}

	cm.Status = status
	return cm, nil
}

// UpdateCampaignArchive updates a campaign's archive properties.
func (c *Core) UpdateCampaignArchive(id int, enabled bool, tplID int, meta models.JSON, archiveSlug string, authID string) error {
	if _, err := c.q.UpdateCampaignArchive.Exec(id, enabled, archiveSlug, tplID, meta, authID); err != nil {
		c.log.Printf("error updating campaign: %v", err)

		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}

	return nil
}

// DeleteCampaign deletes a campaign.
func (c *Core) DeleteCampaign(id int, authID string) error {
	res, err := c.q.DeleteCampaign.Exec(id, authID)
	if err != nil {
		c.log.Printf("error deleting campaign: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.campaign}", "error", pqErrMsg(err)))

	}

	if n, _ := res.RowsAffected(); n == 0 {
		return echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.campaign}"))
	}

	return nil
}

// GetRunningCampaignStats returns the progress stats of running campaigns.
func (c *Core) GetRunningCampaignStats(authID string) ([]models.CampaignStats, error) {
	out := []models.CampaignStats{}
	if err := c.q.GetCampaignStatus.Select(&out, models.CampaignStatusRunning, authID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		c.log.Printf("error fetching campaign stats: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	} else if len(out) == 0 {
		return nil, nil
	}

	return out, nil
}

func (c *Core) GetCampaignAnalyticsCounts(campIDs []int, typ, fromDate, toDate string, authID string) ([]models.CampaignAnalyticsCount, error) {
	// Pick campaign view counts or click counts.
	var stmt *sqlx.Stmt
	switch typ {
	case "views":
		stmt = c.q.GetCampaignViewCounts
	case "clicks":
		stmt = c.q.GetCampaignClickCounts
	case "bounces":
		stmt = c.q.GetCampaignBounceCounts
	default:
		return nil, echo.NewHTTPError(http.StatusBadRequest, c.i18n.T("globals.messages.invalidData"))
	}

	if !strHasLen(fromDate, 10, 30) || !strHasLen(toDate, 10, 30) {
		return nil, echo.NewHTTPError(http.StatusBadRequest, c.i18n.T("analytics.invalidDates"))
	}

	out := []models.CampaignAnalyticsCount{}
	if err := stmt.Select(&out, pq.Array(campIDs), fromDate, toDate, authID); err != nil {
		c.log.Printf("error fetching campaign %s: %v", typ, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.analytics}", "error", pqErrMsg(err)))
	}

	return out, nil
}

// GetCampaignAnalyticsLinks returns link click analytics for the given campaign IDs.
func (c *Core) GetCampaignAnalyticsLinks(campIDs []int, typ, fromDate, toDate string) ([]models.CampaignAnalyticsLink, error) {
	out := []models.CampaignAnalyticsLink{}
	if err := c.q.GetCampaignLinkCounts.Select(&out, pq.Array(campIDs), fromDate, toDate); err != nil {
		c.log.Printf("error fetching campaign %s: %v", typ, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.analytics}", "error", pqErrMsg(err)))
	}

	return out, nil
}

// RegisterCampaignView registers a subscriber's view on a campaign.
func (c *Core) RegisterCampaignView(campUUID, subUUID string, authID string) error {
	if _, err := c.q.RegisterCampaignView.Exec(campUUID, subUUID, authID); err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Column == "campaign_id" {
			return nil
		}

		c.log.Printf("error registering campaign view: %s", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}
	return nil
}

// RegisterCampaignLinkClick registers a subscriber's link click on a campaign.
func (c *Core) RegisterCampaignLinkClick(linkUUID, campUUID, subUUID string, authID string) (string, error) {
	var url string
	if err := c.q.RegisterLinkClick.Get(&url, linkUUID, campUUID, subUUID, authID); err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Column == "link_id" {
			return "", echo.NewHTTPError(http.StatusBadRequest, c.i18n.Ts("public.invalidLink"))
		}

		c.log.Printf("error registering link click: %s", err)
		return "", echo.NewHTTPError(http.StatusInternalServerError, c.i18n.Ts("public.errorProcessingRequest"))
	}

	return url, nil
}

// DeleteCampaignViews deletes campaign views older than a given date.
func (c *Core) DeleteCampaignViews(before time.Time, authID string) error {
	if _, err := c.q.DeleteCampaignViews.Exec(before, authID); err != nil {
		c.log.Printf("error deleting campaign views: %s", err)
		return echo.NewHTTPError(http.StatusInternalServerError, c.i18n.Ts("public.errorProcessingRequest"))
	}

	return nil
}

// DeleteCampaignLinkClicks deletes campaign views older than a given date.
func (c *Core) DeleteCampaignLinkClicks(before time.Time, authID string) error {
	if _, err := c.q.DeleteCampaignLinkClicks.Exec(before, authID); err != nil {
		c.log.Printf("error deleting campaign link clicks: %s", err)
		return echo.NewHTTPError(http.StatusInternalServerError, c.i18n.Ts("public.errorProcessingRequest"))
	}

	return nil
}
