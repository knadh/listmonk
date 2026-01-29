package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/listmonk/internal/migrations"
	"github.com/knadh/stuffbin"
	"github.com/lib/pq"
	"golang.org/x/mod/semver"
)

// migFunc represents a migration function for a particular version.
// fn (generally) executes database migrations and additionally
// takes the filesystem and config objects in case there are additional bits
// of logic to be performed before executing upgrades. fn is idempotent.
type migFunc struct {
	version string
	fn      func(*sqlx.DB, stuffbin.FileSystem, *koanf.Koanf, *log.Logger) error
}

// migList is the list of available migList ordered by the semver.
// Each migration is a Go file in internal/migrations named after the semver.
// The functions are named as: v0.7.0 => migrations.V0_7_0() and are idempotent.
var migList = []migFunc{
	{"v0.4.0", migrations.V0_4_0},
	{"v0.7.0", migrations.V0_7_0},
	{"v0.8.0", migrations.V0_8_0},
	{"v0.9.0", migrations.V0_9_0},
	{"v1.0.0", migrations.V1_0_0},
	{"v2.0.0", migrations.V2_0_0},
	{"v2.1.0", migrations.V2_1_0},
	{"v2.2.0", migrations.V2_2_0},
	{"v2.3.0", migrations.V2_3_0},
	{"v2.4.0", migrations.V2_4_0},
	{"v2.5.0", migrations.V2_5_0},
	{"v3.0.0", migrations.V3_0_0},
	{"v4.0.0", migrations.V4_0_0},
	{"v4.1.0", migrations.V4_1_0},
	{"v5.0.0", migrations.V5_0_0},
	{"v5.1.0", migrations.V5_1_0},
	{"v6.0.0", migrations.V6_0_0},
	{"v6.1.0", migrations.V6_1_0},
}

// upgrade upgrades the database to the current version by running SQL migration files
// for all version from the last known version to the current one.
// If record is false, migration versions are not recorded in the DB (used for nightly builds).
func upgrade(db *sqlx.DB, fs stuffbin.FileSystem, prompt bool, record bool) {
	if prompt {
		var ok string
		fmt.Printf("** IMPORTANT: Take a backup of the database before upgrading.\n")
		fmt.Print("continue (y/n)?  ")
		if _, err := fmt.Scanf("%s", &ok); err != nil {
			lo.Fatalf("error reading value from terminal: %v", err)
		}
		if strings.ToLower(ok) != "y" {
			fmt.Println("upgrade cancelled")
			return
		}
	}

	_, toRun, err := getPendingMigrations(db)
	if err != nil {
		lo.Fatalf("error checking migrations: %v", err)
	}

	// No migrations to run.
	if len(toRun) == 0 {
		lo.Printf("no upgrades to run. Database is up to date.")
		return
	}

	// Execute migrations in succession.
	for _, m := range toRun {
		lo.Printf("running migration %s", m.version)
		if err := m.fn(db, fs, ko, lo); err != nil {
			lo.Fatalf("error running migration %s: %v", m.version, err)
		}

		// Record the migration version in the settings table. There was no
		// settings table until v0.7.0, so ignore the no-table errors.
		// For nightly builds, skip recording so migrations re-run on each boot.
		if record {
			if err := recordMigrationVersion(m.version, db); err != nil {
				if isTableNotExistErr(err) {
					continue
				}
				lo.Fatalf("error recording migration version %s: %v", m.version, err)
			}
		}
	}

	lo.Printf("upgrade complete")
}

// checkUpgrade checks if the current database schema matches the expected
// binary version.
func checkUpgrade(db *sqlx.DB) {
	lastVer, toRun, err := getPendingMigrations(db)
	if err != nil {
		lo.Fatalf("error checking migrations: %v", err)
	}

	// No migrations to run.
	if len(toRun) == 0 {
		return
	}

	var vers []string
	for _, m := range toRun {
		vers = append(vers, m.version)
	}

	lo.Fatalf(`there are %d pending database upgrade(s): %v. The last upgrade was %s. Backup the database and run listmonk --upgrade`,
		len(toRun), vers, lastVer)
}

// getPendingMigrations gets the pending migrations by comparing the last
// recorded migration in the DB against all migrations listed in `migrations`.
func getPendingMigrations(db *sqlx.DB) (string, []migFunc, error) {
	lastVer, err := getLastMigrationVersion(db)
	if err != nil {
		return "", nil, err
	}

	// Iterate through the migration versions and get everything above the last
	// upgraded semver.
	var toRun []migFunc
	for i, m := range migList {
		if semver.Compare(m.version, lastVer) > 0 {
			toRun = migList[i:]
			break
		}
	}

	return lastVer, toRun, nil
}

// getLastMigrationVersion returns the last migration semver recorded in the DB.
// If there isn't any, `v0.0.0` is returned.
func getLastMigrationVersion(db *sqlx.DB) (string, error) {
	var v string
	if err := db.Get(&v, `
		SELECT COALESCE(
			(SELECT value->>-1 FROM settings WHERE key='migrations'),
		'v0.0.0')`); err != nil {
		if isTableNotExistErr(err) {
			return "v0.0.0", nil
		}
		return v, err
	}
	return v, nil
}

// isTableNotExistErr checks if the given error represents a Postgres/pq
// "table does not exist" error.
func isTableNotExistErr(err error) bool {
	if p, ok := err.(*pq.Error); ok {
		// `settings` table does not exist. It was introduced in v0.7.0.
		if p.Code == "42P01" {
			return true
		}
	}
	return false
}
