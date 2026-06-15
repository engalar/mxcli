# Academy Phase 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在仓库根目录创建 `academy/` 学院目录，包含中文版模块 00–05 的完整课程内容（业务需求文档、AI 协作指南、参考实现 MDL、填空练习），以及 Capstone 的业务需求文档。

**Architecture:** 独立目录，不依赖 `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl`。各模块参考实现 MDL 是累加式的：Module 01 从空项目建立领域模型，后续模块在此基础上叠加。英文版仅建目录占位。

**Tech Stack:** Markdown（文档），MDL（Mendix Definition Language，参考实现和练习）

---

## 文件清单

```
academy/
├── README.md
├── en/
│   └── README.md
└── zh/
    ├── README.md
    ├── 00-入门准备/
    │   ├── 应用愿景.md
    │   ├── AI协作指南.md
    │   ├── 参考实现/
    │   │   └── hello-world.mdl
    │   └── 练习/
    │       └── 练习00-第一次exec.mdl
    ├── 01-领域建模/
    │   ├── 业务需求.md
    │   ├── AI协作指南.md
    │   ├── 参考实现/
    │   │   └── domain-model.mdl
    │   └── 练习/
    │       ├── 练习01-客户实体.mdl
    │       └── 练习02-关联设计.mdl
    ├── 02-微流业务逻辑/
    │   ├── 业务需求.md
    │   ├── AI协作指南.md
    │   ├── 参考实现/
    │   │   └── microflows.mdl
    │   └── 练习/
    │       ├── 练习01-提交工单.mdl
    │       └── 练习02-解决与SLA.mdl
    ├── 03-纳流与客户端/
    │   ├── 业务需求.md
    │   ├── AI协作指南.md
    │   ├── 参考实现/
    │   │   └── nanoflows.mdl
    │   └── 练习/
    │       └── 练习01-快速创建工单.mdl
    ├── 04-页面与UI/
    │   ├── 业务需求.md
    │   ├── AI协作指南.md
    │   ├── 参考实现/
    │   │   └── pages.mdl
    │   └── 练习/
    │       └── 练习01-客户列表页.mdl
    ├── 05-安全与权限/
    │   ├── 业务需求.md
    │   ├── AI协作指南.md
    │   ├── 参考实现/
    │   │   └── security.mdl
    │   └── 练习/
    │       └── 练习01-客户角色授权.mdl
    └── capstone-helpdesk/
        └── 业务需求.md
```

---

## Task 1：目录骨架

**Files:**
- Create: `academy/` 及所有子目录

- [ ] **Step 1：创建完整目录结构**

```bash
mkdir -p academy/en
mkdir -p academy/zh/00-入门准备/参考实现 academy/zh/00-入门准备/练习
mkdir -p academy/zh/01-领域建模/参考实现 academy/zh/01-领域建模/练习
mkdir -p academy/zh/02-微流业务逻辑/参考实现 academy/zh/02-微流业务逻辑/练习
mkdir -p academy/zh/03-纳流与客户端/参考实现 academy/zh/03-纳流与客户端/练习
mkdir -p academy/zh/04-页面与UI/参考实现 academy/zh/04-页面与UI/练习
mkdir -p academy/zh/05-安全与权限/参考实现 academy/zh/05-安全与权限/练习
mkdir -p academy/zh/capstone-helpdesk
```

Expected: 14 directories created, no errors.

- [ ] **Step 2：验证目录结构**

```bash
find academy -type d | sort
```

Expected output (14 lines, each directory listed):
```
academy
academy/en
academy/zh
academy/zh/00-入门准备
academy/zh/00-入门准备/参考实现
academy/zh/00-入门准备/练习
academy/zh/01-领域建模
...（以此类推）
academy/zh/capstone-helpdesk
```

- [ ] **Step 3：提交目录骨架**

```bash
git add academy/
git commit -m "chore(academy): create directory skeleton for Phase 1"
```

---

## Task 2：academy/README.md + en/README.md

**Files:**
- Create: `academy/README.md`
- Create: `academy/en/README.md`

- [ ] **Step 1：写 academy/README.md**

