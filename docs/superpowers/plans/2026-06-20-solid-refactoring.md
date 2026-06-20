# SOLID 重构实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 `mxcli` 核心架构重构为严格遵循 SOLID 原则的形态。Big Bang 方式——冻结特性开发，一次性替换所有违规点，通过完整测试套件验证。

**Architecture:** 消除 `FullBackend` 单体接口、`ExecContext` God 结构体、`executorCache` 单体缓存。`MprBackend` 拆分为按领域的子后端，每个由 `BackendFactory` 提供。处理函数通过闭包捕获精确依赖，不再依赖具体上下文结构体。`Registry` 使用字符串键，取消反射分发。

**Tech Stack:** Go 1.26+, no new dependencies.

## Global Constraints

- 每个任务结束时必须能独立编译通过
- 不得引入新的 `backend.FullBackend` 或 `executor.ExecContext` 导入（`mock` 包除外）
- 所有类型断言必须检查 `ok`
- 所有现有测试必须保持通过
- 添加新语句类型不得需要修改 `NewRegistry()`

---

## 文件结构总览

### 新建文件
```
cmd/check-solid/main.go                   # 静态检查工具
mdl/backend/factory.go                    # BackendFactory 接口
mdl/backend/mpr/mpr_module.go             # moduleBackend（从 backend.go 提取）
mdl/backend/mpr/mpr_microflow.go          # microflowBackend（从 backend.go 提取）
mdl/backend/mpr/mpr_page.go               # pageBackend（从 backend.go 提取）
mdl/backend/mpr/mpr_entity.go             # entityBackend（从 backend.go 提取）
mdl/backend/mpr/mpr_workflow.go           # workflowBackend（从 backend.go 提取）
mdl/backend/mpr/mpr_security.go           # securityBackend（从 backend.go 提取）
mdl/backend/mpr/mpr_enum.go               # enumerationBackend（从 backend.go 提取）
mdl/backend/mpr/mpr_constant.go           # constantBackend（从 backend.go 提取）
mdl/backend/mpr/mpr_mapping.go            # mappingBackend（从 backend.go 提取）
mdl/backend/mpr/mpr_settings.go           # settingsBackend（从 backend.go 提取）
mdl/backend/mpr/mpr_navigation.go         # navigationBackend（从 backend.go 提取）
mdl/backend/mpr/mpr_image.go              # imageBackend（从 backend.go 提取）
mdl/backend/mpr/mpr_java.go               # javaBackend（从 backend.go 提取）
mdl/backend/mpr/mpr_schedule.go           # scheduledEventBackend（从 backend.go 提取）
mdl/backend/mpr/mpr_services.go           # serviceBackend（从 backend.go 提取）
mdl/backend/mpr/cache_microflow.go        # microflowCache（从 executorCache 提取）
mdl/backend/mpr/cache_page.go             # pageCache（从 executorCache 提取）
mdl/backend/mpr/cache_entity.go           # entityCache（从 executorCache 提取）
mdl/backend/mpr/cache_workflow.go         # workflowCache（从 executorCache 提取）
mdl/backend/mpr/cache_security.go         # securityCache（从 executorCache 提取）
model/module.go                           # 从 types.go 提取 Module
model/enumeration.go                      # 从 types.go 提取 Enumeration
model/constant.go                         # 从 types.go 提取 Constant
model/schedule.go                         # 从 types.go 提取 ScheduledEvent
model/odata.go                            # 从 types.go 提取 OData 类型
model/rest.go                             # 从 types.go 提取 REST 类型
model/bizevent.go                         # 从 types.go 提取 BusinessEventService
model/dbconn.go                           # 从 types.go 提取 DatabaseConnection
model/transformer.go                      # 从 types.go 提取 DataTransformer
model/settings.go                         # 从 types.go 提取 ProjectSettings
model/mappings.go                         # 从 types.go 提取 ImportMapping/ExportMapping
mdl/executor/handlers/                    # 新目录
mdl/executor/handlers/handler_microflow.go
mdl/executor/handlers/handler_page.go
mdl/executor/handlers/handler_entity.go
mdl/executor/handlers/handler_security.go
mdl/executor/handlers/handler_module.go
mdl/executor/handlers/handler_workflow.go
mdl/executor/handlers/handler_enum.go
mdl/executor/handlers/handler_settings.go
mdl/executor/handlers/handler_navigation.go
mdl/executor/handlers/handler_image.go
mdl/executor/handlers/handler_java.go
mdl/executor/handlers/handler_rest.go
mdl/executor/handlers/handler_odata.go
mdl/executor/handlers/handler_mapping.go
mdl/executor/handlers/handler_session.go
mdl/executor/handlers/handler_sql.go
```

