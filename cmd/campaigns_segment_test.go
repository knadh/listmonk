package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/listmonk/internal/auth"
	"github.com/knadh/listmonk/internal/core"
	"github.com/knadh/listmonk/internal/i18n"
	"github.com/knadh/listmonk/internal/testdb"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

const (
	segQuery     = "subscribers.attribs->>'tenant_id' = 'tenant_x' AND subscribers.attribs->>'lang' = 'tr'"
	segListA     = 201
	segListB     = 202
	segBadTables = "subscribers.id IN (SELECT id FROM users)"
)

// Subscribers 1, 2, 4 match segQuery. 3 doesn't. 5 is blocklisted, 6 has unsubscribed.
// 7 matches but is only on list B.
var segAppFixtures = []string{
	`INSERT INTO lists (id, uuid, name, type, optin) VALUES
		(201, gen_random_uuid(), 'A', 'public', 'single'),
		(202, gen_random_uuid(), 'B', 'public', 'single')`,
	`INSERT INTO subscribers (id, uuid, email, name, status, attribs)
		SELECT id, gen_random_uuid(), 'sub' || id || '@example.com', 'Sub ' || id, status::subscriber_status, attribs::JSONB
		FROM (VALUES
			(1, 'enabled', '{"lang": "tr", "tenant_id": "tenant_x"}'),
			(2, 'enabled', '{"lang": "tr", "tenant_id": "tenant_x"}'),
			(3, 'enabled', '{"lang": "en", "tenant_id": "tenant_x"}'),
			(4, 'enabled', '{"lang": "tr", "tenant_id": "tenant_x"}'),
			(5, 'blocklisted', '{"lang": "tr", "tenant_id": "tenant_x"}'),
			(6, 'enabled', '{"lang": "tr", "tenant_id": "tenant_x"}'),
			(7, 'enabled', '{"lang": "tr", "tenant_id": "tenant_x"}')
		) AS s(id, status, attribs)`,
	`INSERT INTO subscriber_lists (subscriber_id, list_id, status) VALUES
		(1, 201, 'confirmed'), (2, 201, 'confirmed'), (3, 201, 'confirmed'), (4, 201, 'confirmed'),
		(5, 201, 'confirmed'), (6, 201, 'unsubscribed'), (7, 202, 'confirmed')`,
}

type segTestApp struct {
	app *App
	db  *sqlx.DB
	q   *models.Queries
}

func newSegTestApp(t *testing.T) *segTestApp {
	t.Helper()

	db, qMap := testdb.New(t)
	testdb.Exec(t, db, segAppFixtures...)
	q := testdb.PrepareQueries(t, qMap, db)

	b, err := os.ReadFile(filepath.Join(testdb.RootDir(), "i18n", "en.json"))
	if err != nil {
		t.Fatal(err)
	}
	i, err := i18n.New(b)
	if err != nil {
		t.Fatal(err)
	}

	lg := log.New(io.Discard, "", 0)
	co := core.New(&core.Opt{DB: db, Queries: q, I18n: i, Log: lg}, &core.Hooks{})

	return &segTestApp{
		app: &App{db: db, queries: q, core: co, i18n: i, log: lg},
		db:  db,
		q:   q,
	}
}

// newUser returns a user with the given permissions. Without list permissions,
// the user is given access to all lists.
func newUser(perms ...string) auth.User {
	u := auth.User{PermissionsMap: map[string]struct{}{}, ListPermissionsMap: map[int]map[string]struct{}{}}
	for _, p := range perms {
		u.PermissionsMap[p] = struct{}{}
	}
	return u
}

var allPerms = []string{
	auth.PermCampaignsGetAll, auth.PermCampaignsManageAll, auth.PermListGetAll, auth.PermListManageAll,
	auth.PermSubscribersGetAll, auth.PermSubscribersSqlQuery,
}

// newCtx returns an echo context for a JSON request with the given user and campaign ID.
func newCtx(method, target, body string, user auth.User, id int) (echo.Context, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	c := echo.New().NewContext(req, rec)
	c.Set(auth.UserHTTPCtxKey, user)
	if id > 0 {
		c.Set("id", id)
	}
	return c, rec
}

