package mailbox

import (
	"crypto/tls"
	"fmt"
	"io"
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

// NewIMAP returns a new instance of the IMAP mailbox client.
func NewIMAP(opt Opt, lo *log.Logger) *IMAP {
	return &IMAP{
		opt: opt,
		lo:  lo,
	}
}

// Scan scans the IMAP mailbox specified in opt.Folder (or defaults to INBOX)
// and pushes the downloaded messages into the given channel. The scanned
// messages are deleted from the server (or moved to the folder specified in
// opt.TrashFolder). A positive limit caps the number of messages scanned.
func (p *IMAP) Scan(limit int, ch chan models.Bounce) error {
	var (
		imapErr        error
		dialer         = &net.Dialer{Timeout: time.Second * 30}
		c              *client.Client
		processedMessages *imap.SeqSet = new(imap.SeqSet)
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

	section := &imap.BodySectionName{}
	fetchMessages := new(imap.SeqSet)
	fetchMessages.AddRange(1, uint32(count))
	messages := make(chan *imap.Message, 1)
	done := make(chan error, 1)
	go func() {
		done <- c.Fetch(fetchMessages, []imap.FetchItem{section.FetchItem()}, messages)
	}()

	for msg := range messages {
		r := msg.GetBody(section)
		if r == nil {
			p.lo.Printf("error retrieving bounce message %d: no message body returned", msg.SeqNum)
			continue
		}

		b, err := io.ReadAll(r)
		if err != nil {
			p.lo.Printf("error retrieving bounce message %d: %v", msg.SeqNum, err)
			continue
		}

		bounces, err := parseBounce(b)
		if err != nil {
			p.lo.Printf("error parsing bounce message %d: %v", msg.SeqNum, err)
			continue
		}
		for _, bounce := range bounces {
			bounce.Source = p.opt.Host
			ch <- bounce
		}

		// Add message to deletion set
		processedMessages.AddNum(msg.SeqNum)
	}

	if err := <-done; err != nil {
		return fmt.Errorf("retrieving bounce messages via IMAP: %v", err)
	}

	// If TrashFolded is set move processed messages to it otherwise delete them
	if !processedMessages.Empty() {
		if p.opt.TrashFolder == "" {
			// Mark processed messages for deletion
			if err := c.Store(processedMessages,
				imap.FormatFlagsOp(imap.AddFlags, true),
				[]interface{}{imap.DeletedFlag},
				nil); err != nil {
				return fmt.Errorf("marking IMAP messages with \\deleted flag: %v", err)
			}
			// Expunge them
			if err := c.Expunge(nil); err != nil {
				return fmt.Errorf("expunging IMAP messages: %v", err)
			}

		} else {
			if err := c.Move(processedMessages, p.opt.TrashFolder); err != nil {
				return fmt.Errorf("moving messages to %s IMAP error: %v", p.opt.TrashFolder, err)
			}
		}
	}

	return nil
}
