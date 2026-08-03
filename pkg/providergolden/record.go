package providergolden

import (
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
)

// Recorder collects the inputs of a provider's record conversion functions as
// the provider is used, and writes them in the formats CheckToRC and
// CheckToNative read. Duplicates are discarded, so a long run that revisits the
// same records produces a small file.
type Recorder struct {
	mu      sync.Mutex
	records map[string]bool
	natives map[string]bool
	errs    []error
}

// NewRecorder returns a Recorder that has seen nothing.
func NewRecorder() *Recorder {
	return &Recorder{
		records: map[string]bool{},
		natives: map[string]bool{},
	}
}

// Observe adds one round of conversion inputs. desired holds the records the
// provider is about to convert into its own format; existing holds the records
// it has just converted out of its own format, each carrying the native record
// it came from in RecordConfig.Original.
func (r *Recorder) Observe(desired, existing models.Records) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, rc := range desired {
		r.records[formatRecord(rc)] = true
	}

	for _, rc := range existing {
		if rc.Original == nil {
			continue
		}
		native, err := json.Marshal(rc.Original)
		if err != nil {
			r.errs = append(r.errs, fmt.Errorf("%s %s: %w", rc.NameFQDN, rc.Type, err))
			continue
		}
		r.natives[string(native)] = true
	}
}

// WriteTo writes what has been observed to dir as <name>.records and
// <name>.json, sorted so that two runs of the same tests produce the same file.
// A file is not written when nothing of that kind was observed: a provider that
// does not fill in RecordConfig.Original produces no <name>.json. WriteTo
// returns the paths it wrote.
func (r *Recorder) WriteTo(dir, name string) ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var written []string

	if len(r.records) != 0 {
		text := strings.Join(slices.Sorted(maps.Keys(r.records)), "\n") + "\n"
		path, err := writeFile(dir, name+".records", []byte(text))
		if err != nil {
			return written, err
		}
		written = append(written, path)
	}

	if len(r.natives) != 0 {
		natives := make([]json.RawMessage, 0, len(r.natives))
		for _, native := range slices.Sorted(maps.Keys(r.natives)) {
			natives = append(natives, json.RawMessage(native))
		}
		text, err := json.MarshalIndent(natives, "", "  ")
		if err != nil {
			return written, err
		}
		path, err := writeFile(dir, name+".json", append(text, '\n'))
		if err != nil {
			return written, err
		}
		written = append(written, path)
	}

	return written, errors.Join(r.errs...)
}

// TestdataDir returns the testdata directory that belongs to p:
// providers/<package>/testdata under the module root, where <package> is the
// package p is implemented in.
func TestdataDir(p models.DNSProvider) (string, error) {
	t := reflect.TypeOf(p)
	for t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == nil || t.PkgPath() == "" {
		return "", fmt.Errorf("cannot derive a testdata directory from provider type %T", p)
	}

	root, err := moduleRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "providers", path.Base(t.PkgPath()), testdataDir), nil
}

// ResolveDir returns dir as an absolute path, resolving a relative dir against
// the module root.
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

// moduleRoot returns the directory of the nearest go.mod at or above the
// working directory.
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

// writeFile writes data to <dir>/<filename> and returns the path, creating dir
// when it does not exist.
func writeFile(dir, filename string, data []byte) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, filename)
	return path, os.WriteFile(path, data, 0o644)
}

// Record wraps p so that rec observes every record conversion p is asked to
// perform. The wrapper is how the integration tests gather data: they drive a
// real provider in the same process, and every zone they build passes through
// GetZoneRecordsCorrections on its way to the provider's own conversion
// functions.
func Record(p models.DNSProvider, rec *Recorder) models.DNSProvider {
	return &recordingProvider{DNSProvider: p, rec: rec}
}

type recordingProvider struct {
	models.DNSProvider
	rec *Recorder
}

func (p *recordingProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existing models.Records) ([]*models.Correction, int, error) {
	p.rec.Observe(dc.Records, existing)
	return p.DNSProvider.GetZoneRecordsCorrections(dc, existing)
}
