package main

import (
	"log"

	"github.com/gofrs/uuid/v5"
	"github.com/knadh/listmonk/internal/core"
	"github.com/knadh/listmonk/internal/manager"
	"github.com/knadh/listmonk/internal/media"
	"github.com/knadh/listmonk/models"
	"github.com/lib/pq"
)

// store implements DataSource over the primary
// database.
type store struct {
	queries *models.Queries
	core    *core.Core
	media   media.Store
	log     *log.Logger
}

type runningCamp struct {
	CampaignID       int    `db:"campaign_id"`
	CampaignType     string `db:"campaign_type"`
	LastSubscriberID int    `db:"last_subscriber_id"`
	MaxSubscriberID  int    `db:"max_subscriber_id"`
	SegmentQuery     string `db:"segment_query"`
	ListID           int    `db:"list_id"`
}

func newManagerStore(q *models.Queries, c *core.Core, m media.Store, lo *log.Logger) *store {
	return &store{
		queries: q,
		core:    c,
		media:   m,
		log:     lo,
	}
}

// NextCampaigns retrieves active campaigns ready to be processed excluding
// campaigns that are also being processed. Additionally, it takes a map of campaignID:sentCount
// of campaigns that are being processed and updates them in the DB.
func (s *store) NextCampaigns(currentIDs []int64, sentCounts []int64) ([]*models.Campaign, error) {
	var out []*models.Campaign
	if err := s.queries.NextCampaigns.Select(&out, pq.Int64Array(currentIDs), pq.Int64Array(sentCounts)); err != nil {
		return nil, err
	}

	// next-campaigns computes to_send across all the subscribers in a campaign's lists.
	// For segmented campaigns, recompute it with the segment applied.
	res := out[:0]
	for _, c := range out {
		if c.SegmentQuery == "" {
			res = append(res, c)
			continue
		}

		if err := s.updateSegmentToSend(c); err != nil {
			// Pause the campaign as the segment query can't be run.
			s.log.Printf("error counting segment subscribers for campaign (%s). Pausing: %v", c.Name, err)
			if _, err := s.queries.UpdateCampaignStatus.Exec(c.ID, models.CampaignStatusPaused); err != nil {
				s.log.Printf("error pausing campaign (%s): %v", c.Name, err)
			}
			continue
		}

		res = append(res, c)
	}

	return res, nil
}

// updateSegmentToSend counts the subscribers matching a segmented campaign
// and updates the campaign's to_send count.
func (s *store) updateSegmentToSend(c *models.Campaign) error {
	var camps []runningCamp
	if err := s.queries.GetRunningCampaign.Select(&camps, c.ID); err != nil {
		return err
	}
	if len(camps) == 0 {
		return nil
	}

	listIDs := make([]int, 0, len(camps))
	for _, rc := range camps {
		listIDs = append(listIDs, rc.ListID)
	}

	n, err := s.core.CountCampaignSegment(listIDs, camps[0].CampaignType, camps[0].SegmentQuery, camps[0].MaxSubscriberID)
	if err != nil {
		return err
	}

	if _, err := s.queries.UpdateCampaignToSend.Exec(c.ID, n); err != nil {
		return err
	}
	c.ToSend = n

	return nil
}

// NextSubscribers retrieves a subset of subscribers of a given campaign.
// Since batches are processed sequentially, the retrieval is ordered by ID,
// and every batch takes the last ID of the last batch and fetches the next
// batch above that.
func (s *store) NextSubscribers(campID, limit int) ([]models.Subscriber, error) {
	var camps []runningCamp
	if err := s.queries.GetRunningCampaign.Select(&camps, campID); err != nil {
		return nil, err
	}

	var listIDs []int
	for _, c := range camps {
		listIDs = append(listIDs, c.ListID)
	}

	if len(listIDs) == 0 {
		return nil, nil
	}

	// Segmented campaign. Fetch the subscribers matching the segment (in a read-only transaction)
	// and advance the last_subscriber_id checkpoint separately.
	if c := camps[0]; c.SegmentQuery != "" {
		out, err := s.core.GetCampaignSegmentSubscribers(listIDs, c.CampaignType, c.SegmentQuery, c.LastSubscriberID, c.MaxSubscriberID, limit)
		if err != nil || len(out) == 0 {
			return out, err
		}

		if _, err := s.queries.UpdateCampaignCounts.Exec(c.CampaignID, 0, 0, out[len(out)-1].ID); err != nil {
			return nil, err
		}

		return out, nil
	}

	var out []models.Subscriber
	err := s.queries.NextCampaignSubscribers.Select(&out, camps[0].CampaignID, camps[0].CampaignType, camps[0].LastSubscriberID, camps[0].MaxSubscriberID, pq.Array(listIDs), limit)
	return out, err
}

