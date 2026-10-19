package crypto

import (
	"crypto/cipher"
	"io"

	"github.com/nickshiro/veil/common/buf"
)

var _ buf.Writer = (*CryptionWriter)(nil)

type CryptionWriter struct {
	stream    cipher.Stream
	writer    io.Writer
	bufWriter buf.Writer
}

// NewCryptionWriter creates a new CryptionWriter.
func NewCryptionWriter(stream cipher.Stream, writer io.Writer) *CryptionWriter {
	return &CryptionWriter{
		stream:    stream,
		writer:    writer,
		bufWriter: buf.NewWriter(writer),
	}
}

// Write implements io.Writer.Write().
func (w *CryptionWriter) Write(data []byte) (int, error) {
	w.stream.XORKeyStream(data, data)

	if err := buf.WriteAllBytes(w.writer, data, nil); err != nil {
		return 0, err
	}
	return len(data), nil
}

// WriteMultiBuffer implements buf.Writer.
func (w *CryptionWriter) WriteMultiBuffer(mb buf.MultiBuffer) error {
	for _, b := range mb {
		w.stream.XORKeyStream(b.Bytes(), b.Bytes())
	}

	return w.bufWriter.WriteMultiBuffer(mb)
}
