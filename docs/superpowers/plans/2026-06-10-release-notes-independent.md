# Release Notes 独立自动生成 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让三个 GitHub Actions 发布流（launcher/daemon/local）各自只展示上次同类 tag 到本次 tag 之间的变更，互不干扰。

**Architecture:** 在仓库根目录新增 `cliff.toml` 作为共用配置；三个 workflow 各自用 `orhun/git-cliff-action@v4` 并传入不同的 `--tag-pattern` 参数，输出写入 `release-notes.md`，再通过 `body_path` 传给 `softprops/action-gh-release`，替换原有的 `generate_release_notes: true`。

**Tech Stack:** [git-cliff v2](https://github.com/orhun/git-cliff)，`orhun/git-cliff-action@v4`，`softprops/action-gh-release@v3`

---

## 文件变更清单

| 文件 | 操作 |
|---|---|
| `cliff.toml` | 新增 |
| `.github/workflows/release.yml` | 修改 |
| `.github/workflows/release-daemon.yml` | 修改 |
| `.github/workflows/release-local.yml` | 修改 |

---

### Task 1：新增 cliff.toml

**Files:**
- Create: `cliff.toml`

- [ ] **Step 1: 创建 cliff.toml**

在仓库根目录新建文件，内容如下：

```toml
[changelog]
body = """
{% if commits %}\
{% for group, commits in commits | group_by(attribute="group") %}
### {{ group }}

{% for commit in commits %}
- {{ commit.message | split(pat=":") | last | trim }}\
{%- if commit.scope %} (`{{ commit.scope }}`){% endif %}

{% endfor %}
{% endfor %}\
{% else %}
*No user-facing changes in this release.*
{% endif %}
"""
trim = true

[git]
conventional_commits = true
filter_unconventional = false
split_commits = false
protect_breaking_commits = false
filter_commits = false
commit_parsers = [
  { message = "^feat",                      group = "Added"   },
  { message = "^fix",                       group = "Fixed"   },
  { message = "^refactor|^perf",            group = "Changed" },
  { message = "^chore|^docs|^test|^ci",     skip = true       },
]
```

- [ ] **Step 2: 本地验证 cliff.toml 可解析**

安装 git-cliff（如未安装）：
```bash
# macOS
brew install git-cliff

# Linux（下载二进制）
curl -L https://github.com/orhun/git-cliff/releases/download/v2.7.0/git-cliff-x86_64-unknown-linux-musl.tar.gz | tar -xz
sudo mv git-cliff-*/git-cliff /usr/local/bin/
```

在仓库根目录运行，验证三条 tag 流各自的输出：

```bash
# Launcher：仅看 v0.x.x 之间的变更
git cliff --config cliff.toml --tag-pattern '^v[0-9]' --strip header --current

# Daemon：仅看 daemon-v* 之间的变更
git cliff --config cliff.toml --tag-pattern '^daemon-v' --strip header --current

# Local：仅看 local-v* 之间的变更
git cliff --config cliff.toml --tag-pattern '^local-v' --strip header --current
```

预期：三条命令各自输出对应 tag 区间内的 Added / Fixed / Changed 分组列表，内容互不重叠。

- [ ] **Step 3: Commit**

```bash
git add cliff.toml
git commit -m "chore: add cliff.toml for component-scoped release notes"
```

---

### Task 2：更新 release.yml（Launcher）

**Files:**
- Modify: `.github/workflows/release.yml`

- [ ] **Step 1: 在 "Generate SHA256 checksums" 步骤之后插入 git-cliff 步骤**

在 `Generate SHA256 checksums` 和 `Create GitHub Release` 步骤之间插入：

```yaml
      - name: Generate release notes
        uses: orhun/git-cliff-action@v4
        with:
          config: cliff.toml
          args: --tag-pattern '^v[0-9]' --strip header --current
        env:
          OUTPUT: release-notes.md
```

- [ ] **Step 2: 修改 "Create GitHub Release" 步骤**

将 `generate_release_notes: true` 替换为 `body_path: release-notes.md`：

```yaml
      - name: Create GitHub Release
        uses: softprops/action-gh-release@v3
        with:
          body_path: release-notes.md
          files: |
            bin/mxcli-linux-amd64
            bin/mxcli-linux-arm64
            bin/mxcli-darwin-amd64
            bin/mxcli-darwin-arm64
            bin/mxcli-windows-amd64.exe
            bin/mxcli-windows-arm64.exe
            bin/SHA256SUMS
            install.sh
            install.ps1
```

完整修改后的 `jobs.release.steps` 部分应为：

```yaml
    steps:
      - uses: actions/checkout@v6
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v6
        with:
          go-version: '1.26'
      - uses: oven-sh/setup-bun@v2
      - name: Cache ANTLR4 JAR
        uses: actions/cache@v5
        with:
          path: ~/.m2/repository/org/antlr/antlr4
          key: antlr4-4.13.2
      - name: Install ANTLR4
        run: pip install 'antlr4-tools==0.2.2'
      - name: Generate parser
        run: make grammar
        env:
          ANTLR4_TOOLS_ANTLR_VERSION: '4.13.2'

      - name: Build launcher binaries
        run: make release-launcher

      - name: Generate SHA256 checksums
        run: |
          cd bin
          sha256sum mxcli-linux-amd64 mxcli-linux-arm64 \
                    mxcli-darwin-amd64 mxcli-darwin-arm64 \
                    mxcli-windows-amd64.exe mxcli-windows-arm64.exe > SHA256SUMS
          cat SHA256SUMS

      - name: Generate release notes
        uses: orhun/git-cliff-action@v4
        with:
          config: cliff.toml
          args: --tag-pattern '^v[0-9]' --strip header --current
        env:
          OUTPUT: release-notes.md

      - name: Create GitHub Release
        uses: softprops/action-gh-release@v3
        with:
          body_path: release-notes.md
          files: |
            bin/mxcli-linux-amd64
            bin/mxcli-linux-arm64
            bin/mxcli-darwin-amd64
            bin/mxcli-darwin-arm64
            bin/mxcli-windows-amd64.exe
            bin/mxcli-windows-arm64.exe
            bin/SHA256SUMS
            install.sh
            install.ps1
```

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/release.yml
git commit -m "ci(release): use git-cliff for launcher release notes"
```

---

### Task 3：更新 release-daemon.yml（Daemon）

**Files:**
- Modify: `.github/workflows/release-daemon.yml`

- [ ] **Step 1: 在 "Generate SHA256 checksums" 之后插入 git-cliff 步骤**

```yaml
      - name: Generate release notes
        uses: orhun/git-cliff-action@v4
        with:
          config: cliff.toml
          args: --tag-pattern '^daemon-v' --strip header --current
        env:
          OUTPUT: release-notes.md
```

- [ ] **Step 2: 修改 "Create GitHub Release" 步骤**

将 `generate_release_notes: true` 替换为 `body_path: release-notes.md`。

完整修改后的 `jobs.release.steps` 部分应为：

```yaml
    steps:
      - uses: actions/checkout@v6
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v6
        with:
          go-version: '1.26'
      - uses: oven-sh/setup-bun@v2
      - name: Cache ANTLR4 JAR
        uses: actions/cache@v5
        with:
          path: ~/.m2/repository/org/antlr/antlr4
          key: antlr4-4.13.2
      - name: Install ANTLR4
        run: pip install 'antlr4-tools==0.2.2'
      - name: Generate parser
        run: make grammar
        env:
          ANTLR4_TOOLS_ANTLR_VERSION: '4.13.2'
      - name: Install zstd
        run: sudo apt-get install -y zstd

      - name: Build daemon binaries
        run: make release-daemon

      - name: Generate SHA256 checksums
        run: |
          cd bin
          sha256sum mxcli-daemon-* > SHA256SUMS
          cat SHA256SUMS

      - name: Generate release notes
        uses: orhun/git-cliff-action@v4
        with:
          config: cliff.toml
          args: --tag-pattern '^daemon-v' --strip header --current
        env:
          OUTPUT: release-notes.md

      - name: Create GitHub Release
        uses: softprops/action-gh-release@v3
        with:
          body_path: release-notes.md
          prerelease: true
          files: |
            bin/mxcli-daemon-linux-amd64.tar.zst
            bin/mxcli-daemon-linux-arm64.tar.zst
            bin/mxcli-daemon-darwin-amd64.tar.zst
            bin/mxcli-daemon-darwin-arm64.tar.zst
            bin/mxcli-daemon-windows-amd64.exe.zip
            bin/mxcli-daemon-windows-arm64.exe.zip
            bin/SHA256SUMS
```

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/release-daemon.yml
git commit -m "ci(release): use git-cliff for daemon release notes"
```

---

### Task 4：更新 release-local.yml（mxcli-local）

**Files:**
- Modify: `.github/workflows/release-local.yml`

- [ ] **Step 1: 在 "Generate SHA256 checksums" 之后插入 git-cliff 步骤**

```yaml
      - name: Generate release notes
        uses: orhun/git-cliff-action@v4
        with:
          config: cliff.toml
          args: --tag-pattern '^local-v' --strip header --current
        env:
          OUTPUT: release-notes.md
```

- [ ] **Step 2: 修改 "Create GitHub Release" 步骤**

将 `generate_release_notes: true` 替换为 `body_path: release-notes.md`。

完整修改后的 `jobs.release.steps` 部分应为：

```yaml
    steps:
      - uses: actions/checkout@v6
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v6
        with:
          go-version: '1.26'
      - uses: oven-sh/setup-bun@v2
      - name: Cache ANTLR4 JAR
        uses: actions/cache@v5
        with:
          path: ~/.m2/repository/org/antlr/antlr4
          key: antlr4-4.13.2
      - name: Install ANTLR4
        run: pip install 'antlr4-tools==0.2.2'
      - name: Generate parser
        run: make grammar
        env:
          ANTLR4_TOOLS_ANTLR_VERSION: '4.13.2'
      - name: Install zstd
        run: sudo apt-get install -y zstd

      - name: Build mxcli-local binaries
        run: make release-local-bins

      - name: Generate SHA256 checksums
        run: |
          cd bin
          sha256sum mxcli-local-* > SHA256SUMS
          cat SHA256SUMS

      - name: Generate release notes
        uses: orhun/git-cliff-action@v4
        with:
          config: cliff.toml
          args: --tag-pattern '^local-v' --strip header --current
        env:
          OUTPUT: release-notes.md

      - name: Create GitHub Release
        uses: softprops/action-gh-release@v3
        with:
          body_path: release-notes.md
          prerelease: true
          files: |
            bin/mxcli-local-linux-amd64.tar.zst
            bin/mxcli-local-linux-arm64.tar.zst
            bin/mxcli-local-darwin-amd64.tar.zst
            bin/mxcli-local-darwin-arm64.tar.zst
            bin/mxcli-local-windows-amd64.exe.zip
            bin/mxcli-local-windows-arm64.exe.zip
            bin/SHA256SUMS
```

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/release-local.yml
git commit -m "ci(release): use git-cliff for local release notes"
```
