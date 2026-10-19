//go:build linux

package tcp_test

import (
	"context"
	"strings"
	"testing"

	"github.com/nickshiro/veil/common"
	"github.com/nickshiro/veil/testing/servers/tcp"
	"github.com/nickshiro/veil/transport/internet"
	. "github.com/nickshiro/veil/transport/internet/tcp"
)

func TestGetOriginalDestination(t *testing.T) {
	tcpServer := tcp.Server{}
	dest, err := tcpServer.Start()
	common.Must(err)
	defer tcpServer.Close()

	config, err := internet.ToMemoryStreamConfig(nil)
	common.Must(err)
	conn, err := Dial(context.Background(), dest, config)
	common.Must(err)
	defer conn.Close()

	originalDest, err := GetOriginalDestination(conn)
	if !(dest == originalDest || strings.Contains(err.Error(), "failed to call getsockopt")) {
		t.Error("unexpected state")
	}
}
