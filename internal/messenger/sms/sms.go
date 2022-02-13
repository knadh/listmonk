package sms

import (
	"errors"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/knadh/listmonk/internal/messenger"
	"log"
	"math/rand"
)

const emName = "sms"

// Server represents an SMTP server's credentials.
type Server struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	Host       string `json:"host"`
	ApiKey     string `json:"api_key"`
	RestClient *resty.Client
}

type SMSSender struct {
	servers []*Server
}

// New returns an SMS sms Messenger backend with a the given SMTP servers.
func New(servers ...Server) (*SMSSender, error) {
	e := &SMSSender{
		servers: make([]*Server, 0, len(servers)),
	}

	for _, srv := range servers {
		s := srv
		s.RestClient = resty.New()
		e.servers = append(e.servers, &s)
		log.Println("Setting SMS Server on " + srv.Host + " " + srv.ApiKey)
	}

	log.Println(len(servers))

	return e, nil
}

func (e *SMSSender) Name() string {
	return emName
}

//https://github.com/go-resty/resty
func (e *SMSSender) Push(m messenger.Message) error {
	client := resty.New()

	var (
		ln  = len(e.servers)
		srv *Server
	)
	if ln > 1 {
		srv = e.servers[rand.Intn(ln)]
	} else {
		srv = e.servers[0]
	}

	log.Println(`{"to":"` + m.Subscriber.Telephone + `", "message":"` + string([]byte(m.Body)) + `", "from": "` + m.Subscriber.Telephone + `"}`)

	resp, err := client.R().
		SetHeader("Accept", "application/json").
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetHeader("apiKey", srv.ApiKey).
		SetFormData(map[string]string{
			"username": srv.Username,
			"to":       m.Subscriber.Telephone,
			"from":     m.From,
			"message":  string([]byte(m.Body)),
		}).
		EnableTrace().
		Post(srv.Host)

	fmt.Println("Response Info:")
	fmt.Println("  Error      :", err)
	fmt.Println("  Status Code:", resp.StatusCode())
	fmt.Println("  Status     :", resp.Status())
	fmt.Println("  Proto      :", resp.Proto())
	fmt.Println("  Time       :", resp.Time())
	fmt.Println("  Received At:", resp.ReceivedAt())
	fmt.Println("  Body       :\n", resp)
	fmt.Println()
	return err
}

func (e *SMSSender) Flush() error {
	return errors.New("not implemented")
}

func (e *SMSSender) Close() error {
	return errors.New("not implemented")
}
