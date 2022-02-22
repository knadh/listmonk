package metrics

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/VictoriaMetrics/metrics"
	"github.com/labstack/echo"
)

// Manager represents options for storing metrics.
type Manager struct {
	metrics       *metrics.Set
	namespace     string // Optional string to prepend to the metric name.
	HandlerConfig HandlerConfig
}

const (
	notFoundPath = "/not-found"
)

// Labels simply represents a map[string]string but is fewer characters to type.
type Labels map[string]string

// NewStats returns a new configured instance of Manager.
func New(ns string) *Manager {
	return &Manager{
		metrics:   metrics.NewSet(),
		namespace: ns,
	}
}

func (s *Manager) composeMetricString(metric_name string, labels map[string]string) string {
	metric_base := fmt.Sprintf("%s_%s", s.namespace, metric_name)

	if len(labels) > 0 {
		var merged_labels []string

		// Metrics will appear as different metrics if the labels are written in
		// a different order. As maps are unsorted, we have to account for this.
		keys := make([]string, 0, len(labels))
		for k := range labels {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, key := range keys {
			merged_labels = append(merged_labels, fmt.Sprintf("%s=\"%s\"", key, labels[key]))
		}

		metric_with_labels := fmt.Sprintf("%s{%s}", metric_base, strings.Join(merged_labels, ", "))

		return metric_with_labels
	} else {
		return metric_base
	}
}

func normalizeHTTPStatus(status int) string {
	if status < 200 {
		return "1xx"
	} else if status < 300 {
		return "2xx"
	} else if status < 400 {
		return "3xx"
	} else if status < 500 {
		return "4xx"
	}
	return "5xx"
}

func isNotFoundHandler(handler echo.HandlerFunc) bool {
	return reflect.ValueOf(handler).Pointer() == reflect.ValueOf(echo.NotFoundHandler).Pointer()
}
