# SDK 迁移 Phase 1 实现计划：共享 DB + DIP 接口

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 消除 MprBackend 对同一 SQLite 文件开两个独立连接的问题，将 modelsdk/mpr 改为接受外部 `*sql.DB`，并通过 `UnitWriter`/`UnitReader` 接口让 MprBackend 依赖抽象而非具体类型（Dependency Inversion）。

**Architecture:** `modelsdk/mpr.Reader` 新增 `ownsDB bool` 字段和 `OpenWithDB` 构造器；`Writer` 新增 `NewWriterFromDB`。新增 `UnitReader`/`UnitWriter` 接口文件。`MprBackend.msdkWriter` 字段类型从 `*modelsdkmpr.Writer` 改为 `modelsdkmpr.UnitWriter`，`Connect/Disconnect/Wrap` 共享 sdk/mpr Reader 的 `*sql.DB`。

**Tech Stack:** Go 1.26, `modernc.org/sqlite`, `database/sql`, `go.mongodb.org/mongo-driver/bson`

---

## File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `modelsdk/mpr/reader.go` | Modify | 添加 `ownsDB bool` 字段；新增 `OpenWithDB` 构造器；修改 `Close()` 尊重 `ownsDB` |
| `modelsdk/mpr/writer_core.go` | Modify | 新增 `NewWriterFromDB` 构造器 |
| `modelsdk/mpr/interfaces.go` | Create | `UnitReader`、`UnitWriter` 接口定义 |
| `mdl/backend/mpr/backend.go` | Modify | `msdkWriter` 字段类型改为 `UnitWriter`；更新 `Connect`/`Disconnect`/`Wrap` |
| `mdl/backend/mpr/single_connection_test.go` | Create | 验证两个 writer 共享同一 `*sql.DB` |

---

### Task 1: 给 modelsdk/mpr.Reader 添加 `ownsDB` 和 `OpenWithDB`

**Files:**
- Modify: `modelsdk/mpr/reader.go`

**背景**：当前 `Reader.Close()` 总是调用 `r.db.Close()`。引入共享 DB 后，外部传入的 db 不应由 Reader 关闭。需要 `ownsDB bool` 控制。

- [ ] **Step 1: 在 Reader struct 添加 `ownsDB` 字段**

在 `modelsdk/mpr/reader.go` 的 `Reader` struct（当前第 30-38 行）中，在 `readOnly bool` 之后加一行：

```go
type Reader struct {
	path           string
	db             *sql.DB
	version        MPRVersion
	contentsDir    string
	readOnly       bool
	ownsDB         bool  // NEW: false when DB was provided externally
	projectVersion *version.ProjectVersion

	unitCache      []cachedUnit
	unitCacheValid bool
}
```

- [ ] **Step 2: 在 `OpenWithOptions` 中将 `ownsDB` 设为 `true`**

在 `OpenWithOptions` 函数中，在 `r.db = db` 这行之后（约第 104 行）加：

```go
r.db = db
r.ownsDB = true
```

- [ ] **Step 3: 修改 `Close()` 尊重 `ownsDB`**

将 `Close()` 函数（约第 135-139 行）改为：

```go
func (r *Reader) Close() error {
	if r.db != nil && r.ownsDB {
		return r.db.Close()
	}
	return nil
}
```

- [ ] **Step 4: 新增 `OpenWithDB` 构造器**

在 `Close()` 函数之后（约第 140 行）插入：

```go
// OpenWithDB creates a Reader that reuses an existing *sql.DB connection.
// The caller owns the db and is responsible for closing it; this Reader's
// Close() is a no-op with respect to the database.
// contentsDir should be the path to the mprcontents folder (v2) or empty (v1).
func OpenWithDB(db *sql.DB, path, contentsDir string) (*Reader, error) {
	r := &Reader{
		path:    path,
		ownsDB:  false,
		readOnly: false,
	}

	if contentsDir != "" {
		r.version = MPRVersionV2
		r.contentsDir = contentsDir
	} else {
		r.version = MPRVersionV1
	}

	r.db = db

	pv, err := version.DetectFromDB(db)
	if err != nil {
		return nil, fmt.Errorf("detect project version: %w", err)
	}
	r.projectVersion = pv

	if err := r.verify(); err != nil {
		return nil, err
	}

	return r, nil
}
```

