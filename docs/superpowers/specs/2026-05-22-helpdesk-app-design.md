# HelpDesk + KnowledgeBase 应用基线设计规格

**日期：** 2026-05-22  
**用途：** 端到端回归基线（类似 `shop.mdl`），`mx check` 必须通零错误  
**目标文件：** `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl`  
**Mendix 版本要求：** 11.6+（工作流要求 11.0+）

---

## 1. 文件结构

单文件，按 `-- MARK:` 分节，顺序遵循 `mxcli import` 依赖拓扑：

```
MARK: KnowledgeBase — Module & Enumerations
MARK: KnowledgeBase — Entities
MARK: HelpDesk — Module & Enumerations
MARK: HelpDesk — Entities & Non-persistent
MARK: HelpDesk — Cross-Module Associations
MARK: HelpDesk — Constants
MARK: HelpDesk — Microflows (Status Machine)
MARK: KnowledgeBase — Microflows
MARK: HelpDesk — Nanoflows
MARK: KnowledgeBase — Nanoflows
MARK: HelpDesk — Sub-Workflow (WF_SUB_ManagerReview)
MARK: HelpDesk — Parent Workflow (WF_TicketEscalation)
MARK: HelpDesk — Workflow Support Microflows (13 activities)
MARK: Pages — KnowledgeBase
MARK: Pages — HelpDesk
MARK: Security — Module Roles
MARK: Security — User Roles & Grants
MARK: Navigation
```

预计总行数：~1100 行（含注释占位符）。

---

## 2. 域模型

### KnowledgeBase 模块（5 个实体）

| 实体 | 关键属性 | 边界覆盖 |
|------|----------|----------|
| `KB.Category` | Name(string 200), Description(string 500) | — |
| `KB.Tag` | Name(string 100, unique) | — |
| `KB.Article` | Title(string 500), Content(string), Status(KB.ArticleStatus), PublishedAt(datetime), ViewCount(integer) | 状态机目标 |
| `KB.ArticleTag` | （无额外属性） | **多对多**中间表 |
| `KB.ArticleRating` | Score(integer 1-5), Comment(string 1000) | XPath 行级过滤 |

**关联：**
- `KB.Category_Parent`：Category → Category，self-ref，**自引用**
- `KB.Article_Category`：Article → Category
- `KB.ArticleTag_Article`：ArticleTag → Article
- `KB.ArticleTag_Tag`：ArticleTag → Tag
- `KB.ArticleRating_Article`：ArticleRating → Article

**枚举：** `KB.ArticleStatus`（Draft / Published / Archived）

---

### HelpDesk 模块（5 持久化 + 1 非持久化）

| 实体 | 关键属性 | 边界覆盖 |
|------|----------|----------|
| `HD.Customer` | Name(200), Email(200), Company(200) | — |
| `HD.Agent` | Name(200), Email(200), IsActive(boolean) | — |
| `HD.Ticket` | Subject(500), Description(string), Status(enum), Priority(enum), SLADueAt(datetime), ResolvedAt(datetime), IsOverSLA(boolean) | 状态机主体 |
| `HD.TicketComment` | Content(string), IsInternal(boolean) | XPath 行级过滤 |
| `HD.EscalationRequest` | Reason(1000), RequestedAt(datetime), RejectionReason(1000) | 工作流上下文实体 |
| `HD.TicketSearch` *(非持久化)* | SubjectKeyword(200), StatusFilter(enum), PriorityFilter(enum) | **非持久化实体** |

**关联：**
- `HD.Ticket_Customer`：Ticket → Customer
- `HD.Ticket_Agent`：Ticket → Agent（nullable）
- `HD.TicketComment_Ticket`：TicketComment → Ticket
- `HD.EscalationRequest_Ticket`：EscalationRequest → Ticket
- `HD.Ticket_KBArticle`：Ticket → KB.Article（nullable，**跨模块**）

**枚举：**
- `HD.TicketStatus`：Draft / Open / InProgress / Resolved / Closed
- `HD.TicketPriority`：Low / Normal / High / Critical

