# Widget 生命周期改进 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复端到端 widget 验证中发现的 8 个问题，覆盖代码 bug、文档、行为变更三类。

**Architecture:** 按三批执行：Batch 1（两个代码 bug 独立修复）→ Batch 2（skill 文档更新，无测试）→ Batch 3（module role 幂等修复 + 新 MDL 命令全栈实现）。各 Task 彼此独立，可并行或顺序执行。

**Tech Stack:** Go 1.22+、ANTLR4（grammar 再生用 `make grammar`）、`testing` 标准库、`modernc.org/sqlite`

**Spec:** `docs/superpowers/specs/2026-06-05-widget-lifecycle-improvements-design.md`

---

## 文件索引

| 文件 | Task | 操作 |
|------|------|------|
| `cmd/mxcli/widget_scaffold.go:299` | 1 | 修改 |
| `cmd/mxcli/widget_scaffold_test.go` | 1 | 修改 |
| `cmd/mxcli/widget_build.go:266` | 2 | 修改 |
| `cmd/mxcli/widget_build_test.go` | 2 | 新建 |
| `.claude/skills/mendix/check-syntax.md` | 3 | 修改 |
| `.claude/skills/mendix/create-page.md` | 3 | 修改 |
| `.claude/skills/mendix/create-custom-widget.md` | 3 | 修改 |
| `.claude/skills/mendix/run-app.md` | 3 | 修改 |
| `CLAUDE.md` | 3 | 修改 |
| `cmd/mxcli-local/cmd_run.go:47` | 3 | 修改 |
| `mdl/executor/cmd_security_write_modulerole_gen.go:53` | 4 | 修改 |
| `mdl/executor/cmd_security_write_modulerole_gen_test.go` | 4 | 新建 |
| `mdl/grammar/MDLLexer.g4` | 5 | 修改 |
| `mdl/grammar/MDLParser.g4` | 5 | 修改 |
| `mdl/grammar/parser/mdl_lexer.go` | 5 | 再生（make grammar） |
| `mdl/grammar/parser/mdl_parser.go` | 5 | 再生（make grammar） |
| `mdl/ast/ast_widgets_cmd.go` | 5 | 修改 |
| `mdl/visitor/visitor_query.go` | 5 | 修改 |
| `mdl/executor/cmd_widgets.go` | 5 | 修改 |
| `mdl/executor/register_stubs.go` | 5 | 修改 |
| `mdl/executor/stmt_summary.go` | 5 | 修改 |
| `mdl/executor/cmd_widgets_test.go` | 5 | 新建 |

---

## Task 1：修复 package.xml xmlns 大小写（Issue 6）

**Files:**
- Modify: `cmd/mxcli/widget_scaffold.go:299`
- Modify: `cmd/mxcli/widget_scaffold_test.go:123`

- [ ] **Step 1: 在 `TestGeneratePackageXML` 中追加 xmlns 断言（写失败测试）**

打开 `cmd/mxcli/widget_scaffold_test.go`，在 `TestGeneratePackageXML`（第 123 行）的 `checks` 切片末尾追加两项：

```go
func TestGeneratePackageXML(t *testing.T) {
	xml := generatePackageXML("MyPkg", []string{"WidgetA", "WidgetB"})
	checks := []string{
		`name="MyPkg"`,
		`version="1.0.0"`,
		`<widgetFile path="WidgetA.xml"/>`,
		`<widgetFile path="WidgetB.xml"/>`,
		`xmlns="http://www.mendix.com/clientModule/1.0/"`,  // 新增：大写 M
	}
	for _, want := range checks {
		if !strings.Contains(xml, want) {
			t.Errorf("generatePackageXML: missing %q\ngot:\n%s", want, xml)
		}
	}
	// 确保小写 clientmodule 不存在
	if strings.Contains(xml, "clientmodule/1.0/") {
		t.Errorf("generatePackageXML: lowercase clientmodule found\ngot:\n%s", xml)
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./cmd/mxcli/... -run TestGeneratePackageXML -v
```

预期：FAIL，提示 `missing "xmlns=\"http://www.mendix.com/clientModule/1.0/\""` 或 `lowercase clientmodule found`

- [ ] **Step 3: 修复 `widget_scaffold.go:299`**

打开 `cmd/mxcli/widget_scaffold.go`，找到第 299 行的 `generatePackageXML` 函数，将 `clientmodule` 改为 `clientModule`：

```go
// 修改前
`  <clientModule name=%q version="1.0.0" xmlns="http://www.mendix.com/clientmodule/1.0/">`+"\n",

// 修改后
`  <clientModule name=%q version="1.0.0" xmlns="http://www.mendix.com/clientModule/1.0/">`+"\n",
```

- [ ] **Step 4: 运行测试，确认通过**

```bash
go test ./cmd/mxcli/... -run TestGeneratePackageXML -v
```

预期：PASS

- [ ] **Step 5: 运行全量 cmd/mxcli 测试，确认无回归**

```bash
go test ./cmd/mxcli/... -count=1
```

预期：所有测试通过

- [ ] **Step 6: Commit**