- [ ] **Step 5: 验证编译**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go build ./modelsdk/mpr/...
```

Expected: 无错误。

- [ ] **Step 6: 运行 modelsdk/mpr 现有测试**

```bash
go test ./modelsdk/mpr/... -v 2>&1 | tail -20
```

Expected: 所有测试 PASS。

- [ ] **Step 7: Commit**

```bash
git add modelsdk/mpr/reader.go
git commit -m "feat(modelsdk/mpr): add ownsDB field and OpenWithDB constructor"
```

---

### Task 2: 给 modelsdk/mpr.Writer 添加 `NewWriterFromDB`

**Files:**
- Modify: `modelsdk/mpr/writer_core.go`

- [ ] **Step 1: 在 `NewWriter` 之后添加 `NewWriterFromDB`**

在 `modelsdk/mpr/writer_core.go` 的 `NewWriter` 函数（约第 43-48 行）之后插入：

```go
// NewWriterFromDB creates a Writer that reuses an existing *sql.DB connection.
// The caller owns the db lifecycle; this Writer's Close() does not close the db.
// contentsDir should be the mprcontents folder path (v2) or empty string (v1).
func NewWriterFromDB(db *sql.DB, path, contentsDir string) (*Writer, error) {
	reader, err := OpenWithDB(db, path, contentsDir)
	if err != nil {
		return nil, fmt.Errorf("open reader with shared db: %w", err)
	}
	return &Writer{reader: reader}, nil
}
```

- [ ] **Step 2: 验证编译**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go build ./modelsdk/mpr/...
```

Expected: 无错误。

- [ ] **Step 3: Commit**

```bash
git add modelsdk/mpr/writer_core.go
git commit -m "feat(modelsdk/mpr): add NewWriterFromDB constructor for shared-DB usage"
```

---

### Task 3: 定义 `UnitReader` 和 `UnitWriter` 接口（DIP）

**Files:**
- Create: `modelsdk/mpr/interfaces.go`

这两个接口让 `MprBackend` 依赖抽象而非 `*modelsdk/mpr.Writer` 具体类型，满足 Dependency Inversion Principle。

- [ ] **Step 1: 创建 `modelsdk/mpr/interfaces.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package mpr

// UnitReader is the minimal read interface used by modelsdk-based backend writes.
// It is satisfied by *Reader.
type UnitReader interface {
	GetRawUnitBytes(unitID string) ([]byte, error)
	Version() MPRVersion
	ContentsDir() string
	InvalidateCache()
}

// UnitWriter is the minimal write interface for modelsdk-based backend writes.
// It is satisfied by *Writer.
type UnitWriter interface {
	Reader() UnitReader
	BeginWriteTransaction() (*WriteTransaction, error)
	UpdateRawUnit(unitID string, contents []byte) error
	InsertUnit(unitID, containerID, containmentName, unitType string, contents []byte) error
	DeleteUnit(unitID string) error
	Close() error
}
```

- [ ] **Step 2: 확인 — Reader와 Writer가 인터페이스를 실제로 만족하는지 컴파일 시 검증**

`modelsdk/mpr/interfaces.go` 파일에 컴파일 타임 체크를 추가:

（아래는 한국어 실수，한국어로 쓰지 마세요，계속 중국어로）

`modelsdk/mpr/interfaces.go` 파일에 아래를 추가하여 컴파일 시 인터페이스 충족 여부를 확인합니다:

在 `interfaces.go` 末尾加编译期接口检查：

```go
// Compile-time interface satisfaction checks.
var _ UnitReader = (*Reader)(nil)
var _ UnitWriter = (*Writer)(nil)
```

- [ ] **Step 3: 验证 `Reader` 实现 `UnitReader`**

`Reader` 需要有 `InvalidateCache()` 方法。检查是否存在：

```bash
grep -n "InvalidateCache" /mnt/data_sdd/gh/mxcli-wt-02/modelsdk/mpr/reader.go
```

如果不存在，在 `reader.go` 中 `DB()` 方法附近添加：

```go
// InvalidateCache clears the in-memory unit cache.
func (r *Reader) InvalidateCache() {
	r.unitCache = nil
	r.unitCacheValid = false
}
```

