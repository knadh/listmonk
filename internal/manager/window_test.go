package manager

import (
	"io"
	"log"
	"math/rand/v2"
	"sync/atomic"
	"testing"
	"time"

	"github.com/knadh/listmonk/models"
)

func TestSlidingWindow(t *testing.T) {
	w := slidingWindow{rate: 3, duration: time.Hour}
	for _, tc := range []struct {
		try, want string
	}{
		// A burst within the hour goes out right away.
		{"18:55:00", "18:55:00"},
		{"18:55:30", "18:55:30"},
		{"18:56:00", "18:56:00"},

		// The window is full. Each further message waits until an earlier one
		// is an hour old, so no rolling hour has more than 3 messages.
		{"18:56:00", "19:55:00"},
		{"19:55:00", "19:55:30"},
		{"19:55:30", "19:56:00"},

		// After a quiet spell, there's room again right away.
		{"21:00:00", "21:00:00"},
	} {
		now, err := time.Parse(time.DateTime, "2026-10-27 "+tc.try)
		if err != nil {
			t.Fatal(err)
		}
		if wait := w.wait(now); wait > 0 {
			now = now.Add(wait)
		}
		w.add(now)

		if got := now.Format(time.TimeOnly); got != tc.want {
			t.Fatalf("message tried at %s was sent at %s, want %s", tc.try, got, tc.want)
		}
	}
}

// TestSlidingWindowLimit sends messages at random intervals and checks that no
// rolling window ever has more than rate messages, and that no message is held
// for longer than needed.
func TestSlidingWindowLimit(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	for _, rate := range []int{1, 2, 5, 30} {
		var (
			w    = slidingWindow{rate: rate, duration: time.Minute}
			now  = time.Date(2026, time.October, 27, 0, 0, 0, 0, time.UTC)
			sent []time.Time
		)
		for range 10000 {
			now = now.Add(time.Duration(r.IntN(3000)) * time.Millisecond)

			wait := w.wait(now)
			if len(w.sent) > rate {
				t.Fatalf("rate %d: window holds %d send times", rate, len(w.sent))
			}
			if wait > 0 {
				now = now.Add(wait)
			}
			w.add(now)
			sent = append(sent, now)

			// The latest rate+1 messages should span at least the window duration,
			// and exactly that if the last one had to wait.
			n := len(sent)
			if n <= rate {
				continue
			}
			if gap := now.Sub(sent[n-1-rate]); gap < w.duration || (wait > 0 && gap != w.duration) {
				t.Fatalf("rate %d: message %d was sent %v after message %d (waited %v)", rate, n, gap, n-rate, wait)
			}
		}
	}
}

// TestSlidingWindowStop checks that pausing a campaign that is waiting on the
// sliding window releases it right away instead of after the wait.
func TestSlidingWindowStop(t *testing.T) {
	var (
		st = &testStore{subs: []models.Subscriber{{Base: models.Base{ID: 1}}, {Base: models.Base{ID: 2}}}}
		ms = &testMessenger{}
		m  = New(Config{
			Concurrency: 1, MessageRate: 1, UnsubURL: "%s%s",
			SlidingWindow: true, SlidingWindowRate: 1, SlidingWindowDuration: time.Hour,
			ScanCampaigns: true, ScanInterval: 10 * time.Millisecond,
		}, st, nil, log.New(io.Discard, "", 0))
	)
	if err := m.AddMessenger(ms); err != nil {
		t.Fatal(err)
	}
	go m.Run()
	t.Cleanup(m.Close)

	// After the first message, the campaign waits an hour for the next one.
	waitFor(t, "first message", func() bool { return ms.sent.Load() == 1 })
	m.StopCampaign(1)
	waitFor(t, "campaign to stop", func() bool { return !m.HasRunningCampaigns() })

	if n := ms.sent.Load(); n != 1 {
		t.Fatalf("sent %d messages, want 1", n)
	}
}

func waitFor(t *testing.T, what string, ok func() bool) {
	t.Helper()
	for start := time.Now(); !ok(); time.Sleep(10 * time.Millisecond) {
		if time.Since(start) > 5*time.Second {
			t.Fatalf("timed out waiting for %s", what)
		}
	}
}

// testStore serves one running campaign. Methods that aren't overridden
// panic as the embedded Store is nil.
type testStore struct {
	Store
	subs  []models.Subscriber
	given atomic.Bool
}

func (s *testStore) NextCampaigns(_, _, _ []int64) ([]*models.Campaign, error) {
	if s.given.Swap(true) {
		return nil, nil
	}
	c := &models.Campaign{Name: "test", Status: models.CampaignStatusRunning, Messenger: "test", ContentType: models.CampaignContentTypePlain}
	c.ID = 1
	return []*models.Campaign{c}, nil
}

func (s *testStore) NextSubscribers(_, lastID, _ int) ([]models.Subscriber, error) {
	for i, sub := range s.subs {
		if sub.ID > lastID {
			return s.subs[i:], nil
		}
	}
	return nil, nil
}

func (s *testStore) UpdateCampaignCounts(int, int, int, int) error { return nil }

type testMessenger struct {
	sent atomic.Int64
}

func (m *testMessenger) Name() string { return "test" }
func (m *testMessenger) Push(models.Message) error {
	m.sent.Add(1)
	return nil
}
func (m *testMessenger) Flush() error { return nil }
func (m *testMessenger) Close() error { return nil }
