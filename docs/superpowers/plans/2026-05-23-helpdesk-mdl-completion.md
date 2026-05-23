# Helpdesk MDL 完整补全计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` 从"回归基线草稿"升级为覆盖所有已实现 MDL 功能的完整端到端示例，同时实现两个缺失引擎功能（`completion method` + `GenerateJumpTo/ApplyJumpTo`）。

**Architecture:** 分 4 个阶段：Phase A 纯 MDL 内容更新（引擎已支持）；Phase B 多用户任务完成方法扩展（grammar→AST→visitor→executor→MDL）；Phase C GenerateJumpTo/ApplyJumpTo 全栈实现（从零新增）；Phase D 补充 MDL 功能覆盖示例（loop、delete、error handling、真实表单页面）。每个任务独立可测，commit 粒度精确。

**Tech Stack:** Go 1.26, ANTLR4（`make grammar` 重新生成 parser），`modelsdk/gen/microflows`（gen 层类型），`mdl/ast`，`mdl/visitor`，`mdl/executor`，`mdl-examples/use-cases/helpdesk/helpdesk-app.mdl`

---

## 文件变更地图

| 文件 | Phase | 变更类型 |
|------|-------|---------|
| `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` | A/B/C/D | 修改（主要变更目标）|
| `mdl/grammar/domains/MDLWorkflow.g4` | B | 修改（添加 `completion method` 子句）|
| `mdl/grammar/domains/MDLMicroflow.g4` | C | 修改（添加 `generate jump to` / `apply jump to` 规则）|
| `mdl/ast/ast_workflow.go` | B | 修改（`WorkflowUserTaskNode` 添加 `CompletionMethod` 字段）|
| `mdl/ast/ast_microflow_workflow.go` | C | 修改（新增 `GenerateJumpToStmt` / `ApplyJumpToStmt`）|
| `mdl/visitor/visitor_workflow.go` | B | 修改（解析 `completion method` 子句）|
| `mdl/visitor/visitor_microflow_workflow.go` | C | 修改（新增 buildGenerateJumpTo / buildApplyJumpTo）|
| `mdl/executor/cmd_workflows_write_gen2.go` | B | 修改（`buildMultiUserTaskGenActivity` 设置 `CompletionCriteria`）|
| `mdl/executor/flowbuilder_workflow_gen.go` | C | 修改（新增 addGenerateJumpToActionGen / addApplyJumpToActionGen）|
| `mdl/executor/flowbuilder_dispatch_gen.go` | C | 修改（dispatch 新 AST 节点）|
| `mdl/executor/cmd_microflows_format_workflow_gen.go` | C | 修改（新增 format 函数）|
| `mdl/executor/cmd_microflows_format_action_gen.go` | C | 修改（dispatch format）|
| `mdl/executor/nanoflow_validation.go` | C | 修改（白名单阻止 nanoflow 调用这两个活动）|
| `mdl/executor/cmd_workflows_gen.go` | B | 修改（format multi-user task with completion method）|

---

## Phase A — 引擎已实现，替换 helpdesk TODO stubs

### Task A1: 替换 10 个工作流微流活动 TODO stub

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl:762-913`

这 10 个微流当前用 `log info '...'` 占位。真实 MDL 语法已在引擎中完整实现（`flowbuilder_workflow_gen.go`）。

- [ ] **Step 1: 替换 ACT_Workflow_ChangeState（3个操作）**

将 `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` 中 `HD.ACT_Workflow_ChangeState` 整体替换为：

```mdl
create or modify microflow HD.ACT_Workflow_ChangeState
  ($Workflow: System.Workflow, $Operation: string)
begin
  @position(200, 200)
  if $Operation = 'Pause' then
    workflow operation pause $Workflow;
  else
    if $Operation = 'Unpause' then
      workflow operation unpause $Workflow;
    else
      if $Operation = 'Abort' then
        workflow operation abort $Workflow reason 'Administratively aborted';
      end if;
    end if;
  end if;
  @position(800, 200)
  return;
end;
/
```

- [ ] **Step 2: 替换 ACT_Workflow_CompleteTask**

```mdl
create or modify microflow HD.ACT_Workflow_CompleteTask
  ($UserTask: System.WorkflowUserTask, $Outcome: string)
begin
  @position(200, 200)
  set task outcome $UserTask $Outcome;
  @position(400, 200)
  return;
end;
/
```

注意：`SET TASK OUTCOME $UserTask $Outcome;` — grammar 规则是 `SET TASK OUTCOME VARIABLE STRING_LITERAL`，所以 outcome 值必须是字符串字面量而非变量。需要改为硬编码 outcome 示例：

```mdl
create or modify microflow HD.ACT_Workflow_CompleteTask
  ($UserTask: System.WorkflowUserTask, $Outcome: string)
begin
  @position(200, 200)
  -- Demonstrates SET TASK OUTCOME with a literal outcome value.
  -- In production, dispatch on $Outcome with if/else.
  set task outcome $UserTask 'Approve';
  @position(400, 200)
  return;
end;
/
```

- [ ] **Step 3: 替换 ACT_Workflow_GetContext**

```mdl
create or modify microflow HD.ACT_Workflow_GetContext
  ($Workflow: System.Workflow)
  returns HD.EscalationRequest as $Context
begin
  @position(200, 200)
  $Context = get workflow data $Workflow as HD.WF_TicketEscalation;
  @position(400, 200)
  return $Context;
