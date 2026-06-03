// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// TestNoBareStdoutInCmdFiles guards against cobra RunE handlers that write to
// os.Stdout directly via fmt.Printf/Println instead of cmd.OutOrStdout().
//
// In daemon mode the launcher spawns mxcli-daemon with Stdout=nil; only output
// written through cobra's cmd.OutOrStdout() is routed back through the socket
// to the user. Bare fmt.Printf/Println calls produce no output on Windows
// (where commands always go through the daemon) and in any automated pipeline.
//
// The test uses a ratchet: each file has a known allowance of existing
// violations. Any file whose violation count EXCEEDS its allowance fails the
// test. Fixing violations is always safe (count decreases). Adding new
// violations in any file — including previously-clean files — fails the test.
//
// To raise an allowance (never the right answer) you must explicitly update the
// map below and explain why in the commit message.
func TestNoBareStdoutInCmdFiles(t *testing.T) {
	// known violations per file — ratchet down toward zero over time.
	allowances := map[string]int{
		"cmd_bson_compare.go":      5,
		"cmd_bson_dump.go":         9,
		"cmd_check.go":             5,
		"cmd_expr.go":              2,
		"cmd_expr_daemon.go":       3,
		"cmd_extract_templates.go": 11,
		"cmd_lint.go":              5,
		"cmd_new.go":               14,
		// Files not listed here have an implicit allowance of 0.
	}

	pattern := regexp.MustCompile(`\bfmt\.(Printf|Println)\b`)

	files, err := filepath.Glob("cmd_*.go")
	if err != nil {
		t.Fatalf("glob cmd_*.go: %v", err)
	}

	for _, file := range files {
		// Skip test files — they may use fmt.Printf freely.
		if len(file) > 8 && file[len(file)-8:] == "_test.go" {
			continue
		}

		data, err := os.ReadFile(file)
		if err != nil {
			t.Errorf("read %s: %v", file, err)
			continue
		}

		count := len(pattern.FindAll(bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n")), -1))
		allowed := allowances[file] // zero for unlisted files
		if count > allowed {
			t.Errorf(
				"%s: %d bare fmt.Printf/Println call(s), allowance is %d — use cmd.OutOrStdout() instead (output is lost in daemon mode)",
				file, count, allowed,
			)
		}
	}
}
