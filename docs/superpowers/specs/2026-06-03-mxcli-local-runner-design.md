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

## 六、测试夹具设计

**设计原则**：夹具编码契约。夹具知道"合法状态"的全部细节——若实现改变导致夹具不再有效，测试立即失败，而不是悄悄通过。每个夹具对应一个可注入的接口，生产代码与测试代码解耦。

---

### 6.1 FakePAD（`cmd/mxcli/docker/testfixtures/`）

编码 `hasExtractedPADLayout()` 所要求的合法 PAD 结构。若结构约束变化，夹具必须同步更新。

```go
// FakePAD creates a minimal valid PAD directory for StartLocal tests.
type FakePAD struct {
    Dir string
}

func NewFakePAD(t *testing.T) *FakePAD
// 创建：bin/start(可执行)、lib/runtime/launcher/runtimelauncher.jar(空文件)
// lib/runtime/lib/x64/、app/model/lib/userlib/、etc/Default(include chain)
// etc/configurations/Default.conf、etc/variables.conf、etc/StudioPro.conf

func (p *FakePAD) SetJVMHeap(heap string) *FakePAD       // 写 jvm.heap = 512m
func (p *FakePAD) SetJVMParams(params string) *FakePAD   // 写 jvm.params = "-XX:..."
func (p *FakePAD) SetPorts(runtime, admin int) *FakePAD  // 覆盖 Default.conf 端口
func (p *FakePAD) SetDBConfig(dbType, jdbcURL, user, pass string) *FakePAD
```

**防退化价值**：`hasExtractedPADLayout` 的每个检查项在 `NewFakePAD` 里都有对应的创建语句，少一项则 `TestStartLocal_ValidPAD` 失败。

---

### 6.2 ProcessStarter 接口（`cmd/mxcli/docker/`）

`StartLocal()` 通过此接口启动 java 进程，测试注入 `CaptureStarter` 而非真实 `exec.Cmd`。

```go
// ProcessStarter abstracts exec.Cmd start + wait.
type ProcessStarter interface {
    Run(cmd *exec.Cmd) error
}

// RealStarter: cmd.Run()（生产）
// CaptureStarter: 记录调用，立即返回 nil（测试）

type CapturedInvocation struct {
    Binary string
    Args   []string
    Env    []string
}

type CaptureStarter struct {
    Invocations []CapturedInvocation
}

func (c *CaptureStarter) Run(cmd *exec.Cmd) error {
    c.Invocations = append(c.Invocations, CapturedInvocation{
        Binary: cmd.Path,
        Args:   cmd.Args,
        Env:    cmd.Env,
    })
    return nil
}
```

**典型测试**：
```go
func TestStartLocal_JavaArgs(t *testing.T) {
    pad := testfixtures.NewFakePAD(t).SetJVMHeap("512m")
    cs  := &docker.CaptureStarter{}

    err := docker.StartLocal(docker.LocalRunOptions{
        PadDir:   pad.Dir,
        JavaHome: fakeJavaHome(t),  // 任意有效路径（bin/java 存在即可）
        Starter:  cs,
    })
    require.NoError(t, err)

    inv := cs.Invocations[0]
    assert.Contains(t, inv.Args, "-Xmx512m")
    assert.Contains(t, inv.Args, "-Dfile.encoding=UTF-8")
    assert.Contains(t, inv.Args, "-jar")
    // 确保 app/. 和 etc/Default 作为位置参数出现
    assert.Equal(t, inv.Args[len(inv.Args)-1], filepath.Join(pad.Dir, "etc", "Default"))
}
```

---

### 6.3 ComponentPayload（泛化 DaemonPayload，`mxcli-launcher/testfixtures/`）

现有 `DaemonPayload` + `BuildDaemonPayloadForPlatform` 泛化为 `ComponentPayload`，统一支持 daemon / local 两个组件。

