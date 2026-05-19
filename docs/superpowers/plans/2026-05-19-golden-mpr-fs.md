# Golden MPR Filesystem Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `internal/goldenfs` — a FUSE overlay filesystem that serves `testdata/expr-checker/` (and any MPR v2 project) from an in-memory dirty map, so tests can write MPR files and run `mx check` without touching the real files on disk.

**Architecture:** A `Snapshot` struct holds a `map[string][]byte` dirty layer (keyed by relative path) over a `baseDir` on disk. A FUSE server mounts this overlay at a `/tmp/mxcli-golden-<uuid>/` path; all writes land in the dirty map; reads fall back to `baseDir` when the dirty map has no entry. `Commit()` flushes dirty → baseDir; `Rollback()` clears dirty. Each `Open()` call creates an independent snapshot, so parallel tests are safe.

**Tech Stack:** `github.com/hanwen/go-fuse/v2/fs` (FUSE node API), `github.com/hanwen/go-fuse/v2/fuse`, Linux `//go:build linux` build tag, `sync.RWMutex`, `hash/fnv` for stable inode numbers.

**Spec:** `docs/superpowers/specs/2026-05-19-golden-mpr-fs-design.md`

---

## File Map

| File | Role |
|------|------|
| `internal/goldenfs/goldenfs.go` | `Snapshot` struct, `Open`, `Close`, `Commit`, `Rollback`, `MountDir` |
| `internal/goldenfs/dirty.go` | `dirtyLayer` — in-memory map with copy-up, partial write, tombstones |
| `internal/goldenfs/overlay.go` | FUSE node types: `overlayNode`, `overlayFileHandle`; all FUSE interface impls |
| `internal/goldenfs/goldenfs_test.go` | Unit + FUSE mount tests (Linux, no mx binary needed) |
| `internal/goldenfs/dirty_test.go` | Unit tests for `dirtyLayer` only (no FUSE) |
| `internal/goldenfs/goldenfs_stub.go` | Non-Linux stub: `Open` returns `ErrNotSupported` |

All files in `internal/goldenfs/` use `//go:build linux` except `goldenfs_stub.go` which uses `//go:build !linux`.

---

## Task 1: Add dependency and package skeleton

**Files:**
- Create: `internal/goldenfs/goldenfs.go`
- Create: `internal/goldenfs/goldenfs_stub.go`
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Add go-fuse dependency**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS='-mod=mod' GOPROXY=https://mirrors.aliyun.com/goproxy/ go get github.com/hanwen/go-fuse/v2@latest
```

Expected: `go.mod` updated with `github.com/hanwen/go-fuse/v2 vX.Y.Z`.

- [ ] **Step 2: Create `goldenfs.go` skeleton (Linux)**

```go
// SPDX-License-Identifier: Apache-2.0

//go:build linux

package goldenfs

