package conf

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/nickshiro/veil/common/errors"
	"github.com/nickshiro/veil/core"
)

// Load decodes a JSON configuration and builds its runtime configuration.
func Load(reader io.Reader) (*core.Config, error) {
	var config Config
	if err := json.NewDecoder(reader).Decode(&config); err != nil {
		return nil, err
	}

	runtimeConfig, err := config.Build()
	if err != nil {
		return nil, errors.New("failed to parse json config").Base(err)
	}
	return runtimeConfig, nil
}

func decodeSettings(raw json.RawMessage, kind string, factories map[string]func() Buildable) (Buildable, error) {
	kind = strings.ToLower(kind)
	create, ok := factories[kind]
	if !ok {
		return nil, errors.New("unknown config id: ", kind)
	}
	config := create()
	if err := json.Unmarshal(raw, config); err != nil {
		return nil, err
	}
	return config, nil
}
