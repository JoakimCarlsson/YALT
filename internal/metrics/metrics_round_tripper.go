package metrics

import (
	"bytes"
	"io"
	"net/http"
	"time"
)

// RoundTripper is a custom RoundTripper that records metrics
type RoundTripper struct {
	next    http.RoundTripper
	metrics *Metrics
}

// NewMetricsRoundTripper creates a new RoundTripper with metrics
func (m *Metrics) NewMetricsRoundTripper(
	transport *http.Transport,
	metrics *Metrics,
) http.RoundTripper {
	return &RoundTripper{
		next:    transport,
		metrics: metrics,
	}
}

// RoundTrip executes a single HTTP transaction and records metrics
func (m *RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	startTime := time.Now()

	reqClone := cloneRequest(req)

	resp, err := m.next.RoundTrip(req)

	endTime := time.Now()

	var respClone *http.Response
	if resp != nil {
		respClone = cloneResponse(resp)
	}

	m.metrics.AddRequestMetrics(RequestMetrics{
		StartTime: startTime,
		EndTime:   endTime,
		Request:   reqClone,
		Response:  respClone,
		Error:     err,
	})

	return resp, err
}

// cloneRequest clones an HTTP request
func cloneRequest(req *http.Request) *http.Request {
	clone := new(http.Request)
	*clone = *req
	if req.Body != nil {
		bodyBytes, _ := io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		clone.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}
	return clone
}

// cloneResponse clones an HTTP response
func cloneResponse(resp *http.Response) *http.Response {
	clone := new(http.Response)
	*clone = *resp
	if resp.Body != nil {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		clone.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}
	return clone
}
