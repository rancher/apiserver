package metrics

import "github.com/prometheus/client_golang/prometheus"

var prometheusMetrics = false

const (
	resourceLabel = "resource"
	methodLabel   = "method"
	codeLabel     = "code"
)
var (
	// https://prometheus.io/docs/practices/instrumentation/#use-labels explains logic of having 1 total_requests
	// counter with code label vs a counter for each code
	TotalResponses = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: "steve_api",
			Name:      "total_requests",
			Help:      "Total count API requests",
		},
		[]string{"resource", "method", "code", "id"},
	)
)

func IncTotalResponses(resource, method, code, id string) {
	if prometheusMetrics {
		TotalResponses.With(
			prometheus.Labels{
				"resource": resource,
				"method":   method,
				"code":     code,
				"id":       id,
			},
		).Inc()
	}
}