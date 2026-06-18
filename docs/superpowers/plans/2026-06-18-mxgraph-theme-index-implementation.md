# mxgraph Theme/Design/Style Index — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 扩展 mxgraph 图的索引范围，使其包含主题变量（SCSS）、设计属性定义（design-properties.json）和 widget 实例的 Class/Style/DesignProperties（MPR Appearance）。新增 `theme` 领域适配器，支持完整构建和增量 Watch() 事件回放。

**Architecture:** 沿用现有 mxgraph 适配器模式（`IndexAdapter` → `EventSink` → `Graph` + `Persistence`）。新增 2 个独立适配器 + 1 个 PageAdapter 扩展 + 类型化 `graphcatalog` 读取接口。SCSS 用行级解析器（保留注释、imports、格式）。Widget Tree 遍历复用现有 `pages_describe_parse.go` 中的 `extractDesignProperties` 逻辑。

**Tech Stack:** Go 1.21+、`internal/mxgraph`（已有）、`fsnotify`（已有）、`encoding/gob`（已有）、`modelsdk`（已有）

**Spec:** `docs/superpowers/specs/2026-06-15-mxgraph-index-framework-design.md`（持久化增量设计）、`docs/superpowers/plans/2026-06-15-graphcatalog.md`（参考 Task 格式）

---

## 文件变更地图

### Phase 0 — SCSS 行级解析器
| 操作 | 文件 |
|------|------|
| Create | `internal/mxgraph/scss/scss.go` |
| Create | `internal/mxgraph/scss/scss_test.go` |

### Phase 1 — ThemeScssAdapter（SCSS 变量 → 图节点）
| 操作 | 文件 |
|------|------|
| Create | `internal/mxgraph/adapter/themescss/adapter.go` |
| Create | `internal/mxgraph/adapter/themescss/adapter_test.go` |
| Create | `internal/mxgraph/adapter/themescss/nodes.go` |
| Modify | `mdl/executor/cmd_graph.go`（注册适配器） |

### Phase 2 — DesignPropertyAdapter（design-properties.json → 图节点）
| 操作 | 文件 |
|------|------|
| Create | `internal/mxgraph/adapter/designdprops/adapter.go` |
| Create | `internal/mxgraph/adapter/designdprops/adapter_test.go` |
| Create | `internal/mxgraph/adapter/designdprops/nodes.go` |
| Modify | `mdl/executor/cmd_graph.go`（注册适配器） |

### Phase 3 — WidgetInstance 扩展（PageAdapter 扩展）
| 操作 | 文件 |
|------|------|
| Create | `internal/mxgraph/adapter/mpr/widget_instance.go` |
| Create | `internal/mxgraph/adapter/mpr/widget_instance_test.go` |
| Modify | `internal/mxgraph/adapter/mpr/page.go`（Schema + 辅助函数） |

### Phase 4 — graphcatalog 读取接口
| 操作 | 文件 |
|------|------|
| Create | `mdl/graphcatalog/theme_nodes.go` |
| Modify | `mdl/graphcatalog/reader.go`（追加 ThemeReader / StylingReader） |
| Modify | `mdl/graphcatalog/graph.go`（追加实现） |
| Modify | `mdl/graphcatalog/graph_test.go`（追加主题相关测试） |
| Modify | `mdl/graphcatalog/mock/mock.go`（追加 mock 方法） |

### Phase 5 — Persistence 增量接入（Delta 回放）
| 操作 | 文件 |
|------|------|
| Modify | `internal/mxgraph/persist.go`（添加 Persistence struct + Init/Load/Save） |
| Modify | `internal/mxgraph/adapter.go`（添加 SetPersistence + dirty 字段） |
| Modify | `internal/mxgraph/engine.go`（添加 DirtyTracker） |
| Create | `internal/mxgraph/persist_test.go` |

### Phase 6 — Watch() 增量事件（文件监听）
| 操作 | 文件 |
|------|------|
| Modify | `internal/mxgraph/adapter/themescss/adapter.go`（ThemeScssAdapter.Watch） |
| Modify | `internal/mxgraph/adapter/designdprops/adapter.go`（DesignPropertyAdapter.Watch） |
| Modify | `internal/mxgraph/adapter/mpr/widget_instance.go`（WidgetInstanceAdapter.Watch） |
| Create | `internal/mxgraph/adapter/themescss/watch.go` |
| Create | `internal/mxgraph/adapter/designdprops/watch.go` |
| Modify | `mdl/executor/cmd_graph.go`（启动所有 Watch） |

### Phase 7 — cmd_theme.go 对接图查询
| 操作 | 文件 |
|------|------|
| Modify | `mdl/executor/cmd_theme.go`（使用 graphcatalog.ThemeReader 查询而非直接读文件） |
| Modify | `mdl/executor/register_stubs.go`（注册 SHOW THEME 处理器） |
| Modify | `mdl/executor/stmt_summary.go`（追加 summary） |

---

## Task 1：SCSS 行级解析器

> **推理：** 主题变量的核心——SCSS 文件用行级而非完整 AST 解析。保留所有非变量行（注释、import、原始 CSS）。这是 ThemeScssAdapter 的基石。

**Files:**
- Create: `internal/mxgraph/scss/scss.go`
- Create: `internal/mxgraph/scss/scss_test.go`

- [ ] **Step 1: 写失败测试——解析 SASS 变量和 CSS 自定义属性**

```go
// internal/mxgraph/scss/scss_test.go
package scss

import (
	"testing"
)

func TestParse_BasicSassVar(t *testing.T) {
	input := "$brand-primary: #264ae5;"
	doc, err := Parse("test.scss", input)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Vars) != 1 {
		t.Fatalf("Vars = %d, want 1", len(doc.Vars))
	}
	v := doc.Vars[0]
	if v.Name != "$brand-primary" {
		t.Errorf("Name = %q, want $brand-primary", v.Name)
	}
	if v.Value != "#264ae5" {
		t.Errorf("Value = %q, want #264ae5", v.Value)
	}
	if v.IsCSSVar {
		t.Error("IsCSSVar should be false for SASS var")
	}
}

func TestParse_CssCustomProperty(t *testing.T) {
	input := ":root { --brand-primary: #1565C0; }"
	doc, err := Parse("test.scss", input)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Vars) != 1 {
		t.Fatalf("Vars = %d, want 1", len(doc.Vars))
	}
	v := doc.Vars[0]
	if v.Name != "--brand-primary" {
		t.Errorf("Name = %q, want --brand-primary", v.Name)
	}
	if v.Value != "#1565C0" {
		t.Errorf("Value = %q, want #1565C0", v.Value)
	}
	if !v.IsCSSVar {
		t.Error("IsCSSVar should be true")
	}
	if !v.IsInRoot {
		t.Error("IsInRoot should be true")
	}
}

func TestParse_DefaultFlag(t *testing.T) {
	input := "$spacing-small: 8px !default;"
	doc, _ := Parse("test.scss", input)
	if len(doc.Vars) != 1 {
		t.Fatalf("Vars = %d, want 1", len(doc.Vars))
	}
	if !doc.Vars[0].IsDefault {
		t.Error("IsDefault should be true")
	}
}

func TestParse_CommentedVar(t *testing.T) {
	input := "// $brand-primary: #264ae5;"
	doc, _ := Parse("test.scss", input)
	if len(doc.Vars) != 1 {
		t.Fatalf("Vars = %d, want 1", len(doc.Vars))
	}
	if doc.Vars[0].IsActive {
		t.Error("commented var should be IsActive=false")
	}
}

func TestParse_MixedContent(t *testing.T) {
	input := `// Theme options
$brand-logo: false;

:root {
  --brand-primary: #264ae5;
  // --brand-success: #16aa16;
}

@import "variables";`
	doc, _ := Parse("test.scss", input)
	if len(doc.Vars) != 3 {
		t.Fatalf("Vars = %d, want 3 ($brand-logo, --brand-primary, --brand-success)", len(doc.Vars))
	}
	if len(doc.Lines) != 9 {
		t.Errorf("Lines = %d, want 9 (preserve all lines)", len(doc.Lines))
	}
}

func TestParse_CommentPreserved(t *testing.T) {
	input := "$brand-primary: #264ae5; // primary brand color"
	doc, _ := Parse("test.scss", input)
	if doc.Vars[0].Comment != "primary brand color" {
		t.Errorf("Comment = %q, want 'primary brand color'", doc.Vars[0].Comment)
	}
}

func TestSetVar_NewVar(t *testing.T) {
	doc, _ := Parse("test.scss", "@import \"variables\";")
	err := doc.SetVar("--brand-primary", "#1565C0", ScssVarOpts{InsertBefore: "@import"})
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Vars) != 1 {
		t.Fatalf("Vars = %d, want 1", len(doc.Vars))
	}
	if doc.Vars[0].Value != "#1565C0" {
		t.Errorf("Value = %q, want #1565C0", doc.Vars[0].Value)
	}
	// Verify line order: variable before import
	if doc.Lines[0].Raw != "  --brand-primary: #1565C0;" {
		t.Errorf("Line 0 = %q, want indented var", doc.Lines[0].Raw)
	}
}

func TestSetVar_UpdateExisting(t *testing.T) {
	doc, _ := Parse("test.scss", "$brand-primary: #264ae5;")
	err := doc.SetVar("$brand-primary", "#1565C0", ScssVarOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if doc.Vars[0].Value != "#1565C0" {
		t.Errorf("Value = %q, want #1565C0", doc.Vars[0].Value)
	}
	v := doc.findVar("$brand-primary")
	if v == nil {
		t.Fatal("findVar returned nil")
	}
	if v.Value != "#1565C0" {
		t.Errorf("findVar Value = %q", v.Value)
	}
}
```

- [ ] **Step 2: 运行测试，确认全面失败**

