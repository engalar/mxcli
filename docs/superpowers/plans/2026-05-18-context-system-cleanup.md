# mxcli 上下文体系深度清理 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 一次性清理 CLAUDE.md、Memory、分发 skills 三层上下文，使其与 Phase 5 退役后的代码状态完全对齐。

**Architecture:** Phase 1 先生成带行号的过时引用清单作为锚点；Phase 2 对 CLAUDE.md 做六处定点手术；Phase 3 从 12 条施工日记中提炼 feedback 条目后删除原件；Phase 4 新建 expr-checker skill 并验证嵌入。

**Tech Stack:** Bash/grep（审计）、直接文件编辑（CLAUDE.md + Memory）、Makefile sync-skills（skill 嵌入验证）

---

## Phase 1 — 审计清单（验证锚点）

### Task 1: 生成 sdk/mpr 残留清单

**Files:**
- Read: `CLAUDE.md`
- Read: `.claude/skills/mendix/*.md`

- [ ] **Step 1: 扫描 CLAUDE.md 中所有 sdk/mpr 引用**

```bash
grep -n "sdk/mpr" /mnt/data_sdd/gh/mxcli-wt-02/CLAUDE.md
```

预期输出（12 行，全部在下列 Phase 2 任务中处理）：

```
85:│   └── mpr/                 # Legacy MPR reader (Phase 4 read-path migration pending)
86:│       ├── reader.go        ...
87:│       ├── writer.go        ...
88:│       ├── parser.go        ...
89:│       └── utils.go         ...
90:│   # NOTE: All bridge files ...
91:│   # project_tree.go + MprBackend.reader still open sdk/mpr.Reader ...
92:│   # methods (ListEnumerations ...) ...
93:│   # have no modelsdk/mpr equivalent yet. Migrating them is Phase 4 work ...
173:3. Looking at the parser cases in `sdk/mpr/parser_microflow.go`
236:- [ ] **Test written first** — failing test exists before implementation (parser test in `sdk/mpr/`, ...
262:All executor code must go through the backend abstraction layer — the executor must never import `sdk/mpr` for write paths:
263:- [ ] **No `sdk/mpr` write imports in executor** ...
269:- [ ] **New shared types in `mdl/types/`** — types used by both `mdl/` and `sdk/mpr` ...
497:  open `sdk/mpr.Reader` because ~30 lister/getter calls ...
498:  `modelsdk/mpr.Reader` (even though ...
510:- `sdk/mpr/parser.go` ...
511:- `sdk/mpr/writer_widgets.go` ...
```

- [ ] **Step 2: 确认 skills 中无需修改的 sdk/mpr 引用**

```bash
grep -rln "sdk/mpr" /mnt/data_sdd/gh/mxcli-wt-02/.claude/skills/
```

预期：无输出（分发 skills 中不引用 sdk/mpr 内部路径）。若有输出则记录下来单独处理。

- [ ] **Step 3: 确认 internal/expr 子系统存在**

```bash
ls /mnt/data_sdd/gh/mxcli-wt-02/internal/expr/
```

预期输出包含：`daemon  meta  parse  repair  report  scan  validate`

---

## Phase 2 — CLAUDE.md 手术

### Task 2: 更新架构树 — 删除 sdk/mpr 子树，新增 internal/expr

**Files:**
- Modify: `CLAUDE.md:84-93`（sdk/mpr 子树块）
- Modify: `CLAUDE.md`（internal/ 节点，新增 expr 子树）

- [ ] **Step 1: 替换 sdk/mpr 子树块**

定位并替换以下文本块（line 84-93）：

原文：
```
│   └── mpr/                 # Legacy MPR reader (Phase 4 read-path migration pending)
│       ├── reader.go        # read-only MPR access — still primary read path
│       ├── writer.go        # read-write MPR modification (writes routed through modelsdk/mpr)
│       ├── parser.go        # BSON parsing and deserialization (model.* shape)
│       └── utils.go         # UUID generation utilities
│   # NOTE: All bridge files (sdkmpr_bridge.go, repos/sdk_bridge.go, cmd/mxcli/bson_reader_bridge.go)
│   # have been deleted; consumers call modelsdk/mpr directly where possible.
│   # project_tree.go + MprBackend.reader still open sdk/mpr.Reader because ~30 lister/getter
│   # methods (ListEnumerations, GetNavigation, GetProjectSettings, ListBusinessEventServices …)
│   # have no modelsdk/mpr equivalent yet. Migrating them is Phase 4 work — see
│   # memory `project_modelsdk_migration_pattern` and `Not Yet Implemented` below.
```

