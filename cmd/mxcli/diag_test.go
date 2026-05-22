// SPDX-License-Identifier: Apache-2.0
package main

import (
	"fmt"
	"os"
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

func TestCollectErrorStacks_ExtractsErrorLines(t *testing.T) {
	tmpDir := t.TempDir()

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
	result := collectErrorStacks(tmpDir, 20)
	if result == "" {
		t.Error("expected non-empty result even with no logs")
	}
}

func TestCollectErrorStacks_RespectsMaxErrors(t *testing.T) {
	tmpDir := t.TempDir()

	var sb strings.Builder
	for i := 0; i < 25; i++ {
		fmt.Fprintf(&sb, "{\"time\":\"2026-05-21T10:00:%02dZ\",\"level\":\"ERROR\",\"error\":\"err%d\"}\n", i, i)
	}
	if err := os.WriteFile(tmpDir+"/mxcli-2026-05-21.log", []byte(sb.String()), 0644); err != nil {
		t.Fatal(err)
	}

	result := collectErrorStacks(tmpDir, 20)

	// 每个 entry 输出一行 "=== ... ==="，数一下有多少个分隔行
	count := strings.Count(result, "=== ")
	if count != 20 {
		t.Errorf("expected 20 error entries, got %d", count)
	}
}

func TestCollectProjectMeta_SkipsOnEmptyPath(t *testing.T) {
	result := collectProjectMeta("")
	if result != "" {
		t.Errorf("empty path should return empty string, got %q", result)
	}
}

func TestCollectProjectMeta_ReturnsErrorOnBadPath(t *testing.T) {
	result := collectProjectMeta("/nonexistent/path/to/app.mpr")
	if result == "" {
		t.Error("bad path should return non-empty error string")
	}
	if !strings.Contains(result, "failed to open") {
		t.Errorf("expected 'failed to open' in result, got %q", result)
	}
}
