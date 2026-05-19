// SPDX-License-Identifier: Apache-2.0

//go:build linux

package goldenfs

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// dirtyLayer is a copy-on-write in-memory file store.
// keys are slash-separated relative paths (e.g. "mprcontents/00/39/uuid.mxunit").
// A nil value is a tombstone (deleted).
type dirtyLayer struct {
	mu    sync.RWMutex
	files map[string][]byte
}

func newDirtyLayer() *dirtyLayer {
	return &dirtyLayer{files: make(map[string][]byte)}
}

// read returns a copy of the file bytes, or nil if not in the dirty layer
// or tombstoned (deleted).
func (l *dirtyLayer) read(relPath string) []byte {
	l.mu.RLock()
	defer l.mu.RUnlock()
	b, ok := l.files[relPath]
	if !ok || b == nil {
		return nil
	}
	out := make([]byte, len(b))
	copy(out, b)
	return out
}

// has returns true if relPath is present in the dirty layer (even as tombstone).
func (l *dirtyLayer) has(relPath string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	_, ok := l.files[relPath]
	return ok
}

// isDeleted returns true if relPath has a tombstone.
func (l *dirtyLayer) isDeleted(relPath string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	b, ok := l.files[relPath]
	return ok && b == nil
}

// hasDirtyDir returns true if any non-tombstone dirty file key is a descendant
// of relPath, i.e. relPath looks like a directory created in (or under) the
// dirty layer. Used so Lookup/Getattr/Readdir can recognise Mkdir-created
// directories that aren't on disk yet.
func (l *dirtyLayer) hasDirtyDir(relPath string) bool {
	if relPath == "" {
		return false
	}
	prefix := relPath + "/"
	l.mu.RLock()
	defer l.mu.RUnlock()
	for k, v := range l.files {
		if v == nil {
			continue // tombstone
		}
		if strings.HasPrefix(k, prefix) {
			return true
		}
	}
	return false
}

// write stores data at offset into the dirty layer for relPath.
// If relPath is not yet in the layer the existing bytes are set to the written
// data (no copy-up; caller has already loaded the content when needed).
func (l *dirtyLayer) write(relPath string, offset int64, data []byte) {
	l.mu.Lock()
	defer l.mu.Unlock()
	cur := l.files[relPath] // may be nil (new file)
	end := offset + int64(len(data))
	if end > int64(len(cur)) {
		grown := make([]byte, end)
		copy(grown, cur)
		cur = grown
	}
	copy(cur[offset:], data)
	l.files[relPath] = cur
}

// writeWithBase is like write but performs a copy-up from baseDir if relPath is
// not yet in the dirty layer. Used for partial writes (e.g. SQLite pages).
func (l *dirtyLayer) writeWithBase(relPath, baseDir string, offset int64, data []byte) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, ok := l.files[relPath]; !ok {
		// copy-up
		// NOTE: disk read happens under write lock. This is intentional: for a
		// single-writer SQLite session through one FUSE mount, the I/O is bounded
		// and the simpler locking model avoids a double-check pattern.
		if b, err := os.ReadFile(filepath.Join(baseDir, filepath.FromSlash(relPath))); err == nil {
			l.files[relPath] = b
		}
	}
	cur := l.files[relPath]
	end := offset + int64(len(data))
	if end > int64(len(cur)) {
		grown := make([]byte, end)
		copy(grown, cur)
		cur = grown
	}
	copy(cur[offset:], data)
	l.files[relPath] = cur
}

// delete sets a tombstone for relPath.
func (l *dirtyLayer) delete(relPath string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.files[relPath] = nil
}

// truncate resizes the dirty-layer entry to `size` bytes. If the entry isn't
// in the dirty layer yet, it copies up from `baseDir` first. Result is always
// a non-nil byte slice (never a tombstone), even at size 0.
func (l *dirtyLayer) truncate(relPath, baseDir string, size int64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	cur, ok := l.files[relPath]
	if !ok || cur == nil {
		// copy-up if base file exists; otherwise treat as empty.
		if b, err := os.ReadFile(filepath.Join(baseDir, filepath.FromSlash(relPath))); err == nil {
			cur = b
		} else {
			cur = nil
		}
	}
	resized := make([]byte, size)
	if n := int64(len(cur)); n > 0 {
		if size < n {
			n = size
		}
		copy(resized, cur[:n])
	}
	l.files[relPath] = resized
}

// rollback discards all dirty state.
func (l *dirtyLayer) rollback() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.files = make(map[string][]byte)
}

// commit writes all non-WAL dirty files to baseDir.
func (l *dirtyLayer) commit(baseDir string) error {
	l.mu.RLock()
	defer l.mu.RUnlock()
	for rel, content := range l.files {
		if isSQLiteAux(rel) {
			continue
		}
		dest := filepath.Join(baseDir, filepath.FromSlash(rel))
		if content == nil {
			_ = os.Remove(dest)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(dest, content, 0644); err != nil {
			return err
		}
	}
	return nil
}

// isSQLiteAux returns true for SQLite WAL and SHM auxiliary files.
func isSQLiteAux(relPath string) bool {
	return strings.HasSuffix(relPath, "-wal") || strings.HasSuffix(relPath, "-shm")
}
