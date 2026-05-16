# sdk/widgets 退役实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 `modelsdk/widgets` 升级为唯一的 widget 模板加载库，删除 `sdk/widgets` 整目录，统一 `PropertyTypeIDEntry` 三份定义为 `mdl/types` 中的一份。

**Architecture:** 4 步渐进合并，分 2 个 PR。PR1 仅改动 `modelsdk/widgets/`（功能补全），PR2 完成 alias 化、import swap、删除旧包。每步可单独编译+测试，不存在中间态构建失败。

**Tech Stack:** Go 1.26, `go.mongodb.org/mongo-driver/bson`, `encoding/xml`, `sync.Map`

---

## 文件变动清单

| 操作 | 路径 |
|------|------|
| 修改 | `modelsdk/widgets/mpk/mpk.go` — 补齐 7 个 WidgetDefinition 字段 + 3 个函数 |
| 修改 | `modelsdk/widgets/loader.go` — 加 ResetGeneratedCache、getOrGenerateTemplate、variadic dataSourceProperty |
| 修改 | `modelsdk/widgets/augment.go` — 同步 loader 调用 |
| 新建 | `modelsdk/widgets/generate.go` — 从 sdk/widgets/generate.go 复制，改 import |
| 修改 | `sdk/widgets/loader.go` — PropertyTypeIDEntry → type alias |
| 修改 | `modelsdk/widgets/loader.go` — PropertyTypeIDEntry → type alias |
| 修改 | `mdl/backend/mpr/widget_builder.go` — 删除 convertPropertyTypeIDs，改 import |
| 修改 | `mdl/backend/mpr/datagrid_builder.go` — 改 import，直接用 embeddedIDs |
| 删除 | `sdk/widgets/`（整目录） |

---

## Task 1: modelsdk/widgets/mpk — 补全 WidgetDefinition 字段

**PR 1 的第一步。仅改 `modelsdk/widgets/mpk/mpk.go`，不影响任何调用方。**

**Files:**
- Modify: `modelsdk/widgets/mpk/mpk.go`

- [ ] **1.1 写失败测试**

在 `modelsdk/widgets/mpk/mpk_test.go` 追加（或在已有测试文件中添加）：

```go
func TestWidgetDefinitionHasFullFields(t *testing.T) {
    // Compile-time check: if fields are missing the struct literal will fail
    _ = WidgetDefinition{
        ID:                 "test",
        Name:               "Test",
        Description:        "desc",
        Version:            "1.0",
        IsPluggable:        true,
        OfflineCapable:     false,
        NeedsEntityContext: false,
        SupportedPlatform:  "Web",
        HelpURL:            "",
        StudioCategory:     "",
        StudioProCategory:  "",
        Properties:         nil,
        SystemProps:        nil,
    }
}

func TestPropertyDefHasAllowedTypes(t *testing.T) {
    _ = PropertyDef{
        Key:          "attr",
        Type:         "Attribute",
        AllowedTypes: []string{"String", "Integer"},
    }
}
```

- [ ] **1.2 运行确认失败**

```bash
go test ./modelsdk/widgets/mpk/... -run TestWidgetDefinitionHasFullFields -v
```

期望：`FAIL — cannot use struct literal`（字段缺失）

- [ ] **1.3 在 mpk.go 补全 WidgetDefinition 字段**

找到 `type WidgetDefinition struct` 并替换为：

```go
type WidgetDefinition struct {
	ID                 string        // e.g. "com.mendix.widget.web.combobox.Combobox"
	Name               string        // e.g. "Combo box"
	Description        string        // widget description from <description> element
	Version            string        // from package.xml clientModule version
	IsPluggable        bool          // true if pluginWidget="true" (React), false for legacy Dojo
	OfflineCapable     bool          // true if offlineCapable="true"
	NeedsEntityContext bool          // true if needsEntityContext="true"
	SupportedPlatform  string        // "Web", "Native", "All" (empty = Web)
	HelpURL            string        // helpUrl attribute
	StudioCategory     string        // studioCategory attribute
	StudioProCategory  string        // studioProCategory attribute
	Properties         []PropertyDef // regular <property> elements
	SystemProps        []PropertyDef // <systemProperty> elements
}
```

