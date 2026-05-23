# SOLID Handler Objectification Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 `mdl/executor/` 的 handler 从依赖 `*ExecContext` 上帝对象的自由函数，重构为依赖窄接口的可独立测试 handler struct，同时保持所有现有测试绿色。

**Architecture:** 新增 `StatementHandler` 接口 + `HandlerFactory` 工厂类型；`Registry` 双轨并存（旧 `StmtHandler` 函数 + 新 `HandlerFactory`）；每个领域 handler struct 通过构造函数注入最小后端接口，业务逻辑方法只接收 `context.Context` + 具体语句类型。Phase 0 完成基础设施，Phase 1 以 Enumeration / Constant / Image 三个低风险域验证模式。

**Tech Stack:** Go 1.22+，`mdl/executor/`，`mdl/backend/`（窄接口已定义），标准库 `context`，现有 `mock.MockBackend`（过渡期保留）。

---

## 文件映射

| 操作 | 路径 | 职责 |
|------|------|------|
| **新建** | `mdl/executor/handler.go` | `StatementHandler`、`HandlerFactory`、`StatementExecutor` 接口定义 |
| **修改** | `mdl/executor/registry.go` | 新增 `factories` map + `RegisterHandler` + 更新 `Dispatch` |
| **修改** | `mdl/executor/registry_test.go` | 覆盖新 `RegisterHandler` / `Dispatch` 双轨路径 |
| **修改** | `mdl/executor/executor.go` | `*Executor` 实现 `StatementExecutor`（compile-time check） |
| **新建** | `mdl/executor/hierarchy_helpers.go` | `getHierarchyFromCache` / `invalidateCachedHierarchy`（不依赖 ExecContext）|
| **新建** | `mdl/executor/cmd_enumeration_handler.go` | `EnumerationHandler` struct + Execute + create/drop 方法 |
| **新建** | `mdl/executor/cmd_enumeration_handler_test.go` | EnumerationHandler 窄接口单测 |
| **修改** | `mdl/executor/register_stubs.go` | `registerEnumerationHandlers` 改用 `RegisterHandler` |
| **新建** | `mdl/executor/cmd_constant_handler.go` | `ConstantHandler` struct |
| **新建** | `mdl/executor/cmd_constant_handler_test.go` | ConstantHandler 窄接口单测 |
| **修改** | `mdl/executor/register_stubs.go` | `registerConstantHandlers` 改用 `RegisterHandler` |
| **新建** | `mdl/executor/cmd_image_handler.go` | `ImageHandler` struct |
| **新建** | `mdl/executor/cmd_image_handler_test.go` | ImageHandler 窄接口单测 |
| **修改** | `mdl/executor/register_stubs.go` | `registerImageHandlers` 改用 `RegisterHandler` |

---

## Task 1：定义核心接口（handler.go）

**Files:**
- Create: `mdl/executor/handler.go`

- [ ] **Step 1: 写失败编译检查（将在 Task 4 中使用，这里先定义接口）**

创建 `mdl/executor/handler.go`，内容如下：

```go
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

// StatementHandler 是每个 MDL 领域命令的可测试执行单元。
// 实现者通过构造函数接收最小依赖，Execute 只接收标准 context 和具体语句。
type StatementHandler interface {
	Execute(ctx context.Context, stmt ast.Statement) error
}

// HandlerFactory 从 ExecContext 提取最小依赖并构造 StatementHandler。
// 工厂在每次 Dispatch 时调用；handler 实例无状态，每次调用后丢弃。
type HandlerFactory func(ctx *ExecContext) StatementHandler

// StatementExecutor 让需要递归执行的 handler 依赖抽象，而非 ExecContext.ExecuteFn 闭包。
type StatementExecutor interface {
	Execute(stmt ast.Statement) error
}
```

- [ ] **Step 2: 确认编译通过**

```bash
go build ./mdl/executor/
```

期望：无错误。

- [ ] **Step 3: 提交**

```bash
git add mdl/executor/handler.go
git commit -m "feat(executor): define StatementHandler, HandlerFactory, StatementExecutor interfaces"
```

---

## Task 2：Registry 双轨改造

**Files:**
- Modify: `mdl/executor/registry.go`
- Modify: `mdl/executor/registry_test.go`

- [ ] **Step 1: 写新路径的失败测试**

在 `mdl/executor/registry_test.go` 末尾追加：

```go
func TestRegistry_RegisterHandler_DispatchesViaFactory(t *testing.T) {
	r := &Registry{
		handlers:  make(map[reflect.Type]StmtHandler),
		factories: make(map[reflect.Type]HandlerFactory),
	}
	called := false
	r.RegisterHandler(&ast.CreateEnumerationStmt{}, func(ctx *ExecContext) StatementHandler {
		return &testStatementHandler{fn: func(_ context.Context, _ ast.Statement) error {
			called = true
			return nil
		}}
	})
	ctx := &ExecContext{Context: context.Background()}
	err := r.Dispatch(ctx, &ast.CreateEnumerationStmt{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("factory handler was not called")
	}
}

func TestRegistry_RegisterHandler_OldPathStillWorks(t *testing.T) {
	r := &Registry{
		handlers:  make(map[reflect.Type]StmtHandler),
		factories: make(map[reflect.Type]HandlerFactory),
	}
	called := false
	r.Register(&ast.DropEnumerationStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		called = true
		return nil
	})
	ctx := &ExecContext{Context: context.Background()}
	err := r.Dispatch(ctx, &ast.DropEnumerationStmt{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("old path handler was not called")
	}
}

// testStatementHandler is a test double for StatementHandler.
type testStatementHandler struct {
	fn func(context.Context, ast.Statement) error
}

func (h *testStatementHandler) Execute(ctx context.Context, stmt ast.Statement) error {
	return h.fn(ctx, stmt)
}
```

- [ ] **Step 2: 运行确认测试失败**

```bash
go test ./mdl/executor/ -run TestRegistry_RegisterHandler -v
```

期望：编译失败（`factories` 字段和 `RegisterHandler` 方法未定义）。

- [ ] **Step 3: 改造 registry.go**

将 `mdl/executor/registry.go` 替换为：

