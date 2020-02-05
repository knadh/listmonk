package messenger

import (
	"fmt"
	"math/rand"
	"net/smtp"
	"time"

	"github.com/jordan-wright/email"
)

const emName = "email"

// Server represents an SMTP server's credentials.
type Server struct {
	Name         string
	Host         string        `koanf:"host"`
	Port         int           `koanf:"port"`
	AuthProtocol string        `koanf:"auth_protocol"`
	Username     string        `koanf:"username"`
	Password     string        `koanf:"password"`
	SendTimeout  time.Duration `koanf:"send_timeout"`
	MaxConns     int           `koanf:"max_conns"`

	mailer *email.Pool
}

type emailer struct {
	servers     map[string]*Server
	serverNames []string
	numServers  int
}

// NewEmailer creates and returns an e-mail Messenger backend.
// It takes multiple SMTP configurations.
func NewEmailer(srv ...Server) (Messenger, error) {
	e := &emailer{
		servers: make(map[string]*Server),
	}

	for _, server := range srv {
		s := server
		var auth smtp.Auth
		if s.AuthProtocol == "cram" {
			auth = smtp.CRAMMD5Auth(s.Username, s.Password)
		} else if s.AuthProtocol == "plain" {
			auth = smtp.PlainAuth("", s.Username, s.Password, s.Host)
		}

		pool, err := email.NewPool(fmt.Sprintf("%s:%d", s.Host, s.Port), s.MaxConns, auth)
		if err != nil {
			return nil, err
		}

		s.mailer = pool
		e.servers[s.Name] = &s
		e.serverNames = append(e.serverNames, s.Name)
	}

	e.numServers = len(e.serverNames)
	return e, nil
}

// Name returns the Server's name.
func (e *emailer) Name() string {
	return emName
}

// Push pushes a message to the server.
func (e *emailer) Push(fromAddr string, toAddr []string, subject string, m []byte, atts []*Attachment) error {
	var key string

	// If there are more than one SMTP servers, send to a random
	// one from the list.
	if e.numServers > 1 {
		key = e.serverNames[rand.Intn(e.numServers)]
	} else {
		key = e.serverNames[0]
	}

	// Are there attachments?
	var files []*email.Attachment
	if atts != nil {
		files = make([]*email.Attachment, 0, len(atts))
		for _, f := range atts {
			a := &email.Attachment{
				Filename: f.Name,
				Header:   f.Header,
				Content:  make([]byte, len(f.Content)),
			}
			copy(a.Content, f.Content)
			files = append(files, a)
		}
	}

	srv := e.servers[key]
	err := srv.mailer.Send(&email.Email{
		From:        fromAddr,
		To:          toAddr,
		Subject:     subject,
		HTML:        m,
		Attachments: files,
	}, srv.SendTimeout)

	return err
}

// Flush flushes the message queue to the server.
func (e *emailer) Flush() error {
	return nil
}
