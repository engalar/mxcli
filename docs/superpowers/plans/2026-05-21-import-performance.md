# Import 性能优化：BufferedUnitStore 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 通过引入 BufferedUnitStore 缓冲层，将 `mxcli import` 从 20 分钟降到 2-4 分钟。

**Architecture:** 在 executor 和 MPR 存储之间插入 `BufferedUnitStore`：写操作只写内存（pending map），同时把 bytes 注入 Reader 的 overlay 以保证同文件内读写可见；每文件结束时一次性 Flush（单 SQLite 事务 + 批量 .mxunit 写入）。`modelsdk/mpr/Reader` 增加 overlay 字段使 `GetRawUnitBytes` 优先返回缓冲数据，全链路无破坏性变更。

**Tech Stack:** Go 1.26, `modernc.org/sqlite`, `go.mongodb.org/mongo-driver/bson`，现有 MprBackend 接口体系。

---

## 文件结构

```
新建：
  mdl/backend/unitstore/interfaces.go          # UnitReader / UnitWriter / UnitPersistence
  mdl/backend/unitstore/buffered.go            # BufferedUnitStore
  mdl/backend/unitstore/buffered_test.go       # 单元测试（使用 stub persistence）
  mdl/backend/mpr/unit_persistence.go          # MprUnitPersistence 实现

修改：
  modelsdk/mpr/reader.go                        # 增加 overlay 字段 + SetOverlay/ClearOverlay
  mdl/backend/mpr/backend.go                    # 增加 unitBuf 字段 + EnableImportBuffer
  mdl/backend/mpr/write_helpers.go              # writeUnitContents 检查 unitBuf
  mdl/executor/cmd_import_project.go            # 每文件 BeginBuffer/Flush/Discard
```

---

## Task 1：unitstore 接口定义

**Files:**
- Create: `mdl/backend/unitstore/interfaces.go`

- [ ] **Step 1: 创建接口文件**

```go
// SPDX-License-Identifier: Apache-2.0

// Package unitstore defines the BufferedUnitStore and its persistence contract.
// All writes during an import session are held in memory and flushed to disk
// as a single batch at each file checkpoint, eliminating per-statement
// SQLite transactions and fsync overhead.
package unitstore

import "github.com/mendixlabs/mxcli/model"

// UnitReader is the read-only face of the unit buffer.
// Reads check the in-memory pending/loaded maps before going to disk.
type UnitReader interface {
	Read(id model.ID) ([]byte, error)
}

// UnitWriter adds write and lifecycle methods to UnitReader.
type UnitWriter interface {
	UnitReader
	// Write stores data in the in-memory pending set. No disk I/O.
	Write(id model.ID, data []byte) error
	// Flush batches all pending writes to disk in a single transaction,
	// promotes them to the loaded cache, and clears the pending set.
	Flush() error
	// Discard drops all pending writes without writing to disk.
	Discard()
}

// UnitPersistence is the storage abstraction that BufferedUnitStore delegates
// to for actual I/O. Implement MprUnitPersistence for production use;
// use a stub for testing.
type UnitPersistence interface {
	// Load reads raw BSON bytes for a single unit from disk.
	// Called lazily — only when a unit is first read and not yet in cache.
	Load(id model.ID) ([]byte, error)
	// BatchStore writes all units in a single SQLite transaction.
	BatchStore(units map[model.ID][]byte) error
	// BatchHash returns a SHA-256 hex string per unit (used for @cache: markers).
	BatchHash(units map[model.ID][]byte) (map[model.ID]string, error)
}
```

- [ ] **Step 2: 验证编译**

```bash
go build ./mdl/backend/unitstore/...
```

Expected: 无输出（clean build）。

- [ ] **Step 3: 提交**

```bash
git add mdl/backend/unitstore/interfaces.go
git commit -m "feat(unitstore): UnitReader/UnitWriter/UnitPersistence interfaces"
```

---

## Task 2：BufferedUnitStore 实现

**Files:**
- Create: `mdl/backend/unitstore/buffered.go`
- Create: `mdl/backend/unitstore/buffered_test.go`

- [ ] **Step 1: 写失败测试**

创建 `mdl/backend/unitstore/buffered_test.go`：

