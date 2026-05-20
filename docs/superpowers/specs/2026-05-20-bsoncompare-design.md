# BSON Compare — Design Spec

Date: 2026-05-20  
Status: Approved

## Problem

mxcli 的 executor 写路径会产生 BSON 输出。目前没有系统性机制验证：
- 执行 MDL 脚本后，哪些 mxunit 文档发生了变化
- 变化是否符合预期（只改了该改的，没有意外改动）
- 代码重构后 BSON 输出是否退化

## 目标

在 `internal/bsoncompare` 实现一个 BSON 对比库，配合现有 `goldenfs` FUSE overlay，
支持端到端集成测试：对比 golden MPR（A）与 MDL 执行结果（B）的 BSON 差异。

## 不在范围内

- 替换现有 `makeRawPage()` / `makeWidget()` 单元测试
- 提供 CLI 子命令（可后续加）
- 非 Linux 平台（goldenfs FUSE 依赖 Linux）

---

## 架构

### A/B 来源

goldenfs 已提供完整的 copy-on-write FUSE overlay，无需另建抽象：

```
A = goldenDir (磁盘路径，只读)              ← testdata/expr-checker/ 等
B = snap.MountDir() (FUSE 挂载路径)         ← A + dirty layer（MDL 写入）

bsoncompare.Compare(aPath, bPath) → []UnitDiff
```

- **A** 直接指向磁盘上的 testdata，goldenfs 保证写操作不会穿透到 A
- **B** 是 FUSE 挂载路径，MDL executor 通过该路径写入，读取时透明合并 dirty layer
- 测试结束调用 `snap.Rollback()`，不污染 golden
- 更新 golden 时调用 `snap.Commit()` 然后 `git commit`

### 包结构

```
internal/bsoncompare/
├── mprreader.go   // 从 MPR 路径读所有原始 BSON unit（v1 SQLite / v2 mxunit 自动识别）
├── idmap.go       // BuildIDMap(units) — 全局 hexID → QualName 映射
├── normalize.go   // Normalize(bson.D, IDMap, Options) map[string]any
├── align.go       // 数组对齐：ByName / SetDiff / ByPosition
├── diff.go        // Compare(aPath, bPath, opts) []UnitDiff
├── options.go     // Options 及默认值
├── report.go      // FormatDiff([]UnitDiff) string
└── assert.go      // AssertEqual(t, aPath, bPath, matchers...) 薄包装
```

---

## 数据类型

```go
// options.go
type Options struct {
    IgnoreFields        []string // 追加到默认忽略集合
    IgnoreDocumentation bool     // 默认 true
    IgnoreLayout        bool     // 默认 true（ControlVector / Position*）
    IgnoreStableId      bool     // 默认 true（39% 文档有，新建时必然不同）
}

// idmap.go
type IDMap map[string]string  // hex(16B) → "ShortType:Name"

// diff.go
type DiffKind string
const (
    DiffChanged DiffKind = "changed"
    DiffAdded   DiffKind = "added"
    DiffRemoved DiffKind = "removed"
    DiffWarning DiffKind = "warning"  // <ref:?UNKNOWN>
)

type FieldDiff struct {
    Path   string
    Golden string
    Actual string
    Kind   DiffKind
}

type UnitDiff struct {
    UnitName string
    UnitType string
    Kind     DiffKind     // unit 本身 added/removed/changed
    Fields   []FieldDiff
}
```

---

## 核心算法

### 1. IDMap 构建（两张合并，workspace 优先）

```
BuildIDMap(mprPath):
  扫描所有 mxunit / Unit 表
  对每个元素递归：
    if 有 $ID (16B binary) → hex → key
    label = ShortType + ":" + Name  (无 Name 则用 parentName.fieldKey)
    存入 map[key] = label
```

真实数据：corpus-b 14万条 ID，coverage ≈ 100%（UNKNOWN 仅来自系统模块引用）

### 2. Normalize（9条规则）

| 规则 | 处理方式 |
|------|---------|
| `$ID` (self) | 忽略 |
| `*Pointer` / `*Ref` GUID | 替换为 `<ref:QualName>` |
| `StableId` | 忽略 |
| 布局字段（`ControlVector` / `Position*` / `CanvasHeight` / `CanvasWidth`） | 忽略 |
| `Documentation` | 默认忽略，可配置开启 |
| versioned-array 前缀 `int32` | 跳过（Mendix BSON 数组第 0 项是版本号） |
| 有 `Name` 字段的数组元素 | ByName 对齐 |
| 纯 `<ref:…>` 字符串数组 | SetDiff（无序集合） |
| 无 `Name` 的其他数组（如 `Flows` 连线） | ByPosition，只报数量变化 |

