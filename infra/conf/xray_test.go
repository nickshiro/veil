package conf_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/nickshiro/veil/app/dispatcher"
	"github.com/nickshiro/veil/app/log"
	"github.com/nickshiro/veil/app/proxyman"
	"github.com/nickshiro/veil/app/router"
	"github.com/nickshiro/veil/common"
	"github.com/nickshiro/veil/common/geodata"
	clog "github.com/nickshiro/veil/common/log"
	"github.com/nickshiro/veil/common/net"
	"github.com/nickshiro/veil/common/protocol"
	"github.com/nickshiro/veil/common/serial"
	core "github.com/nickshiro/veil/core"
	. "github.com/nickshiro/veil/infra/conf"
	"github.com/nickshiro/veil/proxy/vmess"
	"github.com/nickshiro/veil/proxy/vmess/inbound"
	"github.com/nickshiro/veil/transport/internet"
	"github.com/nickshiro/veil/transport/internet/tls"
	"github.com/nickshiro/veil/transport/internet/websocket"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestXrayConfig(t *testing.T) {
	config := new(Config)
	common.Must(json.Unmarshal([]byte(`{
				"log": {
					"access": "/var/log/xray/access.log",
					"loglevel": "error",
					"error": "/var/log/xray/error.log"
				},
				"inbounds": [{
					"streamSettings": {
						"network": "ws",
						"wsSettings": {
							"host": "example.domain",
							"path": ""
						},
						"tlsSettings": {
							"alpn": "h2"
						},
						"security": "tls"
					},
					"protocol": "vmess",
					"port": "443-500",
					"settings": {
						"clients": [
							{
								"security": "aes-128-gcm",
								"id": "0cdf8a45-303d-4fed-9780-29aa7f54175e"
							}
						]
					}
				}],
				"routing": {
					"rules": [
						{
							"ip": [
								"10.0.0.0/8"
							],
							"outboundTag": "blocked"
						}
					]
				}
			}`), config))

	actual, err := config.Build()
	common.Must(err)

	expected := &core.Config{
		App: []any{
			&log.Config{
				ErrorLogType:  log.LogType_File,
				ErrorLogPath:  "/var/log/xray/error.log",
				ErrorLogLevel: clog.Severity_Error,
				AccessLogType: log.LogType_File,
				AccessLogPath: "/var/log/xray/access.log",
			},
			&dispatcher.Config{},
			&proxyman.InboundConfig{},
			&proxyman.OutboundConfig{},
			&router.RuntimeConfig{
				DomainStrategy: router.Config_AsIs,
				Rule: []*router.RoutingRule{
					{
						Ip: []*geodata.IPRule{
							{
								Value: &geodata.IPRule_Custom{
									Custom: &geodata.CIDRRule{
										Cidr: &geodata.CIDR{Ip: []byte{10, 0, 0, 0}, Prefix: 8},
									},
								},
							},
						},
						TargetTag: &router.RoutingRule_Tag{
							Tag: "blocked",
						},
					},
				},
			},
		},
		Inbound: []*core.InboundConfig{
			{
				ReceiverSettings: &proxyman.ReceiverConfig{
					PortList: &net.PortList{Range: []*net.PortRange{{
						From: 443,
						To:   500,
					}}},
					StreamSettings: &internet.StreamConfig{
						ProtocolName: "websocket",
						TransportSettings: []*internet.TransportConfig{
							{
								ProtocolName: "websocket",
								Settings: serial.ToTypedMessage(&websocket.Config{
									Host: "example.domain",
								}),
							},
						},
						SecurityType: "xray.transport.internet.tls.Config",
						SecuritySettings: []*serial.TypedMessage{
							serial.ToTypedMessage(&tls.Config{
								NextProtocol: []string{"h2"},
							}),
						},
					},
				},
				ProxySettings: &inbound.Config{
					User: []*protocol.User{
						{
							Level: 0,
							Account: serial.ToTypedMessage(&vmess.Account{
								Id: "0cdf8a45-303d-4fed-9780-29aa7f54175e",
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

	if diff := cmp.Diff(actual, expected, protocmp.Transform()); diff != "" {
		t.Fatal(diff)
	}
}

func TestSniffingConfig_Build(t *testing.T) {
	config := &SniffingConfig{
		Enabled:         true,
		DestOverride:    StringList{"http", "tls"},
		DomainsExcluded: StringList{"full:api.example.com", "domain:blocked.example", "regexp:^test[0-9]+\\.internal$"},
		IPsExcluded:     StringList{"192.168.1.1", "2001:db8::/32"},
		MetadataOnly:    true,
		RouteOnly:       true,
	}

	built, err := config.Build()
	if err != nil {
		t.Fatalf("SniffingConfig.Build() failed: %v", err)
	}

	if !built.Enabled || !built.MetadataOnly || !built.RouteOnly {
		t.Fatalf("SniffingConfig.Build() lost sniffing flags: %+v", built)
	}
	if len(built.DestinationOverride) != 2 {
		t.Fatalf("SniffingConfig.Build() lost destination overrides: %+v", built.DestinationOverride)
	}
	if len(built.DomainsExcluded) != 3 {
		t.Fatalf("SniffingConfig.Build() produced %d domain rules", len(built.DomainsExcluded))
	}
	if len(built.IpsExcluded) != 2 {
		t.Fatalf("SniffingConfig.Build() produced %d ip rules", len(built.IpsExcluded))
	}

	want := []struct {
		ruleType geodata.Domain_Type
		value    string
	}{
		{ruleType: geodata.Domain_Full, value: "api.example.com"},
		{ruleType: geodata.Domain_Domain, value: "blocked.example"},
		{ruleType: geodata.Domain_Regex, value: "^test[0-9]+\\.internal$"},
	}
	for i, tc := range want {
		rule := built.DomainsExcluded[i].GetCustom()
		if rule == nil {
			t.Fatalf("SniffingConfig.Build() produced a non-custom rule at index %d", i)
		}
		if rule.Type != tc.ruleType || rule.Value != tc.value {
			t.Fatalf("SniffingConfig.Build() produced wrong rule at index %d: got (%v, %q), want (%v, %q)", i, rule.Type, rule.Value, tc.ruleType, tc.value)
		}
	}

	wantIPs := []struct {
		ip     []byte
		prefix uint32
	}{
		{ip: []byte(net.ParseAddress("192.168.1.1").IP()), prefix: 32},
		{ip: []byte(net.ParseAddress("2001:db8::").IP()), prefix: 32},
	}
	for i, tc := range wantIPs {
		rule := built.IpsExcluded[i].GetCustom()
		if rule == nil {
			t.Fatalf("SniffingConfig.Build() produced a non-custom ip rule at index %d", i)
		}
		cidr := rule.GetCidr()
		if cidr == nil {
			t.Fatalf("SniffingConfig.Build() produced a custom ip rule without cidr at index %d", i)
		}
		if !reflect.DeepEqual(cidr.Ip, tc.ip) || cidr.Prefix != tc.prefix {
			t.Fatalf("SniffingConfig.Build() produced wrong ip rule at index %d: got (%v, %d), want (%v, %d)", i, cidr.Ip, cidr.Prefix, tc.ip, tc.prefix)
		}
	}
}

func TestMuxConfig_Build(t *testing.T) {
	tests := []struct {
		name   string
		fields string
		want   *proxyman.MultiplexingConfig
	}{
		{"default", `{"enabled": true, "concurrency": 16}`, &proxyman.MultiplexingConfig{
			Enabled:         true,
			Concurrency:     16,
			XudpConcurrency: 0,
			XudpProxyUDP443: "reject",
		}},
		{"empty def", `{}`, &proxyman.MultiplexingConfig{
			Enabled:         false,
			Concurrency:     0,
			XudpConcurrency: 0,
			XudpProxyUDP443: "reject",
		}},
		{"not enable", `{"enabled": false, "concurrency": 4}`, &proxyman.MultiplexingConfig{
			Enabled:         false,
			Concurrency:     4,
			XudpConcurrency: 0,
			XudpProxyUDP443: "reject",
		}},
		{"forbidden", `{"enabled": false, "concurrency": -1}`, &proxyman.MultiplexingConfig{
			Enabled:         false,
			Concurrency:     -1,
			XudpConcurrency: 0,
			XudpProxyUDP443: "reject",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MuxConfig{}
			common.Must(json.Unmarshal([]byte(tt.fields), m))
			if got, _ := m.Build(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MuxConfig.Build() = %v, want %v", got, tt.want)
			}
		})
	}
}
