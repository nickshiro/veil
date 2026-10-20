package conf

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/nickshiro/veil/app/dispatcher"
	"github.com/nickshiro/veil/app/proxyman"
	"github.com/nickshiro/veil/app/stats"
	"github.com/nickshiro/veil/common/errors"
	"github.com/nickshiro/veil/common/geodata"
	"github.com/nickshiro/veil/common/net"
	"github.com/nickshiro/veil/common/serial"
	core "github.com/nickshiro/veil/core"
	"github.com/nickshiro/veil/proxy/freedom"
	"github.com/nickshiro/veil/proxy/masque"
	"github.com/nickshiro/veil/proxy/vless"
	"github.com/nickshiro/veil/proxy/vless/inbound"
	"github.com/nickshiro/veil/proxy/vless/outbound"
	"github.com/nickshiro/veil/transport/internet"
	"google.golang.org/protobuf/proto"
)

var (
	inboundConfigLoader = NewJSONConfigLoader(ConfigCreatorCache{
		"tunnel":        func() any { return new(DokodemoConfig) },
		"dokodemo-door": func() any { return new(DokodemoConfig) },
		"http":          func() any { return new(HTTPServerConfig) },
		"shadowsocks":   func() any { return new(ShadowsocksServerConfig) },
		"mixed":         func() any { return new(SocksServerConfig) },
		"socks":         func() any { return new(SocksServerConfig) },
		"vless":         func() any { return new(VLessInboundConfig) },
		"vmess":         func() any { return new(VMessInboundConfig) },
		"trojan":        func() any { return new(TrojanServerConfig) },
		"wireguard":     func() any { return &WireGuardConfig{IsClient: false} },
		"hysteria":      func() any { return new(HysteriaServerConfig) },
		"masque":        func() any { return new(MasqueServerConfig) },
		"tun":           func() any { return new(TunConfig) },
	}, "protocol", "settings")

	outboundConfigLoader = NewJSONConfigLoader(ConfigCreatorCache{
		"block":       func() any { return new(BlackholeConfig) },
		"blackhole":   func() any { return new(BlackholeConfig) },
		"loopback":    func() any { return new(LoopbackConfig) },
		"direct":      func() any { return new(FreedomConfig) },
		"freedom":     func() any { return new(FreedomConfig) },
		"http":        func() any { return new(HTTPClientConfig) },
		"shadowsocks": func() any { return new(ShadowsocksClientConfig) },
		"socks":       func() any { return new(SocksClientConfig) },
		"vless":       func() any { return new(VLessOutboundConfig) },
		"vmess":       func() any { return new(VMessOutboundConfig) },
		"trojan":      func() any { return new(TrojanClientConfig) },
		"hysteria":    func() any { return new(HysteriaClientConfig) },
		"masque":      func() any { return new(MasqueClientConfig) },
		"dns":         func() any { return new(DNSOutboundConfig) },
		"wireguard":   func() any { return &WireGuardConfig{IsClient: true} },
	}, "protocol", "settings")
)

type SniffingConfig struct {
	Enabled         bool       `json:"enabled"`
	DestOverride    StringList `json:"destOverride"`
	DomainsExcluded StringList `json:"domainsExcluded"`
	IPsExcluded     StringList `json:"ipsExcluded"`
	MetadataOnly    bool       `json:"metadataOnly"`
	RouteOnly       bool       `json:"routeOnly"`
}

// Build implements Buildable.
func (c *SniffingConfig) Build() (*proxyman.SniffingConfig, error) {
	var protocols []string
	for _, protocol := range c.DestOverride {
		switch strings.ToLower(protocol) {
		case "http":
			protocols = append(protocols, "http")
		case "tls", "https", "ssl":
			protocols = append(protocols, "tls")
		case "quic":
			protocols = append(protocols, "quic")
		case "fakedns", "fakedns+others":
			protocols = append(protocols, "fakedns")
		default:
			return nil, errors.New("unknown protocol: ", protocol)
		}
	}

	domains, err := geodata.ParseDomainRules(c.DomainsExcluded, geodata.Domain_Substr)
	if err != nil {
		return nil, err
	}

	ips, err := geodata.ParseIPRules(c.IPsExcluded)
	if err != nil {
		return nil, err
	}

	return &proxyman.SniffingConfig{
		Enabled:             c.Enabled,
		DestinationOverride: protocols,
		DomainsExcluded:     domains,
		IpsExcluded:         ips,
		MetadataOnly:        c.MetadataOnly,
		RouteOnly:           c.RouteOnly,
	}, nil
}