end;
/
```

- [ ] **Step 4: 替换 DS_WorkflowInstances**

```mdl
create or modify microflow HD.DS_WorkflowInstances
  ($EscalationRequest: HD.EscalationRequest)
  returns list of System.Workflow as $Workflows
begin
  @position(200, 200)
  $Workflows = get workflows for $EscalationRequest;
  @position(400, 200)
  return $Workflows;
end;
/
```

- [ ] **Step 5: 替换 ACT_Workflow_GetHistory**

```mdl
create or modify microflow HD.ACT_Workflow_GetHistory
  ($Workflow: System.Workflow)
  returns list of System.WorkflowActivityRecord as $Records
begin
  @position(200, 200)
  $Records = get workflow activity records $Workflow;
  @position(400, 200)
  return $Records;
end;
/
```

- [ ] **Step 6: 替换 ACT_Workflow_ShowTaskPage**

```mdl
create or modify microflow HD.ACT_Workflow_ShowTaskPage
  ($UserTask: System.WorkflowUserTask)
begin
  @position(200, 200)
  open user task $UserTask;
  @position(400, 200)
  return;
end;
/
```

- [ ] **Step 7: 替换 ACT_Workflow_ShowAdminPage**

```mdl
create or modify microflow HD.ACT_Workflow_ShowAdminPage
  ($Workflow: System.Workflow)
begin
  @position(200, 200)
  open workflow $Workflow;
  @position(400, 200)
  return;
end;
/
```

- [ ] **Step 8: 替换 ACT_Workflow_Lock**

```mdl
create or modify microflow HD.ACT_Workflow_Lock
  ($Workflow: System.Workflow)
begin
  @position(200, 200)
  lock workflow $Workflow;
  @position(400, 200)
  return;
end;
/
```

- [ ] **Step 9: 替换 ACT_Workflow_Unlock**

```mdl
create or modify microflow HD.ACT_Workflow_Unlock
  ()
begin
  @position(200, 200)
  unlock workflow all;
  @position(400, 200)
  return;
end;
/
```

- [ ] **Step 10: 替换 ACT_Workflow_Notify**

```mdl
create or modify microflow HD.ACT_Workflow_Notify
  ($Workflow: System.Workflow)
  returns boolean as $Notified
begin
  @position(200, 200)
  $Notified = notify workflow $Workflow;
  @position(400, 200)
  return $Notified;
end;
/
```

- [ ] **Step 11: 语法检查**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
```

期望：0 errors。

- [ ] **Step 12: Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk): replace 10 workflow activity TODO stubs with real MDL syntax"
```

---

### Task A2: 解注释用户角色、demo users 和项目安全设置

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl:1185-1197`

- [ ] **Step 1: 替换注释块为真实 MDL**

将文件末尾 `-- TODO: User roles require...` 注释块替换为真实语句：

```mdl
-- MARK: Security — User Roles

create or modify user role Customer (System.User, HD.CustomerRole, KB.Reader);
create or modify user role Agent (System.User, HD.AgentRole, KB.Contributor);
create or modify user role Manager (System.User, HD.ManagerRole, KB.Contributor);

-- MARK: Security — Demo Users

alter project security demo users on;

create or modify demo user 'demo_customer@helpdesk.test' password 'Demo1234!' (Customer);
create or modify demo user 'demo_agent@helpdesk.test'    password 'Demo1234!' (Agent);
create or modify demo user 'demo_manager@helpdesk.test'  password 'Demo1234!' (Manager);
```

- [ ] **Step 2: 语法检查**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
```

期望：0 errors。

- [ ] **Step 3: Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk): add user roles and demo users (executor fully implemented)"
```

---

### Task A3: 修复常量引用表达式（@HD.SLA_HIGH_HOURS）

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl:241-258`

README 中承诺"常量引用：`@HD.SLA_HIGH_HOURS` 在表达式中"，但代码硬编码了 `2` 和 `8`。`expressionToString` 已支持 `ConstantRefExpr`，gen flowbuilder 通过字符串传递给 Mendix，Mendix 接受 `@HD.SLA_HIGH_HOURS` 语法。

- [ ] **Step 1: 替换 ACT_Ticket_Submit 中的硬编码 SLA 小时数**

将 `HD.ACT_Ticket_Submit` 中的 if/else 链替换为：

```mdl
  @position(600, 200)
  if $Ticket/Priority = HD.TicketPriority.Critical then
    change $Ticket (
      Status   = HD.TicketStatus.Open,
      SLADueAt = addHours('[%CurrentDateTime%]', @HD.SLA_CRITICAL_HOURS)
    );
  else
    if $Ticket/Priority = HD.TicketPriority.High then
      change $Ticket (
        Status   = HD.TicketStatus.Open,
        SLADueAt = addHours('[%CurrentDateTime%]', @HD.SLA_HIGH_HOURS)
      );
    else
      change $Ticket (
        Status   = HD.TicketStatus.Open,
        SLADueAt = addHours('[%CurrentDateTime%]', 24)
      );
    end if;
  end if;
```

- [ ] **Step 2: 语法检查**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
```

期望：0 errors。

