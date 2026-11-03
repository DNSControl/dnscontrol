package ionos

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

// countingObserver counts the conversions reported to it, per function name.
type countingObserver struct {
	toRC     map[string]int
	toNative map[string]int
}

func newCountingObserver() *countingObserver {
	return &countingObserver{toRC: map[string]int{}, toNative: map[string]int{}}
}

func (o *countingObserver) BeginToRC(string, any) providers.ConversionSnapshot { return nil }

func (o *countingObserver) EndToRC(function string, _ providers.ConversionSnapshot, _ any, _ models.Records, _ error) {
	o.toRC[function]++
}

func (o *countingObserver) BeginToNative(string, models.Records) providers.ConversionSnapshot {
	return nil
}

func (o *countingObserver) EndToNative(function string, _ providers.ConversionSnapshot, _ models.Records, _ any, _ error) {
	o.toNative[function]++
}

func TestInitializeInstallsConversionObserver(t *testing.T) {
	observer := newCountingObserver()
	p := &ionosProvider{}
	options := &providers.CreateOptions{ConversionObserver: observer}
	if err := p.Initialize(map[string]string{"api_key": "prefix.secret"}, nil, options); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if p.observer != observer {
		t.Fatal("Initialize did not install the supplied observer")
	}
	if err := p.Initialize(map[string]string{"api_key": "prefix.secret"}, nil, nil); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if p.observer == nil {
		t.Fatal("Initialize did not install the default observer")
	}
}

func TestConversionsAreReportedToObserver(t *testing.T) {
	p, _ := newTestProvider(t,
		record{ID: "keep", Name: "keep.example.com", Type: "A", Content: "192.0.2.1", TTL: 300},
		record{ID: "chg", Name: "chg.example.com", Type: "A", Content: "192.0.2.2", TTL: 300},
		record{ID: "del", Name: "del.example.com", Type: "A", Content: "192.0.2.3", TTL: 300},
	)
	observer := newCountingObserver()
	p.SetConversionObserver(observer)
	dc := newTestDomain(t)
	addRecord(dc, "keep", 300, "A", "192.0.2.1")
	addRecord(dc, "chg", 600, "A", "192.0.2.20")
	addRecord(dc, "new", 300, "MX", "10 mx.example.net.")

	applyCorrections(t, p, dc)

	// Three records read from the API; one created and one changed record sent to it.
	if got := observer.toRC["toRC"]; got != 3 {
		t.Errorf("toRC observations = %d, want 3 (all: %v)", got, observer.toRC)
	}
	if got := observer.toNative["toNative"]; got != 2 {
		t.Errorf("toNative observations = %d, want 2 (all: %v)", got, observer.toNative)
	}
}
