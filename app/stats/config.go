package stats

// Config enables the statistics manager.
type Config struct{}

type ChannelConfig struct {
	Blocking        bool
	SubscriberLimit int32
	BufferSize      int32
}
