# Academy Phase 2 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现 Academy 进阶阶段：模块 06（知识库）、模块 07（审批工作流）以及 Capstone 参考实现（可执行、带种子数据的完整演示应用）。

**Architecture:** 在 Phase 1 骨架基础上叠加。模块 06 引入独立的 KB 模块；模块 07 在 HD 模块内用微流状态机实现升级审批（不使用 Mendix Workflow Engine，降低 MDL 复杂度）；Capstone 参考实现将模块 01–07 组合为单一可执行序列。

**Tech Stack:** MDL（所有参考实现和练习）、Markdown（文档）

**前提：** Phase 1 计划已完整执行（academy/zh/00–05 目录及内容存在）。

---

## 文件清单（Phase 2 新增）

```
academy/zh/
├── 06-知识库模块/
│   ├── 业务需求.md
│   ├── AI协作指南.md
│   ├── 参考实现/
│   │   ├── kb-domain.mdl       ← KB 实体、枚举、关联
│   │   ├── kb-microflows.mdl   ← Publish/Archive 微流
│   │   ├── kb-nanoflows.mdl    ← 格式化辅助纳流
│   │   ├── kb-pages.mdl        ← 知识库页面
│   │   └── kb-security.mdl     ← KB 角色、授权、导航更新
│   └── 练习/
│       ├── 练习01-文章生命周期.mdl
│       └── 练习02-分类树设计.mdl
├── 07-审批工作流/
│   ├── 业务需求.md
│   ├── AI协作指南.md
│   ├── 参考实现/
│   │   ├── escalation-domain.mdl     ← EscalationRequest 实体
│   │   ├── escalation-microflows.mdl ← 发起/审批/拒绝
│   │   └── escalation-pages.mdl      ← 升级概览和审批页面
│   └── 练习/
│       └── 练习01-升级请求微流.mdl
└── capstone-helpdesk/
    ├── 业务需求.md    ← Phase 1 已创建，不修改
    ├── 执行说明.md    ← Phase 2 新增：如何运行完整应用
    └── 参考实现/      ← Phase 2 新增
        ├── 01-domain.mdl
        ├── 02-microflows.mdl
        ├── 03-nanoflows.mdl
        ├── 04-pages.mdl
        ├── 05-security.mdl
        ├── 06-kb.mdl
        ├── 07-escalation.mdl
        └── 99-seed-data.mdl
```

---

## Task 1：创建 Phase 2 目录

**Files:** 新目录

- [ ] **Step 1：创建缺失目录**

```bash
mkdir -p academy/zh/06-知识库模块/参考实现 academy/zh/06-知识库模块/练习
mkdir -p academy/zh/07-审批工作流/参考实现  academy/zh/07-审批工作流/练习
mkdir -p academy/zh/capstone-helpdesk/参考实现
```

Expected: 7 directories created, no errors.

- [ ] **Step 2：验证**

```bash
find academy/zh/06-知识库模块 academy/zh/07-审批工作流 academy/zh/capstone-helpdesk -type d | sort
```

Expected: 8 directories listed.

- [ ] **Step 3：提交**

```bash
git add academy/zh/06-知识库模块 academy/zh/07-审批工作流 academy/zh/capstone-helpdesk/参考实现
git commit -m "chore(academy): create Phase 2 directory structure"
```

---

## Task 2：Module 06 — 知识库模块

**Files:**
- Create: `academy/zh/06-知识库模块/业务需求.md`
- Create: `academy/zh/06-知识库模块/AI协作指南.md`
- Create: `academy/zh/06-知识库模块/参考实现/kb-domain.mdl`
- Create: `academy/zh/06-知识库模块/参考实现/kb-microflows.mdl`
- Create: `academy/zh/06-知识库模块/参考实现/kb-nanoflows.mdl`
- Create: `academy/zh/06-知识库模块/参考实现/kb-pages.mdl`
- Create: `academy/zh/06-知识库模块/参考实现/kb-security.mdl`
- Create: `academy/zh/06-知识库模块/练习/练习01-文章生命周期.mdl`
- Create: `academy/zh/06-知识库模块/练习/练习02-分类树设计.mdl`

- [ ] **Step 1：写 业务需求.md**

```markdown
# 模块 06：知识库模块 — 业务需求

## 业务背景

IT 支持团队发现，70% 的工单问题是重复的——同样的 VPN 问题、同样的密码重置流程每周都有人问。
如果能把这些解决方法整理成文章，让客户自己找到答案，客服的工作量可以减少一半。

这就是**知识库**的价值：让知识沉淀，让自助服务成为可能。

---

## 用户故事

### 客户（自助查询）
- 作为客户，我想按关键词搜索知识库文章，这样我可以在联系客服前先自己找答案
- 作为客户，我只能看到已正式发布的文章，不看草稿或已归档的内容

### 客服（知识沉淀）
- 作为客服，我想新建一篇草稿文章，把解决方法整理下来，再发布给大家看
- 作为客服，我想把过时的文章归档，让知识库保持整洁
- 作为客服，我想把文章分到不同分类（如"账号问题"、"网络连接"），方便查找

### 知识库结构
- 作为系统，文章必须属于一个分类
- 作为系统，分类可以有子分类（如"网络问题 > VPN" > "VPN 连接失败"）
- 作为系统，文章可以打多个标签（如 "howto"、"faq"），方便聚合查询

---

## 文章生命周期

```
草稿 ──[发布]──► 已发布 ──[归档]──► 已归档
```

- 发布：内容必须非空；系统记录发布时间
- 归档：只有已发布的文章能被归档

---

## 验收标准

- [ ] 知识库有分类树（分类可以有父分类）
- [ ] 文章属于一个分类，可以打多个标签（多对多）
- [ ] 客户只看"已发布"文章（数据库层面过滤）
- [ ] 发布时内容为空 → 显示错误提示
- [ ] 归档非"已发布"文章 → 静默失败（不报错，直接 return false）
- [ ] 知识库首页列出所有已发布文章，有标题搜索过滤器
- [ ] 文章详情页有"发布"和"归档"按钮（客服/管理员可见）
```

- [ ] **Step 2：写 AI协作指南.md**