替换为：
```
│   # sdk/mpr/ retired in Phase 5 (2026-05-17) — all BSON parsing moved to mdl/backend/mpr/*_compat.go
│   # backend now uses a single *modelsdkmpr.Reader; sdk/ only contains versions/
```

- [ ] **Step 2: 在 internal/ 节点下新增 expr 子树**

在 CLAUDE.md 的 `├── internal/` 节点下（当前只有 `└── codegen/`），替换为：

原文：
```
├── internal/                # Internal packages (not exported)
│   └── codegen/             # Metamodel code generation system
│       ├── schema/          # json reflection data loading
│       ├── transform/       # transform to Go types
│       └── emit/            # Go source code generation
```

替换为：
```
├── internal/                # Internal packages (not exported)
│   ├── codegen/             # Metamodel code generation system
│   │   ├── schema/          # json reflection data loading
│   │   ├── transform/       # transform to Go types
│   │   └── emit/            # Go source code generation
│   └── expr/                # Mendix expression checker subsystem
│       ├── scan/            # BSON walker — extracts expression strings from .mxunit files
│       ├── parse/           # exprcheck parser wrapper — tokenises + detects syntax errors
│       ├── validate/        # SYN-01/02/03 + SEM-04/05/07 validation rules
│       ├── repair/          # ranked repair suggestions for fixable issues
│       ├── report/          # HTML/JSON/text report generator (full pipeline)
│       ├── meta/            # Index interface + CatalogReader (entity attrs, enums, constants)
│       └── daemon/          # Background daemon — JIT index, socket protocol, idle watcher
```

- [ ] **Step 3: 验证**

```bash
grep -n "sdk/mpr" /mnt/data_sdd/gh/mxcli-wt-02/CLAUDE.md | grep -v "# sdk/mpr/ retired\|modelsdk/mpr\|backend/mpr"
```

预期：无输出（架构树中不再有功能性 sdk/mpr 引用）。

```bash
grep -n "internal/expr\|expr/.*daemon\|expr/.*validate" /mnt/data_sdd/gh/mxcli-wt-02/CLAUDE.md
```

预期：看到 `internal/expr/` 子树各行。

- [ ] **Step 4: Commit**

```bash
git -C /mnt/data_sdd/gh/mxcli-wt-02 add CLAUDE.md
git -C /mnt/data_sdd/gh/mxcli-wt-02 commit -m "docs(CLAUDE.md): retire sdk/mpr from arch tree, add internal/expr subtree"
```

---

### Task 3: 更新 BSON 存储名验证指引（line 173）

**Files:**
- Modify: `CLAUDE.md:173`

- [ ] **Step 1: 替换第3条验证方法**

原文（line 173）：
```
3. Looking at the parser cases in `sdk/mpr/parser_microflow.go`
```

替换为：
```
3. Checking the generated types in `modelsdk/gen/` and reflection data in `reference/mendixmodellib/reflection-data/`
```

- [ ] **Step 2: 验证**

```bash
grep -n "parser_microflow\|sdk/mpr/parser" /mnt/data_sdd/gh/mxcli-wt-02/CLAUDE.md
```

预期：无输出。

- [ ] **Step 3: Commit**

```bash
git -C /mnt/data_sdd/gh/mxcli-wt-02 add CLAUDE.md
git -C /mnt/data_sdd/gh/mxcli-wt-02 commit -m "docs(CLAUDE.md): update BSON storage name verification — point to modelsdk/gen instead of retired sdk/mpr"
```

---

### Task 4: 更新 PR checklist — 删除 sdk/mpr 具名引用

**Files:**
- Modify: `CLAUDE.md:236,262-263,269`

- [ ] **Step 1: 修改 Bug fixes checklist（line 236）**

