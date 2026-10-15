// Package testdb provides a disposable, fully installed listmonk Postgres database
// for tests that need to run real queries.
//
// Tests using it are skipped unless the LISTMONK_TEST_DB environment variable is set
// to a Postgres URL (eg: postgres://user:pass@localhost:5432/postgres?sslmode=disable)
// of a server where the user can create databases. Every call creates a new, uniquely
// named database that is dropped when the test finishes. Existing databases are never touched.
package testdb

import (
	"fmt"
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/goyesql/v2"
	goyesqlx "github.com/knadh/goyesql/v2/sqlx"
	"github.com/knadh/listmonk/models"
	_ "github.com/lib/pq"
)

// EnvDSN is the environment variable with the Postgres URL used for creating test databases.
const EnvDSN = "LISTMONK_TEST_DB"

// RootDir returns the absolute path of the repository root.
func RootDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..")
}

// New creates a new database with the listmonk schema and returns a connection to it along
// with the parsed queries. The test is skipped if LISTMONK_TEST_DB is not set.
func New(t *testing.T) (*sqlx.DB, goyesql.Queries) {
	t.Helper()

	dsn := os.Getenv(EnvDSN)
	if dsn == "" {
		t.Skipf("%s is not set; skipping database test", EnvDSN)
	}

	u, err := url.Parse(dsn)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
		t.Fatalf("%s should be a postgres:// URL", EnvDSN)
	}

	admin, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Fatalf("error connecting to test DB server: %v", err)
	}

	name := fmt.Sprintf("listmonk_test_%d_%d", os.Getpid(), time.Now().UnixNano())
	if _, err := admin.Exec("CREATE DATABASE " + name); err != nil {
		admin.Close()
		t.Fatalf("error creating test database: %v", err)
	}

	u.Path = "/" + name
	conn, err := sqlx.Connect("postgres", u.String())
	if err != nil {
		t.Fatalf("error connecting to test database: %v", err)
	}

	// Same as the app (initDB), ignore columns that aren't in the models.
	db := conn.Unsafe()

	t.Cleanup(func() {
		db.Close()
		if _, err := admin.Exec("DROP DATABASE IF EXISTS " + name + " WITH (FORCE)"); err != nil {
			t.Logf("error dropping test database %s: %v", name, err)
		}
		admin.Close()
	})

	// Install the schema.
	schema, err := os.ReadFile(filepath.Join(RootDir(), "schema.sql"))
	if err != nil {
		t.Fatalf("error reading schema.sql: %v", err)
	}
	if _, err := db.Exec(string(schema)); err != nil {
		t.Fatalf("error installing schema: %v", err)
	}

	return db, ReadQueries(t)
}

// ReadQueries parses all the SQL queries in the queries directory.
func ReadQueries(t *testing.T) goyesql.Queries {
	t.Helper()

	files, err := filepath.Glob(filepath.Join(RootDir(), "queries", "*.sql"))
	if err != nil || len(files) == 0 {
		t.Fatalf("error finding query files: %v", err)
	}

	out := goyesql.Queries{}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("error reading %s: %v", f, err)
		}

		mp, err := goyesql.ParseBytes(b)
		if err != nil {
			t.Fatalf("error parsing %s: %v", f, err)
		}
		maps.Copy(out, mp)
	}

	return out
}

// PrepareQueries prepares all the queries into models.Queries. This mirrors the app's
// prepareQueries() with individual tracking turned off.
func PrepareQueries(t *testing.T, qMap goyesql.Queries, db *sqlx.DB) *models.Queries {
	t.Helper()

	sel := func(name string) string { return qMap[name].Query }
	add := func(name, query string) {
		qMap[name] = &goyesql.Query{Query: query, Tags: map[string]string{"name": name}}
	}

	add("get-campaign-view-counts", fmt.Sprintf(sel("get-campaign-analytics-counts"), "campaign_views"))
	add("get-campaign-click-counts", fmt.Sprintf(sel("get-campaign-analytics-counts"), "link_clicks"))
	qMap["get-campaign-link-counts"].Query = fmt.Sprintf(sel("get-campaign-link-counts"), "*")
	for typ, table := range map[string]string{"view": "campaign_views", "click": "link_clicks", "bounce": "bounces"} {
		add("get-list-"+typ+"-counts", fmt.Sprintf(sel("get-list-analytics-counts"), table))
	}
	qMap["get-list-link-counts"].Query = fmt.Sprintf(sel("get-list-link-counts"), "*")

	var q models.Queries
	if err := goyesqlx.ScanToStruct(&q, qMap, db); err != nil {
		t.Fatalf("error preparing queries: %v", err)
	}

	return &q
}

// Exec runs the given SQL statements on the database, failing the test on errors.
func Exec(t *testing.T, db *sqlx.DB, stmts ...string) {
	t.Helper()
	if _, err := db.Exec(strings.Join(stmts, ";\n")); err != nil {
		t.Fatalf("error executing SQL: %v", err)
	}
}
