package layer1

func FillRate(fingerprints []Fingerprint) map[string]float64 {
	rates := map[string]float64{
		"registrar":     0,
		"hosting_ip":    0,
		"nameserver":    0,
		"registered_at": 0,
	}
	if len(fingerprints) == 0 {
		return rates
	}
	for _, fp := range fingerprints {
		if fp.Registrar != nil {
			rates["registrar"]++
		}
		if fp.HostingIP != nil {
			rates["hosting_ip"]++
		}
		if fp.Nameserver != nil {
			rates["nameserver"]++
		}
		if fp.RegisteredAt != nil {
			rates["registered_at"]++
		}
	}
	n := float64(len(fingerprints))
	for k := range rates {
		rates[k] /= n
	}
	return rates
}
