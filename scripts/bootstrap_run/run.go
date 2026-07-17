package main

import (
	"fmt"

	"prime/core/layer1"
)

type Result struct {
	N      int
	M      int
	Misses []string
}

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
