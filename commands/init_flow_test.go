package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DNSControl/dnscontrol/v4/models"
)

// stubAsker drives the init flow from a pre recorded script for
// deterministic tests.
type stubAsker struct {
	t            *testing.T
	selects      []string
	multiSelects [][]string
	inputs       []string
	secrets      []string
	multilines   []string
	confirm      []bool
}

func (stub *stubAsker) Select(_, _ string, options []string, _ string) (string, error) {
	if len(stub.selects) == 0 {
		stub.t.Fatalf("unexpected Select (options %v)", options)
	}
	value := stub.selects[0]
	stub.selects = stub.selects[1:]
	return value, nil
}

func (stub *stubAsker) MultiSelect(_, _ string, _ []string) ([]string, error) {
	if len(stub.multiSelects) == 0 {
		return nil, nil
	}
	value := stub.multiSelects[0]
	stub.multiSelects = stub.multiSelects[1:]
	return value, nil
}

func (stub *stubAsker) Input(_, _, defaultValue string) (string, error) {
	if len(stub.inputs) == 0 {
		return defaultValue, nil
	}
	value := stub.inputs[0]
	stub.inputs = stub.inputs[1:]
	if value == "" {
		return defaultValue, nil
	}
	return value, nil
}

func (stub *stubAsker) Secret(_, _ string) (string, error) {
	if len(stub.secrets) == 0 {
		stub.t.Fatalf("unexpected Secret call")
	}
	value := stub.secrets[0]
	stub.secrets = stub.secrets[1:]
	return value, nil
}

func (stub *stubAsker) Multiline(_, _ string) (string, error) {
	if len(stub.multilines) == 0 {
		stub.t.Fatalf("unexpected Multiline call")
	}
	value := stub.multilines[0]
	stub.multilines = stub.multilines[1:]
	return value, nil
}

func (stub *stubAsker) Confirm(_ string, _ bool) (bool, error) {
	if len(stub.confirm) == 0 {
		stub.t.Fatalf("unexpected Confirm call")
	}
	value := stub.confirm[0]
	stub.confirm = stub.confirm[1:]
	return value, nil
}

func stubFetchNoRecords(t *testing.T) {
	t.Helper()
	origFetch := fetchZoneRecordsFunc
	fetchZoneRecordsFunc = func(_ InitCredsEntry, _ string) (models.Records, error) {
		return nil, nil
	}
	t.Cleanup(func() { fetchZoneRecordsFunc = origFetch })
}

func TestRunInit_VerifyDNSProviderCredsWithZones(t *testing.T) {
	dir := t.TempDir()

	stub := &stubAsker{
		t: t,
		selects: []string{
			"BIND", // DNS provider
			"NONE", // registrar
		},
		inputs: []string{
			"", // BIND: entry name
			"", // BIND: directory
			"", // BIND: filenameformat
			"", // NONE: entry name
		},
		multiSelects: [][]string{
			{"example.com", "example.org"}, // zone selection
		},
		confirm: []bool{
			true,  // "Pick domains from the zone list?"
			false, // "Add another domain manually?"
			true,  // Write these files?

			false, // Run preview now?
		},
	}

	origRun := runSubcommand
	runSubcommand = func(*exec.Cmd) error { return nil }
	t.Cleanup(func() { runSubcommand = origRun })

	origVerify := verifyDNSProviderCredsFunc
	verifyDNSProviderCredsFunc = func(_ InitCredsEntry) ([]string, error) {
		return []string{"example.com", "example.org", "example.net"}, nil
	}
	t.Cleanup(func() { verifyDNSProviderCredsFunc = origVerify })

	stubFetchNoRecords(t)

	args := InitArgs{
		CredsFile:  filepath.Join(dir, "creds.json"),
		ConfigFile: filepath.Join(dir, "dnsconfig.js"),
	}
	if err := runInit(args, stub); err != nil {
		t.Fatalf("runInit: %v", err)
	}

	configBytes, err := os.ReadFile(args.ConfigFile)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(configBytes), `D("example.com"`) {
		t.Errorf("config missing example.com domain: %s", configBytes)
	}
	if !strings.Contains(string(configBytes), `D("example.org"`) {
		t.Errorf("config missing example.org domain: %s", configBytes)
	}
	if strings.Contains(string(configBytes), `D("example.net"`) {
		t.Errorf("config should not contain unselected example.net: %s", configBytes)
	}
}

