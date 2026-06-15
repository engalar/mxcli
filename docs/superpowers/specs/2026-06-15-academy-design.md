# Academy — AI 辅助开发学院 设计规格

**日期：** 2026-06-15  
**状态：** 已批准，待实现  
**目标读者：** 想用 Claude Code + mxcli 做 Mendix 项目的开发者  
**语言策略：** 中文版先行；英文版目录占位，内容待后续填充

---

## 一、背景与目标

### 背景

mxcli 提供了完整的 MDL（Mendix Definition Language）脚本化开发能力，配合 Claude Code 的 skill 体系，可以实现"需求 → AI 设计 → MDL 实现 → 验证"的完整 AI 辅助开发循环。但目前缺乏系统性的学习路径，开发者只能通过阅读源码和示例摸索。

### 目标

建立 **Academy**（学院）——一个独立的学习模块，让开发者通过构建一个完整的 Helpdesk 应用，系统学习 AI 辅助 Mendix 开发的全套技能。

### 核心学习体验

```
业务需求文档           AI 协作               验证
(非技术语言)    →   Claude Code + mxcli  →   mx check
"我们需要工单系统"    /mendix:create-entity    0 错误 = 完成
                  ↓
              参考实现（标准答案）
              reference.mdl
```

---

## 二、范围与约束

### 独立性

- Academy 是独立模块，**不依赖** `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl`
- 从零开始实现同一 Helpdesk 主题，但专注于完成度和视觉美观
- `helpdesk-app.mdl` 继续作为 mxcli 回归基线，两者互不影响

### 语言策略

- **中文版**：完整实现所有内容
- **英文版**：创建目录结构和 README 占位，内容标注"Coming soon"

### 应用主题

以 **IT 支持 Helpdesk** 为统一主题贯穿所有模块：
- 模块：HD（HelpDesk）、KB（KnowledgeBase）
- 角色：Customer（客户）、Agent（客服）、Manager（经理）
- 核心业务：工单生命周期 + 知识库 + 审批工作流

---

## 三、目录结构

```
academy/
├── README.md                        ← 总览（中英双语）
│
├── en/                              ← 英文版（占位）
│   └── README.md                    ← "Content coming soon"
│
└── zh/                              ← 中文版（完整实现）
    ├── README.md                    ← 课程总览与学习路径
    ├── 00-入门准备/
    ├── 01-领域建模/
    ├── 02-微流业务逻辑/
    ├── 03-纳流与客户端/
    ├── 04-页面与UI/
    ├── 05-安全与权限/
    ├── 06-知识库模块/
    ├── 07-审批工作流/
    ├── 08-扩展-Java-Action/
    ├── 09-扩展-JS-Action/
    ├── 10-扩展-Widget开发/
    ├── 11-扩展-主题定制/
    ├── 12-AI协作模式/
    └── capstone-helpdesk/
```

### 每个模块的标准文件结构

```
zh/NN-模块名/
├── 业务需求.md          ← 非技术语言，业务场景和用户故事
├── AI协作指南.md        ← 如何与 Claude Code 配合完成本模块
├── 参考实现/
│   └── *.mdl           ← 完整可执行的标准 MDL 答案
└── 练习/
    └── 练习NN-名称.mdl  ← 填空式练习题（含 TODO 注释）
```

---

## 四、课程大纲

### 基础阶段（模块 00–05）

| 模块 | 主题 | Helpdesk 业务场景 | 核心 MDL 技能 |
|------|------|-----------------|--------------|
| 00 | 入门准备 | 环境搭建，第一次 exec | mxcli 安装、Claude Code 配置、MPR 基本概念 |
| 01 | 领域建模 | Customer / Ticket / Agent 实体 | `create entity`、属性类型、枚举、关联（1-*、*-*） |
| 02 | 微流业务逻辑 | 工单提交、指派、解决、重开 | 状态机、`if/else`、`create`/`commit`、表达式、常量 |
| 03 | 纳流与客户端 | 快速创建、搜索过滤、格式化 | 非持久化实体、`retrieve`、`loop`、返回值 |
| 04 | 页面与 UI | 完整 Helpdesk 界面 | DataGrid、DataView、表单布局、Atlas 组件、徽章 |
| 05 | 安全与权限 | 三角色访问控制 | 模块角色、XPath 行级过滤、页面/微流执行权 |

### 进阶阶段（模块 06–07）

| 模块 | 主题 | Helpdesk 业务场景 | 核心技能 |
|------|------|-----------------|---------|
| 06 | 知识库模块 | KB.Article / Category / Tag | 跨模块设计、多对多中间表、文章状态机 |
| 07 | 审批工作流 | 升级申请 → 经理审批 | Workflow 节点、User Task、边界事件、并行分支 |

### 扩展阶段（模块 08–11）

> **工具要求说明**：扩展阶段超出 mxcli 单工具范围，需要额外环境。每个模块的 `AI协作指南.md` 会列出所需工具。核心路径（00–07）完全不依赖 Studio Pro。

| 模块 | 主题 | Helpdesk 业务场景 | 额外工具 |
|------|------|-----------------|---------|
| 08 | Java Action | 密码哈希、邮件发送（模拟） | Studio Pro（创建 Action）+ JDK；参考实现提供预编译 jar，学员只需学**调用**模式 |
| 09 | JS Action | 剪贴板复制、浏览器通知、时间格式化 | 无需 Studio Pro；JS 代码在 MDL 中内联；仅需 mxcli |
| 10 | Widget 开发 | 自定义工单优先级徽章、SLA 倒计时 | Node.js + `pluggable-widgets-tools`；参考实现提供 `.mpk` 成品，学员可选择只学调用或完整开发 |
| 11 | 主题定制 | Helpdesk 品牌配色和字体 | Atlas UI source（需 Studio Pro 或独立 SCSS 工具链）；提供纯 CSS 变量方案作为无 Studio Pro 替代 |

