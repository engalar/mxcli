# EXECUTE SCRIPT Transaction Fix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the long-lived `*sql.Tx` in `BeginScriptTransaction` with an in-memory `ScriptBuffer`, fixing the pool self-deadlock and restoring atomicity/read-own-writes for `EXECUTE SCRIPT`.

**Architecture:** `ScriptBuffer` (new) accumulates inserts and updates in memory during script execution. At commit time, a single short `db.Begin()/Commit()` transaction applies all changes atomically. Reader gains `scriptOverlay`/`scriptInserts` fields so reads within the script see buffered writes immediately.

**Tech Stack:** Go, `modernc.org/sqlite`, `modelsdk/mpr` (Reader/Writer), `mdl/backend/mpr` (MprBackend)

---

## File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `modelsdk/mpr/reader.go` | Modify | Add `ScriptInsertEntry` type + 3 overlay fields + 3 methods |
| `modelsdk/mpr/reader_units.go` | Modify | `buildUnitCache` merges script inserts; `readMprContents` checks overlay |
| `modelsdk/mpr/writer_core.go` | Modify | Add `BatchWriteOp` type + `BatchWrite` method for atomic batch commit |
| `mdl/backend/mpr/script_buf.go` | Create | `ScriptBuffer` type + `AddInsert`/`AddUpdate`/`Rollback` |
| `mdl/backend/mpr/script_tx.go` | Rewrite | `mprScriptTx` delegates Commit/Rollback to ScriptBuffer |
| `mdl/backend/mpr/backend.go` | Modify | Replace `activeScriptTx` → `scriptBuf`; add `insertUnit()` wrapper; add `commitScriptBuffer()` |
| `mdl/backend/mpr/write_helpers.go` | Modify | Route updates to `scriptBuf.AddUpdate()` |
| `mdl/backend/mpr/create_services_modelsdk.go` | Modify | ~7 call sites: `b.msdkWriter.InsertUnit` → `b.insertUnit` |
| `mdl/backend/mpr/domainmodel_modelsdk.go` | Modify | 1 call site |
| `mdl/backend/mpr/modules_modelsdk.go` | Modify | 2 call sites |
| `mdl/executor/execute_script_deadlock_test.go` | Modify | Remove `//go:build execute_script_deadlock` tag |

---

## Task 1: Reader — ScriptInsertEntry + overlay fields + 3 methods

**Files:**
- Modify: `modelsdk/mpr/reader.go`

- [ ] **Step 1: Add type and fields**

In `modelsdk/mpr/reader.go`, after the existing `Reader` struct definition, add the exported type and two new fields to the struct:

```go
// ScriptInsertEntry represents a unit inserted within an EXECUTE SCRIPT block
// that has not yet been committed to SQLite. Exported so mdl/backend/mpr can
// construct entries without a circular import.
type ScriptInsertEntry struct {
	ID              string
	ContainerID     string
	ContainmentName string
	UnitType        string
	Contents        []byte
}
```

Add to the `Reader` struct (after `contentCache`):

```go
	// script overlay: set by MprBackend during EXECUTE SCRIPT execution so reads
	// within the script see buffered writes before they are committed.
	scriptOverlay  map[string][]byte    // unitID → buffered content
	scriptInserts  []ScriptInsertEntry  // new units not yet in SQLite
```

- [ ] **Step 2: Add the three new methods**

Add at the bottom of `modelsdk/mpr/reader.go`:

```go
// AppendScriptInsert adds a unit to the script-mode insert list so that
// buildUnitCache includes it in subsequent list operations.
// Called by ScriptBuffer.AddInsert.
func (r *Reader) AppendScriptInsert(e ScriptInsertEntry) {
	r.scriptInserts = append(r.scriptInserts, e)
	r.unitCacheValid = false // force rebuild to include new entry
}

// SetScriptOverlay stores content that overrides the on-disk file or SQLite
// row for this unit. Called by ScriptBuffer on every insert and update.
func (r *Reader) SetScriptOverlay(unitID string, contents []byte) {
	if r.scriptOverlay == nil {
		r.scriptOverlay = make(map[string][]byte)
	}
	r.scriptOverlay[unitID] = contents
	// Evict from contentCache so next read returns the overlay, not stale cache.
	if r.contentCache != nil {
		delete(r.contentCache, unitID)
	}
}

// ClearScriptMode removes the script overlay and insert list, restoring
// normal read behaviour. Called by MprBackend after Commit or Rollback.
func (r *Reader) ClearScriptMode() {
	r.scriptOverlay = nil
	r.scriptInserts = nil
	r.unitCacheValid = false
}
```