```bash
go test ./internal/mxgraph/scss/... -v -count=1 2>&1 || true
# 预期：package not found
```

- [ ] **Step 3: 实现 SCSS 行级解析器**

```go
// internal/mxgraph/scss/scss.go
package scss

import (
	"fmt"
	"regexp"
	"strings"
)

// ScssVarOpts 是 SetVar 的操作选项。
type ScssVarOpts struct {
	InsertBefore string // import 语句关键字（如 "@import"），在该行前插入
	Force        bool   // 跳过变量名验证
	Comment      string // 行尾注释（自动加 //）
}

type ScssVarDecl struct {
	Name      string // "$brand-primary" 或 "--brand-primary"
	Value     string // "#264ae5"
	IsDefault bool
	IsCSSVar  bool
	IsActive  bool   // false = 被注释掉了
	IsInRoot  bool   // 在 :root {} 块内
	Comment   string // 行尾注释内容（去除了 //）
	LineIdx   int    // 在 Lines 中的索引
}

type ScssLine struct {
	Raw string
}

// ScssDocument 是 SCSS 文件的内存模型，保留所有行的原始文本。
type ScssDocument struct {
	FilePath string
	Lines    []ScssLine
	Vars     []ScssVarDecl
}

// 匹配 SASS 变量: $name: value [!default] [;//comment]
var sassVarRe = regexp.MustCompile(`^\s*(\$[\w-]+)\s*:\s*(.+?)\s*(!default)?\s*;?\s*(//\s*(.*))?\s*$`)

// 匹配 CSS 自定义属性: --name: value [;//comment]
var cssVarRe = regexp.MustCompile(`^\s*(--[\w-]+)\s*:\s*(.+?)\s*;?\s*(//\s*(.*))?\s*$`)

// 注释行前缀
var commentPrefixRe = regexp.MustCompile(`^\s*//`)

// 检测 :root 块的进入/退出
var rootOpenRe = regexp.MustCompile(`^\s*:root\s*\{`)

// Parse 解析 SCSS 文本内容，返回文档模型。
func Parse(filePath, content string) (*ScssDocument, error) {
	doc := &ScssDocument{
		FilePath: filePath,
	}
	lines := strings.Split(content, "\n")
	inRoot := false

	for i, line := range lines {
		doc.Lines = append(doc.Lines, ScssLine{Raw: line})

		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Track :root block
		if rootOpenRe.MatchString(trimmed) {
			inRoot = true
			continue
		}
		if trimmed == "}" && inRoot {
			inRoot = false
			continue
		}

		// Try to parse as variable (stripping leading comment markers if present)
		isCommented := commentPrefixRe.MatchString(trimmed)

		var decl *ScssVarDecl

		// Try SASS: even inside comments, catch $var patterns
		if m := sassVarRe.FindStringSubmatch(trimmed); m != nil {
			// If commented, the comment prefix is part of the raw line;
			// the regex still matches the $var inside. Mark IsActive=false.
			commentText := ""
			if m[5] != "" {
				commentText = strings.TrimSpace(m[5])
			}
			decl = &ScssVarDecl{
				Name:      m[1],
				Value:     strings.TrimSpace(m[2]),
				IsDefault: strings.TrimSpace(m[3]) == "!default",
				IsCSSVar:  false,
				IsActive:  !isCommented && !strings.HasPrefix(strings.TrimLeft(line, " \t"), "//"),
				IsInRoot:  inRoot,
				Comment:   commentText,
				LineIdx:   i,
			}
		}

		// Try CSS custom property
		if m := cssVarRe.FindStringSubmatch(trimmed); m != nil && decl == nil {
			commentText := ""
			if m[4] != "" {
				commentText = strings.TrimSpace(m[4])
			}
			decl = &ScssVarDecl{
				Name:     m[1],
				Value:    strings.TrimSpace(m[2]),
				IsCSSVar: true,
				IsActive: !isCommented && !strings.HasPrefix(strings.TrimLeft(line, " \t"), "//"),
				IsInRoot: inRoot,
				Comment:  commentText,
				LineIdx:  i,
			}
		}

		if decl != nil {
			doc.Vars = append(doc.Vars, *decl)
		}
	}

	return doc, nil
}

// findVar 返回第一个匹配 name 的变量声明。
func (d *ScssDocument) findVar(name string) *ScssVarDecl {
	for i := range d.Vars {
		if d.Vars[i].Name == name {
			return &d.Vars[i]
		}
	}
	return nil
}

// SetVar 设置（新增或更新）一个变量的值。
func (d *ScssDocument) SetVar(name, value string, opts ScssVarOpts) error {
	// 尝试更新已有变量
	for i, v := range d.Vars {
		if v.Name == name {
			d.Vars[i].Value = value
			d.Vars[i].IsActive = true
			// 更新原始行
			oldLine := d.Lines[v.LineIdx].Raw
			newLine := replaceVarValue(oldLine, v.Name, value)
			d.Lines[v.LineIdx].Raw = newLine
			return nil
		}
	}

	// 新增变量
	var indent string
	if strings.HasPrefix(name, "--") {
		indent = "  " // CSS vars inside :root use 2-space indent
	}

	line := fmt.Sprintf("%s%s: %s;", indent, name, value)

	// 在指定行前插入
	insertAt := len(d.Lines) // default: append
	for i, l := range d.Lines {
		if opts.InsertBefore != "" && strings.Contains(l.Raw, opts.InsertBefore) {
			insertAt = i
			break
		}
	}

	// Insert into Lines
	d.Lines = append(d.Lines[:insertAt], append([]ScssLine{{Raw: line}}, d.Lines[insertAt:]...)...)

	// Track var
	decl := ScssVarDecl{
		Name:     name,
		Value:    value,
		IsCSSVar: strings.HasPrefix(name, "--"),
		IsActive: true,
		IsInRoot: strings.HasPrefix(name, "--"),
		LineIdx:  insertAt,
	}
	d.Vars = append(d.Vars, decl)

	// Adjust line indices for existing vars after insert point
	for i := range d.Vars {
		if d.Vars[i].LineIdx >= insertAt && d.Vars[i].Name != name {
			d.Vars[i].LineIdx++
		}
	}

	return nil
}

// Write 将文档序列化回文本。
func (d *ScssDocument) Write() string {
	lines := make([]string, len(d.Lines))
	for i, l := range d.Lines {
		lines[i] = l.Raw
	}
	return strings.Join(lines, "\n")
}

// replaceVarValue 替换 SCSS 行中的变量值部分。
// 例如: "$brand-primary: #264ae5;" → "$brand-primary: #1565C0;"
func replaceVarValue(line, name, newValue string) string {
	// Remove leading whitespace capture
	trimmed := strings.TrimLeft(line, " \t")
	prefix := line[:len(line)-len(trimmed)]

	re := regexp.MustCompile(`(\s*:\s*).+?\s*(!default)?\s*;`)
	result := re.ReplaceAllString(trimmed, fmt.Sprintf("${1}%s;", newValue))
	return prefix + result
}
```

- [ ] **Step 4: 运行测试**

```bash
go test ./internal/mxgraph/scss/... -v -count=1
```

预期：全部 PASS。包括 SetVar 的插入顺序和更新。

- [ ] **Step 5: Commit**

```bash
git add internal/mxgraph/scss/
git commit -m "feat(mxgraph/scss): add line-level SCSS parser for theme variable extraction"
```

---

## Task 2：ThemeScssAdapter — SCSS 变量 → 图节点

> **推理：** 参照现有 DomainModelAdapter 模式，读取所有主题 SCSS 文件，对每个变量发出 ThemeVariable 节点。Build() 全量扫描文件，Watch() 监听文件变化。

**Files:**
- Create: `internal/mxgraph/adapter/themescss/nodes.go`
- Create: `internal/mxgraph/adapter/themescss/adapter.go`
- Create: `internal/mxgraph/adapter/themescss/adapter_test.go`

- [ ] **Step 1: 写失败测试**

```go
// internal/mxgraph/adapter/themescss/adapter_test.go
package themescss

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

type recordingSink struct {
	events []mxgraph.Event
}

func (s *recordingSink) Emit(events []mxgraph.Event) error {
	s.events = append(s.events, events...)
	return nil
}

func TestThemeScssAdapter_Name(t *testing.T) {
	a := &ThemeScssAdapter{}
	if a.Name() != "themescss" {
		t.Errorf("Name() = %q, want themescss", a.Name())
	}
}

func TestThemeScssAdapter_Schema(t *testing.T) {
	a := &ThemeScssAdapter{}
	s := a.Schema()
	if s == nil {
		t.Fatal("Schema() returned nil")
	}
	found := false
	for _, l := range s.NodeLabels {
		if l == "ThemeVariable" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Schema missing label ThemeVariable")
	}
}

func TestThemeScssAdapter_Build(t *testing.T) {
	// 创建临时测试目录结构
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "theme", "web"), 0700)
	os.MkdirAll(filepath.Join(dir, "themesource", "demo", "web"), 0700)

	// 主题 SCSS 文件
	os.WriteFile(filepath.Join(dir, "theme", "web", "custom-variables.scss"), []byte(`
$brand-logo: false;
$use-css-variables: true;
:root {
  --brand-primary: #264ae5;
  --brand-success: #16aa16;
  // --brand-warning: #cd8501;
}
@import "../../themesource/atlas_core/web/variables";
`), 0600)

	os.WriteFile(filepath.Join(dir, "theme", "web", "main.scss"), []byte(`
$brand-primary: #1565C0;
$brand-primary-dark: #0D47A1;
`), 0600)

	os.WriteFile(filepath.Join(dir, "themesource", "demo", "web", "variables.scss"), []byte(`
$brand-primary: #264ae5 !default;
$bg-color: #fff !default;
`), 0600)

	a := &ThemeScssAdapter{ProjectDir: dir}
	sink := &recordingSink{}
	err := a.Build(context.Background(), sink)
	if err != nil {
		t.Fatal(err)
	}

	if len(sink.events) == 0 {
		t.Fatal("expected at least 1 event")
	}

	// Verify we have events for all 8 variables (6 active + 1 commented + 1 default-only in module)
	// Actually: custom-variables has 4 lines (1 SASS active + 2 CSS active + 1 CSS commented)
	// main.scss has 2 SASS active
	// demo/variables.scss has 2 SASS default
	// Total active: 1+2+2+2 = 7, commented: 1
	var nodeCreated int
	for _, e := range sink.events {
		if e.Type == mxgraph.NodeCreated && e.Node != nil {
			nodeCreated++
		}
	}
	if nodeCreated < 5 {
		t.Fatalf("NodeCreated events = %d, want >= 5", nodeCreated)
	}

	// Check a specific variable's props
	checkVar := func(name, source, value string) bool {
		for _, e := range sink.events {
			if e.Node != nil && e.Node.Props["Name"] == name {
				return e.Node.Props["Value"] == value &&
					e.Node.Props["Source"] == source
			}
		}
		return false
	}
	if !checkVar("$brand-primary", "project-main", "#1565C0") {
		t.Error("$brand-primary from main.scss not found with value #1565C0")
	}
	if !checkVar("--brand-primary", "project-custom-variables", "#264ae5") {
		t.Error("--brand-primary from custom-variables not found")
	}
}