- [ ] **Step 3: Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk): use @HD.SLA_CRITICAL_HOURS / @HD.SLA_HIGH_HOURS constant refs in ACT_Ticket_Submit"
```

---

## Phase B — Multi-user task completion method

### Task B1: 在 AST 中添加 CompletionMethod 字段

**Files:**
- Modify: `mdl/ast/ast_workflow.go`

- [ ] **Step 1: 读取当前 WorkflowUserTaskNode 定义**

先确认字段列表（不修改代码，仅读取）：

```bash
grep -n "WorkflowUserTaskNode\|IsMultiUser\|CompletionMethod" mdl/ast/ast_workflow.go
```

期望：看到 `IsMultiUser bool` 存在，`CompletionMethod` 不存在。

- [ ] **Step 2: 添加 CompletionMethod 和 RequiredThreshold 字段**

在 `WorkflowUserTaskNode` 结构体中 `IsMultiUser bool` 之后添加：

```go
// CompletionMethod only for multi-user tasks.
// Values: "" (default = majority), "majority", "threshold", "consensus"
CompletionMethod    string
// RequiredThreshold: for "threshold" completion method, percentage 0-100.
RequiredThreshold   int
```

- [ ] **Step 3: Commit（仅 AST，无实现）**

```bash
go build ./mdl/ast/...
git add mdl/ast/ast_workflow.go
git commit -m "feat(ast): add CompletionMethod+RequiredThreshold to WorkflowUserTaskNode"
```

---

### Task B2: 扩展 MDLWorkflow.g4 grammar

**Files:**
- Modify: `mdl/grammar/domains/MDLWorkflow.g4`

- [ ] **Step 1: 在 MULTI USER TASK 规则中添加 completion method 子句**

在 `workflowUserTaskStmt` 的 MULTI USER TASK alternative 中，在 `DESCRIPTION` 行之后、`OUTCOMES` 行之前插入：

```antlr
      (COMPLETION METHOD workflowCompletionMethod)?
```

在文件中新增规则（放在 `workflowBoundaryEventClause` 之前）：

```antlr
workflowCompletionMethod
    : MAJORITY
    | THRESHOLD NUMBER_LITERAL PERCENT?
    | CONSENSUS
    ;
```

- [ ] **Step 2: 确保 MAJORITY、THRESHOLD、CONSENSUS、PERCENT 是词法 token 或关键字**

检查 `MDLLexer.g4`：

```bash
grep -n "MAJORITY\|THRESHOLD\|CONSENSUS\|PERCENT" mdl/grammar/MDLLexer.g4
```

若不存在，在 `MDLLexer.g4` 的关键字区域添加：

```antlr
MAJORITY    : 'majority';
THRESHOLD   : 'threshold';
CONSENSUS   : 'consensus';
PERCENT     : '%';
```

- [ ] **Step 3: 重新生成 parser**

```bash
make grammar
```

期望：无错误，`mdl/grammar/parser/` 下文件更新（不 commit 这些文件，它们在 .gitignore 中）。

- [ ] **Step 4: Commit（grammar 源文件，不含生成代码）**

```bash
git add mdl/grammar/domains/MDLWorkflow.g4 mdl/grammar/MDLLexer.g4
git commit -m "feat(grammar): add 'completion method majority/threshold/consensus' to multi user task"
```

---

### Task B3: 更新 visitor 解析 completion method

**Files:**
- Modify: `mdl/visitor/visitor_workflow.go`

- [ ] **Step 1: 找到 IsMultiUser 的解析位置**

```bash
grep -n "IsMultiUser\|MULTI\|utCtx" mdl/visitor/visitor_workflow.go | head -20
```

- [ ] **Step 2: 添加 CompletionMethod 解析**

在设置 `IsMultiUser: utCtx.MULTI() != nil` 之后添加：

```go
CompletionMethod: buildWorkflowCompletionMethod(utCtx.WorkflowCompletionMethod()),
RequiredThreshold: buildWorkflowCompletionThreshold(utCtx.WorkflowCompletionMethod()),
```

新增辅助函数（放在文件末尾）：

```go
// buildWorkflowCompletionMethod maps the grammar completion method clause
// to the canonical method string stored in the AST.
// Returns "" for the default (majority).
func buildWorkflowCompletionMethod(ctx parser.IWorkflowCompletionMethodContext) string {
	if ctx == nil {
		return ""
	}
	switch {
	case ctx.MAJORITY() != nil:
		return "majority"
	case ctx.THRESHOLD() != nil:
		return "threshold"
	case ctx.CONSENSUS() != nil:
		return "consensus"
	}
	return ""
}

