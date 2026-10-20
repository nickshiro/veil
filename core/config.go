package core

import "io"

// Config is the master config of Xray. Xray takes this config as input and
// functions accordingly.
type Config struct {
	// Inbound handler configurations. Must have at least one item.
	Inbound []*InboundHandlerConfig `json:"inbound,omitempty"`
	// Outbound handler configurations. Must have at least one item. The first
	// item is used as default for routing.
	Outbound []*OutboundHandlerConfig `json:"outbound,omitempty"`
	// App is for configurations of all features in Xray. A feature must
	// implement the Feature interface, and its config type must be registered
	// through common.RegisterConfig.
	// Configurations are passed directly to constructors, without serialization.
	// Constructors must treat them as read-only; callers must not mutate them
	// while an instance is using them.
	App []any `json:"app,omitempty"`
}

// ConfigLoader loads a config from an io.Reader.
// It is registered by the JSON configuration package to avoid an import cycle.
var ConfigLoader func(input io.Reader) (*Config, error)

func LoadConfig(input io.Reader) (*Config, error) {
	// TODO(arkriny): resolve cycle dependency, then inline
	return ConfigLoader(input)
}
