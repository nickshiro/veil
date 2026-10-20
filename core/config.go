package core

import (
	"io"

	"github.com/nickshiro/veil/common/cmdarg"
	"github.com/nickshiro/veil/common/errors"
	"github.com/nickshiro/veil/common/serial"
)

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
	App []*serial.TypedMessage `json:"app,omitempty"`
	// Configuration for extensions. The config may not work if corresponding
	// extension is not loaded into Xray. Xray will ignore such config during
	// initialization.
	Extension []*serial.TypedMessage `json:"extension,omitempty"`
}

type ConfigSource struct {
	Name string
}

// ConfigLoader is a utility to load Xray config from external source.
type configLoader func(input interface{}) (*Config, error)

// ConfigBuilder is a builder to build core.Config from filenames and formats
type ConfigBuilder func(files []*ConfigSource) (*Config, error)

// ConfigsMerger merges multiple json configs into a single one
type ConfigsMerger func(files []*ConfigSource) (string, error)

var (
	ConfigLoader          configLoader
	ConfigBuilderForFiles ConfigBuilder
	ConfigMergedFormFiles ConfigsMerger
)

func GetMergedConfig(args cmdarg.Arg) (string, error) {
	var files []*ConfigSource
	for _, file := range args {
		files = append(files, &ConfigSource{
			Name: file,
		})
	}
	return ConfigMergedFormFiles(files)
}

func LoadConfig(input interface{}) (*Config, error) {
	switch v := input.(type) {
	case cmdarg.Arg:
		files := make([]*ConfigSource, len(v))
		for i, file := range v {
			files[i] = &ConfigSource{
				Name: file,
			}
		}

		// to avoid import cycle
		return ConfigBuilderForFiles(files)
	case io.Reader:
		return ConfigLoader(v)
	}

	return nil, errors.New("Unable to load config")
}
