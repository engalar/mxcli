# 开发流程总结：MPR v1 脚本缓冲区可见性修复

## 概述

本次会话修复了 mxcli 在 MPR v1 (SQLite) 上执行 EXECUTE SCRIPT 时的多个脚本缓冲区可见性问题。核心问题是：v1 格式的多个代码路径直接查询 SQLite，不检查内存中的脚本缓冲区 (`scriptInserts`/`scriptOverlay`)，导致新建/更新的 unit 对后续语句不可见。

## 流程步骤

### Step 1 — 最少复现测试
- 每个 bug 先写最小化测试复现，而不是猜根因
- 对 `listUnitsByTypeV1`: 写 `TestListUnitsByTypeV1_IncludesScriptInserts` 验证缓冲区新 unit 对 V1 查询可见
- 对后续 bug: 直接在 `app/minimal.mpr` 上执行 `helpdesk-app.mdl`

### Step 2 — 读代码 + Git 历史
- 从错误栈追踪到源码位置 (`grep -rn "error message"`)
- 沿调用链逐层读代码，不假设任何未读路径
- 发现 `domainmodel/entities.go` (实体创建路径) 与 `executor/cmd_create_entity_v2.go` (另一实体创建路径) 使用不同的缓存清除函数

### Step 3 — 单步验证假设
每次修复后立即验证效果，失败则继续深入：
1. `listUnitsByTypeV1` 加 `scriptInserts` 合并 → 测试通过但 MDL 仍失败 → 发现后端缓存未失效
2. `handler.go` 加 `Backend.InvalidateCache()` → 页面创建成功但 folder move 失败 → 发现 `UpdateUnitContainer` 不查缓冲区
3. `writer_core.go` 加 `scriptContainerUpdateBuf` → folder move 成功但语言添加后 `show languages` 不可见 → 发现 `listUnitsByTypeV1` 缺 overlay 检查
4. `reader_units.go` 加 overlay 检查 (含 `bsonFirstKeyIs` 防护) → mx check 仍 StudioLoadException → 发现 `bson.Marshal` 对 `bson.M` 使用随机 map 迭代

### Step 4 — 调试日志驱动
通过逐步添加最小化调试日志（`fmt.Fprintf(os.Stderr, "DEBUG ...")`）发现：
- `listUnitsByTypeV1` 的 overlay 命中模式：仅 KB domain model 命中，HD 未命中
- `ListDomainModelsGen.load()` 只在 PasswordForm 创建前被调用一次 → 后端缓存未失效
- `bson.Marshal` 对 `bson.M` 的随机键序遍历 → mx check 间歇性崩溃

### Step 5 — 修复演进
```
SQLite 查询不查缓冲区 → 加 scriptInserts 合并 (reader_units.go)
缓存清除不完整 → 加 Backend.InvalidateCache() (handler.go)
容器更新不查缓冲区 → 加 scriptContainerUpdateBuf (writer_core.go + script_buf.go)
listUnitsByTypeV1 缺 overlay → 加 overlay 检查 (reader_units.go)
序列化随机键序 → 58 处 bson.M→bson.D (serialize_*.go)
```

### Step 6 — 零回归守卫
每次修复后：
- `go test ./modelsdk/mpr/` 全量通过
- `make test` 全量通过
- 完整 MDL 执行 `go run ./cmd/mxcli exec -p app-golden/minimal.mpr helpdesk-app.mdl`
- `mx check` 验证

## 变更文件

| 文件 | 修复内容 |
|------|---------|
| `modelsdk/mpr/reader_units.go` | `listUnitsByTypeV1` 合并 `scriptInserts` + 检查 `scriptOverlay` + `bsonFirstKeyIs` 防护 |
| `modelsdk/mpr/reader.go` | `GetRawUnitBytes` V1 路径回退到 `scriptInserts` |
| `modelsdk/mpr/writer_core.go` | `UpdateUnitContainer` 加 `scriptContainerUpdateBuf` 拦截；`BatchWrite` 处理 `NewContainerID` |
| `mdl/backend/mpr/script_buf.go` | 新增 `AddContainerUpdate` + `containerUpdates` 映射 |
| `mdl/backend/mpr/script_tx.go` | 连线 `AddContainerUpdate` 回调 |
| `mdl/executor/domainmodel/handler.go` | `InvalidateDomainModelGenCache/InvalidateDomainModelsCache` 添加 `ectx.Backend.InvalidateCache()` |
| `modelsdk/mpr/serialize_mappings.go` | 7 处 `bson.M`→`bson.D` |
| `modelsdk/mpr/serialize_services.go` | 8 处 `bson.M`→`bson.D` |
| `modelsdk/mpr/serialize_web_services.go` | 43 处 `bson.M`→`bson.D` |
| `modelsdk/mpr/reader_units_test.go` | 新增 `TestListUnitsByTypeV1_IncludesScriptInserts` |

## 关键技术发现

### `bson.Marshal` 对 `bson.M` 不排序键
- Go 的 `bson.M` 是 `map[string]any`，`bson.Marshal` 使用 map 的自然迭代顺序（Go 随机化）
- 同个 map 多次序列化可能产生不同键序
- Mendix 存储引擎要求每个 storage object 的第一个 BSON 键必须是 `$ID`
- 修复：用 `bson.D{{Key: ..., Value: ...}}` 替代 `bson.M{...}`

### 多级缓存失效
- Executor 级别：`executorCache.domainModelsWithContainerGen` 等
- Backend 级别：`domainModelBackend.domainCache`
- Reader 级别：`overlay`, `scriptOverlay`, `scriptInserts`
- 不同代码路径使用不同级别的缓存清除函数，必须全部覆盖
