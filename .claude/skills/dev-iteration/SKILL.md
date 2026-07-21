# mxcli 开发迭代速查

修改任意子系统后选择最快验证路径的参考手册。

## 架构速览

```
mxcli (单一二进制: cmd/mxcli/)
  ├── cmd/mxcli/ — CLI 入口、executor、backend
  ├── mdl/executor/ — MDL 命令执行器
  ├── mdl/backend/mpr/ — BSON 读写（MprBackend）
  ├── internal/expr/ — 表达式检查
  └── cmd/mxcli-local/ — 本地 run/build 独立二进制
```

关键包：
- `cmd/mxcli/` — mxcli 主二进制
- `cmd/mdlrun/` — 无 CLI 框架的开发调试 runner
- `cmd/mxcli-local/` — 本地 run/build 独立二进制

---

## 快速路径表

| 修改了什么 | 最快验证路径 |
|-----------|------------|
| `mdl/executor/` executor 逻辑 | `go run ./cmd/mdlrun -p app.mpr -c "..."` |
| `mdl/executor/` + 需要集成测试 | `go test -tags integration ./mdl/executor/ -run TestXxx` |
| `mdl/backend/mpr/` BSON 序列化 | `go test -tags integration ./mdl/backend/mpr/ -run TestXxx` |
| `mdl/backend/mpr/` + 需要 mx check | `go test -tags integration ./internal/goldenfs/ -run TestHelpdeskGolden_Update -update-golden` → `mx check` |
| `cmd/mxcli/` 子命令逻辑 | `go run ./cmd/mxcli -p app.mpr -c "..."` |
| `cmd/mxcli-local/`、`cmd/mxcli/docker/local.go` | `go test -tags integration ./cmd/mxcli/docker/...` |
| `cmd/mxcli/tui/` TUI 代码 | `go run ./cmd/mxcli tui -p app.mpr` |
| `internal/expr/` 表达式检查 | `go test -tags integration ./internal/expr/...` |

---

## 场景详解

### 场景 A：修改 executor 逻辑（最常见）

**改了：** `mdl/executor/cmd_*.go`、`mdl/visitor/`、`mdl/ast/`

**最快路径：**
```bash
# 1. 直接跑 executor（毫秒级反馈）
go run ./cmd/mdlrun -p testdata/expr-checker/minimal.mpr -c "show entities"
go run ./cmd/mdlrun -p testdata/expr-checker/minimal.mpr script.mdl

# 2. 或跑对应集成测试
go test -tags integration ./mdl/executor/ -run TestShowEntities -v
```

`cmd/mdlrun` 直接调用 `executor.Build()` + `mprbackend.New()`，与完整 mxcli 走相同的代码路径，只是省去了 Cobra CLI 框架层。

---

### 场景 B：修改 BSON 序列化（`mdl/backend/mpr/`）

**改了：** `mdl/backend/mpr/*_compat.go`、`modelsdk/mpr/`、`modelsdk/codec/`

**最快路径（不需要 mx check）：**
```bash
go test -tags integration ./mdl/backend/mpr/ -run TestXxx -v
```

**需要 mx check 时：**
```bash
# 1. 重建 helpdesk golden（默认版本 11.12.1，可设 HELPDESK_VERSION 覆盖）
HELPDESK_VERSION=11.12.1 \
CGO_ENABLED=0 go test ./internal/goldenfs/ \
  -tags linux,integration \
  -run '^TestHelpdeskGolden_Update$' \
  -update-golden \
  -v -timeout 10m

# 2. mx check 验证
~/.mxcli/mxbuild/11.12.1/modeler/mx check \
  testdata/helpdesk-golden-11.12.1/minimal.mpr 2>&1 | grep "\[error\]"

# 3. 还原 testdata
git restore testdata/helpdesk-golden-11.12.1/ testdata/helpdesk-golden-11.6.6/ testdata/helpdesk-golden-11.10.0/
git clean -fd testdata/
```

---

### 场景 C：调试 `cmd/mxcli` 子命令针对外部项目

**改了：** `cmd/mxcli/cmd_git.go`、`cmd/mxcli/cmd_*.go` 中需要真实 git 仓库上下文的命令

```bash
PROJECT=/path/to/mendix-project
GIT_DIR=$PROJECT/.git \
GIT_WORK_TREE=$PROJECT \
go run ./cmd/mxcli git doctor -p $PROJECT/app.mpr
```

---

### 场景 D：BSON 修改 + 重建 helpdesk golden + mx check

**改了：** `mdl/backend/mpr/`、`mdl/executor/`，需要对完整 helpdesk 项目做 mx check 验证

```bash
# 1. 重建 golden
HELPDESK_VERSION=11.6.6 \
CGO_ENABLED=0 go test ./internal/goldenfs/ \
  -tags linux,integration \
  -run '^TestHelpdeskGolden_Update$' \
  -update-golden \
  -timeout 10m

# 可并行多个版本：
for v in 11.12.1 11.6.6 11.10.0; do
  HELPDESK_VERSION=$v CGO_ENABLED=0 go test ./internal/goldenfs/ \
    -tags linux,integration -run '^TestHelpdeskGolden_Update$' \
    -update-golden -timeout 10m &
done
wait

# 2. mx check
for v in 11.12.1 11.6.6 11.10.0; do
  echo "=== $v ==="
  ~/.mxcli/mxbuild/$v/modeler/mx check testdata/helpdesk-golden-$v/minimal.mpr 2>&1 | \
    grep "\[error\]" | wc -l
done

# 3. 还原
git restore testdata/helpdesk-golden-11.12.1/ testdata/helpdesk-golden-11.6.6/ testdata/helpdesk-golden-11.10.0/
git clean -fd testdata/
```

**注意：** `TestHelpdeskGolden_Update` 直接在 `go test` 进程内调用 executor，不走外部进程。源码改动即时生效，无需 `make build`。

---

### 场景 E：迭代 TUI 代码

**改了：** `cmd/mxcli/tui/*.go`、`cmd/mxcli/cmd_tui.go`

```bash
go run ./cmd/mxcli tui -p /path/to/app.mpr
```

---

### 场景 F：修改 `mxcli local` 命令

**改了：** `cmd/mxcli-local/cmd_run.go`、`cmd/mxcli-local/cmd_build.go`、`cmd/mxcli/docker/local.go`

```bash
# 单元测试
go test -tags integration ./cmd/mxcli/docker/... -v

# 端到端冒烟（需要真实 .mpr + 已构建的 PAD）
go run ./cmd/mxcli-local run -p /path/to/app.mpr --port 8081 --admin-port 8091
```

`cmd/mxcli-local` 是独立二进制，不经过 mxcli 主进程，直接 `go run` 即可。

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
