package core

import (
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/listmonk/internal/i18n"
	"github.com/knadh/listmonk/internal/testdb"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	null "gopkg.in/volatiletech/null.v6"
)

const (
	segListSingle = 101
	segListDouble = 102

	segQuery = "subscribers.attribs->>'tenant_id' = 'tenant_x' AND subscribers.attribs->>'lang' = 'tr'"
)

// Fixture subscribers and their expected eligibility:
//
//	1: tr/tenant_x, single list, confirmed          -> eligible, matches
//	2: tr/tenant_x, single list, unsubscribed       -> not eligible
//	3: tr/tenant_x, single list, blocklisted        -> not eligible
//	4: en/tenant_x, single list, unconfirmed        -> eligible (single opt-in), doesn't match
//	5: tr/tenant_x, no lists                        -> not eligible
//	6: tr/tenant_x, double list, confirmed          -> eligible, matches
//	7: tr/tenant_x, double list, unconfirmed        -> eligible only for opt-in campaigns
//	8: tr/acme, single list, confirmed              -> eligible, doesn't match
//	9: tr/tenant_x, single + double lists confirmed -> eligible, matches (deduplicated)
var segFixtures = []string{
	`INSERT INTO lists (id, uuid, name, type, optin) VALUES
		(101, gen_random_uuid(), 'Single', 'public', 'single'),
		(102, gen_random_uuid(), 'Double', 'public', 'double')`,
	`INSERT INTO subscribers (id, uuid, email, name, status, attribs)
		SELECT id, gen_random_uuid(), 'sub' || id || '@example.com', 'Sub ' || id, status::subscriber_status, attribs::JSONB
		FROM (VALUES
			(1, 'enabled', '{"lang": "tr", "tenant_id": "tenant_x"}'),
			(2, 'enabled', '{"lang": "tr", "tenant_id": "tenant_x"}'),
			(3, 'blocklisted', '{"lang": "tr", "tenant_id": "tenant_x"}'),
			(4, 'enabled', '{"lang": "en", "tenant_id": "tenant_x"}'),
			(5, 'enabled', '{"lang": "tr", "tenant_id": "tenant_x"}'),
			(6, 'enabled', '{"lang": "tr", "tenant_id": "tenant_x"}'),
			(7, 'enabled', '{"lang": "tr", "tenant_id": "tenant_x"}'),
			(8, 'enabled', '{"lang": "tr", "tenant_id": "acme"}'),
			(9, 'enabled', '{"lang": "tr", "tenant_id": "tenant_x", "address": {"city": "Ankara"}}')
		) AS s(id, status, attribs)`,
	`INSERT INTO subscriber_lists (subscriber_id, list_id, status) VALUES
		(1, 101, 'confirmed'), (2, 101, 'unsubscribed'), (3, 101, 'confirmed'), (4, 101, 'unconfirmed'),
		(6, 102, 'confirmed'), (7, 102, 'unconfirmed'), (8, 101, 'confirmed'),
		(9, 101, 'confirmed'), (9, 102, 'confirmed')`,
}

// newSegmentTestCore returns a Core on a fresh test database with the segment fixtures.
func newSegmentTestCore(t *testing.T) (*Core, *sqlx.DB) {
	t.Helper()

	db, qMap := testdb.New(t)
	testdb.Exec(t, db, segFixtures...)

	b, err := os.ReadFile(filepath.Join(testdb.RootDir(), "i18n", "en.json"))
	if err != nil {
		t.Fatal(err)
	}
	i, err := i18n.New(b)
	if err != nil {
		t.Fatal(err)
	}

	c := New(&Opt{
		DB:      db,
		Queries: testdb.PrepareQueries(t, qMap, db),
		I18n:    i,
		Log:     log.New(io.Discard, "", 0),
	}, &Hooks{})

	return c, db
}

func subIDs(subs []SegmentSubscriber) []int {
	out := make([]int, 0, len(subs))
	for _, s := range subs {
		out = append(out, s.ID)
	}
	return out
}

func assertHTTPError(t *testing.T, err error, code int) {
	t.Helper()

	var he *echo.HTTPError
	if !errors.As(err, &he) {
		t.Fatalf("expected *echo.HTTPError, got %T: %v", err, err)
	}
	if he.Code != code {
		t.Fatalf("expected HTTP %d, got %d: %v", code, he.Code, he.Message)
	}
}

