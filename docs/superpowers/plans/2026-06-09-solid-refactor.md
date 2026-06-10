# SOLID 架构重构实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 消除 executor 层的三个核心 SOLID 违反：DIP 类型断言、ExecContext God Object、直接 BSON 序列化泄漏。

**Architecture:** Task 1 给 backend 层新增 `ImportBufferBackend` 接口，让 executor 不再对 `*MprBackend` 做类型断言。Task 2 把 ExecContext 的 26 个字段用 5 个语义明确的嵌入子结构分组，利用 Go 字段提升保证所有 `ctx.Xxx` 调用无需改动，只需更新 ~58 个 struct 初始化点。Task 3 把 `genElementToBSONDoc` 移到 `WidgetSerializationBackend` 接口，使 `pages_builder_v3.go` 不再直接 import `modelsdk/codec`。

**Tech Stack:** Go 1.24，`mdl/backend`、`mdl/backend/mpr`、`mdl/executor` 包。

---

## 影响文件概览

| 文件 | Task | 操作 |
|------|------|------|
| `mdl/backend/connection.go` | 1 | 新增 `ImportBufferBackend` 接口 |
| `mdl/backend/backend.go` | 1 | 把 `ImportBufferBackend` 加入 `FullBackend` |
| `mdl/backend/mpr/backend.go` | 1 | 新增 `BeginImportBuffer()` wrapper；更新接口断言 |
| `mdl/backend/mock/mock_backend.go` | 1 | 新增 `BeginImportBufferFunc` stub |
| `mdl/executor/import_project.go` | 1 | 改用 `backend.ImportBufferBackend`；删除 `mprbackend` import |
| `mdl/executor/exec_context.go` | 2 | 定义 5 个子结构体；重写 ExecContext 为嵌入形式 |
| `mdl/executor/executor_dispatch.go` | 2 | 更新 `newExecContext` 工厂 |
| `mdl/executor/cmd_catalog.go` | 2 | 更新局部 ExecContext 初始化 |
| `mdl/executor/widget_fmt_basic.go` | 2 | 更新内联 ExecContext 初始化 |
| `mdl/executor/*_test.go` (~50 个) | 2 | 更新测试初始化（机械替换） |
| `mdl/backend/mutation.go` | 3 | `WidgetSerializationBackend` 新增 `SerializePageGenElement` |
| `mdl/backend/mpr/backend.go` | 3 | 实现 `SerializePageGenElement` |
| `mdl/backend/mock/mock_backend.go` | 3 | 新增 `SerializePageGenElementFunc` stub |
| `mdl/executor/pages_builder_v3.go` | 3 | 删除 `genElementToBSONDoc`；改用 backend 接口 |

---

## Task 1：新增 `ImportBufferBackend` 接口，消除类型断言

**目标违反**：`mdl/executor/import_project.go:98` 直接对 `*mprbackend.MprBackend` 做类型断言，绕过 Backend 接口抽象（DIP 违反）。

### 步骤

- [ ] **Step 1.1：在 `mdl/backend/connection.go` 末尾新增接口**

在文件末尾（`ScriptTransactionBackend` 之后）添加：

```go
// ImportBuffer is a write-buffer handle returned by ImportBufferBackend.BeginImportBuffer.
// Flush commits all buffered units in a single transaction; Discard drops them.
type ImportBuffer interface {
	Flush() error
	Discard()
}

// ImportBufferBackend is optionally implemented by backends that support
// buffered import sessions for bulk-write performance.
// Executor code uses a type assertion: if bufBE, ok := ctx.Backend.(backend.ImportBufferBackend); ok { ... }
type ImportBufferBackend interface {
	BeginImportBuffer() ImportBuffer
	DisableImportBuffer()
}
```

- [ ] **Step 1.2：验证 `unitstore.BufferedUnitStore` 已满足新接口**

运行：
```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
grep -n "func.*BufferedUnitStore.*Flush\|func.*BufferedUnitStore.*Discard" mdl/backend/unitstore/buffered.go
```
预期输出包含两行，分别是 `Flush() error` 和 `Discard()`。

- [ ] **Step 1.3：在 `mdl/backend/mpr/backend.go` 新增 `BeginImportBuffer` wrapper**

在 `EnableImportBuffer` 函数（行 1550）附近新增：

