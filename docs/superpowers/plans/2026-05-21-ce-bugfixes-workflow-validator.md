# CE Bug Fixes: Workflow Write Path + Microflow Validator Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复 mxcli 生成 workflow/microflow 时产生的六类 Mendix Studio Pro 编译错误（CE0053、CE0111、CE1834、CE1859、CE7247 以及 describe formatter 不输出 page/targeting）。

**Architecture:** 修改分三层：①写路径（`cmd_workflows_write_gen2.go`）补全 user task 必填字段；②描述格式化器（`cmd_workflows_gen.go`）读取新字段；③静态验证器（`validate_flow_body.go` + `validate.go`）修正作用域模型和类型检查。CE0053 额外在 gen builder 写入时拦截非持久化实体。

**Tech Stack:** Go 1.26，modelsdk gen 类型（`modelsdk/gen/workflows`、`modelsdk/gen/domainmodels`），executor 层测试（`//go:build integration`）。

---

## 文件变更一览

| 文件 | 操作 | 原因 |
|------|------|------|
| `mdl/executor/cmd_workflows_write_gen2.go` | Modify | CE1834: 加 SetTaskPage；CE1859: 无 targeting 时报错 |
| `mdl/executor/cmd_workflows_gen.go` | Modify | Describe formatter 读取 TaskPage/UserTargeting |
| `mdl/executor/validate_flow_body.go` | Modify | CE0111: 扁平 output var 追踪；CE7247: list 类型检查 |
| `mdl/executor/validate.go` | Modify | CE0053: 非持久化实体在参照验证阶段报错 |
| `mdl/executor/flowbuilder_gen.go` | Modify | CE0053: 懒加载非持久化实体集合 |
| `mdl/executor/flowbuilder_actions_gen.go` | Modify | CE0053: 写入时拦截 create list of 非持久化实体 |

---

## Task 1: CE1834 — user task 写入 TaskPage

**Files:**
- Modify: `mdl/executor/cmd_workflows_write_gen2.go:244-284`
- Test: `mdl/executor/cmd_workflows_write_gen2_test.go`

- [ ] **Step 1: 写失败测试**

在 `cmd_workflows_write_gen2_test.go` 中，在 `TestBuildSingleUserTaskGenActivity_FieldsPropagate` 后添加：

```go
func TestBuildSingleUserTaskGenActivity_PagePropagates(t *testing.T) {
	n := &ast.WorkflowUserTaskNode{
		Name:    "Step",
		Caption: "step caption",
		Page:    ast.QualifiedName{Module: "MyMod", Name: "MyPage"},
	}
	got := buildSingleUserTaskGenActivity(n)
	tp := got.TaskPage()
	if tp == nil {
		t.Fatal("TaskPage is nil — CE1834 bug")
	}
	pr, ok := tp.(*genWf.PageReference)
	if !ok {
		t.Fatalf("TaskPage type = %T, want *PageReference", tp)
	}
	if pr.PageQualifiedName() != "MyMod.MyPage" {
		t.Errorf("PageQualifiedName = %q, want %q", pr.PageQualifiedName(), "MyMod.MyPage")
	}
}

func TestBuildMultiUserTaskGenActivity_PagePropagates(t *testing.T) {
	n := &ast.WorkflowUserTaskNode{
		Name:        "Vote",
		IsMultiUser: true,
		Page:        ast.QualifiedName{Module: "MyMod", Name: "MyPage"},
	}
	got := buildMultiUserTaskGenActivity(n)
	tp := got.TaskPage()
	if tp == nil {
		t.Fatal("TaskPage is nil for multi user task — CE1834 bug")
	}
	pr, ok := tp.(*genWf.PageReference)
	if !ok {
		t.Fatalf("TaskPage type = %T, want *PageReference", tp)
	}
	if pr.PageQualifiedName() != "MyMod.MyPage" {
		t.Errorf("PageQualifiedName = %q, want %q", pr.PageQualifiedName(), "MyMod.MyPage")
	}
}
```

