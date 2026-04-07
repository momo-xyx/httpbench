package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunFixedCountGET(t *testing.T) {
	var hits atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	cfg := Config{
		URL:         server.URL,
		Method:      http.MethodGet,
		Concurrency: 4,
		Total:       25,
		Headers:     make(http.Header),
		Output:      "text",
		Timeout:     5 * time.Second,
	}

	report, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got := int(hits.Load()); got != 25 {
		t.Fatalf("server hits = %d, want 25", got)
	}
	if report.Stats.TotalRequests != 25 {
		t.Fatalf("TotalRequests = %d, want 25", report.Stats.TotalRequests)
	}
	if report.Stats.Successes != 25 {
		t.Fatalf("Successes = %d, want 25", report.Stats.Successes)
	}
	if report.Stats.Failures != 0 {
		t.Fatalf("Failures = %d, want 0", report.Stats.Failures)
	}
	if report.Stats.Errors != 0 {
		t.Fatalf("Errors = %d, want 0", report.Stats.Errors)
	}
	if report.Text == "" {
		t.Fatal("report text is empty")
	}
}

func TestRunRateLimitedDuration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	cfg := Config{
		URL:         server.URL,
		Method:      http.MethodGet,
		Concurrency: 8,
		Duration:    2 * time.Second,
		Rate:        40,
		Headers:     make(http.Header),
		Output:      "text",
		Timeout:     5 * time.Second,
	}

	report, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if report.Stats.TotalRequests < 75 || report.Stats.TotalRequests > 82 {
		t.Fatalf("TotalRequests = %d, want about 80", report.Stats.TotalRequests)
	}
	if report.Stats.Errors != 0 {
		t.Fatalf("Errors = %d, want 0", report.Stats.Errors)
	}
	if report.Stats.Failures != 0 {
		t.Fatalf("Failures = %d, want 0", report.Stats.Failures)
	}
}

func TestRunPOSTJSONOutput(t *testing.T) {
	var hits atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("content type = %q, want application/json", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		defer r.Body.Close()
		if string(body) != `{"hello":"world"}` {
			t.Fatalf("body = %q, want %q", string(body), `{"hello":"world"}`)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"created":true}`))
	}))
	defer server.Close()

	cfg := Config{
		URL:         server.URL,
		Method:      http.MethodPost,
		Concurrency: 3,
		Total:       12,
		Headers:     http.Header{"Content-Type": []string{"application/json"}},
		Body:        `{"hello":"world"}`,
		Output:      "json",
		Timeout:     5 * time.Second,
	}

	report, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got := int(hits.Load()); got != 12 {
		t.Fatalf("server hits = %d, want 12", got)
	}
	if report.Stats.TotalRequests != 12 {
		t.Fatalf("TotalRequests = %d, want 12", report.Stats.TotalRequests)
	}
	if report.Stats.Successes != 12 {
		t.Fatalf("Successes = %d, want 12", report.Stats.Successes)
	}
	if report.Stats.StatusCodes[http.StatusCreated] != 12 {
		t.Fatalf("status 201 count = %d, want 12", report.Stats.StatusCodes[http.StatusCreated])
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(report.Text), &payload); err != nil {
		t.Fatalf("json output unmarshal error = %v", err)
	}
	if got := int(payload["total_requests"].(float64)); got != 12 {
		t.Fatalf("json total_requests = %d, want 12", got)
	}
}