func httpCode(t *testing.T, err error) int {
	t.Helper()
	if err == nil {
		return http.StatusOK
	}

	var he *echo.HTTPError
	if !errors.As(err, &he) {
		t.Fatalf("expected *echo.HTTPError, got %T: %v", err, err)
	}
	return he.Code
}

type previewResp struct {
	Data core.SegmentPreview `json:"data"`
}

func decodePreview(t *testing.T, rec *httptest.ResponseRecorder) core.SegmentPreview {
	t.Helper()
	var out previewResp
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("error decoding response %q: %v", rec.Body.String(), err)
	}
	return out.Data
}

func resultIDs(p core.SegmentPreview) []int {
	out := []int{}
	for _, s := range p.Results {
		out = append(out, s.ID)
	}
	return out
}

// createCampaign inserts a campaign on the given lists with the given segment.
func (s *segTestApp) createCampaign(t *testing.T, name, segment string, listIDs ...int) models.Campaign {
	t.Helper()

	c, err := s.app.core.CreateCampaign(models.Campaign{
		Type: models.CampaignTypeRegular, Name: name, Subject: name, FromEmail: "from@example.com",
		Body: "<p>Hi</p>", ContentType: models.CampaignContentTypeHTML, Messenger: "email",
		Headers: models.Headers{}, ArchiveMeta: json.RawMessage("{}"), SegmentQuery: segment,
	}, listIDs, nil)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestFormatSQLExp(t *testing.T) {
	for in, out := range map[string]string{
		"":                    "",
		"  ":                  "",
		" subscribers.id = 1": "subscribers.id = 1",
		"subscribers.id = 1;": "subscribers.id = 1",
	} {
		if got := formatSQLExp(in); got != out {
			t.Errorf("formatSQLExp(%q) = %q; want %q", in, got, out)
		}
	}
}

func TestValidateCampaignSegment(t *testing.T) {
	s := newSegTestApp(t)

	var (
		admin   = newUser(allPerms...)
		noSQL   = newUser(auth.PermCampaignsManageAll, auth.PermListGetAll)
		oldQ    = "subscribers.attribs->>'lang' = 'en'"
		trimmed = segQuery
	)

	cases := []struct {
		name     string
		user     auth.User
		query    string
		old      string
		code     int
		expected string
	}{
		{"empty", noSQL, "", "", http.StatusOK, ""},
		{"empty clears an existing segment without the permission", noSQL, "  ", oldQ, http.StatusOK, ""},
		{"valid", admin, " " + segQuery + "; ", "", http.StatusOK, trimmed},
		{"unchanged segment is retained without the permission", noSQL, oldQ, oldQ, http.StatusOK, oldQ},
		{"new segment requires the permission", noSQL, segQuery, "", http.StatusForbidden, ""},
		{"changed segment requires the permission", noSQL, segQuery, oldQ, http.StatusForbidden, ""},
		{"disallowed table", admin, segBadTables, "", http.StatusBadRequest, ""},
		{"break out", admin, "1=1) OR (1=1", "", http.StatusBadRequest, ""},
		{"invalid SQL", admin, "subscribers.mail = 'x'", "", http.StatusBadRequest, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := newCtx(http.MethodPost, "/", "", tc.user, 0)
			got, err := s.app.validateCampaignSegment(c, tc.query, tc.old)
			if code := httpCode(t, err); code != tc.code {
				t.Fatalf("got HTTP %d (%v); want %d", code, err, tc.code)
			}
			if err == nil && got != tc.expected {
				t.Errorf("got %q; want %q", got, tc.expected)
			}
		})
	}
}

