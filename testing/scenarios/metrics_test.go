package scenarios

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/nickshiro/veil/app/metrics"
	"github.com/nickshiro/veil/app/proxyman"
	"github.com/nickshiro/veil/app/router"
	"github.com/nickshiro/veil/common"
	"github.com/nickshiro/veil/common/net"
	"github.com/nickshiro/veil/core"
	"github.com/nickshiro/veil/proxy/dokodemo"
	"github.com/nickshiro/veil/proxy/freedom"
	"github.com/nickshiro/veil/testing/servers/tcp"
)

const expectedMessage = "goroutine profile: total"

func TestMetrics(t *testing.T) {
	tcpServer := tcp.Server{
		MsgProcessor: xor,
	}
	dest, err := tcpServer.Start()
	common.Must(err)
	defer tcpServer.Close()

	metricsPort := tcp.PickPort()
	clientConfig := &core.Config{
		App: []any{
			&metrics.Config{
				Tag: "metrics_out",
			},
			&router.Config{
				Rule: []*router.RoutingRule{
					{
						InboundTag: []string{"metrics_in"},
						TargetTag: &router.RoutingRule_Tag{
							Tag: "metrics_out",
						},
					},
				},
			},
		},
		Inbound: []*core.InboundConfig{
			{
				Tag: "metrics_in",
				ReceiverSettings: &proxyman.ReceiverConfig{
					PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(metricsPort)}},
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

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/debug/pprof/goroutine?debug=1", metricsPort))
	common.Must(err)
	if resp == nil {
		t.Error("unexpected pprof nil response")
	}
	if resp.StatusCode != http.StatusOK {
		t.Error("unexpected pprof status code")
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body)[0:len(expectedMessage)] != expectedMessage {
		t.Error("unexpected response body from pprof handler")
	}

	resp2, err2 := http.Get(fmt.Sprintf("http://127.0.0.1:%d/debug/vars", metricsPort))
	common.Must(err2)
	if resp2 == nil {
		t.Error("unexpected expvars nil response")
	}
	if resp2.StatusCode != http.StatusOK {
		t.Error("unexpected expvars status code")
	}
	body2, err2 := io.ReadAll(resp2.Body)
	if err2 != nil {
		t.Fatal(err2)
	}
	var json2 map[string]any
	if json.Unmarshal(body2, &json2) != nil {
		t.Error("unexpected response body from expvars handler")
	}
}