- [ ] **Step 2: 运行确认失败**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./mdl/executor/ -run "TestBuildSingleUserTaskGenActivity_PagePropagates|TestBuildMultiUserTaskGenActivity_PagePropagates" -v
```

预期：FAIL，`TaskPage is nil`

- [ ] **Step 3: 实现修复**

在 `cmd_workflows_write_gen2.go` 的 `buildSingleUserTaskGenActivity`（line ~253）和 `buildMultiUserTaskGenActivity`（line ~274）中，在 `SetUserTargeting` 那段之前插入：

对 `buildSingleUserTaskGenActivity`：
```go
func buildSingleUserTaskGenActivity(n *ast.WorkflowUserTaskNode) *genWf.SingleUserTaskActivity {
	task := genWf.NewSingleUserTaskActivity()
	task.SetID(element.ID(types.GenerateID()))
	task.SetName(n.Name)
	task.SetCaption(n.Caption)
	task.SetDueDate(n.DueDate)
	if n.TaskDescription != "" {
		task.SetTaskDescription(newStringTemplateGen(n.TaskDescription))
	}
	// ▼ CE1834 fix
	if n.Page.Module != "" && n.Page.Name != "" {
		pr := genWf.NewPageReference()
		pr.SetPageQualifiedName(n.Page.Module + "." + n.Page.Name)
		task.SetTaskPage(pr)
	}
	// ▲ CE1834 fix
	if tgt := buildUserTargetingGen(n.Targeting); tgt != nil {
		task.SetUserTargeting(tgt)
	}
	for _, oc := range buildUserTaskOutcomesGen(n.Outcomes) {
		task.AddOutcomes(oc)
	}
	for _, ev := range buildBoundaryEventsGen(n.BoundaryEvents) {
		task.AddBoundaryEvents(ev)
	}
	return task
}
```

对 `buildMultiUserTaskGenActivity` 做同样的改动（在 `SetUserTargeting` 之前加相同的 page block）。

- [ ] **Step 4: 运行确认通过**

```bash
go test ./mdl/executor/ -run "TestBuildSingleUserTaskGenActivity_PagePropagates|TestBuildMultiUserTaskGenActivity_PagePropagates" -v
```

预期：PASS

- [ ] **Step 5: Commit**

```bash
git add mdl/executor/cmd_workflows_write_gen2.go mdl/executor/cmd_workflows_write_gen2_test.go
git commit -m "fix(workflow): write TaskPage to BSON for user tasks (CE1834)"
```

---

## Task 2: CE1859 — user task 无 targeting 时报错

**Files:**
- Modify: `mdl/executor/cmd_workflows_write_gen2.go:244-284`
- Test: `mdl/executor/cmd_workflows_write_gen2_test.go`

背景：`userTargeting` 在 Mendix 11.2+ 是 Required。若 MDL 没有 `targeting` 子句，现在的代码什么都不写，Studio Pro 报 CE1859。空 XPath 也报 CE1859，所以不能写默认空值——必须在写入前报错。

- [ ] **Step 1: 写失败测试**

```go
func TestValidateUserTaskTargeting_NoTargeting_ReturnsError(t *testing.T) {
	n := &ast.WorkflowUserTaskNode{
		Name:    "Step",
		Caption: "no targeting",
		// Targeting.Kind == "" — no targeting clause
	}
	err := validateUserTaskTargeting(n)
	if err == nil {
		t.Error("expected error for missing targeting (CE1859), got nil")
	}
}

func TestValidateUserTaskTargeting_EmptyXPath_ReturnsError(t *testing.T) {
	n := &ast.WorkflowUserTaskNode{
		Name:    "Step",
		Caption: "empty xpath",
		Targeting: ast.WorkflowTargetingNode{Kind: "xpath", XPath: ""},
	}
	err := validateUserTaskTargeting(n)
	if err == nil {
		t.Error("expected error for empty xpath constraint (CE1859), got nil")
	}
}

