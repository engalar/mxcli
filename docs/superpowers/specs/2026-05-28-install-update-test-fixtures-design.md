# 设计：mxcli 安装与更新测试夹具

**日期**：2026-05-28  
**状态**：已批准  
**范围**：`cmd/mxcli-launcher/`

## 背景

mxcli 采用 launcher-daemon 分离架构：轻量级 launcher 通过 Unix socket 转发请求，daemon 在后台运行。升级时 launcher 从 GitHub releases 下载新 daemon，SHA256 校验后原子替换，并保留 N-1 版本备份以支持回滚。

现有单元测试覆盖了解压、路径、校验和解析等函数，但缺少覆盖完整安装/升级生命周期的集成测试，也缺少并发竞争和失败恢复场景的测试。

## 目标

- 两层覆盖：组件级（函数注入）+ 端到端（假 GitHub API + 进程内执行）
- 全部测试在 Go 进程内运行，无需 Docker，用 `t.TempDir()` 隔离文件系统
- 可注入所有外部依赖（HTTP client、HOME 目录），不依赖真实网络或真实 `~/.mxcli/`
- 并发升级场景可断言：恰好一个 goroutine 成功，文件状态一致

## 方案选择

选择**方案 A：Config 注入 + 薄接口层**。

- 改动仅限 `cmd/mxcli-launcher/` 包内部
- 不暴露新的公共 API，不引入新的外部依赖
- 现有 `*_test.go` 只需微调函数签名即可兼容

## 设计

### 1. `Env` 结构体（重构范围）

新建 `cmd/mxcli-launcher/env.go`：

```go
type Env struct {
    HomeDir    string       // 替代 os.UserHomeDir()
    HTTPClient *http.Client // 替代 http.DefaultClient
}

func DefaultEnv() *Env {
    home, _ := os.UserHomeDir()
    return &Env{HomeDir: home, HTTPClient: http.DefaultClient}
}
```

`paths.go` 中的路径函数改为 `(*Env)` 方法（`daemonDir()`、`daemonBin()` 等）。`daemon.go` 和 `update.go` 中所有 HTTP 调用和路径访问改为通过 `*Env`。`main.go` 启动时调用 `DefaultEnv()`，其余逻辑传 `*Env`。

### 2. `FakeGitHub` 测试服务器

新建 `cmd/mxcli-launcher/testfixtures/fake_github.go`（包名 `testfixtures`，仅供测试使用）：

```go
type FakeGitHub struct {
    LatestTag    string  // 返回的最新版本号，e.g. "v0.15.0"
    Checksum     string  // SHA256SUMS 中的正确值（空则自动计算）
    CorruptBinary bool   // true → 下载内容为垃圾字节（触发校验失败）
    DownloadCut  int     // 下载到第 N 字节后断开连接（0 = 不截断）
    StatusCode   int     // 非 0 则所有接口返回此状态码（模拟 500）
    server       *httptest.Server
    payload      []byte  // 预生成的假 daemon tar 包
}

func NewFakeGitHub(t *testing.T) *FakeGitHub  // t.Cleanup 自动关闭服务器
func (f *FakeGitHub) Client() *http.Client    // 返回重定向到假服务器的 HTTP client
func (f *FakeGitHub) RequestLog() []string    // 记录被调用的路径，用于断言
```

服务器实现与真实 GitHub releases API 路径一致的 3 个路由：

| 路由 | 返回内容 |
|------|---------|
| `GET /repos/.../releases/latest` | 含 `tag_name` 的 JSON |
| `GET /releases/download/{tag}/SHA256SUMS` | 校验和文件 |
| `GET /releases/download/{tag}/mxcli-daemon-{os}-{arch}.tar.zst` | 假 daemon 压缩包 |

`Client()` 返回的 `*http.Client` 使用自定义 `http.Transport`，将所有发往 `api.github.com` 和 `github.com` 的请求重定向到假服务器，生产代码的 URL 常量无需修改。

