package repository

import (
	"context"
	"testing"

	"github.com/knadh/listmonk/internal/auth"
	"github.com/knadh/listmonk/internal/media"
	querymodels "github.com/knadh/listmonk/internal/repository/models"
	"github.com/knadh/listmonk/models"
	null "gopkg.in/volatiletech/null.v6"
)

// mockSubscriberRepository implements SubscriberRepository for testing
type mockSubscriberRepository struct{}

func (m *mockSubscriberRepository) Create(ctx context.Context, sub *models.Subscriber) error {
	return nil
}

func (m *mockSubscriberRepository) GetByID(ctx context.Context, id int64) (*models.Subscriber, error) {
	return &models.Subscriber{Base: models.Base{ID: int(id)}}, nil
}

func (m *mockSubscriberRepository) GetByUUID(ctx context.Context, uuid string) (*models.Subscriber, error) {
	return &models.Subscriber{UUID: uuid}, nil
}

func (m *mockSubscriberRepository) GetByEmail(ctx context.Context, email string) (*models.Subscriber, error) {
	return &models.Subscriber{Email: email}, nil
}

func (m *mockSubscriberRepository) Update(ctx context.Context, sub *models.Subscriber) error {
	return nil
}

func (m *mockSubscriberRepository) Delete(ctx context.Context, id int64) error {
	return nil
}

func (m *mockSubscriberRepository) List(ctx context.Context, query querymodels.SubscriberQuery) ([]*models.Subscriber, int64, error) {
	return []*models.Subscriber{}, 0, nil
}

func (m *mockSubscriberRepository) Subscribe(ctx context.Context, subscriberID int64, listID int64, status string) error {
	return nil
}

func (m *mockSubscriberRepository) Unsubscribe(ctx context.Context, subscriberID int64, listID int64) error {
	return nil
}

func (m *mockSubscriberRepository) GetSubscriptions(ctx context.Context, subscriberID int64) ([]*models.Subscription, error) {
	return []*models.Subscription{}, nil
}

func (m *mockSubscriberRepository) BulkUpdate(ctx context.Context, ids []int64, fields map[string]interface{}) error {
	return nil
}

func (m *mockSubscriberRepository) BulkDelete(ctx context.Context, ids []int64) error {
	return nil
}

func (m *mockSubscriberRepository) ExportData(ctx context.Context, subscriberID int64) (*models.SubscriberExportProfile, error) {
	return &models.SubscriberExportProfile{}, nil
}

func (m *mockSubscriberRepository) GetStats(ctx context.Context) (map[string]int64, error) {
	return map[string]int64{}, nil
}

// mockTransaction implements Transaction for testing
type mockTransaction struct{}

func (m *mockTransaction) Commit() error {
	return nil
}

func (m *mockTransaction) Rollback() error {
	return nil
}

func (m *mockTransaction) SubscriberRepo() SubscriberRepository {
	return &mockSubscriberRepository{}
}

func (m *mockTransaction) CampaignRepo() CampaignRepository {
	return nil
}

func (m *mockTransaction) ListRepo() ListRepository {
	return nil
}

func (m *mockTransaction) TemplateRepo() TemplateRepository {
	return nil
}

func (m *mockTransaction) UserRepo() UserRepository {
	return nil
}

func (m *mockTransaction) RoleRepo() RoleRepository {
	return nil
}

func (m *mockTransaction) MediaRepo() MediaRepository {
	return nil
}

func (m *mockTransaction) BounceRepo() BounceRepository {
	return nil
}

func (m *mockTransaction) SettingsRepo() SettingsRepository {
	return nil
}

func (m *mockTransaction) AnalyticsRepo() AnalyticsRepository {
	return nil
}

// mockManager implements Manager for testing
type mockManager struct{}

func (m *mockManager) SubscriberRepo() SubscriberRepository {
	return &mockSubscriberRepository{}
}

func (m *mockManager) CampaignRepo() CampaignRepository {
	return nil
}

func (m *mockManager) ListRepo() ListRepository {
	return nil
}

func (m *mockManager) TemplateRepo() TemplateRepository {
	return nil
}

func (m *mockManager) UserRepo() UserRepository {
	return nil
}

