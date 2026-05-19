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

// read returns a copy of the file bytes, or nil if not in the dirty layer.
func (l *dirtyLayer) read(relPath string) []byte {
	l.mu.RLock()
	defer l.mu.RUnlock()
	b, ok := l.files[relPath]
	if !ok {
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
		if b, err := os.ReadFile(filepath.Join(baseDir, relPath)); err == nil {
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