func TestRunInit_VerifyDNSProviderCredsRetry(t *testing.T) {
	dir := t.TempDir()

	attempt := 0
	stub := &stubAsker{
		t: t,
		selects: []string{
			"BIND",              // DNS provider
			"NONE",              // registrar
			"Retry credentials", // first verify fails
		},
		inputs: []string{
			"", // BIND: entry name
			"", // BIND: directory (first attempt)
			"", // BIND: filenameformat (first attempt)
			"", // BIND: directory (retry)
			"", // BIND: filenameformat (retry)
			"", // NONE: entry name
			"example.com",
		},
		confirm: []bool{
			false, // "Add another domain?"
			true,  // Write these files?

			false, // Run preview now?
		},
	}

	origRun := runSubcommand
	runSubcommand = func(*exec.Cmd) error { return nil }
	t.Cleanup(func() { runSubcommand = origRun })

	origVerify := verifyDNSProviderCredsFunc
	verifyDNSProviderCredsFunc = func(_ InitCredsEntry) ([]string, error) {
		attempt++
		if attempt == 1 {
			return nil, fmt.Errorf("invalid credentials")
		}
		return nil, nil
	}
	t.Cleanup(func() { verifyDNSProviderCredsFunc = origVerify })

	stubFetchNoRecords(t)

	args := InitArgs{
		CredsFile:  filepath.Join(dir, "creds.json"),
		ConfigFile: filepath.Join(dir, "dnsconfig.js"),
	}
	if err := runInit(args, stub); err != nil {
		t.Fatalf("runInit: %v", err)
	}

	if attempt != 2 {
		t.Errorf("expected 2 verification attempts, got %d", attempt)
	}
}

func TestRunInit_VerifyDNSProviderCredsAbort(t *testing.T) {
	dir := t.TempDir()

	stub := &stubAsker{
		t: t,
		selects: []string{
			"BIND",  // DNS provider
			"NONE",  // registrar
			"Abort", // verify fails, user aborts
		},
		inputs: []string{
			"", // BIND: entry name
			"", // BIND: directory
			"", // BIND: filenameformat
		},
	}

	origRun := runSubcommand
	runSubcommand = func(*exec.Cmd) error { return nil }
	t.Cleanup(func() { runSubcommand = origRun })

	origVerify := verifyDNSProviderCredsFunc
	verifyDNSProviderCredsFunc = func(_ InitCredsEntry) ([]string, error) {
		return nil, fmt.Errorf("invalid credentials")
	}
	t.Cleanup(func() { verifyDNSProviderCredsFunc = origVerify })

	args := InitArgs{
		CredsFile:  filepath.Join(dir, "creds.json"),
		ConfigFile: filepath.Join(dir, "dnsconfig.js"),
	}
	err := runInit(args, stub)
	if err != errInitAborted {
		t.Fatalf("expected errInitAborted, got: %v", err)
	}
}

// TestRunInit_NoneBindFlow walks the full init flow using only the built
// in providers (NONE registrar + BIND DNS). It asserts the generated
// creds.json and dnsconfig.js parse cleanly.
func TestRunInit_NoneBindFlow(t *testing.T) {
	dir := t.TempDir()

	stub := &stubAsker{
		t: t,
		selects: []string{
			"BIND", // DNS provider (asked first)
			"NONE", // registrar
		},
		inputs: []string{
			"",            // BIND: entry name (accept default "bind_primary")
			"",            // BIND: directory (accept default)
			"",            // BIND: filenameformat (accept default)
			"",            // NONE: entry name (accept default "none_primary")
			"example.com", // first domain
		},
		confirm: []bool{
			false, // "Add another domain?"
			true,  // Write these files?

			false, // Run preview now?
		},
	}

	// Replace the subprocess seam so the test does not actually exec the
	// dnscontrol binary for `preview` or `get-zones`.
	origRun := runSubcommand
	runSubcommand = func(*exec.Cmd) error { return nil }
	t.Cleanup(func() { runSubcommand = origRun })

	origVerify := verifyDNSProviderCredsFunc
	verifyDNSProviderCredsFunc = func(_ InitCredsEntry) ([]string, error) {
		return nil, nil
	}
	t.Cleanup(func() { verifyDNSProviderCredsFunc = origVerify })

	stubFetchNoRecords(t)

	args := InitArgs{
		CredsFile:  filepath.Join(dir, "creds.json"),
		ConfigFile: filepath.Join(dir, "dnsconfig.js"),
	}
	if err := runInit(args, stub); err != nil {
		t.Fatalf("runInit: %v", err)
	}

	credsBytes, err := os.ReadFile(args.CredsFile)
	if err != nil {
		t.Fatalf("read creds: %v", err)
	}
	if !strings.Contains(string(credsBytes), `"bind_primary"`) {
		t.Errorf("creds.json missing bind_primary entry: %s", credsBytes)
	}
	if !strings.Contains(string(credsBytes), `"TYPE": "BIND"`) {
		t.Errorf("creds.json missing BIND TYPE: %s", credsBytes)
	}

	configBytes, err := os.ReadFile(args.ConfigFile)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(configBytes), `NewDnsProvider("bind_primary")`) {
		t.Errorf("config missing bind_primary provider: %s", configBytes)
	}
	if !strings.Contains(string(configBytes), `D("example.com"`) {
		t.Errorf("config missing example.com domain: %s", configBytes)
	}
}

