// package core is the collection of re-usable functions that primarily provides data (DB / CRUD) operations
// to the app. For instance, creating and mutating objects like lists, subscribers etc.
// All such methods return an echo.HTTPError{} (which implements error.error) that can be directly returned
// as a response to HTTP handlers without further processing.
package core

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/listmonk/internal/i18n"
	"github.com/knadh/listmonk/internal/throttle"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v5"
	"github.com/lib/pq"
)

const (
	matDashboardCharts = "mat_dashboard_charts"
	matDashboardCounts = "mat_dashboard_counts"
	matListCharts      = "mat_list_charts"
	matListSubStats    = "mat_list_subscriber_stats"
)

// Core represents the listmonk core with all shared, global functions.
type Core struct {
	h *Hooks

	consts        Constants
	optinThrottle *throttle.Throttle
	i18n          *i18n.I18n
	db            *sqlx.DB
	q             *models.Queries
	log           *log.Logger
}

// Constants represents constant config.
type Constants struct {
	SendOptinConfirmation bool
	BounceActions         map[string]struct {
		Count  int
		Action string
	}
	CacheSlowQueries bool
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
	ErrNotFound = echo.NewHTTPError(http.StatusNotFound, "not found")
)

var (
	campQuerySortFields     = []string{"name", "status", models.FieldCreatedAt, "updated_at", "started_at", "send_at"}
	subQuerySortFields      = []string{"email", "status", "name", models.FieldCreatedAt, "updated_at"}
	userQuerySortFields     = []string{"username", "email", models.FieldCreatedAt, "loggedin_at"}
	roleQuerySortFields     = []string{"name", models.FieldCreatedAt, "updated_at"}
	templateQuerySortFields = []string{"name", "type", models.FieldCreatedAt, "updated_at"}
	listQuerySortFields     = []string{"name", "status", models.FieldCreatedAt, "updated_at", "subscriber_count"}
)

// New returns a new instance of the core.
func New(o *Opt, h *Hooks) *Core {
	return &Core{
		h:             h,
		consts:        o.Constants,
		optinThrottle: throttle.New(optinThrottleTTL, optinThrottleCap),
		i18n:          o.I18n,
		db:            o.DB,
		q:             o.Queries,
		log:           o.Log,
	}
}

// RefreshMatViews refreshes all materialized views.
func (c *Core) RefreshMatViews(concurrent bool) error {
	for _, v := range []string{matDashboardCharts, matDashboardCounts, matListSubStats, matListCharts} {
		_ = c.RefreshMatView(v, true)
	}
	return nil
}

// RefreshMatView refreshes a Postgres materialized view.
func (c *Core) RefreshMatView(name string, concurrent bool) error {
	q := "REFRESH MATERIALIZED VIEW %s %s"
	if concurrent {
		q = fmt.Sprintf(q, "CONCURRENTLY", name)
	} else {
		q = fmt.Sprintf(q, "", name)
	}

	if _, err := c.db.Exec(q); err != nil {
		c.log.Printf("error refreshing materialized view: %s: %v", name, err)
		return err
	}

	return nil
}

