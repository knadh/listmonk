package subimporter

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/knadh/listmonk/internal/i18n"
	"github.com/knadh/listmonk/models"
	"github.com/lib/pq"
)

// Pure parsing/validation tests run without PostgreSQL. To also run the SQL
// integration tests, set LISTMONK_TEST_DB to a PostgreSQL DSN. Every connection
// uses temporary tables and the production queries; no persistent data is changed.

// importerDB keeps all statements on one connection, where temporary tables
// shadow the application tables. SQL and enum types match the production writes.
type importerDB struct {
	db  *sql.DB
	opt Options
	i   *i18n.I18n
}

type importSubscriber struct {
	UUID      string
	Name      string
	Attribs   models.JSON
	Status    string
	Updated   time.Time
	Lists     map[int]string
	ListDates map[int]time.Time
}

func TestImportScopeJSON(t *testing.T) {
	f := func(raw string) {
		t.Helper()

		opt := SessionOpt{PermittedListIDs: []int{1, 3}}
		if err := json.Unmarshal([]byte(raw), &opt); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(opt.PermittedListIDs, []int{1, 3}) {
			t.Fatalf("JSON changed scope: %s", raw)
		}

		b, err := json.Marshal(opt)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(strings.ToLower(string(b)), "permitted") {
			t.Fatalf("scope leaked into JSON: %s", b)
		}
	}

	f(`{"PermittedListIDs":null}`)
	f(`{"PermittedListIDs":[2]}`)
	f(`{"permitted_list_ids":null}`)
	f(`{"permitted_list_ids":[2]}`)
	f(`{"permittedListIDs":[2]}`)
	f(`{"PermittedListIDs":[],"lists":[2]}`)
}

func TestImportSessionLifecycle(t *testing.T) {
	im := New(Options{}, nil, nil)
	s := importerSession(t, im, SessionOpt{Filename: "first.csv"})
	if got := im.GetStats(); got.Status != StatusImporting || got.Name != "first.csv" {
		t.Fatalf("stats: %+v", got)
	}
	if _, err := im.NewSession(SessionOpt{}); err == nil {
		t.Fatal("concurrent session accepted")
	}

	im.Stop()
	im.Stop() // Repeated stop requests must not block.
	if im.GetStats().Status != StatusStopping {
		t.Fatal("stop did not change status")
	}
	if _, err := im.NewSession(SessionOpt{}); err == nil {
		t.Fatal("session accepted while stopping")
	}

	p := importerFile(t, "stop.csv", []byte("email\nstopped@example.com\n"))
	if err := s.LoadCSV(p, ','); err != nil {
		t.Fatal(err)
	}
	if len(s.subQueue) != 0 || s.failed.Load() {
		t.Fatal("stop queued a row or marked the load failed")
	}

	// A late stop signal must not stop the next import.
	im.setStatus(StatusFinished)
	im.stop <- true
	s = importerSession(t, im, SessionOpt{Filename: "second.csv"})
	if err := s.LoadCSV(p, ','); err != nil {
		t.Fatal(err)
	}
	if len(s.subQueue) != 1 {
		t.Fatal("stale stop request affected next import")
	}

	im.setStatus(StatusFinished)
	im.Stop()
	if got := im.GetStats(); got.Status != StatusNone || got.Imported != 0 || got.Total != 0 || got.Name != "" || len(im.GetLogs()) != 0 {
		t.Fatalf("clear did not reset stats/logs: %+v", got)
	}

	if err := s.LoadCSV(p, ','); !errors.Is(err, ErrIsImporting) {
		t.Fatalf("inactive CSV load: %v", err)
	}
	if _, _, err := s.ExtractZIP(p, 1); !errors.Is(err, ErrIsImporting) {
		t.Fatalf("inactive ZIP extraction: %v", err)
	}
}

func TestImportValidateFields(t *testing.T) {
	im := New(Options{}, nil, importerI18n(t))

	f := func(email, name, wantEmail, wantName string, valid bool) {
		t.Helper()

		got, err := im.ValidateFields(SubReq{Subscriber: models.Subscriber{Email: email, Name: name}})
		if (err == nil) != valid {
			t.Fatalf("ValidateFields(%q): %v", email, err)
		}
		if valid && (got.Email != wantEmail || got.Name != wantName) {
			t.Fatalf("ValidateFields(%q, %q) = %q, %q", email, name, got.Email, got.Name)
		}
	}

	f(" PERSON@EXAMPLE.COM ", " Alice ", "person@example.com", "Alice", true)
	f("jane.doe@example.com", "", "jane.doe@example.com", "Jane Doe", true)
	f("person+tag@example.com", " 日本語 ", "person+tag@example.com", "日本語", true)
	f("", "", "", "", false)
	f("not-an-email", "", "", "", false)
	f("a@", "", "", "", false)
	f("a@example.com\nb@example.com", "", "", "", false)
	f(strings.Repeat("a", 1001)+"@example.com", "", "", "", false)
}