func TestRunInit_ImportRecords(t *testing.T) {
	dir := t.TempDir()

	stub := &stubAsker{
		t: t,
		selects: []string{
			"BIND",
			"NONE",
		},
		inputs: []string{
			"", // BIND: entry name
			"", // BIND: directory
			"", // BIND: filenameformat
			"", // NONE: entry name
		},
		multiSelects: [][]string{
			{"example.com"},
		},
		confirm: []bool{
			true,  // "Pick domains from the zone list?"
			false, // "Add another domain manually?"
			true,  // Write these files?

			false, // Run preview now?
		},
	}

	origRun := runSubcommand
	runSubcommand = func(*exec.Cmd) error { return nil }
	t.Cleanup(func() { runSubcommand = origRun })

	origVerify := verifyDNSProviderCredsFunc
	verifyDNSProviderCredsFunc = func(_ InitCredsEntry) ([]string, error) {
		return []string{"example.com"}, nil
	}
	t.Cleanup(func() { verifyDNSProviderCredsFunc = origVerify })

	origFetch := fetchZoneRecordsFunc
	fetchZoneRecordsFunc = func(_ InitCredsEntry, zone string) (models.Records, error) {
		aRecord := &models.RecordConfig{Type: "A", Name: "www", TTL: 300}
		aRecord.SetTarget("192.0.2.1")
		mxRecord := &models.RecordConfig{Type: "MX", Name: "@", TTL: 300, MxPreference: 10}
		mxRecord.SetTarget("mx.example.com.")
		soaRecord := &models.RecordConfig{Type: "SOA", Name: "@"}
		nsRecord := &models.RecordConfig{Type: "NS", Name: "@"}
		nsRecord.SetTarget("ns1.example.com.")
		return models.Records{aRecord, mxRecord, soaRecord, nsRecord}, nil
	}
	t.Cleanup(func() { fetchZoneRecordsFunc = origFetch })

	args := InitArgs{
		CredsFile:  filepath.Join(dir, "creds.json"),
		ConfigFile: filepath.Join(dir, "dnsconfig.js"),
	}
	if err := runInit(args, stub); err != nil {
		t.Fatalf("runInit: %v", err)
	}

	configBytes, err := os.ReadFile(args.ConfigFile)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	config := string(configBytes)
	if strings.Contains(config, `A("@", "1.2.3.4")`) {
		t.Errorf("config should not contain placeholder when import succeeded: %s", config)
	}
	if !strings.Contains(config, `A("www", "192.0.2.1")`) {
		t.Errorf("config missing imported A record: %s", config)
	}
	if !strings.Contains(config, `MX("@", 10, "mx.example.com.")`) {
		t.Errorf("config missing imported MX record: %s", config)
	}
	if strings.Contains(config, "SOA") {
		t.Errorf("config should not contain SOA record: %s", config)
	}
	if strings.Contains(config, "NAMESERVER") || strings.Contains(config, `NS("@"`) {
		t.Errorf("config should not contain apex NS record: %s", config)
	}
}

func TestRunInit_ImportFallback(t *testing.T) {
	dir := t.TempDir()

	stub := &stubAsker{
		t: t,
		selects: []string{
			"BIND",
			"NONE",
		},
		inputs: []string{
			"", // BIND: entry name
			"", // BIND: directory
			"", // BIND: filenameformat
			"", // NONE: entry name
		},
		multiSelects: [][]string{
			{"example.com"},
		},
		confirm: []bool{
			true,  // "Pick domains from the zone list?"
			false, // "Add another domain manually?"
			true,  // Write these files?

			false, // Run preview now?
		},
	}

	origRun := runSubcommand
	runSubcommand = func(*exec.Cmd) error { return nil }
	t.Cleanup(func() { runSubcommand = origRun })

	origVerify := verifyDNSProviderCredsFunc
	verifyDNSProviderCredsFunc = func(_ InitCredsEntry) ([]string, error) {
		return []string{"example.com"}, nil
	}
	t.Cleanup(func() { verifyDNSProviderCredsFunc = origVerify })

	origFetch := fetchZoneRecordsFunc
	fetchZoneRecordsFunc = func(_ InitCredsEntry, zone string) (models.Records, error) {
		return nil, fmt.Errorf("connection refused")
	}
	t.Cleanup(func() { fetchZoneRecordsFunc = origFetch })

	args := InitArgs{
		CredsFile:  filepath.Join(dir, "creds.json"),
		ConfigFile: filepath.Join(dir, "dnsconfig.js"),
	}
	if err := runInit(args, stub); err != nil {
		t.Fatalf("runInit: %v", err)
	}

	configBytes, err := os.ReadFile(args.ConfigFile)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(configBytes), `A("@", "1.2.3.4")`) {
		t.Errorf("config should contain placeholder when import failed: %s", configBytes)
	}
}
