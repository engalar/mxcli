# Test Filesystem Strategy

This document describes the three filesystem approaches used for test fixture
isolation and when to use each.  The goal is **zero wasted I/O**: no byte-copying
files that can be hard-linked, and no FUSE overhead for high-frequency reads.

## Decision Tree

```
Test needs a writable copy of a fixture project?
│
├─ Test runs on Linux AND reads < 50 .mxunit files per test?
│   └─ ✅ goldenfs — FUSE COW overlay, zero copy
│
├─ Test reads ≥ 50 .mxunit files per test
│   (e.g. ListAll-type tests that scan mprcontents)?
│   └─ ✅ SameFSTempDir — same-fs hard link, O(1) per mprcontents file
│
└─ Test runs on non-Linux?
    └─ ✅ SameFSTempDir — fallback (hard links or CopyFile)
```

## The Three Approaches

### 1. `t.TempDir()` — Go standard (slowest)

**How**: `os.MkdirTemp` using `$TMPDIR` (typically `/tmp`).

**Cost**:
| Step | File | Cost |
|------|------|------|
| Copy `.mpr` | 68 K | byte copy |
| Link mprcontents | 199 files | `os.Link` → EXDEV → byte copy × 199 |
| **Total** | | **~1 MB written per test** |

**Why it's slow**: `/tmp` and `testdata/` are on **different filesystems**.
`os.Link` returns EXDEV (cross-device link), falling back to byte-by-byte
copy for every file in `mprcontents/`.

**Use when**: You don't have access to `testfsutil.SameFSTempDir` (e.g. in a
package that can't import `internal/testfsutil`).

---

### 2. `testfsutil.SameFSTempDir` — Recommended for high-read tests

**Package**: `internal/testfsutil/samefs_tmp.go`

**How**:
```go
import "github.com/mendixlabs/mxcli/internal/testfsutil"

func TestSomething(t *testing.T) {
    tmp := testfsutil.SameFSTempDir(t)
    // tmp is on the same filesystem as testdata/
    // Hard links from testdata/ into tmp succeed (no EXDEV)
}
```

**Cost**:
| Step | File | Cost |
|------|------|------|
| Copy `.mpr` | 68 K | byte copy (trivial) |
| Link mprcontents | 199 files | `os.Link` → OK → O(1) each |
| **Total** | | **~68 K written + 0 (links)** |

**Architecture**:
```
SameFSTempDir creates a temp dir inside the repo root
(/mnt/data_sdb/mxcli/.mxcli-tmp-<random>/t-<random>/).
  ↑ This is the SAME filesystem as testdata/ (/mnt/data_sdb)
  ↑ so os.Link works without EXDEV.
```

The parent dir `.mxcli-tmp-*` is gitignored and persists for one `make test`
run.  Each test gets one `t-*` subdirectory, removed by `t.Cleanup`.

**When to use**: Tests that read many `.mxunit` files (e.g. `ListAll`,
`buildUnitCache`), or non-Linux CI where goldenfs doesn't work.

**Calling pattern** (typically wrapped in a copy helper):
```go
func copyMPRFixture(t *testing.T, srcMPR string) string {
    dstDir := testfsutil.SameFSTempDir(t)
    dstMPR := filepath.Join(dstDir, filepath.Base(srcMPR))
    testfsutil.CopyFile(srcMPR, dstMPR)
    srcContents := filepath.Join(filepath.Dir(srcMPR), "mprcontents")
    if info, err := os.Stat(srcContents); err == nil && info.IsDir() {
        dstContents := filepath.Join(dstDir, "mprcontents")
        testfsutil.HardLinkDir(srcContents, dstContents) // O(1) on same fs
    }
    return dstMPR
}
```

---

### 3. `goldenfs` — FUSE COW overlay, zero copy

**Package**: `internal/goldenfs/`

**How**:
```go
import "github.com/mendixlabs/mxcli/internal/goldenfs"

snap, err := goldenfs.Open(fixtureDir)
defer snap.Close()
mprPath := filepath.Join(snap.MountDir(), "minimal.mpr")
```

**Cost**:
| Step | File | Cost |
|------|------|------|
| FUSE mount | — | ~1 ms (serialised by `mountMu`) |
| Copy-up `.mpr` on first Open | 68 K | byte copy into dirty layer |
| Dirty-layer reads | — | in-memory (fast but single-core) |
| **Total** | | **~1 ms + dirty layer** |

**No files are copied at fixture time.** The test reads from a FUSE overlay
over the real `testdata/` directory.  Writes go to an in-memory dirty layer
that is discarded on `Close()`.

**Caveats**:
1. **FUSE server goroutines consume CPU** (~45% overhead in executor profile).
   Each goldenfs mount runs a go-fuse server in the test process.
2. **SQLite through FUSE is slow** — every page read goes kernel → daemon →
   dirty lookup → kernel.  This eliminates the benefit for tests that
   frequently read `.mxunit` files through `listUnitsByTypeV2`.
3. **Concurrent mount limit** — `mountMu` serialises `fs.Mount()` syscalls;
   `flock`-based lockfiles prevent cross-process cleanup races.

**When to use**:
- Tests that read **few** `.mxunit` files (< 50) per test.
- Tests that write new files without reading existing ones.
- Integration tests that need isolation from the base project.
- **Do NOT use** for tests that call `listUnitsByType` or `buildUnitCache`
  (these read every `.mxunit` and pay the FUSE round-trip for each).

---

## Performance Comparison (executor package as benchmark)

| Approach | executor run time | Bottleneck |
|----------|:-----------------:|------------|
| `t.TempDir()` (EXDEV fallback) | ~23 s | byte copy 199 files × 58 tests |
| `SameFSTempDir` + hard links | **~10 s** | trivial copy |
| `goldenfs` FUSE overlay | ~18 s | 45 % FUSE server CPU |

The fastest approach for most tests is **SameFSTempDir** — it avoids both
the byte-copy fallback and the FUSE runtime overhead.

## Which Package Uses What

| Package | Current | Recommended |
|---------|---------|-------------|
| `mdl/backend/mpr/` | goldenfs → SameFSTempDir | SameFSTempDir |
| `mdl/backend/mpr/repos/` | goldenfs → SameFSTempDir | SameFSTempDir |
| `mdl/executor/` | goldenfs → SameFSTempDir | **SameFSTempDir** (remove goldenfs) |
| `internal/goldenfs/` | goldenfs (own tests) | goldenfs (correct use) |
| `internal/bsoncompare/` | cache only | cache (no FS overhead) |