type MuxConfig struct {
	Enabled         bool   `json:"enabled"`
	Concurrency     int16  `json:"concurrency"`
	XudpConcurrency int16  `json:"xudpConcurrency"`
	XudpProxyUDP443 string `json:"xudpProxyUDP443"`
}

// Build creates MultiplexingConfig, Concurrency < 0 completely disables mux.
func (m *MuxConfig) Build() (*proxyman.MultiplexingConfig, error) {
	switch m.XudpProxyUDP443 {
	case "":
		m.XudpProxyUDP443 = "reject"
	case "reject", "allow", "skip":
	default:
		return nil, errors.New(`unknown "xudpProxyUDP443": `, m.XudpProxyUDP443)
	}
	return &proxyman.MultiplexingConfig{
		Enabled:         m.Enabled,
		Concurrency:     int32(m.Concurrency),
		XudpConcurrency: int32(m.XudpConcurrency),
		XudpProxyUDP443: m.XudpProxyUDP443,
	}, nil
}

type InboundDetourConfig struct {
	Protocol       string           `json:"protocol"`
	PortList       *PortList        `json:"port"`
	ListenOn       *Address         `json:"listen"`
	Settings       *json.RawMessage `json:"settings"`
	Tag            string           `json:"tag"`
	StreamSetting  *StreamConfig    `json:"streamSettings"`
	SniffingConfig *SniffingConfig  `json:"sniffing"`
}

// Build implements Buildable.
func (c *InboundDetourConfig) Build() (*core.InboundHandlerConfig, error) {
	receiverSettings := &proxyman.ReceiverConfig{}

	// TUN inbound doesn't need port configuration as it uses network interface instead
	if strings.ToLower(c.Protocol) == "tun" {
		// Skip port validation for TUN
	} else if c.ListenOn == nil || len(c.ListenOn.String()) == 0 {
		// Listen on anyip, must set PortList
		if c.PortList == nil {
			return nil, errors.New("Listen on AnyIP but no Port(s) set in InboundDetour.")
		}
		receiverSettings.PortList = c.PortList.Build()
	} else {
		// Listen on specific IP or Unix Domain Socket
		receiverSettings.Listen = c.ListenOn.Build()
		listenDS := c.ListenOn.Family().IsDomain() && (filepath.IsAbs(c.ListenOn.Domain()) || c.ListenOn.Domain()[0] == '@')
		listenIP := c.ListenOn.Family().IsIP() || (c.ListenOn.Family().IsDomain() && c.ListenOn.Domain() == "localhost")
		if listenIP {
			// Listen on specific IP, must set PortList
			if c.PortList == nil {
				return nil, errors.New("Listen on specific ip without port in InboundDetour.")
			}
			// Listen on IP:Port
			receiverSettings.PortList = c.PortList.Build()
		} else if listenDS {
			if c.PortList != nil {
				// Listen on Unix Domain Socket, PortList should be nil
				receiverSettings.PortList = nil
			}
		} else {
			return nil, errors.New("unable to listen on domain address: ", c.ListenOn.Domain())
		}
	}

	if c.StreamSetting != nil {
		ss, err := c.StreamSetting.Build()
		if err != nil {
			return nil, err
		}
		receiverSettings.StreamSettings = ss
		if strings.Contains(ss.SecurityType, "reality") && (receiverSettings.PortList == nil ||
			len(receiverSettings.PortList.Ports()) != 1 || receiverSettings.PortList.Ports()[0] != 443) {
			errors.LogWarning(context.Background(), `REALITY: Listening on non-443 ports will increase the likelihood of your server's IP being blocked by the GFW`)
		}
	}
	if c.SniffingConfig != nil {
		s, err := c.SniffingConfig.Build()
		if err != nil {
			return nil, errors.New("failed to build sniffing config").Base(err)
		}
		receiverSettings.SniffingSettings = s
	}

	settings := []byte("{}")
	if c.Settings != nil {
		settings = ([]byte)(*c.Settings)
	}
	rawConfig, err := inboundConfigLoader.LoadWithID(settings, c.Protocol)
	if err != nil {
		return nil, errors.New("failed to load inbound detour config for protocol ", c.Protocol).Base(err)
	}
	if dokodemoConfig, ok := rawConfig.(*DokodemoConfig); ok {
		receiverSettings.ReceiveOriginalDestination = dokodemoConfig.FollowRedirect
	}
	ts, err := rawConfig.(Buildable).Build()
	if err != nil {
		return nil, errors.New("failed to build inbound handler for protocol ", c.Protocol).Base(err)
	}
	if _, ok := ts.(*masque.ServerConfig); !ok && receiverSettings.StreamSettings != nil && receiverSettings.StreamSettings.ProtocolName == "masque" {
		return nil, errors.New("the masque transport can only be used by the masque inbound")
	}
	if err := validateXDrive(ts, receiverSettings.StreamSettings, nil, receiverSettings.PortList); err != nil {
		return nil, err
	}

	return &core.InboundHandlerConfig{
		Tag:              c.Tag,
		ReceiverSettings: serial.ToTypedMessage(receiverSettings),
		ProxySettings:    serial.ToTypedMessage(ts),
	}, nil
}

