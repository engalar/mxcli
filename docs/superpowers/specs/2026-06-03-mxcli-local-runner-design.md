# mxcli local runner — 设计文档

**日期**: 2026-06-03  
**状态**: 待实现  
**范围**: 无 Docker 本地运行支持 + 发布流水线拆分 + upgrade 命令对齐

---

## 背景

mxcli 现有的构建（`mxcli docker build`）和运行（`mxcli docker run`）假设 Docker 可用。
企业内限制 Docker 的用户（Windows/Linux 本地环境）无法使用，必须独立摸索替代方案。

本设计解决两个问题：
1. 新增 `mxcli local` 子命令组，支持无 Docker 的 PAD 构建与运行
2. 重构发布流水线，将三个二进制的 release 完全解耦

---

## 关键发现（设计依据）

- `mxcli docker build` 完全不需要 Docker 进程——它只调用 mxbuild + Java，在 Windows/Linux/macOS 均可运行
- PAD 产物（`.docker/build/`）包含 `bin/start`（POSIX shell 脚本，Windows 不可用）和 `lib/runtime/launcher/runtimelauncher.jar`
- PAD 默认数据库为 HSQLDB（内嵌，无需外部 PostgreSQL），`etc/configurations/Default.conf` 指定
- 配置格式为 HOCON（Typesafe Config），通过环境变量 `RUNTIME_PARAMS_*` 覆盖
- `etc/variables.conf` 完整映射了所有可覆盖的运行时参数

---

## 一、三进制架构

### 1.1 进程拓扑

```
mxcli-launcher          mxcli-daemon             mxcli-local
（用户 PATH，稳定）       （命令执行引擎）           （PAD 构建/运行）
cmd/mxcli-launcher/      cmd/mxcli/               cmd/mxcli-local/
tag: v*                  tag: daemon-v*            tag: local-v*
~/.mxcli/daemon/         ~/.mxcli/daemon/          ~/.mxcli/local/
mxcli-daemon[.exe]       mxcli-daemon[.exe]        mxcli-local[.exe]
```

### 1.2 命令路由

| 命令 | 路由 |
|------|------|
| `mxcli local build` | launcher → 确保 mxcli-local → exec mxcli-local build |
| `mxcli local run` | launcher → 确保 mxcli-local → exec mxcli-local run（继承 stdio，前台阻塞） |
| `mxcli local upgrade` | launcher 直接处理，下载最新 local-v* release |
| `mxcli daemon upgrade` | launcher 直接处理，下载最新 daemon-v* release（现 `mxcli upgrade` 逻辑平移） |
| `mxcli daemon rollback` | launcher 直接处理（现 `mxcli rollback` 逻辑平移） |
| `mxcli upgrade` | launcher self-fork updater 替换自身 |
| 其余命令 | launcher → daemon socket（现有路由不变） |

`mxcli local *` 命令**绕过 daemon**，由 launcher 直接路由到 mxcli-local。
daemon 代码**完全不动**。

---

## 二、CI / Release 流水线拆分

### 2.1 现状

```
v* tag → 同一 GitHub Release → mxcli-launcher + mxcli-daemon 捆绑发布
```

### 2.2 目标：三条独立流水线

```yaml
# .github/workflows/release-launcher.yml  —  触发: tags: ['v*']
artifacts:
  - mxcli-linux-amd64
  - mxcli-linux-arm64
  - mxcli-darwin-amd64
  - mxcli-darwin-arm64
  - mxcli-windows-amd64.exe
  - mxcli-windows-arm64.exe
  - SHA256SUMS
  - install.sh / install.ps1

# .github/workflows/release-daemon.yml   —  触发: tags: ['daemon-v*']
artifacts:
  - mxcli-daemon-linux-amd64.tar.zst
  - mxcli-daemon-linux-arm64.tar.zst
  - mxcli-daemon-darwin-amd64.tar.zst
  - mxcli-daemon-darwin-arm64.tar.zst
  - mxcli-daemon-windows-amd64.exe.zip
  - mxcli-daemon-windows-arm64.exe.zip
  - SHA256SUMS

# .github/workflows/release-local.yml    —  触发: tags: ['local-v*']
artifacts:
  - mxcli-local-linux-amd64.tar.zst
  - mxcli-local-linux-arm64.tar.zst
  - mxcli-local-darwin-amd64.tar.zst
  - mxcli-local-darwin-arm64.tar.zst
  - mxcli-local-windows-amd64.exe.zip
  - mxcli-local-windows-arm64.exe.zip
  - SHA256SUMS
```

