// SPDX-License-Identifier: Apache-2.0

//go:build linux

package goldenfs

import (
	"os"
	"os/exec"
	"sort"
	"sync"

	"github.com/hanwen/go-fuse/v2/fuse"
)

// fuseOverlay is a copy-on-write overlay mounted via FUSE.
// Implements Committer (and therefore DirtyReader and Overlay).
type fuseOverlay struct {
	baseDir  string
	mountDir string
	server   *fuse.Server
	layer    *dirtyLayer

	// closeMu serialises Close so concurrent calls (e.g. from
	// deferred Cleanup + explicit teardown) don't panic on
	// double-unmount.
	closeMu sync.Mutex
}

var _ Committer = (*fuseOverlay)(nil)

// MountDir returns the FUSE mount path. Pass this to mxcli exec and mx check.
func (s *fuseOverlay) MountDir() string { return s.mountDir }

// Commit flushes all dirty (non-WAL) files from the in-memory layer to baseDir.
func (s *fuseOverlay) Commit() error {
	return s.layer.commit(s.baseDir)
}

// Rollback discards all in-memory changes.
func (s *fuseOverlay) Rollback() { s.layer.rollback() }

// Close unmounts the FUSE filesystem and removes the mount directory.
// Does NOT commit.
func (s *fuseOverlay) Close() error {
	s.closeMu.Lock()
	defer s.closeMu.Unlock()

	if s.server == nil {
		return nil // already closed
	}

	activeMountsMu.Lock()
	delete(activeMounts, s.mountDir)
	activeMountsMu.Unlock()

	if err := s.server.Unmount(); err != nil {
		// Fallback: lazy detach via fusermount so the mount dir can be removed.
		exec.Command("fusermount", "-uz", s.mountDir).Run() //nolint:errcheck
	} else {
		s.server.Wait()
	}
	s.server = nil
	return os.Remove(s.mountDir)
}

// DirtyPaths returns the relative paths of all files written or deleted
// through the overlay since Open, sorted lexicographically. Sentinel files
// (e.g. Mkdir markers) are excluded. Useful for test write-path audits.
func (s *fuseOverlay) DirtyPaths() []string {
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
func (s *fuseOverlay) ReadDirtyFile(relPath string) []byte {
	return s.layer.read(relPath)
}
