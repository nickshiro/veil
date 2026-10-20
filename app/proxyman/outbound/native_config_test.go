package outbound_test

import (
	"context"
	"testing"

	"github.com/nickshiro/veil/app/proxyman"
	"github.com/nickshiro/veil/app/proxyman/outbound"
	"github.com/nickshiro/veil/common"
	"github.com/nickshiro/veil/core"
	"github.com/nickshiro/veil/transport"
	"github.com/nickshiro/veil/transport/internet"
	_ "github.com/nickshiro/veil/transport/internet/tcp"
)

type nativeProxyConfig struct {
	Value string
}

type nativeProxy struct{}

func (*nativeProxy) Process(context.Context, *transport.Link, internet.Dialer) error {
	return nil
}

func init() {
	common.Must(common.RegisterConfig((*nativeProxyConfig)(nil), func(_ context.Context, config any) (any, error) {
		_ = config.(*nativeProxyConfig)
		return &nativeProxy{}, nil
	}))
}

func TestNativeProxyConfig(t *testing.T) {
	server, err := core.New(&core.Config{})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	ctx := context.WithValue(context.Background(), xrayKey, server)
	settings := &nativeProxyConfig{Value: "native"}
	sender := &proxyman.SenderConfig{
		MultiplexSettings: &proxyman.MultiplexingConfig{Enabled: true},
	}
	handler, err := outbound.NewHandler(ctx, &core.OutboundConfig{
		Tag: "native", SenderSettings: sender, ProxySettings: settings,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer handler.Close()
	if handler.ProxySettings() != settings {
		t.Fatal("native settings were not passed directly")
	}
	if sender.MultiplexSettings.Concurrency != 0 {
		t.Fatal("constructor modified caller's mux settings")
	}
	if got := handler.SenderSettings().(*proxyman.SenderConfig).MultiplexSettings.Concurrency; got != 8 {
		t.Fatalf("mux concurrency = %d, want 8", got)
	}
}
