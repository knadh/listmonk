package main

import (
	"github.com/labstack/echo/v4"
	"log"
)

type smsDeliveryReq struct {
	ID            string `json:"id"`
	Status        string `json:"status"`
	PhoneNumber   string `json:"phoneNumber"`
	NetworkCode   string `json:"networkCode"`
	FailureReason string `json:"failureReason"`
	RetryCount    int    `json:"retryCount"`
}

func handleDeliveryRequest(c echo.Context) error {
	var (
		app         = c.Get("app").(*App)
		deliveryReq smsDeliveryReq
	)
	if err := c.Bind(&deliveryReq); err != nil {
		return err
	}

	log.Println("ID : " + deliveryReq.ID + " Status: " + deliveryReq.Status)

	sqlStatement := `UPDATE campaign_sms SET status = $2, network_code = $3, failure_reason = $4 WHERE reference = $1;`
	var _, errDb = app.db.Exec(sqlStatement, deliveryReq.ID, deliveryReq.Status, deliveryReq.NetworkCode, deliveryReq.FailureReason)
	if errDb != nil {
		panic(errDb)
	}
	return nil
}
