package repository

import (
	"context"
	"time"

	"github.com/knadh/listmonk/internal/auth"
	"github.com/knadh/listmonk/internal/media"
	querymodels "github.com/knadh/listmonk/internal/repository/models"
	"github.com/knadh/listmonk/models"
)

// SubscriberRepository defines operations for subscriber management
type SubscriberRepository interface {
	Create(ctx context.Context, sub *models.Subscriber) error
	GetByID(ctx context.Context, id int64) (*models.Subscriber, error)
	GetByUUID(ctx context.Context, uuid string) (*models.Subscriber, error)
	GetByEmail(ctx context.Context, email string) (*models.Subscriber, error)
	Update(ctx context.Context, sub *models.Subscriber) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, query querymodels.SubscriberQuery) ([]*models.Subscriber, int64, error)
	
	// Subscription management
	Subscribe(ctx context.Context, subscriberID int64, listID int64, status string) error
	Unsubscribe(ctx context.Context, subscriberID int64, listID int64) error
	GetSubscriptions(ctx context.Context, subscriberID int64) ([]*models.Subscription, error)
	
	// Bulk operations
	BulkUpdate(ctx context.Context, ids []int64, fields map[string]interface{}) error
	BulkDelete(ctx context.Context, ids []int64) error
	
	// Export
	ExportData(ctx context.Context, subscriberID int64) (*models.SubscriberExportProfile, error)
	
	// Statistics
	GetStats(ctx context.Context) (map[string]int64, error)
}

// CampaignRepository defines operations for campaign management
type CampaignRepository interface {
	Create(ctx context.Context, camp *models.Campaign) error
	GetByID(ctx context.Context, id int64) (*models.Campaign, error)
	GetByUUID(ctx context.Context, uuid string) (*models.Campaign, error)
	Update(ctx context.Context, camp *models.Campaign) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, query querymodels.CampaignQuery) ([]*models.Campaign, int64, error)
	
	// Status management
	UpdateStatus(ctx context.Context, id int64, status string) error
	
	// Statistics
	GetStats(ctx context.Context, id int64) (*models.CampaignStats, error)
	GetViewCounts(ctx context.Context, id int64) (map[string]int64, error)
	GetClickCounts(ctx context.Context, id int64) (map[string]int64, error)
	
	// Views and clicks
	RegisterView(ctx context.Context, campaignUUID, subscriberUUID string) error
	RegisterClick(ctx context.Context, campaignUUID, subscriberUUID, linkUUID string) error
	
	// Analytics
	GetAnalytics(ctx context.Context, id int64) (map[string]interface{}, error)
}

// ListRepository defines operations for list management
type ListRepository interface {
	Create(ctx context.Context, list *models.List) error
	GetByID(ctx context.Context, id int64) (*models.List, error)
	GetByUUID(ctx context.Context, uuid string) (*models.List, error)
	Update(ctx context.Context, list *models.List) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, query querymodels.ListQuery) ([]*models.List, int64, error)
	
	// Subscriber management
	GetSubscribers(ctx context.Context, listID int64, query querymodels.SubscriberQuery) ([]*models.Subscriber, int64, error)
	AddSubscriber(ctx context.Context, listID, subscriberID int64, status string) error
	RemoveSubscriber(ctx context.Context, listID, subscriberID int64) error
	
	// Statistics
	GetStats(ctx context.Context, id int64) (map[string]int64, error)
}

// TemplateRepository defines operations for template management
type TemplateRepository interface {
	Create(ctx context.Context, tpl *models.Template) error
	GetByID(ctx context.Context, id int64) (*models.Template, error)
	GetDefault(ctx context.Context, templateType string) (*models.Template, error)
	Update(ctx context.Context, tpl *models.Template) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, query querymodels.TemplateQuery) ([]*models.Template, int64, error)
	
	// Template compilation and rendering
	Render(ctx context.Context, templateID int64, data interface{}) (string, error)
}