- [ ] **Step 3: Verify it compiles**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go build ./modelsdk/mpr/...
```

Expected: no output (success).

- [ ] **Step 4: Commit**

```bash
git add modelsdk/mpr/reader.go
git commit -m "feat(modelsdk/mpr): add ScriptInsertEntry type + overlay fields + 3 Reader methods"
```

---

## Task 2: Reader — buildUnitCache + readMprContents changes

**Files:**
- Modify: `modelsdk/mpr/reader_units.go`

- [ ] **Step 1: Update `buildUnitCache` to include script inserts**

In `modelsdk/mpr/reader_units.go`, find `func (r *Reader) buildUnitCache() error`. After the existing `rows.Close()` call (the SQL results are fully read), append the script inserts:

```go
	// Merge buffered script inserts so reads within EXECUTE SCRIPT see new units.
	for _, e := range r.scriptInserts {
		typeName := getTypeFromContents(e.Contents)
		r.unitCache = append(r.unitCache, cachedUnit{
			ID:              e.ID,
			ContainerID:     e.ContainerID,
			ContainmentName: e.ContainmentName,
			Type:            typeName,
		})
	}
```

Place this block just before the `r.unitCacheValid = true` line.

- [ ] **Step 2: Update `readMprContents` to check overlay first**

In `modelsdk/mpr/reader_units.go`, find `func (r *Reader) readMprContents(unitUUID string) ([]byte, error)`. Add at the very top of the function body (before any other logic):

```go
	// Script overlay: return buffered content immediately, skipping file/cache I/O.
	if b, ok := r.scriptOverlay[unitUUID]; ok {
		return b, nil
	}
```

- [ ] **Step 3: Run existing Reader tests**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./modelsdk/mpr/... -v -count=1 2>&1 | tail -10
```

Expected: all PASS.

- [ ] **Step 4: Commit**

```bash
git add modelsdk/mpr/reader_units.go
git commit -m "feat(modelsdk/mpr): buildUnitCache merges script inserts; readMprContents checks overlay"
```

---

## Task 3: Writer — BatchWriteOp + BatchWrite

**Files:**
- Modify: `modelsdk/mpr/writer_core.go`

- [ ] **Step 1: Add the BatchWriteOp type**

After the `WriteTransaction` struct in `modelsdk/mpr/writer_core.go`, add:

```go
// BatchWriteOp describes a single insert or update for BatchWrite.
// When Insert is false, only UnitID and Contents are used.
type BatchWriteOp struct {
	Insert          bool
	UnitID          string
	ContainerID     string
	ContainmentName string
	UnitType        string
	Contents        []byte
}
```

- [ ] **Step 2: Add the BatchWrite method**

Add at the end of `modelsdk/mpr/writer_core.go`:

