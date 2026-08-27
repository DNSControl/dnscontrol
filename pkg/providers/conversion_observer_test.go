package providers

import (
	"encoding/json"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

type testObserver struct{ noopConversionObserver }

type observerProvider struct {
	observer ConversionObserver
}

func (*observerProvider) GetNameservers(string) ([]*models.Nameserver, error) { return nil, nil }
func (*observerProvider) GetZoneRecords(*models.DomainConfig) (models.Records, error) {
	return nil, nil
}
func (*observerProvider) GetZoneRecordsCorrections(*models.DomainConfig, models.Records) ([]*models.Correction, int, error) {
	return nil, 0, nil
}
func (p *observerProvider) SetConversionObserver(observer ConversionObserver) {
	p.observer = observer
}

func TestCreateDNSProviderInjectsConversionObserver(t *testing.T) {
	const name = "PROVIDERGOLDEN_OBSERVER_TEST"
	old, existed := DNSProviderTypes[name]
	t.Cleanup(func() {
		if existed {
			DNSProviderTypes[name] = old
		} else {
			delete(DNSProviderTypes, name)
		}
	})
	DNSProviderTypes[name] = DspFuncs{
		Initializer: func(map[string]string, json.RawMessage) (DNSServiceProvider, error) {
			return &observerProvider{}, nil
		},
	}

	want := &testObserver{}
	provider, err := CreateDNSProvider(name, nil, nil, WithConversionObserver(want))
	if err != nil {
		t.Fatal(err)
	}
	if got := provider.(*observerProvider).observer; got != want {
		t.Fatalf("observer = %T %p, want %T %p", got, got, want, want)
	}
}
