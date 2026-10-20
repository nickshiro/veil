package json

import (
	"context"
	"io"

	"github.com/nickshiro/veil/common/cmdarg"
	"github.com/nickshiro/veil/common/errors"
	"github.com/nickshiro/veil/core"
	"github.com/nickshiro/veil/infra/conf"
	"github.com/nickshiro/veil/infra/conf/serial"
	"github.com/nickshiro/veil/main/confloader"
)

func init() {
	core.ConfigLoader = func(input any) (*core.Config, error) {
		switch v := input.(type) {
		case cmdarg.Arg:
			cf := &conf.Config{}
			for i, arg := range v {
				errors.LogInfo(context.Background(), "Reading config: ", arg)
				r, err := confloader.LoadConfig(arg)
				if err != nil {
					return nil, errors.New("failed to read config: ", arg).Base(err)
				}
				c, err := serial.DecodeJSONConfig(r)
				if err != nil {
					return nil, errors.New("failed to decode config: ", arg).Base(err)
				}
				if i == 0 {
					// This ensure even if the muti-json parser do not support a setting,
					// It is still respected automatically for the first configure file
					*cf = *c
					continue
				}
				cf.Override(c, arg)
			}
			return cf.Build()
		case io.Reader:
			return serial.LoadJSONConfig(v)
		default:
			return nil, errors.New("unknown type")
		}
	}
}
