# EXECUTE SCRIPT Transaction Redesign

**Date:** 2026-06-03  
**Status:** Approved for implementation  
**Affects:** `mdl/backend/mpr/`, `modelsdk/mpr/`, `mdl/executor/`

---

## Problem

`mxcli -p app.mpr -c "EXECUTE SCRIPT 'script.mdl'"` hangs indefinitely. `mxcli exec script.mdl` with the same script works fine.

### Root cause: pool self-deadlock

`BeginScriptTransaction()` calls `db.Begin()` which acquires the single SQLite connection (enforced by `reader.go:107`: `db.SetMaxOpenConns(1)`). Any subsequent operation inside the script that needs the database — reads (`db.Query`) or new-unit inserts (`db.Exec INSERT`) — blocks waiting for that same connection. Because it's the same goroutine, the connection is never freed. Permanent deadlock.

The `exec` subcommand path (`cmd_exec.go`) never calls `BeginScriptTransaction`, so each statement acquires and releases the connection independently — no deadlock.

### Four compounding bugs identified

| # | Bug | Symptom |
|---|-----|---------|
| 1 | `BeginScriptTransaction` holds connection | Any EXECUTE SCRIPT with a read or insert hangs |
| 2 | `insertUnit` bypasses `activeScriptTx` | New units (CREATE) are not atomic — cannot be rolled back |
| 3 | Reads during script see old content | `ALTER ENTITY` + `CREATE MICROFLOW` referencing new attribute fails |
| 4 | v2 Commit order: SQL before file rename | Crash between SQL commit and file rename leaves DB and files inconsistent |

### Confirmed by failing tests

`mdl/executor/execute_script_deadlock_test.go` (written before this fix):

```
PASS: TestExecPath_CreateThenRead          (0.39s)   ← exec path, no deadlock
FAIL: TestExecuteScriptPath_CreateThenRead (5.07s)   ← hangs until timeout
FAIL: TestExecuteScriptPath_ReadOnly       (5.07s)   ← even pure read hangs
```

---

## Solution: ScriptBuffer — deferred in-memory write accumulation

Replace the long-lived `*sql.Tx` with an in-memory buffer. During script execution, all writes accumulate in the buffer. Reads check the buffer before going to disk/SQLite. A single short SQL transaction is opened only at commit time.

This fixes all four bugs:
- **Bug 1**: No `db.Begin()` during script → connection pool never saturated
- **Bug 2**: All inserts buffered → can be rolled back atomically
- **Bug 3**: Reads check buffer first → scripts see their own writes
- **Bug 4**: v2 commit order reversed: write files → commit SQL → rename

---

## Architecture

### New type: `ScriptBuffer` (`mdl/backend/mpr/script_buf.go`)

```go
type scriptBufEntry struct {
    containerID     string
    containmentName string
    unitType        string
    contents        []byte
}

type ScriptBuffer struct {
    inserts map[string]scriptBufEntry  // new units created this script
    updates map[string][]byte          // existing units modified
    staged  []stagedFile               // v2: temp file paths for cleanup
    reader  *modelsdkmpr.Reader        // to update overlay in real-time
}
```

`ScriptBuffer` holds a reference to the `*Reader` so that `AddInsert` and `AddUpdate` can immediately update the reader's script overlay, making writes visible to subsequent reads within the same script.

### Reader changes (`modelsdk/mpr/reader.go`, `reader_units.go`)

Two new fields and two new methods:

```go
// New exported type (modelsdk/mpr/reader.go) — needed so ScriptBuffer can construct entries
type ScriptInsertEntry struct {
    ID              string
    ContainerID     string
    ContainmentName string
    UnitType        string
    Contents        []byte
}

// New fields on Reader
scriptOverlay  map[string][]byte    // content override for modified units
scriptInserts  []ScriptInsertEntry  // synthetic unit list for new units

// New methods
func (r *Reader) AppendScriptInsert(e ScriptInsertEntry)
func (r *Reader) SetScriptOverlay(id string, contents []byte)
func (r *Reader) ClearScriptMode()
```