func TestImportDomainRules(t *testing.T) {
	i := importerI18n(t)

	f := func(allow, block []string, email string, valid bool) {
		t.Helper()

		im := New(Options{DomainAllowlist: allow, DomainBlocklist: block}, nil, i)
		_, err := im.SanitizeEmail(email)
		if (err == nil) != valid {
			t.Fatalf("allow=%v block=%v email=%q: %v", allow, block, email, err)
		}
	}

	f(nil, nil, "a@example.com", true)
	f(nil, []string{"example.com"}, "A@EXAMPLE.COM", false)
	f(nil, []string{"example.com"}, "a@mail.example.com", true)
	f(nil, []string{"*.example.com"}, "a@example.com", false)
	f(nil, []string{"*.example.com"}, "a@mail.example.com", false)
	f(nil, []string{"*.example.com"}, "a@other.com", true)
	f([]string{"example.com"}, nil, "a@example.com", true)
	f([]string{"example.com"}, nil, "a@other.com", false)
	f([]string{"*.example.com"}, nil, "a@example.com", true)
	f([]string{"*.example.com"}, nil, "a@mail.example.com", true)
	f([]string{"example.com"}, []string{"example.com"}, "a@example.com", true) // Allowlist takes precedence.
	f([]string{"example.com"}, []string{"other.com"}, "a@third.com", false)
}

func TestImportCSV(t *testing.T) {
	i := importerI18n(t)

	f := func(name, input string, delim rune, wantErr bool, total int, log string, want ...SubReq) {
		t.Helper()

		t.Run(name, func(t *testing.T) {
			im := New(Options{}, nil, i)
			s := importerSession(t, im, SessionOpt{Filename: name})
			err := s.LoadCSV(importerFile(t, "input.csv", []byte(input)), delim)
			if (err != nil) != wantErr || s.failed.Load() != wantErr {
				t.Fatalf("load error=%v, failed=%v", err, s.failed.Load())
			}

			var got []SubReq
			for sub := range s.subQueue {
				got = append(got, sub)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("rows=%+v, want %+v", got, want)
			}
			if im.GetStats().Total != total {
				t.Fatalf("total=%d, want %d", im.GetStats().Total, total)
			}
			if log != "" && !strings.Contains(string(im.GetLogs()), log) {
				t.Fatalf("missing %q in logs: %s", log, im.GetLogs())
			}
		})
	}
	row := func(email, name string, attribs models.JSON) SubReq {
		return SubReq{Subscriber: models.Subscriber{Email: email, Name: name, Attribs: attribs}}
	}
	a := row("a@example.com", "A", nil)
	f("empty", "", ',', true, 0, "")
	f("header only", "email,name\n", ',', false, 0, "")
	f("missing email", "name\nA\n", ',', true, 1, "'email' column not found")
	f("wrong delimiter", "email,name\na@example.com,A\n", '|', true, 1, "'email' column not found")

	f("semicolon", "email;name\na@example.com;A\n", ';', false, 1, "", a)
	f("tab", "email\tname\na@example.com\tA\n", '\t', false, 1, "", a)
	f("BOM", "\ufeffemail,name\na@example.com,A\n", ',', false, 1, "", a)
	f("CRLF", "email,name\r\na@example.com,A\r\n", ',', false, 1, "", a)
	f("no final newline", "email,name\na@example.com,A", ',', false, 1, "", a)
	f("blank lines", "email,name\n\na@example.com,A\n\n", ',', false, 3, "", a)

	f("reordered headers", " name ,extra, email \nA,ignored,a@example.com\n", ',', false, 1, "ignoring unknown header", a)
	f("email only", "email\na@example.com\n", ',', false, 1, "", a)
	f("normalization", "email,name\n A@EXAMPLE.COM , A \n", ',', false, 1, "", a)
	f("unknown lists ignored", "email,lists\na@example.com,2\n", ',', false, 1, "ignoring unknown header", a)

	f("invalid email", "email\ninvalid\na@example.com\n", ',', false, 2, "skipping line", a)
	f("too many fields", "email,name\nb@example.com,B,extra\na@example.com,A\n", ',', false, 2, "skipping line", a)
	f("too few fields", "email,name\nb@example.com\na@example.com,A\n", ',', false, 2, "skipping line", a)
	f("invalid UTF8", "email,name\nb@example.com,\xff\na@example.com,A\n", ',', false, 2, "skipping line", a)
	f("NUL", "email,name\nb@example.com,B\x00\na@example.com,A\n", ',', false, 2, "skipping line", a)

	f("quoted comma", "email,name\na@example.com,\"A, B\"\n", ',', false, 1, "", row("a@example.com", "A, B", nil))
	f("multiline name", "email,name\na@example.com,\"A\nB\"\n", ',', false, 2, "", row("a@example.com", "A\nB", nil))

	f("attributes", "email,attributes\na@example.com,\"{\"\"city\"\":\"\"Bengaluru\"\"}\"\n", ',', false, 1, "", row("a@example.com", "A", models.JSON{"city": "Bengaluru"}))
	f("invalid JSON", "email,attributes\na@example.com,{bad\n", ',', false, 1, "skipping invalid attributes JSON", a)
	f("non-object JSON", "email,attributes\na@example.com,[]\n", ',', false, 1, "skipping invalid attributes JSON", a)

	f("malformed quotes", "email,name\na@example.com,\"unfinished", ',', true, 1, "error reading CSV")
	f("error after valid row", "email,name\na@example.com,A\nb@example.com,\"unfinished", ',', true, 2, "error reading CSV", a)

	im := New(Options{}, nil, i)
	s := importerSession(t, im, SessionOpt{})
	if err := s.LoadCSV(filepath.Join(t.TempDir(), "missing.csv"), ','); err == nil || !s.failed.Load() {
		t.Fatal("missing file did not fail")
	}
	if _, ok := <-s.subQueue; ok {
		t.Fatal("failed load did not close queue")
	}
}

