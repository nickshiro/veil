package proxyman

import (
	"github.com/nickshiro/veil/common/geodata"
	"github.com/nickshiro/veil/common/session"
)

// InboundConfig enables the inbound manager.
type InboundConfig struct{}

// OutboundConfig enables the outbound manager.
type OutboundConfig struct{}

func BuildSniffingRequest(config *SniffingConfig) (session.SniffingRequest, error) {
	if config == nil {
		return session.SniffingRequest{}, nil
	}

	request := session.SniffingRequest{
		Enabled:                        config.Enabled,
		OverrideDestinationForProtocol: config.DestinationOverride,
		MetadataOnly:                   config.MetadataOnly,
		RouteOnly:                      config.RouteOnly,
	}
	if len(config.DomainsExcluded) > 0 {
		excludeForDomain, err := geodata.DomainReg.BuildDomainMatcher(config.DomainsExcluded)
		if err != nil {
			return session.SniffingRequest{}, err
		}
		request.ExcludeForDomain = excludeForDomain
	}
	if len(config.IpsExcluded) > 0 {
		excludeForIP, err := geodata.IPReg.BuildIPMatcher(config.IpsExcluded)
		if err != nil {
			return session.SniffingRequest{}, err
		}
		request.ExcludeForIP = excludeForIP
	}
	return request, nil
}