func TestValidateUserTaskTargeting_ValidXPath_NoError(t *testing.T) {
	n := &ast.WorkflowUserTaskNode{
		Name:    "Step",
		Caption: "valid xpath",
		Targeting: ast.WorkflowTargetingNode{Kind: "xpath", XPath: "[%CurrentUser%]"},
	}
	err := validateUserTaskTargeting(n)
	if err != nil {
		t.Errorf("expected no error for valid targeting, got: %v", err)
	}
}
```

- [ ] **Step 2: 运行确认失败**

```bash
go test ./mdl/executor/ -run "TestValidateUserTaskTargeting" -v
```

预期：编译错误（`validateUserTaskTargeting` 不存在）

- [ ] **Step 3: 新增 validateUserTaskTargeting 并在 validateWorkflowActivity 中调用**

`validateWorkflowActivities` / `validateWorkflowActivity` 返回 `error`（非 `[]string`），新函数也遵循同样模式。

在 `cmd_workflows_write_gen2.go` 末尾添加：

```go
// validateUserTaskTargeting checks that a user task has a targeting clause.
// Missing or empty targeting produces CE1859 in Studio Pro
// (userTargeting is Required=true for Mendix 11.2+).
func validateUserTaskTargeting(n *ast.WorkflowUserTaskNode) error {
	if n.Targeting.Kind == "" {
		return fmt.Errorf(
			"user task '%s' is missing a targeting clause (CE1859): add 'targeting xpath ...' or 'targeting microflow ...'",
			n.Name,
		)
	}
	if (n.Targeting.Kind == "xpath" || n.Targeting.Kind == "group_xpath") && n.Targeting.XPath == "" {
		return fmt.Errorf(
			"user task '%s' has empty xpath constraint (CE1859): xpath targeting requires a non-empty expression",
			n.Name,
		)
	}
	return nil
}
```

在 `validateWorkflowActivity`（`cmd_workflows_write_gen2.go:~432`）的 `case *ast.WorkflowUserTaskNode` 分支开头调用：

```go
case *ast.WorkflowUserTaskNode:
    if err := validateUserTaskTargeting(v); err != nil {
        return err
    }
    for _, oc := range v.Outcomes {
        if err := validateWorkflowActivities(oc.Activities); err != nil {
            return err
        }
    }
```

- [ ] **Step 4: 运行确认通过**

```bash
go test ./mdl/executor/ -run "TestExecCreateWorkflowGen_NoTargeting_FailsWithCE1859" -v
```

- [ ] **Step 5: 集成测试验证**

```bash
go test ./mdl/executor/ -run "TestRoundtripWorkflow" -v -tags integration 2>&1 | tail -20
```

预期：所有 roundtrip 测试仍然通过（它们都有 targeting 子句）

- [ ] **Step 6: Commit**

```bash
git add mdl/executor/cmd_workflows_write_gen2.go mdl/executor/cmd_workflows_write_gen2_test.go
git commit -m "fix(workflow): validate targeting clause presence on user tasks (CE1859)"
```

---

## Task 3: Describe Formatter — 输出 page 和 targeting

**Files:**
- Modify: `mdl/executor/cmd_workflows_gen.go:621-755`
- Test: `mdl/executor/roundtrip_workflow_test.go`（扩展现有 roundtrip 测试）

问题：`userTaskShapeGenFor` 对 `SingleUserTaskActivity`/`MultiUserTaskActivity` 不填 `Page` 字段，也读旧的 `UserSource`（已删）而非新的 `UserTargeting`；`formatUserSourceGen` 也没有处理新 targeting 类型。

- [ ] **Step 1: 写失败测试**

在 `roundtrip_workflow_test.go` 中找到 `TestRoundtripWorkflow_Comprehensive`，在 checks 列表里加入：

```go
{"user task page", "page " + mod + ".ReviewPage"},
{"user task targeting", "targeting"},
```

- [ ] **Step 2: 运行确认失败**

```bash
go test ./mdl/executor/ -run "TestRoundtripWorkflow_Comprehensive" -v -tags integration 2>&1 | tail -20
```

预期：FAIL，`page RoundtripTest.ReviewPage` 不在输出中

- [ ] **Step 3: 修复 userTaskShapeGenFor — 填 Page 和 UserTargeting**

在 `cmd_workflows_gen.go` 中修改 `userTaskShapeGenFor` 的 `SingleUserTaskActivity` 和 `MultiUserTaskActivity` case：

```go
case *genWf.SingleUserTaskActivity:
    pageQN := ""
    if tp, ok2 := v.TaskPage().(*genWf.PageReference); ok2 {
        pageQN = tp.PageQualifiedName()
    }
    return userTaskShapeGen{
        Name:           v.Name(),
        Caption:        v.Caption(),
        Annotation:     v.Annotation(),
        Page:           pageQN,          // ← 新增
        UserSource:     v.UserTargeting(), // ← 改为读 UserTargeting
        DueDate:        v.DueDate(),
        Description:    readTextElementGen(v.TaskDescription()),
        Outcomes:       v.OutcomesItems(),
        BoundaryEvents: v.BoundaryEventsItems(),
        IsMulti:        false,
    }, true
