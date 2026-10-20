package serial

import (
	"context"
	"encoding/json"

	"github.com/nickshiro/veil/common/errors"
	creflect "github.com/nickshiro/veil/common/reflect"
	"github.com/nickshiro/veil/infra/conf"
	"github.com/nickshiro/veil/main/confloader"
)

// DumpConfig returns a single configuration as JSON without building it.
func DumpConfig(file string) (string, error) {
	errors.LogInfo(context.Background(), "Reading config: ", file)
	r, err := confloader.LoadConfig(file)
	if err != nil {
		return "", errors.New("failed to read config: ", file).Base(err)
	}
	var config conf.Config
	if err := json.NewDecoder(r).Decode(&config); err != nil {
		return "", errors.New("failed to decode config: ", file).Base(err)
	}
	if j, ok := creflect.MarshalToJson(&config, true); ok {
		return j, nil
	}
	return "", errors.New("marshal to json failed.")
}
