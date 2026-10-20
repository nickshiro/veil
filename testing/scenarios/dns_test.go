package scenarios

import (
	"fmt"
	"testing"
	"time"

	"github.com/nickshiro/veil/app/dns"
	"github.com/nickshiro/veil/app/proxyman"
	"github.com/nickshiro/veil/app/router"
	"github.com/nickshiro/veil/common"
	"github.com/nickshiro/veil/common/geodata"
	"github.com/nickshiro/veil/common/net"
	"github.com/nickshiro/veil/core"
	"github.com/nickshiro/veil/proxy/blackhole"
	"github.com/nickshiro/veil/proxy/freedom"
	"github.com/nickshiro/veil/proxy/socks"
	"github.com/nickshiro/veil/testing/servers/tcp"
	"github.com/nickshiro/veil/transport/internet"
	xproxy "golang.org/x/net/proxy"
)

func TestResolveIP(t *testing.T) {
	tcpServer := tcp.Server{
		MsgProcessor: xor,
	}
	dest, err := tcpServer.Start()
	common.Must(err)
	defer tcpServer.Close()

	serverPort := tcp.PickPort()
	serverConfig := &core.Config{
		App: []any{
			&dns.Config{
				StaticHosts: []*dns.Config_HostMapping{
					{
						Domain: &geodata.DomainRule{Value: &geodata.DomainRule_Custom{Custom: &geodata.Domain{Type: geodata.Domain_Full, Value: "google.com"}}},
						Ip:     [][]byte{dest.Address.IP()},
					},
				},
			},
			&router.RuntimeConfig{
				DomainStrategy: router.DomainIPIfNonMatch,
				Rule: []*router.RuleConfig{
					{
						Ip: []*geodata.IPRule{
							{
								Value: &geodata.IPRule_Custom{
									Custom: &geodata.CIDRRule{
										Cidr: &geodata.CIDR{Ip: []byte{127, 0, 0, 0}, Prefix: 8},
									},
								},
							},
						},
						OutboundTag: "direct",
					},
				},
			},
		},
		Inbound: []*core.InboundConfig{
			{
				ReceiverSettings: &proxyman.ReceiverConfig{
					PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(serverPort)}},
					Listen:   net.NewIPOrDomain(net.LocalHostIP),
				},
				ProxySettings: &socks.ServerConfig{
					AuthType: socks.AuthType_NO_AUTH,
					Accounts: map[string]string{
						"Test Account": "Test Password",
					},
					Address:    net.NewIPOrDomain(net.LocalHostIP),
					UdpEnabled: false,
				},
			},
		},
		Outbound: []*core.OutboundConfig{
			{
				ProxySettings: &blackhole.Config{},
			},
			{
				Tag: "direct",
				SenderSettings: &proxyman.SenderConfig{
					StreamSettings: &internet.StreamConfig{
						SocketSettings: &internet.SocketConfig{
							DomainStrategy: internet.DomainStrategy_USE_IP,
						},
					},
				},
				ProxySettings: &freedom.Config{},
			},
		},
	}

	servers, err := InitializeServerConfigs(serverConfig)
	common.Must(err)
	defer CloseAllServers(servers)

	{
		noAuthDialer, err := xproxy.SOCKS5("tcp", net.TCPDestination(net.LocalHostIP, serverPort).NetAddr(), nil, xproxy.Direct)
		common.Must(err)
		conn, err := noAuthDialer.Dial("tcp", fmt.Sprintf("google.com:%d", dest.Port))
		common.Must(err)
		defer conn.Close()

		if err := testTCPConn2(conn, 1024, time.Second*5)(); err != nil {
			t.Error(err)
		}
	}
}