```markdown
# 模块 06：AI 协作指南 — 知识库模块

## 前提

先运行模块 01–05 的参考实现（或运行 capstone 参考实现的 01–05 文件）。

## 本模块涉及的新 MDL 概念

| 概念 | 语法示例 |
|------|---------|
| 自引用关联（分类树） | `from KB.Category to KB.Category` |
| 多对多中间表 | 创建空实体 KB.ArticleTag，两个关联分别指向两端 |
| unique 约束 | `Name: string(100) not null unique` |
| integer 属性 | `ViewCount: integer default 0` |

## 与 Claude 协作的步骤

### Step 1：让 Claude 设计 KB 领域模型

```
读取 academy/zh/06-知识库模块/业务需求.md，帮我用 MDL 实现知识库的领域模型：
- KB 模块
- KB.ArticleStatus 枚举（Draft/Published/Archived）
- KB.Category 实体（Name, Description, 自引用父分类关联）
- KB.Tag 实体（Name, unique）
- KB.Article 实体（Title, Content, Status, PublishedAt, ViewCount）
  - 关联 KB.Article → KB.Category
- KB.ArticleTag 中间表（多对多：Article ↔ Tag）
```

### Step 2：发布和归档微流

```
帮我实现两个微流：
- KB.ACT_Article_Publish：验证 Content 非空，Status Draft → Published，记录 PublishedAt
- KB.ACT_Article_Archive：校验当前是 Published，Status → Archived
```

### Step 3：常见坑

| 坑 | 解决 |
|----|------|
| 自引用关联命名 | 用 KB.Category_Parent（而非 KB.Category_Category）避免歧义 |
| 多对多中间表 | KB.ArticleTag 实体本身没有属性，只有两个 association |
| XPath 字符串过滤 | `where '[Status = ''Published'']'`（单引号在 XPath 内要双写） |
| PublishedAt 在发布前为空 | 发布微流用 `PublishedAt = '[%CurrentDateTime%]'` 赋值 |
```

- [ ] **Step 3：写 参考实现/kb-domain.mdl**

```mdl
-- ============================================================
-- 模块 06：知识库 — 领域模型
-- 前提：先运行模块 01 的 domain-model.mdl（HD 模块需存在）
-- 运行：mxcli exec kb-domain.mdl -p MyProject.mpr
-- ============================================================

create module KB;

-- ============================================================
-- 枚举：文章状态
-- ============================================================

create or modify enumeration KB.ArticleStatus (
  Draft     'Draft',
  Published 'Published',
  Archived  'Archived'
);

-- ============================================================
-- 实体：分类（支持层级树：分类可以有父分类）
-- ============================================================

create or modify persistent entity KB.Category (
  Name:        string(200) not null,
  Description: string(500)
);

-- 自引用关联：子分类 → 父分类（可空，顶级分类无父）
create or modify association KB.Category_Parent
  from KB.Category to KB.Category
  type reference
  owner default;

-- ============================================================
-- 实体：标签（唯一约束：同名标签只能有一个）
-- ============================================================

create or modify persistent entity KB.Tag (
  Name: string(100) not null unique
);

-- ============================================================
-- 实体：文章
-- index (Status)        快速按状态过滤（只看 Published）
-- index (PublishedAt)   快速按发布时间排序
-- ============================================================

create or modify persistent entity KB.Article (
  Title:       string(500) not null,
  Content:     string,
  Status:      KB.ArticleStatus default Draft,
  PublishedAt: datetime,
  ViewCount:   integer default 0
)
index (Status)
index (PublishedAt);

-- 文章 → 分类（多文章属于一个分类）
create or modify association KB.Article_Category
  from KB.Article to KB.Category
  type reference
  owner default;

-- ============================================================
-- 多对多：文章 ↔ 标签（通过中间表 ArticleTag）
-- KB.ArticleTag 是纯关联实体，没有自己的属性
-- ============================================================

create or modify persistent entity KB.ArticleTag ();

create or modify association KB.ArticleTag_Article
  from KB.ArticleTag to KB.Article
  type reference
  owner default;

create or modify association KB.ArticleTag_Tag
  from KB.ArticleTag to KB.Tag
  type reference
  owner default;
```

- [ ] **Step 4：写 参考实现/kb-microflows.mdl**

```mdl
-- ============================================================
-- 模块 06：知识库 — 微流
-- 前提：先运行 kb-domain.mdl
-- 运行：mxcli exec kb-microflows.mdl -p MyProject.mpr
-- ============================================================

-- ============================================================
-- 发布文章：Draft → Published
-- 验证：Content 非空
-- ============================================================

create or modify microflow KB.ACT_Article_Publish
  ($Article: KB.Article)
  returns boolean as $Success
  folder 'Article'
{
  if $Article/Content = '' or $Article/Content = empty {
    validation feedback $Article/Content message 'Article content is required before publishing.';
    return false;
  }
  change $Article (
    Status      = KB.ArticleStatus.Published,
    PublishedAt = '[%CurrentDateTime%]'
  );
  commit $Article;
  return true;
}
/

-- ============================================================
-- 归档文章：Published → Archived
-- 前置检查：必须是 Published 状态
-- ============================================================

create or modify microflow KB.ACT_Article_Archive
  ($Article: KB.Article)
  returns boolean as $Success
  folder 'Article'
{
  if $Article/Status != KB.ArticleStatus.Published {
    return false;
  }
  change $Article (Status = KB.ArticleStatus.Archived);
  commit $Article;
  return true;
}
/
```

- [ ] **Step 5：写 参考实现/kb-nanoflows.mdl**

```mdl
-- ============================================================
-- 模块 06：知识库 — 纳流
-- 前提：先运行 kb-domain.mdl
-- 运行：mxcli exec kb-nanoflows.mdl -p MyProject.mpr
-- ============================================================

-- ============================================================
-- 文章预览格式化：截取前 200 字作为摘要
-- ============================================================

create or modify nanoflow KB.NF_Article_FormatPreview
  ($Article: KB.Article)
  returns string as $Preview
  folder 'Article'
{
  if length($Article/Content) > 200 {
    return substring($Article/Content, 1, 200) + '...';
  } else {
    return $Article/Content;
  }
}
/
```

- [ ] **Step 6：写 参考实现/kb-pages.mdl**