原文：
```
- [ ] **Test written first** — failing test exists before implementation (parser test in `sdk/mpr/`, backend mutation test in `mdl/backend/mpr/`, executor handler test in `mdl/executor/` using `MockBackend`)
```

替换为：
```
- [ ] **Test written first** — failing test exists before implementation (backend mutation test in `mdl/backend/mpr/`, executor handler test in `mdl/executor/` using `MockBackend`)
```

- [ ] **Step 2: 修改 Backend abstraction compliance 段落标题（line 262）**

原文：
```
All executor code must go through the backend abstraction layer — the executor must never import `sdk/mpr` for write paths:
- [ ] **No `sdk/mpr` write imports in executor** — executor files must not call `sdk/mpr` writer/parser types directly; use `ctx.Backend.*` instead
```

替换为：
```
All executor code must go through the backend abstraction layer — the executor must never import internal BSON packages directly:
- [ ] **No direct BSON imports in executor** — executor files must not call modelsdk/codec or backend-internal types directly; use `ctx.Backend.*` instead
```

- [ ] **Step 3: 修改 shared types 规则（line 269）**

原文：
```
- [ ] **New shared types in `mdl/types/`** — types used by both `mdl/` and `sdk/mpr` go in `mdl/types/`; `sdk/mpr` re-exports as type aliases (`type Foo = types.Foo`), never as duplicate definitions
```

替换为：
```
- [ ] **New shared types in `mdl/types/`** — types shared between executor and backend go in `mdl/types/`; backend re-exports as type aliases (`type Foo = types.Foo`), never as duplicate definitions
```

- [ ] **Step 4: 验证**

```bash
grep -n "sdk/mpr" /mnt/data_sdd/gh/mxcli-wt-02/CLAUDE.md
```

预期：只剩退役注释那一行（`# sdk/mpr/ retired in Phase 5`）。

- [ ] **Step 5: Commit**

```bash
git -C /mnt/data_sdd/gh/mxcli-wt-02 add CLAUDE.md
git -C /mnt/data_sdd/gh/mxcli-wt-02 commit -m "docs(CLAUDE.md): remove sdk/mpr references from PR checklist"
```

---

### Task 5: 更新 Not Yet Implemented 和 Useful Files

**Files:**
- Modify: `CLAUDE.md:492-505`（Not Yet Implemented 的 Phase 4 块）
- Modify: `CLAUDE.md:510-511`（Useful Files 中的两行）

- [ ] **Step 1: 替换 Phase 4 read-path migration 块**

原文（line 492-505）：
```
- **Phase 4 read-path migration** — `MprBackend.reader` and `cmd/mxcli/project_tree.go` still
  open `sdk/mpr.Reader` because ~30 lister/getter calls have no equivalent **method** on
  `modelsdk/mpr.Reader` (even though `modelsdk/mprread/` already exposes same-named free
  functions like `mprread.ListEnumerations(r *mmpr.Reader)`). Switching requires writing a
  full gen→model converter for each domain (downstream executor code consumes deep fields,
  not just ID+Name) across ~15 domains: Enumeration, Constant, Navigation, ProjectSettings,
  ImportMapping, ExportMapping, JsonStructure, BusinessEventService, OData/REST services,
  ScheduledEvent, ImageCollection, AgentEditor types, etc. Write-path migration is 77%
  complete (see memory `project_modelsdk_migration_pattern`); read-path is a separate spec.
```

替换为：
```
- **Phase 4 read-path migration** — `MprBackend` and `cmd/mxcli/project_tree.go` still call
  ~30 lister/getter functions from `modelsdk/mprread/` that lack a gen→model deep-field
  converter. Switching requires writing a full converter per domain (executor code consumes
  deep fields, not just ID+Name) across ~15 domains: Enumeration, Constant, Navigation,
  ProjectSettings, ImportMapping, ExportMapping, JsonStructure, BusinessEventService,
  OData/REST services, ScheduledEvent, ImageCollection, AgentEditor types, etc.
  Phase 5 (sdk/mpr retirement) is complete; Phase 4 is a separate spec.
```

- [ ] **Step 2: 删除 Useful Files 中已不存在的文件引用（line 510-511）**