type OutboundDetourConfig struct {
	Protocol       string           `json:"protocol"`
	SendThrough    *string          `json:"sendThrough"`
	Tag            string           `json:"tag"`
	Settings       *json.RawMessage `json:"settings"`
	StreamSetting  *StreamConfig    `json:"streamSettings"`
	ProxySettings  *json.RawMessage `json:"proxySettings"`
	MuxSettings    *MuxConfig       `json:"mux"`
	TargetStrategy string           `json:"targetStrategy"`
}

func requiresTransportSecurity(address *Address) bool {
	if address == nil || address.Address == nil {
		return false
	}
	if address.Family().IsIP() {
		return !geodata.GetPrivateIPMatcher().Match(address.IP())
	}
	domain := strings.TrimSuffix(strings.ToLower(address.Domain()), ".")
	return !geodata.GetPrivateDomainMatcher().MatchAny(domain)
}

func validateOutboundTransportSecurity(rawConfig any, senderSettings *proxyman.SenderConfig) error {
	if senderSettings.StreamSettings != nil && senderSettings.StreamSettings.GetSecurityType() != "" {
		return nil
	}

	if vlessCfg, ok := rawConfig.(*VLessOutboundConfig); ok {
		if vlessCfg.Encryption != "" && vlessCfg.Encryption != "none" {
			return nil
		}
		if requiresTransportSecurity(vlessCfg.Address) {
			return errors.New("vless without TLS or other encryption is prohibited unless the server address is a private IP or domain")
		}
	}

	if tjCfg, ok := rawConfig.(*TrojanClientConfig); ok {
		if requiresTransportSecurity(tjCfg.Servers[0].Address) {
			return errors.New("trojan without TLS is prohibited unless the server address is a private IP or domain")
		}
	}

	return nil
}

// validateXDrive allows only VLESS with encryption on xdrive, and on the client Mux.Cool for every connection
func validateXDrive(ts proto.Message, ss *internet.StreamConfig, ms *proxyman.MultiplexingConfig, pl *net.PortList) error {
	if ss == nil || ss.ProtocolName != "xdrive" {
		return nil
	}
	switch c := ts.(type) {
	case *outbound.Config:
		a, err := c.Vnext.User.Account.GetInstance()
		if err != nil {
			return err
		}
		account := a.(*vless.Account)
		if account.Flow != "" {
			return errors.New(`the xdrive transport doesn't support VLESS "flow"`)
		}
		if account.Encryption == "none" {
			return errors.New(`the xdrive transport needs VLESS "encryption", run "xray vlessenc" to get one`)
		}
		// reverse opens its own mux
		if account.Reverse == nil && (ms == nil || !ms.Enabled || ms.Concurrency < 0 || ms.XudpConcurrency < 0 || ms.XudpProxyUDP443 == "skip") {
			return errors.New(`the xdrive transport needs "mux" enabled, with no negative "concurrency" or "xudpConcurrency" and no "xudpProxyUDP443":"skip"`)
		}
		if account.Testpre != 0 {
			return errors.New(`the xdrive transport doesn't support VLESS "testpre"`)
		}
	case *inbound.Config:
		if c.Decryption == "none" {
			return errors.New(`the xdrive transport needs VLESS "decryption", run "xray vlessenc" to get one`)
		}
		for _, user := range c.Users {
			a, err := user.Account.GetInstance()
			if err != nil {
				return err
			}
			if a.(*vless.Account).Flow != "" {
				return errors.New(`the xdrive transport doesn't support VLESS "flow"`)
			}
		}
		// every port gets its own listener on the same folder, and each of them accepts the same session
		if pl != nil && len(pl.Ports()) != 1 {
			return errors.New("the xdrive transport can only listen on one port")
		}
	default:
		return errors.New("the xdrive transport can only be used by VLESS")
	}
	return nil
}