```go
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"reflect"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
)

// StmtHandler executes a single statement type.
// Implementations receive the concrete statement via type assertion.
type StmtHandler func(ctx *ExecContext, stmt ast.Statement) error

// Registry maps AST statement types to their handler functions.
// 双轨：handlers 保存旧式函数，factories 保存新式工厂。
// Dispatch 优先走 factories，回退到 handlers。
type Registry struct {
	handlers  map[reflect.Type]StmtHandler
	factories map[reflect.Type]HandlerFactory
}

// NewRegistry creates a Registry with all statement handlers registered.
func NewRegistry() *Registry {
	r := &Registry{
		handlers:  make(map[reflect.Type]StmtHandler),
		factories: make(map[reflect.Type]HandlerFactory),
	}
	registerConnectionHandlers(r)
	registerModuleHandlers(r)
	registerEnumerationHandlers(r)
	registerConstantHandlers(r)
	registerDatabaseConnectionHandlers(r)
	registerEntityHandlers(r)
	registerAssociationHandlers(r)
	registerMicroflowAndNanoflowHandlers(r)
	registerPageHandlers(r)
	registerSecurityHandlers(r)
	registerNavigationHandlers(r)
	registerImageHandlers(r)
	registerWorkflowHandlers(r)
	registerBusinessEventHandlers(r)
	registerSettingsHandlers(r)
	registerODataHandlers(r)
	registerJSONStructureHandlers(r)
	registerMappingHandlers(r)
	registerRESTHandlers(r)
	registerDataTransformerHandlers(r)
	registerQueryHandlers(r)
	registerStylingHandlers(r)
	registerRepositoryHandlers(r)
	registerSessionHandlers(r)
	registerLintHandlers(r)
	registerAlterPageHandlers(r)
	registerFragmentHandlers(r)
	registerSQLHandlers(r)
	registerImportHandlers(r)
	registerAgentEditorHandlers(r)
	return r
}

// Register maps a statement type to its handler (旧路径). Panics on duplicate.
func (r *Registry) Register(stmt ast.Statement, handler StmtHandler) {
	t := reflect.TypeOf(stmt)
	if _, exists := r.handlers[t]; exists {
		panic(fmt.Sprintf("registry: duplicate handler registration for %s", t))
	}
	if _, exists := r.factories[t]; exists {
		panic(fmt.Sprintf("registry: statement %s already registered via RegisterHandler", t))
	}
	r.handlers[t] = handler
}

// RegisterHandler maps a statement type to a HandlerFactory (新路径). Panics on duplicate.
func (r *Registry) RegisterHandler(stmt ast.Statement, f HandlerFactory) {
	t := reflect.TypeOf(stmt)
	if _, exists := r.factories[t]; exists {
		panic(fmt.Sprintf("registry: duplicate handler registration for %s", t))
	}
	if _, exists := r.handlers[t]; exists {
		panic(fmt.Sprintf("registry: statement %s already registered via Register", t))
	}
	r.factories[t] = f
}

// Lookup returns the StmtHandler for the given statement (旧路径), or nil.
func (r *Registry) Lookup(stmt ast.Statement) StmtHandler {
	return r.handlers[reflect.TypeOf(stmt)]
}

// Dispatch finds and executes the handler for stmt.
// 新路径（factories）优先；回退到旧路径（handlers）。
func (r *Registry) Dispatch(ctx *ExecContext, stmt ast.Statement) error {
	t := reflect.TypeOf(stmt)
	if f, ok := r.factories[t]; ok {
		return f(ctx).Execute(ctx.Context, stmt)
	}
	if h, ok := r.handlers[t]; ok {
		return h(ctx, stmt)
	}
	return mdlerrors.NewUnsupported(fmt.Sprintf("unhandled statement type %T", stmt))
}

// Validate checks that every known AST statement type has a registered handler.
func (r *Registry) Validate(knownTypes []ast.Statement) error {
	var missing []string
	for _, s := range knownTypes {
		t := reflect.TypeOf(s)
		if _, ok := r.handlers[t]; !ok {
			if _, ok2 := r.factories[t]; !ok2 {
				missing = append(missing, t.String())
			}
		}
	}
	if len(missing) > 0 {
		return mdlerrors.NewValidationf("registry: %d unregistered statement type(s): %v", len(missing), missing)
	}
	return nil
}

// HandlerCount returns the total number of registered handlers (旧路径 + 新路径).
func (r *Registry) HandlerCount() int {
	return len(r.handlers) + len(r.factories)
}
```

- [ ] **Step 4: 运行测试确认全绿**

```bash
go test ./mdl/executor/ -run TestRegistry -v
```

期望：所有 `TestRegistry_*` 测试 PASS。

- [ ] **Step 5: 运行全量测试确认无回归**

```bash
go test ./mdl/executor/ ./mdl/backend/... 2>&1 | tail -5
```

期望：`ok` 无 FAIL。

- [ ] **Step 6: 提交**

```bash
git add mdl/executor/registry.go mdl/executor/registry_test.go
git commit -m "feat(executor): extend Registry with dual-track HandlerFactory support"
```

---

## Task 3：StatementExecutor compile-time check

**Files:**
- Modify: `mdl/executor/executor.go`

- [ ] **Step 1: 在 executor.go 中添加 compile-time interface check**

在 `executor.go` 的 `package executor` 声明下方，找到第一个 `import` 块之后、第一个 `type` 声明之前，添加：

```go
// Compile-time check: *Executor implements StatementExecutor.
var _ StatementExecutor = (*Executor)(nil)
```

- [ ] **Step 2: 确认 Executor 已有 Execute(stmt ast.Statement) error 方法**

```bash
grep -n "^func (e \*Executor) Execute" mdl/executor/executor.go
```

期望：输出类似 `311:func (e *Executor) Execute(stmt ast.Statement) error {`。

若方法不存在（不应发生），需先实现，但根据代码库现状此步应直接通过。

- [ ] **Step 3: 编译确认**

```bash
go build ./mdl/executor/
```

期望：无错误。

- [ ] **Step 4: 提交**

```bash
git add mdl/executor/executor.go
git commit -m "feat(executor): add compile-time StatementExecutor interface check on Executor"
```

---

## Task 4：hierarchy_helpers.go — 脱离 ExecContext 的 hierarchy 工具

**Files:**
- Create: `mdl/executor/hierarchy_helpers.go`

这些 helper 供后续 handler struct 使用，不依赖 `*ExecContext`。