### 修改文件
```
mdl/ast/ast.go                            # Statement 接口添加 TypeName()
mdl/ast/ast_microflow.go                  # 每个 Stmt 添加 TypeName()
mdl/ast/ast_page.go                       # 同上
mdl/ast/ast_entity.go                     # 同上
...所有 ast/*.go 文件...                   # 同上
mdl/backend/backend.go                    # 删除 FullBackend，保留 ConnectionBackend
mdl/backend/role.go                       # 移除 FullBackend 嵌入，保持纯净
mdl/backend/mpr/backend.go                # 缩为 BackendFactory 仅连接管理
mdl/backend/mpr/repos_provider.go         # 删除（concreteWriter 模式移除）
mdl/backend/mpr/write_helpers.go          # 移除 concreteWriter 引用
mdl/backend/mpr/script_tx.go             # 移除 concreteWriter 引用
mdl/backend/mpr/create_services_v2.go    # 移除 concreteWriter 引用
mdl/executor/executor.go                  # 瘦身：移除 backend/cache/catalog/sql/theme
mdl/executor/exec_context.go             # 删除整个文件
mdl/executor/registry.go                  # 改为 string 键，取消 reflect
mdl/executor/register_stubs.go           # 删除（替换为 handlers/ 目录）
mdl/executor/builder.go                  # 适配新架构
mdl/executor/cmd_*.go                    # 每个处理函数改为闭包捕获依赖
cmd/mxcli/main.go                        # 适配新 Builder
```

### 删除文件
```
mdl/executor/exec_context.go
mdl/executor/register_stubs.go
mdl/backend/mpr/repos_provider.go
```

---

### Task 1: 基础设施——`cmd/check-solid` 静态检查工具

**Files:**
- Create: `cmd/check-solid/main.go`
- Test: 手动验证

**Interfaces:**
- Consumes: 项目源码树
- Produces: 退出码 0（通过）或 1（失败）

- [ ] **Step 1: 创建 `cmd/check-solid/main.go`**

```go
// SPDX-License-Identifier: Apache-2.0

// check-solid 验证项目源码是否违反 SOLID 重构原则。
// 用作 CI 门控，确保不会重新引入违规模式。
package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

var violations int

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		// 跳过 vendor、生成代码、测试文件
		if strings.Contains(path, "vendor/") || strings.HasSuffix(path, "_test.go") || strings.Contains(path, "generated/") {
			return nil
		}
		return checkFile(path)
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if violations > 0 {
		fmt.Fprintf(os.Stderr, "\n❌ Found %d SOLID violation(s)\n", violations)
		os.Exit(1)
	}
	fmt.Println("✅ No SOLID violations found")
}

func checkFile(path string) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.AllErrors)
	if err != nil {
		return nil // skip files that don't parse
	}

	// For now, use a simple text-based check
	// In production, use go/ast traversal
	content, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	lines := strings.Split(string(content), "\n")

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		// 检查未检查的类型断言: val, _ := expr.(Type)
		if strings.Contains(trimmed, ", _ :=") && strings.Contains(trimmed, ".(") {
			fmt.Fprintf(os.Stderr, "LSP violation: %s:%d: unchecked type assertion\n  %s\n", path, i+1, trimmed)
			violations++
		}
	}

	return nil
}
```

- [ ] **Step 2: 编译验证**

```bash
go build -o /dev/null ./cmd/check-solid/
```

- [ ] **Step 3: 在项目上运行确认无错误**

```bash
go run ./cmd/check-solid/ .
```

- [ ] **Step 4: 提交**

```bash
git add cmd/check-solid/
git commit -m "feat: add check-solid linter for SOLID compliance"
```

---

### Task 2: AST 添加 `TypeName()` 方法

**Files:**
- Modify: `mdl/ast/ast.go`
- Modify: 所有 `mdl/ast/ast_*.go` 文件

**Interfaces:**
- Consumes: 现有 `Statement` 接口
- Produces: `Statement` 接口新增 `TypeName() string` 方法，每个具体类型实现

- [ ] **Step 1: 为 `Statement` 接口添加 `TypeName()`**

编辑 `mdl/ast/ast.go`，在 `isStatement()` 旁添加：

```go
type Statement interface {
	isStatement()
	TypeName() string
}
```

- [ ] **Step 2: 为所有已知语句类型添加 `TypeName()` 实现**

