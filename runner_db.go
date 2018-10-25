package main

import (
	"github.com/knadh/listmonk/models"
	"github.com/lib/pq"
)

// runnerDB implements runner.DataSource over the primary
// database.
type runnerDB struct {
	queries *Queries
}

func newRunnerDB(q *Queries) *runnerDB {
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

// PauseCampaign marks a campaign as paused.
func (r *runnerDB) PauseCampaign(campID int) error {
	_, err := r.queries.UpdateCampaignStatus.Exec(campID, models.CampaignStatusPaused)
	return err
}

// CancelCampaign marks a campaign as cancelled.
func (r *runnerDB) CancelCampaign(campID int) error {
	_, err := r.queries.UpdateCampaignStatus.Exec(campID, models.CampaignStatusCancelled)
	return err
}

// FinishCampaign marks a campaign as finished.
func (r *runnerDB) FinishCampaign(campID int) error {
	_, err := r.queries.UpdateCampaignStatus.Exec(campID, models.CampaignStatusFinished)
	return err
}