// Build implements Buildable.
func (c *OutboundDetourConfig) Build() (*core.OutboundHandlerConfig, error) {
	if c.ProxySettings != nil {
		return nil, errors.PrintRemovedFeatureError(`outbound "proxySettings"`, `"streamSettings.sockopt.dialerProxy"`)
	}

	senderSettings := &proxyman.SenderConfig{}
	switch strings.ToLower(c.TargetStrategy) {
	case "asis", "":
		senderSettings.TargetStrategy = internet.DomainStrategy_AS_IS
	case "useip":
		senderSettings.TargetStrategy = internet.DomainStrategy_USE_IP
	case "useipv4":
		senderSettings.TargetStrategy = internet.DomainStrategy_USE_IP4
	case "useipv6":
		senderSettings.TargetStrategy = internet.DomainStrategy_USE_IP6
	case "useipv4v6":
		senderSettings.TargetStrategy = internet.DomainStrategy_USE_IP46
	case "useipv6v4":
		senderSettings.TargetStrategy = internet.DomainStrategy_USE_IP64
	case "forceip":
		senderSettings.TargetStrategy = internet.DomainStrategy_FORCE_IP
	case "forceipv4":
		senderSettings.TargetStrategy = internet.DomainStrategy_FORCE_IP4
	case "forceipv6":
		senderSettings.TargetStrategy = internet.DomainStrategy_FORCE_IP6
	case "forceipv4v6":
		senderSettings.TargetStrategy = internet.DomainStrategy_FORCE_IP46
	case "forceipv6v4":
		senderSettings.TargetStrategy = internet.DomainStrategy_FORCE_IP64
	default:
		return nil, errors.New("unsupported target domain strategy: ", c.TargetStrategy)
	}

	if c.SendThrough != nil {
		address := ParseSendThough(c.SendThrough)
		// Check if CIDR exists
		if strings.Contains(*c.SendThrough, "/") {
			senderSettings.ViaCidr = strings.Split(*c.SendThrough, "/")[1]
		} else {
			if address.Family().IsDomain() {
				domain := address.Address.Domain()
				if domain != "origin" && domain != "srcip" {
					return nil, errors.New("unable to send through: " + address.String())
				}
			}
		}
		senderSettings.Via = address.Build()
	}

	if c.StreamSetting != nil {
		ss, err := c.StreamSetting.Build()
		if err != nil {
			return nil, errors.New("failed to build stream settings for outbound detour").Base(err)
		}
		senderSettings.StreamSettings = ss
	}

	if c.MuxSettings != nil {
		ms, err := c.MuxSettings.Build()
		if err != nil {
			return nil, errors.New("failed to build Mux config").Base(err)
		}
		senderSettings.MultiplexSettings = ms
	}

	settings := []byte("{}")
	if c.Settings != nil {
		settings = ([]byte)(*c.Settings)
	}
	rawConfig, err := outboundConfigLoader.LoadWithID(settings, c.Protocol)
	if err != nil {
		return nil, errors.New("failed to load outbound detour config for protocol ", c.Protocol).Base(err)
	}
	ts, err := rawConfig.(Buildable).Build()
	if err != nil {
		return nil, errors.New("failed to build outbound handler for protocol ", c.Protocol).Base(err)
	}
	if err := validateXDrive(ts, senderSettings.StreamSettings, senderSettings.MultiplexSettings, nil); err != nil {
		return nil, err
	}
	if err := validateOutboundTransportSecurity(rawConfig, senderSettings); err != nil {
		return nil, err
	}

	if _, ok := ts.(*masque.ClientConfig); !ok && senderSettings.StreamSettings != nil && senderSettings.StreamSettings.ProtocolName == "masque" {
		return nil, errors.New("the masque transport can only be used by the masque outbound")
	}

	if fc, ok := ts.(*freedom.Config); ok {
		if senderSettings.StreamSettings != nil &&
			senderSettings.StreamSettings.SocketSettings != nil &&
			senderSettings.StreamSettings.SocketSettings.AddressPortStrategy != internet.AddressPortStrategy_None {
			return nil, errors.New(`freedom outbound does not support "sockopt.addressPortStrategy"`)
		}

		var strategy internet.DomainStrategy
		if strategy = senderSettings.TargetStrategy; strategy != internet.DomainStrategy_AS_IS {
			errors.LogWarning(context.Background(), `The "outbound.targetStrategy" setting is not supported directly by freedom and has been automatically migrated to "sockopt.domainStrategy" with no behavior change.`)
			senderSettings.TargetStrategy = internet.DomainStrategy_AS_IS
		} else if strategy = fc.DomainStrategy; strategy != internet.DomainStrategy_AS_IS {
			errors.LogWarning(context.Background(), `The "freedom.domainStrategy" setting is deprecated and will be removed. For compatibility, its value has been automatically migrated to "sockopt.domainStrategy". Please update your config before removal.`)
		}
		if strategy != internet.DomainStrategy_AS_IS {
			if senderSettings.StreamSettings == nil {
				senderSettings.StreamSettings = &internet.StreamConfig{}
			}
			if senderSettings.StreamSettings.SocketSettings == nil {
				senderSettings.StreamSettings.SocketSettings = &internet.SocketConfig{}
			}
			senderSettings.StreamSettings.SocketSettings.DomainStrategy = strategy
		}
	}

	return &core.OutboundHandlerConfig{
		SenderSettings: serial.ToTypedMessage(senderSettings),
		Tag:            c.Tag,
		ProxySettings:  serial.ToTypedMessage(ts),
	}, nil
}

