# Page Model IR 统一重构设计

**日期**：2026-06-01  
**状态**：待实现  
**目标**：修复 DESCRIBE PAGE 输出空体 `{}`，同时通过引入统一中间模型（PageModel IR）彻底消除写路径与读路径的结构性分歧。

---

## 1. 问题背景

### 症状

`describe page X` 对所有页面输出空体 `{}`，导致 `describe-snapshot.mdl` 无法还原页面 widget 结构，往返（roundtrip）不成立。

### 根本原因

存在两条互不相知的路径：

| 路径 | 入口 | 数据流 | 问题 |
|------|------|--------|------|
| 写路径 | `cmd_pages_create_v3.go` | AST → `bson.D`/`genPg.*` 混合 → BSON | 直接操作 BSON，无中间模型 |
| 读路径 | `cmd_pages_describe.go` | BSON → `rawWidget`（仅读路径私有）→ MDL | `rawWidget` 解析逻辑与 gen-encoded BSON 结构不匹配 |

Gen-encoded（Cat-B）页面写入 `LayoutCall` key 和 `Widget`（DivContainer）结构，读路径的 `parseRawWidget()` 未能正确处理，导致 `getPageWidgetsFromRaw()` 返回空切片。

---

## 2. 设计目标

- **C（两者同步）**：建立统一 IR 规范 + 配套迁移测试，写路径和读路径并行重构，不允许快捷绕过
- **IR 选择 B**：在 `mdl/types/page.go` 新建 MDL 层 `PageModel`/`WidgetNode`，两端分别实现转换器
- **测试选择 B**：新增专门的 roundtrip 单元测试，每种 widget 类型一个，独立于黄金测试

---

## 3. 整体架构

```
┌──────────────────────────────────────────────────────┐
│                   MDL Executor 层                      │
│                                                        │
│  CREATE PAGE:  AST ──→ pageASTToModel() ──→ PageModel  │
│                              │                         │
│                 ctx.Backend.WritePageModel()            │
│                                                        │
│  DESCRIBE PAGE: ctx.Backend.GetPageModel() ──→ PageModel│
│                              │                         │
│                 pageModelToMDL() ──→ MDL 文本           │
└──────────────────────────────────────────────────────┘
                        ↕  types.PageModel
┌──────────────────────────────────────────────────────┐
│              mdl/backend/mpr/ 层                       │
│                                                        │
│  GetPageModel():  BSON → fromBSON() → *types.PageModel │
│  WritePageModel(): *types.PageModel → toBSON() → BSON  │
│                                                        │
│  所有 BSON 字段名硬编码集中于 page_model.go，不外泄    │
└──────────────────────────────────────────────────────┘
```

### 各层职责

| 包 | 职责 | 新增文件 |
|----|------|---------|
| `mdl/types/` | PageModel、WidgetNode 类型定义（共享契约，无逻辑） | `page.go`（新建） |
| `mdl/backend/` | 接口扩展：PageModelBackend | `page_model.go`（新建） |
| `mdl/backend/mpr/` | BSON ↔ PageModel 转换实现 | `page_model.go`（新建） |
| `mdl/backend/mock/` | GetPageModel / WritePageModel 的 Func 字段 stub | `mock_page_model.go`（新建） |
| `mdl/executor/` | AST→PageModel（写）、PageModel→MDL（读）；不碰 BSON | `cmd_pages_ast_to_model.go`、`cmd_pages_model_to_mdl.go`（新建） |
| `mdl/executor/` | 删除 | `cmd_pages_describe_parse.go`、`cmd_pages_describe_output.go`（删除） |

### 关键约束

- `executor/` 中禁止 import `modelsdk/codec` 或 raw BSON 包（`TestNoDirectBSONImportInExecutor` 守护）
- 所有 BSON 字段名硬编码集中在 `mdl/backend/mpr/page_model.go`

---

## 4. PageModel 类型定义（mdl/types/page.go）

### 顶层结构

```go
type PageModel struct {
    ModuleName string
    Name       string
    Title      string
    Layout     string       // e.g. "Atlas_Core.Atlas_Default"
    Folder     string       // e.g. "Ticket/Search"
    Params     []PageParam
    Variables  []PageVariable
    Widgets    []*WidgetNode
}

type PageParam    struct { Name, EntityName string }
type PageVariable struct { Name, EntityName string; IsList bool }
```