在 `PropertyDef` struct 中加字段 `AllowedTypes []string`：

```go
type PropertyDef struct {
	Key            string   // property key
	Type           string   // Mendix property type (Attribute, Boolean, etc.)
	DefaultValue   string   // default value
	Required       bool     // required flag
	IsList         bool     // whether this is a list property
	DataSource     string   // linked datasource property key
	Caption        string   // human-readable name
	Description    string   // property description
	AllowedTypes   []string // for attribute properties: Mendix type names ("String", "Decimal", etc.)
	NestedProps    []PropertyDef // nested properties (for Object type)
}
```

- [ ] **1.4 补全 XML 解析结构体**

在 `mpk.go` 中找到 `xmlWidget struct` 并补充缺失属性：

```go
type xmlWidget struct {
	ID                 string         `xml:"id,attr"`
	PluginWidget       string         `xml:"pluginWidget,attr"`
	OfflineCapable     string         `xml:"offlineCapable,attr"`
	NeedsEntityContext string         `xml:"needsEntityContext,attr"`
	SupportedPlatform  string         `xml:"supportedPlatform,attr"`
	HelpURL            string         `xml:"helpUrl,attr"`
	StudioCategory     string         `xml:"studioCategory,attr"`
	StudioProCategory  string         `xml:"studioProCategory,attr"`
	Name               string         `xml:"name"`
	Description        string         `xml:"description"`
	PropertyGroups     []xmlPropGroup `xml:"properties>propertyGroup"`
}
```

在 `xmlPropertyDef struct` 中加 `AttributeTypes`：

```go
type xmlAttributeType struct {
	Name string `xml:"name,attr"`
}

// xmlPropertyDef — add this field inside the existing struct:
//   AttributeTypes []xmlAttributeType `xml:"attributeTypes>attributeType"`
```

- [ ] **1.5 补全 buildDefinition + getWidgetIDsFromMPK + ParseMPKForWidget + ParseAll**

将 `sdk/widgets/mpk/mpk.go` 中以下函数直接复制到 `modelsdk/widgets/mpk/mpk.go`（package 路径不变，均为 `package mpk`）：

- `buildDefinition(widget *xmlWidget, version string) *WidgetDefinition`
- `getWidgetIDsFromMPK(mpkPath string) ([]string, error)` — 替换现有的 `getWidgetIDFromMPK`（注意：现有函数只返回第一个 widget）
- `ParseMPKForWidget(mpkPath string, widgetID string) (*WidgetDefinition, error)` — 新增
- `ParseAll(mpkPath string) ([]*WidgetDefinition, error)` — 新增

同时把 `getWidgetIDFromMPK` 的调用方（`getTemplateIndex` 函数内）改为使用 `getWidgetIDsFromMPK`，遍历所有返回的 ID：

```go
wids, err := getWidgetIDsFromMPK(mpkPath)
if err != nil {
    continue
}
for _, wid := range wids {
    if wid != "" {
        dirMap[wid] = mpkPath
    }
}
```

- [ ] **1.6 运行测试**

```bash
go test ./modelsdk/widgets/mpk/... -v
```

期望：所有测试通过，包括新的 `TestWidgetDefinitionHasFullFields`

- [ ] **1.7 确认 sdk/widgets/mpk 仍通过**

```bash
go test ./sdk/widgets/mpk/... -v
```

期望：全通（本步未改动 sdk）

- [ ] **1.8 提交**

```bash
git add modelsdk/widgets/mpk/mpk.go modelsdk/widgets/mpk/mpk_test.go
git commit -m "feat(modelsdk/widgets/mpk): bring to parity with sdk/widgets/mpk

Add missing WidgetDefinition fields (Description, OfflineCapable,
NeedsEntityContext, SupportedPlatform, HelpURL, StudioCategory,
StudioProCategory), AllowedTypes to PropertyDef, multi-widget MPK
support (getWidgetIDsFromMPK, ParseMPKForWidget, ParseAll)."
```

