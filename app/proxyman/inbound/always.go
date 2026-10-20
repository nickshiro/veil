package inbound

import (
	"context"
	"slices"

	"github.com/nickshiro/veil/app/proxyman"
	"github.com/nickshiro/veil/common"
	"github.com/nickshiro/veil/common/errors"
	"github.com/nickshiro/veil/common/mux"
	"github.com/nickshiro/veil/common/net"
	"github.com/nickshiro/veil/common/session"
	"github.com/nickshiro/veil/core"
	"github.com/nickshiro/veil/features/policy"
	"github.com/nickshiro/veil/features/stats"
	"github.com/nickshiro/veil/proxy"
	"github.com/nickshiro/veil/transport/internet"
)

func getStatCounter(v *core.Instance, tag string) (stats.Counter, stats.Counter) {
	var uplinkCounter stats.Counter
	var downlinkCounter stats.Counter

	policy := v.GetFeature(policy.ManagerType()).(policy.Manager)
	if len(tag) > 0 && policy.ForSystem().Stats.InboundUplink {
		statsManager := v.GetFeature(stats.ManagerType()).(stats.Manager)
		name := "inbound>>>" + tag + ">>>traffic>>>uplink"
		c, _ := statsManager.GetOrRegisterCounter(name)
		if c != nil {
			uplinkCounter = c
		}
	}
	if len(tag) > 0 && policy.ForSystem().Stats.InboundDownlink {
		statsManager := v.GetFeature(stats.ManagerType()).(stats.Manager)
		name := "inbound>>>" + tag + ">>>traffic>>>downlink"
		c, _ := statsManager.GetOrRegisterCounter(name)
		if c != nil {
			downlinkCounter = c
		}
	}

	return uplinkCounter, downlinkCounter
}

type AlwaysOnInboundHandler struct {
	proxyConfig    any
	receiverConfig *proxyman.ReceiverConfig
	proxy          proxy.Inbound
	workers        []worker
	mux            *mux.Server
	tag            string
}

func NewAlwaysOnInboundHandler(ctx context.Context, tag string, receiverConfig *proxyman.ReceiverConfig, proxyConfig any) (*AlwaysOnInboundHandler, error) {
	sniffingRequest, err := proxyman.BuildSniffingRequest(receiverConfig.SniffingSettings)
	if err != nil {
		return nil, err
	}
	src := net.TCPDestination(net.AnyIP, 0)
	if receiverConfig.Listen != nil {
		src.Address = receiverConfig.Listen.AsAddress()
	}
	if receiverConfig.PortList != nil && len(receiverConfig.PortList.Range) > 0 {
		src.Port = net.Port(receiverConfig.PortList.Range[0].From)
	}
	mss, err := internet.ToMemoryStreamConfig(receiverConfig.StreamSettings)
	if err != nil {
		return nil, errors.New("failed to parse stream config").Base(err)
	}

	newCtx := session.ContextWithInbound(ctx, &session.Inbound{Tag: tag, Source: src})
	newCtx = session.ContextWithContent(newCtx, &session.Content{SniffingRequest: sniffingRequest})
	newCtx = session.ContextWithStreamSettings(newCtx, mss)

	rawProxy, err := common.CreateObject(newCtx, proxyConfig)
	if err != nil {
		return nil, err
	}
	p, ok := rawProxy.(proxy.Inbound)
	if !ok {
		return nil, errors.New("not an inbound proxy.")
	}

	h := &AlwaysOnInboundHandler{
		receiverConfig: receiverConfig,
		proxyConfig:    proxyConfig,
		proxy:          p,
		mux:            mux.NewServer(ctx),
		tag:            tag,
	}

	uplinkCounter, downlinkCounter := getStatCounter(core.MustFromContext(ctx), tag)

	nl := p.Network()
	pl := receiverConfig.PortList
	address := receiverConfig.Listen.AsAddress()
	if address == nil {
		address = net.AnyIP
	}

	if receiverConfig.ReceiveOriginalDestination {
		if mss.SocketSettings == nil {
			mss.SocketSettings = &internet.SocketConfig{}
		}
		if mss.SocketSettings.Tproxy == internet.SocketConfig_Off {
			mss.SocketSettings.Tproxy = internet.SocketConfig_Redirect
		}
		mss.SocketSettings.ReceiveOriginalDestAddress = true
	}
	if pl == nil {
		if slices.Contains(nl, net.Network_UNIX) {
			errors.LogDebug(ctx, "creating unix domain socket worker on ", address)

			worker := &dsWorker{
				address:         address,
				proxy:           p,
				stream:          mss,
				tag:             tag,
				dispatcher:      h.mux,
				sniffingRequest: sniffingRequest,
				uplinkCounter:   uplinkCounter,
				downlinkCounter: downlinkCounter,
				ctx:             ctx,
			}
			h.workers = append(h.workers, worker)
		}
	}
	if pl != nil {
		for _, pr := range pl.Range {
			for port := pr.From; port <= pr.To; port++ {
				if slices.Contains(nl, net.Network_TCP) {
					errors.LogDebug(ctx, "creating stream worker on ", address, ":", port)

					worker := &tcpWorker{
						address:         address,
						port:            net.Port(port),
						proxy:           p,
						stream:          mss,
						recvOrigDest:    receiverConfig.ReceiveOriginalDestination,
						tag:             tag,
						dispatcher:      h.mux,
						sniffingRequest: sniffingRequest,
						uplinkCounter:   uplinkCounter,
						downlinkCounter: downlinkCounter,
						ctx:             ctx,
					}
					h.workers = append(h.workers, worker)
				}

				if slices.Contains(nl, net.Network_UDP) {
					worker := &udpWorker{
						tag:             tag,
						proxy:           p,
						address:         address,
						port:            net.Port(port),
						dispatcher:      h.mux,
						sniffingRequest: sniffingRequest,
						uplinkCounter:   uplinkCounter,
						downlinkCounter: downlinkCounter,
						stream:          mss,
						ctx:             ctx,
					}
					h.workers = append(h.workers, worker)
				}
			}
		}
	}

	return h, nil
}

// Start implements common.Runnable.
func (h *AlwaysOnInboundHandler) Start() error {
	// for inbound without worker (TUN)
	if run, ok := h.proxy.(common.Runnable); ok {
		if err := run.Start(); err != nil {
			return errors.New("failed to start proxy").Base(err)
		}
	}
	for _, worker := range h.workers {
		if err := worker.Start(); err != nil {
			return err
		}
	}
	return nil
}

// Close implements io.Closer.
func (h *AlwaysOnInboundHandler) Close() error {
	var errs []error
	for _, worker := range h.workers {
		errs = append(errs, worker.Close())
	}
	errs = append(errs, h.mux.Close())
	errs = append(errs, common.Close(h.proxy))
	if err := errors.Combine(errs...); err != nil {
		return errors.New("failed to close all resources").Base(err)
	}
	return nil
}

func (h *AlwaysOnInboundHandler) Tag() string {
	return h.tag
}

// ReceiverSettings implements inbound.Handler.
func (h *AlwaysOnInboundHandler) ReceiverSettings() any {
	return h.receiverConfig
}

// ProxySettings implements inbound.Handler.
func (h *AlwaysOnInboundHandler) ProxySettings() any {
	return h.proxyConfig
}
