// Package testutil provides test helpers for the expr subsystem.
package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// FindMPR returns the path to a test MPR file.
//
// Resolution order:
//  1. The environment variable named envVar, if set and the file exists.
//  2. The path repoRelPath resolved relative to the repository root
//     (directory containing go.mod), if the file exists.
//
// If neither location yields an existing file, the test is skipped via t.Skip.
func FindMPR(t *testing.T, envVar, repoRelPath string) string {
	t.Helper()

	if p := os.Getenv(envVar); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	if root, err := findRepoRoot(); err == nil {
		p := filepath.Join(root, repoRelPath)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	t.Skipf("MPR file not found: set %s or place file at %s", envVar, repoRelPath)
	return "" // unreachable
}

// FindMprContents returns the path to the mprcontents/ directory that lives
// next to the given MPR file (v2 projects).
//
// If the directory does not exist the test is skipped via t.Skip.
func FindMprContents(t *testing.T, mprPath string) string {
	t.Helper()
	dir := filepath.Join(filepath.Dir(mprPath), "mprcontents")
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("mprcontents/ directory not found next to %s", mprPath)
	}
	return dir
}

// findRepoRoot walks up from the current working directory until it finds
// a directory that contains a go.mod file.
func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", os.ErrNotExist
}