`buildUnitCache` converts `r.scriptInserts` to `[]cachedUnit` and appends to the cache (~5 lines added).  
`readMprContents` checks `r.scriptOverlay[unitUUID]` before reading the file (~4 lines added).

`ScriptInsertEntry` is the only exported type addition. `cachedUnit` remains private — it is converted inside `buildUnitCache`. Import direction stays `mdl/backend/mpr` → `modelsdk/mpr`, no cycles.

`ScriptBuffer.AddInsert` calls `b.reader.AppendScriptInsert(...)` and `b.reader.SetScriptOverlay(...)` directly — no intermediate slice management needed in ScriptBuffer.

### MprBackend changes (`mdl/backend/mpr/backend.go`, `script_tx.go`)

Replace `activeScriptTx *modelsdkmpr.WriteTransaction` with `scriptBuf *ScriptBuffer`.

**`BeginScriptTransaction` rewritten** — critical change:

```go
// Before: opens SQL transaction → deadlock
func (b *MprBackend) BeginScriptTransaction() (backend.ScriptTransaction, error) {
    wtx, err := b.msdkWriter.BeginWriteTransaction()  // db.Begin() — THE BUG
    b.activeScriptTx = wtx
    ...
}

// After: no SQL transaction during script
func (b *MprBackend) BeginScriptTransaction() (backend.ScriptTransaction, error) {
    buf := newScriptBuffer(b.reader)
    b.scriptBuf = buf
    // ScriptBuffer.AddInsert/AddUpdate call reader.AppendScriptInsert /
    // SetScriptOverlay directly as writes accumulate — no upfront init needed.
    return &mprScriptTx{b: b, buf: buf}, nil
}
```

**New `insertUnit` wrapper** routes to buffer when script is active:

```go
func (b *MprBackend) insertUnit(unitID, containerID, containmentName, unitType string, contents []byte) error {
    if b.scriptBuf != nil {
        return b.scriptBuf.AddInsert(unitID, containerID, containmentName, unitType, contents)
    }
    return b.msdkWriter.InsertUnit(unitID, containerID, containmentName, unitType, contents)
}
```

All ~10 call sites of `b.msdkWriter.InsertUnit()` in the backend are replaced with `b.insertUnit()`.

**`writeUnitContents` updated** to route updates to buffer:

```go
// replaces: if b.activeScriptTx != nil { b.activeScriptTx.WriteUnit(...) }
if b.scriptBuf != nil {
    return b.scriptBuf.AddUpdate(string(unitID), contents)
}
```

### Commit flow

**v2 (file + SQLite):**

1. Write all buffered content to `.mxunit.staged` (inserts) / `.mxunit.tmp` (updates)
2. `db.Begin()` — connection held only for the duration of steps 3-4
3. Batch `INSERT INTO Unit` for all inserts; `UPDATE Unit SET ContentsHash` for all updates
4. `tx.Commit()` — releases connection
5. `os.Rename` all staged/tmp files to final `.mxunit` paths
6. `b.reader.ClearScriptMode()` + `InvalidateCache()`

Files written before SQL commit (Bug 4 fix). Residual crash window: between SQL commit (step 4) and final rename (step 5) — same window as before but now step 1 failures are clean (no SQL side effects). Full two-phase commit is out of scope.

**v1 (SQLite only):**

1. `db.Begin()`
2. Batch `INSERT INTO Unit (..., Contents)` and `UPDATE Unit SET Contents`
3. `tx.Commit()` — fully atomic, no file operations
4. `ClearScriptMode()`

v1 achieves perfect atomicity.

### Rollback flow

```go
func (tx *mprScriptTx) Rollback() error {
    tx.b.scriptBuf.cleanupStaged()   // delete .staged and .tmp files
    tx.b.scriptBuf = nil
    tx.b.reader.ClearScriptMode()
    return nil
    // No SQL rollback needed — no SQL transaction was open during the script
}
```

### `updateTransactionID`

`insertUnit` currently calls `w.updateTransactionID()` after each insert. In script mode, this is deferred to `ScriptBuffer.Commit()` — one call after all inserts, maintaining identical semantics for non-script paths.

