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

---

## BSON 片段移植诊断法

### 动机

传统 golden 测试是**回归验证**——比较两段 BSON 是否一致。但面对 CE0463 这类错误时，先有 diff 再有修复的顺序是：

```
diff → guess → fix → mx check → 猜对了吗？
```

BSON 片段移植法把顺序倒过来：

```
extract → transplant → mx check → 是这颗碎片导致 CE 吗？
```

这是**因果分析**而非相关分析：不是"这里 diff 了所以可能是根因"，而是"把这碎片放进去 CE 出现了／消失了，所以必是根因"。

### 场景

- CE0463 / CE1613 / CE7247 等 BSON 结构错误
- 需要精确定位是 Type 还是 Object 导致
- 需要确认某个字段的具体值（null vs absent vs empty string）是否触发 CE

### 原理

```
golden MPR（mx check ✅） + built 的 widget BSON 碎片 → mx check ❌ → 碎片含 CE 根因
golden MPR（mx check ✅） + golden 的 widget BSON 碎片 → mx check ✅ → 移植操作本身无害
                                           ↑
                                    二分缩小范围
```

### 操作步骤

#### 0. 准备环境

```bash
# 两个 MPR：一个 passing（golden），一个 failing（built）
# 使用 MPR v2 格式（SQLite）
cp fail.mpr  /tmp/built.mpr
cp pass.mpr  /tmp/golden.mpr
```

#### 1. 提取 failing widget 的全量 BSON

用 Python 操作底层 SQLite（MPR v2 的 `Unit` 表）：

```python
import sqlite3, json

bg = sqlite3.connect("/tmp/built.mpr")
cursor = bg.execute("SELECT UnitID, Contents FROM Unit")
for row in cursor:
    blob = row[1]
    if b"CustomWidgets$CustomWidget" in blob:
        name = extract_name_from_bson(blob)
        print(f"Widget: {name}, size={len(blob)}")
```

或用 Go 的 `codec.Store`：

```go
store, _ := codec.Open(path)
for _, u := range store.ListUnits() {
    raw, _ := store.LoadUnit(u.ID)
    elem, _ := decoder.Decode(raw)
    if cw, ok := elem.(*genCw.CustomWidget); ok {
        fmt.Printf("Widget: %s\n", cw.Name())
    }
}
```

#### 2. 移植到 golden

有两种手段，从简单到精确：

**方法 A：全量 widget 替换（SQLite 直写）**

```python
gg = sqlite3.connect("/tmp/golden.mpr")

# 在 golden 中找到同名 widget 的 UnitID
golden_row = gg.execute(
    "SELECT UnitID FROM Unit WHERE Contents LIKE ?",
    (b"%dgTickets%",)
).fetchone()

# 用 built 的 blob 替换
built_blob = bg.execute(
    "SELECT Contents FROM Unit WHERE UnitID=?",
    (golden_row[0],)
).fetchone()

gg.execute("UPDATE Unit SET Contents=? WHERE UnitID=?",
    (built_blob[0], golden_row[0]))
gg.commit()
```

```bash
mx check /tmp/golden.mpr | grep CE0463
# 出现 → 该 widget 必含 CE 根因
# 不出现 → 问题在页面/布局/多 widget 交互
```

**方法 B：BSON 子树替换（Go codec）**

更精细的替换——只替换 Type 骨架（PropertyTypes）或只替换 Object（WidgetValues）：

```go
// 解码两个版本
builtCW := decodeWidget(builtStore, "dgTickets")
goldenCW := decodeWidget(goldenStore, "dgTickets")

// 方案 1: 只替换 Type
raw := goldenCW.Raw()
goldenTypeBytes, _ := bson.Marshal(builtCW.Type().(bson.Marshaler))
newRaw := replaceBSONSubDoc(raw, "Type", goldenTypeBytes)
goldenStore.SaveUnit(goldenCW.ID(), newRaw)
// mx check → CE0463 出现？→ 根因在 Type，不出现→根因在 Object

// 方案 2: 只替换 Object
builtObjectBytes, _ := bson.Marshal(builtCW.Object().(bson.Marshaler))
newRaw := replaceBSONSubDoc(raw, "Object", builtObjectBytes)
goldenStore.SaveUnit(goldenCW.ID(), newRaw)
// mx check → CE0463 出现？→ 根因在 Object
```

**方法 C：局部值替换（BSON 手术）**

当已知可疑字段路径时，直接修改 golden 的 BSON 为该值，确认该值是否触发 CE：

```go
// 路径: Object.Properties[17].Value.TextTemplate
// 假设: golden 是 null，built 是 ClientTemplate
// 验证: 把 TextTemplate 设为 null 是否消除 CE0463？

// 或反过来：把 golden 的某个字段从 null 改为 ClientTemplate
// 看 CE0463 是否出现——确认该字段就是根因
```

#### 3. 二分搜索策略

```
built_widget → 替换 golden_widget → mx check
├── ❌ fail → 该 widget 含问题
│   ├── 替换 Type → mx check
│   │   ├── ❌ fail → Type 有问题
│   │   │   ├── 替换 PropertyType 前 20 个 → mx check
│   │   │   └── 替换 PropertyType 后 20 个 → mx check
│   │   └── ✅ pass → Type 没问题
│   └── 替换 Object → mx check
│       ├── ❌ fail → Object 有问题
│       │   ├── 替换 Properties[0-9] → mx check
│       │   ├── 替换 Properties[10-19] → mx check
│       │   └── ...
│       └── ✅ pass → Object 没问题
└── ✅ pass → 问题不在单 widget
```

每次二分只需 1 次 mx check，O(log N) 次即可定位根因字段。

### 工具辅助

#### 零依赖 SQLite BSON 检查（Python）

