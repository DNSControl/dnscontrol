package commands

import "github.com/DNSControl/dnscontrol/v4/pkg/providers"

// FYI(tlim): This file was originally called zonecache.go. To remove any
// confusion between it and pkg/zonecache, we've renamed it. We've also added
// "cmd" or "Cmd" to various labels too.

// NewCmdZoneCache creates a zoneCache.
func NewCmdZoneCache() *CmdZoneCache {
	return &CmdZoneCache{}
}

func (zc *CmdZoneCache) zoneList(name string, lister providers.ZoneLister) (*[]string, error) {
	zc.Lock()
	defer zc.Unlock()

	if zc.cache == nil {
		zc.cache = map[string]*[]string{}
	}

	if v, ok := zc.cache[name]; ok {
		return v, nil
	}

	zones, err := lister.ListZones()
	if err != nil {
		return nil, err
	}
	zc.cache[name] = &zones
	return &zones, nil
}
