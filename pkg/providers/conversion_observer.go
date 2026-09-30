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

// ConversionObserverSetter supplies observers to legacy providers after
// construction. Providers using Register receive the observer in Initialize's
// options and install it there, before any conversions occur.
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

// CreateOptions holds dependencies supplied while constructing a provider.
type CreateOptions struct {
	ConversionObserver ConversionObserver
	// RequestedRole is set by the typed factory after caller options are applied.
	// It is exactly KindDNS or KindRegistrar for runtime instances.
	RequestedRole ProviderKind
}

// WithDefaults returns a copy with shared defaults applied. A nil receiver and
// zero-valued options are equivalent. The requested role remains unspecified
// unless set by a typed factory.
func (options *CreateOptions) WithDefaults() CreateOptions {
	var result CreateOptions
	if options != nil {
		result = *options
	}
	if result.ConversionObserver == nil {
		result.ConversionObserver = noopConversionObserver{}
	}
	return result
}

// CreateOption customizes provider construction.
type CreateOption func(*CreateOptions)

// WithConversionObserver supplies an observer for provider record conversions.
func WithConversionObserver(observer ConversionObserver) CreateOption {
	return func(options *CreateOptions) {
		options.ConversionObserver = observer
	}
}

func newCreateOptions(opts []CreateOption) CreateOptions {
	var options CreateOptions
	for _, opt := range opts {
		opt(&options)
	}
	return options.WithDefaults()
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
