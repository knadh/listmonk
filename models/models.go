package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"html/template"
	"regexp"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/types"
	"github.com/lib/pq"

	null "gopkg.in/volatiletech/null.v6"
)

// Enum values for various statuses.
const (
	// Subscriber.
	SubscriberStatusEnabled     = "enabled"
	SubscriberStatusDisabled    = "disabled"
	SubscriberStatusBlackListed = "blacklisted"

	// Campaign.
	CampaignStatusDraft     = "draft"
	CampaignStatusScheduled = "scheduled"
	CampaignStatusRunning   = "running"
	CampaignStatusPaused    = "paused"
	CampaignStatusFinished  = "finished"
	CampaignStatusCancelled = "cancelled"

	// List.
	ListTypePrivate = "private"
	ListTypePublic  = "public"

	// User.
	UserTypeSuperadmin = "superadmin"
	UserTypeUser       = "user"
	UserStatusEnabled  = "enabled"
	UserStatusDisabled = "disabled"

	// BaseTpl is the name of the base template.
	BaseTpl = "base"

	// ContentTpl is the name of the compiled message.
	ContentTpl = "content"
)

// Regular expression for matching {{ Track "http://link.com" }} in the template
// and substituting it with {{ Track "http://link.com" .Campaign.UUID .Subscriber.UUID }}
// before compilation. This string gimmick is to make linking easier for users.
var (
	regexpLinkTag        = regexp.MustCompile("{{(\\s+)?TrackLink\\s+?(\"|`)(.+?)(\"|`)(\\s+)?}}")
	regexpLinkTagReplace = `{{ TrackLink "$3" .Campaign.UUID .Subscriber.UUID }}`

	regexpViewTag        = regexp.MustCompile(`{{(\s+)?TrackView(\s+)?}}`)
	regexpViewTagReplace = `{{ TrackView .Campaign.UUID .Subscriber.UUID }}`
)

// AdminNotifCallback is a callback function that's called
// when a campaign's status changes.
type AdminNotifCallback func(subject string, data map[string]interface{}) error

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

	UUID        string            `db:"uuid" json:"uuid"`
	Email       string            `db:"email" json:"email"`
	Name        string            `db:"name" json:"name"`
	Attribs     SubscriberAttribs `db:"attribs" json:"attribs"`
	Status      string            `db:"status" json:"status"`
	CampaignIDs pq.Int64Array     `db:"campaigns" json:"-"`
	Lists       []List            `json:"lists"`

	// Pseudofield for getting the total number of subscribers
	// in searches and queries.
	Total int `db:"total" json:"-"`
}

// SubscriberAttribs is the map of key:value attributes of a subscriber.
type SubscriberAttribs map[string]interface{}

// Subscribers represents a slice of Subscriber.
type Subscribers []Subscriber

// List represents a mailing list.
type List struct {
	Base

	UUID            string         `db:"uuid" json:"uuid"`
	Name            string         `db:"name" json:"name"`
	Type            string         `db:"type" json:"type"`
	Tags            pq.StringArray `db:"tags" json:"tags"`
	SubscriberCount int            `db:"subscriber_count" json:"subscriber_count"`
	SubscriberID    int            `db:"subscriber_id" json:"-"`

	// This is only relevant when querying the lists of a subscriber.
	SubscriptionStatus string `db:"subscription_status" json:"subscription_status,omitempty"`
}

// Campaign represents an e-mail campaign.
type Campaign struct {
	Base
	CampaignMeta

	UUID        string         `db:"uuid" json:"uuid"`
	Name        string         `db:"name" json:"name"`
	Subject     string         `db:"subject" json:"subject"`
	FromEmail   string         `db:"from_email" json:"from_email"`
	Body        string         `db:"body" json:"body,omitempty"`
	SendAt      null.Time      `db:"send_at" json:"send_at"`
	Status      string         `db:"status" json:"status"`
	ContentType string         `db:"content_type" json:"content_type"`
	Tags        pq.StringArray `db:"tags" json:"tags"`
	TemplateID  int            `db:"template_id" json:"template_id"`
	MessengerID string         `db:"messenger" json:"messenger"`
	Lists       types.JSONText `json:"lists"`

	View   int `db:"views" json:"views"`
	Clicks int `db:"clicks" json:"clicks"`
	// TemplateBody is joined in from templates by the next-campaigns query.
	TemplateBody string             `db:"template_body" json:"-"`
	Tpl          *template.Template `json:"-"`
}