```markdown
# AI 辅助开发学院 / AI-Assisted Development Academy

通过构建一个完整的 IT 支持 Helpdesk 应用，学习如何使用 Claude Code + mxcli 实现 AI 辅助 Mendix 开发。

Learn AI-assisted Mendix development by building a complete IT helpdesk application using Claude Code + mxcli.

---

## 核心学习循环 / Core Learning Loop

```
业务需求文档            AI 协作                  验证
(业务语言)    →    Claude Code + mxcli    →    mx check
"需要工单系统"     /mendix:create-entity       0 错误 = 完成
                       ↓
               参考实现（标准答案）
               参考实现/*.mdl
```

## 课程语言 / Languages

| 语言 | 状态 |
|------|------|
| 🇨🇳 中文（zh/） | ✅ 完整实现 |
| 🇬🇧 English（en/） | 🚧 Coming soon |

## 中文课程目录 / Chinese Curriculum

| 模块 | 主题 | 先决条件 |
|------|------|---------|
| [00 入门准备](zh/00-入门准备/) | 环境搭建 & 第一次 exec | 无 |
| [01 领域建模](zh/01-领域建模/) | 实体、枚举、关联 | 00 |
| [02 微流业务逻辑](zh/02-微流业务逻辑/) | 工单状态机 & 业务规则 | 01 |
| [03 纳流与客户端](zh/03-纳流与客户端/) | 客户端计算 & 快速操作 | 01 |
| [04 页面与UI](zh/04-页面与UI/) | Atlas 布局 & 美观界面 | 01–03 |
| [05 安全与权限](zh/05-安全与权限/) | 角色 & 行级过滤 | 01–04 |
| [Capstone](zh/capstone-helpdesk/) | 完整应用交付 | 所有模块 |

## 技术要求

- mxcli（最新版）：`mxcli --version`
- Claude Code：`claude --version`
- Mendix 项目文件（.mpr）：用于 exec 和 mx check 验证
- mx check 工具：`mxcli setup mxbuild -p app.mpr`（自动下载）
```

- [ ] **Step 2：写 academy/en/README.md**

```markdown
# AI-Assisted Development Academy

> 🚧 **English content coming soon.**
>
> The full curriculum is available in [Chinese (zh/)](../zh/).
> English translations are planned for a future release.

## What You'll Build

A complete IT Helpdesk application using Claude Code + mxcli, covering:
domain modeling → microflows → nanoflows → pages → security → extensions.

## Prerequisites

- mxcli (latest): `mxcli --version`
- Claude Code: `claude --version`
- A Mendix project file (.mpr) for exec and validation
```

- [ ] **Step 3：提交**

```bash
git add academy/README.md academy/en/README.md
git commit -m "docs(academy): add root README and English placeholder"
```

---

## Task 3：zh/README.md（课程总览与学习路径）

**Files:**
- Create: `academy/zh/README.md`

- [ ] **Step 1：写 zh/README.md**

```markdown
# 中文课程总览

本课程以构建一个完整的 IT 支持 Helpdesk 系统为主线，从零开始，逐步掌握 AI 辅助 Mendix 开发的全套技能。

---

## 学习目标

完成全部模块后，你将能够：

1. 使用 Claude Code + mxcli 将业务需求转化为可运行的 Mendix 应用
2. 独立设计领域模型、业务逻辑和用户界面
3. 配置多角色安全访问控制
4. 通过 `mx check` 验证你的应用符合 Mendix 平台规范

---

## 我们要构建的应用

**Helpdesk 工单系统** —— 一个拥有以下功能的 IT 支持平台：

- 客户提交和跟踪支持工单
- 客服受理、指派和解决工单
- 经理审批升级请求和查看 SLA 报告
- 三个角色（Customer / Agent / Manager）各有差异化的界面和权限

---

## 模块路径图

```
[00 入门准备]
      │
      ▼
[01 领域建模] ────────────────────────────────────────┐
      │                                              │
      ▼                                              ▼
[02 微流业务逻辑]    [03 纳流与客户端]         [05 安全与权限]
      │                    │                        │
      └──────────┬──────────┘                        │
                 ▼                                   │
          [04 页面与UI]                               │
                 │                                   │
                 └──────────────┬────────────────────┘
                                ▼
                      [Capstone: 完整交付]
```

---

## 每个模块的文件说明

| 文件 | 内容 | 语言风格 |
|------|------|---------|
| `业务需求.md` | 业务背景、用户故事、验收标准 | **纯业务语言，无技术术语** |
| `AI协作指南.md` | 如何用 Claude Code + mxcli 完成本模块 | 技术操作步骤 |
| `参考实现/*.mdl` | 完整可执行的标准答案 | MDL 代码 |
| `练习/*.mdl` | 填空式练习题（含 TODO 注释） | MDL 代码（部分缺失） |

---

## 快速开始

```bash
# 1. 准备一个干净的 Mendix 11.x 项目
mxcli new MyHelpdesk --version 11.6.6

# 2. 从模块 00 开始
# 阅读 00-入门准备/AI协作指南.md

# 3. 验证工具可用
mxcli --version
claude --version
```
```

- [ ] **Step 2：提交**

```bash
git add academy/zh/README.md
git commit -m "docs(academy): add Chinese course overview README"
```

---

## Task 4：Module 00 — 入门准备

**Files:**
- Create: `academy/zh/00-入门准备/应用愿景.md`
- Create: `academy/zh/00-入门准备/AI协作指南.md`
- Create: `academy/zh/00-入门准备/参考实现/hello-world.mdl`
- Create: `academy/zh/00-入门准备/练习/练习00-第一次exec.mdl`

- [ ] **Step 1：写 应用愿景.md**

```markdown
# 应用愿景：Helpdesk 工单系统

## 背景故事

TechCorp 是一家拥有 500 名员工的科技公司。IT 支持团队每天通过电子邮件和即时消息处理几十个问题请求，管理混乱，经常有问题被遗漏，客户不知道自己的问题处理到哪一步了。

公司决定建设一套 **Helpdesk 工单系统**，让一切井井有条。

---

## 用户群体

| 用户类型 | 谁 | 核心需求 |
|---------|-----|---------|
| **客户** | TechCorp 员工，遇到 IT 问题 | 提交问题，实时了解处理进展 |
| **客服** | IT 支持人员，处理问题 | 高效管理所有请求，不遗漏任何问题 |
| **经理** | IT 支持主管 | 了解团队工作量，确保服务质量 |

---

## 我们要构建什么

一个三角色的工单管理平台：

- 客户可以提交工单，追踪自己的所有问题
- 客服可以看到所有工单，受理、更新和解决
- 经理可以查看整体情况，审批特殊请求

---

## 完成后的样子

当所有课程模块完成后，你将拥有：

- ✅ 完整的工单管理功能（提交 → 受理 → 解决）
- ✅ 知识库（客服撰写帮助文章，客户自助查阅）
- ✅ 三角色权限隔离（每个用户只看到自己该看的）
- ✅ 美观的 Atlas UI 界面（徽章、卡片、双栏布局）
- ✅ `mx check` 零错误（Mendix 平台合规）
```

- [ ] **Step 2：写 AI协作指南.md**

```markdown
# 模块 00：AI 协作指南 — 入门准备

## 你需要准备什么

### 1. 安装 mxcli

```bash
# Windows（PowerShell）
winget install mxcli

# 或从 GitHub Release 页面下载
# https://github.com/mxcli/releases 找最新版本

# 验证安装
mxcli --version
```

### 2. 安装 Claude Code

```bash
npm install -g @anthropic-ai/claude-code
claude --version
```

### 3. 准备一个 Mendix 项目

**方式 A：新建项目（推荐）**

```bash
# 自动下载 Mendix 11.6.6 并创建空项目
mxcli new MyHelpdesk --version 11.6.6
cd MyHelpdesk
```

**方式 B：使用已有项目**

```bash
# 确认项目可以打开
mxcli -p MyProject.mpr -c "show structure"
```

### 4. 下载 mx check 工具

```bash
# 自动下载与你的项目版本匹配的 mxbuild
mxcli setup mxbuild -p MyProject.mpr
```

---

## 你的第一次 AI 协作

打开 Claude Code（在项目目录内）：

```bash
claude
```

试着输入：

```
用 MDL 为我创建一个简单的模块 HD，包含一个 Customer 实体，有 Name 和 Email 两个属性
```

Claude 会生成 MDL 代码。把生成的代码保存为 `test.mdl`，然后：

```bash
# 语法检查
mxcli check test.mdl

# 执行到项目
mxcli exec test.mdl -p MyProject.mpr

# Mendix 平台验证
~/.mxcli/mxbuild/11.6.6/modeler/mx check MyProject.mpr 2>&1 | grep -i "Error\|StorageLoadException"
```

**0 错误 = 成功！**

---

## 理解三步验证循环

```
1. mxcli check file.mdl          → 语法是否正确？
2. mxcli exec file.mdl -p app.mpr → 能否写入项目？
3. mx check app.mpr               → Mendix 平台是否接受？
```

每写完一段 MDL，都要跑这三步。这是本课程贯穿始终的验证节奏。

---

## 参考实现

如果遇到困难，查看 `参考实现/hello-world.mdl`。

**规则：先自己尝试，实在卡住再看参考实现。**
```

- [ ] **Step 3：写 参考实现/hello-world.mdl**

```mdl
-- ============================================================
-- 模块 00：Hello World
-- 这是你的第一个 MDL 文件，展示最基本的结构。
-- 运行：mxcli exec hello-world.mdl -p MyProject.mpr
-- ============================================================

create module HD;

create or modify persistent entity HD.Customer (
  Name:  string(200) not null,
  Email: string(200) not null
);
```

- [ ] **Step 4：写 练习/练习00-第一次exec.mdl**

```mdl
-- ============================================================
-- 练习 00：第一次 exec
-- 目标：在 HD 模块中添加一个 Agent 实体
-- ============================================================

create module HD;

-- 已给你 Customer 实体作为参考
create or modify persistent entity HD.Customer (
  Name:  string(200) not null,
  Email: string(200) not null
);

-- TODO：创建 HD.Agent 实体，包含以下属性：
--   Name  (string, 200 字符, 不能为空)
--   Email (string, 200 字符, 不能为空)
-- 提示：参考上面 HD.Customer 的写法

-- 你的代码写在这里：


-- 验证命令：
--   mxcli check 练习00-第一次exec.mdl
--   mxcli exec  练习00-第一次exec.mdl -p MyProject.mpr
```

- [ ] **Step 5：提交**

```bash
git add academy/zh/00-入门准备/
git commit -m "docs(academy): add module 00 — getting started (zh)"
```

---

## Task 5：Module 01 — 领域建模

**Files:**
- Create: `academy/zh/01-领域建模/业务需求.md`
- Create: `academy/zh/01-领域建模/AI协作指南.md`
- Create: `academy/zh/01-领域建模/参考实现/domain-model.mdl`
- Create: `academy/zh/01-领域建模/练习/练习01-客户实体.mdl`
- Create: `academy/zh/01-领域建模/练习/练习02-关联设计.mdl`

- [ ] **Step 1：写 业务需求.md**

```markdown
# 模块 01：领域建模 — 业务需求

## 业务背景

TechCorp 的 Helpdesk 系统需要跟踪三类核心信息：**谁提交了问题**（客户）、**谁在处理问题**（客服）、**具体是什么问题**（工单）。

没有这些基础信息，系统就无从运转——就像图书馆没有书目录一样。

---

## 用户故事

### 客户信息管理
- 作为 IT 客服，我想知道是哪位员工提交了工单，这样我可以联系到他们
- 作为 IT 客服，我想知道员工在哪个部门，这样我可以判断问题的影响范围
- 作为经理，我想按公司/部门查看工单分布，以便合理分配资源

### 工单信息
- 作为客服，我想知道问题的标题和详细描述，以便快速理解要解决什么
- 作为客服，我想知道问题的紧急程度，以便优先处理最重要的事情
- 作为客服，我想看到工单的当前状态（草稿/待处理/处理中/已解决/已关闭），以便知道还有多少工作要做
- 作为客户，我想看到我的工单什么时候必须得到解决（SLA 截止时间），以便设置合理的期望

### 客服信息
- 作为经理，我想知道哪些客服是活跃的，以便合理安排工单指派
- 作为系统，我需要记录工单被指派给了哪位客服，以便追踪责任人

---

## 验收标准

- [ ] 系统能记录客户的姓名、邮箱、所属公司
- [ ] 系统能记录客服的姓名、邮箱，以及是否在岗
- [ ] 每张工单有标题、描述、紧急程度、当前状态、SLA 截止时间
- [ ] 工单必须关联到一位客户（是谁提交的）
- [ ] 工单可以关联到一位客服（被谁处理）
- [ ] 紧急程度分为四级：低 / 普通 / 高 / 紧急
- [ ] 工单状态分为五种：草稿 / 待处理 / 处理中 / 已解决 / 已关闭
```

- [ ] **Step 2：写 AI协作指南.md**

```markdown
# 模块 01：AI 协作指南 — 领域建模

## 本模块产出

运行完本模块的参考实现后，你的项目将拥有：
- HD 模块
- 2 个枚举（工单状态、工单优先级）
- 3 个实体（Customer, Agent, Ticket）+ 1 个非持久化实体（TicketSearch）
- 2 个关联（Ticket→Customer, Ticket→Agent）
- 2 个常量（SLA 小时数）

## 与 Claude 协作的步骤

### Step 1：让 Claude 读取业务需求

在 Claude Code 中输入：

```
读取 academy/zh/01-领域建模/业务需求.md，帮我设计 Mendix 领域模型（实体、枚举、关联），用 MDL 实现
```

### Step 2：逐步确认设计

Claude 会提出一个设计方案。在确认前，检查：

1. 枚举值是否覆盖了需求中提到的所有状态和优先级？
2. 每个实体的属性是否完整？类型是否合适？
3. 关联方向是否正确？（工单→客户，工单→客服）

如果有问题，直接告诉 Claude：

```
工单还需要一个 "解决时间" 属性，类型是日期时间
```

### Step 3：生成 MDL 并验证

```bash
# 保存 Claude 生成的 MDL 为 my-domain.mdl，然后：
mxcli check my-domain.mdl
mxcli exec  my-domain.mdl -p MyProject.mpr
~/.mxcli/mxbuild/*/modeler/mx check MyProject.mpr 2>&1 | grep -c "StorageLoadException"
# 期望：0
```

### Step 4：常见坑

| 坑 | 症状 | 解决方法 |
|----|------|---------|
| 枚举引用错误 | `mxcli check` 报 "unknown type" | 确认枚举在实体之前定义 |
| 关联方向反了 | mx check 报 CE0XXX | from = 拥有外键的一方（Ticket），to = 被引用的一方（Customer） |
| 属性类型错误 | mx check 报 StorageLoadException | string 类型需要长度：`string(200)`，不能直接写 `string` |

## 参考实现

卡住了就看 `参考实现/domain-model.mdl`。注意：实体语法是 `create or modify persistent entity`，不要写成 `create entity`。
```

- [ ] **Step 3：写 参考实现/domain-model.mdl**

