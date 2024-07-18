package metrics

import (
	"net/http"
	"time"
)

type MetricsRoundTripper struct {
	original http.RoundTripper
	metrics  *Metrics
}

func (m *Metrics) NewMetricsRoundTripper(transport *http.Transport, metrics *Metrics) http.RoundTripper {
	return &MetricsRoundTripper{
		original: transport,
		metrics:  metrics,
	}
}

func (mrt *MetricsRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	startTime := time.Now()
	resp, err := mrt.original.RoundTrip(req)
	duration := time.Since(startTime)

	failed := err != nil || (resp != nil && resp.StatusCode >= 400)
	mrt.metrics.AddRequestMetrics(duration, failed)

	return resp, err
}
