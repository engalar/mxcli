# mxcli 上下文体系深度清理 — 设计文档

**日期:** 2026-05-18  
**状态:** 已批准，待实施  
**受众:** mxcli 贡献者、Claude 新会话、mxcli 分发用户

---

## 背景

经过 2026 年 4-5 月的激进重构（~1720 commits），mxcli 的上下文体系严重过时：

- `sdk/mpr/` 整目录已于 Phase 5（2026-05-17）退役，但 CLAUDE.md 仍有 12+ 处引用
- `internal/expr/` 表达式检查子系统（scan/validate/daemon/repair/report）完全未记录
- Memory 索引中约 12 条 Stage 3.x 施工日记已失效，噪音掩盖有效信号
- 分发给用户的 53 个 skills 中没有任何 `mxcli expr` 相关文档

**目标：** 一次性深度清理三个层次（CLAUDE.md、Memory、分发 skills），使上下文与当前代码状态对齐。不建立自动化持续维护机制——代码即文档。

---

## 三个受众

| 受众 | 载体 | 问题 |
|------|------|------|
| Claude 新会话 | CLAUDE.md + Memory | sdk/mpr 残留误导；expr 完全缺失 |
| 人类贡献者 | CLAUDE.md | 同上；PR checklist 规则过时 |
| mxcli 最终用户 | `.claude/skills/mendix/` (via `mxcli init`) | 缺少 expr-checker skill |

---

## 方案选择

**选定：方案 B — 先建清单、再批量执行**

先用自动扫描生成带行号的"过时引用清单"，再按清单并行执行修改。理由：可追踪、不靠记忆、修改有依据。

淘汰方案：
- 方案 A（逐层手工）：535行 CLAUDE.md + 20K 行 skills，通读成本过高
- 方案 C（三个独立 PR）：Phase 5 完成状态需要 CLAUDE.md 和 Memory 同时改，拆 PR 会造成碎片

---

## 实施计划

### Phase 1 — 审计清单（产出物驱动后续）

运行三组扫描，输出带行号的清单：

1. `grep -rn "sdk/mpr"` 覆盖 CLAUDE.md + .claude/skills/ + .claude/commands/，按文件分组
2. 列出 memory 中 `type: project` 的条目，逐一标注：已完成/进行中/仍有效
3. 读取 `internal/expr/` 目录树 + `cmd/mxcli/main.go` 的 expr 子命令定义，提炼 API surface 摘要（供 Phase 4 写 skill 使用）

**产出：** 三份清单，作为 Phase 2-4 的输入。不依赖记忆修改。

---

### Phase 2 — CLAUDE.md 手术

六个定点修改，不做无关清理：

| # | 修改点 | 操作 |
|---|--------|------|
| 1 | 架构树 `sdk/mpr/` 子节点（约 line 80-95） | 替换为退役说明注释；在 `internal/` 节点下新增 `expr/` 子树 |
| 2 | "Phase 4 read-path 待迁移"说明块（line 495-505） | 更新为 Phase 5 完成状态：`sdk/mpr` 整目录已退役，backend 现为单 `*modelsdkmpr.Reader` |
| 3 | PR checklist `sdk/mpr` 三处规则（line 236, 262-263, 269） | 改写为 backend-only 表述，不再点名 sdk/mpr |
| 4 | Useful Files 中已删文件引用（line 510-511） | 删除 `sdk/mpr/parser.go`、`sdk/mpr/writer_widgets.go` 两行 |
| 5 | Implemented 列表（约 line 450-480） | 补充 `mxcli expr` 子命令条目（scan/validate/repair/report/daemon） |
| 6 | BSON 验证指引第3条（line 173） | 删除"看 `sdk/mpr/parser_microflow.go`"，改为"看 `modelsdk/gen/` 生成类型及 reflection-data" |

**约束：** 仅修改以上六处，不重构其他内容。

---

### Phase 3 — Memory 精炼

