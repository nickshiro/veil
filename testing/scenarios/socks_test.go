package scenarios

import (
	"testing"
	"time"

	"github.com/nickshiro/veil/app/proxyman"
	"github.com/nickshiro/veil/app/router"
	"github.com/nickshiro/veil/common"
	"github.com/nickshiro/veil/common/net"
	"github.com/nickshiro/veil/common/protocol"
	"github.com/nickshiro/veil/common/serial"
	"github.com/nickshiro/veil/core"
	"github.com/nickshiro/veil/proxy/blackhole"
	"github.com/nickshiro/veil/proxy/dokodemo"
	"github.com/nickshiro/veil/proxy/freedom"
	"github.com/nickshiro/veil/proxy/http"
	"github.com/nickshiro/veil/proxy/socks"
	"github.com/nickshiro/veil/testing/servers/tcp"
	"github.com/nickshiro/veil/testing/servers/udp"
	xproxy "golang.org/x/net/proxy"
	socks4 "h12.io/socks"
)

func TestSocksBridgeTCP(t *testing.T) {
	tcpServer := tcp.Server{
		MsgProcessor: xor,
	}
	dest, err := tcpServer.Start()
	common.Must(err)
	defer tcpServer.Close()

	serverPort := tcp.PickPort()
	serverConfig := &core.Config{
		Inbound: []*core.InboundConfig{
			{
				ReceiverSettings: &proxyman.ReceiverConfig{
					PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(serverPort)}},
					Listen:   net.NewIPOrDomain(net.LocalHostIP),
				},
				ProxySettings: &socks.ServerConfig{
					AuthType: socks.AuthType_PASSWORD,
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
				ProxySettings: &freedom.Config{
					FinalRules: []*freedom.FinalRuleConfig{{Action: freedom.RuleAction_Allow}},
				},
			},
		},
	}

	clientPort := tcp.PickPort()
	clientConfig := &core.Config{
		Inbound: []*core.InboundConfig{
			{
				ReceiverSettings: &proxyman.ReceiverConfig{
					PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(clientPort)}},
					Listen:   net.NewIPOrDomain(net.LocalHostIP),
				},
				ProxySettings: &dokodemo.Config{
					RewriteAddress:  net.NewIPOrDomain(dest.Address),
					RewritePort:     uint32(dest.Port),
					AllowedNetworks: []net.Network{net.Network_TCP},
				},
			},
		},
		Outbound: []*core.OutboundConfig{
			{
				ProxySettings: &socks.ClientConfig{
					Server: &protocol.ServerEndpoint{
						Address: net.NewIPOrDomain(net.LocalHostIP),
						Port:    uint32(serverPort),
						User: &protocol.User{
							Account: serial.ToTypedMessage(&socks.Account{
								Username: "Test Account",
								Password: "Test Password",
							}),
						},
					},
				},
			},
		},
	}

	servers, err := InitializeServerConfigs(serverConfig, clientConfig)
	common.Must(err)
	defer CloseAllServers(servers)

	if err := testTCPConn(clientPort, 1024, time.Second*2)(); err != nil {
		t.Error(err)
	}
}

func TestSocksWithHttpRequest(t *testing.T) {
	tcpServer := tcp.Server{
		MsgProcessor: xor,
	}
	dest, err := tcpServer.Start()
	common.Must(err)
	defer tcpServer.Close()

	serverPort := tcp.PickPort()
	serverConfig := &core.Config{
		Inbound: []*core.InboundConfig{
			{
				ReceiverSettings: &proxyman.ReceiverConfig{
					PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(serverPort)}},
					Listen:   net.NewIPOrDomain(net.LocalHostIP),
				},
				ProxySettings: &socks.ServerConfig{
					AuthType: socks.AuthType_PASSWORD,
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
				ProxySettings: &freedom.Config{
					FinalRules: []*freedom.FinalRuleConfig{{Action: freedom.RuleAction_Allow}},
				},
			},
		},
	}

	clientPort := tcp.PickPort()
	clientConfig := &core.Config{
		Inbound: []*core.InboundConfig{
			{
				ReceiverSettings: &proxyman.ReceiverConfig{
					PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(clientPort)}},
					Listen:   net.NewIPOrDomain(net.LocalHostIP),
				},
				ProxySettings: &dokodemo.Config{
					RewriteAddress:  net.NewIPOrDomain(dest.Address),
					RewritePort:     uint32(dest.Port),
					AllowedNetworks: []net.Network{net.Network_TCP},
				},
			},
		},
		Outbound: []*core.OutboundConfig{
			{
				ProxySettings: &http.ClientConfig{
					Server: &protocol.ServerEndpoint{
						Address: net.NewIPOrDomain(net.LocalHostIP),
						Port:    uint32(serverPort),
						User: &protocol.User{
							Account: serial.ToTypedMessage(&http.Account{
								Username: "Test Account",
								Password: "Test Password",
							}),
						},
					},
				},
			},
		},
	}

	servers, err := InitializeServerConfigs(serverConfig, clientConfig)
	common.Must(err)
	defer CloseAllServers(servers)

	if err := testTCPConn(clientPort, 1024, time.Second*2)(); err != nil {
		t.Error(err)
	}
}

