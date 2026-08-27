package providers

import "github.com/DNSControl/dnscontrol/v5/models"

// ConversionSnapshot is an opaque snapshot returned before a conversion and
// passed back after it. Its concrete type belongs to the observer.
type ConversionSnapshot any

// ConversionObserver observes the exact inputs and outputs of provider record
// conversions. Implementations may also verify that conversions did not mutate
// their inputs.
type ConversionObserver interface {
	BeginToRC(function string, native any) ConversionSnapshot
	EndToRC(function string, before ConversionSnapshot, nativeAfter any, result models.Records, err error)
	BeginToNative(function string, records models.Records) ConversionSnapshot
	EndToNative(function string, before ConversionSnapshot, recordsAfter models.Records, result any, err error)
}

// ConversionObserverSetter is implemented by providers that expose conversion
// boundaries. CreateDNSProvider calls it before returning the constructed
// provider.
type ConversionObserverSetter interface {
	SetConversionObserver(ConversionObserver)
}

type noopConversionObserver struct{}

func (noopConversionObserver) BeginToRC(string, any) ConversionSnapshot { return nil }
func (noopConversionObserver) EndToRC(string, ConversionSnapshot, any, models.Records, error) {
}
func (noopConversionObserver) BeginToNative(string, models.Records) ConversionSnapshot { return nil }
func (noopConversionObserver) EndToNative(string, ConversionSnapshot, models.Records, any, error) {
}

// CreateOptions holds optional dependencies supplied while constructing a DNS
// provider.
type CreateOptions struct {
	ConversionObserver ConversionObserver
}

// CreateOption customizes DNS provider construction.
type CreateOption func(*CreateOptions)

// WithConversionObserver supplies an observer for provider record conversions.
func WithConversionObserver(observer ConversionObserver) CreateOption {
	return func(options *CreateOptions) {
		options.ConversionObserver = observer
	}
}

func newCreateOptions(opts []CreateOption) CreateOptions {
	options := CreateOptions{ConversionObserver: noopConversionObserver{}}
	for _, opt := range opts {
		opt(&options)
	}
	if options.ConversionObserver == nil {
		options.ConversionObserver = noopConversionObserver{}
	}
	return options
}

// BeginToRC safely begins an observation when observer may be nil.
func BeginToRC(observer ConversionObserver, function string, native any) ConversionSnapshot {
	if observer == nil {
		return nil
	}
	return observer.BeginToRC(function, native)
}

// EndToRC safely completes an observation when observer may be nil.
func EndToRC(observer ConversionObserver, function string, before ConversionSnapshot, native any, result models.Records, err error) {
	if observer != nil {
		observer.EndToRC(function, before, native, result, err)
	}
}

// BeginToNative safely begins an observation when observer may be nil.
func BeginToNative(observer ConversionObserver, function string, records models.Records) ConversionSnapshot {
	if observer == nil {
		return nil
	}
	return observer.BeginToNative(function, records)
}

// EndToNative safely completes an observation when observer may be nil.
func EndToNative(observer ConversionObserver, function string, before ConversionSnapshot, records models.Records, result any, err error) {
	if observer != nil {
		observer.EndToNative(function, before, records, result, err)
	}
}