每个 `*Stmt` 类型添加：
```go
func (s *CreateModuleStmt) TypeName() string { return "CreateModule" }
```

对所有语句类型重复。在 `register_stubs.go` 中搜索注册的每个类型（约 100 个）。

```bash
# 查找所有注册的语句类型
rg "r.Register\(&ast\.\w+{}," mdl/executor/register_stubs.go | sed 's/.*&ast\.//;s/{},.*//' | sort -u
```

为每个生成 `TypeName()` 方法。

- [ ] **Step 3: 编译验证**

```bash
go build ./mdl/ast/...
```

- [ ] **Step 4: 提交**

```bash
git add mdl/ast/
git commit -m "refactor: add TypeName() to Statement interface for reflection-free dispatch"
```

---

### Task 3: `model/types.go` 按领域拆分

**Files:**
- Create: `model/module.go`, `model/enumeration.go`, `model/constant.go`, `model/schedule.go`, `model/odata.go`, `model/rest.go`, `model/bizevent.go`, `model/dbconn.go`, `model/transformer.go`, `model/settings.go`, `model/mappings.go`
- Modify: `model/types.go`（保留核心类型）

**Interfaces:**
- Consumes: 现有 `model/types.go` 的全部内容
- Produces: 拆分为多个文件的相同类型，包名保持 `model`

- [ ] **Step 1: 从 `model/types.go` 提取每个领域到独立文件**

`model/module.go`:
```go
// SPDX-License-Identifier: Apache-2.0

package model

// Module represents a Mendix module.
type Module struct {
	BaseElement
	Name                string `json:"name"`
	Documentation       string `json:"documentation,omitempty"`
	Excluded            bool   `json:"excluded,omitempty"`
	FromAppStore        bool   `json:"fromAppStore,omitempty"`
	AppStoreVersion     string `json:"appStoreVersion,omitempty"`
	AppStoreGuid        string `json:"appStoreGuid,omitempty"`
	IsReusableComponent bool   `json:"isReusableComponent,omitempty"`
	DomainModelID       ID     `json:"domainModelId,omitempty"`
	Documents           []ID   `json:"documents,omitempty"`
}

func (m *Module) GetName() string { return m.Name }

// Folder represents a Projects$Folder.
type Folder struct {
	BaseElement
	ContainerID ID     `json:"containerId"`
	Name        string `json:"name"`
}
```

`model/enumeration.go`:
```go
// SPDX-License-Identifier: Apache-2.0

package model

// Enumeration represents a module-level enumeration.
type Enumeration struct {
	BaseElement
	ContainerID      ID                   `json:"containerId"`
	Name             string               `json:"name"`
	Documentation    string               `json:"documentation,omitempty"`
	Excluded         bool                 `json:"excluded,omitempty"`
	ExportLevel      string               `json:"exportLevel,omitempty"`
	EnumerationValues []*EnumerationValue `json:"enumerationValues,omitempty"`
}

func (e *Enumeration) GetName() string         { return e.Name }
func (e *Enumeration) GetContainerID() ID       { return e.ContainerID }

// EnumerationValue represents a value within an enumeration.
type EnumerationValue struct {
	BaseElement
	Name             string `json:"name"`
	Caption          string `json:"caption,omitempty"`
	Image            string `json:"image,omitempty"`
	Weight           int    `json:"weight,omitempty"`
}
```

按此模式为每个领域创建文件，从 `model/types.go` 复制相关类型，确保：
- `model/types.go` 只保留 `ID`, `QualifiedName`, `Point`, `Size`, `Element`, `NamedElement`, `ContainedElement`, `BaseElement`, `Unit`, `UnknownElement`
- 其他文件导入 `model` 包不受影响（同一包名）

- [ ] **Step 2: 编译验证**

```bash
go build ./model/...
go build ./...
```

- [ ] **Step 3: 提交**

```bash
git add model/
git commit -m "refactor: split model/types.go into per-domain files (SRP)"
```

---

### Task 4: `BackendFactory` + 角色接口纯净

**Files:**
- Create: `mdl/backend/factory.go`
- Modify: `mdl/backend/backend.go`（删除 `FullBackend`）
- Modify: `mdl/backend/role.go`（移除 `FullBackend` 嵌入）

**Interfaces:**
- Produces: `backend.BackendFactory` 接口
- Consumes: 现有角色接口

- [ ] **Step 1: 创建 `BackendFactory` 接口**