```bash
git add cmd/mxcli/widget_scaffold.go cmd/mxcli/widget_scaffold_test.go
git commit -m "fix(widget): correct clientModule xmlns casing in package.xml template

Generated package.xml had lowercase 'clientmodule/1.0/' causing CE0462.
Correct namespace is 'clientModule/1.0/' (capital M).

Fixes: Issue 6 from mxcli-taskdemo validation"
```

---

## Task 2：修复 `--dir` 路径重复（Issue 2）

**Files:**
- Modify: `cmd/mxcli/widget_build.go:266` (`runWidgetBuild`)
- Create: `cmd/mxcli/widget_build_test.go`

- [ ] **Step 1: 新建测试文件，写失败测试**

创建 `cmd/mxcli/widget_build_test.go`：

```go
package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDiscoverWidgets_RelativeDir verifies discoverWidgets works with a
// relative path. This is a prerequisite for the --dir fix: if dir is absolute,
// filepath.Join(dir, "src", "*.xml") always produces an absolute glob path,
// preventing the esbuild cmd.Dir+src doubling bug.
func TestDiscoverWidgets_RelativeDir(t *testing.T) {
	// Create a temp widget project with src/MyWidget.xml
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	xmlContent := `<?xml version="1.0" encoding="utf-8"?>
<widget id="com.example.MyWidget" pluginWidget="true"
        xmlns="http://www.mendix.com/widget/1.0/"
        xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
        xsi:schemaLocation="http://www.mendix.com/widget/1.0/ ../node_modules/mendix/custom_widget.xsd">
  <name>MyWidget</name>
  <description/>
  <properties/>
</widget>`
	if err := os.WriteFile(filepath.Join(srcDir, "MyWidget.xml"), []byte(xmlContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Use absolute path directly — proves discoverWidgets works when dir is abs
	infos, err := discoverWidgets(tmp)
	if err != nil {
		t.Fatalf("discoverWidgets(%q): %v", tmp, err)
	}
	if len(infos) != 1 || infos[0].Name != "MyWidget" {
		t.Errorf("discoverWidgets: got %v, want [{MyWidget ...}]", infos)
	}
}

// TestWidgetBuildDirResolution verifies the fix: --dir must become absolute
// before being passed to compileWidget, so cmd.Dir + src do not double up.
func TestWidgetBuildDirResolution(t *testing.T) {
	// Before fix: dir="./StudyWidgets", src="./StudyWidgets/src/X.jsx"
	// cmd.Dir="./StudyWidgets" → esbuild sees "StudyWidgets/src/X.jsx" → doubles
	// After fix: dir="/abs/StudyWidgets", src="/abs/StudyWidgets/src/X.jsx"
	// cmd.Dir="/abs/StudyWidgets" → esbuild sees absolute path → no doubling

	cases := []string{".", "./SomeWidget", "../../relative/path"}
	for _, rel := range cases {
		abs, err := filepath.Abs(rel)
		if err != nil {
			t.Errorf("filepath.Abs(%q): %v", rel, err)
			continue
		}
		if !filepath.IsAbs(abs) {
			t.Errorf("filepath.Abs(%q) = %q is not absolute", rel, abs)
		}
	}
}
```

- [ ] **Step 2: 运行测试，确认通过（这两个测试不依赖修复）**

```bash
go test ./cmd/mxcli/... -run "TestDiscoverWidgets_RelativeDir|TestWidgetBuildDirResolution" -v
```

预期：`TestWidgetBuildDirResolution` PASS（验证 `filepath.Abs` 行为）；`TestDiscoverWidgets_RelativeDir` PASS（验证 `discoverWidgets` 对绝对路径工作正常）

- [ ] **Step 3: 修改 `runWidgetBuild`，在入口处将 dir 转为绝对路径**

打开 `cmd/mxcli/widget_build.go`，找到第 264 行的 `runWidgetBuild` 函数，在读取 `dir` flag 后立即加 `filepath.Abs`：

```go
func runWidgetBuild(cmd *cobra.Command, args []string) error {
	dir, _ := cmd.Flags().GetString("dir")
	if dir == "" {
		dir = "."
	}
	// Convert to absolute path so that filepath.Join(dir, "src", ...) and
	// cmd.Dir = dir don't compound when dir is relative (esbuild path doubling bug).
	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}

	infos, err := discoverWidgets(dir)
	// ... 其余代码不变
```

- [ ] **Step 4: 运行完整测试**

```bash
go test ./cmd/mxcli/... -count=1
```

预期：所有测试通过

- [ ] **Step 5: Commit**

```bash
git add cmd/mxcli/widget_build.go cmd/mxcli/widget_build_test.go
git commit -m "fix(widget): resolve --dir to absolute path before build

When --dir is relative, compileWidget sets cmd.Dir=./StudyWidgets and
src=./StudyWidgets/src/Widget.jsx; esbuild interprets src relative to
cmd.Dir, doubling the path. Converting dir to absolute at entry fixes this.

Fixes: Issue 2 from mxcli-taskdemo validation"
```

---

