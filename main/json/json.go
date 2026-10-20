package json

import (
	"context"
	"io"

	"github.com/nickshiro/veil/common/errors"
	"github.com/nickshiro/veil/core"
	"github.com/nickshiro/veil/infra/conf/serial"
	"github.com/nickshiro/veil/main/confloader"
)

func init() {
	core.ConfigLoader = func(input any) (*core.Config, error) {
		switch v := input.(type) {
		case string:
			errors.LogInfo(context.Background(), "Reading config: ", v)
			r, err := confloader.LoadConfig(v)
			if err != nil {
				return nil, errors.New("failed to read config: ", v).Base(err)
			}
			c, err := serial.DecodeJSONConfig(r)
			if err != nil {
				return nil, errors.New("failed to decode config: ", v).Base(err)
			}
			return c.Build()
		case io.Reader:
			return serial.LoadJSONConfig(v)
		default:
			return nil, errors.New("unknown type")
		}
	}
}
