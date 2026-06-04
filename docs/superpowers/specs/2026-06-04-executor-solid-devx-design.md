# Executor SOLID 重构 + 开发体验设计

**日期**：2026-06-04  
**状态**：待实施  
**背景**：CLI 改动后必须走完整 daemon→socket→cobra 链路才能验证，根本原因是 SOLID 违规——executor 工厂函数 `newLoggedExecutor` 住在 `main` 包，外部测试 import 不进来，强制端到端测试。

---

## 使用场景（设计锚点）

五个场景，所有设计均从此推导：

```go
// 场景 1：单元测试（外部包，mock backend）
te := testutil.New(t)
te.Mock.ListModulesFunc = func() ([]*model.Module, error) { ... }
out, err := te.Run("show modules")
assert.Contains(t, out, "MyModule")

// 场景 2：独立开发工具（零 daemon）
// go run ./cmd/mdlrun -p testdata/corpus-b/app.mpr -c "show entities"
// go run ./cmd/mdlrun -p testdata/corpus-b/app.mpr script.mdl

// 场景 3：集成测试（真实 MPR 文件）
te := testutil.NewWithProject(t, "../../testdata/corpus-b/app.mpr")
out, _ := te.Run("show entities")
assert.Contains(t, out, "Account")

// 场景 4：Cobra 命令内部（daemon 模式）
exec, logger := buildExec("exec", out)  // 替换 newLoggedExecutor
defer logger.Close(); defer exec.Close()

// 场景 5：FUSE 黄金 MPR 测试
//go:embed testdata/golden-blank.mpr
var blankMPR []byte
te := testutil.NewWithMPRBytes(t, blankMPR)
te.Run("create entity Sales.Order (Name: string)")
testutil.AssertGoldenMPR(t, "testdata/golden-create-entity.mpr", te.MPRBytes())
// AssertGoldenMPR = mx check 验合法性 + 提取目标 BSON 文档对比
```

---

## 现有测试基础设施分析

### 现状：内部 `newMockCtx` 模式（保留）

```go
// mdl/executor/mock_test_helpers_test.go — 现有机制，不动
ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
assertNoError(t, listConstants(ctx, ""))
assertContainsStr(t, buf.String(), "MyModule.AppURL")
```

这套 `ExecContext` 级别的测试在 `package executor` 内运行良好，约 15 行设置一个测试，不需要替换。

### 真正的痛点

| 痛点 | 位置 | 影响 |
|------|------|------|
| `newLoggedExecutor` 在 `main` 包 | `cmd/mxcli/main.go` | 外部包无法 import，只能端到端测 |
| 写路径无闭环验证 | 全部写操作 | 必须 `go:build integration` + mxbuild |
| `persistentDaemonBackend` 依赖具体类型 | `daemon_backend.go` | DIP 违规，`noOpConnectBackend` 嵌入 `*mprbackend.MprBackend` |
| 无独立运行入口 | — | 改完代码必须 `make install-daemon` 才能测试 |

---

## 设计一：`executor.Builder`

**文件**：`mdl/executor/builder.go`（新建）

```go
type Builder struct {
    out      io.Writer
    backend  backend.FullBackend  // 已连接（mock/daemon/fuse）
    factory  BackendFactory        // 懒初始化（普通 CLI）
    progress io.Writer             // 实时进度流（独立于 stdout）
    logger   *diaglog.Logger
    format   OutputFormat
    quiet    bool
}

func Build() *Builder { return &Builder{out: os.Stdout} }

func (b *Builder) Out(w io.Writer) *Builder                { b.out = w; return b }
func (b *Builder) WithBackend(be backend.FullBackend) *Builder { b.backend = be; return b }
func (b *Builder) WithFactory(f BackendFactory) *Builder   { b.factory = f; return b }
func (b *Builder) ProgressOut(w io.Writer) *Builder        { b.progress = w; return b }
func (b *Builder) WithLogger(l *diaglog.Logger) *Builder   { b.logger = l; return b }
func (b *Builder) Format(f OutputFormat) *Builder          { b.format = f; return b }
func (b *Builder) Quiet() *Builder                         { b.quiet = true; return b }

func (b *Builder) Create() *Executor {
    e := New(b.out)
    if b.backend != nil {
        e.SetBackend(b.backend)
    } else if b.factory != nil {
        e.SetBackendFactory(b.factory)
    }
    if b.progress != nil { e.SetProgressOut(b.progress) }
    if b.logger != nil   { e.SetLogger(b.logger) }
    if b.format != ""    { e.SetFormat(b.format) }
    e.SetQuiet(b.quiet)
    return e
}
```

**依赖方向**：`Builder` 只 import `backend`（接口包），不 import `mprbackend`（具体实现）。`mprbackend.New()` 由调用方通过 `WithFactory` 传入。

---

## 设计二：DIP 修复（`PersistentBackend` 接口）

**文件**：`mdl/backend/persistent.go`（新建）

### 问题

