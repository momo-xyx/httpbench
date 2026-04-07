package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"golang.org/x/time/rate"
)

type progressState struct {
	completed atomic.Int64
	errors    atomic.Int64
}

type progressConfig struct {
	totalRequests int
	duration      time.Duration
}

func Run(cfg Config) (Report, error) {
	body, err := cfg.RequestBodyBytes()
	if err != nil {
		return Report{}, err
	}

	client := NewHTTPClient(cfg)
	defer closeIdleConnections(client)

	startedAt := time.Now()
	baseCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	requestCtx := baseCtx
	if cfg.Duration > 0 {
		var cancel context.CancelFunc
		requestCtx, cancel = context.WithTimeout(baseCtx, cfg.Duration)
		defer cancel()
	}

	results := make(chan Result, cfg.Concurrency*4)
	statsCh := make(chan Stats, 1)
	go func() {
		statsCh <- CollectResults(results, startedAt)
	}()

	var limiter *rate.Limiter
	if cfg.Rate > 0 {
		limiter = rate.NewLimiter(rate.Limit(cfg.Rate), 1)
	}

	var progress progressState
	progressDone := make(chan struct{})
	go printProgress(startedAt, progressConfig{totalRequests: cfg.Total, duration: cfg.Duration}, &progress, progressDone)

	var counter atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < cfg.Concurrency; i++ {
		wg.Add(1)
		go worker(requestCtx, client, cfg, body, limiter, &counter, &progress, results, &wg)
	}

	wg.Wait()
	close(results)
	stats := <-statsCh
	close(progressDone)

	return BuildReport(stats, cfg.Output)
}

func worker(
	ctx context.Context,
	client *http.Client,
	cfg Config,
	body []byte,
	limiter *rate.Limiter,
	counter *atomic.Int64,
	progress *progressState,
	results chan<- Result,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	for {
		if ctx.Err() != nil {
			return
		}

		if cfg.Total > 0 {
			next := int(counter.Add(1))
			if next > cfg.Total {
				return
			}
		}

		if limiter != nil {
			if err := limiter.Wait(ctx); err != nil {
				return
			}
		}

		result := doRequest(client, cfg, body)
		results <- result
		progress.completed.Add(1)
		if result.Err != "" {
			progress.errors.Add(1)
		}
	}
}

func doRequest(client *http.Client, cfg Config, body []byte) Result {
	var reader io.Reader
	if len(body) > 0 {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(cfg.Method, cfg.URL, reader)
	if err != nil {
		return Result{Err: err.Error()}
	}

	for key, values := range cfg.Headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start)
	if err != nil {
		return Result{Latency: latency, Err: err.Error()}
	}
	defer resp.Body.Close()

	written, readErr := drainBody(resp.Body)
	if readErr != nil {
		return Result{StatusCode: resp.StatusCode, Latency: latency, Bytes: written, Err: readErr.Error()}
	}

	return Result{
		StatusCode: resp.StatusCode,
		Latency:    latency,
		Bytes:      written,
	}
}

func drainBody(body io.Reader) (int64, error) {
	written, err := io.Copy(io.Discard, body)
	if err != nil {
		return written, fmt.Errorf("read response body: %w", err)
	}
	return written, nil
}

func printProgress(startedAt time.Time, cfg progressConfig, progress *progressState, done <-chan struct{}) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			if progress.completed.Load() > 0 {
				fmt.Fprintln(os.Stderr)
			}
			return
		case <-ticker.C:
			completed := progress.completed.Load()
			elapsed := time.Since(startedAt)
			qps := 0.0
			if elapsed > 0 {
				qps = float64(completed) / elapsed.Seconds()
			}

			if cfg.totalRequests > 0 {
				percent := float64(completed) / float64(cfg.totalRequests) * 100
				if percent > 100 {
					percent = 100
				}
				fmt.Fprintf(os.Stderr, "\rProgress: %d/%d (%.1f%%) errors=%d qps=%.2f elapsed=%s", completed, cfg.totalRequests, percent, progress.errors.Load(), qps, elapsed.Truncate(time.Millisecond))
				continue
			}

			remaining := time.Duration(0)
			if cfg.duration > elapsed {
				remaining = cfg.duration - elapsed
			}
			fmt.Fprintf(os.Stderr, "\rProgress: completed=%d errors=%d qps=%.2f elapsed=%s remaining=%s", completed, progress.errors.Load(), qps, elapsed.Truncate(time.Millisecond), remaining.Truncate(time.Millisecond))
		}
	}
}

func closeIdleConnections(client *http.Client) {
	if transport, ok := client.Transport.(interface{ CloseIdleConnections() }); ok {
		transport.CloseIdleConnections()
	}
}
