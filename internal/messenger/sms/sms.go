package sms

import (
	"errors"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/knadh/listmonk/internal/messenger"
)

const emName = "sms"

// Server represents an SMTP server's credentials.
type Server struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	smsSender *resty.Client
}

type SMSSender struct {
	servers []*Server
}

// New returns an SMTP e-mail Messenger backend with a the given SMTP servers.
func New(servers ...Server) (*SMSSender, error) {
	//var SMS sms.SMS
	//https://github.com/linxGnu/gosmpp/blob/master/example/main.go
	e := &SMSSender{
		servers: make([]*Server, 0, len(servers)),
	}

	for _, srv := range servers {
		s := srv
		//SMS.Auth(s.Username, s.Password)
		e.servers = append(e.servers, &s)
	}

	return e, nil
}

func (e *SMSSender) Name() string {
	return emName
}

//https://github.com/go-resty/resty
func (e *SMSSender) Push(m messenger.Message) error {
	client := resty.New()

	resp, err := client.R().
		SetHeader("Accept", "application/json").
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetHeader("apiKey", "").
		SetBody(`{"to":"to", "message":"1212123", "from": "from"}`).
		EnableTrace().
		Post("https://api.sandbox.africastalking.com/version1/messaging")

	// Explore response object
	fmt.Println("Response Info:")
	fmt.Println("  Error      :", err)
	fmt.Println("  Status Code:", resp.StatusCode())
	fmt.Println("  Status     :", resp.Status())
	fmt.Println("  Proto      :", resp.Proto())
	fmt.Println("  Time       :", resp.Time())
	fmt.Println("  Received At:", resp.ReceivedAt())
	fmt.Println("  Body       :\n", resp)
	fmt.Println()
	return errors.New("Still Not implemented")
}

func (e *SMSSender) Flush() error {
	return errors.New("Not implemented")
}

func (e *SMSSender) Close() error {
	return errors.New("Not implemented")
}