---

## Task 2: modelsdk/widgets — 补全 loader / augment / generate

**PR 1 的第二步。仍只改 `modelsdk/widgets/`，不影响任何调用方。**

**Files:**
- Modify: `modelsdk/widgets/loader.go`
- Modify: `modelsdk/widgets/augment.go`
- Create: `modelsdk/widgets/generate.go`

- [ ] **2.1 写失败测试**

在 `modelsdk/widgets/` 中找现有测试文件，追加：

```go
func TestResetGeneratedCacheExists(t *testing.T) {
    // Should compile — confirms ResetGeneratedCache is present
    ResetGeneratedCache()
}

func TestGetTemplateFullBSONAcceptsDataSourceProperty(t *testing.T) {
    // GetTemplateFullBSON should use getOrGenerateTemplate internally
    // (compile-time: no test for runtime without real MPK)
    _, _, _, _, err := GetTemplateFullBSON("nonexistent", func() string { return "" }, "")
    // nil result is OK for unknown widget; error should be nil
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
}
```

- [ ] **2.2 运行确认失败**

```bash
go test ./modelsdk/widgets/... -run TestResetGeneratedCacheExists -v
```

期望：`FAIL — undefined: ResetGeneratedCache`

- [ ] **2.3 在 loader.go 加 generatedCache + ResetGeneratedCache + getOrGenerateTemplate**

在 `modelsdk/widgets/loader.go` 中，在现有 `templateCache` 声明附近加：

```go
// generatedCache stores MPK-derived templates for the session lifetime.
// Key: widgetID string. Value: *WidgetTemplate.
var generatedCache sync.Map

// ResetGeneratedCache clears the MPK-derived template cache (for testing).
func ResetGeneratedCache() {
	generatedCache.Range(func(k, _ any) bool {
		generatedCache.Delete(k)
		return true
	})
}

// getOrGenerateTemplate returns a WidgetTemplate for widgetID. It checks the embedded
// template cache first, then falls back to deriving a template from the project's .mpk
// widget file. Returns nil, nil when the widget is unknown and no MPK is available.
func getOrGenerateTemplate(widgetID, projectPath string) (*WidgetTemplate, error) {
	// 1. Embedded templates
	if tmpl, err := GetTemplate(widgetID); err != nil || tmpl != nil {
		return tmpl, err
	}
	// 2. Session cache
	if cached, ok := generatedCache.Load(widgetID); ok {
		return cached.(*WidgetTemplate), nil
	}
	// 3. Derive from MPK in project/widgets/
	if projectPath == "" {
		return nil, nil
	}
	projectDir := filepath.Dir(projectPath)
	mpkPath, err := mpk.FindMPK(projectDir, widgetID)
	if err != nil {
		return nil, fmt.Errorf("widget %q: scan MPK directory: %w", widgetID, err)
	}
	if mpkPath == "" {
		return nil, nil
	}
	def, err := mpk.ParseMPKForWidget(mpkPath, widgetID)
	if err != nil {
		return nil, fmt.Errorf("widget %q: parse MPK: %w", widgetID, err)
	}
	if def == nil {
		return nil, nil
	}
	tmpl := GenerateFromMPK(def)
	generatedCache.Store(widgetID, tmpl)
	return tmpl, nil
}
```

确保 `GetTemplateBSON` 和 `GetTemplateFullBSON` 内部将 `GetTemplate(widgetID)` 改为调用 `getOrGenerateTemplate(widgetID, projectPath)`（使 projectPath 参数生效）：