- [ ] **Step 4: 验证编译（含接口检查）**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go build ./modelsdk/mpr/...
```

Expected: 无错误。若有 "does not implement" 错误，说明 Reader 或 Writer 缺少对应方法，按错误信息补充。

- [ ] **Step 5: Commit**

```bash
git add modelsdk/mpr/interfaces.go modelsdk/mpr/reader.go
git commit -m "feat(modelsdk/mpr): add UnitReader/UnitWriter interfaces with compile-time checks"
```

---

### Task 4: 更新 MprBackend 使用共享 DB 和 UnitWriter 接口

**Files:**
- Modify: `mdl/backend/mpr/backend.go`

**核心变更**：
1. `msdkWriter` 字段类型从 `*modelsdkmpr.Writer` → `modelsdkmpr.UnitWriter`
2. `Connect` 从 sdk/mpr Reader 提取 db，传给 `modelsdkmpr.NewWriterFromDB`
3. `Disconnect` 不再调用 `msdkWriter.Close()`（db 由 sdk writer 管理）
4. `Wrap` 从 `writer.Reader().DB()` 提取 db

- [ ] **Step 1: 更新 `msdkWriter` 字段类型**

在 `backend.go` 的 `MprBackend` struct 中，将：
```go
msdkWriter *modelsdkmpr.Writer
```
改为：
```go
msdkWriter modelsdkmpr.UnitWriter
```

- [ ] **Step 2: 更新 `Connect` — 单连接化**

将 `Connect` 函数改为：

```go
func (b *MprBackend) Connect(path string) error {
	w, err := mpr.NewWriter(path)
	if err != nil {
		return err
	}

	db := w.Reader().DB()
	contentsDir := w.Reader().ContentsDir()
	mw, err := modelsdkmpr.NewWriterFromDB(db, path, contentsDir)
	if err != nil {
		_ = w.Close()
		return err
	}

	b.writer = w
	b.reader = w.Reader()
	b.msdkWriter = mw
	b.path = path
	return nil
}
```

- [ ] **Step 3: 更新 `Disconnect` — msdkWriter 不再关闭 db**

将 `Disconnect` 函数改为（去掉 msdkWriter.Close()，因为 db 由 sdk writer 管理）：

```go
func (b *MprBackend) Disconnect() error {
	if b.writer == nil {
		return nil
	}
	err := b.writer.Close() // 关闭 db（因为 sdk writer ownsDB=true）
	b.writer = nil
	b.reader = nil
	b.msdkWriter = nil
	b.path = ""
	return err
}
```

- [ ] **Step 4: 更新 `Wrap` — 从 sdk writer 提取 db**

将 `Wrap` 函数改为：

```go
func Wrap(writer *mpr.Writer, path string) *MprBackend {
	db := writer.Reader().DB()
	contentsDir := writer.Reader().ContentsDir()
	mw, err := modelsdkmpr.NewWriterFromDB(db, path, contentsDir)
	if err != nil {
		log.Printf("mprbackend: Wrap: failed to create modelsdk writer for %s: %v", path, err)
	}
	return &MprBackend{
		reader:     writer.Reader(),
		writer:     writer,
		msdkWriter: mw,
		path:       path,
	}
}
```

- [ ] **Step 5: 确认 sdk/mpr.Reader 有 ContentsDir() 方法**

```bash
grep -n "ContentsDir" /mnt/data_sdd/gh/mxcli-wt-02/sdk/mpr/reader.go
```

如果不存在，在 `sdk/mpr/reader.go` 中添加：

```go
// ContentsDir returns the path to the mprcontents directory (v2) or empty string (v1).
func (r *Reader) ContentsDir() string {
	return r.contentsDir
}
```

- [ ] **Step 6: 验证编译**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go build ./mdl/backend/mpr/...
go build ./...
```

Expected: 无错误。

- [ ] **Step 7: 运行完整测试套件**

```bash
go test ./mdl/backend/mpr/... -v 2>&1 | tail -30
```

Expected: 所有测试 PASS。

- [ ] **Step 8: Commit**

```bash
git add mdl/backend/mpr/backend.go
git commit -m "feat(backend): single-connection MprBackend via shared *sql.DB"
```

---

### Task 5: 集成测试 — 验证单连接

**Files:**
- Create: `mdl/backend/mpr/single_connection_test.go`

