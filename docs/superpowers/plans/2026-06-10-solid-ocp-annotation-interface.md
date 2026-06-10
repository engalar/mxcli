# OCP 修复：Annotated 接口消除 getStatementAnnotations switch

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 `mdl/ast` 的 `MicroflowStatement` 接口上增加 `GetAnnotations()` 方法，彻底消除 `microflow_ast_helpers.go` 中 52 行的 `getStatementAnnotations` type-switch。

**Architecture:** `MicroflowStatement` 接口扩展一个方法 `GetAnnotations() *ActivityAnnotations`；所有 56 个实现类型（`ast_microflow.go` + `ast_microflow_workflow.go`）各添加一行 getter；两个调用点（`microflow_ast_helpers.go`、`flowbuilder_annotations_v2.go`）改为直接调用接口方法。没有运行时行为变更，纯重构。

**Tech Stack:** Go 1.24，`mdl/ast` 包，`mdl/executor` 包。

---

## 影响文件概览

| 文件 | 操作 |
|------|------|
| `mdl/ast/ast_microflow.go` | 扩展 `MicroflowStatement` 接口；为 43 个类型各加 `GetAnnotations()` |
| `mdl/ast/ast_microflow_workflow.go` | 为 13 个 workflow 类型各加 `GetAnnotations()` |
| `mdl/executor/microflow_ast_helpers.go` | 删除 `getStatementAnnotations` 函数（52 行 switch） |
| `mdl/executor/flowbuilder_annotations_v2.go` | 调用改为 `stmt.GetAnnotations()` |

---

## Task 1：扩展 `MicroflowStatement` 接口

### 步骤

- [ ] **Step 1.1：写一个确认 switch 函数覆盖所有类型的测试（先让它通过，作为基准）**

在 `mdl/ast/ast_microflow_test.go` 中（新建文件）：

```go
// SPDX-License-Identifier: Apache-2.0
package ast_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

// TestAnnotatedInterfaceCoverage 确认所有 MicroflowStatement 实现都有 GetAnnotations。
// 一旦接口方法加上，编译期就会强制检查，这个测试是过渡期的文档。
func TestAnnotatedInterfaceCoverage(t *testing.T) {
	stmts := []ast.MicroflowStatement{
		&ast.DeclareStmt{},
		&ast.MfSetStmt{},
		&ast.ReturnStmt{},
		&ast.RaiseErrorStmt{},
		&ast.CreateObjectStmt{},
		&ast.ChangeObjectStmt{},
		&ast.MfCommitStmt{},
		&ast.DeleteObjectStmt{},
		&ast.RollbackStmt{},
		&ast.RetrieveStmt{},
		&ast.IfStmt{},
		&ast.EnumSplitStmt{},
		&ast.InheritanceSplitStmt{},
		&ast.CastObjectStmt{},
		&ast.LoopStmt{},
		&ast.WhileStmt{},
		&ast.LogStmt{},
		&ast.CallMicroflowStmt{},
		&ast.CallNanoflowStmt{},
		&ast.CallJavaActionStmt{},
		&ast.CallJavaScriptActionStmt{},
		&ast.CallWebServiceStmt{},
		&ast.ExecuteDatabaseQueryStmt{},
		&ast.CallExternalActionStmt{},
		&ast.BreakStmt{},
		&ast.ContinueStmt{},
		&ast.ListOperationStmt{},
		&ast.AggregateListStmt{},
		&ast.CreateListStmt{},
		&ast.AddToListStmt{},
		&ast.RemoveFromListStmt{},
		&ast.ShowPageStmt{},
		&ast.ClosePageStmt{},
		&ast.ShowHomePageStmt{},
		&ast.SynchronizeStmt{},
		&ast.ShowMessageStmt{},
		&ast.DownloadFileStmt{},
		&ast.ValidationFeedbackStmt{},
		&ast.RestCallStmt{},
		// workflow statements
		&ast.CallWorkflowStmt{},
		&ast.GetWorkflowDataStmt{},
		&ast.GetWorkflowsStmt{},
		&ast.GetWorkflowActivityRecordsStmt{},
		&ast.WorkflowOperationStmt{},
		&ast.SetTaskOutcomeStmt{},
		&ast.OpenUserTaskStmt{},
		&ast.NotifyWorkflowStmt{},
		&ast.OpenWorkflowStmt{},
		&ast.LockWorkflowStmt{},
		&ast.UnlockWorkflowStmt{},
		&ast.GenerateJumpToStmt{},
		&ast.ApplyJumpToStmt{},
	}
	for _, s := range stmts {
		// 编译期已保证实现了接口；运行期确认 nil 返回不 panic
		_ = s.GetAnnotations()
	}
	t.Logf("all %d MicroflowStatement types implement GetAnnotations()", len(stmts))
}
```

