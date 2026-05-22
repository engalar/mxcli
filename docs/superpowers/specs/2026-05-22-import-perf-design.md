# Import 性能优化设计文档

**日期：** 2026-05-22
**分支：** pr/mpr-pack-expr-testdata
**前置条件：** 删除 REPL

---

## 背景

实测数据（1023 MDL 文件，MacnicaApp.mpr）：
- 完整导入：**6m7s**（4 文件/秒）
- 纯解析（dry-run）：**2.9s**（490 文件/秒）
- 执行 + flush 占 **99.2%** 耗时

pprof 分析结合微基准调试得出三个根因：

| 根因 | 调试依据 | 严重程度 |
|------|---------|---------|
| `GetDomainModelGen` 每语句 3 次 SQLite 读，无缓存 | 场景4：单次耗时 10ms，占 66% execute 时间 | 高 |
| `EnableImportBuffer` 未接线，写直接落盘 | 场景2：`pending=0`，BufferedUnitStore 空转 | 中 |
| Parse 串行，execute 等待 parse | 微基准D：pipeline 1.7x 加速 | 低 |

**排除的假设（有调试数据为证）：**
- GC 不是主因——禁 GC 反而慢 40%
- Flush I/O 不是瓶颈——50 单元仅 10ms
- 批量事务无益——更大批次反而更慢（SQLite WAL 对小事务已优化）

---

## 前置工作：删除 REPL

### 原因

REPL 是唯一依赖 `GetDomainModelGen` 缓存条件判断的场景。删除后：
- Fix 1 缓存可无条件激活，无需 `importBuf != nil` 判断
- 删除 `github.com/chzyer/readline` 依赖
- 减少约 920 行代码

### 删除范围

| 操作 | 路径 |
|------|------|
| 删除目录 | `mdl/repl/`（repl.go 818 行 + color.go 74 行） |
| 局部删除 | `cmd/mxcli/main.go`：REPL 启动分支（约 28 行）+ repl 导入 |
| 移除依赖 | `go.mod`：`github.com/chzyer/readline v1.5.1` |
| 清理文档 | `README.md`、`CLAUDE.md` 中的 REPL 相关引用 |
| 删除图片 | `docs/images/mxcli-repl.png` |

**保留：** `cmd/mxcli/tui/session.go`（TUI 会话，与 REPL 无关）

**删除后 `main.go` 行为：** 无 `-c` 参数时打印使用说明并退出（不再进入交互模式）。

---

## Fix 1：DomainModel write-through 缓存

### 问题

`GetDomainModelGen(moduleID)` 每次调用执行三次 SQLite SELECT：
1. `ListUnitsByType("DomainModels$DomainModel")` — 扫描所有 DM 单元
2. `ListModules()` — 加载所有模块
3. `BuildContainerParent()` — 构建父子关系图

每个 CREATE ENTITY / CREATE ASSOCIATION / GRANT 语句均调用 1-3 次，合计 10ms/次。

### 设计

在 `executorCache` 新增字段：

```go
// executorCache（mdl/executor/executor.go）
type executorCache struct {
    // ... 现有字段 ...

    // domainModelByModule 缓存 moduleID → *genDm.DomainModel。
    // write-through：写时同步更新，绝不用失效策略。
    domainModelByModule map[model.ID]*genDm.DomainModel
}
```

**读路径**（`MprBackend.GetDomainModelGen`）改为通过 `ExecContext` 查缓存：

```
GetDomainModelGen(moduleID):
  if ctx.Cache.domainModelByModule[moduleID] != nil:
    return cached          # 0ms，无 SQLite
  dm = 3×SQLite 读
  ctx.Cache.domainModelByModule[moduleID] = dm
  return dm
```

实现方式：在 `ExecContext` 上提供 `GetCachedDomainModel(id)` 和 `SetCachedDomainModel(id, dm)` 方法，由 executor 层的命令函数调用，不下沉到 backend。

**写路径**（write-through，场景1/4 已验证正确性）：

在 `execCreateEntityGen`、`execCreateAssociationGen` 等写完 DM 后立即更新缓存：

```go
// 写 DM 到磁盘
if err := repo.Update(dm); err != nil { ... }
// write-through：用最新指针更新缓存，后续读无需重查 SQLite
ctx.SetCachedDomainModel(moduleID, dm)
```

**为何用 write-through 而非失效：**
边界情况调试（场景4）证明：同文件内连续 CREATE ENTITY 序列中，写后立即有下一条读。失效策略会触发重读（浪费 10ms），write-through 将最新 dm 指针直接写入缓存，后续读为 0ms 且数据正确。

**无并发问题：** executor 的 `ExecContext` 是单 goroutine 生命周期，`executorCache` 无需锁。

### 预期收益

场景5 量化：`GetDomainModelGen` 6.6ms/次，典型文件调用 15-30 次 → 节省 100-200ms/文件。
外推 1023 文件：**节省约 1.5-3 分钟**，总耗时从 6m7s → 约 3-4.5 分钟。