func TestImportZIP(t *testing.T) {
	// ExtractZIP creates temporary directories of its own, including on errors.
	t.Setenv("TMPDIR", t.TempDir())
	f := func(name string, files []string, valid bool) {
		t.Helper()

		t.Run(name, func(t *testing.T) {
			var b bytes.Buffer
			zw := zip.NewWriter(&b)
			for _, name := range files {
				w, err := zw.Create(name)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.HasSuffix(name, "/") {
					if _, err := io.WriteString(w, "email\na@example.com\n"); err != nil {
						t.Fatal(err)
					}
				}
			}
			if err := zw.Close(); err != nil {
				t.Fatal(err)
			}

			im := New(Options{}, nil, nil)
			s := importerSession(t, im, SessionOpt{})
			path := importerFile(t, "input.zip", b.Bytes())
			before, err := os.ReadDir(os.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			dir, got, err := s.ExtractZIP(path, 1)
			if (err == nil) != valid || s.failed.Load() == valid {
				t.Fatalf("valid=%v: files=%v err=%v failed=%v", valid, got, err, s.failed.Load())
			}
			if !valid {
				after, err := os.ReadDir(os.TempDir())
				if err != nil || len(before) != len(after) {
					t.Fatal("failed extraction left temporary files")
				}
				if _, ok := <-s.subQueue; ok {
					t.Fatal("failed extraction did not close queue")
				}
				return
			}
			// Successful extraction must produce readable, sanitized CSV files.
			defer os.RemoveAll(dir)
			if len(got) != 1 {
				t.Fatal("no CSV extracted")
			}
			for _, name := range got {
				if filepath.Base(name) != name {
					t.Fatalf("unsafe extracted name: %q", name)
				}
				data, err := os.ReadFile(filepath.Join(dir, name))
				if err != nil || string(data) != "email\na@example.com\n" {
					t.Fatalf("extracted content: %q, %v", data, err)
				}
			}
			s.Stop()
		})
	}

	f("single", []string{"one.csv"}, true)
	f("first only", []string{"one.csv", "two.csv"}, true)
	f("uppercase", []string{"ONE.CSV"}, true)
	f("nested and unrelated", []string{"dir/", "readme.txt", "dir/one.csv"}, true)
	f("traversal", []string{"../../escape.csv"}, true)
	f("empty", nil, false)
	f("no CSV", []string{"readme.txt"}, false)
	im := New(Options{}, nil, nil)
	s := importerSession(t, im, SessionOpt{})
	if _, _, err := s.ExtractZIP(importerFile(t, "bad.zip", []byte("not a zip")), 1); err == nil || !s.failed.Load() {
		t.Fatal("corrupt ZIP accepted")
	}
	if _, ok := <-s.subQueue; ok {
		t.Fatal("corrupt ZIP left queue open")
	}
}

func TestImportSubscriberScope(t *testing.T) {
	h := newImporterDB(t)

	f := func(name string, scope, members []int, allowed bool) {
		t.Helper()

		for _, info := range []bool{false, true} {
			for _, overwriteStatus := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/info=%v/status=%v", name, info, overwriteStatus), func(t *testing.T) {
					const status = "confirmed"

					h.seed(t, members, "enabled")
					before := h.subscriber(t, "existing@example.com")

					im := h.run(t, SessionOpt{Mode: ModeSubscribe, ListIDs: []int{1}, PermittedListIDs: scope, SubStatus: status, OverwriteUserInfo: info, OverwriteSubStatus: overwriteStatus}, "existing@example.com", "existing@example.com", "new@example.com", "existing@example.com")

					count := 1
					if allowed {
						count = 4
					}
					importStats(t, im, StatusFinished, count)

					got := h.subscriber(t, "existing@example.com")
					if !allowed {
						if !reflect.DeepEqual(got, before) {
							t.Fatalf("denied subscriber changed: %+v -> %+v", before, got)
						}
						if strings.Count(string(im.GetLogs()), "outside permitted lists") != 3 {
							t.Fatalf("denial logs: %s", im.GetLogs())
						}
					} else {
						wantName, wantAttribs := "Original", models.JSON{"original": true}
						if info {
							wantName, wantAttribs = "Imported", models.JSON{"imported": true}
						}
						if got.Name != wantName || !reflect.DeepEqual(got.Attribs, wantAttribs) || got.UUID != before.UUID || got.Status != "enabled" {
							t.Fatalf("incorrect update: %+v", got)
						}

						wantStatus := status
						if _, ok := before.Lists[1]; ok && !overwriteStatus {
							wantStatus = "unconfirmed"
						}
						if got.Lists[1] != wantStatus {
							t.Fatalf("destination status=%q, want %q", got.Lists[1], wantStatus)
						}
						for id, old := range before.Lists {
							if id != 1 && (got.Lists[id] != old || !got.ListDates[id].Equal(before.ListDates[id])) {
								t.Fatalf("source list %d changed", id)
							}
						}
					}

					fresh := h.subscriber(t, "new@example.com")
					if fresh.Status != "enabled" || fresh.Name != "Imported" || !reflect.DeepEqual(fresh.Attribs, models.JSON{"imported": true}) || !reflect.DeepEqual(fresh.Lists, map[int]string{1: status}) || fresh.UUID == before.UUID {
						t.Fatalf("new subscriber: %+v", fresh)
					}
				})
			}
		}
	}

	f("inaccessible", []int{1, 3}, []int{2}, false)
	f("destination", []int{1, 3}, []int{1}, true)
	f("other manageable", []int{1, 3}, []int{3}, true)
	f("mixed", []int{1, 3}, []int{2, 3}, true)
	f("orphan", []int{1, 3}, nil, false)
	f("no manageable lists", []int{-1}, []int{2}, false)
	f("empty scope", []int{}, []int{1}, false)
	f("full inaccessible", nil, []int{2}, true)
	f("full orphan", nil, nil, true)
}