// buildWorkflowCompletionThreshold extracts the integer percentage from
// a THRESHOLD completion method clause (e.g. "threshold 75 %").
// Returns 0 for non-threshold methods.
func buildWorkflowCompletionThreshold(ctx parser.IWorkflowCompletionMethodContext) int {
	if ctx == nil || ctx.THRESHOLD() == nil {
		return 0
	}
	if lit := ctx.NUMBER_LITERAL(); lit != nil {
		if n, err := strconv.Atoi(lit.GetText()); err == nil {
			return n
		}
	}
	return 0
}
```

- [ ] **Step 3: 验证编译**

```bash
go build ./mdl/visitor/...
```

- [ ] **Step 4: Commit**

```bash
git add mdl/visitor/visitor_workflow.go
git commit -m "feat(visitor): parse completion method clause for multi user task"
```

---

### Task B4: 更新 executor 设置 CompletionCriteria

**Files:**
- Modify: `mdl/executor/cmd_workflows_write_gen2.go`

- [ ] **Step 1: 找到 buildMultiUserTaskGenActivity 函数（约第 270 行）**

确认当前函数末尾（约第 290 行）没有设置 CompletionCriteria：

```bash
sed -n '270,295p' mdl/executor/cmd_workflows_write_gen2.go
```

- [ ] **Step 2: 在 `for _, ev := range ...` 之前添加 CompletionCriteria 设置**

```go
// Set completion criteria based on CompletionMethod.
// Default (empty string) maps to MajorityCompletionCriteria.
switch n.CompletionMethod {
case "", "majority":
    crit := genWf.NewMajorityCompletionCriteria()
    task.SetCompletionCriteria(crit)
case "threshold":
    crit := genWf.NewThresholdCompletionCriteria()
    pct := n.RequiredThreshold
    if pct <= 0 || pct > 100 {
        pct = 50 // sensible default
    }
    crit.SetRequiredPercentage(pct)
    task.SetCompletionCriteria(crit)
case "consensus":
    crit := genWf.NewConsensusCompletionCriteria()
    task.SetCompletionCriteria(crit)
}
```

注意：先验证 `genWf.NewThresholdCompletionCriteria` 和 `SetRequiredPercentage` 存在：

```bash
grep -n "NewThresholdCompletionCriteria\|SetRequiredPercentage\|NewConsensusCompletionCriteria" modelsdk/gen/workflows/types.go | head -10
```

如果 `ThresholdCompletionCriteria` 没有 `SetRequiredPercentage`，改用 `SetCompletionType("threshold")` 加上查阅实际可用 setter。

- [ ] **Step 3: 验证编译**

```bash
go build ./mdl/executor/...
```

- [ ] **Step 4: Commit**

```bash
git add mdl/executor/cmd_workflows_write_gen2.go
git commit -m "feat(executor): set CompletionCriteria on MultiUserTaskActivity based on completion method"
```

---

### Task B5: 更新 format/roundtrip 输出 completion method

**Files:**
- Modify: `mdl/executor/cmd_workflows_gen.go`

- [ ] **Step 1: 找到 MultiUserTaskActivity 的 format 输出（约第 742 行）**

```bash
grep -n "MultiUserTaskActivity\|multiUser\|multi user" mdl/executor/cmd_workflows_gen.go | head -10
```

- [ ] **Step 2: 在 multi user task 的输出行之后添加 completion method**

在渲染 `multi user task` 名称行之后、`outcomes` 之前，根据 `CompletionCriteria` 类型渲染：

```go
// Format completion method (only for multi-user tasks)
switch crit := task.CompletionCriteria().(type) {
case *genWf.ThresholdCompletionCriteria:
    lines = append(lines, indent+"  completion method threshold "+
        strconv.Itoa(int(crit.RequiredPercentage()))+" %")
case *genWf.ConsensusCompletionCriteria:
    lines = append(lines, indent+"  completion method consensus")
// MajorityCompletionCriteria is the default — omit for brevity (roundtrip stable)
}
```

- [ ] **Step 3: 验证编译 + 测试**

```bash
go build ./mdl/executor/...
go test ./mdl/executor/... -run TestWorkflow -v 2>&1 | tail -20
```

- [ ] **Step 4: Commit**

```bash
git add mdl/executor/cmd_workflows_gen.go
git commit -m "feat(executor): format completion method in multi user task MDL output"
```

---

### Task B6: 更新 helpdesk MDL 使用 completion method majority

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl:601-613`

- [ ] **Step 1: 替换 WF_SUB_ManagerReview 中的 multi user task**

将原来的：
```
      multi user task UT_SeniorReview 'Senior Board Review'
        page HD.EscalationReview_Form
        targeting users microflow HD.WFA_GetManagerAssignees
        -- TODO: 'completion method majority' not in grammar; use ALTER WORKFLOW
        outcomes
          'Approve' { call microflow HD.WFS_Approve; }
          'Reject'  { call microflow HD.WFS_Reject;  }
        ;
```

替换为：
```mdl
      multi user task UT_SeniorReview 'Senior Board Review'
        page HD.EscalationReview_Form
        targeting users microflow HD.WFA_GetManagerAssignees
        completion method majority
        outcomes
          'Approve' { call microflow HD.WFS_Approve; }
          'Reject'  { call microflow HD.WFS_Reject;  }
        ;
```

- [ ] **Step 2: 语法检查**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
```

- [ ] **Step 3: Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk): use 'completion method majority' on UT_SeniorReview multi user task"
```

---

## Phase C — GenerateJumpTo / ApplyJumpTo 全栈实现

### Task C1: 在 AST 添加两个新 Stmt 类型

**Files:**
- Modify: `mdl/ast/ast_microflow_workflow.go`

- [ ] **Step 1: 在文件末尾添加两个 AST 类型**

```go
// GenerateJumpToStmt represents:
//   [$Options =] GENERATE JUMP TO OPTIONS FOR $WorkflowVar AS Module.WF_Name
type GenerateJumpToStmt struct {
	OutputVariable   string
	WorkflowVariable string
	WorkflowQN       QualifiedName
	ErrorHandling    *ErrorHandlingClause
	Annotations      *ActivityAnnotations
}

func (*GenerateJumpToStmt) isMicroflowStatement() {}

// ApplyJumpToStmt represents:
//   [$Result =] APPLY JUMP TO OPTION $JumpOptionVar
type ApplyJumpToStmt struct {
	OutputVariable          string
	JumpOptionsVariable     string
	ErrorHandling           *ErrorHandlingClause
	Annotations             *ActivityAnnotations
}

func (*ApplyJumpToStmt) isMicroflowStatement() {}
```

- [ ] **Step 2: 验证编译**

```bash
go build ./mdl/ast/...
```

- [ ] **Step 3: Commit**

```bash
git add mdl/ast/ast_microflow_workflow.go
git commit -m "feat(ast): add GenerateJumpToStmt and ApplyJumpToStmt for workflow jump-to activities"
```

---