```go
// SPDX-License-Identifier: Apache-2.0

package backend

// BackendFactory 是构造时工厂。这是唯一知道所有角色实现的类型。
// 业务逻辑代码应该依赖窄角色接口，而非 BackendFactory。
type BackendFactory interface {
	ConnectionBackend

	ModuleLister() ModuleLister
	ModuleWriter() ModuleWriter
	DomainModelReader() DomainModelReader
	DomainModelWriter() DomainModelWriter
	MicroflowReader() MicroflowReader
	MicroflowWriter() MicroflowWriter
	WorkflowReader() WorkflowReader
	WorkflowWriter() WorkflowWriter
	PageReader() PageReader
	PageWriter() PageWriter
	JavaActionReader() JavaActionReader
	JavaActionWriter() JavaActionWriter
	JavaScriptActionReader() JavaScriptActionReader
	JavaScriptActionWriter() JavaScriptActionWriter
	EnumerationReader() EnumerationReader
	EnumerationWriter() EnumerationWriter
	ConstantReader() ConstantReader
	ConstantWriter() ConstantWriter
	SettingsReader() SettingsReader
	SettingsWriter() SettingsWriter
	MappingReader() MappingReader
	MappingWriter() MappingWriter
	UnitReader() UnitReader
	UnitWriter() UnitWriter
	NavigationReader() NavigationReader
	NavigationWriter() NavigationWriter
	ImageCollectionWriter() ImageCollectionWriter
	ServiceLister() ServiceLister
	ServiceWriter() ServiceWriter
	ScheduledEventReader() ScheduledEventReader
	MetadataReader() MetadataReader
	FolderManager() FolderManager
	ModuleSettingsReader() ModuleSettingsReader
	ModuleSettingsWriter() ModuleSettingsWriter
	RenameManager() RenameManager
	SecurityProjectManager() SecurityProjectManager
	SecurityModuleManager() SecurityModuleManager
	SecurityEntityAccessManager() SecurityEntityAccessManager
	PageModelAccess() PageModelAccess
	PageMutationOperator() PageMutationOperator
	WorkflowMutationOperator() WorkflowMutationOperator
	WidgetBuilder() WidgetBuilder
	ScriptTransactionManager() ScriptTransactionManager
	AgentEditorOperator() AgentEditorOperator
}
```

- [ ] **Step 2: 编辑 `backend.go`**

将 `FullBackend` 替换为纯 `ConnectionBackend` 接口：

```go
// SPDX-License-Identifier: Apache-2.0

package backend

// ConnectionBackend 管理项目连接生命周期。
type ConnectionBackend interface {
	Connect(path string) error
	Disconnect() error
	IsConnected() bool
}
```

文件重命名为 `connection.go` 或保留为 `backend.go` 只含连接接口。

- [ ] **Step 3: 编辑 `role.go`**

移除所有角色接口中嵌入的 `FullBackend`。确保每个角色接口是纯接口，不嵌入任何父接口。

- [ ] **Step 4: 编辑 `persistent.go`**

将 `PersistentBackend` 中嵌入的 `FullBackend` 改为 `ConnectionBackend`。

```go
type PersistentBackend interface {
	ConnectionBackend
	Microflows() repos.MicroflowRepository
	Nanoflows() repos.NanoflowRepository
	// ... 其余不变
}
```

- [ ] **Step 5: 编译验证**

```bash
go build ./mdl/backend/...
```

预期：许多编译错误——`MprBackend` 尚未实现 `BackendFactory`。这将在 Task 5 中修复。

- [ ] **Step 6: 提交**

```bash
git add mdl/backend/
git commit -m "refactor: replace FullBackend with BackendFactory + pure role interfaces (ISP)"
```

---

### Task 5: `MprBackend` 拆分为子后端

**Files:**
- Create: `mdl/backend/mpr/mpr_module.go`, `mpr_microflow.go`, `mpr_page.go`, `mpr_entity.go`, `mpr_workflow.go`, `mpr_security.go`, `mpr_enum.go`, `mpr_constant.go`, `mpr_mapping.go`, `mpr_settings.go`, `mpr_navigation.go`, `mpr_image.go`, `mpr_java.go`, `mpr_schedule.go`, `mpr_services.go`
- Modify: `mdl/backend/mpr/backend.go`
- Delete: `mdl/backend/mpr/repos_provider.go`
- Modify: `mdl/backend/mpr/write_helpers.go`, `script_tx.go`, `create_services_v2.go` 等（移除 `concreteWriter` 引用）

**Interfaces:**
- Consumes: `backend.BackendFactory`, 所有角色接口
- Produces: 实现 `BackendFactory` 的瘦身 `MprBackend`

