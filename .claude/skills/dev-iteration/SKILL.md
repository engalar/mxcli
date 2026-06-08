# mxcli 开发迭代速查

修改任意子系统后选择最快验证路径的参考手册。每次发现新场景时在下方追加。

## 架构速览

```
mxcli (launcher)  →  Unix socket  →  mxcli-daemon (cmd/mxcli/)
                                           ↑
                          socket 路径 = hash(mprPath) + hash(binMtime)
                          二进制更新 → 旧 socket 自动失效 → 下次请求重新启动
```

关键包：
- `cmd/mxcli/` — daemon 主体（executor、backend、daemon_server）
- `cmd/mxcli-launcher/` — 薄 launcher（socket 路由、生命周期、升级）
- `cmd/mdlrun/` — 无 daemon 的开发调试 runner

---

## 快速路径表

| 修改了什么 | 最快验证路径 | 是否需要 install-daemon |
|-----------|------------|----------------------|
| `mdl/executor/` executor 逻辑 | `go run ./cmd/mdlrun -p app.mpr -c "..."` | **不需要** |
| `mdl/executor/` + 需要 flag/socket 路径 | `go test ./mdl/executor/... -run TestXxx` | **不需要** |
| `mdl/backend/mpr/` BSON 序列化 | `go test ./mdl/backend/mpr/... -run TestGolden -update` | **不需要** |
| `mdl/backend/mpr/` + 需要 mx check 确认 | `go test ./internal/goldenfs/ -tags linux,integration -run TestHelpdeskGolden_Update -update-golden` → `mx check` → `git restore testdata/` | **不需要** |
| `cmd/mxcli/daemon_server.go` socket 协议 | `go test ./cmd/mxcli/... -run TestDaemonServer` | **不需要** |
| `cmd/mxcli-launcher/` launcher 路由/升级 | `go test ./cmd/mxcli-launcher/...` (fake_daemon fixture) | **不需要** |
| `internal/expr/daemon/` expr daemon | `go test ./internal/expr/daemon/...` | **不需要** |
| `cmd/mxcli/cmd_git.go` git 子命令针对外部项目 | `GIT_DIR=$PROJECT/.git GIT_WORK_TREE=$PROJECT go run ./cmd/mxcli git doctor -p $PROJECT/app.mpr` | **不需要** |
| `cmd/mxcli/tui/` TUI 代码 | `go run ./cmd/mxcli tui -p app.mpr` | **不需要** |
| 任意改动，需要完整 mxcli 路径端到端确认 | `make install-daemon && mxcli -p app.mpr -c "..."` | **需要** |
| 跨 section BSON 状态传播（回归风险高） | `make test-section-check` | **需要** |

---

## 场景详解

### 场景 A：修改 executor 逻辑（最常见）

**改了：** `mdl/executor/cmd_*.go`、`mdl/visitor/`、`mdl/ast/`

**最快路径：**
```bash
# 1. 直接跑 executor（绕过 daemon/socket，毫秒级反馈）
go run ./cmd/mdlrun -p testdata/expr-checker/minimal.mpr -c "show entities"
go run ./cmd/mdlrun -p testdata/expr-checker/minimal.mpr script.mdl

# 2. 或跑对应单元测试（最快，不需要 .mpr 文件）
go test ./mdl/executor/... -run TestShowEntities -v
```

**为什么 `mdlrun` 够用：** 它直接调用 `executor.Build()` + `mprbackend.New()`，与 daemon 走的是完全相同的代码路径，只是省去了 socket 序列化层。

**何时还是要 `install-daemon`：** 需要验证 socket 帧格式、并发串行化、idle timeout 行为时。

---

### 场景 B：修改 BSON 序列化（`mdl/backend/mpr/`）

**改了：** `mdl/backend/mpr/*_compat.go`、`modelsdk/mpr/`、`modelsdk/codec/`

**最快路径（不需要 mx check）：**
```bash
# Golden 文件测试：对比生成的 BSON 与已知正确快照
go test ./mdl/backend/mpr/... -run TestGolden -v
# 更新快照（确认改动符合预期后）：
go test ./mdl/backend/mpr/... -run TestGolden -update
```

**需要 mx check 时（有潜在 CE 错误风险）—— 无需 install-daemon：**
```bash
# 1. 重建 helpdesk golden（go test 直接在进程内跑 executor，不走 daemon/socket）
HELPDESK_VERSION=11.6.6 \
CGO_ENABLED=0 go test ./internal/goldenfs/ \
  -tags linux,integration \
  -run '^TestHelpdeskGolden_Update$' \
  -update-golden \
  -v -timeout 10m

# 2. mx check 验证（不得引入新 CE 错误）
~/.mxcli/mxbuild/11.6.6/modeler/mx check \
  testdata/helpdesk-golden-11.6.6/minimal.mpr 2>&1 | grep "\[error\]"
# 应输出 0 行；原始 git 基线是 0 errors

# 3. 还原 testdata
git restore testdata/helpdesk-golden-11.6.6/ testdata/helpdesk-golden-11.10.0/
git clean -fd testdata/

# 也可以仅测试单个小项目（更快，但 CE0463 不一定能复现）：
go run ./cmd/mdlrun -p testdata/expr-checker/minimal.mpr \
  -c "create microflow MyFirstModule.ACT_Test () returns Nothing begin return; end;"
~/.mxcli/mxbuild/11.6.6/modeler/mx check testdata/expr-checker/minimal.mpr 2>&1 | grep "\[error\]"
git restore testdata/expr-checker/
```

