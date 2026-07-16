package layer1

import (
	"math"
	"time"
)

type DomainRecord struct {
	Domain      string
	Fingerprint Fingerprint
}

type Cluster struct {
	ID                      string
	HostingIP               string
	Nameserver              string
	Registrar               string
	TLD                     string
	Domains                 []string
	FirstRegistrationDate   *time.Time
	LastRegistrationDate    *time.Time
	RegistrationWindowHours int
	RegistrationBurstScore  *float64
}

// BuildClusters groups by shared hosting IP — "same landlord, same street"
// (PRD §4). It is a GROUP BY, not clustering ML: do not import a
// clustering/ML library here (PJ-402).
func BuildClusters(records []DomainRecord) []Cluster {
	byIP := map[string][]DomainRecord{}
	for _, r := range records {
		if r.Fingerprint.HostingIP == nil {
			continue
		}
		byIP[*r.Fingerprint.HostingIP] = append(byIP[*r.Fingerprint.HostingIP], r)
	}

	clusters := make([]Cluster, 0, len(byIP))
	for ip, group := range byIP {
		if len(group) < 2 {
			continue // a "cluster" of one domain has no sibling to flag
		}
		clusters = append(clusters, buildCluster(ip, group))
	}
	return clusters
}

func buildCluster(ip string, group []DomainRecord) Cluster {
	c := Cluster{HostingIP: ip}
	for _, r := range group {
		c.Domains = append(c.Domains, r.Domain)
		if r.Fingerprint.Nameserver != nil && c.Nameserver == "" {
			c.Nameserver = *r.Fingerprint.Nameserver
		}
		if r.Fingerprint.Registrar != nil && c.Registrar == "" {
			c.Registrar = *r.Fingerprint.Registrar
		}
		if c.TLD == "" {
			c.TLD = r.Fingerprint.TLD
		}
		if r.Fingerprint.RegisteredAt != nil {
			t := *r.Fingerprint.RegisteredAt
			if c.FirstRegistrationDate == nil || t.Before(*c.FirstRegistrationDate) {
				c.FirstRegistrationDate = &t
			}
			if c.LastRegistrationDate == nil || t.After(*c.LastRegistrationDate) {
				c.LastRegistrationDate = &t
			}
		}
	}
	applyBurstScore(&c)
	return c
}

// applyBurstScore computes the registration-burst signal (PJ-403): a dense
// registration window across many domains is the actual PREDATOR/Netcraft
// signal — the gap between bulk registration and campaign activation.
// Degrades cleanly to a nil score when registration dates are unavailable.
func applyBurstScore(c *Cluster) {
	if c.FirstRegistrationDate == nil || c.LastRegistrationDate == nil {
		return
	}
	windowHours := int(math.Ceil(c.LastRegistrationDate.Sub(*c.FirstRegistrationDate).Hours()))
	if windowHours < 1 {
		windowHours = 1
	}
	c.RegistrationWindowHours = windowHours

	score := float64(len(c.Domains)) / float64(windowHours)
	if score > 1 {
		score = 1
	}
	c.RegistrationBurstScore = &score
}
