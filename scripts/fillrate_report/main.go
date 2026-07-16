// Command fillrate_report runs Layer 1 fingerprint extraction over a list of
// domains and prints the per-field fill rate — the PJ-401 T+5 gate.
//
// Usage: go run ./scripts/fillrate_report domains.txt
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"prejudge/core/layer1"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: fillrate_report <domains-file>")
		os.Exit(1)
	}
	domains, err := readDomains(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	extractor := layer1.NewExtractor()
	ctx := context.Background()
	fingerprints := make([]layer1.Fingerprint, 0, len(domains))
	for _, domain := range domains {
		fp, err := extractor.Extract(ctx, domain)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skip %s: %v\n", domain, err)
			continue
		}
		fingerprints = append(fingerprints, fp)
	}

	fmt.Printf("Fill rate over %d domains:\n", len(fingerprints))
	for field, rate := range layer1.FillRate(fingerprints) {
		fmt.Printf("  %-15s %.0f%%\n", field, rate*100)
	}
}

func readDomains(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var domains []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if line := scanner.Text(); line != "" {
			domains = append(domains, line)
		}
	}
	return domains, scanner.Err()
}
