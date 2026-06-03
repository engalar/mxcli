# Issue 009: FILEMANAGER (FILEINPUT) Widget MDL 支持计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 MDL `CREATE PAGE` 语法中支持 `fileinput` widget（对应 Mendix `Pages$FileManager` BSON 类型），让用户能够在文件上传场景（如 Excel 导入）中不必离开 mxcli 工作流。

**Architecture:** `FILEINPUT` 和 `IMAGEINPUT` token 已在 `MDLLexer.g4`（lines 279-280）中定义，且已在 `keyword` 规则中。唯一缺失的是：(1) 将 `FILEINPUT` 加入 `widgetTypeV3` grammar 规则；(2) 在 `cmd_pages_builder_v3.go` 中添加 `"fileinput"` case 和对应的 `buildFileManagerV3()` 函数，使用 `genPg.NewFileManager()` 构造 BSON。

**Tech Stack:** ANTLR4 grammar, Go (`mdl/executor/cmd_pages_builder_v3.go`), gen types (`modelsdk/gen/pages`)

**MDL 目标语法：**

```sql
create page MyModule.ImportPage layout MyFirstModule.Atlas_Default {
  dataview {
    entity: System.FileDocument
    content: {
      fileinput FileInput1 (
        allowed-extensions: 'xlsx,xls',
        label: 'Upload Excel File'
      )
    }
  }
}
```

---

### Task 1: 将 FILEINPUT 加入 widgetTypeV3 grammar 规则

**Files:**
- Modify: `mdl/grammar/domains/MDLPage.g4` (widgetTypeV3 规则，约第 204-248 行)

- [ ] **Step 1: 确认 widgetTypeV3 当前末尾**

```bash
sed -n '204,250p' /mnt/data_sdd/gh/mxcli/.claude/worktrees/dev-fix/mdl/grammar/domains/MDLPage.g4
```

期望看到 `| GROUPBOX` 是最后一行，后面是 `;`。

- [ ] **Step 2: 添加 FILEINPUT 和 IMAGEINPUT**

在 `widgetTypeV3` 规则的 `| GROUPBOX` 行之后、`;` 之前，插入：

```antlr
    | FILEINPUT
    | IMAGEINPUT
```

修改后规则末尾应为：

```antlr
    | CUSTOMCONTAINER
    | TABCONTAINER
    | TABPAGE
    | GROUPBOX
    | FILEINPUT
    | IMAGEINPUT
    ;
```

- [ ] **Step 3: 重新生成 parser**

```bash
cd /mnt/data_sdd/gh/mxcli/.claude/worktrees/dev-fix
make grammar
```

Expected: 无报错。

- [ ] **Step 4: 验证 build**

```bash
make build 2>&1 | tail -5
```

Expected: 无新增错误。

- [ ] **Step 5: Commit**

```bash
git add mdl/grammar/domains/MDLPage.g4
git commit -m "feat(grammar): add FILEINPUT/IMAGEINPUT to widgetTypeV3 rule"
```

---

### Task 2: 实现 buildFileManagerV3 函数

**Files:**
- Modify: `mdl/executor/cmd_pages_builder_v3.go`
- Test: `mdl/executor/cmd_pages_builder_v3_test.go`

#### 2a. 写失败测试

- [ ] **Step 1: 写失败测试**

在 `mdl/executor/cmd_pages_builder_v3_test.go` 末尾追加（或新建测试文件 `mdl/executor/cmd_pages_builder_fileinput_test.go`）：

```go
package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

// TestBuildFileManagerV3_Basic verifies that a fileinput widget builds to a
// Pages$FileManager element with the correct properties set.
func TestBuildFileManagerV3_Basic(t *testing.T) {
	pb := &pageBuilder{
		widgetNames: map[string]struct{}{},
	}

	w := &ast.WidgetV3{
		Type: "fileinput",
		Name: "FileInput1",
		Properties: []ast.WidgetPropertyV3{
			{Key: "allowed-extensions", Value: ast.StringExprV3{Literal: "xlsx,xls"}},
			{Key: "label", Value: ast.StringExprV3{Literal: "Upload Excel File"}},
		},
	}

	elem, err := pb.buildFileManagerV3(w)
	if err != nil {
		t.Fatalf("buildFileManagerV3 returned error: %v", err)
	}
	if elem == nil {
		t.Fatal("buildFileManagerV3 returned nil element")
	}
	if elem.TypeName() != "Pages$FileManager" {
		t.Errorf("TypeName = %q, want Pages$FileManager", elem.TypeName())
	}
}
```

**注意：** `ast.WidgetPropertyV3` 和 `ast.StringExprV3` 的确切结构需要对照 `mdl/ast/` 中的实际定义调整。先运行 grep 确认：

```bash
grep -n "WidgetPropertyV3\|StringExprV3\|WidgetV3\b" \
  /mnt/data_sdd/gh/mxcli/.claude/worktrees/dev-fix/mdl/ast/*.go | head -15
```

用实际字段名修改测试。

- [ ] **Step 2: 运行测试确认失败**

```bash
cd /mnt/data_sdd/gh/mxcli/.claude/worktrees/dev-fix
~/go1.26/bin/go test ./mdl/executor/ -run TestBuildFileManagerV3_Basic -v 2>&1 | tail -15
```

Expected: `FAIL` — `buildFileManagerV3` undefined 或字段名错误。

#### 2b. 实现 buildFileManagerV3

- [ ] **Step 3: 查看 GetLabel / GetProp 等辅助函数的签名**