```go
// GetTemplateBSON 修改：
func GetTemplateBSON(widgetID string, idGenerator func() string, projectPath string) (bson.D, map[string]PropertyTypeIDEntry, error) {
	tmpl, err := getOrGenerateTemplate(widgetID, projectPath)  // 改这行
	// ... rest unchanged
}

// GetTemplateFullBSON 修改：
func GetTemplateFullBSON(widgetID string, idGenerator func() string, projectPath string) (bson.D, bson.D, map[string]PropertyTypeIDEntry, string, error) {
	tmpl, err := getOrGenerateTemplate(widgetID, projectPath)  // 改这行
	// ... rest unchanged
}
```

确保 import 块包含 `"sync"` 和 `"path/filepath"`。

- [ ] **2.4 在 loader.go 加 variadic dataSourceProperty 参数**

将 `jsonValueToBSONWithNestedObjectType` 的签名从：

```go
func jsonValueToBSONWithNestedObjectType(val any, idMapping map[string]string, valueTypeID *string, nestedObjectTypeID *string, nestedPropertyIDs map[string]PropertyTypeIDEntry, defaultValue *string, valueType *string, required *bool) any {
```

改为：

```go
func jsonValueToBSONWithNestedObjectType(val any, idMapping map[string]string, valueTypeID *string, nestedObjectTypeID *string, nestedPropertyIDs map[string]PropertyTypeIDEntry, defaultValue *string, valueType *string, required *bool, dataSourceProperty ...*string) any {
```

在函数体内，`} else if key == "Required"` 的 else 分支之后，追加：

```go
} else if key == "DataSourceProperty" && len(dataSourceProperty) > 0 && dataSourceProperty[0] != nil {
    if dsp, ok := fieldVal.(string); ok {
        *dataSourceProperty[0] = dsp
    }
    elem.Value = jsonValueToBSONSimple(fieldVal, idMapping)
```

在 `jsonToBSONWithMappingAndObjectType` 中，找到调用 `jsonValueToBSONWithNestedObjectType` 的行，在末尾加 `&dataSourceProp`：

```go
elem.Value = jsonValueToBSONWithNestedObjectType(val, idMapping, &valueTypeID, &nestedObjectTypeID, nestedPropertyIDs, &defaultValue, &valueType, &required, &dataSourceProp)
```

同时确保 `dataSourceProp` 变量在调用前已声明：

```go
var dataSourceProp string
```

并将结果存入 `entry.DataSourceProperty = dataSourceProp`。

- [ ] **2.5 同步 augment.go**

比对 `sdk/widgets/augment.go` 和 `modelsdk/widgets/augment.go`：

```bash
diff sdk/widgets/augment.go modelsdk/widgets/augment.go
```

将 sdk 版本中存在但 modelsdk 版本中缺少的逻辑同步过来（只改 `modelsdk/widgets/mpk` → `modelsdk/widgets/mpk`，import path 已经正确）。

- [ ] **2.6 新建 modelsdk/widgets/generate.go**

创建文件，内容为 `sdk/widgets/generate.go` 的完整复制，唯一修改是 import 路径：

```go
// SPDX-License-Identifier: Apache-2.0

package widgets

import "github.com/mendixlabs/mxcli/modelsdk/widgets/mpk"  // 原为 sdk/widgets/mpk

// GenerateFromMPK builds a complete WidgetTemplate from a parsed MPK WidgetDefinition.
// ... (以下内容与 sdk/widgets/generate.go 完全相同)
```

- [ ] **2.7 在 WidgetTemplate struct 加 Generated 字段**

在 `modelsdk/widgets/loader.go` 的 `WidgetTemplate struct` 中加：

```go
Generated bool `json:"-"` // true if derived from MPK, not from embedded template
```

- [ ] **2.8 运行测试**

```bash
go test ./modelsdk/widgets/... -v
```

期望：全通，包括 `TestResetGeneratedCacheExists`

- [ ] **2.9 全量编译确认无破坏**

```bash
go build ./...
```

期望：无错误

- [ ] **2.10 提交**

