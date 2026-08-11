package commands

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v4/models"
	_ "github.com/DNSControl/dnscontrol/v4/pkg/providers/_all"
	_ "github.com/DNSControl/dnscontrol/v4/pkg/rtype"
)

func TestR53Test_1(t *testing.T) {
	rec := models.RecordConfig{
		Type:     "R53_ALIAS",
		Name:     "foo",
		NameFQDN: "foo.domain.tld",
	}
	rec.MustSetTarget("bar")
	rec.R53Alias = make(map[string]string)
	rec.R53Alias["type"] = "A"
	w := `R53_ALIAS("foo", "A", "bar")`
	if g := makeR53alias(&rec, 0); g != w {
		t.Errorf("makeR53alias failure: got `%s` want `%s`", g, w)
	}
}

func TestR53Test_1ttl(t *testing.T) {
	rec := models.RecordConfig{
		Type:     "R53_ALIAS",
		Name:     "foo",
		NameFQDN: "foo.domain.tld",
	}
	rec.MustSetTarget("bar")
	rec.R53Alias = make(map[string]string)
	rec.R53Alias["type"] = "A"
	w := `R53_ALIAS("foo", "A", "bar", TTL(321))`
	if g := makeR53alias(&rec, 321); g != w {
		t.Errorf("makeR53alias failure: got `%s` want `%s`", g, w)
	}
}

func TestR53Test_2(t *testing.T) {
	rec := models.RecordConfig{
		Type:     "R53_ALIAS",
		Name:     "foo",
		NameFQDN: "foo.domain.tld",
	}
	rec.MustSetTarget("bar")
	rec.R53Alias = make(map[string]string)
	rec.R53Alias["type"] = "A"
	rec.R53Alias["zone_id"] = "blarg"
	w := `R53_ALIAS("foo", "A", "bar", R53_ZONE("blarg"))`
	if g := makeR53alias(&rec, 0); g != w {
		t.Errorf("makeR53alias failure: got `%s` want `%s`", g, w)
	}
}

func TestR53Test_2ttl(t *testing.T) {
	rec := models.RecordConfig{
		Type:     "R53_ALIAS",
		Name:     "foo",
		NameFQDN: "foo.domain.tld",
	}
	rec.MustSetTarget("bar")
	rec.R53Alias = make(map[string]string)
	rec.R53Alias["type"] = "A"
	rec.R53Alias["zone_id"] = "blarg"
	w := `R53_ALIAS("foo", "A", "bar", R53_ZONE("blarg"), TTL(123))`
	if g := makeR53alias(&rec, 123); g != w {
		t.Errorf("makeR53alias failure: got `%s` want `%s`", g, w)
	}
}

func TestR53Test_3(t *testing.T) {
	rec := models.RecordConfig{
		Type:     "R53_ALIAS",
		Name:     "foo",
		NameFQDN: "foo.domain.tld",
	}
	rec.MustSetTarget("bar")
	rec.R53Alias = make(map[string]string)
	rec.R53Alias["type"] = "A"
	rec.R53Alias["evaluate_target_health"] = "true"
	w := `R53_ALIAS("foo", "A", "bar", R53_EVALUATE_TARGET_HEALTH(true))`
	if g := makeR53alias(&rec, 0); g != w {
		t.Errorf("makeR53alias failure: got `%s` want `%s`", g, w)
	}
}

func TestR53Test_3ttl(t *testing.T) {
	rec := models.RecordConfig{
		Type:     "R53_ALIAS",
		Name:     "foo",
		NameFQDN: "foo.domain.tld",
	}
	rec.MustSetTarget("bar")
	rec.R53Alias = make(map[string]string)
	rec.R53Alias["type"] = "A"
	rec.R53Alias["evaluate_target_health"] = "true"
	w := `R53_ALIAS("foo", "A", "bar", R53_EVALUATE_TARGET_HEALTH(true), TTL(123))`
	if g := makeR53alias(&rec, 123); g != w {
		t.Errorf("makeR53alias failure: got `%s` want `%s`", g, w)
	}
}

func TestR53Test_4(t *testing.T) {
	rec := models.RecordConfig{
		Type:     "R53_ALIAS",
		Name:     "foo",
		NameFQDN: "foo.domain.tld",
	}
	rec.MustSetTarget("bar")
	rec.R53Alias = make(map[string]string)
	rec.R53Alias["type"] = "A"
	rec.R53Alias["zone_id"] = "blarg"
	rec.R53Alias["evaluate_target_health"] = "true"
	w := `R53_ALIAS("foo", "A", "bar", R53_ZONE("blarg"), R53_EVALUATE_TARGET_HEALTH(true))`
	if g := makeR53alias(&rec, 0); g != w {
		t.Errorf("makeR53alias failure: got `%s` want `%s`", g, w)
	}
}

func TestR53Test_4ttl(t *testing.T) {
	rec := models.RecordConfig{
		Type:     "R53_ALIAS",
		Name:     "foo",
		NameFQDN: "foo.domain.tld",
	}
	rec.MustSetTarget("bar")
	rec.R53Alias = make(map[string]string)
	rec.R53Alias["type"] = "A"
	rec.R53Alias["zone_id"] = "blarg"
	rec.R53Alias["evaluate_target_health"] = "true"
	w := `R53_ALIAS("foo", "A", "bar", R53_ZONE("blarg"), R53_EVALUATE_TARGET_HEALTH(true), TTL(123))`
	if g := makeR53alias(&rec, 123); g != w {
		t.Errorf("makeR53alias failure: got `%s` want `%s`", g, w)
	}
}