```python
def extract_bson_value(blob, path):
    """从 BSON 二进制中递归提取指定路径的值。
    path: ["Object", "Properties", 10, "Value", "TextTemplate"]
    """
    pos = 0
    for key in path:
        pos = find_field(blob, pos, key)
        if pos < 0:
            return None
    return read_value(blob, pos)
```

#### 现成的对比工具

项目已有：
- `cmd/ce0463-compare/` — 解码后对比元素树（跳过 BSON 顺序噪声）
- `cmd/ce0463-compare/dumper.go` — `dumpElement()` 输出 flat path→value map

### 片段合成：用 codec 生成精确 BSON 片段验证假设

前文的移植法依赖一个前提：已经有一个 built MPR 产出了包含可疑字段的 BSON。但更强大的方式是**直接用 codec 合成出特定 BSON 片段**，绕过整个 MDL→executor→backend 管道，直接验证某个字段值是否导致 CE。

#### 工作流

```
假设 → 合成 fragment → 植入 golden → mx check → 确认/排除
  ↑                                                    |
  └──────────── 调整 fragment ──────────────────────────┘
```

三步法：

**第一步：建立假设**

猜测某个字段的值是否导致 CE0463。例如：
```
"Property[17].Value.TextTemplate 从 ClientTemplate 改为 null 会消除 CE0463"
```

**第二步：合成预期片段**

用 codec 生成该字段的预期值 BSON：

```go
// Step 2a: 从 golden 解码目标 widget
store, _ := codec.Open("/tmp/golden.mpr")
goldenCW := decodeWidget(store, "dgTickets") // *genCw.CustomWidget

// Step 2b: 操作 element tree 验证假设
// 方案 A：直接修改 element 的 property
obj := goldenCW.Object()
for _, prop := range obj.Properties() {
    if prop.Name() == "Properties" { // PartList → WidgetProperties
        props := prop.(element.ChildListProperty).ChildElements()
        widgetProp := props[17].(*genCw.WidgetProperty)
        val := widgetProp.Value().(*genCw.WidgetValue)
        val.SetTextTemplate(nil) // 假设验证：设为 null
    }
}

// 方案 B：用 Encoder 重新编码 → 替换 golden 中的 Object
newObjectBytes, _ := codec.NewEncoder(codec.DefaultRegistry).Encode(obj)
newRaw := replaceBSONSubDoc(goldenCW.Raw(), "Object", newObjectBytes)
store.SaveUnit(goldenCW.ID(), newRaw)

// mx check /tmp/golden.mpr → CE0463 消失？→ 假设成立
```

**第三步：验证修复方案**

一旦确认字段是根因，用 codec 生成修复后的完整 widget 验证：

```go
// 用 codec 构造预期的正确 BSON
builder := genCw.NewCustomWidget()
builder.SetName("dgTickets")
// ... 设置所有字段为预期值

// 编码成 BSON
correctBSON, _ := codec.NewEncoder(codec.DefaultRegistry).Encode(builder)

// 替换 golden
store.SaveUnit(goldenCW.ID(), correctBSON)
// mx check → CE0463 消失 → 修复方向正确
```

#### 片段来源对照

| 来源 | 精度 | 周期 | 适合场景 |
|------|------|------|----------|
| **从 built MPR 提取** | 真实但不精确 | 秒级 | 初步定位哪个 widget / 哪个字段组出问题 |
| **手动构造 BSON 片段** | 精确控制每个字段 | 分钟级 | 验证单个字段假设（TextTemplate=null vs absent vs empty string） |
| **codec.Encode 合成** | 精确且可重复 | 毫秒级 | 大批量验证修复方案；回归测试 |
| **MDL→Executor→Backend 管道输出** | 端到端真实 | 30s+ | 最终验证完整管道 |

合成法的威力在于：**你可以在毫秒级生成一个"修复后的"Object BSON，移植到 golden 看 CE0463 是否消失**。如果消失，说明修复方向正确——你不需要等 30 秒跑完整 helpdesk 脚本才能知道。

#### 例：最小测试中的实操

我们在这次 CE0463 修复中多次陷入的循环：
```
修复 augment.go → build → exec MDL → mx check（30s）→ 还不行 → 再改
```

用合成法：
```
从 golden 提取 Object → Decode → 把 Properties[17].Value.TextTemplate 设 nil
→ Encode → 移植回 golden → mx check（2s）
→ CE0463 消失 ✓ → 确认 Properties[17] 是根因
→ 再生成修复后的 Object（TextTemplate=null for non-translation types）
→ 移植 → mx check ✓ → 修复方案已验证
→ 最后才去改 augment.go → 跑完整测试做回归
```

关键收益：**每个假设验证从 30 秒缩短到 2 秒**。

### 与现有 golden 测试的关系

| 方法 | 验证目标 | 手段 |
|------|---------|------|
| `TestGoldenBSON` | Encoder 输出 = golden | byte-by-byte 对比 |
| BSON 移植法 | 某 BSON 片段是否导致 CE 错误 | 移植到 golden MPR → mx check |
| 普通 golden | BSON 结构正确性（回归） | 字节级比较 |
| BSON 移植法 | CE 根因定位（诊断） | 因果验证 |

BSON 移植法不是替代 golden 测试，而是**诊断阶段的锋利工具**——在知道有 CE 错误但不知道哪个字段导致时使用。一旦定位根因，修复后再用 golden 测试做回归验证。

### 局限

- 只适用于 MPR v2（SQLite）。v1 单文件 `.mpr` 需要在内存中重新序列化
- 需要人工判断二分边界（哪些字段属于同一个 logical group）
- 部分 CE 错误可能由 **Type 与 Object 不匹配** 共同导致，需要替换整个 widget 才能复现