func TestPreviewCampaignSegment(t *testing.T) {
	c, _ := newSegmentTestCore(t)

	cases := []struct {
		name     string
		lists    []int
		typ      string
		query    string
		matched  int
		total    int
		expected []int
	}{
		{"no segment matches all eligible", []int{segListSingle}, models.CampaignTypeRegular, "", 4, 4, []int{1, 4, 8, 9}},
		{"segment on single opt-in list", []int{segListSingle}, models.CampaignTypeRegular, segQuery, 2, 4, []int{1, 9}},
		{"segment across lists is deduplicated", []int{segListSingle, segListDouble}, models.CampaignTypeRegular, segQuery, 3, 5, []int{1, 6, 9}},
		{"double opt-in list only picks confirmed", []int{segListDouble}, models.CampaignTypeRegular, segQuery, 2, 2, []int{6, 9}},
		{"opt-in campaign only picks unconfirmed on double opt-in lists", []int{segListSingle, segListDouble}, models.CampaignTypeOptin, segQuery, 1, 1, []int{7}},
		{"ILIKE and other SQL", []int{segListSingle}, models.CampaignTypeRegular, "subscribers.email ILIKE 'SUB9%' OR subscribers.name = 'Sub 8'", 2, 4, []int{8, 9}},
		{"nested attribs", []int{segListSingle}, models.CampaignTypeRegular, "subscribers.attribs->'address'->>'city' = 'Ankara'", 1, 4, []int{9}},
		{"subquery on an allowed table", []int{segListSingle}, models.CampaignTypeRegular, "subscribers.id IN (SELECT subscriber_id FROM subscriber_lists WHERE list_id = 102)", 1, 4, []int{9}},
		{"no matches", []int{segListSingle}, models.CampaignTypeRegular, "subscribers.attribs->>'lang' = 'xx'", 0, 4, []int{}},
		{"literals with parentheses and comment markers", []int{segListSingle}, models.CampaignTypeRegular, "subscribers.name != ')' AND subscribers.email NOT LIKE '%--%' AND subscribers.id = 1", 1, 4, []int{1}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := c.PreviewCampaignSegment(tc.lists, tc.typ, tc.query, 20)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out.Matched != tc.matched || out.Total != tc.total {
				t.Errorf("got matched=%d total=%d; want matched=%d total=%d", out.Matched, out.Total, tc.matched, tc.total)
			}
			if got := subIDs(out.Results); !slices.Equal(got, tc.expected) {
				t.Errorf("got results %v; want %v", got, tc.expected)
			}
		})
	}
}

func TestPreviewCampaignSegmentResults(t *testing.T) {
	c, _ := newSegmentTestCore(t)

	// The sample is limited, while the count isn't.
	out, err := c.PreviewCampaignSegment([]int{segListSingle}, models.CampaignTypeRegular, "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if out.Matched != 4 || !slices.Equal(subIDs(out.Results), []int{1, 4}) {
		t.Errorf("got matched=%d results=%v; want matched=4 results=[1 4]", out.Matched, subIDs(out.Results))
	}

	// Out of range limits fall back to the max limit.
	for _, limit := range []int{0, -1, segmentPreviewMaxLimit + 1} {
		out, err := c.PreviewCampaignSegment([]int{segListSingle}, models.CampaignTypeRegular, "", limit)
		if err != nil {
			t.Fatal(err)
		}
		if len(out.Results) != 4 {
			t.Errorf("limit=%d: got %d results; want 4", limit, len(out.Results))
		}
	}

	// Sample subscribers carry their details.
	out, err = c.PreviewCampaignSegment([]int{segListSingle}, models.CampaignTypeRegular, "subscribers.id = 9", 1)
	if err != nil {
		t.Fatal(err)
	}
	s := out.Results[0]
	if s.Email != "sub9@example.com" || s.Name != "Sub 9" || s.Status != "enabled" || s.UUID == "" || s.Attribs["tenant_id"] != "tenant_x" {
		t.Errorf("unexpected subscriber in results: %+v", s)
	}
}

func TestPreviewCampaignSegmentErrors(t *testing.T) {
	c, db := newSegmentTestCore(t)

	cases := []struct {
		name  string
		query string
	}{
		{"break out of parentheses", "1=1) OR (1=1"},
		{"union break out", "x) UNION ALL (SELECT * FROM subscribers"},
		{"multiple statements", "subscribers.id = 1; DELETE FROM subscribers"},
		{"comment", "subscribers.id = 1 -- )"},
		{"disallowed table: users", "subscribers.id IN (SELECT id FROM users)"},
		{"disallowed table: settings", "(SELECT COUNT(*) FROM settings) > 0"},
		{"write in a CTE", "subscribers.id IN (WITH d AS (DELETE FROM subscribers RETURNING id) SELECT id FROM d)"},
		{"write via a function in a read-only transaction", "nextval('subscribers_id_seq') > 0"},
		{"syntax error", "subscribers.email ilke 'x%'"},
		{"unknown column", "subscribers.mail = 'x'"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := c.PreviewCampaignSegment([]int{segListSingle}, models.CampaignTypeRegular, tc.query, 10)
			assertHTTPError(t, err, http.StatusBadRequest)
		})
	}

	// Nothing should have been modified.
	var n int
	if err := db.Get(&n, "SELECT COUNT(*) FROM subscribers"); err != nil || n != 9 {
		t.Errorf("expected 9 subscribers after the queries, got %d (%v)", n, err)
	}
}

