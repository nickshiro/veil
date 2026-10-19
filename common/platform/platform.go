package platform // import "github.com/nickshiro/veil/common/platform"

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	ConfigLocation  = "xray.location.config"
	ConfdirLocation = "xray.location.confdir"
	AssetLocation   = "xray.location.asset"
	CertLocation    = "xray.location.cert"

	UseReadV         = "xray.buf.readv"
	UseFreedomSplice = "xray.buf.splice"
	UseVmessPadding  = "xray.vmess.padding"
	UseCone          = "xray.cone.disabled"

	BufferSize           = "xray.ray.buffer.size"
	BrowserDialerAddress = "xray.browser.dialer"
	XUDPLog              = "xray.xudp.show"
	XUDPBaseKey          = "xray.xudp.basekey"

	TunFdKey = "xray.tun.fd"
)

type EnvFlag struct {
	Name    string
	AltName string
}

func NewEnvFlag(name string) EnvFlag {
	return EnvFlag{
		Name:    name,
		AltName: NormalizeEnvName(name),
	}
}

func (f EnvFlag) GetValue(defaultValue func() string) string {
	if v, found := os.LookupEnv(f.Name); found {
		return v
	}
	if len(f.AltName) > 0 {
		if v, found := os.LookupEnv(f.AltName); found {
			return v
		}
	}

	return defaultValue()
}

func (f EnvFlag) GetValueAsInt(defaultValue int) int {
	useDefaultValue := false
	s := f.GetValue(func() string {
		useDefaultValue = true
		return ""
	})
	if useDefaultValue {
		return defaultValue
	}
	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return defaultValue
	}
	return int(v)
}

func NormalizeEnvName(name string) string {
	return strings.ReplaceAll(strings.ToUpper(strings.TrimSpace(name)), ".", "_")
}

func getExecutableDir() string {
	exec, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Dir(exec)
}

func GetConfigurationPath() string {
	configPath := NewEnvFlag(ConfigLocation).GetValue(getExecutableDir)
	return filepath.Join(configPath, "config.json")
}

// GetConfDirPath reads "xray.location.confdir"
func GetConfDirPath() string {
	configPath := NewEnvFlag(ConfdirLocation).GetValue(func() string { return "" })
	return configPath
}

func resolveFile(paths []string) (string, error) {
	var tried []string
	for _, path := range paths {
		path, err := filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("failed to resolve file path: %w", err)
		}
		tried = append(tried, path)
		info, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("failed to inspect file %q: %w", path, err)
		}
		if !info.Mode().IsRegular() {
			return "", fmt.Errorf("file is not a regular file: %s", path)
		}
		return path, nil
	}
	return "", fmt.Errorf("file not found; tried %q: %w", tried, os.ErrNotExist)
}