type StatsConfig struct{}

// Build implements Buildable.
func (c *StatsConfig) Build() (*stats.Config, error) {
	return &stats.Config{}, nil
}

type EnvConfig map[string]string

type Config struct {
	// Deprecated: Global transport config is no longer used
	// left for returning error
	Transport map[string]json.RawMessage `json:"transport"`

	Env              EnvConfig               `json:"env"`
	LogConfig        *LogConfig              `json:"log"`
	RouterConfig     *RouterConfig           `json:"routing"`
	DNSConfig        *DNSConfig              `json:"dns"`
	InboundConfigs   []InboundDetourConfig   `json:"inbounds"`
	OutboundConfigs  []OutboundDetourConfig  `json:"outbounds"`
	Policy           *PolicyConfig           `json:"policy"`
	API              *APIConfig              `json:"api"`
	Metrics          *MetricsConfig          `json:"metrics"`
	Stats            *StatsConfig            `json:"stats"`
	Reverse          json.RawMessage         `json:"reverse"` // Retained to reject removed legacy reverse configuration.
	FakeDNS          *FakeDNSConfig          `json:"fakeDns"`
	Observatory      *ObservatoryConfig      `json:"observatory"`
	BurstObservatory *BurstObservatoryConfig `json:"burstObservatory"`
	Geodata          *GeodataConfig          `json:"geodata"`
}

