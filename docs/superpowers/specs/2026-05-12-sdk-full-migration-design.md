# SDK 全量迁移设计：sdk/mpr → modelsdk

**Date**: 2026-05-12
**Status**: Approved
**Goal**: 将 MprBackend 的全部写路径（137 处 `b.writer.` 调用，175 个 Writer 方法）系统性地迁移到 modelsdk WriteTransaction，最终删除 sdk/mpr.Writer，并在读路径上建立相同的替换路径。代码须符合 SOLID 原则。

---

## 背景与动机

- `sdk/mpr.Writer.updateUnit()` v2 分支调用 `updateTransactionID()`，在硬链接 MPR 文件上触发 SQLITE_READONLY_DBMOVED (1544)
- `MprBackend` 当前对同一 SQLite 文件开两个独立 `*sql.DB`，存在缓存不一致和 SQLITE_BUSY 风险
- `modelsdk/mpr.WriteTransaction` 不写 `_Transaction` 表，使用 temp→rename 两阶段写，崩溃安全
- `modelsdk/gen` 已覆盖 53 个 domain，typed element 层已就绪，缺的是基础设施和高层查询层

---

## 成功标准

- `grep "b\.writer\." mdl/backend/mpr/backend.go` 返回零结果（Phase 3 结束）
- `go test ./...` 全绿
- `go vet ./...` 零警告
- `mx check` 对真实 MPR 无新错误
- `FullBackend` 拆分为 domain 接口，executor handler 只依赖所需子接口

---

## 四阶段路线

```
Phase 1 — 基础设施与 SOLID 接口（1-2 周）
Phase 2 — 写路径批量迁移（按 domain，TDD，逐步推进）
Phase 3 — 删除 sdk/mpr.Writer
Phase 4 — 读路径迁移（Phase 3 完成后启动）
```

---

## Phase 1：基础设施与 SOLID 接口

### 1.1 modelsdk/mpr 共享 DB 构造器

**文件**：`modelsdk/mpr/reader.go`、`modelsdk/mpr/writer_core.go`

新增 `OpenWithDB` 和 `NewWriterFromDB`，接受外部 `*sql.DB` 而非自己打开文件：

```go
// OpenWithDB 复用调用方已有的 *sql.DB，不重新打开 SQLite 文件。
// contentsDir 为空字符串时视为 v1（无 mprcontents）。
func OpenWithDB(db *sql.DB, path, contentsDir string) (*Reader, error)

// NewWriterFromDB 复用已有 *sql.DB，避免双连接问题。
func NewWriterFromDB(db *sql.DB, path, contentsDir string) (*Writer, error)
```

原有 `Open`、`OpenWithOptions`、`NewWriter` 保持不变（向后兼容）。

### 1.2 MprBackend 单连接化

**文件**：`mdl/backend/mpr/backend.go`

`Connect()` 改为：
1. 通过 `sdk/mpr.OpenWithOptions(path, ReadOnly:false)` 拿到 `*mpr.Reader`（含 `db`）
2. 用 `modelsdkmpr.NewWriterFromDB(reader.DB(), path, contentsDir)` 创建 modelsdk writer
3. 两个 writer 共享同一 `*sql.DB`，零双连接风险

`Wrap()` 同步：从 `writer.Reader().DB()` 提取 db，传给 modelsdk。

### 1.3 SOLID 接口拆分（Interface Segregation）

**文件**：`mdl/backend/` 下各 domain 接口文件

将 `backend.FullBackend`（巨型接口）拆分为 domain 子接口：

```go
type SecurityBackend interface {
    GetProjectSecurity() (*security.ProjectSecurity, error)
    SetProjectSecurityLevel(unitID model.ID, level string) error
    SetProjectDemoUsersEnabled(unitID model.ID, enabled bool) error
    AddUserRole(unitID model.ID, name string, moduleRoles []string, manageAllRoles bool) error
    RemoveUserRole(unitID model.ID, name string) error
    AlterUserRoleModuleRoles(unitID model.ID, userRoleName string, add bool, moduleRoles []string) error
    // ...
}
type DomainModelBackend interface { ... }
type MicroflowBackend interface { ... }
type PageBackend interface { ... }
// FullBackend 组合所有子接口（向后兼容）
type FullBackend interface {
    ConnectionBackend
    SecurityBackend
    DomainModelBackend
    MicroflowBackend
    PageBackend
    // ...
}
```

### 1.4 SOLID 抽象接口（Dependency Inversion）

**文件**：`modelsdk/mpr/interfaces.go`（新建）

MprBackend 依赖接口而非具体类型：

```go
// UnitReader 是 modelsdk/mpr.Reader 的写路径所需最小接口。
type UnitReader interface {
    GetRawUnitBytes(unitID string) ([]byte, error)
    Version() MPRVersion
}

// UnitWriter 是 modelsdk/mpr.Writer 的最小接口。
type UnitWriter interface {
    Reader() UnitReader
    BeginWriteTransaction() (*WriteTransaction, error)
    UpdateRawUnit(unitID string, contents []byte) error
    Close() error
}
```

`MprBackend.msdkWriter` 字段类型改为 `modelsdkmpr.UnitWriter`。

### 1.5 Phase 1 测试

- `TestSingleConnection`：断言 `b.writer.Reader().DB()` 与 `b.msdkWriter.Reader().DB()` 指针相同
- 全套现有测试继续通过

---

## Phase 2：写路径批量迁移

