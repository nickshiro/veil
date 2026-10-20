package conf_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/nickshiro/veil/app/geodata"
	. "github.com/nickshiro/veil/infra/conf"
)

func TestGeodataConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XRAY_LOCATION_ASSET", dir)
	for _, name := range []string{"geoip.dat", "geosite.dat"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	var input GeodataConfig
	if err := json.Unmarshal([]byte(`{
		"cron": "0 4 * * *",
		"outbound": "proxy",
		"assets": [
			{"url": "https://example.com/geoip.dat", "file": "geoip.dat"},
			{"url": "https://example.com/geosite.dat", "file": "geosite.dat"}
		]
	}`), &input); err != nil {
		t.Fatal(err)
	}
	got, err := input.Build()
	if err != nil {
		t.Fatal(err)
	}
	want := &geodata.Config{
		Cron: "0 4 * * *", Outbound: "proxy",
		Assets: []*geodata.Asset{
			{Url: "https://example.com/geoip.dat", File: "geoip.dat"},
			{Url: "https://example.com/geosite.dat", File: "geosite.dat"},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("config mismatch (-want +got):\n%s", diff)
	}
}

func TestGeodataAssetConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XRAY_LOCATION_ASSET", dir)
	if err := os.WriteFile(filepath.Join(dir, "geoip.dat"), nil, 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := (&GeodataAssetConfig{
		URL:  "https://example.com/geoip.dat",
		File: "geoip.dat",
	}).Build(); err != nil {
		t.Fatal(err)
	}

	if _, err := (&GeodataAssetConfig{
		URL:  "https://example.com/geoip.dat",
		File: "missing.dat",
	}).Build(); err == nil {
		t.Fatal("expected error")
	}
}

func TestGeodataAssetConfigInvalidURL(t *testing.T) {
	t.Setenv("XRAY_LOCATION_ASSET", filepath.Join("..", "..", "resources"))

	for _, rawURL := range []string{
		"",
		"http://example.com/geoip.dat",
		"ftp://example.com/geoip.dat",
		"https:///geoip.dat",
	} {
		if _, err := (&GeodataAssetConfig{
			URL:  rawURL,
			File: "geoip.dat",
		}).Build(); err == nil {
			t.Fatalf("expected error for %q", rawURL)
		}
	}
}