---

## Fix 2：EnableImportBuffer 接线修复

### 问题

`EnableImportBuffer()` 创建了 `BufferedUnitStore` 但从未调用 `msdkWriter.SetSessionBuf(...)`，导致：
- `BufferedUnitStore.pending` 始终为 0
- 每次 `updateUnit` 直接落 SQLite，触发 `InvalidateCache()`
- Flush 是空操作

### 设计

修改 `MprBackend.EnableImportBuffer()`：

```go
func (b *MprBackend) EnableImportBuffer() *unitstore.BufferedUnitStore {
    buf := unitstore.New(b.NewUnitPersistence())
    b.unitBuf = buf

    // 接线：将 writer 的写操作路由到 BufferedUnitStore
    b.msdkWriter.SetSessionBuf(func(unitID string, data []byte) error {
        if err := buf.Write(model.ID(unitID), data); err != nil {
            return err
        }
        // 同步 overlay，保证同批次内的 GetRawUnitBytes 能看到未刷盘的写
        b.msdkReader.SetOverlay(unitID, data)
        return nil
    })

    return buf
}

func (b *MprBackend) DisableImportBuffer() {
    if b.msdkWriter != nil {
        b.msdkWriter.ClearSessionBuf()   // 新增：清除 sessionBuf
    }
    if b.unitBuf != nil {
        b.unitBuf.Discard()
        b.unitBuf = nil
        b.msdkReader.ClearAllOverlays()
    }
}
```

**边界情况（场景2/3 已验证）：**
- `SetOverlay` 必须与 `buf.Write` 同步调用（同一个 callback 内），否则同文件内后续 `GetRawUnitBytes` 读到磁盘旧数据
- `DisableImportBuffer` 必须先 `ClearSessionBuf` 再 `ClearAllOverlays`，防止 Discard 后残留 sessionBuf 仍路由写操作到已销毁的 buf

### 与 Fix 1 的交互

Fix 2 激活后，updateUnit 不再调用 `InvalidateCache()`（走 sessionBuf 短路），Fix 1 的 DM 缓存不会被 reader 层的 InvalidateCache 干扰——两者独立正确。

---

## Fix 3：Parse pipeline（可选，后续迭代）

### 设计

在 `ImportProject` 的文件循环中引入 goroutine pipeline：

```go
type parsedFile struct {
    rel  string
    prog *ast.Program
    errs []error
}

ch := make(chan parsedFile, 4)  // 缓冲 4 个，parse 超前 execute

go func() {
    defer close(ch)
    for _, rel := range sorted {
        content, _ := os.ReadFile(...)
        prog, errs := visitor.Build(string(content))
        ch <- parsedFile{rel, prog, errs}
    }
}()

for p := range ch {
    // 主 goroutine 顺序 execute（无并发写）
    exec.ExecuteProgram(p.prog)
    importBuf.Flush()
}
```

**一致性保证：** execute 仍串行，只有 parse 并发，无共享状态写冲突。

**预期收益：** 微基准D 验证 1.7x（串行 18.9s → pipeline 11.2s for 50 文件）。与 Fix 1+2 叠加后绝对收益约 30-50s（parse 从 critical path 移出）。

**本次不实现原因：** Fix 1+2 已能解决主要瓶颈，Fix 3 改动 `ImportProject` 主循环，需要独立验证，留下一个迭代。

---

## 实现顺序

```
1. 删除 REPL（前置，独立 commit）
2. Fix 2：EnableImportBuffer 接线（改动小，无新接口）
3. Fix 1：DomainModel write-through 缓存（改动 ExecContext + 各 exec* 函数）
4. 验证：运行 perf_import 确认耗时符合预期
5. Fix 3（下一个迭代，可选）
```

---

## 测试计划

| 测试 | 文件 | 覆盖场景 |
|------|------|---------|
| 写后读一致性 | `executor/cache_test.go` | Fix 1：write-through 正确性 |
| 连续写-读单调性 | `executor/cache_test.go` | Fix 1：N 次 CREATE ENTITY 后 entity 数单调增 |
| Buffer 接线验证 | `backend/mpr/import_buf_test.go` | Fix 2：pending>0 证明路由生效 |
| Flush 后磁盘一致 | `backend/mpr/import_buf_test.go` | Fix 2：Flush 后 overlay 清除，磁盘读正确 |
| 性能回归 | `cmd/perf_import/main.go`（调试工具） | 端到端：6m → 目标 < 3m |

现有的边界情况调试代码（`cmd/perf_import/main.go` 场景 1-4）可直接改写为正式测试用例。

---

## 不在范围内

- CHANGELOG / README 的更新（文档任务，单独处理）
- Fix 3 parse pipeline（下一个迭代）
- Windows 兼容性修复（SQLite DSN、os.Rename 问题，独立 issue）
- 其他 executor 命令的缓存（关联、枚举等），仅 DomainModel 频率最高