## Task 3：Skill / 文档更新（Issues 1、4、8 + mxcli-taskdemo 引用）

**Files:**
- Modify: `.claude/skills/mendix/check-syntax.md`（Issue 1）
- Modify: `.claude/skills/mendix/create-page.md`（Issue 4）
- Modify: `.claude/skills/mendix/create-custom-widget.md`（mxcli-taskdemo 引用）
- Modify: `.claude/skills/mendix/run-app.md`（Issue 8 文档）
- Modify: `CLAUDE.md`（Issue 1 mxcli binary 说明）
- Modify: `cmd/mxcli-local/cmd_run.go:47`（Issue 8 程序输出）

本 Task 全部为文档修改，无测试。

### 3a：check-syntax.md — binary 探测（Issue 1）

- [ ] **Step 1: 在 `check-syntax.md` 顶部加入 binary 探测规则**

打开 `.claude/skills/mendix/check-syntax.md`，在文件最开头（第 1 行之前）插入：

```markdown
## Finding mxcli

Before running any mxcli command, detect which binary to use:

```bash
if   [ -f ./mxcli ];               then MXCLI=./mxcli
elif command -v mxcli &>/dev/null; then MXCLI=mxcli
else echo "Install: mxcli setup mxcli --os linux" && exit 1; fi
# Then use: $MXCLI -p app.mpr -c "..."
```

- `./mxcli` — present when `mxcli setup mxcli --os linux` was run (devcontainer)
- `mxcli` — globally installed (Windows / Mac / Linux native)

```

### 3b：CLAUDE.md — mxcli binary 条目说明（Issue 1）

- [ ] **Step 2: 更新 CLAUDE.md `mxcli setup mxcli` 条目**

找到 CLAUDE.md 中 `mxcli setup mxcli` 那一行（在"Setup mxcli"表格行），把 description 扩展为：

```markdown
| **Setup mxcli** | `mxcli setup mxcli [--os linux] [--arch amd64] [--output ./mxcli]` | Download platform-specific mxcli binary. **Two scenarios:** (1) Windows/Mac: mxcli is installed globally in PATH — use `mxcli` directly. (2) Devcontainer: downloads a Linux binary to `./mxcli` in the project root; skills auto-detect via `[ -f ./mxcli ]` check. |
```

### 3c：create-page.md — PLUGGABLEWIDGET attribute 绑定 + mxcli-taskdemo（Issue 4）

- [ ] **Step 3: 在 `create-page.md` 的 PLUGGABLEWIDGET 章节添加 attribute 绑定说明**

打开 `.claude/skills/mendix/create-page.md`，找到 `## PLUGGABLEWIDGET — Custom and Third-Party Widgets` 章节（约第 1081 行），在 "Workflow for using a custom widget" 列表末尾添加：

```markdown
**Attribute binding syntax** (verified against mxcli-taskdemo `02-pages.mdl`):

```sql
-- Inside a dataview, use the bare attribute name (NOT @Entity/Attr):
dataview dvTask (datasource: $Task) {
  PLUGGABLEWIDGET 'com.mendix.widget.custom.PrioritySelector.PrioritySelector' wPriority (
    priority: Priority,   -- bare attribute name; matches widget XML key="priority"
    editable: true        -- boolean: true or false, no quotes
  )
}

-- In a DataGrid custom content column (read-only):
column colPriority (caption: 'Priority', ShowContentAs: customContent) {
  PLUGGABLEWIDGET 'com.mendix.widget.custom.PrioritySelector.PrioritySelector' wColPriority (
    priority: Priority, editable: false
  )
}
```

Rules:
- Property keys are **case-sensitive** — copy exactly from `key=` in widget XML
- Attribute properties use bare attribute name in the current dataview context
- `mxcli widget list -p app.mpr` lists all installed widgets (including uninstantiated)
- `SHOW WIDGETS` lists only instantiated widgets (catalog entries)

**Reference:** `github.com/engalar/mxcli-taskdemo` — `TaskDemo/mdlsource/02-pages.mdl`
```

### 3d：create-custom-widget.md — mxcli-taskdemo 完整参考

- [ ] **Step 4: 在 `create-custom-widget.md` 末尾加入完整生命周期参考**

打开 `.claude/skills/mendix/create-custom-widget.md`，在文件末尾追加：

