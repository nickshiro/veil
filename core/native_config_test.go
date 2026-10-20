package core_test

import (
	"context"
	"testing"

	"github.com/nickshiro/veil/common"
	"github.com/nickshiro/veil/core"
)

// A feature config need not implement proto.Message or have a registered descriptor.
type nativeFeatureConfig struct {
	Value string
}

type nativeFeature struct {
	value   string
	started bool
	closed  bool
}

func (*nativeFeature) Type() any      { return (*nativeFeature)(nil) }
func (f *nativeFeature) Start() error { f.started = true; return nil }
func (f *nativeFeature) Close() error { f.closed = true; return nil }

func init() {
	common.Must(common.RegisterConfig((*nativeFeatureConfig)(nil), func(_ context.Context, config any) (any, error) {
		return &nativeFeature{value: config.(*nativeFeatureConfig).Value}, nil
	}))
}

func TestNativeFeatureConfig(t *testing.T) {
	server, err := core.New(&core.Config{App: []any{&nativeFeatureConfig{Value: "native"}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { server.Close() })
	f, ok := server.GetFeature((*nativeFeature)(nil)).(*nativeFeature)
	if !ok || f.value != "native" {
		t.Fatal("native config was not passed to its feature constructor")
	}
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	if !f.started {
		t.Fatal("feature was not started")
	}
	if err := server.Close(); err != nil {
		t.Fatal(err)
	}
	if !f.closed {
		t.Fatal("feature was not closed")
	}
}
