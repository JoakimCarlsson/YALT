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
	metrics := &RequestMetrics{
		StartTime: time.Now(),
		Request:   cloneRequest(req),
	}

	trace := &httptrace.ClientTrace{
		DNSStart:             func(httptrace.DNSStartInfo) { metrics.DNSStart = time.Now() },
		DNSDone:              func(httptrace.DNSDoneInfo) { metrics.DNSDone = time.Now() },
		ConnectStart:         func(string, string) { metrics.ConnectStart = time.Now() },
		ConnectDone:          func(string, string, error) { metrics.ConnectDone = time.Now() },
		TLSHandshakeStart:    func() { metrics.TLSHandshakeStart = time.Now() },
		TLSHandshakeDone:     func(tls.ConnectionState, error) { metrics.TLSHandshakeDone = time.Now() },
		GotConn:              func(httptrace.GotConnInfo) { metrics.GotConn = time.Now() },
		WroteHeaders:         func() { metrics.WroteHeaders = time.Now() },
		WroteRequest:         func(httptrace.WroteRequestInfo) { metrics.WroteRequest = time.Now() },
		GotFirstResponseByte: func() { metrics.GotFirstResponseByte = time.Now() },
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