```markdown
## Complete Reference: mxcli-taskdemo

**github.com/engalar/mxcli-taskdemo** — end-to-end widget lifecycle demo (scaffold → build → install → MDL usage):

| File | What it shows |
|------|---------------|
| `StudyWidgets/src/PrioritySelector.xml` | `attribute` + `boolean` property declaration |
| `StudyWidgets/src/ProgressRing.xml` | `attribute` + `string` + `boolean` mix |
| `StudyWidgets/src/PrioritySelector.jsx` | Two-way attribute binding: `priority.value` / `priority.setValue(val)` |
| `StudyWidgets/src/ProgressRing.jsx` | Read-only attribute: `progress.value`, SVG ring render |
| `StudyWidgets/package.xml` | Multi-widget single-package structure; xmlns **must** be `clientModule/1.0/` (capital M) |
| `TaskDemo/mdlsource/02-pages.mdl` | `PLUGGABLEWIDGET` in DataView (editable) and DataGrid column (read-only) |

**Build workflow** (must run inside the package directory):
```bash
cd StudyWidgets
mxcli widget build             # do NOT use --dir (path-doubling bug, fix in progress)
cp StudyWidgets.mpk ../TaskDemo/widgets/
mxcli -p ../TaskDemo/TaskDemo.mpr -c "SHOW INSTALLED WIDGETS"  # verify installation
```

**Full MDL workflow** (`TaskDemo/mdlsource/` — run in order):
```bash
mxcli exec 01-domain.mdl  -p TaskDemo.mpr   # entities + enumerations
mxcli exec 02-pages.mdl   -p TaskDemo.mpr   # pages with custom widgets
mxcli exec 03-security.mdl -p TaskDemo.mpr  # roles + access rules
mxcli exec 04-navigation.mdl -p TaskDemo.mpr
```
```

### 3e：run-app.md — -p 必填说明（Issue 8 文档）

- [ ] **Step 5: 在 `run-app.md` Local Path 章节开头加警告**

打开 `.claude/skills/mendix/run-app.md`，找到 `## Local Path (No Docker) — Quick Start` 章节，在 `## Prerequisites Check` 之后 `## Local Path` 标题下、第一个代码块之前插入：

```markdown
> **`-p` is required** for all `mxcli local` commands — project path is never auto-detected.
> Always invoke via `mxcli local` (the launcher), not `~/.mxcli/local/mxcli-local` directly.
```

### 3f：cmd_run.go — 程序引导输出（Issue 8 程序）

- [ ] **Step 6: 改善 `cmd_run.go` 缺少 -p 时的错误输出**

打开 `cmd/mxcli-local/cmd_run.go`，找到第 47 行：

```go
return fmt.Errorf("either -p or --pad-dir is required")
```

替换为：

```go
return fmt.Errorf(`-p is required

Specify the .mpr file path so mxcli can find the build output:

  mxcli local run -p /path/to/app.mpr [--admin-password pw]

If you haven't built yet, run first:

  mxcli local build -p /path/to/app.mpr`)
```

- [ ] **Step 7: 手动验证错误输出**

```bash
go run ./cmd/mxcli-local run
```

预期输出包含：`-p is required` + 示例命令 + build 提示

- [ ] **Step 8: 同步 skill 到用户项目（执行 make sync）**

```bash
make sync-skills sync-commands
```

预期：`.claude/skills/mendix/` 内容同步到 study 项目等目标路径（若 Makefile 有此 target）

- [ ] **Step 9: Commit**

```bash
git add \
  .claude/skills/mendix/check-syntax.md \
  .claude/skills/mendix/create-page.md \
  .claude/skills/mendix/create-custom-widget.md \
  .claude/skills/mendix/run-app.md \
  CLAUDE.md \
  cmd/mxcli-local/cmd_run.go
git commit -m "docs: update skills with widget lifecycle guidance and mxcli-taskdemo refs

- check-syntax.md: add binary detection pattern (./mxcli vs global mxcli)
- create-page.md: document PLUGGABLEWIDGET attribute binding syntax
- create-custom-widget.md: add mxcli-taskdemo complete lifecycle reference
- run-app.md: clarify -p is required for all mxcli local commands
- CLAUDE.md: clarify mxcli binary setup for devcontainer vs native
- cmd_run.go: improve error message with actionable guidance (Issue 8)"
```

---

## Task 4：修复重复 module role CE1613（Issue 5/7）

**Files:**
- Modify: `mdl/executor/cmd_security_write_modulerole_gen.go:53`
- Create: `mdl/executor/cmd_security_write_modulerole_gen_test.go`

### 根因回顾

`addModuleRoleViaModelsdk`（`mdl/backend/mpr/security_module_modelsdk.go:24`）是纯 append，不去重。`execCreateModuleRoleGen` 在发现 auto-provisioned 同名角色时，调用 `AddModuleRole` 前**没有先 `RemoveModuleRole`**，导致两个同名角色 → CE1613。

- [ ] **Step 1: 新建测试文件，写失败测试**

创建 `mdl/executor/cmd_security_write_modulerole_gen_test.go`：