case *genWf.MultiUserTaskActivity:
    pageQN := ""
    if tp, ok2 := v.TaskPage().(*genWf.PageReference); ok2 {
        pageQN = tp.PageQualifiedName()
    }
    return userTaskShapeGen{
        Name:           v.Name(),
        Caption:        v.Caption(),
        Annotation:     v.Annotation(),
        Page:           pageQN,          // ← 新增
        UserSource:     v.UserTargeting(), // ← 改为读 UserTargeting
        DueDate:        v.DueDate(),
        Description:    readTextElementGen(v.TaskDescription()),
        Outcomes:       v.OutcomesItems(),
        BoundaryEvents: v.BoundaryEventsItems(),
        IsMulti:        true,
    }, true
```

- [ ] **Step 4: 修复 formatUserSourceGen — 处理新 Targeting 类型**

在 `formatUserSourceGen` 的 switch 中添加新类型处理（XPathUserTargeting 和 MicroflowUserTargeting 是 Mendix 11.2+ 的新类型，XPathGroupTargeting 和 MicroflowGroupTargeting 已有但归属旧 switch）：

```go
func formatUserSourceGen(src element.Element, indent string) []string {
    if src == nil {
        return nil
    }
    switch v := src.(type) {
    // ── 旧类型（UserTask / legacy storage）──
    case *genWf.MicroflowBasedUserSource:
        if mf := v.MicroflowQualifiedName(); mf != "" {
            return []string{fmt.Sprintf("%stargeting microflow %s", indent, mf)}
        }
    case *genWf.XPathBasedUserSource:
        if xp := v.XPathConstraint(); xp != "" {
            return []string{fmt.Sprintf("%stargeting xpath '%s'", indent, xp)}
        }
    // ── 新类型（SingleUserTaskActivity / MultiUserTaskActivity, Mendix 11.2+）──
    case *genWf.XPathUserTargeting:
        if xp := v.XPathConstraint(); xp != "" {
            return []string{fmt.Sprintf("%stargeting xpath '%s'", indent, xp)}
        }
    case *genWf.MicroflowUserTargeting:
        if mf := v.MicroflowQualifiedName(); mf != "" {
            return []string{fmt.Sprintf("%stargeting microflow %s", indent, mf)}
        }
    // ── Group targeting（旧 + 新共用相同类型名）──
    case *genWf.MicroflowGroupTargeting:
        if mf := v.MicroflowQualifiedName(); mf != "" {
            return []string{fmt.Sprintf("%stargeting groups microflow %s", indent, mf)}
        }
    case *genWf.XPathGroupTargeting:
        if xp := v.XPathConstraint(); xp != "" {
            return []string{fmt.Sprintf("%stargeting groups xpath '%s'", indent, xp)}
        }
    }
    return nil
}
```

注意：需要确认 `*genWf.XPathUserTargeting` 和 `*genWf.MicroflowUserTargeting` 有 `XPathConstraint()` / `MicroflowQualifiedName()` 方法。运行编译验证：

```bash
go build ./mdl/executor/
```

- [ ] **Step 5: 运行测试确认通过**

```bash
go test ./mdl/executor/ -run "TestRoundtripWorkflow_Comprehensive" -v -tags integration 2>&1 | tail -20
```

预期：PASS，包含 `page RoundtripTest.ReviewPage` 和 `targeting`

- [ ] **Step 6: Commit**

```bash
git add mdl/executor/cmd_workflows_gen.go mdl/executor/roundtrip_workflow_test.go
git commit -m "fix(describe): output page and targeting for SingleUserTaskActivity/MultiUserTaskActivity"
```

---

## Task 4: CE7247 — MfSetStmt 对 list 变量的类型检查

**Files:**
- Modify: `mdl/executor/validate_flow_body.go:147-152`
- Test: `mdl/executor/validate_flow_body_test.go`（新建或已有）

- [ ] **Step 1: 写失败测试**

找到或创建 `validate_flow_body_test.go`，添加：

```go
package executor

import (
    "testing"
    "github.com/mendixlabs/mxcli/mdl/ast"
)