```mdl
-- ============================================================
-- 模块 06：知识库 — 页面
-- 前提：先运行 kb-domain.mdl, kb-microflows.mdl
-- 运行：mxcli exec kb-pages.mdl -p MyProject.mpr
-- ============================================================

-- ============================================================
-- 知识库首页：所有已发布文章（含标题过滤器）
-- ============================================================

create or replace page KB.Article_Overview
(
  title:  'Knowledge Base',
  layout: Atlas_Core.Atlas_Default,
  folder: 'Article'
)
{
  layoutgrid lgMain {
    row rHeader {
      column cTitle (desktopwidth: 12) {
        dynamictext txtTitle (content: 'Knowledge Base', rendermode: H2)
      }
    }
    row rMain {
      column cMain (desktopwidth: 12) {
        datagrid dgArticles (
          datasource:     database from KB.Article sort by PublishedAt desc,
          PageSize:       20,
          PagingPosition: both
        ) {
          controlbar cb {
            actionbutton btnNew (
              caption:     'New Article',
              action:      create_object KB.Article then show_page KB.Article_NewEdit,
              buttonstyle: primary
            )
          }
          column colTitle  (attribute: Title,       caption: 'Title') {
            textfilter fTitle
          }
          column colStatus (attribute: Status,      caption: 'Status',    ColumnWidth: manual, Size: 100) {
            dropdownfilter fStatus
          }
          column colDate   (attribute: PublishedAt, caption: 'Published', ColumnWidth: manual, Size: 140)
          column colViews  (attribute: ViewCount,   caption: 'Views',     ColumnWidth: manual, Size: 70)
          column colActions (caption: 'Actions', ShowContentAs: customContent, ColumnWidth: manual, Size: 80) {
            actionbutton btnRead (
              caption:     'Read',
              action:      show_page KB.Article_Detail (Article: $currentObject),
              buttonstyle: link
            )
          }
        }
      }
    }
  }
};

-- ============================================================
-- 文章详情：2 列布局（左内容，右操作）
-- ============================================================

create or replace page KB.Article_Detail
(
  title:  'Article',
  layout: Atlas_Core.Atlas_Default,
  folder: 'Article',
  params: { $Article: KB.Article }
)
{
  layoutgrid lgMain {
    row rContent {
      column cInfo (desktopwidth: 8) {
        dataview dvArticle (datasource: $Article) {
          dynamictext txtTitle    (content: '{1}', contentparams: [{1} = Title],       rendermode: H2)
          dynamictext txtStatus   (content: 'Status: {1}',    contentparams: [{1} = Status])
          dynamictext txtDate     (content: 'Published: {1}', contentparams: [{1} = PublishedAt])
          dynamictext txtViews    (content: 'Views: {1}',     contentparams: [{1} = ViewCount])
          dynamictext txtContent  (content: '{1}',            contentparams: [{1} = Content])
        }
      }
      column cActions (desktopwidth: 4) {
        dataview dvActions (datasource: $Article) {
          footer ftrActions {
            actionbutton btnPublish (
              caption:     'Publish',
              action:      microflow KB.ACT_Article_Publish (Article: $currentObject),
              buttonstyle: primary
            )
            actionbutton btnArchive (
              caption:     'Archive',
              action:      microflow KB.ACT_Article_Archive (Article: $currentObject),
              buttonstyle: default
            )
            actionbutton btnEdit (
              caption:     'Edit',
              action:      show_page KB.Article_NewEdit (Article: $currentObject),
              buttonstyle: default
            )
          }
        }
      }
    }
  }
};

-- ============================================================
-- 文章编辑表单
-- ============================================================

create or replace page KB.Article_NewEdit
(
  title:  'New / Edit Article',
  layout: Atlas_Core.Atlas_Default,
  folder: 'Article',
  params: { $Article: KB.Article }
)
{
  layoutgrid lgMain {
    row rMain {
      column cMain (desktopwidth: 8) {
        dataview dvForm (datasource: $Article) {
          textbox  tbTitle   (label: 'Title',   attribute: Title)
          textarea taContent (label: 'Content', attribute: Content)
          footer ftrButtons {
            actionbutton btnSave   (caption: 'Save',   action: save_changes close_page, buttonstyle: primary)
            actionbutton btnCancel (caption: 'Cancel', action: cancel_changes close_page)
          }
        }
      }
    }
  }
};
```

- [ ] **Step 7：写 参考实现/kb-security.mdl**

```mdl
-- ============================================================
-- 模块 06：知识库 — 安全与导航
-- 前提：先运行模块 05 security.mdl（用户角色 Customer/Agent/Manager 必须存在）
-- 运行：mxcli exec kb-security.mdl -p MyProject.mpr
-- ============================================================

-- ============================================================
-- KB 模块角色
-- ============================================================

create or modify module role KB.Reader;
create or modify module role KB.Contributor;
create or modify module role KB.Admin;

-- ============================================================
-- 实体访问规则
-- ============================================================

-- Reader：只看已发布文章（XPath 行级过滤）
grant KB.Reader on KB.Article (read *)
  where '[Status = ''Published'']';
grant KB.Reader on KB.Category  (read *);
grant KB.Reader on KB.Tag       (read *);
grant KB.Reader on KB.ArticleTag(read *);

-- Contributor：完整文章生命周期
grant KB.Contributor on KB.Article    (create, read *, write *, delete);
grant KB.Contributor on KB.ArticleTag (create, read *, write *, delete);
grant KB.Contributor on KB.Category   (create, read *, write *);
grant KB.Contributor on KB.Tag        (create, read *, write *);

-- Admin：无限制
grant KB.Admin on KB.Article    (create, read *, write *, delete);
grant KB.Admin on KB.ArticleTag (create, read *, write *, delete);
grant KB.Admin on KB.Category   (create, read *, write *, delete);
grant KB.Admin on KB.Tag        (create, read *, write *, delete);

-- ============================================================
-- 微流 / 纳流 / 页面权限
-- ============================================================

grant execute on microflow KB.ACT_Article_Publish to KB.Contributor, KB.Admin;
grant execute on microflow KB.ACT_Article_Archive to KB.Contributor, KB.Admin;
grant execute on nanoflow  KB.NF_Article_FormatPreview to KB.Reader, KB.Contributor, KB.Admin;

grant view on page KB.Article_Overview to KB.Reader, KB.Contributor, KB.Admin;
grant view on page KB.Article_Detail   to KB.Reader, KB.Contributor, KB.Admin;
grant view on page KB.Article_NewEdit  to KB.Contributor, KB.Admin;

-- ============================================================
-- 更新用户角色（加入 KB 角色，create or modify 是幂等的）
-- ============================================================

create or modify user role Customer (System.User, HD.CustomerRole, KB.Reader);
create or modify user role Agent    (System.User, HD.AgentRole,    KB.Contributor);
create or modify user role Manager  (System.User, HD.ManagerRole,  KB.Contributor);

-- ============================================================
-- 更新导航（加入 Knowledge Base 菜单项）
-- ============================================================

create or replace navigation Responsive
  home page HD.MyTickets_Overview  for Customer
  home page HD.Ticket_Overview     for Agent
  home page HD.Ticket_Overview     for Manager
  home page MyFirstModule.Home_Web
  login page Administration.login
  menu (
    menu item 'My Tickets'     page HD.MyTickets_Overview;
    menu item 'All Tickets'    page HD.Ticket_Overview;
    menu item 'Knowledge Base' page KB.Article_Overview;
  );
```

