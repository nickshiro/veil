package platform

import (
	"os"
	"path/filepath"
	"strconv"
)

const (
	ConfigLocation = "XRAY_LOCATION_CONFIG"
	AssetLocation  = "XRAY_LOCATION_ASSET"
	CertLocation   = "XRAY_LOCATION_CERT"

	UseReadV         = "XRAY_BUF_READV"
	UseFreedomSplice = "XRAY_BUF_SPLICE"
	UseVmessPadding  = "XRAY_VMESS_PADDING"
	UseCone          = "XRAY_CONE_DISABLED"

	BufferSize           = "XRAY_RAY_BUFFER_SIZE"
	BrowserDialerAddress = "XRAY_BROWSER_DIALER"
	XUDPLog              = "XRAY_XUDP_SHOW"
	XUDPBaseKey          = "XRAY_XUDP_BASEKEY"

	TunFdKey = "XRAY_TUN_FD"
)

type EnvFlag struct {
	Name string
}

func NewEnvFlag(name string) EnvFlag {
	return EnvFlag{
		Name: name,
	}
}

func (f EnvFlag) GetValue(defaultValue func() string) string {
	if v, found := os.LookupEnv(f.Name); found {
		return v
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