func TestThemeScssAdapter_Build_NoFiles(t *testing.T) {
	dir := t.TempDir() // empty
	a := &ThemeScssAdapter{ProjectDir: dir}
	sink := &recordingSink{}
	err := a.Build(context.Background(), sink)
	if err != nil {
		t.Fatal(err)
	}
	if len(sink.events) != 0 {
		t.Errorf("expected 0 events for empty project, got %d", len(sink.events))
	}
}
```

- [ ] **Step 2: 运行测试**

```bash
go test ./internal/mxgraph/adapter/themescss/... -v -count=1 2>&1 || true
# 预期: package not found
```

- [ ] **Step 3: 实现节点定义**

```go
// internal/mxgraph/adapter/themescss/nodes.go
package themescss

import (
	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/internal/mxgraph/scss"
)

// varNode 从 ScssVarDecl 构造 mxgraph 节点。
func varNode(decl scss.ScssVarDecl, source, module, filePath string) *mxgraph.Node {
	category := inferCategory(decl.Name)
	props := map[string]any{
		"Name":         decl.Name,
		"Value":        decl.Value,
		"VariableType": variableType(decl),
		"IsDefault":    decl.IsDefault,
		"IsActive":     decl.IsActive,
		"IsCSSVar":     decl.IsCSSVar,
		"Source":       source,
		"FilePath":     filePath,
		"LineNumber":   decl.LineIdx + 1,
		"Category":     category,
		"$Type":        "ThemeVariable",
	}
	if module != "" {
		props["Module"] = module
		props["QualifiedName"] = module + "." + decl.Name
	} else {
		props["QualifiedName"] = decl.Name
	}
	return &mxgraph.Node{
		ID:    mxgraph.NodeID(source + ":" + decl.Name),
		Label: "ThemeVariable",
		Props: props,
	}
}

func variableType(d scss.ScssVarDecl) string {
	if d.IsCSSVar {
		return "css-custom-property"
	}
	return "sass"
}

// 根据变量名推断分类
func inferCategory(name string) string {
	switch {
	case matchPrefix(name, "$brand", "--brand"):
		return "brand"
	case matchPrefix(name, "$font", "--font"):
		return "font"
	case matchPrefix(name, "$spacing", "--spacing"):
		return "spacing"
	case matchPrefix(name, "$nav", "--nav", "$navsidebar", "--navsidebar", "$navtopbar", "--navtopbar", "--navigation"):
		return "navigation"
	case matchPrefix(name, "$btn", "--btn"):
		return "button"
	case matchPrefix(name, "$form", "--form"):
		return "form"
	case matchPrefix(name, "$border", "--border"):
		return "border"
	case matchPrefix(name, "$bg", "--bg"):
		return "background"
	case matchPrefix(name, "$grid", "--grid"):
		return "grid"
	case matchPrefix(name, "$tab", "--tab"):
		return "tabs"
	case matchPrefix(name, "$modal", "--modal"):
		return "modal"
	case matchPrefix(name, "$card", "--card"):
		return "card"
	case matchPrefix(name, "$alert", "--alert"):
		return "alert"
	case matchPrefix(name, "$label", "--label"):
		return "label"
	case matchPrefix(name, "$shadow", "--shadow"):
		return "shadow"
	case matchPrefix(name, "$groupbox", "--groupbox"):
		return "groupbox"
	case matchPrefix(name, "$callout", "--callout"):
		return "callout"
	case matchPrefix(name, "$header", "--header"):
		return "header"
	case matchPrefix(name, "$link", "--link"):
		return "link"
	default:
		return "other"
	}
}

