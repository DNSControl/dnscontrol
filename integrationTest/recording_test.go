package main

// Test where a recording is written.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

func TestARecordingIsNamedAfterTheProvidersPackageNotItsType(t *testing.T) {
	provider, err := providers.CreateDNSProvider("GANDI_V5", map[string]string{"token": "not used"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	name, err := providergolden.ProviderName(provider)
	if err != nil {
		t.Fatalf("ProviderName() error: %v", err)
	}
	if name != "gandiv5" {
		t.Errorf("ProviderName() = %q, want %q", name, "gandiv5")
	}
}

func TestRecordingDirResolvesRecordDirFromTheModuleRoot(t *testing.T) {
	rel := filepath.Join("providers", "bind", "test_data")

	old := *recordDirFlag
	*recordDirFlag = rel
	defer func() { *recordDirFlag = old }()

	dir, err := recordingDir(nil)
	if err != nil {
		t.Fatalf("recordingDir() error: %v", err)
	}
	if !filepath.IsAbs(dir) {
		t.Fatalf("recordingDir() = %q, want an absolute path", dir)
	}

	root := strings.TrimSuffix(dir, string(filepath.Separator)+rel)
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Errorf("recordingDir() = %q, want a path under the directory holding go.mod: %v", dir, err)
	}
}