```go
type ComponentPayload struct {
    AssetName string   // e.g. "mxcli-local-linux-amd64.tar.zst"
    Archive   []byte
    Checksum  string
}

// component = "mxcli-daemon" | "mxcli-local" | "mxcli"（launcher）
func BuildComponentPayload(component, goos, goarch string, content []byte) (*ComponentPayload, error)

// 向后兼容：现有 BuildDaemonPayload 委托到 BuildComponentPayload
func BuildDaemonPayload(content []byte) (*DaemonPayload, error) // 保留，内部转发
```

---

### 6.4 MultiReleaseFakeGitHub（扩展 FakeGitHub，`mxcli-launcher/testfixtures/`）

现有 `FakeGitHub` 只服务单一 `LatestTag`。新版支持多个 tag 系列，模拟 `/releases` 列表接口的 tag 前缀过滤。

```go
type ReleaseEntry struct {
    Tag     string
    Assets  map[string]*ComponentPayload // assetName → payload
}

// MultiReleaseFakeGitHub 扩展 FakeGitHub，支持多个 release 系列。
// 现有 FakeGitHub 字段（StatusCode、DownloadCut、CorruptBinary）继续有效。
type MultiReleaseFakeGitHub struct {
    FakeGitHub                        // 嵌入，保留 /releases/latest 兼容
    Releases   []ReleaseEntry         // 按时间降序（index 0 = 最新）
}

// /releases?per_page=N → 返回 Releases 的 JSON 数组
// /releases/latest     → 返回 Releases[0]（全局最新）
// SHA256SUMS           → 从 release 的 Assets 生成
```

**防退化价值**：`TestFetchLatestTagWithPrefix_MultiSeries` 验证 `daemon-v*` 不会拿到 `local-v*` 的 release，反之亦然。

---

### 6.5 PIDWaiter 接口（`cmd/mxcli-launcher/`）

Self-fork updater 的"等待父进程退出"逻辑通过此接口注入，消除定时器依赖。

```go
// PIDWaiter waits for a process to exit.
type PIDWaiter interface {
    WaitForExit(pid int, timeout time.Duration) error
}

// RealPIDWaiter: Windows = WaitForSingleObject; POSIX = kill(pid,0) 轮询
// FakePIDWaiter: channel 驱动，测试完全控制时序

type FakePIDWaiter struct {
    ExitC chan struct{} // close 触发"进程已退出"
}

func NewFakePIDWaiter() *FakePIDWaiter {
    return &FakePIDWaiter{ExitC: make(chan struct{})}
}

func (f *FakePIDWaiter) WaitForExit(pid int, timeout time.Duration) error {
    select {
    case <-f.ExitC:
        return nil
    case <-time.After(timeout):
        return fmt.Errorf("timeout waiting for PID %d", pid)
    }
}

func (f *FakePIDWaiter) SimulateExit() { close(f.ExitC) }
```

**典型测试**：
```go
func TestSelfForkUpdater_WaitsBeforeReplacing(t *testing.T) {
    waiter := testfixtures.NewFakePIDWaiter()
    oldBin := writeFile(t, "old-content")
    newBin := writeFile(t, "new-content")

    done := make(chan error, 1)
    go func() {
        done <- runInternalUpdate(99999, newBin, oldBin, waiter, 5*time.Second)
    }()

    // 父进程未退出 → 文件不应被替换
    time.Sleep(20 * time.Millisecond)
    assert.Equal(t, "old-content", readFile(t, oldBin))

    // 模拟父进程退出
    waiter.SimulateExit()
    require.NoError(t, <-done)
    assert.Equal(t, "new-content", readFile(t, oldBin))
}

func TestSelfForkUpdater_VerificationFail_Rollback(t *testing.T) {
    waiter := testfixtures.NewFakePIDWaiter()
    oldBin := writeFile(t, "good-content")
    newBin := writeFile(t, "bad-binary")  // 这个 binary 执行 `version` 会返回非零

    waiter.SimulateExit()
    err := runInternalUpdate(1, newBin, oldBin, waiter, time.Second)

    assert.Error(t, err)
    assert.Equal(t, "good-content", readFile(t, oldBin)) // 自动回滚
}
```