func TestValidateCE7247_SetOnListVariable(t *testing.T) {
    body := []ast.MicroflowStatement{
        &ast.CreateListStmt{
            Variable:   "ResultList",
            EntityType: ast.QualifiedName{Module: "M", Name: "Entity"},
        },
        &ast.MfSetStmt{
            Target: "ResultList",
            Value:  &ast.LiteralExpr{Value: "empty"},
        },
    }
    errs := validateFlowBody(nil, body)
    if len(errs) == 0 {
        t.Error("expected CE7247 error for set on list variable, got none")
    }
    found := false
    for _, e := range errs {
        if strings.Contains(e, "CE7247") || strings.Contains(e, "list") {
            found = true
        }
    }
    if !found {
        t.Errorf("expected CE7247 mention, got: %v", errs)
    }
}
```

- [ ] **Step 2: 运行确认失败**

```bash
go test ./mdl/executor/ -run "TestValidateCE7247" -v
```

预期：FAIL，`expected CE7247 error`

- [ ] **Step 3: 修复 MfSetStmt handler**

在 `validate_flow_body.go` 的 `validateStatement` 函数中，找到：

```go
case *ast.MfSetStmt:
    if !v.isVariableDeclared(s.Target) {
        v.addErrorWithExample(
            fmt.Sprintf("variable '%s' is not declared", s.Target),
            errorExampleDeclareVariable(s.Target))
    }
```

改为：

```go
case *ast.MfSetStmt:
    name := strippedVarName(s.Target)
    if !v.isVariableDeclared(s.Target) {
        v.addErrorWithExample(
            fmt.Sprintf("variable '%s' is not declared", s.Target),
            errorExampleDeclareVariable(s.Target))
    } else if listType, ok := v.varTypes[name]; ok && strings.HasPrefix(listType, "List of ") {
        v.addError("cannot use 'set' on list variable '$%s' — use 'add ... to $%s' or re-assign via create list (CE7247)", name, name)
    }
```

确认 `strings` 包已 import（`validate_flow_body.go` 如果没有 import strings，需要加）。

- [ ] **Step 4: 运行确认通过**

```bash
go test ./mdl/executor/ -run "TestValidateCE7247" -v
```

预期：PASS

- [ ] **Step 5: Commit**

```bash
git add mdl/executor/validate_flow_body.go mdl/executor/validate_flow_body_test.go
git commit -m "fix(validator): reject set on list variable (CE7247)"
```

---

## Task 5: CE0111 — validator 扁平 output variable 追踪

**Files:**
- Modify: `mdl/executor/validate_flow_body.go:71-119`
- Test: `mdl/executor/validate_flow_body_test.go`

背景：Mendix 微流是全局扁平 output variable 作用域，两个不同 if/else 分支里的同名 create/declare 活动在 Studio Pro 中冲突（CE0111）。当前 validator 用 cloneStringMap 隔离各分支，导致漏报。

Go map 是引用类型，`scoped := *v` 后 `scoped.flatOutputVarNames` 和 `v.flatOutputVarNames` 指向同一底层 map，无需修改 `validateScopedStatements` 本身。

- [ ] **Step 1: 写失败测试**

```go
func TestValidateCE0111_BothBranches(t *testing.T) {
    // if true then $Result = create ... else $Result = create ... end if
    // 两个分支都声明 $Result → CE0111
    body := []ast.MicroflowStatement{
        &ast.IfStmt{
            Condition: &ast.LiteralExpr{Value: "true"},
            ThenBody: []ast.MicroflowStatement{
                &ast.CreateObjectStmt{
                    Variable:   "Result",
                    EntityType: ast.QualifiedName{Module: "M", Name: "CustomerBasicDto"},
                },
            },
            ElseBody: []ast.MicroflowStatement{
                &ast.CreateObjectStmt{
                    Variable:   "Result",
                    EntityType: ast.QualifiedName{Module: "M", Name: "CustomerBasicDto"},
                },
            },
        },
    }
    errs := validateFlowBody(nil, body)
    if len(errs) == 0 {
        t.Error("expected CE0111 for duplicate $Result in if/else branches, got none")
    }
}