// CampaignMeta contains fields tracking a campaign's progress.
type CampaignMeta struct {
	StartedAt null.Time `db:"started_at" json:"started_at"`
	ToSend    int       `db:"to_send" json:"to_send"`
	Sent      int       `db:"sent" json:"sent"`
}

// Campaigns represents a slice of Campaign.
type Campaigns []Campaign

// Media represents an uploaded media item.
type Media struct {
	ID        int       `db:"id" json:"id"`
	UUID      string    `db:"uuid" json:"uuid"`
	Filename  string    `db:"filename" json:"filename"`
	Width     int       `db:"width" json:"width"`
	Height    int       `db:"height" json:"height"`
	CreatedAt null.Time `db:"created_at" json:"created_at"`

	ThumbURI string `json:"thumb_uri"`
	URI      string `json:"uri"`
}

// Template represents a reusable e-mail template.
type Template struct {
	Base

	Name      string `db:"name" json:"name"`
	Body      string `db:"body" json:"body,omitempty"`
	IsDefault bool   `db:"is_default" json:"is_default"`
}

// LoadLists lazy loads the lists for all the subscribers
// in the Subscribers slice and attaches them to their []Lists property.
func (subs Subscribers) LoadLists(stmt *sqlx.Stmt) error {
	var (
		lists  []List
		subIDs = make([]int, len(subs))
	)
	for i := 0; i < len(subs); i++ {
		subIDs[i] = subs[i].ID
		subs[i].Lists = make([]List, 0)
	}

	err := stmt.Select(&lists, pq.Array(subIDs))
	if err != nil {
		return err
	}

	// Loop through each list and attach it to the subscribers by ID.
	for _, l := range lists {
		for i := 0; i < len(subs); i++ {
			if l.SubscriberID == subs[i].ID {
				subs[i].Lists = append(subs[i].Lists, l)
			}
		}
	}

	return nil
}

// Value returns the JSON marshalled SubscriberAttribs.
func (s SubscriberAttribs) Value() (driver.Value, error) {
	return json.Marshal(s)
}

// Scan unmarshals JSON into SubscriberAttribs.
func (s SubscriberAttribs) Scan(src interface{}) error {
	if data, ok := src.([]byte); ok {
		return json.Unmarshal(data, &s)
	}
	return fmt.Errorf("Could not not decode type %T -> %T", src, s)
}

// CompileTemplate compiles a campaign body template into its base
// template and sets the resultant template to Campaign.Tpl.
func (c *Campaign) CompileTemplate(f template.FuncMap) error {
	// Compile the base template.
	t := regexpLinkTag.ReplaceAllString(c.TemplateBody, regexpLinkTagReplace)
	t = regexpViewTag.ReplaceAllString(t, regexpViewTagReplace)
	baseTPL, err := template.New(BaseTpl).Funcs(f).Parse(t)
	if err != nil {
		return fmt.Errorf("error compiling base template: %v", err)
	}

	// Compile the campaign message.
	t = regexpLinkTag.ReplaceAllString(c.Body, regexpLinkTagReplace)
	t = regexpViewTag.ReplaceAllString(t, regexpViewTagReplace)
	msgTpl, err := template.New(ContentTpl).Funcs(f).Parse(t)
	if err != nil {
		return fmt.Errorf("error compiling message: %v", err)
	}

	out, err := baseTPL.AddParseTree(ContentTpl, msgTpl.Tree)
	if err != nil {
		return fmt.Errorf("error inserting child template: %v", err)
	}

	c.Tpl = out
	return nil
}

// FirstName splits the name by spaces and returns the first chunk
// of the name that's greater than 2 characters in length, assuming
// that it is the subscriber's first name.
func (s *Subscriber) FirstName() string {
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
func (s *Subscriber) LastName() string {
	chunks := strings.Split(s.Name, " ")
	for i := len(chunks) - 1; i >= 0; i-- {
		chunk := chunks[i]
		if len(chunk) > 2 {
			return chunk
		}
	}

	return s.Name
}
