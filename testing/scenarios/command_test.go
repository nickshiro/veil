package scenarios

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/nickshiro/veil/app/commander"
	"github.com/nickshiro/veil/app/policy"
	"github.com/nickshiro/veil/app/proxyman"
	"github.com/nickshiro/veil/app/proxyman/command"
	"github.com/nickshiro/veil/app/router"
	"github.com/nickshiro/veil/app/stats"
	statscmd "github.com/nickshiro/veil/app/stats/command"
	"github.com/nickshiro/veil/common"
	"github.com/nickshiro/veil/common/net"
	"github.com/nickshiro/veil/common/protocol"
	"github.com/nickshiro/veil/common/serial"
	core "github.com/nickshiro/veil/core"
	"github.com/nickshiro/veil/proxy/dokodemo"
	"github.com/nickshiro/veil/proxy/freedom"
	"github.com/nickshiro/veil/proxy/vmess"
	"github.com/nickshiro/veil/proxy/vmess/inbound"
	"github.com/nickshiro/veil/proxy/vmess/outbound"
	"github.com/nickshiro/veil/testing/servers/tcp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	reflectionpb "google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestCommanderListenConfigurationItem(t *testing.T) {
	tcpServer := tcp.Server{
		MsgProcessor: xor,
	}
	dest, err := tcpServer.Start()
	common.Must(err)
	defer tcpServer.Close()

	clientPort := tcp.PickPort()
	cmdPort := tcp.PickPort()
	clientConfig := &core.Config{
		App: []any{
			&commander.Config{
				Tag:    "api",
				Listen: fmt.Sprintf("127.0.0.1:%d", cmdPort),
				Service: []any{
					&command.Config{},
					&commander.ReflectionConfig{},
				},
			},
		},
		Inbound: []*core.InboundConfig{
			{
				Tag: "d",
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
				Tag:           "default-outbound",
				ProxySettings: &freedom.Config{FinalRules: []*freedom.FinalRuleConfig{{Action: freedom.RuleAction_Allow}}},
			},
		},
	}

	servers, err := InitializeServerConfigs(clientConfig)
	common.Must(err)
	defer CloseAllServers(servers)

	if err := testTCPConn(clientPort, 1024, time.Second*5)(); err != nil {
		t.Fatal(err)
	}

	cmdConn, err := grpc.Dial(fmt.Sprintf("127.0.0.1:%d", cmdPort), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	common.Must(err)
	defer cmdConn.Close()

	reflectionCtx, cancelReflection := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelReflection()
	stream, err := reflectionpb.NewServerReflectionClient(cmdConn).ServerReflectionInfo(reflectionCtx)
	common.Must(err)
	common.Must(stream.Send(&reflectionpb.ServerReflectionRequest{
		MessageRequest: &reflectionpb.ServerReflectionRequest_ListServices{ListServices: ""},
	}))
	reflectionResponse, err := stream.Recv()
	common.Must(err)
	foundHandlerService := false
	for _, service := range reflectionResponse.GetListServicesResponse().GetService() {
		if service.Name == "xray.app.proxyman.command.HandlerService" {
			foundHandlerService = true
		}
	}
	if !foundHandlerService {
		t.Fatal("reflection did not expose HandlerService")
	}
	common.Must(stream.CloseSend())

	hsClient := command.NewHandlerServiceClient(cmdConn)
	resp, err := hsClient.RemoveInbound(context.Background(), &command.RemoveInboundRequest{
		Tag: "d",
	})
	common.Must(err)
	if resp == nil {
		t.Error("unexpected nil response")
	}

	{
		_, err := net.DialTCP("tcp", nil, &net.TCPAddr{
			IP:   []byte{127, 0, 0, 1},
			Port: int(clientPort),
		})
		if err == nil {
			t.Error("unexpected nil error")
		}
	}
}

func TestCommanderRemoveHandler(t *testing.T) {
	tcpServer := tcp.Server{
		MsgProcessor: xor,
	}
	dest, err := tcpServer.Start()
	common.Must(err)
	defer tcpServer.Close()

	clientPort := tcp.PickPort()
	cmdPort := tcp.PickPort()
	clientConfig := &core.Config{
		App: []any{
			&commander.Config{
				Tag: "api",
				Service: []any{
					&command.Config{},
				},
			},
			&router.Config{
				Rule: []*router.RoutingRule{
					{
						InboundTag: []string{"api"},
						TargetTag: &router.RoutingRule_Tag{
							Tag: "api",
						},
					},
				},
			},
		},
		Inbound: []*core.InboundConfig{
			{
				Tag: "d",
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
			{
				Tag: "api",
				ReceiverSettings: &proxyman.ReceiverConfig{
					PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(cmdPort)}},
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
				Tag:           "default-outbound",
				ProxySettings: &freedom.Config{FinalRules: []*freedom.FinalRuleConfig{{Action: freedom.RuleAction_Allow}}},
			},
		},
	}

	servers, err := InitializeServerConfigs(clientConfig)
	common.Must(err)
	defer CloseAllServers(servers)

	if err := testTCPConn(clientPort, 1024, time.Second*5)(); err != nil {
		t.Fatal(err)
	}

	cmdConn, err := grpc.Dial(fmt.Sprintf("127.0.0.1:%d", cmdPort), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	common.Must(err)
	defer cmdConn.Close()

	hsClient := command.NewHandlerServiceClient(cmdConn)
	resp, err := hsClient.RemoveInbound(context.Background(), &command.RemoveInboundRequest{
		Tag: "d",
	})
	common.Must(err)
	if resp == nil {
		t.Error("unexpected nil response")
	}

	{
		_, err := net.DialTCP("tcp", nil, &net.TCPAddr{
			IP:   []byte{127, 0, 0, 1},
			Port: int(clientPort),
		})
		if err == nil {
			t.Error("unexpected nil error")
		}
	}
}

func TestCommanderListHandlers(t *testing.T) {
	tcpServer := tcp.Server{
		MsgProcessor: xor,
	}
	dest, err := tcpServer.Start()
	common.Must(err)
	defer tcpServer.Close()

	clientPort := tcp.PickPort()
	cmdPort := tcp.PickPort()
	clientConfig := &core.Config{
		App: []any{
			&commander.Config{
				Tag: "api",
				Service: []any{
					&command.Config{},
				},
			},
			&router.Config{
				Rule: []*router.RoutingRule{
					{
						InboundTag: []string{"api"},
						TargetTag: &router.RoutingRule_Tag{
							Tag: "api",
						},
					},
				},
			},
		},
		Inbound: []*core.InboundConfig{
			{
				Tag: "d",
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
			{
				Tag: "api",
				ReceiverSettings: &proxyman.ReceiverConfig{
					PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(cmdPort)}},
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
				Tag:            "default-outbound",
				SenderSettings: &proxyman.SenderConfig{},
				ProxySettings:  &freedom.Config{FinalRules: []*freedom.FinalRuleConfig{{Action: freedom.RuleAction_Allow}}},
			},
		},
	}

	servers, err := InitializeServerConfigs(clientConfig)
	common.Must(err)
	defer CloseAllServers(servers)

	if err := testTCPConn(clientPort, 1024, time.Second*5)(); err != nil {
		t.Fatal(err)
	}

	cmdConn, err := grpc.Dial(fmt.Sprintf("127.0.0.1:%d", cmdPort), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	common.Must(err)
	defer cmdConn.Close()

	hsClient := command.NewHandlerServiceClient(cmdConn)
	inboundResp, err := hsClient.ListInbounds(context.Background(), &command.ListInboundsRequest{})
	common.Must(err)
	if inboundResp == nil {
		t.Error("unexpected nil response")
	}

	var wantInbounds []*core.InboundHandlerConfig
	for _, c := range clientConfig.Inbound {
		wantInbounds = append(wantInbounds, &core.InboundHandlerConfig{
			Tag:              c.Tag,
			ReceiverSettings: serial.ToTypedMessage(c.ReceiverSettings.(proto.Message)),
			ProxySettings:    serial.ToTypedMessage(c.ProxySettings.(proto.Message)),
		})
	}
	if diff := cmp.Diff(
		inboundResp.Inbounds,
		wantInbounds,
		protocmp.Transform(),
		cmpopts.SortSlices(func(a, b *core.InboundHandlerConfig) bool {
			return a.Tag < b.Tag
		}),
	); diff != "" {
		t.Fatalf("inbound response doesn't match config (-want +got):\n%s", diff)
	}

	outboundResp, err := hsClient.ListOutbounds(context.Background(), &command.ListOutboundsRequest{})
	common.Must(err)
	if outboundResp == nil {
		t.Error("unexpected nil response")
	}

	var wantOutbounds []*core.OutboundHandlerConfig
	for _, c := range clientConfig.Outbound {
		var sender *serial.TypedMessage
		if c.SenderSettings != nil {
			sender = serial.ToTypedMessage(c.SenderSettings.(proto.Message))
		}
		wantOutbounds = append(wantOutbounds, &core.OutboundHandlerConfig{
			Tag:            c.Tag,
			SenderSettings: sender,
			ProxySettings:  serial.ToTypedMessage(c.ProxySettings.(proto.Message)),
		})
	}
	if diff := cmp.Diff(
		outboundResp.Outbounds,
		wantOutbounds,
		protocmp.Transform(),
		cmpopts.SortSlices(func(a, b *core.OutboundHandlerConfig) bool {
			return a.Tag < b.Tag
		}),
	); diff != "" {
		t.Fatalf("outbound response doesn't match config (-want +got):\n%s", diff)
	}
}

func TestCommanderAddRemoveUser(t *testing.T) {
	tcpServer := tcp.Server{
		MsgProcessor: xor,
	}
	dest, err := tcpServer.Start()
	common.Must(err)
	defer tcpServer.Close()

	u1 := protocol.NewID(uuid.NewV4())
	u2 := protocol.NewID(uuid.NewV4())

	cmdPort := tcp.PickPort()
	serverPort := tcp.PickPort()
	serverConfig := &core.Config{
		App: []any{
			&commander.Config{
				Tag: "api",
				Service: []any{
					&command.Config{},
				},
			},
			&router.Config{
				Rule: []*router.RoutingRule{
					{
						InboundTag: []string{"api"},
						TargetTag: &router.RoutingRule_Tag{
							Tag: "api",
						},
					},
				},
			},
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
		Inbound: []*core.InboundConfig{
			{
				Tag: "v",
				ReceiverSettings: &proxyman.ReceiverConfig{
					PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(serverPort)}},
					Listen:   net.NewIPOrDomain(net.LocalHostIP),
				},
				ProxySettings: &inbound.Config{
					User: []*protocol.User{
						{
							Account: serial.ToTypedMessage(&vmess.Account{
								Id: u1.String(),
							}),
						},
					},
				},
			},
			{
				Tag: "api",
				ReceiverSettings: &proxyman.ReceiverConfig{
					PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(cmdPort)}},
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
		Inbound: []*core.InboundConfig{
			{
				Tag: "d",
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
								Id: u2.String(),
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

	servers, err := InitializeServerConfigs(serverConfig, clientConfig)
	common.Must(err)
	defer CloseAllServers(servers)

	if err := testTCPConn(clientPort, 1024, time.Second*5)(); err != io.EOF &&
		/*We might wish to drain the connection*/
		(err != nil && !strings.HasSuffix(err.Error(), "i/o timeout")) {
		t.Fatal("expected error: ", err)
	}

	cmdConn, err := grpc.Dial(fmt.Sprintf("127.0.0.1:%d", cmdPort), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	common.Must(err)
	defer cmdConn.Close()

	hsClient := command.NewHandlerServiceClient(cmdConn)
	resp, err := hsClient.AlterInbound(context.Background(), &command.AlterInboundRequest{
		Tag: "v",
		Operation: serial.ToTypedMessage(
			&command.AddUserOperation{
				User: &protocol.User{
					Email: "test@example.com",
					Account: serial.ToTypedMessage(&vmess.Account{
						Id: u2.String(),
					}),
				},
			},
		),
	})
	common.Must(err)
	if resp == nil {
		t.Fatal("nil response")
	}

	if err := testTCPConn(clientPort, 1024, time.Second*5)(); err != nil {
		t.Fatal(err)
	}

	resp, err = hsClient.AlterInbound(context.Background(), &command.AlterInboundRequest{
		Tag:       "v",
		Operation: serial.ToTypedMessage(&command.RemoveUserOperation{Email: "test@example.com"}),
	})
	common.Must(err)
	if resp == nil {
		t.Fatal("nil response")
	}
}

func TestCommanderStats(t *testing.T) {
	tcpServer := tcp.Server{
		MsgProcessor: xor,
	}
	dest, err := tcpServer.Start()
	common.Must(err)
	defer tcpServer.Close()

	userID := protocol.NewID(uuid.NewV4())
	serverPort := tcp.PickPort()
	cmdPort := tcp.PickPort()

	serverConfig := &core.Config{
		App: []any{
			&stats.Config{},
			&commander.Config{
				Tag: "api",
				Service: []any{
					&statscmd.Config{},
				},
			},
			&router.Config{
				Rule: []*router.RoutingRule{
					{
						InboundTag: []string{"api"},
						TargetTag: &router.RoutingRule_Tag{
							Tag: "api",
						},
					},
				},
			},
			&policy.Config{
				Level: map[uint32]*policy.Policy{
					0: {
						Timeout: &policy.Policy_Timeout{
							UplinkOnly:   &policy.Second{Value: 0},
							DownlinkOnly: &policy.Second{Value: 0},
						},
					},
					1: {
						Stats: &policy.Policy_Stats{
							UserUplink:   true,
							UserDownlink: true,
						},
					},
				},
				System: &policy.SystemPolicy{
					Stats: &policy.SystemPolicy_Stats{
						InboundUplink: true,
					},
				},
			},
		},
		Inbound: []*core.InboundConfig{
			{
				Tag: "vmess",
				ReceiverSettings: &proxyman.ReceiverConfig{
					PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(serverPort)}},
					Listen:   net.NewIPOrDomain(net.LocalHostIP),
				},
				ProxySettings: &inbound.Config{
					User: []*protocol.User{
						{
							Level: 1,
							Email: "test",
							Account: serial.ToTypedMessage(&vmess.Account{
								Id: userID.String(),
							}),
						},
					},
				},
			},
			{
				Tag: "api",
				ReceiverSettings: &proxyman.ReceiverConfig{
					PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(cmdPort)}},
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

	servers, err := InitializeServerConfigs(serverConfig, clientConfig)
	if err != nil {
		t.Fatal("Failed to create all servers", err)
	}
	defer CloseAllServers(servers)

	if err := testTCPConn(clientPort, 10240*1024, time.Second*20)(); err != nil {
		t.Fatal(err)
	}

	cmdConn, err := grpc.Dial(fmt.Sprintf("127.0.0.1:%d", cmdPort), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	common.Must(err)
	defer cmdConn.Close()

	const name = "user>>>test>>>traffic>>>uplink"
	sClient := statscmd.NewStatsServiceClient(cmdConn)

	sresp, err := sClient.GetStats(context.Background(), &statscmd.GetStatsRequest{
		Name:   name,
		Reset_: true,
	})
	common.Must(err)
	if r := cmp.Diff(sresp.Stat, &statscmd.Stat{
		Name:  name,
		Value: 10240 * 1024,
	}, cmpopts.IgnoreUnexported(statscmd.Stat{})); r != "" {
		t.Error(r)
	}

	sresp, err = sClient.GetStats(context.Background(), &statscmd.GetStatsRequest{
		Name: name,
	})
	common.Must(err)
	if r := cmp.Diff(sresp.Stat, &statscmd.Stat{
		Name:  name,
		Value: 0,
	}, cmpopts.IgnoreUnexported(statscmd.Stat{})); r != "" {
		t.Error(r)
	}

	sresp, err = sClient.GetStats(context.Background(), &statscmd.GetStatsRequest{
		Name:   "inbound>>>vmess>>>traffic>>>uplink",
		Reset_: true,
	})
	common.Must(err)
	if sresp.Stat.Value <= 10240*1024 {
		t.Error("value < 10240*1024: ", sresp.Stat.Value)
	}
}
