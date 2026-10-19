package core

import (
	"io"
	"slices"
	"strings"

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

// ConfigFormat is a configurable format of Xray config file.
type ConfigFormat struct {
	Name      string
	Extension []string
	Loader    ConfigLoader
}

type ConfigSource struct {
	Name   string
	Format string
}

// ConfigLoader is a utility to load Xray config from external source.
type ConfigLoader func(input interface{}) (*Config, error)

// ConfigBuilder is a builder to build core.Config from filenames and formats
type ConfigBuilder func(files []*ConfigSource) (*Config, error)

// ConfigsMerger merges multiple json configs into a single one
type ConfigsMerger func(files []*ConfigSource) (string, error)

var (
	configLoaderByName    = make(map[string]*ConfigFormat)
	configLoaderByExt     = make(map[string]*ConfigFormat)
	ConfigBuilderForFiles ConfigBuilder
	ConfigMergedFormFiles ConfigsMerger
)

// RegisterConfigLoader add a new ConfigLoader.
func RegisterConfigLoader(format *ConfigFormat) error {
	name := strings.ToLower(format.Name)
	if _, found := configLoaderByName[name]; found {
		return errors.New(format.Name, " already registered.")
	}
	configLoaderByName[name] = format

	for _, ext := range format.Extension {
		lext := strings.ToLower(ext)
		if f, found := configLoaderByExt[lext]; found {
			return errors.New(ext, " already registered to ", f.Name)
		}
		configLoaderByExt[lext] = format
	}

	return nil
}

func GetMergedConfig(args cmdarg.Arg) (string, error) {
	var files []*ConfigSource
	supported := []string{"json"}
	for _, file := range args {
		format := "json"
		if file != "stdin:" {
			format = GetFormat(file)
		}

		if slices.Contains(supported, format) {
			files = append(files, &ConfigSource{
				Name:   file,
				Format: format,
			})
		}
	}
	return ConfigMergedFormFiles(files)
}

func GetFormatByExtension(ext string) string {
	return "json"
}

func getExtension(filename string) string {
	idx := strings.LastIndexByte(filename, '.')
	if idx == -1 {
		return ""
	}
	return filename[idx+1:]
}

func GetFormat(filename string) string {
	return GetFormatByExtension(getExtension(filename))
}

func LoadConfig(formatName string, input interface{}) (*Config, error) {
	switch v := input.(type) {
	case cmdarg.Arg:
		files := make([]*ConfigSource, len(v))
		for i, file := range v {
			var f string

			if formatName == "auto" {
				if file != "stdin:" {
					f = GetFormat(file)
				} else {
					f = "json"
				}
			} else {
				f = formatName
			}

			if f == "" {
				return nil, errors.New("Failed to get format of ", file)
			}

			files[i] = &ConfigSource{
				Name:   file,
				Format: f,
			}
		}

		// to avoid import cycle
		return ConfigBuilderForFiles(files)
	case io.Reader:
		if f, found := configLoaderByName[formatName]; found {
			return f.Loader(v)
		} else {
			return nil, errors.New("Unable to load config in", formatName)
		}
	}

	return nil, errors.New("Unable to load config")
}