```go
// 当前 DIP 违规
var persistentDaemonBackend *mprbackend.MprBackend      // 依赖具体类型
type noOpConnectBackend struct{ *mprbackend.MprBackend } // 嵌入具体类型

// executor 内部 duck-type（私有 sub-interface）
type microflowsRepoProvider interface { Microflows() *repo.Repository }
if rp, ok := b.(microflowsRepoProvider); ok { ... }
```

### 修复

```go
// mdl/backend/persistent.go
// PersistentBackend 是 daemon 持久连接模式下的扩展接口。
// executor 可类型断言到此接口获取 repo 级别访问。
type PersistentBackend interface {
    FullBackend
    Microflows() *microflowrepo.Repository
    Nanoflows() *nanoflowrepo.Repository
    // 其余 executor 内部 duck-type 涉及的所有方法
}

// mdl/backend/mpr/ 中加编译期检查
var _ backend.PersistentBackend = (*MprBackend)(nil)
```

```go
// cmd/mxcli/daemon_backend.go — 修复后
var persistentDaemonBackend backend.PersistentBackend    // 接口类型 ✓

type noOpConnectBackend struct{ backend.PersistentBackend } // 嵌入接口 ✓
func (n *noOpConnectBackend) Connect(string) error { return nil }
func (n *noOpConnectBackend) Disconnect() error    { return nil }
func (n *noOpConnectBackend) IsConnected() bool    { return true }
```

```go
// executor 内部断言改为公开接口
if rp, ok := b.(backend.PersistentBackend); ok { ... }   // ✓
```

---

## 设计三：Progress 流式传输

### 问题

协议层已支持流式（`frameWriter` 每次 `Write` 立即发帧），但语义层有缓冲：`show entities` 等命令先收集全部数据再渲染，AI 无法看到执行中进度。

### 协议扩展

```go
// internal/launcherproto/proto.go — 新增第三条 stream
type Frame struct {
    Stream  string // "stdout" | "stderr" | "progress"（新增）
    Data    []byte
    Exit    *int
    OK      bool
    Version string
}
```

```go
// cmd/mxcli-launcher/forward.go — 收到 progress 帧立即打印
case frame.Stream == "progress":
    fmt.Fprintf(err, "▶ %s\n", frame.Data)
```

### Executor 层

```go
// mdl/executor/executor.go
type Executor struct {
    out      io.Writer  // stdout：结果/表格
    progress io.Writer  // progress 帧：实时进度，独立于 tabwriter 缓冲
    // ...
}

func (e *Executor) Progress(msg string) {
    if e.progress != nil {
        fmt.Fprintln(e.progress, msg)
    }
}
```

### Daemon 配线

```go
// cmd/mxcli/serve.go — handleConn
outW      := &frameWriter{conn: conn, stream: "stdout"}
errW      := &frameWriter{conn: conn, stream: "stderr"}
progressW := &frameWriter{conn: conn, stream: "progress"}  // 新增

exec := executor.Build().
    Out(outW).
    ProgressOut(progressW).
    WithLogger(logger).
    Create()
```

### 效果

```
# mxcli exec large-script.mdl
▶ connecting to app.mpr...
▶ executing: create entity Sales.Order
▶ executing: create microflow Sales.ACT_CreateOrder
▶ commit...
Name        Type          ← stdout 结果最后渲染
Order       entity
```

---

## 设计四：`executor/testutil`（外部包测试）

**文件**：`mdl/executor/testutil/testutil.go`（新包）

### 定位

**不替代** `newMockCtx` + `ExecContext` 内部单测模式。补齐外部包和集成场景：

| 层次 | 机制 | 保留/新增 |
|------|------|---------|
| 内部 handler 单测 | `newMockCtx` (ExecContext 级) | 保留 ✓ |
| 外部包 / 公共 API 测 | `testutil.New(t)` | **新增** |
| 写路径黄金验证 | `testutil.NewWithMPRBytes` | **新增** |

### 类型

```go
type TestExec struct {
    Mock *mock.MockBackend  // 非 nil 仅 New(t) 构造
    t    *testing.T
    exec *executor.Executor
    buf  *strings.Builder
    fuse *fuseMount         // 非 nil 仅 NewWithMPRBytes 构造
}

// 通过公共 Executor.Execute() 接口测试（parse→execute 完整链路）
func New(t *testing.T) *TestExec
func NewWithProject(t *testing.T, mprPath string) *TestExec
func NewWithMPRBytes(t *testing.T, mprBytes []byte) *TestExec

// Run 执行 MDL，失败则 t.Fatal；返回 (stdout, nil)
func (te *TestExec) Run(mdl string) (string, error)

// MPRBytes 返回 FUSE 内存层当前字节；非 FUSE 模式 panic
func (te *TestExec) MPRBytes() []byte

// 顶层黄金断言
func AssertGoldenMPR(t *testing.T, goldenPath string, got []byte)
// 1. 写临时文件 → mx check（无新错误）
// 2. 提取目标 BSON 文档与快照对比
// 3. MXCLI_UPDATE_GOLDEN=1 时更新快照
```

### `New(t)` 内部实现

