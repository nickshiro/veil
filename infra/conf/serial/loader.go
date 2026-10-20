package serial

import (
	"encoding/json"
	"io"

	"github.com/nickshiro/veil/common/errors"
	"github.com/nickshiro/veil/core"
	"github.com/nickshiro/veil/infra/conf"
)

func LoadJSONConfig(reader io.Reader) (*core.Config, error) {
	var jsonConfig conf.Config
	if err := json.NewDecoder(reader).Decode(&jsonConfig); err != nil {
		return nil, err
	}

	pbConfig, err := jsonConfig.Build()
	if err != nil {
		return nil, errors.New("failed to parse json config").Base(err)
	}

	return pbConfig, nil
}