```mdl
-- ============================================================
-- 模块 01：领域建模 — 参考实现
-- 从空项目运行：mxcli exec domain-model.mdl -p MyProject.mpr
-- 验证：~/.mxcli/mxbuild/*/modeler/mx check MyProject.mpr
-- ============================================================

create module HD;

-- ============================================================
-- 枚举：工单状态（5 个状态，构成状态机）
-- ============================================================

create or modify enumeration HD.TicketStatus (
  Draft      'Draft',
  Open       'Open',
  InProgress 'In Progress',
  Resolved   'Resolved',
  Closed     'Closed'
);

-- ============================================================
-- 枚举：工单优先级（4 个级别）
-- ============================================================

create or modify enumeration HD.TicketPriority (
  Low      'Low',
  Normal   'Normal',
  High     'High',
  Critical 'Critical'
);

-- ============================================================
-- 实体：客户（提交工单的员工）
-- system members (owner) 让 Mendix 追踪记录的创建者
-- ============================================================

create or modify persistent entity HD.Customer (
  Name:    string(200) not null,
  Email:   string(200) not null,
  Company: string(200)
)
system members (owner)
index (Email);

-- ============================================================
-- 实体：客服（处理工单的 IT 支持人员）
-- ============================================================

create or modify persistent entity HD.Agent (
  Name:     string(200) not null,
  Email:    string(200) not null,
  IsActive: boolean default true
);

-- ============================================================
-- 实体：工单（核心业务对象）
-- system members (owner, createdDate, changedDate, changedBy)
-- 让 Mendix 自动记录创建者、创建时间、修改时间、修改人
-- ============================================================

create or modify persistent entity HD.Ticket (
  Subject:     string(500) not null,
  Description: string,
  Status:      HD.TicketStatus   default Draft,
  Priority:    HD.TicketPriority default Normal,
  SLADueAt:   datetime,
  ResolvedAt:  datetime,
  IsOverSLA:   boolean default false
)
system members (owner, createdDate, changedDate, changedBy)
index (Status)
index (Priority)
index (SLADueAt);

-- ============================================================
-- 实体：工单评论（记录处理过程中的沟通）
-- ============================================================

create or modify persistent entity HD.TicketComment (
  Content:    string not null,
  IsInternal: boolean default false
);

-- ============================================================
-- 非持久化实体：工单搜索表单（客户端过滤，不存库）
-- ============================================================

create or modify non-persistent entity HD.TicketSearch (
  SubjectKeyword: string(200),
  StatusFilter:   HD.TicketStatus,
  PriorityFilter: HD.TicketPriority
);

-- ============================================================
-- 关联：工单 → 客户（多对一：多张工单属于一个客户）
-- from HD.Ticket to HD.Customer = Ticket 持有 Customer 的外键
-- ============================================================

create or modify association HD.Ticket_Customer
  from HD.Ticket to HD.Customer
  type reference
  owner default;

-- ============================================================
-- 关联：工单 → 客服（可空：工单初始可能尚未指派）
-- ============================================================

create or modify association HD.Ticket_Agent
  from HD.Ticket to HD.Agent
  type reference
  owner default;

-- ============================================================
-- 关联：评论 → 工单（多对一：多条评论属于一张工单）
-- ============================================================

create or modify association HD.TicketComment_Ticket
  from HD.TicketComment to HD.Ticket
  type reference
  owner default;

-- ============================================================
-- 常量：SLA 截止时间（小时数），可按环境配置
-- ============================================================

create or modify constant HD.SLA_HIGH_HOURS
  type integer
  default 8
  comment 'SLA deadline in hours for High-priority tickets.';

create or modify constant HD.SLA_CRITICAL_HOURS
  type integer
  default 2
  comment 'SLA deadline in hours for Critical-priority tickets.';
```

- [ ] **Step 4：写 练习/练习01-客户实体.mdl**

```mdl
-- ============================================================
-- 练习 01：客户实体
-- 目标：补全 HD.Customer 实体的缺失属性
-- ============================================================

create module HD;

-- HD.Customer 实体已给出 Name 属性，请补全其余属性
create or modify persistent entity HD.Customer (
  Name: string(200) not null
  -- TODO 1：添加 Email 属性（string, 200 字符, not null）
  -- TODO 2：添加 Company 属性（string, 200 字符, 可以为空）
)
system members (owner);

-- HD.Agent 实体需要你从头编写
-- 业务需求：姓名、邮箱、是否在岗（默认在岗）
-- TODO 3：在此处创建 HD.Agent 实体

-- 验证命令：
--   mxcli check 练习01-客户实体.mdl
--   mxcli exec  练习01-客户实体.mdl -p MyProject.mpr
```

- [ ] **Step 5：写 练习/练习02-关联设计.mdl**

```mdl
-- ============================================================
-- 练习 02：关联设计
-- 前提：先运行 练习01-客户实体.mdl（或 参考实现/domain-model.mdl）
-- 目标：创建工单实体，并建立与客户的关联
-- ============================================================

-- 工单枚举（已给出，不需要修改）
create or modify enumeration HD.TicketStatus (
  Draft 'Draft', Open 'Open', InProgress 'In Progress',
  Resolved 'Resolved', Closed 'Closed'
);

create or modify enumeration HD.TicketPriority (
  Low 'Low', Normal 'Normal', High 'High', Critical 'Critical'
);

-- HD.Ticket 实体已给出基本属性，请补全
create or modify persistent entity HD.Ticket (
  Subject:     string(500) not null,
  Description: string,
  Status:      HD.TicketStatus   default Draft,
  Priority:    HD.TicketPriority default Normal
  -- TODO 1：添加 SLADueAt 属性（datetime, 可为空）
  -- TODO 2：添加 ResolvedAt 属性（datetime, 可为空）
  -- TODO 3：添加 IsOverSLA 属性（boolean, 默认 false）
)
system members (owner, createdDate, changedDate, changedBy);

-- TODO 4：创建 HD.Ticket_Customer 关联
-- 要求：from HD.Ticket to HD.Customer，type reference，owner default
-- 提示：从 Ticket 到 Customer（Ticket 持有外键）

-- TODO 5：创建 HD.Ticket_Agent 关联（可空，表示尚未指派）
-- 要求：同上，from HD.Ticket to HD.Agent

-- 验证命令：
--   mxcli check 练习02-关联设计.mdl
--   mxcli exec  练习02-关联设计.mdl -p MyProject.mpr
```

- [ ] **Step 6：提交**

```bash
git add academy/zh/01-领域建模/
git commit -m "docs(academy): add module 01 — domain modeling (zh)"
```

---

## Task 6：Module 02 — 微流业务逻辑

**Files:**
- Create: `academy/zh/02-微流业务逻辑/业务需求.md`
- Create: `academy/zh/02-微流业务逻辑/AI协作指南.md`
- Create: `academy/zh/02-微流业务逻辑/参考实现/microflows.mdl`
- Create: `academy/zh/02-微流业务逻辑/练习/练习01-提交工单.mdl`
- Create: `academy/zh/02-微流业务逻辑/练习/练习02-解决与SLA.mdl`

**前提：** 执行者需先在目标项目运行 `01-领域建模/参考实现/domain-model.mdl`。

- [ ] **Step 1：写 业务需求.md**

```markdown
# 模块 02：微流业务逻辑 — 业务需求

## 业务背景

光有数据结构还不够——系统需要知道"在什么情况下可以做什么"。工单不是随便就能提交的，处理过程也要按业务规则一步步来。这些规则由**业务逻辑**来保障。

---

## 工单的一生

```
草稿 ──[提交]──► 待处理 ──[指派]──► 处理中 ──[解决]──► 已解决 ──[关闭]──► 已关闭
                   ▲                                        │
                   └──────────────────[重开]────────────────┘
```

---

## 用户故事

### 提交工单
- 作为客户，我想提交工单，系统自动设置合理的截止时间，这样我知道何时能得到答复
- 作为系统，提交时必须检查工单标题不能为空，否则给出提示
- 作为系统，紧急工单（Critical）的截止时间应该是 2 小时，高优先级（High）是 8 小时，其他是 24 小时

### 指派处理
- 作为客服，我想认领待处理的工单，让工单状态变为"处理中"，并显示是我在负责
- 作为系统，只有"待处理"状态的工单才能被指派

### 解决工单
- 作为客服，当我解决了问题，我想标记工单为"已解决"，并记录解决时间
- 作为系统，如果解决时间超过了承诺的截止时间，自动标记为"逾期"

### 重开工单
- 作为客户，如果问题没有真正解决，我想重新打开工单请求进一步帮助
- 作为系统，只有"已解决"或"已关闭"的工单才能重开

### 关闭工单
- 作为系统，经过确认后，将已解决的工单正式关闭（终态，不可再改变状态）

---

## 验收标准

- [ ] 提交时标题为空 → 显示错误提示，工单不进入待处理
- [ ] 提交 Critical 工单 → SLA 截止时间 = 当前时间 + 2 小时
- [ ] 提交 High 工单 → SLA 截止时间 = 当前时间 + 8 小时
- [ ] 提交 Low/Normal 工单 → SLA 截止时间 = 当前时间 + 24 小时
- [ ] 指派：待处理 → 处理中，记录负责客服
- [ ] 指派非待处理工单 → 给出警告提示
- [ ] 解决：处理中 → 已解决，记录解决时间
- [ ] 解决时超出 SLA → IsOverSLA 自动标记为 true
- [ ] 重开：已解决/已关闭 → 待处理，清除解决时间
- [ ] 关闭：已解决 → 已关闭（终态）
```

- [ ] **Step 2：写 AI协作指南.md**

