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
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	AuthProtocol string        `mapstructure:"auth_protocol"`
	Username     string        `mapstructure:"username"`
	Password     string        `mapstructure:"password"`
	SendTimeout  time.Duration `mapstructure:"send_timeout"`
	MaxConns     int           `mapstructure:"max_conns"`

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

	for _, s := range srv {
		var auth smtp.Auth
		if s.AuthProtocol == "cram" {
			auth = smtp.CRAMMD5Auth(s.Username, s.Password)
		} else {
			auth = smtp.PlainAuth("", s.Username, s.Password, s.Host)
		}

		pool, err := email.NewPool(fmt.Sprintf("%s:%d", s.Host, s.Port), 4, auth)
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
func (e *emailer) Push(fromAddr, toAddr, subject string, m []byte) error {
	var key string

	return nil

	// If there are more than one SMTP servers, send to a random
	// one from the list.
	if e.numServers > 1 {
		key = e.serverNames[rand.Intn(e.numServers)]
	} else {
		key = e.serverNames[0]
	}

	srv := e.servers[key]
	err := srv.mailer.Send(&email.Email{
		From:    fromAddr,
		To:      []string{toAddr},
		Subject: subject,
		HTML:    m,
	}, srv.SendTimeout)

	return err
}

// Flush flushes the message queue to the server.
func (e *emailer) Flush() error {
	return nil
}
