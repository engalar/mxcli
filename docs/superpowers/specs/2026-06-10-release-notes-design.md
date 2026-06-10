# Release Notes: 三组件独立自动生成

**日期**: 2026-06-10  
**状态**: 已批准

## 背景

mxcli 仓库有三条独立的发布流，各自打不同前缀的 tag：

| 组件 | tag 流 | 示例 |
|---|---|---|
| Launcher | `v*` | `v0.23.0` |
| Daemon | `daemon-v*` | `daemon-v0.23.0` |
| mxcli-local | `local-v*` | `local-v0.5.0` |

三个组件可以独立发布，发布节奏不同（例如 local 目前在 v0.5.0，而 launcher 和 daemon 已到 v0.23.0）。

## 问题

三个 workflow 均使用 `generate_release_notes: true`。GitHub 的自动生成算法基于"上一个 release"进行比较，不区分 tag 流。因此：

- 为 `daemon-v0.23.0` 生成 notes 时，GitHub 可能以 `v0.23.0` 为比较基准（而非 `daemon-v0.22.0`），导致 notes 为空或不正确
- 三条 tag 流的 release notes 相互干扰，不能反映各自组件的实际变更

## 设计目标

每个组件的 GitHub Release 只展示**上次同类 tag 到本次 tag 之间**的变更，自动分类，无需人工维护。

## 方案：git-cliff + `--tag-pattern`

使用 `git-cliff`（官方 `orhun/git-cliff-action@v4`），通过 `--tag-pattern` 参数限定每条 tag 流的比较范围。

### cliff.toml

仓库根目录放一个共用配置文件：

```toml
[changelog]
body = """
{% for group, commits in commits | group_by(attribute="group") %}
### {{ group }}
{% for commit in commits %}
- {{ commit.message | split(pat=":") | last | trim }}\
{%- if commit.scope %} (`{{ commit.scope }}`){% endif %}

{% endfor %}
{% endfor %}
"""
trim = true

[git]
conventional_commits = true
filter_unconventional = false
commit_parsers = [
  { message = "^feat",              group = "Added"   },
  { message = "^fix",               group = "Fixed"   },
  { message = "^refactor|^perf",    group = "Changed" },
  { message = "^chore|^docs|^test|^ci", skip = true  },
]
```

跳过 `chore`、`docs`、`test`、`ci` 类提交——这些不是用户关心的面向用户的变更。

### 各 workflow 的 tag-pattern

| Workflow | `--tag-pattern` | 说明 |
|---|---|---|
| release.yml | `^v[0-9]` | 仅匹配 `v0.x.x`，排除 `daemon-v*` / `local-v*` |
| release-daemon.yml | `^daemon-v` | 仅匹配 `daemon-v*` |
| release-local.yml | `^local-v` | 仅匹配 `local-v*` |

### workflow 改动

每个 workflow 新增一个 git-cliff step，输出写入 `release-notes.md`；`softprops/action-gh-release` 改用 `body_path`，去掉 `generate_release_notes: true`：

```yaml
- uses: orhun/git-cliff-action@v4
  with:
    config: cliff.toml
    args: --tag-pattern '^daemon-v' --strip header --current
  env:
    OUTPUT: release-notes.md

- uses: softprops/action-gh-release@v3
  with:
    body_path: release-notes.md
    prerelease: true
    files: |
      ...
```

三个 workflow 的改动结构完全一致，只有 `--tag-pattern` 参数不同。

## 文件变更清单

| 文件 | 变更类型 | 说明 |
|---|---|---|
| `cliff.toml` | 新增 | 共用的 git-cliff 配置 |
| `.github/workflows/release.yml` | 修改 | 添加 git-cliff step，替换 `generate_release_notes: true` |
| `.github/workflows/release-daemon.yml` | 修改 | 同上，`--tag-pattern '^daemon-v'` |
| `.github/workflows/release-local.yml` | 修改 | 同上，`--tag-pattern '^local-v'` |

## 不在此范围内

- CHANGELOG.md 的维护方式不变（仍为手动，可后续用 git-cliff 自动更新）
- 不添加任何 commit scope 规范要求（方案 C 按 tag 区间，不过滤组件归属）
