package core

// InboundConfig contains native settings passed directly to an inbound constructor.
// Callers must not mutate settings while the instance is using them.
type InboundConfig struct {
	Tag              string
	ReceiverSettings any
	ProxySettings    any
}

// OutboundConfig contains native settings passed directly to an outbound constructor.
// Callers must not mutate settings while the instance is using them.
type OutboundConfig struct {
	Tag            string
	SenderSettings any
	ProxySettings  any
}

// Config is the master config of Xray. Xray takes this config as input and
// functions accordingly.
type Config struct {
	// Inbound handler configurations. Must have at least one item.
	Inbound []*InboundConfig `json:"inbound,omitempty"`
	// Outbound handler configurations. Must have at least one item. The first
	// item is used as default for routing.
	Outbound []*OutboundConfig `json:"outbound,omitempty"`
	// App is for configurations of all features in Xray. A feature must
	// implement the Feature interface, and its config type must be registered
	// through common.RegisterConfig.
	// Configurations are passed directly to constructors, without serialization.
	// Constructors must treat them as read-only; callers must not mutate them
	// while an instance is using them.
	App []any `json:"app,omitempty"`
}
