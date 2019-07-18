package messenger

import "net/textproto"

// Messenger is an interface for a generic messaging backend,
// for instance, e-mail, SMS etc.
type Messenger interface {
	Name() string

	Push(fromAddr string, toAddr []string, subject string, message []byte, atts []*Attachment) error
	Flush() error
}

// Attachment represents a file or blob attachment that can be
// sent along with a message by a Messenger.
type Attachment struct {
	Name    string
	Header  textproto.MIMEHeader
	Content []byte
}
