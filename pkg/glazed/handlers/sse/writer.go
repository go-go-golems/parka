package sse

import (
	"io"
)

// SSEWriter implements io.Writer to emit written bytes as SSE over HTTP.
type SSEWriter struct {
	ch chan []byte
}

var _ io.Writer = (*SSEWriter)(nil)

func NewSSEWriter() *SSEWriter {
	return &SSEWriter{
		ch: make(chan []byte),
	}
}

func (w *SSEWriter) Write(p []byte) (n int, err error) {
	w.ch <- p
	return len(p), nil
}

func (w *SSEWriter) Close() {
	close(w.ch)
}
