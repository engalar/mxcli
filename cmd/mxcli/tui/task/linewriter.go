package task

import (
	"bytes"
	"io"
	"strings"
	"sync"
)

type LineWriter struct {
	inner io.Writer
	Lines chan string
	buf   bytes.Buffer
	mu    sync.Mutex
	done  bool
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

	// Strip ANSI escape codes
	cleaned := stripANSI(string(p))
	w.buf.WriteString(cleaned)
	w.flushLines()
	return n, err
}

func (w *LineWriter) flushLines() {
	for {
		raw, err := w.buf.ReadString('\n')
		if err != nil {
			w.buf.WriteString(raw)
			break
		}
		raw = strings.TrimRight(raw, "\n\r")
		raw = strings.ReplaceAll(raw, "\r", "")
		if raw == "" {
			continue
		}
		select {
		case w.Lines <- raw:
		default:
		}
	}
}

func (w *LineWriter) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.done {
		return
	}
	w.done = true
	// Flush any remaining partial line
	remain := strings.TrimRight(w.buf.String(), "\n\r\t ")
	if remain != "" {
		remain = strings.ReplaceAll(remain, "\r", "")
		select {
		case w.Lines <- remain:
		default:
		}
	}
	close(w.Lines)
}

// stripANSI removes ANSI escape sequences from a string.
func stripANSI(s string) string {
	var out strings.Builder
	out.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' {
			i++
			for i < len(s) {
				if s[i] >= 0x40 && s[i] <= 0x7e {
					break
				}
				i++
			}
			continue
		}
		out.WriteByte(s[i])
	}
	return out.String()
}
