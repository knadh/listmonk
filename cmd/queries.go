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
	getDashboardCharts *sqlx.Stmt `query:"get-dashboard-charts"`
	getDashboardCounts *sqlx.Stmt `query:"get-dashboard-counts"`

	insertSubscriber                *sqlx.Stmt `query:"insert-subscriber"`
	upsertSubscriber                *sqlx.Stmt `query:"upsert-subscriber"`
	upsertBlocklistSubscriber       *sqlx.Stmt `query:"upsert-blocklist-subscriber"`
	getSubscriber                   *sqlx.Stmt `query:"get-subscriber"`
	getSubscribersByEmails          *sqlx.Stmt `query:"get-subscribers-by-emails"`
	getSubscriberLists              *sqlx.Stmt `query:"get-subscriber-lists"`
	GetSubscriberListsLazy          *sqlx.Stmt `query:"get-subscriber-lists-lazy"`
	subscriberExists                *sqlx.Stmt `query:"subscriber-exists"`
	updateSubscriber                *sqlx.Stmt `query:"update-subscriber"`
	blocklistSubscribers            *sqlx.Stmt `query:"blocklist-subscribers"`
	addSubscribersToLists           *sqlx.Stmt `query:"add-subscribers-to-lists"`
	deleteSubscriptions             *sqlx.Stmt `query:"delete-subscriptions"`
	confirmSubscriptionOptin        *sqlx.Stmt `query:"confirm-subscription-optin"`
	unsubscribeSubscribersFromLists *sqlx.Stmt `query:"unsubscribe-subscribers-from-lists"`
	deleteSubscribers               *sqlx.Stmt `query:"delete-subscribers"`
	unsubscribe                     *sqlx.Stmt `query:"unsubscribe"`
	exportSubscriberData            *sqlx.Stmt `query:"export-subscriber-data"`

	// Non-prepared arbitrary subscriber queries.
	QuerySubscribers                       string `query:"query-subscribers"`
	QuerySubscribersForExport              string `query:"query-subscribers-for-export"`
	QuerySubscribersTpl                    string `query:"query-subscribers-template"`
	DeleteSubscribersByQuery               string `query:"delete-subscribers-by-query"`
	AddSubscribersToListsByQuery           string `query:"add-subscribers-to-lists-by-query"`
	BlocklistSubscribersByQuery            string `query:"blocklist-subscribers-by-query"`
	DeleteSubscriptionsByQuery             string `query:"delete-subscriptions-by-query"`
	UnsubscribeSubscribersFromListsByQuery string `query:"unsubscribe-subscribers-from-lists-by-query"`

	createList      *sqlx.Stmt `query:"create-list"`
	queryLists      string     `query:"query-lists"`
	getLists        *sqlx.Stmt `query:"get-lists"`
	getListsByOptin *sqlx.Stmt `query:"get-lists-by-optin"`
	updateList      *sqlx.Stmt `query:"update-list"`
	updateListsDate *sqlx.Stmt `query:"update-lists-date"`
	deleteLists     *sqlx.Stmt `query:"delete-lists"`

	createCampaign           *sqlx.Stmt `query:"create-campaign"`
	queryCampaigns           string     `query:"query-campaigns"`
	getCampaign              *sqlx.Stmt `query:"get-campaign"`
	getCampaignForPreview    *sqlx.Stmt `query:"get-campaign-for-preview"`
	getCampaignStats         *sqlx.Stmt `query:"get-campaign-stats"`
	getCampaignStatus        *sqlx.Stmt `query:"get-campaign-status"`
	nextCampaigns            *sqlx.Stmt `query:"next-campaigns"`
	nextCampaignSubscribers  *sqlx.Stmt `query:"next-campaign-subscribers"`
	getOneCampaignSubscriber *sqlx.Stmt `query:"get-one-campaign-subscriber"`
	updateCampaign           *sqlx.Stmt `query:"update-campaign"`
	updateCampaignStatus     *sqlx.Stmt `query:"update-campaign-status"`
	updateCampaignCounts     *sqlx.Stmt `query:"update-campaign-counts"`
	registerCampaignView     *sqlx.Stmt `query:"register-campaign-view"`
	deleteCampaign           *sqlx.Stmt `query:"delete-campaign"`

	insertMedia *sqlx.Stmt `query:"insert-media"`
	getMedia    *sqlx.Stmt `query:"get-media"`
	deleteMedia *sqlx.Stmt `query:"delete-media"`

	createTemplate     *sqlx.Stmt `query:"create-template"`
	getTemplates       *sqlx.Stmt `query:"get-templates"`
	updateTemplate     *sqlx.Stmt `query:"update-template"`
	setDefaultTemplate *sqlx.Stmt `query:"set-default-template"`
	deleteTemplate     *sqlx.Stmt `query:"delete-template"`

	createLink        *sqlx.Stmt `query:"create-link"`
	registerLinkClick *sqlx.Stmt `query:"register-link-click"`

	getSettings    *sqlx.Stmt `query:"get-settings"`
	updateSettings *sqlx.Stmt `query:"update-settings"`

	// GetStats *sqlx.Stmt `query:"get-stats"`
}