### AI 协作与综合（模块 12 + Capstone）

| 模块 | 主题 | 内容 |
|------|------|------|
| 12 | AI 协作模式 | Prompt 设计原则、迭代调试技巧、如何从错误信息让 Claude 修复、验证策略 |
| Capstone | 完整 Helpdesk 交付 | 从业务需求到 `mx check` 零错误的完整应用 |

---

## 五、Capstone 应用规格

### 模块划分

| Mendix 模块 | 实体 | 说明 |
|-------------|------|------|
| HD | Ticket, TicketComment, Customer, Agent, EscalationRequest | 核心工单系统 |
| KB | Article, Category, Tag, ArticleTag | 知识库 |

### 工单生命周期

```
Draft ──[Submit]──► Open ──[Assign]──► InProgress ──[Resolve]──► Resolved ──[Close]──► Closed
                    ▲                                                │
                    └──────────────────[Reopen]─────────────────────┘
```

### 用户角色与权限

| 角色 | 登录后首页 | 核心操作 |
|------|-----------|---------|
| Customer | My Tickets | 提交工单、查看/评论自己的工单、浏览知识库 |
| Agent | All Tickets（过滤视图） | 处理工单、内部备注、发起升级、撰写知识库文章 |
| Manager | 升级审批队列 | 审批升级、全量工单管理、查看 SLA 报告 |

### 视觉完成度标准

| 页面类型 | 要求 |
|---------|------|
| 列表页 | 状态徽章（彩色）、优先级标记、过滤器、分页 |
| 详情页 | 2 列布局（左：信息，右：操作区）、评论时间线 |
| 表单页 | 正确绑定的下拉框、日期选择器、必填验证 |
| 导航 | 角色差异化首页、角色可见的菜单项 |
| 整体 | `mx check` 零 StorageLoadException；Atlas UI 布局规范 |

---

## 六、业务需求文档写作规范

每个模块的 `业务需求.md` 必须：

1. **只用业务语言**：不出现 `entity`、`microflow`、`attribute` 等技术词汇
2. **包含用户故事**：`作为 [角色]，我想要 [功能]，以便 [价值]`
3. **包含验收标准**：用 `- [ ]` 格式，描述"做完是什么样子"（可观察的业务行为）
4. **提供上下文**：解释为什么需要这个功能，给学员感受到真实业务压力

**示例结构（模块 01）：**

```markdown
# 客户管理 — 业务需求

## 业务背景
支持团队每天接收来自不同公司的客户请求。目前用电子表格记录客户信息，
查找效率低，经常搞混不同公司的同名客户。

## 用户故事
- 作为客服，我想按公司名快速搜索客户，以便在电话中立即找到对应记录
- 作为经理，我想查看所有客户的完整列表，以便了解我们服务的客户规模
- 作为管理员，我想添加和编辑客户信息，以便数据库保持准确

## 验收标准
- [ ] 可以用姓名或公司名搜索客户
- [ ] 打开一个客户可以看到姓名、邮箱和所属公司
- [ ] 30 秒内可以完成一个新客户的录入
- [ ] 删除客户前有确认提示
```

---

## 七、AI 协作指南写作规范

每个模块的 `AI协作指南.md` 必须：

1. **说明从哪里开始**：列出要用的 mxcli skill（如 `/mendix:create-entity`）
2. **给出推荐的提问方式**：告诉学员如何把业务需求翻译成对 Claude 的指令
3. **列出常见坑**：本模块最容易出错的地方，以及如何验证是否正确
4. **链接参考实现**：明确说明"如果卡住了，看 `参考实现/xxx.mdl`"

---

## 八、验证标准（全局）

所有参考实现的 MDL 文件必须满足：

```bash
# 语法检查（无需项目）
mxcli check 参考实现/xxx.mdl

# 执行（需要干净的 MPR 项目）
mxcli exec 参考实现/xxx.mdl -p <clean-project>.mpr

# Mendix 平台验证（无新增 StorageLoadException）
~/.mxcli/mxbuild/*/modeler/mx check <project>.mpr \
  2>&1 | grep -c "StorageLoadException" == 0
```

---

## 九、实现优先级

**Phase 1（必须完成）：**
- [ ] 目录骨架（`academy/` 根目录 + `zh/` + `en/` 占位）
- [ ] `academy/README.md`（双语总览）
- [ ] `zh/README.md`（课程路径图）
- [ ] Capstone `业务需求.md`（完整 Helpdesk 业务需求，非技术语言）
- [ ] 模块 00–05 完整内容（业务需求 + AI 指南 + 参考实现 + 练习）

**Phase 2（进阶内容）：**
- [ ] 模块 06–07（KB + 工作流）
- [ ] Capstone 参考实现（完整 MDL + 视觉完成度）

**Phase 3（扩展内容）：**
- [ ] 模块 08–11（Java/JS Action + Widget + 主题）
- [ ] 模块 12（AI 协作模式总结）
- [ ] 英文版内容填充

---

## 十、非目标

- 不依赖 `helpdesk-app.mdl`（两者独立）
- 不修改现有 `mdl-examples/` 目录下的任何文件
- 核心路径（模块 00–07 + Capstone）不要求 Studio Pro 许可证，验证用 `mx check` 即可；扩展模块（08–11）按需标注额外工具要求
- 英文版内容不在本期范围内
