# Issue 004: CALL MICROFLOW 重复变量名 CE0111 修复计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 当微流中多次用同一输出变量名调用 CALL MICROFLOW / CALL NANOFLOW 时，在写入 BSON 阶段检测重复并报明确错误，而不是静默写出会让 Studio Pro 报 CE0111 的 BSON。

**Architecture:** 在 `flowBuilderGen` 的写路径中，`addCallMicroflowActionGen` 和 `addCallNanoflowActionGen` 调用 `fb.isVariableDeclared()` 检查输出变量是否已声明。若已声明，报告错误（`fb.addError`），同时将该 action 的 `UseReturnVariable` 设为 false（调用照常执行，但不捕获返回值）。若未声明，正常设置输出变量并将其加入 `fb.declaredVars`。

**Tech Stack:** Go (`mdl/executor/flowbuilder_calls_flow_gen.go`)

---

### Task 1: 在 addCallMicroflowActionGen 添加重复变量检测

**Files:**
- Modify: `mdl/executor/flowbuilder_calls_flow_gen.go` (lines 69-81)
- Test: `mdl/executor/flowbuilder_calls_flow_gen_test.go` (新建文件)

- [ ] **Step 1: 写失败测试**

新建文件 `mdl/executor/flowbuilder_calls_flow_gen_test.go`：

```go
package executor

import (
	"strings"
	"testing"
)

// TestAddCallMicroflowGen_DuplicateOutputVar verifies that using the same output
// variable name for two CALL MICROFLOW statements results in a build error,
// not a silent CE0111 in Studio Pro.
func TestAddCallMicroflowGen_DuplicateOutputVar(t *testing.T) {
	// Build a minimal flowBuilderGen with declaredVars pre-populated.
	fb := &flowBuilderGen{
		declaredVars: map[string]string{"Result": "Unknown"},
		varTypes:     map[string]string{},
		errors:       nil,
	}

	// Simulate calling CALL MICROFLOW with an already-declared output variable.
	// We call the internal helper that sets the output variable.
	checkOutputVarCollision(fb, "Result")

	if len(fb.errors) == 0 {
		t.Fatal("expected an error for duplicate output variable, got none")
	}
	if !strings.Contains(fb.errors[0], "Result") {
		t.Errorf("error should mention the variable name, got: %q", fb.errors[0])
	}
}

func TestAddCallMicroflowGen_FreshOutputVar(t *testing.T) {
	fb := &flowBuilderGen{
		declaredVars: map[string]string{},
		varTypes:     map[string]string{},
		errors:       nil,
	}

	checkOutputVarCollision(fb, "Result")

	if len(fb.errors) != 0 {
		t.Errorf("expected no error for fresh output variable, got: %v", fb.errors)
	}
	if _, ok := fb.declaredVars["Result"]; !ok {
		t.Error("fresh output variable should be added to declaredVars")
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

```bash
cd /mnt/data_sdd/gh/mxcli/.claude/worktrees/dev-fix
~/go1.26/bin/go test ./mdl/executor/ -run TestAddCallMicroflowGen -v 2>&1 | tail -15
```

Expected: `FAIL` — `checkOutputVarCollision` undefined.

- [ ] **Step 3: 实现 checkOutputVarCollision 辅助函数**

在 `mdl/executor/flowbuilder_calls_flow_gen.go` 的 imports 之后（在 `addCallMicroflowActionGen` 之前）添加：

```go
// checkOutputVarCollision checks if varName is already declared in fb.
// If so, it records an error. If not, it registers varName in declaredVars.
// Call this before setting OutputVariableName on a call action.
// Returns true if the variable is a collision (caller should set UseReturnVariable(false)).
func checkOutputVarCollision(fb *flowBuilderGen, varName string) bool {
	if varName == "" {
		return false
	}
	if fb.isVariableDeclared("$" + varName) {
		fb.addError(
			"output variable $%s is already declared in this microflow; use a unique name (e.g. $%s2)",
			varName, varName,
		)
		return true
	}
	fb.declaredVars[varName] = "Unknown"
	return false
}
```

- [ ] **Step 4: 修改 addCallMicroflowActionGen (lines 73-74)**

原代码（`mdl/executor/flowbuilder_calls_flow_gen.go` 约第 69-81 行）：

```go
	action.SetErrorHandlingType(fb.ehTypeGen(s.ErrorHandling))
	action.SetMicroflowCall(call)
	action.SetOutputVariableName(s.OutputVariable)
	action.SetUseReturnVariable(s.OutputVariable != "")

	// TODO Stage 3.2.3.j: ...

	return fb.genActivityWrap(action, s.ErrorHandling, s.OutputVariable)