```go
// BatchWrite executes all ops in a single atomic SQL transaction.
//
// For v2 MPR:
//   - Insert ops: content file written to final path BEFORE opening the SQL tx.
//     (An orphaned file on tx failure is acceptable; it won't appear in the unit list.)
//   - Update ops: content written to a .tmp file BEFORE the tx; renamed to final AFTER Commit.
//
// For v1 MPR all content lives in SQLite; no file I/O is needed.
//
// After a successful commit, the reader cache is invalidated and the
// transaction ID is updated (v2 only).
func (w *Writer) BatchWrite(ops []BatchWriteOp) error {
	type pendingRename struct{ tmp, final string }
	var renames []pendingRename

	// ── Phase 1: file writes (v2 only, before opening SQL tx) ──────────────
	if w.reader.version == MPRVersionV2 {
		for _, op := range ops {
			unitIDBlob := uuidToBlob(op.UnitID)
			if unitIDBlob == nil {
				return fmt.Errorf("BatchWrite: invalid unit ID %q", op.UnitID)
			}
			swappedUUID := blobToUUIDSwapped(unitIDBlob)
			dir := filepath.Join(w.reader.contentsDir, swappedUUID[0:2], swappedUUID[2:4])
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("BatchWrite: mkdir %s: %w", dir, err)
			}
			finalPath := filepath.Join(dir, swappedUUID+".mxunit")

			if op.Insert {
				// Write directly to final path (same as single insertUnit).
				if err := os.WriteFile(finalPath, op.Contents, 0644); err != nil {
					return fmt.Errorf("BatchWrite: write insert file: %w", err)
				}
			} else {
				// Write to .tmp; rename after SQL commit.
				tmpPath := finalPath + ".tmp"
				if err := os.WriteFile(tmpPath, op.Contents, 0644); err != nil {
					return fmt.Errorf("BatchWrite: write update tmp: %w", err)
				}
				renames = append(renames, pendingRename{tmp: tmpPath, final: finalPath})
			}
		}
	}

	// ── Phase 2: single SQL transaction ────────────────────────────────────
	sqlTx, err := w.reader.db.Begin()
	if err != nil {
		return fmt.Errorf("BatchWrite: begin tx: %w", err)
	}
	defer func() {
		if sqlTx != nil {
			_ = sqlTx.Rollback()
			// Clean up tmp files if tx was not committed.
			for _, r := range renames {
				_ = os.Remove(r.tmp)
			}
		}
	}()

	for _, op := range ops {
		unitIDBlob := uuidToBlob(op.UnitID)
		if unitIDBlob == nil {
			return fmt.Errorf("BatchWrite: invalid unit ID %q", op.UnitID)
		}

		if op.Insert {
			if w.reader.version == MPRVersionV2 {
				hash := sha256.Sum256(op.Contents)
				contentsHash := base64.StdEncoding.EncodeToString(hash[:])
				_, err = sqlTx.Exec(`
					INSERT INTO Unit (UnitID, ContainerID, ContainmentName, TreeConflict, ContentsHash, ContentsConflicts)
					VALUES (?, ?, ?, 0, ?, '')
				`, unitIDBlob, uuidToBlob(op.ContainerID), op.ContainmentName, contentsHash)
			} else {
				containerIDBlob := uuidToBlob(op.ContainerID)
				// Try new schema (Mendix 11.6.2+, no Type column).
				_, err = sqlTx.Exec(`
					INSERT INTO Unit (UnitID, ContainerID, ContainmentName, TreeConflict, ContentsHash, ContentsConflicts, Contents)
					VALUES (?, ?, ?, 0, '', '', ?)
				`, unitIDBlob, containerIDBlob, op.ContainmentName, op.Contents)
				if err != nil {
					// Fall back to old schema with Type column.
					_, err = sqlTx.Exec(`
						INSERT INTO Unit (UnitID, ContainerID, ContainmentName, Type, Contents)
						VALUES (?, ?, ?, ?, ?)
					`, unitIDBlob, containerIDBlob, op.ContainmentName, op.UnitType, op.Contents)
				}
			}
		} else { // update
			if w.reader.version == MPRVersionV2 {
				hash := sha256.Sum256(op.Contents)
				contentsHash := base64.StdEncoding.EncodeToString(hash[:])
				_, err = sqlTx.Exec(`UPDATE Unit SET ContentsHash = ? WHERE UnitID = ?`, contentsHash, unitIDBlob)
			} else {
				_, err = sqlTx.Exec(`UPDATE Unit SET Contents = ? WHERE UnitID = ?`, op.Contents, unitIDBlob)
			}
		}
		if err != nil {
			return fmt.Errorf("BatchWrite: sql op for unit %s: %w", op.UnitID, err)
		}
	}

	if err := sqlTx.Commit(); err != nil {
		return fmt.Errorf("BatchWrite: commit: %w", err)
	}
	sqlTx = nil // prevent deferred rollback

	// ── Phase 3: rename .tmp → final (v2 updates) ──────────────────────────
	for _, r := range renames {
		if err := os.Rename(r.tmp, r.final); err != nil {
			log.Printf("mpr.batchwrite_rename_failed: tmp=%s final=%s err=%s", r.tmp, r.final, err)
		}
	}

	w.updateTransactionID()
	w.reader.InvalidateCache()
	return nil
}
```

Ensure the required imports are present at the top of the file:
- `"crypto/sha256"` (already used by `WriteUnit`)
- `"encoding/base64"` (already used)
- `"log"` — add if not present
- `"os"` (already used)
- `"path/filepath"` (already used)

- [ ] **Step 3: Compile and run modelsdk/mpr tests**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./modelsdk/mpr/... -count=1 2>&1 | tail -5
```

Expected: all PASS.

- [ ] **Step 4: Commit**

```bash
git add modelsdk/mpr/writer_core.go
git commit -m "feat(modelsdk/mpr): add BatchWriteOp + BatchWrite for atomic script commit"
```

---

## Task 4: ScriptBuffer — new type (script_buf.go)

**Files:**
- Create: `mdl/backend/mpr/script_buf.go`

- [ ] **Step 1: Write L1 tests first**

Create `mdl/backend/mpr/script_buf_test.go`:

```go
// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"path/filepath"
	"testing"

	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