func TestValidateCampaignSegment(t *testing.T) {
	c, _ := newSegmentTestCore(t)

	for _, q := range []string{"", "   ", segQuery, "subscribers.email ILIKE 'sub%'"} {
		if err := c.ValidateCampaignSegment(q); err != nil {
			t.Errorf("ValidateCampaignSegment(%q) unexpected error: %v", q, err)
		}
	}

	for _, q := range []string{"1=1) OR (1=1", "subscribers.id IN (SELECT id FROM users)", "subscribers.mail = 'x'"} {
		assertHTTPError(t, c.ValidateCampaignSegment(q), http.StatusBadRequest)
	}
}

func TestCountCampaignSegment(t *testing.T) {
	c, _ := newSegmentTestCore(t)

	cases := []struct {
		lists    []int
		query    string
		maxSubID int
		expected int
	}{
		{[]int{segListSingle}, segQuery, 0, 2},
		{[]int{segListSingle}, segQuery, 1, 1},
		{[]int{segListSingle}, segQuery, 9, 2},
		{[]int{segListSingle, segListDouble}, segQuery, 0, 3},
		{[]int{segListSingle}, "", 0, 4},
	}
	for _, tc := range cases {
		n, err := c.CountCampaignSegment(tc.lists, models.CampaignTypeRegular, tc.query, tc.maxSubID)
		if err != nil {
			t.Fatal(err)
		}
		if n != tc.expected {
			t.Errorf("CountCampaignSegment(%v, %q, %d) = %d; want %d", tc.lists, tc.query, tc.maxSubID, n, tc.expected)
		}
	}

	if _, err := c.CountCampaignSegment([]int{segListSingle}, models.CampaignTypeRegular, "subscribers.id IN (SELECT id FROM users)", 0); err == nil {
		t.Error("expected an error for a disallowed table")
	}
}

func TestGetCampaignSegmentSubscribers(t *testing.T) {
	c, _ := newSegmentTestCore(t)
	lists := []int{segListSingle, segListDouble}

	// Batches of 2, advancing the last subscriber ID like the campaign manager does.
	var (
		lastID  = 0
		batches [][]int
	)
	for {
		subs, err := c.GetCampaignSegmentSubscribers(lists, models.CampaignTypeRegular, segQuery, lastID, 0, 2)
		if err != nil {
			t.Fatal(err)
		}
		if len(subs) == 0 {
			break
		}

		ids := []int{}
		for _, s := range subs {
			ids = append(ids, s.ID)
		}
		batches = append(batches, ids)
		lastID = subs[len(subs)-1].ID
	}
	if len(batches) != 2 || !slices.Equal(batches[0], []int{1, 6}) || !slices.Equal(batches[1], []int{9}) {
		t.Errorf("got batches %v; want [[1 6] [9]]", batches)
	}

	// max_subscriber_id caps the subscribers.
	subs, err := c.GetCampaignSegmentSubscribers(lists, models.CampaignTypeRegular, segQuery, 0, 6, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(subs) != 2 || subs[0].ID != 1 || subs[1].ID != 6 {
		t.Errorf("got %d subscribers with max_subscriber_id=6; want [1 6]", len(subs))
	}

	// The query runs in a read-only transaction.
	if _, err := c.GetCampaignSegmentSubscribers(lists, models.CampaignTypeRegular, "nextval('subscribers_id_seq') > 0", 0, 0, 10); err == nil {
		t.Error("expected an error for a write in a read-only transaction")
	}

	// Break outs are rejected even without the table validation.
	assertHTTPError(t, func() error {
		_, err := c.GetCampaignSegmentSubscribers(lists, models.CampaignTypeRegular, "1=1) OR (1=1", 0, 0, 10)
		return err
	}(), http.StatusBadRequest)
}

func TestCampaignSegmentQueryPersistence(t *testing.T) {
	c, _ := newSegmentTestCore(t)

	camp := models.Campaign{
		Type:         models.CampaignTypeRegular,
		Name:         "Segmented",
		Subject:      "Subject",
		FromEmail:    "from@example.com",
		Body:         "<p>Hello</p>",
		ContentType:  models.CampaignContentTypeHTML,
		Messenger:    "email",
		Headers:      models.Headers{},
		ArchiveMeta:  []byte("{}"),
		SegmentQuery: segQuery,
		AltBody:      null.String{},
	}

	created, err := c.CreateCampaign(camp, []int{segListSingle}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if created.SegmentQuery != segQuery {
		t.Errorf("created segment_query = %q; want %q", created.SegmentQuery, segQuery)
	}

	created.SegmentQuery = "subscribers.attribs->>'lang' = 'en'"
	updated, err := c.UpdateCampaign(created.ID, created, []int{segListSingle}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if updated.SegmentQuery != "subscribers.attribs->>'lang' = 'en'" {
		t.Errorf("updated segment_query = %q", updated.SegmentQuery)
	}

	// Clearing the segment.
	updated.SegmentQuery = ""
	cleared, err := c.UpdateCampaign(created.ID, updated, []int{segListSingle}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if cleared.SegmentQuery != "" {
		t.Errorf("cleared segment_query = %q; want empty", cleared.SegmentQuery)
	}
}