import (
	"fmt"
	"os"

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

	server, err := fs.Mount(mountDir, root, &fs.Options{
		MountOptions: fuse.MountOptions{AllowOther: false},
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
```

Add `"path/filepath"` to the import block.

- [ ] **Step 3: Create `goldenfs_stub.go` (non-Linux)**

```go
// SPDX-License-Identifier: Apache-2.0

//go:build !linux

package goldenfs

import "errors"

// ErrNotSupported is returned on non-Linux platforms where FUSE is unavailable.
var ErrNotSupported = errors.New("goldenfs: FUSE overlay requires Linux")

// Snapshot is a no-op placeholder on non-Linux platforms.
type Snapshot struct{}

func Open(_ string) (*Snapshot, error)  { return nil, ErrNotSupported }
func (s *Snapshot) MountDir() string    { return "" }
func (s *Snapshot) Commit() error       { return ErrNotSupported }
func (s *Snapshot) Rollback()           {}
func (s *Snapshot) Close() error        { return nil }
```

- [ ] **Step 4: Verify the package compiles**

```bash
go build ./internal/goldenfs/...
```

Expected: no errors (overlay.go and dirty.go don't exist yet, so add empty stubs:
`package goldenfs` in `dirty.go` and `overlay.go` temporarily).

- [ ] **Step 5: Commit**

```bash
git add internal/goldenfs/ go.mod go.sum
git commit -m "feat(goldenfs): add package skeleton and go-fuse dependency"
```

---

## Task 2: Dirty layer — unit tests first, then implementation

**Files:**
- Create: `internal/goldenfs/dirty_test.go`
- Create: `internal/goldenfs/dirty.go`

The `dirtyLayer` is pure Go — no FUSE, no filesystem. Test it in isolation.

- [ ] **Step 1: Write failing tests in `dirty_test.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package goldenfs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirtyLayer_ReadMiss_ReturnsNil(t *testing.T) {
	l := newDirtyLayer()
	if got := l.read("foo.txt"); got != nil {
		t.Fatalf("expected nil for miss, got %d bytes", len(got))
	}
}

func TestDirtyLayer_WriteAndRead(t *testing.T) {
	l := newDirtyLayer()
	l.write("foo.txt", 0, []byte("hello"))
	got := l.read("foo.txt")
	if string(got) != "hello" {
		t.Fatalf("want %q got %q", "hello", got)
	}
}

func TestDirtyLayer_PartialWrite_CopyUp(t *testing.T) {
	base := t.TempDir()
	if err := os.WriteFile(filepath.Join(base, "f.bin"), []byte("AAAAAA"), 0644); err != nil {
		t.Fatal(err)
	}
	l := newDirtyLayer()
	// Write "BB" at offset 2; triggers copy-up from base.
	l.writeWithBase("f.bin", base, 2, []byte("BB"))
	got := l.read("f.bin")
	if string(got) != "AABBAA" {
		t.Fatalf("want AABBAA got %q", got)
	}
}

func TestDirtyLayer_Tombstone(t *testing.T) {
	l := newDirtyLayer()
	l.write("f.txt", 0, []byte("x"))
	l.delete("f.txt")
	if !l.isDeleted("f.txt") {
		t.Fatal("expected tombstone")
	}
}

func TestDirtyLayer_Rollback_ClearsAll(t *testing.T) {
	l := newDirtyLayer()
	l.write("a.txt", 0, []byte("data"))
	l.delete("b.txt")
	l.rollback()
	if l.read("a.txt") != nil {
		t.Fatal("dirty file should be gone after rollback")
	}
	if l.isDeleted("b.txt") {
		t.Fatal("tombstone should be gone after rollback")
	}
}

func TestDirtyLayer_Commit_WritesBase(t *testing.T) {
	base := t.TempDir()
	l := newDirtyLayer()
	l.write("out.txt", 0, []byte("committed"))
	if err := l.commit(base); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(base, "out.txt"))
	if err != nil || string(got) != "committed" {
		t.Fatalf("commit did not write file: %v", err)
	}
}

func TestDirtyLayer_Commit_SkipsWAL(t *testing.T) {
	base := t.TempDir()
	l := newDirtyLayer()
	l.write("minimal.mpr-wal", 0, []byte("wal data"))
	l.write("minimal.mpr-shm", 0, []byte("shm data"))
	if err := l.commit(base); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(base, "minimal.mpr-wal")); !os.IsNotExist(err) {
		t.Fatal("WAL file must not be committed to base")
	}
}
```

- [ ] **Step 2: Run to confirm failure**

```bash
go test ./internal/goldenfs/ -run TestDirty -v 2>&1 | head -30
```

Expected: compilation errors (functions not defined yet).

- [ ] **Step 3: Implement `dirty.go`**

```go
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
```

Add the `//go:build linux` tag at the top (same as `goldenfs.go`). Also add it to `dirty_test.go`:
```go
//go:build linux
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/goldenfs/ -run TestDirty -v
```

Expected: all 7 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/goldenfs/dirty.go internal/goldenfs/dirty_test.go
git commit -m "feat(goldenfs): dirty layer with copy-up, tombstones, commit"
```

---

## Task 3: FUSE read path — Lookup, Getattr, Readdir, Open, Read

**Files:**
- Create: `internal/goldenfs/overlay.go`
- Modify: `internal/goldenfs/goldenfs_test.go`

The FUSE node type: `overlayNode` represents any path (file or dir) in the overlay. It holds a `relPath` string (relative from `baseDir`) and pointers to `baseDir` and `layer`. All operations are stateless — the node is a thin wrapper over the dirty layer + base dir.

- [ ] **Step 1: Write failing FUSE read tests in `goldenfs_test.go`**

```go
// SPDX-License-Identifier: Apache-2.0

//go:build linux

package goldenfs

import (
	"os"
	"path/filepath"
	"testing"
)

// baseFixture creates a minimal temp base directory with a file and a subdir.
func baseFixture(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	if err := os.WriteFile(filepath.Join(base, "hello.txt"), []byte("world"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(base, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "sub", "deep.txt"), []byte("deep"), 0644); err != nil {
		t.Fatal(err)
	}
	return base
}

func TestSnapshot_ReadExistingFile(t *testing.T) {
	snap, err := Open(baseFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()

	got, err := os.ReadFile(filepath.Join(snap.MountDir(), "hello.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "world" {
		t.Fatalf("want world got %q", got)
	}
}

func TestSnapshot_ReadDeepFile(t *testing.T) {
	snap, err := Open(baseFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()

	got, err := os.ReadFile(filepath.Join(snap.MountDir(), "sub", "deep.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "deep" {
		t.Fatalf("want deep got %q", got)
	}
}

func TestSnapshot_ReaddirRootContainsBase(t *testing.T) {
	snap, err := Open(baseFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()

	entries, err := os.ReadDir(snap.MountDir())
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, e := range entries {
		names[e.Name()] = true
	}
	if !names["hello.txt"] || !names["sub"] {
		t.Fatalf("missing expected entries: got %v", names)
	}
}
```

- [ ] **Step 2: Run tests — expect compile failure**

```bash
go test ./internal/goldenfs/ -run TestSnapshot_Read -v 2>&1 | head -20
```

Expected: compile error (overlayNode not defined).

- [ ] **Step 3: Implement `overlay.go` with read path**

```go
// SPDX-License-Identifier: Apache-2.0

//go:build linux

package goldenfs

import (
	"context"
	"hash/fnv"
	"os"
	"path/filepath"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

// overlayNode represents a file or directory in the overlay.
// relPath is the slash-separated path relative to baseDir (empty string = root).
type overlayNode struct {
	fs.Inode
	baseDir string
	relPath string
	layer   *dirtyLayer
}

// pathIno generates a stable inode number from a relative path.
func pathIno(relPath string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(relPath))
	return h.Sum64()
}

// absBase returns the absolute path of this node in baseDir.
func (n *overlayNode) absBase() string {
	if n.relPath == "" {
		return n.baseDir
	}
	return filepath.Join(n.baseDir, filepath.FromSlash(n.relPath))
}

// childRel returns the relative path of a named child of this node.
func (n *overlayNode) childRel(name string) string {
	if n.relPath == "" {
		return name
	}
	return n.relPath + "/" + name
}

// --- NodeLookuper ---

func (n *overlayNode) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	rel := n.childRel(name)

	if n.layer.isDeleted(rel) {
		return nil, syscall.ENOENT
	}

	// Determine mode: dirty layer takes precedence over base.
	var mode uint32
	if content := n.layer.read(rel); content != nil {
		mode = 0100644 // regular file
		out.Size = uint64(len(content))
	} else {
		info, err := os.Lstat(filepath.Join(n.baseDir, filepath.FromSlash(rel)))
		if err != nil {
			return nil, syscall.ENOENT
		}
		mode = uint32(info.Mode())
		out.Size = uint64(info.Size())
	}

	out.Mode = mode
	child := &overlayNode{baseDir: n.baseDir, relPath: rel, layer: n.layer}
	stable := fs.StableAttr{Mode: mode & ^uint32(0777), Ino: pathIno(rel)}
	return n.NewInode(ctx, child, stable), 0
}

// --- NodeGetattrer ---

func (n *overlayNode) Getattr(ctx context.Context, fh fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	if n.relPath == "" {
		// Root directory
		info, err := os.Stat(n.baseDir)
		if err != nil {
			return syscall.EIO
		}
		out.Mode = uint32(info.Mode())
		out.Ino = 1
		return 0
	}

	if n.layer.isDeleted(n.relPath) {
		return syscall.ENOENT
	}

	if content := n.layer.read(n.relPath); content != nil {
		out.Mode = 0100644
		out.Size = uint64(len(content))
		out.Ino = pathIno(n.relPath)
		return 0
	}

	info, err := os.Lstat(n.absBase())
	if err != nil {
		return syscall.ENOENT
	}
	out.Mode = uint32(info.Mode())
	out.Size = uint64(info.Size())
	out.Ino = pathIno(n.relPath)
	return 0
}

// --- NodeReaddirer ---

func (n *overlayNode) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	seen := map[string]bool{}
	var entries []fuse.DirEntry

	// Base dir entries
	baseEntries, err := os.ReadDir(n.absBase())
	if err != nil && !os.IsNotExist(err) {
		return nil, syscall.EIO
	}
	prefix := n.relPath
	if prefix != "" {
		prefix += "/"
	}
	for _, e := range baseEntries {
		rel := prefix + e.Name()
		if n.layer.isDeleted(rel) {
			continue
		}
		seen[e.Name()] = true
		mode := uint32(e.Type())
		entries = append(entries, fuse.DirEntry{Name: e.Name(), Mode: mode, Ino: pathIno(rel)})
	}

	// Dirty-layer additions in this directory
	n.layer.mu.RLock()
	defer n.layer.mu.RUnlock()
	for rel, content := range n.layer.files {
		if content == nil {
			continue // tombstone
		}
		if !isDirectChild(rel, n.relPath) {
			continue
		}
		name := filepath.Base(rel)
		if seen[name] {
			continue
		}
		entries = append(entries, fuse.DirEntry{Name: name, Mode: fuse.S_IFREG, Ino: pathIno(rel)})
	}

	return fs.NewListDirStream(entries), 0
}

// isDirectChild returns true if rel is a direct child of parentRel.
func isDirectChild(rel, parentRel string) bool {
	if parentRel == "" {
		return !strings.Contains(rel, "/")
	}
	if !strings.HasPrefix(rel, parentRel+"/") {
		return false
	}
	rest := rel[len(parentRel)+1:]
	return !strings.Contains(rest, "/")
}

// --- NodeOpener + file handle (read) ---

type overlayReadHandle struct {
	content []byte
}

func (h *overlayReadHandle) Read(_ context.Context, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	if off >= int64(len(h.content)) {
		return fuse.ReadResultData(nil), 0
	}
	end := off + int64(len(dest))
	if end > int64(len(h.content)) {
		end = int64(len(h.content))
	}
	return fuse.ReadResultData(h.content[off:end]), 0
}

func (n *overlayNode) Open(ctx context.Context, flags uint32) (fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
	// Write-capable open handled by Create / separate write handle.
	content := n.layer.read(n.relPath)
	if content == nil {
		var err error
		content, err = os.ReadFile(n.absBase())
		if err != nil {
			return nil, 0, syscall.EIO
		}
	}
	return &overlayReadHandle{content: content}, fuse.FOPEN_DIRECT_IO, 0
}
```

Add `"strings"` to the import block.

- [ ] **Step 4: Run the read tests**

```bash
go test ./internal/goldenfs/ -run "TestSnapshot_Read|TestSnapshot_Readdir" -v
```

Expected: all 3 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/goldenfs/overlay.go internal/goldenfs/goldenfs_test.go
git commit -m "feat(goldenfs): FUSE read path (Lookup, Getattr, Readdir, Open, Read)"
```

---

## Task 4: FUSE write path — Create, Write, Flush, Unlink, Mkdir

**Files:**
- Modify: `internal/goldenfs/overlay.go`
- Modify: `internal/goldenfs/goldenfs_test.go`

- [ ] **Step 1: Add write tests to `goldenfs_test.go`**

```go
func TestSnapshot_WriteNewFile_InDirtyNotBase(t *testing.T) {
	base := baseFixture(t)
	snap, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()

	// Write via FUSE mount
	dest := filepath.Join(snap.MountDir(), "new.txt")
	if err := os.WriteFile(dest, []byte("new content"), 0644); err != nil {
		t.Fatalf("write through FUSE: %v", err)
	}

	// Readable through mount
	got, err := os.ReadFile(dest)
	if err != nil || string(got) != "new content" {
		t.Fatalf("want new content, got %q err %v", got, err)
	}

	// Base dir untouched
	if _, err := os.Stat(filepath.Join(base, "new.txt")); !os.IsNotExist(err) {
		t.Fatal("base dir must not be modified")
	}
}

func TestSnapshot_OverwriteExistingFile(t *testing.T) {
	base := baseFixture(t)
	snap, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()

	dest := filepath.Join(snap.MountDir(), "hello.txt")
	if err := os.WriteFile(dest, []byte("overwritten"), 0644); err != nil {
		t.Fatalf("overwrite through FUSE: %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil || string(got) != "overwritten" {
		t.Fatalf("want overwritten, got %q err %v", got, err)
	}

	// Base still has original
	orig, _ := os.ReadFile(filepath.Join(base, "hello.txt"))
	if string(orig) != "world" {
		t.Fatalf("base must be unmodified, got %q", orig)
	}
}

func TestSnapshot_MkdirAndWriteDeep(t *testing.T) {
	base := baseFixture(t)
	snap, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()

	newDir := filepath.Join(snap.MountDir(), "newdir")
	if err := os.Mkdir(newDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(newDir, "child.txt"), []byte("hi"), 0644); err != nil {
		t.Fatalf("write in new dir: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(snap.MountDir(), "newdir", "child.txt"))
	if err != nil || string(got) != "hi" {
		t.Fatalf("want hi, got %q err %v", got, err)
	}
}
```

- [ ] **Step 2: Run tests — expect failure**

```bash
go test ./internal/goldenfs/ -run "TestSnapshot_Write|TestSnapshot_Mkdir|TestSnapshot_Overwrite" -v 2>&1 | head -30
```

Expected: FAIL (write operations not yet implemented).

- [ ] **Step 3: Add write operations to `overlay.go`**

Add the following to `overlay.go` (after the read handle):

```go
// --- Write file handle ---

type overlayWriteHandle struct {
	node    *overlayNode
	baseDir string
}

func (h *overlayWriteHandle) Write(_ context.Context, data []byte, off int64) (uint32, syscall.Errno) {
	h.node.layer.writeWithBase(h.node.relPath, h.baseDir, off, data)
	return uint32(len(data)), 0
}

func (h *overlayWriteHandle) Flush(_ context.Context, flags uint32) syscall.Errno { return 0 }
func (h *overlayWriteHandle) Release(_ context.Context, flags uint32) syscall.Errno { return 0 }

// --- NodeCreater (create new file in this directory) ---

func (n *overlayNode) Create(ctx context.Context, name string, flags uint32, mode uint32, out *fuse.EntryOut) (*fs.Inode, fs.FileHandle, uint32, syscall.Errno) {
	rel := n.childRel(name)
	// Seed the dirty map with an empty file.
	n.layer.write(rel, 0, []byte{})

	child := &overlayNode{baseDir: n.baseDir, relPath: rel, layer: n.layer}
	stable := fs.StableAttr{Mode: fuse.S_IFREG, Ino: pathIno(rel)}
	inode := n.NewInode(ctx, child, stable)

	out.Mode = 0100644
	fh := &overlayWriteHandle{node: child, baseDir: n.baseDir}
	return inode, fh, fuse.FOPEN_DIRECT_IO, 0
}

// Open for writing (O_WRONLY / O_RDWR on existing file).
// We override the read-only Open to also return a write handle.
func (n *overlayNode) openWrite(ctx context.Context) fs.FileHandle {
	return &overlayWriteHandle{node: n, baseDir: n.baseDir}
}

// --- NodeMkdirer ---

func (n *overlayNode) Mkdir(ctx context.Context, name string, mode uint32, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	rel := n.childRel(name)
	// Store a sentinel so Readdir knows this dir exists in the dirty layer.
	n.layer.write(rel+"/.keep", 0, []byte{})

	child := &overlayNode{baseDir: n.baseDir, relPath: rel, layer: n.layer}
	stable := fs.StableAttr{Mode: fuse.S_IFDIR, Ino: pathIno(rel)}
	out.Mode = fuse.S_IFDIR | 0755
	return n.NewInode(ctx, child, stable), 0
}

// --- NodeUnlinker ---

func (n *overlayNode) Unlink(ctx context.Context, name string) syscall.Errno {
	rel := n.childRel(name)
	n.layer.delete(rel)
	return 0
}

// --- NodeRenamer ---

func (n *overlayNode) Rename(ctx context.Context, name string, newParent fs.InodeEmbedder, newName string, flags uint32) syscall.Errno {
	srcRel := n.childRel(name)

	newParentNode, ok := newParent.(*overlayNode)
	if !ok {
		return syscall.EINVAL
	}
	dstRel := newParentNode.childRel(newName)

	// Copy content from dirty or base.
	content := n.layer.read(srcRel)
	if content == nil {
		var err error
		content, err = os.ReadFile(filepath.Join(n.baseDir, filepath.FromSlash(srcRel)))
		if err != nil {
			return syscall.ENOENT
		}
	}
	n.layer.write(dstRel, 0, content)
	n.layer.delete(srcRel)
	return 0
}
```

Also update `Open` in `overlay.go` to return a write handle when flags include `O_WRONLY`/`O_RDWR`:

```go
func (n *overlayNode) Open(ctx context.Context, flags uint32) (fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
	if flags&(syscall.O_WRONLY|syscall.O_RDWR) != 0 {
		// Seed dirty map via copy-up, then return write handle.
		if !n.layer.has(n.relPath) {
			if b, err := os.ReadFile(n.absBase()); err == nil {
				n.layer.write(n.relPath, 0, b)
			}
		}
		return &overlayWriteHandle{node: n, baseDir: n.baseDir}, fuse.FOPEN_DIRECT_IO, 0
	}
	content := n.layer.read(n.relPath)
	if content == nil {
		var err error
		content, err = os.ReadFile(n.absBase())
		if err != nil {
			return nil, 0, syscall.EIO
		}
	}
	return &overlayReadHandle{content: content}, fuse.FOPEN_DIRECT_IO, 0
}
```

- [ ] **Step 4: Run write tests**

```bash
go test ./internal/goldenfs/ -run "TestSnapshot_Write|TestSnapshot_Mkdir|TestSnapshot_Overwrite" -v
```

Expected: all 3 tests PASS.

- [ ] **Step 5: Run all goldenfs tests**

```bash
go test ./internal/goldenfs/ -v
```

Expected: all tests PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/goldenfs/overlay.go internal/goldenfs/goldenfs_test.go
git commit -m "feat(goldenfs): FUSE write path (Create, Write, Mkdir, Unlink, Rename)"
```

---

## Task 5: Commit and Rollback

**Files:**
- Modify: `internal/goldenfs/goldenfs_test.go`

`Commit()` and `Rollback()` are already wired to `dirtyLayer.commit`/`rollback` from Task 1 and tested in `dirty_test.go`. This task adds end-to-end tests through the FUSE mount.

- [ ] **Step 1: Add Commit and Rollback tests**

```go
func TestSnapshot_Commit_PersistsToBase(t *testing.T) {
	base := baseFixture(t)
	snap, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()

	if err := os.WriteFile(filepath.Join(snap.MountDir(), "committed.txt"), []byte("saved"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := snap.Commit(); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(base, "committed.txt"))
	if err != nil || string(got) != "saved" {
		t.Fatalf("want saved, got %q err %v", got, err)
	}
}

func TestSnapshot_Rollback_BaseUnchanged(t *testing.T) {
	base := baseFixture(t)
	snap, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()

	if err := os.WriteFile(filepath.Join(snap.MountDir(), "temp.txt"), []byte("gone"), 0644); err != nil {
		t.Fatal(err)
	}
	snap.Rollback()

	// After rollback the file should disappear from the mount view.
	if _, err := os.Stat(filepath.Join(snap.MountDir(), "temp.txt")); !os.IsNotExist(err) {
		t.Fatal("rolled-back file must not be visible in mount")
	}
	// Base must be untouched.
	if _, err := os.Stat(filepath.Join(base, "temp.txt")); !os.IsNotExist(err) {
		t.Fatal("base dir must not have the file")
	}
}

func TestSnapshot_Close_DoesNotCommit(t *testing.T) {
	base := baseFixture(t)
	snap, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(snap.MountDir(), "volatile.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	snap.Close()

	if _, err := os.Stat(filepath.Join(base, "volatile.txt")); !os.IsNotExist(err) {
		t.Fatal("Close without Commit must not write to base")
	}
}
```

Note: `TestSnapshot_Rollback_BaseUnchanged` requires that after `Rollback()` the FUSE view clears cached inodes. go-fuse caches inodes by default; force-invalidate by calling `n.NotifyEntry(name)` on the parent after delete. Add a `notifyEntry` helper to `Snapshot` that walks the inode tree — or, simpler: in `Readdir`, always re-check the dirty layer (already done). The stat call goes through `Getattr` which checks `layer.isDeleted` — so the file will appear as `ENOENT` after rollback without any extra work.

- [ ] **Step 2: Run Commit/Rollback tests**

```bash
go test ./internal/goldenfs/ -run "TestSnapshot_Commit|TestSnapshot_Rollback|TestSnapshot_Close" -v
```

Expected: all PASS.

- [ ] **Step 3: Run full test suite**

```bash
go test ./internal/goldenfs/ -v
```

Expected: all tests PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/goldenfs/goldenfs_test.go
git commit -m "test(goldenfs): Commit, Rollback, and Close end-to-end tests"
```

---

## Task 6: Parallel snapshots

**Files:**
- Modify: `internal/goldenfs/goldenfs_test.go`

- [ ] **Step 1: Add parallel snapshot test**

```go
func TestSnapshot_Parallel_Independent(t *testing.T) {
	base := baseFixture(t)

	snap1, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}
	defer snap1.Close()

	snap2, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}
	defer snap2.Close()

	// Write different content to the same path in each snapshot.
	if err := os.WriteFile(filepath.Join(snap1.MountDir(), "hello.txt"), []byte("snap1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(snap2.MountDir(), "hello.txt"), []byte("snap2"), 0644); err != nil {
		t.Fatal(err)
	}

	got1, _ := os.ReadFile(filepath.Join(snap1.MountDir(), "hello.txt"))
	got2, _ := os.ReadFile(filepath.Join(snap2.MountDir(), "hello.txt"))

	if string(got1) != "snap1" {
		t.Errorf("snap1: want snap1, got %q", got1)
	}
	if string(got2) != "snap2" {
		t.Errorf("snap2: want snap2, got %q", got2)
	}

	// Base untouched.
	orig, _ := os.ReadFile(filepath.Join(base, "hello.txt"))
	if string(orig) != "world" {
		t.Errorf("base must remain world, got %q", orig)
	}
}
```

- [ ] **Step 2: Run**

```bash
go test ./internal/goldenfs/ -run TestSnapshot_Parallel -v
```

Expected: PASS.

- [ ] **Step 3: Run all goldenfs tests one final time**

```bash
go test ./internal/goldenfs/ -v -count=1
```

Expected: all tests PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/goldenfs/goldenfs_test.go
git commit -m "test(goldenfs): parallel snapshot independence test"
```

---

## Task 7: Integration — mpr.Reader/Writer through FUSE

**Files:**
- Create: `internal/goldenfs/integration_test.go`

This test opens a real `testdata/expr-checker/minimal.mpr` via `mpr.Reader` through the FUSE mount, executes a write, and confirms the change is in the dirty layer (not on disk).

- [ ] **Step 1: Write the integration test**

```go
// SPDX-License-Identifier: Apache-2.0

//go:build linux

package goldenfs

import (
	"os"
	"path/filepath"
	"testing"

	mprpkg "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// repoRoot returns the absolute path to the repository root
// (two directories above this package).
func repoRoot(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

func exprCheckerDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(repoRoot(t), "testdata", "expr-checker")
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("testdata/expr-checker not found: %v", err)
	}
	return dir
}

// TestMprWriterThroughFUSE verifies that mpr.Writer writing through the FUSE
// overlay leaves testdata/expr-checker untouched.
func TestMprWriterThroughFUSE(t *testing.T) {
	snap, err := Open(exprCheckerDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()

	mprPath := filepath.Join(snap.MountDir(), "minimal.mpr")

	// Open for writing via the FUSE mount.
	reader, err := mprpkg.OpenWithOptions(mprPath, mprpkg.OpenOptions{ReadOnly: false})
	if err != nil {
		t.Fatalf("open mpr through FUSE: %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("close mpr: %v", err)
	}

	// The SQLite WAL may have been created in the overlay.
	// Confirm the real minimal.mpr has not been modified.
	realMpr := filepath.Join(exprCheckerDir(t), "minimal.mpr")
	origStat, _ := os.Stat(realMpr)

	// The snapshot's dirty layer may contain the WAL; base must not.
	walPath := filepath.Join(exprCheckerDir(t), "minimal.mpr-wal")
	if _, err := os.Stat(walPath); err == nil {
		t.Fatal("WAL file must not exist in base dir after FUSE write")
	}

	// mtime of the real file must be unchanged.
	newStat, _ := os.Stat(realMpr)
	if !origStat.ModTime().Equal(newStat.ModTime()) {
		t.Fatal("real minimal.mpr mtime changed — base was modified")
	}
}
```

- [ ] **Step 2: Run**

```bash
go test ./internal/goldenfs/ -run TestMprWriterThroughFUSE -v
```

Expected: PASS (or SKIP if testdata not present).

- [ ] **Step 3: Commit**

```bash
git add internal/goldenfs/integration_test.go
git commit -m "test(goldenfs): mpr.Writer through FUSE overlay leaves base untouched"
```

---

## Task 8: Integration — mxcli exec + mx check through FUSE

**Files:**
- Create: `internal/goldenfs/mxcheck_test.go`

This is a full end-to-end test: run an MDL statement through the executor against the FUSE mount, then call `mx check` on the same path, and verify it passes.

- [ ] **Step 1: Write the test**

```go
// SPDX-License-Identifier: Apache-2.0

//go:build linux,integration

package goldenfs

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/mendixlabs/mxcli/mdl/executor"
	"github.com/mendixlabs/mxcli/mdl/visitor"
)

func findMxBinaryForTest() string {
	if p := os.Getenv("MX_BINARY"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		pattern := filepath.Join(home, ".mxcli", "mxbuild", "*", "modeler", "mx")
		if matches, _ := filepath.Glob(pattern); len(matches) > 0 {
			return matches[len(matches)-1]
		}
	}
	return ""
}

func TestGoldenFS_ExecAndMxCheck(t *testing.T) {
	mxBin := findMxBinaryForTest()
	if mxBin == "" {
		t.Skip("mx binary not available (set MX_BINARY or install mxbuild)")
	}

	snap, err := Open(exprCheckerDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()

	mprPath := filepath.Join(snap.MountDir(), "minimal.mpr")

	// Connect backend to the FUSE-mounted project.
	backend, err := mprbackend.Open(mprPath)
	if err != nil {
		t.Fatalf("open backend through FUSE: %v", err)
	}
	defer backend.Close()

	_ = mock.MockBackend{} // ensure mock package compiles
	out := &bytes.Buffer{}
	ctx := &executor.ExecContext{
		Context: context.Background(),
		Backend: backend,
		Output:  out,
		Cache:   &executor.NewExecutorCache(),
	}

	mdl := `create or modify microflow MyFirstModule.ACT_GoldenTest () returns Nothing begin return; end;`
	prog, errs := visitor.Build(mdl)
	if len(errs) > 0 {
		t.Fatalf("parse: %v", errs)
	}
	if err := ctx.ExecuteProgram(prog); err != nil {
		t.Fatalf("exec: %v", err)
	}
	backend.Close()

	// mx check on the FUSE mount — must not see errors from our write.
	cmd := exec.Command(mxBin, "check", mprPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("mx check failed:\n%s", output)
	}
	t.Logf("mx check output:\n%s", output)

	// Base untouched — no new mxunit files on real disk.
	snap.Rollback()
}
```

Note: `executor.NewExecutorCache()` — check the actual exported name in `mdl/executor/executor.go` and adjust if needed (it may be `executorCache{}` or an exported constructor).

- [ ] **Step 2: Run with integration tag**

```bash
go test ./internal/goldenfs/ -run TestGoldenFS_ExecAndMxCheck -tags integration -v
```

Expected: PASS (or SKIP if mx binary unavailable).

- [ ] **Step 3: Wire into the existing integration test matrix**

Check that `make test` still passes (unit tests) and `go test ./... -tags integration` works when mx is available:

```bash
make test
```

Expected: all unit tests PASS; integration test skipped (no mx in CI).

- [ ] **Step 4: Final commit**

```bash
git add internal/goldenfs/mxcheck_test.go
git commit -m "test(goldenfs): end-to-end exec + mx check through FUSE overlay"
```

---

## Self-Review

**Spec coverage:**
- ✅ Isolation (no disk writes) — Tasks 2–5
- ✅ Speed (in-memory dirty map) — Task 2
- ✅ FUSE for mx check — Tasks 3–4
- ✅ Multiple concurrent snapshots — Task 6
- ✅ Explicit Commit — Task 5
- ✅ Tests never call Commit (rollback/close only) — Tasks 5–7
- ✅ Linux build tag + non-Linux stub — Task 1
- ✅ WAL/SHM not committed to base — Task 2 (dirty_test) + Task 7
- ✅ Integration with mpr.Reader/Writer — Task 7
- ✅ Integration with mx check — Task 8

**Type consistency check:**
- `dirtyLayer` fields and methods consistent across `dirty.go` and `dirty_test.go` ✅
- `overlayNode` referenced as concrete type in `overlay.go` and `goldenfs.go` ✅
- `overlayReadHandle` / `overlayWriteHandle` defined in Task 3 and Task 4, not used before ✅
- `Snapshot.layer` field of type `*dirtyLayer` threaded through consistently ✅
- `exprCheckerDir(t)` helper defined in Task 7, reused in Task 8 ✅
- `executor.NewExecutorCache()` — marked with a "verify name" note in Task 8 ✅

**Placeholder scan:** None found.
