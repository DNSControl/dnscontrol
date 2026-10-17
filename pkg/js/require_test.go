package js

import (
	"os"
	"path/filepath"
	"testing"

	testifyrequire "github.com/stretchr/testify/require"
)

// TestRequireStaysInAllowedDirectories checks that require() and
// require_glob() only read files inside the working directory or the
// directory of the main configuration file.
func TestRequireStaysInAllowedDirectories(t *testing.T) {
	tmp := t.TempDir()
	work := filepath.Join(tmp, "work")     // the working directory
	cfgDir := filepath.Join(tmp, "config") // holds the main configuration file
	outside := filepath.Join(tmp, "outside")
	for _, dir := range []string{filepath.Join(cfgDir, "zones"), filepath.Join(work, "lib"), outside} {
		testifyrequire.NoError(t, os.MkdirAll(dir, 0o755))
	}
	writeFile := func(name, content string) {
		t.Helper()
		testifyrequire.NoError(t, os.WriteFile(name, []byte(content), 0o644))
	}
	writeFile(filepath.Join(outside, "leak.js"), `var VALUE = "outside";`)
	writeFile(filepath.Join(work, "lib", "value.js"), `var VALUE = "lib";`)
	writeFile(filepath.Join(cfgDir, "value.js"), `var VALUE = "config";`)
	writeFile(filepath.Join(cfgDir, "zones", "value.js"), `var VALUE = "zones";`)
	hasSymlink := os.Symlink(filepath.Join(outside, "leak.js"), filepath.Join(cfgDir, "link.js")) == nil

	t.Chdir(work)

	tests := []struct {
		name    string
		require string // the statement that loads VALUE
		symlink bool
		want    string // expected VALUE
		wantErr string // expected error text, if the file must not be read
	}{
		{name: "relative to the config file", require: `require("./value.js")`, want: "config"},
		{name: "relative to the working directory", require: `require("lib/value.js")`, want: "lib"},
		{name: "absolute path inside", require: `require(` + jsString(filepath.Join(work, "lib", "value.js")) + `)`, want: "lib"},
		{name: "glob inside", require: `require_glob("./zones/")`, want: "zones"},
		{name: "absolute path outside", require: `require(` + jsString(filepath.Join(outside, "leak.js")) + `)`, wantErr: "outside"},
		{name: "dot-dot outside", require: `require("./../outside/leak.js")`, wantErr: "outside"},
		{name: "relative to the working directory outside", require: `require("lib/../../outside/leak.js")`, wantErr: "outside"},
		{name: "symlink to outside", require: `require("./link.js")`, symlink: true, wantErr: "escapes"},
		{name: "glob outside", require: `require_glob("./../outside/")`, wantErr: "outside"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.symlink && !hasSymlink {
				t.Skip("symlinks are not supported here")
			}
			config := filepath.Join(cfgDir, "dnsconfig.js")
			writeFile(config, tt.require+`;
var REG = NewRegistrar("none");
D("example.com", REG, TXT("@", VALUE));
`)

			conf, err := ExecuteJavaScript(config, false, nil)
			if tt.wantErr != "" {
				testifyrequire.ErrorContains(t, err, tt.wantErr)
				return
			}
			testifyrequire.NoError(t, err)
			testifyrequire.Len(t, conf.Domains, 1)
			testifyrequire.Len(t, conf.Domains[0].Records, 1)
			testifyrequire.Equal(t, tt.want, conf.Domains[0].Records[0].GetTargetTXTJoined())
		})
	}
}

// jsString quotes s as a JavaScript string literal.
func jsString(s string) string {
	return `"` + filepath.ToSlash(s) + `"`
}
