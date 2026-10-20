package scenarios

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"
	"uuid"

	"github.com/nickshiro/veil/app/dispatcher"
	"github.com/nickshiro/veil/app/log"
	"github.com/nickshiro/veil/app/proxyman"
	_ "github.com/nickshiro/veil/app/proxyman/inbound"
	_ "github.com/nickshiro/veil/app/proxyman/outbound"
	"github.com/nickshiro/veil/app/router"
	"github.com/nickshiro/veil/common"
	clog "github.com/nickshiro/veil/common/log"
	"github.com/nickshiro/veil/common/net"
	"github.com/nickshiro/veil/common/protocol"
	"github.com/nickshiro/veil/common/serial"
	core "github.com/nickshiro/veil/core"
	"github.com/nickshiro/veil/proxy/blackhole"
	"github.com/nickshiro/veil/proxy/dokodemo"
	"github.com/nickshiro/veil/proxy/freedom"
	v2http "github.com/nickshiro/veil/proxy/http"
	"github.com/nickshiro/veil/proxy/socks"
	"github.com/nickshiro/veil/proxy/vmess"
	"github.com/nickshiro/veil/proxy/vmess/inbound"
	"github.com/nickshiro/veil/proxy/vmess/outbound"
	"github.com/nickshiro/veil/testing/servers/tcp"
	"github.com/nickshiro/veil/testing/servers/udp"
	"github.com/nickshiro/veil/transport/internet"
	xproxy "golang.org/x/net/proxy"
)

