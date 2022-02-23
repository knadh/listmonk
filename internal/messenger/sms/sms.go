package sms

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/listmonk/internal/messenger"
)

const emName = "sms"

// Server represents an SMTP server's credentials.
type Server struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	Host       string `json:"host"`
	ApiKey     string `json:"api_key"`
	RestClient *resty.Client
	db         *sqlx.DB
}

type SMSSender struct {
	servers []*Server
}

// New returns an SMS sms Messenger backend with a the given SMTP servers.
func New(db *sqlx.DB, servers ...Server) (*SMSSender, error) {
	e := &SMSSender{
		servers: make([]*Server, 0, len(servers)),
	}

	for _, srv := range servers {
		s := srv
		s.RestClient = resty.New()
		s.db = db
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
	/**
	If the subject is set, use the subject, otherwise use username
	*/
	sender := Ternary(len(strings.TrimSpace(m.Campaign.Subject)) > 0, m.Campaign.Subject, srv.Username).(string)

	log.Println(`{"to":"` + m.Subscriber.Telephone + `", "message":"` + string([]byte(m.Campaign.Body)) + `", "from": "` + sender + `"}`)

	resp, err := client.R().SetHeader("Accept", "application/json").SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetHeader("apiKey", srv.ApiKey).
		SetFormData(map[string]string{"username": srv.Username, "to": m.Subscriber.Telephone, "from": sender, "message": string([]byte(m.Campaign.Body))}).Post(srv.Host)

	fmt.Println("  Error      :", err)
	fmt.Println("  Status Code:", resp.StatusCode())
	fmt.Println("  Status     :", resp.Status())
	fmt.Println("  Body       :\n", resp)
	fmt.Println()

	var response Response
	var messageId = ""
	var status = ""
	var statusCode = 0
	json.Unmarshal([]byte(resp.Body()), &response)

	if len(response.SMSMessageData.Recipients) == 0 {
		status = response.SMSMessageData.Message
	} else {
		messageId = response.SMSMessageData.Recipients[0].MessageID
		status = response.SMSMessageData.Recipients[0].Status
		statusCode = response.SMSMessageData.Recipients[0].StatusCode
	}
	// this insert needs to be in a loop and then store each of the Recipients
	sqlStatement := `INSERT INTO campaign_sms(campaign_id, userid, reference, status, statusCode, telephone, metadata) VALUES($1, $2, $3, $4, $5, $6, $7) RETURNING id`
	id := 0
	var errDb = srv.db.QueryRow(sqlStatement, m.Campaign.ID, m.Subscriber.Userid, messageId, status, statusCode, m.Subscriber.Telephone, resp.Body()).Scan(&id)
	if errDb != nil {
		panic(errDb)
	}
	fmt.Println("New record ID is:", id)

	return nil
}

func Ternary(statement bool, a, b interface{}) interface{} {
	if statement {
		return a
	}
	return b
}

func (e *SMSSender) Flush() error {
	return errors.New("not implemented")
}

func (e *SMSSender) Close() error {
	return errors.New("not implemented")
}

type Response struct {
	SMSMessageData struct {
		Message    string `json:"Message"`
		Recipients []struct {
			Cost         string `json:"cost"`
			MessageID    string `json:"messageId"`
			MessageParts int    `json:"messageParts"`
			Number       string `json:"number"`
			Status       string `json:"status"`
			StatusCode   int    `json:"statusCode"`
		} `json:"Recipients"`
	} `json:"SMSMessageData"`
}