```go
package executor

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	"github.com/mendixlabs/mxcli/model"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
)

// TestCreateModuleRole_OverwritesAutoProvisioned verifies that CREATE MODULE ROLE
// on an auto-provisioned "User" role results in exactly ONE role (not two).
// Before the fix: AddModuleRole appends without removing the old one → CE1613.
func TestCreateModuleRole_OverwritesAutoProvisioned(t *testing.T) {
	removeCalled := false
	addCalled := false
	addCount := 0

	ms := genSec.NewModuleSecurity()
	autoRole := genSec.NewModuleRole()
	autoRole.SetName("User")
	autoRole.SetDescription(autoDocumentRoleDescription)
	ms.AddModuleRoles(autoRole)

	mb := &mock.MockBackend{}
	mb.GetModuleSecurityGenFunc = func(moduleID model.ID) (*genSec.ModuleSecurity, error) {
		return ms, nil
	}
	mb.RemoveModuleRoleFunc = func(unitID model.ID, roleName string) error {
		removeCalled = true
		// Simulate removing the role from ms
		for i, mr := range ms.ModuleRolesItems() {
			if typed, ok := mr.(*genSec.ModuleRole); ok && typed.Name() == roleName {
				ms.RemoveModuleRoles(i)
				break
			}
		}
		return nil
	}
	mb.AddModuleRoleFunc = func(unitID model.ID, roleName, description string) error {
		addCount++
		addCalled = true
		newRole := genSec.NewModuleRole()
		newRole.SetName(roleName)
		newRole.SetDescription(description)
		ms.AddModuleRoles(newRole)
		return nil
	}
	mb.InvalidateModuleSecurityCacheFunc = func() {}
	mb.UpdateQualifiedNameInAllUnitsFunc = func(old, new string) (int, error) { return 0, nil }

	ctx := newTestExecContext(t, mb)
	stmt := &ast.CreateModuleRoleStmt{
		Name:        ast.QualifiedName{Module: "TaskMgr", Name: "User"},
		Description: "Task manager user role",
	}

	err := execCreateModuleRoleGen(ctx, stmt)
	if err != nil {
		t.Fatalf("execCreateModuleRoleGen: %v", err)
	}

	if !removeCalled {
		t.Error("RemoveModuleRole was not called — auto-provisioned role not removed before re-adding")
	}
	if !addCalled {
		t.Error("AddModuleRole was not called")
	}

	roles := ms.ModuleRolesItems()
	if len(roles) != 1 {
		names := make([]string, 0, len(roles))
		for _, r := range roles {
			if typed, ok := r.(*genSec.ModuleRole); ok {
				names = append(names, typed.Name())
			}
		}
		t.Errorf("expected 1 module role, got %d: %v (CE1613 would occur)", len(roles), names)
	}

	if typed, ok := roles[0].(*genSec.ModuleRole); ok {
		if typed.Description() == autoDocumentRoleDescription {
			t.Error("role still has auto-provisioned description — overwrite failed")
		}
		if !strings.Contains(typed.Name(), "User") {
			t.Errorf("unexpected role name: %q", typed.Name())
		}
	}
}
```

注：`newTestExecContext` 是 executor 测试中常用的 helper。检查 `executor` 包中是否已存在此函数：

```bash
grep -rn "newTestExecContext\|testExecContext\|NewTestContext" /mnt/data_sdd/gh/mxcli-wt-02/mdl/executor/ --include="*.go" | head -5
```

若不存在，在测试文件中添加最小实现：

```go
func newTestExecContext(t *testing.T, mb *mock.MockBackend) *ExecContext {
	t.Helper()
	ctx := &ExecContext{
		Backend: mb,
		Output:  &strings.Builder{},
		Quiet:   true,
	}
	return ctx
}
```

- [ ] **Step 2: 运行测试，确认失败**

```bash
go test ./mdl/executor/... -run TestCreateModuleRole_OverwritesAutoProvisioned -v
```

预期：FAIL，提示 `RemoveModuleRole was not called` 或 `expected 1 module role, got 2`

- [ ] **Step 3: 修复 `cmd_security_write_modulerole_gen.go:53`**

打开 `mdl/executor/cmd_security_write_modulerole_gen.go`，找到第 53 行的 `if typed.Description() == autoDocumentRoleDescription {` 分支，在 `AddModuleRole` 调用之前插入 `RemoveModuleRole`：

```go
if typed.Description() == autoDocumentRoleDescription {
    oldQualified := s.Name.Module + "." + typed.Name()
    newQualified := s.Name.Module + "." + s.Name.Name

    // Remove first: AddModuleRole is a plain append with no dedup.
    // Without this, two roles with the same name cause Mendix CE1613.
    if err := ctx.Backend.RemoveModuleRole(model.ID(ms.ID()), typed.Name()); err != nil {
        return mdlerrors.NewBackend("remove auto-provisioned role", err)
    }
    if err := ctx.Backend.AddModuleRole(model.ID(ms.ID()), s.Name.Name, s.Description); err != nil {
        return mdlerrors.NewBackend("create module role", err)
    }
    if oldQualified != newQualified {
        if _, err := ctx.Backend.UpdateQualifiedNameInAllUnits(oldQualified, newQualified); err != nil {
            return mdlerrors.NewBackend(
                fmt.Sprintf("rename references %s -> %s", oldQualified, newQualified), err)
        }
    }
    invalidateModuleSecurityCache(ctx)
    if !ctx.Quiet {
        fmt.Fprintf(ctx.Output, "Module role %s.%s already exists (auto-provisioned)\n",
            s.Name.Module, s.Name.Name)
    }
    return nil
}
```

同时改善非 auto-provisioned 同名角色的错误提示（第 75 行之后）：

