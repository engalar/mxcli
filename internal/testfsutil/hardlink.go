// SPDX-License-Identifier: Apache-2.0

// Package testfsutil provides filesystem helpers for test fixture isolation.
package testfsutil

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

// HardLinkDir mirrors src into dst by hard-linking regular files when
// possible, falling back to byte copying on cross-device setups (EXDEV).
//
// Hard links are O(1) — no data is transferred. They are safe for
// mprcontents/ fixture trees because the Writer never modifies existing
// .mxunit files in place; it only creates new files with fresh UUIDs
// (and updateUnit now uses atomic rename, not direct WriteFile).
//
// Falls back transparently to CopyFile on cross-device setups so tests
// work correctly regardless of where /tmp is mounted relative to testdata.
func HardLinkDir(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if linkErr := os.Link(p, target); linkErr != nil {
			if errors.Is(linkErr, syscall.EXDEV) {
				return CopyFile(p, target)
			}
			return linkErr
		}
		return nil
	})
}

// CopyFile copies src to dst using io.Copy.
func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}