原文：
```
- `sdk/mpr/parser.go` - BSON parsing logic (complex, handles polymorphic types)
- `sdk/mpr/writer_widgets.go` - Widget BSON serialization
```

替换为（删除这两行，用对应的 backend 路径代替）：
```
- `mdl/backend/mpr/` - BSON parsing and mutation logic (replaced sdk/mpr post Phase 5)
```

- [ ] **Step 3: 验证**

```bash
grep -n "sdk/mpr/parser\.go\|sdk/mpr/writer_widgets" /mnt/data_sdd/gh/mxcli-wt-02/CLAUDE.md
```

预期：无输出。

```bash
grep -n "Phase 5\|sdk/mpr retirement\|Phase 4 read-path" /mnt/data_sdd/gh/mxcli-wt-02/CLAUDE.md
```

预期：看到更新后的 Phase 4 段落（Phase 5 已完成）。

- [ ] **Step 4: Commit**

```bash
git -C /mnt/data_sdd/gh/mxcli-wt-02 add CLAUDE.md
git -C /mnt/data_sdd/gh/mxcli-wt-02 commit -m "docs(CLAUDE.md): update Phase 4/5 status, remove deleted sdk/mpr file refs"
```

---

### Task 6: 补充 mxcli expr 到 Implemented 列表

**Files:**
- Modify: `CLAUDE.md`（Implemented 列表末尾，在 Marketplace browsing 行之后）

- [ ] **Step 1: 在 Implemented 列表末尾追加 expr 条目**

在以下行之后：
```
- Marketplace browsing (`mxcli marketplace search/info/versions`) with --min-mendix compatibility filtering; install blocked upstream (API does not expose download URLs)
```

追加：
```
- Expression checker (`mxcli expr`) — scan/parse/validate/repair/report pipeline for Mendix expressions in MPR files; SYN-01/02/03 syntax rules + SEM-04/05/07 semantic rules (enum values, constants, entity/association paths); background daemon with JIT index for fast repeated validation; `--no-daemon` flag for CI/syntax-only mode
```

- [ ] **Step 2: 验证**

```bash
grep -n "mxcli expr\|Expression checker\|SYN-01\|SEM-04" /mnt/data_sdd/gh/mxcli-wt-02/CLAUDE.md
```

预期：看到新增的 expr 条目。

- [ ] **Step 3: 最终验证 — sdk/mpr 残留清零**

```bash
grep -n "sdk/mpr" /mnt/data_sdd/gh/mxcli-wt-02/CLAUDE.md
```

预期：只有一行退役注释（`# sdk/mpr/ retired in Phase 5`），无其他功能性引用。

- [ ] **Step 4: Commit**

```bash
git -C /mnt/data_sdd/gh/mxcli-wt-02 add CLAUDE.md
git -C /mnt/data_sdd/gh/mxcli-wt-02 commit -m "docs(CLAUDE.md): add mxcli expr to Implemented list"
```

---

## Phase 3 — Memory 精炼

### Task 7: 提炼 gen-schema-gaps → feedback 条目

**Files:**
- Modify: `/home/claude_dev/.claude/projects/-mnt-data-sdd-gh-mxcli/memory/project_gen_schema_gaps.md`（内容替换为 feedback 类型）
- Modify: `/home/claude_dev/.claude/projects/-mnt-data-sdd-gh-mxcli/memory/MEMORY.md`（更新指针描述）

- [ ] **Step 1: 将 project_gen_schema_gaps.md 重写为 feedback 类型**

用以下内容完整替换该文件：

```markdown
---
name: gen-schema-gaps
description: modelsdk/gen 类型写入前必须 grep init* 函数对照真实 fixture BSON — typed getter 名与真实 BSON key 常不一致
metadata:
  type: feedback
---

写 gen 类型 BSON helper 前，必须先 grep 对应 gen 类型的 `init*` 函数，对比真实 fixture MPR 中的 BSON key，再写代码。

**Why:** gen 类型 method 名对应 modern 别名（如 `OutputVariableName`），与真实 fixture BSON key（如 `_OutputVariableName`）不一致（30+ 已记录实例）。直接按 method 名写 key 会导致 BSON 序列化静默失败，Studio Pro 读取时字段为空。Read 侧需 typed-getter-first / raw-BSON-fallback 两段读取。

**How to apply:** 每次为 gen 类型写新的 BSON setter/getter helper 时：1) `grep "func init" modelsdk/gen/<domain>/<type>.go` 找 init 函数；2) 对比 fixture MPR 中实际 BSON key（用 `bsondump` 或 SQLite browser 查 Units 表）；3) 如不一致，用 `element.Base.AddProperty(rawKey, val)` 写入，不要用 typed setter。
```