- [ ] **Step 1: 写失败测试**

新建 `mdl/executor/hierarchy_helpers_test.go`：

```go
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	"github.com/mendixlabs/mxcli/model"
)

func TestGetHierarchyFromCache_BuildsOnFirstCall(t *testing.T) {
	mod := mkModule("TestMod")
	mb := &mock.MockBackend{
		ListModulesFunc:  func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListFoldersFunc:  func() ([]*model.Folder, error) { return nil, nil },
		IsConnectedFunc:  func() bool { return true },
	}
	cache := &executorCache{}

	h, err := getHierarchyFromCache(cache, mb)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h == nil {
		t.Fatal("expected non-nil hierarchy")
	}
	// Second call should return cached value (no backend call).
	h2, err := getHierarchyFromCache(cache, mb)
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}
	if h != h2 {
		t.Fatal("expected same hierarchy instance on second call (cache hit)")
	}
}

func TestInvalidateCachedHierarchy_ClearsCache(t *testing.T) {
	cache := &executorCache{hierarchy: &ContainerHierarchy{}}
	invalidateCachedHierarchy(cache)
	if cache.hierarchy != nil {
		t.Fatal("expected hierarchy to be nil after invalidation")
	}
}
```

- [ ] **Step 2: 运行确认失败**

```bash
go test ./mdl/executor/ -run TestGetHierarchyFromCache -run TestInvalidateCachedHierarchy -v
```

期望：编译失败（函数未定义）。

- [ ] **Step 3: 实现 hierarchy_helpers.go**

新建 `mdl/executor/hierarchy_helpers.go`：

```go
// SPDX-License-Identifier: Apache-2.0

package executor

import "github.com/mendixlabs/mxcli/mdl/backend"

// getHierarchyFromCache returns the cached ContainerHierarchy or builds one
// from b if the cache is empty. Does not depend on *ExecContext.
func getHierarchyFromCache(cache *executorCache, b backend.ModuleBackend) (*ContainerHierarchy, error) {
	if cache == nil {
		cache = &executorCache{}
	}
	if cache.hierarchy != nil {
		return cache.hierarchy, nil
	}
	h, err := NewContainerHierarchyFromBackend(b)
	if err != nil {
		return nil, err
	}
	cache.hierarchy = h
	return h, nil
}

// invalidateCachedHierarchy clears the cached hierarchy so it will be rebuilt
// on next access. Call after any write that creates or deletes units.
func invalidateCachedHierarchy(cache *executorCache) {
	if cache != nil {
		cache.hierarchy = nil
	}
}
```

- [ ] **Step 4: 运行测试确认通过**

```bash
go test ./mdl/executor/ -run TestGetHierarchyFromCache -run TestInvalidateCachedHierarchy -v
```

期望：PASS。

- [ ] **Step 5: 全量测试无回归**

```bash
go test ./mdl/executor/ 2>&1 | tail -3
```

- [ ] **Step 6: 提交**

```bash
git add mdl/executor/hierarchy_helpers.go mdl/executor/hierarchy_helpers_test.go
git commit -m "feat(executor): add getHierarchyFromCache/invalidateCachedHierarchy helpers (no ExecContext)"
```

---

## Task 5：EnumerationHandler — 试点域 1

**Files:**
- Create: `mdl/executor/cmd_enumeration_handler.go`
- Create: `mdl/executor/cmd_enumeration_handler_test.go`
- Modify: `mdl/executor/register_stubs.go`

**分析：** `execCreateEnumeration` 使用 `ctx.Backend`（EnumerationBackend + ModuleBackend）、`ctx.Cache`、`ctx.Output`。`execDropEnumeration` 同上。`execAlterEnumeration` 目前返回 unsupported，直接保留。

- [ ] **Step 1: 写失败单测**

新建 `mdl/executor/cmd_enumeration_handler_test.go`：

