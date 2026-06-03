# Issue 008: DESCRIBE PAGE show_page roundtrip 修复计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复 `cmd_pages_describe_parse.go` 中 `show_page` action 的输出格式：当前输出 `show_page 'Module.Page'`（页面名有单引号），但 grammar 期望 `show_page Module.Page`（qualifiedName，不带引号），导致 DESCRIBE 输出无法通过 `mxcli check`。

**Architecture:** `cmd_pages_describe_parse.go` 是"从已有 MDL 页面文本解析再序列化"的路径，其 action 输出函数硬编码了带单引号的格式；`cmd_pages_describe_output.go`（从 BSON 读出的路径）已经正确不带引号。只需修改 parse 路径中三处 `show_page` 拼接，去掉单引号，并补充测试。

**Tech Stack:** Go (`mdl/executor/cmd_pages_describe_parse.go`)

---

### Task 1: 修复 describe_parse.go 中 show_page 输出格式

**Files:**
- Modify: `mdl/executor/cmd_pages_describe_parse.go` (lines 461–477)
- Test: `mdl/executor/cmd_pages_describe_action_test.go` (追加)

- [ ] **Step 1: 读取并确认 describe_parse.go 中的三处错误**

在文件中搜索所有 `show_page '` 格式的拼接：

```bash
grep -n "show_page '" /mnt/data_sdd/gh/mxcli/.claude/worktrees/dev-fix/mdl/executor/cmd_pages_describe_parse.go
```

Expected 输出（约 3 处）：
```
461:			return "show_page '" + formName + "'"
467:			return "show_page '" + pageName + "'"
474:			return "show_page '" + pageName + "'"
```

- [ ] **Step 2: 写失败测试**

在 `mdl/executor/cmd_pages_describe_action_test.go` 末尾追加：

```go
// TestExtractButtonAction_ShowPage verifies that show_page action outputs a
// qualified name WITHOUT quotes, matching the SHOW_PAGE grammar rule.
// Issue 008: parse path was producing "show_page 'Module.Page'" (with quotes)
// which STRING_LITERAL does not match actionExprV3's SHOW_PAGE qualifiedName rule.
func TestExtractButtonAction_ShowPage(t *testing.T) {
	ctx := &ExecContext{}
	widget := map[string]any{
		"$Type": "Pages$FormAction",
		"FormSettings": map[string]any{
			"Form": "MyModule.OverviewPage",
		},
	}
	got := extractButtonActionFromParsed(ctx, widget)
	want := "show_page MyModule.OverviewPage"
	if got != want {
		t.Errorf("show_page action: got %q, want %q", got, want)
	}
}
```

**注意：** 若 `extractButtonActionFromParsed` 不存在，改用 describe_parse.go 中实际调用的函数名。先运行以下命令确认：

```bash
grep -n "^func.*extractButton\|^func.*parseButton\|^func.*actionFrom" \
  /mnt/data_sdd/gh/mxcli/.claude/worktrees/dev-fix/mdl/executor/cmd_pages_describe_parse.go | head -5
```

用实际函数名替换测试中的调用。

- [ ] **Step 3: 运行测试确认失败**

```bash
cd /mnt/data_sdd/gh/mxcli/.claude/worktrees/dev-fix
~/go1.26/bin/go test ./mdl/executor/ -run TestExtractButtonAction_ShowPage -v 2>&1 | tail -15
```

Expected: `FAIL` — got `"show_page 'MyModule.OverviewPage'"`, want `"show_page MyModule.OverviewPage"`.

- [ ] **Step 4: 修复 describe_parse.go 中的三处拼接**

打开 `mdl/executor/cmd_pages_describe_parse.go`，找到约第 458–480 行的 switch case（`Forms$FormAction` / `Pages$FormAction`）。

**原代码（三处）：**

```go
return "show_page '" + formName + "'"
// ...
return "show_page '" + pageName + "'"
// ...
return "show_page '" + pageName + "'"
```

**改为（三处，去掉单引号）：**

```go
return "show_page " + formName
// ...
return "show_page " + pageName
// ...
return "show_page " + pageName
```

用 Edit 工具逐一替换，或用 `sed`（注意保留 fallback `return "show_page"` 不变）：

```bash
grep -c "show_page '" /mnt/data_sdd/gh/mxcli/.claire/worktrees/dev-fix/mdl/executor/cmd_pages_describe_parse.go
```

如果 grep 输出 > 3，先审查所有位置再编辑。

- [ ] **Step 5: 同时检查 CREATE_OBJECT 路径中 show_page 是否有引号**

```bash
grep -n "show_page" /mnt/data_sdd/gh/mxcli/.claude/worktrees/dev-fix/mdl/executor/cmd_pages_describe_parse.go
```

若有其他 `show_page '...'` 格式，一并修复（去掉单引号）。

- [ ] **Step 6: 运行测试确认通过**

```bash
~/go1.26/bin/go test ./mdl/executor/ -run TestExtractButtonAction_ShowPage -v 2>&1 | tail -10
```

Expected: `PASS`.

- [ ] **Step 7: 运行全部 describe 相关测试**

```bash
~/go1.26/bin/go test ./mdl/executor/ -run TestExtractButton -v 2>&1 | tail -20
```

Expected: 所有现有测试 PASS，无回归。

- [ ] **Step 8: 运行全部 executor 测试**

```bash
~/go1.26/bin/go test ./mdl/executor/ -count=1 2>&1 | tail -10
```

Expected: 无新增 FAIL。

- [ ] **Step 9: Commit**

```bash
git add mdl/executor/cmd_pages_describe_parse.go mdl/executor/cmd_pages_describe_action_test.go
git commit -m "fix(describe): remove quotes from show_page qualified name for roundtrip correctness"
```

---

### Task 2: 验证 close_page / create_object then show_page 路径

describe_output.go 的 `create_object ... then show_page` 路径也会拼接页面名，需确认无引号。

**Files:**
- Read-only verify: `mdl/executor/cmd_pages_describe_output.go`

- [ ] **Step 1: 确认 describe_output.go 中 show_page 格式正确**

```bash
grep -n "show_page" /mnt/data_sdd/gh/mxcli/.claude/worktrees/dev-fix/mdl/executor/cmd_pages_describe_output.go
```

若有任何 `show_page '` 格式（带引号），同样去掉引号。若全部已是 `"show_page " + pageName`，无需改动。

- [ ] **Step 2: 如有修改，运行测试**

```bash
~/go1.26/bin/go test ./mdl/executor/ -count=1 2>&1 | tail -5
```

- [ ] **Step 3: 如有修改则 Commit**

```bash
git add mdl/executor/cmd_pages_describe_output.go
git commit -m "fix(describe): fix remaining show_page quote format in output path"
```

---

## 自检

- [ ] **Spec 覆盖：** describe_parse.go 的引号格式 → Task 1；describe_output.go 二次确认 → Task 2。
- [ ] **Placeholder 扫描：** 无 TBD。
- [ ] **类型一致性：** 测试使用的函数名需以实际 grep 结果为准（Step 2 有说明）。
