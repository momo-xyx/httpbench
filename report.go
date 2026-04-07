package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

type Report struct {
	Stats Stats
	Text  string
}

type jsonStats struct {
	TotalRequests int            `json:"total_requests"`
	Successes     int            `json:"successes"`
	Failures      int            `json:"failures"`
	Errors        int            `json:"errors"`
	BytesRead     int64          `json:"bytes_read"`
	Duration      string         `json:"duration"`
	QPS           float64        `json:"qps"`
	MinLatency    string         `json:"min_latency"`
	MaxLatency    string         `json:"max_latency"`
	AvgLatency    string         `json:"avg_latency"`
	P50Latency    string         `json:"p50_latency"`
	P95Latency    string         `json:"p95_latency"`
	P99Latency    string         `json:"p99_latency"`
	StatusCodes   map[int]int    `json:"status_codes"`
	ErrorMessages map[string]int `json:"error_messages,omitempty"`
}

func BuildReport(stats Stats, output string) (Report, error) {
	switch output {
	case "json":
		text, err := FormatJSON(stats)
		if err != nil {
			return Report{}, err
		}
		return Report{Stats: stats, Text: string(text)}, nil
	default:
		return Report{Stats: stats, Text: FormatText(stats)}, nil
	}
}

func FormatText(stats Stats) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Requests:\t%d\n", stats.TotalRequests)
	fmt.Fprintf(&b, "Successes:\t%d\n", stats.Successes)
	fmt.Fprintf(&b, "Failures:\t%d\n", stats.Failures)
	fmt.Fprintf(&b, "Errors:\t\t%d\n", stats.Errors)
	fmt.Fprintf(&b, "Duration:\t%s\n", formatDuration(stats.Duration))
	fmt.Fprintf(&b, "QPS:\t\t%.2f\n", stats.QPS)
	fmt.Fprintf(&b, "Bytes:\t\t%d\n", stats.BytesRead)
	fmt.Fprintf(&b, "Latency Avg:\t%s\n", formatDuration(stats.AvgLatency))
	fmt.Fprintf(&b, "Latency Min:\t%s\n", formatDuration(stats.MinLatency))
	fmt.Fprintf(&b, "Latency Max:\t%s\n", formatDuration(stats.MaxLatency))
	fmt.Fprintf(&b, "P50:\t\t%s\n", formatDuration(stats.P50Latency))
	fmt.Fprintf(&b, "P95:\t\t%s\n", formatDuration(stats.P95Latency))
	fmt.Fprintf(&b, "P99:\t\t%s\n", formatDuration(stats.P99Latency))

	if len(stats.StatusCodes) > 0 {
		codes := make([]int, 0, len(stats.StatusCodes))
		for code := range stats.StatusCodes {
			codes = append(codes, code)
		}
		sort.Ints(codes)
		b.WriteString("Status Codes:\n")
		for _, code := range codes {
			fmt.Fprintf(&b, "  %d:\t%d\n", code, stats.StatusCodes[code])
		}
	}

	if len(stats.ErrorMessages) > 0 {
		messages := make([]string, 0, len(stats.ErrorMessages))
		for message := range stats.ErrorMessages {
			messages = append(messages, message)
		}
		sort.Strings(messages)
		b.WriteString("Errors:\n")
		for _, message := range messages {
			fmt.Fprintf(&b, "  %s:\t%d\n", message, stats.ErrorMessages[message])
		}
	}

	return b.String()
}

func FormatJSON(stats Stats) ([]byte, error) {
	payload := jsonStats{
		TotalRequests: stats.TotalRequests,
		Successes:     stats.Successes,
		Failures:      stats.Failures,
		Errors:        stats.Errors,
		BytesRead:     stats.BytesRead,
		Duration:      formatDuration(stats.Duration),
		QPS:           stats.QPS,
		MinLatency:    formatDuration(stats.MinLatency),
		MaxLatency:    formatDuration(stats.MaxLatency),
		AvgLatency:    formatDuration(stats.AvgLatency),
		P50Latency:    formatDuration(stats.P50Latency),
		P95Latency:    formatDuration(stats.P95Latency),
		P99Latency:    formatDuration(stats.P99Latency),
		StatusCodes:   stats.StatusCodes,
		ErrorMessages: stats.ErrorMessages,
	}

	return json.MarshalIndent(payload, "", "  ")
}

func formatDuration(value time.Duration) string {
	if value == 0 {
		return "0s"
	}
	return value.String()
}