```go
// BeginImportBuffer implements backend.ImportBufferBackend.
// Returns the active BufferedUnitStore as backend.ImportBuffer.
func (b *MprBackend) BeginImportBuffer() backend.ImportBuffer {
	return b.EnableImportBuffer()
}
```

- [ ] **Step 1.4：更新 `mdl/backend/mpr/backend.go` 中的接口编译期断言**

找到第 44-45 行：
```go
var _ backend.FullBackend = (*MprBackend)(nil)
var _ backend.PageModelBackend = (*MprBackend)(nil)
```

在后面新增：
```go
var _ backend.ImportBufferBackend = (*MprBackend)(nil)
```

- [ ] **Step 1.5：运行编译确认**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go build ./mdl/backend/... ./mdl/backend/mpr/...
```
预期：无错误。

- [ ] **Step 1.6：更新 `mdl/executor/import_project.go`**

替换 import 块中的：
```go
mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
"github.com/mendirt_project.go:97-100`：
```go
var importBuf *unitstore.BufferedUnitStore
if mprBE, ok := ctx.Backend.(*mprbackend.MprBackend); ok {
    importBuf = mprBE.EnableImportBuffer()
    defer mprBE.DisableImportBuffer()
}
```

替换为：
```go
var importBuf backend.ImportBuffer
if bufBE, ok := ctx.Backend.(backend.ImportBufferBackend); ok {
    importBuf = bufBE.BeginImportBuffer()
    defer bufBE.DisableImportBuffer()
}
```

同时确认文件已经 import 了 `"github.com/mendixlabs/mxcli/mdl/backend"`（应该已经有，因为 `ctx.Backend` 的类型就是 `backend.FullBackend`）。

- [ ] **Step 1.7：更新 `mdl/backend/mock/mock_backend.go`**

在 MockBackend 中查找 `ScriptTransactionBackend` 相关的 Func 字段，在附近新增：

```go
BeginImportBufferFunc  func() backend.ImportBuffer
DisableImportBufferFunc func()
```

以及对应的方法实现：
```go
func (m *MockBackend) BeginImportBuffer() backend.ImportBuffer {
    if m.BeginImportBufferFunc != nil {
        return m.BeginImportBufferFunc()
    }
    return nil // no buffering in mock; executor checks for nil
}

func (m *MockBackend) DisableImportBuffer() {
    if m.DisableImportBufferFunc != nil {
        m.DisableImportBufferFunc()
    }
}
```

注意：mock 返回 nil 是正确的，因为 `import_project.go` 会检查 `importBuf != nil` 再使用。

- [ ] **Step 1.8：编译并运行测试**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go build ./mdl/executor/...
go test ./mdl/executor/ -run TestImport -v -count=1
```

如果没有 `TestImport`，运行：
```bash
go test ./mdl/executor/... -count=1 2>&1 | grep -E "FAIL|ok|import"
```

- [ ] **Step 1.9：commit**

```bash
git add mdl/backend/connection.go mdl/backend/backend.go mdl/backend/mpr/backend.go mdl/backend/mock/mock_backend.go mdl/executor/import_project.go
git commit -m "refactor(backend): add ImportBufferBackend interface, remove type assertion in executor

executor/import_project.go was directly asserting ctx.Backend.(*mprbackend.MprBackend)
to access EnableImportBuffer/DisableImportBuffer, breaking the DIP abstraction.

New ImportBufferBackend interface (backend/connection.go) exposes BeginImportBuffer()
and DisableImportBuffer(). MprBackend.BeginImportBuffer() wraps EnableImportBuffer()
returning backend.ImportBuffer. executor now uses an optional interface assertion
instead of a concrete type assertion.

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 2：ExecContext 拆分为语义子结构体

**目标违反**：`ExecContext` 26+ 个字段混合 8 种无关职责（SRP），且所有 handler 依赖整个 ExecContext 而非它实际需要的字段（ISP）。

**设计决策**：使用 Go 嵌入（embedded struct），而非嵌套字段。这样：
- `ctx.Output`、`ctx.Cache`、`ctx.Microflows` 等所有现有用法**无需改动**
- 只有 struct 初始化点（`&ExecContext{Output: w}`）需要更新为 `&ExecContext{ExecIO: ExecIO{Output: w}}`
- 共需更新约 58 处初始化点，全部是机械替换

