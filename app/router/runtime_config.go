package router

import (
	"github.com/nickshiro/veil/common/geodata"
	"github.com/nickshiro/veil/common/net"
)

// RuntimeConfig is the internal router configuration. Config is retained for
// the protobuf management API.
type RuntimeConfig struct {
	DomainStrategy DomainStrategy
	Rule           []*RuleConfig
	BalancingRule  []*BalancerConfig
}

type BalancerConfig struct {
	Tag              string
	OutboundSelector []string
	Strategy         string
	StrategySettings *LeastLoadConfig
	FallbackTag      string
}

type Weight struct {
	Regexp bool    `json:"regexp"`
	Match  string  `json:"match"`
	Value  float32 `json:"value"`
}

type LeastLoadConfig struct {
	Costs     []*Weight
	Baselines []int64
	Expected  int32
	MaxRTT    int64
	Tolerance float32
}

// DomainStrategy controls when routing resolves destination domains.
type DomainStrategy int32

const (
	DomainAsIs         DomainStrategy = 0
	DomainIPIfNonMatch DomainStrategy = 2
	DomainIPOnDemand   DomainStrategy = 3
)

// RuleConfig describes a routing rule before its matchers are compiled.
// OutboundTag and BalancerTag are mutually exclusive.
type RuleConfig struct {
	OutboundTag    string
	BalancerTag    string
	RuleTag        string
	Domain         []*geodata.DomainRule
	Ip             []*geodata.IPRule
	PortList       *net.PortList
	Networks       []net.Network
	SourceIp       []*geodata.IPRule
	SourcePortList *net.PortList
	UserEmail      []string
	InboundTag     []string
	Protocol       []string
	Attributes     map[string]string
	LocalIp        []*geodata.IPRule
	LocalPortList  *net.PortList
	VlessRouteList *net.PortList
	Process        []string
	Webhook        *WebhookSettings
	LocalOs        []string
}

type WebhookSettings struct {
	Url           string
	Deduplication uint32
	Headers       map[string]string
}