func TestPassiveConnection(t *testing.T) {
	tcpServer := tcp.Server{
		MsgProcessor: xor,
		SendFirst:    []byte("send first"),
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
				ProxySettings: &dokodemo.Config{
					RewriteAddress:  net.NewIPOrDomain(dest.Address),
					RewritePort:     uint32(dest.Port),
					AllowedNetworks: []net.Network{net.Network_TCP},
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

	conn, err := net.DialTCP("tcp", nil, &net.TCPAddr{
		IP:   []byte{127, 0, 0, 1},
		Port: int(serverPort),
	})
	common.Must(err)

	{
		response := make([]byte, 1024)
		nBytes, err := conn.Read(response)
		common.Must(err)
		if string(response[:nBytes]) != "send first" {
			t.Error("unexpected first response: ", string(response[:nBytes]))
		}
	}

	if err := testTCPConn2(conn, 1024, time.Second*5)(); err != nil {
		t.Error(err)
	}
}

func TestDialerProxy(t *testing.T) {
	tcpServer := tcp.Server{
		MsgProcessor: xor,
	}
	dest, err := tcpServer.Start()
	common.Must(err)
	defer tcpServer.Close()

	serverUserID := protocol.NewID(uuid.NewV4())
	serverPort := tcp.PickPort()
	serverConfig := &core.Config{
		Inbound: []*core.InboundConfig{
			{
				ReceiverSettings: &proxyman.ReceiverConfig{
					PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(serverPort)}},
					Listen:   net.NewIPOrDomain(net.LocalHostIP),
				},
				ProxySettings: &inbound.Config{
					User: []*protocol.User{
						{
							Account: serial.ToTypedMessage(&vmess.Account{
								Id: serverUserID.String(),
							}),
						},
					},
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

	proxyUserID := protocol.NewID(uuid.NewV4())
	proxyPort := tcp.PickPort()
	proxyConfig := &core.Config{
		Inbound: []*core.InboundConfig{
			{
				ReceiverSettings: &proxyman.ReceiverConfig{
					PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(proxyPort)}},
					Listen:   net.NewIPOrDomain(net.LocalHostIP),
				},
				ProxySettings: &inbound.Config{
					User: []*protocol.User{
						{
							Account: serial.ToTypedMessage(&vmess.Account{
								Id: proxyUserID.String(),
							}),
						},
					},
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
				ProxySettings: &outbound.Config{
					Receiver: &protocol.ServerEndpoint{
						Address: net.NewIPOrDomain(net.LocalHostIP),
						Port:    uint32(serverPort),
						User: &protocol.User{
							Account: serial.ToTypedMessage(&vmess.Account{
								Id: serverUserID.String(),
							}),
						},
					},
				},
				SenderSettings: &proxyman.SenderConfig{
					StreamSettings: &internet.StreamConfig{
						SocketSettings: &internet.SocketConfig{
							DialerProxy: "proxy",
						},
					},
				},
			},
			{
				Tag: "proxy",
				ProxySettings: &outbound.Config{
					Receiver: &protocol.ServerEndpoint{
						Address: net.NewIPOrDomain(net.LocalHostIP),
						Port:    uint32(proxyPort),
						User: &protocol.User{
							Account: serial.ToTypedMessage(&vmess.Account{
								Id: proxyUserID.String(),
							}),
						},
					},
				},
			},
		},
	}

	servers, err := InitializeServerConfigs(serverConfig, proxyConfig, clientConfig)
	common.Must(err)
	defer CloseAllServers(servers)

	if err := testTCPConn(clientPort, 1024, time.Second*5)(); err != nil {
		t.Error(err)
	}
}

func TestDialerProxyOverKCP(t *testing.T) {
	tcpServer := tcp.Server{
		MsgProcessor: xor,
	}
	dest, err := tcpServer.Start()
	common.Must(err)
	defer tcpServer.Close()

	serverUserID := protocol.NewID(uuid.NewV4())
	serverPort := udp.PickPort()
	serverConfig := &core.Config{
		Inbound: []*core.InboundConfig{
			{
				ReceiverSettings: &proxyman.ReceiverConfig{
					PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(serverPort)}},
					Listen:   net.NewIPOrDomain(net.LocalHostIP),
					StreamSettings: &internet.StreamConfig{
						ProtocolName: "mkcp",
					},
				},
				ProxySettings: &inbound.Config{
					User: []*protocol.User{
						{
							Account: serial.ToTypedMessage(&vmess.Account{
								Id: serverUserID.String(),
							}),
						},
					},
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

	proxyUserID := protocol.NewID(uuid.NewV4())
	proxyPort := tcp.PickPort()
	proxyConfig := &core.Config{
		Inbound: []*core.InboundConfig{
			{
				ReceiverSettings: &proxyman.ReceiverConfig{
					PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(proxyPort)}},
					Listen:   net.NewIPOrDomain(net.LocalHostIP),
				},
				ProxySettings: &inbound.Config{
					User: []*protocol.User{
						{
							Account: serial.ToTypedMessage(&vmess.Account{
								Id: proxyUserID.String(),
							}),
						},
					},
				},
			},
		},
		Outbound: []*core.OutboundConfig{
			{
				ProxySettings: &freedom.Config{
					FinalRules: []*freedom.FinalRuleConfig{{Action: freedom.RuleAction_Allow}},
				},
				SenderSettings: &proxyman.SenderConfig{
					StreamSettings: &internet.StreamConfig{
						ProtocolName: "mkcp",
					},
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
				ProxySettings: &outbound.Config{
					Receiver: &protocol.ServerEndpoint{
						Address: net.NewIPOrDomain(net.LocalHostIP),
						Port:    uint32(serverPort),
						User: &protocol.User{
							Account: serial.ToTypedMessage(&vmess.Account{
								Id: serverUserID.String(),
							}),
						},
					},
				},
				SenderSettings: &proxyman.SenderConfig{
					StreamSettings: &internet.StreamConfig{
						ProtocolName: "mkcp",
						SocketSettings: &internet.SocketConfig{
							DialerProxy: "proxy",
						},
					},
				},
			},
			{
				Tag: "proxy",
				ProxySettings: &outbound.Config{
					Receiver: &protocol.ServerEndpoint{
						Address: net.NewIPOrDomain(net.LocalHostIP),
						Port:    uint32(proxyPort),
						User: &protocol.User{
							Account: serial.ToTypedMessage(&vmess.Account{
								Id: proxyUserID.String(),
							}),
						},
					},
				},
			},
		},
	}

	servers, err := InitializeServerConfigs(serverConfig, proxyConfig, clientConfig)
	common.Must(err)
	defer CloseAllServers(servers)

	if err := testTCPConn(clientPort, 1024, time.Second*5)(); err != nil {
		t.Error(err)
	}
}

func TestBlackhole(t *testing.T) {
	tcpServer := tcp.Server{
		MsgProcessor: xor,
	}
	dest, err := tcpServer.Start()
	common.Must(err)
	defer tcpServer.Close()

	tcpServer2 := tcp.Server{
		MsgProcessor: xor,
	}
	dest2, err := tcpServer2.Start()
	common.Must(err)
	defer tcpServer2.Close()

	serverPort := tcp.PickPort()
	serverPort2 := tcp.PickPort()
	serverConfig := &core.Config{
		Inbound: []*core.InboundConfig{
			{
				ReceiverSettings: &proxyman.ReceiverConfig{
					PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(serverPort)}},
					Listen:   net.NewIPOrDomain(net.LocalHostIP),
				},
				ProxySettings: &dokodemo.Config{
					RewriteAddress:  net.NewIPOrDomain(dest.Address),
					RewritePort:     uint32(dest.Port),
					AllowedNetworks: []net.Network{net.Network_TCP},
				},
			},
			{
				ReceiverSettings: &proxyman.ReceiverConfig{
					PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(serverPort2)}},
					Listen:   net.NewIPOrDomain(net.LocalHostIP),
				},
				ProxySettings: &dokodemo.Config{
					RewriteAddress:  net.NewIPOrDomain(dest2.Address),
					RewritePort:     uint32(dest2.Port),
					AllowedNetworks: []net.Network{net.Network_TCP},
				},
			},
		},
		Outbound: []*core.OutboundConfig{
			{
				Tag:           "direct",
				ProxySettings: &freedom.Config{FinalRules: []*freedom.FinalRuleConfig{{Action: freedom.RuleAction_Allow}}},
			},
			{
				Tag:           "blocked",
				ProxySettings: &blackhole.Config{},
			},
		},
		App: []any{
			&router.RuntimeConfig{
				Rule: []*router.RuleConfig{
					{
						OutboundTag: "blocked",
						PortList: &net.PortList{
							Range: []*net.PortRange{net.SinglePortRange(dest2.Port)},
						},
					},
				},
			},
		},
	}

	servers, err := InitializeServerConfigs(serverConfig)
	common.Must(err)
	defer CloseAllServers(servers)

	if err := testTCPConn(serverPort2, 1024, time.Second*5)(); err == nil {
		t.Error("nil error")
	}
}

