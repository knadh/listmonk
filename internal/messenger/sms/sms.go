package sms

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"

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

	log.Println(`{"to":"` + m.Subscriber.Telephone + `", "message":"` + string([]byte(m.Body)) + `", "from": "` + srv.Username + `"}`)

	resp, err := client.R().
		SetHeader("Accept", "application/json").
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetHeader("apiKey", srv.ApiKey).
		SetFormData(map[string]string{
			"username": srv.Username,
			"to":       m.Subscriber.Telephone,
			// can we make this from be a post request? the senderId can be unique to the customer,
			// it's an added for value, the bank would want VBANK to be the sender not Yournotify.
			// I am already passing this to campaigns->subject (for sms) m.Subject ?
			"from":    srv.Username,
			"message": string([]byte(m.Body)),
		}).
		EnableTrace().
		Post(srv.Host)

	fmt.Println("  Error      :", err)
	fmt.Println("  Status Code:", resp.StatusCode())
	fmt.Println("  Status     :", resp.Status())
	fmt.Println("  Body       :\n", resp)
	fmt.Println()

	var response Response

	json.Unmarshal([]byte(resp.Body()), &response)
	messageId := response.SMSMessageData.Recipients[0].MessageID
	status := response.SMSMessageData.Recipients[0].Status
	statusCode := response.SMSMessageData.Recipients[0].StatusCode
	delivery := ""

	//fmt.Printf(" %s", messageId)

	// if (element.statusCode == 100) {
	// 	delivery = "Processed";
	// } else if (element.statusCode == 101) {
	// 	delivery = "Sent";
	// } else if (element.statusCode == 102) {
	// 	delivery = "Queued";
	// } else if (element.statusCode == 401) {
	// 	delivery = "RiskHold";
	// } else if (element.statusCode == 402) {
	// 	delivery = "InvalidSenderId";
	// } else if (element.statusCode == 403) {
	// 	delivery = "InvalidPhoneNumber";
	// } else if (element.statusCode == 404) {
	// 	delivery = "UnsupportedNumberType";
	// } else if (element.statusCode == 405) {
	// 	delivery = "InsufficientBalance";
	// } else if (element.statusCode == 406) {
	// 	delivery = "UserInBlacklist";
	// } else if (element.statusCode == 407) {
	// 	delivery = "CouldNotRoute";
	// } else if (element.statusCode == 500) {
	// 	delivery = "InternalServerError";
	// } else if (element.statusCode == 501) {
	// 	delivery = "GatewayError";
	// } else if (element.statusCode == 502) {
	// 	delivery = "RejectedByGateway";
	// }

	// this insert needs to be in a loop and then store each of the Recipients
	sqlStatement := `INSERT INTO campaign_sms(campaign_id, userid, reference, status, statusCode, delivery, telephone, metadata) VALUES($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`
	id := 0
	var errDb = srv.db.QueryRow(sqlStatement, m.Campaign.ID, m.Subscriber.Userid, messageId, status, statusCode, delivery, m.Subscriber.Telephone, resp.Body()).Scan(&id)
	if errDb != nil {
		panic(errDb)
	}
	//fmt.Println("New record ID is:", id)
	return nil
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

func (e *SMSSender) Flush() error {
	return errors.New("not implemented")
}

func (e *SMSSender) Close() error {
	return errors.New("not implemented")
}
