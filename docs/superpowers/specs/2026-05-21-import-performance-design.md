# Import 性能优化：BufferedUnitStore 设计规格

**日期：** 2026-05-21  
**背景：** `mxcli import` 导入 1000+ MDL 文件时耗时 20 分钟以上，根本原因是每条语句触发独立 SQLite 事务和全量缓存失效。  
**目标：** 通过引入通用的 `BufferedUnitStore` 组件，将导入时间降至 2-4 分钟，同时为 export、diff、exec 等其他功能提供可复用的懒加载缓冲存储层。

---

## 性能瓶颈分析

| 瓶颈 | 当前成本（1000 文件）| 占比 |
|------|---------------------|------|
| 每语句独立 SQLite 事务 | 50,000 次提交 × 5ms | ~40% |
| 每语句缓存失效 + 重建 | 50,000 次 DB round-trip | ~35% |
| 每 unit 写入磁盘（fsync） | 50,000 次 I/O | ~15% |
| 文件间缓存冷启动 | 每文件首条语句重建缓存 | ~10% |

---

## 架构设计

### 分层结构（SOLID）

```
┌─────────────────────────────────────────────────────────┐
│  高层功能（Import / Export / Diff / exec 批量语句…）     │
│                  depends on ↓ (D: 依赖倒置)             │
├─────────────────────────────────────────────────────────┤
│  UnitReader / UnitWriter  (I: 接口隔离)                 │
│  Read(id) []byte  │  Write(id, []byte)                  │
│                   │  Flush() error                       │
│                   │  Discard()                           │
├─────────────────────────────────────────────────────────┤
│  BufferedUnitStore  (S: 单一职责 — 管理内存状态)         │
│  ┌─────────────┐   ┌──────────────────────────────┐    │
│  │  LazyCache  │   │  DirtySet                    │    │
│  │ loaded: map │   │ pending: map[model.ID][]byte  │    │
│  │ Get/Put/Evict   │ MarkDirty/AllDirty/Clear      │    │
│  └─────────────┘   └──────────────────────────────┘    │
│              depends on ↓                               │
├─────────────────────────────────────────────────────────┤
│  interface UnitPersistence  (O: 开闭 — 可替换实现)      │
│    Load(id model.ID) ([]byte, error)                    │
│    BatchStore(units map[model.ID][]byte) error          │
│    BatchHash(units map[model.ID][]byte) map[model.ID]string │
│    LoadMetadata() (UnitMetaIndex, error)  // 仅 ID/类型 │
├─────────────────┬───────────────────┬───────────────────┤
│ MprV2Persistence│ MprV1Persistence  │ MemPersistence    │
│ (SQLite+mxunit) │ (SQLite only)     │ (测试/沙箱用)     │
└─────────────────┴───────────────────┴───────────────────┘
```

### 关键接口定义

```go
// mdl/backend/unitstore/interfaces.go

// UnitReader — 只读消费者（Export、Diff）依赖此接口
type UnitReader interface {
    Read(id model.ID) ([]byte, error)
}

// UnitWriter — 写入消费者（Import、exec 批量）依赖此接口
type UnitWriter interface {
    UnitReader
    Write(id model.ID, data []byte) error
    Flush() error    // 批量落盘：hash → BatchStore → 清脏集
    Discard() error  // 回滚：清脏集，不落盘
}

// UnitPersistence — 底层存储抽象，由 MprBackend 实现
type UnitPersistence interface {
    Load(id model.ID) ([]byte, error)
    BatchStore(units map[model.ID][]byte) error
    BatchHash(units map[model.ID][]byte) (map[model.ID]string, error)
    LoadMetadataIndex() (UnitMetaIndex, error)
}
```

---

## 核心组件：BufferedUnitStore

### 结构

```go
// mdl/backend/unitstore/buffered.go

type BufferedUnitStore struct {
    persistence UnitPersistence

    // LazyCache：已从磁盘加载的 unit（只读副本）
    loaded map[model.ID][]byte

    // DirtySet：本会话写入的 unit（待落盘）
    pending map[model.ID][]byte

    mu sync.RWMutex  // 保护 loaded 和 pending
}
```

### 读路径（懒加载）

```go
func (b *BufferedUnitStore) Read(id model.ID) ([]byte, error) {
    b.mu.RLock()
    // 1. DirtySet 优先：本会话写入的版本
    if data, ok := b.pending[id]; ok {
        b.mu.RUnlock()
        return data, nil
    }
    // 2. LazyCache：已加载的磁盘版本
    if data, ok := b.loaded[id]; ok {
        b.mu.RUnlock()
        return data, nil
    }
    b.mu.RUnlock()

    // 3. 懒加载：按需从磁盘读入，仅此一次
    data, err := b.persistence.Load(id)
    if err != nil {
        return nil, err
    }
    b.mu.Lock()
    b.loaded[id] = data
    b.mu.Unlock()
    return data, nil
}
```

### 写路径（只写内存）

```go
func (b *BufferedUnitStore) Write(id model.ID, data []byte) error {
    b.mu.Lock()
    defer b.mu.Unlock()
    b.pending[id] = data  // 仅写内存，不触发任何磁盘 I/O
    return nil
}
```

### Flush（批量落盘）

```go
func (b *BufferedUnitStore) Flush() error {
    b.mu.Lock()
    defer b.mu.Unlock()

    if len(b.pending) == 0 {
        return nil
    }

    // 1. 批量计算 BSON hash（用于 @cache: 摘要更新）
    _, err := b.persistence.BatchHash(b.pending)
    if err != nil {
        return fmt.Errorf("batch hash: %w", err)
    }

    // 2. 批量写入磁盘（单次 SQLite 事务 + 批量 .mxunit 写入）
    if err := b.persistence.BatchStore(b.pending); err != nil {
        return fmt.Errorf("batch store: %w", err)
    }

    // 3. 脏集合提升为已加载缓存，清空脏标记
    for id, data := range b.pending {
        b.loaded[id] = data
    }
    b.pending = make(map[model.ID][]byte)
    return nil
}
```

