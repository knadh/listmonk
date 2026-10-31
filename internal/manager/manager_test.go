package manager

import (
	"mime"
	"testing"
)

// TestMakeAttachmentHeaderEncoding asserts that the Content-Disposition and
// Content-Type headers emitted for an attachment carry no raw bytes > 0x7F.
// A non-ASCII byte inside a header makes the whole message SMTPUTF8-dependent
// (RFC 6531 §3.1), and MXs that do not advertise SMTPUTF8 reject it with
// "5.6.7 destination does not support SMTPUTF8 required for message delivery"
// (knadh/listmonk#3316, knadh/listmonk#1495). smtppool writes these two
// headers verbatim, so the filename parameter must be pre-encoded here per
// RFC 2231/5987. The assertions also cover ASCII filenames that contain
// spaces or quotes, which are emitted as invalid syntax today
// (e.g. "filename=my file.pdf").
func TestMakeAttachmentHeaderEncoding(t *testing.T) {
	cases := []struct {
		name     string
		filename string
	}{
		{"plain ASCII", "report.pdf"},
		{"ASCII with space", "my file.pdf"},
		{"ASCII with quotes", `say "hi".txt`},
		{"non-ASCII CJK", "汉语.txt"},
		{"non-ASCII Czech", "Příloha ř.pdf"},
		{"non-ASCII emoji", "📊 stats.csv"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := MakeAttachmentHeader(c.filename, "base64", "application/pdf")

			for _, key := range []string{"Content-Disposition", "Content-Type"} {
				v := h.Get(key)
				for i := 0; i < len(v); i++ {
					if v[i] > 0x7F {
						t.Errorf("%s %q contains raw non-ASCII byte 0x%X at offset %d; the message would require SMTPUTF8", key, v, v[i], i)
						break
					}
				}

				// The parameter value must be recoverable by a
				// standards-compliant parser, i.e. the emitted syntax must
				// be valid.
				_, params, err := mime.ParseMediaType(v)
				if err != nil {
					t.Errorf("%s %q does not parse: %v", key, v, err)
					continue
				}
				param := "filename"
				if key == "Content-Type" {
					param = "name"
				}
				if got := params[param]; got != c.filename {
					t.Errorf("%s %q round-trips to filename %q, want %q", key, v, got, c.filename)
				}
			}
		})
	}
}
