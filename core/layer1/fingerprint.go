// Package layer1 implements Core Engine Layer 1: preemptive detection via
// infrastructure fingerprinting and cluster correlation.
//
// Imports nothing Chrome-shaped, nothing realtime-shaped, nothing UI-shaped
// (PRD.md §8). Network calls go through the dnsLookup/rdapClient interfaces
// so extraction logic is unit-testable without a live network.
package layer1

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Fingerprint struct {
	Domain       string
	Registrar    *string
	HostingIP    *string
	ASNPrefix    *string // IP /24 prefix stands in for ASN (PJ-401: no free stdlib ASN source)
	Nameserver   *string
	TLD          string
	RegisteredAt *time.Time
	Raw          map[string]any
}

type dnsLookup interface {
	LookupIP(domain string) ([]net.IP, error)
	LookupNS(domain string) ([]*net.NS, error)
}

type rdapClient interface {
	Lookup(ctx context.Context, domain string) (map[string]any, error)
}

type systemDNS struct{}

func (systemDNS) LookupIP(domain string) ([]net.IP, error)  { return net.LookupIP(domain) }
func (systemDNS) LookupNS(domain string) ([]*net.NS, error) { return net.LookupNS(domain) }

// httpRDAP queries the public RDAP bootstrap redirector. RDAP (RFC 9083) is
// the IETF-standardized JSON-over-HTTPS successor to WHOIS — using it avoids
// hand-rolling the legacy port-43 text protocol for the same registrar/date
// fields PJ-401 needs.
type httpRDAP struct{ client *http.Client }

func (h httpRDAP) Lookup(ctx context.Context, domain string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://rdap.org/domain/"+domain, nil)
	if err != nil {
		return nil, err
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil // redacted/unregistered — PJ-401: "Redacted WHOIS → None, not exceptions"
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	return body, nil
}

type Extractor struct {
	dns  dnsLookup
	rdap rdapClient

	mu    sync.Mutex
	cache map[string]Fingerprint
}

func NewExtractor() *Extractor {
	return &Extractor{
		dns:   systemDNS{},
		rdap:  httpRDAP{client: &http.Client{Timeout: 8 * time.Second}},
		cache: make(map[string]Fingerprint),
	}
}

// Extract never re-queries a domain already in the cache
// (PJ-401: "Cached — never re-query a fetched domain").
func (e *Extractor) Extract(ctx context.Context, domain string) (Fingerprint, error) {
	e.mu.Lock()
	if fp, ok := e.cache[domain]; ok {
		e.mu.Unlock()
		return fp, nil
	}
	e.mu.Unlock()

	fp := Fingerprint{Domain: domain, TLD: tld(domain)}

	if ips, err := e.dns.LookupIP(domain); err == nil && len(ips) > 0 {
		ip := ips[0].String()
		fp.HostingIP = &ip
		if prefix := ipPrefix24(ips[0]); prefix != "" {
			fp.ASNPrefix = &prefix
		}
	}

	if nss, err := e.dns.LookupNS(domain); err == nil && len(nss) > 0 {
		ns := strings.TrimSuffix(nss[0].Host, ".")
		fp.Nameserver = &ns
	}

	if raw, err := e.rdap.Lookup(ctx, domain); err == nil && raw != nil {
		fp.Raw = raw
		if registrar := registrarFromRDAP(raw); registrar != "" {
			fp.Registrar = &registrar
		}
		if registered := registeredAtFromRDAP(raw); registered != nil {
			fp.RegisteredAt = registered
		}
	}

	e.mu.Lock()
	e.cache[domain] = fp
	e.mu.Unlock()

	return fp, nil
}

func registrarFromRDAP(raw map[string]any) string {
	entities, _ := raw["entities"].([]any)
	for _, e := range entities {
		entity, ok := e.(map[string]any)
		if !ok {
			continue
		}
		roles, _ := entity["roles"].([]any)
		for _, r := range roles {
			if role, ok := r.(string); ok && role == "registrar" {
				if name := vcardFN(entity); name != "" {
					return name
				}
			}
		}
	}
	return ""
}

func vcardFN(entity map[string]any) string {
	vcard, ok := entity["vcardArray"].([]any)
	if !ok || len(vcard) < 2 {
		return ""
	}
	fields, ok := vcard[1].([]any)
	if !ok {
		return ""
	}
	for _, f := range fields {
		field, ok := f.([]any)
		if !ok || len(field) < 4 {
			continue
		}
		if name, _ := field[0].(string); name == "fn" {
			value, _ := field[3].(string)
			return value
		}
	}
	return ""
}

func registeredAtFromRDAP(raw map[string]any) *time.Time {
	events, _ := raw["events"].([]any)
	for _, e := range events {
		event, ok := e.(map[string]any)
		if !ok {
			continue
		}
		if action, _ := event["eventAction"].(string); action != "registration" {
			continue
		}
		dateStr, _ := event["eventDate"].(string)
		if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
			return &t
		}
	}
	return nil
}

func tld(domain string) string {
	parts := strings.Split(domain, ".")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func ipPrefix24(ip net.IP) string {
	ip4 := ip.To4()
	if ip4 == nil {
		return ""
	}
	return fmt.Sprintf("%d.%d.%d.0/24", ip4[0], ip4[1], ip4[2])
}