func matchPrefix(name string, prefixes ...string) bool {
	for _, p := range prefixes {
		if len(name) >= len(p) && name[:len(p)] == p {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: 实现适配器**

```go
// internal/mxgraph/adapter/themescss/adapter.go
package themescss

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/internal/mxgraph/scss"
)

// scssSource 描述一个 SCSS 文件及其在主题系统中的角色。
type scssSource struct {
	RelativePath string // 相对于 ProjectDir
	Source       string // "project-custom-variables" | "project-main" | "module-variables" | "module-main"
	Module       string // 模块名（仅模块级时有效）
}

// ThemeScssAdapter 索引所有 SCSS 文件中的主题变量为 ThemeVariable 节点。
type ThemeScssAdapter struct {
	ProjectDir string
}

func (a *ThemeScssAdapter) Name() string { return "themescss" }

func (a *ThemeScssAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{
		NodeLabels: []mxgraph.Label{"ThemeVariable"},
	}
}

func (a *ThemeScssAdapter) Build(ctx context.Context, sink mxgraph.EventSink) error {
	sources := a.discoverSources()
	var events []mxgraph.Event

	for _, src := range sources {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		fullPath := filepath.Join(a.ProjectDir, src.RelativePath)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}

		doc, err := scss.Parse(src.RelativePath, string(data))
		if err != nil {
			continue
		}

		for _, decl := range doc.Vars {
			events = append(events, mxgraph.Event{
				Type: mxgraph.NodeCreated,
				Node: varNode(decl, src.Source, src.Module, src.RelativePath),
			})
		}
	}

	if len(events) > 0 {
		if err := sink.Emit(events); err != nil {
			return err
		}
	}
	return nil
}

// discoverSources 扫描项目目录，发现所有主题 SCSS 文件。
func (a *ThemeScssAdapter) discoverSources() []scssSource {
	var sources []scssSource

	// 项目级文件
	projectFiles := []struct {
		relPath string
		source  string
	}{
		{"theme/web/custom-variables.scss", "project-custom-variables"},
		{"theme/web/main.scss", "project-main"},
		{"theme/web/exclusion-variables.scss", "project-exclusion-vars"},
		{"theme/web/_theme-dark.scss", "project-theme-variant"},
		{"theme/web/_theme-neutral.scss", "project-theme-variant"},
	}
	for _, pf := range projectFiles {
		if fileExists(filepath.Join(a.ProjectDir, pf.relPath)) {
			sources = append(sources, scssSource{
				RelativePath: pf.relPath,
				Source:       pf.source,
			})
		}
	}

	// Atlas Core 默认值（只读参考）
	atlasVarsPath := filepath.Join("themesource", "atlas_core", "web", "_variables.scss")
	if fileExists(filepath.Join(a.ProjectDir, atlasVarsPath)) {
		sources = append(sources, scssSource{
			RelativePath: atlasVarsPath,
			Source:       "atlas-core-default",
		})
	}

	// 模块级文件
	tsDir := filepath.Join(a.ProjectDir, "themesource")
	entries, err := os.ReadDir(tsDir)
	if err != nil {
		return sources
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		module := entry.Name()
		if module == "atlas_core" {
			continue // 已单独处理
		}
		modSources := []struct {
			relPath string
			source  string
		}{
			{filepath.Join("themesource", module, "web", "variables.scss"), "module-variables"},
			{filepath.Join("themesource", module, "web", "main.scss"), "module-main"},
		}
		for _, ms := range modSources {
			if fileExists(filepath.Join(a.ProjectDir, ms.relPath)) {
				sources = append(sources, scssSource{
					RelativePath: ms.relPath,
					Source:       ms.source,
					Module:       module,
				})
			}
		}
	}

	return sources
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (a *ThemeScssAdapter) Watch(ctx context.Context, sink mxgraph.EventSink) (func(), error) {
	// Phase 6 实现
	return func() {}, nil
}
```

- [ ] **Step 5: 运行测试**

```bash
go test ./internal/mxgraph/adapter/themescss/... -v -count=1
```

预期：全部 PASS。

- [ ] **Step 6: 注册到 cmd_graph.go**

在 `mdl/executor/cmd_graph.go` 的 `buildGraph()` 中追加：

```go
mgr.RegisterAdapter(&themescss.ThemeScssAdapter{ProjectDir: filepath.Dir(ctx.MprPath)})
```

记得在文件头部添加 import：
```go
themescss "github.com/mendixlabs/mxcli/internal/mxgraph/adapter/themescss"
```

- [ ] **Step 7: 运行测试**

```bash
go test ./mdl/executor/... -run TestBuildGraph -v -count=1 2>&1 | head -30
```

预期：不影响已有测试。

- [ ] **Step 8: Commit**

```bash
git add internal/mxgraph/adapter/themescss/ mdl/executor/cmd_graph.go
git commit -m "feat(mxgraph/adapter): add ThemeScssAdapter for SCSS variable indexing"
```

---

## Task 3：DesignPropertyAdapter — 设计属性定义 → 图节点

> **推理：** 复用现有 `theme_reader.go` 中的 `loadThemeRegistry()` 逻辑，但改为走 `EventSink` 而非直接 fmt 输出。DesignProperty 节点包含 WidgetType、选项列表、引用的主题变量名。

**Files:**
- Create: `internal/mxgraph/adapter/designdprops/nodes.go`
- Create: `internal/mxgraph/adapter/designdprops/adapter.go`
- Create: `internal/mxgraph/adapter/designdprops/adapter_test.go`

- [ ] **Step 1: 写失败测试**

```go
// internal/mxgraph/adapter/designdprops/adapter_test.go
package designdprops

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

type recordingSink struct {
	events []mxgraph.Event
}

func (s *recordingSink) Emit(events []mxgraph.Event) error {
	s.events = append(s.events, events...)
	return nil
}

func TestDesignPropertyAdapter_Name(t *testing.T) {
	a := &DesignPropertyAdapter{}
	if a.Name() != "designdprops" {
		t.Errorf("Name() = %q, want designdprops", a.Name())
	}
}

func TestDesignPropertyAdapter_Schema(t *testing.T) {
	a := &DesignPropertyAdapter{}
	s := a.Schema()
	if s == nil {
		t.Fatal("Schema() returned nil")
	}
	found := false
	for _, l := range s.NodeLabels {
		if l == "DesignProperty" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Schema missing label DesignProperty")
	}
}

func TestDesignPropertyAdapter_Build(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "themesource", "atlas_core", "web"), 0700)
	os.MkdirAll(filepath.Join(dir, "themesource", "datawidgets", "web"), 0700)

	// 模拟 design-properties.json
	os.WriteFile(filepath.Join(dir, "themesource", "atlas_core", "web", "design-properties.json"), []byte(`{
	"DivContainer": [
		{
			"name": "Card style",
			"type": "Toggle",
			"description": "Render container as card",
			"class": "card"
		},
		{
			"name": "Background color",
			"type": "ColorPicker",
			"options": [
				{ "name": "Brand Primary", "preview": "--brand-primary", "class": "background-primary" },
				{ "name": "Brand Success", "preview": "--brand-success", "class": "background-success" }
			]
		}
	],
	"Widget": [
		{
			"name": "Spacing",
			"type": "Spacing",
			"margin": [ { "name": "None", "top": { "class": "spacing-outer-top-none" } } ],
			"padding": [ { "name": "None", "top": { "class": "spacing-inner-top-none" } } ]
		}
	]
}`), 0600)

	os.WriteFile(filepath.Join(dir, "themesource", "datawidgets", "web", "design-properties.json"), []byte(`{
	"DataGrid": [
		{
			"name": "Striped rows",
			"type": "Toggle",
			"class": "datagrid-striped"
		}
	]
}`), 0600)

	a := &DesignPropertyAdapter{ProjectDir: dir}
	sink := &recordingSink{}
	err := a.Build(context.Background(), sink)
	if err != nil {
		t.Fatal(err)
	}

	// 验证节点数量: 3 个 DP (Card style, Background color, Spacing, Striped rows)
	// 其中 Spacing 是复合类型也会创建节点
	var nodeCreated int
	for _, e := range sink.events {
		if e.Type == mxgraph.NodeCreated {
			nodeCreated++
		}
	}
	if nodeCreated < 3 {
		t.Fatalf("NodeCreated events = %d, want >= 3", nodeCreated)
	}

	// 验证 Background color 有 referenced vars
	for _, e := range sink.events {
		if e.Node != nil && e.Node.Props["Name"] == "Background color" {
			vars, ok := e.Node.Props["ReferencedVars"].([]string)
			if !ok || len(vars) == 0 {
				t.Error("Background color should have ReferencedVars from preview fields")
			}
			break
		}
	}
}
```

- [ ] **Step 2: 运行测试**

```bash
go test ./internal/mxgraph/adapter/designdprops/... -v -count=1 2>&1 || true
```

- [ ] **Step 3: 实现适配器**

```go
// internal/mxgraph/adapter/designdprops/adapter.go
package designdprops

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

type DesignPropertyAdapter struct {
	ProjectDir string
}

func (a *DesignPropertyAdapter) Name() string { return "designdprops" }

func (a *DesignPropertyAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{
		NodeLabels: []mxgraph.Label{"DesignProperty"},
	}
}

// designPropFile 是 design-properties.json 的解析结构
type designPropFile map[string][]rawDesignPropDef

type rawDesignPropDef struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"` // "Toggle"|"Dropdown"|"ColorPicker"|"Spacing"|"ToggleButtonGroup"
	Category    string            `json:"category,omitempty"`
	Description string            `json:"description,omitempty"`
	Class       string            `json:"class,omitempty"`
	Options     []rawDesignOption `json:"options,omitempty"`
}

type rawDesignOption struct {
	Name    string `json:"name"`
	Class   string `json:"class,omitempty"`
	Preview string `json:"preview,omitempty"`
	Variable string `json:"variable,omitempty"`
}

func (a *DesignPropertyAdapter) Build(ctx context.Context, sink mxgraph.EventSink) error {
	tsDir := filepath.Join(a.ProjectDir, "themesource")
	entries, err := os.ReadDir(tsDir)
	if err != nil {
		return nil
	}

	var events []mxgraph.Event
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		module := entry.Name()
		dpPath := filepath.Join(tsDir, module, "web", "design-properties.json")
		data, err := os.ReadFile(dpPath)
		if err != nil {
			continue
		}

		var fileProps designPropFile
		if err := json.Unmarshal(data, &fileProps); err != nil {
			continue
		}

		for widgetType, props := range fileProps {
			for _, p := range props {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}

				refVars := extractReferencedVars(p)
				options := make([]string, len(p.Options))
				for i, o := range p.Options {
					options[i] = o.Name
				}

				nodeID := mxgraph.NodeID(fmt.Sprintf("%s.%s.%s", module, widgetType, p.Name))
				props := map[string]any{
					"$Type":           "DesignProperty",
					"WidgetType":      widgetType,
					"Name":            p.Name,
					"Type":            p.Type,
					"Category":        p.Category,
					"Description":     p.Description,
					"Class":           p.Class,
					"Options":         options,
					"ReferencedVars":  refVars,
					"SourceModule":    module,
					"QualifiedName":   fmt.Sprintf("%s.%s.%s", module, widgetType, p.Name),
				}

				events = append(events, mxgraph.Event{
					Type: mxgraph.NodeCreated,
					Node: &mxgraph.Node{ID: nodeID, Label: "DesignProperty", Props: props},
				})
			}
		}
	}

	if len(events) > 0 {
		return sink.Emit(events)
	}
	return nil
}

// extractReferencedVars 从设计属性选项中提取引用的主题变量名（preview 或 variable 字段）。
func extractReferencedVars(p rawDesignPropDef) []string {
	var vars []string
	seen := map[string]bool{}
	for _, o := range p.Options {
		for _, ref := range []string{o.Preview, o.Variable} {
			if ref != "" && !seen[ref] {
				seen[ref] = true
				prefixed := ref
				if !strings.HasPrefix(ref, "--") && !strings.HasPrefix(ref, "$") {
					prefixed = "--" + ref
				}
				vars = append(vars, prefixed)
			}
		}
	}
	return vars
}

func (a *DesignPropertyAdapter) Watch(ctx context.Context, sink mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}
```

- [ ] **Step 4: 注册到 cmd_graph.go**

```go
// import 追加:
designdprops "github.com/mendixlabs/mxcli/internal/mxgraph/adapter/designdprops"

// buildGraph 中注册:
mgr.RegisterAdapter(&themescss.ThemeScssAdapter{ProjectDir: filepath.Dir(ctx.MprPath)})
mgr.RegisterAdapter(&designdprops.DesignPropertyAdapter{ProjectDir: filepath.Dir(ctx.MprPath)})
```

- [ ] **Step 5: 运行测试**

```bash
go test ./internal/mxgraph/adapter/designdprops/... -v -count=1
go test ./internal/mxgraph/adapter/... -v -count=1
```

- [ ] **Step 6: Commit**

```bash
git add internal/mxgraph/adapter/designdprops/ mdl/executor/cmd_graph.go
git commit -m "feat(mxgraph/adapter): add DesignPropertyAdapter for design-properties.json indexing"
```

---

## Task 4：WidgetInstanceAdapter — 页面 Widget 的 Class/Style/DesignProperties

> **推理：** 现有 PageAdapter 只索引 Page 节点本身。需要新增子适配器遍历页面内部的 widget 树，提取 Appearance 数据。复用 `extractDesignProperties` 模式。

**Files:**
- Create: `internal/mxgraph/adapter/mpr/widget_instance.go`
- Create: `internal/mxgraph/adapter/mpr/widget_instance_test.go`
- Modify: `internal/mxgraph/adapter/mpr/page.go`（追加 Schema 中新增节点标签）

- [ ] **Step 1: 写失败测试**

```go
// internal/mxgraph/adapter/mpr/widget_instance_test.go
package mpr

