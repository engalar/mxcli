// SPDX-License-Identifier: Apache-2.0

//go:build linux

package goldenfs

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

// Snapshot is a copy-on-write overlay of a base directory, mounted via FUSE.
type Snapshot struct {
	baseDir  string
	mountDir string
	server   *fuse.Server
	layer    *dirtyLayer
}

// Open mounts a FUSE overlay over baseDir.
// The caller must call Close() when done.
func Open(baseDir string) (*Snapshot, error) {
	abs, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("goldenfs: resolve baseDir: %w", err)
	}
	if _, err := os.Stat(abs); err != nil {
		return nil, fmt.Errorf("goldenfs: baseDir not found: %w", err)
	}

	mountDir, err := os.MkdirTemp("", "mxcli-golden-")
	if err != nil {
		return nil, fmt.Errorf("goldenfs: create mountDir: %w", err)
	}

	layer := newDirtyLayer()
	root := &overlayNode{baseDir: abs, relPath: "", layer: layer}

	// EntryTimeout/AttrTimeout = 0 disables kernel dentry+attr caching, so the
	// dirty layer is always consulted. Without this, a Mkdir/Create followed
	// by an immediate Stat through the same mount can race the kernel's
	// 1-second default cache and return stale ENOENT.
	zero := time.Duration(0)
	server, err := fs.Mount(mountDir, root, &fs.Options{
		MountOptions:    fuse.MountOptions{AllowOther: false},
		EntryTimeout:    &zero,
		AttrTimeout:     &zero,
		NegativeTimeout: &zero,
	})
	if err != nil {
		os.Remove(mountDir)
		return nil, fmt.Errorf("goldenfs: fuse mount: %w", err)
	}

	return &Snapshot{
		baseDir:  abs,
		mountDir: mountDir,
		server:   server,
		layer:    layer,
	}, nil
}

// MountDir returns the FUSE mount path. Pass this to mxcli exec and mx check.
func (s *Snapshot) MountDir() string { return s.mountDir }

// Commit flushes all dirty (non-WAL) files from the in-memory layer to baseDir.
func (s *Snapshot) Commit() error {
	return s.layer.commit(s.baseDir)
}

// Rollback discards all in-memory changes.
func (s *Snapshot) Rollback() { s.layer.rollback() }

// Close unmounts the FUSE filesystem and removes the mount directory.
// Does NOT commit.
func (s *Snapshot) Close() error {
	if err := s.server.Unmount(); err != nil {
		return fmt.Errorf("goldenfs: unmount: %w", err)
	}
	s.server.Wait()
	return os.Remove(s.mountDir)
}
