# Verbose Flag + Diag Bundle Enhancement Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add global `-v`/`-vv` trace/debug flag to mxcli and enhance `diag --bundle` with env dump, error stacks, and optional project metadata.

**Architecture:** `diaglog.Logger` gains a `stderrLog *slog.Logger` field (TextHandler for -v, JSONHandler for -vv) that mirrors all log calls to stderr when verbose > 0. A new package-level `globalVerboseLevel int` variable in `main.go` is set by `PersistentPreRunE` and passed to `diaglog.Init`. The bundle enhancement adds three new tar entries via helper functions in `diag.go`.

**Tech Stack:** Go standard library `log/slog`, `runtime`, `archive/tar`, `compress/gzip`, `modelsdk/mpr.Reader`

---

## Pre-flight: Known Flag Conflicts to Resolve

`testRunCmd` and `playwrightVerifyCmd` both define `-v` as a local BoolP flag. Adding a global PersistentFlag CountP with the same shorthand will cause Cobra to panic at startup. Task 2 removes these conflicting shorthands.

---

## Task 1: Update diaglog.Logger to support verbose stderr output (TDD)

**Files:**
- Modify: `mdl/diaglog/diaglog.go`
- Modify: `mdl/diaglog/diaglog_test.go`

- [ ] **Step 1: Write failing tests for verbose stderr output**

Add to `mdl/diaglog/diaglog_test.go` (after the existing `TestInitAndClose` test):