func TestImportOverwriteAndDestinations(t *testing.T) {
	h := newImporterDB(t)

	f := func(name string, opt SessionOpt, status string, wantLists map[int]string) {
		t.Helper()

		t.Run(name, func(t *testing.T) {
			h.seed(t, []int{1, 2}, status)
			opt.Mode = ModeSubscribe
			im := h.run(t, opt, "existing@example.com")
			importStats(t, im, StatusFinished, 1)

			got := h.subscriber(t, "existing@example.com")
			if got.Status != status || !reflect.DeepEqual(got.Lists, wantLists) {
				t.Fatalf("subscriber: %+v", got)
			}
			if opt.Overwrite && got.Name != "Imported" {
				t.Fatal("legacy overwrite did not update profile")
			}
		})
	}

	f("legacy overwrite", SessionOpt{ListIDs: []int{1}, SubStatus: "confirmed", Overwrite: true}, "enabled", map[int]string{1: "confirmed", 2: "unconfirmed"})
	f("multiple destinations", SessionOpt{ListIDs: []int{1, 3}, SubStatus: "confirmed"}, "enabled", map[int]string{1: "unconfirmed", 2: "unconfirmed", 3: "confirmed"})
	f("blocklisted stays blocked", SessionOpt{ListIDs: []int{1, 3}, SubStatus: "confirmed", Overwrite: true}, "blocklisted", map[int]string{1: "unsubscribed", 2: "unconfirmed", 3: "unsubscribed"})
	f("blocklisted scoped", SessionOpt{ListIDs: []int{1, 3}, PermittedListIDs: []int{1}, SubStatus: "confirmed", Overwrite: true}, "blocklisted", map[int]string{1: "unsubscribed", 2: "unconfirmed", 3: "unsubscribed"})
}

func TestImportBlocklistWithoutMemberships(t *testing.T) {
	h := newImporterDB(t)

	f := func(name string, emails ...string) {
		t.Helper()

		t.Run(name, func(t *testing.T) {
			h.seed(t, []int{1, 2}, "enabled")
			h.exec(t, `INSERT INTO subscribers(uuid,email,name,attribs,status) VALUES('00000000-0000-4000-8000-000000000002','orphan@example.com','Original','{"original":true}','enabled')`)
			im := h.run(t, SessionOpt{Mode: ModeBlocklist}, emails...)
			importStats(t, im, StatusFinished, len(emails))

			if strings.Contains(string(im.GetLogs()), "outside permitted lists") {
				t.Fatal("false permission denial")
			}
			for _, email := range emails {
				got := h.subscriber(t, email)
				if got.Status != "blocklisted" {
					t.Fatalf("not blocklisted: %+v", got)
				}
				for _, status := range got.Lists {
					if status != "unsubscribed" {
						t.Fatalf("subscription not unsubscribed: %+v", got)
					}
				}
				if email != "new@example.com" && (got.Name != "Original" || !reflect.DeepEqual(got.Attribs, models.JSON{"original": true})) {
					t.Fatalf("blocklisting overwrote profile: %+v", got)
				}
			}
		})
	}

	f("new", "new@example.com")
	f("orphan", "orphan@example.com")
	f("before member", "new@example.com", "orphan@example.com", "existing@example.com")
	f("after member", "existing@example.com", "new@example.com", "orphan@example.com")
	f("repeated", "new@example.com", "new@example.com", "orphan@example.com", "orphan@example.com", "existing@example.com", "existing@example.com")
}