// Build implements Buildable.
func (c *Config) Build() (*core.Config, error) {
	for key, value := range c.Env {
		if err := os.Setenv(key, value); err != nil {
			return nil, errors.New("failed to apply environment configuration").Base(err)
		}
	}

	if err := PostProcessConfigureFile(c); err != nil {
		return nil, errors.New("failed to post-process configuration file").Base(err)
	}

	config := &core.Config{
		App: []any{
			&dispatcher.Config{},
			&proxyman.InboundConfig{},
			&proxyman.OutboundConfig{},
		},
	}

	if c.API != nil {
		apiConf, err := c.API.Build()
		if err != nil {
			return nil, errors.New("failed to build API configuration").Base(err)
		}
		config.App = append(config.App, apiConf)
	}
	if c.Metrics != nil {
		metricsConf, err := c.Metrics.Build()
		if err != nil {
			return nil, errors.New("failed to build metrics configuration").Base(err)
		}
		config.App = append(config.App, metricsConf)
	}
	if c.Stats != nil {
		statsConf, err := c.Stats.Build()
		if err != nil {
			return nil, errors.New("failed to build stats configuration").Base(err)
		}
		config.App = append(config.App, statsConf)
	}

	var logConfMsg proto.Message
	if c.LogConfig != nil {
		logConfMsg = c.LogConfig.Build()
	} else {
		logConfMsg = DefaultLogConfig()
	}
	// let logger module be the first App to start,
	// so that other modules could print log during initiating
	config.App = append([]any{logConfMsg}, config.App...)

	if c.RouterConfig != nil {
		routerConfig, err := c.RouterConfig.Build()
		if err != nil {
			return nil, errors.New("failed to build routing configuration").Base(err)
		}
		config.App = append(config.App, routerConfig)
	}

	if c.DNSConfig != nil {
		dnsApp, err := c.DNSConfig.Build()
		if err != nil {
			return nil, errors.New("failed to build DNS configuration").Base(err)
		}
		config.App = append(config.App, dnsApp)
	}

	if c.Policy != nil {
		pc, err := c.Policy.Build()
		if err != nil {
			return nil, errors.New("failed to build policy configuration").Base(err)
		}
		config.App = append(config.App, pc)
	}

	if c.Reverse != nil {
		return nil, errors.PrintRemovedFeatureError(`"legacy reverse"`, `"VLESS Reverse Proxy"`)
	}

	if c.FakeDNS != nil {
		r, err := c.FakeDNS.Build()
		if err != nil {
			return nil, errors.New("failed to build fake DNS configuration").Base(err)
		}
		config.App = append([]any{r}, config.App...)
	}

	if c.Observatory != nil {
		r, err := c.Observatory.Build()
		if err != nil {
			return nil, errors.New("failed to build observatory configuration").Base(err)
		}
		config.App = append(config.App, r)
	}

	if c.BurstObservatory != nil {
		r, err := c.BurstObservatory.Build()
		if err != nil {
			return nil, errors.New("failed to build burst observatory configuration").Base(err)
		}
		config.App = append(config.App, r)
	}

	if c.Geodata != nil {
		r, err := c.Geodata.Build()
		if err != nil {
			return nil, errors.New("failed to build geodata configuration").Base(err)
		}
		config.App = append(config.App, r)
	}

	var inbounds []InboundDetourConfig

	if len(c.InboundConfigs) > 0 {
		inbounds = append(inbounds, c.InboundConfigs...)
	}

	if len(c.Transport) > 0 {
		return nil, errors.PrintRemovedFeatureError("Global transport config", "streamSettings in inbounds and outbounds")
	}

	for _, rawInboundConfig := range inbounds {
		ic, err := rawInboundConfig.Build()
		if err != nil {
			return nil, errors.New("failed to build inbound config with tag ", rawInboundConfig.Tag).Base(err)
		}
		config.Inbound = append(config.Inbound, ic)
	}

	var outbounds []OutboundDetourConfig

	if len(c.OutboundConfigs) > 0 {
		outbounds = append(outbounds, c.OutboundConfigs...)
	}

	for _, rawOutboundConfig := range outbounds {
		oc, err := rawOutboundConfig.Build()
		if err != nil {
			return nil, errors.New("failed to build outbound config with tag ", rawOutboundConfig.Tag).Base(err)
		}
		config.Outbound = append(config.Outbound, oc)
	}

	return config, nil
}

// Convert string to Address.
func ParseSendThough(Addr *string) *Address {
	var addr Address
	addr.Address = net.ParseAddress(strings.Split(*Addr, "/")[0])
	return &addr
}