- [ ] **Step 8：写 练习/练习01-文章生命周期.mdl**

```mdl
-- ============================================================
-- 练习 01：文章生命周期微流
-- 前提：先运行 参考实现/kb-domain.mdl
-- 目标：补全 KB.ACT_Article_Publish 微流
-- ============================================================

create or modify microflow KB.ACT_Article_Publish
  ($Article: KB.Article)
  returns boolean as $Success
  folder 'Article'
{
  -- TODO 1：验证 Content 不能为空（包含 empty 和空字符串两种情况）
  -- 失败时：validation feedback $Article/Content message '...'，return false

  -- TODO 2：用 change 更新两个字段
  -- Status = KB.ArticleStatus.Published
  -- PublishedAt = '[%CurrentDateTime%]'

  -- TODO 3：commit + return true
}
/

-- 同时实现归档微流（从头编写）
create or modify microflow KB.ACT_Article_Archive
  ($Article: KB.Article)
  returns boolean as $Success
  folder 'Article'
{
  -- TODO 4：检查 Status 必须是 Published，否则 return false
  -- TODO 5：change Status = Archived
  -- TODO 6：commit + return true
}
/

-- 验证命令：
--   mxcli check 练习01-文章生命周期.mdl
--   mxcli exec  练习01-文章生命周期.mdl -p MyProject.mpr
```

- [ ] **Step 9：写 练习/练习02-分类树设计.mdl**

```mdl
-- ============================================================
-- 练习 02：分类树与多对多关联
-- 前提：先运行 参考实现/kb-domain.mdl（已有 KB.Article）
-- 目标：
--   1. 创建 KB.Category 实体（已有基础，添加自引用关联）
--   2. 创建 KB.Tag 实体（unique 约束）
--   3. 创建 KB.ArticleTag 中间表（多对多）
-- ============================================================

-- KB.Category 分类实体已给出，请添加自引用关联
create or modify persistent entity KB.Category (
  Name:        string(200) not null,
  Description: string(500)
);

-- TODO 1：创建 KB.Category_Parent 关联
-- 要求：from KB.Category to KB.Category，type reference，owner default
-- 语义：子分类 → 父分类

-- 你的代码：


-- TODO 2：创建 KB.Tag 实体
-- 要求：Name 属性（string, 100 字符, not null, unique）

-- 你的代码：


-- TODO 3：创建 KB.ArticleTag 中间表（空实体）
-- 然后创建两个关联：
--   KB.ArticleTag_Article: from KB.ArticleTag to KB.Article
--   KB.ArticleTag_Tag:     from KB.ArticleTag to KB.Tag

-- 你的代码：


-- 验证命令：
--   mxcli check 练习02-分类树设计.mdl
--   mxcli exec  练习02-分类树设计.mdl -p MyProject.mpr
```

- [ ] **Step 10：提交**

```bash
git add academy/zh/06-知识库模块/
git commit -m "docs(academy): add module 06 — knowledge base (zh)"
```

---

## Task 3：Module 07 — 审批工作流

**Files:**
- Create: `academy/zh/07-审批工作流/业务需求.md`
- Create: `academy/zh/07-审批工作流/AI协作指南.md`
- Create: `academy/zh/07-审批工作流/参考实现/escalation-domain.mdl`
- Create: `academy/zh/07-审批工作流/参考实现/escalation-microflows.mdl`
- Create: `academy/zh/07-审批工作流/参考实现/escalation-pages.mdl`
- Create: `academy/zh/07-审批工作流/练习/练习01-升级请求微流.mdl`

- [ ] **Step 1：写 业务需求.md**

```markdown
# 模块 07：审批工作流 — 业务需求

## 业务背景

有些工单问题超出了普通客服的处理能力——比如涉及财务的支付问题、需要外部厂商配合的故障。
这时候需要一个**升级审批**机制：客服提出申请，经理来决定是否提升工单优先级。

这是 IT 支持中非常常见的场景，任何规模的支持团队都有类似流程。

---

## 用户故事

### 客服发起升级
- 作为客服，当我遇到自己处理不了的问题时，我想向经理发起升级申请，说明原因
- 作为系统，升级申请只能针对"处理中"状态的工单
- 作为系统，一张工单同时只能有一个待审批的升级申请

### 经理审批
- 作为经理，我想看到所有待审批的升级申请，按优先级排序
- 作为经理，我可以批准申请（工单优先级自动提升为"紧急"）
- 作为经理，我可以拒绝申请（填写拒绝原因，工单状态不变）

### 升级结果
- 作为客服，升级申请被批准后，我可以看到工单优先级已经变为"紧急"
- 作为客服，升级申请被拒绝后，我可以看到拒绝原因

---

## 验收标准

- [ ] 升级申请包含：原因说明（文字）、申请时间（自动）、审批状态（待审批/已批准/已拒绝）
- [ ] 只有"处理中"的工单才能发起升级申请
- [ ] 批准后：工单 Priority 变为 Critical
- [ ] 拒绝后：记录 RejectionReason，工单不变
- [ ] 升级概览页：经理可以看到所有待审批申请
- [ ] 审批表单：包含工单信息摘要 + 批准/拒绝按钮 + 拒绝原因输入框
```

- [ ] **Step 2：写 AI协作指南.md**

```markdown
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

### Step 3：常见坑

| 坑 | 解决 |
|----|------|
| 修改关联对象的属性 | 需要先 retrieve 再 change：`retrieve $Ticket from $ER/HD.EscalationRequest_Ticket/HD.Ticket limit 1` |
| commit 顺序 | 先 commit EscalationRequest，再 commit Ticket |
| 拒绝原因弹窗 | 用一个 non-persistent 实体作为表单容器，传入 $RejectionReason 参数 |
```

- [ ] **Step 3：写 参考实现/escalation-domain.mdl**

