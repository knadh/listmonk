package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/types"
	"github.com/knadh/listmonk/internal/media"
	"github.com/knadh/listmonk/models"
	"github.com/lib/pq"
	null "gopkg.in/volatiletech/null.v6"
)

// Queries contains all prepared SQL queries.
type Queries struct {
	db *sqlx.DB

	GetDashboardChartsStmt *sqlx.Stmt `query:"get-dashboard-charts"`
	GetDashboardCountsStmt *sqlx.Stmt `query:"get-dashboard-counts"`

	InsertSubscriberStmt                *sqlx.Stmt `query:"insert-subscriber"`
	UpsertSubscriberStmt                *sqlx.Stmt `query:"upsert-subscriber"`
	UpsertBlocklistSubscriberStmt       *sqlx.Stmt `query:"upsert-blocklist-subscriber"`
	GetSubscriberStmt                   *sqlx.Stmt `query:"get-subscriber"`
	GetSubscribersByEmailsStmt          *sqlx.Stmt `query:"get-subscribers-by-emails"`
	GetSubscriberListsStmt              *sqlx.Stmt `query:"get-subscriber-lists"`
	GetSubscriberListsLazyStmt          *sqlx.Stmt `query:"get-subscriber-lists-lazy"`
	SubscriberExistsStmt                *sqlx.Stmt `query:"subscriber-exists"`
	UpdateSubscriberStmt                *sqlx.Stmt `query:"update-subscriber"`
	BlocklistSubscribersStmt            *sqlx.Stmt `query:"blocklist-subscribers"`
	AddSubscribersToListsStmt           *sqlx.Stmt `query:"add-subscribers-to-lists"`
	DeleteSubscriptionsStmt             *sqlx.Stmt `query:"delete-subscriptions"`
	ConfirmSubscriptionOptinStmt        *sqlx.Stmt `query:"confirm-subscription-optin"`
	UnsubscribeSubscribersFromListsStmt *sqlx.Stmt `query:"unsubscribe-subscribers-from-lists"`
	DeleteSubscribersStmt               *sqlx.Stmt `query:"delete-subscribers"`
	UnsubscribeStmt                     *sqlx.Stmt `query:"unsubscribe"`
	ExportSubscriberDataStmt            *sqlx.Stmt `query:"export-subscriber-data"`

	// Non-prepared arbitrary subscriber queries.
	QuerySubscribersStmt                       string `query:"query-subscribers"`
	QuerySubscribersForExportStmt              string `query:"query-subscribers-for-export"`
	QuerySubscribersTplStmt                    string `query:"query-subscribers-template"`
	DeleteSubscribersByQueryStmt               string `query:"delete-subscribers-by-query"`
	AddSubscribersToListsByQueryStmt           string `query:"add-subscribers-to-lists-by-query"`
	BlocklistSubscribersByQueryStmt            string `query:"blocklist-subscribers-by-query"`
	DeleteSubscriptionsByQueryStmt             string `query:"delete-subscriptions-by-query"`
	UnsubscribeSubscribersFromListsByQueryStmt string `query:"unsubscribe-subscribers-from-lists-by-query"`

	CreateListStmt      *sqlx.Stmt `query:"create-list"`
	QueryListsStmt      string     `query:"query-lists"`
	GetListsStmt        *sqlx.Stmt `query:"get-lists"`
	GetListsByOptinStmt *sqlx.Stmt `query:"get-lists-by-optin"`
	UpdateListStmt      *sqlx.Stmt `query:"update-list"`
	UpdateListsDateStmt *sqlx.Stmt `query:"update-lists-date"`
	DeleteListsStmt     *sqlx.Stmt `query:"delete-lists"`

	CreateCampaignStmt           *sqlx.Stmt `query:"create-campaign"`
	QueryCampaignsStmt           string     `query:"query-campaigns"`
	GetCampaignStmt              *sqlx.Stmt `query:"get-campaign"`
	GetCampaignForPreviewStmt    *sqlx.Stmt `query:"get-campaign-for-preview"`
	GetCampaignStatsStmt         *sqlx.Stmt `query:"get-campaign-stats"`
	GetCampaignStatusStmt        *sqlx.Stmt `query:"get-campaign-status"`
	NextCampaignsStmt            *sqlx.Stmt `query:"next-campaigns"`
	NextCampaignSubscribersStmt  *sqlx.Stmt `query:"next-campaign-subscribers"`
	GetOneCampaignSubscriberStmt *sqlx.Stmt `query:"get-one-campaign-subscriber"`
	UpdateCampaignStmt           *sqlx.Stmt `query:"update-campaign"`
	UpdateCampaignStatusStmt     *sqlx.Stmt `query:"update-campaign-status"`
	UpdateCampaignCountsStmt     *sqlx.Stmt `query:"update-campaign-counts"`
	RegisterCampaignViewStmt     *sqlx.Stmt `query:"register-campaign-view"`
	DeleteCampaignStmt           *sqlx.Stmt `query:"delete-campaign"`

	InsertMediaStmt *sqlx.Stmt `query:"insert-media"`
	GetMediaStmt    *sqlx.Stmt `query:"get-media"`
	DeleteMediaStmt *sqlx.Stmt `query:"delete-media"`

	CreateTemplateStmt     *sqlx.Stmt `query:"create-template"`
	GetTemplatesStmt       *sqlx.Stmt `query:"get-templates"`
	UpdateTemplateStmt     *sqlx.Stmt `query:"update-template"`
	SetDefaultTemplateStmt *sqlx.Stmt `query:"set-default-template"`
	DeleteTemplateStmt     *sqlx.Stmt `query:"delete-template"`

	CreateLinkStmt        *sqlx.Stmt `query:"create-link"`
	RegisterLinkClickStmt *sqlx.Stmt `query:"register-link-click"`

	GetSettingsStmt    *sqlx.Stmt `query:"get-settings"`
	UpdateSettingsStmt *sqlx.Stmt `query:"update-settings"`

	// GetStats *sqlx.Stmt `query:"get-stats"`
}

