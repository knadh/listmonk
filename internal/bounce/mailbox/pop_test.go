package mailbox

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/knadh/listmonk/models"
)

func TestStatusBounceType(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		// 5.x.x are hard bounces.
		{"5.1.1", models.BounceTypeHard},
		{"5.4.4", models.BounceTypeHard},
		{"5.7.1", models.BounceTypeHard},

		// 5.6.x are message content / media failures and not recipient failures.
		{"5.6.0", models.BounceTypeSoft},
		{"5.6.1", models.BounceTypeSoft},
		{"5.6.7", models.BounceTypeSoft},
		{"5.6.9", models.BounceTypeSoft},

		// A different class or subject sharing the "6" must not match.
		{"5.60.1", models.BounceTypeHard},
		{"5.1.6", models.BounceTypeHard},

		// 4.x.x are soft bounces.
		{"4.2.2", models.BounceTypeSoft},
		{"4.6.0", models.BounceTypeSoft},

		// Anything else is not a failure status.
		{"2.0.0", ""},
		{"", ""},
		{"garbage", ""},
	}

	for _, tc := range tests {
		if got := statusBounceType(tc.status); got != tc.want {
			t.Errorf("statusBounceType(%q) = %q, want %q", tc.status, got, tc.want)
		}
	}
}

func TestClassifyBounce(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantType   string
		wantReason string
	}{
		{"hard status", "Status: 5.1.1\n", models.BounceTypeHard, "smtp_status=5.1.1"},
		{"soft status", "Status: 4.2.2\n", models.BounceTypeSoft, "smtp_status=4.2.2"},
		{"smtputf8 status is soft", "Status: 5.6.7\n", models.BounceTypeSoft, "smtp_status=5.6.7"},
		{"smtputf8 status is soft despite keywords", "Status: 5.6.7\nundeliverable\n", models.BounceTypeSoft, "smtp_status=5.6.7"},
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
