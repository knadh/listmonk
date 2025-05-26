package bounce

import (
	"errors"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/listmonk/internal/bounce/mailbox"
	"github.com/knadh/listmonk/internal/bounce/webhooks"
	"github.com/knadh/listmonk/models"
)

// Mailbox represents a POP/IMAP mailbox client that can scan messages and pass
// them to a given channel.
type Mailbox interface {
	Scan(limit int, ch chan models.Bounce) error
}

// Opt represents bounce processing options.
type Opt struct {
	MailboxEnabled  bool        `json:"mailbox_enabled"`
	MailboxType     string      `json:"mailbox_type"`
	Mailbox         mailbox.Opt `json:"mailbox"`
	WebhooksEnabled bool        `json:"webhooks_enabled"`
	SESEnabled      bool        `json:"ses_enabled"`
	SendgridEnabled bool        `json:"sendgrid_enabled"`
	SendgridKey     string      `json:"sendgrid_key"`
	Postmark        struct {
		Enabled  bool
		Username string
		Password string
	}
	ForwardEmail struct {
		Enabled bool
		Key     string
	}
	MailgunEnabled    bool
	MailgunWebhookKey string

	RecordBounceCB func(models.Bounce) error
}

// Manager handles e-mail bounces.
type Manager struct {
	queue        chan models.Bounce
	mailbox      Mailbox
	SES          *webhooks.SES
	Sendgrid     *webhooks.Sendgrid
	Postmark     *webhooks.Postmark
	Forwardemail *webhooks.Forwardemail
	Mailgun      *webhooks.Mailgun
	queries      *Queries
	opt          Opt
	log          *log.Logger
}

// Queries contains the queries.
type Queries struct {
	DB          *sqlx.DB
	RecordQuery *sqlx.Stmt
}

// New returns a new instance of the bounce manager.
func New(opt Opt, q *Queries, lo *log.Logger) (*Manager, error) {
	m := &Manager{
		opt:     opt,
		queries: q,
		queue:   make(chan models.Bounce, 1000),
		log:     lo,
	}

	// Is there a mailbox?
	if opt.MailboxEnabled {
		switch opt.MailboxType {
		case "pop":
			m.mailbox = mailbox.NewPOP(opt.Mailbox)
		default:
			return nil, errors.New("unknown bounce mailbox type")
		}
	}

	if opt.WebhooksEnabled {
		if opt.SESEnabled {
			m.SES = webhooks.NewSES()
		}

		if opt.SendgridEnabled {
			sg, err := webhooks.NewSendgrid(opt.SendgridKey)
			if err != nil {
				lo.Printf("error initializing sendgrid webhooks: %v", err)
			} else {
				m.Sendgrid = sg
			}
		}

		if opt.Postmark.Enabled {
			m.Postmark = webhooks.NewPostmark(opt.Postmark.Username, opt.Postmark.Password)
		}

		if opt.ForwardEmail.Enabled {
			fe := webhooks.NewForwardemail([]byte(opt.ForwardEmail.Key))
			m.Forwardemail = fe
		}

		// Mailgun initialization is now handled externally after New() is called,
		// typically in cmd/main.go or equivalent setup code,
		// based on the bounce.Opt fields (MailgunEnabled, MailgunWebhookKey).
	}

	return m, nil
}

// Run is a blocking function that listens for bounce events from webhooks and or mailboxes
// and executes them on the DB.
func (m *Manager) Run() {
	if m.opt.MailboxEnabled {
		go m.runMailboxScanner()
	}

	for b := range m.queue {
		if b.CreatedAt.IsZero() {
			b.CreatedAt = time.Now()
		}

		if err := m.opt.RecordBounceCB(b); err != nil {
			continue
		}
	}
}

// runMailboxScanner runs a blocking loop that scans the mailbox at given intervals.
func (m *Manager) runMailboxScanner() {
	for {
		if err := m.mailbox.Scan(1000, m.queue); err != nil {
			m.log.Printf("error scanning bounce mailbox: %v", err)
		}

		time.Sleep(m.opt.Mailbox.ScanInterval)
	}
}

// Record records a new bounce event given the subscriber's email or UUID.
func (m *Manager) Record(b models.Bounce) error {
	m.queue <- b
	return nil
}
