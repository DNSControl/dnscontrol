package main

// Test where a recording is written.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRecordingDirResolvesRecordDirFromTheModuleRoot(t *testing.T) {
	rel := filepath.Join("providers", "bind", "testdata")

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