处理流程：

```
对每条 type:project 施工日记：
  1. 判断是否含"下次遇到同类问题应注意的教训"
  2. 有 → 写成 type:feedback 条目（rule + Why: + How to apply:）
  3. 无 → 直接删除文件，从 MEMORY.md 移除指针
```

预判处理结果：

| 文件 | 处理 | 理由 |
|------|------|------|
| `project_gen_schema_gaps.md` | 提炼为 feedback | gen 类型写入前必须对照 fixture 验证，仍有效 |
| `project_d_phase_gen_test_pattern.md` | 提炼为 feedback | roundtrip 测试四件套模式，仍适用 |
| `project_stage_3_3_marathon_progress.md` | 删除（合并已有 feedback） | `feedback_implementer_batch_size.md` 已覆盖核心教训 |
| `project_stage_3_3_domainmodel_progress.md` | 删除 | 纯进度快照，代码即记录 |
| `project_pages_d5_boundary.md` | 删除 | 阶段完成，边界已固化在代码中 |
| `project_stage_3_3_5_session_progress.md` | 删除 | 纯进度快照 |
| `project_widget_roundtrip_baseline.md` | 删除 | 基线已在 mdl-examples/ 中存在 |
| `project_d1_widget_engine_cascade.md` | 删除 | 阻塞已解除（gen/customwidgets 已有 CustomWidget） |
| `project_gen_customwidgets_discovery.md` | 删除 | 发现即解决，无剩余教训 |
| `project_stage_3_2_complete.md` | 删除 | 纯完成通知 |
| `project_pr5_phase2_outcome.md` | 提炼为 feedback | bridge 文件删除决策 + Phase 4 延期原因仍有参考价值 |
| `project_phase5_complete.md` | 提炼为 feedback | sdk/mpr 退役完成状态 → 更新 project_modelsdk_migration_pattern |

**约束：** `MEMORY.md` 处理后行数应降至 35 行以下（当前 46 行）。

---

### Phase 4 — 新增 expr skill

新建 `.claude/skills/mendix/expr-checker.md`，内容从代码和 commit 提炼，不自行发明。

文件结构：

```markdown
# mxcli expr — Mendix 表达式检查器

## 用途与适用场景
## 核心子命令
  - expr scan      # 扫描 MPR 中所有表达式
  - expr validate  # 语法 + 语义校验
  - expr repair    # 自动修复可修复问题
  - expr report    # 生成 HTML/JSON/text 报告
  - expr daemon start|stop|status  # 后台检查服务
## 重要选项
  --no-daemon      # 跳过 daemon，直接运行
  --socket PATH    # 自定义 socket 路径
  --format text|json|html
## 错误码体系
  SYN-01..SYN-0x  # 语法类
  SEM-01..SEM-0x  # 语义类（属性引用、枚举值、常量、XPath 实体）
  E0xx            # 其他（数值、类型）
## 典型工作流
## 与 LSP / VS Code 的关系
```

运行 `make sync-skills` 验证嵌入到 `cmd/mxcli/skills/` 成功。

---

## 成功标准

- [ ] `grep -rn "sdk/mpr" CLAUDE.md` 输出为空（或仅剩退役说明注释）
- [ ] `internal/expr/` 在 CLAUDE.md 架构树中有完整记录
- [ ] Memory MEMORY.md 行数 ≤ 35，无 Stage 3.x 进度条目
- [ ] `.claude/skills/mendix/expr-checker.md` 存在且 `make sync-skills` 无报错
- [ ] `grep -rn "sdk/mpr/parser_microflow\|sdk/mpr/writer_widgets" CLAUDE.md` 为空

---

## 不在范围内

- 53 个现有 mendix skills 的内容审计（语法类内容相对稳定，留待下次）
- 建立自动化持续维护机制
- CONTRIBUTING.md 更新
- MDL_QUICK_REFERENCE.md 更新