- [ ] **Step 2: 更新 MEMORY.md 中该条目的描述**

在 MEMORY.md 中找到：
```
- [gen schema gaps](project_gen_schema_gaps.md) — gen 类型 BSON key alias gap (typed-first/raw-fallback) + setter gap (setRawBSONField)，写 helper 前必先对照 fixture 验证
```

替换为：
```
- [gen schema gaps](project_gen_schema_gaps.md) — gen 类型写入前必须对照 fixture BSON 验证 key 名，typed getter 与真实 key 常不一致
```

- [ ] **Step 3: 验证**

```bash
grep "type:" /home/claude_dev/.claude/projects/-mnt-data-sdd-gh-mxcli/memory/project_gen_schema_gaps.md
```

预期：`  type: feedback`

---

### Task 8: 提炼 d-phase-gen-test-pattern → feedback 条目

**Files:**
- Modify: `/home/claude_dev/.claude/projects/-mnt-data-sdd-gh-mxcli/memory/project_d_phase_gen_test_pattern.md`
- Modify: `/home/claude_dev/.claude/projects/-mnt-data-sdd-gh-mxcli/memory/MEMORY.md`

- [ ] **Step 1: 重写为 feedback 类型**

```markdown
---
name: d-phase-gen-test-pattern
description: gen 类型新增写路径方法时，BSON 测试用 shape+round-trip-stable 模式，不做字节级对比
metadata:
  type: feedback
---

为 gen 类型新增 `*Gen` sibling 写路径方法时，不要写 `bytes.Equal(legacyBSON, genBSON)` 的全字节对比测试。

**Why:** Legacy 手写序列化路径会发出 backward-decode default 字段，gen 路径（codec.Encoder）只发出实际设置的属性。两条路径字节注定不同，但语义等价（Studio Pro 都能读）。字节对比测试会永远失败，误导开发者以为写路径有 bug。

**How to apply:** 用 D7 测试模式（在 Stage 3.3.3 的 `workflow_mutator_gen_test.go` 和 Stage 3.3.5 的 `page_mutator_gen_test.go` 中已验证）：1) shape test — 断言 BSON 中关键字段存在且值正确；2) round-trip-stable test — 写入再读取，断言读取值等于写入值；3) roundtrip baseline — 用已知好的 fixture 跑完整 mxcli exec + mx check 流程验证 Studio Pro 兼容性。
```

- [ ] **Step 2: 更新 MEMORY.md**

找到：
```
- [D-phase gen test pattern](project_d_phase_gen_test_pattern.md) — D7 模式：shape + tree-stitch + round-trip-stable + roundtrip baseline，不强求 byte-identity vs legacy
```

替换为：
```
- [gen test pattern](project_d_phase_gen_test_pattern.md) — gen 写路径方法用 shape+round-trip-stable 测试，不做字节对比（legacy 路径发出额外默认字段）
```

- [ ] **Step 3: 验证**

```bash
grep "type:" /home/claude_dev/.claude/projects/-mnt-data-sdd-gh-mxcli/memory/project_d_phase_gen_test_pattern.md
```

预期：`  type: feedback`

---

### Task 9: 提炼 project-pr5-phase2-outcome → feedback 条目

**Files:**
- Modify: `/home/claude_dev/.claude/projects/-mnt-data-sdd-gh-mxcli/memory/project_pr5_phase2_outcome.md`
- Modify: `/home/claude_dev/.claude/projects/-mnt-data-sdd-gh-mxcli/memory/MEMORY.md`

- [ ] **Step 1: 重写为 feedback 类型**