现有 `release.yml` 拆分为上述三个文件，`ci.yml` 不变。

### 2.3 Makefile 对应目标

```makefile
release-launcher:   # 构建 6 平台 launcher
release-daemon:     # 构建 6 平台 daemon（含压缩）
release-local:      # 构建 6 平台 mxcli-local（含压缩）
release:            # 三者全构建（向后兼容本地开发用）
```

---

## 三、Upgrade 命令对齐

### 3.1 命令矩阵

| 命令 | 目标 | 实现 |
|------|------|------|
| `mxcli upgrade` | 升级 launcher 自身 | self-fork updater |
| `mxcli upgrade --list` | 查看当前/可用版本 | 查询 v* release |
| `mxcli daemon upgrade` | 升级 mxcli-daemon | 现有 `downloadDaemon` 逻辑，指向 daemon-v* |
| `mxcli daemon rollback` | 回滚 daemon | 现有 `rollback()` 逻辑平移 |
| `mxcli daemon status` | daemon 运行状态 | 现有状态查询逻辑平移 |
| `mxcli local upgrade` | 升级 mxcli-local | 同一套下载+替换逻辑，指向 local-v* |
| `mxcli local rollback` | 回滚 mxcli-local | 同一套 .bak 机制 |

### 3.2 共享 upgradeComponent 函数

三个 upgrade 操作共享同一实现，差异只在路径和 release tag 前缀：

```go
type ComponentConfig struct {
    Name       string   // "daemon" / "local"
    InstallDir string   // ~/.mxcli/daemon/ 或 ~/.mxcli/local/
    BinaryName string   // mxcli-daemon / mxcli-local
    TagPrefix  string   // "daemon-v" / "local-v" / "v"（launcher 自身）
    AssetFmt   string   // "mxcli-daemon-{os}-{arch}" 等
}

func (e *Env) upgradeComponent(cfg ComponentConfig) error
func (e *Env) rollbackComponent(cfg ComponentConfig) error
```

**Tag 前缀过滤**：GitHub `/releases/latest` 返回全局最新 release，无法按 tag 前缀过滤。
改用 `/releases?per_page=20` 列表接口，遍历找第一个 `tag_name` 以 `TagPrefix` 开头的 release。
同一 `fetachLatestTagWithPrefix(prefix string)` 函数供三个 component 共用。

现有 `downloadDaemon` / `extractTarZst` / `fetchAssetChecksum` 等函数
重构为 `upgradeComponent` 的内部实现，对外接口不变。

### 3.3 Launcher 自升级（Self-fork Updater）

```
mxcli upgrade
  │
  ├─ fetchLatestTag("v*")
  ├─ 比较当前版本，若已最新则退出
  ├─ 下载新 launcher → {installDir}/mxcli.new[.exe]
  ├─ SHA256 校验
  ├─ 启动自身副本（隐藏窗口）：
  │    mxcli --internal-update \
  │          --pid=<自己的PID> \
  │          --new=<mxcli.new路径> \
  │          --target=<当前二进制路径>
  └─ os.Exit(0)   ← 父进程立即退出，释放文件锁

子进程（--internal-update 模式）：
  ├─ 等待父 PID 消失（轮询 100ms，超时 30s）
  │    Windows: OpenProcess + WaitForSingleObject
  │    POSIX:   kill(pid, 0) 返回 ESRCH
  ├─ POSIX:   os.Rename(new → target)（原子）
  │   Windows: os.Rename(target → target.old)
  │            os.Rename(new → target)
  ├─ exec 新版本 `mxcli version` 验证（非零退出则回滚：os.Rename(target.old → target)）
  └─ 写入新版本到 version 文件
```

