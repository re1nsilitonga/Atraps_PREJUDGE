package main

import (
	"fmt"

	"prejudge/core/layer1"
)

type Result struct {
	N      int // Layer 2 confirmations
	M      int // Layer 1 preemptive catches
	Misses []string
}

// Run computes the cold-start proof numbers (PJ-703): N confirmations
// bought M preemptive catches, with individual misses logged.
//
// confirmed is the L2-confirmed domain set (core.ConfirmedDomains.ListConfirmed).
// candidates is the pool of domains to test as potential Layer 1 catches —
// domains nobody has visited, never run through Layer 2.
//
// The leakage assertion is the whole ticket (PRD §14, TASKS.md PJ-703): a
// domain Layer 2 already confirmed can never be counted as a Layer 1 catch,
// or the ratio is a fabricated number. This is checked in code, not assumed
// true of the candidate list — any overlap aborts the run with an error
// rather than silently producing a subtly-wrong count.
func Run(confirmed, candidates []string, extract func(domain string) (layer1.Fingerprint, error), clusters []layer1.Cluster) (Result, error) {
	confirmedSet := make(map[string]bool, len(confirmed))
	for _, d := range confirmed {
		confirmedSet[d] = true
	}

	result := Result{N: len(confirmed)}
	for _, candidate := range candidates {
		if confirmedSet[candidate] {
			return Result{}, fmt.Errorf(
				"leakage: candidate %q is already L2-confirmed — candidate lists must only contain domains nobody has visited",
				candidate,
			)
		}

		fp, err := extract(candidate)
		if err != nil {
			result.Misses = append(result.Misses, candidate)
			continue
		}

		if layer1.Match(fp, clusters) != nil {
			result.M++
		} else {
			result.Misses = append(result.Misses, candidate)
		}
	}

	return result, nil
}
