package main

import (
	"database/sql"
	"encoding/json"

	"github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx/types"
	"github.com/knadh/listmonk/internal/media"
	"github.com/knadh/listmonk/models"
	"github.com/lib/pq"
	null "gopkg.in/volatiletech/null.v6"
)

type Store interface {
	GetDashboardCharts(c *types.JSONText) error
	GetDashboardCounts(c *types.JSONText) error

	InsertSubscriber(id *int, uuid, email, name, status string, attribs models.SubscriberAttribs, lists pq.Int64Array, listUUIDs pq.StringArray, subStatus string) error
	GetSubscriber(s *models.Subscribers, id int, uuid, email string) error
	GetSubscribersByEmails(s *models.Subscribers, emails pq.StringArray) error
	GetSubscriberLists(lists *[]models.List, id int, uuid string, listIDs pq.Int64Array, listUUIDs pq.StringArray, subStatus, optin string) error
	SubscriberExists(exists *bool, id int, uuid string) error
	UpdateSubscriber(id int64, email, name, status string, attribs json.RawMessage, lists pq.Int64Array) error
	BlocklistSubscribers(list pq.Int64Array) error
	AddSubscribersToLists(subscribers, lists pq.Int64Array) error
	DeleteSubscriptions(subscribers, lists pq.Int64Array) error
	UnsubscribeSubscribersFromLists(subscribers, lists pq.Int64Array) error
	ConfirmSubscriptionOptin(subUUID string, listUUIDs pq.StringArray) error
	DeleteSubscribers(subIDs pq.Int64Array, subUUIDs pq.StringArray) error
	Unsubscribe(campaignUUID, subUUID string, blocklist bool) error
	ExportSubscriberData(data interface{}, id int64, uuidOrNil interface{}) error
	CreateList(list *int, uuid interface{}, name, listType, optin string, tags pq.StringArray) error
	GetLists(lists *[]models.List, listType string) error
	GetListsByOptin(lists *[]models.List, optin string, listIDs pq.Int64Array, listUUIDs pq.StringArray) error
	UpdateList(id int, name, listType, optin string, tags pq.StringArray) (sql.Result, error)
	DeleteLists(ids pq.Int64Array) error
	CreateCampaign(id *int, uuid uuid.UUID, campaignType, name, subject, fromEmail, body string, altBody null.String, contentType string, sendAt null.Time, tags pq.StringArray, messenger string, templateID int, listIDs pq.Int64Array) error
	GetCampaign(campaign *models.Campaign, id int, uuid *string) error
	GetCampaignForPreview(campaign *models.Campaign, id int) error
	GetCampaignStatus(stats interface{}, status string) error
	NextCampaigns(c *[]*models.Campaign, excludeIDs pq.Int64Array) error
	NextCampaignSubscribers(subs *[]models.Subscriber, campID, limit int) error
	UpdateCampaign(id int, name, subject, fromEmail, body string, altBody null.String, contentType string, sendAt null.Time, sendLater bool, tags pq.StringArray, messenger string, templateID int, listIDs pq.Int64Array) error
	UpdateCampaignStatus(id int, status string) (sql.Result, error)
	RegisterCampaignView(campUUID, subUUID string) error
	DeleteCampaign(id int) error
	InsertMedia(id uuid.UUID, filename, thumbnailFilename, mediaProvider string) error
	GetMedia(media *[]media.Media, provider string) error
	DeleteMedia(media *media.Media, id int) error
	CreateTemplate(id *int, name, body string) error
	GetTemplates(out *[]models.Template, id int, noBody bool) error
	UpdateTemplate(id int, name, body string) (sql.Result, error)
	SetDefaultTemplate(id int) error
	DeleteTemplate(delID *int, id int) error
	CreateLink(out *string, id uuid.UUID, url string) error
	RegisterLinkClick(url *string, linkUUID, campUUID, subUUID string) error
	UpdateSettings(settings []byte) error
	QueryLists(listID, offset, limit int, orderBy, order string) ([]models.List, error)
	QueryCampaigns(id, offset, limit int, status []string, query, orderBy, order string) ([]models.Campaign, error)
	UpsertSubscriber(uuid uuid.UUID, email, name string, attribs models.SubscriberAttribs, listIDs pq.Int64Array, overwrite bool, tx *sql.Tx) error
	UpsertBlocklistSubscriber(uu uuid.UUID, email, name string, attribs models.SubscriberAttribs, tx *sql.Tx) error
	UpdateListsDate(listIDs pq.Int64Array, tx *sql.Tx) error
	GetCampaignStats(campaignIDs []int) ([]models.CampaignMeta, error)
	GetSettings() (json.RawMessage, error)
	GetSubscriberListsLazy(subscriberIDs []int) ([]SubscriberLists, error)
	QuerySubscribers(listIDs pq.Int64Array, query, orderBy, order string, offset, limit int) ([]models.Subscriber, error)
	QuerySubscribersForExport(query string, listIDs pq.Int64Array, batchSize int, onFetch func([]models.SubscriberExport) error) error
	DeleteSubscriptionsByQuery(exp string, listIDs pq.Int64Array) error
	BlocklistSubscribersByQuery(exp string, listIDs pq.Int64Array) error
	DeleteSubscribersByQuery(exp string, listIDs pq.Int64Array) error
	UnsubscribeSubscribersFromListsByQuery(exp string, listIDs, targetListIDs pq.Int64Array) error
	AddSubscribersToListsByQuery(exp string, listIDs, targetListIDs pq.Int64Array) error
}
