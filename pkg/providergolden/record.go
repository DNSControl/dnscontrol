package providergolden

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

type recordingMetadata struct {
	Version    int                   `json:"version"`
	Recordings []recordingDescriptor `json:"recordings"`
}

type recordingDescriptor struct {
	Domain   string   `json:"domain"`
	ToRC     []string `json:"to_rc,omitempty"`
	ToNative []string `json:"to_native,omitempty"`
}

type toRCObservation struct {
	input  json.RawMessage
	output []string
}

type toNativeObservation struct {
	input  []string
	output json.RawMessage
}

// Recorder collects exact conversion inputs and outputs from a provider.
type Recorder struct {
	mu       sync.Mutex
	toRC     map[string]map[string]map[string]toRCObservation
	toNative map[string]map[string]map[string]toNativeObservation
	errs     []error
}

// NewRecorder returns an empty conversion recorder.
func NewRecorder() *Recorder {
	return &Recorder{
		toRC:     map[string]map[string]map[string]toRCObservation{},
		toNative: map[string]map[string]map[string]toNativeObservation{},
	}
}

// ForDomain returns a constructor-injectable observer scoped to one zone.
func (r *Recorder) ForDomain(domain string) providers.ConversionObserver {
	return &domainObserver{recorder: r, domain: strings.TrimSuffix(strings.ToLower(domain), ".")}
}

type domainObserver struct {
	recorder *Recorder
	domain   string
}

type conversionSnapshot struct {
	serialized []byte
	records    []string
}

func (o *domainObserver) BeginToRC(_ string, native any) providers.ConversionSnapshot {
	data, err := json.Marshal(native)
	if err != nil {
		o.recorder.addError(err)
	}
	return conversionSnapshot{serialized: data}
}

func (o *domainObserver) EndToRC(function string, before providers.ConversionSnapshot, nativeAfter any, result models.Records, convertErr error) {
	snapshot, ok := before.(conversionSnapshot)
	if !ok {
		o.recorder.addError(fmt.Errorf("%s ToRC returned an invalid observer snapshot", function))
		return
	}
	after, err := json.Marshal(nativeAfter)
	if err != nil {
		o.recorder.addError(err)
		return
	}
	if !bytes.Equal(snapshot.serialized, after) {
		o.recorder.addError(fmt.Errorf("%s ToRC mutated its native input", function))
		return
	}
	if convertErr != nil {
		o.recorder.addError(fmt.Errorf("%s ToRC: %w", function, convertErr))
		return
	}
	output := recordStrings(result)
	o.recorder.addToRC(o.domain, function, snapshot.serialized, output)
}

func (o *domainObserver) BeginToNative(_ string, records models.Records) providers.ConversionSnapshot {
	data, err := json.Marshal(records)
	if err != nil {
		o.recorder.addError(err)
	}
	return conversionSnapshot{serialized: data, records: recordStrings(records)}
}

func (o *domainObserver) EndToNative(function string, before providers.ConversionSnapshot, recordsAfter models.Records, result any, convertErr error) {
	snapshot, ok := before.(conversionSnapshot)
	if !ok {
		o.recorder.addError(fmt.Errorf("%s ToNative returned an invalid observer snapshot", function))
		return
	}
	after, err := json.Marshal(recordsAfter)
	if err != nil {
		o.recorder.addError(err)
		return
	}
	if !bytes.Equal(snapshot.serialized, after) {
		o.recorder.addError(fmt.Errorf("%s ToNative mutated its RecordConfig input", function))
		return
	}
	if convertErr != nil {
		o.recorder.addError(fmt.Errorf("%s ToNative: %w", function, convertErr))
		return
	}
	output, err := json.Marshal(result)
	if err != nil {
		o.recorder.addError(err)
		return
	}
	o.recorder.addToNative(o.domain, function, snapshot.records, output)
}

func recordStrings(records models.Records) []string {
	result := make([]string, 0, len(records))
	for _, rc := range records {
		if rc != nil {
			result = append(result, rc.StringWithMeta())
		}
	}
	return result
}

func (r *Recorder) addError(err error) {
	if err == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.errs = append(r.errs, err)
}

func (r *Recorder) addToRC(domain, function string, input []byte, output []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.toRC[domain] == nil {
		r.toRC[domain] = map[string]map[string]toRCObservation{}
	}
	if r.toRC[domain][function] == nil {
		r.toRC[domain][function] = map[string]toRCObservation{}
	}
	key := string(input)
	observation := toRCObservation{input: slices.Clone(input), output: slices.Clone(output)}
	if old, ok := r.toRC[domain][function][key]; ok && !slices.Equal(old.output, output) {
		r.errs = append(r.errs, fmt.Errorf("%s ToRC produced different outputs for the same input", function))
		return
	}
	r.toRC[domain][function][key] = observation
}

func (r *Recorder) addToNative(domain, function string, input []string, output []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.toNative[domain] == nil {
		r.toNative[domain] = map[string]map[string]toNativeObservation{}
	}
	if r.toNative[domain][function] == nil {
		r.toNative[domain][function] = map[string]toNativeObservation{}
	}
	key := strings.Join(input, "\n")
	observation := toNativeObservation{input: slices.Clone(input), output: slices.Clone(output)}
	if old, ok := r.toNative[domain][function][key]; ok && !bytes.Equal(old.output, output) {
		r.errs = append(r.errs, fmt.Errorf("%s ToNative produced different outputs for the same input", function))
		return
	}
	r.toNative[domain][function][key] = observation
}

type indexedJSON struct {
	Index int             `json:"index"`
	Value json.RawMessage `json:"value"`
}

