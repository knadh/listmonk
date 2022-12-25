package models

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"regexp"
	"strings"
	txttpl "text/template"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/types"
	"github.com/lib/pq"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	null "gopkg.in/volatiletech/null.v6"
)

// Enum values for various statuses.
const (
	// Subscriber.
	SubscriberStatusEnabled     = "enabled"
	SubscriberStatusDisabled    = "disabled"
	SubscriberStatusBlockListed = "blocklisted"

	// Subscription.
	SubscriptionStatusUnconfirmed  = "unconfirmed"
	SubscriptionStatusConfirmed    = "confirmed"
	SubscriptionStatusUnsubscribed = "unsubscribed"

	// Campaign.
	CampaignStatusDraft         = "draft"
	CampaignStatusScheduled     = "scheduled"
	CampaignStatusRunning       = "running"
	CampaignStatusPaused        = "paused"
	CampaignStatusFinished      = "finished"
	CampaignStatusCancelled     = "cancelled"
	CampaignTypeRegular         = "regular"
	CampaignTypeOptin           = "optin"
	CampaignContentTypeRichtext = "richtext"
	CampaignContentTypeHTML     = "html"
	CampaignContentTypeMarkdown = "markdown"
	CampaignContentTypePlain    = "plain"

	// List.
	ListTypePrivate = "private"
	ListTypePublic  = "public"
	ListOptinSingle = "single"
	ListOptinDouble = "double"

	// User.
	UserTypeSuperadmin = "superadmin"
	UserTypeUser       = "user"
	UserStatusEnabled  = "enabled"
	UserStatusDisabled = "disabled"

	// BaseTpl is the name of the base template.
	BaseTpl = "base"

	// ContentTpl is the name of the compiled message.
	ContentTpl = "content"

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

	BounceTypeHard = "hard"
	BounceTypeSoft = "soft"

	// Templates.
	TemplateTypeCampaign = "campaign"
	TemplateTypeTx       = "tx"
)

// Headers represents an array of string maps used to represent SMTP, HTTP headers etc.
// similar to url.Values{}
type Headers []map[string]string

// regTplFunc represents contains a regular expression for wrapping and
// substituting a Go template function from the user's shorthand to a full
// function call.
type regTplFunc struct {
	regExp  *regexp.Regexp
	replace string
}

var regTplFuncs = []regTplFunc{
	// Regular expression for matching {{ TrackLink "http://link.com" }} in the template
	// and substituting it with {{ Track "http://link.com" . }} (the dot context)
	// before compilation. This is to make linking easier for users.
	{
		regExp:  regexp.MustCompile("{{(\\s+)?TrackLink(\\s+)?(.+?)(\\s+)?}}"),
		replace: `{{ TrackLink $3 . }}`,
	},

	// Convert the shorthand https://google.com@TrackLink to {{ TrackLink ... }}.
	// This is for WYSIWYG editors that encode and break quotes {{ "" }} when inserted
	// inside <a href="{{ TrackLink "https://these-quotes-break" }}>.
	{
		regExp:  regexp.MustCompile(`(https?://.+?)@TrackLink`),
		replace: `{{ TrackLink "$1" . }}`,
	},

	{
		regExp:  regexp.MustCompile(`{{(\s+)?(TrackView|UnsubscribeURL|ManageURL|OptinURL|MessageURL)(\s+)?}}`),
		replace: `{{ $2 . }}`,
	},
}

// AdminNotifCallback is a callback function that's called
// when a campaign's status changes.
type AdminNotifCallback func(subject string, data interface{}) error

