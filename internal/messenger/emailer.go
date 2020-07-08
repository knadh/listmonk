package messenger

import (
	"crypto/tls"
	"fmt"
	"math/rand"
	"net/smtp"
	"net/textproto"

	"github.com/jaytaylor/html2text"
	"github.com/knadh/smtppool"
)

const emName = "email"

// Server represents an SMTP server's credentials.
type Server struct {
	Username      string            `json:"username"`
	Password      string            `json:"password"`
	AuthProtocol  string            `json:"auth_protocol"`
	EmailFormat   string            `json:"email_format"`
	TLSEnabled    bool              `json:"tls_enabled"`
	TLSSkipVerify bool              `json:"tls_skip_verify"`
	EmailHeaders  map[string]string `json:"email_headers"`

	// Rest of the options are embedded directly from the smtppool lib.
	// The JSON tag is for config unmarshal to work.
	smtppool.Opt `json:",squash"`

	pool *smtppool.Pool
}

// Emailer is the SMTP e-mail messenger.
type Emailer struct {
	servers []*Server
}

// NewEmailer creates and returns an e-mail Messenger backend.
// It takes multiple SMTP configurations.
func NewEmailer(servers ...Server) (*Emailer, error) {
	e := &Emailer{
		servers: make([]*Server, 0, len(servers)),
	}

	for _, srv := range servers {
		s := srv
		var auth smtp.Auth
		switch s.AuthProtocol {
		case "cram":
			auth = smtp.CRAMMD5Auth(s.Username, s.Password)
		case "plain":
			auth = smtp.PlainAuth("", s.Username, s.Password, s.Host)
		case "login":
			auth = &smtppool.LoginAuth{Username: s.Username, Password: s.Password}
		case "":
		default:
			return nil, fmt.Errorf("unknown SMTP auth type '%s'", s.AuthProtocol)
		}
		s.Opt.Auth = auth

		// TLS config.
		if s.TLSEnabled {
			s.TLSConfig = &tls.Config{}
			if s.TLSSkipVerify {
				s.TLSConfig.InsecureSkipVerify = s.TLSSkipVerify
			} else {
				s.TLSConfig.ServerName = s.Host
			}
		}

		pool, err := smtppool.New(s.Opt)
		if err != nil {
			return nil, err
		}

		s.pool = pool
		e.servers = append(e.servers, &s)
	}

	return e, nil
}

// Name returns the Server's name.
func (e *Emailer) Name() string {
	return emName
}

// Push pushes a message to the server.
func (e *Emailer) Push(fromAddr string, toAddr []string, subject string, m []byte, atts []Attachment) error {
	// If there are more than one SMTP servers, send to a random
	// one from the list.
	var (
		ln  = len(e.servers)
		srv *Server
	)
	if ln > 1 {
		srv = e.servers[rand.Intn(ln)]
	} else {
		srv = e.servers[0]
	}

	// Are there attachments?
	var files []smtppool.Attachment
	if atts != nil {
		files = make([]smtppool.Attachment, 0, len(atts))
		for _, f := range atts {
			a := smtppool.Attachment{
				Filename: f.Name,
				Header:   f.Header,
				Content:  make([]byte, len(f.Content)),
			}
			copy(a.Content, f.Content)
			files = append(files, a)
		}
	}

	mtext, err := html2text.FromString(string(m), html2text.Options{PrettyTables: true})
	if err != nil {
		return err
	}

	em := smtppool.Email{
		From:        fromAddr,
		To:          toAddr,
		Subject:     subject,
		Attachments: files,
	}

	// If there are custom e-mail headers, attach them.
	if len(srv.EmailHeaders) > 0 {
		em.Headers = textproto.MIMEHeader{}
		for k, v := range srv.EmailHeaders {
			em.Headers.Set(k, v)
		}
	}

	switch srv.EmailFormat {
	case "html":
		em.HTML = m
	case "plain":
		em.Text = []byte(mtext)
	default:
		em.HTML = m
		em.Text = []byte(mtext)
	}

	return srv.pool.Send(em)
}

// Flush flushes the message queue to the server.
func (e *Emailer) Flush() error {
	return nil
}

// Close closes the SMTP pools.
func (e *Emailer) Close() error {
	for _, s := range e.servers {
		s.pool.Close()
	}
	return nil
}