// WriteTo writes all recorded conversion inputs and their expected outputs.
// Existing recordings for other domains are preserved.
func (r *Recorder) WriteTo(dir string) ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.errs) != 0 {
		return nil, errors.Join(r.errs...)
	}

	metadata, err := readMetadata(dir)
	if err != nil {
		return nil, err
	}
	byDomain := map[string]recordingDescriptor{}
	for _, recording := range metadata.Recordings {
		byDomain[recording.Domain] = recording
	}

	var written []string
	for _, domain := range sortedUnionKeys(r.toRC, r.toNative) {
		descriptor := byDomain[domain]
		descriptor.Domain = domain
		for _, function := range slices.Sorted(maps.Keys(r.toRC[domain])) {
			paths, err := writeToRC(dir, function, domain, r.toRC[domain][function])
			if err != nil {
				return written, err
			}
			written = append(written, paths...)
			descriptor.ToRC = appendUnique(descriptor.ToRC, function)
		}
		for _, function := range slices.Sorted(maps.Keys(r.toNative[domain])) {
			paths, err := writeToNative(dir, function, domain, r.toNative[domain][function])
			if err != nil {
				return written, err
			}
			written = append(written, paths...)
			descriptor.ToNative = appendUnique(descriptor.ToNative, function)
		}
		slices.Sort(descriptor.ToRC)
		slices.Sort(descriptor.ToNative)
		byDomain[domain] = descriptor
	}

	metadata.Version = 1
	metadata.Recordings = metadata.Recordings[:0]
	for _, domain := range slices.Sorted(maps.Keys(byDomain)) {
		metadata.Recordings = append(metadata.Recordings, byDomain[domain])
	}
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return written, err
	}
	path, err := writeFile(dir, "meta.json", append(data, '\n'))
	if err != nil {
		return written, err
	}
	written = append([]string{path}, written...)
	return written, nil
}

func writeToRC(dir, function, domain string, observations map[string]toRCObservation) ([]string, error) {
	keys := slices.Sorted(maps.Keys(observations))
	inputs := make([]indexedJSON, 0, len(keys))
	var outputs strings.Builder
	for i, key := range keys {
		index := i + 1
		observation := observations[key]
		inputs = append(inputs, indexedJSON{Index: index, Value: observation.input})
		for _, record := range observation.output {
			fmt.Fprintf(&outputs, "%d\t%s\n", index, record)
		}
	}
	data, err := json.MarshalIndent(inputs, "", "  ")
	if err != nil {
		return nil, err
	}
	inputPath, err := writeFile(dir, toRCInputFile(function, domain), append(data, '\n'))
	if err != nil {
		return nil, err
	}
	outputPath, err := writeFile(dir, toRCOutputFile(function, domain), []byte(outputs.String()))
	return []string{inputPath, outputPath}, err
}

func writeToNative(dir, function, domain string, observations map[string]toNativeObservation) ([]string, error) {
	keys := slices.Sorted(maps.Keys(observations))
	outputs := make([]indexedJSON, 0, len(keys))
	var inputs strings.Builder
	for i, key := range keys {
		index := i + 1
		observation := observations[key]
		for _, record := range observation.input {
			fmt.Fprintf(&inputs, "%d\t%s\n", index, record)
		}
		outputs = append(outputs, indexedJSON{Index: index, Value: observation.output})
	}
	inputPath, err := writeFile(dir, toNativeInputFile(function, domain), []byte(inputs.String()))
	if err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(outputs, "", "  ")
	if err != nil {
		return nil, err
	}
	outputPath, err := writeFile(dir, toNativeOutputFile(function, domain), append(data, '\n'))
	return []string{inputPath, outputPath}, err
}

func readMetadata(dir string) (recordingMetadata, error) {
	data, err := os.ReadFile(filepath.Join(dir, "meta.json"))
	if errors.Is(err, os.ErrNotExist) {
		return recordingMetadata{Version: 1}, nil
	}
	if err != nil {
		return recordingMetadata{}, err
	}
	var metadata recordingMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return recordingMetadata{}, err
	}
	return metadata, nil
}

func sortedUnionKeys[A, B any](a map[string]A, b map[string]B) []string {
	keys := map[string]bool{}
	for key := range a {
		keys[key] = true
	}
	for key := range b {
		keys[key] = true
	}
	return slices.Sorted(maps.Keys(keys))
}

func appendUnique(items []string, item string) []string {
	if !slices.Contains(items, item) {
		return append(items, item)
	}
	return items
}

// ProviderName returns the package name of p's implementation.
func ProviderName(p models.DNSProvider) (string, error) {
	t := reflect.TypeOf(p)
	for t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == nil || t.PkgPath() == "" {
		return "", fmt.Errorf("cannot derive a provider name from provider type %T", p)
	}
	return path.Base(t.PkgPath()), nil
}

// TestdataDir returns providers/<package>/test_data under the module root.
func TestdataDir(p models.DNSProvider) (string, error) {
	name, err := ProviderName(p)
	if err != nil {
		return "", err
	}
	root, err := moduleRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "providers", name, testDataDir), nil
}

// ResolveDir resolves dir against the module root when it is relative.
func ResolveDir(dir string) (string, error) {
	if filepath.IsAbs(dir) {
		return dir, nil
	}
	root, err := moduleRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, dir), nil
}

func moduleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("no go.mod above the working directory")
		}
		dir = parent
	}
}

func writeFile(dir, filename string, data []byte) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, filename)
	return path, os.WriteFile(path, data, 0o644)
}
