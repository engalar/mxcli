# Issue 003: DESCRIBE MICROFLOW 保留字未加引号修复计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复两个相关 bug：(1) `TEMPLATE`、`ROW`、`ITEM` 等 widget 类型 token 不在 `keyword` 规则里，导致 `ExcelImporter.Template` 等类型名作为 qualifiedName 解析失败；(2) DESCRIBE MICROFLOW 输出的参数名如果是保留字，未加反引号引号，导致 roundtrip 失败。

**Architecture:** 双修复策略：先在 grammar 层把缺失的 token 加入 `keyword` 规则（让 qualifiedName 可以接受这些 token），再在 executor 层在输出参数名时做保留字检测并加反引号。两处修复互补，解决不同场景下的失败。

**Tech Stack:** ANTLR4 grammar (MDLSettings.g4), Go (cmd_microflows_show_gen.go)

---

### Task 1: 把缺失 widget token 加入 `keyword` 规则

grammar 文件的 `keyword` 规则（MDLSettings.g4）已包含大多数 widget 类型 token，但 `TEMPLATE`、`ROW`、`ITEM`、`FILTER` 缺席。这导致 `Module.Template` 等类型名无法作为 `qualifiedName` 解析。

**Files:**
- Modify: `mdl/grammar/domains/MDLSettings.g4`

- [ ] **Step 1: 定位 keyword 规则中 widget 类型 token 的插入点**

打开 `mdl/grammar/domains/MDLSettings.g4`，找到包含 `ACTIONBUTTON | CHECKBOX | COMBOBOX` 的 widget types 部分（约第 525 行）：

```antlr
    // Widget types
    | ACTIONBUTTON | CHECKBOX | COMBOBOX | CONTAINER | CONTROLBAR
    | CUSTOMCONTAINER | CUSTOMWIDGET | DATAGRID | DATEPICKER | DATAVIEW
    | DATEFILTER | DROPDOWN | DROPDOWNFILTER | DROPDOWNSORT | DYNAMICTEXT
    | FILEINPUT | GALLERY | GROUPBOX | IMAGE | IMAGEINPUT
    | INPUTREFERENCESETSELECTOR | LAYOUTGRID | LINKBUTTON | LISTVIEW
    | NAVIGATIONLIST | NUMBERFILTER | PLACEHOLDER | PLUGGABLEWIDGET
    | RADIOBUTTONS | REFERENCESELECTOR | SEARCHBAR | SNIPPETCALL
    | STATICIMAGE | STATICTEXT | DYNAMICIMAGE | TEXTAREA | TEXTBOX | TEXTFILTER
    | TABCONTAINER | TABPAGE | WIDGET | WIDGETS
```

- [ ] **Step 2: 添加缺失的 token**

在该块末尾添加缺失的 widget type token：

```antlr
    // Widget types
    | ACTIONBUTTON | CHECKBOX | COMBOBOX | CONTAINER | CONTROLBAR
    | CUSTOMCONTAINER | CUSTOMWIDGET | DATAGRID | DATEPICKER | DATAVIEW
    | DATEFILTER | DROPDOWN | DROPDOWNFILTER | DROPDOWNSORT | DYNAMICTEXT
    | FILEINPUT | GALLERY | GROUPBOX | IMAGE | IMAGEINPUT | ITEM
    | INPUTREFERENCESETSELECTOR | LAYOUTGRID | LINKBUTTON | LISTVIEW
    | NAVIGATIONLIST | NUMBERFILTER | PLACEHOLDER | PLUGGABLEWIDGET
    | RADIOBUTTONS | REFERENCESELECTOR | SEARCHBAR | SNIPPETCALL
    | STATICIMAGE | STATICTEXT | DYNAMICIMAGE | TEXTAREA | TEXTBOX | TEXTFILTER
    | TABCONTAINER | TABPAGE | TEMPLATE | WIDGET | WIDGETS | ROW | FILTER
```

（新增：`ITEM`、`TEMPLATE`、`ROW`、`FILTER`，其中 `FILTER` 已在 widgetTypeV3 中使用）

- [ ] **Step 3: 重新生成 parser**

```bash
cd /mnt/data_sdd/gh/mxcli/.claude/worktrees/dev-fix
make grammar
```

Expected: 无报错，`mdl/grammar/parser/` 下文件更新。

- [ ] **Step 4: 验证 build**

```bash
make build 2>&1 | tail -5
```

Expected: `build finished successfully` 或无新增错误。

- [ ] **Step 5: Commit**

```bash
git add mdl/grammar/domains/MDLSettings.g4
git commit -m "fix(grammar): add TEMPLATE/ROW/ITEM/FILTER to keyword rule for qualifiedName use"
```

---

### Task 2: DESCRIBE MICROFLOW 输出保留字参数名时加反引号