```mdl
-- ============================================================
-- 模块 07：审批工作流 — 领域模型
-- 前提：先运行模块 01 domain-model.mdl（HD.Ticket 必须存在）
-- 运行：mxcli exec escalation-domain.mdl -p MyProject.mpr
-- ============================================================

-- ============================================================
-- 枚举：升级申请审批状态
-- ============================================================

create or modify enumeration HD.EscalationStatus (
  Pending  'Pending',
  Approved 'Approved',
  Rejected 'Rejected'
);

-- ============================================================
-- 实体：升级申请
-- ============================================================

create or modify persistent entity HD.EscalationRequest (
  Reason:           string(1000) not null,
  RequestedAt:      datetime,
  ApprovalStatus:   HD.EscalationStatus default Pending,
  RejectionReason:  string(1000)
);

-- 升级申请 → 工单（一个申请对应一张工单）
create or modify association HD.EscalationRequest_Ticket
  from HD.EscalationRequest to HD.Ticket
  type reference
  owner default;

-- ============================================================
-- 非持久化实体：拒绝原因表单（弹窗输入）
-- ============================================================

create or modify non-persistent entity HD.RejectionForm (
  Reason: string(1000) not null
);
```

- [ ] **Step 4：写 参考实现/escalation-microflows.mdl**

```mdl
-- ============================================================
-- 模块 07：审批工作流 — 微流
-- 前提：先运行 escalation-domain.mdl
-- 运行：mxcli exec escalation-microflows.mdl -p MyProject.mpr
-- ============================================================

-- ============================================================
-- 发起升级申请
-- 前置：工单必须是 InProgress 状态
-- ============================================================

create or modify microflow HD.ACT_StartEscalation
  ($Ticket: HD.Ticket, $Reason: string)
  returns boolean as $Success
  folder 'Escalation'
{
  if $Ticket/Status != HD.TicketStatus.InProgress {
    show message 'Only In Progress tickets can be escalated.' type warning;
    return false;
  }
  $ER = create HD.EscalationRequest (
    Reason                        = $Reason,
    RequestedAt                   = '[%CurrentDateTime%]',
    ApprovalStatus                = HD.EscalationStatus.Pending,
    HD.EscalationRequest_Ticket   = $Ticket
  );
  commit $ER;
  return true;
}
/

-- ============================================================
-- 批准升级申请
-- 效果：ApprovalStatus → Approved；工单 Priority → Critical
-- ============================================================

create or modify microflow HD.ACT_Escalation_Approve
  ($EscalationRequest: HD.EscalationRequest)
  returns boolean as $Success
  folder 'Escalation'
{
  change $EscalationRequest (ApprovalStatus = HD.EscalationStatus.Approved);
  commit $EscalationRequest;

  retrieve $Ticket from HD.EscalationRequest_Ticket/HD.Ticket
    where [HD.EscalationRequest_Ticket = $EscalationRequest]
    limit 1;
  if $Ticket != empty {
    change $Ticket (Priority = HD.TicketPriority.Critical);
    commit $Ticket;
  }
  return true;
}
/

-- ============================================================
-- 拒绝升级申请（需要拒绝原因）
-- ============================================================

create or modify microflow HD.ACT_Escalation_Reject
  ($EscalationRequest: HD.EscalationRequest, $Reason: string)
  returns boolean as $Success
  folder 'Escalation'
{
  change $EscalationRequest (
    ApprovalStatus  = HD.EscalationStatus.Rejected,
    RejectionReason = $Reason
  );
  commit $EscalationRequest;
  return true;
}
/

-- ============================================================
-- 数据源：当前经理可见的待审批升级申请
-- ============================================================

create or modify microflow HD.DS_PendingEscalations
  ()
  returns list of HD.EscalationRequest as $List
  folder 'Escalation'
{
  retrieve $List from HD.EscalationRequest
    where [ApprovalStatus = 'Pending']
    sort by RequestedAt asc
    limit 0;
  return $List;
}
/
```

- [ ] **Step 5：写 参考实现/escalation-pages.mdl**

```mdl
-- ============================================================
-- 模块 07：审批工作流 — 页面
-- 前提：先运行 escalation-domain.mdl 和 escalation-microflows.mdl
-- 运行：mxcli exec escalation-pages.mdl -p MyProject.mpr
-- ============================================================

-- ============================================================
-- 弹窗：发起升级申请（客服使用）
-- ============================================================

create or replace page HD.EscalationStart_Form
(
  title:  'Request Escalation',
  layout: Atlas_Core.PopupLayout,
  folder: 'Escalation',
  params: { $Ticket: HD.Ticket }
)
{
  dataview dvForm (datasource: new HD.EscalationRequest) {
    textarea taReason (label: 'Reason for Escalation', attribute: Reason)
    footer ftrButtons {
      actionbutton btnSubmit (
        caption:     'Submit Request',
        action:      microflow HD.ACT_StartEscalation (Ticket: $Ticket, Reason: $currentObject/Reason) close_page,
        buttonstyle: primary
      )
      actionbutton btnCancel (caption: 'Cancel', action: cancel_changes close_page)
    }
  }
};

-- ============================================================
-- 升级审批概览（经理首页）
-- ============================================================

create or replace page HD.EscalationWorkflow_Overview
(
  title:  'Escalation Requests',
  layout: Atlas_Core.Atlas_Default,
  folder: 'Escalation'
)
{
  layoutgrid lgMain {
    row rHeader {
      column cTitle (desktopwidth: 12) {
        dynamictext txtTitle (content: 'Pending Escalation Requests', rendermode: H2)
      }
    }
    row rMain {
      column cMain (desktopwidth: 12) {
        datagrid dgEscalations (
          datasource: microflow HD.DS_PendingEscalations,
          PageSize:   20
        ) {
          column colReason    (attribute: Reason,          caption: 'Reason')
          column colStatus    (attribute: ApprovalStatus,  caption: 'Status',      ColumnWidth: manual, Size: 110)
          column colRequested (attribute: RequestedAt,     caption: 'Requested At', ColumnWidth: manual, Size: 140)
          column colActions (caption: 'Actions', ShowContentAs: customContent, ColumnWidth: manual, Size: 160) {
            actionbutton btnApprove (
              caption:     'Approve',
              action:      microflow HD.ACT_Escalation_Approve (EscalationRequest: $currentObject),
              buttonstyle: primary
            )
            actionbutton btnReject (
              caption:     'Reject',
              action:      show_page HD.EscalationReject_Form (EscalationRequest: $currentObject),
              buttonstyle: default
            )
          }
        }
      }
    }
  }
};

-- ============================================================
-- 弹窗：填写拒绝原因（经理使用）
-- ============================================================

create or replace page HD.EscalationReject_Form
(
  title:  'Reject Escalation',
  layout: Atlas_Core.PopupLayout,
  folder: 'Escalation',
  params: { $EscalationRequest: HD.EscalationRequest }
)
{
  dataview dvForm (datasource: new HD.RejectionForm) {
    textarea taReason (label: 'Rejection Reason', attribute: Reason)
    footer ftrButtons {
      actionbutton btnReject (
        caption:     'Confirm Reject',
        action:      microflow HD.ACT_Escalation_Reject (EscalationRequest: $EscalationRequest, Reason: $currentObject/Reason) close_page,
        buttonstyle: danger
      )
      actionbutton btnCancel (caption: 'Cancel', action: cancel_changes close_page)
    }
  }
};
```

