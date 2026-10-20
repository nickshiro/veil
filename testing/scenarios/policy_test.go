package scenarios

import (
	"io"
	"testing"
	"time"
	"uuid"

	"github.com/nickshiro/veil/app/log"
	"github.com/nickshiro/veil/app/policy"
	"github.com/nickshiro/veil/app/proxyman"
	"github.com/nickshiro/veil/common"
	clog "github.com/nickshiro/veil/common/log"
	"github.com/nickshiro/veil/common/net"
	"github.com/nickshiro/veil/common/protocol"
	"github.com/nickshiro/veil/common/serial"
	"github.com/nickshiro/veil/core"
	"github.com/nickshiro/veil/proxy/dokodemo"
	"github.com/nickshiro/veil/proxy/freedom"
	"github.com/nickshiro/veil/proxy/vmess"
	"github.com/nickshiro/veil/proxy/vmess/inbound"
	"github.com/nickshiro/veil/proxy/vmess/outbound"
	"github.com/nickshiro/veil/testing/servers/tcp"
	"golang.org/x/sync/errgroup"
)

func startQuickClosingTCPServer() (net.Listener, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				break
			}
			b := make([]byte, 1024)
			conn.Read(b)
			conn.Close()
		}
	}()
	return listener, nil
}

func TestVMessClosing(t *testing.T) {
	tcpServer, err := startQuickClosingTCPServer()
	common.Must(err)
	defer tcpServer.Close()

	dest := net.DestinationFromAddr(tcpServer.Addr())

	userID := protocol.NewID(uuid.NewV4())
	serverPort := tcp.PickPort()
	serverConfig := &core.Config{
		App: []any{
			&policy.Config{
				Level: map[uint32]*policy.Policy{
					0: {
						Timeout: &policy.Policy_Timeout{
							UplinkOnly:   &policy.Second{Value: 0},
							DownlinkOnly: &policy.Second{Value: 0},
						},
					},
				},
			},
		},
		Inbound: []*core.InboundHandlerConfig{
			{
				ReceiverSettings: serial.ToTypedMessage(&proxyman.ReceiverConfig{
					PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(serverPort)}},
					Listen:   net.NewIPOrDomain(net.LocalHostIP),
				}),
				ProxySettings: serial.ToTypedMessage(&inbound.Config{
					User: []*protocol.User{
						{
							Account: serial.ToTypedMessage(&vmess.Account{
								Id: userID.String(),
							}),
						},
					},
				}),
			},
		},
		Outbound: []*core.OutboundHandlerConfig{
			{
				ProxySettings: serial.ToTypedMessage(&freedom.Config{
					FinalRules: []*freedom.FinalRuleConfig{{Action: freedom.RuleAction_Allow}},
				}),
			},
		},
	}

	clientPort := tcp.PickPort()
	clientConfig := &core.Config{
		App: []any{
			&policy.Config{
				Level: map[uint32]*policy.Policy{
					0: {
						Timeout: &policy.Policy_Timeout{
							UplinkOnly:   &policy.Second{Value: 0},
							DownlinkOnly: &policy.Second{Value: 0},
						},
					},
				},
			},
		},
		Inbound: []*core.InboundHandlerConfig{
			{
				ReceiverSettings: serial.ToTypedMessage(&proxyman.ReceiverConfig{
					PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(clientPort)}},
					Listen:   net.NewIPOrDomain(net.LocalHostIP),
				}),
				ProxySettings: serial.ToTypedMessage(&dokodemo.Config{
					RewriteAddress:  net.NewIPOrDomain(dest.Address),
					RewritePort:     uint32(dest.Port),
					AllowedNetworks: []net.Network{net.Network_TCP},
				}),
			},
		},
		Outbound: []*core.OutboundHandlerConfig{
			{
				ProxySettings: serial.ToTypedMessage(&outbound.Config{
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
				}),
			},
		},
	}

	servers, err := InitializeServerConfigs(serverConfig, clientConfig)
	common.Must(err)
	defer CloseAllServers(servers)

	if err := testTCPConn(clientPort, 1024, time.Second*2)(); err != io.EOF {
		t.Error(err)
	}
}

func TestZeroBuffer(t *testing.T) {
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
			&policy.Config{
				Level: map[uint32]*policy.Policy{
					0: {
						Timeout: &policy.Policy_Timeout{
							UplinkOnly:   &policy.Second{Value: 0},
							DownlinkOnly: &policy.Second{Value: 0},
						},
						Buffer: &policy.Policy_Buffer{
							Connection: 0,
						},
					},
				},
			},
		},
		Inbound: []*core.InboundHandlerConfig{
			{
				ReceiverSettings: serial.ToTypedMessage(&proxyman.ReceiverConfig{
					PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(serverPort)}},
					Listen:   net.NewIPOrDomain(net.LocalHostIP),
				}),
				ProxySettings: serial.ToTypedMessage(&inbound.Config{
					User: []*protocol.User{
						{
							Account: serial.ToTypedMessage(&vmess.Account{
								Id: userID.String(),
							}),
						},
					},
				}),
			},
		},
		Outbound: []*core.OutboundHandlerConfig{
			{
				ProxySettings: serial.ToTypedMessage(&freedom.Config{
					FinalRules: []*freedom.FinalRuleConfig{{Action: freedom.RuleAction_Allow}},
				}),
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
		Inbound: []*core.InboundHandlerConfig{
			{
				ReceiverSettings: serial.ToTypedMessage(&proxyman.ReceiverConfig{
					PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(clientPort)}},
					Listen:   net.NewIPOrDomain(net.LocalHostIP),
				}),
				ProxySettings: serial.ToTypedMessage(&dokodemo.Config{
					RewriteAddress:  net.NewIPOrDomain(dest.Address),
					RewritePort:     uint32(dest.Port),
					AllowedNetworks: []net.Network{net.Network_TCP},
				}),
			},
		},
		Outbound: []*core.OutboundHandlerConfig{
			{
				ProxySettings: serial.ToTypedMessage(&outbound.Config{
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
				}),
			},
		},
	}

	servers, err := InitializeServerConfigs(serverConfig, clientConfig)
	common.Must(err)
	defer CloseAllServers(servers)

	var errg errgroup.Group
	for range 3 {
		errg.Go(testTCPConn(clientPort, 10240*1024, time.Second*20))
	}
	if err := errg.Wait(); err != nil {
		t.Error(err)
	}
}
