package mailbox

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/knadh/listmonk/models"
)

// IMAP represents a IMAP mailbox.
type IMAP struct {
	opt Opt
	lo  *log.Logger
}

// NewPOP returns a new instance of the IMAP mailbox client.
func NewIMAP(opt Opt, lo *log.Logger) *IMAP {
	return &IMAP{
		opt: opt,
		lo:  lo,
	}
}

// Scan scans the IMAP mailbox specified in opt.Folder (or defaults to INBOX)
// and pushes the downloaded messages into the given channel if they are smaller
// than `maxMessageSize`. The messages that are successfully parsed are deleted
// from the server (or moved to the folder specified in opt.TrashFolder). If
// limit > 0, all messages on the server are downloaded.
func (p *IMAP) Scan(limit int, ch chan models.Bounce) error {
	var (
		imapErr        error
		dialer         = &net.Dialer{Timeout: time.Second * 30}
		c              *client.Client
		validMessages  *imap.SeqSet = new(imap.SeqSet)
		parsedMessages *imap.SeqSet = new(imap.SeqSet)
	)

	if p.opt.TLSEnabled {
		c, imapErr = client.DialWithDialerTLS(dialer, fmt.Sprintf("%s:%d", p.opt.Host, p.opt.Port),
			&tls.Config{InsecureSkipVerify: p.opt.TLSSkipVerify})
	} else {
		c, imapErr = client.DialWithDialer(dialer, fmt.Sprintf("%s:%d", p.opt.Host, p.opt.Port))
	}
	if imapErr != nil {
		return fmt.Errorf("dialing IMAP server: %v", imapErr)
	}

	defer c.Logout()
	c.Timeout = time.Second * 30
	if err := c.Login(p.opt.Username, p.opt.Password); err != nil {
		return fmt.Errorf("IMAP login error: %v", err)
	}

	// Default to INBOX if Folder not set
	folder := "INBOX"
	if p.opt.Folder != "" {
		folder = p.opt.Folder
	}

	// Get the total number of messages on the server.
	mbox, err := c.Select(folder, false)
	if err != nil {
		return fmt.Errorf("selecting IMAP mailbox %s: %v", folder, err)
	}
	count := int(mbox.Messages)

	// No messages.
	if count == 0 {
		return nil
	}

	if limit > 0 && count > limit {
		count = limit
	}

	// Find messages larger than 1mb and add to ignore list
	seqset := new(imap.SeqSet)
	seqset.AddRange(uint32(1), uint32(limit))
	iampMessages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope, imap.FetchRFC822Size}, iampMessages)
	}()

	for msg := range iampMessages {
		if msg.Size > uint32(maxMessageSize) {
			p.lo.Printf("skipping bounce message %d as > %0.2f KB in size", msg.SeqNum, float64(maxMessageSize)/1024)
		} else {
			validMessages.AddNum(msg.SeqNum)
		}
	}

	if err := <-done; err != nil {
		return fmt.Errorf("listing IMAP mailbox %s: %v", folder, err)
	}

	// Download messages
	if validMessages.Empty() {
		return nil
	}

	section := &imap.BodySectionName{}
	messages := make(chan *imap.Message, 10)
	done = make(chan error, 1)
	go func() {
		done <- c.Fetch(validMessages, []imap.FetchItem{section.FetchItem()}, messages)
	}()

	for msg := range messages {
		r := msg.GetBody(section)
		if r == nil {
			p.lo.Printf("error retrieving bounce message %d: no message body returned", msg.SeqNum)
			continue
		}

		var bounce models.Bounce
		bounce, err = parseDSN(r)
		if err != nil {
			p.lo.Printf("error parsing bounce message as DSN %d: %v", msg.SeqNum, err)
			// Don't stop processing message if we have fallen back to regex
			// parser
			if err != errorNotMultipartReport {
				continue
			}
		}

		// Final checks that we can process the bounce
		if bounce.Email == "" && bounce.SubscriberUUID == "" {
			p.lo.Printf("unable to detrmine subscriber email or UUID in message: %d", msg.SeqNum)
			continue
		}
		bounce.Source = p.opt.Host

		// Add message to deletion set
		parsedMessages.AddNum(msg.SeqNum)

		select {
		case ch <- bounce:
		default:
		}
	}

	if err := <-done; err != nil {
		return fmt.Errorf("retrieving bounce messages via IMAP: %v", err)
	}

	// If TrashFolded is set move parsed messages to it otherwise delete them
	if !parsedMessages.Empty() {
		if p.opt.TrashFolder == "" {
			// Mark parsed messages for deletion
			if err := c.Store(parsedMessages,
				imap.FormatFlagsOp(imap.AddFlags, true),
				[]interface{}{imap.DeletedFlag},
				nil); err != nil {
				p.lo.Printf("error marking IMAP messages with \\deleted flag: %v", err)
			}
			// Expunge them
			if err := c.Expunge(nil); err != nil {
				return fmt.Errorf("expunging IMAP messages: %v", err)
			}

		} else {
			if err := c.Move(parsedMessages, p.opt.TrashFolder); err != nil {
				return fmt.Errorf("moving messages to %s IMAP error: %v", p.opt.TrashFolder, err)
			}
		}
	}

	return nil
}