func (q *Queries) GetDashboardCharts(c *types.JSONText) error {
	return q.GetDashboardChartsStmt.Get(c)
}

func (q *Queries) GetDashboardCounts(c *types.JSONText) error {
	return q.GetDashboardCountsStmt.Get(c)
}

func (q *Queries) InsertSubscriber(
	id *int,
	uuid, email, name, status string,
	attribs models.SubscriberAttribs,
	lists pq.Int64Array,
	listUUIDs pq.StringArray,
	subStatus string,
) error {
	return q.InsertSubscriberStmt.Get(&id, uuid, email, name, status, attribs, lists, listUUIDs, subStatus)
}

func (q *Queries) GetSubscriber(s *models.Subscribers, id int, uuid, email string) error {
	return q.GetSubscriberStmt.Select(s, id, uuid, email)
}

func (q *Queries) GetSubscribersByEmails(s *models.Subscribers, emails pq.StringArray) error {
	return q.GetSubscribersByEmailsStmt.Select(s, emails)
}

func (q *Queries) GetSubscriberLists(
	lists *[]models.List,
	id int, uuid string,
	listIDs pq.Int64Array,
	listUUIDs pq.StringArray,
	subStatus, optin string,
) error {
	return q.GetSubscriberListsStmt.Select(lists, id, uuid, listIDs, listUUIDs, subStatus, optin)
}

func (q *Queries) SubscriberExists(
	exists *bool,
	id int, uuid string,
) error {
	return q.SubscriberExistsStmt.Select(exists, id, uuid)
}

func (q *Queries) UpdateSubscriber(
	id int64,
	email, name, status string,
	attribs json.RawMessage,
	lists pq.Int64Array,
) error {
	_, err := q.UpdateSubscriberStmt.Exec(id, email, name, status, attribs, lists)
	return err
}

func (q *Queries) BlocklistSubscribers(
	list pq.Int64Array,
) error {
	_, err := q.BlocklistSubscribersStmt.Exec(list)
	return err
}

