// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// daemonRatchet holds per-file violation allowances for one bad pattern.
// A file not listed has an implicit allowance of 0.
// Allowances must only decrease over time; raising one requires a commit message explaining why.
type daemonRatchet struct {
	label      string
	pattern    *regexp.Regexp
	allowances map[string]int
}

// TestNoBareStdoutInCmdFiles guards against three classes of daemon-mode bug
// in cmd_*.go handlers:
//
//  1. fmt.Printf/Println — write to os.Stdout, which is /dev/null when the
//     daemon is spawned by the launcher. Use cmd.OutOrStdout() instead.
//
//  2. fmt.Fprintf(os.Stderr, ...) — write to os.Stderr, also /dev/null.
//     Use cmd.ErrOrStderr() instead.
//
//  3. os.Exit — terminates the entire daemon process, not just the current
//     request. Subsequent commands fail because the daemon is dead.
//     Use RunE and return fmt.Errorf(...) instead.
//
// All three ratchets allow existing violations but prevent NEW ones.
func TestNoBareStdoutInCmdFiles(t *testing.T) {
	ratchets := []daemonRatchet{
		{
			label:   "bare fmt.Printf/Println (use cmd.OutOrStdout())",
			pattern: regexp.MustCompile(`\bfmt\.(Printf|Println)\b`),
			allowances: map[string]int{
				"cmd_bson_compare.go":      5,
				"cmd_bson_dump.go":         11,
				"cmd_diff.go":              5,
				"cmd_expr.go":              2,
				"cmd_expr_daemon.go":       3,
				"cmd_extract_templates.go": 11,
				"cmd_lint.go":              5,
				"cmd_new.go":               14,
			},
		},
		{
			label:   "fmt.Fprintf(os.Stderr (use cmd.ErrOrStderr())",
			pattern: regexp.MustCompile(`fmt\.Fprintf\(os\.Stderr`),
			allowances: map[string]int{
				"cmd_bson_compare.go":  9,
				"cmd_bson_discover.go": 5,
				"cmd_bson_dump.go":     11,
				"cmd_describe.go":      13,
				"cmd_diff.go":          16,
				"cmd_eval.go":          8,
				"cmd_exec.go":          4,
				"cmd_export.go":        5,
				"cmd_fmt.go":           1,
				"cmd_import.go":        5,
				"cmd_lint.go":          4,
				"cmd_mpr_pack.go":      4,
				"cmd_new.go":           10,
				"cmd_playwright.go":    3,
				"cmd_query.go":         2,
				"cmd_rename.go":        4,
				"cmd_report.go":        6,
				"cmd_sql.go":           6,
				"cmd_test_run.go":      3,
				"cmd_tui.go":           5,
				"cmd_show.go":          1,
			},
		},
		{
			label:   "os.Exit in handler (kills daemon — use RunE + return error)",
			pattern: regexp.MustCompile(`\bos\.Exit\b`),
			allowances: map[string]int{
				"cmd_bson_compare.go":  11,
				"cmd_bson_discover.go": 6,
				"cmd_bson_dump.go":     12,
				"cmd_describe.go":      14,
				"cmd_diff.go":          19,
				"cmd_eval.go":          8,
				"cmd_exec.go":          4,
				"cmd_export.go":        4,
				"cmd_import.go":        4,
				"cmd_lint.go":          7,
				"cmd_new.go":           8,
				"cmd_playwright.go":    4,
				"cmd_query.go":         9,
				"cmd_rename.go":        5,
				"cmd_report.go":        7,
				"cmd_show.go":          2,
				"cmd_sql.go":           6,
				"cmd_test_run.go":      5,
				"cmd_tui.go":           4,
			},
		},
	}

	files, err := filepath.Glob("cmd_*.go")
	if err != nil {
		t.Fatalf("glob cmd_*.go: %v", err)
	}

	for _, r := range ratchets {
		for _, file := range files {
			if len(file) > 8 && file[len(file)-8:] == "_test.go" {
				continue
			}
			data, err := os.ReadFile(file)
			if err != nil {
				t.Errorf("read %s: %v", file, err)
				continue
			}
			count := len(r.pattern.FindAll(bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n")), -1))
			allowed := r.allowances[file]
			if count > allowed {
				t.Errorf("%s: %d occurrence(s) of [%s], allowance is %d",
					file, count, r.label, allowed)
			}
		}
	}
}
