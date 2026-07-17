package layer1

const matchThreshold = 0.6

var matchWeights = map[string]float64{
	"hosting_ip":         0.30,
	"nameserver":         0.25,
	"registration_burst": 0.25,
	"registrar":          0.10,
	"tld":                0.10,
}

type MatchResult struct {
	ClusterID     string
	Score         float64
	MatchedFields []string
}

func Match(fp Fingerprint, clusters []Cluster) *MatchResult {
	var best *MatchResult
	for _, c := range clusters {
		score := 0.0
		var matched []string

		if fp.HostingIP != nil && *fp.HostingIP == c.HostingIP {
			score += matchWeights["hosting_ip"]
			matched = append(matched, "hosting_ip")
		}
		if fp.Nameserver != nil && c.Nameserver != "" && *fp.Nameserver == c.Nameserver {
			score += matchWeights["nameserver"]
			matched = append(matched, "nameserver")
		}
		if fp.Registrar != nil && c.Registrar != "" && *fp.Registrar == c.Registrar {
			score += matchWeights["registrar"]
			matched = append(matched, "registrar")
		}
		if fp.TLD != "" && fp.TLD == c.TLD {
			score += matchWeights["tld"]
			matched = append(matched, "tld")
		}
		if c.RegistrationBurstScore != nil && *c.RegistrationBurstScore >= 0.5 {
			score += matchWeights["registration_burst"] * *c.RegistrationBurstScore
			matched = append(matched, "registration_burst")
		}

		if score < matchThreshold {
			continue
		}
		if best == nil || score > best.Score {
			best = &MatchResult{ClusterID: c.ID, Score: score, MatchedFields: matched}
		}
	}
	return best
}
