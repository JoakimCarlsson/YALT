package metrics

import (
	"net/http"
	"time"
)

// RoundTripper is a custom RoundTripper that records metrics for each HTTP request.
type RoundTripper struct {
	original http.RoundTripper // The original RoundTripper to delegate to
	metrics  *Metrics          // The Metrics instance to record metrics
}

// NewRoundTripper creates a new instance of RoundTripper.
func NewRoundTripper(transport *http.Transport, metrics *Metrics) http.RoundTripper {
	return &RoundTripper{
		original: transport,
		metrics:  metrics,
	}
}

// RoundTrip executes a single HTTP transaction and records metrics.
func (rt *RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	startTime := time.Now()
	resp, err := rt.original.RoundTrip(req)
	duration := time.Since(startTime)

	failed := err != nil || (resp != nil && resp.StatusCode >= 400)

	rt.metrics.AddRequestMetrics(duration, failed)

	return resp, err
}