### Task C2: 扩展 MDLMicroflow.g4 语法规则

**Files:**
- Modify: `mdl/grammar/domains/MDLMicroflow.g4`

- [ ] **Step 1: 在 microflowStatement alternatives 中添加两条规则**

在 `unlockWorkflowStatement` 行之后添加：

```antlr
    | annotation* generateJumpToStatement SEMICOLON?
    | annotation* applyJumpToStatement SEMICOLON?
```

- [ ] **Step 2: 在文件末尾添加规则定义**

```antlr
// [$Options =] GENERATE JUMP TO OPTIONS FOR $wf AS Module.WF_Name;
generateJumpToStatement
    : (VARIABLE EQUALS)? GENERATE JUMP TO OPTIONS FOR VARIABLE AS qualifiedName onErrorClause?
    ;

// [$Result =] APPLY JUMP TO OPTION $options;
applyJumpToStatement
    : (VARIABLE EQUALS)? APPLY JUMP TO OPTION VARIABLE onErrorClause?
    ;
```

- [ ] **Step 3: 检查 GENERATE、APPLY、OPTIONS 是否已是 lexer token**

```bash
grep -n "^GENERATE\|^APPLY\|^OPTIONS\b" mdl/grammar/MDLLexer.g4
```

若不存在，在 MDLLexer.g4 关键字区添加缺失的 token（通常在 A-Z 字母排序区）：

```antlr
APPLY       : 'apply';
GENERATE    : 'generate';
OPTIONS     : 'options';
```

注意：`JUMP`, `TO`, `FOR`, `AS` 极可能已存在，先检查再添加，避免重复。

- [ ] **Step 4: 重新生成 parser**

```bash
make grammar
```

期望：无错误。

- [ ] **Step 5: Commit**

```bash
git add mdl/grammar/domains/MDLMicroflow.g4 mdl/grammar/MDLLexer.g4
git commit -m "feat(grammar): add generateJumpToStatement and applyJumpToStatement rules"
```

---

### Task C3: 实现 visitor 构建函数

**Files:**
- Modify: `mdl/visitor/visitor_microflow_workflow.go`

- [ ] **Step 1: 在文件末尾添加两个 build 函数**

```go
// buildGenerateJumpToStatement builds the AST for:
//   [$Out =] GENERATE JUMP TO OPTIONS FOR $wfVar AS Module.WF_Name
func buildGenerateJumpToStatement(ctx parser.IGenerateJumpToStatementContext) *ast.GenerateJumpToStmt {
	c := ctx.(*parser.GenerateJumpToStatementContext)
	stmt := &ast.GenerateJumpToStmt{}
	if c.VARIABLE(0) != nil && c.EQUALS() != nil {
		stmt.OutputVariable = strings.TrimPrefix(c.VARIABLE(0).GetText(), "$")
		stmt.WorkflowVariable = strings.TrimPrefix(c.VARIABLE(1).GetText(), "$")
	} else {
		stmt.WorkflowVariable = strings.TrimPrefix(c.VARIABLE(0).GetText(), "$")
	}
	if qnCtx := c.QualifiedName(); qnCtx != nil {
		stmt.WorkflowQN = buildQualifiedName(qnCtx)
	}
	return stmt
}

// buildApplyJumpToStatement builds the AST for:
//   [$Out =] APPLY JUMP TO OPTION $optionsVar
func buildApplyJumpToStatement(ctx parser.IApplyJumpToStatementContext) *ast.ApplyJumpToStmt {
	c := ctx.(*parser.ApplyJumpToStatementContext)
	stmt := &ast.ApplyJumpToStmt{}
	if c.VARIABLE(0) != nil && c.EQUALS() != nil {
		stmt.OutputVariable = strings.TrimPrefix(c.VARIABLE(0).GetText(), "$")
		stmt.JumpOptionsVariable = strings.TrimPrefix(c.VARIABLE(1).GetText(), "$")
	} else {
		stmt.JumpOptionsVariable = strings.TrimPrefix(c.VARIABLE(0).GetText(), "$")
	}
	return stmt
}
```

- [ ] **Step 2: 在 microflow statement dispatch 处调用这两个函数**

找到 visitor 中分发 microflow 语句的 switch（通常在 `visitor_microflow.go` 或 `visitor_microflow_workflow.go`）：

```bash
grep -n "GetWorkflowActivityRecordsStatement\|EnterWorkflowOperation\|buildGetWorkflow" mdl/visitor/visitor_microflow_workflow.go | head -10
```

在同一 dispatch 位置添加：

```go
if s := ctx.GenerateJumpToStatement(); s != nil {
    return buildGenerateJumpToStatement(s)
}
if s := ctx.ApplyJumpToStatement(); s != nil {
    return buildApplyJumpToStatement(s)
}
```

- [ ] **Step 3: 验证编译**

```bash
go build ./mdl/visitor/...
```

- [ ] **Step 4: Commit**

```bash
git add mdl/visitor/visitor_microflow_workflow.go
git commit -m "feat(visitor): parse generateJumpToStatement and applyJumpToStatement"
```

---

### Task C4: 实现 executor gen builder（写入 BSON）

**Files:**
- Modify: `mdl/executor/flowbuilder_workflow_gen.go`

- [ ] **Step 1: 在文件末尾添加两个 builder 函数**