```go
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/model"
)

// --- 窄接口 mock ---

type mockEnumBackend struct {
	listFn   func() ([]*model.Enumeration, error)
	createFn func(*model.Enumeration) error
	updateFn func(*model.Enumeration) error
	deleteFn func(model.ID) error
}

func (m *mockEnumBackend) ListEnumerations() ([]*model.Enumeration, error) {
	if m.listFn != nil {
		return m.listFn()
	}
	return nil, nil
}
func (m *mockEnumBackend) GetEnumeration(id model.ID) (*model.Enumeration, error) { return nil, nil }
func (m *mockEnumBackend) CreateEnumeration(e *model.Enumeration) error {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return nil
}
func (m *mockEnumBackend) UpdateEnumeration(e *model.Enumeration) error {
	if m.updateFn != nil {
		return m.updateFn(e)
	}
	return nil
}
func (m *mockEnumBackend) MoveEnumeration(e *model.Enumeration) error   { return nil }
func (m *mockEnumBackend) DeleteEnumeration(id model.ID) error {
	if m.deleteFn != nil {
		return m.deleteFn(id)
	}
	return nil
}

type mockModuleBackendForEnum struct {
	listFn func() ([]*model.Module, error)
}

func (m *mockModuleBackendForEnum) ListModules() ([]*model.Module, error) {
	if m.listFn != nil {
		return m.listFn()
	}
	return nil, nil
}
func (m *mockModuleBackendForEnum) ListFolders() ([]*model.Folder, error) { return nil, nil }
func (m *mockModuleBackendForEnum) GetModule(id model.ID) (*model.Module, error) {
	return nil, errors.New("not found")
}
func (m *mockModuleBackendForEnum) GetModuleByName(name string) (*model.Module, error) {
	mods, _ := m.listFn()
	for _, mod := range mods {
		if mod.Name == name {
			return mod, nil
		}
	}
	return nil, errors.New("not found")
}
func (m *mockModuleBackendForEnum) CreateModule(mod *model.Module) error   { return nil }
func (m *mockModuleBackendForEnum) UpdateModule(mod *model.Module) error   { return nil }
func (m *mockModuleBackendForEnum) DeleteModule(id model.ID) error         { return nil }
func (m *mockModuleBackendForEnum) DeleteModuleWithCleanup(id model.ID, name string) error {
	return nil
}

// --- テスト ---

func TestEnumerationHandler_Create_Success(t *testing.T) {
	mod := mkModule("MyModule")
	var created *model.Enumeration

	h := &EnumerationHandler{
		enum: &mockEnumBackend{
			listFn:   func() ([]*model.Enumeration, error) { return nil, nil },
			createFn: func(e *model.Enumeration) error { created = e; return nil },
		},
		modules: &mockModuleBackendForEnum{
			listFn: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		},
		cache:  &executorCache{},
		output: &bytes.Buffer{},
	}

	stmt := &ast.CreateEnumerationStmt{
		Name:   ast.QualifiedName{Module: "MyModule", Name: "Status"},
		Values: []ast.EnumerationValue{{Name: "Active", Caption: "Active"}, {Name: "Inactive", Caption: "Inactive"}},
	}
	err := h.Execute(context.Background(), stmt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created == nil {
		t.Fatal("CreateEnumeration was not called")
	}
	if created.Name != "Status" {
		t.Errorf("expected Name=Status, got %s", created.Name)
	}
	if len(created.Values) != 2 {
		t.Errorf("expected 2 values, got %d", len(created.Values))
	}
}

func TestEnumerationHandler_Create_AlreadyExists_ReturnsError(t *testing.T) {
	mod := mkModule("MyModule")
	existing := &model.Enumeration{
		BaseElement: model.BaseElement{ID: nextID("e")},
		ContainerID: mod.ID,
		Name:        "Status",
	}
	h := buildEnumHierarchy(mod, existing)
	_ = h

	handler := &EnumerationHandler{
		enum: &mockEnumBackend{
			listFn: func() ([]*model.Enumeration, error) { return []*model.Enumeration{existing}, nil },
		},
		modules: &mockModuleBackendForEnum{
			listFn: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		},
		cache:  &executorCache{},
		output: &bytes.Buffer{},
	}

	stmt := &ast.CreateEnumerationStmt{
		Name:           ast.QualifiedName{Module: "MyModule", Name: "Status"},
		CreateOrModify: false,
	}
	err := handler.Execute(context.Background(), stmt)
	if err == nil {
		t.Fatal("expected error for duplicate enumeration")
	}
}

func TestEnumerationHandler_Drop_Success(t *testing.T) {
	mod := mkModule("MyModule")
	existing := &model.Enumeration{
		BaseElement: model.BaseElement{ID: nextID("e")},
		ContainerID: mod.ID,
		Name:        "Status",
	}
	var deletedID model.ID

	handler := &EnumerationHandler{
		enum: &mockEnumBackend{
			listFn:   func() ([]*model.Enumeration, error) { return []*model.Enumeration{existing}, nil },
			deleteFn: func(id model.ID) error { deletedID = id; return nil },
		},
		modules: &mockModuleBackendForEnum{
			listFn: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		},
		cache:  &executorCache{},
		output: &bytes.Buffer{},
	}

	stmt := &ast.DropEnumerationStmt{
		Names: []ast.QualifiedName{{Module: "MyModule", Name: "Status"}},
	}
	err := handler.Execute(context.Background(), stmt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deletedID != existing.ID {
		t.Errorf("expected DeleteEnumeration(%s), got %s", existing.ID, deletedID)
	}
}

// buildEnumHierarchy is a test helper that builds a ContainerHierarchy
// with a module containing an enumeration.
func buildEnumHierarchy(mod *model.Module, enum *model.Enumeration) *ContainerHierarchy {
	h := mkHierarchy(mod)
	withContainer(h, enum.ContainerID, mod.ID)
	return h
}
```

- [ ] **Step 2: 运行确认失败**

```bash
go test ./mdl/executor/ -run TestEnumerationHandler -v
```

期望：编译失败（`EnumerationHandler` 未定义）。

- [ ] **Step 3: 实现 EnumerationHandler**

新建 `mdl/executor/cmd_enumeration_handler.go`：