func (q *Queries) GetDashboardCharts(c *types.JSONText) error {
	return q.getDashboardCharts.Get(c)
}

func (q *Queries) GetDashboardCounts(c *types.JSONText) error {
	return q.getDashboardCounts.Get(c)
}

func (q *Queries) InsertSubscriber(
	id *int,
	uuid, email, name, status string,
	attribs models.SubscriberAttribs,
	lists pq.Int64Array,
	listUUIDs pq.StringArray,
	subStatus string,
) error {
	return q.insertSubscriber.Get(&id, uuid, email, name, status, attribs, lists, listUUIDs, subStatus)
}

func (q *Queries) GetSubscriber(s *models.Subscribers, id int, uuid, email string) error {
	return q.getSubscriber.Select(s, id, uuid, email)
}

func (q *Queries) GetSubscribersByEmails(s *models.Subscribers, emails pq.StringArray) error {
	return q.getSubscribersByEmails.Select(s, emails)
}

func (q *Queries) GetSubscriberLists(
	lists *[]models.List,
	id int, uuid string,
	listIDs pq.Int64Array,
	listUUIDs pq.StringArray,
	subStatus, optin string,
) error {
	return q.getSubscriberLists.Select(lists, id, uuid, listIDs, listUUIDs, subStatus, optin)
}

func (q *Queries) SubscriberExists(
	exists *bool,
	id int, uuid string,
) error {
	return q.subscriberExists.Select(exists, id, uuid)
}

func (q *Queries) UpdateSubscriber(
	id int64,
	email, name, status string,
	attribs json.RawMessage,
	lists pq.Int64Array,
) error {
	_, err := q.updateSubscriber.Exec(id, email, name, status, attribs, lists)
	return err
}

func (q *Queries) BlocklistSubscribers(
	list pq.Int64Array,
) error {
	_, err := q.blocklistSubscribers.Exec(list)
	return err
}

func (q *Queries) AddSubscribersToLists(
	subscribers, lists pq.Int64Array,
) error {
	_, err := q.blocklistSubscribers.Exec(subscribers, lists)
	return err
}

func (q *Queries) DeleteSubscriptions(
	subscribers, lists pq.Int64Array,
) error {
	_, err := q.deleteSubscriptions.Exec(subscribers, lists)
	return err
}

func (q *Queries) UnsubscribeSubscribersFromLists(
	subscribers, lists pq.Int64Array,
) error {
	_, err := q.unsubscribeSubscribersFromLists.Exec(subscribers, lists)
	return err
}

func (q *Queries) ConfirmSubscriptionOptin(
	subUUID string, listUUIDs pq.StringArray,
) error {
	_, err := q.confirmSubscriptionOptin.Exec(subUUID, listUUIDs)
	return err
}

func (q *Queries) DeleteSubscribers(
	subIDs pq.Int64Array, subUUIDs pq.StringArray,
) error {
	_, err := q.deleteSubscribers.Exec(subIDs, subUUIDs)
	return err
}

func (q *Queries) Unsubscribe(
	campaignUUID, subUUID string,
	blocklist bool,
) error {
	_, err := q.unsubscribe.Exec(campaignUUID, subUUID, blocklist)
	return err
}

func (q *Queries) ExportSubscriberData(
	data interface{},
	id int64,
	uuidOrNil interface{},
) error {
	return q.exportSubscriberData.Get(data, id, uuidOrNil)
}

func (q *Queries) CreateList(
	list *int,
	uuid interface{},
	name, listType, optin string,
	tags pq.StringArray,
) error {
	return q.exportSubscriberData.Get(list, uuid, name, listType, optin, tags)
}

func (q *Queries) GetLists(
	lists *[]models.List,
	listType string,
) error {
	return q.getLists.Select(lists, listType)
}

func (q *Queries) GetListsByOptin(
	lists *[]models.List,
	optin string,
	listIDs pq.Int64Array,
	listUUIDs pq.StringArray,
) error {
	return q.getListsByOptin.Select(lists, optin, listIDs, listUUIDs)
}

func (q *Queries) UpdateList(
	id int,
	name, listType, optin string,
	tags pq.StringArray,
) (sql.Result, error) {
	return q.updateList.Exec(id, name, listType, optin, tags)
}

func (q *Queries) DeleteLists(ids pq.Int64Array) error {
	_, err := q.deleteLists.Exec(ids)
	return err
}

func (q *Queries) CreateCampaign(
	id *int,
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
	_, err := q.createCampaign.Exec(id, uuid, campaignType, name, subject, fromEmail, body, altBody, contentType, sendAt, tags, messenger, templateID, listIDs)
	return err
}

func (q *Queries) GetCampaign(campaign *models.Campaign, id int, uuid *string) error {
	_, err := q.getCampaign.Exec(campaign, id, uuid)
	return err
}

func (q *Queries) GetCampaignForPreview(campaign *models.Campaign, id int) error {
	_, err := q.getCampaignForPreview.Exec(campaign, id)
	return err
}

