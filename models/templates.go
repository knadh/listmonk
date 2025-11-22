package models

import (
	"fmt"
	"html/template"
	"strings"
	txttpl "text/template"
	"time"

	null "gopkg.in/volatiletech/null.v6"
)

const (
	BaseTpl                    = "base"
	ContentTpl                 = "content"
	TemplateTypeCampaign       = "campaign"
	TemplateTypeCampaignVisual = "campaign_visual"
	TemplateTypeTx             = "tx"
)

// Template represents a reusable e-mail template.
type Template struct {
	Base

	Name string `db:"name" json:"name"`
	// Subject is only for type=tx.
	Subject    string      `db:"subject" json:"subject"`
	Type       string      `db:"type" json:"type"`
	Body       string      `db:"body" json:"body,omitempty"`
	BodySource null.String `db:"body_source" json:"body_source,omitempty"`
	IsDefault  bool        `db:"is_default" json:"is_default"`

	// Only relevant to tx (transactional) templates.
	SubjectTpl *txttpl.Template   `json:"-"`
	Tpl        *template.Template `json:"-"`
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