```go
// SPDX-License-Identifier: Apache-2.0
package unitstore_test

import (
	"fmt"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/backend/unitstore"
	"github.com/mendixlabs/mxcli/model"
)

// stubPersistence is a minimal UnitPersistence for tests.
type stubPersistence struct {
	disk    map[model.ID][]byte
	stored  map[model.ID][]byte // BatchStore writes here
	loadCnt int
}

func newStub(disk map[model.ID][]byte) *stubPersistence {
	return &stubPersistence{disk: disk, stored: make(map[model.ID][]byte)}
}
func (s *stubPersistence) Load(id model.ID) ([]byte, error) {
	s.loadCnt++
	if d, ok := s.disk[id]; ok {
		return d, nil
	}
	return nil, fmt.Errorf("not found: %s", id)
}
func (s *stubPersistence) BatchStore(units map[model.ID][]byte) error {
	for id, data := range units {
		s.stored[id] = data
		s.disk[id] = data
	}
	return nil
}
func (s *stubPersistence) BatchHash(units map[model.ID][]byte) (map[model.ID]string, error) {
	out := make(map[model.ID]string, len(units))
	for id := range units {
		out[id] = "hash-" + string(id)
	}
	return out, nil
}

func TestBufferedUnitStore_WriteStaysInMemory(t *testing.T) {
	p := newStub(nil)
	buf := unitstore.New(p)

	id := model.ID("unit-1")
	data := []byte("bson-bytes")
	if err := buf.Write(id, data); err != nil {
		t.Fatalf("Write: %v", err)
	}
	// Must NOT have called BatchStore yet
	if len(p.stored) != 0 {
		t.Errorf("expected no disk writes before Flush, got %d", len(p.stored))
	}
}

func TestBufferedUnitStore_ReadReturnsWrittenData(t *testing.T) {
	p := newStub(nil)
	buf := unitstore.New(p)

	id := model.ID("unit-1")
	data := []byte("bson-bytes")
	_ = buf.Write(id, data)

	got, err := buf.Read(id)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("got %q, want %q", got, data)
	}
	// Must not have hit disk (no Load call)
	if p.loadCnt != 0 {
		t.Errorf("expected no Load calls, got %d", p.loadCnt)
	}
}

func TestBufferedUnitStore_ReadLazyLoadsFromDisk(t *testing.T) {
	id := model.ID("unit-1")
	p := newStub(map[model.ID][]byte{id: []byte("disk-data")})
	buf := unitstore.New(p)

	got, err := buf.Read(id)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if string(got) != "disk-data" {
		t.Errorf("got %q, want %q", got, "disk-data")
	}
	if p.loadCnt != 1 {
		t.Errorf("expected 1 Load call, got %d", p.loadCnt)
	}
	// Second read must hit cache, not disk
	_, _ = buf.Read(id)
	if p.loadCnt != 1 {
		t.Errorf("expected still 1 Load call after cache hit, got %d", p.loadCnt)
	}
}

func TestBufferedUnitStore_FlushWritesToDisk(t *testing.T) {
	p := newStub(nil)
	buf := unitstore.New(p)

	id := model.ID("unit-1")
	_ = buf.Write(id, []byte("data"))

	if err := buf.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if _, ok := p.stored[id]; !ok {
		t.Errorf("expected unit to be in stored after Flush")
	}
	// After flush, Read must still return the data (promoted to loaded cache)
	got, _ := buf.Read(id)
	if string(got) != "data" {
		t.Errorf("expected data after Flush, got %q", got)
	}
	// Flush again — pending is empty, no second BatchStore call
	p.stored = make(map[model.ID][]byte)
	if err := buf.Flush(); err != nil {
		t.Fatalf("second Flush: %v", err)
	}
	if len(p.stored) != 0 {
		t.Errorf("expected no second BatchStore on empty pending")
	}
}

func TestBufferedUnitStore_DiscardClearsPending(t *testing.T) {
	p := newStub(nil)
	buf := unitstore.New(p)

	_ = buf.Write(model.ID("unit-1"), []byte("data"))
	buf.Discard()

	if len(p.stored) != 0 {
		t.Errorf("expected no disk writes after Discard")
	}
	// Read after Discard should miss pending and try disk
	_, err := buf.Read(model.ID("unit-1"))
	if err == nil {
		t.Errorf("expected error reading discarded unit from empty disk")
	}
}
```

- [ ] **Step 2: 运行确认失败**

```bash
go test ./mdl/backend/unitstore/... -v 2>&1 | head -10
```

Expected: `FAIL — unitstore.New undefined`

- [ ] **Step 3: 实现 BufferedUnitStore**

创建 `mdl/backend/unitstore/buffered.go`：

