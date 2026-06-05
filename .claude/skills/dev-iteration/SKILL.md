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
| `mdl/backend/mpr/` + 需要 mx check 确认 | `make install-daemon` → `mxcli -p testdata/...` → `mx check` | **需要** |
| `cmd/mxcli/daemon_server.go` socket 协议 | `go test ./cmd/mxcli/... -run TestDaemonServer` | **不需要** |
| `cmd/mxcli-launcher/` launcher 路由/升级 | `go test ./cmd/mxcli-launcher/...` (fake_daemon fixture) | **不需要** |
| `internal/expr/daemon/` expr daemon | `go test ./internal/expr/daemon/...` | **不需要** |
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

**需要 mx check 时（有潜在 CE 错误风险）：**
```bash
# 1. 应用改动
make install-daemon
./bin/mxcli -p testdata/expr-checker/minimal.mpr -c "create microflow MyFirstModule.ACT_Test () returns Nothing begin return; end;"

# 2. mx check 验证（不得引入新 StorageLoadException）
~/.mxcli/mxbuild/11.6.6/modeler/mx check testdata/expr-checker/minimal.mpr 2>&1 | grep -i "StorageLoadException\|Invalid"

# 3. 还原 testdata
git restore testdata/expr-checker/
```

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