- [ ] **Step 6：写 练习/练习01-升级请求微流.mdl**

```mdl
-- ============================================================
-- 练习 01：发起升级申请微流
-- 前提：先运行 参考实现/escalation-domain.mdl（和模块01的 domain-model.mdl）
-- 目标：补全 HD.ACT_StartEscalation 微流
-- ============================================================

create or modify microflow HD.ACT_StartEscalation
  ($Ticket: HD.Ticket, $Reason: string)
  returns boolean as $Success
  folder 'Escalation'
{
  -- TODO 1：前置检查——工单必须是 InProgress 状态
  -- 如果不是，show message '...' type warning，return false

  -- TODO 2：创建 EscalationRequest 对象
  -- 必填字段：Reason = $Reason，RequestedAt = '[%CurrentDateTime%]'
  --           ApprovalStatus = HD.EscalationStatus.Pending
  --           关联：HD.EscalationRequest_Ticket = $Ticket

  -- TODO 3：commit 对象，return true
}
/

-- 验证命令：
--   mxcli check 练习01-升级请求微流.mdl
--   mxcli exec  练习01-升级请求微流.mdl -p MyProject.mpr
```

- [ ] **Step 7：提交**

```bash
git add academy/zh/07-审批工作流/
git commit -m "docs(academy): add module 07 — escalation workflow (zh)"
```

---

## Task 4：Capstone 执行说明 + 参考实现文件

**Files:**
- Create: `academy/zh/capstone-helpdesk/执行说明.md`
- Create: `academy/zh/capstone-helpdesk/参考实现/01-domain.mdl`
- Create: `academy/zh/capstone-helpdesk/参考实现/02-microflows.mdl`
- Create: `academy/zh/capstone-helpdesk/参考实现/03-nanoflows.mdl`
- Create: `academy/zh/capstone-helpdesk/参考实现/04-pages.mdl`
- Create: `academy/zh/capstone-helpdesk/参考实现/05-security.mdl`
- Create: `academy/zh/capstone-helpdesk/参考实现/06-kb.mdl`
- Create: `academy/zh/capstone-helpdesk/参考实现/07-escalation.mdl`

- [ ] **Step 1：写 执行说明.md**

```markdown
# Capstone 执行说明

## 快速开始（在干净项目上运行全套）

```bash
# 1. 准备干净项目
mxcli new MyHelpdesk --version 11.6.6

# 2. 按顺序执行参考实现
mxcli exec academy/zh/capstone-helpdesk/参考实现/01-domain.mdl      -p MyHelpdesk/MyHelpdesk.mpr
mxcli exec academy/zh/capstone-helpdesk/参考实现/02-microflows.mdl  -p MyHelpdesk/MyHelpdesk.mpr
mxcli exec academy/zh/capstone-helpdesk/参考实现/03-nanoflows.mdl   -p MyHelpdesk/MyHelpdesk.mpr
mxcli exec academy/zh/capstone-helpdesk/参考实现/04-pages.mdl       -p MyHelpdesk/MyHelpdesk.mpr
mxcli exec academy/zh/capstone-helpdesk/参考实现/05-security.mdl    -p MyHelpdesk/MyHelpdesk.mpr
mxcli exec academy/zh/capstone-helpdesk/参考实现/06-kb.mdl          -p MyHelpdesk/MyHelpdesk.mpr
mxcli exec academy/zh/capstone-helpdesk/参考实现/07-escalation.mdl  -p MyHelpdesk/MyHelpdesk.mpr
mxcli exec academy/zh/capstone-helpdesk/参考实现/99-seed-data.mdl   -p MyHelpdesk/MyHelpdesk.mpr