func (q *Queries) AddSubscribersToLists(
	subscribers, lists pq.Int64Array,
) error {
	_, err := q.BlocklistSubscribersStmt.Exec(subscribers, lists)
	return err
}

func (q *Queries) DeleteSubscriptions(
	subscribers, lists pq.Int64Array,
) error {
	_, err := q.DeleteSubscriptionsStmt.Exec(subscribers, lists)
	return err
}

func (q *Queries) UnsubscribeSubscribersFromLists(
	subscribers, lists pq.Int64Array,
) error {
	_, err := q.UnsubscribeSubscribersFromListsStmt.Exec(subscribers, lists)
	return err
}

func (q *Queries) ConfirmSubscriptionOptin(
	subUUID string, listUUIDs pq.StringArray,
) error {
	_, err := q.ConfirmSubscriptionOptinStmt.Exec(subUUID, listUUIDs)
	return err
}

func (q *Queries) DeleteSubscribers(
	subIDs pq.Int64Array, subUUIDs pq.StringArray,
) error {
	_, err := q.DeleteSubscribersStmt.Exec(subIDs, subUUIDs)
	return err
}

func (q *Queries) Unsubscribe(
	campaignUUID, subUUID string,
	blocklist bool,
) error {
	_, err := q.UnsubscribeStmt.Exec(campaignUUID, subUUID, blocklist)
	return err
}

func (q *Queries) ExportSubscriberData(
	data interface{},
	id int64,
	uuidOrNil interface{},
) error {
	return q.ExportSubscriberDataStmt.Get(data, id, uuidOrNil)
}

func (q *Queries) CreateList(
	list *int,
	uuid interface{},
	name, listType, optin string,
	tags pq.StringArray,
) error {
	return q.CreateListStmt.Get(list, uuid, name, listType, optin, tags)
}

func (q *Queries) GetLists(
	lists *[]models.List,
	listType string,
) error {
	return q.GetListsStmt.Select(lists, listType)
}

func (q *Queries) GetListsByOptin(
	lists *[]models.List,
	optin string,
	listIDs pq.Int64Array,
	listUUIDs pq.StringArray,
) error {
	return q.GetListsByOptinStmt.Select(lists, optin, listIDs, listUUIDs)
}

func (q *Queries) UpdateList(
	id int,
	name, listType, optin string,
	tags pq.StringArray,
) (sql.Result, error) {
	return q.UpdateListStmt.Exec(id, name, listType, optin, tags)
}

func (q *Queries) DeleteLists(ids pq.Int64Array) error {
	_, err := q.DeleteListsStmt.Exec(ids)
	return err
}

func (q *Queries) CreateCampaign(
	newID *int,
	uuid uuid.UUID,
	campaignType, name, subject, fromEmail, body string,
	altBody null.String,
	contentType string,
	sendAt null.Time,
	tags pq.StringArray,
	messenger string,
	templateID int,
	listIDs pq.Int64Array,
) error {
	return q.CreateCampaignStmt.Get(newID, uuid, campaignType, name, subject, fromEmail, body, altBody, contentType, sendAt, tags, messenger, templateID, listIDs)
}

func (q *Queries) GetCampaign(campaign *models.Campaign, id int, uuid *string) error {
	_, err := q.GetCampaignStmt.Exec(campaign, id, uuid)
	return err
}

func (q *Queries) GetCampaignForPreview(campaign *models.Campaign, id int) error {
	_, err := q.GetCampaignForPreviewStmt.Exec(campaign, id)
	return err
}

func (q *Queries) GetCampaignStatus(stats interface{}, status string) error {
	return q.GetCampaignStatusStmt.Select(stats, status)
}

func (q *Queries) NextCampaigns(c *[]*models.Campaign, excludeIDs pq.Int64Array) error {
	return q.NextCampaignsStmt.Select(c, excludeIDs)
}