func TestValidateCE0111_FlatScope_StillCaught(t *testing.T) {
    // 确保原有平铺作用域检测不退化
    body := []ast.MicroflowStatement{
        &ast.CreateObjectStmt{Variable: "Result", EntityType: ast.QualifiedName{Module: "M", Name: "Dto"}},
        &ast.CreateObjectStmt{Variable: "Result", EntityType: ast.QualifiedName{Module: "M", Name: "Dto"}},
    }
    errs := validateFlowBody(nil, body)
    if len(errs) == 0 {
        t.Error("flat-scope CE0111 regression")
    }
}
```

- [ ] **Step 2: 运行确认失败**

```bash
go test ./mdl/executor/ -run "TestValidateCE0111" -v
```

预期：`TestValidateCE0111_BothBranches` FAIL，`TestValidateCE0111_FlatScope_StillCaught` PASS

- [ ] **Step 3: 修复 flowValidator — 添加 flatOutputVarNames**

在 `validate_flow_body.go` 的 `flowValidator` struct 中添加一个字段：

```go
type flowValidator struct {
    varTypes           map[string]string
    declaredVars       map[string]string
    flatOutputVarNames map[string]bool  // ← 新增：跨所有分支共享，不 clone
    errors             []string
}
```

在 `newFlowValidator()` 中初始化：

```go
func newFlowValidator() *flowValidator {
    return &flowValidator{
        varTypes:           make(map[string]string),
        declaredVars:       make(map[string]string),
        flatOutputVarNames: make(map[string]bool),  // ← 新增
    }
}
```

修改 `validateOutputVariable`，使用 `flatOutputVarNames`（Go map 引用语义保证跨分支共享）：

```go
func (v *flowValidator) validateOutputVariable(varName, statement string) {
    if varName == "" {
        return
    }
    name := strippedVarName(varName)
    if v.flatOutputVarNames[name] {
        v.addError("duplicate variable name '$%s' — %s output variable is already declared in this microflow (CE0111)", name, statement)
        return
    }
    v.flatOutputVarNames[name] = true
}
```

修改 `DeclareStmt` handler，也使用 `flatOutputVarNames` 做全局检查：

```go
case *ast.DeclareStmt:
    name := s.Variable
    if v.flatOutputVarNames[name] {
        v.addError("duplicate variable name '$%s' — variable is already declared in this microflow (CE0111)", name)
    } else {
        v.flatOutputVarNames[name] = true
    }
    if s.Type.EntityRef != nil {
        v.varTypes[s.Variable] = s.Type.EntityRef.Module + "." + s.Type.EntityRef.Name
    } else {
        v.declaredVars[s.Variable] = s.Type.Kind.String()
    }
```

同理修改 `CreateListStmt` handler（也有 `validateOutputVariable` 调用，已覆盖）。

**注意**：`validateScopedStatements` 不需要改动 — Go map 是引用类型，`scoped := *v` 后 `scoped.flatOutputVarNames` 指向同一底层 map，分支内的记录自动对父作用域可见。

- [ ] **Step 4: 运行确认通过**

```bash
go test ./mdl/executor/ -run "TestValidateCE0111" -v
```

预期：两个测试均 PASS

- [ ] **Step 5: 全量非集成测试（确保无退化）**

```bash
go test ./mdl/executor/ -v 2>&1 | grep -E "FAIL|PASS|ok" | tail -20
```

预期：全部 PASS

- [ ] **Step 6: Commit**

```bash
git add mdl/executor/validate_flow_body.go mdl/executor/validate_flow_body_test.go
git commit -m "fix(validator): use flat output-var tracking for CE0111 across if/else branches"
```

---

## Task 6: CE0053 — 非持久化实体 create list 检查（写入 + 参照验证）

**Files:**
- Modify: `mdl/executor/flowbuilder_gen.go` — 新增 `nonPersistentEntities` 字段 + `isNonPersistentEntity()` 方法
- Modify: `mdl/executor/flowbuilder_actions_gen.go:155-167` — `addCreateListActionGen` 加检查
- Modify: `mdl/executor/validate.go` — 新增 `buildNonPersistentEntityQualifiedNames()` + 扩展 entity 验证

### 6a — Gen builder 写入时拦截

- [ ] **Step 1: 写失败测试（unit，不需 integration）**

在 `flowbuilder_actions_gen_test.go` 中添加（使用 `newActionTestFb()` 已有 helper，直接预置 `nonPersistentEntities` map 绕过 backend 依赖）：

```go
func TestAddCreateListActionGen_NonPersistentEntity_ReportsError(t *testing.T) {
    fb := newActionTestFb()
    // 直接预置 nonPersistentEntities，绕过 backend 调用（backend=nil 时 isNonPersistentEntity 返回 false）
    fb.nonPersistentEntities = map[string]bool{"M.NonPersistDto": true}

    stmt := &ast.CreateListStmt{
        Variable:   "Items",
        EntityType: ast.QualifiedName{Module: "M", Name: "NonPersistDto"},
    }
    fb.addCreateListActionGen(stmt)

    if len(fb.errors) == 0 {
        t.Error("expected CE0053 error for non-persistent entity in create list, got none")
    }
    if len(fb.objects) != 0 {
        t.Error("expected no action created for non-persistent entity (should return early)")
    }
}