func TestImportBatches(t *testing.T) {
	h := newImporterDB(t)

	f := func(name, mode string, n int, fail bool) {
		t.Helper()

		t.Run(name, func(t *testing.T) {
			h.seed(t, []int{2}, "enabled")
			before := h.subscriber(t, "existing@example.com")

			opt := h.opt
			var notifications []importStatusTpl
			opt.PostCB = func(_ string, data any) error {
				notifications = append(notifications, data.(importStatusTpl))
				return nil
			}

			im := New(opt, h.db, h.i)
			s := importerSession(t, im, SessionOpt{Mode: mode, ListIDs: []int{1}, SubStatus: "confirmed", PermittedListIDs: []int{1}})
			s.failed.Store(fail)

			done := make(chan struct{})
			go func() {
				defer close(done)
				s.Start()
			}()

			for j := 0; j < n; j++ {
				if mode == ModeSubscribe && (j == 0 || j == commitBatchSize) {
					s.subQueue <- SubReq{Subscriber: models.Subscriber{Email: "existing@example.com", Attribs: models.JSON{}}}
				}
				s.subQueue <- SubReq{Subscriber: models.Subscriber{Email: fmt.Sprintf("batch%d@example.com", j), Name: "Batch", Attribs: models.JSON{}}}
			}
			if mode == ModeSubscribe {
				s.subQueue <- SubReq{Subscriber: models.Subscriber{Email: "existing@example.com", Attribs: models.JSON{}}}
			}
			close(s.subQueue)
			importerWait(t, done)

			count, status := n, StatusFinished
			if fail {
				count = n / commitBatchSize * commitBatchSize
				status = StatusFailed
			}
			importStats(t, im, status, count)

			var stored int
			if err := h.db.QueryRow(`SELECT count(*) FROM subscribers WHERE email LIKE 'batch%@example.com'`).Scan(&stored); err != nil {
				t.Fatal(err)
			}
			if stored != count {
				t.Fatalf("stored=%d, counted=%d", stored, count)
			}
			if !reflect.DeepEqual(h.subscriber(t, "existing@example.com"), before) {
				t.Fatal("denied subscriber changed")
			}
			if len(notifications) != 1 || notifications[0].Imported != count || notifications[0].Status != status {
				t.Fatalf("notifications: %+v", notifications)
			}
		})
	}

	f("all denied", ModeSubscribe, 0, false)
	f("exact batch", ModeSubscribe, commitBatchSize, false)
	f("boundary with denials", ModeSubscribe, commitBatchSize+1, false)
	f("load error retains committed batch", ModeSubscribe, commitBatchSize+1, true)
	f("blocklist boundary", ModeBlocklist, commitBatchSize+1, false)
}

func TestImportCSVTransactions(t *testing.T) {
	h := newImporterDB(t)

	f := func(name, body string, wantErr bool, count int) {
		t.Helper()

		t.Run(name, func(t *testing.T) {
			h.seed(t, []int{2}, "enabled")
			before := h.subscriber(t, "existing@example.com")

			im := New(h.opt, h.db, h.i)
			s := importerSession(t, im, SessionOpt{Mode: ModeSubscribe, ListIDs: []int{1}, PermittedListIDs: []int{1}, SubStatus: "confirmed", Overwrite: true})

			// Small files fit in the queue. Loading first also makes rollback tests
			// deterministic: a parse failure is known before Start consumes the queue.
			err := s.LoadCSV(importerFile(t, "input.csv", []byte(body)), ',')
			if (err != nil) != wantErr {
				t.Fatalf("load: %v", err)
			}

			s.Start()

			status := StatusFinished
			if wantErr {
				status = StatusFailed
			}
			importStats(t, im, status, count)

			var stored int
			if err := h.db.QueryRow(`SELECT count(*) FROM subscribers WHERE email!='existing@example.com'`).Scan(&stored); err != nil {
				t.Fatal(err)
			}
			if stored != count {
				t.Fatalf("stored=%d, want %d", stored, count)
			}
			if !reflect.DeepEqual(h.subscriber(t, "existing@example.com"), before) {
				t.Fatal("CSV changed inaccessible subscriber")
			}

			// A failed load must not poison subsequent sessions.
			next := importerSession(t, im, SessionOpt{Mode: ModeSubscribe, ListIDs: []int{1}, SubStatus: "unconfirmed"})
			if err := next.LoadCSV(importerFile(t, "next.csv", []byte("email\nrecovery@example.com\n")), ','); err != nil {
				t.Fatal(err)
			}

			next.Start()
			importStats(t, im, StatusFinished, 1)
		})
	}

	f("mixed and normalized", "email,name\n EXISTING@EXAMPLE.COM ,Changed\nnew@example.com,New\n", false, 1)
	f("bad row then valid", "email,name\ninvalid,Invalid\nnew@example.com,New\n", false, 1)
	f("rollback on malformed trailing row", "email,name\nnew@example.com,New\nbad@example.com,\"unfinished", true, 0)
	f("empty", "", true, 0)
	f("header only", "email\n", false, 0)
}