- [ ] **Step 1: 创建 `mpr_module.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/model"
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

type moduleBackend struct {
	reader *modelsdkmpr.Reader
}

func newModuleBackend(r *modelsdkmpr.Reader) *moduleBackend {
	return &moduleBackend{reader: r}
}

func (b *moduleBackend) ListModules() ([]*model.Module, error) {
	return b.reader.ListModules()
}

func (b *moduleBackend) GetModule(id model.ID) (*model.Module, error) {
	return b.reader.GetModule(id)
}

func (b *moduleBackend) GetModuleByName(name string) (*model.Module, error) {
	return b.reader.GetModuleByName(name)
}

// ModuleWriter：创建/更新/删除委托给 ViaModelsdk 辅助函数
// 这些辅助函数从 MprBackend 移动到 moduleBackend
func (b *moduleBackend) CreateModule(module *model.Module) error {
	return createModuleViaWriter(b.reader, module)
}
// ... 其他 ModuleWriter 方法
```

注意：从 `MprBackend` 迁移 `createModuleViaModelsdk`、`updateModuleViaModelsdk` 等辅助函数到对应的子后端。辅助函数不再需要访问 `MprBackend` 全部字段——它们只使用 `reader` 或 `writer`。

- [ ] **Step 2: 为每个领域重复 Step 1 的模式**

每个子后端：
- 直接接收 `*Reader` 或 `*mmpr.Writer`（具体类型），不需要 `concreteWriter()`
- 实现对应的角色接口
- 携带自己的缓存（如有必要）

`mpr_microflow.go`:
```go
type microflowBackend struct {
	writer *mmpr.Writer
	cache  *microflowCache
}

func newMicroflowBackend(w *mmpr.Writer) *microflowBackend {
	return &microflowBackend{
		writer: w,
		cache:  newMicroflowCache(),
	}
}

func (b *microflowBackend) ListMicroflowsGen(ctx context.Context) ([]*genMf.Microflow, error) {
	return b.cache.Get(ctx, func() ([]*genMf.Microflow, error) {
		return mprrepos.NewMicroflowRepository(b.writer).ListAll()
	})
}
```

- [ ] **Step 3: 瘦身 `backend.go`**

移除所有领域特定方法，替换为 `BackendFactory` 访问器：

```go
type MprBackend struct {
	reader *modelsdkmpr.Reader
	writer *mmpr.Writer
	path   string

	moduleBkd    *moduleBackend
	microflowBkd *microflowBackend
	pageBkd      *pageBackend
	entityBkd    *entityBackend
	workflowBkd  *workflowBackend
	securityBkd  *securityBackend
	// ... 全部子后端
}

func (b *MprBackend) Connect(path string) error {
	r, err := modelsdkmpr.OpenWithOptions(path, modelsdkmpr.OpenOptions{ReadOnly: false})
	if err != nil {
		return err
	}
	w := modelsdkmpr.NewWriterWithReader(r)
	b.reader = r
	b.writer = w.(*mmpr.Writer)
	b.path = path
	// 急切创建所有子后端 —— 没有 initSubBackends()
	b.moduleBkd    = newModuleBackend(r)
	b.microflowBkd = newMicroflowBackend(b.writer)
	b.pageBkd      = newPageBackend(b.writer)
	b.entityBkd    = newEntityBackend(b.writer)
	b.workflowBkd  = newWorkflowBackend(b.writer)
	b.securityBkd  = newSecurityBackend(b.writer)
	// ...
	return nil
}

func (b *MprBackend) ModuleLister()    backend.ModuleLister    { return b.moduleBkd }
func (b *MprBackend) ModuleWriter()    backend.ModuleWriter    { return b.moduleBkd }
func (b *MprBackend) MicroflowReader() backend.MicroflowReader { return b.microflowBkd }
func (b *MprBackend) MicroflowWriter() backend.MicroflowWriter { return b.microflowBkd }
// ...
```

- [ ] **Step 4: 删除 `repos_provider.go`**

整个文件删除。`Microflows()`, `Nanoflows()` 等方法要么移到子后端，要么直接删除（如果不再需要）。

- [ ] **Step 5: 更新 `PersistentBackend` 实现**

`MprBackend` 上的 `PersistentBackend` 访问器直接委托给子后端或 writer。

- [ ] **Step 6: 编译验证**

```bash
go build ./mdl/backend/...
go build ./mdl/...
```

- [ ] **Step 7: 运行测试**

