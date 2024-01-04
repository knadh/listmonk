package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/labstack/echo/v4"
)

// handleEventStream serves an endpoint that never closes and pushes a
// live event stream (text/event-stream) such as a error messages.
func handleEventStream(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
	)

	h := c.Response().Header()
	h.Set(echo.HeaderContentType, "text/event-stream")
	h.Set(echo.HeaderCacheControl, "no-store")
	h.Set(echo.HeaderConnection, "keep-alive")

	// Subscribe to the event stream with a random ID.
	id := fmt.Sprintf("api:%v", time.Now().UnixNano())
	sub, err := app.events.Subscribe(id)
	if err != nil {
		log.Fatalf("error subscribing to events: %v", err)
	}

	ctx := c.Request().Context()
	for {
		select {
		case e := <-sub:
			b, err := json.Marshal(e)
			if err != nil {
				app.log.Printf("error marshalling event: %v", err)
				continue
			}

			c.Response().Write([]byte(fmt.Sprintf("retry: 3000\ndata: %s\n\n", b)))
			c.Response().Flush()

		case <-ctx.Done():
			// On HTTP connection close, unsubscribe.
			app.events.Unsubscribe(id)
			return nil
		}
	}

	return nil
}