func TestImportListDates(t *testing.T) {
	h := newImporterDB(t)

	h.seed(t, []int{2}, "enabled")
	im := h.run(t, SessionOpt{Mode: ModeSubscribe, ListIDs: []int{1, 3}, SubStatus: "confirmed"}, "new@example.com")
	importStats(t, im, StatusFinished, 1)

	var ids pq.Int64Array
	if err := h.db.QueryRow(`SELECT array_agg(id ORDER BY id) FROM lists WHERE updated_at>'2000-01-01'`).Scan(&ids); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ids, pq.Int64Array{1, 3}) {
		t.Fatalf("updated lists=%v", ids)
	}
}

func TestImportSubscriptionStatus(t *testing.T) {
	h := newImporterDB(t)

	f := func(old, requested string, overwrite bool) {
		t.Helper()

		t.Run(fmt.Sprintf("%s/%s/overwrite=%v", old, requested, overwrite), func(t *testing.T) {
			h.seed(t, []int{1}, "enabled")
			h.exec(t, `UPDATE subscriber_lists SET status=$1`, old)
			im := h.run(t, SessionOpt{Mode: ModeSubscribe, ListIDs: []int{1}, PermittedListIDs: []int{1}, SubStatus: requested, OverwriteSubStatus: overwrite}, "existing@example.com")
			importStats(t, im, StatusFinished, 1)

			want := old
			if overwrite {
				want = requested
			}
			got := h.subscriber(t, "existing@example.com")
			if got.Lists[1] != want || got.Name != "Original" || !reflect.DeepEqual(got.Attribs, models.JSON{"original": true}) {
				t.Fatalf("subscription-only overwrite changed the wrong fields: %+v", got)
			}
		})
	}

	for _, old := range []string{"unconfirmed", "confirmed", "unsubscribed"} {
		for _, requested := range []string{"unconfirmed", "confirmed", "unsubscribed"} {
			f(old, requested, false)
			f(old, requested, true)
		}
	}
}

func TestImportZIPTransactions(t *testing.T) {
	h := newImporterDB(t)

	t.Setenv("TMPDIR", t.TempDir())
	f := func(name string, files []string, corrupt bool, count int) {
		t.Helper()

		t.Run(name, func(t *testing.T) {
			h.seed(t, []int{2}, "enabled")
			before := h.subscriber(t, "existing@example.com")

			var b bytes.Buffer
			zw := zip.NewWriter(&b)
			for j, name := range files {
				w, err := zw.Create(name)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := fmt.Fprintf(w, "email\nexisting@example.com\nzip%d@example.com\n", j); err != nil {
					t.Fatal(err)
				}
			}
			if err := zw.Close(); err != nil {
				t.Fatal(err)
			}

			data := b.Bytes()
			if corrupt {
				data = []byte("not a ZIP")
			}

			im := New(h.opt, h.db, h.i)
			opt := SessionOpt{Mode: ModeSubscribe, ListIDs: []int{1}, PermittedListIDs: []int{1}, SubStatus: "confirmed"}
			s := importerSession(t, im, opt)
			dir, extracted, err := s.ExtractZIP(importerFile(t, "input.zip", data), 1)
			status := StatusFinished
			if count == 0 {
				status = StatusFailed
				if err == nil {
					t.Fatal("invalid ZIP accepted")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if err := s.LoadCSV(filepath.Join(dir, extracted[0]), ','); err != nil {
					t.Fatal(err)
				}
			}
			s.Start()
			importStats(t, im, status, count)

			if !reflect.DeepEqual(h.subscriber(t, "existing@example.com"), before) {
				t.Fatal("ZIP changed inaccessible subscriber")
			}

			var stored int
			if err := h.db.QueryRow(`SELECT count(*) FROM subscribers WHERE email LIKE 'zip%@example.com'`).Scan(&stored); err != nil {
				t.Fatal(err)
			}
			if stored != count {
				t.Fatalf("stored=%d, want %d", stored, count)
			}

			next := importerSession(t, im, opt)
			if err := next.LoadCSV(importerFile(t, "next.csv", []byte("email\nrecovery@example.com\n")), ','); err != nil {
				t.Fatal(err)
			}

			next.Start()
			importStats(t, im, StatusFinished, 1)
		})
	}

	f("CSV import", []string{"one.csv"}, false, 1)
	f("corrupt", nil, true, 0)
}

func TestImportStopAndRestart(t *testing.T) {
	h := newImporterDB(t)

	h.seed(t, []int{2}, "enabled")
	im := New(h.opt, h.db, h.i)
	opt := SessionOpt{Mode: ModeSubscribe, ListIDs: []int{1}, SubStatus: "confirmed"}
	s := importerSession(t, im, opt)
	im.Stop()
	p := importerFile(t, "input.csv", []byte("email\nnew@example.com\n"))
	if err := s.LoadCSV(p, ','); err != nil {
		t.Fatal(err)
	}
	s.Start()
	importStats(t, im, StatusFinished, 0)

	var count int
	if err := h.db.QueryRow(`SELECT count(*) FROM subscribers`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("stopped import wrote subscribers: %d", count)
	}
	im.Stop()
	s = importerSession(t, im, opt)
	if err := s.LoadCSV(p, ','); err != nil {
		t.Fatal(err)
	}
	s.Start()
	importStats(t, im, StatusFinished, 1)

	if got := h.subscriber(t, "new@example.com"); got.Lists[1] != "confirmed" {
		t.Fatalf("restart subscriber: %+v", got)
	}
}

func importerI18n(t *testing.T) *i18n.I18n {
	t.Helper()

	b, err := os.ReadFile("../../i18n/en.json")
	if err != nil {
		t.Fatal(err)
	}

	i, err := i18n.New(b)
	if err != nil {
		t.Fatal(err)
	}

	return i
}

func importerFile(t *testing.T, name string, data []byte) string {
	t.Helper()

	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, data, 0600); err != nil {
		t.Fatal(err)
	}

	return p
}

