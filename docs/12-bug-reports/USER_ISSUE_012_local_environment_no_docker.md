# Issue 012: 本地环境（无 Docker）支持不足

## 元数据

| 字段 | 值 |
|------|-----|
| Reporter | Miwa |
| 分类 | MxCli |
| 状态 | Open（文档缺口） |
| 优先级 | 低 |
| 发现日期 | 2026-06-02 |

## 问题描述

mxcli 的 skills（`run-app.md`、`test-app.md`）和工具（`playwright-cli`）均假设 Docker/Devcontainer 环境，企业内限制 Docker 的用户无法使用，必须独立摸索替代方案。

## 受影响的工作流

### 1. 应用启动

| 环境 | 工具 | 状态 |
|------|------|------|
| Docker | `mxcli docker run` | ✅ 支持 |
| 本地（无 Docker） | Deploy for Eclipse (`runtimelauncher.jar`) | ❌ 未文档化 |

### 2. 测试执行

| 环境 | 工具 | 状态 |
|------|------|------|
| Docker/Devcontainer | `playwright-cli` | ✅ 内置 |
| 本地 | npm Playwright | ❌ 未文档化 |

### 3. 模型变更后重新部署

mxcli exec 后需要：停止 runtime → mxbuild 重新编译 → 重启 runtime，但此流程无文档说明。

## 期望行为

Skills 和文档中提供"无 Docker 路径"，包括：

1. **启动**：`runtimelauncher.jar` 使用方式或 `mxcli run --local` 命令
2. **测试**：基于 npm Playwright 的等效测试流程
3. **重新部署**：mxbuild + restart 的标准化命令序列

## 可能的实现

```bash
# 建议的命令（目前不存在）
mxcli run --local -p app.mpr         # 使用 runtimelauncher.jar 启动
mxcli redeploy --local -p app.mpr    # 停止 → mxbuild → 重启
```

## 文档修改建议

在 `.claude/skills/run-app.md` 和 `.claude/skills/test-app.md` 中加入"无 Docker 环境"章节，说明 Deploy for Eclipse 路径和 npm Playwright 使用方式。

## 关联文件

- `.claude/skills/run-app.md` — 需补充本地启动路径
- `.claude/skills/test-app.md` — 需补充本地测试路径
- `cmd/mxcli/docker/` — docker 相关命令（参考实现）