- [ ] **Step 1.2：运行确认测试目前会编译失败（接口方法还没加）**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./mdl/ast/... -run TestAnnotatedInterfaceCoverage -v 2>&1 | head -20
```

预期：编译错误 `cannot use ... as type ast.MicroflowStatement: missing method GetAnnotations`（因为接口还没加方法）。

- [ ] **Step 1.3：在 `mdl/ast/ast_microflow.go` 扩展 `MicroflowStatement` 接口**

找到（行 9-12）：
```go
// MicroflowStatement represents a statement inside a microflow body.
type MicroflowStatement interface {
	isMicroflowStatement()
}
```

改为：
```go
// MicroflowStatement represents a statement inside a microflow body.
// Every concrete statement type carries an optional Annotations field;
// GetAnnotations returns it (nil when the statement has no annotations or
// the field was not set by the parser).
type MicroflowStatement interface {
	isMicroflowStatement()
	GetAnnotations() *ActivityAnnotations
}
```

- [ ] **Step 1.4：运行编译查看缺失实现**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go build ./mdl/ast/... 2>&1 | head -30
```

这会列出所有未实现 `GetAnnotations` 的类型。用输出来核对实现数量。

---

## Task 2：为 ast_microflow.go 中的 43 个类型实现方法

- [ ] **Step 2.1：在 `mdl/ast/ast_microflow.go` 末尾批量添加 GetAnnotations 实现**

在文件末尾追加（每个类型一行，与现有 `isMicroflowStatement()` 风格一致）：