### 3. 数组对齐（运行时自动选择）

```
判断数组内容：
  if 所有元素都是 string("<ref:…>")  → SetDiff
  elif 元素是 map 且含 "Name" key    → ByName（锚定后递归 diffDoc）
  else                               → ByPosition（zip，仅报长度变化）
```

### 4. UnitCompare 流程

```
Compare(aPath, bPath, opts):
  aUnits = readAllUnits(aPath)      // UnitName → bson.D
  bUnits = readAllUnits(bPath)
  idMap  = merge(BuildIDMap(aPath), BuildIDMap(bPath))  // B 优先

  for name in union(aUnits, bUnits):
    if only in a  → UnitDiff{Kind: Removed}
    if only in b  → UnitDiff{Kind: Added}
    if in both:
      aN = Normalize(aUnits[name], idMap, opts)
      bN = Normalize(bUnits[name], idMap, opts)
      fields = diffDoc(aN, bN)
      if len(fields) > 0 → UnitDiff{Kind: Changed, Fields: fields}
```

---

## 测试集成

### 典型端到端测试

```go
func TestCreateMicroflow(t *testing.T) {
    const goldenDir = "../../testdata/expr-checker"

    snap, err := goldenfs.Open(goldenDir)
    require.NoError(t, err)
    defer snap.Close()
    defer snap.Rollback()

    mprPath := filepath.Join(snap.MountDir(), "minimal.mpr")
    runMDL(t, mprPath, `
        create microflow MyFirstModule.ACT_Test ()
        returns Nothing begin return; end;
    `)

    bsoncompare.AssertEqual(t,
        filepath.Join(goldenDir, "minimal.mpr"),  // A: baseline
        mprPath,                                   // B: post-MDL
        bsoncompare.ExpectAdded("ACT_Test"),
        bsoncompare.ExpectNoOtherChanges(),
    )
}
```

### 失败输出格式

```
--- FAIL: TestCreateMicroflow
    bsoncompare: 2 unit(s) differ

    [CHANGED] MyFirstModule (DomainModel)
      ~ .Entities[Customer].Attributes[CreatedDate].Type
          - Date
          + DateTime

    [ADDED] ACT_Test (Microflows$Microflow)
      + .ExportLevel = "Hidden"
      + .Parameters  = []
```

### Golden 更新流程

```bash
# 1. 在测试里临时调用 snap.Commit() 写回磁盘
# 2. mx check 验证新 golden 无 StorageLoadException
# 3. git add testdata/... && git commit -m "update golden"
```

### 与现有测试的关系

- 现有 `makeRawPage()` / `makeWidget()` 单元测试**不变**，测内部 helper
- `bsoncompare` 补充**集成层**：验证整条 MDL→executor→BSON 路径的输出
- 两者互补，不替代

---

## 推演验证的关键结论

从 corpus-b（4153 个 mxunit，14万条 IDMap 条目）的实际数据得出：

| 场景 | 结论 |
|------|------|
| 场景1：无变更对比 | 零差异，normalize 幂等 ✓ |
| 场景2：ExportLevel 变更 | 精确捕获单字段变化 ✓ |
| 场景3：跨文档 ref 解析 | IDMap 覆盖率接近 100% ✓ |
| 场景4：UNKNOWN ref | `<ref:?>` warning，不中断 ✓ |
| 场景5：新建微流（47个 GUID 全变） | Name 锚点对齐，GUID 全部忽略 ✓ |
| 场景6：新增参数 + 字段变更 | 混合 diff 输出正确 ✓ |
| 场景7：无 Name 的 Flows 数组 | 只报数量变化，不误报 ✓ |
| 场景8：AllowedModuleRoles set diff | 无序集合正确处理 ✓ |
| 场景9：StableId 噪声 | ignore 后不产生 diff ✓ |

---

## 实现顺序

1. `options.go` + `idmap.go` — IDMap 构建
2. `mprreader.go` — v1/v2 原始 BSON 读取
3. `normalize.go` — 9条规则
4. `align.go` — 三种数组策略
5. `diff.go` — `Compare()` 主函数
6. `report.go` — `FormatDiff()`
7. `assert.go` — `AssertEqual()` + matchers
8. 集成测试：接入 `internal/goldenfs/` 的现有 workflow_integration_test 模式
