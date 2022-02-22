package metrics

type Counter struct {
	Name   string
	Labels Labels
}

// CounterAdd adds a value to a metric counter given a combination of metric
// name and labels.
func (s *Manager) CounterAdd(val int, name string, labels Labels) {
	m := s.composeMetricString(name, labels)
	s.metrics.GetOrCreateCounter(m).Add(val)
}

// CounterIncrement increments a metric counter for a given combination of
// metric name and labels.
func (s *Manager) CounterIncrement(name string, labels Labels) {
	m := s.composeMetricString(name, labels)
	s.metrics.GetOrCreateCounter(m).Inc()
}

// CounterDecrement decrements a metric counter for a given combination of
// metric name and labels.
func (s *Manager) CounterDecrement(name string, labels Labels) {
	m := s.composeMetricString(name, labels)
	s.metrics.GetOrCreateCounter(m).Dec()
}

// CounterSet sets a metric counter to a given value for a given combination of
// metric name and labels.
func (s *Manager) CounterSet(val uint64, name string, labels Labels) {
	m := s.composeMetricString(name, labels)
	s.metrics.GetOrCreateCounter(m).Set(val)
}
