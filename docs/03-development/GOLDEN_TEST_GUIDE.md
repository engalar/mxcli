# Golden Test Guide — BSON 黄金测试开发指南

## 概述

Golden 测试是一套 BSON 级别的回归测试体系。核心思路：从 Studio Pro 生成的 MPR 中提取一个文档的原始 BSON，保存为 "golden file"，然后让 mxcli 的管道输出与 golden byte-by-byte 匹配。

与 `TESTING_GUIDE.md` 中定义的标准层（L4=Executor Gen, L5=Decode, L6a=Roundtrip）的关系：golden 测试是横切补充，在标准层之上增加了 BSON 精确度验证。标准层用 shape 断言（类型/字段/数量），golden 测试用字节级对比。

现分三层：

| 标签 | 测试名 | 位置 | 验证 |
|------|--------|------|------|
| Encoder Golden | `TestGoldenBSON` | `modelsdk/codec/golden/golden_test.go` | `codec.Encoder` 输出 BSON = golden |
| Pipeline MDL | `TestMDLToGolden` | `modelsdk/codec/golden/integration_test.go` | 手写 MDL → 完整管道 → BSON = golden |
| Full Roundtrip | `TestGoldenRoundtrip` | `mdl/executor/golden_roundtrip_test.go` | golden BSON → describe → MDL → 管道 → BSON ≈ golden |

### 文件结构

```
modelsdk/codec/golden/
├── entry.go                   ← GoldenEntry 类型定义
├── registry.go                ← Registry() — 所有 golden entry 注册
├── comparator.go              ← CompareBSON — 字节级 BSON diff 引擎
├── comparator_test.go         ← BSON diff 引擎单元测试
├── golden_test.go             ← TestGoldenBSON — encoder 黄金测试
├── nanoflow_builder.go        ← BuildNanoflow — element tree 构造器
├── integration_test.go        ← TestMDLToGolden — MDL 管道集成测试
├── memory/
│   └── mpr.go                 ← MemoryMPR 夹具（:memory: SQLite + 项目根单元）
├── snapshot/
│   ├── golden.go              ← UnitSnapshot + Canonical JSON 转换
│   ├── extract.go             ← 从真实 MPR 提取 golden
│   └── compare.go             ← 比较适配器（委托 comparator.go）
└── testdata/
    ├── MyFirstModule.Nanoflow.mxunit  ← golden BSON 文件（嵌入）
    └── nanoflow/create.golden.json    ← Canonical JSON golden

mdl/executor/
└── golden_roundtrip_test.go   ← TestGoldenRoundtrip — 双向 roundtrip 测试
```

---

## 三层详解

### Encoder Golden: `TestGoldenBSON` — Encoder 黄金测试

这是最底层、最严格的测试。它验证 `codec.Encoder` 从 gen 类型构建的 BSON 与 Studio Pro 输出的 BSON 完全一致。

**管道**：
```
Registry().Builder() → element tree → codec.Encoder.Encode → got []byte
                                                         ↓
                                              CompareBSON(got, entry.BSON)
```

**断言**：任何 diff（type/extra/missing/order/value/marker）→ `t.Errorf`，test FAIL。

**适用场景**：
- 修改 `codec.Encoder` 实现后验证无回归
- 添加新的 gen 类型时验证编码正确性

**不适用场景**：
- 验证 executor、backend、writer 书写逻辑

### Pipeline MDL: `TestMDLToGolden` — MDL 管道集成测试

验证手写的 MDL 通过完整的 executor/backend/writer 管道后，产生的 BSON 与 golden 一致。

**管道**：
```
手写 MDL → visitor.Build → AST → Executor.ExecuteProgram → MprBackend → Writer → SQLite
                                                                                   ↓
                                                                readUnitBSONByType → got
                                                                                       ↓
                                                                            CompareBSON(got, golden)
```

**基础设施**：
- `memory.NewFile`：创建临时 SQLite 文件，含 `_MetaData`、`Unit` 表、`Projects$Project` 根单元
- `mprbackend.New().Connect(path)`：打开 MprBackend
- `executor.New(output).SetBackend(backend)`：创建 Executor

**断言**：任何 BSON diff → `t.Errorf`，test FAIL。

**适用场景**：
- MDL 执行管道重构后验证无回归
- 验证新 MDL 语法的完整写入路径

### Full Roundtrip: `TestGoldenRoundtrip` — 双向 Roundtrip 测试

验证 `BSON → describe → MDL → parser → executor → BSON` 整个 identity。