func (q *Queries) NextCampaignSubscribers(subs *[]models.Subscriber, campID, limit int) error {
	return q.NextCampaignSubscribersStmt.Select(subs, campID, limit)
}

func (q *Queries) UpdateCampaign(
	id int,
	name, subject, fromEmail, body string,
	altBody null.String,
	contentType string,
	sendAt null.Time,
	sendLater bool,
	tags pq.StringArray,
	messenger string,
	templateID int,
	listIDs pq.Int64Array,
) error {
	_, err := q.UpdateCampaignStmt.Exec(id, name, subject, fromEmail, body, altBody, contentType, sendAt, sendLater, tags, messenger, templateID, listIDs)
	return err
}

func (q *Queries) UpdateCampaignStatus(
	id int,
	status string,
) (sql.Result, error) {
	return q.UpdateCampaignStatusStmt.Exec(id, status)
}

func (q *Queries) RegisterCampaignView(campUUID, subUUID string) error {
	_, err := q.RegisterCampaignViewStmt.Exec(campUUID, subUUID)
	return err
}

func (q *Queries) DeleteCampaign(id int) error {
	_, err := q.DeleteCampaignStmt.Exec(id)
	return err
}

func (q *Queries) InsertMedia(id uuid.UUID, filename, thumbnailFilename, mediaProvider string) error {
	_, err := q.InsertMediaStmt.Exec(id, filename, thumbnailFilename, mediaProvider)
	return err
}

func (q *Queries) GetMedia(media *[]media.Media, provider string) error {
	return q.GetMediaStmt.Select(media, provider)
}

func (q *Queries) DeleteMedia(media *media.Media, id int) error {
	return q.DeleteMediaStmt.Get(media, id)
}

func (q *Queries) CreateTemplate(id *int, name, body string) error {
	return q.CreateTemplateStmt.Get(id, name, body)
}

func (q *Queries) GetTemplates(out *[]models.Template, id int, noBody bool) error {
	return q.GetTemplatesStmt.Select(out, id, noBody)
}

func (q *Queries) UpdateTemplate(id int, name, body string) (sql.Result, error) {
	return q.UpdateTemplateStmt.Exec(id, name, body)
}

func (q *Queries) SetDefaultTemplate(id int) error {
	_, err := q.SetDefaultTemplateStmt.Exec(id)
	return err
}

func (q *Queries) DeleteTemplate(delID *int, id int) error {
	_, err := q.DeleteTemplateStmt.Exec(delID, id)
	return err
}

func (q *Queries) CreateLink(out *string, id uuid.UUID, url string) error {
	return q.CreateLinkStmt.Get(out, id, url)
}

func (q *Queries) RegisterLinkClick(url *string, linkUUID, campUUID, subUUID string) error {
	return q.RegisterLinkClickStmt.Get(url, linkUUID, campUUID, subUUID)
}

func (q *Queries) UpdateSettings(settings []byte) error {
	_, err := q.UpdateSettingsStmt.Exec(settings)
	return err
}

func (q *Queries) QueryLists(listID, offset, limit int, orderBy, order string) ([]models.List, error) {
	query := fmt.Sprintf(q.QueryListsStmt, orderBy, order)

	var results []models.List
	if err := db.Select(&results, query, listID, offset, limit); err != nil {
		return nil, err
	}

	return results, nil
}

func (q *Queries) QueryCampaigns(id, offset, limit int, status []string, query, orderBy, order string) ([]models.Campaign, error) {
	stmt := fmt.Sprintf(q.QueryCampaignsStmt, orderBy, order)

	var results []models.Campaign
	if err := db.Select(&results, stmt, id, pq.StringArray(status), query, offset, limit); err != nil {
		return nil, err
	}

	return results, nil
}

func (q *Queries) UpsertSubscriber(
	uuid uuid.UUID,
	email, name string,
	attribs models.SubscriberAttribs,
	listIDs pq.Int64Array,
	overwrite bool,
	tx *sql.Tx,
) error {
	stmt := q.UpsertSubscriberStmt.Stmt
	if tx != nil {
		stmt = tx.Stmt(stmt)
	}
	_, err := q.UpsertSubscriberStmt.Exec(uuid, email, name, attribs, listIDs, overwrite)
	return err
}

