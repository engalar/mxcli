# 模块 07：AI 协作指南 — 审批工作流

## 本模块的设计选择

**本模块使用微流状态机实现升级审批，而非 Mendix Workflow Engine。**

原因：Workflow Engine 的 MDL 语法复杂，学习曲线较陡。微流状态机能清晰展示审批逻辑，
是你在 Mendix 项目中最常见的实现方式，也是理解 Workflow Engine 的基础。

Mendix Workflow Engine 的 MDL 实现请参考 `参考实现/escalation-workflow.mdl`。

## 与 Claude 协作的步骤

### Step 1：升级申请实体

```
帮我用 MDL 实现 HD.EscalationRequest 实体：
- Reason：字符串，不能为空，升级原因
- RequestedAt：日期时间，申请时间
- ApprovalStatus：枚举（Pending/Approved/Rejected），默认 Pending
- RejectionReason：字符串，可为空

关联：HD.EscalationRequest_Ticket（from EscalationRequest to Ticket）
```

### Step 2：三个审批微流

```
帮我实现三个微流：
1. HD.ACT_StartEscalation($Ticket, $Reason)：
   - 前置：Ticket.Status = InProgress
   - 创建 EscalationRequest，设置 Reason 和 RequestedAt

2. HD.ACT_Escalation_Approve($EscalationRequest)：
   - 设置 ApprovalStatus = Approved
   - 把关联工单的 Priority 改为 Critical
   - Commit 两个对象

3. HD.ACT_Escalation_Reject($EscalationRequest, $Reason)：
   - 设置 ApprovalStatus = Rejected，RejectionReason = $Reason
   - Commit
```

### Step 3：弹窗页面的正确写法（关键）

Mendix 的 dataview **不能**用 `datasource: new HD.Entity` 这种语法。
正确做法是：先用一个微流创建对象，再 `show page` 把对象作为参数传进去，页面用
`datasource: $参数` 绑定。

本模块两个弹窗都遵循这个模式：

| 弹窗 | 打开它的微流 | 页面参数 | 保存动作 |
|------|------------|---------|---------|
| EscalationStart_Form | `HD.ACT_OpenEscalationForm($Ticket)` | `$EscalationRequest` | `microflow HD.ACT_StartEscalation_FromObject` |
| EscalationReject_Form | `HD.ACT_OpenRejectForm($EscalationRequest)` | `$EscalationRequest` | `microflow HD.ACT_Escalation_Reject_FromObject` |

`ACT_OpenEscalationForm` 在创建对象**之前**先校验工单状态（必须 InProgress），
所以前置检查发生在打开表单这一步，而不是保存这一步——用户体验更好。

### Step 4：常见坑

| 坑 | 解决 |
|----|------|
| `datasource: new` 语法无效 | 用"创建对象的微流 + show page + datasource: $参数"三段式 |
| 修改关联对象的属性 | 沿关联路径 retrieve（关联名即路径，结尾不带实体段、不带 limit）：`retrieve $Ticket from $EscalationRequest/HD.EscalationRequest_Ticket` |
| commit 顺序 | 先 commit EscalationRequest，再 commit Ticket |
| 拒绝原因怎么收集 | 直接编辑 EscalationRequest 自带的 RejectionReason 字段，无需额外的非持久实体 |
| `retrieve $T from $ER/HD.AssocName` | 关联检索 BSON key 有历史 bug，改用 `retrieve $T from HD.Ticket where [HD.Assoc/HD.EscalationRequest = $ER] limit 1` |
| `limit 0` | 不代表无限制，省略 LIMIT 子句即可 |

---

## 路径 B：原生 Mendix Workflow（选修）

### 何时选择原生 Workflow 而非微流

| 场景 | 推荐方案 |
|------|---------|
| 简单的状态机审批（批准/拒绝） | 微流（路径 A，本模块主路径） |
| 需要工作流收件箱（inbox）、任务分配 | 原生 Workflow |
| 需要超时自动处理（边界事件） | 原生 Workflow |
| 需要多人投票（majority / all） | 原生 Workflow |
| 需要并行处理多个审批路径 | 原生 Workflow |