```markdown
---
name: phase4-read-path-scope
description: Phase 4 read-path migration 起点是 mprread 自由函数，需为 ~15 个域写 gen→model deep-field converter，不是薄 ID+Name wrapper
metadata:
  type: feedback
---

开 Phase 4 read-path migration spec 时，起点是 `modelsdk/mprread/reader_documents.go` 的 39 个自由函数，需要为每个域写完整 gen→model converter 后再迁移 `backend.go` + `project_tree.go` 调用点。

**Why:** PR5 Task 1 原计划"切换 sdkReader alias 到 modelsdk/mpr"被发现不可行——modelsdk/mpr.Reader 只有 43 个方法，backend 需要 125 个；下游 executor 消费深度字段（非薄 ID+Name wrapper 可行），需要 ~15 个域的完整 gen→model converter（约 1500-2500 行）。这是独立 Phase 4 工程，不是 bridge 替换。

**How to apply:** 不要尝试"直接把 backend.reader 换成 msdkReader"——字段访问会编译失败。正确路径：先写 `gen→model.Enumeration` converter，再迁移 `backend.ListEnumerations`，逐域推进。参见 [[project_modelsdk_migration_pattern]]。
```

- [ ] **Step 2: 更新 MEMORY.md**

找到：
```
- [PR5 Phase 2 outcome](project_pr5_phase2_outcome.md) — 3 个 bridge 文件已删；sdk/mpr 读路径迁移延期 Phase 4（需 gen→model deep-field converter，非薄 ID+Name wrapper 可行）
```

替换为：
```
- [Phase 4 read-path scope](project_pr5_phase2_outcome.md) — Phase 4 需为 ~15 域写完整 gen→model converter；不是 bridge 替换；起点是 mprread 39 个自由函数
```

- [ ] **Step 3: 验证**

```bash
grep "type:" /home/claude_dev/.claude/projects/-mnt-data-sdd-gh-mxcli/memory/project_pr5_phase2_outcome.md
```

预期：`  type: feedback`

---

### Task 10: 删除 9 条纯进度快照 memory 条目

**Files:**
- Delete: `project_stage_3_3_marathon_progress.md`
- Delete: `project_stage_3_3_domainmodel_progress.md`
- Delete: `project_pages_d5_boundary.md`
- Delete: `project_stage_3_3_5_session_progress.md`
- Delete: `project_widget_roundtrip_baseline.md`
- Delete: `project_d1_widget_engine_cascade.md`
- Delete: `project_gen_customwidgets_discovery.md`
- Delete: `project_stage_3_2_complete.md`
- Delete: `project_phase5_complete.md`
- Modify: `MEMORY.md`（移除对应指针行）

- [ ] **Step 1: 删除 9 个文件**

```bash
cd /home/claude_dev/.claude/projects/-mnt-data-sdd-gh-mxcli/memory/ && \
rm project_stage_3_3_marathon_progress.md \
   project_stage_3_3_domainmodel_progress.md \
   project_pages_d5_boundary.md \
   project_stage_3_3_5_session_progress.md \
   project_widget_roundtrip_baseline.md \
   project_d1_widget_engine_cascade.md \
   project_gen_customwidgets_discovery.md \
   project_stage_3_2_complete.md \
   project_phase5_complete.md
```

- [ ] **Step 2: 从 MEMORY.md 移除对应的 9 行指针**

在 MEMORY.md 中删除以下各行：
```
- [Stage 3.3 marathon progress](project_stage_3_3_marathon_progress.md) — ...
- [Stage 3.3.4 domainmodel progress](project_stage_3_3_domainmodel_progress.md) — ...
- [Pages D5 boundary](project_pages_d5_boundary.md) — ...
- [Stage 3.3.5 session progress](project_stage_3_3_5_session_progress.md) — ...
- [Widget roundtrip baseline](project_widget_roundtrip_baseline.md) — ...
- [D1 widget_engine cascade](project_d1_widget_engine_cascade.md) — ...
- [Gen customwidgets discovery](project_gen_customwidgets_discovery.md) — ...
- [Stage 3.2 complete](project_stage_3_2_complete.md) — ...
- [Phase 5 complete](project_phase5_complete.md) — ...
```

- [ ] **Step 3: 验证 MEMORY.md 行数**

```bash
wc -l /home/claude_dev/.claude/projects/-mnt-data-sdd-gh-mxcli/memory/MEMORY.md
```