`--internal-update` 为内部隐藏 flag，`cmd_local.go` 里 `cobra.Command.Hidden = true`。
Windows 窗口隐藏复用现有 `hideDaemonWindow(cmd)`（`spawn_windows.go`）。
`.old` 文件在下次 launcher 启动时清理（`cleanupOldBinary()` 在 `main()` 入口调用）。

---

## 四、mxcli-local 二进制

### 4.1 代码结构

```
cmd/mxcli-local/
├── main.go          — Cobra 入口，Version 变量由 LDFLAGS 注入
├── cmd_build.go     — build 子命令 → docker.Build()
└── cmd_run.go       — run 子命令 → docker.StartLocal()

cmd/mxcli/docker/
└── local.go         — 新增 StartLocal() Go 原生 java 启动器
```

共享包：`cmd/mxcli/docker/`（Build、resolveJDK21、detect.go、download.go 全部复用）。

### 4.2 命令接口

```
mxcli local build -p app.mpr [--skip-check] [--skip-update-widgets]
    # 等价于 mxcli docker build，产物在 .docker/build/

mxcli local run -p app.mpr [--db <url>] [--port <n>] [--admin-port <n>]
    # 前台阻塞，Ctrl+C 停止
    # 默认: HSQLDB，port=8080，admin-port=8090
    # --db postgres://user:pass@host/db → 注入 RUNTIME_PARAMS_* 环境变量
```

### 4.3 安装路径与版本追踪

```
~/.mxcli/local/
├── mxcli-local[.exe]      ← 主二进制
├── mxcli-local.bak[.exe]  ← rollback 备份
├── version                ← 当前版本 tag
├── version.bak            ← rollback 版本
├── last-check             ← 上次版本检查时间戳
└── update-available       ← 有新版本时写入（后台检查）
```

---

## 五、StartLocal() Go 原生 Java 启动器

### 5.1 实现位置

`cmd/mxcli/docker/local.go`

### 5.2 PAD 目录校验

调用现有 `hasExtractedPADLayout(padDir)`，若未找到则报错：
```
no PAD found at .docker/build/: run 'mxcli local build -p app.mpr' first
```

### 5.3 Java 命令构造（复刻 bin/start 逻辑）

```go
// 平台差异
libPathSep  := ":" // POSIX
if runtime.GOOS == "windows" { libPathSep = ";" }

javaBin := filepath.Join(javaHome, "bin", "java")
if runtime.GOOS == "windows" { javaBin += ".exe" }

libDir       := filepath.Join(padDir, "lib")
launcherJar  := filepath.Join(libDir, "runtime", "launcher", "runtimelauncher.jar")
nativeLibs   := filepath.Join(libDir, "runtime", "lib", "x64") + libPathSep +
                filepath.Join(padDir, "app", "model", "lib", "userlib")

args := []string{
    "-DMX_LOG_LEVEL=INFO",
    "-Dfile.encoding=UTF-8",
    "-Djava.io.tmpdir=" + os.TempDir(),
    "-Djava.library.path=" + nativeLibs,
    "-DMX_INSTALL_PATH=" + libDir,
}
// JVM opts from HOCON config (jvm.heap / jvm.params，简单 grep，非完整 HOCON 解析)
args = append(args, parseJVMOpts(padDir)...)
args = append(args, "-jar", launcherJar)
args = append(args, filepath.Join(padDir, "app")+string(os.PathSeparator))
args = append(args, filepath.Join(padDir, "etc", "Default"))
```

### 5.4 数据库配置注入