func (m *mockManager) RoleRepo() RoleRepository {
	return nil
}

func (m *mockManager) MediaRepo() MediaRepository {
	return nil
}

func (m *mockManager) BounceRepo() BounceRepository {
	return nil
}

func (m *mockManager) SettingsRepo() SettingsRepository {
	return nil
}

func (m *mockManager) AnalyticsRepo() AnalyticsRepository {
	return nil
}

func (m *mockManager) BeginTx(ctx context.Context) (Transaction, error) {
	return &mockTransaction{}, nil
}

func (m *mockManager) Ping(ctx context.Context) error {
	return nil
}

func (m *mockManager) Close() error {
	return nil
}

func TestSubscriberRepositoryInterface(t *testing.T) {
	var repo SubscriberRepository = &mockSubscriberRepository{}
	
	ctx := context.Background()
	
	// Test Create
	sub := &models.Subscriber{Email: "test@example.com"}
	err := repo.Create(ctx, sub)
	if err != nil {
		t.Errorf("Create failed: %v", err)
	}
	
	// Test GetByID
	result, err := repo.GetByID(ctx, 1)
	if err != nil {
		t.Errorf("GetByID failed: %v", err)
	}
	if result.ID != 1 {
		t.Errorf("Expected ID 1, got %d", result.ID)
	}
	
	// Test GetByEmail
	result, err = repo.GetByEmail(ctx, "test@example.com")
	if err != nil {
		t.Errorf("GetByEmail failed: %v", err)
	}
	if result.Email != "test@example.com" {
		t.Errorf("Expected email test@example.com, got %s", result.Email)
	}
	
	// Test List
	query := querymodels.SubscriberQuery{}
	subscribers, total, err := repo.List(ctx, query)
	if err != nil {
		t.Errorf("List failed: %v", err)
	}
	if subscribers == nil {
		t.Error("Expected non-nil subscribers slice")
	}
	if total != 0 {
		t.Errorf("Expected total 0, got %d", total)
	}
}

func TestTransactionInterface(t *testing.T) {
	var tx Transaction = &mockTransaction{}
	
	// Test Commit
	err := tx.Commit()
	if err != nil {
		t.Errorf("Commit failed: %v", err)
	}
	
	// Test Rollback
	err = tx.Rollback()
	if err != nil {
		t.Errorf("Rollback failed: %v", err)
	}
	
	// Test repository access
	repo := tx.SubscriberRepo()
	if repo == nil {
		t.Error("Expected non-nil SubscriberRepository")
	}
}