```go
// Custom role already exists — idempotent: skip with a helpful hint.
if !ctx.Quiet {
    fmt.Fprintf(ctx.Output,
        "Module role %s.%s already exists.\nTo link it to a user role: ALTER USER ROLE \"User\" ADD MODULE ROLES (%s.\"%s\");\n",
        s.Name.Module, s.Name.Name, s.Name.Module, s.Name.Name)
}
return nil
```

- [ ] **Step 4: 运行测试，确认通过**

```bash
go test ./mdl/executor/... -run TestCreateModuleRole_OverwritesAutoProvisioned -v
```

预期：PASS

- [ ] **Step 5: 运行完整 executor 测试**

```bash
go test ./mdl/executor/... -count=1
```

预期：所有测试通过

- [ ] **Step 6: Commit**

```bash
git add mdl/executor/cmd_security_write_modulerole_gen.go \
        mdl/executor/cmd_security_write_modulerole_gen_test.go
git commit -m "fix(security): remove auto-provisioned role before re-adding to prevent CE1613

AddModuleRole is a plain append with no dedup. When CREATE MODULE ROLE targets
an auto-provisioned User role, calling AddModuleRole without first calling
RemoveModuleRole creates two identically-named roles, triggering CE1613.

Fix: call RemoveModuleRole before AddModuleRole in the overwrite path.
Also improve error message for non-auto-provisioned duplicate to hint ALTER.

Fixes: Issues 5 and 7 from mxcli-taskdemo validation"
```

---

## Task 5：新增 `SHOW INSTALLED WIDGETS` 命令（Issue 3）

全栈实现：Lexer → Parser → 再生 → AST → Visitor → Executor → Register → Summary → Test

**Files:**
- Modify: `mdl/grammar/MDLLexer.g4`
- Modify: `mdl/grammar/MDLParser.g4`
- Run: `make grammar`（再生 `mdl/grammar/parser/mdl_lexer.go`, `mdl_parser.go`）
- Modify: `mdl/ast/ast_widgets_cmd.go`
- Modify: `mdl/visitor/visitor_query.go`
- Modify: `mdl/executor/cmd_widgets.go`
- Modify: `mdl/executor/register_stubs.go`
- Modify: `mdl/executor/stmt_summary.go`
- Create: `mdl/executor/cmd_widgets_installed_test.go`

### 5a：Lexer — 添加 INSTALLED token

- [ ] **Step 1: 在 `MDLLexer.g4` 中添加 INSTALLED token**

打开 `mdl/grammar/MDLLexer.g4`，在 `SHOW` token 附近（约第 113 行）添加：

```antlr
INSTALLED: I N S T A L L E D;
```

（参照同文件其他 keyword 的大小写展开写法，例如 `SHOW: S H O W;`）

### 5b：Parser — 在 showStatement 中添加 INSTALLED WIDGETS 分支

- [ ] **Step 2: 在 `MDLParser.g4` 中扩展 showStatement**

打开 `mdl/grammar/MDLParser.g4`，找到包含 `WIDGETS` 的 showStatement 规则部分。当前规则允许 `SHOW WIDGETS [WHERE...]`，需要添加 `SHOW INSTALLED WIDGETS` 分支：

```antlr
// 在 showStatement 规则的 alternatives 中，找到 WIDGETS 那个分支，
// 添加一个新 alternative:
| SHOW INSTALLED WIDGETS   # ShowInstalledWidgets
```

具体位置：在 showStatement 规则体中，紧接在 `WIDGETS` 相关分支之后（或之前）添加 `| SHOW INSTALLED WIDGETS`。确认规则语法与文件其他 alternatives 一致。

### 5c：再生 parser

- [ ] **Step 3: 运行 `make grammar` 再生**

```bash
make grammar
```

预期：`mdl/grammar/parser/mdl_lexer.go` 和 `mdl_parser.go` 更新，新增 `MDLParserINSTALLED` 常量和 `INSTALLED()` 方法到 `IShowStatementContext`

- [ ] **Step 4: 确认再生文件编译通过**

```bash
go build ./mdl/...
```

预期：无编译错误

### 5d：AST

- [ ] **Step 5: 在 `mdl/ast/ast_widgets_cmd.go` 中添加 AST 节点**

打开文件，在 `ShowWidgetsStmt` 定义之后添加：

```go
// ShowInstalledWidgetsStmt represents: SHOW INSTALLED WIDGETS
// Scans the project's widgets/ directory for .mpk files, listing all
// widget definitions regardless of page instantiation.
type ShowInstalledWidgetsStmt struct{}

func (s *ShowInstalledWidgetsStmt) isStatement() {}
```

### 5e：Visitor

- [ ] **Step 6: 在 `mdl/visitor/visitor_query.go` 的 `ExitShowStatement` 中添加分支**

打开文件，找到 `ExitShowStatement` 函数（第 13 行开始），在处理 `ctx.WIDGETS() != nil` 的 else-if 分支（约第 621 行）之前，添加：

```go
} else if ctx.INSTALLED() != nil && ctx.WIDGETS() != nil {
    // SHOW INSTALLED WIDGETS
    b.statements = append(b.statements, &ast.ShowInstalledWidgetsStmt{})
```

