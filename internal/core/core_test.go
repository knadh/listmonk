package core

import (
	"reflect"
	"testing"
)

func TestStrSliceContains(t *testing.T) {
	tests := []struct {
		str      string
		sl       []string
		expected bool
	}{
		{"apple", []string{"apple", "banana", "cherry"}, true},
		{"grape", []string{"apple", "banana", "cherry"}, false},
		{"", []string{"apple", "banana", "cherry"}, false},
		{"apple", []string{}, false},
	}

	for _, test := range tests {
		if result := strSliceContains(test.str, test.sl); result != test.expected {
			t.Errorf("strSliceContains(%q, %v) = %v; want %v", test.str, test.sl, result, test.expected)
		}
	}
}

func TestMakeSearchQuery(t *testing.T) {
	sortFields := []string{"name", "created_at", "updated_at"}
	queryTpl := "SELECT * FROM items ORDER BY %order%"

	tests := []struct {
		searchStr      string
		orderBy        string
		order          string
		expectedSearch string
		expectedQuery  string
	}{
		{
			searchStr:      "  hello   world  ",
			orderBy:        "name",
			order:          "asc",
			expectedSearch: "hello world",
			expectedQuery:  "SELECT * FROM items ORDER BY name asc",
		},
		{
			searchStr:      "test",
			orderBy:        "name; DROP TABLE users;",
			order:          "desc",
			expectedSearch: "test",
			expectedQuery:  "SELECT * FROM items ORDER BY created_at desc",
		},
		{
			searchStr:      "test",
			orderBy:        "name",
			order:          "asc; DROP TABLE users;",
			expectedSearch: "test",
			expectedQuery:  "SELECT * FROM items ORDER BY name desc",
		},
		{
			searchStr:      "",
			orderBy:        "invalid_field",
			order:          "invalid_order",
			expectedSearch: "",
			expectedQuery:  "SELECT * FROM items ORDER BY created_at desc",
		},
	}

	for _, tc := range tests {
		gotSearch, gotQuery := makeSearchQuery(tc.searchStr, tc.orderBy, tc.order, queryTpl, sortFields)
		if gotSearch != tc.expectedSearch {
			t.Errorf("makeSearchQuery searchStr = %q; want %q", gotSearch, tc.expectedSearch)
		}
		if gotQuery != tc.expectedQuery {
			t.Errorf("makeSearchQuery query = %q; want %q", gotQuery, tc.expectedQuery)
		}
	}
}

func TestNormalizeTags(t *testing.T) {
	tests := []struct {
		tags     []string
		expected []string
	}{
		{[]string{"tag1", "tag 2", "  tag3  "}, []string{"tag1", "tag-2", "tag3"}},
		{[]string{"  ", ""}, nil},
		{[]string{"TAG1", "Tag2"}, []string{"tag1", "tag2"}}, // tags should be lower-cased
		{[]string{"multi   space\ttag"}, []string{"multi-space-tag"}},
	}

	for _, test := range tests {
		result := normalizeTags(test.tags)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("normalizeTags(%v) = %v; want %v", test.tags, result, test.expected)
		}
	}
}

func BenchmarkMakeSearchString(b *testing.B) {
	input := "  hello   world   from   listmonk  search  "
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = makeSearchString(input)
	}
}

func BenchmarkNormalizeTags(b *testing.B) {
	tags := []string{"  NewsLetter ", "Weekly   Update", "TECH\ttag", " Announcements "}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = normalizeTags(tags)
	}
}

func TestStrHasLen(t *testing.T) {
	tests := []struct {
		str      string
		min, max int
		expected bool
	}{
		{"abc", 1, 5, true},
		{"abc", 3, 3, true},
		{"abc", 4, 5, false},
		{"", 0, 0, true},
	}

	for _, test := range tests {
		if result := strHasLen(test.str, test.min, test.max); result != test.expected {
			t.Errorf("strHasLen(%q, %d, %d) = %v; want %v", test.str, test.min, test.max, result, test.expected)
		}
	}
}

// TestSanitizeSQLExp ensures arbitrary SQL expressions from the frontend
// cannot inject additional statements or comments into a query.
func TestSanitizeSQLExp(t *testing.T) {
	valid := []string{
		"",
		"subscribers.name = 'foo'",
		"subscribers.id IN (SELECT id FROM lists)",
		"subscribers.status != 'deleted'",   // "deleted" is not the disallowed "delete" keyword
		"subscribers.updated_at > NOW()",    // "updated" is not the disallowed "update" keyword
		"subscribers.email = 'x@delete.io'", // "delete" inside data is fine
		"TRUE",
	}
	for _, q := range valid {
		out, err := sanitizeSQLExp(q)
		if err != nil {
			t.Errorf("sanitizeSQLExp(%q) returned error: %v", q, err)
		}
		if out != q {
			t.Errorf("sanitizeSQLExp(%q) = %q; want %q", q, out, q)
		}
	}

	invalid := []string{
		"true; DROP TABLE subscribers",
		"subscribers.id = 1; DELETE FROM subscribers",
		"true -- drop table subscribers",
		"true /* drop table */",
		"DROP TABLE subscribers",
		"DELETE FROM subscribers WHERE id = 1",
		"UPDATE subscribers SET status = 'blocklisted'",
		"1; TRUNCATE subscribers",
		"1 /* comment */",
	}
	for _, q := range invalid {
		if _, err := sanitizeSQLExp(q); err == nil {
			t.Errorf("sanitizeSQLExp(%q) should reject unsafe SQL", q)
		}
	}
}

