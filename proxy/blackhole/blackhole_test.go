package blackhole_test

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"net/http"
	"testing"

	"github.com/nickshiro/veil/common"
	"github.com/nickshiro/veil/common/buf"
	"github.com/nickshiro/veil/common/session"
	"github.com/nickshiro/veil/proxy/blackhole"
	"github.com/nickshiro/veil/transport"
	"github.com/nickshiro/veil/transport/pipe"
)

func TestBlackholeHTTPResponse(t *testing.T) {
	ctx := session.ContextWithOutbounds(context.Background(), []*session.Outbound{{}})
	handler, err := blackhole.New(ctx, &blackhole.Config{
		Response: &blackhole.Response{Type: "http"},
	})
	common.Must(err)

	reader, writer := pipe.New(pipe.WithoutSizeLimit())

	dataCh := make(chan buf.MultiBuffer, 1)
	go func() {
		mb := common.Must2(reader.ReadMultiBuffer())
		dataCh <- mb
	}()
	link := transport.Link{
		Reader: reader,
		Writer: writer,
	}
	common.Must(handler.Process(ctx, &link, nil))
	mb := <-dataCh
	data := make([]byte, mb.Len())
	mb.Copy(data)
	resp := common.Must2(http.ReadResponse(bufio.NewReader(bytes.NewBuffer(data)), nil))
	if resp.StatusCode != 403 {
		t.Errorf("expected 403 response, got %d", resp.StatusCode)
	}
}

func TestBlackholeCustomResponse(t *testing.T) {
	ctx := session.ContextWithOutbounds(context.Background(), []*session.Outbound{{}})
	// slightly bigger than a buffer
	expected := make([]byte, buf.Size+1000)
	if _, err := rand.Read(expected); err != nil {
		t.Fatal(err)
	}
	handler, err := blackhole.New(ctx, &blackhole.Config{
		Response: &blackhole.Response{
			Type:               "custom",
			CustomResponseData: expected,
		},
	})
	common.Must(err)

	reader, writer := pipe.New(pipe.WithoutSizeLimit())
	var actual buf.MultiBuffer
	var rerr error
	go func() {
		actual, rerr = reader.ReadMultiBuffer()
	}()

	link := transport.Link{Reader: reader, Writer: writer}
	common.Must(handler.Process(ctx, &link, nil))
	common.Must(rerr)

	if actual.String() != string(expected) {
		t.Errorf("custom response mismatch")
	}
}
