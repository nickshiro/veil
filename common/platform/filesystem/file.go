package filesystem

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/nickshiro/veil/common/buf"
	"github.com/nickshiro/veil/common/platform"
)

func ReadFile(path string) ([]byte, error) {
	reader, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return buf.ReadAllToBytes(reader)
}

func OpenAsset(file string) (io.ReadCloser, int64, error) {
	path, info, err := getAssetFileLocation(file)
	if err != nil {
		return nil, 0, err
	}
	r, err := os.Open(path)
	return r, info.Size(), err
}

func StatAsset(file string) (os.FileInfo, error) {
	_, info, err := getAssetFileLocation(file)
	return info, err
}

func ResolveAsset(file string) (string, error) {
	path, _, err := getAssetFileLocation(file)
	return path, err
}

func getAssetFileLocation(file string) (string, os.FileInfo, error) {
	if !filepath.IsLocal(file) || file == "." {
		return "", nil, errors.New("asset path must stay in asset directory")
	}
	local, err := filepath.Localize(file)
	if err != nil {
		return "", nil, err
	}
	path := platform.GetAssetLocation(local)
	info, err := os.Stat(path)
	if err != nil {
		return "", nil, err
	}
	if !info.Mode().IsRegular() {
		return "", nil, errors.New("asset is not a regular file")
	}
	return path, info, nil
}

func ReadCert(file string) ([]byte, error) {
	if filepath.IsAbs(file) {
		return ReadFile(file)
	}
	return ReadFile(platform.GetCertLocation(file))
}
