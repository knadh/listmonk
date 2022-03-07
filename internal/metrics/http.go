package metrics

import (
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
)

const (
	httpRequestsCount    = "requests_total"
	httpRequestsDuration = "request_duration_seconds"
	notFoundPath         = "/not-found"
)

// HTTPMiddleware returns an echo Middleware for instrumenting HTTP Requests.
func (s *Manager) HTTPMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var (
				req  = c.Request()
				path = c.Path()
			)

			if c.Response().Status == 404 {
				path = notFoundPath
			}

			// Perform the request.
			start := time.Now()
			err := next(c)
			if err != nil {
				c.Error(err)
			}

			// Construct metrics on the fly.
			var (
				requestsTotal = s.composeMetricString(httpRequestsCount, Labels{
					"path":   path,
					"method": req.Method,
					"status": fmt.Sprintf("%d", (c.Response().Status)),
				})
				requestsDuration = s.composeMetricString(httpRequestsDuration, Labels{
					"path":   path,
					"method": req.Method,
				})
			)

			// Update the metric values.
			s.metrics.GetOrCreateCounter(requestsTotal).Inc()
			s.metrics.GetOrCreateHistogram(requestsDuration).UpdateDuration(start)

			return err
		}
	}
}