预期：≤ 37 行（当前 46 行，删 9 行）。

```bash
ls /home/claude_dev/.claude/projects/-mnt-data-sdd-gh-mxcli/memory/project_stage_3_3*.md 2>/dev/null
```

预期：无输出（文件已删除）。

---

## Phase 4 — 新增 expr-checker skill

### Task 11: 创建 .claude/skills/mendix/expr-checker.md

**Files:**
- Create: `/mnt/data_sdd/gh/mxcli-wt-02/.claude/skills/mendix/expr-checker.md`

- [ ] **Step 1: 创建 skill 文件**

内容如下：

```markdown
# mxcli expr — Mendix 表达式检查器

`mxcli expr` 是一条完整的表达式检查流水线，可扫描 MPR 中所有表达式字符串，发现语法和语义错误，并提供修复建议。

## 核心子命令

```bash
# 扫描 mprcontents/ 中所有表达式，输出 JSONL
mxcli expr scan <mprcontents>...

# 解析收集到的表达式（检测 token 级错误）
mxcli expr parse <mprcontents>...

# 应用 SYN + SEM 验证规则（推荐：与 -p 一起用）
mxcli expr validate -p app.mpr

# 为可修复问题生成修复建议
mxcli expr repair <mprcontents>...

# 完整流水线：scan → parse → validate → 生成报告
mxcli expr report -p app.mpr --format html -o report.html

# 管理后台 daemon
mxcli expr daemon start   -p app.mpr
mxcli expr daemon status
mxcli expr daemon stop    -p app.mpr
```

## 重要选项

| 选项 | 适用命令 | 说明 |
|------|----------|------|
| `--no-daemon` | `validate` | 跳过 daemon，仅做语法校验（适合 CI） |
| `--socket PATH` | `daemon start` | 自定义 Unix socket 路径 |
| `--format json\|html\|text` | `report`, `scan`, `validate` | 输出格式（默认 json） |
| `--filter <substring>` | `validate`, `report` | 按 unit_type 过滤（如 `Microflow`） |
| `--severity ERROR\|WARNING\|INFO` | `validate`, `report` | 按严重程度过滤 |
| `--summary` | `scan` | 输出人类可读统计而非 JSONL |

## 错误码体系

### SYN — 语法规则

| 码 | 含义 | 严重程度 |
|----|------|----------|
| `SYN-01` | 表达式解析失败（token 级错误） | ERROR |
| `SYN-02` | 字段存储了 URL 而非表达式 | INFO |
| `SYN-03` | if-then 缺少 else 分支（启发式） | WARNING |

### SEM — 语义规则（需要 -p 和 daemon）

| 码 | 含义 | 严重程度 |
|----|------|----------|
| `SEM-04` | 枚举值引用不存在（如 `Status.Active` 但枚举无此值） | ERROR |
| `SEM-05` | 常量引用不存在（如 `MyModule.CONST_X`） | ERROR |
| `SEM-07` | 实体属性或关联路径不存在（如 `$Var/Module.Entity/UnknownAttr`） | ERROR |

## 典型工作流

### 快速语法扫描（无需打开项目）

```bash
mxcli expr validate -p app.mpr --no-daemon --format text
```

### 完整语义检查（需要 MPR）

```bash
# daemon 会自动启动并缓存 index
mxcli expr validate -p app.mpr --format json | jq '.[] | select(.Severity=="ERROR")'
```

### CI 集成

```bash
# 仅语法检查，不启动 daemon，非零退出码表示有 ERROR
mxcli expr validate -p app.mpr --no-daemon --severity ERROR --format json
echo "Exit: $?"
```

### 生成 HTML 报告

```bash
mxcli expr report -p app.mpr --format html -o expr-report.html
open expr-report.html
```

## Daemon 工作原理

`mxcli expr validate`（不带 `--no-daemon`）会自动启动一个后台 daemon，daemon：
- 为 MPR 建立 JIT 语义索引（实体属性、枚举值、常量）
- 通过 Unix socket 提供校验服务（socket 路径从 MPR 路径派生，默认 `/tmp/mxexpr-*.sock`）
- 空闲超时后自动退出

手动管理：`mxcli expr daemon start|status|stop -p app.mpr`

## 与 LSP / VS Code 的关系

`mxcli expr` 是独立的批量检查工具，与 LSP 的实时表达式诊断是不同路径。LSP 诊断在编辑器里逐表达式触发；`mxcli expr` 适合全项目扫描和 CI 场景。
```