# 3. 验证
~/.mxcli/mxbuild/*/modeler/mx check MyHelpdesk/MyHelpdesk.mpr \
  2>&1 | grep -c "StorageLoadException"
# 期望：0

# 4. 启动应用
mxcli local run -p MyHelpdesk/MyHelpdesk.mpr --admin-password Admin1234

# 5. 登录演示账号
# 地址：http://localhost:8080
# Customer: demo_customer@helpdesk.test / Demo12345678
# Agent:    demo_agent@helpdesk.test    / Demo12345678
# Manager:  demo_manager@helpdesk.test  / Demo12345678
# 以 Manager 登录后，在导航菜单中点"Initialize Demo Data"初始化种子数据
```

## 文件说明

| 文件 | 内容 | 依赖 |
|------|------|------|
| 01-domain.mdl     | HD 实体、枚举、关联、常量 | 无 |
| 02-microflows.mdl | 工单状态机微流、数据源 | 01 |
| 03-nanoflows.mdl  | 纳流：快速创建、搜索、格式化 | 01 |
| 04-pages.mdl      | 所有 HD 页面 | 01–03 |
| 05-security.mdl   | HD 角色、授权、演示用户、导航 | 01–04 |
| 06-kb.mdl         | KB 领域模型 + 微流 + 页面 + 安全 | 05 |
| 07-escalation.mdl | 升级审批实体 + 微流 + 页面 | 02 |
| 99-seed-data.mdl  | 演示数据微流（登录后手动触发）| 所有 |
```

- [ ] **Step 2：写 参考实现/01-domain.mdl**

内容与 `academy/zh/01-领域建模/参考实现/domain-model.mdl` 完全相同，直接复制：

```bash
cp academy/zh/01-领域建模/参考实现/domain-model.mdl \
   academy/zh/capstone-helpdesk/参考实现/01-domain.mdl
```

- [ ] **Step 3：写 参考实现/02-microflows.mdl**

```bash
cp academy/zh/02-微流业务逻辑/参考实现/microflows.mdl \
   academy/zh/capstone-helpdesk/参考实现/02-microflows.mdl
```

- [ ] **Step 4：写 参考实现/03-nanoflows.mdl**

```bash
cp academy/zh/03-纳流与客户端/参考实现/nanoflows.mdl \
   academy/zh/capstone-helpdesk/参考实现/03-nanoflows.mdl
```

- [ ] **Step 5：写 参考实现/04-pages.mdl**

```bash
cp academy/zh/04-页面与UI/参考实现/pages.mdl \
   academy/zh/capstone-helpdesk/参考实现/04-pages.mdl
```

- [ ] **Step 6：写 参考实现/05-security.mdl**

```bash
cp academy/zh/05-安全与权限/参考实现/security.mdl \
   academy/zh/capstone-helpdesk/参考实现/05-security.mdl
```

- [ ] **Step 7：写 参考实现/06-kb.mdl（合并 KB 所有文件）**

```mdl
-- ============================================================
-- Capstone 06：知识库完整模块（合并 domain + microflows + nanoflows + pages + security）
-- 前提：先运行 01–05
-- ============================================================
```

然后依次将以下文件内容追加进去（去掉各自的文件头注释，保留所有 MDL 语句）：
1. `academy/zh/06-知识库模块/参考实现/kb-domain.mdl`
2. `academy/zh/06-知识库模块/参考实现/kb-microflows.mdl`
3. `academy/zh/06-知识库模块/参考实现/kb-nanoflows.mdl`
4. `academy/zh/06-知识库模块/参考实现/kb-pages.mdl`
5. `academy/zh/06-知识库模块/参考实现/kb-security.mdl`

使用以下命令合并：

```bash
{
  echo "-- ============================================================"
  echo "-- Capstone 06：知识库完整模块"
  echo "-- 前提：01-05 已执行"
  echo "-- ============================================================"
  echo ""
  tail -n +12 academy/zh/06-知识库模块/参考实现/kb-domain.mdl
  echo ""
  tail -n +6  academy/zh/06-知识库模块/参考实现/kb-microflows.mdl
  echo ""
  tail -n +6  academy/zh/06-知识库模块/参考实现/kb-nanoflows.mdl
  echo ""
  tail -n +6  academy/zh/06-知识库模块/参考实现/kb-pages.mdl
  echo ""
  tail -n +6  academy/zh/06-知识库模块/参考实现/kb-security.mdl
} > academy/zh/capstone-helpdesk/参考实现/06-kb.mdl
```

- [ ] **Step 8：写 参考实现/07-escalation.mdl（合并升级工作流文件）**

```bash
{
  echo "-- ============================================================"
  echo "-- Capstone 07：升级审批（domain + microflows + pages）"
  echo "-- 前提：01-06 已执行"
  echo "-- ============================================================"
  echo ""
  tail -n +6 academy/zh/07-审批工作流/参考实现/escalation-domain.mdl
  echo ""
  tail -n +6 academy/zh/07-审批工作流/参考实现/escalation-microflows.mdl
  echo ""
  tail -n +6 academy/zh/07-审批工作流/参考实现/escalation-pages.mdl
} > academy/zh/capstone-helpdesk/参考实现/07-escalation.mdl
```

- [ ] **Step 9：为 07-escalation.mdl 补充权限和导航（追加到文件末尾）**

```bash
cat >> academy/zh/capstone-helpdesk/参考实现/07-escalation.mdl << 'GRANTS'

-- ============================================================
-- 升级功能权限
-- ============================================================

create or modify module role HD.ManagerRole;

grant HD.AgentRole   on HD.EscalationRequest (create, read *);
grant HD.ManagerRole on HD.EscalationRequest (create, read *, write *, delete);

grant execute on microflow HD.ACT_StartEscalation    to HD.AgentRole, HD.ManagerRole;
grant execute on microflow HD.ACT_Escalation_Approve to HD.ManagerRole;
grant execute on microflow HD.ACT_Escalation_Reject  to HD.ManagerRole;
grant execute on microflow HD.DS_PendingEscalations  to HD.ManagerRole;

grant view on page HD.EscalationStart_Form          to HD.AgentRole, HD.ManagerRole;
grant view on page HD.EscalationWorkflow_Overview   to HD.ManagerRole;
grant view on page HD.EscalationReject_Form         to HD.ManagerRole;

-- 更新导航：经理首页指向升级审批页面
create or replace navigation Responsive
  home page HD.MyTickets_Overview           for Customer
  home page HD.Ticket_Overview              for Agent
  home page HD.EscalationWorkflow_Overview  for Manager
  home page MyFirstModule.Home_Web
  login page Administration.login
  menu (
    menu item 'My Tickets'       page HD.MyTickets_Overview;
    menu item 'All Tickets'      page HD.Ticket_Overview;
    menu item 'Knowledge Base'   page KB.Article_Overview;
    menu item 'Escalations'      page HD.EscalationWorkflow_Overview;
  );
GRANTS
```

- [ ] **Step 10：提交**

```bash
git add academy/zh/capstone-helpdesk/执行说明.md academy/zh/capstone-helpdesk/参考实现/
git commit -m "docs(academy): add capstone reference implementation (01-07)"
```

---

## Task 5：Capstone 种子数据微流

**Files:**
- Create: `academy/zh/capstone-helpdesk/参考实现/99-seed-data.mdl`

- [ ] **Step 1：写 99-seed-data.mdl**

```mdl
-- ============================================================
-- Capstone 99：演示种子数据
-- 前提：先运行 01–07；在运行时由 Manager 触发（导航菜单）
-- 运行：mxcli exec 99-seed-data.mdl -p MyProject.mpr
-- 触发：登录为 demo_manager，导航菜单"Initialize Demo Data"
-- ============================================================

create or modify microflow HD.ACT_SeedDemoData ()
  returns boolean as $OK
  folder 'System'
{
  -- 幂等保护：已有工单则直接返回
  retrieve $existing from HD.Ticket limit 1;
  if $existing != empty {
    return true;
  }

  -- 客户
  $Cust1 = create HD.Customer (Name = 'Alice Tan',  Email = 'alice@acme.com',    Company = 'Acme Corp');
  $Cust2 = create HD.Customer (Name = 'Bob Lee',    Email = 'bob@globex.com',    Company = 'Globex Inc');
  $Cust3 = create HD.Customer (Name = 'Carol Wu',   Email = 'carol@initech.com', Company = 'Initech Ltd');
  commit $Cust1;
  commit $Cust2;
  commit $Cust3;

  -- 客服
  $Agent1 = create HD.Agent (Name = 'Alice Smith', Email = 'alice.smith@helpdesk.internal', IsActive = true);
  $Agent2 = create HD.Agent (Name = 'Bob Jones',   Email = 'bob.jones@helpdesk.internal',  IsActive = true);
  commit $Agent1;
  commit $Agent2;

  -- 工单（5 个，覆盖所有状态）
  $T1 = create HD.Ticket (
    Subject = 'Cannot login to portal',
    Description = 'Getting 403 error after password reset.',
    Status = HD.TicketStatus.Draft, Priority = HD.TicketPriority.Low,
    HD.Ticket_Customer = $Cust1
  );
  $T2 = create HD.Ticket (
    Subject = 'Dashboard loads slowly',
    Description = 'Main dashboard takes 30 seconds to render.',
    Status = HD.TicketStatus.Open, Priority = HD.TicketPriority.Normal,
    SLADueAt = addHours('[%CurrentDateTime%]', 24),
    HD.Ticket_Customer = $Cust2
  );
  $T3 = create HD.Ticket (
    Subject = 'Payment error on checkout',
    Description = 'Card declined for a valid Visa. Error: PAY_003.',
    Status = HD.TicketStatus.InProgress, Priority = HD.TicketPriority.High,
    SLADueAt = addHours('[%CurrentDateTime%]', 8),
    HD.Ticket_Customer = $Cust1, HD.Ticket_Agent = $Agent1
  );
  $T4 = create HD.Ticket (
    Subject = 'Data export returns empty file',
    Description = 'CSV export downloads a 0-byte file.',
    Status = HD.TicketStatus.Resolved, Priority = HD.TicketPriority.Critical,
    SLADueAt = '[%CurrentDateTime%]', ResolvedAt = '[%CurrentDateTime%]',
    IsOverSLA = true,
    HD.Ticket_Customer = $Cust3, HD.Ticket_Agent = $Agent2
  );
  $T5 = create HD.Ticket (
    Subject = 'Table columns misaligned in Firefox',
    Description = 'Header does not align with rows in Firefox 124.',
    Status = HD.TicketStatus.Closed, Priority = HD.TicketPriority.Normal,
    HD.Ticket_Customer = $Cust2
  );
  commit $T1;
  commit $T2;
  commit $T3;
  commit $T4;
  commit $T5;

  -- 每张工单一条评论（loop 示例）
  retrieve $AllTickets from HD.Ticket limit 10;
  loop $T in $AllTickets {
    $C = create HD.TicketComment (
      Content    = 'Demo comment for: ' + $T/Subject,
      IsInternal = false,
      HD.TicketComment_Ticket = $T
    );
    commit $C;
  }

  -- 升级申请（针对 T3：处理中 + 高优先级）
  $ER = create HD.EscalationRequest (
    Reason       = 'Customer threatening chargeback. Needs manager review.',
    RequestedAt  = '[%CurrentDateTime%]',
    ApprovalStatus = HD.EscalationStatus.Pending,
    HD.EscalationRequest_Ticket = $T3
  );
  commit $ER;

  -- KB 分类
  $Cat1 = create KB.Category (Name = 'Getting Started', Description = 'Guides for new users.');
  $Cat2 = create KB.Category (Name = 'Troubleshooting',  Description = 'Common issues and fixes.');
  commit $Cat1;
  commit $Cat2;

  -- KB 文章
  $Art1 = create KB.Article (
    Title   = 'How to submit a support ticket',
    Content = 'Navigate to My Tickets, click New Ticket, fill in Subject and Description, then Submit.',
    Status  = KB.ArticleStatus.Published,
    PublishedAt = '[%CurrentDateTime%]',
    KB.Article_Category = $Cat1
  );
  $Art2 = create KB.Article (
    Title   = 'Common login errors and fixes',
    Content = 'Clear browser cache, try incognito mode. If problem persists, contact support.',
    Status  = KB.ArticleStatus.Published,
    PublishedAt = '[%CurrentDateTime%]',
    KB.Article_Category = $Cat2
  );
  $Art3 = create KB.Article (
    Title   = 'SLA policy overview (Draft)',
    Content = 'Draft — under review.',
    Status  = KB.ArticleStatus.Draft,
    KB.Article_Category = $Cat2
  );
  commit $Art1;
  commit $Art2;
  commit $Art3;

  return true;
}
/

grant execute on microflow HD.ACT_SeedDemoData to HD.ManagerRole;

-- 在导航中加入"初始化演示数据"菜单项
create or replace navigation Responsive
  home page HD.MyTickets_Overview           for Customer
  home page HD.Ticket_Overview              for Agent
  home page HD.EscalationWorkflow_Overview  for Manager
  home page MyFirstModule.Home_Web
  login page Administration.login
  menu (
    menu item 'My Tickets'         page HD.MyTickets_Overview;
    menu item 'All Tickets'        page HD.Ticket_Overview;
    menu item 'Knowledge Base'     page KB.Article_Overview;
    menu item 'Escalations'        page HD.EscalationWorkflow_Overview;
    menu 'Admin' (
      menu item 'Initialize Demo Data' microflow HD.ACT_SeedDemoData;
    );
  );
```

- [ ] **Step 2：提交**

```bash
git add academy/zh/capstone-helpdesk/参考实现/99-seed-data.mdl
git commit -m "docs(academy): add capstone seed data microflow"
```

---

## Task 6：全量验证与最终提交

- [ ] **Step 1：对所有 Phase 2 参考实现 MDL 运行语法检查**

```bash
for f in \
  academy/zh/06-知识库模块/参考实现/*.mdl \
  academy/zh/07-审批工作流/参考实现/*.mdl \
  academy/zh/capstone-helpdesk/参考实现/*.mdl; do
  echo "Checking: $f"
  ./bin/mxcli check "$f"
done
```

Expected: 每个文件无语法错误。

- [ ] **Step 2：对所有 Phase 2 练习 MDL 运行语法检查**

```bash
for f in \
  academy/zh/06-知识库模块/练习/*.mdl \
  academy/zh/07-审批工作流/练习/*.mdl; do
  echo "Checking: $f"
  ./bin/mxcli check "$f"
done
```

Expected: 所有文件无语法错误（TODO 填空处不影响语法正确性）。

- [ ] **Step 3：修复任何语法错误**

如有错误，根据 mxcli check 输出定位行号，修复后重跑 Step 1–2。

- [ ] **Step 4：最终提交**

```bash
git add -A
git commit -m "docs(academy): Phase 2 complete — KB, escalation, capstone reference"
```

---

## 自检：Spec 覆盖确认

| 设计规格要求 | 对应 Task |
|------------|----------|
| 模块 06（KB）完整内容 | Task 2 |
| 模块 07（工作流/审批）完整内容 | Task 3 |
| Capstone 参考实现（完整 MDL） | Task 4 |
| Capstone 种子数据 | Task 5 |
| 视觉完成度（2列布局、过滤器、操作按钮） | Task 2（kb-pages）, Task 3（escalation-pages）|
| mx check 兼容性（通过 mxcli check）| Task 6 |
