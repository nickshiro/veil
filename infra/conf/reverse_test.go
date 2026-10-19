package conf_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/nickshiro/veil/infra/conf"
)

func TestRemovedReverseConfig(t *testing.T) {
	for _, input := range []string{
		`{"reverse":{}}`,
		`{"reverse":{"bridges":[{"tag":"test","domain":"test.example.com"}]}}`,
		`{"reverse":{"portals":[{"tag":"test","domain":"test.example.com"}]}}`,
	} {
		var config conf.Config
		if err := json.Unmarshal([]byte(input), &config); err != nil {
			t.Fatal(err)
		}
		if _, err := config.Build(); err == nil || !strings.Contains(err.Error(), "legacy reverse") {
			t.Fatalf("expected legacy reverse error for %s, got %v", input, err)
		}
	}
}
