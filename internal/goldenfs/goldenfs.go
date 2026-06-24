// SPDX-License-Identifier: Apache-2.0

//go:build linux

package goldenfs

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

var (
	activeMountsMu sync.Mutex
	activeMounts   = map[string]struct{}{}
)

// cleanupOrphanMounts unmounts any stale mxcli-golden-* FUSE mounts left by
// a previously crashed process. Called automatically by Open.
func cleanupOrphanMounts() {
	entries, err := filepath.Glob(filepath.Join(os.TempDir(), "mxcli-golden-*"))
	if err != nil || len(entries) == 0 {
		return
	}
	activeMountsMu.Lock()
	snap := make(map[string]struct{}, len(activeMounts))
	for k := range activeMounts {
		snap[k] = struct{}{}
	}
	activeMountsMu.Unlock()

	mountData, _ := os.ReadFile("/proc/mounts")
	for _, dir := range entries {
		if _, live := snap[dir]; live {
			continue // owned by this process — never touch it
		}
		if !strings.Contains(string(mountData), dir) {
			os.Remove(dir)
			continue
		}
		// Stale FUSE mount from a crashed process — detach with lazy unmount.
		if err := exec.Command("fusermount", "-uz", dir).Run(); err == nil {
			os.Remove(dir)
		}
	}
}

// Open mounts a FUSE overlay over baseDir.
// The caller must call Close() when done, and should call Rollback()
// to discard writes (or Commit() to flush them to baseDir).
func Open(baseDir string, opts ...Option) (Committer, error) {
	var cfg overlayConfig
	for _, o := range opts {
		o(&cfg)
	}
	if !cfg.skipOrphanCheck {
		cleanupOrphanMounts()
	}

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

	// Register before mounting so concurrent Open calls don't mistake this
	// directory for an orphan.
	activeMountsMu.Lock()
	activeMounts[mountDir] = struct{}{}
	activeMountsMu.Unlock()

	layer := newDirtyLayer()
	over := &fuseOverlay{
		baseDir:  abs,
		mountDir: mountDir,
		layer:    layer,
	}
	root := &overlayNode{over: over, relPath: ""}

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
		activeMountsMu.Lock()
		delete(activeMounts, mountDir)
		activeMountsMu.Unlock()
		os.Remove(mountDir)
		return nil, fmt.Errorf("goldenfs: fuse mount: %w", err)
	}

	over.server = server
	return over, nil
}
