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
	DNSStart, DNSDone                   time.Time
	ConnectStart, ConnectDone           time.Time
	TLSHandshakeStart, TLSHandshakeDone time.Time
	GotConn                             time.Time
	WroteHeaders                        time.Time
	WroteRequest                        time.Time
	GotFirstResponseByte                time.Time
	StartTime, EndTime                  time.Time
	Request                             *http.Request
	Response                            *http.Response
	Error                               error
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
	rps := float64(totalRequests) / totalDuration.Seconds()

	var failedRequests int64
	var totalReqDuration, totalDNS, totalConnect, totalTLS, totalTTFB time.Duration
	var totalDataSent, totalDataReceived int64
	durations := make([]time.Duration, 0, totalRequests)
	dnsDurations := make([]time.Duration, 0, totalRequests)
	connectDurations := make([]time.Duration, 0, totalRequests)
	tlsDurations := make([]time.Duration, 0, totalRequests)
	ttfbDurations := make([]time.Duration, 0, totalRequests)

	statusCodes := make(map[int]int)

	for _, req := range m.requests {
		duration := req.EndTime.Sub(req.StartTime)
		durations = append(durations, duration)
		totalReqDuration += duration

		if req.Error != nil || (req.Response != nil && req.Response.StatusCode >= 400) {
			failedRequests++
		}

		if req.Response != nil {
			statusCodes[req.Response.StatusCode]++
		}

		totalDataSent += estimateRequestSize(req.Request)
		if req.Response != nil {
			totalDataReceived += estimateResponseSize(req.Response)
		}

		if !req.DNSStart.IsZero() && !req.DNSDone.IsZero() {
			dnsDuration := req.DNSDone.Sub(req.DNSStart)
			dnsDurations = append(dnsDurations, dnsDuration)
			totalDNS += dnsDuration
		}
		if !req.ConnectStart.IsZero() && !req.ConnectDone.IsZero() {
			connectDuration := req.ConnectDone.Sub(req.ConnectStart)
			connectDurations = append(connectDurations, connectDuration)
			totalConnect += connectDuration
		}
		if !req.TLSHandshakeStart.IsZero() && !req.TLSHandshakeDone.IsZero() {
			tlsDuration := req.TLSHandshakeDone.Sub(req.TLSHandshakeStart)
			tlsDurations = append(tlsDurations, tlsDuration)
			totalTLS += tlsDuration
		}
		if !req.WroteRequest.IsZero() && !req.GotFirstResponseByte.IsZero() {
			ttfbDuration := req.GotFirstResponseByte.Sub(req.WroteRequest)
			ttfbDurations = append(ttfbDurations, ttfbDuration)
			totalTTFB += ttfbDuration
		}
	}

	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	sort.Slice(dnsDurations, func(i, j int) bool {
		return dnsDurations[i] < dnsDurations[j]
	})

	sort.Slice(connectDurations, func(i, j int) bool {
		return connectDurations[i] < connectDurations[j]
	})

	sort.Slice(tlsDurations, func(i, j int) bool {
		return tlsDurations[i] < tlsDurations[j]
	})

	sort.Slice(ttfbDurations, func(i, j int) bool {
		return ttfbDurations[i] < ttfbDurations[j]
	})

	minDuration := durations[0]
	maxDuration := durations[len(durations)-1]
	medianDuration := durations[len(durations)/2]
	avgDuration := totalReqDuration / time.Duration(totalRequests)

	minDNS, medianDNS, maxDNS, avgDNS := calculateMetrics(dnsDurations, totalDNS, totalRequests)
	minConnect, medianConnect, maxConnect, avgConnect := calculateMetrics(connectDurations, totalConnect, totalRequests)
	minTLS, medianTLS, maxTLS, avgTLS := calculateMetrics(tlsDurations, totalTLS, totalRequests)
	minTTFB, medianTTFB, maxTTFB, avgTTFB := calculateMetrics(ttfbDurations, totalTTFB, totalRequests)

	p90Index := int(float64(len(durations)) * 0.9)
	p95Index := int(float64(len(durations)) * 0.95)
	p99Index := int(float64(len(durations)) * 0.99)

	p90Duration := durations[p90Index]
	p95Duration := durations[p95Index]
	p99Duration := durations[p99Index]

	failureRate := float64(failedRequests) / float64(totalRequests)

	convertBytes := func(bytes int64) string {
		kb := float64(bytes) / 1024
		mb := kb / 1024
		if mb >= 1 {
			return fmt.Sprintf("%.2f MB", mb)
		} else if kb >= 1 {
			return fmt.Sprintf("%.2f KB", kb)
		}
		return fmt.Sprintf("%d bytes", bytes)
	}

	dataRateSent := int64(float64(totalDataSent) / totalDuration.Seconds())
	dataRateReceived := int64(float64(totalDataReceived) / totalDuration.Seconds())

	format := func(label string, value interface{}) string {
		return fmt.Sprintf("%-*s: %-*v", 25, label, 20, value)
	}

	fmt.Print(format("Total Requests", fmt.Sprintf("%d (%.2f/s)", totalRequests, rps)))
	fmt.Print("\n")
	fmt.Print(format("Data Sent", fmt.Sprintf("%s (%s/s)", convertBytes(totalDataSent), convertBytes(dataRateSent))))
	fmt.Print("\n")
	fmt.Print(format("Data Received", fmt.Sprintf("%s (%s/s)", convertBytes(totalDataReceived), convertBytes(dataRateReceived))))
	fmt.Print("\n")

	fmt.Print(format("HTTP Request Duration", fmt.Sprintf("min=%7.2fms, med=%7.2fms, max=%7.2fms, avg=%7.2fms",
		minDuration.Seconds()*1000, medianDuration.Seconds()*1000, maxDuration.Seconds()*1000, avgDuration.Seconds()*1000)))
	fmt.Print("\n")

	fmt.Print(format("Percentiles", fmt.Sprintf("90th=%7.2fms, 95th=%7.2fms, 99th=%7.2fms",
		p90Duration.Seconds()*1000, p95Duration.Seconds()*1000, p99Duration.Seconds()*1000)))
	fmt.Print("\n")

	fmt.Print(format("DNS Lookup", fmt.Sprintf("min=%7.2fms, med=%7.2fms, max=%7.2fms, avg=%7.2fms",
		minDNS.Seconds()*1000, medianDNS.Seconds()*1000, maxDNS.Seconds()*1000, avgDNS.Seconds()*1000)))
	fmt.Print("\n")

	fmt.Print(format("TCP Connect", fmt.Sprintf("min=%7.2fms, med=%7.2fms, max=%7.2fms, avg=%7.2fms",
		minConnect.Seconds()*1000, medianConnect.Seconds()*1000, maxConnect.Seconds()*1000, avgConnect.Seconds()*1000)))
	fmt.Print("\n")

	fmt.Print(format("TLS Handshake", fmt.Sprintf("min=%7.2fms, med=%7.2fms, max=%7.2fms, avg=%7.2fms",
		minTLS.Seconds()*1000, medianTLS.Seconds()*1000, maxTLS.Seconds()*1000, avgTLS.Seconds()*1000)))
	fmt.Print("\n")

	fmt.Print(format("Time to First Byte", fmt.Sprintf("min=%7.2fms, med=%7.2fms, max=%7.2fms, avg=%7.2fms",
		minTTFB.Seconds()*1000, medianTTFB.Seconds()*1000, maxTTFB.Seconds()*1000, avgTTFB.Seconds()*1000)))
	fmt.Print("\n")

	fmt.Printf("Status Code Distribution:\n")
	var sortedStatusCodes []int
	for code := range statusCodes {
		sortedStatusCodes = append(sortedStatusCodes, code)
	}
	sort.Ints(sortedStatusCodes)
	for _, code := range sortedStatusCodes {
		fmt.Printf("  %d: %d (%.2f%%)\n", code, statusCodes[code], float64(statusCodes[code])/float64(totalRequests)*100)
	}
	fmt.Println()
	fmt.Println("Threshold Evaluation:")
	m.evaluateThresholds(failureRate, minDuration, maxDuration, durations)
}

// calculateMetrics calculates the min, median, max, and avg values of a slice of durations
func calculateMetrics(
	durations []time.Duration,
	total time.Duration,
	count int64,
) (min, median, max, avg time.Duration) {
	if len(durations) == 0 {
		return 0, 0, 0, 0
	}
	min = durations[0]
	max = durations[len(durations)-1]
	median = durations[len(durations)/2]
	avg = total / time.Duration(count)
	return
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