func TestPreviewCampaignSegmentHandler(t *testing.T) {
	s := newSegTestApp(t)

	listAOnly := newUser(auth.PermCampaignsManage, auth.PermSubscribersGet, auth.PermSubscribersSqlQuery)
	listAOnly.ListPermissionsMap[segListA] = map[string]struct{}{auth.PermListGet: {}}

	cases := []struct {
		name    string
		user    auth.User
		body    string
		code    int
		matched int
		total   int
		results []int
	}{
		{"segment", newUser(allPerms...), `{"lists": [201], "segment_query": "` + segQuery + `", "limit": 10}`, http.StatusOK, 3, 4, []int{1, 2, 4}},
		{"limit", newUser(allPerms...), `{"lists": [201], "segment_query": "` + segQuery + `", "limit": 2}`, http.StatusOK, 3, 4, []int{1, 2}},
		{"no segment", newUser(allPerms...), `{"lists": [201, 202]}`, http.StatusOK, 5, 5, []int{1, 2, 3, 4, 7}},
		{"trailing semicolon", newUser(allPerms...), `{"lists": [201], "segment_query": "subscribers.id = 3;"}`, http.StatusOK, 1, 4, []int{3}},
		{"lists filtered by permission", listAOnly, `{"lists": [201, 202], "segment_query": "` + segQuery + `"}`, http.StatusOK, 3, 4, []int{1, 2, 4}},
		{"no subscriber permission hides the sample", newUser(auth.PermCampaignsManageAll, auth.PermListGetAll, auth.PermSubscribersSqlQuery), `{"lists": [201], "segment_query": "` + segQuery + `"}`, http.StatusOK, 3, 4, []int{}},
		{"segment without sql_query permission", newUser(auth.PermCampaignsManageAll, auth.PermListGetAll, auth.PermSubscribersGetAll), `{"lists": [201], "segment_query": "` + segQuery + `"}`, http.StatusForbidden, 0, 0, nil},
		{"no segment without sql_query permission", newUser(auth.PermCampaignsManageAll, auth.PermListGetAll), `{"lists": [201]}`, http.StatusOK, 4, 4, []int{}},
		{"no lists", newUser(allPerms...), `{"lists": [], "segment_query": "` + segQuery + `"}`, http.StatusBadRequest, 0, 0, nil},
		{"no permitted lists", listAOnly, `{"lists": [202]}`, http.StatusBadRequest, 0, 0, nil},
		{"disallowed table", newUser(allPerms...), `{"lists": [201], "segment_query": "` + segBadTables + `"}`, http.StatusBadRequest, 0, 0, nil},
		{"break out", newUser(allPerms...), `{"lists": [201], "segment_query": "1=1) OR (1=1"}`, http.StatusBadRequest, 0, 0, nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, rec := newCtx(http.MethodPost, "/api/campaigns/segment/preview", tc.body, tc.user, 0)
			err := s.app.PreviewCampaignSegment(c)
			if code := httpCode(t, err); code != tc.code {
				t.Fatalf("got HTTP %d (%v); want %d", code, err, tc.code)
			}
			if err != nil {
				return
			}

			out := decodePreview(t, rec)
			if out.Matched != tc.matched || out.Total != tc.total || !slices.Equal(resultIDs(out), tc.results) {
				t.Errorf("got matched=%d total=%d results=%v; want matched=%d total=%d results=%v",
					out.Matched, out.Total, resultIDs(out), tc.matched, tc.total, tc.results)
			}
		})
	}
}