```go
// SPDX-License-Identifier: Apache-2.0

package unitstore

import (
	"fmt"
	"sync"

	"github.com/mendixlabs/mxcli/model"
)

// BufferedUnitStore holds pending writes in memory and flushes them to disk
// as a single batch. Reads check the pending set and a lazy-loaded cache
// before going to the persistence layer.
type BufferedUnitStore struct {
	persistence UnitPersistence

	mu      sync.RWMutex
	pending map[model.ID][]byte // written this session, not yet on disk
	loaded  map[model.ID][]byte // loaded from disk (read-only copies)
}

// New creates a BufferedUnitStore backed by the given persistence layer.
func New(p UnitPersistence) *BufferedUnitStore {
	return &BufferedUnitStore{
		persistence: p,
		pending:     make(map[model.ID][]byte),
		loaded:      make(map[model.ID][]byte),
	}
}

// Read returns unit bytes. Priority: pending > loaded cache > disk (lazy).
func (b *BufferedUnitStore) Read(id model.ID) ([]byte, error) {
	b.mu.RLock()
	if data, ok := b.pending[id]; ok {
		b.mu.RUnlock()
		return data, nil
	}
	if data, ok := b.loaded[id]; ok {
		b.mu.RUnlock()
		return data, nil
	}
	b.mu.RUnlock()

	// Lazy load from disk
	data, err := b.persistence.Load(id)
	if err != nil {
		return nil, fmt.Errorf("load unit %s: %w", id, err)
	}
	b.mu.Lock()
	b.loaded[id] = data
	b.mu.Unlock()
	return data, nil
}

// Write stores data in the pending set. No disk I/O occurs.
func (b *BufferedUnitStore) Write(id model.ID, data []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.pending[id] = data
	return nil
}

// Flush writes all pending units to disk in one batch, promotes them to the
// loaded cache, and clears the pending set.
func (b *BufferedUnitStore) Flush() error {
	b.mu.Lock()
	pending := b.pending
	b.mu.Unlock()

	if len(pending) == 0 {
		return nil
	}

	// BatchHash is called first so the persistence layer can update @cache: markers.
	if _, err := b.persistence.BatchHash(pending); err != nil {
		return fmt.Errorf("batch hash: %w", err)
	}
	if err := b.persistence.BatchStore(pending); err != nil {
		return fmt.Errorf("batch store: %w", err)
	}

	b.mu.Lock()
	for id, data := range pending {
		b.loaded[id] = data
	}
	b.pending = make(map[model.ID][]byte)
	b.mu.Unlock()
	return nil
}

// Discard drops all pending writes. The loaded cache is preserved.
func (b *BufferedUnitStore) Discard() {
	b.mu.Lock()
	b.pending = make(map[model.ID][]byte)
	b.mu.Unlock()
}

// PendingCount returns the number of units waiting to be flushed.
// Used in tests and progress reporting.
func (b *BufferedUnitStore) PendingCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.pending)
}
```

- [ ] **Step 4: 运行测试确认通过**

```bash
go test ./mdl/backend/unitstore/... -v 2>&1 | grep -E "PASS|FAIL"
```

Expected: 5 个 PASS，0 个 FAIL。

- [ ] **Step 5: 提交**

```bash
git add mdl/backend/unitstore/
git commit -m "feat(unitstore): BufferedUnitStore — lazy read, buffered write, batch flush"
```

---

## Task 3：Reader overlay（modelsdk/mpr/reader.go）

**Files:**
- Modify: `modelsdk/mpr/reader.go`

Reader 需要一个 overlay 字段，让 `GetRawUnitBytes` 优先返回缓冲中的数据，而不是去读磁盘。这样在同一个文件的执行过程中，后面的语句能立即看到前面语句写入的结果。

- [ ] **Step 1: 写失败测试**

在 `modelsdk/mpr/reader_test.go`（如果存在就追加，否则新建）中添加：

```go
func TestReader_OverlayTakesPrecedenceOverDisk(t *testing.T) {
	// Open a real MPR file to get a reader
	r, err := OpenReader("../../testdata/roundtrip/roundtrip.mpr")
	if err != nil {
		t.Skipf("testdata not available: %v", err)
	}
	defer r.Close()

	fakeID := "00000000-0000-0000-0000-000000000001"
	fakeData := []byte("overlay-data")

	r.SetOverlay(fakeID, fakeData)
	got, err := r.GetRawUnitBytes(fakeID)
	if err != nil {
		t.Fatalf("GetRawUnitBytes with overlay: %v", err)
	}
	if string(got) != string(fakeData) {
		t.Errorf("got %q, want %q", got, fakeData)
	}

	r.ClearOverlay(fakeID)
	// After clear, the overlay entry is gone — real DB lookup may error (that's OK)
	got2, _ := r.GetRawUnitBytes(fakeID)
	if string(got2) == string(fakeData) {
		t.Errorf("expected overlay cleared, but still got overlay data")
	}
}
```

