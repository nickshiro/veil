package conf

import (
	"encoding/json"
	"strings"

	"github.com/nickshiro/veil/common/errors"
)

type ConfigFactory func() Buildable

type ConfigFactories map[string]ConfigFactory

func (v ConfigFactories) CreateConfig(id string) (Buildable, error) {
	creator, found := v[id]
	if !found {
		return nil, errors.New("unknown config id: ", id)
	}
	return creator(), nil
}

type JSONConfigLoader struct {
	factories ConfigFactories
	idKey     string
}

func NewJSONConfigLoader(factories ConfigFactories, idKey string) *JSONConfigLoader {
	return &JSONConfigLoader{
		idKey:     idKey,
		factories: factories,
	}
}

func (v *JSONConfigLoader) LoadWithID(raw []byte, id string) (Buildable, error) {
	id = strings.ToLower(id)
	config, err := v.factories.CreateConfig(id)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, config); err != nil {
		return nil, err
	}
	return config, nil
}

func (v *JSONConfigLoader) Load(raw []byte) (Buildable, string, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, "", err
	}
	rawID, found := obj[v.idKey]
	if !found {
		return nil, "", errors.New(v.idKey, " not found in JSON context")
	}
	var id string
	if err := json.Unmarshal(rawID, &id); err != nil {
		return nil, "", err
	}
	config, err := v.LoadWithID(raw, id)
	if err != nil {
		return nil, id, err
	}
	return config, id, nil
}