func TestPreviewCampaignSegmentByIDHandler(t *testing.T) {
	s := newSegTestApp(t)

	segmented := s.createCampaign(t, "Segmented", segQuery, segListA, segListB)
	plain := s.createCampaign(t, "Plain", "", segListA)

	// A user with only campaign read access can run the saved segment (it was validated
	// when it was saved), but doesn't see the sample subscribers.
	readOnly := newUser(auth.PermCampaignsGetAll, auth.PermListGetAll)

	cases := []struct {
		name    string
		user    auth.User
		id      int
		target  string
		matched int
		total   int
		results []int
	}{
		{"saved segment", newUser(allPerms...), segmented.ID, "/?limit=2", 4, 5, []int{1, 2}},
		{"no segment", newUser(allPerms...), plain.ID, "/", 4, 4, []int{1, 2, 3, 4}},
		{"read-only user", readOnly, segmented.ID, "/", 4, 5, []int{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, rec := newCtx(http.MethodGet, tc.target, "", tc.user, tc.id)
			if err := s.app.PreviewCampaignSegmentByID(c); err != nil {
				t.Fatal(err)
			}

			out := decodePreview(t, rec)
			if out.Matched != tc.matched || out.Total != tc.total || !slices.Equal(resultIDs(out), tc.results) {
				t.Errorf("got matched=%d total=%d results=%v; want matched=%d total=%d results=%v",
					out.Matched, out.Total, resultIDs(out), tc.matched, tc.total, tc.results)
			}
		})
	}

	// A user without access to the campaign's lists.
	noAccess := newUser(auth.PermCampaignsGet)
	c, _ := newCtx(http.MethodGet, "/", "", noAccess, segmented.ID)
	if code := httpCode(t, s.app.PreviewCampaignSegmentByID(c)); code != http.StatusForbidden {
		t.Errorf("got HTTP %d for a user without list access; want 403", code)
	}

	// Non-existent campaign.
	c, _ = newCtx(http.MethodGet, "/", "", newUser(allPerms...), 9999)
	if code := httpCode(t, s.app.PreviewCampaignSegmentByID(c)); code == http.StatusOK {
		t.Error("expected an error for a non-existent campaign")
	}
}

