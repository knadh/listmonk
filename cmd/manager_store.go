package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gofrs/uuid"
	"github.com/knadh/listmonk/internal/core"
	"github.com/knadh/listmonk/internal/media"
	"github.com/knadh/listmonk/internal/messenger"
	"github.com/knadh/listmonk/models"
	"github.com/lib/pq"
)

// runnerDB implements runner.DataSource over the primary
// database.
type runnerDB struct {
	queries *models.Queries
	core    *core.Core
	med     media.Store
	h       *http.Client
}

func newManagerStore(q *models.Queries, c *core.Core, m media.Store) *runnerDB {
	timeout := time.Second * 10

	return &runnerDB{
		queries: q,
		core:    c,
		med:     m,
		h: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConnsPerHost:   10,
				MaxConnsPerHost:       10,
				ResponseHeaderTimeout: timeout,
				IdleConnTimeout:       timeout,
			},
		},
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
func (r *runnerDB) NextSubscribers(campID, limit int) ([]models.Subscriber, error) {
	var out []models.Subscriber
	err := r.queries.NextCampaignSubscribers.Select(&out, campID, limit)
	return out, err
}

// GetCampaign fetches a campaign from the database.
func (r *runnerDB) GetCampaign(campID int) (*models.Campaign, error) {
	var out = &models.Campaign{}
	err := r.queries.GetCampaign.Get(out, campID, nil, "default")
	return out, err
}

// UpdateCampaignStatus updates a campaign's status.
func (r *runnerDB) UpdateCampaignStatus(campID int, status string) error {
	_, err := r.queries.UpdateCampaignStatus.Exec(campID, status)
	return err
}

// GetAttachment fetches a media attachment blob.
func (r *runnerDB) GetAttachment(mediaID int) (models.Attachment, error) {
	m, err := r.core.GetMedia(mediaID, "", r.med)
	if err != nil {
		return models.Attachment{}, err
	}

	fmt.Println(m.URL)

	resp, err := r.h.Get(m.URL)
	if err != nil {
		return models.Attachment{}, err
	}

	defer func() {
		// Drain and close the body to let the Transport reuse the connection
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return models.Attachment{}, err
	}

	return models.Attachment{
		Name:    m.Filename,
		Content: body,
		Header:  messenger.MakeAttachmentHeader(m.Filename, "base64"),
	}, nil
}

// CreateLink registers a URL with a UUID for tracking clicks and returns the UUID.
func (r *runnerDB) CreateLink(url string) (string, error) {
	// Create a new UUID for the URL. If the URL already exists in the DB
	// the UUID in the database is returned.
	uu, err := uuid.NewV4()
	if err != nil {
		return "", err
	}

	var out string
	if err := r.queries.CreateLink.Get(&out, uu, url); err != nil {
		return "", err
	}

	return out, nil
}

// RecordBounce records a bounce event and returns the bounce count.
func (r *runnerDB) RecordBounce(b models.Bounce) (int64, int, error) {
	var res = struct {
		SubscriberID int64 `db:"subscriber_id"`
		Num          int   `db:"num"`
	}{}

	err := r.queries.UpdateCampaignStatus.Select(&res,
		b.SubscriberUUID,
		b.Email,
		b.CampaignUUID,
		b.Type,
		b.Source,
		b.Meta)

	return res.SubscriberID, res.Num, err
}

func (r *runnerDB) BlocklistSubscriber(id int64) error {
	_, err := r.queries.BlocklistSubscribers.Exec(pq.Int64Array{id})
	return err
}

func (r *runnerDB) DeleteSubscriber(id int64) error {
	_, err := r.queries.DeleteSubscribers.Exec(pq.Int64Array{id})
	return err
}
