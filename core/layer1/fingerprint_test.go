package layer1

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

var errNotFound = errors.New("not found")

type fakeDNS struct {
	ip      net.IP
	ns      string
	ipCalls int
	nsCalls int
}

func (f *fakeDNS) LookupIP(domain string) ([]net.IP, error) {
	f.ipCalls++
	if f.ip == nil {
		return nil, errNotFound
	}
	return []net.IP{f.ip}, nil
}

func (f *fakeDNS) LookupNS(domain string) ([]*net.NS, error) {
	f.nsCalls++
	if f.ns == "" {
		return nil, errNotFound
	}
	return []*net.NS{{Host: f.ns + "."}}, nil
}

type fakeRDAP struct {
	data  map[string]any
	calls int
}

func (f *fakeRDAP) Lookup(ctx context.Context, domain string) (map[string]any, error) {
	f.calls++
	return f.data, nil
}

func TestExtractPopulatesFields(t *testing.T) {
	dns := &fakeDNS{ip: net.ParseIP("203.0.113.10"), ns: "ns1.fixture-host.test"}
	rdap := &fakeRDAP{data: map[string]any{
		"entities": []any{
			map[string]any{
				"roles": []any{"registrar"},
				"vcardArray": []any{
					"vcard",
					[]any{[]any{"fn", map[string]any{}, "text", "Fixture Registrar"}},
				},
			},
		},
		"events": []any{
			map[string]any{"eventAction": "registration", "eventDate": "2026-07-01T00:00:00Z"},
		},
	}}
	e := &Extractor{dns: dns, rdap: rdap, cache: make(map[string]Fingerprint)}

	fp, err := e.Extract(context.Background(), "gacor88x.xyz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fp.HostingIP == nil || *fp.HostingIP != "203.0.113.10" {
		t.Fatalf("expected hosting ip, got %v", fp.HostingIP)
	}
	if fp.ASNPrefix == nil || *fp.ASNPrefix != "203.0.113.0/24" {
		t.Fatalf("expected /24 prefix, got %v", fp.ASNPrefix)
	}
	if fp.Nameserver == nil || *fp.Nameserver != "ns1.fixture-host.test" {
		t.Fatalf("expected nameserver, got %v", fp.Nameserver)
	}
	if fp.Registrar == nil || *fp.Registrar != "Fixture Registrar" {
		t.Fatalf("expected registrar, got %v", fp.Registrar)
	}
	want := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	if fp.RegisteredAt == nil || !fp.RegisteredAt.Equal(want) {
		t.Fatalf("expected registered_at %v, got %v", want, fp.RegisteredAt)
	}
	if fp.TLD != "xyz" {
		t.Fatalf("expected tld xyz, got %s", fp.TLD)
	}
}

func TestExtractRedactedFieldsAreNilNotError(t *testing.T) {
	dns := &fakeDNS{}
	rdap := &fakeRDAP{data: nil}
	e := &Extractor{dns: dns, rdap: rdap, cache: make(map[string]Fingerprint)}

	fp, err := e.Extract(context.Background(), "redacted.test")
	if err != nil {
		t.Fatalf("expected no error for redacted fields, got %v", err)
	}
	if fp.HostingIP != nil || fp.Registrar != nil || fp.RegisteredAt != nil {
		t.Fatalf("expected nil fields for redacted domain, got %+v", fp)
	}
}

func TestExtractCachesRepeatedLookups(t *testing.T) {
	dns := &fakeDNS{ip: net.ParseIP("203.0.113.10"), ns: "ns1.test"}
	rdap := &fakeRDAP{data: map[string]any{}}
	e := &Extractor{dns: dns, rdap: rdap, cache: make(map[string]Fingerprint)}

	ctx := context.Background()
	if _, err := e.Extract(ctx, "cached.test"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Extract(ctx, "cached.test"); err != nil {
		t.Fatal(err)
	}
	if dns.ipCalls != 1 || rdap.calls != 1 {
		t.Fatalf("expected single lookup per domain, got dns=%d rdap=%d", dns.ipCalls, rdap.calls)
	}
}
