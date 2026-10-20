package command

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/nickshiro/veil/app/router"
	"github.com/nickshiro/veil/common/geodata"
	"github.com/nickshiro/veil/common/net"
	"github.com/nickshiro/veil/common/serial"
	"github.com/nickshiro/veil/features/routing"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

type updatingRouter struct {
	routing.Router
	config *router.RuntimeConfig
	append bool
}

func (r *updatingRouter) ReloadRules(c *router.RuntimeConfig, append bool) error {
	r.config, r.append = c, append
	return nil
}

func TestAddRuleConvertsWireConfig(t *testing.T) {
	wire := &router.Config{
		DomainStrategy: router.Config_IpOnDemand,
		Rule:           []*router.RoutingRule{{RuleTag: "rule", TargetTag: &router.RoutingRule_Tag{Tag: "direct"}}},
		BalancingRule: []*router.BalancingRule{{Tag: "balance", OutboundSelector: []string{"proxy"}, Strategy: "leastLoad", FallbackTag: "direct", StrategySettings: serial.ToTypedMessage(&router.StrategyLeastLoadConfig{
			Costs: []*router.StrategyWeight{{Regexp: true, Match: "proxy.*", Value: 2}}, Baselines: []int64{100, 200}, Expected: 2, MaxRTT: 300, Tolerance: 0.5,
		})}},
	}
	// Exercise the actual protobuf request format before crossing the API boundary.
	data, err := proto.Marshal(&AddRuleRequest{Config: serial.ToTypedMessage(wire), ShouldAppend: true})
	if err != nil {
		t.Fatal(err)
	}
	request := new(AddRuleRequest)
	if err := proto.Unmarshal(data, request); err != nil {
		t.Fatal(err)
	}
	target := new(updatingRouter)
	if _, err := NewRoutingServer(target, nil).AddRule(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	want := &router.RuntimeConfig{DomainStrategy: router.DomainIPOnDemand, Rule: []*router.RuleConfig{{RuleTag: "rule", OutboundTag: "direct"}}, BalancingRule: []*router.BalancerConfig{{Tag: "balance", OutboundSelector: []string{"proxy"}, Strategy: "leastLoad", FallbackTag: "direct", StrategySettings: &router.LeastLoadConfig{Costs: []*router.Weight{{Regexp: true, Match: "proxy.*", Value: 2}}, Baselines: []int64{100, 200}, Expected: 2, MaxRTT: 300, Tolerance: 0.5}}}}
	if diff := cmp.Diff(want, target.config, protocmp.Transform()); diff != "" {
		t.Fatal(diff)
	}
	if !target.append {
		t.Fatal("append flag lost")
	}
}

func TestDecodeRuntimeConfigRejectsInvalidSettings(t *testing.T) {
	for name, message := range map[string]*serial.TypedMessage{
		"missing":          nil,
		"wrong config":     serial.ToTypedMessage(&router.RoutingRule{}),
		"missing settings": serial.ToTypedMessage(&router.Config{BalancingRule: []*router.BalancingRule{{Strategy: "leastload"}}}),
		"wrong settings":   serial.ToTypedMessage(&router.Config{BalancingRule: []*router.BalancingRule{{Strategy: "leastload", StrategySettings: serial.ToTypedMessage(&router.RoutingRule{})}}}),
		"malformed":        {Type: "xray.app.router.Config", Value: []byte{0xff}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeRuntimeConfig(message); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestDecodeRoutingRuleFields(t *testing.T) {
	domain := []*geodata.DomainRule{{Value: &geodata.DomainRule_Custom{Custom: &geodata.Domain{Type: geodata.Domain_Domain, Value: "example.com"}}}}
	ips := []*geodata.IPRule{{Value: &geodata.IPRule_Custom{Custom: &geodata.CIDRRule{Cidr: &geodata.CIDR{Ip: []byte{10, 0, 0, 0}, Prefix: 8}}}}}
	ports := &net.PortList{Range: []*net.PortRange{{From: 80, To: 443}}}
	wire := &router.RoutingRule{
		TargetTag: &router.RoutingRule_BalancingTag{BalancingTag: "balance"}, RuleTag: "all-fields",
		Domain: domain, Ip: ips, PortList: ports, Networks: []net.Network{net.Network_TCP},
		SourceIp: ips, SourcePortList: ports, LocalIp: ips, LocalPortList: ports, VlessRouteList: ports,
		UserEmail: []string{"user"}, InboundTag: []string{"in"}, Protocol: []string{"http"},
		Attributes: map[string]string{"key": "pattern"}, Process: []string{"curl"}, LocalOs: []string{"linux"},
		Webhook: &router.WebhookConfig{Url: "https://example.com/hook", Deduplication: 10, Headers: map[string]string{"X-Test": "value"}},
	}
	got, err := decodeRuntimeConfig(serial.ToTypedMessage(&router.Config{Rule: []*router.RoutingRule{wire}}))
	if err != nil {
		t.Fatal(err)
	}
	want := &router.RuleConfig{
		BalancerTag: "balance", RuleTag: "all-fields", Domain: domain, Ip: ips, PortList: ports, Networks: []net.Network{net.Network_TCP},
		SourceIp: ips, SourcePortList: ports, LocalIp: ips, LocalPortList: ports, VlessRouteList: ports,
		UserEmail: []string{"user"}, InboundTag: []string{"in"}, Protocol: []string{"http"},
		Attributes: map[string]string{"key": "pattern"}, Process: []string{"curl"}, LocalOs: []string{"linux"},
		Webhook: &router.WebhookSettings{Url: "https://example.com/hook", Deduplication: 10, Headers: map[string]string{"X-Test": "value"}},
	}
	if diff := cmp.Diff([]*router.RuleConfig{want}, got.Rule, protocmp.Transform()); diff != "" {
		t.Fatal(diff)
	}
}

func TestAddRuleUpdatesNativeRouter(t *testing.T) {
	ctx := context.Background()
	r := new(router.Router)
	if err := r.Init(ctx, &router.RuntimeConfig{}, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	server := NewRoutingServer(r, nil)
	for _, appendRules := range []bool{false, true, false} {
		outbound := "first"
		if appendRules {
			outbound = "second"
		}
		config := &router.Config{Rule: []*router.RoutingRule{{InboundTag: []string{outbound}, TargetTag: &router.RoutingRule_Tag{Tag: outbound}}}}
		if _, err := server.AddRule(ctx, &AddRuleRequest{Config: serial.ToTypedMessage(config), ShouldAppend: appendRules}); err != nil {
			t.Fatal(err)
		}
		route, err := server.TestRoute(ctx, &TestRouteRequest{RoutingContext: &RoutingContext{InboundTag: outbound}})
		if err != nil {
			t.Fatal(err)
		}
		if route.OutboundTag != outbound {
			t.Fatalf("unexpected outbound %q", route.OutboundTag)
		}
		_, err = server.TestRoute(ctx, &TestRouteRequest{RoutingContext: &RoutingContext{InboundTag: "second"}})
		if (err == nil) != appendRules {
			t.Fatalf("append=%v: second route error=%v", appendRules, err)
		}
	}
}
