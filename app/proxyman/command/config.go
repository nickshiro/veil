package command

import (
	"fmt"

	"github.com/nickshiro/veil/common/serial"
	"github.com/nickshiro/veil/core"
	"google.golang.org/protobuf/proto"
)

func decodeSettings(settings *serial.TypedMessage) (any, error) {
	if settings == nil {
		return nil, nil
	}
	return settings.GetInstance()
}

func decodeInbound(config *core.InboundHandlerConfig) (*core.InboundConfig, error) {
	if config == nil {
		return nil, fmt.Errorf("missing inbound config")
	}
	receiver, err := decodeSettings(config.ReceiverSettings)
	if err != nil {
		return nil, fmt.Errorf("decode receiver settings: %w", err)
	}
	proxy, err := decodeSettings(config.ProxySettings)
	if err != nil {
		return nil, fmt.Errorf("decode inbound proxy settings: %w", err)
	}
	return &core.InboundConfig{Tag: config.Tag, ReceiverSettings: receiver, ProxySettings: proxy}, nil
}

func decodeOutbound(config *core.OutboundHandlerConfig) (*core.OutboundConfig, error) {
	if config == nil {
		return nil, fmt.Errorf("missing outbound config")
	}
	sender, err := decodeSettings(config.SenderSettings)
	if err != nil {
		return nil, fmt.Errorf("decode sender settings: %w", err)
	}
	proxy, err := decodeSettings(config.ProxySettings)
	if err != nil {
		return nil, fmt.Errorf("decode outbound proxy settings: %w", err)
	}
	return &core.OutboundConfig{Tag: config.Tag, SenderSettings: sender, ProxySettings: proxy}, nil
}

func encodeSettings(settings any) (*serial.TypedMessage, error) {
	if settings == nil {
		return nil, nil
	}
	message, ok := settings.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("settings %T have no protobuf API representation", settings)
	}
	value, err := proto.Marshal(message)
	if err != nil {
		return nil, err
	}
	return &serial.TypedMessage{Type: serial.GetMessageType(message), Value: value}, nil
}
