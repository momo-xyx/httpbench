package main

import (
	"sort"
	"time"
)

type Result struct {
	StatusCode int
	Latency    time.Duration
	Bytes      int64
	Err        string
}

type Stats struct {
	TotalRequests int            `json:"total_requests"`
	Successes     int            `json:"successes"`
	Failures      int            `json:"failures"`
	Errors        int            `json:"errors"`
	BytesRead     int64          `json:"bytes_read"`
	Duration      time.Duration  `json:"duration"`
	QPS           float64        `json:"qps"`
	MinLatency    time.Duration  `json:"min_latency"`
	MaxLatency    time.Duration  `json:"max_latency"`
	AvgLatency    time.Duration  `json:"avg_latency"`
	P50Latency    time.Duration  `json:"p50_latency"`
	P95Latency    time.Duration  `json:"p95_latency"`
	P99Latency    time.Duration  `json:"p99_latency"`
	StatusCodes   map[int]int    `json:"status_codes"`
	ErrorMessages map[string]int `json:"error_messages,omitempty"`
}

type Snapshot struct {
	Elapsed   time.Duration
	Completed int
	Errors    int
	QPS       float64
}

func CollectResults(results <-chan Result, startedAt time.Time) Stats {
	stats := Stats{
		StatusCodes:   make(map[int]int),
		ErrorMessages: make(map[string]int),
	}

	latencies := make([]time.Duration, 0, 1024)
	var latencySum time.Duration

	for result := range results {
		stats.TotalRequests++
		stats.BytesRead += result.Bytes

		if result.Err != "" {
			stats.Errors++
			stats.Failures++
			stats.ErrorMessages[result.Err]++
		} else {
			stats.StatusCodes[result.StatusCode]++
			if result.StatusCode >= 200 && result.StatusCode < 400 {
				stats.Successes++
			} else {
				stats.Failures++
			}
		}

		if result.Latency > 0 {
			latencies = append(latencies, result.Latency)
			latencySum += result.Latency
			if stats.MinLatency == 0 || result.Latency < stats.MinLatency {
				stats.MinLatency = result.Latency
			}
			if result.Latency > stats.MaxLatency {
				stats.MaxLatency = result.Latency
			}
		}
	}

	stats.Duration = time.Since(startedAt)
	if stats.Duration > 0 {
		stats.QPS = float64(stats.TotalRequests) / stats.Duration.Seconds()
	}

	if len(latencies) == 0 {
		return stats
	}

	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	stats.AvgLatency = latencySum / time.Duration(len(latencies))
	stats.P50Latency = Percentile(latencies, 50)
	stats.P95Latency = Percentile(latencies, 95)
	stats.P99Latency = Percentile(latencies, 99)

	return stats
}

func Percentile(sorted []time.Duration, p int) time.Duration {
	if len(sorted) == 0 {
		return 0
	}

	if p <= 0 {
		return sorted[0]
	}

	if p >= 100 {
		return sorted[len(sorted)-1]
	}

	idx := (len(sorted) - 1) * p / 100
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}

	return sorted[idx]
}
