package metrics

import (
	"log"
	"sync"
	"time"
)

type Metrics struct {
	mu               sync.RWMutex
	TotalRequests    int
	FailedRequests   int
	RequestDurations []time.Duration
}

var metrics = &Metrics{
	RequestDurations: make([]time.Duration, 0),
}

func AddRequest(duration time.Duration, success bool) {
	metrics.mu.Lock()
	defer metrics.mu.Unlock()
	metrics.TotalRequests++
	if !success {
		metrics.FailedRequests++
	}
	metrics.RequestDurations = append(metrics.RequestDurations, duration)
}

func TrackMetrics() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			metrics.mu.RLock()
			log.Printf("Total Requests: %d, Failed Requests: %d", metrics.TotalRequests, metrics.FailedRequests)
			metrics.mu.RUnlock()
		}
	}
}

func CalculatePercentile(p float64) time.Duration {
	metrics.mu.RLock()
	defer metrics.mu.RUnlock()
	if len(metrics.RequestDurations) == 0 {
		return 0
	}
	n := int(float64(len(metrics.RequestDurations)) * p / 100.0)
	return metrics.RequestDurations[n]
}

func CheckThresholds(thresholds map[string][]string) {
	for metric, conditions := range thresholds {
		switch metric {
		case "http_req_duration":
			for _, condition := range conditions {
				if condition == "p(99) < 3000" {
					percentile := CalculatePercentile(99)
					if percentile > 3000*time.Millisecond {
						log.Printf("Threshold failed: %s, value: %v", condition, percentile)
					} else {
						log.Printf("Threshold passed: %s, value: %v", condition, percentile)
					}
				}
			}
		}
	}
}

func Summary() {
	metrics.mu.RLock()
	defer metrics.mu.RUnlock()
	log.Printf("Final Summary - Total Requests: %d, Failed Requests: %d", metrics.TotalRequests, metrics.FailedRequests)
}