```go
// GetAnnotations implements MicroflowStatement.
func (s *DeclareStmt) GetAnnotations() *ActivityAnnotations             { return s.Annotations }
func (s *EnumSplitStmt) GetAnnotations() *ActivityAnnotations           { return s.Annotations }
func (s *InheritanceSplitStmt) GetAnnotations() *ActivityAnnotations    { return s.Annotations }
func (s *CastObjectStmt) GetAnnotations() *ActivityAnnotations          { return s.Annotations }
func (s *MfSetStmt) GetAnnotations() *ActivityAnnotations               { return s.Annotations }
func (s *ReturnStmt) GetAnnotations() *ActivityAnnotations              { return s.Annotations }
func (s *RaiseErrorStmt) GetAnnotations() *ActivityAnnotations          { return s.Annotations }
func (s *CreateObjectStmt) GetAnnotations() *ActivityAnnotations        { return s.Annotations }
func (s *ChangeObjectStmt) GetAnnotations() *ActivityAnnotations        { return s.Annotations }
func (s *MfCommitStmt) GetAnnotations() *ActivityAnnotations            { return s.Annotations }
func (s *DeleteObjectStmt) GetAnnotations() *ActivityAnnotations        { return s.Annotations }
func (s *RollbackStmt) GetAnnotations() *ActivityAnnotations            { return s.Annotations }
func (s *RetrieveStmt) GetAnnotations() *ActivityAnnotations            { return s.Annotations }
func (s *IfStmt) GetAnnotations() *ActivityAnnotations                  { return s.Annotations }
func (s *LoopStmt) GetAnnotations() *ActivityAnnotations                { return s.Annotations }
func (s *WhileStmt) GetAnnotations() *ActivityAnnotations               { return s.Annotations }
func (s *LogStmt) GetAnnotations() *ActivityAnnotations                 { return s.Annotations }
func (s *CallMicroflowStmt) GetAnnotations() *ActivityAnnotations       { return s.Annotations }
func (s *CallNanoflowStmt) GetAnnotations() *ActivityAnnotations        { return s.Annotations }
func (s *CallJavaActionStmt) GetAnnotations() *ActivityAnnotations      { return s.Annotations }
func (s *CallJavaScriptActionStmt) GetAnnotations() *ActivityAnnotations { return s.Annotations }
func (s *CallWebServiceStmt) GetAnnotations() *ActivityAnnotations      { return s.Annotations }
func (s *ExecuteDatabaseQueryStmt) GetAnnotations() *ActivityAnnotations { return s.Annotations }
func (s *CallExternalActionStmt) GetAnnotations() *ActivityAnnotations  { return s.Annotations }
func (s *BreakStmt) GetAnnotations() *ActivityAnnotations               { return s.Annotations }
func (s *ContinueStmt) GetAnnotations() *ActivityAnnotations            { return s.Annotations }
func (s *ListOperationStmt) GetAnnotations() *ActivityAnnotations       { return s.Annotations }
func (s *AggregateListStmt) GetAnnotations() *ActivityAnnotations       { return s.Annotations }
func (s *CreateListStmt) GetAnnotations() *ActivityAnnotations          { return s.Annotations }
func (s *AddToListStmt) GetAnnotations() *ActivityAnnotations           { return s.Annotations }
func (s *RemoveFromListStmt) GetAnnotations() *ActivityAnnotations      { return s.Annotations }
func (s *ShowPageStmt) GetAnnotations() *ActivityAnnotations            { return s.Annotations }
func (s *ClosePageStmt) GetAnnotations() *ActivityAnnotations           { return s.Annotations }
func (s *ShowHomePageStmt) GetAnnotations() *ActivityAnnotations        { return s.Annotations }
func (s *SynchronizeStmt) GetAnnotations() *ActivityAnnotations         { return s.Annotations }
func (s *ShowMessageStmt) GetAnnotations() *ActivityAnnotations         { return s.Annotations }
func (s *DownloadFileStmt) GetAnnotations() *ActivityAnnotations        { return s.Annotations }
func (s *ValidationFeedbackStmt) GetAnnotations() *ActivityAnnotations  { return s.Annotations }
func (s *RestCallStmt) GetAnnotations() *ActivityAnnotations            { return s.Annotations }
```

> **注意**：检查 `ast_microflow.go` 中是否有其他 `isMicroflowStatement` 实现类型（如 `SynchronizeStmt`、`ShowMessageStmt` 等），grep 确认：
> ```bash
> grep -n "func.*isMicroflowStatement" mdl/ast/ast_microflow.go | grep -v "//\|_test"
> ```
> 对输出中的每个类型都要有对应的 `GetAnnotations` 行。

- [ ] **Step 2.2：为 `ast_microflow_workflow.go` 中的 13 个类型添加实现**

在 `ast_microflow_workflow.go` 末尾追加：

```go
// GetAnnotations implements MicroflowStatement.
func (s *CallWorkflowStmt) GetAnnotations() *ActivityAnnotations             { return s.Annotations }
func (s *GetWorkflowDataStmt) GetAnnotations() *ActivityAnnotations          { return s.Annotations }
func (s *GetWorkflowsStmt) GetAnnotations() *ActivityAnnotations             { return s.Annotations }
func (s *GetWorkflowActivityRecordsStmt) GetAnnotations() *ActivityAnnotations { return s.Annotations }
func (s *WorkflowOperationStmt) GetAnnotations() *ActivityAnnotations        { return s.Annotations }
func (s *SetTaskOutcomeStmt) GetAnnotations() *ActivityAnnotations           { return s.Annotations }
func (s *OpenUserTaskStmt) GetAnnotations() *ActivityAnnotations             { return s.Annotations }
func (s *NotifyWorkflowStmt) GetAnnotations() *ActivityAnnotations           { return s.Annotations }
func (s *OpenWorkflowStmt) GetAnnotations() *ActivityAnnotations             { return s.Annotations }
func (s *LockWorkflowStmt) GetAnnotations() *ActivityAnnotations             { return s.Annotations }
func (s *UnlockWorkflowStmt) GetAnnotations() *ActivityAnnotations           { return s.Annotations }
func (s *GenerateJumpToStmt) GetAnnotations() *ActivityAnnotations           { return s.Annotations }
func (s *ApplyJumpToStmt) GetAnnotations() *ActivityAnnotations              { return s.Annotations }
```