// PageResults is a generic HTTP response container for paginated results of list of items.
type PageResults struct {
	Results interface{} `json:"results"`

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

// User represents an admin user.
type User struct {
	Base

	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"-"`
	Type     string `json:"type"`
	Status   string `json:"status"`
}

// Subscriber represents an e-mail subscriber.
type Subscriber struct {
	Base

	UUID    string         `db:"uuid" json:"uuid"`
	Email   string         `db:"email" json:"email" form:"email"`
	Name    string         `db:"name" json:"name" form:"name"`
	Attribs JSON           `db:"attribs" json:"attribs"`
	Status  string         `db:"status" json:"status"`
	Lists   types.JSONText `db:"lists" json:"lists"`
}
type subLists struct {
	SubscriberID int            `db:"subscriber_id"`
	Lists        types.JSONText `db:"lists"`
}

// Subscription represents a list attached to a subscriber.
type Subscription struct {
	List
	SubscriptionStatus    null.String `db:"subscription_status" json:"subscription_status"`
	SubscriptionCreatedAt null.String `db:"subscription_created_at" json:"subscription_created_at"`
}

// SubscriberExportProfile represents a subscriber's collated data in JSON for export.
type SubscriberExportProfile struct {
	Email         string          `db:"email" json:"-"`
	Profile       json.RawMessage `db:"profile" json:"profile,omitempty"`
	Subscriptions json.RawMessage `db:"subscriptions" json:"subscriptions,omitempty"`
	CampaignViews json.RawMessage `db:"campaign_views" json:"campaign_views,omitempty"`
	LinkClicks    json.RawMessage `db:"link_clicks" json:"link_clicks,omitempty"`
}

// JSON is is the wrapper for reading and writing arbitrary JSONB fields from the DB.
type JSON map[string]interface{}

// StringIntMap is used to define DB Scan()s.
type StringIntMap map[string]int

// Subscribers represents a slice of Subscriber.
type Subscribers []Subscriber

// SubscriberExport represents a subscriber record that is exported to raw data.
type SubscriberExport struct {
	Base

	UUID    string `db:"uuid" json:"uuid"`
	Email   string `db:"email" json:"email"`
	Name    string `db:"name" json:"name"`
	Attribs string `db:"attribs" json:"attribs"`
	Status  string `db:"status" json:"status"`
}

// List represents a mailing list.
type List struct {
	Base

	UUID             string         `db:"uuid" json:"uuid"`
	Name             string         `db:"name" json:"name"`
	Type             string         `db:"type" json:"type"`
	Optin            string         `db:"optin" json:"optin"`
	Tags             pq.StringArray `db:"tags" json:"tags"`
	Description      string         `db:"description" json:"description"`
	SubscriberCount  int            `db:"-" json:"subscriber_count"`
	SubscriberCounts StringIntMap   `db:"subscriber_statuses" json:"subscriber_statuses"`
	SubscriberID     int            `db:"subscriber_id" json:"-"`

	// This is only relevant when querying the lists of a subscriber.
	SubscriptionStatus    string    `db:"subscription_status" json:"subscription_status,omitempty"`
	SubscriptionCreatedAt null.Time `db:"subscription_created_at" json:"subscription_created_at,omitempty"`
	SubscriptionUpdatedAt null.Time `db:"subscription_updated_at" json:"subscription_updated_at,omitempty"`

	// Pseudofield for getting the total number of subscribers
	// in searches and queries.
	Total int `db:"total" json:"-"`
}

// Campaign represents an e-mail campaign.
type Campaign struct {
	Base
	CampaignMeta

	UUID              string          `db:"uuid" json:"uuid"`
	Type              string          `db:"type" json:"type"`
	Name              string          `db:"name" json:"name"`
	Subject           string          `db:"subject" json:"subject"`
	FromEmail         string          `db:"from_email" json:"from_email"`
	Body              string          `db:"body" json:"body"`
	AltBody           null.String     `db:"altbody" json:"altbody"`
	SendAt            null.Time       `db:"send_at" json:"send_at"`
	Status            string          `db:"status" json:"status"`
	ContentType       string          `db:"content_type" json:"content_type"`
	Tags              pq.StringArray  `db:"tags" json:"tags"`
	Headers           Headers         `db:"headers" json:"headers"`
	TemplateID        int             `db:"template_id" json:"template_id"`
	Messenger         string          `db:"messenger" json:"messenger"`
	Archive           bool            `db:"archive" json:"archive"`
	ArchiveTemplateID int             `db:"archive_template_id" json:"archive_template_id"`
	ArchiveMeta       json.RawMessage `db:"archive_meta" json:"archive_meta"`

	// TemplateBody is joined in from templates by the next-campaigns query.
	TemplateBody        string             `db:"template_body" json:"-"`
	ArchiveTemplateBody string             `db:"archive_template_body" json:"-"`
	Tpl                 *template.Template `json:"-"`
	SubjectTpl          *txttpl.Template   `json:"-"`
	AltBodyTpl          *template.Template `json:"-"`

	// Pseudofield for getting the total number of subscribers
	// in searches and queries.
	Total int `db:"total" json:"-"`
}

// CampaignMeta contains fields tracking a campaign's progress.
type CampaignMeta struct {
	CampaignID int `db:"campaign_id" json:"-"`
	Views      int `db:"views" json:"views"`
	Clicks     int `db:"clicks" json:"clicks"`
	Bounces    int `db:"bounces" json:"bounces"`

	// This is a list of {list_id, name} pairs unlike Subscriber.Lists[]
	// because lists can be deleted after a campaign is finished, resulting
	// in null lists data to be returned. For that reason, campaign_lists maintains
	// campaign-list associations with a historical record of id + name that persist
	// even after a list is deleted.
	Lists types.JSONText `db:"lists" json:"lists"`

	StartedAt null.Time `db:"started_at" json:"started_at"`
	ToSend    int       `db:"to_send" json:"to_send"`
	Sent      int       `db:"sent" json:"sent"`
}

type CampaignStats struct {
	ID        int       `db:"id" json:"id"`
	Status    string    `db:"status" json:"status"`
	ToSend    int       `db:"to_send" json:"to_send"`
	Sent      int       `db:"sent" json:"sent"`
	Started   null.Time `db:"started_at" json:"started_at"`
	UpdatedAt null.Time `db:"updated_at" json:"updated_at"`
	Rate      int       `json:"rate"`
	NetRate   int       `json:"net_rate"`
}

type CampaignAnalyticsCount struct {
	CampaignID int       `db:"campaign_id" json:"campaign_id"`
	Count      int       `db:"count" json:"count"`
	Timestamp  time.Time `db:"timestamp" json:"timestamp"`
}

type CampaignAnalyticsLink struct {
	URL   string `db:"url" json:"url"`
	Count int    `db:"count" json:"count"`
}

// Campaigns represents a slice of Campaigns.
type Campaigns []Campaign

// Template represents a reusable e-mail template.
type Template struct {
	Base

	Name string `db:"name" json:"name"`
	// Subject is only for type=tx.
	Subject   string `db:"subject" json:"subject"`
	Type      string `db:"type" json:"type"`
	Body      string `db:"body" json:"body,omitempty"`
	IsDefault bool   `db:"is_default" json:"is_default"`

	// Only relevant to tx (transactional) templates.
	SubjectTpl *txttpl.Template   `json:"-"`
	Tpl        *template.Template `json:"-"`
}

// Bounce represents a single bounce event.
type Bounce struct {
	ID        int             `db:"id" json:"id"`
	Type      string          `db:"type" json:"type"`
	Source    string          `db:"source" json:"source"`
	Meta      json.RawMessage `db:"meta" json:"meta"`
	CreatedAt time.Time       `db:"created_at" json:"created_at"`

	// One of these should be provided.
	Email          string `db:"email" json:"email,omitempty"`
	SubscriberUUID string `db:"subscriber_uuid" json:"subscriber_uuid,omitempty"`
	SubscriberID   int    `db:"subscriber_id" json:"subscriber_id,omitempty"`

	CampaignUUID string           `db:"campaign_uuid" json:"campaign_uuid,omitempty"`
	Campaign     *json.RawMessage `db:"campaign" json:"campaign"`

	// Pseudofield for getting the total number of bounces
	// in searches and queries.
	Total int `db:"total" json:"-"`
}

// TxMessage represents an e-mail campaign.
type TxMessage struct {
	SubscriberEmails []string `json:"subscriber_emails"`
	SubscriberIDs    []int    `json:"subscriber_ids"`

	// Deprecated.
	SubscriberEmail string `json:"subscriber_email"`
	SubscriberID    int    `json:"subscriber_id"`

	TemplateID  int                    `json:"template_id"`
	Data        map[string]interface{} `json:"data"`
	FromEmail   string                 `json:"from_email"`
	Headers     Headers                `json:"headers"`
	ContentType string                 `json:"content_type"`
	Messenger   string                 `json:"messenger"`

	Subject    string             `json:"-"`
	Body       []byte             `json:"-"`
	Tpl        *template.Template `json:"-"`
	SubjectTpl *txttpl.Template   `json:"-"`
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
	),
)

// GetIDs returns the list of subscriber IDs.
func (subs Subscribers) GetIDs() []int {
	IDs := make([]int, len(subs))
	for i, c := range subs {
		IDs[i] = c.ID
	}

	return IDs
}

// LoadLists lazy loads the lists for all the subscribers
// in the Subscribers slice and attaches them to their []Lists property.
func (subs Subscribers) LoadLists(stmt *sqlx.Stmt) error {
	var sl []subLists
	err := stmt.Select(&sl, pq.Array(subs.GetIDs()))
	if err != nil {
		return err
	}

	if len(subs) != len(sl) {
		return errors.New("campaign stats count does not match")
	}

	for i, s := range sl {
		if s.SubscriberID == subs[i].ID {
			subs[i].Lists = s.Lists
		}
	}

	return nil
}

// Value returns the JSON marshalled SubscriberAttribs.
func (s JSON) Value() (driver.Value, error) {
	return json.Marshal(s)
}

// Scan unmarshals JSONB from the DB.
func (s JSON) Scan(src interface{}) error {
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
func (s StringIntMap) Scan(src interface{}) error {
	if src == nil {
		s = make(StringIntMap)
		return nil
	}

	if data, ok := src.([]byte); ok {
		return json.Unmarshal(data, &s)
	}
	return fmt.Errorf("could not not decode type %T -> %T", src, s)
}

// GetIDs returns the list of campaign IDs.
func (camps Campaigns) GetIDs() []int {
	IDs := make([]int, len(camps))
	for i, c := range camps {
		IDs[i] = c.ID
	}

	return IDs
}

// LoadStats lazy loads campaign stats onto a list of campaigns.
func (camps Campaigns) LoadStats(stmt *sqlx.Stmt) error {
	var meta []CampaignMeta
	if err := stmt.Select(&meta, pq.Array(camps.GetIDs())); err != nil {
		return err
	}

	if len(camps) != len(meta) {
		return errors.New("campaign stats count does not match")
	}

	for i, c := range meta {
		if c.CampaignID == camps[i].ID {
			camps[i].Lists = c.Lists
			camps[i].Views = c.Views
			camps[i].Clicks = c.Clicks
			camps[i].Bounces = c.Bounces
		}
	}

	return nil
}

// CompileTemplate compiles a campaign body template into its base
// template and sets the resultant template to Campaign.Tpl.
func (c *Campaign) CompileTemplate(f template.FuncMap) error {
	// If the subject line has a template string, compile it.
	if strings.Contains(c.Subject, "{{") {
		subj := c.Subject
		for _, r := range regTplFuncs {
			subj = r.regExp.ReplaceAllString(subj, r.replace)
		}

		var txtFuncs map[string]interface{} = f
		subjTpl, err := txttpl.New(ContentTpl).Funcs(txtFuncs).Parse(subj)
		if err != nil {
			return fmt.Errorf("error compiling subject: %v", err)
		}
		c.SubjectTpl = subjTpl
	}

	// Compile the base template.
	body := c.TemplateBody
	for _, r := range regTplFuncs {
		body = r.regExp.ReplaceAllString(body, r.replace)
	}
	baseTPL, err := template.New(BaseTpl).Funcs(f).Parse(body)
	if err != nil {
		return fmt.Errorf("error compiling base template: %v", err)
	}

	// If the format is markdown, convert Markdown to HTML.
	if c.ContentType == CampaignContentTypeMarkdown {
		var b bytes.Buffer
		if err := markdown.Convert([]byte(c.Body), &b); err != nil {
			return err
		}
		body = b.String()
	} else {
		body = c.Body
	}

	// Compile the campaign message.
	for _, r := range regTplFuncs {
		body = r.regExp.ReplaceAllString(body, r.replace)
	}

	msgTpl, err := template.New(ContentTpl).Funcs(f).Parse(body)
	if err != nil {
		return fmt.Errorf("error compiling message: %v", err)
	}

	out, err := baseTPL.AddParseTree(ContentTpl, msgTpl.Tree)
	if err != nil {
		return fmt.Errorf("error inserting child template: %v", err)
	}
	c.Tpl = out

	if strings.Contains(c.AltBody.String, "{{") {
		b := c.AltBody.String
		for _, r := range regTplFuncs {
			b = r.regExp.ReplaceAllString(b, r.replace)
		}
		bTpl, err := template.New(ContentTpl).Funcs(f).Parse(b)
		if err != nil {
			return fmt.Errorf("error compiling alt plaintext message: %v", err)
		}
		c.AltBodyTpl = bTpl
	}

	return nil
}

// ConvertContent converts a campaign's body from one format to another,
// for example, Markdown to HTML.
func (c *Campaign) ConvertContent(from, to string) (string, error) {
	body := c.Body
	for _, r := range regTplFuncs {
		body = r.regExp.ReplaceAllString(body, r.replace)
	}

	// If the format is markdown, convert Markdown to HTML.
	var out string
	if from == CampaignContentTypeMarkdown &&
		(to == CampaignContentTypeHTML || to == CampaignContentTypeRichtext) {
		var b bytes.Buffer
		if err := markdown.Convert([]byte(c.Body), &b); err != nil {
			return out, err
		}
		out = b.String()
	} else {
		return out, errors.New("unknown formats to convert")
	}

	return out, nil
}

// Compile compiles a template body and subject (only for tx templates) and
// caches the templat references to be executed later.
func (t *Template) Compile(f template.FuncMap) error {
	tpl, err := template.New(BaseTpl).Funcs(f).Parse(t.Body)
	if err != nil {
		return fmt.Errorf("error compiling transactional template: %v", err)
	}
	t.Tpl = tpl

	// If the subject line has a template string, compile it.
	if strings.Contains(t.Subject, "{{") {
		subj := t.Subject

		subjTpl, err := txttpl.New(BaseTpl).Funcs(txttpl.FuncMap(f)).Parse(subj)
		if err != nil {
			return fmt.Errorf("error compiling subject: %v", err)
		}
		t.SubjectTpl = subjTpl
	}

	return nil
}

func (m *TxMessage) Render(sub Subscriber, tpl *Template) error {
	data := struct {
		Subscriber Subscriber
		Tx         *TxMessage
	}{sub, m}

	// Render the body.
	b := bytes.Buffer{}
	if err := tpl.Tpl.ExecuteTemplate(&b, BaseTpl, data); err != nil {
		return err
	}
	m.Body = make([]byte, b.Len())
	copy(m.Body, b.Bytes())
	b.Reset()

	// If the subject is also a template, render that.
	if tpl.SubjectTpl != nil {
		if err := tpl.SubjectTpl.ExecuteTemplate(&b, BaseTpl, data); err != nil {
			return err
		}
		m.Subject = b.String()
		b.Reset()
	} else {
		m.Subject = tpl.Subject
	}

	return nil
}

// FirstName splits the name by spaces and returns the first chunk
// of the name that's greater than 2 characters in length, assuming
// that it is the subscriber's first name.
func (s Subscriber) FirstName() string {
	for _, s := range strings.Split(s.Name, " ") {
		if len(s) > 2 {
			return s
		}
	}

	return s.Name
}

// LastName splits the name by spaces and returns the last chunk
// of the name that's greater than 2 characters in length, assuming
// that it is the subscriber's last name.
func (s Subscriber) LastName() string {
	chunks := strings.Split(s.Name, " ")
	for i := len(chunks) - 1; i >= 0; i-- {
		chunk := chunks[i]
		if len(chunk) > 2 {
			return chunk
		}
	}

	return s.Name
}

// Scan implements the sql.Scanner interface.
func (h *Headers) Scan(src interface{}) error {
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
