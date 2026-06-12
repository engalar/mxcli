# Performance & Debugging Playbook

## Profiling mxcli with Go native tools

### CPU profiling on Windows

mxcli supports a `MXCLI_CPU_PROFILE` environment variable for ad-hoc profiling:

```powershell
# PowerShell
$env:MXCLI_CPU_PROFILE = "cpu.prof"
.\mxcli.exe -p minimal.mpr exec script.mdl

# Analyze
go tool pprof -top -cum cpu.prof
go tool pprof -list <FuncName> cpu.prof
go tool pprof -tree cpu.prof
```

Profile output always flushes even on error exit (os.Exit bypasses defers; we use a `run() int` wrapper so `pprof.StopCPUProfile()` runs unconditionally in `main()`).

### Key metrics to watch

| Metric | What it means |
|--------|---------------|
| `runtime.cgocall` dominating flat % | On Windows, syscalls map to cgocall. Look at cum% children for actual I/O hotspots. |
| `os.ReadFile` / `os.OpenFile` high cum% | Heavy file I/O. Check `readMprContents` → find callers. |
| `buildUnitCache` high cum% | Cold cache rebuild reading all mxunit files. Can't avoid on first call, but should be incremental thereafter. |
| `ListUnitsByType` / `ListModules` high cum% | Called too frequently — check if higher-level caches are working. |
| `modernc.org/sqlite` high cum% | SQLite pure-Go driver overhead. Large transactions or many small queries. |

### Profiling workflow

1. **Identify the script duration** — if > 5s for simple operations, profile.
2. **Open the MPR first** — the first `Open()` builds the cold cache (unavoidable). Run the profile on a pre-warmed MPR or a second invocation.
3. **Look at cum%** not flat% — `runtime.cgocall` is a red herring on Windows. Follow the cum% chain down.
4. **Check `readMprContents`** — if it's > 20% cum, content cache may not be working.

## BSON Debugging Techniques

### Reading raw BSON from MPR v2 mxunit files

The MPR v2 format stores each unit as a separate `.mxunit` file under `mprcontents/XX/YY/<uuid>.mxunit`. The file path uses standard UUID hex format. SQLite stores the UnitID column as Microsoft GUID byte-swapped blobs.

**UUID ↔ blob conversions (all in `modelsdk/mpr/reader.go`):**

| Function | Input | Output |
|----------|-------|--------|
| `blobToUUID(blob)` | 16-byte Microsoft GUID blob | Standard UUID hex string |
| `uuidToBlob(uuid)` | Standard UUID hex string | 16-byte Microsoft GUID blob |
| `blobToUUIDSwapped(blob)` | 16-byte Microsoft GUID blob | Standard UUID hex string (identical to blobToUUID) |

**File path from SQLite UnitID blob:**
```
blobToUUID(sqliteBlob) → "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
path = mprcontents/XX/YY/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx.mxunit
where XX = uuid[0:2], YY = uuid[2:4]
```

### Dumping ViewEntitySourceDocument BSON

Use the Go test in `mdl/executor/dump_viewentity_test.go`:
```powershell
$env:MXPR = "D:\Mendix\App1110\App1110.mpr"
$env:MQN = "FT.GoodViewEntity"
go test -v -run TestDumpViewEntitySourceDoc ./mdl/executor/
```

### Comparing BSON field orders between Studio Pro and mxcli

1. Dump both entities' source documents with the test above
2. Check the raw BSON field order (not JSON key order — BSON preserves insertion order):
   ```python
   import bson
   with open(path, 'rb') as f:
       doc = bson.decode_all(f.read())[0]
   print(list(doc.keys()))  # Shows insertion order
   ```
3. If field orders differ, check `internal/codegen/supplements.json` → `property_order_overrides` for the affected type name.
4. Add the override with field names in Studio Pro's order, then run `go run ./cmd/modelsdk-codegen/` to regenerate.

### Using the graphify query tool for codebase exploration

```bash
graphify query "How does view entity creation work?"
graphify path "buildUnitCache" "readMprContents"
graphify explain "contentCache"
```

## Performance Lessons Learned

### Lesson 1: Content cache must work in all modes

**Problem:** `contentCache` was only enabled via `EnableContentCache()` which was only called in daemon mode. `exec` commands using `OpenWithOptions(ReadOnly=true)` never got the cache.

**Fix:** Lazy-initialize `contentCache` on first `readMprContents()` call. Always active, zero overhead when unused.

**Takeaway:** Caches that are behind opt-in flags silently degrade non-flagged paths. Make them always-on with lazy init.

### Lesson 2: InvalidateCache should be incremental, not nuke-and-rebuild

**Problem:** Every write called `InvalidateCache()` which set `unitCacheValid=false` AND cleared `contentCache`. Next `ListModules` rebuilt from scratch reading all files.

**Fix:** Keep `unitCacheValid=false` (metadata still needs rebuild) but don't clear `contentCache`. Instead, use `ContentsHash` comparison in `buildUnitCache` to determine which entries are stale. Writers push new content directly to `contentCache` after successful writes.

**Takeaway:** Nuke-and-rebuild caches are correct but slow. Incremental invalidation via content hashing eliminates redundant I/O.

### Lesson 3: Avoid double-reading in cache-aware code paths

**Problem:** `buildUnitCache` read every file to extract `$Type`, then `listUnitsByTypeV2` read matching files AGAIN for contents.

**Fix:** Store `Contents` in `cachedUnit` during `buildUnitCache`. `listUnitsByTypeV2` uses `cu.Contents` directly.

**Takeaway:** When a "build" phase reads data and a "query" phase re-reads the same data, the data should be stored in the build output. Treat the cache as a write-through store.

### Lesson 4: Property order in generated code must match Studio Pro BSON

**Problem:** The codegen for `ViewEntitySourceDocument` serialized fields in SDK declaration order (`Name, Documentation, ...`), but Studio Pro uses `Documentation, Excluded, ExportLevel, Name, Oql`.

**Fix:** Added `property_order_overrides` in `supplements.json` for `DomainModels$ViewEntitySourceDocument`. The override system already existed for `NoGeneralization` — same mechanism.

**Takeaway:** When generated BSON output doesn't match Studio Pro, check `supplements.json` `property_order_overrides` first. It's a declarative fix — no code changes needed.