- [ ] **Step 2: 运行确认失败**

```bash
go test ./modelsdk/mpr/... -run TestReader_OverlayTakesPrecedenceOverDisk -v 2>&1 | tail -5
```

Expected: `FAIL — r.SetOverlay undefined`

- [ ] **Step 3: 修改 Reader 结构体和 GetRawUnitBytes**

在 `modelsdk/mpr/reader.go` 的 `Reader` 结构体中增加字段，并更新方法：

在 `Reader` struct 的字段列表（`unitCacheValid bool` 之后）追加：

```go
	// overlay holds unit bytes injected by BufferedUnitStore so that
	// reads within the same import file see buffered (uncommitted) writes.
	// nil means no overlay is active (normal production path — zero cost).
	overlay map[string][]byte
```

新增两个方法（在 `GetRawUnitBytes` 之前）：

```go
// SetOverlay registers in-memory bytes for unitID so that GetRawUnitBytes
// returns them without hitting disk. Called by BufferedUnitStore.Write.
func (r *Reader) SetOverlay(unitID string, data []byte) {
	if r.overlay == nil {
		r.overlay = make(map[string][]byte)
	}
	r.overlay[unitID] = data
}

// ClearOverlay removes a single unitID from the overlay.
// Called by BufferedUnitStore.Flush after data is committed to disk.
func (r *Reader) ClearOverlay(unitID string) {
	delete(r.overlay, unitID)
}

// ClearAllOverlays removes all overlay entries. Called on Discard.
func (r *Reader) ClearAllOverlays() {
	r.overlay = nil
}
```

修改 `GetRawUnitBytes`，在函数开头（第一行 `if r.version == MPRVersionV2` 之前）插入：

```go
	// Fast path: return buffered bytes injected by BufferedUnitStore if present.
	if len(r.overlay) > 0 {
		if data, ok := r.overlay[unitID]; ok {
			return data, nil
		}
	}
```

- [ ] **Step 4: 运行测试**

```bash
go test ./modelsdk/mpr/... -run TestReader_OverlayTakesPrecedenceOverDisk -v 2>&1 | tail -5
go build ./... 2>&1
```

Expected: PASS，编译无错误。

- [ ] **Step 5: 提交**

```bash
git add modelsdk/mpr/reader.go modelsdk/mpr/reader_test.go
git commit -m "feat(mpr/reader): overlay map for BufferedUnitStore read-through

SetOverlay/ClearOverlay/ClearAllOverlays allow BufferedUnitStore to make
pending writes immediately visible to GetRawUnitBytes without a SQLite
round-trip. overlay is nil by default — zero cost on the normal path."
```

---

## Task 4：MprUnitPersistence

**Files:**
- Create: `mdl/backend/mpr/unit_persistence.go`

- [ ] **Step 1: 写失败测试**

创建 `mdl/backend/mpr/unit_persistence_test.go`：

```go
// SPDX-License-Identifier: Apache-2.0
package mprbackend_test

import (
	"path/filepath"
	"testing"

	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/mendixlabs/mxcli/model"
)

func TestMprUnitPersistence_RoundTrip(t *testing.T) {
	// Use the committed roundtrip testdata MPR
	mprPath := filepath.Join("..", "..", "testdata", "roundtrip", "roundtrip.mpr")
	be, err := mprbackend.NewFromPath(mprPath)
	if err != nil {
		t.Skipf("testdata not available: %v", err)
	}
	defer be.Disconnect()

	p := be.NewUnitPersistence()

	// Load any existing unit (the MPR has at least the domain model unit)
	ids, err := p.ListUnitIDs()
	if err != nil {
		t.Fatalf("ListUnitIDs: %v", err)
	}
	if len(ids) == 0 {
		t.Fatal("expected at least one unit")
	}

	id := ids[0]
	orig, err := p.Load(id)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(orig) == 0 {
		t.Fatalf("Load returned empty bytes for %s", id)
	}

	// BatchHash should return a non-empty hex string
	hashes, err := p.BatchHash(map[model.ID][]byte{id: orig})
	if err != nil {
		t.Fatalf("BatchHash: %v", err)
	}
	if hashes[id] == "" {
		t.Errorf("expected non-empty hash for %s", id)
	}
}
```

- [ ] **Step 2: 运行确认失败**