**关键原因**：`TestHelpdeskGolden_Update` 调用 `executor.ExecuteProgram` + `mprbackend.New()` 完全在进程内，与 daemon 代码路径完全相同，不经 socket 层。`install-daemon` 仅在需要验证 socket 帧格式、并发序列化时才必要。

---

### 场景 C：修改 daemon server（socket 协议、并发控制）

**改了：** `cmd/mxcli/daemon_server.go`、`cmd/mxcli/daemon_backend.go`

**最快路径（in-process，无需安装）：**
```bash
go test ./cmd/mxcli/... -run TestDaemonServer -v
```

`TestDaemonServer_HealthCheck` 和 `TestDaemonServer_IdleTimeout` 直接在测试进程内启动真实的 `runDaemonServer`，通过 unix socket 发请求。无需任何外部进程。

---

### 场景 D：修改 launcher（路由、升级、生命周期）

**改了：** `cmd/mxcli-launcher/*.go`

**最快路径（fake_daemon fixture，无需真实 daemon）：**
```bash
go test ./cmd/mxcli-launcher/... -v
```

`testfixtures/fake_daemon.go` 提供假的 daemon 归档包（tar.zst/zip），`testfixtures/fake_github.go` 提供假的 GitHub Release API，完全不依赖网络或真实 daemon 二进制。

---

### 场景 E：修改 expr daemon（`internal/expr/daemon/`）

**改了：** `internal/expr/daemon/daemon.go`、`socket.go`、`proto.go`

**最快路径：**
```bash
go test ./internal/expr/daemon/... -v
```

---

### 场景 F：端到端回归测试（跨 section 状态传播）

**触发条件：** 修改了 entity 解析、return type 缓存、executor session 状态等。

**路径（慢，需要 mxbuild）：**
```bash
make test-section-check
```

该命令逐 `-- MARK:` section 执行 helpdesk-app.mdl，每个 section 用独立的 `mxcli exec` 进程，最后跑 `mx check` 确认错误数不超过基线。

---

## install-daemon 的工作原理

```bash
make install-daemon
# = make build
#   + pkill mxcli-daemon（旧进程）
#   + cp bin/mxcli-daemon ~/.mxcli/daemon/mxcli-daemon
#   + cp bin/mxcli <launcher-in-PATH>  (如果 PATH 里有 mxcli)
```

**为什么不需要手动清理旧 socket：** socket 路径含 `hash(binMtime)`，二进制更新后 mtime 变化，launcher 自动计算新路径并启动新 daemon；旧 socket 无人连接，30 分钟 idle timeout 后旧进程自退出。

**安装后第一次请求会有 ~100-300ms 冷启动**（新 daemon 进程启动 + SQLite 打开）。

---

## 调试 daemon 进程本身的日志

默认情况下 daemon 的 stdout/stderr 被丢弃（`cmd.Stderr = nil`）。临时查看日志：

```bash
# 手动启动 daemon，终端可见日志
~/.mxcli/daemon/mxcli-daemon --serve /tmp/debug.sock --idle-timeout 30m &

# 用自定义 socket 发请求（需绕过 launcher）
# 目前 mxcli launcher 不支持 MXCLI_DAEMON_SOCK env var — 需直接用 nc 或写 Go 测试
```

**已知缺失：** 没有 `MXCLI_DAEMON_BIN` env var 让 launcher 直接用 `./bin/mxcli-daemon` 而不走 `~/.mxcli/daemon/` 路径。如果实现了，可省去 install-daemon 步骤。

---

### 场景 G：调试 `cmd/mxcli` 子命令（git doctor/fix/commit）针对外部项目

**改了：** `cmd/mxcli/cmd_git.go`、`cmd/mxcli/cmd_*.go` 中需要真实 git 仓库上下文的命令

**核心问题：** `go run ./cmd/mxcli` 必须在 mxcli 源码目录执行（go.mod 所在处），但命令内部的 `git` 子进程继承进程 CWD，默认会操作 mxcli 源码仓库而非目标项目。

**解决方案：** 用 `GIT_DIR` + `GIT_WORK_TREE` 环境变量重定向 git 上下文：