**管道**：
```
entry.BSON → bsonToMDL(nil, unitType, name, content) → MDL text
                ↓
          sanitizeDescribedMDL (keyword rename, 删 TODO/@position/尾部/)
                ↓
          visitor.Build → AST → Executor.ExecuteProgram → SQLite
                                                              ↓
                                                   readUnitBSONByType → got
                                                                           ↓
                                                                CompareBSON(got, entry.BSON)
```

**关键细节**：
- `bsonToMDL` 是 `mdl/executor/diff_local.go` 中的非导出函数，所以此测试必须在 `package executor` 中
- 传入 `nil ctx`，离线执行（不需要 backend connection）
- 模块名通过 `extractModuleFromSource(entry.Source)` 从 `"Studio Pro X.Y.Z — Module.Element"` 提取
- `sanitizeDescribedMDL` 处理已知 describe 输出限制（见下文）

**断言策略**：

| 阶段 | 断言方式 | 说明 |
|------|----------|------|
| Phase 1-3 (parse + execute) | `t.Fatalf` (硬) | MDL 必须可解析、可执行 |
| Phase 4-5 (compare) | `t.Logf` (软) | BSON 差异因 describe TODO 预期存在 |

**为什么 Phase 4-5 是软断言？**

当前 `DescribeNanoflowGenToString` 有未实现的 TODO：
- Synchronize 活动→ `// TODO Stage 3.2.2: format *microflows.ActionActivity`
- ActionActivity 布局渲染

这些导致 golden Nanoflow 的 9 个活动对象中只有 3 个被正确渲染。一旦 TODO 完成，diff 会自然减少，无需修改测试。

---

## TDD 流程：添加新 Golden Entry

### 步骤 0：用 Studio Pro 创建参考文档

在 Studio Pro 中创建你要支持的新文档类型实例。导出该实例的原始 BSON。

### 步骤 1：提取 golden BSON

```bash
# 方法 A：用 snapshot 包提取
go test -run TestExtractNanoflowFromMinimal ./modelsdk/codec/golden/snapshot/ -v
# 输出显示 golden BSON 的 $Type 和大小

# 方法 B：用 bson dump
mxcli bson dump -p /tmp/minimal.mpr --type nanoflow --object "MyModule.MyNanoflow" --format raw > testdata/MyModule.MyNanoflow.mxunit
```

### 步骤 2：写 Builder（RED）

创建 Builder 函数，用 gen 类型构造与 golden 一致的 element tree。

```go
// nanoflow_builder.go
func BuildMyNanoflow() element.Element {
    nf := genMf.NewNanoflow()
    nf.SetName("MyNanoflow")
    return nf
}
```

先在 registry.go 中注册，运行测试确认失败：

```bash
go test -run TestGoldenBSON/MyNanoflow ./modelsdk/codec/golden/ -v
# 预期：FAIL — BSON 不匹配
```

### 步骤 3：迭代调整 Builder（GREEN）

逐步填充 Builder，每次 `go test -run TestGoldenBSON` 观察 diff 减少。

关键工具：
- `CompareBSON` 的 Diff 输出（value/missing/extra/order/type/marker）
- `GOLDEN_WRITE=1` 模式输出当前编码 BSON 以对比

```bash
GOLDEN_WRITE=1 go test -run TestGoldenBSON ./modelsdk/codec/golden/ -v
# 在 testdata/ 写入 MyNanoflow.bson
```

### 步骤 4：写 MDL 管道集成测试（RED → GREEN）

在 `integration_test.go` 的 `goldenCases()` 中添加 case：

```go
{
    Name: "mynanoflow/create",
    SetupMDL: `create module MyModule;
create entity MyModule.MyEntity (Name: String(200));`,
    MDL: `create nanoflow MyModule.MyNanoflow () returns list of MyModule.MyEntity as $List
{
  retrieve $List from MyModule.MyEntity;
  return $List;
}`,
    GoldenFile:     "mynanoflow/create.golden.json",
    TargetUnitType: "Microflows$Nanoflow",
},
```

运行 `GOLDEN_WRITE=1` 生成 golden JSON：

```bash
GOLDEN_WRITE=1 go test -run TestMDLToGolden/mynanoflow/create ./modelsdk/codec/golden/ -v
```

再正常模式运行确认通过：

```bash
go test -run TestMDLToGolden/mynanoflow/create ./modelsdk/codec/golden/ -v
```

### 步骤 5：写 Roundtrip 测试