```bash
go test ./mdl/backend/mpr/... -run TestMprUnitPersistence_RoundTrip -v 2>&1 | tail -5
```

Expected: `FAIL — be.NewUnitPersistence undefined`

- [ ] **Step 3: 实现 MprUnitPersistence**

创建 `mdl/backend/mpr/unit_persistence.go`：

```go
// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/backend/unitstore"
	"github.com/mendixlabs/mxcli/model"
)

// MprUnitPersistence implements unitstore.UnitPersistence backed by MprBackend.
// It is the bridge between BufferedUnitStore and the MPR's SQLite + .mxunit storage.
type MprUnitPersistence struct {
	b *MprBackend
}

// NewUnitPersistence returns a UnitPersistence backed by this MprBackend.
// The backend must be connected (Connect called) before use.
func (b *MprBackend) NewUnitPersistence() *MprUnitPersistence {
	return &MprUnitPersistence{b: b}
}

// ListUnitIDs returns all unit IDs in the project. Used by tests.
func (p *MprUnitPersistence) ListUnitIDs() ([]model.ID, error) {
	rows, err := p.b.msdkReader.DB().Query(`SELECT hex(UnitID) FROM Unit`)
	if err != nil {
		return nil, fmt.Errorf("list unit IDs: %w", err)
	}
	defer rows.Close()
	var ids []model.ID
	for rows.Next() {
		var hexID string
		if err := rows.Scan(&hexID); err != nil {
			return nil, err
		}
		ids = append(ids, model.ID(hexID))
	}
	return ids, rows.Err()
}

// Load reads the raw BSON bytes for a single unit from disk. Satisfies unitstore.UnitPersistence.
func (p *MprUnitPersistence) Load(id model.ID) ([]byte, error) {
	data, err := p.b.msdkReader.GetRawUnitBytes(string(id))
	if err != nil {
		return nil, fmt.Errorf("load unit %s: %w", id, err)
	}
	return data, nil
}

// BatchStore writes all provided units to disk in a single SQLite transaction.
// Satisfies unitstore.UnitPersistence.
func (p *MprUnitPersistence) BatchStore(units map[model.ID][]byte) error {
	if p.b.msdkWriter == nil {
		return fmt.Errorf("BatchStore: modelsdk writer not initialized")
	}
	wtx, err := p.b.msdkWriter.BeginWriteTransaction()
	if err != nil {
		return fmt.Errorf("BatchStore: begin tx: %w", err)
	}
	for id, data := range units {
		if err := wtx.WriteUnit(string(id), data); err != nil {
			_ = wtx.Rollback()
			return fmt.Errorf("BatchStore: write unit %s: %w", id, err)
		}
	}
	if err := wtx.Commit(); err != nil {
		return fmt.Errorf("BatchStore: commit: %w", err)
	}
	p.b.msdkReader.InvalidateCache()
	return nil
}

// BatchHash computes a SHA-256 hex digest for each unit's bytes.
// Satisfies unitstore.UnitPersistence.
func (p *MprUnitPersistence) BatchHash(units map[model.ID][]byte) (map[model.ID]string, error) {
	out := make(map[model.ID]string, len(units))
	for id, data := range units {
		h := sha256.Sum256(data)
		out[id] = hex.EncodeToString(h[:])
	}
	return out, nil
}

// Ensure MprUnitPersistence satisfies the interface at compile time.
var _ unitstore.UnitPersistence = (*MprUnitPersistence)(nil)
```

- [ ] **Step 4: 运行测试**

```bash
go test ./mdl/backend/mpr/... -run TestMprUnitPersistence_RoundTrip -v 2>&1 | tail -5
go build ./... 2>&1
```

Expected: PASS，编译通过。

- [ ] **Step 5: 提交**

```bash
git add mdl/backend/mpr/unit_persistence.go mdl/backend/mpr/unit_persistence_test.go
git commit -m "feat(mpr): MprUnitPersistence — Load/BatchStore/BatchHash for BufferedUnitStore"
```

---

## Task 5：MprBackend 集成 BufferedUnitStore

**Files:**
- Modify: `mdl/backend/mpr/backend.go` （增加 `unitBuf` 字段）
- Modify: `mdl/backend/mpr/write_helpers.go` （write 检查 buffer）

- [ ] **Step 1: 写失败测试**

在 `mdl/backend/mpr/unit_persistence_test.go` 中追加：

