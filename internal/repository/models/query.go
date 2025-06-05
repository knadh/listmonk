package models

import (
	"time"
)

// SortOrder represents the sort order for queries
type SortOrder string

const (
	SortAsc  SortOrder = "ASC"
	SortDesc SortOrder = "DESC"
)

// PaginationQuery represents common pagination parameters
type PaginationQuery struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Offset  int `json:"offset"`
}

// SortQuery represents sorting parameters
type SortQuery struct {
	Field string    `json:"field"`
	Order SortOrder `json:"order"`
}

// BaseQuery represents common query parameters
type BaseQuery struct {
	Pagination PaginationQuery `json:"pagination"`
	Sort       SortQuery       `json:"sort"`
}

// SubscriberQuery represents parameters for querying subscribers
type SubscriberQuery struct {
	BaseQuery
	Email         string   `json:"email"`
	Name          string   `json:"name"`
	Status        []string `json:"status"`
	ListIDs       []int64  `json:"list_ids"`
	SubscribedAt  *time.Time `json:"subscribed_at"`
	Query         string   `json:"query"`         // Raw SQL query for advanced filtering
	QueryParams   []interface{} `json:"query_params"` // Parameters for the raw query
}

// CampaignQuery represents parameters for querying campaigns
type CampaignQuery struct {
	BaseQuery
	Name     string   `json:"name"`
	Status   []string `json:"status"`
	Type     []string `json:"type"`
	ListIDs  []int64  `json:"list_ids"`
	Tags     []string `json:"tags"`
	Query    string   `json:"query"`
	QueryParams []interface{} `json:"query_params"`
}

// ListQuery represents parameters for querying lists
type ListQuery struct {
	BaseQuery
	Name   string `json:"name"`
	Type   string `json:"type"`
	Tags   []string `json:"tags"`
	Query  string `json:"query"`
	QueryParams []interface{} `json:"query_params"`
}

// TemplateQuery represents parameters for querying templates
type TemplateQuery struct {
	BaseQuery
	Name string   `json:"name"`
	Type []string `json:"type"`
	Tags []string `json:"tags"`
}

// UserQuery represents parameters for querying users
type UserQuery struct {
	BaseQuery
	Email  string `json:"email"`
	Name   string `json:"name"`
	Status string `json:"status"`
	RoleID int64  `json:"role_id"`
}

// MediaQuery represents parameters for querying media files
type MediaQuery struct {
	BaseQuery
	Filename string `json:"filename"`
	Provider string `json:"provider"`
}

// BounceQuery represents parameters for querying bounces
type BounceQuery struct {
	BaseQuery
	Email        string     `json:"email"`
	CampaignID   int64      `json:"campaign_id"`
	SubscriberID int64      `json:"subscriber_id"`
	Type         string     `json:"type"`
	Source       string     `json:"source"`
	CreatedAt    *time.Time `json:"created_at"`
}

// QueryResult represents the result of a paginated query
type QueryResult struct {
	Total   int64       `json:"total"`
	Page    int         `json:"page"`
	PerPage int         `json:"per_page"`
	Data    interface{} `json:"data"`
}

// Validate validates the BaseQuery and sets defaults
func (q *BaseQuery) Validate() {
	if q.Pagination.Page <= 0 {
		q.Pagination.Page = 1
	}
	if q.Pagination.PerPage <= 0 || q.Pagination.PerPage > 1000 {
		q.Pagination.PerPage = 20
	}
	q.Pagination.Offset = (q.Pagination.Page - 1) * q.Pagination.PerPage

	if q.Sort.Order != SortAsc && q.Sort.Order != SortDesc {
		q.Sort.Order = SortAsc
	}
}

// Validate validates the SubscriberQuery
func (q *SubscriberQuery) Validate() {
	q.BaseQuery.Validate()
	if q.Sort.Field == "" {
		q.Sort.Field = "id"
	}
}

// Validate validates the CampaignQuery
func (q *CampaignQuery) Validate() {
	q.BaseQuery.Validate()
	if q.Sort.Field == "" {
		q.Sort.Field = "created_at"
		q.Sort.Order = SortDesc
	}
}

// Validate validates the ListQuery
func (q *ListQuery) Validate() {
	q.BaseQuery.Validate()
	if q.Sort.Field == "" {
		q.Sort.Field = "name"
	}
}

// Validate validates the TemplateQuery
func (q *TemplateQuery) Validate() {
	q.BaseQuery.Validate()
	if q.Sort.Field == "" {
		q.Sort.Field = "name"
	}
}

// Validate validates the UserQuery
func (q *UserQuery) Validate() {
	q.BaseQuery.Validate()
	if q.Sort.Field == "" {
		q.Sort.Field = "email"
	}
}

// Validate validates the MediaQuery
func (q *MediaQuery) Validate() {
	q.BaseQuery.Validate()
	if q.Sort.Field == "" {
		q.Sort.Field = "filename"
	}
}

// Validate validates the BounceQuery
func (q *BounceQuery) Validate() {
	q.BaseQuery.Validate()
	if q.Sort.Field == "" {
		q.Sort.Field = "created_at"
		q.Sort.Order = SortDesc
	}
}