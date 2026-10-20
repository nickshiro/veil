package conf

import (
	"strings"

	"github.com/nickshiro/veil/app/commander"
	loggerservice "github.com/nickshiro/veil/app/log/command"
	observatoryservice "github.com/nickshiro/veil/app/observatory/command"
	handlerservice "github.com/nickshiro/veil/app/proxyman/command"
	routerservice "github.com/nickshiro/veil/app/router/command"
	statsservice "github.com/nickshiro/veil/app/stats/command"
	"github.com/nickshiro/veil/common/errors"
)

type APIConfig struct {
	Tag      string   `json:"tag"`
	Listen   string   `json:"listen"`
	Services []string `json:"services"`
}

func (c *APIConfig) Build() (*commander.Config, error) {
	if c.Tag == "" {
		return nil, errors.New("API tag can't be empty.")
	}

	services := make([]any, 0, len(c.Services))
	for _, s := range c.Services {
		switch strings.ToLower(s) {
		case "reflectionservice":
			services = append(services, &commander.ReflectionConfig{})
		case "handlerservice":
			services = append(services, &handlerservice.Config{})
		case "loggerservice":
			services = append(services, &loggerservice.Config{})
		case "statsservice":
			services = append(services, &statsservice.Config{})
		case "observatoryservice":
			services = append(services, &observatoryservice.Config{})
		case "routingservice":
			services = append(services, &routerservice.Config{})
		}
	}

	return &commander.Config{
		Tag:     c.Tag,
		Listen:  c.Listen,
		Service: services,
	}, nil
}