func TestForward(t *testing.T) {
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
					DestinationOverride: &freedom.DestinationOverride{
						Server: &protocol.ServerEndpoint{
							Address: net.NewIPOrDomain(net.LocalHostIP),
							Port:    uint32(dest.Port),
						},
					},
				},
			},
		},
	}

	servers, err := InitializeServerConfigs(serverConfig)
	common.Must(err)
	defer CloseAllServers(servers)

	{
		noAuthDialer, err := xproxy.SOCKS5("tcp", net.TCPDestination(net.LocalHostIP, serverPort).NetAddr(), nil, xproxy.Direct)
		common.Must(err)
		conn, err := noAuthDialer.Dial("tcp", "google.com:80")
		common.Must(err)
		defer conn.Close()

		if err := testTCPConn2(conn, 1024, time.Second*5)(); err != nil {
			t.Error(err)
		}
	}
}

func TestUDPConnection(t *testing.T) {
	udpServer := udp.Server{
		MsgProcessor: xor,
	}
	dest, err := udpServer.Start()
	common.Must(err)
	defer udpServer.Close()

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
				ProxySettings: &freedom.Config{
					FinalRules: []*freedom.FinalRuleConfig{{Action: freedom.RuleAction_Allow}},
				},
			},
		},
	}

	servers, err := InitializeServerConfigs(clientConfig)
	common.Must(err)
	defer CloseAllServers(servers)

	if err := testUDPConn(clientPort, 1024, time.Second*5)(); err != nil {
		t.Error(err)
	}

	time.Sleep(20 * time.Second)

	if err := testUDPConn(clientPort, 1024, time.Second*5)(); err != nil {
		t.Error(err)
	}
}

