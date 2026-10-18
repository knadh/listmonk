package models

import (
	"bytes"
	"html/template"
	"reflect"
	"strings"
	"testing"
)

func TestTrackLinkUnicode(t *testing.T) {
	f := func(url string) {
		t.Helper()
		r := regTplFuncs[1]
		input := url + "@TrackLink"
		if got := r.regExp.FindStringSubmatch(input); len(got) != 2 || got[1] != url {
			t.Fatalf("incorrect URL match: %q", got)
		}

		if got, want := r.regExp.ReplaceAllString(input, r.replace), `{{ TrackLink "`+url+`" . }}`; got != want {
			t.Fatalf("replacement = %q, want %q", got, want)
		}

		testTrackLinkCompile(t, url, CampaignContentTypeHTML)
		testTrackLinkCompile(t, url, CampaignContentTypeRichtext)
		testTrackLinkCompile(t, url, CampaignContentTypePlain)
		testTrackLinkCompile(t, url, CampaignContentTypeMarkdown)
	}

	// ASCII.
	f("http://example.com/path?q=a%20b&x=$1#fragment")
	f("https://[::1]:8080/a_b-~!$&'()*+,;=:@?q=%22%5C%7B%7D")

	// Unicode TLDs.
	f("https://例え.jp/日本語?数=１２３")

	// Unicode scripts, ZWNJs.
	f("https://example.com/മലയാളം/ക്\u200c/ക\u0d46\u0d3e")
	f("https://example.com/தமிழ்/க\u0bc6\u0bbe")
	f("https://example.com/中文?语言=汉语")

	// Emojis.
	f("https://client.platform.com/sites/Project/Documents%20shared/Client%20form%20✅.aspx?web=1&param=value")
	f("https://client.platform.com/:x:/r/sites/Management/Shared%20Documents/Reports/2026/Marketing%20campaign%20report%20-%20final%20version%20📊.xlsx?d=w123456789abcdef&csf=1&web=1&e=AbCdEf")
	f("https://client.example.com/document?id=12345&name=client-presentation&status=validated&emoji=✅")
	f("https://client.platform.com/sites/Project/Shared%20Documents/Team%20meeting%20notes%20🚀.docx?web=1&download=1")
	f("https://example.com/👩🏽‍💻/❤️/1️⃣/🇮🇳")
	f("https://example.com/🏴\U000e0067\U000e0062\U000e0065\U000e006e\U000e0067\U000e007f")
	// Unicode symbols and punctuation.
	f("https://example.com/€£₹/∑∞≠/‘hello’—world…")
	f("https://example.com/" + strings.Repeat("文🚀e\u0301", 1000) + "?download=1")
}

func TestTrackLinkBoundaries(t *testing.T) {
	f := func(separator string) {
		t.Helper()
		r := regTplFuncs[1]
		input := "https://example.com/a" + separator + "b@TrackLink"
		if got := r.regExp.ReplaceAllString(input, r.replace); got != input {
			t.Fatalf("matched across separator: %q became %q", input, got)
		}
	}

	// ASCII whitespace and controls (DEL, C1 chars).
	f(" ")
	f("\t")
	f("\r")
	f("\n")
	f("\v")
	f("\f")
	f("\x00")
	f("\x7f")
	f("\u0085")
	f("\u009f")
	// Unicode space, line separators.
	f("\u00a0")
	f("\u2003")
	f("\u2028")
	f("\u2029")
	f("\u3000")
	// HTML and Go template delims and escape chars.
	f(`"`)
	f(`\`)
	f("<")
	f(">")
	f("{")
	f("}")
	f("`")
}

func TestTrackLinkReplacement(t *testing.T) {
	f := func(input, want string) {
		t.Helper()
		r := regTplFuncs[1]
		if got := r.regExp.ReplaceAllString(input, r.replace); got != want {
			t.Fatalf("replacement = %q, want %q", got, want)
		}
	}

	// Adjacent links must be untouched.
	f(`<a href="https://example.com/🚀@TrackLink">one</a><a href="http://example.com/é@TrackLink">two</a>`,
		`<a href="{{ TrackLink "https://example.com/🚀" . }}">one</a><a href="{{ TrackLink "http://example.com/é" . }}">two</a>`)
	f(`https://example.com/€@TrackLink https://example.com/∑@TrackLink`,
		`{{ TrackLink "https://example.com/€" . }} {{ TrackLink "https://example.com/∑" . }}`)
	// Unmarked URLs + unsupported schemes remain untouched.
	f("https://example.com/🚀", "https://example.com/🚀")
	f("ftp://example.com/🚀@TrackLink", "ftp://example.com/🚀@TrackLink")
}

func testTrackLinkCompile(t *testing.T, url, contentType string) {
	t.Helper()
	link := `<a href="` + url + "@TrackLink" + `">link</a>`
	c := Campaign{ContentType: contentType, Body: link, TemplateBody: link + `{{ template "content" . }}`}

	var gotURLs []string
	context := &struct{ ID int }{ID: 1}
	if err := c.CompileTemplate(template.FuncMap{
		"TrackLink": func(url string, data any) string {
			if data != context {
				t.Error("TrackLink received incorrect context")
			}

			gotURLs = append(gotURLs, url)
			return "https://tracking.example/link"
		},
	}); err != nil {
		t.Fatalf("%s: %v", contentType, err)
	}

	var out bytes.Buffer
	if err := c.Tpl.ExecuteTemplate(&out, BaseTpl, context); err != nil {
		t.Fatalf("%s: %v", contentType, err)
	}
	if want := []string{url, url}; !reflect.DeepEqual(gotURLs, want) {
		t.Fatalf("%s: TrackLink URLs = %q, want %q", contentType, gotURLs, want)
	}

	tracked := `<a href="https://tracking.example/link">link</a>`
	want := tracked + tracked
	if contentType == CampaignContentTypeMarkdown {
		want = tracked + "<p>" + tracked + "</p>\n"
	}

	// Matches?
	if out.String() != want {
		t.Fatalf("%s: rendered output = %q, want %q", contentType, out.String(), want)
	}
}
