# Golden MPR Filesystem Design

**Date:** 2026-05-19  
**Status:** Approved

## Problem

The current dev workflow for validating MDL writes against `mx check` requires
operating directly on `testdata/expr-checker/minimal.mpr` and its `mprcontents/`
folder, then running `git restore` to clean up. This is:

- **Slow to restore** — `git restore testdata/expr-checker/` touches hundreds of files
- **Fragile** — a failed test or interrupted session leaves the golden files dirty
- **Not parallelisable** — concurrent tests would corrupt each other's golden baseline

## Goal

A copy-on-write overlay filesystem (`internal/goldenfs`) that:

1. **Isolates** every caller behind its own in-memory dirty layer — `testdata/` is never touched
2. **Is fast** — dirty layer lives in RAM; no disk I/O during `mxcli exec`
3. **Exposes a real path** via FUSE so `mx check` (an external process) can read it
4. **Supports multiple concurrent snapshots** — each `Open()` call gets its own mount point and dirty map
5. **Supports explicit Commit** — a developer (not a test) can flush the overlay to disk to update the golden baseline intentionally

## Non-Goals

- Tests never call `Commit()` — the golden files are read-only fixtures for tests
- No MDL `BEGIN`/`COMMIT` syntax in this phase (Phase 2)
- No Windows or macOS support (FUSE requires Linux; dev environment is Linux)

## Package Layout

```
internal/goldenfs/
├── goldenfs.go         — Snapshot struct, Open/Close/Commit/Rollback
├── overlay.go          — FUSE filesystem implementation
├── goldenfs_linux.go   — Linux build tag, real implementation
├── goldenfs_stub.go    — Non-Linux stub (returns ErrNotSupported)
└── goldenfs_test.go
```

Dependency: `github.com/hanwen/go-fuse/v2` (Linux only, build-tagged).

## Public API

```go
// Open mounts a FUSE overlay over baseDir and returns a Snapshot.
// Each call creates an independent /tmp/mxcli-golden-<uuid>/ mount point.
// Multiple concurrent snapshots over the same baseDir are safe.
func Open(baseDir string) (*Snapshot, error)

// MountDir returns the FUSE mount path.
// Pass this to mxcli exec (-p flag) and mx check as the project directory.
func (s *Snapshot) MountDir() string

// Commit flushes all dirty files from the in-memory layer to baseDir.
// Only for intentional golden-baseline updates; never called from tests.
func (s *Snapshot) Commit() error

// Rollback discards all in-memory changes.
// The FUSE view reverts to serving baseDir contents unchanged.
func (s *Snapshot) Rollback()

// Close unmounts the FUSE filesystem and removes the tmp mount directory.
// Does NOT commit. Safe to defer — always cleans up even on test failure.
func (s *Snapshot) Close() error
```

## Snapshot Internal Structure

```go
type Snapshot struct {
    baseDir  string
    mountDir string          // /tmp/mxcli-golden-<uuid>/
    server   *fuse.Server
    mu       sync.RWMutex
    dirty    map[string][]byte  // relPath → full file bytes (nil = tombstone)
}
```

`relPath` is relative to `baseDir`/`mountDir`, e.g.:
- `minimal.mpr`
- `minimal.mpr-wal`
- `mprcontents/00/39/003913f0-....mxunit`
- `mprcontents/mprname`

## FUSE Overlay Semantics

Each file or directory in the mount is an `overlayNode` holding its `relPath`
and a pointer back to the `Snapshot`. Key operations:

| FUSE op | Logic |
|---------|-------|
| `Read` | dirty map has relPath → serve from memory; else → `os.ReadFile(base/relPath)` |
| `Write` / `Create` | copy-up if not in dirty map, then patch at offset |
| `Readdir` | `os.ReadDir(base/dir)` ∪ dirty-map keys under that dir, minus tombstones |
| `Getattr` | dirty map hit → synthesise stat from `len([]byte)`; else → `os.Lstat(base)` |
| `Unlink` | write `nil` tombstone to dirty map |
| `Rename` | dirty[src] = nil tombstone; dirty[dst] = old content |
| `Mkdir` | store empty-dir sentinel in dirty map |
| `Fsync` | no-op (durability deferred to `Commit()`) |

