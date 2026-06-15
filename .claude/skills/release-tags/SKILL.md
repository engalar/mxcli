---
name: release-tags
description: Use when creating release tags for mxcli components (launcher, daemon, local) on the current branch
---

# mxcli Release Tags

## 组件 Tag 规则

| 组件 | Tag 格式 | 版本独立性 |
|------|---------|-----------|
| launcher (`cmd/mxcli-launcher/`) | `v*` | 与 daemon 同步 |
| daemon (`cmd/mxcli/`) | `daemon-v*` | 与 launcher 同步 |
| local (`cmd/mxcli-local/`) | `local-v*` | 独立版本号 |

**launcher 和 daemon 版本号始终保持一致**（如 v0.24.0 对应 daemon-v0.24.0）。

## 流程

### 1. 查看当前最新版本

```bash
git tag --sort=-version:refname | grep -E '^v[0-9]' | head -3
git tag --sort=-version:refname | grep '^daemon-v' | head -3
git tag --sort=-version:refname | grep '^local-v' | head -3
```

### 2. 确定下一版本号

- launcher + daemon：patch 号 +1（如 v0.23.0 → v0.24.0）
- local：patch 号 +1（如 local-v0.5.0 → local-v0.6.0）
- 含 breaking change 时升 minor，跨越里程碑时升 major

### 3. 创建 3 个 tag

```bash
git tag v0.24.0
git tag daemon-v0.24.0
git tag local-v0.6.0
```

### 4. Push 触发 CI

```bash
git push origin v0.24.0 daemon-v0.24.0 local-v0.6.0
```

## 示例（v0.23.0 → v0.24.0）

```bash
# 查询现状
git tag --sort=-version:refname | grep -E '^(v[0-9]|daemon-v|local-v)' | head -3

# 创建
git tag v0.24.0 && git tag daemon-v0.24.0 && git tag local-v0.6.0

# 验证
git tag --sort=-version:refname | grep -E '^(v[0-9]|daemon-v|local-v)' | head -3

# Push
git push origin v0.24.0 daemon-v0.24.0 local-v0.6.0
```

## 注意

- Tag 打在哪个 commit 上，CI 就从那个 commit 构建——确认当前 HEAD 是要发布的状态
- 误打的 tag 可删除：`git tag -d <tag> && git push origin :refs/tags/<tag>`
- 不需要提前合并到 main，tag 可以打在任意分支