import (
	"context"
	"testing"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

func TestWidgetInstanceAdapter_Name(t *testing.T) {
	a := &WidgetInstanceAdapter{}
	if a.Name() != "widgetinstance" {
		t.Errorf("Name() = %q, want widgetinstance", a.Name())
	}
}

func TestWidgetInstanceAdapter_Schema(t *testing.T) {
	a := &WidgetInstanceAdapter{}
	s := a.Schema()
	if s == nil {
		t.Fatal("Schema() returned nil")
	}
	labels := map[mxgraph.Label]bool{}
	for _, l := range s.NodeLabels {
		labels[l] = true
	}
	for _, want := range []mxgraph.Label{"WidgetInstance"} {
		if !labels[want] {
			t.Errorf("Schema missing label %q", want)
		}
	}
	rels := map[mxgraph.RelType]bool{}
	for _, et := range s.EdgeTypes {
		rels[et.Type] = true
	}
	for _, want := range []mxgraph.RelType{"HAS_WIDGET_INSTANCE"} {
		if !rels[want] {
			t.Errorf("Schema missing edge type %q", want)
		}
	}
}
```

- [ ] **Step 2: 运行测试**

```bash
go test ./internal/mxgraph/adapter/mpr/... -run TestWidgetInstance -v
```

- [ ] **Step 3: 实现适配器**

```go
// internal/mxgraph/adapter/mpr/widget_instance.go
package mpr

import (
	"context"
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/modelsdk"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// WidgetInstanceAdapter 遍历页面和片段中的 widget 树，提取 Appearance 数据。
// 每个带样式的 widget 实例生成一个 WidgetInstance 节点，通过 HAS_WIDGET_INSTANCE 边连接到其页面/片段。
type WidgetInstanceAdapter struct {
	Model *modelsdk.Model
}

func (a *WidgetInstanceAdapter) Name() string { return "widgetinstance" }

func (a *WidgetInstanceAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{
		NodeLabels: []mxgraph.Label{"WidgetInstance"},
		EdgeTypes: []struct {
			Type mxgraph.RelType
			From mxgraph.Label
			To   mxgraph.Label
		}{
			{"HAS_WIDGET_INSTANCE", "Page", "WidgetInstance"},
			{"HAS_WIDGET_INSTANCE", "Snippet", "WidgetInstance"},
		},
	}
}

func (a *WidgetInstanceAdapter) Build(ctx context.Context, sink mxgraph.EventSink) error {
	var events []mxgraph.Event

	for _, unit := range a.Model.Units() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		elem, err := a.Model.LoadUnit(unit.ID)
		if err != nil {
			continue
		}

		typeName := elem.TypeName()
		var containerLabel mxgraph.Label
		switch {
		case strings.HasSuffix(typeName, "$Page") || strings.HasSuffix(typeName, "$Form"):
			containerLabel = "Page"
		case strings.HasSuffix(typeName, "$Snippet"):
			containerLabel = "Snippet"
		default:
			continue
		}

		module := a.Model.ResolveModuleName(unit.ID)
		containerID := mxgraph.NodeID(elem.ID())

		// 遍历 widget 树
		wiEvents := a.walkWidgets(elem, containerID, containerLabel, module, unit.ID, elem.ID())
		events = append(events, wiEvents...)
	}

	if len(events) > 0 {
		return sink.Emit(events)
	}
	return nil
}

// walkWidgets 递归遍历 widget 树，提取 Appearance 数据。
func (a *WidgetInstanceAdapter) walkWidgets(
	elem element.Element,
	containerID mxgraph.NodeID,
	containerLabel mxgraph.Label,
	module string,
	unitID string,
	containerElemID element.ID,
) []mxgraph.Event {
	var events []mxgraph.Event

	// 遍历所有子属性查找 widget 容器
	for _, prop := range elem.Properties() {
		switch p := prop.(type) {
		case element.ChildProperty:
			// 单个子元素（如 Widget、Layout 等）
			if child := p.ChildElement(); child != nil {
				evts := a.inspectWidget(child, containerID, containerLabel, module, unitID, containerElemID)
				events = append(events, evts...)
				// 递归到子元素
				events = append(events, a.walkWidgets(child, containerID, containerLabel, module, unitID, containerElemID)...)
			}
		case element.ChildListProperty:
			// 子元素列表（如 Widgets 列表）
			for _, child := range p.ChildElements() {
				if child == nil {
					continue
				}
				evts := a.inspectWidget(child, containerID, containerLabel, module, unitID, containerElemID)
				events = append(events, evts...)
				events = append(events, a.walkWidgets(child, containerID, containerLabel, module, unitID, containerElemID)...)
			}
		}
	}

	return events
}

// inspectWidget 检查单个 widget 元素，如果有 Appearance 则创建 WidgetInstance 节点。
func (a *WidgetInstanceAdapter) inspectWidget(
	widget element.Element,
	containerID mxgraph.NodeID,
	containerLabel mxgraph.Label,
	module string,
	unitID string,
	containerElemID element.ID,
) []mxgraph.Event {
	typeName := widget.TypeName()
	// 只处理 widget 类型（跳过容器类型如 LayoutCall、FormCall 等）
	if !isWidgetType(typeName) {
		return nil
	}

	// 获取 widget name
	widgetName := getWidgetNameFromElement(widget)
	if widgetName == "" {
		return nil
	}

	// 提取 Appearance
	appearance := extractAppearanceFromWidget(widget)
	if appearance == nil {
		return nil
	}

	// 提取 class, style, design properties
	cls, _ := appearance["Class"].(string)
	style, _ := appearance["Style"].(string)
	designProps := extractDesignPropsMap(appearance)

	// 如果没有 class/style/design props，跳过（或也索引空样式的 widget？）
	// 策略：只索引有 style 信息的 widget
	if cls == "" && style == "" && len(designProps) == 0 {
		return nil
	}

	widgetID := mxgraph.NodeID(widget.ID())
	qn := fmt.Sprintf("%s.%s", module, widgetName)
	if module == "" {
		qn = widgetName
	}

	props := map[string]any{
		"$Type":            "WidgetInstance",
		"Name":             widgetName,
		"WidgetType":       widgetTypeName(typeName),
		"Class":            cls,
		"Style":            style,
		"DesignProperties": designProps,
		"ElementID":        string(widget.ID()),
		"QualifiedName":    qn,
		"Module":           module,
	}

	var events []mxgraph.Event
	events = append(events, mxgraph.Event{
		Type: mxgraph.NodeCreated,
		Node: &mxgraph.Node{ID: widgetID, Label: "WidgetInstance", Props: props},
	})
	events = append(events, mxgraph.Event{
		Type: mxgraph.EdgeCreated,
		Edge: &mxgraph.Edge{
			ID:   mxgraph.NodeID(fmt.Sprintf("%s--HAS_WIDGET_INSTANCE-->%s", containerID, widgetID)),
			From: containerID,
			To:   widgetID,
			Type: "HAS_WIDGET_INSTANCE",
		},
	})

	return events
}

// isWidgetType 判断 BSON 类型名是否为可渲染 widget。
func isWidgetType(typeName string) bool {
	// 排除非 widget 类型
	skipTypes := map[string]bool{
		"Forms$LayoutCall":   true,
		"Forms$FormCall":     true,
		"Forms$Appearance":   true,
		"Forms$DesignPropertyValue": true,
	}
	if skipTypes[typeName] {
		return false
	}
	// Widget 类型通常以 Forms$ 或 Pages$ 开头但不是上面排除的类型
	return strings.HasPrefix(typeName, "Forms$") || strings.HasPrefix(typeName, "Pages$")
}

// getWidgetNameFromElement 从 element.Element 提取 widget 的 Name 属性。
func getWidgetNameFromElement(elem element.Element) string {
	for _, p := range elem.Properties() {
		if p.Name() == "Name" {
			if wp, ok := p.(element.WritableProperty); ok {
				if v := wp.BSONValue(); v != nil {
					if s, ok := v.(string); ok {
						return s
					}
				}
			}
		}
	}
	return ""
}

// extractAppearanceFromWidget 从 widget 元素中提取 Appearance 子元素。
func extractAppearanceFromWidget(widget element.Element) map[string]any {
	for _, prop := range widget.Properties() {
		if prop.Name() != "Appearance" {
			continue
		}
		cp, ok := prop.(element.ChildProperty)
		if !ok {
			continue
		}
		app := cp.ChildElement()
		if app == nil {
			return nil
		}
		// 将 Appearance 转换为 map 以提取字段
		result := map[string]any{
			"$Type": app.TypeName(),
		}
		for _, ap := range app.Properties() {
			if wp, ok := ap.(element.WritableProperty); ok {
				if v := wp.BSONValue(); v != nil {
					result[ap.Name()] = v
				}
			}
		}
		// 提取 DesignProperties (child list)
		for _, ap := range app.Properties() {
			if ap.Name() != "DesignProperties" {
				continue
			}
			cl, ok := ap.(element.ChildListProperty)
			if !ok {
				continue
			}
			var dpList []map[string]any
			for _, dpv := range cl.ChildElements() {
				if dpv == nil {
					continue
				}
				dpMap := map[string]any{"$Type": dpv.TypeName()}
				for _, dpProp := range dpv.Properties() {
					if wp, ok := dpProp.(element.WritableProperty); ok {
						if v := wp.BSONValue(); v != nil {
							dpMap[dpProp.Name()] = v
						}
					}
				}
				// 提取嵌套 Value
				for _, dpProp := range dpv.Properties() {
					if dpProp.Name() != "Value" {
						continue
					}
					cp2, ok := dpProp.(element.ChildProperty)
					if !ok {
						continue
					}
					val := cp2.ChildElement()
					if val == nil {
						continue
					}
					valMap := map[string]any{"$Type": val.TypeName()}
					for _, vp := range val.Properties() {
						if wp, ok := vp.(element.WritableProperty); ok {
							if bv := wp.BSONValue(); bv != nil {
								valMap[vp.Name()] = bv
							}
						}
					}
					dpMap["Value"] = valMap
				}
				dpList = append(dpList, dpMap)
			}
			result["DesignProperties"] = dpList
		}
		return result
	}
	return nil
}

// extractDesignPropsMap 提取设计属性为 Key→Value 的 map。
func extractDesignPropsMap(appearance map[string]any) map[string]string {
	result := make(map[string]string)
	dps, ok := appearance["DesignProperties"].([]map[string]any)
	if !ok {
		return result
	}
	for _, dp := range dps {
		key, _ := dp["Key"].(string)
		if key == "" {
			continue
		}
		valueMap, ok := dp["Value"].(map[string]any)
		if !ok {
			result[key] = "ON"
			continue
		}
		innerType, _ := valueMap["$Type"].(string)
		switch {
		case strings.Contains(innerType, "ToggleDesignPropertyValue"):
			result[key] = "ON"
		case strings.Contains(innerType, "OptionDesignPropertyValue"):
			opt, _ := valueMap["Option"].(string)
			result[key] = opt
		case strings.Contains(innerType, "CustomDesignPropertyValue"):
			val, _ := valueMap["Value"].(string)
			result[key] = val
		default:
			result[key] = "ON"
		}
	}
	return result
}

// widgetTypeName 将 BSON 类型名转为简短的可读名称。
func widgetTypeName(bsonType string) string {
	parts := strings.Split(bsonType, "$")
	if len(parts) < 2 {
		return bsonType
	}
	return parts[1]
}

func (a *WidgetInstanceAdapter) Watch(ctx context.Context, sink mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}
```

- [ ] **Step 4: 注册到 cmd_graph.go**

```go
mgr.RegisterAdapter(&mpradapter.WidgetInstanceAdapter{Model: m})
```

- [ ] **Step 5: 运行测试**

```bash
go test ./internal/mxgraph/adapter/mpr/... -run TestWidgetInstance -v -count=1
go test ./internal/mxgraph/adapter/... -v -count=1
```

- [ ] **Step 6: 用 HelpDeskE2E 验证全量构建是否正常**

```bash
go build ./...
# 确认编译通过
```

- [ ] **Step 7: Commit**

```bash
git add internal/mxgraph/adapter/mpr/widget_instance.go internal/mxgraph/adapter/mpr/widget_instance_test.go
git commit -m "feat(mxgraph/adapter): add WidgetInstanceAdapter for widget Appearance indexing"
```

---

## Task 5：graphcatalog 主题查询接口

> **推理：** 现有 `graphcatalog/reader.go` 定义了 DomainReader / BehaviorReader / SecurityReader 等。追加 ThemeReader / StylingReader 接口 + 类型化节点 + ProjectGraph 实现。

**Files:**
- Create: `mdl/graphcatalog/theme_nodes.go`
- Modify: `mdl/graphcatalog/reader.go`
- Modify: `mdl/graphcatalog/graph.go`
- Modify: `mdl/graphcatalog/mock/mock.go`

- [ ] **Step 1: 创建主题类型化节点**

```go
// mdl/graphcatalog/theme_nodes.go
package graphcatalog

// ThemeVariableNode 对应 label="ThemeVariable" 的节点。
type ThemeVariableNode struct {
	Name         string
	Value        string
	VariableType string // "sass" | "css-custom-property"
	IsDefault    bool
	IsActive     bool
	Source       string // "project-custom-variables" | "project-main" | "module:xxx" | "atlas-core"
	Module       string
	Category     string
	FilePath     string
	LineNumber   int
}

// WidgetInstanceNode 对应 label="WidgetInstance" 的节点。
type WidgetInstanceNode struct {
	ID               string
	Name             string
	WidgetType       string
	Class            string
	Style            string
	DesignProperties map[string]string
	Page             string
}

// DesignPropertyNode 对应 label="DesignProperty" 的节点。
type DesignPropertyNode struct {
	WidgetType      string
	Name            string
	Type            string
	Category        string
	Description     string
	Options         []string
	ReferencedVars  []string
	SourceModule    string
}
```

- [ ] **Step 2: 扩展 reader.go**

在 `mdl/graphcatalog/reader.go` 末尾追加：

```go
// ThemeReader 读取主题变量。SHOW THEME 命令和 AI 工具使用。
type ThemeReader interface {
	ThemeVariables(module string, filter ThemeVarFilter) []ThemeVariableNode
	ThemeVariable(name string) *ThemeVariableNode
	ThemeVariablesByCategory(category string) []ThemeVariableNode
	OverriddenVariables() []ThemeVariableNode
}

type ThemeVarFilter struct {
	ActiveOnly bool
	Like       string
	Source     string // "project" | "module" | "atlas-core"
}

// StylingReader 读取 widget 样式信息。
type StylingReader interface {
	WidgetInstances(pageQN string) []WidgetInstanceNode
	DesignProperties(widgetType string) []DesignPropertyNode
	DesignProperty(widgetType, name string) *DesignPropertyNode
	WidgetsByDesignProperty(dpQN string) []WidgetInstanceNode
	WidgetsByClass(className string) []WidgetInstanceNode
}

// LintReader 扩展
type LintReader interface {
	DomainReader
	BehaviorReader
	SecurityReader
	ExtensionReader
	// ThemeReader      // 可选：lint 暂时不需要
	// StylingReader    // 可选
}

// 编译期检查
var _ ThemeReader = (*ProjectGraph)(nil)
var _ StylingReader = (*ProjectGraph)(nil)
```

- [ ] **Step 3: 运行测试确认编译失败**

```bash
go test ./mdl/graphcatalog/... -v -count=1 2>&1 | head -30
# 预期: ProjectGraph 缺少 ThemeReader/StylingReader 方法
```

- [ ] **Step 4: 实现 ProjectGraph 的方法**

在 `mdl/graphcatalog/graph.go` 末尾追加：

```go
// ── ThemeReader 实现 ─────────────────────────────────────────

func (pg *ProjectGraph) ThemeVariables(module string, filter ThemeVarFilter) []ThemeVariableNode {
	g := pg.mgr.Query()
	nodes := g.FindNodes("ThemeVariable", nil)
	var result []ThemeVariableNode
	for _, n := range nodes {
		if filter.ActiveOnly {
			if active, ok := n.Props["IsActive"].(bool); ok && !active {
				continue
			}
		}
		if filter.Like != "" {
			name, _ := n.Props["Name"].(string)
			if !strings.Contains(name, filter.Like) {
				continue
			}
		}
		if filter.Source != "" {
			src, _ := n.Props["Source"].(string)
			if src != filter.Source {
				continue
			}
		}
		if module != "" {
			m, _ := n.Props["Module"].(string)
			if m != module {
				continue
			}
		}
		result = append(result, toThemeVarNode(n))
	}
	return result
}

func (pg *ProjectGraph) ThemeVariable(name string) *ThemeVariableNode {
	g := pg.mgr.Query()
	// FindNodes with prop filter
	nodes := g.FindNodes("ThemeVariable", map[string]any{"Name": name})
	if len(nodes) == 0 {
		return nil
	}
	result := toThemeVarNode(nodes[0])
	return &result
}

func (pg *ProjectGraph) ThemeVariablesByCategory(category string) []ThemeVariableNode {
	return pg.ThemeVariables("", ThemeVarFilter{ActiveOnly: true, Like: ""})
}

func (pg *ProjectGraph) OverriddenVariables() []ThemeVariableNode {
	g := pg.mgr.Query()
	nodes := g.FindNodes("ThemeVariable", nil)
	var result []ThemeVariableNode
	for _, n := range nodes {
		src, _ := n.Props["Source"].(string)
		if src == "atlas-core-default" {
			continue
		}
		active, _ := n.Props["IsActive"].(bool)
		if !active {
			continue
		}
		result = append(result, toThemeVarNode(n))
	}
	return result
}

func toThemeVarNode(n *mxgraph.Node) ThemeVariableNode {
	return ThemeVariableNode{
		Name:         getStrProp(n, "Name"),
		Value:        getStrProp(n, "Value"),
		VariableType: getStrProp(n, "VariableType"),
		IsDefault:    getBoolProp(n, "IsDefault"),
		IsActive:     getBoolProp(n, "IsActive"),
		Source:       getStrProp(n, "Source"),
		Module:       getStrProp(n, "Module"),
		Category:     getStrProp(n, "Category"),
		FilePath:     getStrProp(n, "FilePath"),
		LineNumber:   getIntProp(n, "LineNumber"),
	}
}

// ── StylingReader 实现 ────────────────────────────────────────

func (pg *ProjectGraph) WidgetInstances(pageQN string) []WidgetInstanceNode {
	g := pg.mgr.Query()
	nodes := g.FindNodes("WidgetInstance", nil)
	var result []WidgetInstanceNode
	for _, n := range nodes {
		page, _ := n.Props["Page"].(string)
		if page != pageQN {
			continue
		}
		result = append(result, toWidgetInstanceNode(n))
	}
	return result
}

func (pg *ProjectGraph) DesignProperties(widgetType string) []DesignPropertyNode {
	g := pg.mgr.Query()
	nodes := g.FindNodes("DesignProperty", nil)
	var result []DesignPropertyNode
	for _, n := range nodes {
		wt, _ := n.Props["WidgetType"].(string)
		if widgetType != "" && wt != widgetType {
			continue
		}
		result = append(result, toDesignPropertyNode(n))
	}
	return result
}

func (pg *ProjectGraph) DesignProperty(widgetType, name string) *DesignPropertyNode {
	g := pg.mgr.Query()
	// Prop index lookup
	nodes := g.FindNodes("DesignProperty", map[string]any{
		"WidgetType": widgetType,
		"Name":       name,
	})
	if len(nodes) == 0 {
		return nil
	}
	result := toDesignPropertyNode(nodes[0])
	return &result
}

func (pg *ProjectGraph) WidgetsByDesignProperty(dpQN string) []WidgetInstanceNode {
	g := pg.mgr.Query()
	// 通过 DesignProperty 的边找 WidgetInstance
	// 遍历 APPLIES_DESIGN_PROPERTY 边 (Future: 需要先创建这些边)
	return nil
}

func (pg *ProjectGraph) WidgetsByClass(className string) []WidgetInstanceNode {
	g := pg.mgr.Query()
	nodes := g.FindNodes("WidgetInstance", nil)
	var result []WidgetInstanceNode
	for _, n := range nodes {
		cls, _ := n.Props["Class"].(string)
		if strings.Contains(cls, className) {
			result = append(result, toWidgetInstanceNode(n))
		}
	}
	return result
}

func toWidgetInstanceNode(n *mxgraph.Node) WidgetInstanceNode {
	dp, _ := n.Props["DesignProperties"].(map[string]string)
	return WidgetInstanceNode{
		ID:               string(n.ID),
		Name:             getStrProp(n, "Name"),
		WidgetType:       getStrProp(n, "WidgetType"),
		Class:            getStrProp(n, "Class"),
		Style:            getStrProp(n, "Style"),
		DesignProperties: dp,
		Page:             getStrProp(n, "Page"),
	}
}

func toDesignPropertyNode(n *mxgraph.Node) DesignPropertyNode {
	refVars, _ := n.Props["ReferencedVars"].([]string)
	opts, _ := n.Props["Options"].([]string)
	return DesignPropertyNode{
		WidgetType:     getStrProp(n, "WidgetType"),
		Name:           getStrProp(n, "Name"),
		Type:           getStrProp(n, "Type"),
		Category:       getStrProp(n, "Category"),
		Description:    getStrProp(n, "Description"),
		Options:        opts,
		ReferencedVars: refVars,
		SourceModule:   getStrProp(n, "SourceModule"),
	}
}

// 辅助：从 Props 中安全提取值的函数
func getStrProp(n *mxgraph.Node, key string) string {
	if n == nil {
		return ""
	}
	v, _ := n.Props[key].(string)
	return v
}

func getBoolProp(n *mxgraph.Node, key string) bool {
	if n == nil {
		return false
	}
	v, _ := n.Props[key].(bool)
	return v
}

func getIntProp(n *mxgraph.Node, key string) int {
	if n == nil {
		return 0
	}
	v, _ := n.Props[key].(int)
	return v
}
```

确保 import 追加 `"strings"`。

- [ ] **Step 5: 扩展 mock.go**

```go
// 追加到 MockProjectGraph

ThemeVariablesFunc          func(module string, filter graphcatalog.ThemeVarFilter) []graphcatalog.ThemeVariableNode
ThemeVariableFunc           func(name string) *graphcatalog.ThemeVariableNode
ThemeVariablesByCategoryFunc func(category string) []graphcatalog.ThemeVariableNode
OverriddenVariablesFunc     func() []graphcatalog.ThemeVariableNode
WidgetInstancesFunc         func(pageQN string) []graphcatalog.WidgetInstanceNode
DesignPropertiesFunc        func(widgetType string) []graphcatalog.DesignPropertyNode
DesignPropertyFunc          func(widgetType, name string) *graphcatalog.DesignPropertyNode
WidgetsByDesignPropertyFunc func(dpQN string) []graphcatalog.WidgetInstanceNode
WidgetsByClassFunc          func(className string) []graphcatalog.WidgetInstanceNode

// 编译期检查追加
var _ graphcatalog.ThemeReader = (*MockProjectGraph)(nil)
var _ graphcatalog.StylingReader = (*MockProjectGraph)(nil)

// 方法实现（panic-on-default 模式，同上）
```

- [ ] **Step 6: 运行测试**

```bash
go test ./mdl/graphcatalog/... -v -count=1
```

预期：`TestInterfaceCompliance` PASS（包含 ThemeReader 和 StylingReader）。

- [ ] **Step 7: Commit**

```bash
git add mdl/graphcatalog/
git commit -m "feat(graphcatalog): add ThemeReader and StylingReader interfaces with ProjectGraph impl"
```

---

## Task 6：Persistence 增量系统接入

> **推理：** `persist.go` 已有 `DeltaWriter`/`DeltaReader` 结构体，仅需在 `IndexManager` 中接入，添加 `Persistence` 生命周期。

**Files:**
- Modify: `internal/mxgraph/persist.go`（Persistence struct + Init）
- Modify: `internal/mxgraph/adapter.go`（IndexManager 接入）
- Modify: `internal/mxgraph/engine.go`（DirtyTracker）
- Create: `internal/mxgraph/persist_test.go`

- [ ] **Step 1: 写 Persistence 测试**

```go
// internal/mxgraph/persist_test.go
package mxgraph

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPersistence_SaveAndLoadSnapshot(t *testing.T) {
	dir := t.TempDir()
	p := NewPersistence(filepath.Join(dir, "mxgraph"))

	// Build a graph
	mgr := NewIndexManager()
	mgr.SetPersistence(p)
	g := mgr.Query()
	g.AddNode("n1", "ThemeVariable", map[string]any{"Name": "$brand", "Value": "#1565C0"})
	g.AddNode("n2", "ThemeVariable", map[string]any{"Name": "$font", "Value": "14px"})
	g.AddEdge("e1", "n1", "n2", "REFERENCES", nil)

	// Save
	if err := p.SaveSnapshot(g); err != nil {
		t.Fatal(err)
	}

	// Verify files exist
	if _, err := os.Stat(filepath.Join(dir, "mxgraph", "snapshot")); err != nil {
		t.Errorf("snapshot file missing: %v", err)
	}
}

func TestPersistence_DeltaAppendAndReplay(t *testing.T) {
	dir := t.TempDir()
	p := NewPersistence(filepath.Join(dir, "mxgraph"))

	// Append events
	p.AppendEvent(Event{Type: NodeCreated, Node: &Node{ID: "n1", Label: "Test"}})
	p.AppendEvent(Event{Type: NodeCreated, Node: &Node{ID: "n2", Label: "Test"}})
	p.AppendEvent(Event{Type: EdgeCreated, Edge: &Edge{ID: "e1", From: "n1", To: "n2", Type: "LINK"}})

	// Read back
	g2 := New()
	events, err := p.ReadDelta()
	if err != nil {
		t.Fatal(err)
	}
	g2.Apply(events)

	if g2.GetNode("n1") == nil {
		t.Error("n1 not found after replay")
	}
	if g2.GetNode("n2") == nil {
		t.Error("n2 not found after replay")
	}
}
```

- [ ] **Step 2: 实现 Persistence**

```go
// 追加到 internal/mxgraph/persist.go

import "path/filepath"

// Persistence 管理图快照和增量日志的生命周期。
type Persistence struct {
	Dir          string
	SnapshotPath string
	DeltaPath    string
	dw           *DeltaWriter
}

func NewPersistence(dir string) *Persistence {
	return &Persistence{
		Dir:          dir,
		SnapshotPath: filepath.Join(dir, "snapshot"),
		DeltaPath:    filepath.Join(dir, "delta.log"),
	}
}

// Init 尝试从持久化状态恢复：
// 1. 读取 snapshot
// 2. 回放 delta.log 中快照之后的变更
func (p *Persistence) Init() (*Graph, error) {
	os.MkdirAll(p.Dir, 0700)

	// 尝试加载快照
	snapData, err := os.ReadFile(p.SnapshotPath)
	if err != nil {
		return nil, err // 没有快照，触发全量构建
	}

	g, err := UnmarshalSnapshot(snapData)
	if err != nil {
		return nil, err
	}

	// 回放 delta 日志中 checkpoint 之后的变更
	deltaEvents, err := p.ReadDelta()
	if err != nil {
		return g, nil // delta 回放非致命
	}
	g.Apply(deltaEvents)

	// 打开 delta 写入流
	f, err := os.OpenFile(p.DeltaPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return g, nil
	}
	p.dw = NewDeltaWriter(f)

	return g, nil
}

func (p *Persistence) AppendEvent(ev Event) error {
	if p.dw == nil {
		return nil
	}
	return p.dw.WriteEvent(ev)
}

func (p *Persistence) SaveSnapshot(g *Graph) error {
	os.MkdirAll(p.Dir, 0700)
	data, err := MarshalSnapshot(g)
	if err != nil {
		return err
	}
	// 原子写入
	tmpPath := p.SnapshotPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, p.SnapshotPath); err != nil {
		return err
	}
	// 写入 checkpoint 到 delta
	if p.dw != nil {
		p.dw.WriteCheckpoint()
	}
	return nil
}