func importerSession(t *testing.T, im *Importer, opt SessionOpt) *Session {
	t.Helper()

	s, err := im.NewSession(opt)
	if err != nil {
		t.Fatal(err)
	}

	return s
}

func importerWait(t *testing.T, done <-chan struct{}) {
	t.Helper()

	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("import did not finish within 15s")
	}
}

func newImporterDB(t *testing.T) *importerDB {
	t.Helper()

	dsn := os.Getenv("LISTMONK_TEST_DB")
	if dsn == "" {
		t.Skip("set LISTMONK_TEST_DB to run PostgreSQL integration tests")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	h := &importerDB{db: db, i: importerI18n(t)}
	h.exec(t, `CREATE TYPE pg_temp.subscription_status AS ENUM ('unconfirmed','confirmed','unsubscribed');
 CREATE TEMP TABLE lists (id INT PRIMARY KEY, updated_at TIMESTAMPTZ DEFAULT NOW());
 CREATE TEMP TABLE subscribers (id SERIAL PRIMARY KEY, uuid UUID NOT NULL, email TEXT UNIQUE NOT NULL,
 name TEXT, attribs JSONB, status TEXT NOT NULL, updated_at TIMESTAMPTZ DEFAULT NOW());
 CREATE TEMP TABLE subscriber_lists (subscriber_id INT REFERENCES subscribers(id), list_id INT REFERENCES lists(id),
 status pg_temp.subscription_status, updated_at TIMESTAMPTZ DEFAULT NOW(), UNIQUE(subscriber_id,list_id));
 INSERT INTO lists(id) VALUES (1),(2),(3);`)

	prepare := func(file, name string) *sql.Stmt {
		t.Helper()

		b, err := os.ReadFile("../../queries/" + file)
		if err != nil {
			t.Fatal(err)
		}

		_, tail, ok := strings.Cut(string(b), "-- name: "+name+"\n")
		if !ok {
			t.Fatalf("query %s missing", name)
		}

		query, _, _ := strings.Cut(tail, "-- name:")
		stmt, err := db.Prepare(query)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { stmt.Close() })

		return stmt
	}

	h.opt = Options{
		UpsertStmt:         prepare("subscribers.sql", "upsert-subscriber"),
		BlocklistStmt:      prepare("subscribers.sql", "upsert-blocklist-subscriber"),
		UpdateListDateStmt: prepare("lists.sql", "update-lists-date"),
		PostCB:             func(string, any) error { return nil },
	}

	return h
}

func (h *importerDB) exec(t *testing.T, q string, args ...any) {
	t.Helper()

	if _, err := h.db.Exec(q, args...); err != nil {
		t.Fatal(err)
	}
}

