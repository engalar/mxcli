# Executor BSON 反模式清除设计

**日期：** 2026-05-27  
**范围：** `mdl/executor/` — 全量清除直接 BSON 导入（12 个文件）  
**目标：** executor 代码只通过 `ctx.Backend.*` 访问数据，不再 import `bson`/`codec`/`primitive`  

---

## 背景

CLAUDE.md 明确约束：

> "No direct BSON imports in executor — executor files must not call modelsdk/codec or backend-internal types directly; use `ctx.Backend.*` instead."

扫描发现 12 个文件违反此约束：

| 严重程度 | 文件数 | 主要问题 |
|---------|-------|---------|
| 严重 | 1 | `cmd_pages_builder_v3.go`：37 个 `bson.D{}`/`bson.A{}` 字面量 |
| 高 | 6 | 混用 `codec.*` + `bson.Marshal` |
| 中 | 5 | 只读 `codec` 调用或 `primitive.*` 类型断言 |

---

## 修复方案：分类批量重构

### 总体分批

| 批次 | 文件数 | 违规性质 | 策略 |
|-----|------|---------|------|
| Batch 1：format/describe | 8 | codec 读 BSON 字段用于输出 | gen-typed 路径 + 补缺 getter |
| Batch 2：builder | 1 | BSON 字面量构建 widget | 重定义 `DataGridSpec`，下沉 BSON 到 mpr |
| Batch 3：diff/raw-setter | 3 | diff 比对 + raw 字段写入 | 调研阶段（见下）|
| Guard | — | — | Go 测试防护 |

每批完成后独立提交，Guard 在 Batch 2 完成后加入。

---

## Batch 1 — format/describe（8 文件）

### 分组

**组 A（已干净或仅残留无用 import）**

文件：
- `cmd_pages_describe_pluggable.go`
- `cmd_pages_describe_output.go`
- `cmd_microflows_format_action_gen.go`

操作：移除无用 import，验证编译即可。无需新增 backend 方法。

---

**组 B（gen getter 缺口）**

文件：
- `cmd_microflows_format_data_gen.go`
- `cmd_microflows_format_external_gen.go`
- `cmd_workflows_gen.go`

问题：gen 类型缺少某些字段的 getter，executor 因此退而求其次直接读 BSON。

缺失 getter 汇总：

| gen 类型 | 缺失 getter | 所在域 |
|---------|-----------|------|
| `CastAction` | `ObjectVariableName()` | microflows |
| `ConstantRange` | `LimitExpression()`、`OffsetExpression()` | microflows |
| `RetrieveAction` | （BSON key 为 `ResultVariableName`，与 gen getter 对应） | microflows |
| `RestCallAction.ResultHandling` | 内部 BSON key 与 gen 期望不一致 | microflows |
| `Annotation` | `Description()` | workflows |

操作：
1. 新增 supplement 文件：
   - `modelsdk/gen/microflows/supplement_format.go` — 补 `CastAction`/`ConstantRange` 的缺失 getter
   - `modelsdk/gen/workflows/supplement_format.go` — 补 `Annotation.Description()` getter
2. executor 改为从 gen 对象读字段，移除 codec import

---

**组 C（JavaAction 命名空间混乱）**

文件：
- `cmd_javaactions_gen.go`

问题：
- gen 注册名是 `JavaActions$X`，但 BSON 存的是 `CodeActions$X`
- 多个类型（`VoidType`、`LongType`、`FileDocumentType` 等）未在 DefaultRegistry 注册工厂
- 参数字段双轨：`Parameters` vs `ActionParameters`
- `MicroflowActionInfo.Icon` 的 `ImageData` 字段无 getter

操作：
1. 新增类型定义 `mdl/types/javaaction_desc.go`：
   ```go
   type JavaParamDesc struct {
       Name     string
       Category string  // "basic", "entity", "enum", "list", ...
       TypeName string
       // ... 其他展示字段
   }
   ```
2. `mdl/backend/backend.go` → `JavaBackend` 接口增加：
   ```go
   DescribeJavaActionParameters(id model.ID) ([]types.JavaParamDesc, error)
   ```
3. `mdl/backend/mpr/javaactions.go` → 实现该方法，封装命名空间映射、工厂注册兜底、双轨字段选择
4. `mdl/backend/mock/mock_java.go` → 增加 `DescribeJavaActionParametersFunc` 字段 stub
5. executor `cmd_javaactions_gen.go` → 调用 `ctx.Backend.DescribeJavaActionParameters()`，移除 codec import

---

**组 D（类型未注册）**

文件：
- `cmd_microflows_format_calls_gen.go`

问题：`ExpressionBasedCodeActionParameterValue` 未在 DefaultRegistry 注册，decoder 返回 `*element.Base`，executor 被迫读 raw BSON。

操作：在 gen 的 supplement 文件里补注册，executor 可直接 type-assert。

---

## Batch 2 — Builder（1 文件）

文件：`cmd_pages_builder_v3.go`（37 个 BSON 字面量）

### 问题

executor 在传给 `BuildDataGrid2WidgetGen()` 之前预先组装 BSON：
- `buildDataGridDataSourceBSON()` → 返回 `bson.D`
- `buildWidgetBSON()` → 返回 `bson.D`
- `buildMinimalClientTemplate()` → 返回 `bson.D`
- `buildMinimalAppearance()` → 返回 `bson.D`

这些 BSON 被塞进 `DataGridSpec` 结构体，再传给后端。

### 修复

