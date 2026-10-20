package conf_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	apppolicy "github.com/nickshiro/veil/app/policy"
	"github.com/nickshiro/veil/common"
	featurepolicy "github.com/nickshiro/veil/features/policy"
	. "github.com/nickshiro/veil/infra/conf"
)

func TestPolicyTimeoutPresence(t *testing.T) {
	defaults := featurepolicy.SessionDefault().Timeouts
	tests := []struct {
		name string
		json string
		want featurepolicy.Timeout
	}{
		{"absent", `{"levels":{"0":{}}}`, defaults},
		{"null", `{"levels":{"0":{"handshake":null,"connIdle":null,"uplinkOnly":null,"downlinkOnly":null}}}`, defaults},
		{"zero", `{"levels":{"0":{"handshake":0,"connIdle":0,"uplinkOnly":0,"downlinkOnly":0}}}`, featurepolicy.Timeout{}},
		{"values", `{"levels":{"0":{"handshake":1,"connIdle":2,"uplinkOnly":3,"downlinkOnly":4}}}`, featurepolicy.Timeout{
			Handshake: time.Second, ConnectionIdle: 2 * time.Second,
			UplinkOnly: 3 * time.Second, DownlinkOnly: 4 * time.Second,
		}},
		{"partial", `{"levels":{"0":{"handshake":0}}}`, featurepolicy.Timeout{
			ConnectionIdle: defaults.ConnectionIdle,
			UplinkOnly:     defaults.UplinkOnly, DownlinkOnly: defaults.DownlinkOnly,
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var input PolicyConfig
			if err := json.Unmarshal([]byte(tt.json), &input); err != nil {
				t.Fatal(err)
			}
			config, err := input.Build()
			if err != nil {
				t.Fatal(err)
			}
			manager, err := apppolicy.New(context.Background(), config)
			if err != nil {
				t.Fatal(err)
			}
			if got := manager.ForLevel(0).Timeouts; got != tt.want {
				t.Fatalf("timeouts = %+v, want %+v", got, tt.want)
			}
			if got := manager.ForLevel(1).Timeouts; got != defaults {
				t.Fatalf("unconfigured level = %+v, want defaults %+v", got, defaults)
			}
		})
	}
}

func TestBufferSize(t *testing.T) {
	cases := []struct {
		Input  int32
		Output int32
	}{
		{
			Input:  0,
			Output: 0,
		},
		{
			Input:  -1,
			Output: -1,
		},
		{
			Input:  1,
			Output: 1024,
		},
	}

	for _, c := range cases {
		bs := c.Input
		pConf := Policy{
			BufferSize: &bs,
		}
		p, err := pConf.Build()
		common.Must(err)
		if p.Buffer.Connection != c.Output {
			t.Error("expected buffer size ", c.Output, " but got ", p.Buffer.Connection)
		}
	}
}