// UserRepository defines operations for user management
type UserRepository interface {
	Create(ctx context.Context, user *auth.User) error
	GetByID(ctx context.Context, id int64) (*auth.User, error)
	GetByEmail(ctx context.Context, email string) (*auth.User, error)
	Update(ctx context.Context, user *auth.User) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, query querymodels.UserQuery) ([]*auth.User, int64, error)
	
	// Authentication
	Authenticate(ctx context.Context, email, password string) (*auth.User, error)
	UpdatePassword(ctx context.Context, id int64, password string) error
}

// RoleRepository defines operations for role and permission management
type RoleRepository interface {
	Create(ctx context.Context, role *auth.Role) error
	GetByID(ctx context.Context, id int64) (*auth.Role, error)
	Update(ctx context.Context, role *auth.Role) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context) ([]*auth.Role, error)
	
	// Permission management
	GetPermissions(ctx context.Context, roleID int64) ([]string, error)
	UpdatePermissions(ctx context.Context, roleID int64, permissions []string) error
}

// MediaRepository defines operations for media file management
type MediaRepository interface {
	Create(ctx context.Context, m *media.Media) error
	GetByID(ctx context.Context, id int64) (*media.Media, error)
	GetByUUID(ctx context.Context, uuid string) (*media.Media, error)
	Update(ctx context.Context, m *media.Media) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, query querymodels.MediaQuery) ([]*media.Media, int64, error)
}

// BounceRepository defines operations for bounce management
type BounceRepository interface {
	Create(ctx context.Context, bounce *models.Bounce) error
	GetByID(ctx context.Context, id int64) (*models.Bounce, error)
	List(ctx context.Context, query querymodels.BounceQuery) ([]*models.Bounce, int64, error)
	Delete(ctx context.Context, id int64) error
	
	// Bulk operations
	BulkDelete(ctx context.Context, ids []int64) error
	DeleteBySubscriber(ctx context.Context, subscriberID int64) error
	
	// Statistics
	GetStats(ctx context.Context) (map[string]int64, error)
}

// SettingsRepository defines operations for settings management
type SettingsRepository interface {
	Get(ctx context.Context, key string) (interface{}, error)
	Set(ctx context.Context, key string, value interface{}) error
	GetAll(ctx context.Context) (map[string]interface{}, error)
	SetMultiple(ctx context.Context, settings map[string]interface{}) error
	Delete(ctx context.Context, key string) error
}

// AnalyticsRepository defines operations for analytics and dashboard
type AnalyticsRepository interface {
	GetDashboardStats(ctx context.Context) (map[string]interface{}, error)
	GetCampaignAnalytics(ctx context.Context, campaignID int64, from, to time.Time) (map[string]interface{}, error)
	GetSubscriberGrowth(ctx context.Context, from, to time.Time) ([]map[string]interface{}, error)
	GetListStats(ctx context.Context) ([]map[string]interface{}, error)
	
	// Materialized view refresh (PostgreSQL specific, may be no-op for other DBs)
	RefreshMaterializedViews(ctx context.Context) error
}

// Transaction represents a database transaction
type Transaction interface {
	Commit() error
	Rollback() error
	
	// Repository access within transaction
	SubscriberRepo() SubscriberRepository
	CampaignRepo() CampaignRepository
	ListRepo() ListRepository
	TemplateRepo() TemplateRepository
	UserRepo() UserRepository
	RoleRepo() RoleRepository
	MediaRepo() MediaRepository
	BounceRepo() BounceRepository
	SettingsRepo() SettingsRepository
	AnalyticsRepo() AnalyticsRepository
}

// Manager provides access to all repositories and transaction management
type Manager interface {
	// Repository access
	SubscriberRepo() SubscriberRepository
	CampaignRepo() CampaignRepository
	ListRepo() ListRepository
	TemplateRepo() TemplateRepository
	UserRepo() UserRepository
	RoleRepo() RoleRepository
	MediaRepo() MediaRepository
	BounceRepo() BounceRepository
	SettingsRepo() SettingsRepository
	AnalyticsRepo() AnalyticsRepository
	
	// Transaction management
	BeginTx(ctx context.Context) (Transaction, error)
	
	// Health check
	Ping(ctx context.Context) error
	
	// Connection management
	Close() error
}