```go
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	"io"
)

// enumModuleBackend is the minimal ModuleBackend surface needed by EnumerationHandler.
// Satisfied by backend.ModuleBackend (which is already defined in mdl/backend/).
type enumModuleBackend interface {
	backend.ModuleBackend
}

// EnumerationHandler handles CREATE / DROP ENUMERATION statements.
// Dependencies are injected via NewEnumerationHandler; no ExecContext in Execute.
type EnumerationHandler struct {
	enum    backend.EnumerationBackend
	modules enumModuleBackend
	cache   *executorCache
	output  io.Writer
}

// NewEnumerationHandler extracts minimal dependencies from ctx.
func NewEnumerationHandler(ctx *ExecContext) *EnumerationHandler {
	return &EnumerationHandler{
		enum:    ctx.Backend,
		modules: ctx.Backend,
		cache:   ctx.Cache,
		output:  ctx.Output,
	}
}

// Execute dispatches to the appropriate method based on statement type.
func (h *EnumerationHandler) Execute(_ context.Context, stmt ast.Statement) error {
	switch s := stmt.(type) {
	case *ast.CreateEnumerationStmt:
		return h.create(s)
	case *ast.AlterEnumerationStmt:
		return mdlerrors.NewUnsupported("alter enumeration not yet implemented")
	case *ast.DropEnumerationStmt:
		return h.drop(s)
	default:
		return fmt.Errorf("EnumerationHandler: unhandled %T", stmt)
	}
}

func (h *EnumerationHandler) create(s *ast.CreateEnumerationStmt) error {
	if violations := ValidateEnumeration(s); len(violations) > 0 {
		var msgs []string
		for _, v := range violations {
			msgs = append(msgs, v.Message)
		}
		return mdlerrors.NewValidationf("invalid enumeration '%s':\n  - %s",
			s.Name.String(), strings.Join(msgs, "\n  - "))
	}

	h2, err := getHierarchyFromCache(h.cache, h.modules)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	module, err := h.findModule(s.Name.Module)
	if err != nil {
		return err
	}

	existingEnum := h.findEnumeration(h2, s.Name.Module, s.Name.Name)
	if existingEnum != nil && !s.CreateOrModify {
		return mdlerrors.NewAlreadyExistsMsg("enumeration", s.Name.String(),
			fmt.Sprintf("enumeration already exists: %s (use create or modify to update)", s.Name))
	}

	var values []model.EnumerationValue
	for _, v := range s.Values {
		values = append(values, model.EnumerationValue{
			Name: v.Name,
			Caption: &model.Text{
				Translations: map[string]string{"en_US": v.Caption},
			},
		})
	}

	enum := &model.Enumeration{
		ContainerID:   module.ID,
		Name:          s.Name.Name,
		Documentation: s.Documentation,
		Values:        values,
	}

	if existingEnum != nil {
		enum.ID = existingEnum.ID
		if err := h.enum.UpdateEnumeration(enum); err != nil {
			return mdlerrors.NewBackend("update enumeration", err)
		}
		invalidateCachedHierarchy(h.cache)
		fmt.Fprintf(h.output, "Modified enumeration: %s\n", s.Name)
		return nil
	}

	if err := h.enum.CreateEnumeration(enum); err != nil {
		return mdlerrors.NewBackend("create enumeration", err)
	}
	invalidateCachedHierarchy(h.cache)
	fmt.Fprintf(h.output, "Created enumeration: %s\n", s.Name)
	return nil
}

func (h *EnumerationHandler) drop(s *ast.DropEnumerationStmt) error {
	enums, err := h.enum.ListEnumerations()
	if err != nil {
		return mdlerrors.NewBackend("list enumerations", err)
	}

	h2, err := getHierarchyFromCache(h.cache, h.modules)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	for _, name := range s.Names {
		found := false
		for _, e := range enums {
			modID := h2.FindModuleID(e.ContainerID)
			modName := h2.GetModuleName(modID)
			if e.Name == name.Name && (name.Module == "" || modName == name.Module) {
				if err := h.enum.DeleteEnumeration(e.ID); err != nil {
					return mdlerrors.NewBackend("delete enumeration", err)
				}
				invalidateCachedHierarchy(h.cache)
				fmt.Fprintf(h.output, "Dropped enumeration: %s.%s\n", modName, e.Name)
				found = true
				break
			}
		}
		if !found && !s.IfExists {
			return mdlerrors.NewNotFoundMsg("enumeration", name.String(), "enumeration not found: "+name.String())
		}
	}
	return nil
}

func (h *EnumerationHandler) findModule(name string) (*model.Module, error) {
	mods, err := h.modules.ListModules()
	if err != nil {
		return nil, mdlerrors.NewBackend("list modules", err)
	}
	for _, m := range mods {
		if m.Name == name {
			return m, nil
		}
	}
	return nil, mdlerrors.NewNotFoundMsg("module", name, "module not found: "+name)
}

func (h *EnumerationHandler) findEnumeration(hier *ContainerHierarchy, moduleName, enumName string) *model.Enumeration {
	enums, err := h.enum.ListEnumerations()
	if err != nil {
		return nil
	}
	for _, e := range enums {
		modID := hier.FindModuleID(e.ContainerID)
		modName := hier.GetModuleName(modID)
		if e.Name == enumName && modName == moduleName {
			return e
		}
	}
	return nil
}
```

- [ ] **Step 4: 运行测试确认通过**

```bash
go test ./mdl/executor/ -run TestEnumerationHandler -v
```

期望：所有 3 个 `TestEnumerationHandler_*` PASS。

- [ ] **Step 5: 更新 registerEnumerationHandlers 使用新路径**

在 `mdl/executor/register_stubs.go` 中，将 `registerEnumerationHandlers` 替换为：

```go
func registerEnumerationHandlers(r *Registry) {
	factory := func(ctx *ExecContext) StatementHandler {
		return NewEnumerationHandler(ctx)
	}
	r.RegisterHandler(&ast.CreateEnumerationStmt{}, factory)
	r.RegisterHandler(&ast.AlterEnumerationStmt{}, factory)
	r.RegisterHandler(&ast.DropEnumerationStmt{}, factory)
}
```

- [ ] **Step 6: 全量测试确认无回归**

```bash
go test ./mdl/executor/ 2>&1 | tail -5
```

期望：`ok` 无 FAIL。

- [ ] **Step 7: 提交**

```bash
git add mdl/executor/cmd_enumeration_handler.go mdl/executor/cmd_enumeration_handler_test.go mdl/executor/register_stubs.go
git commit -m "feat(executor): migrate EnumerationHandler to StatementHandler pattern (pilot domain 1)"
```

---

## Task 6：ConstantHandler — 试点域 2

**Files:**
- Create: `mdl/executor/cmd_constant_handler.go`
- Create: `mdl/executor/cmd_constant_handler_test.go`
- Modify: `mdl/executor/register_stubs.go`

**分析：** `createConstant` 和 `dropConstant` 依赖 `ConstantBackend`、`ModuleBackend`（hierarchy）、`executorCache`、`output`。还需要 `SettingsBackend`（`listConstantValues` 使用，但 CREATE/DROP 不需要，SHOW 可单独处理）。

- [ ] **Step 1: 写失败单测**

新建 `mdl/executor/cmd_constant_handler_test.go`：

