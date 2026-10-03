package manager

import (
	"log"
	"sync"
	"testing"

	"github.com/knadh/listmonk/models"
)

// mockStore implements Store, recording how the fetch cursor and the durable
// checkpoint are used.
type mockStore struct {
	mu sync.Mutex

	// Scripted subscriber batches returned by successive NextSubscribers calls.
	batches [][]models.Subscriber

	// Fetch cursors passed to successive NextSubscribers calls.
	gotCursors []int

	// Durable checkpoint writes (UpdateCampaignCounts lastSubID args).
	checkpointWrites []int
}

func (s *mockStore) NextCampaigns(currentIDs []int64, sentCounts []int64, lastSubIDs []int64) ([]*models.Campaign, error) {
	return nil, nil
}

func (s *mockStore) NextSubscribers(campID, lastFetchedID, limit int) ([]models.Subscriber, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gotCursors = append(s.gotCursors, lastFetchedID)
	if len(s.batches) == 0 {
		return nil, nil
	}
	out := s.batches[0]
	s.batches = s.batches[1:]
	return out, nil
}

func (s *mockStore) GetCampaign(campID int) (*models.Campaign, error) {
	return &models.Campaign{}, nil
}

func (s *mockStore) GetAttachment(mediaID int) (models.Attachment, error) {
	return models.Attachment{}, nil
}

func (s *mockStore) GetInlineAttachmentByFilename(filename string) (models.Attachment, string, error) {
	return models.Attachment{}, "", nil
}

func (s *mockStore) GetMediaURLByFilename(filename string) (string, error) {
	return "", nil
}

func (s *mockStore) UpdateCampaignStatus(campID int, status string) error { return nil }

func (s *mockStore) UpdateCampaignCounts(campID int, toSend int, sent int, lastSubID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkpointWrites = append(s.checkpointWrites, lastSubID)
	return nil
}

func (s *mockStore) CreateLink(url string) (string, error) { return "", nil }
func (s *mockStore) BlocklistSubscriber(id int64) error    { return nil }
func (s *mockStore) DeleteSubscriber(id int64) error       { return nil }

type mockMessenger struct{}

func (m mockMessenger) Name() string              { return "email" }
func (m mockMessenger) Push(models.Message) error { return nil }
func (m mockMessenger) Flush() error              { return nil }
func (m mockMessenger) Close() error              { return nil }

func newTestManager(store Store) *Manager {
	m := &Manager{
		cfg: Config{
			BatchSize: 2,
			UnsubURL:  "http://localhost:9000/subscription/%s/%s",
		},
		log:        log.New(testWriter{}, "", 0),
		messengers: map[string]Messenger{"email": mockMessenger{}},
		store:      store,
		pipes:      make(map[int]*pipe),
		links:      make(map[string]string),
		campMsgQ:   make(chan CampaignMessage, 100),
		msgQ:       make(chan models.Message, 100),
	}
	return m
}

type testWriter struct{}

func (testWriter) Write(b []byte) (int, error) { return len(b), nil }

func newTestCampaign(lastSubID int) *models.Campaign {
	c := &models.Campaign{
		UUID:             "camp-uuid",
		Type:             models.CampaignTypeRegular,
		Name:             "test",
		Subject:          "hi",
		FromEmail:        "test@example.com",
		Body:             "hello",
		Status:           models.CampaignStatusRunning,
		ContentType:      models.CampaignContentTypePlain,
		Messenger:        "email",
		LastSubscriberID: lastSubID,
	}
	c.ID = 1
	return c
}

func makeSubs(ids ...int) []models.Subscriber {
	out := make([]models.Subscriber, 0, len(ids))
	for _, id := range ids {
		s := models.Subscriber{UUID: "sub-uuid", Email: "u@example.com", Name: "u"}
		s.ID = id
		out = append(out, s)
	}
	return out
}