func TestDomainSniffing(t *testing.T) {
	sniffingPort := tcp.PickPort()
	httpPort := tcp.PickPort()
	serverConfig := &core.Config{
		Inbound: []*core.InboundConfig{
			{
				Tag: "snif",
				ReceiverSettings: &proxyman.ReceiverConfig{
					PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(sniffingPort)}},
					Listen:   net.NewIPOrDomain(net.LocalHostIP),
					SniffingSettings: &proxyman.SniffingConfig{
						Enabled:             true,
						DestinationOverride: []string{"tls"},
					},
				},
				ProxySettings: &dokodemo.Config{
					RewriteAddress:  net.NewIPOrDomain(net.LocalHostIP),
					RewritePort:     443,
					AllowedNetworks: []net.Network{net.Network_TCP},
				},
			},
			{
				Tag: "http",
				ReceiverSettings: &proxyman.ReceiverConfig{
					PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(httpPort)}},
					Listen:   net.NewIPOrDomain(net.LocalHostIP),
				},
				ProxySettings: &v2http.ServerConfig{},
			},
		},
		Outbound: []*core.OutboundConfig{
			{
				Tag: "redir",
				ProxySettings: &freedom.Config{
					DestinationOverride: &freedom.DestinationOverride{
						Server: &protocol.ServerEndpoint{
							Address: net.NewIPOrDomain(net.LocalHostIP),
							Port:    uint32(sniffingPort),
						},
					},
				},
			},
			{
				Tag:           "direct",
				ProxySettings: &freedom.Config{FinalRules: []*freedom.FinalRuleConfig{{Action: freedom.RuleAction_Allow}}},
			},
		},
		App: []any{
			&router.RuntimeConfig{
				Rule: []*router.RuleConfig{
					{
						OutboundTag: "direct",
						InboundTag:  []string{"snif"},
					}, {
						OutboundTag: "redir",
						InboundTag:  []string{"http"},
					},
				},
			},
			&log.Config{
				ErrorLogLevel: clog.Severity_Debug,
				ErrorLogType:  log.LogType_Console,
			},
		},
	}

	servers, err := InitializeServerConfigs(serverConfig)
	common.Must(err)
	defer CloseAllServers(servers)

	{
		transport := &http.Transport{
			Proxy: func(req *http.Request) (*url.URL, error) {
				return url.Parse("http://127.0.0.1:" + httpPort.String())
			},
		}

		client := &http.Client{
			Transport: transport,
		}

		resp, err := client.Get("https://www.github.com/")
		common.Must(err)
		if resp.StatusCode != 200 {
			t.Error("unexpected status code: ", resp.StatusCode)
		}
		common.Must(resp.Write(io.Discard))
	}
}

func TestDialXray(t *testing.T) {
	tcpServer := tcp.Server{
		MsgProcessor: xor,
	}
	dest, err := tcpServer.Start()
	common.Must(err)
	defer tcpServer.Close()

	userID := protocol.NewID(uuid.NewV4())
	serverPort := tcp.PickPort()
	serverConfig := &core.Config{
		App: []any{
			&log.Config{
				ErrorLogLevel: clog.Severity_Debug,
				ErrorLogType:  log.LogType_Console,
			},
		},
		Inbound: []*core.InboundConfig{
			{
				ReceiverSettings: &proxyman.ReceiverConfig{
					PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(serverPort)}},
					Listen:   net.NewIPOrDomain(net.LocalHostIP),
				},
				ProxySettings: &inbound.Config{
					User: []*protocol.User{
						{
							Account: serial.ToTypedMessage(&vmess.Account{
								Id: userID.String(),
							}),
						},
					},
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

	clientConfig := &core.Config{
		App: []any{
			&dispatcher.Config{},
			&proxyman.InboundConfig{},
			&proxyman.OutboundConfig{},
		},
		Inbound: []*core.InboundConfig{},
		Outbound: []*core.OutboundConfig{
			{
				ProxySettings: &outbound.Config{
					Receiver: &protocol.ServerEndpoint{
						Address: net.NewIPOrDomain(net.LocalHostIP),
						Port:    uint32(serverPort),
						User: &protocol.User{
							Account: serial.ToTypedMessage(&vmess.Account{
								Id: userID.String(),
								SecuritySettings: &protocol.SecurityConfig{
									Type: protocol.SecurityType_AES128_GCM,
								},
							}),
						},
					},
				},
			},
		},
	}

	servers, err := InitializeServerConfigs(serverConfig)
	common.Must(err)
	defer CloseAllServers(servers)

	client, err := core.New(clientConfig)
	common.Must(err)

	conn, err := core.Dial(context.Background(), client, dest)
	common.Must(err)
	defer conn.Close()

	if err := testTCPConn2(conn, 1024, time.Second*5)(); err != nil {
		t.Error(err)
	}
}
