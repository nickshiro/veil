package crypto_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/nickshiro/veil/common"
	"github.com/nickshiro/veil/common/buf"
	. "github.com/nickshiro/veil/common/crypto"
)

func TestChunkStreamReader(t *testing.T) {
	// Two length-prefixed chunks followed by the zero-length end marker.
	cache := bytes.NewReader([]byte{0, 4, 'a', 'b', 'c', 'd', 0, 3, 'e', 'f', 'g', 0, 0})
	reader := NewChunkStreamReaderWithChunkCount(PlainChunkSizeParser{}, cache, 0)

	mb, err := reader.ReadMultiBuffer()
	common.Must(err)

	if s := mb.String(); s != "abcd" {
		t.Error("content: ", s)
	}

	buf.ReleaseMulti(mb)
	mb, err = reader.ReadMultiBuffer()
	common.Must(err)

	if s := mb.String(); s != "efg" {
		t.Error("content: ", s)
	}

	buf.ReleaseMulti(mb)
	_, err = reader.ReadMultiBuffer()
	if err != io.EOF {
		t.Error("error: ", err)
	}
}