```go
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"bytes"
	"context"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/model"
)

type mockConstantBackend struct {
	listFn   func() ([]*model.Constant, error)
	createFn func(*model.Constant) error
	updateFn func(*model.Constant) error
	deleteFn func(model.ID) error
}

func (m *mockConstantBackend) ListConstants() ([]*model.Constant, error) {
	if m.listFn != nil {
		return m.listFn()
	}
	return nil, nil
}
func (m *mockConstantBackend) GetConstant(id model.ID) (*model.Constant, error) { return nil, nil }
func (m *mockConstantBackend) CreateConstant(c *model.Constant) error {
	if m.createFn != nil {
		return m.createFn(c)
	}
	return nil
}
func (m *mockConstantBackend) UpdateConstant(c *model.Constant) error {
	if m.updateFn != nil {
		return m.updateFn(c)
	}
	return nil
}
func (m *mockConstantBackend) MoveConstant(c *model.Constant) error { return nil }
func (m *mockConstantBackend) DeleteConstant(id model.ID) error {
	if m.deleteFn != nil {
		return m.deleteFn(id)
	}
	return nil
}

func TestConstantHandler_Create_Success(t *testing.T) {
	mod := mkModule("MyModule")
	var created *model.Constant

	h := &ConstantHandler{
		constants: &mockConstantBackend{
			listFn:   func() ([]*model.Constant, error) { return nil, nil },
			createFn: func(c *model.Constant) error { created = c; return nil },
		},
		modules: &mockModuleBackendForEnum{
			listFn: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		},
		cache:  &executorCache{},
		output: &bytes.Buffer{},
	}

	stmt := &ast.CreateConstantStmt{
		Name:         ast.QualifiedName{Module: "MyModule", Name: "MaxItems"},
		DataType:     ast.DataTypeInteger,
		DefaultValue: 100,
	}
	err := h.Execute(context.Background(), stmt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created == nil {
		t.Fatal("CreateConstant was not called")
	}
	if created.Name != "MaxItems" {
		t.Errorf("expected Name=MaxItems, got %s", created.Name)
	}
}

func TestConstantHandler_Create_AlreadyExists_ReturnsError(t *testing.T) {
	mod := mkModule("MyModule")
	existing := &model.Constant{
		BaseElement: model.BaseElement{ID: nextID("c")},
		ContainerID: mod.ID,
		Name:        "MaxItems",
	}

	h := &ConstantHandler{
		constants: &mockConstantBackend{
			listFn: func() ([]*model.Constant, error) { return []*model.Constant{existing}, nil },
		},
		modules: &mockModuleBackendForEnum{
			listFn: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		},
		cache:  &executorCache{},
		output: &bytes.Buffer{},
	}

	stmt := &ast.CreateConstantStmt{
		Name:           ast.QualifiedName{Module: "MyModule", Name: "MaxItems"},
		DataType:       ast.DataTypeInteger,
		CreateOrModify: false,
	}
	err := h.Execute(context.Background(), stmt)
	if err == nil {
		t.Fatal("expected error for duplicate constant")
	}
}

func TestConstantHandler_Drop_Success(t *testing.T) {
	mod := mkModule("MyModule")
	existing := &model.Constant{
		BaseElement: model.BaseElement{ID: nextID("c")},
		ContainerID: mod.ID,
		Name:        "MaxItems",
	}
	var deletedID model.ID

	h := &ConstantHandler{
		constants: &mockConstantBackend{
			listFn:   func() ([]*model.Constant, error) { return []*model.Constant{existing}, nil },
			deleteFn: func(id model.ID) error { deletedID = id; return nil },
		},
		modules: &mockModuleBackendForEnum{
			listFn: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		},
		cache:  &executorCache{},
		output: &bytes.Buffer{},
	}

	stmt := &ast.DropConstantStmt{
		Name: ast.QualifiedName{Module: "MyModule", Name: "MaxItems"},
	}
	err := h.Execute(context.Background(), stmt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deletedID != existing.ID {
		t.Errorf("expected DeleteConstant(%s), got %s", existing.ID, deletedID)
	}
}
```

- [ ] **Step 2: 运行确认失败**

```bash
go test ./mdl/executor/ -run TestConstantHandler -v
```

期望：编译失败（`ConstantHandler` 未定义）。

- [ ] **Step 3: 实现 ConstantHandler**

新建 `mdl/executor/cmd_constant_handler.go`：

```go
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
)

// ConstantHandler handles CREATE / DROP CONSTANT statements.
type ConstantHandler struct {
	constants backend.ConstantBackend
	modules   backend.ModuleBackend
	cache     *executorCache
	output    io.Writer
}

// NewConstantHandler extracts minimal dependencies from ctx.
func NewConstantHandler(ctx *ExecContext) *ConstantHandler {
	return &ConstantHandler{
		constants: ctx.Backend,
		modules:   ctx.Backend,
		cache:     ctx.Cache,
		output:    ctx.Output,
	}
}

// Execute dispatches to the appropriate method based on statement type.
func (h *ConstantHandler) Execute(_ context.Context, stmt ast.Statement) error {
	switch s := stmt.(type) {
	case *ast.CreateConstantStmt:
		return h.create(s)
	case *ast.DropConstantStmt:
		return h.drop(s)
	default:
		return fmt.Errorf("ConstantHandler: unhandled %T", stmt)
	}
}

func (h *ConstantHandler) create(s *ast.CreateConstantStmt) error {
	if s.Name.Module == "" {
		return mdlerrors.NewValidation("module name required for constant: use create constant Module.ConstantName")
	}

	module, err := h.findModule(s.Name.Module)
	if err != nil {
		return err
	}

	constType := astDataTypeToConstantDataType(s.DataType)
	defaultValue := ""
	if s.DefaultValue != nil {
		defaultValue = fmt.Sprintf("%v", s.DefaultValue)
	}

	existing, existingMod, err := h.findConstant(s.Name.Module, s.Name.Name)
	if err == nil && existing != nil {
		if s.CreateOrModify {
			doc := s.Comment
			if doc == "" {
				doc = s.Documentation
			}
			existing.Documentation = doc
			existing.Type = constType
			existing.DefaultValue = defaultValue
			existing.ExposedToClient = s.ExposedToClient
			if err := h.constants.UpdateConstant(existing); err != nil {
				return mdlerrors.NewBackend("update constant", err)
			}
			invalidateCachedHierarchy(h.cache)
			fmt.Fprintf(h.output, "Modified constant: %s.%s\n", existingMod, existing.Name)
			return nil
		}
		return mdlerrors.NewAlreadyExistsMsg("constant", s.Name.String(),
			fmt.Sprintf("constant already exists: %s (use create or modify to update)", s.Name))
	}

	doc := s.Comment
	if doc == "" {
		doc = s.Documentation
	}

	constant := &model.Constant{
		ContainerID:     module.ID,
		Name:            s.Name.Name,
		Documentation:   doc,
		Type:            constType,
		DefaultValue:    defaultValue,
		ExposedToClient: s.ExposedToClient,
	}

	if err := h.constants.CreateConstant(constant); err != nil {
		return mdlerrors.NewBackend("create constant", err)
	}
	invalidateCachedHierarchy(h.cache)
	fmt.Fprintf(h.output, "Created constant: %s.%s\n", s.Name.Module, s.Name.Name)
	return nil
}

func (h *ConstantHandler) drop(s *ast.DropConstantStmt) error {
	existing, modName, err := h.findConstant(s.Name.Module, s.Name.Name)
	if err != nil || existing == nil {
		return mdlerrors.NewNotFoundMsg("constant", s.Name.String(), "constant not found: "+s.Name.String())
	}
	if err := h.constants.DeleteConstant(existing.ID); err != nil {
		return mdlerrors.NewBackend("delete constant", err)
	}
	invalidateCachedHierarchy(h.cache)
	fmt.Fprintf(h.output, "Dropped constant: %s.%s\n", modName, existing.Name)
	return nil
}

func (h *ConstantHandler) findModule(name string) (*model.Module, error) {
	mods, err := h.modules.ListModules()
	if err != nil {
		return nil, mdlerrors.NewBackend("list modules", err)
	}
	for _, m := range mods {
		if m.Name == name {
			return m, nil
		}
	}
	return nil, mdlerrors.NewNotFoundMsg("module", name, "module not found: "+name)
}

func (h *ConstantHandler) findConstant(moduleName, constName string) (*model.Constant, string, error) {
	constants, err := h.constants.ListConstants()
	if err != nil {
		return nil, "", mdlerrors.NewBackend("list constants", err)
	}
	hier, err := getHierarchyFromCache(h.cache, h.modules)
	if err != nil {
		return nil, "", mdlerrors.NewBackend("build hierarchy", err)
	}
	for _, c := range constants {
		modID := hier.FindModuleID(c.ContainerID)
		modName := hier.GetModuleName(modID)
		if strings.EqualFold(modName, moduleName) && strings.EqualFold(c.Name, constName) {
			return c, modName, nil
		}
	}
	return nil, "", mdlerrors.NewNotFoundMsg("constant", moduleName+"."+constName, "not found")
}
```