func TestAddCreateListActionGen_PersistentEntity_NoError(t *testing.T) {
    fb := newActionTestFb()
    // PersistDto 不在 nonPersistentEntities 中 → 视为持久化
    fb.nonPersistentEntities = map[string]bool{"M.NonPersistDto": true}

    stmt := &ast.CreateListStmt{
        Variable:   "Items",
        EntityType: ast.QualifiedName{Module: "M", Name: "PersistDto"},
    }
    fb.addCreateListActionGen(stmt)

    if len(fb.errors) != 0 {
        t.Errorf("expected no error for persistent entity, got: %v", fb.errors)
    }
    if len(fb.objects) == 0 {
        t.Error("expected action to be created for persistent entity")
    }
}
```

- [ ] **Step 2: 在 flowbuilder_gen.go 中添加 nonPersistentEntities 字段**

在 `flowBuilderGen` struct 末尾添加两个字段（`mdl/executor/flowbuilder_gen.go`）：

```go
// nonPersistentEntities is lazily loaded on first isNonPersistentEntity call.
// nil means not yet loaded; empty map means loaded but all entities are persistent.
nonPersistentEntities map[string]bool
```

在同文件（或 `flowbuilder_assoc_lookup_gen.go`，保持一致）添加方法：

```go
// isNonPersistentEntity reports whether qualifiedName (Module.Entity) refers to
// a non-persistent entity. Lazily loads from fb.backend on first call.
// Returns false when backend is nil or the entity is not found.
func (fb *flowBuilderGen) isNonPersistentEntity(qualifiedName string) bool {
    if fb.backend == nil {
        return false
    }
    if fb.nonPersistentEntities == nil {
        fb.nonPersistentEntities = loadNonPersistentEntitySet(fb.backend, fb.hierarchy)
    }
    return fb.nonPersistentEntities[qualifiedName]
}

// loadNonPersistentEntitySet builds the set of non-persistent entity qualified names
// by walking all domain models via the backend.
func loadNonPersistentEntitySet(b backend.FullBackend, h *ContainerHierarchy) map[string]bool {
    result := make(map[string]bool)
    dms, err := b.ListDomainModelsGen()
    if err != nil || h == nil {
        return result
    }
    for _, dm := range dms {
        if dm == nil {
            continue
        }
        modName := h.GetModuleName(h.FindModuleID(model.ID(dm.ID())))
        if modName == "" {
            continue
        }
        for _, entityElem := range dm.EntitiesItems() {
            ent, ok := entityElem.(*genDm.Entity)
            if !ok {
                continue
            }
            if !entityPersistableGen(ent) {  // 复用已有函数（flowbuilder_assoc_lookup_gen.go:324）
                result[modName+"."+ent.Name()] = true
            }
        }
    }
    return result
}
```

所需 import（`flowbuilder_gen.go` 顶部）：`genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"`（如果不已有）。

- [ ] **Step 3: 在 addCreateListActionGen 中添加检查**

在 `flowbuilder_actions_gen.go` 的 `addCreateListActionGen`（line ~155）中，在写入 action 之前：

```go
func (fb *flowBuilderGen) addCreateListActionGen(s *ast.CreateListStmt) element.ID {
    entityQN := ""
    if s.EntityType.Module != "" && s.EntityType.Name != "" {
        entityQN = s.EntityType.Module + "." + s.EntityType.Name
    }
    // ▼ CE0053 check
    if entityQN != "" && fb.isNonPersistentEntity(entityQN) {
        fb.addError("cannot create list of non-persistent entity '%s' (CE0053): "+
            "Mendix does not allow list variables for non-persistent entities; "+
            "pass the list as a microflow parameter instead", entityQN)
        return ""
    }
    // ▲ CE0053 check
    if fb.varTypes != nil && entityQN != "" {
        fb.varTypes[s.Variable] = "List of " + entityQN
    }
    // ... 后续不变 ...
```