`--db postgres://user:pass@host:5432/dbname` 解析后注入环境变量：

| --db 解析字段 | 注入环境变量 |
|--------------|-------------|
| scheme=postgresql | `RUNTIME_PARAMS_DATABASETYPE=PostgreSQL` |
| host:port | `RUNTIME_PARAMS_DATABASEHOST=host:port` |
| path=/dbname | `RUNTIME_PARAMS_DATABASENAME=dbname` |
| user | `RUNTIME_PARAMS_DATABASEUSERNAME=user` |
| password | `RUNTIME_PARAMS_DATABASEPASSWORD=pass` |
| （完整 URL） | `RUNTIME_PARAMS_DATABASEJDBCURL=jdbc:postgresql://host:5432/dbname` |

`etc/variables.conf` 已将这些环境变量映射到 HOCON 配置，无需修改 PAD 文件。

### 5.5 进程启动

```go
cmd := exec.Command(javaBin, args...)
cmd.Env = append(os.Environ(), dbEnvVars...)
cmd.Stdout = opts.Stdout  // 继承 launcher 的 stdout
cmd.Stderr = opts.Stderr
return cmd.Run()           // 阻塞直到 Ctrl+C 或 runtime 退出
```

---

## 六、测试策略

| 层级 | 覆盖内容 |
|------|---------|
| Unit | `parseJVMOpts`、`buildJavaArgs`、`parseDBURL`、`upgradeComponent`、self-fork updater flag 解析 |
| Integration | `StartLocal` 在 Linux CI 上用 testdata PAD 验证 java 命令可构造（不实际启动 runtime） |
| CI | `release-local.yml` 跑 `make release-local` 验证交叉编译通过 |
| Windows | `spawn_windows.go` 的 `hideDaemonWindow` 已有测试；updater PID 等待逻辑用 mock process 测试 |

---

## 七、不在范围内

- `mxcli local logs`（日志已直接流到终端）
- `mxcli local status`（前台模式下进程状态即终端状态）
- macOS 支持（Studio Pro 路径已有，但本次需求只提 Windows + Linux）
- skill 文件更新（`run-app.md`、`test-app.md`）—— 独立 PR

---

## 八、文件改动清单

| 文件 | 类型 | 说明 |
|------|------|------|
| `cmd/mxcli-local/main.go` | 新增 | mxcli-local Cobra 入口 |
| `cmd/mxcli-local/cmd_build.go` | 新增 | build 子命令 |
| `cmd/mxcli-local/cmd_run.go` | 新增 | run 子命令 |
| `cmd/mxcli/docker/local.go` | 新增 | StartLocal() 实现 |
| `cmd/mxcli-launcher/local.go` | 新增 | mxcli-local 下载/路由 |
| `cmd/mxcli-launcher/self_update.go` | 新增 | self-fork updater |
| `cmd/mxcli-launcher/self_update_windows.go` | 新增 | Windows PID 等待 |
| `cmd/mxcli-launcher/self_update_unix.go` | 新增 | POSIX PID 等待 |
| `cmd/mxcli-launcher/upgrade.go` | 重构 | upgradeComponent 共享函数 |
| `cmd/mxcli-launcher/main.go` | 修改 | 注册 local 路由 + internal-update flag |
| `cmd/mxcli-launcher/daemon.go` | 重构 | downloadDaemon → upgradeComponent |
| `.github/workflows/release.yml` | 拆分 | 拆为三个独立 workflow |
| `.github/workflows/release-launcher.yml` | 新增 | v* tag 触发 |
| `.github/workflows/release-daemon.yml` | 新增 | daemon-v* tag 触发 |
| `.github/workflows/release-local.yml` | 新增 | local-v* tag 触发 |
| `Makefile` | 修改 | release-launcher / release-daemon / release-local 目标 |
| `cmd/mxcli-launcher/main.go` | 修改 | `cleanupOldBinary()` 在入口清理 .old 文件 |