### WidgetNode

```go
type WidgetNode struct {
    Kind      WidgetKind
    Name      string
    Children  []*WidgetNode

    // === 数据绑定（dataview/datagrid/gallery/listview 用）===
    DataSource *DataSourceDef
    EntityAttr  string   // 输入控件属性绑定
    EntityCtx   string   // dataview 提供给子控件的实体类型

    // === 显示 ===
    Caption     string   // button/label 文本
    Content     string   // text/title 静态内容

    // === 布局（column 用）===
    ColWidth    ColWidthDef

    // === 动作 ===
    OnClick     string   // microflow/nanoflow/page 限定名
    ButtonStyle string   // Primary/Success/Warning/Danger/Default/Link/Icon

    // === 输入控件公共属性 ===
    Editable    string   // "Always"/"Never"/"Conditional"
    EditableIf  string   // Conditional 时的表达式
    ShowLabel   bool
    LabelPos    string   // "Left"/"Top"
    ReadOnly    string   // "Inherit"/"Control"/"Text"

    // === 条件可见 ===
    VisibleIf   string

    // === 外观 ===
    Class       string
    Style       string
    DesignProps []DesignProp

    // === Kind-specific 子结构（非对应 Kind 时为 nil）===
    GroupBox  *GroupBoxProps
    DataGrid  *DataGridProps
    Gallery   *GalleryProps
    Image     *ImageProps
    Snippet   *SnippetProps
    Unknown   *UnknownProps   // 未识别 pluggable widget
}

type ColWidthDef struct {
    Desktop, Tablet, Phone int  // 1-12，0 = 未设置
}
```

### Kind-specific 子结构

```go
type GroupBoxProps struct {
    Collapsible string  // "No"/"YesInitiallyExpanded"/"YesInitiallyCollapsed"
    HeaderMode  string  // "Div"/"H1"-"H6"
}

type DataGridProps struct {
    Columns       []ColumnDef
    FilterWidgets []*WidgetNode
    ControlBar    []*WidgetNode
    PageSize      int
    Pagination    string  // "buttons"/"virtualScrolling"/"loadMore"
    PagingPos     string  // "bottom"/"top"/"both"
}

type ColumnDef struct {
    Name, Attribute, Caption string
    ShowContentAs    string       // "attribute"/"customContent"/"dynamicText"
    ContentWidgets   []*WidgetNode
    DynamicText      string
    Alignment        string       // "left"/"center"/"right"
    WrapText, Sortable, Resizable, Draggable bool
    Hidable          string       // "yes"/"hidden"/"no"
    ColumnWidth      string       // "autoFill"/"autoFit"/"manual"
    Size, Visible, CellClass, Tooltip string
}

type GalleryProps struct {
    DesktopColumns, TabletColumns, PhoneColumns int
    Selection      string   // "Single"/"Multi"/"None"
    FilterWidgets  []*WidgetNode
    ContentWidgets []*WidgetNode  // template body
}

type ImageProps struct {
    URL, AltText    string
    Width, Height   string
    WidthUnit       string  // "auto"/"pixels"/"percentage"
    HeightUnit      string  // "auto"/"pixels"/"percentage"/"viewport"
    DisplayAs       string  // "fullImage"/"thumbnail"
    Responsive      bool
    ImageType       string  // "image"/"imageUrl"/"icon"
    OnClickType     string  // "action"/"enlarge"
}

type SnippetProps struct {
    SnippetName string  // qualified name
}

type UnknownProps struct {
    WidgetID      string  // e.g. "com.mendix.widget.custom.switch.Switch"
    ExplicitProps []ExplicitProp
}

type ExplicitProp struct {
    Key, Value string
    IsRef      bool
}
```

### 数据源类型

```go
type DataSourceDef struct {
    Kind            DataSourceKind
    Reference       string   // mf/nf/param 限定名
    Entity          string   // database 源实体
    XPathConstraint string
    SortColumns     []SortDef
}

type DataSourceKind string
const (
    DataSourceDatabase  DataSourceKind = "database"
    DataSourceMicroflow DataSourceKind = "microflow"
    DataSourceNanoflow  DataSourceKind = "nanoflow"
    DataSourceParameter DataSourceKind = "parameter"
    DataSourceSelection DataSourceKind = "selection"
)

type SortDef struct {
    Attribute string
    Order     string  // "ASC"/"DESC"
}

type DesignProp struct {
    Key, Option string
    ValueType   string  // "toggle"/"option"
}
```