- [ ] **Step 4: 运行测试确认通过**

```bash
go test ./mdl/executor/ -run TestConstantHandler -v
```

期望：所有 3 个 `TestConstantHandler_*` PASS。

- [ ] **Step 5: 更新 registerConstantHandlers**

在 `mdl/executor/register_stubs.go` 中，将 `registerConstantHandlers` 替换为：

```go
func registerConstantHandlers(r *Registry) {
	factory := func(ctx *ExecContext) StatementHandler {
		return NewConstantHandler(ctx)
	}
	r.RegisterHandler(&ast.CreateConstantStmt{}, factory)
	r.RegisterHandler(&ast.DropConstantStmt{}, factory)
}
```

- [ ] **Step 6: 全量测试确认无回归**

```bash
go test ./mdl/executor/ 2>&1 | tail -5
```

- [ ] **Step 7: 提交**

```bash
git add mdl/executor/cmd_constant_handler.go mdl/executor/cmd_constant_handler_test.go mdl/executor/register_stubs.go
git commit -m "feat(executor): migrate ConstantHandler to StatementHandler pattern (pilot domain 2)"
```

---

## Task 7：ImageHandler — 试点域 3

**Files:**
- Create: `mdl/executor/cmd_image_handler.go`
- Create: `mdl/executor/cmd_image_handler_test.go`
- Modify: `mdl/executor/register_stubs.go`

**分析：** `execCreateImageCollection` / `execDropImageCollection` 依赖 `ImageBackend`、`ModuleBackend`（hierarchy）、`executorCache`、`output`。

- [ ] **Step 1: 写失败单测**

新建 `mdl/executor/cmd_image_handler_test.go`：

```go
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"bytes"
	"context"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
)

type mockImageBackend struct {
	listFn   func() ([]*types.ImageCollection, error)
	createFn func(*types.ImageCollection) error
	deleteFn func(string) error
}

func (m *mockImageBackend) ListImageCollections() ([]*types.ImageCollection, error) {
	if m.listFn != nil {
		return m.listFn()
	}
	return nil, nil
}
func (m *mockImageBackend) CreateImageCollection(ic *types.ImageCollection) error {
	if m.createFn != nil {
		return m.createFn(ic)
	}
	return nil
}
func (m *mockImageBackend) UpdateImageCollection(ic *types.ImageCollection) error { return nil }
func (m *mockImageBackend) DeleteImageCollection(id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(id)
	}
	return nil
}

func TestImageHandler_Create_Success(t *testing.T) {
	mod := mkModule("Icons")
	var created *types.ImageCollection

	h := &ImageHandler{
		images: &mockImageBackend{
			listFn:   func() ([]*types.ImageCollection, error) { return nil, nil },
			createFn: func(ic *types.ImageCollection) error { created = ic; return nil },
		},
		modules: &mockModuleBackendForEnum{
			listFn: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		},
		cache:  &executorCache{},
		output: &bytes.Buffer{},
	}

	stmt := &ast.CreateImageCollectionStmt{
		Name: ast.QualifiedName{Module: "Icons", Name: "AppIcons"},
	}
	err := h.Execute(context.Background(), stmt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created == nil {
		t.Fatal("CreateImageCollection was not called")
	}
	if created.Name != "AppIcons" {
		t.Errorf("expected Name=AppIcons, got %s", created.Name)
	}
}

func TestImageHandler_Drop_Success(t *testing.T) {
	mod := mkModule("Icons")
	existing := &types.ImageCollection{
		BaseElement: model.BaseElement{ID: nextID("ic")},
		ContainerID: mod.ID,
		Name:        "AppIcons",
	}
	var deletedID string

	h := &ImageHandler{
		images: &mockImageBackend{
			listFn:   func() ([]*types.ImageCollection, error) { return []*types.ImageCollection{existing}, nil },
			deleteFn: func(id string) error { deletedID = id; return nil },
		},
		modules: &mockModuleBackendForEnum{
			listFn: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		},
		cache:  &executorCache{},
		output: &bytes.Buffer{},
	}

	stmt := &ast.DropImageCollectionStmt{
		Name: ast.QualifiedName{Module: "Icons", Name: "AppIcons"},
	}
	err := h.Execute(context.Background(), stmt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deletedID != string(existing.ID) {
		t.Errorf("expected DeleteImageCollection(%s), got %s", existing.ID, deletedID)
	}
}
```

- [ ] **Step 2: 运行确认失败**

```bash
go test ./mdl/executor/ -run TestImageHandler -v
```

期望：编译失败（`ImageHandler` 未定义）。

- [ ] **Step 3: 实现 ImageHandler**

新建 `mdl/executor/cmd_image_handler.go`：