- [ ] **Step 2: 验证文件存在且内容正确**

```bash
wc -l /mnt/data_sdd/gh/mxcli-wt-02/.claude/skills/mendix/expr-checker.md
```

预期：≥ 80 行。

```bash
grep "SYN-01\|SEM-04\|daemon\|--no-daemon" /mnt/data_sdd/gh/mxcli-wt-02/.claude/skills/mendix/expr-checker.md
```

预期：4 行匹配。

---

### Task 12: 运行 sync-skills 并提交

**Files:**
- Auto-sync: `cmd/mxcli/skills/expr-checker.md`（由 Makefile 生成）

- [ ] **Step 1: 运行 sync-skills**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02 && make sync-skills
```

预期：输出包含 `expr-checker.md` 已同步的确认信息，无报错。

- [ ] **Step 2: 验证 skill 已嵌入**

```bash
ls /mnt/data_sdd/gh/mxcli-wt-02/cmd/mxcli/skills/expr-checker.md
```

预期：文件存在。

- [ ] **Step 3: Commit**

```bash
git -C /mnt/data_sdd/gh/mxcli-wt-02 add .claude/skills/mendix/expr-checker.md cmd/mxcli/skills/expr-checker.md
git -C /mnt/data_sdd/gh/mxcli-wt-02 commit -m "feat(skills): add expr-checker skill — mxcli expr subcommand documentation"
```

---

## 最终验收检查

### Task 13: 运行成功标准验证

- [ ] **Check 1: CLAUDE.md 中无功能性 sdk/mpr 引用**

```bash
grep -n "sdk/mpr" /mnt/data_sdd/gh/mxcli-wt-02/CLAUDE.md | grep -v "# sdk/mpr/ retired"
```

预期：无输出

- [ ] **Check 2: internal/expr 在架构树中有记录**

```bash
grep -c "internal/expr\|expr/.*daemon\|expr/.*validate\|expr/.*scan" /mnt/data_sdd/gh/mxcli-wt-02/CLAUDE.md
```

预期：≥ 6

- [ ] **Check 3: Memory 索引行数**

```bash
wc -l /home/claude_dev/.claude/projects/-mnt-data-sdd-gh-mxcli/memory/MEMORY.md
```

预期：≤ 37 行

- [ ] **Check 4: 9 个已删除文件确认不存在**

```bash
ls /home/claude_dev/.claude/projects/-mnt-data-sdd-gh-mxcli/memory/project_stage_3_3*.md \
   /home/claude_dev/.claude/projects/-mnt-data-sdd-gh-mxcli/memory/project_phase5_complete.md \
   /home/claude_dev/.claude/projects/-mnt-data-sdd-gh-mxcli/memory/project_widget_roundtrip_baseline.md \
   /home/claude_dev/.claude/projects/-mnt-data-sdd-gh-mxcli/memory/project_d1_widget_engine_cascade.md \
   /home/claude_dev/.claude/projects/-mnt-data-sdd-gh-mxcli/memory/project_gen_customwidgets_discovery.md \
   /home/claude_dev/.claude/projects/-mnt-data-sdd-gh-mxcli/memory/project_pages_d5_boundary.md \
   2>/dev/null
```

预期：无输出（所有文件已删除）

- [ ] **Check 5: expr-checker skill 存在**

```bash
ls /mnt/data_sdd/gh/mxcli-wt-02/.claude/skills/mendix/expr-checker.md /mnt/data_sdd/gh/mxcli-wt-02/cmd/mxcli/skills/expr-checker.md
```

预期：两个文件都存在

- [ ] **Check 6: CLAUDE.md 中有 mxcli expr 条目**

```bash
grep "mxcli expr\|Expression checker" /mnt/data_sdd/gh/mxcli-wt-02/CLAUDE.md
```

预期：看到 Implemented 列表中的 expr 条目