```bash
git add modelsdk/widgets/loader.go modelsdk/widgets/augment.go modelsdk/widgets/generate.go
git commit -m "feat(modelsdk/widgets): bring loader/augment/generate to parity with sdk/widgets

Add ResetGeneratedCache, getOrGenerateTemplate (MPK fallback path),
variadic dataSourceProperty param, generate.go (GenerateFromMPK).
GetTemplateBSON/FullBSON now use getOrGenerateTemplate internally."
```

---

## Task 3: PropertyTypeIDEntry alias 化 + convertPropertyTypeIDs 删除

**PR 2 的第一步。统一三份类型定义，删除 bridge 函数。**

**Files:**
- Modify: `sdk/widgets/loader.go` (line ~759)
- Modify: `modelsdk/widgets/loader.go` (line ~709)
- Modify: `mdl/backend/mpr/widget_builder.go` (lines ~723-741, ~45-53)
- Modify: `mdl/backend/mpr/datagrid_builder.go` (line ~47)

- [ ] **3.1 确认 types.PropertyTypeIDEntry 字段与两个包中的 struct 字段完全相同**

```bash
grep -A 12 "type PropertyTypeIDEntry struct" \
  sdk/widgets/loader.go \
  modelsdk/widgets/loader.go \
  mdl/types/widget_property_type.go
```

期望：三者字段名和类型完全一致（PropertyTypeID、ValueTypeID、DefaultValue、ValueType、Required、DataSourceProperty、ObjectTypeID、NestedPropertyIDs）。

- [ ] **3.2 替换 sdk/widgets/loader.go 中的 struct 为 alias**

找到（约第 759 行）：

```go
// PropertyTypeIDEntry holds the IDs for a property type.
type PropertyTypeIDEntry struct {
	PropertyTypeID     string
	ValueTypeID        string
	DefaultValue       string
	ValueType          string
	Required           bool
	DataSourceProperty string
	ObjectTypeID       string
	NestedPropertyIDs  map[string]PropertyTypeIDEntry
}
```

替换为：

```go
// PropertyTypeIDEntry is now defined in mdl/types; re-exported here for compatibility.
type PropertyTypeIDEntry = types.PropertyTypeIDEntry
```

在文件 import 块中加 `"github.com/mendixlabs/mxcli/mdl/types"`。

- [ ] **3.3 替换 modelsdk/widgets/loader.go 中的 struct 为 alias**

同上操作（约第 709 行），替换为相同 alias 定义并加 import。

- [ ] **3.4 编译确认两个 widgets 包无错**

```bash
go build ./sdk/widgets/... ./modelsdk/widgets/...
```

期望：无错误

- [ ] **3.5 删除 widget_builder.go 中的 convertPropertyTypeIDs 函数**

找到并删除（约第 722-741 行）：

```go
func convertPropertyTypeIDs(src map[string]widgets.PropertyTypeIDEntry) map[string]types.PropertyTypeIDEntry {
	result := make(map[string]types.PropertyTypeIDEntry)
	for k, v := range src {
		entry := types.PropertyTypeIDEntry{
			PropertyTypeID:     v.PropertyTypeID,
			ValueTypeID:        v.ValueTypeID,
			DefaultValue:       v.DefaultValue,
			ValueType:          v.ValueType,
			Required:           v.Required,
			DataSourceProperty: v.DataSourceProperty,
			ObjectTypeID:       v.ObjectTypeID,
		}
		if len(v.NestedPropertyIDs) > 0 {
			entry.NestedPropertyIDs = convertPropertyTypeIDs(v.NestedPropertyIDs)
		}
		result[k] = entry
	}
	return result
}
```

- [ ] **3.6 修改 widget_builder.go 中的 LoadWidgetTemplate 调用**

找到（约第 53 行）：

```go
propertyTypeIDs := convertPropertyTypeIDs(embeddedIDs)
```

改为：

```go
propertyTypeIDs := embeddedIDs
```

（`embeddedIDs` 现在已经是 `map[string]types.PropertyTypeIDEntry`，无需转换）

