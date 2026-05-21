# 导出/导入往返一致性分类检查

**日期：** 2026-05-21  
**范围：** `mdl/executor/` 测试层 + 新增 `testdata/roundtrip/app.mpr`

---

## 背景

`mxcli export` 把 MPR 项目导出为 14 种 MDL 文档类型，`mxcli import` 按拓扑顺序将其重新写入。当前有 10 种类型缺少专项往返测试，已通过用户 bug 报告（@Position 分号、GRANT 关联名前缀、GRANT 保留关键字）证明静默回归风险是真实存在的。

## 目标

对 10 种缺口类型在三个层面验证往返正确性：

| 层 | 断言内容 | 通过标准 |
|---|---|---|
| **L1 语法** | 导出的 MDL 字符串可被 parser 解析 | `assertRoundtrip()` 内隐式检查，无 parse error |
| **L2 语义** | describe → import → re-describe 输出等价 | 两次 DESCRIBE 字符串相同（经 normalizer 处理）|
| **L3 存储** | BSON 变化仅限预期 unit | `bsoncompare.AssertEqual` + `ExpectNoOtherChanges()` |

## 覆盖矩阵

| 类型 | L1 语法 | L2 语义 | L3 存储 | 说明 |
|---|---|---|---|---|
| Associations | ✓ | ✓ | ✓ | 跨实体引用，BSON 漂移风险高 |
| Constants | ✓ | ✓ | ✓ | 常量值/类型容易在序列化中丢失 |
| Module Roles | ✓ | ✓ | ✓ | 其他类型的安全引用基础 |
| User Roles | ✓ | ✓ | — | 依赖 Module Roles，语义层已足够 |
| Navigation | ✓ | ✓ | — | 结构复杂，引用少 |
| Layouts | ✓ | ✓ | — | 静态模板，语义层足够 |
| Snippets | ✓ | ✓ | — | Widget 树，语义层足够 |
| Settings | ✓ | ✓ | — | 项目级配置 |
| Java Actions | ✓ | — | — | MDL 无 body，语义等价性无法字符串比较 |
| JavaScript Actions | ✓ | — | — | 同上 |

## 架构

### 新增 testdata MPR

**路径：** `testdata/roundtrip/app.mpr`（Mendix 11.6.6，与 expr-checker 同版本）

**预置内容（`RoundtripModule`）：**

```
RoundtripModule
  ├── Entity: Item        (Name: String, Price: Decimal)
  ├── Entity: Category    (Label: String)
  ├── Association: Item_Category  (Item → Category, Many-to-One)
  ├── Enumeration: Status (Active, Inactive, Pending)
  ├── Constant: ApiBaseUrl        (String = "https://example.com")
  ├── Constant: MaxRetries        (Integer = 3)
  ├── Module Role: Viewer
  ├── Module Role: Editor
  ├── User Role: BasicUser        (→ Viewer)
  ├── User Role: PowerUser        (→ Editor)
  ├── Navigation Profile: Responsive  (首页 → Item overview page)
  ├── Layout: RoundtripLayout         (最小 Atlas 兼容 layout)
  ├── Snippet: ItemCard               (DataView + Label)
  ├── Java Action: ExternalCall       (param: String → returns String)
  └── JS Action: FormatDate           (param: DateTime → returns String)
```

MPR 通过 `mxcli new RoundtripApp --version 11.6.6` 创建后，用 `mxcli exec` 注入上述内容，提交到 `testdata/roundtrip/`。

### 测试文件

10 个新测试文件，全部位于 `mdl/executor/`：

```
roundtrip_association_test.go
roundtrip_constant_test.go
roundtrip_module_role_test.go
roundtrip_user_role_test.go
roundtrip_navigation_test.go
roundtrip_layout_test.go
roundtrip_snippet_test.go
roundtrip_settings_test.go
roundtrip_java_action_test.go
roundtrip_js_action_test.go
```

### 测试函数命名规范