---

## 3. 常量

| 常量 | 类型 | 默认值 | 用途 |
|------|------|--------|------|
| `HD.SLA_HIGH_HOURS` | integer | 8 | High 优先级 SLA 时限（小时） |
| `HD.SLA_CRITICAL_HOURS` | integer | 2 | Critical 优先级 SLA 时限（小时） |

在微流表达式中引用：`@HD.SLA_HIGH_HOURS`

---

## 4. 微流（状态机 + CRUD）

### HelpDesk 微流

| 微流 | 前缀 | 边界覆盖 |
|------|------|----------|
| `HD.ACT_Ticket_Submit` | ACT_ | Draft→Open，Subject 校验，SLADueAt 按 Priority+常量计算，rollback 分支 |
| `HD.ACT_Ticket_Assign` | ACT_ | 前置状态断言（必须为 Open），设 Agent，设 InProgress |
| `HD.ACT_Ticket_Resolve` | ACT_ | InProgress→Resolved，设 ResolvedAt，计算 IsOverSLA（时间比较） |
| `HD.ACT_Ticket_Reopen` | ACT_ | Resolved/Closed→Open，清空 ResolvedAt（**状态倒退**） |
| `HD.ACT_Ticket_Close` | ACT_ | Resolved→Closed |

### KnowledgeBase 微流

| 微流 | 前缀 | 边界覆盖 |
|------|------|----------|
| `KB.ACT_Article_Publish` | ACT_ | Content 非空校验，Draft→Published，设 PublishedAt |
| `KB.ACT_Article_Archive` | ACT_ | Published→Archived |

---

## 5. 纳流

> 注：纳流可完整使用对象活动（Create/Change/Commit/Retrieve），区别是每次操作为独立网络请求。

| 纳流 | 演示能力 |
|------|----------|
| `HD.NF_Ticket_QuickCreate` | Create 持久化对象 + Commit（客户端快速创建草稿工单） |
| `HD.NF_TicketSearch_Apply` | Retrieve from database（XPath 查询）+ 返回列表 |
| `HD.NF_Priority_GetLabel` | 纯计算返回 string（if/else 链），无 DB 操作 |
| `KB.NF_Article_FormatPreview` | Call microflow（调用服务端微流） |

---

## 6. 工作流

### 子工作流：`HD.WF_SUB_ManagerReview`

上下文实体：`HD.EscalationRequest`

```
Start
  → [Annotation] 说明审核目的
  → [User task] UT_PrimaryReview
       WFA_GetManagerAssignees（targeting microflow）
       Outcomes: Approve / Escalate
       Timer boundary（非中断，12h）→ [Call microflow WFS_SendReminder]
  → [Decision] 基于 outcome
      ├─ Approve → [Call microflow WFS_Approve] → End_Approved
      └─ Escalate
           → [Multi-user task] UT_SeniorReview
                Decision method: Majority
                Outcomes: Approve / Reject
           → [Decision]
               ├─ Approve → [Call microflow WFS_Approve] → End_Approved
               └─ Reject  → [Call microflow WFS_Reject]  → End_Rejected
```

### 父工作流：`HD.WF_TicketEscalation`

上下文实体：`HD.EscalationRequest`

```
Start
  → [Annotation] 工单升级审批全流程说明
  → [Call microflow WFS_Escalation_Initialize]
  → [Decision] 基于 Ticket.Priority（Enumeration）
      ├─ Critical → [Jump] → End_AutoApproved
      └─ Default  →
           [Wait for notification] WaitForManagerAvailable
               Timer boundary（非中断，12h，可重复）
               → [Call microflow WFS_SendReminder]
           → [Call workflow HD.WF_SUB_ManagerReview]
               Timer boundary（中断，48h）
               → [Call microflow WFS_AutoReject]
               → End_Timeout
           → [Parallel split]
               Path A: [Call microflow WFS_UpdateTicketPriority]
               Path B: [Call microflow WFS_NotifyAgent]
           → End_Completed

End_AutoApproved
End_Completed
End_Timeout
```

