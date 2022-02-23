package main

import (
	"github.com/labstack/echo/v4"
)

type smsDeliveryReq struct {
	id            string `json:"id"`
	status        string `json:"status"`
	phoneNumber   string `json:"phoneNumber"`
	networkCode   string `json:"networkCode"`
	failureReason string `json:"failureReason"`
	retryCount    int    `json:"retryCount"`
}

func handleDeliveryRequest(c echo.Context) error {
	var (
		app         = c.Get("app").(*App)
		deliveryReq smsDeliveryReq
	)

	println(app)
	println(deliveryReq)
	return nil
}