func TestManagerInterface(t *testing.T) {
	var manager Manager = &mockManager{}
	
	ctx := context.Background()
	
	// Test repository access
	repo := manager.SubscriberRepo()
	if repo == nil {
		t.Error("Expected non-nil SubscriberRepository")
	}
	
	// Test transaction creation
	tx, err := manager.BeginTx(ctx)
	if err != nil {
		t.Errorf("BeginTx failed: %v", err)
	}
	if tx == nil {
		t.Error("Expected non-nil Transaction")
	}
	
	// Test ping
	err = manager.Ping(ctx)
	if err != nil {
		t.Errorf("Ping failed: %v", err)
	}
	
	// Test close
	err = manager.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestRepositoryInterfaces(t *testing.T) {
	// Test that interface types compile correctly by creating variables
	var (
		_ SubscriberRepository = (*mockSubscriberRepository)(nil)
		_ Transaction         = (*mockTransaction)(nil)
		_ Manager            = (*mockManager)(nil)
	)
}

// Test that the interfaces work with the expected model types
func TestModelTypes(t *testing.T) {
	ctx := context.Background()
	
	// Test with actual model instances
	subscriber := &models.Subscriber{
		Base:  models.Base{ID: 1},
		Email: "test@example.com",
		Name:  "Test User",
	}
	
	user := &auth.User{
		Base:     auth.Base{ID: 1},
		Username: "testuser",
		Email:    null.NewString("test@example.com", true),
	}
	
	role := &auth.Role{
		Base: auth.Base{ID: 1},
		Name: null.NewString("Test Role", true),
	}
	
	mediaItem := &media.Media{
		ID:       1,
		Filename: "test.jpg",
		URL:      "http://example.com/test.jpg",
	}
	
	bounce := &models.Bounce{
		ID:    1,
		Email: "bounce@example.com",
		Type:  "hard",
	}
	
	// Test that we can pass these to repository methods
	var (
		subRepo   SubscriberRepository = &mockSubscriberRepository{}
		userRepo  UserRepository       = &mockUserRepository{}
		roleRepo  RoleRepository       = &mockRoleRepository{}
		mediaRepo MediaRepository      = &mockMediaRepository{}
		bounceRepo BounceRepository    = &mockBounceRepository{}
	)
	
	// These should compile without errors
	_ = subRepo.Create(ctx, subscriber)
	_ = userRepo.Create(ctx, user)
	_ = roleRepo.Create(ctx, role)
	_ = mediaRepo.Create(ctx, mediaItem)
	_ = bounceRepo.Create(ctx, bounce)
}

// Additional mock repositories for testing
type mockUserRepository struct{}

func (m *mockUserRepository) Create(ctx context.Context, user *auth.User) error { return nil }
func (m *mockUserRepository) GetByID(ctx context.Context, id int64) (*auth.User, error) { return nil, nil }
func (m *mockUserRepository) GetByEmail(ctx context.Context, email string) (*auth.User, error) { return nil, nil }
func (m *mockUserRepository) Update(ctx context.Context, user *auth.User) error { return nil }
func (m *mockUserRepository) Delete(ctx context.Context, id int64) error { return nil }
func (m *mockUserRepository) List(ctx context.Context, query querymodels.UserQuery) ([]*auth.User, int64, error) { return nil, 0, nil }
func (m *mockUserRepository) Authenticate(ctx context.Context, email, password string) (*auth.User, error) { return nil, nil }
func (m *mockUserRepository) UpdatePassword(ctx context.Context, id int64, password string) error { return nil }

type mockRoleRepository struct{}

func (m *mockRoleRepository) Create(ctx context.Context, role *auth.Role) error { return nil }
func (m *mockRoleRepository) GetByID(ctx context.Context, id int64) (*auth.Role, error) { return nil, nil }
func (m *mockRoleRepository) Update(ctx context.Context, role *auth.Role) error { return nil }
func (m *mockRoleRepository) Delete(ctx context.Context, id int64) error { return nil }
func (m *mockRoleRepository) List(ctx context.Context) ([]*auth.Role, error) { return nil, nil }
func (m *mockRoleRepository) GetPermissions(ctx context.Context, roleID int64) ([]string, error) { return nil, nil }
func (m *mockRoleRepository) UpdatePermissions(ctx context.Context, roleID int64, permissions []string) error { return nil }

type mockMediaRepository struct{}

func (m *mockMediaRepository) Create(ctx context.Context, media *media.Media) error { return nil }
func (m *mockMediaRepository) GetByID(ctx context.Context, id int64) (*media.Media, error) { return nil, nil }
func (m *mockMediaRepository) GetByUUID(ctx context.Context, uuid string) (*media.Media, error) { return nil, nil }
func (m *mockMediaRepository) Update(ctx context.Context, media *media.Media) error { return nil }
func (m *mockMediaRepository) Delete(ctx context.Context, id int64) error { return nil }
func (m *mockMediaRepository) List(ctx context.Context, query querymodels.MediaQuery) ([]*media.Media, int64, error) { return nil, 0, nil }

type mockBounceRepository struct{}

func (m *mockBounceRepository) Create(ctx context.Context, bounce *models.Bounce) error { return nil }
func (m *mockBounceRepository) GetByID(ctx context.Context, id int64) (*models.Bounce, error) { return nil, nil }
func (m *mockBounceRepository) List(ctx context.Context, query querymodels.BounceQuery) ([]*models.Bounce, int64, error) { return nil, 0, nil }
func (m *mockBounceRepository) Delete(ctx context.Context, id int64) error { return nil }
func (m *mockBounceRepository) BulkDelete(ctx context.Context, ids []int64) error { return nil }
func (m *mockBounceRepository) DeleteBySubscriber(ctx context.Context, subscriberID int64) error { return nil }
func (m *mockBounceRepository) GetStats(ctx context.Context) (map[string]int64, error) { return nil, nil }