### 工作流元素覆盖清单

| 元素 | 工作流 |
|------|--------|
| Start / End（多个） | 两个 |
| Annotation | 父工作流 |
| User task（单用户） | 子工作流 |
| Multi-user task（Majority） | 子工作流 |
| Decision（Enumeration 类型） | 父工作流 |
| Decision（outcome 类型） | 子工作流 |
| Parallel split | 父工作流 |
| Call workflow（父子） | 父工作流 |
| Call microflow | 两个 |
| Wait for notification | 父工作流 |
| Jump | 父工作流 |
| Timer boundary（非中断 + 重复） | Wait for notification |
| Timer boundary（中断） | Call workflow |

---

## 7. 工作流配套微流（覆盖全部 13 个工作流活动）

| 微流 | 演示的微流工作流活动 |
|------|-------------------|
| `HD.ACT_StartEscalation` | **Call workflow**（触发父工作流） |
| `HD.WFC_EscalationRequest_OnCreate` | 工作流创建事件处理器（WFC_前缀） |
| `HD.WFA_GetManagerAssignees` | 工作流 targeting（WFA_前缀） |
| `HD.WFS_Escalation_Initialize` | 由工作流 Call microflow 调用（WFS_前缀） |
| `HD.WFS_Approve` | 由工作流调用，审批通过处理 |
| `HD.WFS_Reject` | 由工作流调用，审批拒绝处理 |
| `HD.WFS_SendReminder` | 由边界事件触发，发送提醒 |
| `HD.WFS_AutoReject` | 由超时边界事件触发 |
| `HD.WFS_UpdateTicketPriority` | 并行路径：更新工单优先级 |
| `HD.WFS_NotifyAgent` | 并行路径：通知 Agent |
| `HD.ACT_Workflow_Notify` | **Notify workflow** |
| `HD.ACT_Workflow_ChangeState` | **Change workflow state**（Pause/Unpause/Abort/Retry） |
| `HD.ACT_Workflow_CompleteTask` | **Complete user task** |
| `HD.ACT_Workflow_JumpTo` | **Generate jump-to options** + **Apply jump-to option** |
| `HD.ACT_Workflow_GetHistory` | **Retrieve workflow activity records** |
| `HD.ACT_Workflow_GetContext` | **Retrieve workflow context** |
| `HD.DS_WorkflowInstances` | **Retrieve workflows**（数据源微流） |
| `HD.ACT_Workflow_Lock` | **Lock workflow** |
| `HD.ACT_Workflow_Unlock` | **Unlock workflow** |
| `HD.ACT_Workflow_ShowAdminPage` | **Show workflow admin page** |
| `HD.ACT_Workflow_ShowTaskPage` | **Show user task page** |

---

## 8. 页面

### KnowledgeBase（2 个页面）

| 页面 | 布局 | 组件 | 覆盖 |
|------|------|------|------|
| `KB.Article_Overview` | Atlas_Default | Gallery，Published 过滤，NF 预览 | Gallery + 纳流调用 |
| `KB.Article_Detail` | Atlas_Default | DataView + ReferenceSetSelector（Tags） + 评分 DataGrid | 多对多 ReferenceSetSelector |

### HelpDesk（4 个页面）

| 页面 | 布局 | 组件 | 覆盖 |
|------|------|------|------|
| `HD.Ticket_Overview` | Atlas_Default | DataGrid2 + 列过滤（text/dropdown） | DataGrid2 列过滤 |
| `HD.Ticket_Detail` | Atlas_Default | DataView + Agent ref + KB article ref + Comments list（IsInternal 条件可见） + 多个 ActionButton | 跨模块 ref + 条件可见性 |
| `HD.TicketSearch_Form` | PopupLayout | 非持久化 DataView + 纳流重置 + 关闭弹窗 | 非持久化表单 + PopupLayout |
| `HD.EscalationReview_Form` | PopupLayout | 工作流用户任务页面（必须含 `$WorkflowUserTask: System.WorkflowUserTask` 参数） | 工作流任务页面 |

