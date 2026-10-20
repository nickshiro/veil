package router

// RuntimeConfig is the internal router configuration. Config is retained for
// the protobuf management API.
type RuntimeConfig struct {
	DomainStrategy Config_DomainStrategy
	Rule           []*RoutingRule
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