func TestViewCampaignSegmentTabRedirect(t *testing.T) {
	c, rec := newCtx(http.MethodGet, "/", "", newUser(allPerms...), 5)
	c.SetParamNames("tab")
	c.SetParamValues("segment")

	if err := (&App{}).ViewCampaign(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusFound || !strings.HasSuffix(rec.Header().Get("Location"), "/campaigns/5#tab=segment") {
		t.Errorf("got %d %q; want a redirect to the segment tab", rec.Code, rec.Header().Get("Location"))
	}
}

// startCampaign marks a campaign as running so that the manager picks it up.
func (s *segTestApp) startCampaign(t *testing.T, id int) {
	t.Helper()
	if _, err := s.db.Exec("UPDATE campaigns SET status='running' WHERE id=$1", id); err != nil {
		t.Fatal(err)
	}
}

func (s *segTestApp) campaignRow(t *testing.T, id int) (status string, toSend, lastSubID int) {
	t.Helper()
	if err := s.db.QueryRow("SELECT status, to_send, last_subscriber_id FROM campaigns WHERE id=$1", id).
		Scan(&status, &toSend, &lastSubID); err != nil {
		t.Fatal(err)
	}
	return
}

func TestManagerStoreSegmentedCampaign(t *testing.T) {
	s := newSegTestApp(t)
	st := newManagerStore(s.q, s.app.core, nil, log.New(io.Discard, "", 0))

	segmented := s.createCampaign(t, "Segmented", segQuery, segListA)
	plain := s.createCampaign(t, "Plain", "", segListA)
	s.startCampaign(t, segmented.ID)
	s.startCampaign(t, plain.ID)

	camps, err := st.NextCampaigns([]int64{}, []int64{})
	if err != nil {
		t.Fatal(err)
	}

	toSend := map[int]int{}
	for _, c := range camps {
		toSend[c.ID] = c.ToSend
	}
	if len(camps) != 2 || toSend[segmented.ID] != 3 {
		t.Fatalf("got campaigns with to_send %v; want 2 campaigns, segmented to_send=3", toSend)
	}

	// to_send is updated in the DB for the segmented campaign and untouched for the regular one.
	if _, n, _ := s.campaignRow(t, segmented.ID); n != 3 {
		t.Errorf("segmented campaign to_send = %d in DB; want 3", n)
	}
	if _, n, _ := s.campaignRow(t, plain.ID); n != 4 {
		t.Errorf("regular campaign to_send = %d in DB; want 4", n)
	}

	// Fetch the segmented campaign's subscribers in batches of 2.
	var got []int
	for range 5 {
		subs, err := st.NextSubscribers(segmented.ID, 2)
		if err != nil {
			t.Fatal(err)
		}
		if len(subs) == 0 {
			break
		}
		for _, sb := range subs {
			got = append(got, sb.ID)
		}
	}
	if !slices.Equal(got, []int{1, 2, 4}) {
		t.Errorf("segmented campaign subscribers = %v; want [1 2 4]", got)
	}
	if _, _, last := s.campaignRow(t, segmented.ID); last != 4 {
		t.Errorf("last_subscriber_id = %d; want 4", last)
	}

	// The regular campaign still gets all its eligible subscribers.
	subs, err := st.NextSubscribers(plain.ID, 100)
	if err != nil {
		t.Fatal(err)
	}
	got = got[:0]
	for _, sb := range subs {
		got = append(got, sb.ID)
	}
	if !slices.Equal(got, []int{1, 2, 3, 4}) {
		t.Errorf("regular campaign subscribers = %v; want [1 2 3 4]", got)
	}

	// Paused campaigns aren't fetched.
	if _, err := s.db.Exec("UPDATE campaigns SET status='paused', last_subscriber_id=0 WHERE id=$1", segmented.ID); err != nil {
		t.Fatal(err)
	}
	if subs, err := st.NextSubscribers(segmented.ID, 100); err != nil || len(subs) != 0 {
		t.Errorf("got %d subscribers (%v) for a paused campaign; want 0", len(subs), err)
	}
}

func TestManagerStorePausesInvalidSegment(t *testing.T) {
	s := newSegTestApp(t)
	st := newManagerStore(s.q, s.app.core, nil, log.New(io.Discard, "", 0))

	// A segment that has become invalid (eg: saved before validation rules changed).
	camp := s.createCampaign(t, "Bad segment", "", segListA)
	if _, err := s.db.Exec("UPDATE campaigns SET segment_query=$2 WHERE id=$1", camp.ID, segBadTables); err != nil {
		t.Fatal(err)
	}
	s.startCampaign(t, camp.ID)

	camps, err := st.NextCampaigns([]int64{}, []int64{})
	if err != nil {
		t.Fatal(err)
	}
	if len(camps) != 0 {
		t.Fatalf("got %d campaigns; want the invalid segmented campaign to be skipped", len(camps))
	}
	if status, _, _ := s.campaignRow(t, camp.ID); status != models.CampaignStatusPaused {
		t.Errorf("campaign status = %s; want paused", status)
	}
}

// Older API clients that don't know about segment_query must not alter a campaign's segment on update,
// as UpdateCampaign binds the request on top of the campaign in the DB.
func TestUpdateCampaignRequestBackwardCompat(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		expected string
	}{
		{"field omitted retains the segment", `{"name": "Renamed", "lists": [1]}`, segQuery},
		{"field set changes the segment", `{"name": "Renamed", "lists": [1], "segment_query": "subscribers.id = 1"}`, "subscribers.id = 1"},
		{"empty field clears the segment", `{"name": "Renamed", "lists": [1], "segment_query": ""}`, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o := campReq{Campaign: models.Campaign{Name: "Original", SegmentQuery: segQuery}}
			c, _ := newCtx(http.MethodPut, "/", tc.body, newUser(), 0)
			if err := c.Bind(&o); err != nil {
				t.Fatal(err)
			}
			if o.Name != "Renamed" || o.SegmentQuery != tc.expected {
				t.Errorf("got name=%q segment_query=%q; want name=Renamed segment_query=%q", o.Name, o.SegmentQuery, tc.expected)
			}
		})
	}

	// Campaigns created by older clients have no segment.
	var o campReq
	c, _ := newCtx(http.MethodPost, "/", `{"name": "New", "lists": [1]}`, newUser(), 0)
	if err := c.Bind(&o); err != nil {
		t.Fatal(err)
	}
	if o.SegmentQuery != "" {
		t.Errorf("segment_query = %q for a request without it; want empty", o.SegmentQuery)
	}
}
