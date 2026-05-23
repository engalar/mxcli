# HelpDesk + KnowledgeBase — 应用基线

## 用途

本目录是 mxcli 的**端到端回归基线**，类似 `mdl-examples/shop/shop.mdl`。

每次 `mx check` 必须通零报错。覆盖所有 MDL 功能层的边界情况，是验证 BSON 写入正确性的综合用例。

---

## 文件清单

| 文件 | 说明 |
|------|------|
| `helpdesk-app.mdl` | 完整应用 MDL，单文件可直接 `mxcli exec` |
| `README.md` | 本文件：业务需求 + 运行说明 |

相关文档：
- 设计规格：`docs/superpowers/specs/2026-05-22-helpdesk-app-design.md`
- 平台事实参考：`docs/mendix-platform-reference.md`

---

## 业务场景

### 两个模块

**HelpDesk（HD）** — 客服工单系统

工单从客户提交，由客服处理，经历 5 个状态，必要时通过审批工作流升级优先级。

**KnowledgeBase（KB）** — 知识库

客服发布文章帮助客户自助解决问题，文章可被关联到工单。

---

## 业务需求

### 用户角色

| 用户角色 | 描述 |
|---------|------|
| Customer | 提交工单、查看自己的工单、浏览已发布知识库文章 |
| Agent | 处理工单、撰写知识库文章、发起升级请求 |
| Manager | 审批升级、全量工单管理、管理知识库 |
| Administrator | 系统管理员，具有所有权限 |

---

### 工单生命周期（HD.Ticket）

```
Draft ──[Submit]──► Open ──[Assign]──► InProgress ──[Resolve]──► Resolved ──[Close]──► Closed
                     ▲                                                │
                     └──────────────────[Reopen]─────────────────────┘
```

- **Submit**：校验 Subject 非空；按 Priority 和环境常量计算 SLA 截止时间
- **Assign**：指定 Agent，状态变为 InProgress
- **Resolve**：记录解决时间，计算 IsOverSLA 标志（ResolvedAt > SLADueAt）
- **Reopen**：从 Resolved/Closed 退回 Open（清空 ResolvedAt）
- **Close**：终态

SLA 小时数通过模块常量配置（`HD.SLA_HIGH_HOURS = 8`，`HD.SLA_CRITICAL_HOURS = 2`），可按环境覆盖。

---

### 升级审批工作流

Agent 可对 InProgress 工单提交升级请求。审批通过后工单优先级自动升为 Critical。

**子工作流（WF_SUB_ManagerReview）**：
1. Manager 主审（User task）
2. 若 Manager 选择"升级"，转入高级委员会多数投票（Multi-user task，Majority 决策）
3. Approve → 更新工单优先级；Reject → 记录拒绝原因

**父工作流（WF_TicketEscalation）**：
1. Critical 优先级直接跳过审批（Jump → 自动批准）
2. 其余优先级等待 Manager 空闲信号（Wait for notification）
3. 调用子工作流审批；超时 48h 自动拒绝（Interrupting boundary event）
4. 审批通过后并行执行：更新工单 + 通知 Agent（Parallel split）

---

### 知识库文章生命周期（KB.Article）

```
Draft ──[Publish]──► Published ──[Archive]──► Archived
```

- Publish：校验 Content 非空，设 PublishedAt
- Customer 只能读 Published 状态文章（XPath 行级过滤）

---

### 数据访问规则（行级安全）

| 角色 | 实体 | 约束 |
|------|------|------|
| CustomerRole | HD.Ticket | 只读/写自己的工单（`System.owner = CurrentUser`） |
| CustomerRole | HD.TicketComment | 只读非 Internal 评论（`IsInternal = false`） |
| KB.Reader | KB.Article | 只读 Published 文章 |
| KB.Reader | KB.ArticleRating | 只读/写自己的评分 |

---

## 覆盖的边界情况

### 域模型边界
- **自引用关联**：`KB.Category_Parent`（分类树）
- **跨模块关联**：`HD.Ticket → KB.Article`
- **多对多中间表**：`KB.ArticleTag`（Article ↔ Tag）
- **非持久化实体**：`HD.TicketSearch`（搜索表单）

### 微流边界
- **5 态状态机 + 状态倒退**：Ticket 工单状态
- **rollback 分支**：ACT_Ticket_Submit 校验失败路径
- **常量引用**：`@HD.SLA_HIGH_HOURS` 在表达式中
- **时间计算 + 布尔标志**：IsOverSLA 计算

### 纳流边界
- **DB retrieve（XPath 查询）**：NF_TicketSearch_Apply
- **Create + Commit 持久化对象**：NF_Ticket_QuickCreate
- **Call microflow**：KB.NF_Article_FormatPreview → KB.SUB_Article_TruncateContent
- **纯计算返回 string**：NF_Priority_GetLabel

### 工作流边界
| 元素 | 所在工作流 |
|------|----------|
| Annotation | 父工作流 |
| Decision（Enumeration 类型） | 父工作流 |
| Decision（outcome 类型） | 子工作流 |
| User task（单用户，WFA_ targeting） | 子工作流 |
| Multi-user task（Majority 决策）† | 子工作流 |
| Call workflow（父→子） | 父工作流 |
| Wait for notification | 父工作流 |
| Jump（到指定 End event） | 父工作流 |
| Parallel split（双路径） | 父工作流 |
| Timer boundary（非中断 + 重复）† | Wait for notification |
| Timer boundary（中断） | Call workflow |
| Multiple End events | 两个工作流 |

† 标注项已在 MDL 中以注释形式占位，引擎实现后替换。

### 微流工作流活动（全部 13 个）
Call、ChangeState、CompleteTask、ApplyJumpTo、GenerateJumpTo、
RetrieveActivityRecords、RetrieveContext、RetrieveWorkflows、
ShowTaskPage、ShowAdminPage、Lock、Unlock、Notify

### 安全边界
- XPath 行级过滤（4 条不同约束）
- 模块角色 → 用户角色多对多映射
- 页面/微流/纳流执行权限分级
- 应用级安全设置 + Demo users

---

## 运行方式

```bash
# 1. 语法检查（无需项目）
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl

# 2. 完整执行（需要 MPR 项目）
./bin/mxcli -p testdata/corpus-b/app.mpr \
  exec mdl-examples/use-cases/helpdesk/helpdesk-app.mdl

# 3. mx check 验证（无新增报错即通过）
~/.mxcli/mxbuild/11.6.4/modeler/mx check testdata/corpus-b/app.mpr \
  2>&1 | grep -i "StorageLoadException\|Invalid\|Error"

# 4. 清理（MPR v2 写入新文件，需同时 clean 未跟踪文件）
git restore testdata/corpus-b/ && git clean -fd testdata/corpus-b/
```

---

## 注释占位规范

MDL 引擎尚未支持的特性用以下格式标注：

```mdl
-- TODO: [功能名称] — 待实现
-- 预期语法：
--   ...
-- 实现后移除注释，替换为正式 MDL
```

当前占位项：（无 — 所有已知缺口已在 2026-05-23 实现）

实现的功能：
- 10 个工作流微流活动（workflow operation, get workflow data, get workflows, get workflow activity records, set task outcome, open user task, open/lock/unlock workflow, notify workflow）
- 用户角色与 demo users
- Multi-user task `completion method majority/threshold/consensus`
- `generate jump to options for` / `apply jump to option` 微流活动（全栈新实现）
- 常量引用表达式 `@HD.SLA_CRITICAL_HOURS`
- loop/delete/error-handling/真实表单页面/referenceselector/count 功能演示