func TestSocksBridageUDP(t *testing.T) {
	udpServer := udp.Server{
		MsgProcessor: xor,
	}
	dest, err := udpServer.Start()
	common.Must(err)
	defer udpServer.Close()

	retry := 1
	serverPort := tcp.PickPort()
	for {
		serverConfig := &core.Config{
			Inbound: []*core.InboundConfig{
				{
					ReceiverSettings: &proxyman.ReceiverConfig{
						PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(serverPort)}},
						Listen:   net.NewIPOrDomain(net.LocalHostIP),
					},
					ProxySettings: &socks.ServerConfig{
						AuthType: socks.AuthType_PASSWORD,
						Accounts: map[string]string{
							"Test Account": "Test Password",
						},
						Address:    net.NewIPOrDomain(net.LocalHostIP),
						UdpEnabled: true,
					},
				},
				{
					ReceiverSettings: &proxyman.ReceiverConfig{
						PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(serverPort + 1)}},
						Listen:   net.NewIPOrDomain(net.LocalHostIP),
					},
					ProxySettings: &dokodemo.Config{
						RewriteAddress:  net.NewIPOrDomain(dest.Address),
						RewritePort:     uint32(dest.Port),
						AllowedNetworks: []net.Network{net.Network_UDP},
					},
				},
			},
			Outbound: []*core.OutboundConfig{
				{
					ProxySettings: &freedom.Config{FinalRules: []*freedom.FinalRuleConfig{{Action: freedom.RuleAction_Allow}}},
				},
			},
		}

		server, _ := InitializeServerConfig(serverConfig)
		if server != nil && WaitConnAvailableWithTest(t, testUDPConn(serverPort+1, 1024, time.Second*2)) {
			defer CloseServer(server)
			break
		}
		if server != nil {
			CloseServer(server)
		}
		retry++
		if retry > 5 {
			t.Fatal("All attempts failed to start server")
		}
		serverPort = tcp.PickPort()
	}

	clientPort := udp.PickPort()
	clientConfig := &core.Config{
		Inbound: []*core.InboundConfig{
			{
				ReceiverSettings: &proxyman.ReceiverConfig{
					PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(clientPort)}},
					Listen:   net.NewIPOrDomain(net.LocalHostIP),
				},
				ProxySettings: &dokodemo.Config{
					RewriteAddress:  net.NewIPOrDomain(dest.Address),
					RewritePort:     uint32(dest.Port),
					AllowedNetworks: []net.Network{net.Network_UDP},
				},
			},
		},
		Outbound: []*core.OutboundConfig{
			{
				ProxySettings: &socks.ClientConfig{
					Server: &protocol.ServerEndpoint{
						Address: net.NewIPOrDomain(net.LocalHostIP),
						Port:    uint32(serverPort),
						User: &protocol.User{
							Account: serial.ToTypedMessage(&socks.Account{
								Username: "Test Account",
								Password: "Test Password",
							}),
						},
					},
				},
			},
		},
	}

	server, err := InitializeServerConfig(clientConfig)
	common.Must(err)
	defer CloseServer(server)

	if !WaitConnAvailableWithTest(t, testUDPConn(clientPort, 1024, time.Second*2)) {
		t.Fail()
	}
}

