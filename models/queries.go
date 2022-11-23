package models

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// Queries contains all prepared SQL queries.
type Queries struct {
	GetDashboardCharts *sqlx.Stmt `query:"get-dashboard-charts"`
	GetDashboardCounts *sqlx.Stmt `query:"get-dashboard-counts"`

	InsertSubscriber                *sqlx.Stmt `query:"insert-subscriber"`
	UpsertSubscriber                *sqlx.Stmt `query:"upsert-subscriber"`
	UpsertBlocklistSubscriber       *sqlx.Stmt `query:"upsert-blocklist-subscriber"`
	GetSubscriber                   *sqlx.Stmt `query:"get-subscriber"`
	GetSubscribersByEmails          *sqlx.Stmt `query:"get-subscribers-by-emails"`
	GetSubscriberLists              *sqlx.Stmt `query:"get-subscriber-lists"`
	GetSubscriptions                *sqlx.Stmt `query:"get-subscriptions"`
	GetSubscriberListsLazy          *sqlx.Stmt `query:"get-subscriber-lists-lazy"`
	UpdateSubscriber                *sqlx.Stmt `query:"update-subscriber"`
	UpdateSubscriberWithLists       *sqlx.Stmt `query:"update-subscriber-with-lists"`
	BlocklistSubscribers            *sqlx.Stmt `query:"blocklist-subscribers"`
	AddSubscribersToLists           *sqlx.Stmt `query:"add-subscribers-to-lists"`
	DeleteSubscriptions             *sqlx.Stmt `query:"delete-subscriptions"`
	DeleteUnconfirmedSubscriptions  *sqlx.Stmt `query:"delete-unconfirmed-subscriptions"`
	ConfirmSubscriptionOptin        *sqlx.Stmt `query:"confirm-subscription-optin"`
	UnsubscribeSubscribersFromLists *sqlx.Stmt `query:"unsubscribe-subscribers-from-lists"`
	DeleteSubscribers               *sqlx.Stmt `query:"delete-subscribers"`
	DeleteBlocklistedSubscribers    *sqlx.Stmt `query:"delete-blocklisted-subscribers"`
	DeleteOrphanSubscribers         *sqlx.Stmt `query:"delete-orphan-subscribers"`
	UnsubscribeByCampaign           *sqlx.Stmt `query:"unsubscribe-by-campaign"`
	ExportSubscriberData            *sqlx.Stmt `query:"export-subscriber-data"`

	// Non-prepared arbitrary subscriber queries.
	QuerySubscribers                       string `query:"query-subscribers"`
	QuerySubscribersCount                  string `query:"query-subscribers-count"`
	QuerySubscribersForExport              string `query:"query-subscribers-for-export"`
	QuerySubscribersTpl                    string `query:"query-subscribers-template"`
	DeleteSubscribersByQuery               string `query:"delete-subscribers-by-query"`
	AddSubscribersToListsByQuery           string `query:"add-subscribers-to-lists-by-query"`
	BlocklistSubscribersByQuery            string `query:"blocklist-subscribers-by-query"`
	DeleteSubscriptionsByQuery             string `query:"delete-subscriptions-by-query"`
	UnsubscribeSubscribersFromListsByQuery string `query:"unsubscribe-subscribers-from-lists-by-query"`

	CreateList      *sqlx.Stmt `query:"create-list"`
	QueryLists      string     `query:"query-lists"`
	GetLists        *sqlx.Stmt `query:"get-lists"`
	GetListsByOptin *sqlx.Stmt `query:"get-lists-by-optin"`
	UpdateList      *sqlx.Stmt `query:"update-list"`
	UpdateListsDate *sqlx.Stmt `query:"update-lists-date"`
	DeleteLists     *sqlx.Stmt `query:"delete-lists"`

	CreateCampaign        *sqlx.Stmt `query:"create-campaign"`
	QueryCampaigns        string     `query:"query-campaigns"`
	GetCampaign           *sqlx.Stmt `query:"get-campaign"`
	GetCampaignForPreview *sqlx.Stmt `query:"get-campaign-for-preview"`
	GetCampaignStats      *sqlx.Stmt `query:"get-campaign-stats"`
	GetCampaignStatus     *sqlx.Stmt `query:"get-campaign-status"`
	GetArchivedCampaigns  *sqlx.Stmt `query:"get-archived-campaigns"`

	// These two queries are read as strings and based on settings.individual_tracking=on/off,
	// are interpolated and copied to view and click counts. Same query, different tables.
	GetCampaignAnalyticsCounts       string     `query:"get-campaign-analytics-counts"`
	GetCampaignAnalyticsCountsUnique string     `query:"get-campaign-analytics-unique-counts"`
	GetCampaignViewCounts            *sqlx.Stmt `query:"get-campaign-view-counts"`
	GetCampaignClickCounts           *sqlx.Stmt `query:"get-campaign-click-counts"`
	GetCampaignLinkCounts            *sqlx.Stmt `query:"get-campaign-link-counts"`
	GetCampaignBounceCounts          *sqlx.Stmt `query:"get-campaign-bounce-counts"`
	DeleteCampaignViews              *sqlx.Stmt `query:"delete-campaign-views"`
	DeleteCampaignLinkClicks         *sqlx.Stmt `query:"delete-campaign-link-clicks"`

	NextCampaigns            *sqlx.Stmt `query:"next-campaigns"`
	NextCampaignSubscribers  *sqlx.Stmt `query:"next-campaign-subscribers"`
	GetOneCampaignSubscriber *sqlx.Stmt `query:"get-one-campaign-subscriber"`
	UpdateCampaign           *sqlx.Stmt `query:"update-campaign"`
	UpdateCampaignStatus     *sqlx.Stmt `query:"update-campaign-status"`
	UpdateCampaignCounts     *sqlx.Stmt `query:"update-campaign-counts"`
	UpdateCampaignArchive    *sqlx.Stmt `query:"update-campaign-archive"`
	RegisterCampaignView     *sqlx.Stmt `query:"register-campaign-view"`
	DeleteCampaign           *sqlx.Stmt `query:"delete-campaign"`

	InsertMedia *sqlx.Stmt `query:"insert-media"`
	GetAllMedia *sqlx.Stmt `query:"get-all-media"`
	GetMedia    *sqlx.Stmt `query:"get-media"`
	DeleteMedia *sqlx.Stmt `query:"delete-media"`

	CreateTemplate     *sqlx.Stmt `query:"create-template"`
	GetTemplates       *sqlx.Stmt `query:"get-templates"`
	UpdateTemplate     *sqlx.Stmt `query:"update-template"`
	SetDefaultTemplate *sqlx.Stmt `query:"set-default-template"`
	DeleteTemplate     *sqlx.Stmt `query:"delete-template"`

	CreateLink        *sqlx.Stmt `query:"create-link"`
	RegisterLinkClick *sqlx.Stmt `query:"register-link-click"`

	GetSettings    *sqlx.Stmt `query:"get-settings"`
	UpdateSettings *sqlx.Stmt `query:"update-settings"`

	// GetStats *sqlx.Stmt `query:"get-stats"`
	RecordBounce              *sqlx.Stmt `query:"record-bounce"`
	QueryBounces              string     `query:"query-bounces"`
	DeleteBounces             *sqlx.Stmt `query:"delete-bounces"`
	DeleteBouncesBySubscriber *sqlx.Stmt `query:"delete-bounces-by-subscriber"`
}

// CompileSubscriberQueryTpl takes an arbitrary WHERE expressions
// to filter subscribers from the subscribers table and prepares a query
// out of it using the raw `query-subscribers-template` query template.
// While doing this, a readonly transaction is created and the query is
// dry run on it to ensure that it is indeed readonly.
func (q *Queries) CompileSubscriberQueryTpl(exp string, db *sqlx.DB) (string, error) {
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

// compileSubscriberQueryTpl takes an arbitrary WHERE expressions and a subscriber
// query template that depends on the filter (eg: delete by query, blocklist by query etc.)
// combines and executes them.
func (q *Queries) ExecSubQueryTpl(exp, tpl string, listIDs []int, db *sqlx.DB, args ...interface{}) error {
	// Perform a dry run.
	filterExp, err := q.CompileSubscriberQueryTpl(exp, db)
	if err != nil {
		return err
	}

	if len(listIDs) == 0 {
		listIDs = []int{}
	}

	// First argument is the boolean indicating if the query is a dry run.
	a := append([]interface{}{false, pq.Array(listIDs)}, args...)
	if _, err := db.Exec(fmt.Sprintf(tpl, filterExp), a...); err != nil {
		return err
	}

	return nil
}
