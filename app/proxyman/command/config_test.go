package command

import (
	"testing"

	"github.com/nickshiro/veil/app/proxyman"
	"github.com/nickshiro/veil/common/serial"
	"github.com/nickshiro/veil/core"
	"github.com/nickshiro/veil/proxy/dokodemo"
	"github.com/nickshiro/veil/proxy/freedom"
	"google.golang.org/protobuf/proto"
)

func TestHandlerConfigAdapters(t *testing.T) {
	in := &core.InboundHandlerConfig{
		Tag:              "in",
		ReceiverSettings: serial.ToTypedMessage(&proxyman.ReceiverConfig{}),
		ProxySettings:    serial.ToTypedMessage(&dokodemo.Config{RewritePort: 1234}),
	}
	out := &core.OutboundHandlerConfig{
		Tag:            "out",
		SenderSettings: serial.ToTypedMessage(&proxyman.SenderConfig{ViaCidr: "24"}),
		ProxySettings:  serial.ToTypedMessage(&freedom.Config{}),
	}
	nativeIn, err := decodeInbound(in)
	if err != nil {
		t.Fatal(err)
	}
	nativeOut, err := decodeOutbound(out)
	if err != nil {
		t.Fatal(err)
	}
	if nativeIn.Tag != in.Tag || nativeOut.Tag != out.Tag {
		t.Fatal("lost handler tags")
	}
	for _, tt := range []struct {
		native any
		wire   *serial.TypedMessage
	}{
		{nativeIn.ReceiverSettings, in.ReceiverSettings},
		{nativeIn.ProxySettings, in.ProxySettings},
		{nativeOut.SenderSettings, out.SenderSettings},
		{nativeOut.ProxySettings, out.ProxySettings},
	} {
		got, err := encodeSettings(tt.native)
		if err != nil {
			t.Fatal(err)
		}
		if !proto.Equal(got, tt.wire) {
			t.Fatalf("settings changed: got %v, want %v", got, tt.wire)
		}
	}
}

func TestHandlerConfigAdapterErrors(t *testing.T) {
	if _, err := decodeInbound(nil); err == nil {
		t.Fatal("accepted missing inbound")
	}
	if _, err := decodeOutbound(nil); err == nil {
		t.Fatal("accepted missing outbound")
	}
	if _, err := encodeSettings(struct{}{}); err == nil {
		t.Fatal("accepted unsupported API settings")
	}
	for _, settings := range []*serial.TypedMessage{
		{Type: "unknown.Config"},
		{Type: serial.GetMessageType(&freedom.Config{}), Value: []byte{0xff}},
	} {
		if _, err := decodeOutbound(&core.OutboundHandlerConfig{ProxySettings: settings}); err == nil {
			t.Fatal("accepted invalid protobuf settings")
		}
	}
}
