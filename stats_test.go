package main

import (
	"testing"
	"time"
)

func TestPercentile(t *testing.T) {
	values := []time.Duration{10 * time.Millisecond, 20 * time.Millisecond, 30 * time.Millisecond, 40 * time.Millisecond, 50 * time.Millisecond}

	if got := Percentile(values, 0); got != 10*time.Millisecond {
		t.Fatalf("p0 = %s, want %s", got, 10*time.Millisecond)
	}

	if got := Percentile(values, 50); got != 30*time.Millisecond {
		t.Fatalf("p50 = %s, want %s", got, 30*time.Millisecond)
	}

	if got := Percentile(values, 95); got != 40*time.Millisecond {
		t.Fatalf("p95 = %s, want %s", got, 40*time.Millisecond)
	}

	if got := Percentile(values, 100); got != 50*time.Millisecond {
		t.Fatalf("p100 = %s, want %s", got, 50*time.Millisecond)
	}
}

func TestPercentileEmpty(t *testing.T) {
	if got := Percentile(nil, 95); got != 0 {
		t.Fatalf("empty percentile = %s, want 0", got)
	}
}

func TestCollectResults(t *testing.T) {
	results := make(chan Result, 3)
	results <- Result{StatusCode: 200, Latency: 10 * time.Millisecond, Bytes: 100}
	results <- Result{StatusCode: 500, Latency: 20 * time.Millisecond, Bytes: 50}
	results <- Result{Latency: 30 * time.Millisecond, Err: "timeout"}
	close(results)

	startedAt := time.Now().Add(-time.Second)
	stats := CollectResults(results, startedAt)

	if stats.TotalRequests != 3 {
		t.Fatalf("TotalRequests = %d, want 3", stats.TotalRequests)
	}
	if stats.Successes != 1 {
		t.Fatalf("Successes = %d, want 1", stats.Successes)
	}
	if stats.Failures != 2 {
		t.Fatalf("Failures = %d, want 2", stats.Failures)
	}
	if stats.Errors != 1 {
		t.Fatalf("Errors = %d, want 1", stats.Errors)
	}
	if stats.BytesRead != 150 {
		t.Fatalf("BytesRead = %d, want 150", stats.BytesRead)
	}
	if stats.StatusCodes[200] != 1 || stats.StatusCodes[500] != 1 {
		t.Fatalf("unexpected status codes: %#v", stats.StatusCodes)
	}
	if stats.ErrorMessages["timeout"] != 1 {
		t.Fatalf("unexpected error messages: %#v", stats.ErrorMessages)
	}
	if stats.MinLatency != 10*time.Millisecond {
		t.Fatalf("MinLatency = %s, want %s", stats.MinLatency, 10*time.Millisecond)
	}
	if stats.MaxLatency != 30*time.Millisecond {
		t.Fatalf("MaxLatency = %s, want %s", stats.MaxLatency, 30*time.Millisecond)
	}
	if stats.AvgLatency != 20*time.Millisecond {
		t.Fatalf("AvgLatency = %s, want %s", stats.AvgLatency, 20*time.Millisecond)
	}
	if stats.P50Latency != 20*time.Millisecond {
		t.Fatalf("P50Latency = %s, want %s", stats.P50Latency, 20*time.Millisecond)
	}
	if stats.P95Latency != 20*time.Millisecond {
		t.Fatalf("P95Latency = %s, want %s", stats.P95Latency, 20*time.Millisecond)
	}
	if stats.P99Latency != 20*time.Millisecond {
		t.Fatalf("P99Latency = %s, want %s", stats.P99Latency, 20*time.Millisecond)
	}
	if stats.Duration <= 0 {
		t.Fatalf("Duration = %s, want > 0", stats.Duration)
	}
	if stats.QPS <= 0 {
		t.Fatalf("QPS = %f, want > 0", stats.QPS)
	}
}
