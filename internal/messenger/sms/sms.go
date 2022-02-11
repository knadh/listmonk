package sms

import (
	"errors"
	"github.com/knadh/listmonk/internal/messenger"
)

const emName = "sms"

// Server represents an SMTP server's credentials.
type Server struct {
	Username      string            `json:"username"`
	Password      string            `json:"password"`
	AuthProtocol  string            `json:"auth_protocol"`
	TLSType       string            `json:"tls_type"`
	TLSSkipVerify bool              `json:"tls_skip_verify"`
	EmailHeaders  map[string]string `json:"email_headers"`
}

type SMSSender struct {
	servers []*Server
}

// New returns an SMTP e-mail Messenger backend with a the given SMTP servers.
func New(servers ...Server) (*SMSSender, error) {
	e := &SMSSender{
		servers: make([]*Server, 0, len(servers)),
	}

	for _, srv := range servers {
		s := srv
		e.servers = append(e.servers, &s)
	}

	return e, nil
}

func (e *SMSSender) Name() string {
	return emName
}

func (e *SMSSender) Push(m messenger.Message) error {
	return errors.New("Not implemented")
}

func (e *SMSSender) Flush() error {
	return errors.New("Not implemented")
}

func (e *SMSSender) Close() error {
	return errors.New("Not implemented")
}
