# mxcli 手动安装与更新手册

自动安装脚本失效时的回落手段。适用于网络受限、企业代理、GitHub 访问异常等场景。

---

## 组件说明

mxcli 由三个独立发布的二进制组成：

| 组件 | GitHub tag | 安装路径 | 说明 |
|------|-----------|---------|------|
| `mxcli`（launcher） | `v*` | 用户 PATH | 路由命令、管理子组件 |
| `mxcli-daemon` | `daemon-v*` | `~/.mxcli/daemon/` | MDL 命令、LSP、catalog |
| `mxcli-local` | `local-v*` | `~/.mxcli/local/` | 本地构建与运行（local build/run） |

---

## Linux / macOS 手动安装

### 1. 下载 launcher

前往 https://github.com/engalar/mxcli/releases，找最新 `v*` release（非 `daemon-v*` / `local-v*`）。

```bash
# 示例：v0.18.0，Linux amd64
RELEASE="v0.18.0"
curl -fsSL -o /tmp/mxcli \
  "https://github.com/engalar/mxcli/releases/download/${RELEASE}/mxcli-linux-amd64"

# 校验（SHA256SUMS 文件同一 release 页面下载）
curl -fsSL -o /tmp/SHA256SUMS \
  "https://github.com/engalar/mxcli/releases/download/${RELEASE}/SHA256SUMS"
grep "mxcli-linux-amd64" /tmp/SHA256SUMS | sha256sum -c

chmod +x /tmp/mxcli
mv /tmp/mxcli ~/.local/bin/mxcli          # 或 /usr/local/bin/mxcli（需要 sudo）
```

macOS arm64 替换为 `mxcli-darwin-arm64`，amd64 替换为 `mxcli-darwin-amd64`。

### 2. 手动安装 daemon

```bash
DAEMON_RELEASE="daemon-v0.18.0"          # 查看 releases 页面找最新 daemon-v*
curl -fsSL -o /tmp/mxcli-daemon.tar.zst \
  "https://github.com/engalar/mxcli/releases/download/${DAEMON_RELEASE}/mxcli-daemon-linux-amd64.tar.zst"

mkdir -p ~/.mxcli/daemon
tar -C ~/.mxcli/daemon -xf /tmp/mxcli-daemon.tar.zst
chmod +x ~/.mxcli/daemon/mxcli-daemon
```

### 3. 手动安装 mxcli-local（仅需要 local build/run 时）

```bash
LOCAL_RELEASE="local-v0.1.0"             # 查看 releases 页面找最新 local-v*
curl -fsSL -o /tmp/mxcli-local.tar.zst \
  "https://github.com/engalar/mxcli/releases/download/${LOCAL_RELEASE}/mxcli-local-linux-amd64.tar.zst"

mkdir -p ~/.mxcli/local
tar -C ~/.mxcli/local -xf /tmp/mxcli-local.tar.zst
chmod +x ~/.mxcli/local/mxcli-local
```

### 4. 验证

```bash
mxcli version
# 期望：mxcli launcher vX.Y.Z
```

---

## Windows 手动安装

在 PowerShell 中执行：

```powershell
# 1. 下载 launcher
$Release = "v0.18.0"
$InstallDir = "$env:LOCALAPPDATA\mxcli"
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

Invoke-WebRequest -Uri "https://github.com/engalar/mxcli/releases/download/$Release/mxcli-windows-amd64.exe" `
  -OutFile "$InstallDir\mxcli.exe" -UseBasicParsing
Unblock-File "$InstallDir\mxcli.exe"

# 添加到 PATH（仅当前用户）
$UserPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($UserPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("PATH", "$UserPath;$InstallDir", "User")
}

# 2. 下载 daemon
$DaemonRelease = "daemon-v0.18.0"
$DaemonDir = "$env:USERPROFILE\.mxcli\daemon"
New-Item -ItemType Directory -Force -Path $DaemonDir | Out-Null

$Tmp = "$env:TEMP\mxcli-daemon.zip"
Invoke-WebRequest -Uri "https://github.com/engalar/mxcli/releases/download/$DaemonRelease/mxcli-daemon-windows-amd64.exe.zip" `
  -OutFile $Tmp -UseBasicParsing
Expand-Archive -Path $Tmp -DestinationPath $DaemonDir -Force
Remove-Item $Tmp

# 3. 下载 mxcli-local（可选）
$LocalRelease = "local-v0.1.0"
$LocalDir = "$env:USERPROFILE\.mxcli\local"
New-Item -ItemType Directory -Force -Path $LocalDir | Out-Null

$Tmp = "$env:TEMP\mxcli-local.zip"
Invoke-WebRequest -Uri "https://github.com/engalar/mxcli/releases/download/$LocalRelease/mxcli-local-windows-amd64.exe.zip" `
  -OutFile $Tmp -UseBasicParsing
Expand-Archive -Path $Tmp -DestinationPath $LocalDir -Force
Remove-Item $Tmp

# 重新打开终端后验证
mxcli version
```

---

## 手动更新

### 更新 launcher

```bash
# Linux/macOS
NEW="v0.19.0"
curl -fsSL -o /tmp/mxcli \
  "https://github.com/engalar/mxcli/releases/download/${NEW}/mxcli-linux-amd64"
chmod +x /tmp/mxcli
mv /tmp/mxcli $(which mxcli)
```

或使用自动命令（网络正常时）：
```bash
mxcli upgrade
```

### 更新 daemon

```bash
mxcli daemon upgrade        # 自动
# 手动：重复第 2 步，覆盖 ~/.mxcli/daemon/mxcli-daemon
```

回滚：
```bash
mxcli daemon rollback
```

### 更新 mxcli-local

```bash
mxcli local upgrade         # 自动
# 手动：重复第 3 步，覆盖 ~/.mxcli/local/mxcli-local
```

回滚：
```bash
mxcli local rollback
```

---

## 查看当前版本

```bash
mxcli version               # launcher 版本
mxcli daemon status         # daemon 版本 + 运行状态
mxcli local --version       # mxcli-local 版本
```

---

## 常见问题

### 自动安装时 "GitHub releases: HTTP 403"

GitHub API 限流（60次/小时）。解决方法：
- 等 1 小时后重试，或
- 按本手册手动安装对应版本

### "mxcli-local not found" 但网络正常

先手动安装 mxcli-local（第 3 步），再执行 `mxcli local build`。

### Windows：执行策略阻止脚本

```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

### 找最新版本号

- Launcher：https://github.com/engalar/mxcli/releases（找 `v*` tag）
- Daemon：同页面找 `daemon-v*` tag
- Local：同页面找 `local-v*` tag

或命令行：
```bash
curl -fsSL -o /dev/null -w '%{url_effective}' \
  "https://github.com/engalar/mxcli/releases/latest" | sed 's|.*/tag/||'
```