```bash
grep -n "func.*GetLabel\|func.*GetProp\|func.*propStr\b" \
  /mnt/data_sdd/gh/mxcli/.claude/worktrees/dev-fix/mdl/executor/cmd_pages_builder_v3.go | head -10
grep -n "func.*GetAttribute\b" \
  /mnt/data_sdd/gh/mxcli/.claude/worktrees/dev-fix/mdl/executor/cmd_pages_builder_v3.go | head -5
```

- [ ] **Step 4: 实现 buildFileManagerV3 函数**

在 `cmd_pages_builder_v3.go` 中，紧接在 `buildDatePickerV3` 函数之后添加（约第 2563 行之后）：

```go
// buildFileManagerV3 builds a Pages$FileManager widget.
// MDL: fileinput WidgetName (allowed-extensions: 'xlsx,xls', label: 'Upload')
func (pb *pageBuilder) buildFileManagerV3(w *ast.WidgetV3) (element.Element, error) {
	fm := genPg.NewFileManager()
	assignFreshID(fm)
	fm.SetName(w.Name)
	applyFormWidgetDefaults(fm)

	if ext := w.GetStringProp("allowed-extensions"); ext != "" {
		fm.SetAllowedExtensions(ext)
	}

	if label := w.GetLabel(); label != "" {
		fm.SetLabel(genSimpleLabel(label))
	}

	if err := pb.registerWidgetName(w.Name, model.ID(fm.ID())); err != nil {
		return nil, err
	}

	return fm, nil
}
```

**注意：** 若 `WidgetV3` 没有 `GetStringProp` 方法，参考 `buildTextBoxV3` 中的属性读取模式（可能是 `w.GetProp("allowed-extensions")` 或直接遍历 `w.Properties`）。先 grep 确认：

```bash
grep -n "func.*GetStringProp\|w\.GetProp\|w\.GetLabel" \
  /mnt/data_sdd/gh/mxcli/.claude/worktrees/dev-fix/mdl/executor/cmd_pages_builder_v3.go | head -10
```

用实际方法名修改上述代码。

- [ ] **Step 5: 确认 genPg import**

检查文件顶部是否已有 `genPg` import：

```bash
grep "genPg\|gen/pages" /mnt/data_sdd/gh/mxcli/.claude/worktrees/dev-fix/mdl/executor/cmd_pages_builder_v3.go | head -3
```

若已有则跳过；若无则在 import 块中加入：

```go
genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
```

- [ ] **Step 6: 运行测试确认通过**

```bash
~/go1.26/bin/go test ./mdl/executor/ -run TestBuildFileManagerV3_Basic -v 2>&1 | tail -10
```

Expected: `PASS`.

- [ ] **Step 7: Commit**

```bash
git add mdl/executor/cmd_pages_builder_v3.go
git commit -m "feat(pages): implement buildFileManagerV3 for FILEINPUT widget"
```

---

### Task 3: 将 "fileinput" 注册到 builder switch

**Files:**
- Modify: `mdl/executor/cmd_pages_builder_v3.go` (dispatch switch，约第 365-430 行)

- [ ] **Step 1: 找到 dispatch switch 中 "textbox" case**

```bash
grep -n '"textbox"\|"checkbox"\|"datepicker"' \
  /mnt/data_sdd/gh/mxcli/.claude/worktrees/dev-fix/mdl/executor/cmd_pages_builder_v3.go | head -5
```

- [ ] **Step 2: 在 switch 中添加 "fileinput" case**

在 `"checkbox"` case 之后添加：

```go
	case "fileinput":
		widget, err = pb.buildFileManagerV3(w)
```

- [ ] **Step 3: 运行全部 builder 测试**

```bash
~/go1.26/bin/go test ./mdl/executor/ -run TestBuildPage -v 2>&1 | tail -20
~/go1.26/bin/go test ./mdl/executor/ -count=1 2>&1 | tail -5
```

Expected: 无新增 FAIL。

- [ ] **Step 4: Commit**

```bash
git add mdl/executor/cmd_pages_builder_v3.go
git commit -m "feat(pages): dispatch fileinput widget type to buildFileManagerV3"
```

---

### Task 4: 添加 MDL 集成测试示例

**Files:**
- Create: `mdl-examples/doctype-tests/page-fileinput.mdl`

- [ ] **Step 1: 创建测试脚本**

```bash
cat > /mnt/data_sdd/gh/mxcli/.claude/worktrees/dev-fix/mdl-examples/doctype-tests/page-fileinput.mdl << 'EOF'
-- Test: FILEINPUT (FileManager) widget in a page
-- Expected: creates a valid page with file upload widget

create page MyFirstModule.TestImportPage layout MyFirstModule.Atlas_Default {
  dataview {
    entity: System.FileDocument
    content: {
      fileinput FileInput1 (
        allowed-extensions: 'xlsx,xls',
        label: 'Upload Excel'
      )
    }
  }
}
EOF
```

- [ ] **Step 2: 语法检查**

```bash
./bin/mxcli check mdl-examples/doctype-tests/page-fileinput.mdl 2>&1
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add mdl-examples/doctype-tests/page-fileinput.mdl
git commit -m "test(examples): add fileinput widget MDL test case"
```

---

## 自检

- [ ] **Spec 覆盖：** grammar 层 → Task 1；builder 实现 → Task 2；dispatch → Task 3；MDL 测试 → Task 4。
- [ ] **Placeholder 扫描：** Step 4 在 Task 2 提醒了确认实际方法名，非 TBD。
- [ ] **类型一致性：** `buildFileManagerV3` 在 Task 2 Step 4 定义，在 Task 3 Step 2 引用，名称一致。`genPg.NewFileManager()` 已在 modelsdk/gen/pages/types.go:29467 确认存在。
