package json

import (
	"io"

	"github.com/nickshiro/veil/core"
	"github.com/nickshiro/veil/infra/conf/serial"
)

func init() {
	core.ConfigLoader = func(input io.Reader) (*core.Config, error) {
		return serial.LoadJSONConfig(input)
	}
}
