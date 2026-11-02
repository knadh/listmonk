package i18n

import (
	"os"
	"testing"
	"time"
)

func TestFormatDate(t *testing.T) {
	date := time.Date(2026, time.October, 28, 15, 4, 5, 0, time.UTC)
	for _, tc := range []struct {
		lang, layout, want string
	}{
		{"en", "_.date_long", "October 28, 2026"},
		{"en", "_.date_short", "Wed, 28 Oct 2026"},
		{"fr", "_.date_long", "28 octobre 2026"},
		{"fr", "_.date_short", "mer. 28 oct. 2026"},
		{"fr", "Monday 02 January 2006, 15:04:05 MST", "mercredi 28 octobre 2026, 15:04:05 UTC"},
		{"fr", time.RFC3339, "2026-10-28T15:04:05Z"},
	} {
		t.Run(tc.lang+"/"+tc.layout, func(t *testing.T) {
			b, err := os.ReadFile("../../i18n/" + tc.lang + ".json")
			if err != nil {
				t.Fatal(err)
			}
			i, err := New(b)
			if err != nil {
				t.Fatal(err)
			}
			if got := i.FormatDate(date, i.T(tc.layout)); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFormatDateFallback(t *testing.T) {
	i, err := New([]byte(`{"_.code":"test","_.name":"Test","dates.months":"invalid","dates.weekdaysShort":""}`))
	if err != nil {
		t.Fatal(err)
	}
	for month := time.January; month <= time.December; month++ {
		for day := 1; day <= 7; day++ {
			date := time.Date(2026, month, day, 15, 4, 5, 0, time.UTC)
			layout := "Monday Mon January Jan 02 2006 15:04:05"
			if got, want := i.FormatDate(date, layout), date.Format(layout); got != want {
				t.Fatalf("got %q, want %q", got, want)
			}
		}
	}
}
