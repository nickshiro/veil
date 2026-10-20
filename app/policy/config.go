package policy

import (
	"time"

	"github.com/nickshiro/veil/features/policy"
)

// Second is an optional timeout value in seconds. A nil *Second leaves the
// default timeout unchanged; a non-nil value explicitly sets it, including zero.
type Second struct {
	Value uint32
}

type Policy_Timeout struct {
	Handshake      *Second
	ConnectionIdle *Second
	UplinkOnly     *Second
	DownlinkOnly   *Second
}

type Policy_Stats struct {
	UserUplink   bool
	UserDownlink bool
	UserOnline   bool
}

type Policy_Buffer struct {
	// Connection is the per-connection buffer size in bytes; -1 means unlimited.
	Connection int32
}

type Policy struct {
	Timeout *Policy_Timeout
	Stats   *Policy_Stats
	Buffer  *Policy_Buffer
}

type SystemPolicy_Stats struct {
	InboundUplink    bool
	InboundDownlink  bool
	OutboundUplink   bool
	OutboundDownlink bool
}

type SystemPolicy struct {
	Stats *SystemPolicy_Stats
}

type Config struct {
	Level  map[uint32]*Policy
	System *SystemPolicy
}

// Duration converts Second to time.Duration.
func (s *Second) Duration() time.Duration {
	if s == nil {
		return 0
	}
	return time.Second * time.Duration(s.Value)
}

func defaultPolicy() *Policy {
	p := policy.SessionDefault()

	return &Policy{
		Timeout: &Policy_Timeout{
			Handshake:      &Second{Value: uint32(p.Timeouts.Handshake / time.Second)},
			ConnectionIdle: &Second{Value: uint32(p.Timeouts.ConnectionIdle / time.Second)},
			UplinkOnly:     &Second{Value: uint32(p.Timeouts.UplinkOnly / time.Second)},
			DownlinkOnly:   &Second{Value: uint32(p.Timeouts.DownlinkOnly / time.Second)},
		},
		Buffer: &Policy_Buffer{
			Connection: p.Buffer.PerConnection,
		},
	}
}

func (p *Policy_Timeout) overrideWith(another *Policy_Timeout) {
	if another.Handshake != nil {
		p.Handshake = &Second{Value: another.Handshake.Value}
	}
	if another.ConnectionIdle != nil {
		p.ConnectionIdle = &Second{Value: another.ConnectionIdle.Value}
	}
	if another.UplinkOnly != nil {
		p.UplinkOnly = &Second{Value: another.UplinkOnly.Value}
	}
	if another.DownlinkOnly != nil {
		p.DownlinkOnly = &Second{Value: another.DownlinkOnly.Value}
	}
}

func (p *Policy) overrideWith(another *Policy) {
	if another.Timeout != nil {
		p.Timeout.overrideWith(another.Timeout)
	}
	if another.Stats != nil && p.Stats == nil {
		p.Stats = &Policy_Stats{}
		p.Stats = another.Stats
	}
	if another.Buffer != nil {
		p.Buffer = &Policy_Buffer{
			Connection: another.Buffer.Connection,
		}
	}
}

// ToCorePolicy converts this Policy to policy.Session.
func (p *Policy) ToCorePolicy() policy.Session {
	cp := policy.SessionDefault()

	if p.Timeout != nil {
		cp.Timeouts.ConnectionIdle = p.Timeout.ConnectionIdle.Duration()
		cp.Timeouts.Handshake = p.Timeout.Handshake.Duration()
		cp.Timeouts.DownlinkOnly = p.Timeout.DownlinkOnly.Duration()
		cp.Timeouts.UplinkOnly = p.Timeout.UplinkOnly.Duration()
	}
	if p.Stats != nil {
		cp.Stats.UserUplink = p.Stats.UserUplink
		cp.Stats.UserDownlink = p.Stats.UserDownlink
		cp.Stats.UserOnline = p.Stats.UserOnline
	}
	if p.Buffer != nil {
		cp.Buffer.PerConnection = p.Buffer.Connection
	}
	return cp
}

// ToCorePolicy converts this SystemPolicy to policy.System.
func (p *SystemPolicy) ToCorePolicy() policy.System {
	return policy.System{
		Stats: policy.SystemStats{
			InboundUplink:    p.Stats.InboundUplink,
			InboundDownlink:  p.Stats.InboundDownlink,
			OutboundUplink:   p.Stats.OutboundUplink,
			OutboundDownlink: p.Stats.OutboundDownlink,
		},
	}
}