### WidgetKind 枚举与 BSON $type 映射

```go
type WidgetKind string

const (
    WidgetContainer    WidgetKind = "container"
    WidgetScrollView   WidgetKind = "scrollview"
    WidgetGroupBox     WidgetKind = "groupbox"
    WidgetLayoutGrid   WidgetKind = "layoutgrid"
    WidgetLayoutRow    WidgetKind = "row"
    WidgetLayoutCol    WidgetKind = "column"
    WidgetTabContainer WidgetKind = "tabcontainer"
    WidgetTabPage      WidgetKind = "tabpage"
    WidgetDataView     WidgetKind = "dataview"
    WidgetListView     WidgetKind = "listview"
    WidgetGallery      WidgetKind = "gallery"
    WidgetButton       WidgetKind = "button"
    WidgetTextBox      WidgetKind = "textbox"
    WidgetTextArea     WidgetKind = "textarea"
    WidgetDatePicker   WidgetKind = "datepicker"
    WidgetRadioButtons WidgetKind = "radiobuttons"
    WidgetCheckBox     WidgetKind = "checkbox"
    WidgetLabel        WidgetKind = "label"
    WidgetText         WidgetKind = "text"
    WidgetDynamicText  WidgetKind = "dynamictext"
    WidgetTitle        WidgetKind = "title"
    WidgetNavList      WidgetKind = "navigationlist"
    WidgetSnippet      WidgetKind = "snippet"
    WidgetDataGrid     WidgetKind = "datagrid"   // CustomWidget type=datagrid2
    WidgetComboBox     WidgetKind = "combobox"   // CustomWidget type=combobox
    WidgetImage        WidgetKind = "image"      // CustomWidget type=image
    WidgetUnknown      WidgetKind = "unknown"    // 兜底
)
```

**BSON $type → WidgetKind 映射**（锁定在 `mdl/backend/mpr/page_model.go`）：

| BSON $type（v1 Forms$ / v2 Pages$） | WidgetKind |
|--------------------------------------|------------|
| `Forms$DivContainer` / `Pages$DivContainer` | `container` |
| `Forms$ScrollContainer` / `Pages$ScrollContainer` | `scrollview` |
| `Forms$GroupBox` / `Pages$GroupBox` | `groupbox` |
| `Forms$LayoutGrid` / `Pages$LayoutGrid` | `layoutgrid` |
| `Forms$LayoutGridRow` / `Pages$LayoutGridRow` | `row` |
| `Forms$LayoutGridColumn` / `Pages$LayoutGridColumn` | `column` |
| `Forms$TabControl` / `Pages$TabControl` | `tabcontainer` |
| `Pages$TabPage` | `tabpage` |
| `Forms$DataView` / `Pages$DataView` | `dataview` |
| `Forms$ListView` / `Pages$ListView` | `listview` |
| `Forms$Gallery` / `Pages$Gallery` | `gallery` |
| `Forms$ActionButton` / `Pages$ActionButton` | `button` |
| `Forms$TextBox` / `Pages$TextBox` | `textbox` |
| `Forms$TextArea` / `Pages$TextArea` | `textarea` |
| `Forms$DatePicker` / `Pages$DatePicker` | `datepicker` |
| `Forms$RadioButtons` / `Pages$RadioButtons` | `radiobuttons` |
| `Forms$CheckBox` / `Pages$CheckBox` | `checkbox` |
| `Forms$Label` / `Pages$Label` | `label` |
| `Forms$Text` / `Pages$Text` | `text` |
| `Forms$DynamicText` / `Pages$DynamicText` | `dynamictext` |
| `Forms$Title` / `Pages$Title` | `title` |
| `Forms$NavigationList` / `Pages$NavigationList` | `navigationlist` |
| `Forms$SnippetCallWidget` / `Pages$SnippetCallWidget` | `snippet` |
| `CustomWidgets$CustomWidget` (type=datagrid2) | `datagrid` |
| `CustomWidgets$CustomWidget` (type=gallery) | `gallery` |
| `CustomWidgets$CustomWidget` (type=combobox) | `combobox` |
| `CustomWidgets$CustomWidget` (type=image) | `image` |
| `CustomWidgets$CustomWidget` (其他) | `unknown` |

