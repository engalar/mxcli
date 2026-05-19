// SPDX-License-Identifier: Apache-2.0

//go:build linux

package goldenfs

import "github.com/hanwen/go-fuse/v2/fs"

// overlayNode is a FUSE inode that overlays the in-memory dirtyLayer
// on top of the baseDir filesystem.
// Real implementation (Lookup/Getattr/Readdir/Open/Read/Create/Write/...)
// lands in Tasks 3 and 4.
type overlayNode struct {
	fs.Inode

	baseDir string
	relPath string
	layer   *dirtyLayer
}