func TestSocksBridageUDPWithRouting(t *testing.T) {
	udpServer := udp.Server{
		MsgProcessor: xor,
	}
	dest, err := udpServer.Start()
	common.Must(err)
	defer udpServer.Close()

	retry := 1
	serverPort := tcp.PickPort()
	for {
		serverConfig := &core.Config{
			App: []any{
				&router.RuntimeConfig{
					Rule: []*router.RoutingRule{
						{
							TargetTag: &router.RoutingRule_Tag{
								Tag: "out",
							},
							InboundTag: []string{"socks", "dokodemo"},
						},
					},
				},
			},
			Inbound: []*core.InboundConfig{
				{
					Tag: "socks",
					ReceiverSettings: &proxyman.ReceiverConfig{
						PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(serverPort)}},
						Listen:   net.NewIPOrDomain(net.LocalHostIP),
					},
					ProxySettings: &socks.ServerConfig{
						AuthType:   socks.AuthType_NO_AUTH,
						Address:    net.NewIPOrDomain(net.LocalHostIP),
						UdpEnabled: true,
					},
				},
				{
					Tag: "dokodemo",
					ReceiverSettings: &proxyman.ReceiverConfig{
						PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(serverPort + 1)}},
						Listen:   net.NewIPOrDomain(net.LocalHostIP),
					},
					ProxySettings: &dokodemo.Config{
						RewriteAddress:  net.NewIPOrDomain(dest.Address),
						RewritePort:     uint32(dest.Port),
						AllowedNetworks: []net.Network{net.Network_UDP},
					},
				},
			},
			Outbound: []*core.OutboundConfig{
				{
					ProxySettings: &blackhole.Config{},
				},
				{
					Tag:           "out",
					ProxySettings: &freedom.Config{FinalRules: []*freedom.FinalRuleConfig{{Action: freedom.RuleAction_Allow}}},
				},
			},
		}

		server, _ := InitializeServerConfig(serverConfig)
		if server != nil && WaitConnAvailableWithTest(t, testUDPConn(serverPort+1, 1024, time.Second*2)) {
			defer CloseServer(server)
			break
		}
		if server != nil {
			CloseServer(server)
		}
		retry++
		if retry > 5 {
			t.Fatal("All attempts failed to start server")
		}
		serverPort = tcp.PickPort()
	}

	clientPort := udp.PickPort()
	clientConfig := &core.Config{
		Inbound: []*core.InboundConfig{
			{
				ReceiverSettings: &proxyman.ReceiverConfig{
					PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(clientPort)}},
					Listen:   net.NewIPOrDomain(net.LocalHostIP),
				},
				ProxySettings: &dokodemo.Config{
					RewriteAddress:  net.NewIPOrDomain(dest.Address),
					RewritePort:     uint32(dest.Port),
					AllowedNetworks: []net.Network{net.Network_UDP},
				},
			},
		},
		Outbound: []*core.OutboundConfig{
			{
				ProxySettings: &socks.ClientConfig{
					Server: &protocol.ServerEndpoint{
						Address: net.NewIPOrDomain(net.LocalHostIP),
						Port:    uint32(serverPort),
					},
				},
			},
		},
	}

	server, err := InitializeServerConfig(clientConfig)
	common.Must(err)
	defer CloseServer(server)

	if !WaitConnAvailableWithTest(t, testUDPConn(clientPort, 1024, time.Second*2)) {
		t.Fail()
	}
}

func TestSocksConformanceMod(t *testing.T) {
	tcpServer := tcp.Server{
		MsgProcessor: xor,
	}
	dest, err := tcpServer.Start()
	common.Must(err)
	defer tcpServer.Close()

	authPort := tcp.PickPort()
	noAuthPort := tcp.PickPort()
	serverConfig := &core.Config{
		Inbound: []*core.InboundConfig{
			{
				ReceiverSettings: &proxyman.ReceiverConfig{
					PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(authPort)}},
					Listen:   net.NewIPOrDomain(net.LocalHostIP),
				},
				ProxySettings: &socks.ServerConfig{
					AuthType: socks.AuthType_PASSWORD,
					Accounts: map[string]string{
						"Test Account": "Test Password",
					},
					Address:    net.NewIPOrDomain(net.LocalHostIP),
					UdpEnabled: false,
				},
			},
			{
				ReceiverSettings: &proxyman.ReceiverConfig{
					PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(noAuthPort)}},
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
				ProxySettings: &freedom.Config{
					FinalRules: []*freedom.FinalRuleConfig{{Action: freedom.RuleAction_Allow}},
				},
			},
		},
	}

	servers, err := InitializeServerConfigs(serverConfig)
	common.Must(err)
	defer CloseAllServers(servers)

	{
		noAuthDialer, err := xproxy.SOCKS5("tcp", net.TCPDestination(net.LocalHostIP, noAuthPort).NetAddr(), nil, xproxy.Direct)
		common.Must(err)
		conn, err := noAuthDialer.Dial("tcp", dest.NetAddr())
		common.Must(err)
		defer conn.Close()

		if err := testTCPConn2(conn, 1024, time.Second*5)(); err != nil {
			t.Error(err)
		}
	}

	{
		authDialer, err := xproxy.SOCKS5("tcp", net.TCPDestination(net.LocalHostIP, authPort).NetAddr(), &xproxy.Auth{User: "Test Account", Password: "Test Password"}, xproxy.Direct)
		common.Must(err)
		conn, err := authDialer.Dial("tcp", dest.NetAddr())
		common.Must(err)
		defer conn.Close()

		if err := testTCPConn2(conn, 1024, time.Second*5)(); err != nil {
			t.Error(err)
		}
	}

	{
		dialer := socks4.Dial("socks4://" + net.TCPDestination(net.LocalHostIP, noAuthPort).NetAddr())
		conn, err := dialer("tcp", dest.NetAddr())
		common.Must(err)
		defer conn.Close()

		if err := testTCPConn2(conn, 1024, time.Second*5)(); err != nil {
			t.Error(err)
		}
	}

	{
		dialer := socks4.Dial("socks4://" + net.TCPDestination(net.LocalHostIP, noAuthPort).NetAddr())
		conn, err := dialer("tcp", net.TCPDestination(net.LocalHostIP, tcpServer.Port).NetAddr())
		common.Must(err)
		defer conn.Close()

		if err := testTCPConn2(conn, 1024, time.Second*5)(); err != nil {
			t.Error(err)
		}
	}
}
