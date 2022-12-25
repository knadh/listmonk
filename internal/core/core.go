// package core is the collection of re-usable functions that primarily provides data (DB / CRUD) operations
// to the app. For instance, creating and mutating objects like lists, subscribers etc.
// All such methods return an echo.HTTPError{} (which implements error.error) that can be directly returned
// as a response to HTTP handlers without further processing.
package core

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/listmonk/internal/i18n"
	"github.com/knadh/listmonk/models"
	"github.com/lib/pq"
)

const (
	SortAsc  = "asc"
	SortDesc = "desc"
)

// Core represents the listmonk core with all shared, global functions.
type Core struct {
	h *Hooks

	constants Constants
	i18n      *i18n.I18n
	db        *sqlx.DB
	q         *models.Queries
	log       *log.Logger
}

// Constants represents constant config.
type Constants struct {
	SendOptinConfirmation bool
	MaxBounceCount        int
	BounceAction          string
}

// Hooks contains external function hooks that are required by the core package.
type Hooks struct {
	SendOptinConfirmation func(models.Subscriber, []int) (int, error)
}

// Opt contains the controllers required to start the core.
type Opt struct {
	Constants Constants
	I18n      *i18n.I18n
	DB        *sqlx.DB
	Queries   *models.Queries
	Log       *log.Logger
}

var (
	regexFullTextQuery = regexp.MustCompile(`\s+`)
	regexpSpaces       = regexp.MustCompile(`[\s]+`)
	querySortFields    = []string{"name", "status", "created_at", "updated_at"}
)

// New returns a new instance of the core.
func New(o *Opt, h *Hooks) *Core {
	return &Core{
		h:         h,
		constants: o.Constants,
		i18n:      o.I18n,
		db:        o.DB,
		q:         o.Queries,
		log:       o.Log,
	}
}

// Given an error, pqErrMsg will try to return pq error details
// if it's a pq error.
func pqErrMsg(err error) string {
	if err, ok := err.(*pq.Error); ok {
		if err.Detail != "" {
			return fmt.Sprintf("%s. %s", err, err.Detail)
		}
	}
	return err.Error()
}

// makeSearchQuery cleans an optional search string and prepares the
// query SQL statement (string interpolated) and returns the
// search query string along with the SQL expression.
func makeSearchQuery(searchStr, orderBy, order, query string) (string, string) {
	if searchStr != "" {
		searchStr = `%` + string(regexFullTextQuery.ReplaceAll([]byte(searchStr), []byte("&"))) + `%`
	}

	// Sort params.
	if !strSliceContains(orderBy, querySortFields) {
		orderBy = "created_at"
	}
	if order != SortAsc && order != SortDesc {
		order = SortDesc
	}

	query = strings.ReplaceAll(query, "%order%", orderBy+" "+order)

	return searchStr, query
}

// strSliceContains checks if a string is present in the string slice.
func strSliceContains(str string, sl []string) bool {
	for _, s := range sl {
		if s == str {
			return true
		}
	}

	return false
}

// normalizeTags takes a list of string tags and normalizes them by
// lower casing and removing all special characters except for dashes.
func normalizeTags(tags []string) []string {
	var (
		out  []string
		dash = []byte("-")
	)

	for _, t := range tags {
		rep := regexpSpaces.ReplaceAll(bytes.TrimSpace([]byte(t)), dash)

		if len(rep) > 0 {
			out = append(out, string(rep))
		}
	}
	return out
}

// sanitizeSQLExp does basic sanitisation on arbitrary
// SQL query expressions coming from the frontend.
func sanitizeSQLExp(q string) string {
	if len(q) == 0 {
		return ""
	}
	q = strings.TrimSpace(q)

	// Remove semicolon suffix.
	if q[len(q)-1] == ';' {
		q = q[:len(q)-1]
	}
	return q
}

// strHasLen checks if the given string has a length within min-max.
func strHasLen(str string, min, max int) bool {
	return len(str) >= min && len(str) <= max
}
