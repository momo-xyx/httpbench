package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type headerFlags []string

func (h *headerFlags) String() string {
	return strings.Join(*h, ",")
}

func (h *headerFlags) Set(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("header cannot be empty")
	}
	*h = append(*h, value)
	return nil
}

type Config struct {
	URL         string
	Method      string
	Concurrency int
	Total       int
	Duration    time.Duration
	Rate        int
	Headers     http.Header
	Body        string
	BodyFile    string
	Output      string
	Timeout     time.Duration
}

func ParseConfig() (Config, error) {
	var cfg Config
	var rawHeaders headerFlags

	flag.StringVar(&cfg.URL, "url", "", "target URL")
	flag.StringVar(&cfg.Method, "method", http.MethodGet, "HTTP method")
	flag.IntVar(&cfg.Concurrency, "c", 50, "number of concurrent workers")
	flag.IntVar(&cfg.Total, "n", 0, "total number of requests")
	flag.DurationVar(&cfg.Duration, "d", 0, "benchmark duration (e.g. 30s)")
	flag.IntVar(&cfg.Rate, "rate", 0, "maximum requests per second (0 = unlimited)")
	flag.Var(&rawHeaders, "H", "custom header, can be specified multiple times")
	flag.StringVar(&cfg.Body, "body", "", "inline request body")
	flag.StringVar(&cfg.BodyFile, "body-file", "", "path to request body file")
	flag.StringVar(&cfg.Output, "o", "text", "output format: text or json")
	flag.DurationVar(&cfg.Timeout, "timeout", 30*time.Second, "per-request timeout")
	flag.Parse()

	cfg.Method = strings.ToUpper(strings.TrimSpace(cfg.Method))
	cfg.Output = strings.ToLower(strings.TrimSpace(cfg.Output))

	headers, err := parseHeaders(rawHeaders)
	if err != nil {
		return Config{}, err
	}
	cfg.Headers = headers

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (cfg Config) Validate() error {
	if strings.TrimSpace(cfg.URL) == "" {
		return fmt.Errorf("-url is required")
	}

	if cfg.Concurrency <= 0 {
		return fmt.Errorf("-c must be greater than 0")
	}

	if cfg.Total < 0 {
		return fmt.Errorf("-n must be greater than or equal to 0")
	}

	if cfg.Duration < 0 {
		return fmt.Errorf("-d must be greater than or equal to 0")
	}

	if (cfg.Total == 0 && cfg.Duration == 0) || (cfg.Total > 0 && cfg.Duration > 0) {
		return fmt.Errorf("specify exactly one of -n or -d")
	}

	if cfg.Rate < 0 {
		return fmt.Errorf("-rate must be greater than or equal to 0")
	}

	if cfg.Output != "text" && cfg.Output != "json" {
		return fmt.Errorf("-o must be either text or json")
	}

	if cfg.Body != "" && cfg.BodyFile != "" {
		return fmt.Errorf("-body and -body-file cannot be used together")
	}

	if cfg.Timeout <= 0 {
		return fmt.Errorf("-timeout must be greater than 0")
	}

	return nil
}

func (cfg Config) RequestBodyBytes() ([]byte, error) {
	if cfg.Body != "" {
		return []byte(cfg.Body), nil
	}

	if cfg.BodyFile == "" {
		return nil, nil
	}

	data, err := os.ReadFile(cfg.BodyFile)
	if err != nil {
		return nil, fmt.Errorf("read body file: %w", err)
	}

	return data, nil
}

func parseHeaders(values []string) (http.Header, error) {
	headers := make(http.Header)

	for _, value := range values {
		parts := strings.SplitN(value, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid header %q, expected Key: Value", value)
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if key == "" {
			return nil, fmt.Errorf("invalid header %q, key cannot be empty", value)
		}

		headers.Add(key, val)
	}

	return headers, nil
}
