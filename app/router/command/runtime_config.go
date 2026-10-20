package command

import (
	"fmt"
	"slices"
	"strings"

	"github.com/nickshiro/veil/app/router"
	"github.com/nickshiro/veil/common/serial"
)

func decodeRuntimeConfig(message *serial.TypedMessage) (*router.RuntimeConfig, error) {
	if message == nil {
		return nil, fmt.Errorf("missing routing config")
	}
	value, err := message.GetInstance()
	if err != nil {
		return nil, err
	}
	config, ok := value.(*router.Config)
	if !ok {
		return nil, fmt.Errorf("unexpected routing config %T", value)
	}
	result := &router.RuntimeConfig{DomainStrategy: router.DomainStrategy(config.DomainStrategy)}
	for _, rule := range config.Rule {
		if rule == nil {
			return nil, fmt.Errorf("missing routing rule")
		}
		native := &router.RuleConfig{
			OutboundTag: rule.GetTag(), BalancerTag: rule.GetBalancingTag(), RuleTag: rule.RuleTag,
			Domain: rule.Domain, Ip: rule.Ip, PortList: rule.PortList, Networks: rule.Networks,
			SourceIp: rule.SourceIp, SourcePortList: rule.SourcePortList, UserEmail: rule.UserEmail,
			InboundTag: rule.InboundTag, Protocol: rule.Protocol, Attributes: rule.Attributes,
			LocalIp: rule.LocalIp, LocalPortList: rule.LocalPortList, VlessRouteList: rule.VlessRouteList,
			Process: rule.Process, LocalOs: rule.LocalOs,
		}
		if w := rule.Webhook; w != nil {
			native.Webhook = &router.WebhookSettings{Url: w.Url, Deduplication: w.Deduplication, Headers: w.Headers}
		}
		result.Rule = append(result.Rule, native)
	}
	for _, rule := range config.BalancingRule {
		if rule == nil {
			return nil, fmt.Errorf("missing balancer config")
		}
		balancer := &router.BalancerConfig{
			Tag: rule.Tag, OutboundSelector: rule.OutboundSelector,
			Strategy: rule.Strategy, FallbackTag: rule.FallbackTag,
		}
		if strings.EqualFold(rule.Strategy, "leastload") {
			if rule.StrategySettings == nil {
				return nil, fmt.Errorf("missing leastload settings")
			}
			value, err := rule.StrategySettings.GetInstance()
			if err != nil {
				return nil, err
			}
			settings, ok := value.(*router.StrategyLeastLoadConfig)
			if !ok {
				return nil, fmt.Errorf("unexpected leastload settings %T", value)
			}
			native := &router.LeastLoadConfig{
				Baselines: slices.Clone(settings.Baselines),
				Expected:  settings.Expected, MaxRTT: settings.MaxRTT, Tolerance: settings.Tolerance,
			}
			for _, cost := range settings.Costs {
				if cost == nil {
					return nil, fmt.Errorf("missing strategy weight")
				}
				native.Costs = append(native.Costs, &router.Weight{Regexp: cost.Regexp, Match: cost.Match, Value: cost.Value})
			}
			balancer.StrategySettings = native
		}
		result.BalancingRule = append(result.BalancingRule, balancer)
	}
	return result, nil
}