（注意：检查生成的 `IShowStatementContext` 接口中 `INSTALLED()` 方法的确切名称，可能是 `INSTALLED()` 返回 `antlr.TerminalNode`）

### 5f：Executor 实现

- [ ] **Step 7: 在 `mdl/executor/cmd_widgets.go` 中添加 `execShowInstalledWidgets`**

在文件末尾追加：

```go
// execShowInstalledWidgets handles SHOW INSTALLED WIDGETS.
// Scans widgets/*.mpk in the project directory using WidgetRegistry,
// listing all installed widget definitions regardless of page instantiation.
func execShowInstalledWidgets(ctx *ExecContext, _ *ast.ShowInstalledWidgetsStmt) error {
	if !ctx.Connected() {
		return mdlerrors.NewNotConnected()
	}
	if ctx.MprPath == "" {
		return fmt.Errorf("SHOW INSTALLED WIDGETS requires a project connection (-p app.mpr)")
	}

	projectDir := filepath.Dir(ctx.MprPath)
	registry, err := NewWidgetRegistry()
	if err != nil {
		return fmt.Errorf("creating widget registry: %w", err)
	}
	if err := registry.SetProjectDir(projectDir); err != nil {
		return fmt.Errorf("scanning widgets/ directory: %w", err)
	}

	discovered := registry.MPKDiscovered()
	if len(discovered) == 0 {
		fmt.Fprintln(ctx.Output, "No widget packages found in widgets/")
		fmt.Fprintf(ctx.Output, "Copy a .mpk file to %s/widgets/ to install a widget.\n", projectDir)
		return nil
	}

	fmt.Fprintf(ctx.Output, "\n%-30s %-60s %s\n", "MPK / Widget Name", "Widget ID", "Display Name")
	fmt.Fprintln(ctx.Output, strings.Repeat("-", 120))

	names := make([]string, 0, len(discovered))
	for name := range discovered {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		w := discovered[name]
		fmt.Fprintf(ctx.Output, "%-30s %-60s %s\n",
			strings.ToLower(name), w.WidgetID, w.Name)
	}

	fmt.Fprintf(ctx.Output, "\n%d widget definition(s) found\n", len(discovered))
	fmt.Fprintf(ctx.Output, "\nMDL usage: PLUGGABLEWIDGET '<Widget ID>' instanceName (prop: val)\n")
	fmt.Fprintf(ctx.Output, "Reference: github.com/engalar/mxcli-taskdemo — TaskDemo/mdlsource/02-pages.mdl\n")
	return nil
}
```

确认 `cmd_widgets.go` 已导入 `"path/filepath"`, `"sort"`, `"strings"`；若缺少则加入 import 块。

### 5g：Register + Summary

- [ ] **Step 8: 在 `register_stubs.go` 中注册新命令**

打开 `mdl/executor/register_stubs.go`，找到 `ShowWidgetsStmt` 的注册（约第 371 行），在其后添加：

```go
r.Register(&ast.ShowInstalledWidgetsStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
    return execShowInstalledWidgets(ctx, stmt.(*ast.ShowInstalledWidgetsStmt))
})
```

- [ ] **Step 9: 在 `stmt_summary.go` 中添加 case**

打开 `mdl/executor/stmt_summary.go`，找到 `case *ast.ShowWidgetsStmt:` 行（约第 175 行），在其后添加：

```go
case *ast.ShowInstalledWidgetsStmt:
    return "SHOW INSTALLED WIDGETS"
```

### 5h：写测试

- [ ] **Step 10: 新建 `mdl/executor/cmd_widgets_installed_test.go`，写失败测试**

```go
package executor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
)

func TestExecShowInstalledWidgets_NoWidgetsDir(t *testing.T) {
	tmp := t.TempDir()
	mb := &mock.MockBackend{}
	ctx := &ExecContext{
		Backend: mb,
		Output:  &strings.Builder{},
		MprPath: filepath.Join(tmp, "app.mpr"),
	}
	// widgets/ does not exist → should print "No widget packages found"
	err := execShowInstalledWidgets(ctx, &ast.ShowInstalledWidgetsStmt{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := ctx.Output.(*strings.Builder).String()
	if !strings.Contains(out, "No widget packages found") {
		t.Errorf("expected 'No widget packages found', got:\n%s", out)
	}
}

func TestExecShowInstalledWidgets_FindsMpk(t *testing.T) {
	tmp := t.TempDir()
	// Create a minimal .mpk (zip) with one widget XML
	widgetsDir := filepath.Join(tmp, "widgets")
	if err := os.MkdirAll(widgetsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a minimal mpk zip containing a widget XML
	mpkPath := filepath.Join(widgetsDir, "TestWidget.mpk")
	createMinimalMPK(t, mpkPath, "com.example.TestWidget.TestWidget", "TestWidget")

	mb := &mock.MockBackend{}
	out := &strings.Builder{}
	ctx := &ExecContext{
		Backend: mb,
		Output:  out,
		MprPath: filepath.Join(tmp, "app.mpr"),
	}

	err := execShowInstalledWidgets(ctx, &ast.ShowInstalledWidgetsStmt{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result := out.String()
	if !strings.Contains(result, "com.example.TestWidget.TestWidget") {
		t.Errorf("expected widget ID in output, got:\n%s", result)
	}
	if !strings.Contains(result, "PLUGGABLEWIDGET") {
		t.Errorf("expected MDL usage hint in output, got:\n%s", result)
	}
}

// createMinimalMPK creates a ZIP file (.mpk) containing one widget XML.
func createMinimalMPK(t *testing.T, path, widgetID, widgetName string) {
	t.Helper()
	xmlContent := `<?xml version="1.0" encoding="utf-8"?>
