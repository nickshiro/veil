package tagged

import (
	"context"

	"github.com/nickshiro/veil/common/net"
	"github.com/nickshiro/veil/features/routing"
)

type DialFunc func(ctx context.Context, dispatcher routing.Dispatcher, dest net.Destination, tag string) (net.Conn, error)

var Dialer DialFunc
