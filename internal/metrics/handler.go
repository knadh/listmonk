package metrics

import (
	"strings"

	"github.com/VictoriaMetrics/metrics"
	"github.com/labstack/echo/v4"
)

type HandlerConfig struct {
	Enabled         bool `koanf:"enabled"`
	ExportGoMetrics bool `koanf:"export_go_metrics"`
}

// SetHandlerConfig holds the Prometheus http handler configuration for a given
// metrics set.
func (s *Manager) SetHandlerConfig(c HandlerConfig) {
	s.HandlerConfig = c
}

// GetHandlerConfig returns the Prometheus http handler configuration for a given
// metrics set.
func (s *Manager) GetHandlerConfig() HandlerConfig {
	return s.HandlerConfig
}

// HandlePromMetrics performs the work of writing the metrics to the body of an
// http request for metrics to be scraped.
func (s *Manager) HandlePromMetrics() echo.HandlerFunc {
	return echo.HandlerFunc(
		func(c echo.Context) error {
			metrics.WritePrometheus(c.Response().Writer, s.HandlerConfig.ExportGoMetrics)
			return nil
		},
	)
}

// MetricsMiddlewareWithConfig performs the work of instrumenting requests flowing
// through an Echo instance.
func (s *Manager) MetricsMiddlewareWithConfig() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()

			var (
				path          string
				allPathParts  []string
				keptPathParts []string
			)

			allPathParts = strings.Split(c.Path(), "/")
			for i, part := range allPathParts {
				if i < 3 {
					keptPathParts = append(keptPathParts, part)
				} else {
					break
				}
			}

			path = strings.Join(keptPathParts, "/")

			// To avoid high cardinality of 404s we set all unfounds to a common value
			if c.Response().Status == 404 {
				path = notFoundPath
			}

			err := next(c)

			if err != nil {
				c.Error(err)
			}

			requestsTotalMetricName := s.composeMetricString("requests_total", Labels{
				"path":   path,
				"method": req.Method,
				"status": normalizeHTTPStatus(c.Response().Status),
			})
			metrics.GetOrCreateCounter(requestsTotalMetricName).Inc()

			responseSizeMetricName := s.composeMetricString("response_size", Labels{
				"path":   path,
				"method": req.Method,
			})
			metrics.GetOrCreateHistogram(responseSizeMetricName).Update(float64(c.Response().Size))

			return err
		}
	}
}