- [ ] **Step 1: 写失败测试**

创建 `mdl/backend/mpr/single_connection_test.go`：

```go
// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestSingleConnection_SharedDB(t *testing.T) {
	// Create minimal v1 MPR
	dir := t.TempDir()
	mprPath := filepath.Join(dir, "test.mpr")

	db, err := sql.Open("sqlite", mprPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if _, err := db.Exec(`
		CREATE TABLE _MetaData (_FormatVersion INTEGER, _ProductVersion TEXT,
		                        _BuildVersion TEXT, _SchemaHash TEXT);
		INSERT INTO _MetaData VALUES (1, '10.18.0', '10.18.0.0', '');
		CREATE TABLE _Transaction (LastTransactionID TEXT);
		INSERT INTO _Transaction VALUES ('00000000-0000-0000-0000-000000000000');
		CREATE TABLE Unit (UnitID BLOB PRIMARY KEY NOT NULL, ContainerID BLOB,
		                   ContainmentName TEXT, TreeConflict LONG,
		                   ContentsHash TEXT, ContentsConflicts TEXT, Contents BLOB);
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	db.Close()

	b := New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	// Both writers must share the same *sql.DB pointer.
	sdkDB := b.reader.DB()
	// Access msdkWriter.Reader().DB() via interface — need type assertion for test
	type dbGetter interface{ DB() *sql.DB }
	msdkReader, ok := b.msdkWriter.Reader().(dbGetter)
	if !ok {
		t.Fatal("msdkWriter.Reader() does not implement DB() — shared-DB broken")
	}
	msdkDB := msdkReader.DB()

	if sdkDB != msdkDB {
		t.Errorf("db pointers differ: sdk=%p msdk=%p — two connections open", sdkDB, msdkDB)
	}
}
```

- [ ] **Step 2: 运行测试 — 期望 PASS**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./mdl/backend/mpr/... -run TestSingleConnection_SharedDB -v
```

Expected:
```
=== RUN   TestSingleConnection_SharedDB
--- PASS: TestSingleConnection_SharedDB (0.XXs)
PASS
```

若 FAIL，最常见原因：`msdkWriter.Reader()` 返回的 `UnitReader` 接口没有 `DB()` 方法。在 `modelsdk/mpr/interfaces.go` 的 `UnitReader` 接口中添加 `DB() *sql.DB`，然后重新编译。

- [ ] **Step 3: 运行完整后端测试套件**

```bash
go test ./mdl/backend/mpr/... -v 2>&1 | tail -30
```

Expected: 全部 PASS，含 `TestSetProjectSecurityLevel_ViaModelsdk`。

- [ ] **Step 4: 全量 build + vet**

```bash
make build && make vet
```

Expected: 无错误。

- [ ] **Step 5: Commit**

```bash
git add mdl/backend/mpr/single_connection_test.go
git commit -m "test(backend): verify single *sql.DB connection in MprBackend"
```

---

## Self-Review

**Spec coverage:**
- ✅ `OpenWithDB` / `NewWriterFromDB` 构造器：Task 1 + 2
- ✅ `ownsDB` 字段（Close 不关闭外部 db）：Task 1
- ✅ `UnitReader` / `UnitWriter` 接口（DIP）：Task 3
- ✅ 编译期接口检查：Task 3
- ✅ `MprBackend` 单连接化：Task 4
- ✅ 单连接集成测试：Task 5
- ✅ ISP（FullBackend 接口拆分）：已在 `mdl/backend/` 中完成，无需本计划处理

**Placeholder scan:** 无 TBD/TODO。Task 3 Step 3 的 InvalidateCache 条件处理已给出完整代码。

**Type consistency:**
- `modelsdkmpr.UnitWriter`：Task 3（定义）→ Task 4（字段类型）→ Task 5（测试使用）✅
- `OpenWithDB(db, path, contentsDir string) (*Reader, error)`：Task 1（实现）→ Task 2（被 NewWriterFromDB 调用）✅
- `w.Reader().DB()`：Task 4 Connect 中调用 sdk/mpr.Reader.DB()（存在于 reader.go:182）✅
- `w.Reader().ContentsDir()`：Task 4 Step 5 确认存在（若缺失给出完整添加代码）✅
