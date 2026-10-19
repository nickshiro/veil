package serial

import (
	"context"
	"io"

	"github.com/nickshiro/veil/common/errors"
	creflect "github.com/nickshiro/veil/common/reflect"
	"github.com/nickshiro/veil/core"
	"github.com/nickshiro/veil/infra/conf"
	"github.com/nickshiro/veil/main/confloader"
)

func MergeConfigFromFiles(files []*core.ConfigSource) (string, error) {
	c, err := mergeConfigs(files)
	if err != nil {
		return "", err
	}

	if j, ok := creflect.MarshalToJson(c, true); ok {
		return j, nil
	}
	return "", errors.New("marshal to json failed.")
}

func mergeConfigs(files []*core.ConfigSource) (*conf.Config, error) {
	cf := &conf.Config{}
	for i, file := range files {
		errors.LogInfo(context.Background(), "Reading config: ", file)
		r, err := confloader.LoadConfig(file.Name)
		if err != nil {
			return nil, errors.New("failed to read config: ", file).Base(err)
		}
		decoder := ReaderDecoderByFormat[file.Format]
		c, err := decoder(r)
		if err != nil {
			return nil, errors.New("failed to decode config: ", file).Base(err)
		}
		if i == 0 {
			*cf = *c
			continue
		}
		cf.Override(c, file.Name)
	}
	return cf, nil
}

func BuildConfig(files []*core.ConfigSource) (*core.Config, error) {
	config, err := mergeConfigs(files)
	if err != nil {
		return nil, err
	}
	return config.Build()
}

type readerDecoder func(io.Reader) (*conf.Config, error)

var ReaderDecoderByFormat = make(map[string]readerDecoder)

func init() {
	ReaderDecoderByFormat["json"] = DecodeJSONConfig

	core.ConfigBuilderForFiles = BuildConfig
	core.ConfigMergedFormFiles = MergeConfigFromFiles
}