---

## 5. Backend 接口扩展

### PageModelBackend 接口（mdl/backend/page_model.go）

```go
type PageModelBackend interface {
    GetPageModel(id model.ID) (*types.PageModel, error)
    GetSnippetModel(id model.ID) (*types.PageModel, error)
    GetLayoutModel(id model.ID) (*types.PageModel, error)
    WritePageModel(id model.ID, m *types.PageModel) error
    WriteSnippetModel(id model.ID, m *types.PageModel) error
}

// 编译期守卫（mdl/backend/mpr/backend.go）
var _ backend.PageModelBackend = (*MprBackend)(nil)
```

### FullBackend 扩展（mdl/backend/backend.go）

```go
type FullBackend interface {
    // ... 现有接口 ...
    PageBackend
    PageModelBackend  // 新增
    // ...
}
```

### Mock Stub（mdl/backend/mock/mock_page_model.go）

```go
func (m *MockBackend) GetPageModel(id model.ID) (*types.PageModel, error) {
    if m.GetPageModelFunc != nil { return m.GetPageModelFunc(id) }
    return nil, fmt.Errorf("MockBackend.GetPageModel not configured")
}

func (m *MockBackend) WritePageModel(id model.ID, pm *types.PageModel) error {
    if m.WritePageModelFunc != nil { return m.WritePageModelFunc(id, pm) }
    return fmt.Errorf("MockBackend.WritePageModel not configured")
}
// GetSnippetModel, GetLayoutModel, WriteSnippetModel 同上模式
```

---

## 6. 测试策略

### Roundtrip 验收不变式

```
describe(create(X)) == describe(modify(describe(create(X))))
```

第二次 describe 必须与第一次相同。这是真正的往返稳定保证。

### 测试结构（roundtrip_page_model_test.go）

```go
// 命名规则：TestRoundtrip_Page_<WidgetKind>
func TestRoundtrip_Page_DataGridColumns(t *testing.T) {
    env := setupTestEnv(t)
    defer env.teardown()

    // 步骤1：CREATE PAGE
    env.executeMDL(`create page M.P (...) { datagrid dg (...) { column colA (...) } }`)

    // 步骤2：DESCRIBE PAGE
    described := env.describeMDL(`describe page M.P;`)

    // 步骤3：验证语义
    prog, _ := visitor.Build(described)
    page := requirePageStmt(t, prog, "M.P")
    dg := requireWidget(t, page, "datagrid")
    assert.Len(t, dg.Columns, 1)

    // 步骤4：稳定性验证
    env.executeMDL(described)
    redescribed := env.describeMDL(`describe page M.P;`)
    assert.Equal(t, described, redescribed)
}
```

### 初始版本必须覆盖的 WidgetKind（helpdesk app 使用）

`container`、`layoutgrid`、`dataview`、`datagrid`、`gallery`、`button`、`textbox`、`tabcontainer`、`groupbox`、`snippet`

### 黄金测试补充断言

在 `TestHelpdeskGolden_DescribeSnapshot` 中增加：每个 `create or modify page` 语句的 body 至少包含一个 widget 关键字，防止空体被快照静默接受。

---

## 7. 五阶段迁移计划

### 阶段 1：定义 IR 契约（纯类型，零逻辑）

**新建：**
- `mdl/types/page.go` — PageModel, WidgetNode 及所有子类型
- `mdl/backend/page_model.go` — PageModelBackend 接口
- `mdl/backend/mock/mock_page_model.go` — Func 字段 stub

**修改：**
- `mdl/backend/backend.go` — FullBackend 加入 PageModelBackend

**验收**：`make build` 通过（无运行时行为改变）

---

### 阶段 2：写 Roundtrip 测试（先红）

**新建：**
- `mdl/executor/roundtrip_page_model_test.go` — 9 个核心 widget 类型测试

**修改：**
- `internal/goldenfs/helpdesk_regression_test.go` — 加 page-body-nonempty 断言

**验收**：测试有预期的失败（建立验收基线）

