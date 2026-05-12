package models

import (
	"context"
	"database/sql"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// Queries contains all prepared SQL queries.
type Queries struct {
	GetDashboardCharts      *sqlx.Stmt `query:"get-dashboard-charts"`
	GetDashboardCounts      *sqlx.Stmt `query:"get-dashboard-counts"`
	GetDashboardFeatureCounts *sqlx.Stmt `query:"get-dashboard-feature-counts"`

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
	GetSubscriberActivity           *sqlx.Stmt `query:"get-subscriber-activity"`

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
	GetCampaignAnalyticsCounts string     `query:"get-campaign-analytics-counts"`
	GetCampaignViewCounts      *sqlx.Stmt `query:"get-campaign-view-counts"`
	GetCampaignClickCounts     *sqlx.Stmt `query:"get-campaign-click-counts"`
	GetCampaignLinkCounts      *sqlx.Stmt `query:"get-campaign-link-counts"`
	GetCampaignBounceCounts    *sqlx.Stmt `query:"get-campaign-bounce-counts"`
	DeleteCampaignViews        *sqlx.Stmt `query:"delete-campaign-views"`
	DeleteCampaignLinkClicks   *sqlx.Stmt `query:"delete-campaign-link-clicks"`

	NextCampaigns            *sqlx.Stmt `query:"next-campaigns"`
	GetRunningCampaign       *sqlx.Stmt `query:"get-running-campaign"`
	NextCampaignSubscribers  *sqlx.Stmt `query:"next-campaign-subscribers"`
	GetOneCampaignSubscriber *sqlx.Stmt `query:"get-one-campaign-subscriber"`
	UpdateCampaign           *sqlx.Stmt `query:"update-campaign"`
	UpdateCampaignStatus     *sqlx.Stmt `query:"update-campaign-status"`
	UpdateCampaignCounts     *sqlx.Stmt `query:"update-campaign-counts"`
	UpdateCampaignArchive    *sqlx.Stmt `query:"update-campaign-archive"`
	RefreshCampaignsToSend   *sqlx.Stmt `query:"refresh-campaigns-to-send"`
	GetEvergreenCampaignsWithNewSubs *sqlx.Stmt `query:"get-evergreen-campaigns-with-new-subs"`
	ResetEvergreenProgress           *sqlx.Stmt `query:"reset-evergreen-progress"`
	SetCampaignEvergreen             *sqlx.Stmt `query:"set-campaign-evergreen"`
	RegisterCampaignView      *sqlx.Stmt `query:"register-campaign-view"`
	InsertCampaignSendLog     *sqlx.Stmt `query:"insert-campaign-send-log"`
	QueryCampaignSendLog      *sqlx.Stmt `query:"query-campaign-send-log"`
	QueryCampaignSendLogStats *sqlx.Stmt `query:"query-campaign-send-log-stats"`
	DeleteFailedCampaignSends *sqlx.Stmt `query:"delete-failed-campaign-sends"`
	DeleteCampaign            *sqlx.Stmt `query:"delete-campaign"`
	DeleteCampaigns          *sqlx.Stmt `query:"delete-campaigns"`

	InsertMedia *sqlx.Stmt `query:"insert-media"`
	GetMedia    *sqlx.Stmt `query:"get-media"`
	QueryMedia  *sqlx.Stmt `query:"query-media"`
	DeleteMedia *sqlx.Stmt `query:"delete-media"`

	CreateTemplate     *sqlx.Stmt `query:"create-template"`
	GetTemplates       *sqlx.Stmt `query:"get-templates"`
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
	GetUsers           *sqlx.Stmt `query:"get-users"`
	GetUser            *sqlx.Stmt `query:"get-user"`
	GetAPITokens       *sqlx.Stmt `query:"get-api-tokens"`
	RegenerateAPIToken *sqlx.Stmt `query:"regenerate-api-token"`
	LoginUser          *sqlx.Stmt `query:"login-user"`
	DeleteUserSessions *sqlx.Stmt `query:"delete-user-sessions"`

	// Segments.
	CreateSegment      *sqlx.Stmt `query:"create-segment"`
	QuerySegments      string     `query:"query-segments"`
	GetSegment         *sqlx.Stmt `query:"get-segment"`
	UpdateSegment      *sqlx.Stmt `query:"update-segment"`
	UpdateSegmentCount *sqlx.Stmt `query:"update-segment-count"`
	DeleteSegment      *sqlx.Stmt `query:"delete-segment"`

	// Drip campaigns.
	CreateDripCampaign       *sqlx.Stmt `query:"create-drip-campaign"`
	QueryDripCampaigns       string     `query:"query-drip-campaigns"`
	GetDripCampaign          *sqlx.Stmt `query:"get-drip-campaign"`
	UpdateDripCampaign       *sqlx.Stmt `query:"update-drip-campaign"`
	UpdateDripCampaignStatus *sqlx.Stmt `query:"update-drip-campaign-status"`
	DeleteDripCampaign       *sqlx.Stmt `query:"delete-drip-campaign"`
	UpdateDripCampaignCounts *sqlx.Stmt `query:"update-drip-campaign-counts"`

	// Drip steps.
	CreateDripStep       *sqlx.Stmt `query:"create-drip-step"`
	GetDripSteps         *sqlx.Stmt `query:"get-drip-steps"`
	GetDripStep          *sqlx.Stmt `query:"get-drip-step"`
	UpdateDripStep       *sqlx.Stmt `query:"update-drip-step"`
	DeleteDripStep       *sqlx.Stmt `query:"delete-drip-step"`
	UpdateDripStepCounts *sqlx.Stmt `query:"update-drip-step-counts"`

	// Drip enrollments.
	EnrollSubscriberInDrip  *sqlx.Stmt `query:"enroll-subscriber-in-drip"`
	GetPendingDripSends     *sqlx.Stmt `query:"get-pending-drip-sends"`
	AdvanceDripEnrollment   *sqlx.Stmt `query:"advance-drip-enrollment"`
	ExitDripEnrollment      *sqlx.Stmt `query:"exit-drip-enrollment"`
	GetDripEnrollments      string     `query:"get-drip-enrollments"`
	GetDripEnrollmentCount  *sqlx.Stmt `query:"get-drip-enrollment-count"`
	InsertDripSendLog       *sqlx.Stmt `query:"insert-drip-send-log"`
	GetActiveDripsByTrigger *sqlx.Stmt `query:"get-active-drips-by-trigger"`

	// Drip production queries.
	GetDripSendsToday           *sqlx.Stmt `query:"get-drip-sends-today"`
	UpdateDripStepSent          *sqlx.Stmt `query:"update-drip-step-sent"`
	UpdateDripStepOpened        *sqlx.Stmt `query:"update-drip-step-opened"`
	UpdateDripStepClicked       *sqlx.Stmt `query:"update-drip-step-clicked"`
	UpdateDripCampaignEntered   *sqlx.Stmt `query:"update-drip-campaign-entered"`
	UpdateDripCampaignCompleted *sqlx.Stmt `query:"update-drip-campaign-completed"`
	BulkEnrollInDrip            *sqlx.Stmt `query:"bulk-enroll-in-drip"`
	GetDripCampaignByUUID       *sqlx.Stmt `query:"get-drip-campaign-by-uuid"`
	GetDripStepByUUID           *sqlx.Stmt `query:"get-drip-step-by-uuid"`

	// Warming.
	GetWarmingAddresses       *sqlx.Stmt `query:"get-warming-addresses"`
	CreateWarmingAddress      *sqlx.Stmt `query:"create-warming-address"`
	UpdateWarmingAddress      *sqlx.Stmt `query:"update-warming-address"`
	DeleteWarmingAddress      *sqlx.Stmt `query:"delete-warming-address"`
	GetActiveWarmingAddresses *sqlx.Stmt `query:"get-active-warming-addresses"`
	GetWarmingSenders         *sqlx.Stmt `query:"get-warming-senders"`
	CreateWarmingSender       *sqlx.Stmt `query:"create-warming-sender"`
	UpdateWarmingSender       *sqlx.Stmt `query:"update-warming-sender"`
	DeleteWarmingSender       *sqlx.Stmt `query:"delete-warming-sender"`
	GetActiveWarmingSenders   *sqlx.Stmt `query:"get-active-warming-senders"`
	GetWarmingTemplates       *sqlx.Stmt `query:"get-warming-templates"`
	CreateWarmingTemplate     *sqlx.Stmt `query:"create-warming-template"`
	UpdateWarmingTemplate     *sqlx.Stmt `query:"update-warming-template"`
	DeleteWarmingTemplate     *sqlx.Stmt `query:"delete-warming-template"`
	GetActiveWarmingTemplates *sqlx.Stmt `query:"get-active-warming-templates"`
	GetWarmingConfig          *sqlx.Stmt `query:"get-warming-config"`
	UpdateWarmingConfig       *sqlx.Stmt `query:"update-warming-config"`
	GetWarmingSendsToday      *sqlx.Stmt `query:"get-warming-sends-today"`
	InsertWarmingSendLog      *sqlx.Stmt `query:"insert-warming-send-log"`
	GetWarmingSendLog         string     `query:"get-warming-send-log"`
	GetWarmingSendLogCount    *sqlx.Stmt `query:"get-warming-send-log-count"`

	// Warming campaigns.
	GetWarmingCampaigns          *sqlx.Stmt `query:"get-warming-campaigns"`
	CreateWarmingCampaign        *sqlx.Stmt `query:"create-warming-campaign"`
	UpdateWarmingCampaign        *sqlx.Stmt `query:"update-warming-campaign"`
	DeleteWarmingCampaign        *sqlx.Stmt `query:"delete-warming-campaign"`
	GetActiveWarmingCampaigns    *sqlx.Stmt `query:"get-active-warming-campaigns"`
	GetWarmingSendersByDomains   *sqlx.Stmt `query:"get-warming-senders-by-domains"`
	GetWarmingSendsTodayByCampaign *sqlx.Stmt `query:"get-warming-sends-today-by-campaign"`
	InsertWarmingSendLogCampaign      *sqlx.Stmt `query:"insert-warming-send-log-campaign"`
	GetWarmingSendLogByCampaign       string     `query:"get-warming-send-log-by-campaign"`
	GetWarmingSendLogCountByCampaign  *sqlx.Stmt `query:"get-warming-send-log-count-by-campaign"`
	GetWarmingSendsLastHourByCampaign *sqlx.Stmt `query:"get-warming-sends-last-hour-by-campaign"`
	SetWarmingCampaignStartDate      *sqlx.Stmt `query:"set-warming-campaign-start-date"`
	GetWarmingCampaignStatsByID      *sqlx.Stmt `query:"get-warming-campaign-stats-by-id"`
	GetWarmingSenderByID             *sqlx.Stmt `query:"get-warming-sender-by-id"`

	// A/B tests.
	CreateABTest          *sqlx.Stmt `query:"create-ab-test"`
	GetABTest             *sqlx.Stmt `query:"get-ab-test"`
	GetABTestByCampaign   *sqlx.Stmt `query:"get-ab-test-by-campaign"`
	UpdateABTest          *sqlx.Stmt `query:"update-ab-test"`
	UpdateABTestStatus    *sqlx.Stmt `query:"update-ab-test-status"`
	DeleteABTest          *sqlx.Stmt `query:"delete-ab-test"`
	CreateABVariant       *sqlx.Stmt `query:"create-ab-variant"`
	GetABVariants         *sqlx.Stmt `query:"get-ab-variants"`
	GetABVariant          *sqlx.Stmt `query:"get-ab-variant"`
	UpdateABVariant       *sqlx.Stmt `query:"update-ab-variant"`
	UpdateABVariantCounts *sqlx.Stmt `query:"update-ab-variant-counts"`
	DeleteABVariant       *sqlx.Stmt `query:"delete-ab-variant"`
	AssignSubscriberToVariant *sqlx.Stmt `query:"assign-subscriber-to-variant"`
	GetSubscriberVariant     *sqlx.Stmt `query:"get-subscriber-variant"`
	GetRunningABTests        *sqlx.Stmt `query:"get-running-ab-tests"`

	// Automations.
	CreateAutomation              *sqlx.Stmt `query:"create-automation"`
	QueryAutomations              string     `query:"query-automations"`
	GetAutomation                 *sqlx.Stmt `query:"get-automation"`
	UpdateAutomation              *sqlx.Stmt `query:"update-automation"`
	UpdateAutomationStatus        *sqlx.Stmt `query:"update-automation-status"`
	DeleteAutomation              *sqlx.Stmt `query:"delete-automation"`
	CreateAutomationNode          *sqlx.Stmt `query:"create-automation-node"`
	GetAutomationNodes            *sqlx.Stmt `query:"get-automation-nodes"`
	GetAutomationNode             *sqlx.Stmt `query:"get-automation-node"`
	UpdateAutomationNode          *sqlx.Stmt `query:"update-automation-node"`
	DeleteAutomationNode          *sqlx.Stmt `query:"delete-automation-node"`
	CreateAutomationEdge          *sqlx.Stmt `query:"create-automation-edge"`
	GetAutomationEdges            *sqlx.Stmt `query:"get-automation-edges"`
	DeleteAutomationEdge          *sqlx.Stmt `query:"delete-automation-edge"`
	DeleteAutomationEdgesByAuto   *sqlx.Stmt `query:"delete-automation-edges-by-automation"`
	GetPendingAutoEnrollments     *sqlx.Stmt `query:"get-pending-automation-enrollments"`
	EnrollInAutomation            *sqlx.Stmt `query:"enroll-in-automation"`
	UpdateAutomationEnrollment    *sqlx.Stmt `query:"update-automation-enrollment"`

	// Contact scoring.
	CreateScoringRule      *sqlx.Stmt `query:"create-scoring-rule"`
	GetScoringRules        *sqlx.Stmt `query:"get-scoring-rules"`
	GetScoringRule         *sqlx.Stmt `query:"get-scoring-rule"`
	GetScoringRulesByEvent *sqlx.Stmt `query:"get-scoring-rules-by-event"`
	UpdateScoringRule      *sqlx.Stmt `query:"update-scoring-rule"`
	DeleteScoringRule      *sqlx.Stmt `query:"delete-scoring-rule"`
	UpdateSubscriberScore  *sqlx.Stmt `query:"update-subscriber-score"`
	InsertScoreLog         *sqlx.Stmt `query:"insert-score-log"`
	GetSubscriberScoreLog  string     `query:"get-subscriber-score-log"`
	DecayInactiveScores    string     `query:"decay-inactive-scores"`

	// CRM: deals and activities.
	CreateDeal              *sqlx.Stmt `query:"create-deal"`
	QueryDeals              string     `query:"query-deals"`
	GetDeal                 *sqlx.Stmt `query:"get-deal"`
	UpdateDeal              *sqlx.Stmt `query:"update-deal"`
	DeleteDeal              *sqlx.Stmt `query:"delete-deal"`
	GetDealPipeline         *sqlx.Stmt `query:"get-deal-pipeline"`
	CreateActivity          *sqlx.Stmt `query:"create-activity"`
	GetSubscriberActivities string     `query:"get-subscriber-activities"`
	DeleteActivity          *sqlx.Stmt `query:"delete-activity"`

	// Webhooks.
	CreateWebhook      *sqlx.Stmt `query:"create-webhook"`
	QueryWebhooks      string     `query:"query-webhooks"`
	GetWebhook         *sqlx.Stmt `query:"get-webhook"`
	UpdateWebhook      *sqlx.Stmt `query:"update-webhook"`
	DeleteWebhook      *sqlx.Stmt `query:"delete-webhook"`
	GetWebhooksByEvent *sqlx.Stmt `query:"get-webhooks-by-event"`
	InsertWebhookLog   *sqlx.Stmt `query:"insert-webhook-log"`
	QueryWebhookLog    string     `query:"query-webhook-log"`

	CreateRole            *sqlx.Stmt `query:"create-role"`
	GetUserRoles          *sqlx.Stmt `query:"get-user-roles"`
	GetListRoles          *sqlx.Stmt `query:"get-list-roles"`
	UpdateRole            *sqlx.Stmt `query:"update-role"`
	DeleteRole            *sqlx.Stmt `query:"delete-role"`
	UpsertListPermissions *sqlx.Stmt `query:"upsert-list-permissions"`
	DeleteListPermission  *sqlx.Stmt `query:"delete-list-permission"`

	// Companies (multi-tenant, v7.17.0+).
	GetCompanies     *sqlx.Stmt `query:"get-companies"`
	GetCompany       *sqlx.Stmt `query:"get-company"`
	CreateCompany    *sqlx.Stmt `query:"create-company"`
	UpdateCompany    *sqlx.Stmt `query:"update-company"`
	DeleteCompany    *sqlx.Stmt `query:"delete-company"`
	GetCompanyStats  *sqlx.Stmt `query:"get-company-stats"`
}

// compileSubscriberQueryTpl takes an arbitrary WHERE expressions
// to filter subscribers from the subscribers table and prepares a query
// out of it using the raw `query-subscribers-template` query template.
// While doing this, a readonly transaction is created and the query is
// dry run on it to ensure that it is indeed readonly.
func (q *Queries) compileSubscriberQueryTpl(searchStr, queryExp string, db *sqlx.DB, subStatus string) (string, error) {
	tx, err := db.BeginTxx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	// There's an arbitrary query condition.
	cond := "TRUE"
	if queryExp != "" {
		cond = queryExp
	}

	// Perform the dry run.
	stmt := strings.ReplaceAll(q.QuerySubscribersTpl, "%query%", cond)
	if _, err := tx.Exec(stmt, true, pq.Int64Array{}, subStatus, searchStr); err != nil {
		return "", err
	}

	return stmt, nil
}

// compileSubscriberQueryTpl takes an arbitrary WHERE expressions and a subscriber
// query template that depends on the filter (eg: delete by query, blocklist by query etc.)
// combines and executes them.
func (q *Queries) ExecSubQueryTpl(searchStr, queryExp, baseQueryTpl string, listIDs []int, db *sqlx.DB, subStatus string, args ...any) error {
	// Perform a dry run.
	filterExp, err := q.compileSubscriberQueryTpl(searchStr, queryExp, db, subStatus)
	if err != nil {
		return err
	}

	if len(listIDs) == 0 {
		listIDs = []int{}
	}

	// Insert the subscriber filter query into the target query.
	stmt := strings.ReplaceAll(baseQueryTpl, "%query%", filterExp)

	// First argument is the boolean indicating if the query is a dry run.
	a := append([]any{false, pq.Array(listIDs), subStatus, searchStr}, args...)

	// Execute the query on the DB.
	if _, err := db.Exec(stmt, a...); err != nil {
		return err
	}
	return nil
}