### 子结构体设计

```
ExecRepos     — Microflows, Nanoflows, Security, JavaActions, JavaScriptActions, DomainModels, Workflows, Pages, Layouts, Snippets
ExecIO        — Output, StatusOutput, Format, Quiet
ExecSession   — Cache, Fragments, Settings, ScriptDepth, DescribingMicroflowHasReturnValue
ExecConnection — MprPath, BackendFactory, SqlMgr, ThemeRegistry, Catalog
ExecCallbacks  — ExecuteFn, ExecuteProgramFn, FinalizeFn, SyncCatalog
```

### 步骤

- [ ] **Step 2.1：重写 `mdl/executor/exec_context.go` 中的 ExecContext 定义**

在文件顶部（imports 之后）新增 5 个子结构体，然后重写 ExecContext：

```go
// ExecRepos holds Stage 3 modelsdk-native repositories.
// These are domain-specific read/write interfaces that avoid
// the FullBackend mega-interface per handler.
type ExecRepos struct {
	Microflows        repos.MicroflowRepository
	Nanoflows         repos.NanoflowRepository
	Security          repos.SecurityRepository
	JavaActions       repos.JavaActionRepository
	JavaScriptActions repos.JavaScriptActionRepository
	DomainModels      repos.DomainModelRepository
	Workflows         repos.WorkflowRepository
	Pages             repos.PageRepository
	Layouts           repos.LayoutRepository
	Snippets          repos.SnippetRepository
}

// ExecIO holds output writers and formatting settings.
type ExecIO struct {
	Output       io.Writer
	StatusOutput io.Writer
	Format       OutputFormat
	Quiet        bool
}

// ExecSession holds per-session mutable state.
// ScriptDepth and cache are reset between top-level executions.
type ExecSession struct {
	Cache     *executorCache
	Fragments map[string]*ast.DefineFragmentStmt
	Settings  map[string]any
	// ScriptDepth tracks EXECUTE SCRIPT nesting; max is maxScriptDepth.
	ScriptDepth int
	// DescribingMicroflowHasReturnValue is set while rendering a microflow body.
	DescribingMicroflowHasReturnValue bool
}

// ExecConnection holds project connection state and external services.
type ExecConnection struct {
	MprPath        string
	BackendFactory BackendFactory
	SqlMgr         *sqllib.Manager
	ThemeRegistry  *ThemeRegistry
	Catalog        *catalog.Catalog
}

// ExecCallbacks holds function references for recursive execution.
// All are set by newExecContext; nil in ad-hoc test contexts.
type ExecCallbacks struct {
	ExecuteFn        func(ast.Statement) error
	ExecuteProgramFn func(*ast.Program) error
	FinalizeFn       func() error
	SyncCatalog      func(*catalog.Catalog)
}

// ExecContext carries all dependencies a statement handler needs.
//
// Fields are grouped into embedded sub-structs by responsibility.
// All ctx.Xxx field accesses continue to work via Go field promotion;
// only struct literal initializers need to use the sub-struct names.
type ExecContext struct {
	context.Context

	// Backend provides all domain operations. Nil when not connected.
	Backend backend.FullBackend

	// Logger is the session diagnostics logger (nil = no logging).
	Logger *diaglog.Logger

	ExecRepos
	ExecIO
	ExecSession
	ExecConnection
	ExecCallbacks
}
```