---

### 阶段 3：实现读路径（BSON → PageModel → MDL）

**新建：**
- `mdl/backend/mpr/page_model.go` — `fromBSON()` 实现（含完整 $type 映射表）
- `mdl/executor/cmd_pages_model_to_mdl.go` — `pageModelToMDL()` 渲染器

**修改：**
- `mdl/executor/cmd_pages_describe.go` — 换用 `ctx.Backend.GetPageModel()` → `pageModelToMDL()`

**删除（阶段3完成后）：**
- `mdl/executor/cmd_pages_describe_parse.go`（903 行）
- `mdl/executor/cmd_pages_describe_output.go`（1354 行）
- `mdl/executor/cmd_pages_describe_pluggable.go`（1244 行）— pluggable widget BSON 解析逻辑并入 `fromBSON()`

**验收**：阶段2 roundtrip 测试转绿；`mx check` 无新增 StorageLoadException

---

### 阶段 4：实现写路径（AST → PageModel → BSON）

**新建：**
- `mdl/executor/cmd_pages_ast_to_model.go` — `pageASTToModel()` 转换器

**修改：**
- `mdl/executor/cmd_pages_create_v3.go` — 换用 `pageASTToModel()` → `ctx.Backend.WritePageModel()`
- `mdl/executor/cmd_pages_builder_v3.go` — widget 构建逻辑迁移到 `pageASTToModel()`；仅保留 page metadata 写入
- `mdl/backend/mpr/page_model.go` — 补充 `toBSON()` 实现

**删除（阶段4完成后）：**
- `mdl/executor/cmd_pages_builder.go`（492 行）— `pageBuilder` struct 及其 helper，由 `pageASTToModel()` 取代
- `mdl/executor/cmd_pages_builder_input.go` — 输入控件 BSON 构建，逻辑迁入 `toBSON()`
- `mdl/executor/cmd_pages_builder_input_filters.go` — filter widget 构建，逻辑迁入 `toBSON()`

**验收**：roundtrip 稳定性断言（第二次 describe == 第一次）转绿；BSON golden 不变

---

### 阶段 5：更新黄金快照 + 清理

```bash
make update-helpdesk-golden
git add testdata/helpdesk-golden-*/describe-snapshot.mdl
```

删除已废弃的 rawWidget 类型定义残留。

**验收**：`make test` 全绿，PR 可合并

---

## 8. 文件变更汇总

| 类型 | 文件 | 变化 |
|------|------|------|
| 新建 | `mdl/types/page.go` | ~200 行类型定义 |
| 新建 | `mdl/backend/page_model.go` | ~30 行接口 |
| 新建 | `mdl/backend/mock/mock_page_model.go` | ~40 行 stub |
| 新建 | `mdl/backend/mpr/page_model.go` | ~400 行 BSON 转换 |
| 新建 | `mdl/executor/cmd_pages_ast_to_model.go` | ~500 行 AST→IR |
| 新建 | `mdl/executor/cmd_pages_model_to_mdl.go` | ~300 行 IR→MDL |
| 修改 | `mdl/backend/backend.go` | +1 行接口组合 |
| 修改 | `mdl/executor/cmd_pages_describe.go` | 读路径切换 |
| 修改 | `mdl/executor/cmd_pages_create_v3.go` | 写路径切换 |
| 修改 | `mdl/executor/cmd_pages_builder_v3.go` | widget 逻辑迁出 |
| 修改 | `internal/goldenfs/helpdesk_regression_test.go` | 补充断言 |
| 删除 | `mdl/executor/cmd_pages_describe_parse.go` | -903 行（阶段3） |
| 删除 | `mdl/executor/cmd_pages_describe_output.go` | -1354 行（阶段3） |
| 删除 | `mdl/executor/cmd_pages_describe_pluggable.go` | -1244 行（阶段3） |
| 删除 | `mdl/executor/cmd_pages_builder.go` | -492 行（阶段4） |
| 删除 | `mdl/executor/cmd_pages_builder_input.go` | -删除（阶段4） |
| 删除 | `mdl/executor/cmd_pages_builder_input_filters.go` | -删除（阶段4） |

**净效果**：删除约 4000+ 行旧代码，新增约 1470 行，**净减少 ~2500 行**。