```go
// addGenerateJumpToActionGen emits `[$Out =] generate jump to options for $wf as Mod.WF_Name;`.
func (fb *flowBuilderGen) addGenerateJumpToActionGen(s *ast.GenerateJumpToStmt) element.ID {
	action := genMf.NewGenerateJumpToOptionsAction()
	action.SetID(element.ID(types.GenerateID()))
	action.SetWorkflowVariable(s.WorkflowVariable)
	action.SetWorkflowQualifiedName(s.WorkflowQN.Module + "." + s.WorkflowQN.Name)
	if s.OutputVariable != "" {
		action.SetOutputVariableName(s.OutputVariable)
	}
	return fb.addActionGen(action)
}

// addApplyJumpToActionGen emits `[$Out =] apply jump to option $options;`.
func (fb *flowBuilderGen) addApplyJumpToActionGen(s *ast.ApplyJumpToStmt) element.ID {
	action := genMf.NewApplyJumpToOptionAction()
	action.SetID(element.ID(types.GenerateID()))
	action.SetWorkflowJumpToDetailsVariable(s.JumpOptionsVariable)
	if s.OutputVariable != "" {
		action.SetOutputVariableName(s.OutputVariable)
	}
	return fb.addActionGen(action)
}
```

- [ ] **Step 2: 验证 addActionGen 方法存在**

```bash
grep -n "func.*addActionGen" mdl/executor/flowbuilder_workflow_gen.go mdl/executor/flowbuilder_gen.go | head -5
```

如果方法名不同（例如 `addGenElement`），更新上述代码使用正确方法名。

- [ ] **Step 3: 验证编译**

```bash
go build ./mdl/executor/...
```

- [ ] **Step 4: Commit**

```bash
git add mdl/executor/flowbuilder_workflow_gen.go
git commit -m "feat(executor): add gen builders for GenerateJumpTo/ApplyJumpTo workflow actions"
```

---

### Task C5: 注册到 dispatch

**Files:**
- Modify: `mdl/executor/flowbuilder_dispatch_gen.go`
- Modify: `mdl/executor/nanoflow_validation.go`

- [ ] **Step 1: 在 dispatch switch 的 Workflow 区添加两个 case（约第 218 行之后）**

```go
case *ast.GenerateJumpToStmt:
    return fb.addGenerateJumpToActionGen(s)
case *ast.ApplyJumpToStmt:
    return fb.addApplyJumpToActionGen(s)
```

- [ ] **Step 2: 在 nanoflow_validation.go 的禁用列表中添加两个类型**

找到现有 workflow 活动的 `isMicroflowOnlyStatement` 函数（约第 77-86 行），在列表末尾添加：

```go
*ast.GenerateJumpToStmt,
*ast.ApplyJumpToStmt,
```

- [ ] **Step 3: 验证编译**

```bash
go build ./mdl/executor/...
```

- [ ] **Step 4: Commit**

```bash
git add mdl/executor/flowbuilder_dispatch_gen.go mdl/executor/nanoflow_validation.go
git commit -m "feat(executor): dispatch GenerateJumpTo/ApplyJumpTo; block in nanoflows"
```

---

### Task C6: 实现 format 函数（DESCRIBE roundtrip）

**Files:**
- Modify: `mdl/executor/cmd_microflows_format_workflow_gen.go`
- Modify: `mdl/executor/cmd_microflows_format_action_gen.go`

- [ ] **Step 1: 在 cmd_microflows_format_workflow_gen.go 末尾添加两个 format 函数**

```go
// formatGenerateJumpToOptionsActionGen emits
// `[$Out =] generate jump to options for $wf as Module.WF_Name;`
func formatGenerateJumpToOptionsActionGen(a *genMf.GenerateJumpToOptionsAction) string {
	wfVar := a.WorkflowVariable()
	wfQN := a.WorkflowQualifiedName()
	if outVar := a.OutputVariableName(); outVar != "" {
		return fmt.Sprintf("$%s = generate jump to options for $%s as %s;", outVar, wfVar, wfQN)
	}
	return fmt.Sprintf("generate jump to options for $%s as %s;", wfVar, wfQN)
}

// formatApplyJumpToOptionActionGen emits
// `[$Out =] apply jump to option $options;`
func formatApplyJumpToOptionActionGen(a *genMf.ApplyJumpToOptionAction) string {
	optVar := a.WorkflowJumpToDetailsVariable()
	if outVar := a.OutputVariableName(); outVar != "" {
		return fmt.Sprintf("$%s = apply jump to option $%s;", outVar, optVar)
	}
	return fmt.Sprintf("apply jump to option $%s;", optVar)
}
```

- [ ] **Step 2: 在 cmd_microflows_format_action_gen.go 的 type switch 中添加 case**

找到现有 workflow action 的 case 区（搜索 `GetWorkflowActivityRecordsAction`），在之后添加：

```go
case *genMf.GenerateJumpToOptionsAction:
    return formatGenerateJumpToOptionsActionGen(a)
case *genMf.ApplyJumpToOptionAction:
    return formatApplyJumpToOptionActionGen(a)
```

- [ ] **Step 3: 验证编译**

```bash
go build ./mdl/executor/...
```

- [ ] **Step 4: 运行现有 workflow 相关测试**

```bash
go test ./mdl/executor/... -run "Workflow\|workflow" -v 2>&1 | tail -30
```

- [ ] **Step 5: Commit**

```bash
git add mdl/executor/cmd_microflows_format_workflow_gen.go mdl/executor/cmd_microflows_format_action_gen.go
git commit -m "feat(executor): format GenerateJumpTo/ApplyJumpTo actions in MDL output (DESCRIBE roundtrip)"
```

---