- [ ] **Step 2.2：运行编译查看所有初始化点报错**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go build ./mdl/executor/... 2>&1 | grep "cannot use promoted field\|unknown field\|has no field" | head -30
```

这会列出所有需要更新的初始化点。

- [ ] **Step 2.3：更新 `executor_dispatch.go` 工厂函数**

找到 `newExecContext` 中的 `return &ExecContext{...}` 块（约 70 行），替换为：

```go
return &ExecContext{
    Context: ctx,
    Backend: e.backend,
    Logger:  e.logger,
    ExecRepos: ExecRepos{
        Microflows:        extractMicroflowsRepo(e.backend),
        Nanoflows:         extractNanoflowsRepo(e.backend),
        Security:          extractSecurityRepo(e.backend),
        JavaActions:       extractJavaActionsRepo(e.backend),
        JavaScriptActions: extractJavaScriptActionsRepo(e.backend),
        DomainModels:      extractDomainModelsRepo(e.backend),
        Workflows:         extractWorkflowsRepo(e.backend),
        Pages:             extractPagesRepo(e.backend),
        Layouts:           extractLayoutsRepo(e.backend),
        Snippets:          extractSnippetsRepo(e.backend),
    },
    ExecIO: ExecIO{
        Output:       e.output,
        StatusOutput: e.statusOutput,
        Format:       e.format,
        Quiet:        e.quiet,
    },
    ExecSession: ExecSession{
        Fragments: e.fragments,
        Cache:     e.cache,
        Settings:  e.settings,
    },
    ExecConnection: ExecConnection{
        MprPath:        e.mprPath,
        SqlMgr:         e.sqlMgr,
        ThemeRegistry:  e.themeRegistry,
        Catalog:        cat,
        BackendFactory: e.backendFactory,
    },
    ExecCallbacks: ExecCallbacks{
        ExecuteFn:        e.Execute,
        ExecuteProgramFn: e.ExecuteProgram,
        FinalizeFn:       e.finalizeProgramExecution,
        SyncCatalog: func(cat *catalog.Catalog) {
            e.catalogMu.Lock()
            defer e.catalogMu.Unlock()
            if e.catalogGen != gen {
                cat.Close()
                return
            }
            old := e.catalog
            e.catalog = cat
            if old != nil {
                old.Close()
            }
        },
    },
}
```

- [ ] **Step 2.4：更新 `cmd_catalog.go:738` 的局部 ExecContext**

找到：
```go
localCtx := &ExecContext{
    Context: ...,
    Backend: ...,
    Output: ...,
    ...
}
```
按相同模式更新为使用子结构体。

- [ ] **Step 2.5：更新 `widget_fmt_basic.go:48`**

```go
// 原来
outputWidgetMDLV3(&ExecContext{Output: ctx.Output}, rawWidgetFromMap(f.raw), ctx.Indent)
// 更新后
outputWidgetMDLV3(&ExecContext{ExecIO: ExecIO{Output: ctx.Output}}, rawWidgetFromMap(f.raw), ctx.Indent)
```

- [ ] **Step 2.6：批量更新测试文件**

对每个使用字段名初始化的测试文件，按以下映射规则替换：

| 原字段 | 新位置 |
|--------|--------|
| `Output:` | `ExecIO: ExecIO{Output: ...}` |
| `StatusOutput:` | `ExecIO: ExecIO{StatusOutput: ...}` |
| `Format:` | `ExecIO: ExecIO{Format: ...}` |
| `Quiet:` | `ExecIO: ExecIO{Quiet: ...}` |
| `Cache:` | `ExecSession: ExecSession{Cache: ...}` |
| `Fragments:` | `ExecSession: ExecSession{Fragments: ...}` |
| `Settings:` | `ExecSession: ExecSession{Settings: ...}` |
| `ScriptDepth:` | `ExecSession: ExecSession{ScriptDepth: ...}` |
| `Microflows:` | `ExecRepos: ExecRepos{Microflows: ...}` |
| `Nanoflows:` | `ExecRepos: ExecRepos{Nanoflows: ...}` |
| `Security:` | `ExecRepos: ExecRepos{Security: ...}` |
| `JavaActions:` | `ExecRepos: ExecRepos{JavaActions: ...}` |
| `Workflows:` | `ExecRepos: ExecRepos{Workflows: ...}` |
| `Pages:` | `ExecRepos: ExecRepos{Pages: ...}` |
| `Layouts:` | `ExecRepos: ExecRepos{Layouts: ...}` |
| `Snippets:` | `ExecRepos: ExecRepos{Snippets: ...}` |
| `MprPath:` | `ExecConnection: ExecConnection{MprPath: ...}` |
| `Catalog:` | `ExecConnection: ExecConnection{Catalog: ...}` |
| `BackendFactory:` | `ExecConnection: ExecConnection{BackendFactory: ...}` |
| `SqlMgr:` | `ExecConnection: ExecConnection{SqlMgr: ...}` |
| `ThemeRegistry:` | `ExecConnection: ExecConnection{ThemeRegistry: ...}` |
| `ExecuteFn:` | `ExecCallbacks: ExecCallbacks{ExecuteFn: ...}` |
| `ExecuteProgramFn:` | `ExecCallbacks: ExecCallbacks{ExecuteProgramFn: ...}` |
| `FinalizeFn:` | `ExecCallbacks: ExecCallbacks{FinalizeFn: ...}` |
| `SyncCatalog:` | `ExecCallbacks: ExecCallbacks{SyncCatalog: ...}` |

特殊规则：多字段同属一个子结构体时合并到同一个嵌入字段：
```go
// 原
&ExecContext{Output: w, Format: FormatTable}
// 更新后
&ExecContext{ExecIO: ExecIO{Output: w, Format: FormatTable}}
```

`Backend:` 和 `Logger:` 字段保持顶层，无需移动。

- [ ] **Step 2.7：编译确认**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go build ./mdl/executor/...
```
预期：无错误。

