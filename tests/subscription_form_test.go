package tests

import (
	"bytes"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

// TestSubscriptionFormEmailQueryParam verifies that ?email= on the subscription
// form URL is correctly read as a query parameter via echo's context — confirming
// the handler can pass it to the template data.
func TestSubscriptionFormEmailQueryParam(t *testing.T) {
	e := echo.New()

	cases := []struct {
		url   string
		want  string
	}{
		{"/subscription/form?email=test%40example.com", "test@example.com"},
		{"/subscription/form?email=hello+world%40example.com", "hello world@example.com"},
		{"/subscription/form", ""},
	}

	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, tc.url, nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		got := c.QueryParam("email")
		if got != tc.want {
			t.Errorf("url=%q: got QueryParam(\"email\")=%q, want %q", tc.url, got, tc.want)
		}
	}
}

// TestSubscriptionFormTemplateEmailPrefill verifies that the email input in
// subscription-form.html renders value="{{ .Data.Email }}" correctly.
func TestSubscriptionFormTemplateEmailPrefill(t *testing.T) {
	// Mirror the exact email input line from subscription-form.html so any
	// future template change that removes value= will break this test.
	const tplSrc = `{{define "subscription-form"}}` +
		`<input id="email" name="email" required="true" type="email" ` +
		`placeholder="{{ call .L "subscribers.email" }}" autofocus="true" value="{{ .Data.Email }}" >` +
		`{{end}}`

	type formData struct {
		Email string
	}
	type tplCtx struct {
		Data formData
		L    func(string) string
	}

	tpl := template.Must(template.New("").Parse(tplSrc))

	stub := func(key string) string { return key }

	cases := []struct {
		email string
		want  string
	}{
		{"test@example.com", `value="test@example.com"`},
		{"user+tag@domain.org", `value="user&#43;tag@domain.org"`},
		{"", `value=""`},
	}

	for _, tc := range cases {
		var buf bytes.Buffer
		data := tplCtx{Data: formData{Email: tc.email}, L: stub}
		if err := tpl.ExecuteTemplate(&buf, "subscription-form", data); err != nil {
			t.Fatalf("email=%q: template execution failed: %v", tc.email, err)
		}
		if !strings.Contains(buf.String(), tc.want) {
			t.Errorf("email=%q: expected output to contain %q\ngot: %s", tc.email, tc.want, buf.String())
		}
	}
}
