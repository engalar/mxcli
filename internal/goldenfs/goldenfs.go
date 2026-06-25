// SPDX-License-Identifier: Apache-2.0

//go:build linux

package goldenfs

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
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

var (
	activeMountsMu sync.Mutex
	activeMounts   = map[string]struct{}{}
	cleanupOnce    sync.Once
	// mountMu serialises fs.Mount calls to prevent go-fuse inode tree races.
	// Unlike mountSem (which limited concurrency), this serialises only the
	// Mount syscall — once the mount is created, the lock is released and tests
	// proceed fully in parallel.
	mountMu sync.Mutex
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
// The caller must call Close() when done.
func Open(baseDir string) (*Snapshot, error) {
	cleanupOnce.Do(cleanupOrphanMounts)
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
	root := &overlayNode{baseDir: abs, relPath: "", layer: layer}

	// Serialise fs.Mount to prevent go-fuse inode tree races on concurrent
	// mount creation. The lock is released immediately after Mount returns.
	mountMu.Lock()
	zero := time.Duration(0)
	server, err := fs.Mount(mountDir, root, &fs.Options{
		MountOptions:    fuse.MountOptions{AllowOther: false},
		EntryTimeout:    &zero,
		AttrTimeout:     &zero,
		NegativeTimeout: &zero,
	})
	mountMu.Unlock()
	if err != nil {
		activeMountsMu.Lock()
		delete(activeMounts, mountDir)
		activeMountsMu.Unlock()
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
	activeMountsMu.Lock()
	delete(activeMounts, s.mountDir)
	activeMountsMu.Unlock()

	if err := s.server.Unmount(); err != nil {
		// Fallback: lazy detach via fusermount so the mount dir can be removed.
		exec.Command("fusermount", "-uz", s.mountDir).Run() //nolint:errcheck
	} else {
		s.server.Wait()
	}
	return os.Remove(s.mountDir)
}

// DirtyPaths returns the relative paths of all files written or deleted
// through the overlay since Open, sorted lexicographically. Sentinel files
// (e.g. Mkdir markers) are excluded. Useful for test write-path audits.
func (s *Snapshot) DirtyPaths() []string {
	s.layer.mu.RLock()
	defer s.layer.mu.RUnlock()
	paths := make([]string, 0, len(s.layer.files))
	for k := range s.layer.files {
		if !isSentinel(k) {
			paths = append(paths, k)
		}
	}
	sort.Strings(paths)
	return paths
}

// ReadDirtyFile returns the current in-memory bytes for a file written
// through the overlay. Returns nil if the path was not written or was deleted.
func (s *Snapshot) ReadDirtyFile(relPath string) []byte {
	return s.layer.read(relPath)
}
