package main

import (
	"github.com/knadh/listmonk/models"
	"github.com/lib/pq"
	uuid "github.com/satori/go.uuid"
)

// runnerDB implements runner.DataSource over the primary
// database.
type runnerDB struct {
	queries *Queries
}

func newManagerDB(q *Queries) *runnerDB {
	return &runnerDB{
		queries: q,
	}
}

// NextCampaigns retrieves active campaigns ready to be processed.
func (r *runnerDB) NextCampaigns(excludeIDs []int64) ([]*models.Campaign, error) {
	var out []*models.Campaign
	err := r.queries.NextCampaigns.Select(&out, pq.Int64Array(excludeIDs))
	return out, err
}

// NextSubscribers retrieves a subset of subscribers of a given campaign.
// Since batches are processed sequentially, the retrieval is ordered by ID,
// and every batch takes the last ID of the last batch and fetches the next
// batch above that.
func (r *runnerDB) NextSubscribers(campID, limit int) ([]*models.Subscriber, error) {
	var out []*models.Subscriber
	err := r.queries.NextCampaignSubscribers.Select(&out, campID, limit)
	return out, err
}

// GetCampaign fetches a campaign from the database.
func (r *runnerDB) GetCampaign(campID int) (*models.Campaign, error) {
	var out = &models.Campaign{}
	err := r.queries.GetCampaigns.Get(out, campID, "", 0, 1)
	return out, err
}

// UpdateCampaignStatus updates a campaign's status.
func (r *runnerDB) UpdateCampaignStatus(campID int, status string) error {
	_, err := r.queries.UpdateCampaignStatus.Exec(campID, status)
	return err
}

// CreateLink registers a URL with a UUID for tracking clicks and returns the UUID.
func (r *runnerDB) CreateLink(url string) (string, error) {
	// Create a new UUID for the URL. If the URL already exists in the DB
	// the UUID in the database is returned.
	var uu string
	if err := r.queries.CreateLink.Get(&uu, uuid.NewV4(), url); err != nil {
		return "", err
	}

	return uu, nil
}