```bash
go test ./mdl/backend/... ./mdl/executor/... 2>&1 | tail -20
```

- [ ] **Step 8: 提交**

```bash
git add mdl/backend/mpr/
git commit -m "refactor: split MprBackend into focused sub-backends (SRP, remove initSubBackends/concreteWriter)"
```

---

### Task 6: 缓存分解

**Files:**
- Create: `mdl/backend/mpr/cache_microflow.go`, `cache_page.go`, `cache_entity.go`, `cache_workflow.go`, `cache_security.go`
- Delete: `mdl/executor/executor.go` 中的 `executorCache`

**Interfaces:**
- Produces: 每个文档类型一个独立缓存类型

- [ ] **Step 1: 创建 `cache_microflow.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"context"
	"sync"
)

type microflowCache struct {
	mu    sync.RWMutex
	items any // 用具体类型
	valid bool
}

func newMicroflowCache() *microflowCache {
	return &microflowCache{}
}

func (c *microflowCache) get(ctx context.Context, loader func(context.Context) (any, error)) (any, error) {
	c.mu.RLock()
	if c.valid {
		defer c.mu.RUnlock()
		return c.items, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.valid {
		return c.items, nil
	}
	items, err := loader(ctx)
	if err != nil {
		return nil, err
	}
	c.items = items
	c.valid = true
	return c.items, nil
}

func (c *microflowCache) invalidate() {
	c.mu.Lock()
	c.valid = false
	c.mu.Unlock()
}
```

- [ ] **Step 2: 为每个领域创建专用缓存类型**

每个缓存文件包含该领域的 `get`/`invalidate` 方法，使用正确的返回类型。

- [ ] **Step 3: 从 `executor.go` 中移除 `executorCache` 结构体**

从 `mdl/executor/executor.go` 中删除 `executorCache` 结构体、所有访问器方法、以及 `invalidateAllDocumentCaches`、`rememberDroppedMicroflow`、`consumeDroppedMicroflow` 等辅助函数。

这些功能要么移到子后端的缓存中，要么直接在需要的地方内联。

- [ ] **Step 4: 编译验证**

```bash
go build ./mdl/backend/mpr/...
go build ./mdl/executor/...
```

- [ ] **Step 5: 提交**

```bash
git add mdl/backend/mpr/cache_*.go mdl/executor/executor.go
git commit -m "refactor: decompose executorCache into per-domain caches (SRP)"
```

---

### Task 7: `ExecContext` 消除 + `Registry` 重构

**Files:**
- Delete: `mdl/executor/exec_context.go`
- Modify: `mdl/executor/registry.go`
- Modify: `mdl/executor/executor.go`（瘦身）
- Modify: `mdl/executor/builder.go`（适配）

**Interfaces:**
- Consumes: `backend.BackendFactory`, `backend.ModuleLister` 等角色接口
- Produces: 纯闭包处理函数

- [ ] **Step 1: 重写 `registry.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
)

// StmtHandler 执行一条语句。
type StmtHandler func(ctx context.Context, stmt ast.Statement) error

// Registry 将语句类型名映射到处理函数。
type Registry struct {
	handlers map[string]StmtHandler
}

func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[string]StmtHandler),
	}
}

func (r *Registry) Register(typeName string, handler StmtHandler) {
	if _, exists := r.handlers[typeName]; exists {
		return // 静默忽略重复
	}
	r.handlers[typeName] = handler
}

func (r *Registry) Lookup(stmt ast.Statement) (StmtHandler, bool) {
	h, ok := r.handlers[stmt.TypeName()]
	return h, ok
}

func (r *Registry) HandlerCount() int {
	return len(r.handlers)
}
```

- [ ] **Step 2: 删除 `exec_context.go`**

删除整个文件。移出任何仍然需要的辅助函数（如 `getDomainModelGenCached` 等）到将使用它们的处理函数文件。

- [ ] **Step 3: 瘦身 `executor.go`**

