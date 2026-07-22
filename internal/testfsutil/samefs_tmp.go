// SPDX-License-Identifier: Apache-2.0

package testfsutil

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

var findRootOnce sync.Once
var repoRoot string

// findRepoRoot walks up from the current working directory to find go.mod.
// Results are cached after the first call.
func findRepoRoot(t *testing.T) string {
	t.Helper()
	findRootOnce.Do(func() {
		pkgDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("getwd: %v", err)
		}
		root := pkgDir
		for {
			if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
				break
			}
			parent := filepath.Dir(root)
			if parent == root {
				t.Fatal("cannot find repo root (go.mod not found)")
			}
			root = parent
		}
		repoRoot = root
	})
	return repoRoot
}

// SameFSTempDir creates a temp subdirectory on the same filesystem as the
// repo root (where testdata lives). Hard links from testdata/ into this
// directory succeed (no EXDEV), avoiding the byte-copy fallback.
//
// The subdirectory is automatically removed when the test finishes.
func SameFSTempDir(t *testing.T) string {
	t.Helper()
	root := findRepoRoot(t)
	d, err := os.MkdirTemp(root, ".mxcli-tmp-*")
	if err != nil {
		t.Fatalf("create same-fs tmp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(d) })
	return d
}
