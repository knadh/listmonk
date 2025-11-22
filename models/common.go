package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	null "gopkg.in/volatiletech/null.v6"
)

// Enum values for various statuses.
const (
	// Headers attached to e-mails for bounce tracking.
	EmailHeaderSubscriberUUID = "X-Listmonk-Subscriber"
	EmailHeaderCampaignUUID   = "X-Listmonk-Campaign"

	// Standard e-mail headers.
	EmailHeaderDate        = "Date"
	EmailHeaderFrom        = "From"
	EmailHeaderSubject     = "Subject"
	EmailHeaderMessageId   = "Message-Id"
	EmailHeaderDeliveredTo = "Delivered-To"
	EmailHeaderReceived    = "Received"

	// TwoFA types.
	TwofaTypeNone = "none"
	TwofaTypeTOTP = "totp"
)

// regTplFunc represents contains a regular expression for wrapping and
// substituting a Go template function from the user's shorthand to a full
// function call.
type regTplFunc struct {
	regExp  *regexp.Regexp
	replace string
}

var regTplFuncs = []regTplFunc{
	// Regular expression for matching {{ TrackLink "http://link.com" }} in the template
	// and substituting it with {{ TrackLink "http://link.com" . }} (the dot context)
	// before compilation. This is to make linking easier for users.
	{
		regExp:  regexp.MustCompile(`{{\s*TrackLink\s+"([^"]+)"\s*}}`),
		replace: `{{ TrackLink "$1" . }}`,
	},

	// Convert the shorthand https://google.com@TrackLink to {{ TrackLink ... }}.
	// This is for WYSIWYG editors that encode and break quotes {{ "" }} when inserted
	// inside <a href="{{ TrackLink "https://these-quotes-break" }}>.
	// The regex matches all characters that may occur in an URL
	// (see "2. Characters" in RFC3986: https://www.ietf.org/rfc/rfc3986.txt)
	{
		regExp:  regexp.MustCompile(`(https?://[\p{L}\p{N}_\-\.~!#$&'()*+,/:;=?@\[\]]*)@TrackLink`),
		replace: `{{ TrackLink "$1" . }}`,
	},

	{
		regExp:  regexp.MustCompile(`{{(\s+)?(TrackView|UnsubscribeURL|ManageURL|OptinURL|MessageURL)(\s+)?}}`),
		replace: `{{ $2 . }}`,
	},
}

// markdown is a global instance of Markdown parser and renderer.
var markdown = goldmark.New(
	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(),
	),
	goldmark.WithRendererOptions(
		html.WithXHTML(),
		html.WithUnsafe(),
	),
	goldmark.WithExtensions(
		extension.Table,
		extension.Strikethrough,
		extension.TaskList,
		extension.NewTypographer(
			extension.WithTypographicSubstitutions(extension.TypographicSubstitutions{
				extension.LeftDoubleQuote:  []byte(`"`),
				extension.RightDoubleQuote: []byte(`"`),
			}),
		),
	),
)

// Headers represents an array of string maps used to represent SMTP, HTTP headers etc.
// similar to url.Values{}
type Headers []map[string]string

// PageResults is a generic HTTP response container for paginated results of list of items.
type PageResults struct {
	Results any `json:"results"`

	Search  string `json:"search"`
	Query   string `json:"query"`
	Total   int    `json:"total"`
	PerPage int    `json:"per_page"`
	Page    int    `json:"page"`
}

// Base holds common fields shared across models.
type Base struct {
	ID        int       `db:"id" json:"id"`
	CreatedAt null.Time `db:"created_at" json:"created_at"`
	UpdatedAt null.Time `db:"updated_at" json:"updated_at"`
}

// JSON is the wrapper for reading and writing arbitrary JSONB fields from the DB.
type JSON map[string]any

// StringIntMap is used to define DB Scan()s.
type StringIntMap map[string]int

// Value returns the JSON marshalled SubscriberAttribs.
func (s JSON) Value() (driver.Value, error) {
	return json.Marshal(s)
}

// Scan unmarshals JSONB from the DB.
func (s JSON) Scan(src any) error {
	if src == nil {
		s = make(JSON)
		return nil
	}

	if data, ok := src.([]byte); ok {
		return json.Unmarshal(data, &s)
	}
	return fmt.Errorf("could not not decode type %T -> %T", src, s)
}

// Scan unmarshals JSONB from the DB.
func (s StringIntMap) Scan(src any) error {
	if src == nil {
		s = make(StringIntMap)
		return nil
	}

	if data, ok := src.([]byte); ok {
		return json.Unmarshal(data, &s)
	}
	return fmt.Errorf("could not not decode type %T -> %T", src, s)
}

// Scan implements the sql.Scanner interface.
func (h *Headers) Scan(src any) error {
	var b []byte
	switch src := src.(type) {
	case []byte:
		b = src
	case string:
		b = []byte(src)
	case nil:
		return nil
	}

	if err := json.Unmarshal(b, h); err != nil {
		return err
	}

	return nil
}

// Value implements the driver.Valuer interface.
func (h Headers) Value() (driver.Value, error) {
	if h == nil {
		return nil, nil
	}

	if n := len(h); n > 0 {
		b, err := json.Marshal(h)
		if err != nil {
			return nil, err
		}

		return b, nil
	}

	return "[]", nil
}
