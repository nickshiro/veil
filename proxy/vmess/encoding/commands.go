package encoding

import (
	"io"

	"github.com/nickshiro/veil/common/errors"
	"github.com/nickshiro/veil/common/protocol"
)

var (
	ErrCommandTooLarge     = errors.New("Command too large.")
	ErrCommandTypeMismatch = errors.New("Command type mismatch.")
	ErrInvalidAuth         = errors.New("Invalid auth.")
	ErrInsufficientLength  = errors.New("Insufficient length.")
	ErrUnknownCommand      = errors.New("Unknown command.")
)

func MarshalCommand(command any, writer io.Writer) error {
	return ErrUnknownCommand
}

func UnmarshalCommand(cmdID byte, data []byte) (protocol.ResponseCommand, error) {
	return nil, ErrUnknownCommand
}

type CommandFactory interface {
	Marshal(command any, writer io.Writer) error
	Unmarshal(data []byte) (any, error)
}