- [ ] **Step 2.8：检查所有现有 handler 用法无需改动**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
# 这些应该都编译通过（ctx.Xxx 通过字段提升仍然有效）
grep -rn "ctx\.Output\|ctx\.Cache\|ctx\.Microflows\|ctx\.Format\|ctx\.MprPath" mdl/executor/ --include="*.go" | grep -v "_test.go" | wc -l
```
预期：数量与之前相同（这些用法无需修改）。

- [ ] **Step 2.9：运行全量 executor 测试**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./mdl/executor/... -count=1 2>&1 | tail -20
```
预期：全部 `ok`。

- [ ] **Step 2.10：commit**

```bash
git add mdl/executor/exec_context.go mdl/executor/executor_dispatch.go mdl/executor/cmd_catalog.go mdl/executor/widget_fmt_basic.go mdl/executor/*_test.go
git commit -m "refactor(executor): group ExecContext fields into 5 embedded sub-structs

ExecContext's 26 fields mixed 8 unrelated concerns (SRP violation).
Introduce ExecRepos / ExecIO / ExecSession / ExecConnection / ExecCallbacks
as embedded structs. Go field promotion means all ctx.Xxx accesses
in handlers are unchanged. Only struct literal initializers (~58 sites)
need mechanical updates to use the sub-struct names.

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Task 3：把 `genElementToBSONDoc` 移到 backend 接口

**目标违反**：`pages_builder_v3.go` 直接 import `modelsdk/codec`（DIP 违反）。`genElementToBSONDoc` 是 executor 层手动调用底层 codec 序列化的唯一函数，应该通过 backend 接口暴露。

注意：`pages_builder_v3.go` 的其他 `bson.D{...}` 直接构造（DataGrid datasource builder）是更大的 BSON 泄漏问题，超出本次计划范围，留作后续专项 PR。

### 步骤

- [ ] **Step 3.1：在 `mdl/backend/mutation.go` 的 `WidgetSerializationBackend` 接口新增方法**

找到（约行 258）：
```go
type WidgetSerializationBackend interface {
	SerializeWorkflowActivityGen(a element.Element) (any, error)
}
```

新增一个方法：
```go
type WidgetSerializationBackend interface {
	SerializeWorkflowActivityGen(a element.Element) (any, error)
	// SerializePageGenElement encodes a gen-typed element to raw BSON bytes
	// for page/widget builder paths that need to embed a pre-encoded sub-document.
	// Returns the same bytes as codec.Encoder{}.Encode(elem).
	SerializePageGenElement(elem element.Element) ([]byte, error)
}
```

- [ ] **Step 3.2：在 `mdl/backend/mpr/backend.go` 实现新方法**

在 `SerializeWorkflowActivityGen` 实现附近（行 1377）新增：

```go
// SerializePageGenElement implements backend.WidgetSerializationBackend.
func (b *MprBackend) SerializePageGenElement(elem element.Element) ([]byte, error) {
	if elem == nil {
		return nil, fmt.Errorf("SerializePageGenElement: nil element")
	}
	enc := b.newEncoder()
	return enc.Encode(elem)
}
```

- [ ] **Step 3.3：在 `mdl/backend/mock/mock_backend.go` 新增 stub**

```go
SerializePageGenElementFunc func(elem element.Element) ([]byte, error)

