package conf_test

import (
	"encoding/json"
	"testing"
	"time"
	_ "unsafe"

	"github.com/google/go-cmp/cmp"
	"github.com/nickshiro/veil/app/router"
	"github.com/nickshiro/veil/common/geodata"
	"github.com/nickshiro/veil/common/net"
	. "github.com/nickshiro/veil/infra/conf"

	"google.golang.org/protobuf/testing/protocmp"
)

func TestRouterConfig(t *testing.T) {
	createParser := func() func(string) (*router.RuntimeConfig, error) {
		return func(s string) (*router.RuntimeConfig, error) {
			config := new(RouterConfig)
			if err := json.Unmarshal([]byte(s), config); err != nil {
				return nil, err
			}
			return config.Build()
		}
	}

	cases := []struct {
		Input  string
		Parser func(string) (*router.RuntimeConfig, error)
		Output *router.RuntimeConfig
	}{
		{
			Input: `{
				"domainStrategy": "AsIs",
				"rules": [
					{
						"domain": [
							"baidu.com",
							"qq.com"
						],
						"outboundTag": "direct"
					},
					{
						"ip": [
							"10.0.0.0/8",
							"::1/128"
						],
						"outboundTag": "test"
					},{
						"port": "53, 443, 1000-2000",
						"outboundTag": "test"
					},{
						"port": 123,
						"outboundTag": "test"
					}
				],
				"balancers": [
					{
						"tag": "b1",
						"selector": ["test"],
						"fallbackTag": "fall"
					},
					{
						"tag": "b2",
						"selector": ["test"],
						"strategy": {
							"type": "leastload",
							"settings": {
								"healthCheck": {
									"interval": "5m0s",
									"sampling": 2,
									"timeout": "5s",
									"destination": "dest",
									"connectivity": "conn"
								},
								"costs": [
									{
										"regexp": true,
										"match": "\\d+(\\.\\d+)",
										"value": 5
									}
								],
								"baselines": ["400ms", "600ms"],
								"expected": 6,
								"maxRTT": "1000ms",
								"tolerance": 0.5
							}
						},
						"fallbackTag": "fall"
					}
				]
			}`,
			Parser: createParser(),
			Output: &router.RuntimeConfig{
				DomainStrategy: router.Config_AsIs,
				BalancingRule: []*router.BalancerConfig{
					{
						Tag:              "b1",
						OutboundSelector: []string{"test"},
						Strategy:         "random",
						FallbackTag:      "fall",
					},
					{
						Tag:              "b2",
						OutboundSelector: []string{"test"},
						Strategy:         "leastload",
						StrategySettings: &router.LeastLoadConfig{
							Costs: []*router.Weight{
								{
									Regexp: true,
									Match:  "\\d+(\\.\\d+)",
									Value:  5,
								},
							},
							Baselines: []int64{
								int64(time.Duration(400) * time.Millisecond),
								int64(time.Duration(600) * time.Millisecond),
							},
							Expected:  6,
							MaxRTT:    int64(time.Duration(1000) * time.Millisecond),
							Tolerance: 0.5,
						},
						FallbackTag: "fall",
					},
				},
				Rule: []*router.RoutingRule{
					{
						Domain: []*geodata.DomainRule{
							{Value: &geodata.DomainRule_Custom{Custom: &geodata.Domain{Type: geodata.Domain_Substr, Value: "baidu.com"}}},
							{Value: &geodata.DomainRule_Custom{Custom: &geodata.Domain{Type: geodata.Domain_Substr, Value: "qq.com"}}},
						},
						TargetTag: &router.RoutingRule_Tag{
							Tag: "direct",
						},
					},
					{
						Ip: []*geodata.IPRule{
							{
								Value: &geodata.IPRule_Custom{
									Custom: &geodata.CIDRRule{
										Cidr: &geodata.CIDR{Ip: []byte{10, 0, 0, 0}, Prefix: 8},
									},
								},
							},
							{
								Value: &geodata.IPRule_Custom{
									Custom: &geodata.CIDRRule{
										Cidr: &geodata.CIDR{Ip: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}, Prefix: 128},
									},
								},
							},
						},
						TargetTag: &router.RoutingRule_Tag{
							Tag: "test",
						},
					},
					{
						PortList: &net.PortList{
							Range: []*net.PortRange{
								{From: 53, To: 53},
								{From: 443, To: 443},
								{From: 1000, To: 2000},
							},
						},
						TargetTag: &router.RoutingRule_Tag{
							Tag: "test",
						},
					},
					{
						PortList: &net.PortList{
							Range: []*net.PortRange{
								{From: 123, To: 123},
							},
						},
						TargetTag: &router.RoutingRule_Tag{
							Tag: "test",
						},
					},
				},
			},
		},
		{
			Input: `{
				"domainStrategy": "IPIfNonMatch",
				"rules": [
					{
						"domain": [
							"baidu.com",
							"qq.com"
						],
						"outboundTag": "direct"
					},
					{
						"ip": [
							"10.0.0.0/8",
							"::1/128"
						],
						"outboundTag": "test"
					}
				]
			}`,
			Parser: createParser(),
			Output: &router.RuntimeConfig{
				DomainStrategy: router.Config_IpIfNonMatch,
				Rule: []*router.RoutingRule{
					{
						Domain: []*geodata.DomainRule{
							{Value: &geodata.DomainRule_Custom{Custom: &geodata.Domain{Type: geodata.Domain_Substr, Value: "baidu.com"}}},
							{Value: &geodata.DomainRule_Custom{Custom: &geodata.Domain{Type: geodata.Domain_Substr, Value: "qq.com"}}},
						},
						TargetTag: &router.RoutingRule_Tag{
							Tag: "direct",
						},
					},
					{
						Ip: []*geodata.IPRule{
							{
								Value: &geodata.IPRule_Custom{
									Custom: &geodata.CIDRRule{
										Cidr: &geodata.CIDR{Ip: []byte{10, 0, 0, 0}, Prefix: 8},
									},
								},
							},
							{
								Value: &geodata.IPRule_Custom{
									Custom: &geodata.CIDRRule{
										Cidr: &geodata.CIDR{Ip: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}, Prefix: 128},
									},
								},
							},
						},
						TargetTag: &router.RoutingRule_Tag{
							Tag: "test",
						},
					},
				},
			},
		},
	}
	for _, tc := range cases {
		got, err := tc.Parser(tc.Input)
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(tc.Output, got, protocmp.Transform()); diff != "" {
			t.Error(diff)
		}
	}
}