### Discard（回滚，不落盘）

```go
func (b *BufferedUnitStore) Discard() error {
    b.mu.Lock()
    defer b.mu.Unlock()
    b.pending = make(map[model.ID][]byte)
    return nil
}
```

---

## UnitPersistence 实现：MprV2Persistence

### 懒加载元数据

```go
// LoadMetadataIndex 只加载 unit ID、类型、容器关系，不加载 BSON 内容
// 在 ImportSession 初始化时调用一次，约 50-200ms（vs 全量加载 2-5s）
func (p *MprV2Persistence) LoadMetadataIndex() (UnitMetaIndex, error) {
    rows, err := p.db.Query(`SELECT ID, UnitType, ContainerID FROM Unit`)
    // 返回轻量索引，不包含 Contents 列（BLOB）
    ...
}
```

### BatchStore

```go
// BatchStore 在单次 SQLite 事务内写入所有脏 unit
func (p *MprV2Persistence) BatchStore(units map[model.ID][]byte) error {
    tx, err := p.db.Begin()
    if err != nil { return err }
    defer tx.Rollback()

    stmt, err := tx.Prepare(`INSERT OR REPLACE INTO Unit (ID, Contents) VALUES (?, ?)`)
    if err != nil { return err }
    defer stmt.Close()

    for id, data := range units {
        if _, err := stmt.Exec(string(id), data); err != nil {
            return err
        }
        // 同步写 .mxunit 文件（MPR v2）
        if err := p.writeMxunit(id, data); err != nil {
            return err
        }
    }

    return tx.Commit()
}
```

### .mpr 文件原子写回

```go
// 写到临时文件再 rename，确保原子性
func (p *MprV2Persistence) atomicWriteMpr() error {
    tmp := p.mprPath + ".tmp"
    if err := p.db.Backup(tmp); err != nil { return err }
    return os.Rename(tmp, p.mprPath)
}
```

---

## Import 集成

### 修改前后对比

**当前**（每语句）：
```
ExecuteProgram → stmt1 → writeUnitContents → BeginTx → WriteUnit → Commit → InvalidateCache
               → stmt2 → writeUnitContents → BeginTx → WriteUnit → Commit → InvalidateCache
               ... × 50
```

**修改后**（每文件）：
```
ExecuteProgram → stmt1 → bufStore.Write() [仅内存]
               → stmt2 → bufStore.Write() [仅内存，读 stmt1 的结果直接命中]
               ... × 50
               → bufStore.Flush() [单次事务，批量 hash，批量写磁盘]
```

### cmd_import_project.go 集成点

```go
func (e *Executor) ImportProject(inputDir string, opts ImportOptions) error {
    ctx := e.newExecContext(context.Background())

    // 初始化 BufferedUnitStore（懒加载，只加载元数据）
    bufStore, err := unitstore.NewBuffered(ctx.Backend.UnitPersistence())
    if err != nil { return err }
    ctx.Backend.SetUnitStore(bufStore)  // 替换底层存储

    for _, rel := range sorted {
        // ... 读取并解析 MDL 文件 ...

        if err := e.ExecuteProgram(prog); err != nil {
            bufStore.Discard()  // 回滚本文件的内存写入
            if !opts.SkipErrors { return err }
            continue
        }

        // 文件级检查点：批量落盘
        if err := bufStore.Flush(); err != nil {
            return fmt.Errorf("flush after %s: %w", rel, err)
        }
    }
    return nil
}
```

---

## 缓存一致性

executor 的 domain model / microflow / security 缓存以 `bufStore.Read()` 为数据来源：

- **文件内可见性**：stmt1 写 Entity A → 进 pending；stmt2 读 Entity A → pending 命中 → 立即可见，无 DB 查询
- **缓存失效**：`invalidateDomainModelsCache()` 改为"标记该模块需要从 bufStore 重读"，不再触发 SQLite round-trip
- **文件间一致性**：Flush 后 pending → loaded，下个文件读到的是落盘后的状态

---

## 预期性能提升

| 指标 | 当前 | 优化后 |
|------|------|--------|
| SQLite 事务数/文件 | 50+ | 1 |
| 磁盘写入次数/文件 | 50+ fsync | 1 次批量 |
| 文件内 DB 读次数 | 每语句读 DB | 0（命中 pending/loaded）|
| hash 计算时机 | 每 unit 单独计算 | 每文件批量 |
| 启动内存占用 | 全量加载 | 仅元数据索引（懒加载）|
| **预计总时间** | **~20 分钟** | **~2-4 分钟** |

---

## 文件结构

```
mdl/backend/unitstore/
  interfaces.go          # UnitReader, UnitWriter, UnitPersistence 接口
  buffered.go            # BufferedUnitStore 实现
  buffered_test.go       # 单元测试
  meta_index.go          # UnitMetaIndex 类型

mdl/backend/mpr/
  unit_persistence.go    # MprV2Persistence, MprV1Persistence 实现
  unit_persistence_test.go

mdl/executor/
  cmd_import_project.go  # 集成 BufferedUnitStore（修改）
```

---

## 不在范围内

- 并行文件解析（增量收益 <5%，复杂度高）
- 内存上限 / LRU 淘汰策略（大型项目内存管控，留作后续）
- Export / Diff 切换到 UnitReader（可独立进行，接口已预留）
- SQLite `:memory:` 全内存模式（当前设计是懒加载 + 批量写，不是全内存拷贝）