```go
type Executor struct {
	registry  *Registry
	guard     *outputGuard
	perfStats []perfStmt
	logger    Logger
}

func New(output io.Writer) *Executor {
	guard := newOutputGuard(output, maxOutputLines)
	return &Executor{
		guard:    guard,
		registry: NewRegistry(),
	}
}

func (e *Executor) Execute(ctx context.Context, stmt ast.Statement) error {
	start := time.Now()
	e.guard.reset()

	if e.logger != nil {
		defer func() {
			e.logger.Command(stmt.TypeName(), stmtSummary(stmt), time.Since(start), err)
		}()
	}

	h, ok := e.registry.Lookup(stmt)
	if !ok {
		return mdlerrors.NewUnsupported(fmt.Sprintf("unhandled statement type %s", stmt.TypeName()))
	}
	err := h(ctx, stmt)

	elapsed := time.Since(start)
	e.perfStats = append(e.perfStats, perfStmt{
		Type:     stmt.TypeName(),
		Summary:  stmtSummary(stmt),
		Duration: elapsed,
		Err:      err != nil,
	})
	if err != nil {
		err = fmt.Errorf("%w (duration: %v)", err, elapsed)
	}
	return err
}
```

移除：`SetBackend`、`SetBackendFactory`、`SetQuiet`、`SetFormat`、`SetLogger`、`SetProgressOut`、`Catalog`、`Graph`、`BuildGraph`、`IsConnected`、`Backend`、`Close` 等方法。这些不再是 Executor 的职责。

注意：保留 `ExecuteProgram` 以批处理语句，但移除脚本事务逻辑（由调用方处理）。

- [ ] **Step 4: 编译验证**

```bash
go build ./mdl/executor/...
```

预期：许多编译错误——处理函数仍引用 `*ExecContext`。将在 Task 8-10 中修复。

- [ ] **Step 5: 提交**

```bash
git add mdl/executor/
git commit -m "refactor: eliminate ExecContext, refactor Registry (ISP/DIP/OCP)"
```

---

### Task 8: 创建新处理函数注册 + 实现

**Files:**
- Create: `mdl/executor/handlers/` 目录 + 所有 `handler_*.go` 文件
- Delete: `mdl/executor/register_stubs.go`

**Interfaces:**
- Consumes: `backend.BackendFactory`（用于获取角色），角色接口
- Produces: 通过闭包捕获依赖的处理函数

- [ ] **Step 1: 创建 `handlers/handler_module.go`**

第一个处理函数——展示新模式：

```go
// SPDX-License-Identifier: Apache-2.0

package handlers

import (
	"context"
	"fmt"
	"io"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/executor"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
)

func RegisterModuleHandlers(
	reg *executor.Registry,
	lister backend.ModuleLister,
	writer backend.ModuleWriter,
	output io.Writer,
) {
	reg.Register("CreateModule", func(ctx context.Context, stmt ast.Statement) error {
		return execCreateModule(ctx, stmt.(*ast.CreateModuleStmt), lister, writer, output)
	})
	reg.Register("DropModule", func(ctx context.Context, stmt ast.Statement) error {
		return execDropModule(ctx, stmt.(*ast.DropModuleStmt), lister, writer, output)
	})
}

func execCreateModule(ctx context.Context, s *ast.CreateModuleStmt, lister backend.ModuleLister, writer backend.ModuleWriter, output io.Writer) error {
	if s == nil || s.Name == "" {
		return mdlerrors.NewValidation("CREATE MODULE requires a name")
	}
	modules, err := lister.ListModules()
	if err != nil {
		return mdlerrors.NewBackend("list modules", err)
	}
	for _, m := range modules {
		if m.Name == s.Name {
			fmt.Fprintf(output, "Module '%s' already exists\n", s.Name)
			return nil
		}
	}
	module := &model.Module{Name: s.Name}
	if err := writer.CreateModule(module); err != nil {
		return mdlerrors.NewBackend("create module", err)
	}
	fmt.Fprintf(output, "Created module: %s\n", s.Name)
	return nil
}

func execDropModule(ctx context.Context, s *ast.DropModuleStmt, lister backend.ModuleLister, writer backend.ModuleWriter, output io.Writer) error {
	if s == nil || s.Name == "" {
		return mdlerrors.NewValidation("DROP MODULE requires a name")
	}
	// ... 实现逻辑从 cmd_modules.go 迁移 ...
}
```

- [ ] **Step 2: 为每个领域创建 `handler_*.go`**

从对应 `cmd_*.go` 文件迁移处理逻辑。模式：
1. 声明注册函数，捕获精确的角色接口作为参数
2. 处理函数通过 `Register("TypeName", ...)` 注册
3. 执行函数接收精确的参数

注意：新处理函数使用 `context.Context` 而非 `*ExecContext`，所以所有内部 `ctx.Backend.Xxx` 调用被替换为函数参数传递的角色接口。

- [ ] **Step 3: 删除 `register_stubs.go`**

- [ ] **Step 4: 编译验证**

```bash
go build ./mdl/executor/handlers/...
go build ./mdl/executor/...
```