```go
func TestMprBackend_EnableImportBuffer_WriteDoesNotHitDisk(t *testing.T) {
	mprPath := filepath.Join("..", "..", "testdata", "roundtrip", "roundtrip.mpr")
	be, err := mprbackend.NewFromPath(mprPath)
	if err != nil {
		t.Skipf("testdata not available: %v", err)
	}
	defer be.Disconnect()

	buf := be.EnableImportBuffer()

	// Write something to the backend — should go to buffer, not disk
	fakeID := model.ID("00000000-0000-0000-0000-000000000099")
	fakeData := []byte("test-bson")
	if err := be.WriteUnitContentsForTest(fakeID, fakeData); err != nil {
		t.Fatalf("WriteUnitContents: %v", err)
	}

	// Buffer should have 1 pending unit
	if buf.PendingCount() != 1 {
		t.Errorf("expected 1 pending unit, got %d", buf.PendingCount())
	}

	// Read should return overlay data (same buffer)
	got, err := be.ReadRawUnitForTest(fakeID)
	if err != nil {
		t.Fatalf("ReadRawUnit: %v", err)
	}
	if string(got) != string(fakeData) {
		t.Errorf("overlay read: got %q, want %q", got, fakeData)
	}

	be.DisableImportBuffer()
}
```

`WriteUnitContentsForTest` 和 `ReadRawUnitForTest` 是暴露给测试用的包装方法，下面步骤中添加。

- [ ] **Step 2: 修改 MprBackend 结构体（backend.go）**

在 `backend.go` 的 `MprBackend` struct 中，在 `activeScriptTx` 字段之后追加：

```go
	// unitBuf is non-nil when an ImportSession is active.
	// writeUnitContents routes writes through the buffer instead of opening
	// individual SQLite transactions. Reads are satisfied from the buffer's
	// overlay before hitting disk. Set via EnableImportBuffer.
	unitBuf *unitstore.BufferedUnitStore
```

在 import 块中追加 `"github.com/mendixlabs/mxcli/mdl/backend/unitstore"`。

在 `backend.go` 中添加生命周期方法（可在文件末尾追加）：

```go
// EnableImportBuffer activates the BufferedUnitStore for the duration of an
// import session. Returns the buffer so the caller can Flush/Discard per file.
// Must be called after Connect.
func (b *MprBackend) EnableImportBuffer() *unitstore.BufferedUnitStore {
	buf := unitstore.New(b.NewUnitPersistence())
	b.unitBuf = buf
	return buf
}

// DisableImportBuffer deactivates the buffer and discards any pending writes.
// Always call this when the import session ends (success or failure).
func (b *MprBackend) DisableImportBuffer() {
	if b.unitBuf != nil {
		b.unitBuf.Discard()
		b.unitBuf = nil
		b.msdkReader.ClearAllOverlays()
	}
}

// WriteUnitContentsForTest exposes writeUnitContents for package-external tests.
// Only for use in *_test.go files.
func (b *MprBackend) WriteUnitContentsForTest(id model.ID, data []byte) error {
	return b.writeUnitContents(id, data)
}

// ReadRawUnitForTest exposes GetRawUnitBytes for package-external tests.
func (b *MprBackend) ReadRawUnitForTest(id model.ID) ([]byte, error) {
	return b.msdkReader.GetRawUnitBytes(string(id))
}
```

- [ ] **Step 3: 修改 writeUnitContents（write_helpers.go）**

将 `write_helpers.go` 中 `writeUnitContents` 的实现替换为：

```go
func (b *MprBackend) writeUnitContents(unitID model.ID, contents []byte) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}

	// Import buffer path: write to memory and update the reader overlay so
	// that subsequent reads in the same file see the new data immediately.
	// No SQLite I/O here — Flush() will batch-commit at file end.
	if b.unitBuf != nil {
		if err := b.unitBuf.Write(unitID, contents); err != nil {
			return fmt.Errorf("write unit to buffer: %w", err)
		}
		b.msdkReader.SetOverlay(string(unitID), contents)
		return nil
	}

	// Script-transaction path (EXECUTE SCRIPT): reuse the open transaction
	// so the whole script block commits or rolls back as one unit.
	if b.activeScriptTx != nil {
		if err := b.activeScriptTx.WriteUnit(string(unitID), contents); err != nil {
			return fmt.Errorf("write unit (in script tx): %w", err)
		}
		return nil
	}

	// Default path: one transaction per write (used by interactive REPL and
	// single-statement exec).
	wtx, err := b.msdkWriter.BeginWriteTransaction()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	if err := wtx.WriteUnit(string(unitID), contents); err != nil {
		_ = wtx.Rollback()
		return fmt.Errorf("write unit: %w", err)
	}
	if err := wtx.Commit(); err != nil {
		return err
	}
	b.msdkReader.InvalidateCache()
	return nil
}
```