### Task C7: 更新 helpdesk MDL 使用 GenerateJumpTo/ApplyJumpTo

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl:793-804`

- [ ] **Step 1: 替换 ACT_Workflow_JumpTo**

```mdl
create or modify microflow HD.ACT_Workflow_JumpTo
  ($Workflow: System.Workflow)
begin
  @position(200, 200)
  $JumpOptions = generate jump to options for $Workflow as HD.WF_TicketEscalation;
  @position(400, 200)
  apply jump to option $JumpOptions;
  @position(600, 200)
  return;
end;
/
```

- [ ] **Step 2: 语法检查**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
```

- [ ] **Step 3: Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk): use GenerateJumpTo/ApplyJumpTo in ACT_Workflow_JumpTo"
```

---

## Phase D — 补充 MDL 功能覆盖（loop、delete、error handling、真实页面）

### Task D1: Loop 循环——批量添加评论标签

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl`（在 KnowledgeBase Microflows 区之后新增）

- [ ] **Step 1: 新增 ACT_Ticket_MarkCommentsRead 微流（演示 loop）**

在 `KB.ACT_Article_Archive` 之后、`-- MARK: HelpDesk — Nanoflows` 之前插入：

```mdl
-- ACT_Ticket_MarkCommentsRead: loop over comments, mark internal ones as read
-- Demonstrates: loop $item in $List, change inside loop
create or modify microflow HD.ACT_Ticket_MarkCommentsRead
  ($Ticket: HD.Ticket)
begin
  @position(200, 200)
  retrieve $Comments from HD.TicketComment
    where [HD.TicketComment_Ticket/HD.Ticket = $Ticket and IsInternal = true]
    limit 100;
  @position(400, 200)
  loop $Comment in $Comments
    begin
      change $Comment (IsInternal = false);
      commit $Comment;
    end loop;
  @position(800, 200)
  return;
end;
/
```

- [ ] **Step 2: 语法检查**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
```

- [ ] **Step 3: Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk): add ACT_Ticket_MarkCommentsRead demonstrating loop activity"
```

---

### Task D2: Delete 操作——清理过期 EscalationRequests

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl`

- [ ] **Step 1: 新增 ACT_EscalationRequest_Cleanup 微流（演示 delete）**

在 `ACT_Ticket_MarkCommentsRead` 之后插入：

```mdl
-- ACT_EscalationRequest_Cleanup: delete closed escalations for a ticket
-- Demonstrates: retrieve list + loop + delete inside loop
create or modify microflow HD.ACT_EscalationRequest_Cleanup
  ($Ticket: HD.Ticket)
begin
  @position(200, 200)
  retrieve $Requests from HD.EscalationRequest
    where [HD.EscalationRequest_Ticket/HD.Ticket = $Ticket]
    limit 50;
  @position(400, 200)
  loop $Req in $Requests
    begin
      delete $Req;
    end loop;
  @position(800, 200)
  return;
end;
/
```

- [ ] **Step 2: 语法检查**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
```

- [ ] **Step 3: Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk): add ACT_EscalationRequest_Cleanup demonstrating delete activity"
```

---

### Task D3: Error handling / rollback 分支

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl`

- [ ] **Step 1: 新增 ACT_Article_Publish_Safe 微流（演示 on error rollback）**

在 `KB.ACT_Article_Archive` 之后插入：

```mdl
-- ACT_Article_Publish_Safe: wraps publish in error-handling rollback
-- Demonstrates: on error rollback pattern
create or modify microflow KB.ACT_Article_Publish_Safe
  ($Article: KB.Article)
  returns boolean as $Success
begin
  @position(200, 200)
  declare $Success boolean = false;
  on error rollback begin
    change $Article (
      Status      = KB.ArticleStatus.Published,
      PublishedAt = '[%CurrentDateTime%]'
    );
    commit $Article;
    set $Success = true;
  end on error;
  @position(600, 200)
  return $Success;
end;
/
```

- [ ] **Step 2: 语法检查**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
```

如果 `on error rollback begin ... end on error` 语法检查报错，先检查实际语法：

```bash
grep -n "on error\|onError\|error rollback\|ErrorHandling" mdl/grammar/domains/MDLMicroflow.g4 | head -10
```

用实际正确语法替换。

- [ ] **Step 3: Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk): add ACT_Article_Publish_Safe demonstrating on-error rollback pattern"
```

---

### Task D4: 真实编辑表单页面——HD.Ticket_NewEdit

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl:1043-1057`

- [ ] **Step 1: 替换 placeholder 为真实表单**

将 `HD.Ticket_NewEdit` 替换为：

```mdl
create page HD.Ticket_NewEdit
(
  title: 'New / Edit Ticket',
  layout: Atlas_Core.Atlas_Default,
  params: { $Ticket: HD.Ticket }
)
{
  layoutgrid lgMain {
    row rMain {
      column cMain (desktopwidth: 12) {
        dataview dvTicket (datasource: $Ticket) {
          textbox tbSubject      (attribute: Subject,     label: 'Subject',      required: true)
          textarea taDescription (attribute: Description, label: 'Description')
          combobox cbPriority    (attribute: Priority,    label: 'Priority')
          combobox cbStatus      (attribute: Status,      label: 'Status')
          datepicker dpSLADue    (attribute: SLADueAt,    label: 'SLA Due Date')
          actionbutton btnSave   (caption: 'Save')
          actionbutton btnCancel (caption: 'Cancel')
        }
      }
    }
  }
};
```

- [ ] **Step 2: 语法检查**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
```

