package metrics

import (
	"github.com/VictoriaMetrics/metrics"
)

// InfoSet sets an info type metric. Under the hood it's actually Gauge type
// metric with a fixed value. The value given to `name` will be suffixed with
// `_info`.
func (s *Manager) InfoSet(name string, labels Labels) {
	m := s.composeMetricString(name+"_info", labels)
	metrics.GetOrCreateGauge(m, func() float64 { return 1 })
}