func (q *Queries) UpsertBlocklistSubscriber(
	uu uuid.UUID,
	email, name string,
	attribs models.SubscriberAttribs,
	tx *sql.Tx,
) error {
	stmt := q.UpsertBlocklistSubscriberStmt.Stmt
	if tx != nil {
		stmt = tx.Stmt(stmt)
	}
	_, err := stmt.Exec(uu, email, name, attribs)
	return err
}

func (q *Queries) UpdateListsDate(listIDs pq.Int64Array, tx *sql.Tx) error {
	stmt := q.UpdateListsDateStmt.Stmt
	if tx != nil {
		stmt = tx.Stmt(stmt)
	}
	_, err := stmt.Exec(listIDs)
	return err
}

func (q *Queries) GetCampaignStats(campaignIDs []int) ([]models.CampaignMeta, error) {
	var meta []models.CampaignMeta
	if err := q.GetCampaignStatsStmt.Select(&meta, pq.Array(campaignIDs)); err != nil {
		return meta, err
	}

	if len(campaignIDs) != len(meta) {
		return meta, errors.New("campaign stats count does not match")
	}

	return meta, nil
}

func (q *Queries) GetSettings() (json.RawMessage, error) {
	var s types.JSONText
	if err := q.GetSettingsStmt.Get(&s); err != nil {
		return nil, fmt.Errorf("error reading settings from DB: %s", pqErrMsg(err))
	}
	return json.RawMessage(s), nil
}

type SubscriberLists struct {
	SubscriberID int            `db:"subscriber_id"`
	Lists        types.JSONText `db:"lists"`
}

func (q *Queries) GetSubscriberListsLazy(subscriberIDs []int) ([]SubscriberLists, error) {
	var sl []SubscriberLists
	if err := q.GetSubscriberListsLazyStmt.Select(&sl, pq.Array(subscriberIDs)); err != nil {
		return sl, err
	}

	if len(subscriberIDs) != len(sl) {
		return sl, errors.New("campaign stats count does not match")
	}

	return sl, nil
}

func (q *Queries) QuerySubscribers(listIDs pq.Int64Array, query, orderBy, order string, offset, limit int) ([]models.Subscriber, error) {
	// Create a readonly transaction to prevent mutations.
	tx, err := q.db.BeginTxx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("error preparing subscriber query: %w", err)
	}
	defer tx.Rollback()

	// There's an arbitrary query condition.
	cond := ""
	if query != "" {
		cond = " AND " + query
	}

	stmt := fmt.Sprintf(q.QuerySubscribersStmt, cond, orderBy, order)

	// Run the query. stmt is the raw SQL query.
	var results []models.Subscriber
	if err := tx.Select(&results, stmt, listIDs, offset, limit); err != nil {
		return results, err
	}

	return results, nil
}

func (q *Queries) QuerySubscribersForExport(
	query string,
	listIDs pq.Int64Array,
	batchSize int,
	onFetch func([]models.SubscriberExport) error,
) error {
	// There's an arbitrary query condition.
	cond := ""
	if query != "" {
		cond = " AND " + query
	}

	sqlStr := fmt.Sprintf(q.QuerySubscribersForExportStmt, cond)

	// Verify that the arbitrary SQL search expression is read only.
	if cond != "" {
		tx, err := q.db.Unsafe().BeginTxx(context.Background(), &sql.TxOptions{ReadOnly: true})
		if err != nil {
			return err
		}
		defer tx.Rollback()

		if _, err := tx.Query(sqlStr, nil, 0, 1); err != nil {
			return err
		}
	}

	// Prepare the actual query statement.
	stmt, err := q.db.Preparex(sqlStr)
	if err != nil {
		return err
	}

	for id := 0; ; {
		var out []models.SubscriberExport
		if err := stmt.Select(&out, listIDs, id, nil, batchSize); err != nil {
			return err
		}
		if len(out) == 0 {
			break
		}

		if err := onFetch(out); err != nil {
			return err
		}

		id = out[len(out)-1].ID
	}

	return nil
}

