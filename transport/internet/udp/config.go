package udp

import (
	"github.com/nickshiro/veil/common"
	"github.com/nickshiro/veil/transport/internet"
)

func init() {
	common.Must(internet.RegisterProtocolConfigCreator(protocolName, func() any {
		return new(Config)
	}))
}