```go
func New(t *testing.T) *TestExec {
    m := &mock.MockBackend{
        IsConnectedFunc: func() bool { return true },
    }
    var buf strings.Builder
    exec := executor.Build().Out(&buf).WithBackend(m).Quiet().Create()
    t.Cleanup(func() { exec.Close() })
    return &TestExec{Mock: m, t: t, exec: exec, buf: &buf}
}
```

---

## 设计五：FUSE 内存挂载（场景 5）

**文件**：`mdl/executor/testutil/fuse.go`

```go
// MPRMount 把 []byte 挂载为 FUSE 虚拟文件，backend.Connect(path) 透明。
// 库：github.com/hanwen/go-fuse/v2
type MPRMount struct {
    path   string
    buf    []byte
    server *fuse.Server
}

func MountMPR(t *testing.T, mprBytes []byte) *MPRMount
func (m *MPRMount) Path() string     // 传给 backend.Connect()
func (m *MPRMount) Bytes() []byte    // 含写入后的修改
func (m *MPRMount) Unmount()         // t.Cleanup 自动调用
```

**v1/v2 范围**：

| 版本 | 格式 | 实现 |
|------|------|------|
| MPR v1（< Mendix 10.18） | 单 SQLite 文件 | 单文件 FUSE FS，初始实现 ✓ |
| MPR v2（≥ Mendix 10.18） | `.mpr` + `mprcontents/` | 虚拟目录树，v2 阶段后补 |

v2 支持前，`NewWithMPRBytes` 检测版本，v2 项目 `t.Skip("v2 fuse: TODO")`。

---

## 设计六：`cmd/mdlrun`（独立开发工具）

**文件**：`cmd/mdlrun/main.go`（约 50 行）

```go
// 用法：
//   go run ./cmd/mdlrun -p app.mpr -c "show entities"
//   go run ./cmd/mdlrun -p app.mpr script.mdl
// 无 Cobra、无 daemon、无 socket。利用 Go build cache，增量编译。
func main() {
    p := flag.String("p", "", "path to .mpr file")
    c := flag.String("c", "", "MDL command")
    flag.Parse()

    exec := executor.Build().
        Out(os.Stdout).
        ProgressOut(os.Stderr).
        WithFactory(func() backend.FullBackend { return mprbackend.New() }).
        Create()
    defer exec.Close()

    if *p != "" {
        must(runMDL(exec, fmt.Sprintf("CONNECT LOCAL '%s'", *p)))
    }
    if *c != "" {
        must(runMDL(exec, *c))
    } else if f := flag.Arg(0); f != "" {
        content, _ := os.ReadFile(f)
        must(runMDL(exec, string(content)))
    }
}
```

**Makefile 新增**：

```makefile
mdlrun:
	go build -o bin/mdlrun ./cmd/mdlrun

.PHONY: mdlrun
```

---

## 设计七：`buildExec` 迁移（替换 `newLoggedExecutor`）

`newLoggedExecutor` 改名为 `buildExec`，内部改用 `executor.Build()`，行为不变：

```go
// cmd/mxcli/main.go — 修复后（仍在 main 包，但现在是 Builder 的薄包装）
func buildExec(mode string, out io.Writer) (*executor.Executor, *diaglog.Logger) {
    logger := diaglog.Init(version, mode, globalVerboseLevel)
    b := executor.Build().Out(out).WithLogger(logger)
    if persistentDaemonBackend != nil {
        b = b.WithBackend(&noOpConnectBackend{persistentDaemonBackend})
    } else {
        b = b.WithFactory(func() backend.FullBackend { return mprbackend.New() })
    }
    if globalJSONFlag {
        b = b.Format(executor.FormatJSON)
    }
    return b.Create(), logger
}
```

所有 `newLoggedExecutor` 调用（约 10 处）机械替换为 `buildExec`，用 `grep -r newLoggedExecutor` 逐一确认。

---

## 实施顺序

```
PR 1: executor.Builder          (基础，无破坏性变更)
PR 2: cmd/mdlrun                (立即解决开发慢问题，依赖 PR 1)
PR 3: executor/testutil         (外部包测试，依赖 PR 1)
PR 4: DIP 修复 + buildExec 迁移 (SOLID 合规，依赖 PR 1)
PR 5: Progress 流式             (独立，协议变更)
PR 6: FUSE 黄金测试             (独立，最复杂，v1 先上)
```

PR 2 在 PR 1 合并后立即可用，不依赖 3-6。

---

## 包依赖方向（修复后）

```
cmd/mxcli-launcher
    → internal/launcherproto

cmd/mxcli (daemon)
    → mdl/executor          ✓ (通过 executor.Build())
    → mdl/backend           ✓ (interface)
    → mdl/backend/mpr       ✓ (main 包，依赖树顶端)
    → internal/launcherproto

mdl/executor
    → mdl/backend           ✓ (interface only，不 import mprbackend)

mdl/executor/testutil
    → mdl/executor          ✓
    → mdl/backend/mock      ✓ (test helper)
    → mdl/backend/mpr       ✓ (test helper)

cmd/mdlrun
    → mdl/executor          ✓
    → mdl/backend/mpr       ✓ (main 包)

mdl/backend/mpr
    → mdl/backend           ✓ (implements interface)
    implements backend.PersistentBackend (compile-time check)
```
