package metrics

import (
	"bytes"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptrace"
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
	var startTime = time.Now()

	metrics := &RequestMetrics{
		StartTime: startTime,
		Request:   cloneRequest(req),
	}

	trace := &httptrace.ClientTrace{
		DNSStart:             func(httptrace.DNSStartInfo) { metrics.DNSStart = startTime },
		DNSDone:              func(httptrace.DNSDoneInfo) { metrics.DNSDone = startTime },
		ConnectStart:         func(string, string) { metrics.ConnectStart = startTime },
		ConnectDone:          func(string, string, error) { metrics.ConnectDone = startTime },
		TLSHandshakeStart:    func() { metrics.TLSHandshakeStart = startTime },
		TLSHandshakeDone:     func(tls.ConnectionState, error) { metrics.TLSHandshakeDone = startTime },
		GotConn:              func(httptrace.GotConnInfo) { metrics.GotConn = startTime },
		WroteHeaders:         func() { metrics.WroteHeaders = startTime },
		WroteRequest:         func(httptrace.WroteRequestInfo) { metrics.WroteRequest = startTime },
		GotFirstResponseByte: func() { metrics.GotFirstResponseByte = startTime },
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	resp, err := m.next.RoundTrip(req)

	metrics.EndTime = time.Now()
	if resp != nil {
		metrics.Response = cloneResponse(resp)
	}
	metrics.Error = err

	m.metrics.AddRequestMetrics(*metrics)

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
