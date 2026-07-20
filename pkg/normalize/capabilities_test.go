package normalize

import (
	"go/constant"
	"go/types"
	"sort"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

const (
	providersImportDir   = "../../pkg/providers"
	providersPackageName = "providers"
)

func TestCapabilitiesAreFiltered(t *testing.T) {
	// Any capabilities which we wish to whitelist because it's not directly
	// something we can test against.
	skipCheckCapabilities := make(map[string]struct{})
	// skipCheckCapabilities["CanUseBlahBlahBlah"] = struct{}{}

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo,
		Dir:  providersImportDir,
	}
	pkgs, err := packages.Load(cfg, ".")
	if err != nil || len(pkgs) == 0 {
		t.Fatalf("unable to load Go code from providers: %v", err)
	}
	var providers *packages.Package
	for _, pkg := range pkgs {
		if pkg.Name == providersPackageName {
			providers = pkg
			break
		}
	}
	if providers == nil {
		t.Fatalf("did not find package %q in %q", providersPackageName, providersImportDir)
	}

	constantNames := make([]string, 0, 50)
	capabilityInts := make(map[string]int, 50)

	for ident, obj := range providers.TypesInfo.Defs {
		if !strings.HasPrefix(ident.Name, "CanUse") {
			continue
		}
		c, ok := obj.(*types.Const)
		if !ok {
			continue
		}
		val, exact := constant.Int64Val(c.Val())
		if !exact {
			t.Fatalf("unable to get int value for %s", ident.Name)
		}
		constantNames = append(constantNames, ident.Name)
		capabilityInts[ident.Name] = int(val)
	}
	sort.Strings(constantNames)

	if len(providerCapabilityChecks) == 0 {
		t.Fatal("missing entries in providerCapabilityChecks")
	}

	capIntsToNames := make(map[int]string, len(providerCapabilityChecks))
	for _, pair := range providerCapabilityChecks {
		for _, cap := range pair.caps {
			capIntsToNames[int(cap)] = pair.rType
		}
	}

	for _, capName := range constantNames {
		capInt := capabilityInts[capName]
		if _, ok := skipCheckCapabilities[capName]; ok {
			t.Logf("ok: providers.%s (%d) is exempt from checkProviderCapabilities", capName, capInt)
		} else if rType, ok := capIntsToNames[capInt]; ok {
			t.Logf("ok: providers.%s (%d) is checked for with %q", capName, capInt, rType)
		} else {
			t.Errorf("MISSING: providers.%s (%d) is not checked by checkProviderCapabilities", capName, capInt)
		}
	}
}
