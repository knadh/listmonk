package main

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

// Queries contains all prepared SQL queries.
type Queries struct {
	UpsertSubscriber            *sqlx.Stmt `query:"upsert-subscriber"`
	GetSubscriber               *sqlx.Stmt `query:"get-subscriber"`
	GetSubscriberLists          *sqlx.Stmt `query:"get-subscriber-lists"`
	QuerySubscribers            string     `query:"query-subscribers"`
	QuerySubscribersCount       string     `query:"query-subscribers-count"`
	QuerySubscribersByList      string     `query:"query-subscribers-by-list"`
	QuerySubscribersByListCount string     `query:"query-subscribers-by-list-count"`
	UpdateSubscriber            *sqlx.Stmt `query:"update-subscriber"`
	DeleteSubscribers           *sqlx.Stmt `query:"delete-subscribers"`
	Unsubscribe                 *sqlx.Stmt `query:"unsubscribe"`
	QuerySubscribersIntoLists   string     `query:"query-subscribers-into-lists"`

	CreateList  *sqlx.Stmt `query:"create-list"`
	GetLists    *sqlx.Stmt `query:"get-lists"`
	UpdateList  *sqlx.Stmt `query:"update-list"`
	DeleteLists *sqlx.Stmt `query:"delete-lists"`

	CreateCampaign           *sqlx.Stmt `query:"create-campaign"`
	GetCampaigns             *sqlx.Stmt `query:"get-campaigns"`
	GetCampaignForPreview    *sqlx.Stmt `query:"get-campaign-for-preview"`
	GetCampaignStats         *sqlx.Stmt `query:"get-campaign-stats"`
	NextCampaigns            *sqlx.Stmt `query:"next-campaigns"`
	NextCampaignSubscribers  *sqlx.Stmt `query:"next-campaign-subscribers"`
	GetOneCampaignSubscriber *sqlx.Stmt `query:"get-one-campaign-subscriber"`
	UpdateCampaign           *sqlx.Stmt `query:"update-campaign"`
	UpdateCampaignStatus     *sqlx.Stmt `query:"update-campaign-status"`
	UpdateCampaignCounts     *sqlx.Stmt `query:"update-campaign-counts"`
	DeleteCampaign           *sqlx.Stmt `query:"delete-campaign"`

	CreateUser *sqlx.Stmt `query:"create-user"`
	GetUsers   *sqlx.Stmt `query:"get-users"`
	UpdateUser *sqlx.Stmt `query:"update-user"`
	DeleteUser *sqlx.Stmt `query:"delete-user"`

	InsertMedia *sqlx.Stmt `query:"insert-media"`
	GetMedia    *sqlx.Stmt `query:"get-media"`
	DeleteMedia *sqlx.Stmt `query:"delete-media"`

	CreateTemplate     *sqlx.Stmt `query:"create-template"`
	GetTemplates       *sqlx.Stmt `query:"get-templates"`
	UpdateTemplate     *sqlx.Stmt `query:"update-template"`
	SetDefaultTemplate *sqlx.Stmt `query:"set-default-template"`
	DeleteTemplate     *sqlx.Stmt `query:"delete-template"`

	// GetStats *sqlx.Stmt `query:"get-stats"`
}

// connectDB initializes a database connection.
func connectDB(host string, port int, user, pwd, dbName string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres",
		fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s", host, port, user, pwd, dbName))
	if err != nil {
		return nil, err
	}

	return db, nil
}