### escalation-workflow.mdl 关键语法说明

#### 声明 Workflow 和上下文实体

```mdl
create or replace workflow HD.WF_TicketEscalation
  parameter $WorkflowContext: HD.EscalationRequest
{
  ...
}
```

`$WorkflowContext` 是 Workflow 贯穿始终的上下文对象，所有活动都可以访问它。

#### 用户任务

```mdl
user task UT_PrimaryReview 'Primary Manager Review'
  page HD.EscalationReview_Form
  params: { $WorkflowUserTask: System.WorkflowUserTask }
  targeting users microflow HD.WFA_GetManagerAssignees
  outcomes: approve, reject;
```

- `page` 指定审批页面，页面必须声明 `params: { $WorkflowUserTask: System.WorkflowUserTask }`
- `targeting users microflow` 指定分配微流（返回 `list of System.User`）
- `outcomes` 列出所有可能的结果分支

#### 多人投票决策

```mdl
multi user task UT_SeniorCommitteeReview 'Senior Committee Review'
  page HD.SeniorCommitteeReview_Form
  params: { $WorkflowUserTask: System.WorkflowUserTask }
  targeting users microflow HD.WFA_GetSeniorCommitteeMembers
  completion method majority
  outcomes: approve, reject;
```

`completion method majority` 表示超过一半成员完成任务即可继续（另有 `all` 要求所有人完成）。

#### 调用子工作流

```mdl
call workflow HD.WF_SUB_ManagerReview;
```

子工作流将复杂审批逻辑封装为独立单元，父工作流等待其完成后继续执行。

#### 高级模式：并行分支与等待通知

参考实现未包含这些模式（保持可运行），但 Mendix Workflow 支持：

**并行分支：** 多条路径并行执行，均完成后 Workflow 继续。
```mdl
parallel split
  path 1 { ... }
  path 2 { ... }
;
```

**等待通知：** 配合 `notify workflow` 使用，等待外部系统触发继续。
```mdl
wait for notification comment 'WaitForExternalSignal';
```

在另一个微流中调用 `notify workflow` 解锁等待的活动。

#### 边界事件（非中断型 / 中断型定时器）

边界事件通过 `alter workflow` 附加到已有活动上：

```mdl
-- 非中断型：12 小时后发送提醒，子工作流继续正常执行
alter workflow HD.WF_TicketEscalation
  insert boundary event on WF_SUB_ManagerReview@1
    non interrupting timer 'addHours([%CurrentDateTime%], 12)'
  call microflow HD.WFS_SendReminder;

-- 中断型：48 小时后终止子工作流，走拒绝分支
alter workflow HD.WF_TicketEscalation
  insert boundary event on WF_SUB_ManagerReview@1
    interrupting timer 'addHours([%CurrentDateTime%], 48)'
  call microflow HD.WFS_Reject;
```

- `non interrupting`：触发后活动继续，适合"提醒"场景
- `interrupting`：触发后活动终止，适合"超时自动处理"场景

#### 从微流触发 Workflow

```mdl
call workflow HD.WF_TicketEscalation (EscalationRequest = $ER);
```

在微流中用 `call workflow` 启动一个 Workflow 实例，传入上下文对象。

### WFA_ 微流签名规范（重要）

用户任务的 `targeting users microflow` 必须遵守固定签名：

```
参数：
  $workflow: System.Workflow
  $Context: HD.EscalationRequest   （与 Workflow 的 parameter 类型一致）

返回值：list of System.User
```

签名不符会导致 Studio Pro 报错，且错误信息不会指向微流签名问题，难以排查。

### Workflow 用户任务页面规范

所有用户任务页面必须在 `params` 中声明 WorkflowUserTask 参数：

```mdl
create or replace page HD.EscalationReview_Form
  params: { $WorkflowUserTask: System.WorkflowUserTask }
  ...
```

页面通过 `$WorkflowUserTask` 获取任务信息（分配人、截止时间等），也用于提交审批结果。