```

改为：

```go
	action.SetErrorHandlingType(fb.ehTypeGen(s.ErrorHandling))
	action.SetMicroflowCall(call)

	collision := checkOutputVarCollision(fb, s.OutputVariable)
	if !collision {
		action.SetOutputVariableName(s.OutputVariable)
		action.SetUseReturnVariable(s.OutputVariable != "")
	}

	// TODO Stage 3.2.3.j: ...

	return fb.genActivityWrap(action, s.ErrorHandling, s.OutputVariable)
```

- [ ] **Step 5: 运行测试确认通过**

```bash
~/go1.26/bin/go test ./mdl/executor/ -run TestAddCallMicroflowGen -v 2>&1 | tail -10
```

Expected: `PASS`.

- [ ] **Step 6: Commit 阶段性成果**

```bash
git add mdl/executor/flowbuilder_calls_flow_gen.go mdl/executor/flowbuilder_calls_flow_gen_test.go
git commit -m "fix(executor): detect duplicate CALL MICROFLOW output variable before CE0111"
```

---

### Task 2: 同步修复 addCallNanoflowActionGen

`addCallNanoflowActionGen` 有相同问题（lines 110-111）。

**Files:**
- Modify: `mdl/executor/flowbuilder_calls_flow_gen.go` (lines 107-118)
- Test: 在同一测试文件中追加

- [ ] **Step 1: 添加 nanoflow 测试**

在 `mdl/executor/flowbuilder_calls_flow_gen_test.go` 末尾追加：

```go
func TestAddCallNanoflowGen_DuplicateOutputVar(t *testing.T) {
	fb := &flowBuilderGen{
		declaredVars: map[string]string{"Result": "Unknown"},
		varTypes:     map[string]string{},
		errors:       nil,
	}

	collision := checkOutputVarCollision(fb, "Result")

	if !collision {
		t.Fatal("expected collision=true for duplicate variable, got false")
	}
	if len(fb.errors) == 0 {
		t.Fatal("expected error recorded, got none")
	}
}
```

- [ ] **Step 2: 修改 addCallNanoflowActionGen (lines 110-111)**

原代码：

```go
	action.SetErrorHandlingType(fb.ehTypeGen(s.ErrorHandling))
	action.SetNanoflowCall(call)
	action.SetOutputVariableName(s.OutputVariable)
	action.SetUseReturnVariable(s.OutputVariable != "")

	// TODO Stage 3.2.3.j: ...

	return fb.genActivityWrap(action, s.ErrorHandling, s.OutputVariable)
```

改为：

```go
	action.SetErrorHandlingType(fb.ehTypeGen(s.ErrorHandling))
	action.SetNanoflowCall(call)

	collision := checkOutputVarCollision(fb, s.OutputVariable)
	if !collision {
		action.SetOutputVariableName(s.OutputVariable)
		action.SetUseReturnVariable(s.OutputVariable != "")
	}

	// TODO Stage 3.2.3.j: ...

	return fb.genActivityWrap(action, s.ErrorHandling, s.OutputVariable)
```

- [ ] **Step 3: 运行全部 executor 测试**

```bash
~/go1.26/bin/go test ./mdl/executor/ -count=1 2>&1 | tail -10
```

Expected: 无新增 FAIL。

- [ ] **Step 4: Commit**

```bash
git add mdl/executor/flowbuilder_calls_flow_gen.go mdl/executor/flowbuilder_calls_flow_gen_test.go
git commit -m "fix(executor): same duplicate-variable guard for CALL NANOFLOW"
```

---

## 自检

- [ ] **Spec 覆盖：** CE0111 检测 → Task 1+2；microflow + nanoflow 双覆盖 ✓
- [ ] **Placeholder 扫描：** 无 TBD。
- [ ] **类型一致性：** `checkOutputVarCollision(fb *flowBuilderGen, varName string) bool` 在 Task 1 Step 3 定义，在 Task 1 Step 4、Task 2 Step 2 使用，签名一致。
