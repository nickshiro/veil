package udp

import (
	"context"

	"github.com/nickshiro/veil/common"
	"github.com/nickshiro/veil/common/net"
	"github.com/nickshiro/veil/transport/internet"
	"github.com/nickshiro/veil/transport/internet/stat"
)

func init() {
	common.Must(internet.RegisterTransportDialer(protocolName,
		func(ctx context.Context, dest net.Destination, streamSettings *internet.MemoryStreamConfig) (stat.Connection, error) {
			if streamSettings != nil && streamSettings.FinalMask != nil {
				return streamSettings.FinalMask.DialUDP(ctx, dest)
			} else {
				var sockopt *internet.SocketConfig
				if streamSettings != nil && streamSettings.SocketSettings != nil {
					sockopt = streamSettings.SocketSettings
				}
				return internet.DialSystem(ctx, dest, sockopt)
			}
		}))
}
