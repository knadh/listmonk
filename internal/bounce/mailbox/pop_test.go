package mailbox

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/knadh/listmonk/models"
)

func TestClassifyBounce(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantType   string
		wantReason string
	}{
		{"hard status", "Status: 5.1.1\n", models.BounceTypeHard, "smtp_status=5.1.1"},
		{"soft status", "Status: 4.2.2\n", models.BounceTypeSoft, "smtp_status=4.2.2"},
		{"smtputf8 status is soft despite keywords", "Status: 5.6.7\nundeliverable\n", models.BounceTypeSoft, "smtp_status=5.6.7"},
		{"different subject", "Status: 5.60.1\n", models.BounceTypeHard, "smtp_status=5.60.1"},
		{"content status prefixed with reply code", "550 5.6.7 SMTPUTF8 required\n", models.BounceTypeSoft, "smtp_status=5.6.7"},
		{"status prefixed with reply code", "550 5.1.1 user unknown\n", models.BounceTypeHard, "smtp_status=5.1.1"},
		{"keyword fallback", "The address does not exist.\n", models.BounceTypeHard, "body_match=does not exist"},
		{"default", "Something went wrong.\n", models.BounceTypeSoft, "default"},
	}

	for _, tc := range tests {
		gotType, gotReason := classifyBounce([]byte(tc.body))
		if gotType != tc.wantType || gotReason != tc.wantReason {
			t.Errorf("%s: classifyBounce() = (%q, %q), want (%q, %q)", tc.name, gotType, gotReason, tc.wantType, tc.wantReason)
		}
	}
}

func TestParseBounceDSN(t *testing.T) {
	tests := []struct {
		file       string
		wantType   string
		wantReason string
	}{
		// 5.1.1 user unknown.
		{"dsn7.eml", models.BounceTypeHard, "smtp_status=5.1.1"},
		// 5.7.1 policy rejection.
		{"dsn6.eml", models.BounceTypeHard, "smtp_status=5.7.1"},
		// 5.6.7 SMTPUTF8 not supported by the destination.
		{"dsn9.eml", models.BounceTypeSoft, "smtp_status=5.6.7"},
	}

	for _, tc := range tests {
		b, err := os.ReadFile(filepath.Join("testdata", tc.file))
		if err != nil {
			t.Fatal(err)
		}

		out, err := parseBounce(b)
		if err != nil {
			t.Fatalf("%s: parseBounce() error: %v", tc.file, err)
		}
		if len(out) != 1 {
			t.Fatalf("%s: got %d bounces, want 1", tc.file, len(out))
		}

		if out[0].Type != tc.wantType {
			t.Errorf("%s: bounce type = %q, want %q", tc.file, out[0].Type, tc.wantType)
		}
		if got := string(out[0].Meta); !strings.Contains(got, tc.wantReason) {
			t.Errorf("%s: meta = %s, want classify_reason %q", tc.file, got, tc.wantReason)
		}
	}
}

func TestBounceStatus(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "dsn9.eml"))
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		status  string
		want    string
		wantErr bool
	}{
		{"5.6.0", models.BounceTypeSoft, false},
		{"5.6.1", models.BounceTypeSoft, false},
		{"5.6.7", models.BounceTypeSoft, false},
		{"5.6.9", models.BounceTypeSoft, false},
		{"5.60.1", models.BounceTypeHard, false},
		{"5.1.6", models.BounceTypeHard, false},
		{"5.4.4", models.BounceTypeHard, false},
		{"4.2.2", models.BounceTypeSoft, false},
		{"4.6.0", models.BounceTypeSoft, false},
		{"2.0.0", "", false},
		{"", "", true},
		{"garbage", "", true},
	} {
		t.Run(tc.status, func(t *testing.T) {
			// Exercise the same status through the fallback and structured DSN paths.
			if tc.want != "" {
				got, _ := classifyBounce([]byte("Status: " + tc.status + "\n"))
				if got != tc.want {
					t.Errorf("classifyBounce() = %q, want %q", got, tc.want)
				}
			}

			dsn := strings.Replace(string(raw), "5.6.7 (destination does not support SMTPUTF8 required for message delivery)", tc.status, 1)
			if tc.status == "2.0.0" {
				dsn = strings.Replace(dsn, "Action: failed", "Action: delivered", 1)
			}
			out, err := parseBounce([]byte(dsn))
			if (err != nil) != tc.wantErr {
				t.Fatalf("parseBounce() error = %v, want error = %v", err, tc.wantErr)
			}
			if tc.want == "" {
				if len(out) != 0 {
					t.Fatalf("got %d bounces, want none", len(out))
				}
			} else if len(out) != 1 || out[0].Type != tc.want {
				t.Fatalf("parseBounce() = %+v, want one %s bounce", out, tc.want)
			}
		})
	}
}