func TestSanitizeSQLExpFunctionAllowlist(t *testing.T) {
	allowed := []string{
		"lower(email) LIKE '%@example.com'",
		"created_at > now() - interval '7 day'",
		"email ~* '@(a|b)\\.com$'",
		"substring(email from position('@' in email) + 1) = 'example.com'",
		"COALESCE(NULLIF(TRIM(attribs->>'company'), ''), 'unknown') = 'x'",
		"date_trunc('month', created_at) = date_trunc('month', now())",
		"list_id IN (1, 2, 3)",
		"EXISTS (SELECT 1 FROM subscriber_lists WHERE subscriber_id = subscribers.id)",
		// Strings and dollar-quoted strings containing parens are not calls.
		"email LIKE '%foo(%'",
		"name = $$weird( value$$",
		"name = 'it''s ('",
		// Whitespace between a safe function and its paren is fine
		// (comments are rejected by the expression gate).
		"lower (email) = 'x'",
		// Cast type names with precision arguments.
		"CAST(attribs->>'name' AS varchar(255)) = 'x'",
		"(attribs->>'age')::numeric(10, 2) > 5",
		// Misc allowed built-ins.
		"overlay(email placing '***' from 1 for 3) LIKE '%@%'",
		// A quoted identifier concatenated, not called, is not a function call.
		`"PG" || '_sleep' = 'PG_sleep'`,
	}
	for _, q := range allowed {
		if _, err := sanitizeSQLExp(q); err != nil {
			t.Errorf("sanitizeSQLExp(%q) rejected a legitimate expression: %v", q, err)
		}
	}

	rejected := []string{
		// DoS via read-only-transaction-safe sleep.
		"1=1 OR pg_sleep(60) IS NULL",
		"pg_sleep ( 60 ) IS NULL",
		// Schema-qualified and quoted-identifier obfuscation.
		"pg_catalog.pg_sleep(1) IS NOT NULL",
		`"pg_sleep"(1) IS NULL`,
		// Server file reads / settings / side channels.
		"pg_read_file('/etc/passwd') IS NOT NULL",
		"set_config('log_statement', 'all', false) IS NULL",
		"dblink('host=x', 'SELECT 1') IS NOT NULL",
		"lo_import('/etc/passwd') IS NOT NULL",
		"pg_terminate_backend(1) IS NULL",
		"pg_notify('x', 'y') IS NULL",
		// Sequence mutation.
		"nextval('subscribers_id_seq') IS NOT NULL",
		// Unknown/custom functions are rejected by default.
		"my_custom_function(1) = 2",
		// Hidden inside a legit outer call.
		"lower(pg_sleep(1)::text) = 'x'",
		// Comments are whitespace in PostgreSQL; a comment anywhere is
		// rejected by the expression gate before the allowlist even runs.
		"pg_sleep/**/(1) IS NULL",
	}
	for _, q := range rejected {
		if _, err := sanitizeSQLExp(q); err == nil {
			t.Errorf("sanitizeSQLExp(%q) accepted a dangerous function call", q)
		}
	}
}

// TestValidateSQLExpFunctionsCommentBypass exercises the allowlist lexer
// directly: PostgreSQL treats comments (including nested /* */) as
// whitespace, so a call may be written name/**/(args). The lexer must not
// let that slip past even though the outer expression gate also rejects
// comments (defence in depth).
func TestValidateSQLExpFunctionsCommentBypass(t *testing.T) {
	rejected := []string{
		"pg_sleep/**/(1) IS NULL",
		"pg_sleep /* comment */ (1) IS NULL",
		"pg_sleep /* nested /* comment */ tail */ (1) IS NULL",
		"pg_sleep-- line comment\n(1) IS NULL",
		`"pg_sleep"/**/(1) IS NULL`,
		"pg_catalog/**/./**/pg_sleep/**/(1) IS NULL",
		"pg_catalog . pg_sleep (1) IS NULL",
	}
	for _, q := range rejected {
		if err := validateSQLExpFunctions(q); err == nil {
			t.Errorf("validateSQLExpFunctions(%q) accepted a comment-obfuscated call", q)
		}
	}

	allowed := []string{
		"lower /* note */ (email) = 'x'",
		"lower-- note\n(email) = 'x'",
		"email LIKE '%-- not a comment%'",
		"email LIKE '%/* not a comment */%'",
	}
	for _, q := range allowed {
		if err := validateSQLExpFunctions(q); err != nil {
			t.Errorf("validateSQLExpFunctions(%q) false positive: %v", q, err)
		}
	}
}