func (p *Persistence) ReadDelta() ([]Event, error) {
	data, err := os.ReadFile(p.DeltaPath)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}
	dr := NewDeltaReader(bytes.NewReader(data))
	var events []Event
	for {
		ev, err := dr.ReadEvent()
		if err == io.EOF {
			break
		}
		if err != nil {
			return events, err // 部分回放
		}
		events = append(events, ev)
	}
	return events, nil
}
```

- [ ] **Step 3: 在 IndexManager 中接入**

```go
// 修改 internal/mxgraph/adapter.go

type IndexManager struct {
	graph    *Graph
	adapters map[string]IndexAdapter
	persist  *Persistence  // NEW
}

func (m *IndexManager) SetPersistence(p *Persistence) {
	m.persist = p
}

func (m *IndexManager) Emit(events []Event) error {
	m.graph.Apply(events)
	// 持久化每个事件到 delta 日志
	if m.persist != nil {
		for _, ev := range events {
			if err := m.persist.AppendEvent(ev); err != nil {
				// 日志持久化失败不能阻塞构建
				// 记录日志即可
			}
		}
	}
	return nil
}

// DirtyGraph 返回脏跟踪器，用于增量 Watch() 的变更检测。
func (m *IndexManager) DirtyGraph() *Graph {
	return m.graph
}
```

- [ ] **Step 4: 运行测试**

```bash
go test ./internal/mxgraph/... -run TestPersistence -v -count=1
```

- [ ] **Step 5: Commit**

```bash
git add internal/mxgraph/persist.go internal/mxgraph/persist_test.go internal/mxgraph/adapter.go
git commit -m "feat(mxgraph): wire up Persistence with delta log and IndexManager"
```

---

## Task 7：Watch() 增量事件 — 文件监听

> **推理：** 主题变量和设计属性定义来自文件系统。SCSS 变更通过 fsnotify 监听 → diff 旧/新文档 → 发送增量事件。

**Files:**
- Create: `internal/mxgraph/adapter/themescss/watch.go`
- Create: `internal/mxgraph/adapter/designdprops/watch.go`
- Modify: `internal/mxgraph/adapter/themescss/adapter.go`
- Modify: `internal/mxgraph/adapter/designdprops/adapter.go`
- Modify: `mdl/executor/cmd_graph.go`

- [ ] **Step 1: 实现 ThemeScssAdapter 的 Watch**

```go
// internal/mxgraph/adapter/themescss/watch.go
package themescss

