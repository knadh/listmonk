package core

import "testing"

func TestCheckSegmentExpr(t *testing.T) {
	// Expressions that could break out of the enclosing `AND (...)` in the query.
	for _, c := range []string{
		"1=1) OR (1=1",
		"subscribers.attribs->>'lang' = 'tr')",
		"(subscribers.attribs->>'lang' = 'tr'",
		"subscribers.attribs->>'lang' = 'tr'; DELETE FROM subscribers",
		"subscribers.attribs->>'lang' = 'tr' -- )",
		"subscribers.attribs->>'lang' = 'tr' /* ) */",
		"subscribers.attribs->>'lang' = $$ ) $$",
		"subscribers.name = 'unterminated",
		`subscribers.name = "unterminated`,
		`subscribers.name = E'\') OR (TRUE`,
	} {
		if err := checkSegmentExpr(c); err == nil {
			t.Errorf("checkSegmentExpr(%q): expected error", c)
		}
	}

	// Valid expressions. Parentheses, comment markers, and semicolons inside literals are fine.
	for _, c := range []string{
		"",
		"subscribers.attribs->>'tenant_id' = 'tenant_x' AND subscribers.attribs->>'lang' = 'tr'",
		"(subscribers.attribs->'address'->>'city' = 'Ankara' OR (subscribers.attribs->>'lang' = 'en'))",
		"(subscribers.attribs->>'score')::INT > 10",
		"subscribers.name = ')' AND subscribers.email LIKE '%--%'",
		`subscribers.name = E'it\'s ) here'`,
		// A single literal: \\ is a backslash and '' is a quote.
		`subscribers.name = E'\\'') OR (1=1'`,
		`"name" = 'a;b'`,
	} {
		if err := checkSegmentExpr(c); err != nil {
			t.Errorf("checkSegmentExpr(%q) unexpected error: %v", c, err)
		}
	}
}

func TestIsIdentChar(t *testing.T) {
	for _, c := range []byte("azAZ09_") {
		if !isIdentChar(c) {
			t.Errorf("isIdentChar(%q) = false; want true", c)
		}
	}
	for _, c := range []byte(" '\"().-;$") {
		if isIdentChar(c) {
			t.Errorf("isIdentChar(%q) = true; want false", c)
		}
	}
}

func TestMakeSegmentCountStmt(t *testing.T) {
	if got := makeSegmentCountStmt("SELECT 1"); got != "SELECT COUNT(*) FROM (SELECT 1) AS segment" {
		t.Errorf("makeSegmentCountStmt() = %q", got)
	}
}
