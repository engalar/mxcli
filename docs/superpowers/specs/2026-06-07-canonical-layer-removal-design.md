# canonical 生命周期层删除设计

**日期：** 2026-06-07  
**状态：** 已批准  
**范围：** `mdl/canonical/entity/`、`mdl/canonical/association/`、`mdl/canonical/registry.go`、`mdl/canonical/context.go`

---

## 背景与动机

`mdl/canonical/` 最初设计为一个通用中间层，用 Lift / Hydrate / Persist / Codec 模式覆盖所有 Mendix 域（实体、关联、微流、页面……）。但这一模式只落地了两个域（entity、association），剩余 50+ 域全部走 executor → backend 直接路径，证明该层无法合理扩展。

继续维护 canonical 生命周期层的代价：
- 额外的翻译层（AST → canonical model → gen types → BSON）与其他域的直接路径不一致
- `DefaultRegistry`、`HydrateFrom`、`RegisterCodec` 基础设施只被 6 个文件的少数调用点使用
- 新贡献者需要理解两套不同的写路径

---

## 范围界定

### 删除（生命周期基础设施）

| 路径 | 内容 |
|------|------|
| `mdl/canonical/entity/` | Lift / Hydrate / Persist / Codec（约 800 行）|
| `mdl/canonical/association/` | Lift / Hydrate / Persist / Codec（约 450 行）|
| `mdl/canonical/registry.go` | `DefaultRegistry`、`HydrateFrom`、`RegisterCodec`（约 83 行）|
| `mdl/canonical/context.go` | `HydrateCtx`（约 15 行）|

### 保留（共享类型）

| 路径 | 原因 |
|------|------|
| `mdl/canonical/datatype.go` | `DataType`、`DataTypeKind`、`Kind*` 常量——14 个 executor 文件广泛使用 |
| `mdl/canonical/doc.go` | 更新说明包职责已缩窄为共享类型 |

---

## 替代路径

### 2a. `cmd_diff_mdl.go` — 两处 `HydrateFrom` 替换

当前路径：
```
HydrateFrom(genEnt, HydrateCtx{...}) → EntityModel/AssocModel → ToMDL()
```

替换为直接用 gen-typed getter 内联构建 MDL 字符串（与现有 `viewEntityFromProjectToMDLGen` 模式一致）：

```go
// 替换前
doc, _, err := ctx.ModelCodecs.HydrateFrom(entity, canonical.HydrateCtx{...})
return doc.ToMDL() + ";\n/"

// 替换后
return entityGenToMDL(moduleName, entity) + ";\n/"  // 新私有函数
```

新增私有函数：
- `entityGenToMDL(moduleName string, entity *genDm.Entity) string`
- `assocGenToMDL(moduleName string, assoc *genDm.Association, entityNames map[string]string) string`

### 2b. `cmd_create_entity_gen.go` — `persistEntityCanonical` 替换

当前路径：
```
Lift(stmt) → EntityModel → Persist(PersistContext) → buildGenEntity → backend.CreateEntityGen()
```

替换为：
```
buildGenEntityFromAST(stmt) → backend.CreateEntityGen()
```

`buildGenEntityFromAST` 是将 `entity/lift.go` 的 Lift 逻辑与 `entity/persist.go` 的 `buildGenEntity()` 合并后移入 executor 的私有函数（约 150 行）。关联 create 同理：`buildGenAssocFromAST(stmt)` → `backend.CreateAssociationGen()`。

**无需新增 backend 方法**：以下方法已存在于 `DomainModelBackend` 接口：
- `CreateEntityGen(domainModelID, entity)`
- `UpdateEntityGen(domainModelID, entity)`
- `CreateAssociationGen(domainModelID, assoc)`
- `GetEntityIDByQualifiedName(qualifiedName)`

### 2c. executor 层清理

| 文件 | 变更 |
|------|------|
| `exec_context.go` | 删除 `ModelCodecs *canonical.DefaultRegistry` 字段 |
| `executor.go` | 删除 `hydrateEntityModel()`、`NewDefaultRegistry()`、两个 `RegisterCodec()` 调用（约 20 行）|
| `executor_dispatch.go` | 移除 canonical 相关 import |

---

## 测试策略

### 删除后验证

| 测试 | 期望结果 |
|------|---------|
| `go build ./...` | 零报错 |
| `go test ./mdl/executor/... ./mdl/canonical/...` | 零 FAIL |
| `TestNoDirectBSONImportInExecutor` | 继续通过（本次不涉及 BSON import）|

### 新增 import guard

在 `mdl/canonical/import_guard_test.go` 中新增规则：禁止 executor 导入 `mdl/canonical/entity`、`mdl/canonical/association`、`mdl/canonical/registry`（确保删除后不反弹）。

### 新增单元测试

```go
// cmd_diff_mdl_test.go
func TestEntityGenToMDL_roundtrip(t *testing.T)  // gen entity → MDL 包含关键字段
func TestAssocGenToMDL_roundtrip(t *testing.T)   // gen assoc + entityNames → MDL from/to/type
```

### 不做

- 不引入新 golden snapshot
- 不修改 helpdesk testdata
- `mdl/canonical/entity/`、`mdl/canonical/association/` 中的测试文件随子包一起删除，不迁移（逻辑已覆盖在 executor 测试中）

---

## 实现文件汇总

| 文件 | 操作 |
|------|------|
| `mdl/canonical/entity/` (整个目录) | **删除** |
| `mdl/canonical/association/` (整个目录) | **删除** |
| `mdl/canonical/registry.go` | **删除** |
| `mdl/canonical/context.go` | **删除** |
| `mdl/canonical/doc.go` | **修改**（更新包说明）|
| `mdl/executor/exec_context.go` | **修改**（删除 ModelCodecs 字段）|
| `mdl/executor/executor.go` | **修改**（删除 hydrateEntityModel + 注册调用）|
| `mdl/executor/executor_dispatch.go` | **修改**（移除 import）|
| `mdl/executor/cmd_create_entity_gen.go` | **修改**（替换 persistEntityCanonical，内联 buildGenEntityFromAST）|
| `mdl/executor/cmd_diff_mdl.go` | **修改**（替换 entityToMDLGen / associationToMDLGen）|
| `mdl/executor/cmd_diff_mdl_test.go` | **修改**（新增两个 roundtrip 测试）|
| `mdl/canonical/import_guard_test.go` | **新增**（禁止 canonical 子包反向 import）|

---

## 风险与缓解

| 风险 | 缓解 |
|------|------|
| `buildGenEntityFromAST` 遗漏字段 | 从现有 `entity/lift.go` + `entity/persist.go` 直接移植，不重写逻辑 |
| `entityGenToMDL` 输出格式漂移 | 新增 roundtrip 测试 + `make test` 全量跑 |
| 其他文件仍引用已删包 | `go build ./...` 编译失败立即可见 |