```markdown
# 模块 02：AI 协作指南 — 微流业务逻辑

## 前提

先运行领域模型：

```bash
mxcli exec academy/zh/01-领域建模/参考实现/domain-model.mdl -p MyProject.mpr
```

## 与 Claude 协作的步骤

### Step 1：让 Claude 实现状态机

```
读取 academy/zh/02-微流业务逻辑/业务需求.md，
帮我用 MDL 实现工单的业务逻辑微流：
- HD.ACT_Ticket_Submit（提交）
- HD.ACT_Ticket_Assign（指派）
- HD.ACT_Ticket_Resolve（解决）
- HD.ACT_Ticket_Reopen（重开）
- HD.ACT_Ticket_Close（关闭）
```

### Step 2：验证 SLA 计算

SLA 计算用 `addHours('[%CurrentDateTime%]', N)` 函数，N 可以是常量引用 `@HD.SLA_HIGH_HOURS`。

要求 Claude 确认生成的微流中包含：

```mdl
change $Ticket (
  Status   = HD.TicketStatus.Open,
  SLADueAt = addHours('[%CurrentDateTime%]', @HD.SLA_CRITICAL_HOURS)
);
```

### Step 3：验证状态前置检查

每个微流都应该先检查当前状态是否正确，例如：

```mdl
if $Ticket/Status != HD.TicketStatus.Open {
  show message 'Only Open tickets can be assigned.' type warning;
  return false;
}
```

### Step 4：常见坑

| 坑 | 原因 | 解决 |
|----|------|------|
| 常量引用报错 | 写成 `HD.SLA_HIGH_HOURS` 而非 `@HD.SLA_HIGH_HOURS` | 加 `@` 前缀 |
| SLA 计算只比较日期不比较时间 | 逾期判断漏了 `and $Ticket/SLADueAt != empty` | 加空值保护 |
| 重开时没清除 ResolvedAt | 只改了 Status | 同时 `change $Ticket (ResolvedAt = empty)` |

## 参考实现

看 `参考实现/microflows.mdl`。
```

- [ ] **Step 3：写 参考实现/microflows.mdl**

```mdl
-- ============================================================
-- 模块 02：微流业务逻辑 — 参考实现
-- 前提：先运行 01-领域建模/参考实现/domain-model.mdl
-- 运行：mxcli exec microflows.mdl -p MyProject.mpr
-- ============================================================

-- ============================================================
-- 提交工单：Draft → Open，计算 SLA 截止时间
-- ============================================================

create or modify microflow HD.ACT_Ticket_Submit
  ($Ticket: HD.Ticket)
  returns boolean as $Success
  folder 'Ticket'
{
  if $Ticket/Subject = '' {
    validation feedback $Ticket/Subject message 'Subject is required.';
    return false;
  }

  if $Ticket/Priority = HD.TicketPriority.Critical {
    change $Ticket (
      Status   = HD.TicketStatus.Open,
      SLADueAt = addHours('[%CurrentDateTime%]', @HD.SLA_CRITICAL_HOURS)
    );
  } else {
    if $Ticket/Priority = HD.TicketPriority.High {
      change $Ticket (
        Status   = HD.TicketStatus.Open,
        SLADueAt = addHours('[%CurrentDateTime%]', @HD.SLA_HIGH_HOURS)
      );
    } else {
      change $Ticket (
        Status   = HD.TicketStatus.Open,
        SLADueAt = addHours('[%CurrentDateTime%]', 24)
      );
    }
  }
  commit $Ticket;
  return true;
}
/

-- ============================================================
-- 指派客服：Open → InProgress
-- ============================================================

create or modify microflow HD.ACT_Ticket_Assign
  ($Ticket: HD.Ticket, $Agent: HD.Agent)
  returns boolean as $Success
  folder 'Ticket'
{
  if $Ticket/Status != HD.TicketStatus.Open {
    show message 'Only Open tickets can be assigned.' type warning;
    return false;
  }
  change $Ticket (
    Status         = HD.TicketStatus.InProgress,
    HD.Ticket_Agent = $Agent
  );
  commit $Ticket;
  return true;
}
/

-- ============================================================
-- 解决工单：InProgress → Resolved，计算是否逾期
-- ============================================================

create or modify microflow HD.ACT_Ticket_Resolve
  ($Ticket: HD.Ticket)
  returns boolean as $Success
  folder 'Ticket'
{
  if $Ticket/Status != HD.TicketStatus.InProgress {
    return false;
  }
  declare $Now: datetime    = '[%CurrentDateTime%]';
  declare $IsOver: boolean = $Now > $Ticket/SLADueAt and $Ticket/SLADueAt != empty;
  change $Ticket (
    Status     = HD.TicketStatus.Resolved,
    ResolvedAt = $Now,
    IsOverSLA  = $IsOver
  );
  commit $Ticket;
  return true;
}
/

-- ============================================================
-- 重开工单：Resolved/Closed → Open，清除解决时间
-- ============================================================

create or modify microflow HD.ACT_Ticket_Reopen
  ($Ticket: HD.Ticket)
  returns boolean as $Success
  folder 'Ticket'
{
  if $Ticket/Status != HD.TicketStatus.Resolved
     and $Ticket/Status != HD.TicketStatus.Closed {
    return false;
  }
  change $Ticket (
    Status     = HD.TicketStatus.Open,
    ResolvedAt = empty
  );
  commit $Ticket;
  return true;
}
/

-- ============================================================
-- 关闭工单：Resolved → Closed（终态）
-- ============================================================

create or modify microflow HD.ACT_Ticket_Close
  ($Ticket: HD.Ticket)
  returns boolean as $Success
  folder 'Ticket'
{
  if $Ticket/Status != HD.TicketStatus.Resolved {
    return false;
  }
  change $Ticket (Status = HD.TicketStatus.Closed);
  commit $Ticket;
  return true;
}
/

-- ============================================================
-- 数据源：当前用户的工单（用于 My Tickets 页面）
-- ============================================================

create or modify microflow HD.DS_MyTickets
  ()
  returns list of HD.Ticket as $Tickets
  folder 'Ticket'
{
  retrieve $Tickets from HD.Ticket
    where [System.owner = '[%CurrentUser%]']
    limit 0;
  return $Tickets;
}
/
```

- [ ] **Step 4：写 练习/练习01-提交工单.mdl**

```mdl
-- ============================================================
-- 练习 01：提交工单微流
-- 前提：先运行 01-领域建模/参考实现/domain-model.mdl
-- 目标：补全 ACT_Ticket_Submit 的缺失部分
-- ============================================================

create or modify microflow HD.ACT_Ticket_Submit
  ($Ticket: HD.Ticket)
  returns boolean as $Success
  folder 'Ticket'
{
  -- TODO 1：添加标题非空校验
  -- 如果 $Ticket/Subject = '' 则显示验证反馈并 return false
  -- 提示：validation feedback $Ticket/Subject message '...';

  -- TODO 2：按优先级计算 SLA 截止时间并更新状态
  -- Critical → addHours('[%CurrentDateTime%]', @HD.SLA_CRITICAL_HOURS)
  -- High     → addHours('[%CurrentDateTime%]', @HD.SLA_HIGH_HOURS)
  -- 其他     → addHours('[%CurrentDateTime%]', 24)
  -- 提示：用 if / else 嵌套

  -- TODO 3：commit $Ticket 并 return true

}
/

-- 验证命令：
--   mxcli check 练习01-提交工单.mdl
--   mxcli exec  练习01-提交工单.mdl -p MyProject.mpr
```

- [ ] **Step 5：写 练习/练习02-解决与SLA.mdl**

```mdl
-- ============================================================
-- 练习 02：解决工单与 SLA 逾期计算
-- 前提：先运行练习01（或参考实现）
-- 目标：实现 ACT_Ticket_Resolve 微流
-- ============================================================

create or modify microflow HD.ACT_Ticket_Resolve
  ($Ticket: HD.Ticket)
  returns boolean as $Success
  folder 'Ticket'
{
  -- TODO 1：前置状态检查
  -- 如果 Status 不是 InProgress，直接 return false

  -- TODO 2：用 declare 声明当前时间变量
  -- declare $Now: datetime = '[%CurrentDateTime%]';

  -- TODO 3：用 declare 计算是否逾期
  -- declare $IsOver: boolean = ...
  -- 逾期条件：当前时间 > SLA 截止时间，且 SLA 截止时间不为空

  -- TODO 4：change $Ticket 更新三个字段：Status, ResolvedAt, IsOverSLA

  -- TODO 5：commit 并 return true

}
/

-- 验证命令：
--   mxcli check 练习02-解决与SLA.mdl
--   mxcli exec  练习02-解决与SLA.mdl -p MyProject.mpr
```

- [ ] **Step 6：提交**

```bash
git add academy/zh/02-微流业务逻辑/
git commit -m "docs(academy): add module 02 — microflows (zh)"
```

---

## Task 7：Module 03 — 纳流与客户端

**Files:**
- Create: `academy/zh/03-纳流与客户端/业务需求.md`
- Create: `academy/zh/03-纳流与客户端/AI协作指南.md`
- Create: `academy/zh/03-纳流与客户端/参考实现/nanoflows.mdl`
- Create: `academy/zh/03-纳流与客户端/练习/练习01-快速创建工单.mdl`

- [ ] **Step 1：写 业务需求.md**

```markdown
# 模块 03：纳流与客户端 — 业务需求

## 业务背景

并非所有操作都需要访问服务器。有些简单操作——比如快速创建一张新工单、在列表里搜索过滤——可以直接在用户的浏览器里完成，响应更快，体验更好。

---

## 用户故事

### 快速创建工单
- 作为客户，我想在一个简洁的弹窗里快速提交工单标题，不需要跳转到另一个页面，这样我可以更快求助
- 作为系统，快速创建的工单默认是"草稿"状态，优先级为"普通"

### 工单搜索过滤
- 作为客服，我想在工单列表页面快速搜索包含某个关键词的工单，而不需要刷新整个页面
- 作为系统，搜索结果应该实时响应，无需服务器往返

### 优先级标签格式化
- 作为客服，我想在工单列表中看到人性化的优先级标签（如"⚠️ 高"），而不是英文枚举值
- 作为系统，格式化逻辑应该在客户端完成，不需要服务器计算

---

## 验收标准

- [ ] 快速创建：输入标题 → 工单保存到数据库 → 状态草稿，优先级普通
- [ ] 搜索：输入关键词 → 过滤结果不刷新整页
- [ ] 优先级标签：`Critical` → `🔴 紧急`，`High` → `🟠 高`，`Normal` → `🟡 普通`，`Low` → `🟢 低`
```