// Fetching subscriber batches must never advance the durable checkpoint:
// messages are sent asynchronously after the fetch, so a checkpoint that runs
// ahead of confirmed sends permanently skips subscribers on crash/restart.
// The pipe must instead track its own in-memory fetch cursor.
func TestNextSubscribersUsesInMemoryFetchCursor(t *testing.T) {
	store := &mockStore{
		batches: [][]models.Subscriber{
			makeSubs(1, 2),
			makeSubs(3, 4),
			{},
		},
	}
	m := newTestManager(store)

	p, err := m.newPipe(newTestCampaign(0))
	if err != nil {
		t.Fatalf("newPipe: %v", err)
	}

	for i, want := range []bool{true, true, false} {
		has, err := p.NextSubscribers()
		if err != nil {
			t.Fatalf("NextSubscribers call %d: %v", i, err)
		}
		if has != want {
			t.Fatalf("NextSubscribers call %d: has=%v, want %v", i, has, want)
		}
	}

	// The cursor passed to the store must advance in memory across batches:
	// seeded from the durable checkpoint (0), then the last ID of each batch.
	want := []int{0, 2, 4}
	if len(store.gotCursors) != len(want) {
		t.Fatalf("got %d fetch calls, want %d", len(store.gotCursors), len(want))
	}
	for i, w := range want {
		if store.gotCursors[i] != w {
			t.Fatalf("fetch call %d: cursor=%d, want %d", i, store.gotCursors[i], w)
		}
	}

	// The fetch path must not write the durable checkpoint at all.
	if len(store.checkpointWrites) != 0 {
		t.Fatalf("fetch path wrote the durable checkpoint %d times; it must only advance from the send side",
			len(store.checkpointWrites))
	}
}

// After a restart, a new pipe must resume fetching from the durable send-side
// checkpoint (last confirmed send), re-covering any subscribers whose messages
// were fetched but never sent by the previous process.
func TestNewPipeResumesFromDurableSendCheckpoint(t *testing.T) {
	store := &mockStore{batches: [][]models.Subscriber{makeSubs(8, 9)}}
	m := newTestManager(store)

	// Durable checkpoint says subscriber 7 was the last confirmed send.
	p, err := m.newPipe(newTestCampaign(7))
	if err != nil {
		t.Fatalf("newPipe: %v", err)
	}

	if _, err := p.NextSubscribers(); err != nil {
		t.Fatalf("NextSubscribers: %v", err)
	}

	if len(store.gotCursors) != 1 || store.gotCursors[0] != 7 {
		t.Fatalf("resume fetch started at cursor %v, want [7]", store.gotCursors)
	}
}

// The periodic campaign scan must flush the send-side watermark (last
// confirmed-sent subscriber ID) as a monotonic watermark, not a delta: unlike
// the sent counter, it must not reset between scans.
func TestGetCurrentCampaignsFlushesSendSideWatermark(t *testing.T) {
	m := newTestManager(&mockStore{})

	p, err := m.newPipe(newTestCampaign(0))
	if err != nil {
		t.Fatalf("newPipe: %v", err)
	}
	p.sent.Store(3)
	p.lastID.Store(5)

	ids, counts, lastIDs := m.getCurrentCampaigns()
	if len(ids) != 1 || len(counts) != 1 || len(lastIDs) != 1 {
		t.Fatalf("got ids=%v counts=%v lastIDs=%v", ids, counts, lastIDs)
	}
	if counts[0] != 3 || lastIDs[0] != 5 {
		t.Fatalf("got count=%d lastID=%d, want 3 and 5", counts[0], lastIDs[0])
	}

	// sent is a delta and resets; the watermark must not.
	_, counts, lastIDs = m.getCurrentCampaigns()
	if counts[0] != 0 || lastIDs[0] != 5 {
		t.Fatalf("second scan: got count=%d lastID=%d, want 0 and 5", counts[0], lastIDs[0])
	}
}
