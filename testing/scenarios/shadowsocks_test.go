package scenarios

import (
	"testing"
	"time"

	"github.com/nickshiro/veil/app/log"
	"github.com/nickshiro/veil/app/proxyman"
	"github.com/nickshiro/veil/common"
	clog "github.com/nickshiro/veil/common/log"
	"github.com/nickshiro/veil/common/net"
	"github.com/nickshiro/veil/common/protocol"
	"github.com/nickshiro/veil/common/serial"
	"github.com/nickshiro/veil/core"
	"github.com/nickshiro/veil/proxy/dokodemo"
	"github.com/nickshiro/veil/proxy/freedom"
	"github.com/nickshiro/veil/proxy/shadowsocks"
	"github.com/nickshiro/veil/testing/servers/tcp"
	"github.com/nickshiro/veil/testing/servers/udp"
	"golang.org/x/sync/errgroup"
)

func TestShadowsocksChaCha20Poly1305TCP(t *testing.T) {
	tcpServer := tcp.Server{
		MsgProcessor: xor,
	}
	dest, err := tcpServer.Start()
	common.Must(err)
	defer tcpServer.Close()

	account := serial.ToTypedMessage(&shadowsocks.Account{
		Password:   "shadowsocks-password",
		CipherType: shadowsocks.CipherType_CHACHA20_POLY1305,
	})

	serverPort := tcp.PickPort()
	serverConfig := &core.Config{
		Inbound: []*core.InboundConfig{
			{
				ReceiverSettings: &proxyman.ReceiverConfig{
					PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(serverPort)}},
					Listen:   net.NewIPOrDomain(net.LocalHostIP),
				},
				ProxySettings: &shadowsocks.ServerConfig{
					Users: []*protocol.User{{
						Account: account,
						Level:   1,
					}},
					Network: []net.Network{net.Network_TCP},
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
				ProxySettings: &shadowsocks.ClientConfig{
					Server: &protocol.ServerEndpoint{
						Address: net.NewIPOrDomain(net.LocalHostIP),
						Port:    uint32(serverPort),
						User: &protocol.User{
							Account: account,
						},
					},
				},
			},
		},
	}

	servers, err := InitializeServerConfigs(serverConfig, clientConfig)
	common.Must(err)
	defer CloseAllServers(servers)

	var errGroup errgroup.Group
	for range 3 {
		errGroup.Go(testTCPConn(clientPort, 10240*1024, time.Second*20))
	}
	if err := errGroup.Wait(); err != nil {
		t.Error(err)
	}
}

func TestShadowsocksAES256GCMTCP(t *testing.T) {
	tcpServer := tcp.Server{
		MsgProcessor: xor,
	}
	dest, err := tcpServer.Start()
	common.Must(err)
	defer tcpServer.Close()

	account := serial.ToTypedMessage(&shadowsocks.Account{
		Password:   "shadowsocks-password",
		CipherType: shadowsocks.CipherType_AES_256_GCM,
	})

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
				ProxySettings: &shadowsocks.ServerConfig{
					Users: []*protocol.User{{
						Account: account,
						Level:   1,
					}},
					Network: []net.Network{net.Network_TCP},
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
		App: []any{
			&log.Config{
				ErrorLogLevel: clog.Severity_Debug,
				ErrorLogType:  log.LogType_Console,
			},
		},
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
				ProxySettings: &shadowsocks.ClientConfig{
					Server: &protocol.ServerEndpoint{
						Address: net.NewIPOrDomain(net.LocalHostIP),
						Port:    uint32(serverPort),
						User: &protocol.User{
							Account: account,
						},
					},
				},
			},
		},
	}

	servers, err := InitializeServerConfigs(serverConfig, clientConfig)
	common.Must(err)
	defer CloseAllServers(servers)

	var errGroup errgroup.Group
	for range 3 {
		errGroup.Go(testTCPConn(clientPort, 10240*1024, time.Second*20))
	}

	if err := errGroup.Wait(); err != nil {
		t.Error(err)
	}
}

func TestShadowsocksAES128GCMUDP(t *testing.T) {
	udpServer := udp.Server{
		MsgProcessor: xor,
	}
	dest, err := udpServer.Start()
	common.Must(err)
	defer udpServer.Close()

	account := serial.ToTypedMessage(&shadowsocks.Account{
		Password:   "shadowsocks-password",
		CipherType: shadowsocks.CipherType_AES_128_GCM,
	})

	serverPort := udp.PickPort()
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
				ProxySettings: &shadowsocks.ServerConfig{
					Users: []*protocol.User{{
						Account: account,
						Level:   1,
					}},
					Network: []net.Network{net.Network_UDP},
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

	clientPort := udp.PickPort()
	clientConfig := &core.Config{
		App: []any{
			&log.Config{
				ErrorLogLevel: clog.Severity_Debug,
				ErrorLogType:  log.LogType_Console,
			},
		},
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
				ProxySettings: &shadowsocks.ClientConfig{
					Server: &protocol.ServerEndpoint{
						Address: net.NewIPOrDomain(net.LocalHostIP),
						Port:    uint32(serverPort),
						User: &protocol.User{
							Account: account,
						},
					},
				},
			},
		},
	}

	servers, err := InitializeServerConfigs(serverConfig, clientConfig)
	common.Must(err)
	defer CloseAllServers(servers)

	var errGroup errgroup.Group
	for range 3 {
		errGroup.Go(testUDPConn(clientPort, 1024, time.Second*5))
	}
	if err := errGroup.Wait(); err != nil {
		t.Error(err)
	}
}

func TestShadowsocksAES128GCMUDPMux(t *testing.T) {
	udpServer := udp.Server{
		MsgProcessor: xor,
	}
	dest, err := udpServer.Start()
	common.Must(err)
	defer udpServer.Close()

	account := serial.ToTypedMessage(&shadowsocks.Account{
		Password:   "shadowsocks-password",
		CipherType: shadowsocks.CipherType_AES_128_GCM,
	})

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
				ProxySettings: &shadowsocks.ServerConfig{
					Users: []*protocol.User{{
						Account: account,
						Level:   1,
					}},
					Network: []net.Network{net.Network_TCP},
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

	clientPort := udp.PickPort()
	clientConfig := &core.Config{
		App: []any{
			&log.Config{
				ErrorLogLevel: clog.Severity_Debug,
				ErrorLogType:  log.LogType_Console,
			},
		},
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
				SenderSettings: &proxyman.SenderConfig{
					MultiplexSettings: &proxyman.MultiplexingConfig{
						Enabled:     true,
						Concurrency: 8,
					},
				},
				ProxySettings: &shadowsocks.ClientConfig{
					Server: &protocol.ServerEndpoint{
						Address: net.NewIPOrDomain(net.LocalHostIP),
						Port:    uint32(serverPort),
						User: &protocol.User{
							Account: account,
						},
					},
				},
			},
		},
	}

	servers, err := InitializeServerConfigs(serverConfig, clientConfig)
	common.Must(err)
	defer CloseAllServers(servers)

	var errGroup errgroup.Group
	for range 3 {
		errGroup.Go(testUDPConn(clientPort, 1024, time.Second*5))
	}
	if err := errGroup.Wait(); err != nil {
		t.Error(err)
	}
}