- [ ] **Step 2：写 AI协作指南.md**

```markdown
# 模块 03：AI 协作指南 — 纳流与客户端

## 微流 vs 纳流

| 特性 | 微流（Microflow） | 纳流（Nanoflow） |
|------|-----------------|----------------|
| 运行环境 | 服务器 | 客户端（浏览器） |
| 数据库访问 | 支持（完整） | 支持（有限） |
| 适合场景 | 复杂业务逻辑 | 简单操作、快速响应 |
| MDL 语法 | `create or modify microflow` | `create or modify nanoflow` |

## 与 Claude 协作的步骤

```
帮我用 MDL 实现三个纳流：
1. HD.NF_Ticket_QuickCreate：参数 $Customer（HD.Customer）和 $Subject（string），
   创建一张草稿工单并 commit，返回 HD.Ticket
2. HD.NF_TicketSearch_Apply：参数 $Search（HD.TicketSearch），
   从数据库检索工单（limit 100），返回 HD.Ticket 列表
3. HD.NF_Priority_GetLabel：参数 $Priority（HD.TicketPriority），
   返回字符串（Critical→'🔴 紧急', High→'🟠 高', Normal→'🟡 普通', Low→'🟢 低'）
```

## 常见坑

- 纳流的 `retrieve` 不支持复杂 XPath 引用，用简单的 `where [Status = 'Open']` 即可
- 纳流不支持 `show message`，只能 `return`
- 纳流返回列表时类型写法：`returns list of HD.Ticket as $Tickets`
```

- [ ] **Step 3：写 参考实现/nanoflows.mdl**

```mdl
-- ============================================================
-- 模块 03：纳流与客户端 — 参考实现
-- 前提：先运行 01-领域建模/参考实现/domain-model.mdl
-- 运行：mxcli exec nanoflows.mdl -p MyProject.mpr
-- ============================================================

-- ============================================================
-- 快速创建工单：客户端直接创建 Draft 工单
-- ============================================================

create or modify nanoflow HD.NF_Ticket_QuickCreate
  ($Customer: HD.Customer, $Subject: string)
  returns HD.Ticket as $Ticket
  folder 'Ticket'
{
  $Ticket = create HD.Ticket (
    Subject          = $Subject,
    Status           = HD.TicketStatus.Draft,
    Priority         = HD.TicketPriority.Normal,
    HD.Ticket_Customer = $Customer
  );
  commit $Ticket;
  return $Ticket;
}
/

-- ============================================================
-- 工单搜索：返回匹配工单列表（简化版：全量检索）
-- ============================================================

create or modify nanoflow HD.NF_TicketSearch_Apply
  ($Search: HD.TicketSearch)
  returns list of HD.Ticket as $Tickets
  folder 'Ticket/Search'
{
  retrieve $Tickets from HD.Ticket limit 100;
  return $Tickets;
}
/

-- ============================================================
-- 优先级标签格式化：枚举值 → 人性化字符串
-- ============================================================

create or modify nanoflow HD.NF_Priority_GetLabel
  ($Priority: HD.TicketPriority)
  returns string as $Label
  folder 'Ticket'
{
  if $Priority = HD.TicketPriority.Critical {
    return '🔴 紧急';
  } else {
    if $Priority = HD.TicketPriority.High {
      return '🟠 高';
    } else {
      if $Priority = HD.TicketPriority.Normal {
        return '🟡 普通';
      } else {
        return '🟢 低';
      }
    }
  }
}
/
```

- [ ] **Step 4：写 练习/练习01-快速创建工单.mdl**

```mdl
-- ============================================================
-- 练习 01：快速创建工单纳流
-- 前提：先运行 01-领域建模/参考实现/domain-model.mdl
-- 目标：补全 NF_Ticket_QuickCreate 的缺失部分
-- ============================================================

create or modify nanoflow HD.NF_Ticket_QuickCreate
  ($Customer: HD.Customer, $Subject: string)
  returns HD.Ticket as $Ticket
  folder 'Ticket'
{
  -- TODO 1：用 create 创建工单对象
  -- 要求：Subject = $Subject, Status = Draft, Priority = Normal
  -- 用 HD.Ticket_Customer = $Customer 设置关联
  -- 提示：$Ticket = create HD.Ticket ( ... );

  -- TODO 2：commit $Ticket

  -- TODO 3：return $Ticket
}
/

-- 验证命令：
--   mxcli check 练习01-快速创建工单.mdl
--   mxcli exec  练习01-快速创建工单.mdl -p MyProject.mpr
```

- [ ] **Step 5：提交**

```bash
git add academy/zh/03-纳流与客户端/
git commit -m "docs(academy): add module 03 — nanoflows (zh)"
```

---

## Task 8：Module 04 — 页面与 UI

**Files:**
- Create: `academy/zh/04-页面与UI/业务需求.md`
- Create: `academy/zh/04-页面与UI/AI协作指南.md`
- Create: `academy/zh/04-页面与UI/参考实现/pages.mdl`
- Create: `academy/zh/04-页面与UI/练习/练习01-客户列表页.mdl`

- [ ] **Step 1：写 业务需求.md**

```markdown
# 模块 04：页面与 UI — 业务需求

## 业务背景

数据和逻辑有了，还需要让用户能看到、能操作。一个好的界面让客服效率倍增，一个差的界面让人用得抓狂。TechCorp 的管理层要求界面要直观、整洁，重要信息一眼看清。

---

## 用户故事

### 工单列表（所有工单）
- 作为客服，我想看到所有工单的列表，可以按状态和优先级过滤，这样我可以快速找到需要处理的工单
- 作为客服，我想在列表中看到每张工单的状态（显示为彩色徽章），这样不需要打开工单就知道进度
- 作为客服，我想点击一张工单直接打开详情，而不是在新窗口里跳来跳去

### 我的工单（客户视角）
- 作为客户，我想看到只属于我的工单，不想看到其他人的问题
- 作为客户，我想直接在列表页创建新工单，快捷方便

### 工单详情
- 作为任何人，我想在工单详情页看到所有重要信息（标题、描述、状态、优先级、截止时间）
- 作为客服，我想在工单详情页直接执行操作（提交、指派、解决、重开）
- 作为任何人，我想看到该工单的所有评论记录
- 作为客服，我想在详情页添加评论（内部备注或外部回复）

### 新建/编辑工单
- 作为客户，我想填写工单标题和描述，系统自动设置默认状态
- 作为客服，我想在编辑时调整优先级

---

## 验收标准

- [ ] 工单列表：显示标题、状态（彩色徽章）、优先级、SLA 截止时间、操作按钮
- [ ] 工单列表：有状态下拉过滤器和标题文字搜索过滤器
- [ ] 工单列表："New Ticket" 按钮导向新建表单
- [ ] 工单详情：2 列布局（左边信息，右边操作按钮）
- [ ] 工单详情：底部显示评论列表
- [ ] 工单详情：有"Add Comment"按钮弹出评论输入框
- [ ] 新建表单：标题（文本框）、描述（多行文本）、优先级（下拉选择）
- [ ] 我的工单：仅显示属于当前用户的工单，有"New Ticket"按钮
```

- [ ] **Step 2：写 AI协作指南.md**

```markdown
# 模块 04：AI 协作指南 — 页面与 UI

## 前提

先运行模块 01–03 的参考实现。

## Atlas UI 组件速查

| 需求 | Atlas 组件 | MDL 关键字 |
|------|-----------|-----------|
| 列表页 | DataGrid | `datagrid` |
| 详情/表单 | DataView | `dataview` |
| 弹窗页面 | PopupLayout | `layout: Atlas_Core.PopupLayout` |
| 2 列布局 | LayoutGrid | `layoutgrid ... row ... column (desktopwidth: 8)` |
| 按钮 | ActionButton | `actionbutton ... buttonstyle: primary` |
| 文本展示 | DynamicText | `dynamictext` |
| 文本输入 | TextBox | `textbox` |
| 多行输入 | TextArea | `textarea` |

## 与 Claude 协作的步骤

```
帮我用 MDL 创建 Helpdesk 的主要页面：
1. HD.Ticket_Overview：工单列表（所有工单），用 Atlas_Default 布局，DataGrid 展示，
   列包括 Subject/Status/Priority/SLADueAt，有文字过滤器和下拉过滤器，
   有 New Ticket 和 Open（打开详情）按钮
2. HD.Ticket_Detail：工单详情，用 2 列布局（左 8 右 4），左边 DataView 展示字段，
   右边放操作按钮（Submit/Assign/Resolve/Reopen/Close），底部有评论 DataGrid
3. HD.Ticket_NewEdit：新建/编辑表单，Subject textbox + Description textarea + Save/Cancel
4. HD.MyTickets_Overview：我的工单列表，datasource: microflow HD.DS_MyTickets，
   有 New Ticket 按钮和 Open 按钮
5. HD.AddComment_Form：弹窗，输入 Content（textarea）和 IsInternal（checkbox），Save 按钮
```

## 常见坑

| 坑 | 解决 |
|----|------|
| `action: show_page` 需要 page params | 写法：`show_page HD.Ticket_Detail (Ticket: $currentObject)` |
| 弹窗页面需要 PopupLayout | `layout: Atlas_Core.PopupLayout` |
| 页面 params 格式 | `params: { $Ticket: HD.Ticket }` |
| DataView 绑定 page param | `datasource: $Ticket` |
| 按钮在 footer 里 | `footer ftrActions { actionbutton ... }` |
```

