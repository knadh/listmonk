package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/labstack/echo/v4"
)

// EventStream serves an endpoint that never closes and pushes a
// live event stream (text/event-stream) such as a error messages.
func (a *App) EventStream(c echo.Context) error {
	hdr := c.Response().Header()
	hdr.Set(echo.HeaderContentType, "text/event-stream")
	hdr.Set(echo.HeaderCacheControl, "no-store")
	hdr.Set(echo.HeaderConnection, "keep-alive")

	// Subscribe to the event stream with a random ID.
	id := fmt.Sprintf("api:%v", time.Now().UnixNano())
	sub, err := a.events.Subscribe(id)
	if err != nil {
		log.Fatalf("error subscribing to events: %v", err)
	}

	ctx := c.Request().Context()
	for {
		select {
		case e := <-sub:
			b, err := json.Marshal(e)
			if err != nil {
				a.log.Printf("error marshalling event: %v", err)
				continue
			}

			c.Response().Write([]byte(fmt.Sprintf("retry: 3000\ndata: %s\n\n", b)))
			c.Response().Flush()

		case <-ctx.Done():
			// On HTTP connection close, unsubscribe.
			a.events.Unsubscribe(id)
			return nil
		}
	}

}
