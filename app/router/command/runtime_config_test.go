package command

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/nickshiro/veil/app/router"
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
	want := &router.RuntimeConfig{DomainStrategy: wire.DomainStrategy, Rule: wire.Rule, BalancingRule: []*router.BalancerConfig{{Tag: "balance", OutboundSelector: []string{"proxy"}, Strategy: "leastLoad", FallbackTag: "direct", StrategySettings: &router.LeastLoadConfig{Costs: []*router.Weight{{Regexp: true, Match: "proxy.*", Value: 2}}, Baselines: []int64{100, 200}, Expected: 2, MaxRTT: 300, Tolerance: 0.5}}}}
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
		"malformed":        {Type: "veil.app.router.Config", Value: []byte{0xff}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeRuntimeConfig(message); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
