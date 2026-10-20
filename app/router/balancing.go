package router

import (
	"context"
	sync "sync"

	"github.com/nickshiro/veil/app/observatory"
	"github.com/nickshiro/veil/common"
	"github.com/nickshiro/veil/common/errors"
	"github.com/nickshiro/veil/core"
	"github.com/nickshiro/veil/features/extension"
	"github.com/nickshiro/veil/features/outbound"
)

type BalancingStrategy interface {
	PickOutbound([]string) string
}

type RoundRobinStrategy struct {
	FallbackTag string

	ctx         context.Context
	observatory extension.Observatory
	mu          sync.Mutex
	index       int
}

func (s *RoundRobinStrategy) InjectContext(ctx context.Context) {
	s.ctx = ctx
	if len(s.FallbackTag) > 0 {
		common.Must(core.RequireFeatures(s.ctx, func(observatory extension.Observatory) error {
			s.observatory = observatory
			return nil
		}))
	}
}

func (s *RoundRobinStrategy) PickOutbound(tags []string) string {
	if s.observatory != nil {
		observeReport, err := s.observatory.GetObservation(s.ctx)
		if err == nil {
			aliveTags := make([]string, 0)
			if result, ok := observeReport.(*observatory.ObservationResult); ok {
				status := result.Status
				statusMap := make(map[string]*observatory.OutboundStatus)
				for _, outboundStatus := range status {
					statusMap[outboundStatus.OutboundTag] = outboundStatus
				}
				for _, candidate := range tags {
					if outboundStatus, found := statusMap[candidate]; found {
						if outboundStatus.Alive {
							aliveTags = append(aliveTags, candidate)
						}
					} else {
						// unfound candidate is considered alive
						aliveTags = append(aliveTags, candidate)
					}
				}
				tags = aliveTags
			}
		}
	}

	n := len(tags)
	if n == 0 {
		// goes to fallbackTag
		return ""
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	tag := tags[s.index%n]
	s.index = (s.index + 1) % n
	return tag
}

type Balancer struct {
	selectors   []string
	strategy    BalancingStrategy
	ohm         outbound.Manager
	fallbackTag string
}

// PickOutbound picks the tag of a outbound
func (b *Balancer) PickOutbound() (string, error) {
	candidates, err := b.SelectOutbounds()
	if err != nil {
		if b.fallbackTag != "" {
			errors.LogInfo(context.Background(), "fallback to [", b.fallbackTag, "], due to error: ", err)
			return b.fallbackTag, nil
		}
		return "", err
	}
	tag := b.strategy.PickOutbound(candidates)
	if tag == "" {
		if b.fallbackTag != "" {
			errors.LogInfo(context.Background(), "fallback to [", b.fallbackTag, "], due to empty tag returned")
			return b.fallbackTag, nil
		}
		// will use default handler
		return "", errors.New("balancing strategy returns empty tag")
	}
	return tag, nil
}

func (b *Balancer) InjectContext(ctx context.Context) {
	if contextReceiver, ok := b.strategy.(extension.ContextReceiver); ok {
		contextReceiver.InjectContext(ctx)
	}
}

// SelectOutbounds select outbounds with selectors of the Balancer
func (b *Balancer) SelectOutbounds() ([]string, error) {
	hs, ok := b.ohm.(outbound.HandlerSelector)
	if !ok {
		return nil, errors.New("outbound.Manager is not a HandlerSelector")
	}
	tags := hs.Select(b.selectors)
	return tags, nil
}