// GetCampaign fetches a campaign from the database.
func (s *store) GetCampaign(campID int) (*models.Campaign, error) {
	var out = &models.Campaign{}
	err := s.queries.GetCampaign.Get(out, campID, nil, nil, "default")
	return out, err
}

// UpdateCampaignStatus updates a campaign's status.
func (s *store) UpdateCampaignStatus(campID int, status string) error {
	_, err := s.queries.UpdateCampaignStatus.Exec(campID, status)
	return err
}

// UpdateCampaignCounts updates a campaign's status.
func (s *store) UpdateCampaignCounts(campID int, toSend int, sent int, lastSubID int) error {
	_, err := s.queries.UpdateCampaignCounts.Exec(campID, toSend, sent, lastSubID)
	return err
}

// GetAttachment fetches a media attachment blob.
func (s *store) GetAttachment(mediaID int) (models.Attachment, error) {
	m, err := s.core.GetMedia(mediaID, "", "", s.media)
	if err != nil {
		return models.Attachment{}, err
	}

	b, err := s.media.GetBlob(m.URL)
	if err != nil {
		return models.Attachment{}, err
	}

	return models.Attachment{
		Name:    m.Filename,
		Content: b,
		Header:  manager.MakeAttachmentHeader(m.Filename, "base64", m.ContentType),
	}, nil
}

// GetInlineAttachmentByFilename fetches a media item by filename and returns
// it as an inline attachment along with the Content-ID value. The lookup is
// uniform across filesystem and S3 providers because both use the same media
// store interface; the first match for a given filename is returned.
func (s *store) GetInlineAttachmentByFilename(filename string) (models.Attachment, string, error) {
	m, err := s.core.GetMedia(0, "", filename, s.media)
	if err != nil {
		return models.Attachment{}, "", err
	}

	b, err := s.media.GetBlob(m.URL)
	if err != nil {
		return models.Attachment{}, "", err
	}

	cid := manager.MakeContentID(m.Filename)
	return models.Attachment{
		Name:     m.Filename,
		Content:  b,
		Header:   manager.MakeInlineAttachmentHeader(m.Filename, "", m.ContentType, cid),
		IsInline: true,
	}, cid, nil
}

// GetMediaURLByFilename returns the public URL of a media item by filename.
func (s *store) GetMediaURLByFilename(filename string) (string, error) {
	m, err := s.core.GetMedia(0, "", filename, s.media)
	if err != nil {
		return "", err
	}

	return m.URL, nil
}

// CreateLink registers a URL with a UUID for tracking clicks and returns the UUID.
func (s *store) CreateLink(url string) (string, error) {
	// Create a new UUID for the URL. If the URL already exists in the DB
	// the UUID in the database is returned.
	uu, err := uuid.NewV4()
	if err != nil {
		return "", err
	}

	var out string
	if err := s.queries.CreateLink.Get(&out, uu, url); err != nil {
		return "", err
	}

	return out, nil
}

// RecordBounce records a bounce event and returns the bounce count.
func (s *store) RecordBounce(b models.Bounce) (int64, int, error) {
	var res = struct {
		SubscriberID int64 `db:"subscriber_id"`
		Num          int   `db:"num"`
	}{}

	err := s.queries.UpdateCampaignStatus.Select(&res,
		b.SubscriberUUID,
		b.Email,
		b.CampaignUUID,
		b.Type,
		b.Source,
		b.Meta)

	return res.SubscriberID, res.Num, err
}

// BlocklistSubscriber blocklists a subscriber permanently.
func (s *store) BlocklistSubscriber(id int64) error {
	_, err := s.queries.BlocklistSubscribers.Exec(pq.Int64Array{id})
	return err
}

// DeleteSubscriber deletes a subscriber from the DB.
func (s *store) DeleteSubscriber(id int64) error {
	_, err := s.queries.DeleteSubscribers.Exec(pq.Int64Array{id})
	return err
}