> **确认 workflow 文件的实现类型列表**：
> ```bash
> grep -n "func.*isMicroflowStatement" mdl/ast/ast_microflow_workflow.go
> ```

- [ ] **Step 2.3：编译 ast 包确认全部实现**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go build ./mdl/ast/...
```

预期：无错误。

- [ ] **Step 2.4：运行测试确认通过**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./mdl/ast/... -run TestAnnotatedInterfaceCoverage -v
```

预期：`PASS`，打印"all 52 MicroflowStatement types implement GetAnnotations()"。

---

## Task 3：删除 switch，改用接口方法

- [ ] **Step 3.1：在 `microflow_ast_helpers.go` 中删除 `getStatementAnnotations` 函数**

找到行 75-187（从注释到最后的 `}`）：

```go
// getStatementAnnotations extracts the *ast.ActivityAnnotations from
// any microflow statement. Returns nil when the statement has no
// annotations field. Pure switch over AST types.
func getStatementAnnotations(stmt ast.MicroflowStatement) *ast.ActivityAnnotations {
	switch s := stmt.(type) {
	case *ast.DeclareStmt:
		return s.Annotations
	// ... 50 more cases ...
	default:
		return nil
	}
}
```

**完整删除这 113 行**（从注释开始到函数结束的 `}`）。

- [ ] **Step 3.2：更新 `flowbuilder_annotations_v2.go` 的调用点**

找到（行 36）：
```go
ann := getStatementAnnotations(stmt)
```

替换为：
```go
ann := stmt.GetAnnotations()
```

- [ ] **Step 3.3：编译 executor 确认无其他调用点**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go build ./mdl/executor/... 2>&1
```

预期：无错误。如果有 `undefined: getStatementAnnotations` 错误，说明有遗漏调用点，逐一修改为 `stmt.GetAnnotations()`。

- [ ] **Step 3.4：运行 ast 和 executor 的全量测试**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./mdl/ast/... ./mdl/executor/... -count=1 2>&1 | tail -10
```

预期：全部 `ok`，无 FAIL。

- [ ] **Step 3.5：commit**

```bash
git add mdl/ast/ast_microflow.go mdl/ast/ast_microflow_workflow.go mdl/ast/ast_microflow_test.go mdl/executor/microflow_ast_helpers.go mdl/executor/flowbuilder_annotations_v2.go
git commit -m "$(cat <<'EOF'
refactor(ast): add GetAnnotations() to MicroflowStatement, remove 52-case switch

executor/microflow_ast_helpers.go had a 52-case type-switch
getStatementAnnotations() that returned s.Annotations for every
MicroflowStatement type. This violates OCP: every new statement type
required a manual addition to the switch.

Add GetAnnotations() *ActivityAnnotations to MicroflowStatement interface;
implement it on all 56 concrete types (one line each). The switch is
deleted; flowbuilder_annotations_v2.go calls stmt.GetAnnotations() directly.

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## 自检 Checklist

- [ ] `go build ./mdl/ast/... ./mdl/executor/...` 无错误
- [ ] `go test ./mdl/ast/... ./mdl/executor/...` 无 FAIL
- [ ] `grep -n "getStatementAnnotations" mdl/executor/` 无输出（函数已删除）
- [ ] `grep -c "GetAnnotations" mdl/ast/ast_microflow.go` 输出 ≥ 39（39 个实现 + 1 个接口声明）
- [ ] `grep -c "GetAnnotations" mdl/ast/ast_microflow_workflow.go` 输出 ≥ 13
