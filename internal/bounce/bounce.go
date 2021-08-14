package bounce

import (
	"errors"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/listmonk/internal/bounce/mailbox"
	"github.com/knadh/listmonk/internal/bounce/webhooks"
	"github.com/knadh/listmonk/models"
	"github.com/lib/pq"
)

const (
	// subID is the identifying subscriber ID header to look for in
	// bounced e-mails.
	subID  = "X-Listmonk-Subscriber"
	campID = "X-Listmonk-Campaign"
)

// Mailbox represents a POP/IMAP mailbox client that can scan messages and pass
// them to a given channel.
type Mailbox interface {
	Scan(limit int, ch chan models.Bounce) error
}

// Opt represents bounce processing options.
type Opt struct {
	BounceCount  int    `json:"count"`
	BounceAction string `json:"action"`

	MailboxEnabled  bool        `json:"mailbox_enabled"`
	MailboxType     string      `json:"mailbox_type"`
	Mailbox         mailbox.Opt `json:"mailbox"`
	WebhooksEnabled bool        `json:"webhooks_enabled"`
	SESEnabled      bool        `json:"ses_enabled"`
	SendgridEnabled bool        `json:"sendgrid_enabled"`
	SendgridKey     string      `json:"sendgrid_key"`
}

// Manager handles e-mail bounces.
type Manager struct {
	queue    chan models.Bounce
	mailbox  Mailbox
	SES      *webhooks.SES
	Sendgrid *webhooks.Sendgrid
	queries  *Queries
	opt      Opt
	log      *log.Logger
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
		case "imap":
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
	}

	return m, nil
}

// Run is a blocking function that listens for bounce events from webhooks and or mailboxes
// and executes them on the DB.
func (m *Manager) Run() {
	if m.opt.MailboxEnabled {
		go m.runMailboxScanner()
	}

	for {
		select {
		case b, ok := <-m.queue:
			if !ok {
				return
			}

			_, err := m.queries.RecordQuery.Exec(b.SubscriberUUID,
				b.Email,
				b.CampaignUUID,
				b.Type,
				b.Source,
				b.Meta,
				b.CreatedAt,
				m.opt.BounceCount,
				m.opt.BounceAction)
			if err != nil {
				// Ignore the error if it complained of no subscriber.
				if pqErr, ok := err.(*pq.Error); ok && pqErr.Column == "subscriber_id" {
					m.log.Printf("bounced subscriber (%s / %s) not found", b.SubscriberUUID, b.Email)
					continue
				}
				m.log.Printf("error recording bounce: %v", err)
			}
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
	select {
	case m.queue <- b:
	}
	return nil
}
