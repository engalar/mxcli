# mxcli Launcher + Daemon 架构设计

**日期：** 2026-05-27  
**状态：** 待实现  
**问题：** 用户更新成本高（全量下载），无自动更新机制，高频调用慢

---

## 背景与目标

当前痛点：

- 全量下载 63-82 MB binary，网络成本高
- 无版本检查，无自动更新提示
- 高频 CLI 调用（脚本/CI 循环）每次冷启动 ~100ms，100 次 = 10s
- 单一大 binary 导致任何改动都需全量更新

目标：

- 用户感知到的更新成本大幅下降（launcher 极稳定，daemon 压缩后 ~20MB）
- 高频调用：daemon 热启动后每次 ~5ms
- 自动提示更新，`mxcli upgrade` 一键更新，带回滚保障
- 一行命令完成初次安装，支持所有主流平台和架构
- 安装脚本幂等，重复执行安全

---

## 架构概览

```
用户机器
─────────────────────────────────────────────────────────
mxcli (launcher)
  ~2MB | 跨平台 | CGO-free | 极稳定，几乎不需更新

  启动时：
    1. 检查 ~/.mxcli/daemon/mxcli-daemon 是否存在且版本匹配
       → 否：下载 mxcli-daemon-<os>-<arch>.tar.zst，解压
    2. 连接 ~/.mxcli/daemon/mxcli.sock
       → 未运行：启动 daemon，等待 socket ready（超时 5s）
    3. 发送 {argv, cwd, env}，流式接收 stdout/stderr，透传 exit code
    4. 后台 goroutine（不阻塞主流程）：
       → last-check > 1h：查 GitHub releases API
       → 有新版：写 ~/.mxcli/daemon/update-available
    5. 退出前：如 update-available 存在，打印一行提示后删除标记文件

mxcli-daemon (全量业务逻辑)
  ~63MB raw | 压缩后 ~20MB 下载 | 平台专属 binary
  所有当前 mxcli 命令（exec / show / lint / lsp / sql / export …）
  监听 unix socket（Linux/macOS）/ named pipe（Windows）
  Idle 5 分钟无请求 → 自动退出
  收到 SIGUSR1 → 当前请求完成后优雅退出（供 upgrade 使用）
─────────────────────────────────────────────────────────
```

---

## 存储布局

```
~/.mxcli/daemon/
  mxcli-daemon        ← 当前运行版本 binary
  mxcli-daemon.bak    ← 上一版本（始终保留，供 rollback；下次 upgrade 才覆盖）
  version             ← 当前版本字符串（如 "v0.14.0"）
  version.bak         ← 上一版本字符串
  last-check          ← 上次检查时间戳（Unix epoch）
  update-available    ← 有新版时由后台 goroutine 创建，内容为新版本号
  mxcli.sock          ← Unix socket（运行时动态创建）
  mxcli-daemon.pid    ← daemon PID（用于检测 stale socket）
```

---

## Launcher 职责

launcher 只做五件事，不含任何业务逻辑：

| 职责 | 说明 |
|------|------|
| Daemon 存在性检查 | 比对 `version` 文件与内嵌的预期版本，不匹配则触发下载 |
| Daemon 启停 | 检查 socket + PID，按需 start，daemon 自动 idle 退出 |
| 请求转发 | argv / cwd / env 序列化，stdout/stderr 流式透传，exit code 透传 |
| 后台版本检查 | 非阻塞 goroutine，限频 1h，结果写文件 |
| `mxcli upgrade` / `mxcli rollback` / `mxcli version` | 无需 daemon 即可执行 |

---

## Daemon 通信协议

Launcher 与 daemon 通过 unix socket 通信，使用简单 length-prefixed JSON 帧：

```
Request  → {"argv": ["exec", "foo.mdl"], "cwd": "/proj", "env": {"MX_DEBUG": "1"}}
Response → 流式帧：
  {"stream": "stdout", "data": "base64..."}  （可多次）
  {"stream": "stderr", "data": "base64..."}  （可多次）
  {"exit": 0}                                 （终止帧）
```

Health-check 请求：`{"argv": ["__healthcheck__"]}` → `{"ok": true, "version": "v0.14.0"}`

---

## 更新流程（`mxcli upgrade`）

```
mxcli upgrade
  │
  ├─ 1. 查 GitHub releases API，获取最新版本和下载 URL
  │
  ├─ 2. 下载 mxcli-daemon-<os>-<arch>.tar.zst 到临时文件
  │      → 校验 SHA256
  │
  ├─ 3. 滚动备份（丢弃 N-2）：
  │      mxcli-daemon      → mxcli-daemon.bak  （覆盖旧 .bak）
  │      version           → version.bak
  │
  ├─ 4. 解压 → mxcli-daemon.new，原子 rename → mxcli-daemon
  │      写入新 version 文件
  │
  ├─ 5. Health-check 新 daemon（超时 10s）：
  │      ├─ 成功 → 向旧 daemon 发 SIGUSR1（如仍在运行），打印 "Upgraded to v0.14.0"
  │      │         .bak 保留不删
  │      └─ 失败 → 自动回滚：
  │                  kill 新 daemon
  │                  mxcli-daemon.bak → mxcli-daemon
  │                  version.bak     → version
  │                  打印 "Upgrade failed (<reason>), rolled back to v0.13.2"
```

### Backup 保留策略

**始终保留一份 N-1 版本**，upgrade 成功后不删除 .bak，用户随时可 `mxcli rollback`：

```
初始：  mxcli-daemon=v0.13.2  .bak=（无）
升级后：mxcli-daemon=v0.14.0  .bak=v0.13.2  ← 保留
再升级：mxcli-daemon=v0.15.0  .bak=v0.14.0  ← 覆盖（v0.13.2 退出历史）
```