- [ ] **Step 4: 编译验证**

```bash
go build ./mdl/executor/
```

预期：编译通过

- [ ] **Step 5: Commit（6a 部分）**

```bash
git add mdl/executor/flowbuilder_gen.go mdl/executor/flowbuilder_actions_gen.go
git commit -m "fix(mf-builder): reject create list of non-persistent entity at write time (CE0053)"
```

### 6b — 参照验证阶段报错

- [ ] **Step 6: 写失败测试（validate.go 侧）**

在 `validate.go` 的测试文件（或 `validate_flow_body_test.go`）添加：

```go
// 验证：mxcli check -p 时，create list of 非持久化实体在参照验证阶段报错
// （集成测试，需要连接 MPR）
// 此处用 flowRefCollector 的单元测试验证 source 字段被正确设置
func TestFlowRefCollector_CreateListStmt_HasSource(t *testing.T) {
    stmts := []ast.MicroflowStatement{
        &ast.CreateListStmt{
            Variable:   "L",
            EntityType: ast.QualifiedName{Module: "M", Name: "NonPersistDto"},
        },
    }
    c := &flowRefCollector{}
    c.collectFromStatements(stmts)
    if len(c.entities) != 1 {
        t.Fatalf("expected 1 entity ref, got %d", len(c.entities))
    }
    if c.entities[0].source != "create list of" {
        t.Errorf("source = %q, want %q", c.entities[0].source, "create list of")
    }
}
```

- [ ] **Step 7: 运行确认通过（source 已正确设置）**

```bash
go test ./mdl/executor/ -run "TestFlowRefCollector_CreateListStmt_HasSource" -v
```

预期：PASS（此 source 在现有代码中已正确设置）

- [ ] **Step 8: 添加 buildNonPersistentEntityQualifiedNames 并扩展 entity 验证**

在 `validate.go` 的 `buildEntityQualifiedNames` 之后添加新函数：

```go
// buildNonPersistentEntityQualifiedNames returns the set of non-persistent entity
// qualified names in the project. Used for CE0053 detection.
func buildNonPersistentEntityQualifiedNames(ctx *ExecContext) map[string]bool {
    result := make(map[string]bool)
    h, err := getHierarchy(ctx)
    if err != nil {
        return result
    }
    dms, err := cachedDomainModelsGen(ctx)
    if err != nil {
        return result
    }
    for _, dm := range dms {
        if dm == nil {
            continue
        }
        modName := h.GetModuleName(h.FindModuleID(model.ID(dm.ID())))
        if modName == "" {
            continue
        }
        for _, entityElem := range dm.EntitiesItems() {
            ent, ok := entityElem.(*genDm.Entity)
            if !ok {
                continue
            }
            if !entityPersistableGen(ent) {
                result[modName+"."+ent.Name()] = true
            }
        }
    }
    return result
}
```

在 `validateFlowBodyReferences` 的 entity 验证块中，扩展 "create list of" 检查：

```go
if len(refs.entities) > 0 {
    known := buildEntityQualifiedNames(ctx)
    nonPersistent := buildNonPersistentEntityQualifiedNames(ctx) // ← 新增
    for _, ref := range refs.entities {
        if !known[ref.name] && !sc.entities[ref.name] {
            errors = append(errors, fmt.Sprintf("entity not found: %s (referenced by %s)", ref.name, ref.source))
        } else if ref.source == "create list of" && nonPersistent[ref.name] { // ← 新增
            errors = append(errors, fmt.Sprintf(
                "entity '%s' is non-persistent: cannot create list of non-persistent entity (CE0053)", ref.name))
        }
    }
}
```

- [ ] **Step 9: 编译验证**

```bash
go build ./mdl/executor/
```

- [ ] **Step 10: Commit（6b 部分）**

```bash
git add mdl/executor/validate.go
git commit -m "fix(validator): report CE0053 for create list of non-persistent entity in reference check"
```

---

## 最终验证

- [ ] **全量非集成测试**

```bash
go test ./mdl/executor/ 2>&1 | tail -5
```

预期：`ok github.com/mendixlabs/mxcli/mdl/executor`

- [ ] **全量集成测试**

```bash
go test ./mdl/executor/ -tags integration 2>&1 | tail -10
```

预期：全部 PASS

- [ ] **编译 CLI**

```bash
make build 2>&1 | tail -5
```

预期：无错误
