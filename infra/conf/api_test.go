package conf_test

import (
	"reflect"
	"testing"

	"github.com/nickshiro/veil/app/commander"
	logservice "github.com/nickshiro/veil/app/log/command"
	observatoryservice "github.com/nickshiro/veil/app/observatory/command"
	handlerservice "github.com/nickshiro/veil/app/proxyman/command"
	routerservice "github.com/nickshiro/veil/app/router/command"
	statsservice "github.com/nickshiro/veil/app/stats/command"
	"github.com/nickshiro/veil/infra/conf"
)

func TestAPIConfigNativeServices(t *testing.T) {
	input := &conf.APIConfig{
		Tag: "api", Listen: "127.0.0.1:10085",
		Services: []string{
			"ReflectionService", "HandlerService", "LoggerService",
			"StatsService", "ObservatoryService", "RoutingService", "unknown",
		},
	}
	got, err := input.Build()
	if err != nil {
		t.Fatal(err)
	}
	if got.Tag != input.Tag || got.Listen != input.Listen {
		t.Fatal("listener settings changed")
	}
	want := []any{
		&commander.ReflectionConfig{}, &handlerservice.Config{}, &logservice.Config{},
		&statsservice.Config{}, &observatoryservice.Config{}, &routerservice.Config{},
	}
	if len(got.Service) != len(want) {
		t.Fatalf("got %d services, want %d", len(got.Service), len(want))
	}
	for i := range want {
		if reflect.TypeOf(got.Service[i]) != reflect.TypeOf(want[i]) {
			t.Fatalf("service %d: got %T, want %T", i, got.Service[i], want[i])
		}
	}
	if _, err := (&conf.APIConfig{}).Build(); err == nil {
		t.Fatal("accepted empty API tag")
	}
}