```go
func TestVerboseLevel0_NoStderr(t *testing.T) {
	tmpDir := t.TempDir()
	setHomeDir(t, tmpDir)

	// Capture stderr
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	l := Init("test", "batch", 0)
	if l == nil {
		t.Fatal("expected non-nil logger")
	}
	l.Info("should not appear on stderr")
	l.Close()

	w.Close()
	os.Stderr = old
	var buf strings.Builder
	io.Copy(&buf, r)

	if buf.Len() != 0 {
		t.Errorf("verboseLevel=0 should produce no stderr, got: %q", buf.String())
	}
}

func TestVerboseLevel1_TextOnStderr(t *testing.T) {
	tmpDir := t.TempDir()
	setHomeDir(t, tmpDir)

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	l := Init("test", "batch", 1)
	if l == nil {
		t.Fatal("expected non-nil logger")
	}
	l.Info("hello from trace")
	l.Close()

	w.Close()
	os.Stderr = old
	var buf strings.Builder
	io.Copy(&buf, r)

	output := buf.String()
	if !strings.Contains(output, "hello from trace") {
		t.Errorf("verboseLevel=1 should write info to stderr, got: %q", output)
	}
	// TextHandler output should NOT be JSON (no leading '{')
	if strings.Contains(output, `{"level"`) {
		t.Errorf("verboseLevel=1 should use text format, not JSON, got: %q", output)
	}
}

func TestVerboseLevel2_JSONOnStderr(t *testing.T) {
	tmpDir := t.TempDir()
	setHomeDir(t, tmpDir)

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	l := Init("test", "batch", 2)
	if l == nil {
		t.Fatal("expected non-nil logger")
	}
	l.Info("hello from debug")
	l.Close()

	w.Close()
	os.Stderr = old
	var buf strings.Builder
	io.Copy(&buf, r)

	output := buf.String()
	if !strings.Contains(output, "hello from debug") {
		t.Errorf("verboseLevel=2 should write info to stderr, got: %q", output)
	}
	// JSONHandler output should be JSON
	if !strings.Contains(output, `"msg"`) {
		t.Errorf("verboseLevel=2 should use JSON format, got: %q", output)
	}
}

func TestVerboseCommandMirror(t *testing.T) {
	tmpDir := t.TempDir()
	setHomeDir(t, tmpDir)

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	l := Init("test", "batch", 1)
	if l == nil {
		t.Fatal("expected non-nil logger")
	}
	l.Command("ShowStmt", "SHOW ENTITIES", 42*time.Millisecond, nil)
	l.Command("BadStmt", "BAD", time.Millisecond, errors.New("something went wrong"))
	l.Close()

	w.Close()
	os.Stderr = old
	var buf strings.Builder
	io.Copy(&buf, r)

	output := buf.String()
	if !strings.Contains(output, "SHOW ENTITIES") {
		t.Errorf("expected SHOW ENTITIES in stderr output, got: %q", output)
	}
	if !strings.Contains(output, "something went wrong") {
		t.Errorf("expected error message in stderr output, got: %q", output)
	}
}
```

Also add needed imports at the top of the test file (after the existing imports):
```go
import (
	"errors"
	"io"
	"os"
	// ... existing imports ...
)
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./mdl/diaglog/... -run "TestVerbose" -v 2>&1 | tail -20
```

Expected: `FAIL` with `undefined: Init` (wrong arg count) or compile error.

- [ ] **Step 3: Update diaglog.Logger struct and Init function**

Replace the `Logger` struct and `Init` function in `mdl/diaglog/diaglog.go`:

```go
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
// verboseLevel: 0=off, 1=trace (human-readable text to stderr), 2=debug (JSON to stderr).
// Returns nil if logging is disabled (MXCLI_LOG=0) or the log file cannot be created.
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

	// Configure stderr handler based on verbose level
	if verboseLevel == 1 {
		l.stderrLog = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	} else if verboseLevel >= 2 {
		l.stderrLog = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	}

	// Write session header
	l.slog.Info("session_start",
		"version", version,
		"go", runtime.Version(),
		"os", runtime.GOOS,
		"arch", runtime.GOARCH,
		"mode", mode,
		"args", os.Args,
		"pid", os.Getpid(),
	)
	if l.stderrLog != nil {
		l.stderrLog.Info("session_start",
			"version", version,
			"mode", mode,
		)
	}

	return l
}
```

- [ ] **Step 4: Add stderr mirroring to all log methods**

Replace the `Command`, `Connect`, `ParseError`, `Info`, `Warn`, `Error` methods in `mdl/diaglog/diaglog.go`:

```go
// Command logs a statement execution with timing and optional error.
func (l *Logger) Command(stmtType, summary string, duration time.Duration, err error) {
	if l == nil {
		return
	}
	l.cmdCount++
	if err != nil {
		l.errCount++
		l.slog.Error("execute_error",
			"stmt_type", stmtType,
			"stmt_summary", summary,
			"error", err.Error(),
			"duration_ms", duration.Milliseconds(),
		)
		if l.stderrLog != nil {
			l.stderrLog.Error("execute_error",
				"stmt_type", stmtType,
				"stmt_summary", summary,
				"error", err.Error(),
				"duration_ms", duration.Milliseconds(),
			)
		}
	} else {
		l.slog.Info("execute",
			"stmt_type", stmtType,
			"stmt_summary", summary,
			"duration_ms", duration.Milliseconds(),
		)
		if l.stderrLog != nil {
			l.stderrLog.Info("execute",
				"stmt_type", stmtType,
				"stmt_summary", summary,
				"duration_ms", duration.Milliseconds(),
			)
		}
	}
}

// Connect logs a project connection event.
func (l *Logger) Connect(mprPath, mendixVersion string, formatVersion int) {
	if l == nil {
		return
	}
	l.slog.Info("connect",
		"mpr_path", mprPath,
		"mendix_version", mendixVersion,
		"mpr_format", formatVersion,
	)
	if l.stderrLog != nil {
		l.stderrLog.Info("connect",
			"mpr_path", mprPath,
			"mendix_version", mendixVersion,
			"mpr_format", formatVersion,
		)
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
	l.slog.Warn("parse_error",
		"input_preview", truncate(inputPreview, 200),
		"errors", errStrings,
	)
	if l.stderrLog != nil {
		l.stderrLog.Warn("parse_error",
			"input_preview", truncate(inputPreview, 200),
			"errors", errStrings,
		)
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
```

- [ ] **Step 5: Run all diaglog tests**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./mdl/diaglog/... -v 2>&1 | tail -30
```

Expected: all tests pass. If compile errors, fix the import list (`errors`, `io` needed in test file).

- [ ] **Step 6: Commit**

```bash
git add mdl/diaglog/diaglog.go mdl/diaglog/diaglog_test.go
git commit -m "feat(diaglog): add verboseLevel parameter with stderr mirroring"
```

---

## Task 2: Fix flag conflicts and add global -v/-vv to root command

**Files:**
- Modify: `cmd/mxcli/main.go`

**Background:** `testRunCmd` uses `BoolP("verbose", "v", ...)` and `playwrightVerifyCmd` uses `BoolP("verbose", "v", ...)`. Adding a global `CountP("verbose", "v", ...)` will cause a Cobra panic. We remove the `-v` shorthand from both local flags and replace them with long-form only. We also remove `"-v"` from `shouldSuppressWarning()` since it no longer means "version".

- [ ] **Step 1: Remove -v from shouldSuppressWarning**

In `cmd/mxcli/main.go`, find the `shouldSuppressWarning` function. Change:

```go
		case "-q", "--quiet", "-h", "--help", "--version", "-v":
```

to:

```go
		case "-q", "--quiet", "-h", "--help", "--version":
```

- [ ] **Step 2: Remove -v shorthand from testRunCmd**

In `cmd/mxcli/main.go`, find:

```go
	testRunCmd.Flags().BoolP("verbose", "v", false, "Show all runtime log output")
```

Change to:

```go
	testRunCmd.Flags().Bool("verbose", false, "Show all runtime log output")
```

- [ ] **Step 3: Remove -v shorthand from playwrightVerifyCmd**

In `cmd/mxcli/cmd_playwright.go`, find:

```go
	playwrightVerifyCmd.Flags().BoolP("verbose", "v", false, "Show script stdout/stderr")
```

Change to:

```go
	playwrightVerifyCmd.Flags().Bool("verbose", false, "Show script stdout/stderr")
```

- [ ] **Step 4: Add globalVerboseLevel and global -v flag**

In `cmd/mxcli/main.go`, after the existing `var globalJSONFlag bool` declaration, add:

```go
// globalVerboseLevel is set by PersistentPreRunE when -v or -vv is passed.
// 0 = off, 1 = trace (human-readable), 2 = debug (JSON).
var globalVerboseLevel int
```

In the `init()` function, after the line:
```go
	rootCmd.PersistentFlags().Bool("json", false, "Output in JSON format")
```

Add:

```go
	rootCmd.PersistentFlags().CountP("verbose", "v", "Enable verbose output (-v trace, -vv debug)")
```

- [ ] **Step 5: Set globalVerboseLevel in PersistentPreRunE**

In `cmd/mxcli/main.go`, find the `PersistentPreRunE` body:

```go
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if p, _ := cmd.Root().PersistentFlags().GetString("project"); p != "" {
```

Add these lines at the **start** of the function body, before the `if p, _ := ...` block:

```go
		// Read verbose level (CountP: -v=1, -vv=2)
		if v, err := cmd.Root().PersistentFlags().GetCount("verbose"); err == nil {
			globalVerboseLevel = v
		}
```

- [ ] **Step 6: Update newLoggedExecutor to pass verbose level**

In `cmd/mxcli/main.go`, find `newLoggedExecutor`:

```go
func newLoggedExecutor(mode string) (*executor.Executor, *diaglog.Logger) {
	logger := diaglog.Init(version, mode)
```

Change to:

```go
func newLoggedExecutor(mode string) (*executor.Executor, *diaglog.Logger) {
	logger := diaglog.Init(version, mode, globalVerboseLevel)
```

- [ ] **Step 7: Update REPL path diaglog.Init call**

In `cmd/mxcli/main.go`, find the REPL path (inside the `else` branch of the rootCmd Run function):

```go
			logger := diaglog.Init(version, "repl")
```

Change to:

```go
			logger := diaglog.Init(version, "repl", 0)
```

(REPL mode does not support verbose; this call will be removed when REPL is deleted.)

- [ ] **Step 8: Build to verify no Cobra panic**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go build ./cmd/mxcli/ 2>&1
```

Expected: no output (clean build). If there's a "flag redefined" panic, double-check that both `testRunCmd` and `playwrightVerifyCmd` had their `-v` shorthands removed.

- [ ] **Step 9: Smoke test verbose output**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go run ./cmd/mxcli/ -v -p testdata/expr-checker/minimal.mpr -c "show entities" 2>&1 | head -20
```

Expected: stderr contains lines like `level=INFO msg=connect mpr_path=... mendix_version=...` followed by `level=INFO msg=execute stmt_type=ShowStmt ...`.

- [ ] **Step 10: Commit**

```bash
git add cmd/mxcli/main.go cmd/mxcli/cmd_playwright.go
git commit -m "feat(cli): add global -v/-vv verbose flag, fix testrun/playwright flag conflicts"
```

---

## Task 3: Enhance diag --bundle with env dump and error stacks

**Files:**
- Modify: `cmd/mxcli/diag.go`

This task adds two new bundle entries that don't require an MPR project: environment dump and error stacks.

- [ ] **Step 1: Write failing tests for collectEnvDump**

Add to `cmd/mxcli/diag_test.go` (create file if it doesn't exist):

```go
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"strings"
	"testing"
)

func TestCollectEnvDump_RedactsSensitiveKeys(t *testing.T) {
	t.Setenv("MY_SECRET_TOKEN", "supersecret")
	t.Setenv("NORMAL_VAR", "visible")

	dump := collectEnvDump()

	if strings.Contains(dump, "supersecret") {
		t.Error("sensitive value must be redacted")
	}
	if !strings.Contains(dump, "[REDACTED]") {
		t.Error("redacted placeholder must appear")
	}
	if !strings.Contains(dump, "NORMAL_VAR=visible") {
		t.Error("normal variables must appear with their values")
	}
}

func TestCollectEnvDump_ContainsRuntimeSection(t *testing.T) {
	dump := collectEnvDump()

	if !strings.Contains(dump, "=== Go Runtime ===") {
		t.Error("runtime section header must be present")
	}
	if !strings.Contains(dump, "NumCPU:") {
		t.Error("NumCPU must be present")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./cmd/mxcli/ -run "TestCollectEnv" -v 2>&1 | tail -10
```

Expected: compile error `undefined: collectEnvDump`.

- [ ] **Step 3: Implement collectEnvDump in diag.go**

Add the following import to `diag.go` (merge with existing imports):

```go
import (
    // existing imports...
    "runtime"  // already imported for runDiagInfo
    "sort"
    "strings"
)
```

Add this function to `cmd/mxcli/diag.go`:

```go
// collectEnvDump returns a human-readable dump of Go runtime stats and
// environment variables. Sensitive variables (matching *_TOKEN, *_KEY,
// *_SECRET, *_PASSWORD, *_PASS) have their values replaced with [REDACTED].
func collectEnvDump() string {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	var sb strings.Builder
	sb.WriteString("=== Go Runtime ===\n")
	fmt.Fprintf(&sb, "MemSys:       %d MB\n", ms.Sys/1024/1024)
	fmt.Fprintf(&sb, "HeapAlloc:    %d MB\n", ms.HeapAlloc/1024/1024)
	fmt.Fprintf(&sb, "NumGC:        %d\n", ms.NumGC)
	fmt.Fprintf(&sb, "NumCPU:       %d\n", runtime.NumCPU())
	fmt.Fprintf(&sb, "NumGoroutine: %d\n", runtime.NumGoroutine())
	sb.WriteString("\n=== Environment Variables ===\n")

	environ := os.Environ()
	sort.Strings(environ)
	for _, kv := range environ {
		eq := strings.IndexByte(kv, '=')
		if eq < 0 {
			fmt.Fprintf(&sb, "%s\n", kv)
			continue
		}
		key := kv[:eq]
		val := kv[eq+1:]
		upper := strings.ToUpper(key)
		if strings.HasSuffix(upper, "_TOKEN") ||
			strings.HasSuffix(upper, "_KEY") ||
			strings.HasSuffix(upper, "_SECRET") ||
			strings.HasSuffix(upper, "_PASSWORD") ||
			strings.HasSuffix(upper, "_PASS") {
			val = "[REDACTED]"
		}
		fmt.Fprintf(&sb, "%s=%s\n", key, val)
	}
	return sb.String()
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./cmd/mxcli/ -run "TestCollectEnv" -v 2>&1 | tail -10
```

Expected: `PASS`.

- [ ] **Step 5: Write failing test for collectErrorStacks**

Add to `cmd/mxcli/diag_test.go`:

```go
func TestCollectErrorStacks_ExtractsErrorLines(t *testing.T) {
	tmpDir := t.TempDir()

	// Write a fake log file with one ERROR and one INFO line
	logContent := `{"time":"2026-05-21T10:00:00Z","level":"ERROR","msg":"execute_error","error":"boom","stmt_summary":"SHOW X"}
{"time":"2026-05-21T10:00:01Z","level":"INFO","msg":"execute","stmt_type":"ShowStmt"}
`
	if err := os.WriteFile(tmpDir+"/mxcli-2026-05-21.log", []byte(logContent), 0644); err != nil {
		t.Fatal(err)
	}

	result := collectErrorStacks(tmpDir, 20)

	if !strings.Contains(result, "boom") {
		t.Error("error message must appear in output")
	}
	if strings.Contains(result, "ShowStmt") {
		t.Error("INFO lines must not appear in error stacks output")
	}
}

func TestCollectErrorStacks_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	// No log files → should return "no errors found" message
	result := collectErrorStacks(tmpDir, 20)
	if result == "" {
		t.Error("expected non-empty result even with no logs")
	}
}
```

- [ ] **Step 6: Run test to verify failure**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./cmd/mxcli/ -run "TestCollectError" -v 2>&1 | tail -10
```

Expected: compile error `undefined: collectErrorStacks`.

- [ ] **Step 7: Implement collectErrorStacks in diag.go**

Add to `cmd/mxcli/diag.go`:

```go
// collectErrorStacks scans log files in logDir and returns the most recent
// maxErrors ERROR-level entries formatted as a human-readable report.
func collectErrorStacks(logDir string, maxErrors int) string {
	files, _ := listLogFiles(logDir)
	if len(files) == 0 {
		return "(no log files found)\n"
	}

	type errorEntry struct {
		ts  string
		raw string
	}
	var entries []errorEntry

	for i := len(files) - 1; i >= 0 && len(entries) < maxErrors; i-- {
		lines := readFileLines(filepath.Join(logDir, files[i].Name()))
		for j := len(lines) - 1; j >= 0 && len(entries) < maxErrors; j-- {
			line := lines[j]
			if !strings.Contains(line, `"ERROR"`) {
				continue
			}
			var rec map[string]any
			if json.Unmarshal([]byte(line), &rec) != nil {
				continue
			}
			ts, _ := rec["time"].(string)
			entries = append(entries, errorEntry{ts: ts, raw: line})
		}
	}

	if len(entries) == 0 {
		return "(no errors found in recent logs)\n"
	}

	var sb strings.Builder
	for _, e := range entries {
		fmt.Fprintf(&sb, "=== %s ===\n%s\n\n", e.ts, e.raw)
	}
	return sb.String()
}
```

- [ ] **Step 8: Run tests to verify they pass**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./cmd/mxcli/ -run "TestCollectError" -v 2>&1 | tail -10
```

Expected: `PASS`.

- [ ] **Step 9: Wire env-dump and error-stacks into runDiagBundle**

In `cmd/mxcli/diag.go`, replace the `runDiagBundle` function:

```go
// runDiagBundle creates a tar.gz archive of logs and diagnostic data.
func runDiagBundle(logDir string) {
	timestamp := time.Now().Format("20060102-150405")
	outFile := fmt.Sprintf("mxcli-diag-%s.tar.gz", timestamp)

	f, err := os.Create(outFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating bundle: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	// system-info.txt (existing)
	info := fmt.Sprintf("Version: %s\nGo: %s %s/%s\nTime: %s\n",
		version, runtime.Version(), runtime.GOOS, runtime.GOARCH, time.Now().Format(time.RFC3339))
	addTarEntry(tw, "system-info.txt", []byte(info))

	// env-dump.txt (new)
	addTarEntry(tw, "env-dump.txt", []byte(collectEnvDump()))

	// error-stacks.txt (new)
	addTarEntry(tw, "error-stacks.txt", []byte(collectErrorStacks(logDir, 20)))

	// logs/ directory (existing)
	files, _ := listLogFiles(logDir)
	for _, entry := range files {
		path := filepath.Join(logDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		addTarEntry(tw, filepath.Join("logs", entry.Name()), data)
	}

	fmt.Printf("Created: %s\n", outFile)
	fmt.Printf("Contents: system-info.txt, env-dump.txt, error-stacks.txt, %d log file(s)\n", len(files))
}
```

- [ ] **Step 10: Build and verify**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go build ./cmd/mxcli/ 2>&1
```

Expected: no output.

- [ ] **Step 11: Commit**

```bash
git add cmd/mxcli/diag.go cmd/mxcli/diag_test.go
git commit -m "feat(diag): add env-dump and error-stacks to bundle output"
```

---

## Task 4: Add optional project-meta to bundle (requires -p flag)

**Files:**
- Modify: `cmd/mxcli/diag.go`

This task adds `project-meta.txt` to the bundle when `-p` is specified. It uses `modelsdk/mpr.Reader` which is already imported in `diag.go`.

- [ ] **Step 1: Write failing test for collectProjectMeta**

Add to `cmd/mxcli/diag_test.go`:

```go
func TestCollectProjectMeta_SkipsOnEmptyPath(t *testing.T) {
	result := collectProjectMeta("")
	if result != "" {
		t.Errorf("empty path should return empty string, got %q", result)
	}
}
```

- [ ] **Step 2: Run test to verify failure**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./cmd/mxcli/ -run "TestCollectProjectMeta" -v 2>&1 | tail -10
```

Expected: compile error `undefined: collectProjectMeta`.

- [ ] **Step 3: Implement collectProjectMeta in diag.go**

Add `"crypto/sha256"` to the import block in `diag.go` (merge with existing imports).

Note: `mmpr` is already imported as `mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"` and `runtime`, `strings`, `fmt`, `os` are already imported.

Add this function to `cmd/mxcli/diag.go`:

```go
// collectProjectMeta returns a human-readable summary of MPR project metadata.
// Returns empty string if mprPath is empty or the project cannot be opened.
func collectProjectMeta(mprPath string) string {
	if mprPath == "" {
		return ""
	}

	reader, err := mmpr.Open(mprPath)
	if err != nil {
		return fmt.Sprintf("project-meta: failed to open %s: %v\n", mprPath, err)
	}
	defer reader.Close()

	var sb strings.Builder

	mendixVersion, err := reader.GetMendixVersion()
	if err != nil {
		mendixVersion = "(unknown)"
	}
	fmt.Fprintf(&sb, "MPRPath:       %s\n", mprPath)
	fmt.Fprintf(&sb, "MendixVersion: %s\n", mendixVersion)

	mprFormat := "v1"
	if reader.ContentsDir() != "" {
		mprFormat = "v2"
	}
	fmt.Fprintf(&sb, "MPRFormat:     %s\n", mprFormat)

	modules, err := reader.ListModules()
	if err == nil {
		fmt.Fprintf(&sb, "ModuleCount:   %d\n", len(modules))
	}

	unitIDs, err := reader.ListAllUnitIDs()
	if err == nil {
		fmt.Fprintf(&sb, "DocumentCount: %d\n", len(unitIDs))
	}

	if data, err := os.ReadFile(mprPath); err == nil {
		h := sha256.New()
		h.Write(data)
		fmt.Fprintf(&sb, "MPRFileHash:   sha256:%x\n", h.Sum(nil))
	}

	return sb.String()
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./cmd/mxcli/ -run "TestCollectProjectMeta" -v 2>&1 | tail -10
```

Expected: `PASS`.

- [ ] **Step 5: Wire project-meta into runDiagBundle**

In `cmd/mxcli/diag.go`, update the diag command's `Run` function to pass the project path:

Find the `bundle` branch inside the `Run` function:

```go
		if bundle {
			runDiagBundle(logDir)
			return
		}
```

Change to:

```go
		if bundle {
			projectPath, _ := cmd.Root().PersistentFlags().GetString("project")
			runDiagBundle(logDir, projectPath)
			return
		}
```

Then update `runDiagBundle` signature to accept `mprPath string` and add the project-meta entry:

```go
// runDiagBundle creates a tar.gz archive of logs and diagnostic data.
func runDiagBundle(logDir, mprPath string) {
	timestamp := time.Now().Format("20060102-150405")
	outFile := fmt.Sprintf("mxcli-diag-%s.tar.gz", timestamp)

	f, err := os.Create(outFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating bundle: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	// system-info.txt
	info := fmt.Sprintf("Version: %s\nGo: %s %s/%s\nTime: %s\n",
		version, runtime.Version(), runtime.GOOS, runtime.GOARCH, time.Now().Format(time.RFC3339))
	addTarEntry(tw, "system-info.txt", []byte(info))

	// env-dump.txt
	addTarEntry(tw, "env-dump.txt", []byte(collectEnvDump()))

	// error-stacks.txt
	addTarEntry(tw, "error-stacks.txt", []byte(collectErrorStacks(logDir, 20)))

	// project-meta.txt (only when -p is provided)
	if meta := collectProjectMeta(mprPath); meta != "" {
		addTarEntry(tw, "project-meta.txt", []byte(meta))
	}

	// logs/ directory
	files, _ := listLogFiles(logDir)
	for _, entry := range files {
		path := filepath.Join(logDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		addTarEntry(tw, filepath.Join("logs", entry.Name()), data)
	}

	extras := []string{"system-info.txt", "env-dump.txt", "error-stacks.txt"}
	if mprPath != "" {
		extras = append(extras, "project-meta.txt")
	}
	fmt.Printf("Created: %s\n", outFile)
	fmt.Printf("Contents: %s, %d log file(s)\n", strings.Join(extras, ", "), len(files))
}
```

- [ ] **Step 6: Final build and integration test**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go build ./cmd/mxcli/ && echo "build ok"
```

Then run a full bundle smoke test:

```bash
cd /tmp
/mnt/data_sdd/gh/mxcli-wt-02/bin/mxcli diag --bundle -p /mnt/data_sdd/gh/mxcli-wt-02/testdata/expr-checker/minimal.mpr 2>&1
```

Then inspect the bundle:

```bash
tar tzf mxcli-diag-*.tar.gz
```

Expected output should include:
```
system-info.txt
env-dump.txt
error-stacks.txt
project-meta.txt
logs/mxcli-...log
```

- [ ] **Step 7: Run all tests**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./mdl/diaglog/... ./cmd/mxcli/... 2>&1 | tail -20
```

Expected: all pass.

- [ ] **Step 8: Commit**

```bash
git add cmd/mxcli/diag.go cmd/mxcli/diag_test.go
git commit -m "feat(diag): add project-meta.txt to bundle when -p is specified"
```

---

## Task 5: Update Cobra help text to document verbose flag

**Files:**
- Modify: `cmd/mxcli/main.go`

- [ ] **Step 1: Update rootCmd Long description**

In `cmd/mxcli/main.go`, find the `Long` field of `rootCmd` and add a verbose usage example. Change the Examples block to include:

```go
Long: `mxcli is a command-line interface for working with Mendix projects.

It supports MDL (Mendix Definition Language), a SQL-like syntax for
reading and modifying Mendix domain models.

Examples:
  # Get started with Claude Code in a Mendix project
  mxcli init /path/to/mendix-project; claude

  # Execute MDL commands directly
  mxcli -c "CONNECT LOCAL 'app.mpr'; SHOW ENTITIES;"

  # Connect to project and show entities
  mxcli -p app.mpr -c "SHOW ENTITIES"

  # Enable trace output for debugging
  mxcli -v -p app.mpr -c "SHOW ENTITIES"

  # Enable full debug output (JSON to stderr)
  mxcli -vv -p app.mpr -c "SHOW ENTITIES"
`,
```

Note: Remove the REPL example (`# Start interactive REPL / mxcli`) since REPL is being deleted.

- [ ] **Step 2: Update diag command Long description**

In `cmd/mxcli/diag.go`, update the `Long` field to mention the new bundle contents:

```go
Long: `Show diagnostic information and manage session log files.

Examples:
  mxcli diag              # Show version, platform, log dir, recent errors
  mxcli diag --log-path   # Print log directory path
  mxcli diag --tail 20    # Show last 20 log entries
  mxcli diag --bundle     # Create tar.gz with logs + env dump + error stacks
  mxcli diag --bundle -p app.mpr  # Also include project metadata
`,
```

- [ ] **Step 3: Build and verify help text**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go run ./cmd/mxcli/ --help 2>&1 | grep -A5 "verbose\|Flags"
```

Expected: global flags section shows `-v, --verbose count`.

- [ ] **Step 4: Commit**

```bash
git add cmd/mxcli/main.go cmd/mxcli/diag.go
git commit -m "docs(cli): update help text for -v verbose flag and enhanced bundle"
```

---

## Spec Coverage Check

| Spec Requirement | Task |
|------------------|------|
| Global -v/-vv CountP flag | Task 2 |
| diaglog verboseLevel param | Task 1 |
| Trace output (TextHandler, INFO) on -v | Task 1 |
| Debug output (JSONHandler, DEBUG) on -vv | Task 1 |
| All log methods mirrored to stderr | Task 1 |
| Remove -v flag conflicts (testRunCmd, playwright) | Task 2 |
| bundle: env-dump.txt with redacted secrets | Task 3 |
| bundle: error-stacks.txt last 20 ERROR entries | Task 3 |
| bundle: project-meta.txt with -p | Task 4 |
| Help text updated | Task 5 |