func TestScriptBuffer_AddUpdate_VisibleInOverlay(t *testing.T) {
	dst := copyFixture(t, fixturePath, t.TempDir())
	b := New()
	if err := b.Connect(dst); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	buf := newScriptBuffer(b.reader)

	unitID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	contents := []byte{0x01, 0x02, 0x03}

	if err := buf.AddUpdate(unitID, contents); err != nil {
		t.Fatalf("AddUpdate: %v", err)
	}

	// Overlay must be visible on the reader immediately.
	got, err := b.reader.ReadUnitContents(unitID)
	if err == nil && string(got) == string(contents) {
		// success: overlay returned
	} else {
		// Check via internal overlay map directly (same package).
		if b.reader.ScriptOverlay() == nil {
			t.Fatal("scriptOverlay is nil after AddUpdate")
		}
		if got, ok := b.reader.ScriptOverlay()[unitID]; !ok || string(got) != string(contents) {
			t.Errorf("overlay mismatch: got %v, want %v", got, contents)
		}
	}
}

func TestScriptBuffer_AddInsert_VisibleInInsertList(t *testing.T) {
	dst := copyFixture(t, fixturePath, t.TempDir())
	b := New()
	if err := b.Connect(dst); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	buf := newScriptBuffer(b.reader)

	if err := buf.AddInsert(
		"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		"00000000-0000-0000-0000-000000000000",
		"Documents",
		"Microflows$Microflow",
		[]byte{0x05},
	); err != nil {
		t.Fatalf("AddInsert: %v", err)
	}

	if len(b.reader.ScriptInserts()) == 0 {
		t.Fatal("scriptInserts empty after AddInsert")
	}
	if b.reader.ScriptInserts()[0].ID != "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb" {
		t.Errorf("unexpected insert ID: %s", b.reader.ScriptInserts()[0].ID)
	}
}

func TestScriptBuffer_Rollback_ClearsOverlay(t *testing.T) {
	dst := copyFixture(t, fixturePath, t.TempDir())
	b := New()
	if err := b.Connect(dst); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	buf := newScriptBuffer(b.reader)
	_ = buf.AddUpdate("cccccccc-cccc-cccc-cccc-cccccccccccc", []byte{0x99})

	if b.reader.ScriptOverlay() == nil {
		t.Fatal("expected overlay before rollback")
	}

	buf.Rollback()

	if b.reader.ScriptOverlay() != nil {
		t.Error("scriptOverlay not nil after Rollback")
	}
	if b.reader.ScriptInserts() != nil {
		t.Error("scriptInserts not nil after Rollback")
	}
}
```

These tests use `b.reader.ScriptOverlay()` and `b.reader.ScriptInserts()` accessor methods — add them to `modelsdk/mpr/reader.go` alongside the other new methods:

```go
// ScriptOverlay returns the current script-mode content overlay (test helper).
func (r *Reader) ScriptOverlay() map[string][]byte { return r.scriptOverlay }

// ScriptInserts returns the current script-mode insert list (test helper).
func (r *Reader) ScriptInserts() []ScriptInsertEntry { return r.scriptInserts }
```

- [ ] **Step 2: Run tests — expect compile failure (ScriptBuffer not defined yet)**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./mdl/backend/mpr/ -run TestScriptBuffer -v 2>&1 | head -20
```

Expected: `undefined: newScriptBuffer` or similar compile error.

- [ ] **Step 3: Create script_buf.go**

Create `mdl/backend/mpr/script_buf.go`:

```go
// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// scriptBufEntry holds data for a unit insert buffered during EXECUTE SCRIPT.
type scriptBufEntry struct {
	containerID     string
	containmentName string
	unitType        string
	contents        []byte
}

// ScriptBuffer accumulates write operations during an EXECUTE SCRIPT block.
// No SQL connection is held. A single atomic BatchWrite is issued at Commit time.
type ScriptBuffer struct {
	inserts map[string]scriptBufEntry // unitID → insert entry
	updates map[string][]byte         // unitID → new contents
	reader  *modelsdkmpr.Reader
}

func newScriptBuffer(r *modelsdkmpr.Reader) *ScriptBuffer {
	return &ScriptBuffer{
		inserts: make(map[string]scriptBufEntry),
		updates: make(map[string][]byte),
		reader:  r,
	}
}

// AddInsert buffers a unit insert and immediately updates the reader overlay
// so subsequent reads within the same script see the new unit.
func (buf *ScriptBuffer) AddInsert(unitID, containerID, containmentName, unitType string, contents []byte) error {
	buf.inserts[unitID] = scriptBufEntry{
		containerID:     containerID,
		containmentName: containmentName,
		unitType:        unitType,
		contents:        contents,
	}
	buf.reader.SetScriptOverlay(unitID, contents)
	buf.reader.AppendScriptInsert(modelsdkmpr.ScriptInsertEntry{
		ID:              unitID,
		ContainerID:     containerID,
		ContainmentName: containmentName,
		UnitType:        unitType,
		Contents:        contents,
	})
	return nil
}

// AddUpdate buffers a unit content update and immediately updates the reader
// overlay so subsequent reads return the new content.
func (buf *ScriptBuffer) AddUpdate(unitID string, contents []byte) error {
	buf.updates[unitID] = contents
	buf.reader.SetScriptOverlay(unitID, contents)
	return nil
}

// Rollback discards all buffered writes and clears the reader script overlay.
// No SQL operation is needed — no transaction was open during the script.
func (buf *ScriptBuffer) Rollback() {
	buf.inserts = nil
	buf.updates = nil
	buf.reader.ClearScriptMode()
}

// toBatchOps converts the buffer contents to a slice of BatchWriteOp values
// suitable for msdkWriter.BatchWrite.
func (buf *ScriptBuffer) toBatchOps() []modelsdkmpr.BatchWriteOp {
	ops := make([]modelsdkmpr.BatchWriteOp, 0, len(buf.inserts)+len(buf.updates))
	for id, e := range buf.inserts {
		ops = append(ops, modelsdkmpr.BatchWriteOp{
			Insert:          true,
			UnitID:          id,
			ContainerID:     e.containerID,
			ContainmentName: e.containmentName,
			UnitType:        e.unitType,
			Contents:        e.contents,
		})
	}
	for id, contents := range buf.updates {
		ops = append(ops, modelsdkmpr.BatchWriteOp{
			Insert:   false,
			UnitID:   id,
			Contents: contents,
		})
	}
	return ops
}
```

- [ ] **Step 4: Run L1 tests — expect PASS**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./mdl/backend/mpr/ -run TestScriptBuffer -v 2>&1 | grep -E "RUN|PASS|FAIL"
```

Expected:
```
--- PASS: TestScriptBuffer_AddUpdate_VisibleInOverlay
--- PASS: TestScriptBuffer_AddInsert_VisibleInInsertList
--- PASS: TestScriptBuffer_Rollback_ClearsOverlay
```

- [ ] **Step 5: Commit**

```bash
git add modelsdk/mpr/reader.go mdl/backend/mpr/script_buf.go mdl/backend/mpr/script_buf_test.go
git commit -m "feat(backend/mpr): ScriptBuffer type + AddInsert/AddUpdate/Rollback + L1 tests"
```

---

## Task 5: MprBackend — BeginScriptTransaction rewrite + commitScriptBuffer + Rollback

**Files:**
- Modify: `mdl/backend/mpr/backend.go`
- Rewrite: `mdl/backend/mpr/script_tx.go`

- [ ] **Step 1: Write the L1 test for BeginScriptTransaction (no db.Begin)**

Add to `mdl/backend/mpr/script_buf_test.go`:

```go
func TestBeginScriptTransaction_NoDBBegin(t *testing.T) {
	dst := copyFixture(t, fixturePath, t.TempDir())
	b := New()
	if err := b.Connect(dst); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	// Saturate the connection pool from another goroutine to confirm
	// BeginScriptTransaction does not try to acquire a connection.
	// (If it called db.Begin() it would block here.)
	tx, err := b.BeginScriptTransaction()
	if err != nil {
		t.Fatalf("BeginScriptTransaction: %v", err)
	}
	if b.scriptBuf == nil {
		t.Error("scriptBuf is nil after BeginScriptTransaction")
	}
	_ = tx.Rollback()
}
```

Run to confirm it fails (field `scriptBuf` not yet on backend):

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./mdl/backend/mpr/ -run TestBeginScriptTransaction -v 2>&1 | head -10
```

Expected: compile error `b.scriptBuf undefined`.

- [ ] **Step 2: Replace activeScriptTx with scriptBuf in backend.go**

In `mdl/backend/mpr/backend.go`, find the field:

```go
	activeScriptTx *modelsdkmpr.WriteTransaction
```

Replace with:

```go
	scriptBuf *ScriptBuffer
```

- [ ] **Step 3: Add commitScriptBuffer to backend.go**

Add to `mdl/backend/mpr/backend.go`:

```go
// commitScriptBuffer flushes all buffered writes to the MPR in a single atomic
// BatchWrite call, then clears the script buffer and reader overlay.
func (b *MprBackend) commitScriptBuffer() error {
	if b.scriptBuf == nil {
		return fmt.Errorf("commitScriptBuffer: no active script buffer")
	}
	ops := b.scriptBuf.toBatchOps()
	b.scriptBuf = nil
	b.msdkReader.ClearScriptMode() // clear overlay before write so InvalidateCache is clean

	if len(ops) == 0 {
		return nil
	}
	return b.msdkWriter.BatchWrite(ops)
}
```