import (
	"context"
	"log"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/internal/mxgraph/scss"
)

// 缓存每个文件的文档内容，用于 diff
type fileCache map[string]*scss.ScssDocument

func (a *ThemeScssAdapter) Watch(ctx context.Context, sink mxgraph.EventSink) (func(), error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	// 扫描需要监听的目录
	watchDirs := []string{
		filepath.Join(a.ProjectDir, "theme", "web"),
	}
	tsDir := filepath.Join(a.ProjectDir, "themesource")
	if entries, err := filepath.Glob(filepath.Join(tsDir, "*", "web")); err == nil {
		watchDirs = append(watchDirs, entries...)
	}

	for _, dir := range watchDirs {
		watcher.Add(dir)
	}

	cache := make(fileCache)

	go func() {
		defer watcher.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if !strings.HasSuffix(event.Name, ".scss") {
					continue
				}
				if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
					continue
				}

				// 读取新内容
				relPath, _ := filepath.Rel(a.ProjectDir, event.Name)
				data, err := filepath.Rel(a.ProjectDir, event.Name)
				if err != nil {
					continue
				}
				_ = data

				newDoc, err := scss.Parse(event.Name, readFile(event.Name))
				if err != nil {
					continue
				}

				// 对比缓存
				oldDoc := cache[event.Name]
				if oldDoc == nil {
					// 首次变更：全量 emit
					var events []mxgraph.Event
					for _, decl := range newDoc.Vars {
						source := classifySource(relPath)
						module := extractModule(relPath)
						events = append(events, mxgraph.Event{
							Type: mxgraph.NodeCreated,
							Node: varNode(decl, source, module, relPath),
						})
					}
					if len(events) > 0 {
						sink.Emit(events)
					}
				} else {
					// diff 增量
					events := diffDocuments(oldDoc, newDoc, relPath)
					if len(events) > 0 {
						sink.Emit(events)
					}
				}
				cache[event.Name] = newDoc

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("themescss watch error: %v", err)
			}
		}
	}()

	return watcher.Close, nil
}

