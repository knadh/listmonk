package metrics

import (
	"fmt"
	"sort"
	"strings"
)

// Labels represents a K/V pair to inject inside metric names.
type Labels map[string]string

func (s *Manager) composeMetricString(metric_name string, labels map[string]string) string {
	base := fmt.Sprintf("%s_%s", s.Opts.Namespace, metric_name)

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

		metric_with_labels := fmt.Sprintf("%s{%s}", base, strings.Join(merged_labels, ", "))

		return metric_with_labels
	} else {
		return base
	}
}