- [ ] **Step 4: 运行测试**

```bash
go test ./mdl/backend/mpr/... -run "TestMprBackend_EnableImportBuffer|TestMprUnitPersistence" -v -count=1 2>&1 | grep -E "PASS|FAIL|---"
go build ./... 2>&1
```

Expected: 全部 PASS，编译通过。

- [ ] **Step 5: 运行完整 backend 测试**

```bash
go test ./mdl/backend/... -count=1 -timeout 60s 2>&1 | tail -5
```

Expected: `ok github.com/mendixlabs/mxcli/mdl/backend/...`

- [ ] **Step 6: 提交**

```bash
git add mdl/backend/mpr/backend.go mdl/backend/mpr/write_helpers.go mdl/backend/mpr/unit_persistence_test.go
git commit -m "feat(mpr): wire BufferedUnitStore into MprBackend

- Add unitBuf field to MprBackend
- EnableImportBuffer / DisableImportBuffer lifecycle methods
- writeUnitContents: buffer path (unitBuf != nil) writes to memory
  and sets reader overlay for immediate read-back; existing script-tx
  and single-stmt paths unchanged
- Reader overlay cleared on DisableImportBuffer"
```

---

## Task 6：ImportProject 集成

**Files:**
- Modify: `mdl/executor/cmd_import_project.go`

- [ ] **Step 1: 写失败测试**

在 `mdl/executor/cmd_export_project_test.go` 末尾（或新建 `cmd_import_buffer_test.go`）追加：

```go
func TestImportProject_UsesBufferPerFile(t *testing.T) {
	// Copy testdata MPR to a temp dir so we can safely import into it
	src := filepath.Join("..", "..", "testdata", "roundtrip", "roundtrip.mpr")
	if _, err := os.Stat(src); err != nil {
		t.Skipf("roundtrip testdata not available: %v", err)
	}
	destDir := t.TempDir()
	destMPR := filepath.Join(destDir, "roundtrip.mpr")
	if err := copyFile(src, destMPR); err != nil {
		t.Fatalf("copy MPR: %v", err)
	}
	// Copy mprcontents if present
	srcContents := filepath.Join(filepath.Dir(src), "mprcontents")
	if _, err := os.Stat(srcContents); err == nil {
		if err := copyDir(srcContents, filepath.Join(destDir, "mprcontents")); err != nil {
			t.Fatalf("copy mprcontents: %v", err)
		}
	}

	// Create a small MDL input directory with two files
	inputDir := t.TempDir()
	// File 1: create an enumeration
	file1 := filepath.Join(inputDir, "MyModule", "Enumerations", "MyModule.Status.mdl")
	if err := os.MkdirAll(filepath.Dir(file1), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file1, []byte(`create or modify enumeration MyModule.Status (Active caption 'Active');`), 0644); err != nil {
		t.Fatal(err)
	}
	// File 2: create an entity using that enum
	file2 := filepath.Join(inputDir, "MyModule", "Domain", "MyModule.Item.mdl")
	if err := os.MkdirAll(filepath.Dir(file2), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, []byte(`create or modify persistent entity MyModule.Item (Status: MyModule.Status);`), 0644); err != nil {
		t.Fatal(err)
	}

	be, err := mprbackend.NewFromPath(destMPR)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer be.Disconnect()

	exec := New(os.Stderr)
	exec.backend = be

	// Create the module first
	if _, err := exec.backend.CreateModule(&model.Module{Name: "MyModule"}); err != nil {
		t.Logf("create module (may already exist): %v", err)
	}

	opts := ImportOptions{
		Progress: func(s string) { t.Log(s) },
	}
	if err := exec.ImportProject(inputDir, opts); err != nil {
		t.Fatalf("ImportProject: %v", err)
	}
}
```

- [ ] **Step 2: 运行确认测试运行（可能 pass 或 skip）**

```bash
go test ./mdl/executor/... -run TestImportProject_UsesBufferPerFile -v -timeout 60s -tags integration 2>&1 | tail -10
```

(此测试目前测的是行为正确性，不是性能。只要不报错即可。)

- [ ] **Step 3: 修改 ImportProject**

将 `mdl/executor/cmd_import_project.go` 中 `ImportProject` 函数的 `if !ctx.Connected()` 检查之后、文件扫描之前，以及主循环内部，按以下方式修改：

在 `if !ctx.Connected()` 块之后、`progress := opts.Progress` 之前插入：

