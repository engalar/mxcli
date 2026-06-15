# 模块 07：AI 协作指南 — 审批工作流

## 本模块的设计选择

**本模块使用微流状态机实现升级审批，而非 Mendix Workflow Engine。**

原因：Workflow Engine 的 MDL 语法复杂，学习曲线较陡。微流状态机能清晰展示审批逻辑，
是你在 Mendix 项目中最常见的实现方式，也是理解 Workflow Engine 的基础。

Mendix Workflow Engine 的 MDL 实现请参考进阶文档（链接待补充）。

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