func (q *Queries) DeleteSubscriptionsByQuery(exp string, listIDs pq.Int64Array) error {
	return q.execSubscriberQueryTpl(sanitizeSQLExp(exp), q.DeleteSubscriptionsByQueryStmt, listIDs, q.db)
}

func (q *Queries) BlocklistSubscribersByQuery(exp string, listIDs pq.Int64Array) error {
	return q.execSubscriberQueryTpl(sanitizeSQLExp(exp), q.BlocklistSubscribersByQueryStmt, listIDs, q.db)
}

func (q *Queries) DeleteSubscribersByQuery(exp string, listIDs pq.Int64Array) error {
	return q.execSubscriberQueryTpl(sanitizeSQLExp(exp), q.DeleteSubscribersByQueryStmt, listIDs, q.db)
}

func (q *Queries) UnsubscribeSubscribersFromListsByQuery(exp string, listIDs, targetListIDs pq.Int64Array) error {
	return q.execSubscriberQueryTpl(sanitizeSQLExp(exp), q.UnsubscribeSubscribersFromListsByQueryStmt, listIDs, q.db, targetListIDs)
}

func (q *Queries) AddSubscribersToListsByQuery(exp string, listIDs, targetListIDs pq.Int64Array) error {
	return q.execSubscriberQueryTpl(sanitizeSQLExp(exp), q.AddSubscribersToListsByQueryStmt, listIDs, q.db, targetListIDs)
}

// dbConf contains database config required for connecting to a DB.
type dbConf struct {
	Host        string        `koanf:"host"`
	Port        int           `koanf:"port"`
	User        string        `koanf:"user"`
	Password    string        `koanf:"password"`
	DBName      string        `koanf:"database"`
	SSLMode     string        `koanf:"ssl_mode"`
	MaxOpen     int           `koanf:"max_open"`
	MaxIdle     int           `koanf:"max_idle"`
	MaxLifetime time.Duration `koanf:"max_lifetime"`
}

// connectDB initializes a database connection.
func connectDB(c dbConf) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres",
		fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(c.MaxOpen)
	db.SetMaxIdleConns(c.MaxIdle)
	db.SetConnMaxLifetime(c.MaxLifetime)
	return db, nil
}

// compileSubscriberQueryTpl takes a arbitrary WHERE expressions
// to filter subscribers from the subscribers table and prepares a query
// out of it using the raw `query-subscribers-template` query template.
// While doing this, a readonly transaction is created and the query is
// dry run on it to ensure that it is indeed readonly.
func (q *Queries) compileSubscriberQueryTpl(exp string, db *sqlx.DB) (string, error) {
	tx, err := db.BeginTxx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	// Perform the dry run.
	if exp != "" {
		exp = " AND " + exp
	}
	stmt := fmt.Sprintf(q.QuerySubscribersTplStmt, exp)
	if _, err := tx.Exec(stmt, true, pq.Int64Array{}); err != nil {
		return "", err
	}

	return stmt, nil
}

// compileSubscriberQueryTpl takes a arbitrary WHERE expressions and a subscriber
// query template that depends on the filter (eg: delete by query, blocklist by query etc.)
// combines and executes them.
func (q *Queries) execSubscriberQueryTpl(exp, tpl string, listIDs []int64, db *sqlx.DB, args ...interface{}) error {
	// Perform a dry run.
	filterExp, err := q.compileSubscriberQueryTpl(exp, db)
	if err != nil {
		return err
	}

	if len(listIDs) == 0 {
		listIDs = pq.Int64Array{}
	}
	// First argument is the boolean indicating if the query is a dry run.
	a := append([]interface{}{false, pq.Int64Array(listIDs)}, args...)
	if _, err := db.Exec(fmt.Sprintf(tpl, filterExp), a...); err != nil {
		return err
	}

	return nil
}
