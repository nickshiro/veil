package masque

import (
	"time"

	"github.com/nickshiro/veil/common"
	"github.com/nickshiro/veil/transport/internet"
)

const protocolName = "masque"

const DefaultPath = "/.well-known/masque/ip/*/*/"

func (c *Config) keepAlivePeriod() time.Duration {
	if c.Xmux == nil {
		return 0
	}
	return time.Duration(c.Xmux.HKeepAlivePeriod) * time.Second
}

func init() {
	common.Must(internet.RegisterProtocolConfigCreator(protocolName, func() any {
		return &Config{
			Path: DefaultPath,
		}
	}))
}