```bash
PROJECT=/path/to/mendix-project
GIT_DIR=$PROJECT/.git \
GIT_WORK_TREE=$PROJECT \
go run ./cmd/mxcli git doctor -p $PROJECT/app.mpr

# 实际示例（调试客户项目）：
PROJECT=/mnt/data_sdd/jack-mom-platform-feature_1.0.0_2
GIT_DIR=$PROJECT/.git \
GIT_WORK_TREE=$PROJECT \
go run ./cmd/mxcli git doctor -p $PROJECT/jack-mom-platform.mpr

# git fix（修复缺失 notes）：
GIT_DIR=$PROJECT/.git \
GIT_WORK_TREE=$PROJECT \
go run ./cmd/mxcli git fix -p $PROJECT/jack-mom-platform.mpr
```

**注意事项：**
- `GIT_DIR` 让 git 找到 `.git` 元数据，`GIT_WORK_TREE` 让 git 知道工作树根路径，二者缺一不可
- `go run` 的编译仍在 mxcli 源码目录完成，首次运行有编译耗时（约 3-10s），后续热缓存很快
- 不需要 `make build` 或 `install-daemon`，修改源码后直接重跑即可

**何时还是要 `install-daemon`：** 需要验证 `mxcli git notes push` 等涉及网络推送的操作时（因为需要真实的 mxcli launcher 路由）。

---

### 场景 H：BSON 修改 + 重建 helpdesk golden + mx check（无需 install-daemon）

**改了：** `mdl/backend/mpr/`、`mdl/executor/`，需要对完整 helpdesk 项目做 mx check 验证

**核心洞察：** `TestHelpdeskGolden_Update` 直接在 `go test` 进程内调用 executor，与 daemon 完全相同的代码路径，不走 socket。无需安装 daemon。

**最快路径：**
```bash
# 1. 重建 golden（约 40s，in-process executor）
HELPDESK_VERSION=11.6.6 \
CGO_ENABLED=0 go test ./internal/goldenfs/ \
  -tags linux,integration \
  -run '^TestHelpdeskGolden_Update$' \
  -update-golden \
  -timeout 10m

# 可并行两个版本（约省 40s）：
for v in 11.6.6 11.10.0; do
  HELPDESK_VERSION=$v CGO_ENABLED=0 go test ./internal/goldenfs/ \
    -tags linux,integration -run '^TestHelpdeskGolden_Update$' \
    -update-golden -timeout 10m &
done
wait

# 2. mx check（原始 git 基线 = 0 errors）
for v in 11.6.6 11.10.0; do
  echo "=== $v ==="
  ~/.mxcli/mxbuild/$v/modeler/mx check testdata/helpdesk-golden-$v/minimal.mpr 2>&1 | \
    grep "\[error\]" | wc -l
done

# 3. 还原（避免污染工作区）
git restore testdata/helpdesk-golden-11.6.6/ testdata/helpdesk-golden-11.10.0/
git clean -fd testdata/
```

**注意事项：**
- `TestHelpdeskGolden_Update` 调用 `e.SetBackendFactory(func() backend.FullBackend { return mprbackend.New() })`，完全 in-process，无 socket 层
- 重建时 `go test` 会重新编译所有依赖包——源码改动即时生效，无需 `make build`
- golden 重建后 mx check 预期 **0 errors**（CE0463 会在重建时出现 5 个，是 mxcli widget template 与 Studio Pro 差异的预存在问题，不是代码引入的）
- 如果要验证真实 daemon socket 路径（如 `BeginPageBuild` 跨请求状态），才需要 install-daemon

---

### 场景 I：迭代 TUI 代码

**改了：** `cmd/mxcli/tui/*.go`、`cmd/mxcli/cmd_tui.go`

**最快路径：**
```bash
go run ./cmd/mxcli tui -p /path/to/app.mpr
```

**原理：** `cmd/mxcli/` 是 daemon 二进制（`package main`），包含完整的 `tuiCmd`。正常路径是 launcher 检测到 TTY 命令后 exec daemon 二进制——`go run ./cmd/mxcli` 直接跳过这一步，效果相同。

**注意事项：**
- `MXCLI_LAUNCHER_PATH` 未设置，TUI 内部的 `resolveMxcliPath()` fallback 到 `os.Executable()`（go run 的临时二进制）。TUI 发起的子命令（project-tree、describe 等）每次都是新进程 + 新 SQLite 连接，没有 per-MPR daemon 缓存，但功能正确
- 修改 TUI 源码后直接重跑即可，无需 `make build`
- 需要验证 TUI ↔ per-MPR daemon 交互（持久连接、unit cache）时，才需要 `make install-daemon`

---

## 添加新场景的规范

在本文件末尾按以下格式追加：

```markdown
### 场景 X：<简短描述>

**改了：** <相关文件路径或包>

**最快路径：**
```bash
<验证命令>
```

**注意事项：** <坑、前提条件、限制>
```

同时在「快速路径表」中补一行。