// refreshCache refreshes a Postgres materialized view if caching is disabled.
func (c *Core) refreshCache(name string, concurrent bool) error {
	if c.consts.CacheSlowQueries {
		return nil
	}

	return c.RefreshMatView(name, concurrent)
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
func makeSearchQuery(searchStr, orderBy, order, query string, querySortFields []string) (string, string) {
	searchStr = makeSearchString(searchStr)

	// Sort params.
	cleanOrderBy := models.FieldCreatedAt
	for _, f := range querySortFields {
		if f == orderBy {
			cleanOrderBy = f
			break
		}
	}

	cleanOrder := models.OrderDesc
	if order == models.OrderAsc {
		cleanOrder = models.OrderAsc
	}

	query = strings.ReplaceAll(query, "%order%", cleanOrderBy+" "+cleanOrder)

	return searchStr, query
}

// makeSearchString normalizes a search string for use in SQL search queries.
// Performance optimization: Avoids regexp overhead and allocations by using
// strings.Fields and strings.Join.
func makeSearchString(searchStr string) string {
	fields := strings.Fields(searchStr)
	if len(fields) == 0 {
		return ""
	}
	return strings.Join(fields, " ")
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
// lower casing and replacing whitespace with dashes.
// Performance optimization: Avoids regexp overhead, []byte conversions, and allocations
// by using strings.ToLower, strings.Fields and strings.Join.
func normalizeTags(tags []string) []string {
	var out []string

	for _, t := range tags {
		fields := strings.Fields(strings.ToLower(t))
		if len(fields) > 0 {
			out = append(out, strings.Join(fields, "-"))
		}
	}
	return out
}

// disallowedSQL matches statement-introducing DDL/DML keywords at the start
// of an expression. A filter expression is always embedded inside a WHERE
// clause (or a CTE body expecting a SELECT), so none of these are legitimate
// as the leading keyword. Anchoring to the start avoids false positives on
// subscriber data that merely contains these words (e.g. an email like
// "x@delete.io"). Multi-statement injection is already blocked by rejecting
// ";", so a statement keyword can only appear at the very start.
var disallowedSQL = regexp.MustCompile(`(?i)^(drop|truncate|create|alter|insert|update|delete|copy|vacuum|exec|execute)\b`)

// sanitizeSQLExp does basic sanitisation on arbitrary SQL query expressions
// coming from the frontend. It rejects multi-statement input, SQL comments
// and DDL/DML keywords so the expression can only be used as a WHERE clause.
func sanitizeSQLExp(q string) (string, error) {
	q = strings.TrimSpace(q)
	if len(q) == 0 {
		return "", nil
	}

	// Multi-statement queries and SQL comments are never part of a filter
	// expression; they are the classic SQL injection enablers.
	if strings.Contains(q, ";") || strings.Contains(q, "--") || strings.Contains(q, "/*") || strings.Contains(q, "*/") {
		return "", fmt.Errorf("unsafe SQL expression: multiple statements or comments are not allowed")
	}

	// DDL/DML keywords must not appear in a subscriber filter expression.
	if disallowedSQL.MatchString(q) {
		return "", fmt.Errorf("unsafe SQL expression: %q contains a disallowed statement", q)
	}

	if err := validateSQLExpFunctions(q); err != nil {
		return "", err
	}

	return q, nil
}

// allowedSQLExpFuncs is the allowlist of functions and parenthesised keyword
// constructs permitted in a subscriber filter expression. Anything else that
// parses as a function call is rejected. Arbitrary function calls
// (pg_sleep(), pg_read_file(), set_config(), dblink(), nextval(), ...) are
// the residual injection channel once multi-statement input, comments and
// non-allowlisted tables are locked down: they need no relation, so the
// EXPLAIN table check cannot see them, and several (pg_sleep) run fine
// inside the read-only transactions.
var allowedSQLExpFuncs = map[string]struct{}{
	// Parenthesised SQL constructs.
	"and": {}, "or": {}, "not": {}, "in": {}, "exists": {}, "any": {}, "all": {},
	"array": {}, "row": {}, "cast": {}, "values": {}, "filter": {},
	// Conditional / null-safe.
	"coalesce": {}, "nullif": {}, "greatest": {}, "least": {},
	// String functions.
	"lower": {}, "upper": {}, "initcap": {}, "length": {}, "char_length": {},
	"character_length": {}, "octet_length": {}, "bit_length": {},
	"btrim": {}, "ltrim": {}, "rtrim": {}, "trim": {},
	"left": {}, "right": {}, "repeat": {}, "replace": {}, "reverse": {},
	"split_part": {}, "strpos": {}, "position": {}, "substring": {}, "substr": {},
	"concat": {}, "concat_ws": {}, "lpad": {}, "rpad": {}, "translate": {},
	"starts_with": {}, "ascii": {}, "chr": {}, "md5": {},
	// Date/time functions.
	"now": {}, "age": {}, "date_trunc": {}, "date_part": {}, "extract": {},
	"to_char": {}, "to_date": {}, "to_timestamp": {}, "to_number": {}, "to_hex": {},
	"justify_days": {}, "justify_hours": {}, "justify_interval": {}, "timezone": {},
	"make_date": {}, "make_time": {}, "make_timestamp": {}, "make_interval": {},
	// Numeric functions.
	"abs": {}, "round": {}, "ceil": {}, "ceiling": {}, "floor": {}, "sign": {},
	"mod": {}, "trunc": {}, "sqrt": {}, "power": {}, "exp": {}, "ln": {}, "log": {},
	// Aggregates: never valid in a WHERE clause, so the DB rejects them anyway.
	"count": {}, "sum": {}, "avg": {}, "min": {}, "max": {},
	"bool_and": {}, "bool_or": {}, "string_agg": {}, "array_agg": {},
	// Misc built-ins with no side effects.
	"overlay": {},
	// Type names usable as cast calls or with precision arguments inside
	// CAST(... AS type(n)): side-effect-free conversions only.
	"int": {}, "int2": {}, "int4": {}, "int8": {}, "integer": {}, "bigint": {},
	"smallint": {}, "float4": {}, "float8": {}, "real": {}, "text": {},
	"bool": {}, "boolean": {}, "date": {}, "json": {}, "jsonb": {},
	"uuid": {}, "oid": {}, "name": {}, "bytea": {},
	"varchar": {}, "char": {}, "bpchar": {}, "character": {},
	"numeric": {}, "decimal": {}, "timestamp": {}, "timestamptz": {},
	"time": {}, "interval": {}, "bit": {},
}

// validateSQLExpFunctions scans a filter expression with a small lexer that
// skips string literals, quoted identifiers and dollar-quoted strings, and
// rejects any remaining identifier followed by "(" whose (final, lowercased)
// component is not in allowedSQLExpFuncs.
// skipSQLSpace returns the index of the next character at or after i that is
// neither whitespace nor a comment. PostgreSQL treats comments as whitespace
// everywhere (including between a function name and its parenthesis), and
// supports nested /* */ comments, so both -- and nested /* */ forms must be
// skipped to keep the allowlist from being bypassed with name/**/(args).
// An unterminated comment consumes the rest of the string; the DB then
// rejects the expression, which is fail-safe.
func skipSQLSpace(q string, i int) int {
	for i < len(q) {
		switch {
		case q[i] == ' ' || q[i] == '\t' || q[i] == '\n' || q[i] == '\r' || q[i] == '\v' || q[i] == '\f':
			i++
		case i+1 < len(q) && q[i] == '-' && q[i+1] == '-':
			for i < len(q) && q[i] != '\n' {
				i++
			}
		case i+1 < len(q) && q[i] == '/' && q[i+1] == '*':
			depth := 1
			i += 2
			for i < len(q) && depth > 0 {
				switch {
				case i+1 < len(q) && q[i] == '/' && q[i+1] == '*':
					depth++
					i += 2
				case i+1 < len(q) && q[i] == '*' && q[i+1] == '/':
					depth--
					i += 2
				default:
					i++
				}
			}
		default:
			return i
		}
	}
	return i
}

func validateSQLExpFunctions(q string) error {
	for i := 0; i < len(q); {
		c := q[i]
		switch {
		case c == '-' && i+1 < len(q) && q[i+1] == '-':
			// Line comment: skip its contents.
			for i < len(q) && q[i] != '\n' {
				i++
			}
		case c == '/' && i+1 < len(q) && q[i+1] == '*':
			// Block comment (PostgreSQL allows nesting): skip its contents.
			depth := 1
			i += 2
			for i < len(q) && depth > 0 {
				switch {
				case i+1 < len(q) && q[i] == '/' && q[i+1] == '*':
					depth++
					i += 2
				case i+1 < len(q) && q[i] == '*' && q[i+1] == '/':
					depth--
					i += 2
				default:
					i++
				}
			}
		case c == '\'':
			// String literal: skip, honouring '' escapes.
			i++
			for i < len(q) {
				if q[i] == '\'' {
					if i+1 < len(q) && q[i+1] == '\'' {
						i += 2
						continue
					}
					i++
					break
				}
				i++
			}
		case c == '"':
			// Quoted identifier: it can itself name a function ("pg_sleep"(1)).
			j := i + 1
			for j < len(q) {
				if q[j] == '"' {
					if j+1 < len(q) && q[j+1] == '"' {
						j += 2
						continue
					}
					break
				}
				j++
			}
			name := strings.ToLower(strings.ReplaceAll(q[i+1:min(j, len(q))], `""`, `"`))
			k := skipSQLSpace(q, j+1)
			if k < len(q) && q[k] == '(' {
				if _, ok := allowedSQLExpFuncs[name]; !ok {
					return fmt.Errorf("unsafe SQL expression: function %q is not allowed", name)
				}
			}
			i = j + 1
		case c == '$':
			// Dollar-quoted string $tag$...$tag$: skip if well-formed.
			j := i + 1
			for j < len(q) && (q[j] == '_' || (q[j] >= '0' && q[j] <= '9') ||
				(q[j] >= 'a' && q[j] <= 'z') || (q[j] >= 'A' && q[j] <= 'Z')) {
				j++
			}
			if j < len(q) && q[j] == '$' {
				tag := q[i : j+1]
				if end := strings.Index(q[j+1:], tag); end >= 0 {
					i = j + 1 + end + len(tag)
				} else {
					// Unterminated: let the DB reject it; stop scanning here.
					return nil
				}
			} else {
				i++
			}
		case c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z'):
			j := i + 1
			for j < len(q) && (q[j] == '_' || q[j] == '$' || (q[j] >= '0' && q[j] <= '9') ||
				(q[j] >= 'a' && q[j] <= 'z') || (q[j] >= 'A' && q[j] <= 'Z')) {
				j++
			}
			name := strings.ToLower(q[i:j])
			k := skipSQLSpace(q, j)
			if k < len(q) && q[k] == '(' {
				if _, ok := allowedSQLExpFuncs[name]; !ok {
					return fmt.Errorf("unsafe SQL expression: function %q is not allowed", name)
				}
			}
			i = j
		default:
			i++
		}
	}
	return nil
}

// strHasLen checks if the given string has a length within min-max.
func strHasLen(str string, min, max int) bool {
	return len(str) >= min && len(str) <= max
}