### 迁移顺序（由简入繁）

| 顺序 | Domain | 估算方法数 | 复杂度 |
|------|--------|-----------|--------|
| 1 | Security（剩余） | ~5 | 低 |
| 2 | Modules | ~4 | 低 |
| 3 | Enumerations | ~3 | 低 |
| 4 | Folders / Navigation | ~5 | 低-中 |
| 5 | Domain Models | ~8 | 中 |
| 6 | Pages / Snippets | ~12 | 高 |
| 7 | Microflows / Nanoflows | ~20 | 高 |
| 8 | REST / Workflow / BusinessEvents 等 | ~剩余 | 中-高 |

### 每个 domain 的标准工作流（TDD）

```
1. grep "b\.writer\." backend.go | grep <domain>   → 列出待迁方法
2. 写失败集成测试（makeTestMPR + 调用 backend 方法 + 断言 BSON）
3. 新建 mdl/backend/mpr/<domain>_modelsdk.go，实现六步模板：
   nil 守卫 → GetRawUnitBytes → Decode → 类型断言 → Mutate → Encode → WriteTransaction
4. backend.go 替换调用 → 测试变绿
5. go test ./mdl/backend/mpr/... + go vet
6. commit（一个 domain 一个 commit）
7. mx check 对真实 MPR 验证
```

### 六步迁移模板（符合 Single Responsibility）

每个 domain 一个文件，每个文件只处理一个 domain 的写逻辑：

```go
func (b *MprBackend) setXxxViaModelsdk(unitID model.ID, ...) error {
    if b.msdkWriter == nil {
        return fmt.Errorf("modelsdk writer not initialized")
    }
    rawBytes, err := b.msdkWriter.Reader().GetRawUnitBytes(string(unitID))
    elem, err := codec.NewDecoder(codec.DefaultRegistry).Decode(bson.Raw(rawBytes))
    typed, ok := elem.(*genxxx.YourType)
    if !ok { return fmt.Errorf("unexpected type %T", elem) }
    typed.SetXxx(value)
    newBytes, err := (&codec.Encoder{}).Encode(typed)
    wtx, err := b.msdkWriter.BeginWriteTransaction()
    if err := wtx.WriteUnit(string(unitID), newBytes); err != nil {
        _ = wtx.Rollback()
        return fmt.Errorf("write: %w", err)
    }
    return wtx.Commit()
}
```

---

## Phase 3：删除 sdk/mpr.Writer

**触发条件**：`grep "b\.writer\." mdl/backend/mpr/backend.go` 返回零结果。

### 删除步骤

1. 删除 `MprBackend.writer *mpr.Writer` 字段
2. 删除 `Connect/Disconnect/Wrap` 中 sdk writer 相关代码
3. 删除 `sdk/mpr/writer_*.go`（保留 reader.go、parser.go 等读路径文件）
4. `go build ./...` → 修复编译错误
5. `go test ./...` + `mx check` 端到端验证
6. 更新 MockBackend（移除已不存在的 writer stub）

### 终态结构

```
MprBackend
  ├── reader     *sdk_mpr.Reader       ← 读路径（Phase 3 后仍保留）
  ├── msdkWriter modelsdkmpr.UnitWriter ← 唯一写路径（接口，非具体类型）
  └── path       string

sdk/mpr/    → 只剩 reader + parser（~20 个文件）
modelsdk/   → 完整读写
```

---

## Phase 4：读路径迁移

**启动条件**：Phase 3 完成，双连接已消除。

### 挑战

| 挑战 | 说明 |
|------|------|
| 高层查询层缺失 | `sdk/mpr.Reader.GetProjectSecurity()` 等业务级方法需在 modelsdk 上重建 |
| Catalog 深度耦合 | `mdl/catalog/` 依赖 sdk/mpr.Reader 内部结构，需专项解耦 |
| 单元缓存一致性 | modelsdk 和 sdk 各有 `InvalidateCache()`，替换时需统一 |

### 方案

在 `modelsdk/mpr/` 建立高层查询 API：

```go
// QueryLayer 在 Reader 上提供业务级查询方法
type QueryLayer struct { reader *Reader }

func (q *QueryLayer) GetProjectSecurity() (*gen_security.ProjectSecurity, error)
func (q *QueryLayer) ListModules() ([]*model.Module, error)
func (q *QueryLayer) GetDomainModel(moduleID model.ID) (*gen_domainmodels.DomainModel, error)
// ...
```

`MprBackend.reader` 字段替换为 `*modelsdkmpr.QueryLayer`，Catalog 改用 QueryLayer。

Phase 4 需单独的 spec → plan → 实现循环，不在本次计划范围内展开。

---

## 风险与缓解

| 风险 | 缓解 |
|------|------|
| modelsdk/gen 某 domain 覆盖不完整 | 用 `debug-bson.md` + supplements 补充，不阻断迁移 |
| 复杂 PartList 序列化与 sdk 行为不一致 | 每个 domain 迁完后 `mx check` 验证，不等 Phase 3 |
| sdk/mpr.Reader 内部类型被 writer 引用 | 先做 reader/writer 文件解耦再删除 |
| Phase 1 接口拆分破坏现有 executor | 用 embedding 保持 FullBackend 向后兼容 |

---

## 不在本次范围

- executor 层改动
- MDL 语法变更
- modelsdk/gen 新 domain 生成（已有 53 个，按需补充）
- Phase 4 详细设计（单独 spec）
