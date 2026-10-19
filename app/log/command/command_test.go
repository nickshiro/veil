package command_test

import (
	"context"
	"testing"

	"github.com/nickshiro/veil/app/dispatcher"
	"github.com/nickshiro/veil/app/log"
	. "github.com/nickshiro/veil/app/log/command"
	"github.com/nickshiro/veil/app/proxyman"
	_ "github.com/nickshiro/veil/app/proxyman/inbound"
	_ "github.com/nickshiro/veil/app/proxyman/outbound"
	"github.com/nickshiro/veil/common"
	"github.com/nickshiro/veil/common/serial"
	"github.com/nickshiro/veil/core"
)

func TestLoggerRestart(t *testing.T) {
	v, err := core.New(&core.Config{
		App: []*serial.TypedMessage{
			serial.ToTypedMessage(&log.Config{}),
			serial.ToTypedMessage(&dispatcher.Config{}),
			serial.ToTypedMessage(&proxyman.InboundConfig{}),
			serial.ToTypedMessage(&proxyman.OutboundConfig{}),
		},
	})
	common.Must(err)
	common.Must(v.Start())

	server := &LoggerServer{
		V: v,
	}
	common.Must2(server.RestartLogger(context.Background(), &RestartLoggerRequest{}))
}
