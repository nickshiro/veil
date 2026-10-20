package core_test

import (
	"testing"
	"uuid"

	"github.com/nickshiro/veil/app/dispatcher"
	"github.com/nickshiro/veil/app/proxyman"
	"github.com/nickshiro/veil/common"
	"github.com/nickshiro/veil/common/net"
	"github.com/nickshiro/veil/common/protocol"
	"github.com/nickshiro/veil/common/serial"
	. "github.com/nickshiro/veil/core"
	"github.com/nickshiro/veil/features/dns"
	"github.com/nickshiro/veil/features/dns/localdns"
	_ "github.com/nickshiro/veil/main/distro/all"
	"github.com/nickshiro/veil/proxy/dokodemo"
	"github.com/nickshiro/veil/proxy/vmess"
	"github.com/nickshiro/veil/proxy/vmess/outbound"
	"github.com/nickshiro/veil/testing/servers/tcp"
)

func TestXrayDependency(t *testing.T) {
	instance := new(Instance)

	wait := make(chan bool, 1)
	instance.RequireFeatures(func(d dns.Client) {
		if d == nil {
			t.Error("expected dns client fulfilled, but actually nil")
		}
		wait <- true
	}, false)
	instance.AddFeature(localdns.New())
	<-wait
}

func TestXrayClose(t *testing.T) {
	port := tcp.PickPort()

	userID := uuid.NewV4()
	config := &Config{
		App: []any{
			&dispatcher.Config{},
			&proxyman.InboundConfig{},
			&proxyman.OutboundConfig{},
		},
		Inbound: []*InboundConfig{
			{
				ReceiverSettings: &proxyman.ReceiverConfig{
					PortList: &net.PortList{
						Range: []*net.PortRange{net.SinglePortRange(port)},
					},
					Listen: net.NewIPOrDomain(net.LocalHostIP),
				},
				ProxySettings: &dokodemo.Config{
					RewriteAddress:  net.NewIPOrDomain(net.LocalHostIP),
					RewritePort:     uint32(0),
					AllowedNetworks: []net.Network{net.Network_TCP},
				},
			},
		},
		Outbound: []*OutboundConfig{
			{
				ProxySettings: &outbound.Config{
					Receiver: &protocol.ServerEndpoint{
						Address: net.NewIPOrDomain(net.LocalHostIP),
						Port:    uint32(0),
						User: &protocol.User{
							Account: serial.ToTypedMessage(&vmess.Account{
								Id: userID.String(),
							}),
						},
					},
				},
			},
		},
	}

	server, err := New(config)
	common.Must(err)
	common.Must(server.Start())
	server.Close()
}
