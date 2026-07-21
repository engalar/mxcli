package task

import (
	"bytes"
	"io"
	"sync"
)

type LineWriter struct {
	inner io.Writer
	Lines chan string
	buf   bytes.Buffer
	mu    sync.Mutex
}

func NewLineWriter(inner io.Writer) *LineWriter {
	return &LineWriter{
		inner: inner,
		Lines: make(chan string, 500),
	}
}

func (w *LineWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	n, err := w.inner.Write(p)

	w.buf.Write(p)
	for {
		line, err := w.buf.ReadString('\n')
		if err != nil {
			// Put partial line back
			w.buf.WriteString(line)
			break
		}
		line = line[:len(line)-1] // strip \n
		select {
		case w.Lines <- line:
		default:
			// drop if full
		}
	}
	return n, err
}

func (w *LineWriter) Close() {
	close(w.Lines)
}
