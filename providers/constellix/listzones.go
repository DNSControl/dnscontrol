package constellix

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/printer"
	"github.com/DNSControl/dnscontrol/v5/pkg/zonecache"
)

// zone is what the provider keeps about a zone of the account.
type zone struct {
	id          flexID
	nameservers []string
	template    bool

	// err is set for a zone this run tried to create but could not read
	// back with its nameservers (or could not create at all). Every use of
	// the zone returns it until a later run lists the zone normally.
	err error
}

func zoneFromAPI(d apiDomain) zone {
	t := bytes.TrimSpace(d.Template)
	return zone{
		id:          d.ID,
		nameservers: d.Nameservers,
		template:    len(t) > 0 && !bytes.Equal(t, []byte("null")),
	}
}

// fetchZones loads the zone cache.
func (p *constellixProvider) fetchZones() (map[string]zone, error) {
	domains, err := p.client.listDomains()
	if err != nil {
		return nil, err
	}

	zones := make(map[string]zone, len(domains))
	for _, d := range domains {
		zones[d.Name] = zoneFromAPI(d)
	}
	return zones, nil
}

// getZone returns a usable zone, zonecache.ErrZoneNotFound if the account does
// not have it, or the error of a creation attempt that left it unusable.
func (p *constellixProvider) getZone(domain string) (zone, error) {
	z, err := p.zones.GetZone(domain)
	if err != nil {
		return zone{}, err
	}
	return z, z.err
}

// getZoneForRecords is getZone for the record operations, which need the zone
// to exist.
func (p *constellixProvider) getZoneForRecords(domain string) (zone, error) {
	z, err := p.getZone(domain)
	if errors.Is(err, zonecache.ErrZoneNotFound) {
		return zone{}, fmt.Errorf("CONSTELLIX: zone %s: %w", domain, err)
	}
	return z, err
}

// ListZones returns every zone the credentials can see.
func (p *constellixProvider) ListZones() ([]string, error) {
	return p.zones.GetZoneNames()
}

// EnsureZoneExists creates the zone if the account does not have it yet.
//
// The new zone is read back once for its nameservers, which push asks for
// right after creating zones. If that read fails, or the creation itself
// does, the zone is remembered with the error so that this run does not
// carry on as if the zone had no nameservers.
func (p *constellixProvider) EnsureZoneExists(dc *models.DomainConfig) error {
	z, err := p.zones.GetZone(dc.Name)
	if err == nil {
		return z.err
	}
	if !errors.Is(err, zonecache.ErrZoneNotFound) {
		return err
	}

	id, err := p.client.createDomain(dc.Name)
	if err != nil {
		err = fmt.Errorf("creating zone %s failed: %w", dc.Name, err)
		p.zones.SetZone(dc.Name, zone{err: err})
		return err
	}

	d, err := p.client.getDomain(id)
	if err == nil && len(d.Nameservers) == 0 {
		err = errors.New("CONSTELLIX: the API lists no nameservers for it")
	}
	if err != nil {
		err = fmt.Errorf("zone %s was created but is not readable yet, run push again: %w", dc.Name, err)
		p.zones.SetZone(dc.Name, zone{id: id, err: err})
		return err
	}

	p.zones.SetZone(dc.Name, zoneFromAPI(d))
	printer.Printf("Added zone for %s to Constellix account: %s\n", dc.Name, id)
	return nil
}
