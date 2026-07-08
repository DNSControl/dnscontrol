package main

// Functions for all tests in this directory.

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"testing"

	"github.com/DNSControl/dnscontrol/v4/pkg/credsfile"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
	"github.com/DNSControl/dnscontrol/v4/providers/cloudflare"
)

var (
	providerFlag         = flag.String("provider", "", "Provider to run (if empty, deduced from -profile)")
	profileFlag          = flag.String("profile", "", "Entry in profiles.json to use (if empty, copied from -provider)")
	enableCFWorkers      = flag.Bool("cfworkers", true, "enable CF worker tests (default false)")
	enableCFRedirectMode = flag.Bool("cfredirect", true, "enable CF SingleRedirect tests (default false)")
	enableCFFlatten      = flag.Bool("cfflatten", false, "enable CF CNAME flattening tests (requires paid plan, default false)")
	enableCFTags         = flag.Bool("cftags", false, "enable CF tag tests (requires paid plan, default false)")
)

func init() {
	testing.Init()

	flag.Parse()
}

// ---

func panicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}

func getProvider(t *testing.T) (providers.DNSServiceProvider, string, map[string]string) {
	if *providerFlag == "" && *profileFlag == "" {
		t.Log("No -provider or -profile specified")
		return nil, "", nil
	}

	// Load the profile values

	jsons, err := credsfile.LoadProviderConfigs("profiles.json")
	if err != nil {
		t.Fatalf("Error loading provider configs: %s", err)
	}

	// Which profile are we using? Use the profile but default to the provider.
	targetProfile := *profileFlag
	if targetProfile == "" {
		targetProfile = *providerFlag
	}

	var profileName, profileType string
	var cfg map[string]string

	// Find the profile we want to use.
	for p, c := range jsons {
		if p == targetProfile {
			cfg = c
			profileName = p
			profileType = cfg["TYPE"]
			if profileType == "" {
				t.Fatalf("profiles.json profile %q does not have a TYPE field", *profileFlag)
			}
			break
		}
	}
	if profileName == "" {
		t.Fatalf("Profile not found: -profile=%q -provider=%q", *profileFlag, *providerFlag)
		return nil, "", nil
	}

	// Fill in -profile if blank.
	if *profileFlag == "" {
		*profileFlag = profileName
	}
	// Fill in -provider if blank.
	if *providerFlag == "" {
		*providerFlag = profileType
	}

	// Sanity check. If the user-specifed -provider flag doesn't match what was in the file, warn them.
	if *providerFlag != profileType {
		fmt.Printf("WARNING: -provider=%q does not match profile TYPE=%q.  Using profile TYPE.\n", *providerFlag, profileType)
		*providerFlag = profileType
	}

	// fmt.Printf("DEBUG flag=%q Profile=%q TYPE=%q\n", *providerFlag, profileName, profileType)
	fmt.Printf("Testing Profile=%q (TYPE=%q)\n", profileName, profileType)

	var metadata json.RawMessage

	// CLOUDFLAREAPI tests related to CLOUDFLAREAPI_SINGLE_REDIRECT/CF_REDIRECT/CF_TEMP_REDIRECT
	// requires metadata to enable this feature.
	// In hindsight, I have no idea why this metadata flag is required to
	// use this feature. Maybe because we didn't have the capabilities
	// feature at the time?
	if profileType == "CLOUDFLAREAPI" {
		items := []string{}
		if *enableCFWorkers {
			items = append(items, `"manage_workers": true`)
		}
		if *enableCFRedirectMode {
			items = append(items, `"manage_single_redirects": true`)
		}
		metadata = []byte(`{ ` + strings.Join(items, `, `) + ` }`)
	}

	provider, err := providers.CreateDNSProvider(profileType, cfg, metadata)
	if err != nil {
		t.Fatal(err)
	}

	if profileType == "CLOUDFLAREAPI" && *enableCFWorkers {
		// Cloudflare only. Will do nothing if provider != *cloudflareProvider.
		if err := cloudflare.PrepareCloudflareTestWorkers(provider); err != nil {
			t.Fatal(err)
		}
	}

	return provider, cfg["domain"], cfg
}
