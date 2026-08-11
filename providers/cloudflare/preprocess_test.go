package cloudflare

import (
	"net/netip"
	"strings"
	"testing"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/transform"
)

func newDomainConfig() *models.DomainConfig {
	return &models.DomainConfig{
		Name:     "test.com",
		Records:  []*models.RecordConfig{},
		Metadata: map[string]string{},
	}
}

func makeRCmeta(meta map[string]string) *models.RecordConfig {
	rc := models.RecordConfig{
		Type:     "A",
		Metadata: meta,
	}
	rc.SetLabel("foo", "example.tld")
	rc.MustSetTarget("1.2.3.4")
	return &rc
}

func TestPreprocess_BoolValidation(t *testing.T) {
	cf := &cloudflareProvider{}

	domain := newDomainConfig()
	domain.Records = append(domain.Records, makeRCmeta(map[string]string{metaProxy: "on"}))
	domain.Records = append(domain.Records, makeRCmeta(map[string]string{metaProxy: "fUll"}))
	domain.Records = append(domain.Records, makeRCmeta(map[string]string{}))
	domain.Records = append(domain.Records, makeRCmeta(map[string]string{metaProxy: "Off"}))
	domain.Records = append(domain.Records, makeRCmeta(map[string]string{metaProxy: "off"}))
	err := cf.preprocessConfig(domain)
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"on", "full", "off", "off", "off"}
	// make sure only "on" or "off", and "full" are actually set
	for i, rec := range domain.Records {
		if rec.Metadata[metaProxy] != expected[i] {
			t.Fatalf("At index %d: expect '%s' but found '%s'", i, expected[i], rec.Metadata[metaProxy])
		}
	}
}

func TestPreprocess_BoolValidation_Fails(t *testing.T) {
	cf := &cloudflareProvider{}
	domain := newDomainConfig()
	domain.Records = append(domain.Records, &models.RecordConfig{Metadata: map[string]string{metaProxy: "true"}})
	err := cf.preprocessConfig(domain)
	if err == nil {
		t.Fatal("Expected validation error, but got none")
	}
}

func TestPreprocess_DefaultProxy(t *testing.T) {
	cf := &cloudflareProvider{}
	domain := newDomainConfig()
	domain.Metadata[metaProxyDefault] = "full"
	domain.Records = append(domain.Records, makeRCmeta(map[string]string{metaProxy: "on"}))
	domain.Records = append(domain.Records, makeRCmeta(map[string]string{metaProxy: "off"}))
	domain.Records = append(domain.Records, makeRCmeta(map[string]string{}))
	err := cf.preprocessConfig(domain)
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"on", "off", "full"}
	for i, rec := range domain.Records {
		if rec.Metadata[metaProxy] != expected[i] {
			t.Fatalf("At index %d: expect '%s' but found '%s'", i, expected[i], rec.Metadata[metaProxy])
		}
	}
}

func TestPreprocess_DefaultProxy_Validation(t *testing.T) {
	cf := &cloudflareProvider{}
	domain := newDomainConfig()
	domain.Metadata[metaProxyDefault] = "true"
	err := cf.preprocessConfig(domain)
	if err == nil {
		t.Fatal("Expected validation error, but got none")
	}
}

func TestPreprocess_CNAMEFlattenProxyMutualExclusion(t *testing.T) {
	cf := &cloudflareProvider{}
	domain := newDomainConfig()
	rec := &models.RecordConfig{
		Type:     "CNAME",
		Metadata: map[string]string{metaCNAMEFlatten: "on", metaProxy: "on"},
	}
	rec.SetLabel("foo", "test.com")
	rec.MustSetTarget("example.com.")
	domain.Records = append(domain.Records, rec)
	err := cf.preprocessConfig(domain)
	if err == nil {
		t.Fatal("Expected validation error for CNAME with both flatten and proxy, but got none")
	}
}

// When CF_MANAGE_COMMENTS is enabled, preprocessConfig should ensure every
// record has the metaComment key in its metadata (empty string if not set).
// This is critical for modifyRecord, where the "ok" check on the metadata key
// determines whether to send the comment field to the API.
func TestPreprocess_ManageComments_SetsEmptyKey(t *testing.T) {
	cf := &cloudflareProvider{}
	domain := newDomainConfig()
	domain.Metadata[metaManageComments] = "true"

	// Record without any comment
	recNoComment := makeRCmeta(map[string]string{})
	// Record with a comment
	recWithComment := makeRCmeta(map[string]string{metaComment: "hello"})

	domain.Records = append(domain.Records, recNoComment, recWithComment)
	err := cf.preprocessConfig(domain)
	if err != nil {
		t.Fatal(err)
	}

	// Both records should have the metaComment key
	if _, ok := domain.Records[0].Metadata[metaComment]; !ok {
		t.Fatal("Expected metaComment key to exist on record without comment")
	}
	if domain.Records[0].Metadata[metaComment] != "" {
		t.Fatalf("Expected empty comment, got %q", domain.Records[0].Metadata[metaComment])
	}
	if domain.Records[1].Metadata[metaComment] != "hello" {
		t.Fatalf("Expected comment 'hello', got %q", domain.Records[1].Metadata[metaComment])
	}
}

// When CF_MANAGE_COMMENTS is NOT enabled, preprocessConfig should NOT add
// the metaComment key to records.
func TestPreprocess_NoManageComments_NoKey(t *testing.T) {
	cf := &cloudflareProvider{}
	domain := newDomainConfig()
	// No CF_MANAGE_COMMENTS

	rec := makeRCmeta(map[string]string{})
	domain.Records = append(domain.Records, rec)
	err := cf.preprocessConfig(domain)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := domain.Records[0].Metadata[metaComment]; ok {
		t.Fatal("Expected metaComment key to NOT exist when management is disabled")
	}
}

