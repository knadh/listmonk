package models

import "github.com/jmoiron/sqlx"

// Queries contains all prepared SQL queries.
type Queries struct {
	GetDashboardCharts *sqlx.Stmt `query:"get-dashboard-charts"`
	GetDashboardCounts *sqlx.Stmt `query:"get-dashboard-counts"`

	InsertSubscriber                *sqlx.Stmt `query:"insert-subscriber"`
	UpsertSubscriber                *sqlx.Stmt `query:"upsert-subscriber"`
	UpsertBlocklistSubscriber       *sqlx.Stmt `query:"upsert-blocklist-subscriber"`
	GetSubscriber                   *sqlx.Stmt `query:"get-subscriber"`
	HasSubscriberLists              *sqlx.Stmt `query:"has-subscriber-list"`
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
	GetSubscriberCharts             *sqlx.Stmt `query:"get-subscriber-charts"`
	GetSubscriberActivity           *sqlx.Stmt `query:"get-subscriber-activity"`
	DeleteSubscriberCampaignViews   *sqlx.Stmt `query:"delete-subscriber-campaign-views"`
	DeleteSubscriberLinkClicks      *sqlx.Stmt `query:"delete-subscriber-link-clicks"`

	// Non-prepared arbitrary subscriber queries.
	QuerySubscribers                       string     `query:"query-subscribers"`
	QuerySubscribersCount                  string     `query:"query-subscribers-count"`
	QuerySubscribersCountAll               *sqlx.Stmt `query:"query-subscribers-count-all"`
	QuerySubscribersForExport              string     `query:"query-subscribers-for-export"`
	QuerySubscribersTpl                    string     `query:"query-subscribers-template"`
	DeleteSubscribersByQuery               string     `query:"delete-subscribers-by-query"`
	AddSubscribersToListsByQuery           string     `query:"add-subscribers-to-lists-by-query"`
	BlocklistSubscribersByQuery            string     `query:"blocklist-subscribers-by-query"`
	DeleteSubscriptionsByQuery             string     `query:"delete-subscriptions-by-query"`
	UnsubscribeSubscribersFromListsByQuery string     `query:"unsubscribe-subscribers-from-lists-by-query"`

	GetListAnalyticsItems       *sqlx.Stmt `query:"get-list-analytics-items"`
	GetCampaignAnalyticsItems   *sqlx.Stmt `query:"get-campaign-analytics-items"`
	GetListAnalyticsCounts      string     `query:"get-list-analytics-counts"`
	GetListViewCounts           *sqlx.Stmt `query:"get-list-view-counts"`
	GetListClickCounts          *sqlx.Stmt `query:"get-list-click-counts"`
	GetListTotalViewCounts      *sqlx.Stmt `query:"get-list-total-view-counts"`
	GetListTotalClickCounts     *sqlx.Stmt `query:"get-list-total-click-counts"`
	GetListBounceCounts         *sqlx.Stmt `query:"get-list-bounce-counts"`
	GetListSubscriptionCounts   *sqlx.Stmt `query:"get-list-subscription-counts"`
	GetListUnsubscriptionCounts *sqlx.Stmt `query:"get-list-unsubscription-counts"`
	GetListLinkCounts           *sqlx.Stmt `query:"get-list-link-counts"`
	GetListCharts               *sqlx.Stmt `query:"get-list-charts"`
	GetListChartsLive           string     `query:"get-list-charts-live"`

	CreateList      *sqlx.Stmt `query:"create-list"`
	QueryLists      string     `query:"query-lists"`
	GetLists        *sqlx.Stmt `query:"get-lists"`
	GetListsByOptin *sqlx.Stmt `query:"get-lists-by-optin"`
	GetListTypes    *sqlx.Stmt `query:"get-list-types"`
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
	CampaignHasLists      *sqlx.Stmt `query:"campaign-has-lists"`

	// These two queries are read as strings and based on settings.individual_tracking=on/off,
	// are interpolated and copied to view and click counts. Same query, different tables.
	GetCampaignAnalyticsCounts  string     `query:"get-campaign-analytics-counts"`
	GetCampaignViewCounts       *sqlx.Stmt `query:"get-campaign-view-counts"`
	GetCampaignClickCounts      *sqlx.Stmt `query:"get-campaign-click-counts"`
	GetCampaignLinkCounts       *sqlx.Stmt `query:"get-campaign-link-counts"`
	GetCampaignBounceCounts     *sqlx.Stmt `query:"get-campaign-bounce-counts"`
	DeleteCampaignViews         *sqlx.Stmt `query:"delete-campaign-views"`
	DeleteCampaignLinkClicks    *sqlx.Stmt `query:"delete-campaign-link-clicks"`
	ExportCampaignViews         *sqlx.Stmt `query:"export-campaign-views"`
	ExportCampaignLinkClicks    *sqlx.Stmt `query:"export-campaign-link-clicks"`
	GetCampaignTotalViewCounts  *sqlx.Stmt `query:"get-campaign-total-view-counts"`
	GetCampaignTotalClickCounts *sqlx.Stmt `query:"get-campaign-total-click-counts"`

	NextCampaigns            *sqlx.Stmt `query:"next-campaigns"`
	GetRunningCampaign       *sqlx.Stmt `query:"get-running-campaign"`
	NextCampaignSubscribers  *sqlx.Stmt `query:"next-campaign-subscribers"`
	GetOneCampaignSubscriber *sqlx.Stmt `query:"get-one-campaign-subscriber"`
	UpdateCampaign           *sqlx.Stmt `query:"update-campaign"`
	UpdateCampaignStatus     *sqlx.Stmt `query:"update-campaign-status"`
	UpdateCampaignCounts     *sqlx.Stmt `query:"update-campaign-counts"`
	UpdateCampaignArchive    *sqlx.Stmt `query:"update-campaign-archive"`
	RegisterCampaignView     *sqlx.Stmt `query:"register-campaign-view"`
	DeleteCampaign           *sqlx.Stmt `query:"delete-campaign"`
	DeleteCampaigns          *sqlx.Stmt `query:"delete-campaigns"`

	InsertMedia *sqlx.Stmt `query:"insert-media"`
	GetMedia    *sqlx.Stmt `query:"get-media"`
	QueryMedia  *sqlx.Stmt `query:"query-media"`
	DeleteMedia *sqlx.Stmt `query:"delete-media"`

	CreateTemplate     *sqlx.Stmt `query:"create-template"`
	GetTemplates       string     `query:"get-templates"`
	UpdateTemplate     *sqlx.Stmt `query:"update-template"`
	SetDefaultTemplate *sqlx.Stmt `query:"set-default-template"`
	DeleteTemplate     *sqlx.Stmt `query:"delete-template"`

	CreateLink        *sqlx.Stmt `query:"create-link"`
	GetLinkURL        *sqlx.Stmt `query:"get-link-url"`
	RegisterLinkClick *sqlx.Stmt `query:"register-link-click"`

	GetSettings         *sqlx.Stmt `query:"get-settings"`
	UpdateSettings      *sqlx.Stmt `query:"update-settings"`
	UpdateSettingsByKey *sqlx.Stmt `query:"update-settings-by-key"`

	// GetStats *sqlx.Stmt `query:"get-stats"`
	RecordBounce                *sqlx.Stmt `query:"record-bounce"`
	QueryBounces                string     `query:"query-bounces"`
	BlocklistBouncedSubscribers *sqlx.Stmt `query:"blocklist-bounced-subscribers"`
	DeleteBounces               *sqlx.Stmt `query:"delete-bounces"`
	DeleteBouncesBySubscriber   *sqlx.Stmt `query:"delete-bounces-by-subscriber"`
	GetDBInfo                   string     `query:"get-db-info"`

	CreateUser         *sqlx.Stmt `query:"create-user"`
	UpdateUser         *sqlx.Stmt `query:"update-user"`
	UpdateUserProfile  *sqlx.Stmt `query:"update-user-profile"`
	UpdateUserLogin    *sqlx.Stmt `query:"update-user-login"`
	SetUserTwoFA       *sqlx.Stmt `query:"set-user-twofa"`
	DeleteUsers        *sqlx.Stmt `query:"delete-users"`
	GetUsers           string     `query:"get-users"`
	GetUser            *sqlx.Stmt `query:"get-user"`
	GetAPITokens       *sqlx.Stmt `query:"get-api-tokens"`
	LoginUser          *sqlx.Stmt `query:"login-user"`
	DeleteUserSessions *sqlx.Stmt `query:"delete-user-sessions"`

	CreateRole            *sqlx.Stmt `query:"create-role"`
	GetUserRoles          string     `query:"get-user-roles"`
	GetListRoles          string     `query:"get-list-roles"`
	UpdateRole            *sqlx.Stmt `query:"update-role"`
	DeleteRole            *sqlx.Stmt `query:"delete-role"`
	UpsertListPermissions *sqlx.Stmt `query:"upsert-list-permissions"`
	DeleteListPermission  *sqlx.Stmt `query:"delete-list-permission"`
}

// ApplyOrphanFilter handles the special subscription_status value "none", which
// filters orphan subscribers (no list subscriptions).
func ApplyOrphanFilter(cond, subStatus string) (string, string) {
	const orphanSubCond = "NOT EXISTS (SELECT 1 FROM subscriber_lists WHERE subscriber_id = subscribers.id)"
	if subStatus == "none" {
		if cond == "" {
			cond = "TRUE"
		}
		return "(" + cond + ") AND " + orphanSubCond, ""
	}

	return cond, subStatus
}
