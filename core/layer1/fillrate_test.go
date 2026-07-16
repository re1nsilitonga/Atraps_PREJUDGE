package layer1

import "testing"

func TestFillRateComputesPerFieldFraction(t *testing.T) {
	registrar := "Fixture Registrar"
	ip := "203.0.113.10"
	fps := []Fingerprint{
		{Registrar: &registrar, HostingIP: &ip},
		{HostingIP: &ip},
		{},
	}
	rates := FillRate(fps)
	if rates["registrar"] != 1.0/3.0 {
		t.Fatalf("expected registrar rate 1/3, got %v", rates["registrar"])
	}
	if rates["hosting_ip"] != 2.0/3.0 {
		t.Fatalf("expected hosting_ip rate 2/3, got %v", rates["hosting_ip"])
	}
	if rates["registered_at"] != 0 {
		t.Fatalf("expected registered_at rate 0, got %v", rates["registered_at"])
	}
}

func TestFillRateEmptyInput(t *testing.T) {
	rates := FillRate(nil)
	for field, rate := range rates {
		if rate != 0 {
			t.Fatalf("expected zero rate for %s on empty input, got %v", field, rate)
		}
	}
}