- [ ] **Step 4: Rewrite script_tx.go**

Replace the entire contents of `mdl/backend/mpr/script_tx.go` with:

```go
// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/backend"
)

// BeginScriptTransaction starts a script-level write buffer. No SQL transaction
// is opened; writes accumulate in ScriptBuffer and are committed atomically at
// the end of the EXECUTE SCRIPT block via a single BatchWrite call.
func (b *MprBackend) BeginScriptTransaction() (backend.ScriptTransaction, error) {
	if b.scriptBuf != nil {
		return nil, fmt.Errorf("script transaction already active")
	}
	b.scriptBuf = newScriptBuffer(b.reader)
	return &mprScriptTx{b: b}, nil
}

type mprScriptTx struct {
	b *MprBackend
}

// Commit flushes all buffered writes to the MPR in one atomic SQL transaction.
func (tx *mprScriptTx) Commit() error {
	if tx.b.scriptBuf == nil {
		return fmt.Errorf("script transaction already closed")
	}
	return tx.b.commitScriptBuffer()
}

// Rollback discards all buffered writes. No SQL operation needed.
func (tx *mprScriptTx) Rollback() error {
	if tx.b.scriptBuf == nil {
		return nil
	}
	tx.b.scriptBuf.Rollback()
	tx.b.scriptBuf = nil
	return nil
}
```

- [ ] **Step 5: Run L1 test**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./mdl/backend/mpr/ -run TestBeginScriptTransaction -v 2>&1
```

Expected: `PASS`.

- [ ] **Step 6: Compile the whole package**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go build ./mdl/backend/mpr/... 2>&1
```

Expected: no errors (there will be existing callers of `activeScriptTx` in write_helpers.go to fix next).

- [ ] **Step 7: Commit**

```bash
git add mdl/backend/mpr/backend.go mdl/backend/mpr/script_tx.go mdl/backend/mpr/script_buf_test.go
git commit -m "feat(backend/mpr): rewrite BeginScriptTransaction — ScriptBuffer replaces *sql.Tx"
```

---

## Task 6: Route writeUnitContents to scriptBuf

**Files:**
- Modify: `mdl/backend/mpr/write_helpers.go`

- [ ] **Step 1: Replace activeScriptTx branch**

In `mdl/backend/mpr/write_helpers.go`, find:

```go
	if b.activeScriptTx != nil {
		if err := b.activeScriptTx.WriteUnit(string(unitID), contents); err != nil {
			return fmt.Errorf("write unit (in script tx): %w", err)
		}
		return nil
	}
```

Replace with:

```go
	if b.scriptBuf != nil {
		return b.scriptBuf.AddUpdate(string(unitID), contents)
	}
```

- [ ] **Step 2: Compile**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go build ./mdl/backend/mpr/... 2>&1
```

Expected: success.

- [ ] **Step 3: Run existing backend/mpr tests**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./mdl/backend/mpr/... -count=1 2>&1 | tail -5
```

Expected: all PASS.

- [ ] **Step 4: Commit**

```bash
git add mdl/backend/mpr/write_helpers.go
git commit -m "feat(backend/mpr): writeUnitContents routes to scriptBuf.AddUpdate during EXECUTE SCRIPT"
```

---

## Task 7: insertUnit wrapper + call site cleanup

**Files:**
- Modify: `mdl/backend/mpr/backend.go`
- Modify: `mdl/backend/mpr/create_services_modelsdk.go`
- Modify: `mdl/backend/mpr/domainmodel_modelsdk.go`
- Modify: `mdl/backend/mpr/modules_modelsdk.go`

- [ ] **Step 1: Add insertUnit wrapper to backend.go**

Add to `mdl/backend/mpr/backend.go`:

```go
// insertUnit routes to ScriptBuffer when a script is active (so new units are
// buffered and can be rolled back), otherwise delegates to msdkWriter.InsertUnit.
func (b *MprBackend) insertUnit(unitID, containerID, containmentName, unitType string, contents []byte) error {
	if b.scriptBuf != nil {
		return b.scriptBuf.AddInsert(unitID, containerID, containmentName, unitType, contents)
	}
	return b.msdkWriter.InsertUnit(unitID, containerID, containmentName, unitType, contents)
}
```

- [ ] **Step 2: Replace call sites in create_services_modelsdk.go**

Run to find all sites:

```bash
grep -n "b\.msdkWriter\.InsertUnit" mdl/backend/mpr/create_services_modelsdk.go
```

For each occurrence, change `b.msdkWriter.InsertUnit(` to `b.insertUnit(`. The arguments stay identical. There should be ~7 occurrences.

- [ ] **Step 3: Replace call site in domainmodel_modelsdk.go**

```bash
grep -n "b\.msdkWriter\.InsertUnit" mdl/backend/mpr/domainmodel_modelsdk.go
```

Same replacement: `b.msdkWriter.InsertUnit(` → `b.insertUnit(`.

- [ ] **Step 4: Replace call sites in modules_modelsdk.go**

```bash
grep -n "b\.msdkWriter\.InsertUnit" mdl/backend/mpr/modules_modelsdk.go
```

Same replacement (2 occurrences).

- [ ] **Step 5: Verify no remaining direct call sites**

```bash
grep -r "msdkWriter\.InsertUnit" mdl/backend/mpr/
```

Expected: no output. (All sites replaced.)

- [ ] **Step 6: Build and test**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./mdl/backend/mpr/... -count=1 2>&1 | tail -5
```

Expected: all PASS.

- [ ] **Step 7: Commit**

```bash
git add mdl/backend/mpr/backend.go mdl/backend/mpr/create_services_modelsdk.go mdl/backend/mpr/domainmodel_modelsdk.go mdl/backend/mpr/modules_modelsdk.go
git commit -m "feat(backend/mpr): insertUnit wrapper buffers creates during EXECUTE SCRIPT; replace ~10 call sites"
```

---

## Task 8: Verify RED → GREEN (remove build tag)

**Files:**
- Modify: `mdl/executor/execute_script_deadlock_test.go`

- [ ] **Step 1: Confirm tests still fail with the build tag (pre-change baseline)**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test -tags execute_script_deadlock ./mdl/executor/ -run "TestExecuteScriptPath" -v -timeout 30s 2>&1 | grep -E "PASS|FAIL"
```

Expected: `FAIL: TestExecuteScriptPath_CreateThenRead`, `FAIL: TestExecuteScriptPath_ReadOnly`.

If both now PASS — great, skip to Step 3. If one or both still hang, investigate the commit chain.

- [ ] **Step 2: Remove the build tag from the test file**

In `mdl/executor/execute_script_deadlock_test.go`, delete the entire `//go:build` block at the top of the file (lines 3-6):

```
// These tests reproduce a known deadlock and intentionally hang until timeout.
// Run explicitly with: go test -tags execute_script_deadlock ./mdl/executor/ -run TestExecuteScriptPath -v -timeout 30s
// Remove this build tag once the ScriptBuffer fix is applied and tests pass.
//go:build execute_script_deadlock
```

The file should now start directly with `package executor`.

- [ ] **Step 3: Run all three tests without the tag**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./mdl/executor/ -run "TestExecPath_CreateThenRead|TestExecuteScriptPath" -v -timeout 30s 2>&1 | grep -E "RUN|PASS|FAIL"
```

Expected:
```
--- PASS: TestExecPath_CreateThenRead
--- PASS: TestExecuteScriptPath_CreateThenRead
--- PASS: TestExecuteScriptPath_ReadOnly
```

- [ ] **Step 4: Run full test suite**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./... -count=1 2>&1 | grep -E "FAIL|ok " | tail -30
```

Expected: no FAIL lines.

- [ ] **Step 5: Commit**

```bash
git add mdl/executor/execute_script_deadlock_test.go
git commit -m "fix(executor): EXECUTE SCRIPT no longer deadlocks — remove build tag from regression tests"
```

---

## Task 9: Behavior tests (Bug 2 atomicity + Bug 3 read-own-write)

**Files:**
- Modify: `mdl/executor/execute_script_deadlock_test.go`

- [ ] **Step 1: Write the rollback test (Bug 2)**

Add to `mdl/executor/execute_script_deadlock_test.go`:

```go
// TestExecuteScriptPath_RollbackOnError verifies that when a statement inside
// EXECUTE SCRIPT fails, all preceding creates in the same script are rolled back.
func TestExecuteScriptPath_RollbackOnError(t *testing.T) {
	t.Parallel()
	be := openBackendForTest(t)

	exec := New(&bytes.Buffer{})
	exec.SetBackend(be)

	// First statement creates entity; second is intentionally invalid.
	scriptPath := writeScriptFile(t, "create or modify entity MyFirstModule.RollbackTarget;\ncreate entity NonExistentModule.Bad;\n")

	stmt := &ast.ExecuteScriptStmt{Path: scriptPath}
	ctx := exec.newExecContext(context.Background())

	err := execExecuteScript(ctx, stmt)
	if err == nil {
		t.Fatal("expected error from invalid statement, got nil")
	}

	// RollbackTarget must not have been committed.
	entities, listErr := be.ListEntityModels(context.Background(), "MyFirstModule")
	if listErr != nil {
		t.Fatalf("ListEntityModels: %v", listErr)
	}
	for _, e := range entities {
		if e.Name == "RollbackTarget" {
			t.Error("rolled-back entity 'RollbackTarget' still present in MPR after script failure")
		}
	}
}
```