// When CF_MANAGE_TAGS is enabled, preprocessConfig should ensure every
// record has the metaTags key in its metadata (empty string if not set).
func TestPreprocess_ManageTags_SetsEmptyKey(t *testing.T) {
	cf := &cloudflareProvider{}
	domain := newDomainConfig()
	domain.Metadata[metaManageTags] = "true"

	// Record without any tags
	recNoTags := makeRCmeta(map[string]string{})
	// Record with tags
	recWithTags := makeRCmeta(map[string]string{metaTags: "tag1,tag2"})

	domain.Records = append(domain.Records, recNoTags, recWithTags)
	err := cf.preprocessConfig(domain)
	if err != nil {
		t.Fatal(err)
	}

	// Both records should have the metaTags key
	if _, ok := domain.Records[0].Metadata[metaTags]; !ok {
		t.Fatal("Expected metaTags key to exist on record without tags")
	}
	if domain.Records[0].Metadata[metaTags] != "" {
		t.Fatalf("Expected empty tags, got %q", domain.Records[0].Metadata[metaTags])
	}
	if domain.Records[1].Metadata[metaTags] != "tag1,tag2" {
		t.Fatalf("Expected tags 'tag1,tag2', got %q", domain.Records[1].Metadata[metaTags])
	}
}

// When CF_MANAGE_TAGS is NOT enabled, preprocessConfig should NOT add
// the metaTags key to records.
func TestPreprocess_NoManageTags_NoKey(t *testing.T) {
	cf := &cloudflareProvider{}
	domain := newDomainConfig()
	// No CF_MANAGE_TAGS

	rec := makeRCmeta(map[string]string{})
	domain.Records = append(domain.Records, rec)
	err := cf.preprocessConfig(domain)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := domain.Records[0].Metadata[metaTags]; ok {
		t.Fatal("Expected metaTags key to NOT exist when management is disabled")
	}
}

// genComparableWithMgmt: when management is enabled, records with no
// comment/tags should produce "comment=" / "tags=" (empty values), and
// records with values should include them.
func TestGenComparableWithMgmt(t *testing.T) {
	tests := []struct {
		name           string
		meta           map[string]string
		manageComments bool
		manageTags     bool
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:           "comments enabled, no comment on record",
			meta:           map[string]string{metaProxy: "off"},
			manageComments: true,
			wantContains:   []string{"comment="},
			wantNotContain: []string{"tags="},
		},
		{
			name:           "comments enabled, with comment",
			meta:           map[string]string{metaProxy: "off", metaComment: "hello"},
			manageComments: true,
			wantContains:   []string{"comment=hello"},
		},
		{
			name:           "tags enabled, no tags on record",
			meta:           map[string]string{metaProxy: "off"},
			manageTags:     true,
			wantContains:   []string{"tags="},
			wantNotContain: []string{"comment="},
		},
		{
			name:         "tags enabled, with tags",
			meta:         map[string]string{metaProxy: "off", metaTags: "a,b"},
			manageTags:   true,
			wantContains: []string{"tags=a,b"},
		},
		{
			name:           "both enabled, both empty",
			meta:           map[string]string{metaProxy: "off"},
			manageComments: true,
			manageTags:     true,
			wantContains:   []string{"comment=", "tags="},
		},
		{
			name:           "neither enabled",
			meta:           map[string]string{metaProxy: "off"},
			manageComments: false,
			manageTags:     false,
			wantNotContain: []string{"comment=", "tags="},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := makeRCmeta(tt.meta)
			got := genComparableWithMgmt(rec, tt.manageComments, tt.manageTags)
			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("expected comparable to contain %q, got %q", want, got)
				}
			}
			for _, notWant := range tt.wantNotContain {
				if strings.Contains(got, notWant) {
					t.Errorf("expected comparable to NOT contain %q, got %q", notWant, got)
				}
			}
		})
	}
}

func TestIpRewriting(t *testing.T) {
	tests := []struct {
		Given, Expected string
		Proxy           string
	}{
		// outside of range
		{"5.5.5.5", "5.5.5.5", "full"},
		{"5.5.5.5", "5.5.5.5", "on"},
		// inside range, but not proxied
		{"1.2.3.4", "1.2.3.4", "on"},
		// inside range and proxied
		{"1.2.3.4", "255.255.255.4", "full"},
	}
	cf := &cloudflareProvider{}
	domain := newDomainConfig()
	cf.ipConversions = []transform.IPConversion{{
		Low:      netip.MustParseAddr("1.2.3.0"),
		High:     netip.MustParseAddr("1.2.3.40"),
		NewBases: []netip.Addr{netip.MustParseAddr("255.255.255.0")},
		NewIPs:   nil,
	}}
	for _, tst := range tests {
		rec := &models.RecordConfig{Type: "A", Metadata: map[string]string{metaProxy: tst.Proxy}}
		rec.MustSetTarget(tst.Given)
		domain.Records = append(domain.Records, rec)
	}
	err := cf.preprocessConfig(domain)
	if err != nil {
		t.Fatal(err)
	}
	for i, tst := range tests {
		rec := domain.Records[i]
		if rec.GetTargetField() != tst.Expected {
			t.Fatalf("At index %d, expected target of %s, but found %s.", i, tst.Expected, rec.GetTargetField())
		}
		if tst.Proxy == "full" && tst.Given != tst.Expected && rec.Metadata[metaOriginalIP] != tst.Given {
			t.Fatalf("At index %d, expected original_ip to be set", i)
		}
	}
}