func (h *importerDB) seed(t *testing.T, members []int, status string) {
	t.Helper()

	h.exec(t, `TRUNCATE subscribers, subscriber_lists RESTART IDENTITY;
 UPDATE lists SET updated_at='2000-01-01';`)
	h.exec(t, `INSERT INTO subscribers(uuid,email,name,attribs,status,updated_at)
 VALUES('00000000-0000-4000-8000-000000000001','existing@example.com','Original','{"original":true}',$1,'2000-01-01')`, status)
	for _, id := range members {
		h.exec(t, `INSERT INTO subscriber_lists VALUES(1,$1,'unconfirmed','2000-01-01')`, id)
	}
}

func (h *importerDB) subscriber(t *testing.T, email string) importSubscriber {
	t.Helper()

	var s importSubscriber
	var attribs []byte
	err := h.db.QueryRow(`SELECT uuid,name,attribs,status,updated_at FROM subscribers WHERE email=$1`, email).Scan(&s.UUID, &s.Name, &attribs, &s.Status, &s.Updated)
	if err != nil {
		t.Fatalf("subscriber %q: %v", email, err)
	}
	if err := json.Unmarshal(attribs, &s.Attribs); err != nil {
		t.Fatal(err)
	}

	s.Lists = map[int]string{}
	s.ListDates = map[int]time.Time{}
	rows, err := h.db.Query(`SELECT list_id,sl.status,sl.updated_at FROM subscriber_lists sl JOIN subscribers s ON s.id=sl.subscriber_id WHERE s.email=$1`, email)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var status string
		var updated time.Time
		if err := rows.Scan(&id, &status, &updated); err != nil {
			t.Fatal(err)
		}
		s.Lists[id] = status
		s.ListDates[id] = updated
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	return s
}

func (h *importerDB) run(t *testing.T, opt SessionOpt, emails ...string) *Importer {
	t.Helper()

	im := New(h.opt, h.db, h.i)
	s := importerSession(t, im, opt)
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.Start()
	}()

	for _, email := range emails {
		s.subQueue <- SubReq{Subscriber: models.Subscriber{Email: email, Name: "Imported", Attribs: models.JSON{"imported": true}}}
	}
	close(s.subQueue)
	importerWait(t, done)

	return im
}

func importStats(t *testing.T, im *Importer, status string, count int) {
	t.Helper()

	if got := im.GetStats(); got.Status != status || got.Imported != count {
		t.Fatalf("stats=%+v, want %s/%d; logs: %s", got, status, count, im.GetLogs())
	}
}

func TestRecordLimit(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		fail        bool
	}{
		{"boundary", strings.Repeat("a", maxRecordBytes-1) + "\n", false},
		{"boundary EOF", strings.Repeat("a", maxRecordBytes), false},
		{"oversized", strings.Repeat("a", maxRecordBytes) + "\n", true},
		{"quoted multiline", "\"" + strings.Repeat("a\n", maxRecordBytes/2) + "\"\n", true},
		{"unterminated", "\"" + strings.Repeat("a", maxRecordBytes+1), true},
		{"many fields", strings.Repeat(",", maxRecordBytes) + "\n", true},
		{"escaped quotes", "\"" + strings.Repeat("a\"\"\n", 100) + "\"\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := &rowReader{Reader: strings.NewReader(tc.input)}
			rd := csv.NewReader(input)
			_, err := input.ReadRow(rd)
			if tc.fail && !errors.Is(err, errRecordTooLarge) {
				t.Fatalf("expected record limit, got %v", err)
			}
			if !tc.fail && err != nil {
				t.Fatal(err)
			}
			if input.read > maxRecordBytes+1 {
				t.Fatalf("read %d bytes", input.read)
			}
		})
	}
}

func TestRecordLimitReadAhead(t *testing.T) {
	input := &rowReader{Reader: strings.NewReader("header\n" + strings.Repeat("a", maxRecordBytes-1) + "\nlast\n")}
	rd := csv.NewReader(input)
	for i := 0; i < 3; i++ {
		if _, err := input.ReadRow(rd); err != nil {
			t.Fatalf("record %d: %v", i, err)
		}
	}
	if _, err := input.ReadRow(rd); err != io.EOF {
		t.Fatalf("EOF: %v", err)
	}
}

func TestImportZIPExpansionLimit(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	var b bytes.Buffer
	zw := zip.NewWriter(&b)
	if _, err := zw.CreateRaw(&zip.FileHeader{Name: "large.csv", UncompressedSize64: maxExtractedBytes + 1}); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	path := importerFile(t, "large.zip", b.Bytes())
	before, err := os.ReadDir(os.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s := importerSession(t, New(Options{}, nil, nil), SessionOpt{})
	if _, _, err := s.ExtractZIP(path, 1); !errors.Is(err, errZIPTooLarge) {
		t.Fatalf("expected expansion limit: %v", err)
	}
	after, err := os.ReadDir(os.TempDir())
	if err != nil || len(before) != len(after) {
		t.Fatal("failed extraction left temporary files")
	}
	if !s.failed.Load() {
		t.Fatal("oversized archive did not fail the session")
	}
}