func (m *MockBackend) SerializePageGenElement(elem element.Element) ([]byte, error) {
    if m.SerializePageGenElementFunc != nil {
        return m.SerializePageGenElementFunc(elem)
    }
    return nil, fmt.Errorf("MockBackend.SerializePageGenElement not configured")
}
```

- [ ] **Step 3.4：编译确认接口实现**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go build ./mdl/backend/... ./mdl/backend/mpr/... 2>&1
```

- [ ] **Step 3.5：在 `pages_builder_v3.go` 替换 `genElementToBSONDoc`**

找到（行 58-67）：
```go
// genElementToBSONDoc encodes a gen element to bson.D via the codec.
// Used in BSON-builder functions that need raw bson.D rather than a gen element.
func genElementToBSONDoc(elem element.Element) (bson.D, error) {
	enc := codec.Encoder{}
	raw, err := enc.Encode(elem)
	if err != nil {
		return nil, err
	}
	var doc bson.D
	return doc, bson.Unmarshal(raw, &doc)
}
```

替换为：
```go
// genElementToBSONDoc encodes a gen element to bson.D via the backend serializer.
// Routes through WidgetSerializationBackend.SerializePageGenElement so executor
// does not import modelsdk/codec directly.
func (pb *pageBuilder) genElementToBSONDoc(elem element.Element) (bson.D, error) {
	raw, err := pb.ctx.Backend.(backend.WidgetSerializationBackend).SerializePageGenElement(elem)
	if err != nil {
		return nil, err
	}
	var doc bson.D
	return doc, bson.Unmarshal(raw, &doc)
}
```

注意：原函数是包级 free function，现在改为 `pageBuilder` 的方法，以访问 `pb.ctx.Backend`。

- [ ] **Step 3.6：更新调用点**

找到 `pages_builder_v3.go:2129`：
```go
doc, err := genElementToBSONDoc(ns)
```
替换为：
```go
doc, err := pb.genElementToBSONDoc(ns)
```

（只有这一处调用点。）

- [ ] **Step 3.7：删除 `codec` import**

从 `pages_builder_v3.go` 的 import 块中删除：
```go
"github.com/mendixlabs/mxcli/modelsdk/codec"
```

- [ ] **Step 3.8：编译并运行测试**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go build ./mdl/executor/...
go test ./mdl/backend/mpr/... -run TestSerializePage -v -count=1
go test ./mdl/executor/... -count=1 2>&1 | tail -10
```

- [ ] **Step 3.9：commit**

```bash
git add mdl/backend/mutation.go mdl/backend/mpr/backend.go mdl/backend/mock/mock_backend.go mdl/executor/pages_builder_v3.go
git commit -m "refactor(backend): move genElementToBSONDoc to WidgetSerializationBackend

pages_builder_v3.go imported modelsdk/codec directly (DIP violation).
New SerializePageGenElement method on WidgetSerializationBackend exposes
the encode operation through the backend interface. The function is now
a pageBuilder method that routes through ctx.Backend.

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>"
```

---

## 自检 Checklist

- [ ] `go build ./...` 无错误
- [ ] `go test ./mdl/executor/... ./mdl/backend/...` 无 FAIL
- [ ] `git grep "mprbackend\." mdl/executor/` 无输出（executor 中无 mpr 具体类型引用）
- [ ] `git grep "modelsdk/codec" mdl/executor/` 只剩 `flowbuilder_raw_setter_v2.go` 和 `diff_local.go`（这两个有更深的 BSON 依赖，留后续 PR）
- [ ] `git grep 'ctx\.Backend\.(\*mprbackend' mdl/executor/` 无输出

## 范围外（后续 PR）

以下问题确认存在但超出本次安全重构范围：

1. **`flowbuilder_raw_setter_v2.go` codec 用法** — `setRawBSONChildElement` 操作 gen element 内部 BSON 结构，需要专门的 backend 接口方法，依赖 gen element 架构改变
2. **`diff_local.go` bson/codec 用法** — git diff 功能直接操作 git blob 字节，移至 backend 需要新的 `DecodeRawUnit([]byte) element.Element` 接口方法
3. **ExecContext 的 10 个 repo 字段** — Stage 3 完成后，可以进一步把 ExecRepos 合并回 Backend 的子接口，让 handler 直接从 Backend 获取 repo
4. **FullBackend 29 个子接口** — 正确的拆分需要所有 handler 函数签名改为接受具体子接口而非 ExecContext，是独立的架构演进
