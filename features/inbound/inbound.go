package inbound

import (
	"context"

	"github.com/nickshiro/veil/common"
	"github.com/nickshiro/veil/features"
)

// Handler is the interface for handlers that process inbound connections.
type Handler interface {
	common.Runnable
	// The tag of this handler.
	Tag() string
	// Returns the active receiver settings. The returned value is read-only.
	ReceiverSettings() any
	// Returns the active proxy settings. The returned value is read-only.
	ProxySettings() any
}

// Manager is a feature that manages InboundHandlers.
type Manager interface {
	features.Feature
	// GetHandler returns an InboundHandler for the given tag.
	GetHandler(ctx context.Context, tag string) (Handler, error)
	// AddHandler adds the given handler into this Manager.
	AddHandler(ctx context.Context, handler Handler) error

	// RemoveHandler removes a handler from Manager.
	RemoveHandler(ctx context.Context, tag string) error

	// ListHandlers returns a list of inbound.Handler.
	ListHandlers(ctx context.Context) []Handler
}

// ManagerType returns the type of Manager interface. Can be used for implementing common.HasType.
func ManagerType() any {
	return (*Manager)(nil)
}