- [ ] **Step 3：写 参考实现/pages.mdl**

```mdl
-- ============================================================
-- 模块 04：页面与 UI — 参考实现
-- 前提：先运行模块 01–03 的参考实现
-- 运行：mxcli exec pages.mdl -p MyProject.mpr
-- ============================================================

-- ============================================================
-- 添加评论微流（支撑 AddComment_Form 页面）
-- ============================================================

create or modify microflow HD.ACT_AddComment
  ($Ticket: HD.Ticket, $Comment: HD.TicketComment)
  returns boolean as $Success
  folder 'Ticket'
{
  change $Comment (HD.TicketComment_Ticket = $Ticket);
  commit $Comment;
  return true;
}
/

-- ============================================================
-- 弹窗：新建评论（Content + IsInternal）
-- ============================================================

create or replace page HD.AddComment_Form
(
  title:  'Add Comment',
  layout: Atlas_Core.PopupLayout,
  folder: 'Ticket',
  params: { $Ticket: HD.Ticket }
)
{
  dataview dvComment (datasource: new HD.TicketComment) {
    textarea   taContent    (label: 'Comment', attribute: Content)
    footer ftrButtons {
      actionbutton btnSave (
        caption:     'Save',
        action:      microflow HD.ACT_AddComment (Ticket: $Ticket, Comment: $currentObject) close_page,
        buttonstyle: primary
      )
      actionbutton btnCancel (caption: 'Cancel', action: cancel_changes close_page)
    }
  }
};

-- ============================================================
-- 工单列表（全部工单）
-- 2 列过滤器 + 状态/优先级/SLA 列 + Action 列
-- ============================================================

create or replace page HD.Ticket_Overview
(
  title:  'All Tickets',
  layout: Atlas_Core.Atlas_Default,
  folder: 'Ticket'
)
{
  layoutgrid lgMain {
    row rHeader {
      column cTitle (desktopwidth: 12) {
        dynamictext txtTitle (content: 'All Tickets', rendermode: H2)
      }
    }
    row rMain {
      column cMain (desktopwidth: 12) {
        datagrid dgTickets (
          datasource:     database from HD.Ticket sort by SLADueAt asc,
          PageSize:       20,
          PagingPosition: both
        ) {
          controlbar cb {
            actionbutton btnNew (
              caption:     'New Ticket',
              action:      create_object HD.Ticket then show_page HD.Ticket_NewEdit,
              buttonstyle: primary
            )
          }
          column colSubject  (attribute: Subject,   caption: 'Subject') {
            textfilter fSubject
          }
          column colStatus   (attribute: Status,    caption: 'Status',   ColumnWidth: manual, Size: 120) {
            dropdownfilter fStatus
          }
          column colPriority (attribute: Priority,  caption: 'Priority', ColumnWidth: manual, Size: 100) {
            dropdownfilter fPriority
          }
          column colSLADue   (attribute: SLADueAt,  caption: 'SLA Due',  ColumnWidth: manual, Size: 140)
          column colOverdue  (attribute: IsOverSLA, caption: 'Overdue',  ColumnWidth: manual, Size: 80)
          column colActions  (caption: 'Actions', ShowContentAs: customContent, ColumnWidth: manual, Size: 80) {
            actionbutton btnOpen (
              caption:     'Open',
              action:      show_page HD.Ticket_Detail (Ticket: $currentObject),
              buttonstyle: link
            )
          }
        }
      }
    }
  }
};

-- ============================================================
-- 我的工单（客户视角，使用微流数据源过滤）
-- ============================================================

create or replace page HD.MyTickets_Overview
(
  title:  'My Tickets',
  layout: Atlas_Core.Atlas_Default,
  folder: 'Ticket'
)
{
  layoutgrid lgMain {
    row rHeader {
      column cTitle (desktopwidth: 12) {
        dynamictext txtTitle (content: 'My Tickets', rendermode: H2)
      }
    }
    row rMain {
      column cMain (desktopwidth: 12) {
        datagrid dgMyTickets (
          datasource: microflow HD.DS_MyTickets,
          PageSize:   20
        ) {
          controlbar cb {
            actionbutton btnNew (
              caption:     'New Ticket',
              action:      create_object HD.Ticket then show_page HD.Ticket_NewEdit,
              buttonstyle: primary
            )
          }
          column colSubject  (attribute: Subject,  caption: 'Subject')
          column colStatus   (attribute: Status,   caption: 'Status',   ColumnWidth: manual, Size: 120)
          column colPriority (attribute: Priority, caption: 'Priority', ColumnWidth: manual, Size: 100)
          column colSLADue   (attribute: SLADueAt, caption: 'SLA Due',  ColumnWidth: manual, Size: 140)
          column colActions  (caption: 'Actions', ShowContentAs: customContent, ColumnWidth: manual, Size: 80) {
            actionbutton btnOpen (
              caption:     'Open',
              action:      show_page HD.Ticket_Detail (Ticket: $currentObject),
              buttonstyle: link
            )
          }
        }
      }
    }
  }
};

-- ============================================================
-- 工单详情：2 列布局（左信息，右操作）+ 底部评论
-- ============================================================

create or replace page HD.Ticket_Detail
(
  title:  'Ticket Detail',
  layout: Atlas_Core.Atlas_Default,
  folder: 'Ticket',
  params: { $Ticket: HD.Ticket }
)
{
  layoutgrid lgMain {
    row rContent {
      column cInfo (desktopwidth: 8) {
        dataview dvTicket (datasource: $Ticket) {
          dynamictext txtSubject    (content: '{1}', contentparams: [{1} = Subject],   rendermode: H2)
          dynamictext txtStatus     (content: 'Status: {1}',   contentparams: [{1} = Status])
          dynamictext txtPriority   (content: 'Priority: {1}', contentparams: [{1} = Priority])
          dynamictext txtSLA        (content: 'SLA Due: {1}',  contentparams: [{1} = SLADueAt])
          dynamictext txtOverdue    (content: 'Overdue: {1}',  contentparams: [{1} = IsOverSLA])
          dynamictext txtDescription(content: '{1}', contentparams: [{1} = Description])
        }
      }
      column cActions (desktopwidth: 4) {
        dataview dvActions (datasource: $Ticket) {
          footer ftrActions {
            actionbutton btnSubmit (
              caption:     'Submit',
              action:      microflow HD.ACT_Ticket_Submit (Ticket: $currentObject),
              buttonstyle: primary
            )
            actionbutton btnResolve (
              caption:     'Resolve',
              action:      microflow HD.ACT_Ticket_Resolve (Ticket: $currentObject),
              buttonstyle: default
            )
            actionbutton btnReopen (
              caption:     'Reopen',
              action:      microflow HD.ACT_Ticket_Reopen (Ticket: $currentObject),
              buttonstyle: default
            )
            actionbutton btnClose (
              caption:     'Close',
              action:      microflow HD.ACT_Ticket_Close (Ticket: $currentObject),
              buttonstyle: default
            )
            actionbutton btnComment (
              caption:     'Add Comment',
              action:      show_page HD.AddComment_Form (Ticket: $currentObject),
              buttonstyle: default
            )
          }
        }
      }
    }
    row rComments {
      column cComments (desktopwidth: 12) {
        dynamictext txtCommentsTitle (content: 'Comments', rendermode: H3)
        datagrid dgComments (
          datasource: $Ticket/HD.TicketComment_Ticket/HD.TicketComment
        ) {
          column colContent    (attribute: Content,    caption: 'Comment')
          column colIsInternal (attribute: IsInternal, caption: 'Internal', ColumnWidth: manual, Size: 80)
        }
      }
    }
  }
};

-- ============================================================
-- 新建/编辑工单表单
-- ============================================================

create or replace page HD.Ticket_NewEdit
(
  title:  'New / Edit Ticket',
  layout: Atlas_Core.Atlas_Default,
  folder: 'Ticket',
  params: { $Ticket: HD.Ticket }
)
{
  layoutgrid lgMain {
    row rMain {
      column cMain (desktopwidth: 8) {
        dataview dvForm (datasource: $Ticket) {
          textbox  tbSubject     (label: 'Subject',     attribute: Subject)
          textarea taDescription (label: 'Description', attribute: Description)
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

- [ ] **Step 4：写 练习/练习01-客户列表页.mdl**

```mdl
-- ============================================================
-- 练习 01：客户列表页
-- 前提：先运行模块 01 的参考实现
-- 目标：创建 HD.Customer_Overview 页面
-- ============================================================

-- HD.Customer_Overview：
-- 要求：
--   title: 'Customers'，layout: Atlas_Core.Atlas_Default
--   DataGrid：datasource = database from HD.Customer，sort by Name asc
--   列：Name（文字过滤器）、Email、Company
--   操作列：Open 按钮（链接样式，打开 HD.Customer_Detail——暂不存在，先占位）
--   控制栏：New Customer 按钮（create_object HD.Customer then show_page HD.Customer_Detail）

-- TODO：补全页面定义
create or replace page HD.Customer_Overview
(
  title:  'Customers',
  layout: Atlas_Core.Atlas_Default,
  folder: 'Customer'
)
{
  -- 提示：用 layoutgrid → row → column (desktopwidth: 12) → datagrid
  -- 你的代码写在这里：

};

