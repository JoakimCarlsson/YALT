package metrics

import (
	"net/http"
	"time"
)

// RoundTripper is a custom RoundTripper that records metrics
type RoundTripper struct {
	original http.RoundTripper
	metrics  *Metrics
}

// NewMetricsRoundTripper creates a new RoundTripper with metrics
func (m *Metrics) NewMetricsRoundTripper(transport *http.Transport, metrics *Metrics) http.RoundTripper {
	return &RoundTripper{
		original: transport,
		metrics:  metrics,
	}
}

// RoundTrip executes a single HTTP transaction and records metrics
func (mrt *RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	startTime := time.Now()
	resp, err := mrt.original.RoundTrip(req)
	duration := time.Since(startTime)

	failed := err != nil || (resp != nil && resp.StatusCode >= 400)
	mrt.metrics.AddRequestMetrics(duration, failed)

	return resp, err
}