如果 describe 已支持该文档类型，roundtrip 测试自动覆盖（遍历 `Registry()`）。

需要做的：
1. 在 `entry.go` 的 `GoldenEntry` 的 `SetupMDL` 中添加前置依赖
2. 如果描述类型涉及需要 rename 的关键字名称，在 `sanitizeDescribedMDL` 中添加
3. 运行确认：

```bash
go test -run TestGoldenRoundtrip ./mdl/executor/ -v
```

### 步骤 6：提交前验证

```bash
# 全量 golden 测试
go test ./modelsdk/codec/golden/... -v -count=1

# roundtrip 测试
go test -run TestGoldenRoundtrip ./mdl/executor/ -count=1

# vet
go vet ./modelsdk/codec/golden/... ./mdl/executor/
```

---

## BSON diff 解读

`CompareBSON` 报告 6 种 diff：

| Kind | 含义 | 常见原因 |
|------|------|----------|
| `missing` | golden 有但输出没有的字段 | Builder 漏设属性 |
| `extra` | 输出有但 golden 没有的字段 | Builder 多设了默认值未提供的字段 |
| `value` | 字段值不同 | 字符串、数字、布尔值错误 |
| `order` | 字段在文档中的顺序不同 | BSON 元素顺序不匹配 |
| `type` | 字段类型不同 | int32 vs int64、string vs binary |
| `marker` | PartList 版本标记不同 | 空列表的 marker 值不匹配 |

### 常见陷阱

**int32 vs int64**：Go 的 `bson.Marshal` 默认写 int32，但 Studio Pro 写 int64。如果 codec encoder 输出 int32，会触发 type diff。需要使用显式 `int64()` 转换。

**空列表 marker**：Mendix BSON 的空列表是 `[int32(2)]`（标记为版本 2），不是 `[]`。缺失 marker 导致数据结构解析失败。

**$ID 自动跳过**：`CompareBSON` 自动跳过所有 `$ID` 字段（UUID 每次生成不同）。如果其他字段也有动态值，用 `SkipFields` 列出路径。

---

## 常见问题

### Q: roundtrip 测试产生大量 diff

A: 正常。describe 的 TODO 列表：
- `// TODO Stage 3.2.2: format *microflows.ActionActivity`
- `@position` 指令未格式化

随着这些 TODO 实现，diff 会自然减少。用 `GOLDEN_WRITE=1` 可以更新 golden baseline。

### Q: 添加 golden entry 后 `TestGoldenRoundtrip` 报 `sql: no rows in result set`

A: 通常是 `SetupMDL` 不足——describe 出来的 MDL 引用了不存在的模块/实体。添加 `SetupMDL` 创建前置依赖。

### Q: "mismatched input '.' expecting '('" 解析错误

A: 文档名是关键字（如 `Nanoflow` 被 token 化为 `NANOFLOW`）。在 `sanitizeDescribedMDL` 中添加 rename 映射。

### Q: describe 输出模块名为 `<unknown>`

A: 调用 `bsonToMDL(nil, ...)` 时没有后端连接，模块名无法从 SQLite 解析。从 `entry.Source` 提取并在输出中 `ReplaceAll("<unknown>", module)`。

---

## 架构决策

### 为什么 roundtrip 测试在 `mdl/executor/` 而不是 `modelsdk/codec/golden/`？

`bsonToMDL` 是 `mdl/executor/diff_local.go` 中的非导出函数。为了调用它，测试必须在 `package executor` 中。L4.5 和 L5 测试在 `golden` 包（可以导出），因为 `CompareBSON` 和 `Registry()` 本身就是导出的。

### 为什么 roundtrip 的 Phase 4-5 不是硬断言？

describe 输出的完整性取决于 `DescribeNanoflowGenToString` 的实现进度。当前有多个 TODO。硬断言意味着每次 describe 添加新功能时都需要更新 golden，这是噪音。软报告让测试通过，同时文档化当前差距。

### 为什么 `GoldenEntry` 有 `SetupMDL` 字段？

describe 出来的 MDL 可能引用跨模块的实体/关联。`SetupMDL` 描述了这些前置依赖，在 roundtrip 测试中先执行。

### 为什么用 `memory.NewFile` 而不是 `setupTestEnv`？

`setupTestEnv` 复制真实项目文件，有 I/O 开销和依赖（需要共享项目）。`memory.NewFile` 零盘依赖，创建最小 SQLite 文件即可满足 mprbackend 连接要求。
