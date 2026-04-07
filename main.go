package main

import (
	"fmt"
	"os"
)

func main() {
	cfg, err := ParseConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}

	report, err := Run(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	if report.Text == "" {
		fmt.Fprintln(os.Stderr, "error: empty report")
		os.Exit(1)
	}

	fmt.Print(report.Text)
}
