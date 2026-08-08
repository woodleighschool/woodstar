package handlers

import (
	"context"
	"log/slog"
	"net/netip"

	"github.com/woodleighschool/woodstar/internal/geoip"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/munki/mdp"
)

type geoIPLookup interface {
	Lookup(netip.Addr) (*geoip.Result, error)
}

func enrichHostPublicIPs(
	ctx context.Context,
	rows []hosts.Host,
	distribution *mdp.Store,
	geo geoIPLookup,
	logger *slog.Logger,
) {
	addresses := make([]netip.Addr, 0, len(rows))
	seen := make(map[netip.Addr]struct{}, len(rows))
	for i := range rows {
		if rows[i].PublicIP == nil {
			continue
		}
		address := rows[i].PublicIP.Unmap()
		if !address.IsValid() {
			continue
		}
		if _, ok := seen[address]; ok {
			continue
		}
		seen[address] = struct{}{}
		addresses = append(addresses, address)
	}

	candidates := make(map[netip.Addr][]mdp.ClientCandidate)
	if distribution != nil && len(addresses) > 0 {
		loaded, err := distribution.CandidatesForClients(ctx, addresses)
		if err != nil {
			logger.WarnContext(ctx, "load MDP candidates for host public IPs", "err", err)
		} else {
			candidates = loaded
		}
	}

	locations := make(map[netip.Addr]*geoip.Result, len(addresses))
	if geo != nil {
		for _, address := range addresses {
			location, err := geo.Lookup(address)
			if err != nil {
				logger.DebugContext(ctx, "look up host public IP", "err", err)
				continue
			}
			locations[address] = location
		}
	}

	for i := range rows {
		if rows[i].PublicIP == nil {
			continue
		}
		address := rows[i].PublicIP.Unmap()
		var details hosts.PublicIPDetails
		location := locations[address]
		if matches := candidates[address]; len(matches) > 0 {
			details.DistributionPoint = &hosts.PublicIPDistributionPoint{
				ID:   matches[0].ID,
				Name: matches[0].Name,
			}
		}
		if location != nil {
			details.City = location.City
			details.Region = location.Region
			details.CountryCode = location.CountryCode
			details.Country = location.Country
			details.Latitude = &location.Latitude
			details.Longitude = &location.Longitude
			details.ASN = location.ASN
			details.Organization = location.Organization
		}
		if details.DistributionPoint != nil || location != nil {
			rows[i].PublicIPDetails = &details
		}
	}
}
