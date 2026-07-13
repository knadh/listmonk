package manager

import (
	"testing"
	"time"

	"github.com/knadh/listmonk/models"
	null "gopkg.in/volatiletech/null.v6"
)

func TestPacedBatchLimit(t *testing.T) {
	now := time.Now()
	p := &pipe{
		camp: &models.Campaign{
			CampaignMeta: models.CampaignMeta{
				StartedAt: null.NewTime(now.Add(-1*time.Hour), true),
				ToSend:    10,
			},
			SendUntil: null.NewTime(now.Add(time.Hour), true),
		},
		m: &Manager{cfg: Config{BatchSize: 100}},
	}

	limit, wait := p.pacedBatchLimit()
	if wait != 0 {
		t.Fatalf("expected no wait at halfway point, got %s", wait)
	}
	if limit != 5 {
		t.Fatalf("expected 5 messages to be allowed at halfway point, got %d", limit)
	}

	p.queued.Store(5)
	limit, wait = p.pacedBatchLimit()
	if limit != 0 {
		t.Fatalf("expected no messages to be allowed after queueing allowance, got %d", limit)
	}
	if wait <= 0 {
		t.Fatalf("expected a positive wait for the next paced slot, got %s", wait)
	}
}

func TestPacedBatchLimitExpiredWindow(t *testing.T) {
	now := time.Now()
	p := &pipe{
		camp: &models.Campaign{
			CampaignMeta: models.CampaignMeta{
				StartedAt: null.NewTime(now.Add(-2*time.Hour), true),
				ToSend:    10,
			},
			SendUntil: null.NewTime(now.Add(-time.Hour), true),
		},
		m: &Manager{cfg: Config{BatchSize: 100}},
	}

	limit, wait := p.pacedBatchLimit()
	if wait != 0 {
		t.Fatalf("expected no wait after delivery window expiry, got %s", wait)
	}
	if limit != 100 {
		t.Fatalf("expected expired window to allow the configured batch size, got %d", limit)
	}
}
