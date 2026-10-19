package salamander

import (
	"github.com/nickshiro/veil/common/net"
	"github.com/nickshiro/veil/transport/internet/finalmask"
)

func (c *Config) HeaderConn() {}

func (c *Config) WrapPacketConnClient(conn net.PacketConn, dest *net.Destination, dialer *finalmask.Dialer) (net.PacketConn, error) {
	return NewSalamanderConnClient(c, conn)
}

func (c *Config) WrapPacketConnServer(conn net.PacketConn, addr net.Addr, lc *finalmask.ListenConfig) (net.PacketConn, error) {
	return NewSalamanderConnServer(c, conn)
}

func (c *GeckoConfig) WrapPacketConnClient(conn net.PacketConn, dest *net.Destination, dialer *finalmask.Dialer) (net.PacketConn, error) {
	return NewGeckoConnClient(c, conn)
}

func (c *GeckoConfig) WrapPacketConnServer(conn net.PacketConn, addr net.Addr, lc *finalmask.ListenConfig) (net.PacketConn, error) {
	return NewGeckoConnServer(c, conn)
}
