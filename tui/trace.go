package tui

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Trace logs TUI events to ~/.mxcli/tui-debug.log when MXCLI_TUI_DEBUG=1.
//
// Usage:
//   MXCLI_TUI_DEBUG=1 mxcli tui -p app.mpr
//   tail -f ~/.mxcli/tui-debug.log

var (
	traceOnce   sync.Once
	traceLogger *log.Logger
	traceFile   *os.File
	traceActive bool
)

func initTrace() {
	traceOnce.Do(func() {
		if os.Getenv("MXCLI_TUI_DEBUG") != "1" {
			return
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return
		}
		dir := filepath.Join(home, ".mxcli")
		_ = os.MkdirAll(dir, 0o755)

		path := filepath.Join(dir, "tui-debug.log")
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			return
		}
		traceFile = f
		traceLogger = log.New(f, "", 0)
		traceActive = true
		traceLogger.Printf("=== TUI debug started at %s ===", time.Now().Format(time.RFC3339))
	})
}

// Trace logs a formatted message to the debug log file.
func Trace(format string, args ...any) {
	initTrace()
	if !traceActive {
		return
	}
	ts := time.Now().Format("15:04:05.000")
	traceLogger.Printf("%s %s", ts, fmt.Sprintf(format, args...))
}

// CloseTrace flushes and closes the debug log file.
func CloseTrace() {
	if traceFile != nil {
		traceFile.Close()
	}
}