- [ ] **Step 3: Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk): replace HD.Ticket_NewEdit placeholder with real edit form"
```

---

### Task D5: 真实编辑表单页面——KB.Article_NewEdit

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl:1058-1072`

- [ ] **Step 1: 替换 placeholder 为真实表单**

将 `KB.Article_NewEdit` 替换为：

```mdl
create page KB.Article_NewEdit
(
  title: 'New / Edit Article',
  layout: Atlas_Core.Atlas_Default,
  params: { $Article: KB.Article }
)
{
  layoutgrid lgMain {
    row rMain {
      column cMain (desktopwidth: 12) {
        dataview dvArticle (datasource: $Article) {
          textbox  tbTitle      (attribute: Title,   label: 'Title',   required: true)
          textarea taContent    (attribute: Content, label: 'Content')
          combobox cbStatus     (attribute: Status,  label: 'Status')
          actionbutton btnPublish (caption: 'Publish', microflow: KB.ACT_Article_Publish)
          actionbutton btnSave    (caption: 'Save')
          actionbutton btnCancel  (caption: 'Cancel')
        }
      }
    }
  }
};
```

- [ ] **Step 2: 语法检查**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
```

- [ ] **Step 3: Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk): replace KB.Article_NewEdit placeholder with real edit form"
```

---

### Task D6: 关联选择控件——在 Ticket_Detail 添加 Agent 选择

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl:996-1024`

- [ ] **Step 1: 在 Ticket_Detail dataview 中添加 referenceselector**

在 `dvTicket` 的 `actionbutton btnReopen` 之后添加：

```mdl
          referenceselector rsAgent (
            attribute: HD.Ticket_Agent,
            label: 'Assigned Agent',
            datasource: database HD.Agent
          )
```

- [ ] **Step 2: 语法检查**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
```

如果 `referenceselector` 的属性语法有差异，先查阅：

```bash
grep -rn "referenceselector" mdl-examples/doctype-tests/03-page-examples.mdl | head -10
```

使用实际有效语法。

- [ ] **Step 3: Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk): add referenceselector for Agent assignment in Ticket_Detail"
```

---

### Task D7: 聚合操作——统计超期工单数

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl`

- [ ] **Step 1: 新增 DS_OverdueTicketCount 微流（演示 aggregate）**

在 Phase D 微流块末尾插入：

```mdl
-- DS_OverdueTicketCount: aggregate list to count overdue tickets
-- Demonstrates: retrieve + aggregate list count
create or modify microflow HD.DS_OverdueTicketCount
  ()
  returns integer as $Count
begin
  @position(200, 200)
  retrieve $Overdue from HD.Ticket
    where [IsOverSLA = true and Status != 'Closed']
    limit 0;
  @position(400, 200)
  declare $Count integer = count($Overdue);
  @position(600, 200)
  return $Count;
end;
/
```

注意：Mendix 表达式 `count($list)` 是内置函数。若 MDL executor 使用 `aggregate $List count` 语法，改为：

```bash
grep -n "aggregate\|count(" mdl-examples/doctype-tests/02-microflow-examples.mdl | head -10
```

使用项目实际语法。

- [ ] **Step 2: 语法检查**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
```

- [ ] **Step 3: Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk): add DS_OverdueTicketCount demonstrating aggregate/count"
```

---

### Task D8: 运行完整测试套件并确认 mx check 干净

- [ ] **Step 1: 构建 mxcli**

```bash
make build
```

- [ ] **Step 2: 语法检查**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
```

期望：0 errors。

- [ ] **Step 3: 运行测试套件（含 report）**

```bash
make test 2>&1 | tail -30
```

期望：无新增 FAIL。

- [ ] **Step 4: 执行 helpdesk MDL 到测试项目（如可用）**

```bash
./bin/mxcli -p testdata/corpus-b/app.mpr \
  exec mdl-examples/use-cases/helpdesk/helpdesk-app.mdl 2>&1 | tail -20
```

```bash
~/.mxcli/mxbuild/11.6.4/modeler/mx check testdata/corpus-b/app.mpr \
  2>&1 | grep -i "StorageLoadException\|Invalid\|Error"
```

期望：无新增 StorageLoadException。

```bash
git restore testdata/corpus-b/
```

- [ ] **Step 5: 更新 README.md 中的占位规范章节**

删除 `README.md` 中 `## 注释占位规范` 下"当前占位项"列表（所有项目已实现），替换为：

```markdown
当前占位项：（无 — 所有已知缺口已在本轮实现）
```

- [ ] **Step 6: Commit**

```bash
git add mdl-examples/use-cases/helpdesk/
git commit -m "docs(helpdesk): update README — all placeholder items resolved"
```

---

## 自检 checklist（plan 自查）

- [x] Phase A：10 个 TODO stub 均已提供真实 MDL 语法（来自 grammar 规则确认）
- [x] Phase A：用户角色/demo users 使用已实现的 executor 函数
- [x] Phase A：`@HD.SLA_CRITICAL_HOURS` 使用 `ConstantRefExpr` 已支持的语法
- [x] Phase B：grammar 修改 → AST 扩展 → visitor → executor → MDL 全链路覆盖
- [x] Phase C：gen 类型（`NewGenerateJumpToOptionsAction` / `NewApplyJumpToOptionAction`）在 `modelsdk/gen/microflows/types.go` 已确认存在
- [x] Phase C：dispatch + nanoflow 禁用 + format 三处均已覆盖
- [x] Phase D：每个新微流均有语法检查步骤
- [x] `make grammar` 在 B2/C2 后显式调用
- [x] 所有 commit 粒度：单一原子变更