## Partial Read/Write (SQLite)

`.mxunit` files are always fully overwritten by `mpr.Writer` — simple.

`minimal.mpr` is opened by SQLite with random page-level reads and writes (4 KB
pages). The dirty map stores the **complete file bytes**; partial writes are
applied with a copy-up + in-place patch:

```
Write(relPath, offset, data):
    if relPath not in dirty:
        dirty[relPath] = os.ReadFile(base/relPath)   // copy-up
    if offset+len(data) > len(dirty[relPath]):
        grow slice
    copy(dirty[relPath][offset:], data)
```

### WAL / SHM files

SQLite creates `minimal.mpr-wal` and `minimal.mpr-shm` alongside the main file
during a write transaction. Both land in the dirty map via `Create` + `Write`
and never touch `baseDir`. `Commit()` skips them — SQLite clears WAL/SHM on
checkpoint, so they are not part of the persistent state.

### Memory budget

| Scenario | Memory |
|----------|--------|
| Idle (nothing written) | 0 — dirty map empty |
| 10 mxunit files modified | ~0.5 MB |
| `minimal.mpr` copy-up | +532 KB |
| WAL peak | +~1 MB |
| **Typical total** | **< 2 MB** |

## Commit Logic

```
for relPath, content := range dirty:
    if isSQLiteAux(relPath):  // -wal, -shm
        skip
    if content == nil:        // tombstone
        os.Remove(base/relPath)
    else:
        os.MkdirAll(parent)
        os.WriteFile(base/relPath, content, 0644)
```

## Integration

### Tests (primary consumer)

```go
func TestExecWithMxCheck(t *testing.T) {
    snap, err := goldenfs.Open("../../testdata/expr-checker")
    require.NoError(t, err)
    defer snap.Close()  // always cleans up; baseDir untouched

    ctx := newExecContext(snap.MountDir() + "/minimal.mpr")
    require.NoError(t, runMDL(ctx, "create microflow MyFirstModule.ACT_Test ..."))

    out, err := exec.Command(mxbuildPath, "check",
        snap.MountDir()+"/minimal.mpr").CombinedOutput()
    if err != nil {
        t.Fatalf("mx check failed:\n%s", out)
    }
    // defer snap.Close() discards all changes — golden files stay clean
}
```

Parallel tests: each `t.Run` calls its own `goldenfs.Open`; independent mount
points make `t.Parallel()` safe.

### Developer golden-baseline update

```bash
MOUNT=$(mxcli golden-mount testdata/expr-checker/)
./bin/mxcli -p $MOUNT/minimal.mpr -c "create microflow ..."
~/.mxcli/mxbuild/.../mx check $MOUNT/minimal.mpr \
    && mxcli golden-commit $MOUNT \
    || mxcli golden-rollback $MOUNT
```

### Phase 2 — MDL transaction syntax (not in scope now)

```sql
BEGIN SNAPSHOT;
create microflow MyFirstModule.ACT_Test () returns Nothing begin return; end;
CHECK;      -- calls mx check internally; ROLLBACK on failure
COMMIT;     -- flushes overlay to the real MPR project on disk
```

`Snapshot` is the direct backend for this syntax; only the executor frontend
needs to be added in Phase 2.

## Risks

| Risk | Mitigation |
|------|-----------|
| SQLite POSIX locking through FUSE | Test with real SQLite writes early; `go-fuse` passes `SETLK`/`GETLK` to kernel VFS |
| FUSE unavailable in CI | Build-tag Linux only; CI must run on Linux (already the case) |
| Large projects with many dirty files | Cap dirty map or warn; typical testdata is well under budget |
