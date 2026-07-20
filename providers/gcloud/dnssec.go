package gcloud

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/DNSControl/dnscontrol/v4/models"
	gdns "google.golang.org/api/dns/v1"
)

func (g *gcloudProvider) getDnssecCorrections(dc *models.DomainConfig) ([]*models.Correction, error) {
	// Don't allow combining AUTODNSSEC_{ON,OFF} with metadata DnssecConfig
	if dc.AutoDNSSEC != "" && dc.Metadata["DnssecConfig"] != "" {
		return nil, fmt.Errorf("cannot use AUTODNSSEC and DnssecConfig-metadata at the same time")
	}
	if dc.Metadata["DnssecConfig"] != "" {
		return g.getDnssecCorrectionsFromMetadata(dc)
	}

	enabled, err := g.isDnssecEnabled(dc.Name)
	if err != nil {
		return nil, err
	}
	if enabled && dc.AutoDNSSEC == "off" {
		return []*models.Correction{
			{
				Msg: "Disable AUTODNSSEC",
				F:   func() error { err := g.disableDnssec(dc.Name); return err },
			},
		}, nil
	}

	if !enabled && dc.AutoDNSSEC == "on" {
		return []*models.Correction{
			{
				Msg: "Enable AUTODNSSEC",
				F:   func() error { err := g.enableDnssec(dc.Name); return err },
			},
		}, nil
	}
	return nil, nil
}

func (g *gcloudProvider) isDnssecEnabled(domain string) (bool, error) {
	// Zones that never had DNSSEC enabled will have nil in the DnssecConfig field
	if g.zones[domain+"."].DnssecConfig == nil {
		return false, nil
	}
	if g.zones[domain+"."].DnssecConfig.State == "on" {
		return true, nil
	}
	// Google Cloud DNS has a "transfer" state for DNSSEC. We treat it as "on".
	// It basically means that DNSSEC is enabled, but Google won't rotate ZSKs.
	if g.zones[domain+"."].DnssecConfig.State == "transfer" {
		return true, nil
	}
	return false, nil
}

func (g *gcloudProvider) enableDnssec(domain string) error {
	dnssecPatch := &gdns.ManagedZone{
		DnssecConfig: &gdns.ManagedZoneDnsSecConfig{
			State: "on",
		},
	}
	resp, err := g.client.ManagedZones.Patch(g.project, g.zones[domain+"."].Name, dnssecPatch).Do()
	if err != nil {
		return err
	}
	if resp.Status != "done" {
		// Should we have a timeout here?
		for {
			checkOperation, err := g.client.ManagedZoneOperations.Get(g.project, g.zones[domain+"."].Name, resp.Id).Do()
			if err != nil {
				return err
			}
			if checkOperation.Status == "done" {
				return nil
			}
			time.Sleep(2 * time.Second)
		}
	}
	return nil
}

func (g *gcloudProvider) disableDnssec(domain string) error {
	dnssecPatch := &gdns.ManagedZone{
		DnssecConfig: &gdns.ManagedZoneDnsSecConfig{
			State: "off",
		},
	}
	resp, err := g.client.ManagedZones.Patch(g.project, g.zones[domain+"."].Name, dnssecPatch).Do()
	if err != nil {
		return err
	}
	if resp.Status != "done" {
		// Should we have a timeout here?
		for {
			checkOperation, err := g.client.ManagedZoneOperations.Get(g.project, g.zones[domain+"."].Name, resp.Id).Do()
			if err != nil {
				return err
			}
			if checkOperation.Status == "done" {
				return nil
			}
			time.Sleep(2 * time.Second)
		}
	}
	return nil
}

func (g *gcloudProvider) getDnssecCorrectionsFromMetadata(dc *models.DomainConfig) ([]*models.Correction, error) {
	var corrections []*models.Correction

	// Get current state
	current := g.zones[dc.Name+"."].DnssecConfig
	if current == nil {
		current = &gdns.ManagedZoneDnsSecConfig{}
	}
	jsonStr := dc.Metadata["DnssecConfig"]
	if jsonStr == "" {
		return nil, nil
	}

	desired := &gdns.ManagedZoneDnsSecConfig{}
	if err := json.Unmarshal([]byte(jsonStr), desired); err != nil {
		return nil, fmt.Errorf("invalid DnssecConfig JSON for %s: %w", dc.Name, err)
	}

	changes := getDnssecDiff(current, desired)
	if len(changes) == 0 {
		return nil, nil
	}

	msg := fmt.Sprintf("UPDATE DNSSEC from metadata: %s", strings.Join(changes, ", "))
	corrections = append(corrections, &models.Correction{
		Msg: msg,
		F: func() error {
			err := g.updateDnssecFromMetadata(dc.Name, desired)
			return err
		},
	})
	return corrections, nil
}

func getDnssecDiff(current, desired *gdns.ManagedZoneDnsSecConfig) []string {
	var diffs []string

	if desired.State != "" && desired.State != current.State {
		diffs = append(diffs, fmt.Sprintf("State: %q -> %q", current.State, desired.State))
	}

	if desired.NonExistence != "" && desired.NonExistence != current.NonExistence {
		diffs = append(diffs, fmt.Sprintf("NonExistence: %q -> %q", current.NonExistence, desired.NonExistence))
	}

	if len(desired.DefaultKeySpecs) > 0 {
		keySpecDiffs := getKeySpecDiffs(current.DefaultKeySpecs, desired.DefaultKeySpecs)
		if len(keySpecDiffs) != 0 {
			diffs = append(diffs, keySpecDiffs...)
		}
	}
	return diffs
}

func getKeySpecDiffs(current, desired []*gdns.DnsKeySpec) []string {
	var details []string
	currentMap := make(map[string]bool)
	for _, k := range current {
		currentMap[keySpecToString(k)] = true
	}

	desiredMap := make(map[string]bool)
	for _, k := range desired {
		desiredMap[keySpecToString(k)] = true
	}
	for _, k := range current {
		fp := keySpecToString(k)
		if !desiredMap[fp] {
			details = append(details, fmt.Sprintf("Remove key %s", fp))
		}
	}

	for _, k := range desired {
		fp := keySpecToString(k)
		if !currentMap[fp] {
			details = append(details, fmt.Sprintf("Add key %s", fp))
		}
	}
	return details
}

func keySpecToString(ks *gdns.DnsKeySpec) string {
	return fmt.Sprintf("alg:%s|len:%d|type:%s", ks.Algorithm, ks.KeyLength, ks.KeyType)
}

func (g *gcloudProvider) updateDnssecFromMetadata(domain string, desired *gdns.ManagedZoneDnsSecConfig) error {
	dnssecPatch := &gdns.ManagedZone{
		DnssecConfig: desired,
	}
	resp, err := g.client.ManagedZones.Patch(g.project, g.zones[domain+"."].Name, dnssecPatch).Do()
	if err != nil {
		return err
	}
	if resp.Status != "done" {
		// Should we have a timeout here?
		for {
			checkOperation, err := g.client.ManagedZoneOperations.Get(g.project, g.zones[domain+"."].Name, resp.Id).Do()
			if err != nil {
				return err
			}
			if checkOperation.Status == "done" {
				return nil
			}
			time.Sleep(2 * time.Second)
		}
	}
	return nil
}