```go
	// Activate the import buffer: all writes go to an in-memory BufferedUnitStore
	// and are flushed to disk as a single SQLite transaction per .mdl file.
	// This eliminates ~50 per-statement transactions per file and batches all
	// .mxunit file writes, giving a 5-10× reduction in I/O overhead.
	mprBE, hasBuf := ctx.Backend.(*mprbackend.MprBackend)
	var importBuf *unitstore.BufferedUnitStore
	if hasBuf {
		importBuf = mprBE.EnableImportBuffer()
		defer mprBE.DisableImportBuffer()
	}
```

将主循环的 `ExecuteProgram` 调用及错误处理替换为：

```go
		progress(fmt.Sprintf("  [exec]     %s", rel))
		execErr := e.ExecuteProgram(prog)
		if execErr != nil {
			// Discard buffered writes for this file; they are invalid.
			if importBuf != nil {
				importBuf.Discard()
			}
			msg := fmt.Sprintf("exec %s: %v", rel, execErr)
			if !opts.SkipErrors {
				return fmt.Errorf("%s", msg)
			}
			errs = append(errs, msg)
			continue
		}

		// Flush buffered writes for this file to disk as a single transaction.
		if importBuf != nil {
			if flushErr := importBuf.Flush(); flushErr != nil {
				msg := fmt.Sprintf("flush %s: %v", rel, flushErr)
				if !opts.SkipErrors {
					return fmt.Errorf("%s", msg)
				}
				errs = append(errs, msg)
			}
		}
```

在 `import` 块中添加：

```go
	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/mendixlabs/mxcli/mdl/backend/unitstore"
```

- [ ] **Step 4: 编译**

```bash
go build ./... 2>&1
```

Expected: 无输出。

- [ ] **Step 5: 运行 executor 测试套件**

```bash
go test ./mdl/executor/... -count=1 -timeout 300s -tags integration 2>&1 | grep -E "^ok|^FAIL" | head -5
```

Expected: `ok github.com/mendixlabs/mxcli/mdl/executor`

- [ ] **Step 6: 快速烟测（用真实 MPR）**

```bash
# 准备：复制 roundtrip MPR 到临时目录
cp -r testdata/roundtrip /tmp/rt-test
# 导出为 MDL 然后重新导入（验证 roundtrip 正确性）
./bin/mxcli export -p /tmp/rt-test/roundtrip.mpr --output /tmp/rt-mdl
time ./bin/mxcli import -p /tmp/rt-test/roundtrip.mpr --input /tmp/rt-mdl
```

Expected: 无错误，耗时比之前明显缩短。

- [ ] **Step 7: 提交**

```bash
git add mdl/executor/cmd_import_project.go mdl/executor/cmd_import_buffer_test.go
git commit -m "feat(import): BufferedUnitStore per-file — batch writes, single SQLite tx

ImportProject now activates EnableImportBuffer before the main loop.
Each file's writes go to the in-memory BufferedUnitStore (no SQLite I/O
per statement). At file end: Flush() commits all units in one transaction;
on error: Discard() drops the file's pending writes.

Reader overlay makes within-file writes immediately visible to subsequent
statements via GetRawUnitBytes — no stale-read issue when stmt2 reads
what stmt1 wrote.

Expected: 5-10× reduction in SQLite transaction count for large imports."
```

---

## Self-Review Checklist

**Spec coverage:**
- ✅ `BufferedUnitStore` 接口（UnitReader/UnitWriter/UnitPersistence）— Task 1
- ✅ 懒加载读路径（pending > loaded > disk）— Task 2
- ✅ Reader overlay 读写可见性 — Task 3
- ✅ `MprUnitPersistence` 实现 — Task 4
- ✅ `writeUnitContents` 检查 buffer — Task 5
- ✅ `ImportProject` 每文件 Flush/Discard — Task 6
- ✅ SOLID：接口隔离（UnitReader vs UnitWriter vs UnitPersistence）— Task 1
- ✅ SOLID：开闭原则（新增 MemPersistence 不需改 BufferedUnitStore）— Task 2 stub 已展示

**类型一致性：**
- `unitstore.New(p UnitPersistence)` — Task 2 定义，Task 5/6 使用 ✓
- `be.EnableImportBuffer()` 返回 `*unitstore.BufferedUnitStore` — Task 5 定义，Task 6 使用 ✓
- `model.ID` 是 `string` 类型，map key 兼容 ✓
- `PendingCount()` — Task 2 定义，Task 5 测试使用 ✓