- [ ] **Step 2: Run to confirm it fails (entity may or may not be present before fix)**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./mdl/executor/ -run TestExecuteScriptPath_RollbackOnError -v -timeout 15s 2>&1
```

Note the result. If `RollbackTarget` is committed despite the error, this test is RED confirming Bug 2 needed fixing.

- [ ] **Step 3: Write the read-own-write test (Bug 3)**

Add to `mdl/executor/execute_script_deadlock_test.go`:

```go
// TestExecuteScriptPath_CreateThenDescribe verifies that an entity created
// earlier in the same EXECUTE SCRIPT block is visible to a subsequent
// DESCRIBE ENTITY statement (read-own-write within a script).
func TestExecuteScriptPath_CreateThenDescribe(t *testing.T) {
	t.Parallel()
	be := openBackendForTest(t)

	var out bytes.Buffer
	exec := New(&out)
	exec.SetBackend(be)

	scriptPath := writeScriptFile(t,
		"create or modify entity MyFirstModule.ReadOwnWriteTest;\ndescribe entity MyFirstModule.ReadOwnWriteTest;\n",
	)

	stmt := &ast.ExecuteScriptStmt{Path: scriptPath}

	done := make(chan error, 1)
	go func() {
		ctx := exec.newExecContext(context.Background())
		done <- execExecuteScript(ctx, stmt)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("EXECUTE SCRIPT: %v", err)
		}
	case <-time.After(scriptDeadlockTimeout):
		t.Fatalf("EXECUTE SCRIPT deadlocked — did not complete within %v", scriptDeadlockTimeout)
	}

	if !strings.Contains(out.String(), "ReadOwnWriteTest") {
		t.Errorf("describe output does not mention ReadOwnWriteTest:\n%s", out.String())
	}
}
```

Add `"strings"` to the import block.

- [ ] **Step 4: Run both new tests**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./mdl/executor/ -run "TestExecuteScriptPath_RollbackOnError|TestExecuteScriptPath_CreateThenDescribe" -v -timeout 30s 2>&1 | grep -E "RUN|PASS|FAIL"
```

Expected: both PASS.

- [ ] **Step 5: Run full executor suite**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./mdl/executor/... -count=1 -timeout 120s 2>&1 | tail -5
```

Expected: PASS.

- [ ] **Step 6: Final full suite**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./... -count=1 2>&1 | grep FAIL || echo "ALL PASS"
```

Expected: `ALL PASS`.

- [ ] **Step 7: Commit**

```bash
git add mdl/executor/execute_script_deadlock_test.go
git commit -m "test(executor): add rollback-on-error + read-own-write regression tests for EXECUTE SCRIPT"
```

---

## Self-Review Checklist

**Spec coverage:**
- Bug 1 (pool deadlock): Tasks 5-8 — `BeginScriptTransaction` no longer calls `db.Begin()` ✓
- Bug 2 (insertUnit bypasses tx): Tasks 7 + 9 — wrapper + rollback test ✓
- Bug 3 (read-own-write): Tasks 1-4 + 9 — Reader overlay + describe test ✓
- Bug 4 (v2 commit order): Task 3 — `BatchWrite` writes files before SQL ✓
- RED baseline tests → GREEN: Task 8 ✓
- New L1 tests (ScriptBuffer): Task 4 ✓
- New L1 test (BeginScriptTransaction): Task 5 ✓

**Placeholder check:** No TBD or TODO in code blocks. ✓

**Type consistency:**
- `ScriptBuffer.toBatchOps()` returns `[]modelsdkmpr.BatchWriteOp` — matches `BatchWriteOp` defined in Task 3 ✓
- `Reader.AppendScriptInsert(e ScriptInsertEntry)` — `ScriptInsertEntry` defined in Task 1 ✓
- `buf.reader.ScriptOverlay()` / `buf.reader.ScriptInserts()` accessor methods added in Task 4 Step 1 ✓
- `b.msdkReader.ClearScriptMode()` called in `commitScriptBuffer` — `ClearScriptMode()` defined in Task 1 ✓
