package command

import (
	"context"
	"testing"

	"github.com/nickshiro/veil/app/router"
	"github.com/nickshiro/veil/common/serial"
)

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