```go
// L1: 语法层（全部 10 种）
func TestRoundtrip_<Type>_Syntax(t *testing.T)

// L2: 语义层（8 种）
func TestRoundtrip_<Type>_Semantic(t *testing.T)

// L3: 存储层（3 种）
func TestRoundtrip_<Type>_Storage(t *testing.T)
```

## 每层实现模式

### L1 语法层

使用现有 `assertRoundtrip()` 辅助函数。该函数内部执行 CREATE → DESCRIBE，若 DESCRIBE 输出能被 parser 解析则通过。对于无 CREATE 路径的类型（Java Action、JS Action），改用 `ctx.Backend.DescribeJavaAction()` 直接拿到导出字符串，再调用 `parseOnly(mdl)` 验证。

```go
func TestRoundtrip_Association_Syntax(t *testing.T) {
    ctx := newRoundtripCtx(t, roundtripMPRPath())
    out := describeAssociation(ctx, "RoundtripModule.Item_Category")
    requireParseOK(t, out)
}
```

### L2 语义层

```go
func TestRoundtrip_Association_Semantic(t *testing.T) {
    ctx := newRoundtripCtx(t, roundtripMPRPath())

    mdl1 := describeAssociation(ctx, "RoundtripModule.Item_Category")
    execMDL(ctx, mdl1)                                          // re-import
    mdl2 := describeAssociation(ctx, "RoundtripModule.Item_Category")

    assertNormalizedEqual(t, mdl1, mdl2)
}
```

`assertNormalizedEqual` 复用 `roundtrip_helpers_test.go` 里已有的 normalizer（去除 `@Position`、尾分号、空白差异）。

### L3 存储层

```go
func TestRoundtrip_Association_Storage(t *testing.T) {
    base := readAllUnits(roundtripMPRPath())

    ctx := newRoundtripCtx(t, roundtripMPRPath())
    mdl := describeAssociation(ctx, "RoundtripModule.Item_Category")
    execMDL(ctx, mdl)   // re-import（create or modify）

    after := readAllUnits(roundtripMPRPath())
    bsoncompare.AssertEqual(t, base, after,
        bsoncompare.ExpectNoOtherChanges(),
    )
}
```

存储层要求 re-import 后 MPR 与原始状态 BSON 完全一致（zero-diff），证明导出格式是幂等的。

## 辅助函数

在 `roundtrip_helpers_test.go` 新增（或新建 `roundtrip_roundtrip_mpr_test.go`）：

```go
// roundtripMPRPath 返回专用测试 MPR 的路径
func roundtripMPRPath() string {
    return filepath.Join(testdataDir(), "roundtrip", "app.mpr")
}

// requireParseOK 验证 MDL 字符串无 parse error
func requireParseOK(t *testing.T, mdl string)

// describeXxx 系列函数：每种类型一个 describe 辅助
func describeAssociation(ctx *ExecContext, qn string) string
func describeConstant(ctx *ExecContext, qn string) string
func describeModuleRole(ctx *ExecContext, qn string) string
// ... 以此类推
```

## 执行顺序

1. 创建并提交 `testdata/roundtrip/app.mpr`
2. 在 `roundtrip_helpers_test.go` 添加公共辅助函数
3. 按优先级顺序实现 10 个测试文件：
   - 批次 1（L3 高风险）：Associations、Constants、Module Roles
   - 批次 2（L2 语义）：User Roles、Navigation、Layouts、Snippets、Settings
   - 批次 3（L1 语法）：Java Actions、JavaScript Actions

## 成功标准

- `go test ./mdl/executor/...` 全绿
- 10 种类型全部有 L1 测试，8 种有 L2，3 种有 L3
- 三个 Bug（@Position、GRANT 关联名、GRANT 保留字）各有对应回归测试用例覆盖
- 新 MPR 提交后 `mx check testdata/roundtrip/app.mpr` 无报错

## 不在范围内

- 已有专项测试的 4 种类型（Entity、Microflow、Page、Security）不重复覆盖
- Workflow 类型已有独立测试，不纳入本次
- 导出增量缓存（`@cache:` marker）的正确性不在本次范围
