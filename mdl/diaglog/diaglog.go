// SPDX-License-Identifier: Apache-2.0

// Package diaglog provides lightweight session logging for mxcli diagnostics.
// Logs are written as JSON Lines to ~/.mxcli/logs/mxcli-YYYY-MM-DD.log.
// Logging is enabled by default and can be disabled with MXCLI_LOG=0.
// If the log directory cannot be created, logging silently degrades to a no-op.
package diaglog

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Logger wraps slog.Logger with session tracking and convenience methods.
// A nil Logger is safe to use — all methods are no-ops on nil receivers.
type Logger struct {
	slog      *slog.Logger
	stderrLog *slog.Logger // non-nil when verbose > 0
	verbose   int          // 0=off, 1=trace (text), 2=debug (json)
	file      *os.File
	cmdCount  int
	errCount  int
	startTime time.Time
}

// Init creates the daily log file and writes a session header.
// Returns nil if logging is disabled (MXCLI_LOG=0) or the log file cannot be created.
// verboseLevel controls stderr mirroring: 0=off, 1=text (INFO+), 2=JSON (DEBUG+).
func Init(version, mode string, verboseLevel int) *Logger {
	if isDisabled() {
		return nil
	}

	logDir := logDirectory()
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil
	}

	// Clean old logs (best-effort, errors ignored)
	cleanOldLogs(logDir, 7*24*time.Hour)

	filename := fmt.Sprintf("mxcli-%s.log", time.Now().Format("2006-01-02"))
	logPath := filepath.Join(logDir, filename)

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil
	}

	handler := slog.NewJSONHandler(f, &slog.HandlerOptions{Level: slog.LevelInfo})
	l := &Logger{
		slog:      slog.New(handler),
		verbose:   verboseLevel,
		file:      f,
		startTime: time.Now(),
	}

	// Set up stderr mirroring if requested
	switch verboseLevel {
	case 1:
		h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
		l.stderrLog = slog.New(h)
	case 2:
		h := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})
		l.stderrLog = slog.New(h)
	}

	// Write session header
	sessionArgs := []any{
		"version", version,
		"go", runtime.Version(),
		"os", runtime.GOOS,
		"arch", runtime.GOARCH,
		"mode", mode,
		"args", os.Args,
		"pid", os.Getpid(),
	}
	l.slog.Info("session_start", sessionArgs...)
	if l.stderrLog != nil {
		l.stderrLog.Info("session_start", sessionArgs...)
	}

	return l
}

// Close writes a session summary and closes the log file.
func (l *Logger) Close() {
	if l == nil {
		return
	}
	l.slog.Info("session_end",
		"commands_executed", l.cmdCount,
		"errors_count", l.errCount,
		"duration_s", int(time.Since(l.startTime).Seconds()),
	)
	l.file.Close()
}

// Command logs a statement execution with timing and optional error.
func (l *Logger) Command(stmtType, summary string, duration time.Duration, err error) {
	if l == nil {
		return
	}
	l.cmdCount++
	if err != nil {
		l.errCount++
		args := []any{
			"stmt_type", stmtType,
			"stmt_summary", summary,
			"error", err.Error(),
			"duration_ms", duration.Milliseconds(),
		}
		l.slog.Error("execute_error", args...)
		if l.stderrLog != nil {
			l.stderrLog.Error("execute_error", args...)
		}
	} else {
		args := []any{
			"stmt_type", stmtType,
			"stmt_summary", summary,
			"duration_ms", duration.Milliseconds(),
		}
		l.slog.Info("execute", args...)
		if l.stderrLog != nil {
			l.stderrLog.Info("execute", args...)
		}
	}
}

// Connect logs a project connection event.
func (l *Logger) Connect(mprPath, mendixVersion string, formatVersion int) {
	if l == nil {
		return
	}
	args := []any{
		"mpr_path", mprPath,
		"mendix_version", mendixVersion,
		"mpr_format", formatVersion,
	}
	l.slog.Info("connect", args...)
	if l.stderrLog != nil {
		l.stderrLog.Info("connect", args...)
	}
}

// ParseError logs parse failures with a truncated input preview.
func (l *Logger) ParseError(inputPreview string, errs []error) {
	if l == nil {
		return
	}
	l.errCount++
	errStrings := make([]string, len(errs))
	for i, e := range errs {
		errStrings[i] = e.Error()
	}
	args := []any{
		"input_preview", truncate(inputPreview, 200),
		"errors", errStrings,
	}
	l.slog.Warn("parse_error", args...)
	if l.stderrLog != nil {
		l.stderrLog.Warn("parse_error", args...)
	}
}

// Info logs a general informational message.
func (l *Logger) Info(msg string, args ...any) {
	if l == nil {
		return
	}
	l.slog.Info(msg, args...)
	if l.stderrLog != nil {
		l.stderrLog.Info(msg, args...)
	}
}

// Warn logs a warning message.
func (l *Logger) Warn(msg string, args ...any) {
	if l == nil {
		return
	}
	l.slog.Warn(msg, args...)
	if l.stderrLog != nil {
		l.stderrLog.Warn(msg, args...)
	}
}

// Error logs an error message.
func (l *Logger) Error(msg string, args ...any) {
	if l == nil {
		return
	}
	l.errCount++
	l.slog.Error(msg, args...)
	if l.stderrLog != nil {
		l.stderrLog.Error(msg, args...)
	}
}

// LogDir returns the log directory path.
func LogDir() string {
	return logDirectory()
}

// isDisabled checks if logging is disabled via environment variable.
func isDisabled() bool {
	v := os.Getenv("MXCLI_LOG")
	switch strings.ToLower(v) {
	case "0", "false", "off", "no":
		return true
	}
	return false
}

// logDirectory returns ~/.mxcli/logs/.
func logDirectory() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".mxcli", "logs")
}

// cleanOldLogs removes log files older than maxAge from the log directory.
func cleanOldLogs(logDir string, maxAge time.Duration) {
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-maxAge)
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "mxcli-") || !strings.HasSuffix(entry.Name(), ".log") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(logDir, entry.Name()))
		}
	}
}

// truncate shortens a string to maxLen characters, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
