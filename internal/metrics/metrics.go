package metrics

import (
	"fmt"
	"io"
	"time"

	"github.com/VictoriaMetrics/metrics"
)

// General config options for exposing metrics.
type Opts struct {
	Namespace         string // Prefix for all label names separated with `_`.
	ExportGoMetrics   bool   // Export Go process metrics.
	ExportHTTPMetrics bool   // Export HTTP Request metrics by injecting a middleware in all requests.
}

// Manager represents options for interacting with app metrics.
type Manager struct {
	metrics   *metrics.Set // Initialise a new scope for storing/reading metrics.
	startTime time.Time    // Used for calculating uptime of the app.
	Opts      Opts
}

// Init returns a new configured instance of metrics manager.
func Init(opts Opts) *Manager {
	return &Manager{
		metrics:   metrics.NewSet(),
		startTime: time.Now(),
		Opts:      opts,
	}
}

// Increment increments a metric counter for a given combination of
// metric name and labels.
// This is generally used for Counter metric type.
func (s *Manager) Increment(name string, labels Labels) {
	s.metrics.GetOrCreateCounter(s.composeMetricString(name, labels)).Inc()
}

// FlushMetrics writes the metrics data from the internal store
// to the buffer.
func (s *Manager) FlushMetrics(buf io.Writer) {
	s.metrics.WritePrometheus(buf)
	if s.Opts.ExportGoMetrics {
		metrics.WriteProcessMetrics(buf)
	}
	// Export uptime in seconds.
	fmt.Fprintf(buf, fmt.Sprintf("%s_uptime_seconds %d\n", s.Opts.Namespace, int(time.Since(s.startTime).Seconds())))
}
