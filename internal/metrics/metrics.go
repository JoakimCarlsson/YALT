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

// NewMetrics creates a new Metrics instance with the given thresholds
func NewMetrics(thresholds map[string][]string) *Metrics {
	return &Metrics{
		thresholds: thresholds,
	}
}

// Start starts the timer for the Metrics instance
func (m *Metrics) Start() {
	m.startTime = time.Now()
}

// AddRequestMetrics adds a new RequestMetrics instance to the Metrics instance
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

	avgDuration := totalDuration / time.Duration(totalRequests)
	failureRate := float64(failedRequests) / float64(totalRequests)

	fmt.Printf("Total Requests: %d\n", totalRequests)
	fmt.Printf("Failed Requests: %d\n", failedRequests)
	fmt.Printf("Failure Rate: %.2f%%\n", failureRate*100)
	fmt.Printf("Average Request Duration: %s\n", avgDuration)

	percentiles := m.extractPercentilesFromThresholds()
	for _, percentile := range percentiles {
		value := calculatePercentile(durations, percentile)
		fmt.Printf("%dth Percentile Request Duration: %s\n", percentile, value)
	}
}

func (m *Metrics) extractPercentilesFromThresholds() []int {
	percentileSet := make(map[int]struct{})
	for key := range m.thresholds {
		if key == "http_req_duration" {
			for _, condition := range m.thresholds[key] {
				var percentile int
				if _, err := fmt.Sscanf(condition, "p(%d)", &percentile); err == nil {
					percentileSet[percentile] = struct{}{}
				}
			}
		}
	}
	var percentiles []int
	for percentile := range percentileSet {
		percentiles = append(percentiles, percentile)
	}
	sort.Ints(percentiles)
	return percentiles
}

func calculatePercentile(durations []time.Duration, percentile int) time.Duration {
	index := int((float64(percentile) / 100) * float64(len(durations)-1))
	return durations[index]
}
