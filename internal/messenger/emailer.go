package messenger

import (
	"errors"
	"fmt"
	"math/rand"
	"net/smtp"
	"time"

	"github.com/jaytaylor/html2text"
	"github.com/jordan-wright/email"
)

const emName = "email"

// loginAuth is used for enabling SMTP "LOGIN" auth.
type loginAuth struct {
	username string
	password string
}

// Server represents an SMTP server's credentials.
type Server struct {
	Name          string
	Host          string        `koanf:"host"`
	Port          int           `koanf:"port"`
	AuthProtocol  string        `koanf:"auth_protocol"`
	Username      string        `koanf:"username"`
	Password      string        `koanf:"password"`
	EmailFormat   string        `koanf:"email_format"`
	HelloHostname string        `koanf:"hello_hostname"`
	SendTimeout   time.Duration `koanf:"send_timeout"`
	MaxConns      int           `koanf:"max_conns"`

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
		switch s.AuthProtocol {
		case "cram":
			auth = smtp.CRAMMD5Auth(s.Username, s.Password)
		case "plain":
			auth = smtp.PlainAuth("", s.Username, s.Password, s.Host)
		case "login":
			auth = &loginAuth{username: s.Username, password: s.Password}
		case "":
		default:
			return nil, fmt.Errorf("unknown SMTP auth type '%s'", s.AuthProtocol)
		}

		pool, err := email.NewPool(fmt.Sprintf("%s:%d", s.Host, s.Port), s.MaxConns, auth)
		if err != nil {
			return nil, err
		}

		// Optional SMTP HELLO hostname.
		if server.HelloHostname != "" {
			pool.SetHelloHostname(server.HelloHostname)
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

	mtext, err := html2text.FromString(string(m), html2text.Options{PrettyTables: true})
	if err != nil {
		return err
	}

	srv := e.servers[key]
	em := &email.Email{
		From:        fromAddr,
		To:          toAddr,
		Subject:     subject,
		Attachments: files,
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

	return srv.mailer.Send(em, srv.SendTimeout)
}

// Flush flushes the message queue to the server.
func (e *emailer) Flush() error {
	return nil
}

// https://gist.github.com/andelf/5118732
// Adds support for SMTP LOGIN auth.
func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte{}, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch string(fromServer) {
		case "Username:":
			return []byte(a.username), nil
		case "Password:":
			return []byte(a.password), nil
		default:
			return nil, errors.New("unkown SMTP fromServer")
		}
	}
	return nil, nil
}