即使 Task 1 修复了类型名的解析问题，当参数名本身是保留字时（如 `Template` 直接作为 parameterName 而非 `$Template` VARIABLE），仍需要引号。加入保留字检测函数，输出时自动加反引号。

**Files:**
- Modify: `mdl/executor/cmd_microflows_show_gen.go`
- Test: `mdl/executor/cmd_microflows_show_gen_test.go` (新建文件)

- [ ] **Step 1: 写失败测试**

新建文件 `mdl/executor/cmd_microflows_show_gen_test.go`：

```go
package executor

import "testing"

func TestQuoteIfReserved_Reserved(t *testing.T) {
	for _, kw := range []string{"Template", "Attribute", "Column", "List", "Row", "Item"} {
		got := quoteIfReserved(kw)
		want := "`" + kw + "`"
		if got != want {
			t.Errorf("quoteIfReserved(%q) = %q, want %q", kw, got, want)
		}
	}
}

func TestQuoteIfReserved_Plain(t *testing.T) {
	for _, plain := range []string{"ImportData", "FileContent", "OrderLine"} {
		got := quoteIfReserved(plain)
		if got != plain {
			t.Errorf("quoteIfReserved(%q) = %q, want %q", plain, got, plain)
		}
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

```bash
cd /mnt/data_sdd/gh/mxcli/.claude/worktrees/dev-fix
~/go1.26/bin/go test ./mdl/executor/ -run TestQuoteIfReserved -v 2>&1 | tail -10
```

Expected: `FAIL` — `quoteIfReserved` undefined.

- [ ] **Step 3: 实现 quoteIfReserved 和修改输出**

在 `mdl/executor/cmd_microflows_show_gen.go` 中，在文件顶部（imports 之后）添加保留字集合和检测函数，然后修改第 144 行的参数输出：

**在 `genMicroflowParameters` 函数之前添加：**

```go
// mdlReservedWords is the set of MDL lexer token names (lowercase) that are
// keywords and would fail to parse as bare identifiers in parameterName context.
// Subset of keyword rule in MDLSettings.g4.
var mdlReservedWords = func() map[string]struct{} {
	words := []string{
		"template", "attribute", "attributes", "column", "columns",
		"list", "row", "item", "filter",
		"select", "from", "where", "group", "order", "join",
		"create", "alter", "drop", "insert", "delete", "update",
		"return", "returns", "begin", "end", "if", "else", "then",
		"call", "declare", "commit", "rollback", "loop",
		"module", "entity", "association", "enumeration",
	}
	m := make(map[string]struct{}, len(words))
	for _, w := range words {
		m[w] = struct{}{}
	}
	return m
}()

// quoteIfReserved wraps name in backticks if it matches an MDL reserved word.
// Used when outputting parameter names in DESCRIBE MICROFLOW to ensure roundtrip.
func quoteIfReserved(name string) string {
	if _, ok := mdlReservedWords[strings.ToLower(name)]; ok {
		return "`" + name + "`"
	}
	return name
}
```

**修改第 144 行：**

原：
```go
lines = append(lines, fmt.Sprintf("  $%s: %s%s", p.name, p.declType, comma))
```

改为：
```go
lines = append(lines, fmt.Sprintf("  $%s: %s%s", quoteIfReserved(p.name), p.declType, comma))
```

- [ ] **Step 4: 确认需要 strings import**

检查文件头 import 是否已包含 `"strings"`；若无则添加。

```bash
grep '"strings"' /mnt/data_sdd/gh/mxcli/.claude/worktrees/dev-fix/mdl/executor/cmd_microflows_show_gen.go
```

若无输出，在 import 块中加入 `"strings"`。

- [ ] **Step 5: 运行测试确认通过**

```bash
~/go1.26/bin/go test ./mdl/executor/ -run TestQuoteIfReserved -v 2>&1 | tail -10
```

Expected: `PASS`.

- [ ] **Step 6: 运行全部 executor 测试**

```bash
~/go1.26/bin/go test ./mdl/executor/ -count=1 2>&1 | tail -10
```

Expected: 无新增 FAIL。

- [ ] **Step 7: Commit**

```bash
git add mdl/executor/cmd_microflows_show_gen.go mdl/executor/cmd_microflows_show_gen_test.go
git commit -m "fix(describe): quote reserved-word parameter names in DESCRIBE MICROFLOW output"
```

---

## 自检

- [ ] **Spec 覆盖：** 保留字类型名（`ExcelImporter.Template`）→ Task 1 修复；保留字参数名（`$Template`）→ Task 2 修复。
- [ ] **Placeholder 扫描：** 无 TBD。
- [ ] **类型一致性：** `quoteIfReserved` 在 Task 2 Step 3 定义、在 Step 3 引用，一致。