---

## 9. 安全配置

### 模块角色

**KnowledgeBase（3 个）：**
- `KB.Reader`：只读 Published 文章，自己的评分
- `KB.Contributor`：Reader + 写文章、标签
- `KB.Admin`：全部权限

**HelpDesk（3 个，不建 Admin 模块角色）：**
- `HD.CustomerRole`：自己的工单（XPath），非 Internal 评论
- `HD.AgentRole`：CustomerRole + 全部工单 + Internal 评论 + 升级请求
- `HD.ManagerRole`：AgentRole + 审批 + 删除工单

> 注：Admin 权限通过用户角色层面叠加所有模块角色实现，不在业务模块内单独建 AdminRole。

### 用户角色（4 个）

| 用户角色 | 模块角色映射 |
|---------|------------|
| `Customer` | HD.CustomerRole + KB.Reader |
| `Agent` | HD.AgentRole + KB.Contributor |
| `Manager` | HD.ManagerRole + KB.Contributor |
| `Administrator` | HD.ManagerRole + KB.Admin（+ Administration 内置权限） |

### XPath 行级过滤

```
HD.CustomerRole on HD.Ticket:
  read * where [HD.Ticket_Customer/HD.Customer/System.owner = '[%CurrentUser%]']

HD.CustomerRole on HD.TicketComment:
  read * where [IsInternal = false]

KB.Reader on KB.ArticleRating:
  read * where [System.owner = '[%CurrentUser%]']

KB.Reader on KB.Article:
  read * where [Status = 'Published']
```

### 应用级安全

```mdl
alter project security (security level: production, demo users: true);
create demo user 'demo_customer@helpdesk.test' (password: 'Demo1234!', user role: Customer);
create demo user 'demo_agent@helpdesk.test' (password: 'Demo1234!', user role: Agent);
create demo user 'demo_manager@helpdesk.test' (password: 'Demo1234!', user role: Manager);
```

---

## 10. 导航

```
Responsive profile:
  home page: HD.Ticket_Overview
  login page: Administration.login

menu:
  '我的工单'   → HD.Ticket_Overview
  '知识库'     → KB.Article_Overview
  '工单管理'  (子菜单)
    '所有工单' → HD.Ticket_Overview
    '升级审批' → HD.EscalationReview_Form
  '系统管理'   → Administration.Account_Overview
```

> 注：菜单项可见性**自动派生**自页面访问权限，MDL 菜单项语法无 `roles (...)` 属性。

---

## 11. 尺寸预估

| 分节 | 预计行数 |
|------|---------|
| 域模型（两模块） | ~150 |
| 常量 | ~15 |
| 微流（状态机 + KB） | ~130 |
| 纳流 | ~60 |
| 工作流（子+父） | ~120 |
| 工作流配套微流（13个活动） | ~180 |
| 页面（6个） | ~160 |
| 安全 | ~90 |
| 导航 | ~35 |
| 注释 + MARK 标题 | ~80 |
| **总计** | **~1020** |

---

## 12. 注释占位规范

对 MDL 引擎尚未实现的功能，使用以下格式占位：

```mdl
-- TODO: [功能名称] — 待实现
-- 预期语法：
--   alter workflow HD.WF_TicketEscalation
--     insert multi user task UT_SeniorReview
--     ...
-- 实现后移除注释，替换为正式 MDL
```

---

## 13. 验证流程

```bash
# 语法检查（无项目）
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl

# 完整执行 + mx check
./bin/mxcli -p testdata/corpus-b/app.mpr \
  -c "$(cat mdl-examples/use-cases/helpdesk/helpdesk-app.mdl)"
~/.mxcli/mxbuild/11.6.4/modeler/mx check testdata/corpus-b/app.mpr \
  2>&1 | grep -i "StorageLoadException\|Invalid\|Error"

# 清理
git restore testdata/corpus-b/
```