`fake_daemon.go` 生成最小假 daemon payload（标准库 `archive/tar`，不依赖 zstd）。SHA256 在 `NewFakeGitHub` 时自动计算，`CorruptBinary: true` 时写入错误值。

### 3. 测试场景矩阵

新建 `cmd/mxcli-launcher/install_update_test.go`，`t.Parallel()` 全开，每个 `t.Run` 独立 `t.TempDir()` + 独立 `FakeGitHub`。

**核心路径**

| 场景 | 断言 |
|------|------|
| 全新安装（daemon 目录不存在） | daemon 二进制写入，`version` 文件内容正确 |
| 版本已最新 | `RequestLog` 不含下载路径，跳过下载 |
| 旧版升级到新版 | 旧 binary 移到 `.bak`，新 binary 就位，`version` / `version.bak` 均正确 |

**失败恢复**

| 场景 | 断言 |
|------|------|
| SHA256 校验失败（`CorruptBinary: true`） | 返回错误，原 daemon 和 `version` 文件不变 |
| 下载中断（`DownloadCut: 1024`） | 返回错误，临时文件被清理 |
| GitHub 500（`StatusCode: 500`） | 返回错误，不修改任何现有文件 |
| 新 daemon 无法执行 | `runUpgrade` 失败后触发 `runRollback`，`.bak` 文件恢复 |

**并发竞争**

| 场景 | 断言 |
|------|------|
| 两个 goroutine 同时调用 `runUpgrade()` | 恰好一个返回 `nil`，另一个返回"upgrade in progress"；最终文件状态一致 |

**后台检查节流**

| 场景 | 断言 |
|------|------|
| 距上次检查不足 1 小时 | `shouldCheckUpdate()` 返回 false，无 HTTP 请求 |
| `last-check` 不存在（首次运行） | 允许检查，写入新时间戳 |

### 4. 并发锁机制

`update.go` 中 `runUpgrade()` 加文件锁：

```go
func (e *Env) runUpgrade() error {
    lockPath := filepath.Join(e.daemonDir(), "upgrade.lock")
    lf, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
    if err != nil {
        return err
    }
    defer lf.Close()
    if err := flock(lf, true); err != nil { // 非阻塞 exclusive lock
        return fmt.Errorf("upgrade in progress")
    }
    defer funlock(lf)
    // ... 原有升级逻辑
}
```

平台差异通过 build tag 分文件处理：

- `lock_unix.go`（`//go:build !windows`）：`syscall.Flock` + `LOCK_EX|LOCK_NB`
- `lock_windows.go`（`//go:build windows`）：`LockFileEx` with `LOCKFILE_FAIL_IMMEDIATELY`

### 5. 文件布局

```
cmd/mxcli-launcher/
├── env.go                    # 新增：Env + DefaultEnv()
├── lock_unix.go              # 新增：flock（linux/darwin）
├── lock_windows.go           # 新增：LockFileEx（windows）
├── daemon.go                 # 改动：函数加 *Env 参数
├── update.go                 # 改动：函数加 *Env 参数，加锁
├── paths.go                  # 改动：改为 (*Env) 方法
├── main.go                   # 改动：启动时 DefaultEnv()
│
├── testfixtures/
│   ├── fake_github.go        # 新增：FakeGitHub 服务器
│   └── fake_daemon.go        # 新增：假 daemon tar 生成
│
├── install_update_test.go    # 新增：完整场景矩阵
├── daemon_test.go            # 已有，签名微调
├── update_test.go            # 已有，签名微调
└── paths_test.go             # 已有，签名微调
```

`testfixtures/` 为独立包，仅供 `_test.go` 引用，不进入生产构建，不引入新的外部依赖。

## 不在范围内

- launcher 进程自身退出码的端到端测试（需 `os/exec`，留待后续）
- `mxcli setup mxcli` 命令（`cmd/mxcli/setup.go`）的测试夹具
- Windows 平台的 `.zip` 解压路径测试（逻辑已有，结构相同）
