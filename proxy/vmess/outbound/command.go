package outbound

import (
	"github.com/nickshiro/veil/common/net"
	"github.com/nickshiro/veil/common/protocol"
)

// As a stub command consumer.
func (h *Handler) handleCommand(dest net.Destination, cmd protocol.ResponseCommand) {
	switch cmd.(type) {
	default:
	}
}
