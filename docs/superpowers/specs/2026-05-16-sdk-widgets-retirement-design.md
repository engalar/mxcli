# sdk/widgets 退役设计文档

**日期**: 2026-05-16  
**分支**: feature/expression-checker  
**目标**: 将 `modelsdk/widgets` 升级为唯一的 widget 模板加载库，彻底删除 `sdk/widgets`

---

## 背景

`sdk/widgets` 和 `modelsdk/widgets` 目前是两个几乎相同的 fork：

- `sdk/widgets`：当前被 `widget_builder.go` 和 `datagrid_builder.go` 使用，功能更完整（mpk 多 widget 支持、更多 WidgetDefinition 字段）
- `modelsdk/widgets`：已存在但尚未被任何外部代码使用，是 sdk/widgets 的早期 fork，功能有所缺失
- `PropertyTypeIDEntry` 在三个位置各有一份定义：`sdk/widgets`、`modelsdk/widgets`、`mdl/types`

`mdl/backend/mpr/widget_builder.go` 中有 `convertPropertyTypeIDs()` bridge 函数（18 行）负责在 `sdk/widgets.PropertyTypeIDEntry` 和 `types.PropertyTypeIDEntry` 之间做字段复制。

---

## 目标状态

- `modelsdk/widgets` 是唯一的 widget 模板加载库
- `sdk/widgets` 整目录完全删除
- `PropertyTypeIDEntry` 只有 `mdl/types` 这一份定义（其余两处变为 type alias）
- `mdl/backend/mpr/widget_builder.go` 和 `datagrid_builder.go` 零 `sdk/` 依赖
- `convertPropertyTypeIDs()` bridge 函数删除

---

## 不在范围内

- `sdk/mpr` 内部对 `sdk/pages` 的依赖（独立 Stage）
- `sdk/domainmodel`、`sdk/security`、`sdk/workflows` 退役
- widget BSON 序列化逻辑本身不变

---

## 实现方案：渐进式合并（方法 B）

4 个独立步骤，每步都可单独编译+测试。合并为 2 个 PR。

### Step 1 — modelsdk/widgets/mpk 补全

**文件**: `modelsdk/widgets/mpk/mpk.go`

补齐 `sdk/widgets/mpk/mpk.go` 中多出的内容：

- `WidgetDefinition` 加 7 个字段：
  - `Description string`
  - `OfflineCapable bool`
  - `NeedsEntityContext bool`
  - `SupportedPlatform string`（默认 "Web"）
  - `HelpURL string`
  - `StudioCategory string`
  - `StudioProCategory string`
- `PropertyDef` 加 `AllowedTypes []string`
- XML 解析结构体 `xmlWidget` 补充对应属性（`offlineCapable`、`needsEntityContext`、`supportedPlatform`、`helpUrl`、`studioCategory`、`studioProCategory`）
- XML `xmlPropertyDef` 加 `AttributeTypes []xmlAttributeType`
- 加 `xmlAttributeType struct`
- 加 `buildDefinition()` 工厂函数（从 sdk/widgets/mpk 直接复制，改 package 路径）
- 加 `getWidgetIDsFromMPK()` — 支持多 widget MPK
- 加 `ParseMPKForWidget()` — 按 widgetID 精确解析
- 加 `ParseAll()` — 解析 MPK 中所有 widget

**验证**: `go test ./modelsdk/widgets/mpk/...`

---

### Step 2 — modelsdk/widgets loader/augment 补全

**文件**: `modelsdk/widgets/loader.go`、`modelsdk/widgets/augment.go`、新建 `modelsdk/widgets/generate.go`

- `loader.go`：
  - 加 `ResetGeneratedCache()` 函数（test helpers 用）
  - 加 `getOrGenerateTemplate()` — 从项目路径扫描 `.mpk` 文件（直接复制 sdk/widgets 实现，改 import path）
  - `jsonValueToBSONWithNestedObjectType` 加 variadic `dataSourceProperty ...*string` 参数，并同步更新调用方
- `augment.go`：同步对应 loader 调用变化
- `generate.go`：从 `sdk/widgets/generate.go` 复制，改 import path

**验证**: `go test ./modelsdk/widgets/...`  
**补充验证**: 在有真实 `.mpr` 项目的环境下运行含 DataGrid2 pluggable widget 的 `mxcli exec` 命令，确认模板加载正常

---

### Step 3 — PropertyTypeIDEntry alias 化

**文件**: `sdk/widgets/loader.go`、`modelsdk/widgets/loader.go`、`mdl/backend/mpr/widget_builder.go`

- `sdk/widgets/loader.go`：删除 `PropertyTypeIDEntry struct`，加：
  ```go
  type PropertyTypeIDEntry = types.PropertyTypeIDEntry
  ```
- `modelsdk/widgets/loader.go`：同上
- `widget_builder.go`：
  - 删除 `convertPropertyTypeIDs()` 函数（约 18 行）
  - `LoadWidgetTemplate` 中 `convertPropertyTypeIDs(embeddedIDs)` → 直接使用 `embeddedIDs`（类型已一致）
  - 同样处理 `datagrid_builder.go` 中的调用

**注意**: 两个包的 `NestedPropertyIDs map[string]PropertyTypeIDEntry` 递归引用在 alias 化后保持透明，无需额外处理。

**验证**: `go build ./...`（无编译错误，`convertPropertyTypeIDs` 消失）

---

### Step 4 — Import swap + 删除

**文件**: `widget_builder.go`、`datagrid_builder.go`、删除 `sdk/widgets/`

- `widget_builder.go` import：`sdk/widgets` → `modelsdk/widgets`（1 行）
- `datagrid_builder.go` import：`sdk/widgets` → `modelsdk/widgets`（1 行）
- `git rm -r sdk/widgets/`
- 验证无残留：`grep -r "sdk/widgets" . --include="*.go"` 应无结果

**验证**: `go build ./... && go test ./...`（全量通过）  
**回归保护**: 运行 `mdl-examples/widget-roundtrip/` 三脚本确认 Studio Pro 级别行为不变

---

## PR 结构

| PR | 包含步骤 | 变更范围 | 审查重点 |
|----|---------|---------|---------|
| PR 1 | Step 1 + Step 2 | 仅 `modelsdk/widgets/` | 新增函数与 sdk/widgets 行为一致 |
| PR 2 | Step 3 + Step 4 | alias 化 + swap + 删除 | `convertPropertyTypeIDs` 删除、全量编译无残留 |

---

## 测试验证表

| Step | 命令 | 预期结果 |
|------|------|---------|
| 1 | `go test ./modelsdk/widgets/mpk/...` | 全通 |
| 2 | `go test ./modelsdk/widgets/...` | 全通 |
| 3 | `go build ./...` | 无编译错误 |
| 4 | `go build ./... && go test ./...` | 全量通过；`grep -r sdk/widgets .` 无结果 |

---

## 完成后的收益

- `mdl/backend/mpr/` 实现 **零 `sdk/` 依赖**（本次 `sdk/widgets` + 之前完成的 `sdk/mpr` 写路径退役）
- `sdk/widgets` 整目录消失（约 3400 行代码）
- `PropertyTypeIDEntry` 三份定义合并为一份（`mdl/types`）
- `convertPropertyTypeIDs()` bridge 函数（18 行）删除
- fork 格局终结，widget 模板加载有唯一权威实现
