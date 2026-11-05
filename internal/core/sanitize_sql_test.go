package core

import "testing"

// TestSanitizeSQLExpRejectsInjection covers the subscriber filter expression
// sanitizer. These expressions are interpolated into raw SQL, so anything that
// could break out of the WHERE clause must be rejected.
func TestSanitizeSQLExpRejectsInjection(t *testing.T) {
	reject := []string{
		"; DROP TABLE subscribers; --",
		"-- comment",
		"/* comment */",
		"DROP TABLE subscribers",
		"TRUNCATE subscribers",
		"CREATE TABLE x",
		"ALTER TABLE subscribers",
		"INSERT INTO subscribers",
		"UPDATE subscribers SET x=1",
		"DELETE FROM subscribers",
		"COPY subscribers TO '/tmp/x'",
		"VACUUM",
		"EXEC pg_sleep(5)",
		"EXECUTE pg_sleep(5)",
		"1=1; SELECT pg_sleep(5)",
		"pg_sleep(5)",
		"pg_read_file('/etc/passwd')",
		"set_config('x','y')",
		"dblink('x','y')",
		"nextval('x')",
		"1 AND pg_sleep(5)",
		"name = 'x'; DROP TABLE subscribers",
	}
	for _, q := range reject {
		if _, err := sanitizeSQLExp(q); err == nil {
			t.Errorf("sanitizeSQLExp(%q) should have been rejected, got nil error", q)
		}
	}

	accept := []string{
		"",
		"name = 'foo'",
		"email LIKE '%example.com'",
		"status = 1 AND (list_id = 2 OR list_id = 3)",
		"lower(email) = lower('FOO@Example.COM')",
		"length(name) > 3",
		"created_at > now() - interval '7 days'",
		"coalesce(name, '') <> ''",
		"name ILIKE '%foo%'",
	}
	for _, q := range accept {
		if _, err := sanitizeSQLExp(q); err != nil {
			t.Errorf("sanitizeSQLExp(%q) should have been accepted, got %v", q, err)
		}
	}
}