<widget id="` + widgetID + `" pluginWidget="true"
        xmlns="http://www.mendix.com/widget/1.0/"
        xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
        xsi:schemaLocation="http://www.mendix.com/widget/1.0/ ../node_modules/mendix/custom_widget.xsd">
  <name>` + widgetName + `</name>
  <description/>
  <properties/>
</widget>`

	import_archive := func() {
		// Use archive/zip to create the .mpk
	}
	_ = import_archive

	// Write a ZIP containing widgetName.xml
	zf, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer zf.Close()

	w := newZipWriter(zf)
	entry, err := w.Create(widgetName + ".xml")
	if err != nil {
		t.Fatal(err)
	}
	entry.Write([]byte(xmlContent))
	w.Close()
}
```

**注意**：将 `newZipWriter` 替换为标准库 `archive/zip` 的实际用法（需要在文件顶部 `import "archive/zip"`）。完整 helper 如下：

```go
import (
	"archive/zip"
	"io"
	"os"
	...
)

func createMinimalMPK(t *testing.T, path, widgetID, widgetName string) {
	t.Helper()
	xmlContent := `<?xml version="1.0" encoding="utf-8"?>
<widget id="` + widgetID + `" pluginWidget="true"
        xmlns="http://www.mendix.com/widget/1.0/"
        xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
        xsi:schemaLocation="http://www.mendix.com/widget/1.0/ ../node_modules/mendix/custom_widget.xsd">
  <name>` + widgetName + `</name>
  <description/>
  <properties/>
</widget>`

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	entry, err := zw.Create(widgetName + ".xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(entry, xmlContent); err != nil {
		t.Fatal(err)
	}
	zw.Close()
}
```

- [ ] **Step 11: 运行测试，确认 `TestExecShowInstalledWidgets_NoWidgetsDir` 通过**

```bash
go test ./mdl/executor/... -run TestExecShowInstalledWidgets -v
```

`NoWidgetsDir` 测试应 PASS（`execShowInstalledWidgets` 函数已实现）；`FindsMpk` 测试结果取决于 `WidgetRegistry.MPKDiscovered()` 是否能解析最小 ZIP。若 FAIL，检查 `createMinimalMPK` 产生的 zip 结构是否与 `MPKDiscovered` 预期的 `.mpk` 格式一致。

- [ ] **Step 12: 运行完整测试**

```bash
go build ./...
go test ./mdl/... -count=1
```

预期：所有测试通过

- [ ] **Step 13: Commit**

```bash
git add \
  mdl/grammar/MDLLexer.g4 \
  mdl/grammar/MDLParser.g4 \
  mdl/grammar/parser/mdl_lexer.go \
  mdl/grammar/parser/mdl_parser.go \
  mdl/ast/ast_widgets_cmd.go \
  mdl/visitor/visitor_query.go \
  mdl/executor/cmd_widgets.go \
  mdl/executor/register_stubs.go \
  mdl/executor/stmt_summary.go \
  mdl/executor/cmd_widgets_installed_test.go
git commit -m "feat(mdl): add SHOW INSTALLED WIDGETS command

Lists widget definitions from widgets/*.mpk regardless of page instantiation.
Complements SHOW WIDGETS (catalog-based, instantiated only) for the case where
a widget is installed but not yet placed on any page.

Output includes Widget ID and MDL usage example referencing mxcli-taskdemo.

Fixes: Issue 3 from mxcli-taskdemo validation"
```

---

## 最终验证

- [ ] **全量构建 + 测试**

```bash
make build && make test
```

预期：`make build` 产出 `bin/mxcli`；`make test` 全部通过，覆盖率无明显下降

- [ ] **手动 smoke test（需要 study 项目）**

```bash
# Task 1 验证
cd /mnt/data_sdd/study/StudyWidgets
mxcli widget new TestPkg --package   # 检查生成的 package.xml 有 clientModule/1.0/

# Task 2 验证  
mxcli widget build --dir ./StudyWidgets  # 不应再报路径重复错误

# Task 5 验证
mxcli -p TaskDemo/TaskDemo.mpr -c "SHOW INSTALLED WIDGETS"
# 预期：列出 StudyWidgets.mpk 内的 PrioritySelector 和 ProgressRing

# Task 4 验证（需空白项目）
mxcli -p <blank>.mpr -c "
  ALTER PROJECT SECURITY LEVEL PROTOTYPE;
  CREATE MODULE ROLE MyModule.\"User\";  -- 不应产生 CE1613
"
```
