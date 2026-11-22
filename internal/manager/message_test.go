package manager

import (
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/knadh/listmonk/internal/urlsign"
	"github.com/knadh/listmonk/models"
)

func TestSignedMessageLinks(t *testing.T) {
	signer := urlsign.New([]byte(strings.Repeat("k", 32)), true, time.Hour, "")
	m := New(Config{
		UnsubURL:   "https://example.com/news/subscription/%s/%s",
		ManageURL:  "https://example.com/news/subscription/%s/%s?manage=true",
		OptinURL:   "https://example.com/news/subscription/optin/%s?%s",
		MessageURL: "https://example.com/news/campaign/%s/%s",
	}, nil, nil, signer, nil)

	// Include both current helpers and legacy URL suffixes.
	c := &models.Campaign{
		UUID:        "campaign",
		ContentType: models.CampaignContentTypeHTML,
		ListUUIDs:   []string{"cccc", "dddd"},
		Body: `<a href="{{ UnsubscribeURL }}">unsub</a>
<a href="{{ ManageURL }}">manage</a>
<a href="{{ MessageURL }}">message</a>
<a href="{{ OptinURL }}l=aaaa&l=bbbb">legacy optin</a>
<a href="{{ OptinURL }}l=aaaa&amp;l=bbbb">escaped legacy optin</a>
<a href="{{ OptinURL . "l=aaaa&l=bbbb" }}">optin</a>
<a href="{{ UnsubscribeURL }}?manage=true">legacy manage</a>
<a href="{{ OptinURL }}">campaign optin</a>`,
	}
	c.AltBody.String = "{{ UnsubscribeURL }}"
	c.AltBody.Valid = true
	if err := c.CompileTemplate(m.TemplateFuncs(c)); err != nil {
		t.Fatal(err)
	}

	// All links in a message should have the same expiry.
	expiry := signer.GetExpiry()

	msg, err := m.NewCampaignMessage(c, models.Subscriber{UUID: "subscriber"}, expiry)
	if err != nil {
		t.Fatal(err)
	}

	// Plaintext should retain '&' in signed URLs.
	if string(msg.AltBody()) != msg.unsubURL {
		t.Fatalf("plaintext alternative escaped link: %s", msg.AltBody())
	}

	links := regexp.MustCompile(`href="([^"]+)"`).FindAllStringSubmatch(string(msg.Body()), -1)
	if len(links) != 8 {
		t.Fatalf("wrong links: %s", msg.Body())
	}

	// HTML escaping should preserve signature when decoded.
	urls := make([]*url.URL, len(links))
	for i, l := range links {
		u, err := url.Parse(html.UnescapeString(l[1]))
		if err != nil {
			t.Fatal(err)
		}

		if u.Query().Get("exp") != strconv.FormatInt(expiry.Unix(), 10) {
			t.Fatalf("message link did not inherit expiry: %s", u)
		}
		if _, ok := signer.Verify(u); !ok {
			t.Fatalf("unsigned link: %s", l[1])
		}
		urls[i] = u
	}

	// Body === header links.
	unsub, manage, legacyManage := urls[0], urls[1], urls[6]
	if unsub.String() != msg.unsubURL {
		t.Fatal("header/body unsubscribe URL mismatch")
	}

	if manage.Query().Get("manage") != "true" || legacyManage.Query().Get("manage") != "true" {
		t.Fatal("missing manage parameter")
	}

	// Explicit arguments + legacy separators should have the same lists.
	for _, optin := range urls[3:6] {
		if strings.Join(optin.Query()["l"], ",") != "aaaa,bbbb" {
			t.Fatalf("lost opt-in scope: %v", optin)
		}
	}
	if got := strings.Join(urls[7].Query()["l"], ","); got != "cccc,dddd" {
		t.Fatalf("incorrect campaign opt-in scope: %s", got)
	}

	// Adding a list later cannot expand the link already sent.
	c.ListUUIDs = append(c.ListUUIDs, "eeee")
	tampered := *urls[7]
	q := tampered.Query()
	q.Add("l", "eeee")
	tampered.RawQuery = q.Encode()
	if _, ok := signer.Verify(&tampered); ok {
		t.Fatal("accepted expanded opt-in scope")
	}

	// A template preview without campaign lists must not generate an unscoped link.
	c.ListUUIDs = nil
	c.Body = "{{ OptinURL }}"
	if err := c.CompileTemplate(m.TemplateFuncs(c)); err != nil {
		t.Fatal(err)
	}
	msg, err = m.NewCampaignMessage(c, models.Subscriber{UUID: "subscriber"})
	if err != nil {
		t.Fatal(err)
	}
	if len(msg.Body()) != 0 {
		t.Fatalf("generated unscoped opt-in link: %s", msg.Body())
	}
}

func TestHostedMessageExpiry(t *testing.T) {
	signer := urlsign.New([]byte(strings.Repeat("k", 32)), true, 120*24*time.Hour, "")
	m := New(Config{
		UnsubURL:   "https://example.com/subscription/%s/%s",
		ManageURL:  "https://example.com/subscription/%s/%s?manage=true",
		OptinURL:   "https://example.com/subscription/optin/%s?%s",
		MessageURL: "https://example.com/campaign/%s/%s",
	}, nil, nil, signer, nil)

	c := &models.Campaign{
		UUID:        "campaign",
		ContentType: models.CampaignContentTypePlain,
		ListUUIDs:   []string{"aaaa", "bbbb"},
		Body: `{{ UnsubscribeURL }}
{{ ManageURL }}
{{ OptinURL }}
{{ MessageURL }}`,
	}
	if err := c.CompileTemplate(m.TemplateFuncs(c)); err != nil {
		t.Fatal(err)
	}

	// Assert link expiry.
	for _, expiry := range []time.Time{{}, time.Now().Add(-time.Minute)} {
		// The zero time is encoded as exp=0 not a Unix timestamp.
		var unixExpiry int64
		if !expiry.IsZero() {
			unixExpiry = expiry.Unix()
		}

		msg, err := m.NewCampaignMessage(c, models.Subscriber{UUID: "subscriber"}, expiry)
		if err != nil {
			t.Fatal(err)
		}

		// Rendering an archive message shouldn't result in TTL renewal.
		for _, raw := range strings.Fields(string(msg.Body())) {
			u, err := url.Parse(raw)
			if err != nil {
				t.Fatal(err)
			}

			if u.Query().Get("exp") != strconv.FormatInt(unixExpiry, 10) {
				t.Fatalf("hosted link renewed: %s", u)
			}
			if _, ok := signer.Verify(u); ok != expiry.IsZero() {
				t.Fatalf("incorrect inherited expiry: %s", u)
			}
		}
	}
}

func TestLegacyOptinScope(t *testing.T) {
	m := New(Config{OptinURL: "https://example.com/subscription/optin/%s?%s"}, nil, nil, urlsign.New(nil, false, 0, ""), nil)
	for _, lists := range [][]string{nil, {"aaaa", "bbbb"}} {
		c := &models.Campaign{ContentType: models.CampaignContentTypePlain, ListUUIDs: lists, Body: `{{ OptinURL }}`}
		if err := c.CompileTemplate(m.TemplateFuncs(c)); err != nil {
			t.Fatal(err)
		}
		msg, err := m.NewCampaignMessage(c, models.Subscriber{UUID: "subscriber"})
		if err != nil {
			t.Fatal(err)
		}
		if got, want := string(msg.Body()), "https://example.com/subscription/optin/subscriber?"; got != want {
			t.Fatalf("legacy opt-in scope changed: got %q, want %q", got, want)
		}
	}
}
