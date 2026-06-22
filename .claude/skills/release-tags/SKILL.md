---
name: release-tags
description: Use when creating release tags for mxcli on the current branch
---

# mxcli Release Tags

## Tag 规则

| 组件 | Tag 格式 |
|------|---------|
| mxcli (唯一二进制) | `v*` |

合并后只有单一 `mxcli` 二进制，不再需要 daemon-v* 和 local-v* tag。

## 流程

### 1. 查看当前最新版本

```bash
git tag --sort=-version:refname | grep -E '^v[0-9]' | head -3
```

### 2. 确定下一版本号

- patch 号 +1（如 v0.28.0 → v0.29.0）
- 含 breaking change 时升 minor

### 3. 创建 tag

```bash
git tag v0.29.0
```

### 4. Push 触发 CI

```bash
git push origin v0.29.0
```

## 注意

- Tag 打在哪个 commit 上，CI 就从那个 commit 构建
- 误打的 tag 可删除：`git tag -d <tag> && git push origin :refs/tags/<tag>`