-- 验证命令：
--   mxcli check 练习01-客户列表页.mdl
--   mxcli exec  练习01-客户列表页.mdl -p MyProject.mpr
```

- [ ] **Step 5：提交**

```bash
git add academy/zh/04-页面与UI/
git commit -m "docs(academy): add module 04 — pages and UI (zh)"
```

---

## Task 9：Module 05 — 安全与权限

**Files:**
- Create: `academy/zh/05-安全与权限/业务需求.md`
- Create: `academy/zh/05-安全与权限/AI协作指南.md`
- Create: `academy/zh/05-安全与权限/参考实现/security.mdl`
- Create: `academy/zh/05-安全与权限/练习/练习01-客户角色授权.mdl`

- [ ] **Step 1：写 业务需求.md**

```markdown
# 模块 05：安全与权限 — 业务需求

## 业务背景

TechCorp 的 IT 支持系统有严格的数据隔离要求：客户只能看到自己的问题，不能看到别人的；客服可以管理所有工单；经理有最高权限。

这不只是"隐藏菜单"，而是从数据库层面就隔离——即使有人猜到了页面 URL，也看不到不属于他的数据。

---

## 用户故事

### 客户权限
- 作为客户，我只能看到自己提交的工单（系统从数据库层面过滤，不是界面隐藏）
- 作为客户，我可以创建工单和修改工单标题、描述
- 作为客户，我看不到标记为"内部"的评论（仅限客服内部查看）
- 作为客户，我无法执行"指派"或"解决"操作（只有客服才能做）

### 客服权限
- 作为客服，我可以查看和管理所有客户的工单
- 作为客服，我可以添加内部评论（其他客服可见，客户不可见）
- 作为客服，我可以指派工单和标记解决
- 作为客服，我看不到管理类的权限设置页面

### 经理权限
- 作为经理，我拥有客服的所有权限，并且可以删除工单
- 作为经理，我可以访问用户管理功能
- 作为经理，我可以初始化演示数据

---

## 验收标准

- [ ] 用 demo_customer 登录，只能看到自己的工单
- [ ] 用 demo_agent 登录，可以看到所有工单
- [ ] 用 demo_customer 登录，看不到标记 IsInternal=true 的评论
- [ ] 角色差异化首页：Customer 看我的工单，Agent 看所有工单，Manager 看……（与 Agent 相同，扩展留给进阶）
- [ ] 导航菜单按角色自动隐藏不可访问的菜单项
- [ ] 三个演示账号可以直接登录：demo_customer / demo_agent / demo_manager，密码均为 Demo12345678
```

- [ ] **Step 2：写 AI协作指南.md**

```markdown
# 模块 05：AI 协作指南 — 安全与权限

## 前提

先运行模块 01–04 的参考实现。

## 安全配置的四个层次

```
1. 模块角色（Module Role）     ← 定义角色名称
2. 实体访问规则（Entity Grant） ← 决定谁能读/写/删哪些实体
3. 微流/页面权限（Grants）      ← 决定谁能运行哪些逻辑和页面
4. 用户角色（User Role）        ← 把模块角色组合成系统用户类型
```

## 与 Claude 协作的步骤

```
帮我用 MDL 为 Helpdesk 配置三角色安全体系：
- CustomerRole：只读/写自己的工单（XPath 行级过滤），看不到 Internal 评论
- AgentRole：所有工单的完整访问
- ManagerRole：在 AgentRole 基础上增加删除权限

同时需要：
- 用户角色：Customer（含 HD.CustomerRole）, Agent（含 HD.AgentRole）, Manager（含 HD.ManagerRole）
- 演示用户：demo_customer / demo_agent / demo_manager，密码 Demo12345678
- 导航：Customer 首页→ My Tickets，Agent 首页→ All Tickets，Manager 首页→ All Tickets
```

## XPath 行级过滤说明

行级过滤用 `where '[xpath]'` 语法：

```mdl
-- 只看自己的工单：从 Ticket 找到关联的 Customer，再看 Customer 的 owner
grant HD.CustomerRole on HD.Ticket (create, read *, write *)
  where '[HD.Ticket_Customer/HD.Customer/System.owner=''[%CurrentUser%]'']';

-- 只看非内部评论
grant HD.CustomerRole on HD.TicketComment (create, read *)
  where '[IsInternal = false]';
```

注意：XPath 字符串内的单引号要**双写**（`''`）。

## 常见坑

| 坑 | 解决 |
|----|------|
| 演示用户密码不满足密码策略 | 先 `alter project security password policy (min_length: 8, require_digit: true, require_mixed_case: true)` |
| 演示用户登录时提示 "unknown user" | 检查是否先执行了 `alter project security demo users on` |
| 导航首页设置未生效 | `home page ... for Customer` 语法需要 User Role 名称完全匹配 |
```

- [ ] **Step 3：写 参考实现/security.mdl**

```mdl
-- ============================================================
-- 模块 05：安全与权限 — 参考实现
-- 前提：先运行模块 01–04 的参考实现
-- 运行：mxcli exec security.mdl -p MyProject.mpr
-- ============================================================

-- ============================================================
-- 模块角色定义
-- ============================================================

create or modify module role HD.CustomerRole;
create or modify module role HD.AgentRole;
create or modify module role HD.ManagerRole;

-- ============================================================
-- 实体访问规则
-- ============================================================

alter project security level production;

-- CustomerRole：只读/写自己的工单（行级过滤）
grant HD.CustomerRole on HD.Ticket (create,
  read (Subject, Description, Status, Priority, SLADueAt, ResolvedAt, IsOverSLA,
        HD.Ticket_Customer, HD.Ticket_Agent),
  write (Subject, Description))
  where '[HD.Ticket_Customer/HD.Customer/System.owner=''[%CurrentUser%]'']';

-- CustomerRole：只看非内部评论
grant HD.CustomerRole on HD.TicketComment (create, read *)
  where '[IsInternal = false]';

grant HD.CustomerRole on HD.Customer (read *, write *);
grant HD.CustomerRole on HD.TicketSearch (read *, write *);

-- AgentRole：所有工单完整访问
grant HD.AgentRole on HD.Ticket (create, read *, write *);
grant HD.AgentRole on HD.TicketComment (create, read *, write *);
grant HD.AgentRole on HD.Customer (read *);
grant HD.AgentRole on HD.Agent (read *);
grant HD.AgentRole on HD.TicketSearch (read *, write *);

-- ManagerRole：在 AgentRole 基础上增加删除
grant HD.ManagerRole on HD.Ticket (create, read *, write *, delete);
grant HD.ManagerRole on HD.TicketComment (create, read *, write *, delete);
grant HD.ManagerRole on HD.Customer (read *, write *);
grant HD.ManagerRole on HD.Agent (read *, write *);
grant HD.ManagerRole on HD.TicketSearch (read *, write *);

-- ============================================================
-- 微流权限
-- ============================================================

grant execute on microflow HD.ACT_Ticket_Submit  to HD.CustomerRole, HD.AgentRole, HD.ManagerRole;
grant execute on microflow HD.ACT_Ticket_Assign  to HD.AgentRole, HD.ManagerRole;
grant execute on microflow HD.ACT_Ticket_Resolve to HD.AgentRole, HD.ManagerRole;
grant execute on microflow HD.ACT_Ticket_Reopen  to HD.AgentRole, HD.ManagerRole;
grant execute on microflow HD.ACT_Ticket_Close   to HD.AgentRole, HD.ManagerRole;
grant execute on microflow HD.ACT_AddComment     to HD.CustomerRole, HD.AgentRole, HD.ManagerRole;
grant execute on microflow HD.DS_MyTickets       to HD.CustomerRole, HD.AgentRole, HD.ManagerRole;

-- 纳流权限
grant execute on nanoflow HD.NF_Ticket_QuickCreate  to HD.CustomerRole, HD.AgentRole;
grant execute on nanoflow HD.NF_TicketSearch_Apply   to HD.CustomerRole, HD.AgentRole, HD.ManagerRole;
grant execute on nanoflow HD.NF_Priority_GetLabel    to HD.CustomerRole, HD.AgentRole, HD.ManagerRole;

-- 页面权限
grant view on page HD.Ticket_Overview     to HD.CustomerRole, HD.AgentRole, HD.ManagerRole;
grant view on page HD.MyTickets_Overview  to HD.CustomerRole, HD.AgentRole, HD.ManagerRole;
grant view on page HD.Ticket_Detail       to HD.CustomerRole, HD.AgentRole, HD.ManagerRole;
grant view on page HD.Ticket_NewEdit      to HD.CustomerRole, HD.AgentRole, HD.ManagerRole;
grant view on page HD.AddComment_Form     to HD.CustomerRole, HD.AgentRole, HD.ManagerRole;

-- ============================================================
-- 用户角色
-- ============================================================

create or modify user role Customer (System.User, HD.CustomerRole);
create or modify user role Agent    (System.User, HD.AgentRole);
create or modify user role Manager  (System.User, HD.ManagerRole);

-- ============================================================
-- 演示用户
-- ============================================================

alter project security demo users on;

alter project security password policy (
  min_length:         8,
  require_digit:      true,
  require_mixed_case: true,
  require_symbol:     false
);

create or modify demo user 'demo_customer@helpdesk.test' password 'Demo12345678'
  entity Administration.Account (Customer);
create or modify demo user 'demo_agent@helpdesk.test'    password 'Demo12345678'
  entity Administration.Account (Agent);
create or modify demo user 'demo_manager@helpdesk.test'  password 'Demo12345678'
  entity Administration.Account (Manager);

-- ============================================================
-- 导航
-- ============================================================

create or replace navigation Responsive
  home page HD.MyTickets_Overview  for Customer
  home page HD.Ticket_Overview     for Agent
  home page HD.Ticket_Overview     for Manager
  home page MyFirstModule.Home_Web
  login page Administration.login
  menu (
    menu item 'My Tickets'  page HD.MyTickets_Overview;
    menu item 'All Tickets' page HD.Ticket_Overview;
  );