```go
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"fmt"
	"io"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
)

// ImageHandler handles CREATE / DROP IMAGE COLLECTION statements.
type ImageHandler struct {
	images  backend.ImageBackend
	modules backend.ModuleBackend
	cache   *executorCache
	output  io.Writer
}

// NewImageHandler extracts minimal dependencies from ctx.
func NewImageHandler(ctx *ExecContext) *ImageHandler {
	return &ImageHandler{
		images:  ctx.Backend,
		modules: ctx.Backend,
		cache:   ctx.Cache,
		output:  ctx.Output,
	}
}

// Execute dispatches to the appropriate method based on statement type.
func (h *ImageHandler) Execute(_ context.Context, stmt ast.Statement) error {
	switch s := stmt.(type) {
	case *ast.CreateImageCollectionStmt:
		return h.create(s)
	case *ast.DropImageCollectionStmt:
		return h.drop(s)
	default:
		return fmt.Errorf("ImageHandler: unhandled %T", stmt)
	}
}

func (h *ImageHandler) create(s *ast.CreateImageCollectionStmt) error {
	module, err := h.findModule(s.Name.Module)
	if err != nil {
		return err
	}

	// Check duplicate
	existing := h.findCollection(s.Name.Module, s.Name.Name)
	if existing != nil && !s.CreateOrModify {
		return mdlerrors.NewAlreadyExistsMsg("image collection", s.Name.String(),
			fmt.Sprintf("image collection already exists: %s (use create or modify to update)", s.Name))
	}

	if existing != nil {
		fmt.Fprintf(h.output, "Image collection already exists (no changes): %s\n", s.Name)
		return nil
	}

	ic := &types.ImageCollection{
		BaseElement: model.BaseElement{ID: model.NewID()},
		ContainerID: module.ID,
		Name:        s.Name.Name,
		ExportLevel: "Hidden",
	}
	if err := h.images.CreateImageCollection(ic); err != nil {
		return mdlerrors.NewBackend("create image collection", err)
	}
	invalidateCachedHierarchy(h.cache)
	fmt.Fprintf(h.output, "Created image collection: %s\n", s.Name)
	return nil
}

func (h *ImageHandler) drop(s *ast.DropImageCollectionStmt) error {
	existing := h.findCollection(s.Name.Module, s.Name.Name)
	if existing == nil {
		return mdlerrors.NewNotFoundMsg("image collection", s.Name.String(), "image collection not found: "+s.Name.String())
	}
	if err := h.images.DeleteImageCollection(string(existing.ID)); err != nil {
		return mdlerrors.NewBackend("delete image collection", err)
	}
	invalidateCachedHierarchy(h.cache)
	fmt.Fprintf(h.output, "Dropped image collection: %s\n", s.Name)
	return nil
}

func (h *ImageHandler) findModule(name string) (*model.Module, error) {
	mods, err := h.modules.ListModules()
	if err != nil {
		return nil, mdlerrors.NewBackend("list modules", err)
	}
	for _, m := range mods {
		if m.Name == name {
			return m, nil
		}
	}
	return nil, mdlerrors.NewNotFoundMsg("module", name, "module not found: "+name)
}

func (h *ImageHandler) findCollection(moduleName, collName string) *types.ImageCollection {
	collections, err := h.images.ListImageCollections()
	if err != nil {
		return nil
	}
	hier, err := getHierarchyFromCache(h.cache, h.modules)
	if err != nil {
		return nil
	}
	for _, ic := range collections {
		modID := hier.FindModuleID(ic.ContainerID)
		modName := hier.GetModuleName(modID)
		if ic.Name == collName && modName == moduleName {
			return ic
		}
	}
	return nil
}
```

- [ ] **Step 4: 运行测试确认通过**

```bash
go test ./mdl/executor/ -run TestImageHandler -v
```

期望：所有 2 个 `TestImageHandler_*` PASS。

- [ ] **Step 5: 更新 registerImageHandlers**

在 `mdl/executor/register_stubs.go` 中，将 `registerImageHandlers` 替换为：

```go
func registerImageHandlers(r *Registry) {
	factory := func(ctx *ExecContext) StatementHandler {
		return NewImageHandler(ctx)
	}
	r.RegisterHandler(&ast.CreateImageCollectionStmt{}, factory)
	r.RegisterHandler(&ast.DropImageCollectionStmt{}, factory)
}
```

- [ ] **Step 6: 全量测试 + lint 确认**

```bash
go test ./mdl/executor/ ./mdl/backend/... 2>&1 | tail -5
go vet ./mdl/executor/
```

期望：`ok` 无 FAIL，`vet` 无报告。

- [ ] **Step 7: 提交**

```bash
git add mdl/executor/cmd_image_handler.go mdl/executor/cmd_image_handler_test.go mdl/executor/register_stubs.go
git commit -m "feat(executor): migrate ImageHandler to StatementHandler pattern (pilot domain 3)"
```

---

## Task 8：验收 — 回归 + 覆盖率确认

- [ ] **Step 1: 运行完整测试套件**

```bash
make test 2>&1 | tail -10
```

期望：`ok` 无 FAIL。

- [ ] **Step 2: 确认新 handler 有独立测试文件**

```bash
ls mdl/executor/cmd_enumeration_handler_test.go \
   mdl/executor/cmd_constant_handler_test.go \
   mdl/executor/cmd_image_handler_test.go
```

期望：三个文件均存在。

- [ ] **Step 3: 确认 HandlerCount 包含新路径**

```bash
go test ./mdl/executor/ -run TestRegistry -v 2>&1 | grep -i "PASS\|FAIL"
```

- [ ] **Step 4: 确认豁免命令仍走旧路径**

```bash
grep "execConnect\|execDisconnect\|execStatus" mdl/executor/register_stubs.go
```

期望：这三行仍使用 `r.Register`（旧路径），未被迁移。

- [ ] **Step 5: 最终提交**

```bash
git add -u
git commit -m "test(executor): Phase 0+1 SOLID handler objectification complete — 3 pilot domains migrated"
```

---

## 后续阶段提示（不在本计划范围内）

Phase 2（主力域）和 Phase 3（MprBackend 内部拆分）按照相同模式，每个域建立独立 PR：

- 每个域：新建 `cmd_xxx_handler.go` + `cmd_xxx_handler_test.go` + 更新 `register_stubs.go`
- 可用本计划的 Task 5-7 作为模板，替换域名和 backend 接口
- Phase 3（MprBackend 拆分）是独立 PR，不依赖 Phase 2 完成
