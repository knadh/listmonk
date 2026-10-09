package mailbox

import (
	"log"
	"slices"

	_ "github.com/emersion/go-message/charset"
	"github.com/knadh/go-pop3"
	"github.com/knadh/listmonk/models"
)

// POP represents a POP mailbox.
type POP struct {
	opt    Opt
	client *pop3.Client
	lo     *log.Logger
}

// NewPOP returns a new instance of the POP mailbox client.
func NewPOP(opt Opt, lo *log.Logger) *POP {
	return &POP{
		opt: opt,
		client: pop3.New(pop3.Opt{
			Host:          opt.Host,
			Port:          opt.Port,
			TLSEnabled:    opt.TLSEnabled,
			TLSSkipVerify: opt.TLSSkipVerify,
		}),
		lo: lo,
	}
}

// Scan scans the mailbox and pushes the downloaded messages into the given
// channel if they are smaller than `maxMessageSize`. The messages that are
// successfully parsed are deleted from the server. If limit > 0, all messages
// on the server are downloaded.
func (p *POP) Scan(limit int, ch chan models.Bounce) error {
	var (
		oversizeMessages []int
		parsedMessageIds []int
	)

	c, err := p.client.NewConn()
	if err != nil {
		return err
	}
	defer c.Quit()

	// Authenticate.
	if p.opt.AuthProtocol != "none" {
		if err := c.Auth(p.opt.Username, p.opt.Password); err != nil {
			return err
		}
	}

	// Get the total number of messages on the server.
	count, _, err := c.Stat()
	if err != nil {
		return err
	}

	// No messages.
	if count == 0 {
		return nil
	}

	if limit > 0 && count > limit {
		count = limit
	}

	// Find messages larger than 1mb and add to ignore list
	messages, err := c.List(0)
	if err != nil {
		return err
	}

	for i, msg := range messages {
		if i > count {
			break
		}
		if msg.Size > maxMessageSize {
			oversizeMessages = append(oversizeMessages, msg.ID)
		}
	}

	// Download messages.
	for id := 1; id <= count; id++ {
		// Exclude oversized
		if slices.Contains(oversizeMessages, id) {
			p.lo.Printf("skipping bounce message %d as > %0.2f KB in size", id, float64(maxMessageSize)/1024)
			continue
		}

		// Retrieve the raw bytes of the message.
		b, err := c.RetrRaw(id)
		if err != nil {
			p.lo.Printf("error retrieving bounce message %d: %v", id, err)
			continue
		}

		var bounce models.Bounce
		bounce, err = parseDSN(b)
		if err != nil {
			p.lo.Printf("error parsing bounce message as DSN %d: %v", id, err)
			// Don't stop processing message if we have fallen back to regex
			// parser
			if err != errorNotMultipartReport {
				continue
			}
		}

		// Final checks that we can process the bounce
		if bounce.Email == "" && bounce.SubscriberUUID == "" {
			p.lo.Printf("unable to detrmine subscriber email or UUID in message: %d", id)
			continue
		}
		bounce.Source = p.opt.Host

		parsedMessageIds = append(parsedMessageIds, id)
		select {
		case ch <- bounce:
		default:
		}
	}

	// Delete successfully parsed messages only.
	for _, id := range parsedMessageIds {
		if err := c.Dele(id); err != nil {
			return err
		}
	}

	return nil
}
