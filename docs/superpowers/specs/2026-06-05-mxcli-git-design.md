# mxcli git — Mendix Git 兼容性命令组

**日期**: 2026-06-05  
**状态**: 已批准，待实现

## 背景

Mendix Studio Pro 在每次 commit 时通过 libgit2 写入三样东西，原生 `git commit` 不会生成：

1. `.git/config` 中的 `[mendix]` section（`core.autocrlf=False`、`mendix.commits-since-gc=0`、`mendix.lineEndingResetDone=True`）
2. 每个 commit 上的 `mx_metadata` Git Notes（紧凑 JSON，无末尾换行，via `git hash-object -w --stdin`）
3. 远端的 `refs/notes/mx_metadata` ref

缺失任意一项，Studio Pro 的 Version Control 面板无法正常工作（无法识别版本控制仓库、无法提交推送）。

AI Agent（如 Claude）在开发 Mendix 项目时使用原生 `git commit`，导致每次 AI commit 都缺失 `mx_metadata`，累积后需要人工修复。

**主要参考实现**：[fix_mendix_git.py](https://github.com/engalar/ob-work-siemens/blob/master/kb/mendix/fix_mendix_git.py)

## 目标

- AI Agent 可用 `mxcli git commit` 替换 `git commit`，自动补齐 `mx_metadata`
- 每条命令输出均含"下一步怎么做"的引导，满足 AI co-design 三层要求
- 为已有受损仓库提供 `doctor` / `fix` 命令（人工或 AI 诊断修复）
- 无 Python 依赖，跨平台（Windows / Linux / macOS）

## 范围

**包含**：
- `mxcli git commit` — commit 包装器，自动写 note
- `mxcli git notes push` — 推送 notes 到远端
- `mxcli git doctor` — 5 项健康检查
- `mxcli git fix` — 批量修复已有仓库

**不包含**：
- `mxcli git push` 完整包装（用户自行 `git push && mxcli git notes push`）
- 对 git commit 的完整 flag 透传以外的业务逻辑（不改变 commit 内容）

## 命令设计

### `mxcli git commit`

**语法**：
```
mxcli git commit [flags] [--] [pathspec...]
  -m, --message string   commit 消息（透传给 git）
  -a, --all              stage 所有已跟踪文件的修改（透传给 git）
      --amend            修改上次 commit（透传给 git）
      --allow-empty      允许空 commit（透传给 git）
  -p, --project string   MPR 文件路径，用于读取 Mendix 版本号
      --version string   手动指定 Mendix 版本号（如 10.6.0.0），跳过 MPR 检测
```

所有未被 mxcli 拦截的 flags 透传给底层 `git commit`。

**执行流程**：
1. 收集所有 flags，组装 `git commit` 命令并执行
2. 若 `git commit` 失败（exit code != 0），直接返回，不写 note
3. 读取新 commit SHA：`git rev-parse HEAD`
4. 检测 Mendix 版本（见"版本检测策略"）
5. 构造紧凑 JSON note（见"Note 格式"）
6. 写入 blob：`git hash-object -w --stdin` < metadata
7. 关联 note：`git notes --ref=mx_metadata add -C <blob> <sha>`
8. 输出 git commit 的原始输出 + mxcli 提示块

**输出格式**：
```
[main a1b2c3d] Add OrderItem entity
 3 files changed, 42 insertions(+)

[mendix] mx_metadata note added to a1b2c3d (Mendix 10.6.0.0)

  Push notes when ready:
    mxcli git notes push

  Or push code + notes together:
    git push && mxcli git notes push
```

**错误场景**：
- 版本无法检测且无 `--version` flag：
  ```
  [mendix] WARNING: could not detect Mendix version. Note not written.
  
    Fix: specify project file or version:
      mxcli git commit -p app.mpr -m "..."
      mxcli git commit --version 10.6.0 -m "..."
  ```
- `git notes add` 失败（note 已存在，如 amend 场景）：先用 `-f` 覆盖再提示

### `mxcli git notes push`

**语法**：
```
mxcli git notes push [--remote origin] [--force]
```

**执行流程**：
1. 检测 remote（tracking remote → `origin` → 报错）
2. `git push <remote> refs/notes/mx_metadata`（加 `--force` 时追加 `-f`）
3. 输出推送结果

**输出示例**：
```
[mendix] notes pushed to origin/refs/notes/mx_metadata

  Studio Pro can now read version history.
  Remember to also push your commits:
    git push
```

### `mxcli git doctor`

**语法**：
```
mxcli git doctor [-p app.mpr] [--remote origin]
```

**5 项检查**（与 Python 脚本对齐）：
| # | 检查项 | 合格条件 |
|---|--------|----------|
| 1 | Git 本地配置 | `core.autocrlf=False`，`mendix.*` 存在 |
| 2 | 远端 URL 协议 | HTTPS（不是 SSH） |
| 3 | 远端 `refs/notes/mx_metadata` | `git ls-remote` 可见 |
| 4 | 本地 commits notes 完整性 | 所有 commit 有 `mx_metadata` |
| 5 | Notes JSON 格式 | 可解析为合法 JSON |

**输出格式**：
```
Diagnosing Git repo: /path/to/repo

  [✓] Git local config (core.autocrlf, mendix.*)
  [✗] Remote URL: git@github.com:... (must be HTTPS)
  [✓] Remote refs/notes/mx_metadata
  [✗] 3/12 commits missing mx_metadata
       - a1b2c3d Add entity
       - b2c3d4e Fix microflow
       - ...
  [✓] Notes JSON format

Diagnosis: 2 issues found.
Run 'mxcli git fix' to repair.
```

### `mxcli git fix`

**语法**：
```
mxcli git fix [-p app.mpr] [--version 10.6.0.0] [--remote origin]
```

**修复步骤**（顺序执行）：
1. 补齐 `.git/config` 中缺失的 mendix 配置键
2. 转换 SSH remote URL → HTTPS（自动执行，输出转换前后对比；AI 场景不需要交互确认）
3. 为缺失 note 的 commits 批量写入 `mx_metadata`（**不覆盖**已有合法 note）
4. 用 `--force` 覆盖 JSON 解析失败的损坏 note
5. 推送 notes 到远端

**输出**：
```
Fixing Git repo: /path/to/repo

Step 1: Git local config — OK (already set)
Step 2: Remote URL — converted git@github.com:... → https://github.com/...
Step 3: mx_metadata notes — fixed 3 commits, skipped 9 (already valid)
Step 4: Malformed notes — none found
Step 5: Push notes — pushed to origin

Done! Restart Mendix Studio Pro to verify.
```

## 版本检测策略

优先级（从高到低）：
1. `--version X.Y.Z` 命令行显式指定
2. `-p app.mpr` 读取 MPR 文件中的 Mendix 版本（通过 `modelsdk.Open()`）
3. 自动 glob 仓库根目录 `*.mpr`，读第一个 MPR 文件
4. 扫描现有 `mx_metadata` notes，取 `ModelerVersion`
5. 失败：报错并提示 `--version` 或 `-p`

## mx_metadata Note 格式

与 libgit2 / Studio Pro 完全一致：
- **紧凑 JSON**（`separators=(',', ':')`，无空格）
- **无末尾换行符**（通过 `git hash-object -w --stdin` 写 blob，不经 `git notes add -m`）

```json
{"BranchName":"","ModelerVersion":"10.6.0.0","ModelChanges":[],"RelatedStories":[],"SolutionVersion":"","MPRFormatVersion":"Version2","HasModelerVersion":true}
```

`MPRFormatVersion` 根据实际 MPR 文件版本填写：v1 → `"Version1"`，v2 → `"Version2"`。若无 MPR 文件，默认 `"Version2"`。

## 实现位置

| 文件 | 内容 |
|------|------|
| `cmd/mxcli/cmd_git.go` | Cobra 命令定义（`gitCmd`、`gitCommitCmd`、`gitNotesPushCmd`、`gitDoctorCmd`、`gitFixCmd`） |
| `cmd/mxcli/main.go` | 注册 `gitCmd` |

所有 git 操作通过 `os/exec` 调用 native git，**不引入 libgit2 Go bindings**。note 写入逻辑（`hashObject` + `notesAdd`）实现为包内私有函数，便于测试时 mock `execCommand`（复用 `cmd_diff_local.go` 中已有的 `execCommand` 变量模式）。

## AI Co-design 三层验证

| 层 | 满足方式 |
|----|---------|
| 初始引导 | CLAUDE.md skill 列表更新；`mxcli git --help` 列出全部子命令 |
| 发现路径 | `mxcli git commit` 输出的 hint block 直接给出下一条命令 |
| 出错反馈 | 版本检测失败、note 写入失败均输出**为什么**（原因）+**怎么修**（具体命令） |

## 测试策略

- 单元测试：mock `execCommand`，验证 note blob 格式（紧凑 JSON、无末尾换行、正确的 SHA 关联）
- 集成测试：用临时 git repo 跑完整 `doctor → fix → notes push` 流程
- 不需要 MPR 文件的测试：`--version` flag 覆盖版本检测
