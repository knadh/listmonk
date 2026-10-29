package manager

import (
	"bytes"
	"io"
	"log"
	"strings"
	"testing"

	"github.com/knadh/listmonk/models"
)

// mediaStore resolves inline images from the media library by filename.
type mediaStore struct {
	Store
	lookups []string
}

func (s *mediaStore) GetInlineAttachmentByFilename(fname string) (models.Attachment, string, error) {
	s.lookups = append(s.lookups, fname)
	cid := MakeContentID(fname)
	return models.Attachment{Name: fname, Content: []byte("x"), IsInline: true}, cid, nil
}

// 1x1 transparent PNG.
const testPNG = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=="

func TestApplyInlineImagesDataURI(t *testing.T) {
	var logBuf bytes.Buffer
	m := &Manager{log: log.New(&logBuf, "", 0)}

	src := "data:image/png;base64," + testPNG
	body := `<p><img src="` + src + `" width="30" alt="" data-embed="true"></p>` +
		`<img src="` + src + `" data-embed="true">`

	out, atts := m.ApplyInlineImages(body)
	if len(atts) != 1 {
		t.Fatalf("got %d attachments, want 1 (deduplicated)", len(atts))
	}

	a := atts[0]
	cid := MakeContentID(src)
	if !a.IsInline || len(a.Content) == 0 {
		t.Fatalf("expected a non-empty inline attachment: %+v", a)
	}
	if got := a.Header.Get("Content-ID"); got != "<"+cid+">" {
		t.Fatalf("Content-ID = %q, want <%s>", got, cid)
	}
	if got := a.Header.Get("Content-Type"); !strings.HasPrefix(got, "image/png;") {
		t.Fatalf("Content-Type = %q, want image/png", got)
	}
	if !strings.HasSuffix(a.Name, ".png") {
		t.Fatalf("Name = %q, want a .png filename", a.Name)
	}
	if strings.Contains(out, "data:") || strings.Count(out, `src="cid:`+cid+`"`) != 2 {
		t.Fatalf("img src not rewritten to cid: %s", out)
	}
	if logBuf.Len() != 0 {
		t.Fatalf("unexpected log output: %s", logBuf.String())
	}
}

func TestApplyInlineImagesMixed(t *testing.T) {
	st := &mediaStore{}
	m := &Manager{store: st, log: log.New(io.Discard, "", 0)}

	body := `<img src="https://example.com/uploads/logo.png" data-embed="true">` +
		`<img src="data:image/jpeg;base64,` + testPNG + `" data-embed="true">` +
		`<img src="https://example.com/not-embedded.png">`

	out, atts := m.ApplyInlineImages(body)
	if len(atts) != 2 {
		t.Fatalf("got %d attachments, want 2", len(atts))
	}
	if len(st.lookups) != 1 || st.lookups[0] != "logo.png" {
		t.Fatalf("media lookups = %v, want [logo.png]", st.lookups)
	}
	if !strings.Contains(out, `src="cid:`+MakeContentID("logo.png")+`"`) {
		t.Fatalf("media image not rewritten: %s", out)
	}
	if !strings.Contains(out, `<img src="https://example.com/not-embedded.png">`) {
		t.Fatalf("non-embedded image changed: %s", out)
	}
}

func TestApplyInlineImagesInvalidDataURI(t *testing.T) {
	var logBuf bytes.Buffer
	m := &Manager{log: log.New(&logBuf, "", 0)}

	for _, src := range []string{
		"data:text/plain;base64,aGVsbG8=",
		"data:image/svg+xml,%3Csvg%3E%3C/svg%3E",
		"data:image/png;base64,!!notbase64!!",
		"data:image/png;base64",
	} {
		logBuf.Reset()
		body := `<img src="` + src + `" data-embed="true">`
		out, atts := m.ApplyInlineImages(body)
		if len(atts) != 0 || out != body {
			t.Fatalf("%q: expected body untouched and no attachments, got %d: %s", src, len(atts), out)
		}
		if !strings.Contains(logBuf.String(), "not embedded") {
			t.Fatalf("%q: expected a log line, got %q", src, logBuf.String())
		}
	}

	// Long data URIs are truncated in the log.
	logBuf.Reset()
	long := "data:text/plain;base64," + strings.Repeat("A", 5000)
	m.ApplyInlineImages(`<img src="` + long + `" data-embed="true">`)
	if logBuf.Len() > 300 {
		t.Fatalf("log line not truncated: %d bytes", logBuf.Len())
	}
}

func TestApplyArchiveImagesDataURI(t *testing.T) {
	// A nil store panics if a data: URI is looked up as a media filename.
	m := &Manager{log: log.New(io.Discard, "", 0)}

	body := `<img src="data:image/png;base64,` + testPNG + `" data-embed="true">`
	if got := m.applyArchiveImages(body, map[string]string{}); got != body {
		t.Fatalf("archive body changed: %s", got)
	}
}