1. **重定义 `DataGridSpec`**（在 `mdl/backend/` 中）：
   - 去掉 `bson.D` 字段，改为纯 Go 类型（struct）
   - 数据源用 `DataSourceSpec struct { Type string; EntityPath string; ... }` 替代 `bson.D`

2. **下沉 BSON 构建函数**：
   - `buildDataGridDataSourceBSON()` → `mdl/backend/mpr/datagrid_builder.go`（私有）
   - `buildWidgetBSON()` → `mdl/backend/mpr/datagrid_builder.go`（私有）
   - `buildMinimalClientTemplate()` → `mdl/backend/mpr/datagrid_builder.go`（私有）
   - `buildMinimalAppearance()` → `mdl/backend/mpr/datagrid_builder.go`（私有）

3. **executor 只传 Go 类型**：executor 不再触碰 BSON，只填充 `DataGridSpec`（纯 Go struct），再调用 `ctx.Backend.BuildDataGrid2WidgetGen()`。

4. **接口签名不变**：`BuildDataGrid2WidgetGen` 方法签名保持，仅 `DataGridSpec` 的内部字段类型变化；mock 的 Func 字段不需改动。

---

## Batch 3 — Diff/Raw-setter（调研阶段）

**文件：**
- `cmd_diff_local.go`（2 marshals + codec）
- `flowbuilder_raw_setter_gen.go`（1 marshal + codec）

**现有 `RawUnitBackend` 接口：**

```go
type RawUnitBackend interface {
    GetRawUnit(id model.ID) (bson.D, error)
    GetRawUnitBytes(id model.ID) ([]byte, error)
    ListRawUnitsByType(typeName string) ([]model.ID, error)
    UpdateRawUnit(id model.ID, doc bson.D) error
}
```

注意：`RawUnitBackend` 本身返回 `bson.D`，executor 调用它仍会拿到 BSON。

**调研问题：**

| 文件 | 问题 |
|-----|-----|
| `cmd_diff_local.go` | diff 操作的本质是比较两个 unit 的序列化表示；是否可以在 backend 增加 `DiffUnits(idA, idB model.ID) ([]DiffLine, error)` 方法，把比较逻辑下沉？ |
| `flowbuilder_raw_setter_gen.go` | 这是 microflow builder 的活动字段 setter；是否可以对标 PageMutator 引入 `MicroflowMutator` 模式？ |

**Batch 3 在本 spec 中标注为"第二阶段"——先完成 Batch 1/2，再做推演后补充设计。**

---

## Pre-commit Guard

**实现方式：** Go 测试文件，集成进 `make test`，无需额外工具。

文件：`mdl/executor/import_guard_test.go`

```go
// TestNoDirectBSONImportInExecutor 扫描 mdl/executor/*.go，
// 禁止直接 import BSON/codec 相关包。
// 所有 BSON 操作必须通过 ctx.Backend.* 进行。
func TestNoDirectBSONImportInExecutor(t *testing.T) {
    forbidden := []string{
        "go.mongodb.org/mongo-driver/bson",
        "go.mongodb.org/mongo-driver/bson/primitive",
        "go.mongodb.org/mongo-driver/bson/bsoncore",
        "github.com/mendixlabs/mxcli/modelsdk/codec",
    }
    // 用 go/parser 扫描 mdl/executor/*.go（排除 *_test.go 文件）
    // 对每个文件，检查 import 声明列表
    // 任何命中 forbidden 列表的 import 路径 → t.Errorf(...)
}
```

**加入时机：** Batch 2 完成后的独立 commit。

**效果：** 任何向 executor 引入直接 BSON import 的 commit 都会导致 `make test` 失败。

---

## 新增文件汇总

| 文件 | 类型 | 说明 |
|-----|-----|-----|
| `modelsdk/gen/microflows/supplement_format.go` | 新增 | 补 CastAction/ConstantRange 缺失 getter |
| `modelsdk/gen/workflows/supplement_format.go` | 新增 | 补 Annotation.Description() getter |
| `mdl/types/javaaction_desc.go` | 新增 | JavaParamDesc struct（executor ↔ backend 共享） |
| `mdl/backend/mpr/javaactions.go` | 修改 | 增加 DescribeJavaActionParameters() 实现 |
| `mdl/backend/mock/mock_java.go` | 修改 | 增加 DescribeJavaActionParametersFunc stub |
| `mdl/executor/import_guard_test.go` | 新增 | 禁止 executor 直接 import BSON 的 Go 测试 |

**接口变更：**
- `mdl/backend/backend.go` → `JavaBackend` 增加 `DescribeJavaActionParameters()` 方法
- `mdl/backend/` → `DataGridSpec` 数据源字段从 `bson.D` 改为纯 Go struct

---

## 实施顺序

```
Batch 1 组 A  →  Batch 1 组 D  →  Batch 1 组 B  →  Batch 1 组 C
      ↓                                                     ↓
  （最小风险）                                          （独立 commit）
                                                            ↓
                                                       Batch 2
                                                            ↓
                                                       Guard commit
                                                            ↓
                                                  Batch 3（第二阶段）
```

---

## 验收标准

- [ ] `make test` 通过（包括 import_guard_test.go）
- [ ] `make lint` 通过
- [ ] `grep -r '"go.mongodb.org/mongo-driver/bson"' mdl/executor/*.go` 输出为空
- [ ] `grep -r '"modelsdk/codec"' mdl/executor/*.go` 输出为空（排除测试文件）
- [ ] Batch 3 调研结论已记录（作为本 spec 的附录提交）