// diffDocuments 比较两个版本的 SCSS 文档，返回增量事件。
func diffDocuments(oldDoc, newDoc *scss.ScssDocument, relPath string) []mxgraph.Event {
	var events []mxgraph.Event

	// 建立索引
	oldVars := make(map[string]scss.ScssVarDecl)
	for _, v := range oldDoc.Vars {
		oldVars[v.Name] = v
	}
	newVars := make(map[string]scss.ScssVarDecl)
	for _, v := range newDoc.Vars {
		newVars[v.Name] = v
	}

	// 检测新增和更新
	for name, newDecl := range newVars {
		if oldDecl, ok := oldVars[name]; !ok {
			// 新增
			source := classifySource(relPath)
			module := extractModule(relPath)
			events = append(events, mxgraph.Event{
				Type: mxgraph.NodeCreated,
				Node: varNode(newDecl, source, module, relPath),
			})
		} else if oldDecl.Value != newDecl.Value || oldDecl.IsActive != newDecl.IsActive {
			// 值或状态变更
			source := classifySource(relPath)
			module := extractModule(relPath)
			node := varNode(newDecl, source, module, relPath)
			events = append(events, mxgraph.Event{
				Type: mxgraph.NodeUpdated,
				Node: node,
			})
		}
	}

	// 检测删除
	for name, oldDecl := range oldVars {
		if _, ok := newVars[name]; !ok {
			source := classifySource(relPath)
			node := varNode(oldDecl, source, "", relPath)
			events = append(events, mxgraph.Event{
				Type: mxgraph.NodeDeleted,
				Node: node,
			})
		}
	}

	return events
}

func classifySource(relPath string) string {
	switch {
	case strings.Contains(relPath, "theme/web/custom-variables.scss"):
		return "project-custom-variables"
	case strings.Contains(relPath, "theme/web/main.scss"):
		return "project-main"
	case strings.Contains(relPath, "themesource/"):
		if strings.Contains(relPath, "_variables.scss") {
			return "atlas-core-default"
		}
		if strings.Contains(relPath, "/main.scss") {
			return "module-main"
		}
		return "module-variables"
	default:
		return "other"
	}
}

func extractModule(relPath string) string {
	if strings.HasPrefix(relPath, "themesource/") {
		parts := strings.SplitN(relPath, "/", 3)
		if len(parts) >= 2 {
			return parts[1]
		}
	}
	return ""
}

func readFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}
```

- [ ] **Step 2: 设计 DesignPropertyAdapter 的 Watch 实现**

```go
// internal/mxgraph/adapter/designdprops/watch.go
// 与 themescss.Watch 模式相同，但监听 *.json 文件
// 关键差异:
//   - 过滤 design-properties.json
//   - diff 基于文件整体 JSON（按 widgetType+name 作为 key）
```

- [ ] **Step 3: 在 cmd_graph.go 中启动 Watch**

```go
func buildGraph(ctx *ExecContext) error {
    // ... 注册 adapter 和 BuildAll ...

    // NEW: 启动增量 Watch
    watchCtx, cancelWatch := context.WithCancel(context.Background())
    for _, name := range []string{"themescss", "designdprops"} {
        stop, err := mgr.Watch(watchCtx, name)
        if err == nil {
            defer stop()
        }
    }

    // ...
}
```

- [ ] **Step 4: 编译验证**

```bash
go build ./...
```

- [ ] **Step 5: Commit**

```bash
git add internal/mxgraph/adapter/themescss/watch.go internal/mxgraph/adapter/designdprops/watch.go mdl/executor/cmd_graph.go
git commit -m "feat(mxgraph): add Watch() incremental file change tracking for theme/design adapters"
```

---

## Task 8：cmd_theme.go 对接图查询

> **推理：** `cmd_theme.go`的 `SHOW THEME VARIABLES` 不再直接解析 SCSS 文件，而是通过 `ctx.Graph.ThemeVariables()` 从 mxgraph 查询。假设图已构建好。

- [ ] **Step 1: 修改 execShowThemeVariables 使用图查询**

```go
func execShowThemeVariables(ctx *ExecContext, s *ast.ShowThemeVariablesStmt) error {
    if ctx.Graph == nil {
        return mdlerrors.NewValidationf("graph not built — run REFRESH CATALOG first")
    }

    filter := graphcatalog.ThemeVarFilter{
        ActiveOnly: !s.ShowDefaults,
        Like:       s.LikePattern,
    }
    if s.ShowDefaults {
        filter.Source = "atlas-core-default"
    }
    vars := ctx.Graph.ThemeVariables(s.InModule, filter)
    // ... 格式化和输出 ...
}
```

- [ ] **Step 2: 注册 handler**

```go
// register_stubs.go
func registerThemeCommandHandlers(r *Registry) {
    r.Register(&ast.ShowThemeStatusStmt{}, func(ctx *ExecContext, s ast.Statement) error {
        return execShowThemeStatus(ctx, s.(*ast.ShowThemeStatusStmt))
    })
    r.Register(&ast.ShowThemeVariablesStmt{}, func(ctx *ExecContext, s ast.Statement) error {
        return execShowThemeVariables(ctx, s.(*ast.ShowThemeVariablesStmt))
    })
    r.Register(&ast.AlterThemeVariableStmt{}, func(ctx *ExecContext, s ast.Statement) error {
        return execAlterThemeVariable(ctx, s.(*ast.AlterThemeVariableStmt))
    })
    r.Register(&ast.CleanThemeDuplicatesStmt{}, func(ctx *ExecContext, s ast.Statement) error {
        return execCleanThemeDuplicates(ctx, s.(*ast.CleanThemeDuplicatesStmt))
    })
}
```

- [ ] **Step 3: 编译验证**

```bash
go build ./...
```

- [ ] **Step 4: 最终 Commit**

```bash
git add cmd/mxcli/ mdl/executor/cmd_theme.go mdl/executor/register_stubs.go mdl/executor/stmt_summary.go
git commit -m "feat(theme): wire SHOW/ALTER THEME commands to mxgraph-backed queries"
```

---

## 实施顺序依赖图

```
Task 1 (SCSS解析器)
    │
    ▼
Task 2 (ThemeScssAdapter) ───────────────────┐
    │                                          │
    ▼                                          ▼
Task 3 (DesignPropertyAdapter) ───→ Task 6 (增量持久化)
    │                                          │
    ▼                                          ▼
Task 4 (WidgetInstanceAdapter) ───→ Task 7 (Watch)
    │                                          │
    ▼                                          ▼
Task 5 (graphcatalog接口) ←──────────────────┘
    │
    ▼
Task 8 (cmd_theme 对接图)
```

**推荐并行执行：**
- Task 1（解析器基础，无外部依赖）
- Task 6（持久化系统，与源数据无关）

其余任务有前置依赖，不可并行。