### 手动回滚

```sh
mxcli rollback            # 将 .bak 换回当前，重启 daemon
mxcli rollback --list     # 显示当前版本和备份版本
```

---

## 后台版本检查

非阻塞，不影响当前命令延迟：

1. launcher 启动时开一个 goroutine
2. 读 `last-check`，若距今 < 1h → goroutine 直接退出
3. 若 > 1h：GET `https://api.github.com/repos/engalar/mxcli/releases/latest`
4. 若新版 > 当前版：写 `update-available` 文件（内容为新版本号）
5. 更新 `last-check`
6. 主进程退出前：若 `update-available` 存在，打印提示，删除该文件：
   ```
   🆕 mxcli-daemon v0.14.0 available → run `mxcli upgrade`
   ```

---

## 安装脚本

### Linux / macOS

```sh
curl -fsSL https://raw.githubusercontent.com/engalar/mxcli/main/install.sh | sh
```

`install.sh` 核心逻辑（幂等）：

```sh
#!/bin/sh
set -e

REPO="engalar/mxcli"
INSTALL_DIR="${MXCLI_INSTALL_DIR:-}"

# 1. 检测平台
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported arch: $ARCH"; exit 1 ;;
esac

# 2. 获取最新 launcher 版本
LATEST=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
  | grep '"tag_name"' | cut -d'"' -f4)

# 3. 幂等检查
if command -v mxcli >/dev/null 2>&1; then
  CURRENT=$(mxcli version --short 2>/dev/null || echo "")
  if [ "$CURRENT" = "$LATEST" ]; then
    echo "mxcli $CURRENT already installed. Nothing to do."
    exit 0
  fi
  echo "Updating mxcli $CURRENT → $LATEST"
fi

# 4. 确定安装目录
if [ -z "$INSTALL_DIR" ]; then
  if [ -w "/usr/local/bin" ]; then
    INSTALL_DIR="/usr/local/bin"
  else
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
    # 幂等写入 PATH（去重）
    SHELL_RC="$HOME/.bashrc"
    [ -n "$ZSH_VERSION" ] && SHELL_RC="$HOME/.zshrc"
    grep -qF "$INSTALL_DIR" "$SHELL_RC" 2>/dev/null || \
      printf '\nexport PATH="%s:$PATH"\n' "$INSTALL_DIR" >> "$SHELL_RC"
  fi
fi

# 5. 下载 launcher（原子替换）
BIN_URL="https://github.com/$REPO/releases/download/$LATEST/mxcli-${OS}-${ARCH}"
TMP=$(mktemp)
curl -fsSL "$BIN_URL" -o "$TMP"
chmod +x "$TMP"
mv "$TMP" "$INSTALL_DIR/mxcli"

echo "✅ mxcli $LATEST installed to $INSTALL_DIR/mxcli"
echo "   Run 'mxcli version' to verify. Daemon will be downloaded on first use."
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/engalar/mxcli/main/install.ps1 | iex
```

`install.ps1` 要点：
- 检测 AMD64 / ARM64
- 下载 `mxcli-windows-<arch>.exe`
- 安装到 `$env:LOCALAPPDATA\mxcli\` 并加入用户 PATH（幂等）
- 已安装且版本相同 → 直接退出

### 幂等保证

| 场景 | 行为 |
|------|------|
| 相同版本重复运行 | 直接退出，不下载，不覆盖 |
| 新版本重复运行 | 原子替换 launcher binary |
| PATH 已包含安装目录 | 不重复写入 |
| 下载中断 | 临时文件清理，现有安装不受影响 |
| daemon .bak | 安装脚本不触碰 daemon 目录 |

---

## 发布产物（GitHub Releases）

```
mxcli-linux-amd64              ← launcher
mxcli-linux-arm64
mxcli-darwin-amd64
mxcli-darwin-arm64
mxcli-windows-amd64.exe
mxcli-windows-arm64.exe

mxcli-daemon-linux-amd64.tar.zst      ← daemon（压缩）
mxcli-daemon-linux-arm64.tar.zst
mxcli-daemon-darwin-amd64.tar.zst
mxcli-daemon-darwin-arm64.tar.zst
mxcli-daemon-windows-amd64.zip
mxcli-daemon-windows-arm64.zip

SHA256SUMS                             ← 所有产物的校验文件
```

Daemon 压缩预估：63MB → ~20MB（zstd --best），下载节省 ~68%。

---

## 性能对比

| 场景 | 当前 | 新架构 |
|------|------|--------|
| 单次冷调用 | ~100ms | ~100ms（daemon 冷启动） |
| 脚本循环 100 次 | ~10s | ~0.6s（daemon 热启动后 ~5ms/次） |
| 首次安装下载量 | 63MB | 2MB launcher + 20MB daemon = 22MB |
| 更新下载量（launcher 不变） | 63MB | 20MB daemon |

---

## 迁移路径

1. **阶段 1（本 spec）：** 实现 launcher + daemon 拆分 + 安装脚本 + upgrade/rollback
2. **阶段 2（未来）：** 在 daemon 内部按功能域划分 Go package 边界，为后续更细粒度更新做准备
3. **阶段 3（长期）：** 高频变更的业务规则（如 lint rules）迁移到 Starlark/WASM，实现脚本级热更新

---

## 不在本 spec 范围内

- Daemon 内部的 package 重组（阶段 2）
- WASM 组件（阶段 3）
- Delta patch / binary diff（被全量压缩方案替代）
- 企业私有更新服务器（可后续通过环境变量 `MXCLI_UPDATE_URL` 支持）
