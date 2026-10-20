package external

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/nickshiro/veil/common/errors"
	"github.com/nickshiro/veil/main/confloader"
)

func ConfigLoader(arg string) (out io.Reader, err error) {
	var data []byte
	switch {
	case arg == "stdin:":
		data, err = io.ReadAll(os.Stdin)

	default:
		file, openErr := os.Open(arg)
		if openErr != nil {
			return nil, openErr
		}
		defer file.Close()
		data, err = readConfigFile(file)
	}

	if err != nil {
		return
	}
	out = bytes.NewBuffer(data)
	return
}

func readConfigFile(file *os.File) ([]byte, error) {
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if info.Mode().IsRegular() {
		path, err := filepath.Abs(file.Name())
		if err != nil {
			return nil, err
		}
		path, err = filepath.EvalSymlinks(path)
		if err != nil {
			return nil, err
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".json":
		default:
			return nil, errors.New("unsupported config file extension: ", path)
		}
	}
	return io.ReadAll(file)
}

func init() {
	confloader.EffectiveConfigFileLoader = ConfigLoader
}