---

## File Change List

| File | Change | Type |
|------|--------|------|
| `mdl/backend/mpr/script_buf.go` | New: ScriptBuffer + AddInsert + AddUpdate + Commit + Rollback | NEW |
| `mdl/backend/mpr/script_tx.go` | Rewrite: mprScriptTx delegates to ScriptBuffer | MOD |
| `mdl/backend/mpr/backend.go` | Replace `activeScriptTx` with `scriptBuf`; add `insertUnit()` wrapper | MOD |
| `mdl/backend/mpr/write_helpers.go` | Route to `scriptBuf.AddUpdate()` instead of `activeScriptTx.WriteUnit()` | MOD |
| `mdl/backend/mpr/create_services_modelsdk.go` | ~7 call sites: `msdkWriter.InsertUnit` → `b.insertUnit` | MOD |
| `mdl/backend/mpr/domainmodel_modelsdk.go` | 1 call site | MOD |
| `mdl/backend/mpr/modules_modelsdk.go` | 2 call sites | MOD |
| `modelsdk/mpr/reader.go` | Add `scriptOverlay`, `scriptInserts` fields; `SetScriptMode`/`ClearScriptMode` | MOD |
| `modelsdk/mpr/reader_units.go` | `buildUnitCache`: append scriptInserts; `readMprContents`: check overlay | MOD |

---

## Test Coverage

### RED baseline (written before fix — must go green)

| Test | File | Validates |
|------|------|-----------|
| `TestExecPath_CreateThenRead` | `mdl/executor/execute_script_deadlock_test.go` | exec path stays correct (control group) |
| `TestExecuteScriptPath_CreateThenRead` | same | Bug 1: EXECUTE SCRIPT create+read no deadlock |
| `TestExecuteScriptPath_ReadOnly` | same | Bug 1: even pure-read script no deadlock |

### New tests (written alongside implementation)

| Layer | Test | File | Validates |
|-------|------|------|-----------|
| L1 | `TestScriptBuffer_AddInsert_VisibleInOverlay` | `mdl/backend/mpr/` | AddInsert updates reader overlay immediately |
| L1 | `TestScriptBuffer_AddUpdate_VisibleInOverlay` | `mdl/backend/mpr/` | AddUpdate updates reader overlay immediately |
| L1 | `TestScriptBuffer_Rollback_ClearsOverlay` | `mdl/backend/mpr/` | Rollback clears reader overlay, deletes staged files |
| L1 | `TestBeginScriptTransaction_NoDBBegin` | `mdl/backend/mpr/` | BeginScriptTransaction does not open a SQL transaction |
| L3 | `TestExecuteScriptPath_ReadOwnWrite` | `mdl/executor/` | Bug 3: ALTER then reference new attribute in same script |
| L3 | `TestExecuteScriptPath_RollbackOnError` | `mdl/executor/` | Bug 2: mid-script failure rolls back prior creates |
| L3 | `TestExecuteScriptPath_CreateThenCreate` | `mdl/executor/` | Multiple CREATEs all committed correctly |
| L3 | `TestExecuteScriptPath_NestedDepthLimit` | `mdl/executor/` | maxScriptDepth guard still works |

### Completion criteria

```bash
go test ./mdl/executor/ -run TestExecPath              # PASS (stays green)
go test ./mdl/executor/ -run TestExecuteScriptPath     # PASS (2 RED → GREEN)
go test ./mdl/backend/mpr/ -run TestScriptBuffer       # PASS (new L1)
go test ./...                                          # all pass, no regressions
```

---

## Scope and non-goals

**In scope:**
- Fix deadlock (Bug 1)
- Make inserts atomic with script transaction (Bug 2)
- Read-your-own-writes within a script (Bug 3)
- Fix v2 commit order: files before SQL (Bug 4 — partial; perfect two-phase commit excluded)

**Out of scope:**
- Perfect crash-safety between SQL commit and file rename (requires full two-phase commit)
- Memory limits / spill-to-disk for very large script buffers
- OOM protection for scripts modifying thousands of units
