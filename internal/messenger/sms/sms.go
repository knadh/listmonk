package sms

import (
	"errors"
	"github.com/comilio/go-sms-send"
	"github.com/knadh/listmonk/internal/messenger"
)

const emName = "sms"

// Server represents an SMTP server's credentials.
type Server struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	smsSender *sms.SMS
}

type SMSSender struct {
	servers []*Server
}

// New returns an SMTP e-mail Messenger backend with a the given SMTP servers.
func New(servers ...Server) (*SMSSender, error) {
	var SMS sms.SMS

	e := &SMSSender{
		servers: make([]*Server, 0, len(servers)),
	}

	for _, srv := range servers {
		s := srv
		SMS.Auth(s.Username, s.Password)
		e.servers = append(e.servers, &s)
	}

	return e, nil
}

func (e *SMSSender) Name() string {
	return emName
}

func (e *SMSSender) Push(m messenger.Message) error {
	/*var (
		ln  = len(e.servers)
		srv *Server
	)
	if ln > 1 {
		srv = e.servers[rand.Intn(ln)]
	} else {
		srv = e.servers[0]
	}*/
	//return srv.smsSender.Send(m.To, m.Body, "Classic")
	return errors.New("Not implemented")
}

func (e *SMSSender) Flush() error {
	return errors.New("Not implemented")
}

func (e *SMSSender) Close() error {
	return errors.New("Not implemented")
}