```

- [ ] **Step 4：写 练习/练习01-客户角色授权.mdl**

```mdl
-- ============================================================
-- 练习 01：客户角色授权
-- 前提：先运行模块 01 和 05 参考实现
-- 目标：补全 CustomerRole 的实体访问规则（含 XPath 行级过滤）
-- ============================================================

-- 已定义好模块角色（不需要修改）
create or modify module role HD.CustomerRole;

-- TODO 1：为 HD.Ticket 配置 CustomerRole 的访问规则
-- 要求：
--   操作：create, read（Subject/Description/Status/Priority/SLADueAt/ResolvedAt/IsOverSLA/关联字段), write（Subject/Description）
--   行级过滤（XPath）：只能看到自己提交的工单
--   提示：where '[HD.Ticket_Customer/HD.Customer/System.owner=''[%CurrentUser%]'']'

-- 你的代码写在这里：


-- TODO 2：为 HD.TicketComment 配置 CustomerRole 的访问规则
-- 要求：create + read，只看 IsInternal = false 的评论
-- 提示：where '[IsInternal = false]'

-- 你的代码写在这里：


-- 验证命令：
--   mxcli check 练习01-客户角色授权.mdl
--   mxcli exec  练习01-客户角色授权.mdl -p MyProject.mpr
```

- [ ] **Step 5：提交**

```bash
git add academy/zh/05-安全与权限/
git commit -m "docs(academy): add module 05 — security and permissions (zh)"
```

---

## Task 10：Capstone 业务需求文档

**Files:**
- Create: `academy/zh/capstone-helpdesk/业务需求.md`

- [ ] **Step 1：写 capstone-helpdesk/业务需求.md**

```markdown
# Capstone：完整 Helpdesk 系统 — 业务需求

> 这是你的最终目标。从第一个模块开始，你就在一点一点地构建这个系统。
> 当你完成所有模块后，你的应用应该满足这里描述的全部需求。

---

## 公司背景

TechCorp 是一家拥有 500 名员工的科技公司。IT 支持团队（8 名客服）每天处理来自全公司的技术问题：笔记本电脑、系统访问、网络连接、软件安装……

**问题：** 目前完全靠电子邮件和即时消息处理，问题经常丢失，客户不知道进度，经理无法掌握团队工作量。

**目标：** 建立一套自助式工单系统，让整个支持过程透明、可追踪、有 SLA 保障。

---

## 用户类型与他们的一天

### Alice — 市场部经理（客户）

Alice 的笔记本突然连不上公司 VPN 了，下午有重要会议。

她打开 Helpdesk，登录后看到"我的工单"页面。
- 点"新建工单"，填写标题"VPN 无法连接"，描述症状，标记优先级为"高"，提交。
- 系统告诉她：截止时间是 8 小时后。
- 2 小时后，她看到工单状态变成"处理中"，有客服留言"正在排查，请保持网络连接"。
- 问题解决后，她收到"已解决"通知，确认后关闭工单。

### Bob — IT 客服（客服）

Bob 上班第一件事：打开"所有工单"，按 SLA 截止时间排序，快速找出最紧急的工单。
- 找到一张"紧急"级别的支付问题工单，认领（指派给自己）。
- 打开工单详情，查看描述，添加一条内部备注："已联系财务部，等待确认"。
- 问题解决后，标记为"已解决"，添加外部评论说明解决方法。
- 空闲时写了一篇知识库文章"VPN 常见问题排查"，帮助客户自助解决类似问题。

### Carol — IT 主管（经理）

Carol 每天早上查看整体情况：
- 打开"所有工单"，看有多少工单超 SLA 了（IsOverSLA=true）。
- 发现一张逾期的工单，重新分配给另一位客服。
- 批准一个升级申请（Agent 申请将某工单优先级提升为"紧急"）。

---

## 功能需求清单

### 工单管理

| 功能 | 谁能操作 | 说明 |
|------|---------|------|
| 提交工单 | 客户 / 客服 | 标题必填，SLA 自动计算 |
| 查看自己的工单 | 客户 | 只看自己的，不看别人的 |
| 查看所有工单 | 客服 / 经理 | 全量，可按状态/优先级过滤 |
| 指派客服 | 客服 / 经理 | 工单状态变"处理中" |
| 解决工单 | 客服 / 经理 | 记录解决时间，计算是否逾期 |
| 重开工单 | 客服 / 经理 | 从已解决/已关闭退回待处理 |
| 关闭工单 | 客服 / 经理 | 终态，不可再修改状态 |
| 添加评论 | 所有人 | 客户只看非内部评论 |

### 知识库

| 功能 | 谁能操作 | 说明 |
|------|---------|------|
| 浏览文章 | 所有人（已登录） | 只看"已发布"状态文章 |
| 搜索文章 | 所有人 | 按标题关键词搜索 |
| 撰写文章 | 客服 / 经理 | 有草稿和已发布两种状态 |
| 发布文章 | 客服 / 经理 | 草稿 → 已发布 |

### 用户与权限

| 角色 | 登录账号 | 密码 | 登录后首页 |
|------|---------|------|-----------|
| 客户（Customer） | demo_customer@helpdesk.test | Demo12345678 | 我的工单 |
| 客服（Agent） | demo_agent@helpdesk.test | Demo12345678 | 所有工单 |
| 经理（Manager） | demo_manager@helpdesk.test | Demo12345678 | 所有工单 |

---

## 界面要求

### 视觉标准

- **工单列表**：状态列用彩色徽章区分（绿=已解决，黄=处理中，红=逾期）
- **工单详情**：2 列布局（左边展示信息，右边放操作按钮）
- **评论区**：显示在详情页底部，内部评论有明显标识
- **导航**：根据角色自动显示不同菜单项

### 演示效果验收

以下演示路径必须从头到尾跑通，无错误弹窗：

**路径 1（客户视角）：**
1. 用 demo_customer 登录
2. 创建一张"普通"优先级工单
3. 查看工单，添加一条评论
4. 退出登录

**路径 2（客服视角）：**
1. 用 demo_agent 登录
2. 打开"所有工单"，找到上面客户创建的工单
3. 指派给自己
4. 标记为"已解决"，添加解决说明评论
5. 退出登录

**路径 3（经理视角）：**
1. 用 demo_manager 登录
2. 查看所有工单
3. 退出登录

---

## 技术验收标准

```bash
# 1. 执行完整应用 MDL（模块 01-05 顺序运行，或 capstone 参考实现）
mxcli exec academy/zh/01-领域建模/参考实现/domain-model.mdl -p MyProject.mpr
mxcli exec academy/zh/02-微流业务逻辑/参考实现/microflows.mdl  -p MyProject.mpr
mxcli exec academy/zh/03-纳流与客户端/参考实现/nanoflows.mdl   -p MyProject.mpr
mxcli exec academy/zh/04-页面与UI/参考实现/pages.mdl           -p MyProject.mpr
mxcli exec academy/zh/05-安全与权限/参考实现/security.mdl      -p MyProject.mpr

# 2. Mendix 平台验证（必须 0 个 StorageLoadException）
~/.mxcli/mxbuild/*/modeler/mx check MyProject.mpr \
  2>&1 | grep -c "StorageLoadException"
# 期望输出：0
```
```

- [ ] **Step 2：提交**

```bash
git add academy/zh/capstone-helpdesk/
git commit -m "docs(academy): add capstone business requirements (zh)"
```

---

## Task 11：全量 MDL 语法检查

**Files:** 无新增文件，仅验证。

- [ ] **Step 1：对所有参考实现 MDL 运行语法检查**

```bash
for f in academy/zh/*/参考实现/*.mdl; do
  echo "Checking: $f"
  ./bin/mxcli check "$f"
done
```

Expected: 每个文件输出 "OK" 或无错误，无 "parse error" 或 "unknown type" 类型消息。

- [ ] **Step 2：对所有练习 MDL 运行语法检查**

```bash
for f in academy/zh/*/练习/*.mdl; do
  echo "Checking: $f"
  ./bin/mxcli check "$f"
done
```

Expected: 所有 TODO 填空处即使为空，也不能有语法错误（已填写的框架代码应该可以通过 check）。

- [ ] **Step 3：修复任何语法错误**

如果某个文件 check 失败，根据错误信息定位并修复，然后重跑 Step 1–2。

- [ ] **Step 4：最终提交**

```bash
git add -A
git commit -m "docs(academy): Phase 1 complete — all MDL syntax verified"
```

---

## 自检：Spec 覆盖确认

| 设计规格要求 | 对应 Task |
|------------|----------|
| 目录骨架（academy/ + zh/ + en/） | Task 1 |
| academy/README.md（双语总览） | Task 2 |
| en/README.md（占位） | Task 2 |
| zh/README.md（课程路径图） | Task 3 |
| 模块 00 完整内容 | Task 4 |
| 模块 01 完整内容（领域建模） | Task 5 |
| 模块 02 完整内容（微流） | Task 6 |
| 模块 03 完整内容（纳流） | Task 7 |
| 模块 04 完整内容（页面） | Task 8 |
| 模块 05 完整内容（安全） | Task 9 |
| Capstone 业务需求文档 | Task 10 |
| 所有 MDL mxcli check 通过 | Task 11 |
| 每模块包含 业务需求/AI协作指南/参考实现/练习 | Task 4–9 各自包含 |
| 中文版先行，英文版占位 | Task 2（en/README.md 为 "coming soon"） |
| 不依赖 helpdesk-app.mdl | ✅ 所有 MDL 从独立模块构建 |
| 页面视觉完成度（Atlas 布局、2列详情、操作按钮） | Task 8（pages.mdl） |
| 参考实现 MDL 可执行 | Task 11（全量 check） |
