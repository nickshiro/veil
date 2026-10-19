package httpupgrade

import (
	"github.com/nickshiro/veil/common"
	"github.com/nickshiro/veil/transport/internet"
)

func (c *Config) GetNormalizedPath() string {
	path := c.Path
	if path == "" {
		return "/"
	}
	if path[0] != '/' {
		return "/" + path
	}
	return path
}

func init() {
	common.Must(internet.RegisterProtocolConfigCreator(protocolName, func() any {
		return new(Config)
	}))
}
