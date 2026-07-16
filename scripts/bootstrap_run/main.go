// Command bootstrap_run produces the cold-start proof (PJ-703): N Layer 2
// confirmations → M Layer 1 preemptive catches on domains nobody has
// visited, with individual misses. Writes the result to bootstrap_runs.
//
// Usage: go run ./scripts/bootstrap_run candidates.txt
//
// candidates.txt lists domains nobody has run through Layer 2 — curated,
// plausibly-related judol domains (PJ-801). The script fails loudly if any
// candidate turns out to already be L2-confirmed (the leakage assertion);
// fix the candidate list, don't ignore the error.
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"prejudge/core/layer1"
	"prejudge/db"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: bootstrap_run <candidates-file>")
		os.Exit(1)
	}
	candidates, err := readLines(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "db connect:", err)
		os.Exit(1)
	}
	defer pool.Close()

	domainRepo := db.NewDomainRepository(pool)
	clusterRepo := db.NewClusterRepository(pool)

	confirmed, err := domainRepo.ListConfirmed(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "list confirmed:", err)
		os.Exit(1)
	}

	clusters, err := clusterRepo.ListClusters(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "list clusters:", err)
		os.Exit(1)
	}

	extractor := layer1.NewExtractor()
	result, err := Run(confirmed, candidates, func(domain string) (layer1.Fingerprint, error) {
		return extractor.Extract(ctx, domain)
	}, clusters)
	if err != nil {
		fmt.Fprintln(os.Stderr, "LEAKAGE ERROR — fix the candidate list, do not ignore this:", err)
		os.Exit(1)
	}

	notes := fmt.Sprintf("candidates=%d misses=%s", len(candidates), strings.Join(result.Misses, ","))
	if err := domainRepo.RecordBootstrapRun(ctx, result.N, result.M, len(result.Misses), notes); err != nil {
		fmt.Fprintln(os.Stderr, "record bootstrap run:", err)
		os.Exit(1)
	}

	fmt.Printf("Cold-start proof: N=%d L2 confirmations, M=%d Layer 1 preemptive catches, %d misses\n",
		result.N, result.M, len(result.Misses))
	if len(result.Misses) > 0 {
		fmt.Println("Misses (candidates that matched no cluster):")
		for _, m := range result.Misses {
			fmt.Println("  -", m)
		}
	}
	if result.N > 0 {
		fmt.Printf("Ratio: %.2f (%d/%d) — report this honestly, do not tune it (PRD §14 risk #9)\n",
			float64(result.M)/float64(result.N), result.M, result.N)
	}
}

func readLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if line := strings.TrimSpace(scanner.Text()); line != "" {
			lines = append(lines, line)
		}
	}
	return lines, scanner.Err()
}