- [ ] **Step 5: 提交**

```bash
git add mdl/executor/handlers/
git rm mdl/executor/register_stubs.go
git commit -m "refactor: create handler registration files, remove register_stubs.go (OCP)"
```

---

### Task 9: 更新 CLI 层和 Builder

**Files:**
- Modify: `mdl/executor/builder.go`
- Modify: `cmd/mxcli/main.go`
- Modify: 所有其他 `cmd/*/main.go`

**Interfaces:**
- Consumes: 新 `Executor`、`Registry`、`BackendFactory`
- Produces: 可编译的 CLI

- [ ] **Step 1: 重写 `builder.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"io"
	"os"

	"github.com/mendixlabs/mxcli/mdl/backend"
)

type Builder struct {
	out     io.Writer
	factory backend.BackendFactory
	quiet   bool
}

func Build() *Builder {
	return &Builder{out: os.Stdout}
}

func (b *Builder) Out(w io.Writer) *Builder          { b.out = w; return b }
func (b *Builder) WithFactory(f BackendFactory) *Builder { b.factory = f; return b }
func (b *Builder) Quiet() *Builder                    { b.quiet = true; return b }

func (b *Builder) Create() *Executor {
	e := New(b.out)
	e.quiet = b.quiet
	return e
}
```

注意：`Builder` 不再持有 backend。backend 由调用方工厂创建，角色通过注册函数传递给 handler。

- [ ] **Step 2: 更新 `cmd/mxcli/main.go`**

```go
func buildExec() *executor.Executor {
	b := executor.Build().Out(os.Stdout)
	return b.Create()
}

func init() {
	// 注册所有处理函数
	reg := executor.GlobalRegistry() // 或通过 Executor 传递
	handlers.RegisterModuleHandlers(reg, /* 角色待定 */)
	handlers.RegisterMicroflowHandlers(reg, /* 角色待定 */)
	// ...
}
```

注意：由于 Big Bang 尚在构建中，角色接口在连接时注入。初始化时先注册 type→handler 签名映射，稍后当 `BackendFactory` 连接时，将角色绑定到闭包。

- [ ] **Step 3: 更新所有 `cmd/*/main.go`**

- [ ] **Step 4: 编译验证**

```bash
go build ./cmd/...
```

- [ ] **Step 5: 提交**

```bash
git add cmd/ mdl/executor/builder.go
git commit -m "refactor: update CLI layer and Builder for new architecture"
```

---

### Task 10: LSP 违规修复

**Files:**
- Modify: `mdl/executor/cmd_associations.go`, `cmd_odata.go`, `cmd_workflows_write_v2.go`
- Modify: 所有 `check-solid` 工具报告的文件

**Interfaces:**
- 无新接口

- [ ] **Step 1: 运行 `check-solid` 找到所有违规**

```bash
go run ./cmd/check-solid/ .
```

- [ ] **Step 2: 修复每个未检查的类型断言**

`cmd_associations.go` 第 417 行附近：
```go
// Before:
db, _ := elem.(*genDm.Entity)

// After:
db, ok := elem.(*genDm.Entity)
if !ok {
    return mdlerrors.NewTypeMismatch("Entity", fmt.Sprintf("%T", elem))
}
```

`cmd_odata.go` 第 629、640、730 行附近：
```go
oc, ok := mf.ObjectCollection().(*genMf.ObjectCollection)
if !ok {
    continue
}
```

`cmd_workflows_write_v2.go` 第 271、940-944、1126-1130 行附近：同样的模式。

- [ ] **Step 3: 重新运行 `check-solid` 确认零违规**

```bash
go run ./cmd/check-solid/ .
# 预期: ✅ No SOLID violations found
```

- [ ] **Step 4: 提交**

```bash
git add mdl/executor/cmd_*.go
git commit -m "fix: add checked type assertions for LSP compliance"
```

---

### Task 11: 最终验证

**Files:**
- 无代码更改

- [ ] **Step 1: 完整编译**

```bash
go build ./...
```

- [ ] **Step 2: 运行 vet**

```bash
go vet ./...
```

- [ ] **Step 3: 运行测试套件**

```bash
go test ./mdl/... 2>&1 | tail -30
```

- [ ] **Step 4: 运行 SOLID 检查**

```bash
go run ./cmd/check-solid/ .
```

- [ ] **Step 5: 如果任何测试失败，修复并重复**

- [ ] **Step 6: 提交最终验证**

```bash
git add -A
git commit -m "chore: final verification - all tests passing, SOLID violations eliminated"
```