func (q *Queries) GetCampaignStatus(stats interface{}, status string) error {
	return q.getCampaignStatus.Select(stats, status)
}

func (q *Queries) NextCampaigns(c *[]*models.Campaign, excludeIDs pq.Int64Array) error {
	return q.nextCampaigns.Select(c, excludeIDs)
}

func (q *Queries) NextCampaignSubscribers(subs *[]models.Subscriber, campID, limit int) error {
	return q.nextCampaignSubscribers.Select(subs, campID, limit)
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
	_, err := q.updateCampaign.Exec(id, name, subject, fromEmail, body, altBody, contentType, sendAt, sendLater, tags, messenger, templateID, listIDs)
	return err
}

func (q *Queries) UpdateCampaignStatus(
	id int,
	status string,
) (sql.Result, error) {
	return q.updateCampaignStatus.Exec(id, status)
}

func (q *Queries) RegisterCampaignView(campUUID, subUUID string) error {
	_, err := q.registerCampaignView.Exec(campUUID, subUUID)
	return err
}

func (q *Queries) DeleteCampaign(id int) error {
	_, err := q.deleteCampaign.Exec(id)
	return err
}

func (q *Queries) InsertMedia(id uuid.UUID, filename, thumbnailFilename, mediaProvider string) error {
	_, err := q.insertMedia.Exec(id, filename, thumbnailFilename, mediaProvider)
	return err
}

func (q *Queries) GetMedia(media *[]media.Media, provider string) error {
	return q.getMedia.Select(media, provider)
}

func (q *Queries) DeleteMedia(media *media.Media, id int) error {
	return q.deleteMedia.Get(media, id)
}

func (q *Queries) CreateTemplate(id *int, name, body string) error {
	return q.createTemplate.Get(id, name, body)
}

func (q *Queries) GetTemplates(out *[]models.Template, id int, noBody bool) error {
	return q.getTemplates.Select(out, id, noBody)
}

func (q *Queries) UpdateTemplate(id int, name, body string) (sql.Result, error) {
	return q.updateTemplate.Exec(id, name, body)
}

func (q *Queries) SetDefaultTemplate(id int) error {
	_, err := q.setDefaultTemplate.Exec(id)
	return err
}

func (q *Queries) DeleteTemplate(delID *int, id int) error {
	_, err := q.deleteTemplate.Exec(delID, id)
	return err
}

func (q *Queries) CreateLink(out *string, id uuid.UUID, url string) error {
	return q.createLink.Get(out, id, url)
}

func (q *Queries) RegisterLinkClick(url *string, linkUUID, campUUID, subUUID string) error {
	return q.registerLinkClick.Get(url, linkUUID, campUUID, subUUID)
}

func (q *Queries) UpdateSettings(settings []byte) error {
	_, err := q.updateSettings.Exec(settings)
	return err
}

func (q *Queries) QueryLists(listID, offset, limit int, orderBy, order string) ([]models.List, error) {
	query := fmt.Sprintf(q.queryLists, orderBy, order)

	var results []models.List
	if err := db.Select(&results, query, listID, offset, limit); err != nil {
		return nil, err
	}

	return results, nil
}

func (q *Queries) QueryCampaigns(id, offset, limit int, status []string, query, orderBy, order string) ([]models.Campaign, error) {
	stmt := fmt.Sprintf(q.queryCampaigns, orderBy, order)

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
	stmt := q.upsertSubscriber.Stmt
	if tx != nil {
		stmt = tx.Stmt(stmt)
	}
	_, err := q.upsertSubscriber.Exec(uuid, email, name, attribs, listIDs, overwrite)
	return err
}

func (q *Queries) UpsertBlocklistSubscriber(
	uu uuid.UUID,
	email, name string,
	attribs models.SubscriberAttribs,
	tx *sql.Tx,
) error {
	stmt := q.upsertBlocklistSubscriber.Stmt
	if tx != nil {
		stmt = tx.Stmt(stmt)
	}
	_, err := stmt.Exec(uu, email, name, attribs)
	return err
}

func (q *Queries) UpdateListsDate(listIDs pq.Int64Array, tx *sql.Tx) error {
	stmt := q.updateListsDate.Stmt
	if tx != nil {
		stmt = tx.Stmt(stmt)
	}
	_, err := stmt.Exec(listIDs)
	return err
}

func (q *Queries) GetCampaignStats(campaignIDs []int) ([]models.CampaignMeta, error) {
	var meta []models.CampaignMeta
	if err := q.getCampaignStats.Select(&meta, pq.Array(campaignIDs)); err != nil {
		return meta, err
	}

	if len(campaignIDs) != len(meta) {
		return meta, errors.New("campaign stats count does not match")
	}

	return meta, nil
}

func (q *Queries) GetSettings() (json.RawMessage, error) {
	var s types.JSONText
	if err := q.getSettings.Get(&s); err != nil {
		return nil, fmt.Errorf("error reading settings from DB: %s", pqErrMsg(err))
	}
	return json.RawMessage(s), nil
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
	stmt := fmt.Sprintf(q.QuerySubscribersTpl, exp)
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
