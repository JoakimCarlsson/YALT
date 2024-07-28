package metrics

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sort"
	"sync"
	"time"
)

// Metrics represents a collection of request metrics
type Metrics struct {
	mu         sync.Mutex
	requests   []RequestMetrics
	thresholds map[string][]string
	startTime  time.Time
}

// RequestMetrics represents a single request metric
type RequestMetrics struct {
	StartTime time.Time
	EndTime   time.Time
	Request   *http.Request
	Response  *http.Response
	Error     error
}

// NewMetrics creates a new Metrics instance
func NewMetrics(thresholds map[string][]string) *Metrics {
	return &Metrics{
		thresholds: thresholds,
		startTime:  time.Now(),
	}
}

// AddRequestMetrics adds a new request metric
func (m *Metrics) AddRequestMetrics(metrics RequestMetrics) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requests = append(m.requests, metrics)
}

// CalculateAndDisplayMetrics calculates and displays the metrics
func (m *Metrics) CalculateAndDisplayMetrics() {
	m.mu.Lock()
	defer m.mu.Unlock()

	totalDuration := time.Since(m.startTime)
	totalRequests := int64(len(m.requests))
	totalSeconds := totalDuration.Seconds()
	rps := float64(totalRequests) / totalSeconds

	var failedRequests int64
	var totalReqDuration time.Duration
	var totalDataSent, totalDataReceived int64
	durations := make([]time.Duration, 0, len(m.requests))

	for _, req := range m.requests {
		duration := req.EndTime.Sub(req.StartTime)
		durations = append(durations, duration)
		totalReqDuration += duration

		if req.Error != nil || (req.Response != nil && req.Response.StatusCode >= 400) {
			failedRequests++
		}

		totalDataSent += estimateRequestSize(req.Request)
		if req.Response != nil {
			totalDataReceived += estimateResponseSize(req.Response)
		}
	}

	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	minDuration := durations[0]
	maxDuration := durations[len(durations)-1]
	medianDuration := durations[len(durations)/2]
	avgDuration := totalReqDuration / time.Duration(len(m.requests))

	failureRate := float64(failedRequests) / float64(totalRequests)

	fmt.Printf("Total Requests: %d, RPS: %.2f /s\n", totalRequests, rps)
	fmt.Printf("Failed Requests: %d\n", failedRequests)
	fmt.Printf("Failure Rate: %.2f%%\n", failureRate*100)
	fmt.Printf("Total Data Sent: %d bytes\n", totalDataSent)
	fmt.Printf("Total Data Received: %d bytes\n", totalDataReceived)
	fmt.Printf("Duration min = %v avg = %v med = %v max = %v\n", minDuration, avgDuration, medianDuration, maxDuration)

	m.evaluateThresholds(failureRate, minDuration, maxDuration, durations)
}

// estimateRequestSize estimates the size of an HTTP request
func estimateRequestSize(req *http.Request) int64 {
	size := int64(0)
	size += int64(len(req.Method))
	size += int64(len(req.URL.String()))
	size += int64(len(req.Proto))
	for name, values := range req.Header {
		size += int64(len(name))
		for _, value := range values {
			size += int64(len(value))
		}
	}
	if req.Body != nil {
		body, _ := io.ReadAll(req.Body)
		size += int64(len(body))
		req.Body = io.NopCloser(bytes.NewBuffer(body))
	}
	return size
}

// estimateResponseSize estimates the size of an HTTP response
func estimateResponseSize(resp *http.Response) int64 {
	size := int64(0)
	size += int64(len(resp.Status))
	size += int64(len(resp.Proto))
	for name, values := range resp.Header {
		size += int64(len(name))
		for _, value := range values {
			size += int64(len(value))
		}
	}
	if resp.Body != nil {
		body, _ := io.ReadAll(resp.Body)
		size += int64(len(body))
		resp.Body = io.NopCloser(bytes.NewBuffer(body))
	}
	return size
}

// calculatePercentile calculates the value at a given percentile
func calculatePercentile(
	durations []time.Duration,
	percentile int,
) time.Duration {
	index := int((float64(percentile) / 100) * float64(len(durations)-1))
	return durations[index]
}

// evaluateThresholds evaluates the defined thresholds against the calculated metrics
func (m *Metrics) evaluateThresholds(
	failureRate float64,
	minDuration, maxDuration time.Duration,
	durations []time.Duration,
) {
	for key, conditions := range m.thresholds {
		for _, condition := range conditions {
			if key == "http_req_duration" {
				var percentile int
				var operator string
				var threshold int
				if _, err := fmt.Sscanf(condition, "p(%d) %s %d", &percentile, &operator, &threshold); err == nil {
					value := calculatePercentile(durations, percentile)
					m.evaluateCondition(fmt.Sprintf("http_req_duration p(%d)", percentile), value.Milliseconds(), operator, int64(threshold))
				} else if _, err := fmt.Sscanf(condition, "min %s %d", &operator, &threshold); err == nil {
					m.evaluateCondition("http_req_duration min", minDuration.Milliseconds(), operator, int64(threshold))
				} else if _, err := fmt.Sscanf(condition, "max %s %d", &operator, &threshold); err == nil {
					m.evaluateCondition("http_req_duration max", maxDuration.Milliseconds(), operator, int64(threshold))
				}
			} else if key == "http_req_failed" {
				var operator string
				var threshold float64
				if _, err := fmt.Sscanf(condition, "rate%s%f", &operator, &threshold); err == nil {
					m.evaluateCondition("http_req_failed rate", failureRate, operator, threshold)
				}
			}
		}
	}
}

// evaluateCondition evaluates a single condition against a metric
func (m *Metrics) evaluateCondition(
	metric string,
	value interface{},
	operator string,
	threshold interface{},
) {
	pass := false
	switch v := value.(type) {
	case int64:
		t := threshold.(int64)
		switch operator {
		case "<":
			pass = v < t
		case "<=":
			pass = v <= t
		case ">":
			pass = v > t
		case ">=":
			pass = v >= t
		case "==":
			pass = v == t
		}
	case float64:
		t := threshold.(float64)
		switch operator {
		case "<":
			pass = v < t
		case "<=":
			pass = v <= t
		case ">":
			pass = v > t
		case ">=":
			pass = v >= t
		case "==":
			pass = v == t
		}
	}

	if pass {
		fmt.Printf("%s %s %v: PASS (value: %v)\n", metric, operator, threshold, value)
	} else {
		fmt.Printf("%s %s %v: FAIL (value: %v)\n", metric, operator, threshold, value)
	}
}