- [ ] **3.7 修改 datagrid_builder.go 中的 convertPropertyTypeIDs 调用**

找到（约第 47 行）：

```go
propertyTypeIDs := convertPropertyTypeIDs(embeddedIDs)
```

改为：

```go
propertyTypeIDs := embeddedIDs
```

- [ ] **3.8 全量编译**

```bash
go build ./...
```

期望：无错误。注意：此时 import `"github.com/mendixlabs/mxcli/sdk/widgets"` 在 widget_builder.go 和 datagrid_builder.go 中仍然存在（Task 4 才删）。

- [ ] **3.9 运行测试**

```bash
go test ./sdk/widgets/... ./modelsdk/widgets/... ./mdl/backend/mpr/... -v 2>&1 | tail -20
```

期望：全通

- [ ] **3.10 提交**

```bash
git add sdk/widgets/loader.go modelsdk/widgets/loader.go \
        mdl/backend/mpr/widget_builder.go mdl/backend/mpr/datagrid_builder.go
git commit -m "refactor(widgets): unify PropertyTypeIDEntry via type alias to mdl/types

sdk/widgets and modelsdk/widgets now re-export types.PropertyTypeIDEntry.
Remove convertPropertyTypeIDs bridge (18 lines) from widget_builder.go;
both buildDataGrid2WidgetDoc and LoadWidgetTemplate use embeddedIDs directly."
```

---

## Task 4: Import swap + sdk/widgets 删除

**PR 2 的第二步。最终退役 sdk/widgets。**

**Files:**
- Modify: `mdl/backend/mpr/widget_builder.go` (import block)
- Modify: `mdl/backend/mpr/datagrid_builder.go` (import block)
- Delete: `sdk/widgets/` (整目录)

- [ ] **4.1 修改 widget_builder.go 的 import**

找到：

```go
"github.com/mendixlabs/mxcli/sdk/widgets"
```

替换为：

```go
"github.com/mendixlabs/mxcli/modelsdk/widgets"
```

- [ ] **4.2 修改 datagrid_builder.go 的 import**

同上，将 `sdk/widgets` → `modelsdk/widgets`。

- [ ] **4.3 编译确认 import 替换正确**

```bash
go build ./mdl/backend/mpr/...
```

期望：无错误

- [ ] **4.4 删除 sdk/widgets 整目录**

```bash
git rm -r sdk/widgets/
```

- [ ] **4.5 全量编译**

```bash
go build ./...
```

期望：无错误，`sdk/widgets` 相关 import 已全部消失

- [ ] **4.6 验证无残留引用**

```bash
grep -r "sdk/widgets" . --include="*.go"
```

期望：**空输出**（零结果）

- [ ] **4.7 全量测试**

```bash
go test ./... 2>&1 | grep -E "FAIL|ok" | tail -20
```

期望：全部 `ok`，无 `FAIL`

- [ ] **4.8 widget-roundtrip 回归（如有真实项目路径）**

```bash
# 在有 .mpr 项目的环境中运行：
mxcli exec mdl-examples/widget-roundtrip/*.mdl -p /path/to/app.mpr
```

期望：执行成功，DataGrid2 等 pluggable widget 正常写入

- [ ] **4.9 提交**

```bash
git commit -m "feat(mpr): retire sdk/widgets — switch to modelsdk/widgets

widget_builder.go and datagrid_builder.go now import modelsdk/widgets.
Delete sdk/widgets/ (3400 lines). mdl/backend/mpr/ is now sdk/-free."
```

---

## 验收清单

完成 Task 4 后验证：

```bash
# 1. 零残留
grep -r "sdk/widgets" . --include="*.go"
# 期望：空

# 2. 全量编译
go build ./...
# 期望：无错误

# 3. 全量测试
go test ./...
# 期望：全通

# 4. PropertyTypeIDEntry 只剩一份定义
grep -rn "type PropertyTypeIDEntry struct" . --include="*.go"
# 期望：只有 mdl/types/widget_property_type.go 一处
```
