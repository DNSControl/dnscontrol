package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

func TestRefineProviderAliases(t *testing.T) {
	for _, explicit := range []string{"", "-", providerSyntaxTestType, "TEST_PROVIDER_SYNTAX_ALIAS"} {
		for _, credentialType := range []string{providerSyntaxTestType, "TEST_PROVIDER_SYNTAX_ALIAS"} {
			got, _, err := refineProviderType("account", explicit, map[string]string{"TYPE": credentialType}, "NewDnsProvider")
			require.NoError(t, err)
			require.Equal(t, providerSyntaxTestType, got)
		}
	}
	for _, fields := range []map[string]string{nil, {"token": "test"}} {
		got, warning, err := refineProviderType("account", "TEST_PROVIDER_SYNTAX_ALIAS", fields, "NewDnsProvider")
		require.NoError(t, err)
		require.Equal(t, providerSyntaxTestType, got)
		require.Contains(t, warning, "TEST_PROVIDER_SYNTAX_ALIAS", "diagnostics should retain the user's spelling")
	}
	_, _, err := refineProviderType("account", "TEST_PROVIDER_SYNTAX_ALIAS", map[string]string{"TYPE": "NONE"}, "NewDnsProvider")
	require.ErrorContains(t, err, "Mismatch")
	require.ErrorContains(t, err, "TEST_PROVIDER_SYNTAX_ALIAS")
}

func TestWizardListsCanonicalDefinitions(t *testing.T) {
	for _, role := range []providers.ProviderKind{providers.KindDNS, providers.KindRegistrar} {
		names := providerNamesForRole(role)
		require.True(t, slices.IsSorted(names))
		require.Contains(t, names, providerSyntaxTestType)
		require.Equal(t, len(names), len(slices.Compact(slices.Clone(names))))
		require.NotContains(t, names, "TEST_PROVIDER_SYNTAX_ALIAS")
	}
	require.Equal(t, "Syntax test", displayName("TEST_PROVIDER_SYNTAX_ALIAS"))
}

func TestGetZonesCLIAliases(t *testing.T) {
	for _, tc := range []struct {
		name, credentialType string
		args                 []string
	}{
		{"credentials alias", "TEST_PROVIDER_SYNTAX_ALIAS", []string{"all"}},
		{"explicit alias", providerSyntaxTestType, []string{"TEST_PROVIDER_SYNTAX_ALIAS", "all"}},
		{"multiple zones", "TEST_PROVIDER_SYNTAX_ALIAS", []string{"one.example", "two.example"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _ = registerSyntaxTestProvider(t)
			dir := t.TempDir()
			creds, output := filepath.Join(dir, "creds.json"), filepath.Join(dir, "zones.txt")
			require.NoError(t, os.WriteFile(creds, []byte(fmt.Sprintf(`{"account":{"TYPE":%q}}`, tc.credentialType)), 0o600))
			var getZones *cli.Command
			for _, command := range commands {
				if command.Name == "get-zones" {
					getZones = command
					break
				}
			}
			require.NotNil(t, getZones)
			app := &cli.Command{Commands: []*cli.Command{getZones}}
			args := []string{"dnscontrol", "get-zones", "--creds", creds, "--format=nameonly", "--out", output, "account"}
			require.NoError(t, app.Run(context.Background(), append(args, tc.args...)))
			data, err := os.ReadFile(output)
			require.NoError(t, err)
			want := "example.com\n"
			if tc.name == "multiple zones" {
				want = strings.Join(tc.args, "\n") + "\n"
			}
			require.Equal(t, want, string(data))
		})
	}
}
