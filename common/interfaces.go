package common

import (
	"io"

	"github.com/nickshiro/veil/common/errors"
)

// Interruptible is an interface for objects that can be stopped before its completion.
//
// xray:api:beta
type Interruptible interface {
	Interrupt()
}

// Close closes obj if it implements io.Closer.
//
// xray:api:beta
func Close(obj any) error {
	if c, ok := obj.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

// Interrupt calls Interrupt() if object implements Interruptible interface, or Close() if the object implements io.Closer interface.
//
// xray:api:beta
func Interrupt(obj any) error {
	if c, ok := obj.(Interruptible); ok {
		c.Interrupt()
		return nil
	}
	return Close(obj)
}

// Runnable is the interface for objects that can start to work and stop on demand.
type Runnable interface {
	// Start starts the runnable object. Upon the method returning nil, the object begins to function properly.
	Start() error

	io.Closer
}

// HasType is the interface for objects that knows its type.
type HasType interface {
	// Type returns the type of the object.
	// Usually it returns (*Type)(nil) of the object.
	Type() any
}

// ChainedClosable closes a sequence of io.Closer objects.
type ChainedClosable []io.Closer

// Close implements io.Closer.
func (cc ChainedClosable) Close() error {
	var errs []error
	for _, c := range cc {
		if err := c.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Combine(errs...)
}