---

### 6.6 UpgradeHarness（端到端升级场景，`mxcli-launcher/testfixtures/`）

组合 MultiReleaseFakeGitHub + Env + temp HomeDir，提供一行式场景构建。

```go
type UpgradeHarness struct {
    Env *Env
    GH  *MultiReleaseFakeGitHub
    HomeDir string
}

// NewUpgradeHarness 构建包含 FakeGitHub 的完整 Env，注入 LocalBinaryResolver。
func NewUpgradeHarness(t *testing.T) *UpgradeHarness

// AddRelease 向 FakeGitHub 注册一个 release（daemon 或 local）。
func (h *UpgradeHarness) AddRelease(tag string, component string, content []byte) *UpgradeHarness

// InstalledVersion 读取 ~/.mxcli/{component}/version 文件。
func (h *UpgradeHarness) InstalledVersion(component string) string

// InstalledBinaryContent 读取已安装二进制的内容（用于断言"binary 确实被替换"）。
func (h *UpgradeHarness) InstalledBinaryContent(component string) []byte
```

**典型测试**：
```go
func TestUpgradeDaemon_FreshInstall(t *testing.T) {
    h := testfixtures.NewUpgradeHarness(t).
        AddRelease("daemon-v1.2.0", "mxcli-daemon", []byte("daemon-v1.2.0-binary"))

    err := h.Env.upgradeComponent(daemonComponentConfig)

    require.NoError(t, err)
    assert.Equal(t, "daemon-v1.2.0", h.InstalledVersion("daemon"))
    assert.Equal(t, []byte("daemon-v1.2.0-binary"), h.InstalledBinaryContent("daemon"))
}

func TestUpgradeLocal_TagPrefixIsolation(t *testing.T) {
    h := testfixtures.NewUpgradeHarness(t).
        AddRelease("daemon-v2.0.0", "mxcli-daemon", []byte("daemon")).
        AddRelease("local-v0.3.0", "mxcli-local", []byte("local-v030"))

    err := h.Env.upgradeComponent(localComponentConfig)

    require.NoError(t, err)
    // local upgrade 不应拿到 daemon release
    assert.Equal(t, "local-v0.3.0", h.InstalledVersion("local"))
    assert.Equal(t, []byte("local-v030"), h.InstalledBinaryContent("local"))
    // daemon 未被动到
    assert.Equal(t, "", h.InstalledVersion("daemon"))
}
```

---

### 6.7 测试矩阵（夹具 × 场景）

| 测试 | 夹具 | 防退化断言 |
|------|------|-----------|
| java 命令参数构造 | FakePAD + CaptureStarter | 每个 -D 标志均断言 |
| JVM heap 从 HOCON 读取 | FakePAD.SetJVMHeap + CaptureStarter | -Xmx 出现在 args |
| DB URL 注入环境变量 | FakePAD + CaptureStarter | RUNTIME_PARAMS_* 出现在 cmd.Env |
| PAD 缺失时的错误信息 | 空 tmpdir + CaptureStarter | error 含"mxcli local build" |
| 升级组件 - 新装 | UpgradeHarness.AddRelease | 版本文件 + 二进制内容 |
| 升级组件 - tag 前缀隔离 | UpgradeHarness 多 release | local 不拿 daemon release |
| 升级组件 - checksum 不符 | UpgradeHarness + CorruptBinary | 返回 error，原文件不变 |
| self-fork 等待父进程退出 | FakePIDWaiter | 替换时序精确控制 |
| self-fork 新版本验证失败回滚 | FakePIDWaiter + bad binary | 原文件恢复 |
| launcher 路由 local 子命令 | Env.LocalBinaryPath 注入 | 传递正确 args |
| 多系列 release 列表过滤 | MultiReleaseFakeGitHub | GET /releases 请求路径 |

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
