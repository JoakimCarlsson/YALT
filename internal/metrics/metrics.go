package metrics

import (
	"sort"
	"sync"
	"time"
)

type Metrics struct {
	HttpReqDuration []time.Duration
	mu              sync.Mutex
}

var instance *Metrics
var once sync.Once

func GetMetrics() *Metrics {
	once.Do(func() {
		instance = &Metrics{}
	})
	return instance
}

func (m *Metrics) AddHttpReqDuration(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.HttpReqDuration = append(m.HttpReqDuration, duration)
}

func (m *Metrics) CalculateMetrics() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	metrics := make(map[string]interface{})
	if len(m.HttpReqDuration) > 0 {
		metrics["http_req_duration_p99"] = calculatePercentile(m.HttpReqDuration, 99)
	}
	return metrics
}

func calculatePercentile(
	durations []time.Duration,
	percentile int,
) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})
	index := int(float64(percentile) / 100.0 * float64(len(durations)))
	if index >= len(durations) {
		index = len(durations) - 1
	}
	return durations[index]
}
