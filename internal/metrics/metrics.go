package metrics

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

type Metrics struct {
	mu         sync.Mutex
	requests   []RequestMetrics
	thresholds map[string][]string
	startTime  time.Time
}

type RequestMetrics struct {
	Duration time.Duration
	Failed   bool
}

// NewMetrics creates a new Metrics instance
func NewMetrics(thresholds map[string][]string) *Metrics {
	return &Metrics{
		thresholds: thresholds,
	}
}

// AddRequestMetrics adds a request metric to the metrics
func (m *Metrics) AddRequestMetrics(duration time.Duration, failed bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requests = append(m.requests, RequestMetrics{Duration: duration, Failed: failed})
}

// CalculateAndDisplayMetrics calculates and displays the metrics
func (m *Metrics) CalculateAndDisplayMetrics() {
	m.mu.Lock()
	defer m.mu.Unlock()

	totalRequests := len(m.requests)
	failedRequests := 0
	var totalDuration time.Duration

	durations := make([]time.Duration, 0, totalRequests)

	for _, req := range m.requests {
		if req.Failed {
			failedRequests++
		}
		durations = append(durations, req.Duration)
		totalDuration += req.Duration
	}

	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	failureRate := float64(failedRequests) / float64(totalRequests)

	m.evaluateThresholds(failureRate, durations)
}

// calculatePercentile calculates the value at a given percentile in a slice of durations
func calculatePercentile(durations []time.Duration, percentile int) time.Duration {
	index := int((float64(percentile) / 100) * float64(len(durations)-1))
	return durations[index]
}

// evaluateThresholds evaluates the thresholds defined in the metrics configuration
func (m *Metrics) evaluateThresholds(failureRate float64, durations []time.Duration) {
	for key, conditions := range m.thresholds {
		for _, condition := range conditions {
			if key == "http_req_duration" {
				var percentile int
				var operator string
				var threshold int
				if _, err := fmt.Sscanf(condition, "p(%d) %s %d", &percentile, &operator, &threshold); err == nil {
					value := calculatePercentile(durations, percentile)
					m.evaluateCondition(fmt.Sprintf("http_req_duration p(%d)", percentile), value.Milliseconds(), operator, int64(threshold))
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

// evaluateCondition evaluates a condition based on a metric, an operator and a threshold
func (m *Metrics) evaluateCondition(metric string, value interface{}, operator string, threshold interface{}) {
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
