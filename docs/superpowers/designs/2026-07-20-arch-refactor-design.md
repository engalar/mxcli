# 架构重构设计文档 — mxcli

> 基于 2026-07-20 架构审计。
> 对应 REFACTOR_PLAN.md 中的阶段，但按 DDD/分层/模式/SOLID/DRY/性能分别展开。
> 每个设计独立可实施，互不阻塞。

---

## 目录

1. [FullBackend 解体 — 窄接口注入](#1-fullbackend-解体--窄接口注入)
2. [双依赖容器合并 — ExecContext 退役](#2-双依赖容器合并--execcontext-退役)
3. [MprBackend 分解 — 域专属后端](#3-mprbackend-分解--域专属后端)
4. [repos/ 包激活 — 仓储层统一](#4-repos-包激活--仓储层统一)
5. [executorCache 拆分 — 按域失效](#5-executorcache-拆分--按域失效)
6. [附录：Cobra 命令层规范化](#6-附录cobra-命令层规范化)

---

## 1. FullBackend 解体 — 窄接口注入

### 问题

`FullBackend`（`mdl/backend/backend.go:19`）组合了 50 个接口，但审计发现实际 handler 代码中只有少数几个地方还在使用 `deps.Backend.Xxx()`：

| 调用 | 位置 | 数量 |
|------|------|------|
| `InvalidateCache()` | `handlers_future.go:96,319`、`domainmodel/handler.go:78,88` | 4 |
| `UpdateWorkflowGen()` | `handlers_future.go:881` | 1 |
| `CreateWorkflowGen()` | `handlers_future.go:885` | 1 |
| `DeleteWorkflow()` | `handlers_future.go:1201` | 1 |
| `GetModuleSecurityGen()` | `handlers_future.go:395` | 1 |
| `IsConnected()` | `executor.go:968` | 1 |
| 类型断言 | `executor.go:862`、`executor_connect.go:48`、`import_project.go:97` | 3 |

**共 12 处实际依赖**（其余 36 处为注释或测试引用）。这说明窄接口注入已经推进了 ~90%，剩下的主要是 `InvalidateCache()`（基础设施关注点）和少量域操作。

### 设计

**目标：** 删除 `HandlerDeps.Backend` 字段，所有 handler 通过窄角色接口操作。

#### 第 1 步：InvalidateCache 提取

`InvalidateCache()` 不是域操作——它属于连接生命周期。创建一个新的基础设施角色接口：

```go
// mdl/backend/infrastructure.go
type CacheInvalidator interface {
    InvalidateCache()
}
```

在 `HandlerDeps` 中替换 `Backend` 为：

```go
type HandlerDeps struct {
    // ❌ 删除: Backend backend.FullBackend
    // ✅ 新增:
    ConnectionManager   backend.ConnectionManager  // Already exists
    CacheInvalidator    backend.CacheInvalidator
    // ...
}
```

`MockBackend` 实现：

```go
// mdl/backend/mock/mock_backend.go
type MockBackend struct {
    // ...
    InvalidateCacheFunc func()
}
func (m *MockBackend) InvalidateCache() {
    if m.InvalidateCacheFunc != nil {
        m.InvalidateCacheFunc()
    }
}
var _ backend.CacheInvalidator = (*MockBackend)(nil)
```

#### 第 2 步：Workflow 操作迁移

`deps.Backend.UpdateWorkflowGen()` → `deps.WorkflowWriter.UpdateWorkflowGen()`
`deps.Backend.CreateWorkflowGen()` → `deps.WorkflowWriter.CreateWorkflowGen()`
`deps.Backend.DeleteWorkflow()` → `deps.WorkflowWriter.DeleteWorkflow()`

这些已经在 `handler_deps.go:38-39` 中作为 `WorkflowWriter` 接口存在。只需在 `handlers_future.go:881,885,1201` 中把 `deps.Backend` 替换为 `deps.WorkflowWriter`。

#### 第 3 步：Security 操作迁移

`deps.Backend.GetModuleSecurityGen()` → `deps.SecurityModuleManager.GetModuleSecurityGen()`

已在 `handler_deps.go:64` 中存在为 `SecurityModuleManager`。

#### 第 4 步：连接操作迁移

`ctx.Backend.IsConnected()` (executor.go:968) → `ctx.ConnectionManager.IsConnected()`

已在 `handler_deps.go:28` 中存在。

#### 第 5 步：类型断言迁移

```go
// import_project.go:97 — 使用类型守卫
if bufBE, ok := ctx.Backend.(backend.ImportBufferBackend); ok {
```
→ 提取为 `HandlerDeps.ImportBuffer` 字段，或其他窄接口。

```go
// executor.go:862 — initRoles 中的回退逻辑
bf, _ = ctx.Backend.(backend.BackendFactory)
```
→ 只在 `ExecContext` 中保留，不阻塞 `HandlerDeps` 的删除。

```go
// executor_connect.go:48 — 仅在设置 Backend 时使用
if bg, ok := ctx.Backend.(interface{ Build() error }); ok {
```
→ 提取为具名接口。

#### 第 6 步：删除 HandlerDeps.Backend

当以上 5 步完成后，删除 `handler_deps.go:26` 的 `Backend backend.FullBackend`。编译会确认没有遗漏引用。

### 迁移顺序与风险

```
步骤 1 (InvalidateCache) ──► 步骤 2 (Workflow) ──► 步骤 3 (Security)
                                                      │
                                                      ▼
                                 步骤 4 (Connection) ──► 步骤 5 (类型断言)
                                                          │
                                                          ▼
                                              步骤 6 (删除 Backend 字段)
```

每个步骤都可独立提交。步骤 1 和 5 风险最低（机械替换）。步骤 6 是最终验证——编译通过即完成。

### 影响文件

| 文件 | 影响 |
|------|------|
| `mdl/backend/backend.go` | 删除 `FullBackend` 接口（或标记为仅供 `MprBackend` 编译期检查） |
| `mdl/backend/infrastructure.go` | 新增 `CacheInvalidator` 接口 |
| `mdl/backend/mock/mock_backend.go` | 新增 `InvalidateCacheFunc` + 编译期检查 |
| `mdl/executor/handler_deps.go` | 删除 `Backend` 字段，新增 `CacheInvalidator` |
| `mdl/executor/handlers_future.go` | 替换 6 处 `deps.Backend.Xxx()` |
| `mdl/executor/domainmodel/handler.go` | 替换 2 处 `ectx.Backend.InvalidateCache()` |
| `mdl/executor/executor.go` | 删除 `Backend` 回退逻辑 |
| `mdl/executor/executor_connect.go` | 类型断言提取 |
| `mdl/executor/import_project.go` | 类型断言提取 |

---

## 2. 双依赖容器合并 — ExecContext 退役

### 问题

当前系统有两个依赖容器：

1. **`HandlerDeps`** （`handler_deps.go:20`）— 扁平的 50+ 字段，是新的目标
2. **`ExecContext`** （`executor.go:783`）— 5 个嵌入式子结构体 + 40+ 角色字段 + `context.Context` 嵌入

它们之间有两个双向桥接函数，共 240+ 行样板代码：

| 函数 | 方向 | 行数 | 位置 |
|------|------|------|------|
| `execContextToDeps()` | ExecContext → HandlerDeps | 85 | `executor_dispatch.go:889-973` |
| `NewExecContext()` | HandlerDeps → ExecContext | 94 | `handlers_future.go:2901-2994` |

`ExecContext` 的字段与 `HandlerDeps` 几乎完全相同。维护两者并行意味着每次新增一个后端角色接口，就需要在两个结构体和两个桥接函数中同步添加。

### 设计

**目标：** 删除 `ExecContext` 结构体、`ExecRepos`/`ExecIO`/`ExecSession`/`ExecConnection`/`ExecCallbacks` 子结构体，以及两个桥接函数。所有 handler 统一使用 `*HandlerDeps`。

#### 当前使用情况统计

审计发现 `ExecContext` 的引用模式：

| 使用模式 | 数量 | 举例 |
|----------|------|------|
| 通过 `ectx` 变量访问 | ~10 处 | `domainmodel/handler.go`、旧 handler |
| 桥接到 `HandlerDeps` | 1 处 | `execContextToDeps()` 调用者 |
| 从 `HandlerDeps` 桥接 | ~30 处 | `handlers_future.go` 中的 `NewExecContext()` 调用 |

实际接入 `HandlerDeps` 的 handler 数量已经远超 `ExecContext`。约 30 个旧 handler 通过 `NewExecContext()` 桥接调用。

#### 迁移策略

**第 1 步：将 ExecContext 子结构体压平到 HandlerDeps**

`ExecRepos`、`ExecIO`、`ExecSession`、`ExecConnection`、`ExecCallbacks` 的字段已经在 `HandlerDeps` 中存在。删除子结构体类型定义，直接使用 `HandlerDeps`。

```go
// 删除：
type ExecRepos struct { ... }
type ExecIO struct { ... }
type ExecSession struct { ... }
type ExecConnection struct { ... }
type ExecCallbacks struct { ... }

// ExecContext 保留但不再嵌入子结构体
type ExecContext struct {
    context.Context
    // 字段全部内联，不再有 ExecRepos 等
}
```

这一步减少 80 行定义，不更改 handler 代码。

**第 2 步：逐个迁移旧 handler**

迁移约 30 个使用 `NewExecContext()` 的旧 handler。每个 handler 的迁移模式：

```go
// Before: 桥接到旧容器
r.RegisterFuture("CreateEntity", func(ctx context.Context, stmt ast.Statement) error {
    ectx := NewExecContext(ctx, deps)
    return ExecCreateEntity(ectx, stmt.(*ast.CreateEntityStmt))
})

// After: 直接使用 HandlerDeps
r.RegisterFuture("CreateEntity", func(ctx context.Context, stmt ast.Statement) error {
    return ExecCreateEntityFn(ctx, stmt.(*ast.CreateEntityStmt), deps)
})
```

每个旧 handler 函数签名从 `(ectx *ExecContext, stmt)` 改为 `(ctx context.Context, stmt, deps *HandlerDeps)`。

**第 3 步：删除桥接函数**

当 0 个 handler 使用 `NewExecContext()` 且 0 处调用 `execContextToDeps()` 时：

```go
// 删除：
func execContextToDeps(ectx *ExecContext) *HandlerDeps  // executor_dispatch.go:889-973
func NewExecContext(ctx context.Context, deps *HandlerDeps) *ExecContext  // handlers_future.go:2901-2994
```

**第 4 步：删除 ExecContext**

当 0 个引用指向 `ExecContext` 时，删除整个结构体和 `initRoles()` 方法（`executor.go:783-960`）。

### 旧 handler 迁移清单

通过 grep 找到需要迁移的 handler：

```
rg "NewExecContext" mdl/executor/handlers_future.go
```

预期约 30 个 handler，逐个迁移。每个 handler 的迁移公式固定：

| 旧 | 新 |
|----|----|
| 函数接受 `(ectx *ExecContext, stmt)` | 函数接受 `(ctx context.Context, stmt, deps *HandlerDeps)` |
| `ectx.Xxx` 访问 | `deps.Xxx` 访问 |
| `ectx.Backend.Xxx` | `deps.RoleInterface.Xxx` |
| `ectx.Context` | `ctx` |
| 通过 `NewExecContext()` 注册 | 直接注册为 `(ctx, stmt) error { return fn(ctx, stmt, deps) }` |

### 收益

| 指标 | 变化 |
|------|------|
| 结构体数量 (HandlerDeps + ExecContext + 5 子结构体) | 7 → 1 |
| 桥接样板代码 | 240 行 → 0 行 |
| 重复字段 | ~50 个 → 0 个 |
| 每次新后端接口的成本 | 两处 + 两桥接 → 一处 |

---

## 3. MprBackend 分解 — 域专属后端

### 问题

`MprBackend` 是一个单结构体，在 `mdl/backend/mpr/` 的 100 个文件中实现了 30+ 接口。它是一个"神对象"——持有所有依赖（`*mpr.Reader`、`*mpr.Writer`、`*modelsdk.Model`、各类缓存/构建器）但任何域实现都只使用其中一小部分。

当前结构：

```
mdl/backend/mpr/
  100 个文件
  全部属于 package mprbackend
  全部在同一个 MprBackend struct 上加方法
  所有域共享同一个 Reader/Writer 连接
```

### 设计

**目标：** 将每个域的实现提取到独立的 struct 类型中，每个只接受它需要的依赖。

#### 第 1 步：识别域边界

基于当前文件组织结构，可以自然提取以下域后端：

| 域 | 当前文件 | 方法数 | 依赖 |
|-----|-----------|---------|------|
| Module | `modules.go`, `repos_provider.go` | 8 | `*mpr.Reader` |
| Entity/Domain model | `domainmodel.go`, `domainmodel_v2.go`, `domainmodel_layout.go` | 12 | `*mpr.Reader`, `*mpr.Writer`, `*modelsdk.Model` |
| Microflow | `services_create_v2.go`, `convert.go`, `convert_reader.go` | 15 | `*mpr.Reader`, `*mpr.Writer` |
| Page | `page_model.go`, `page_mutator.go`, `page_mutator_v2.go` | 20 | `*mpr.Reader`, `*mpr.Writer`, `widgetBuilder` |
| Workflow | `workflow_mutator.go`, `workflow_mutator_v2.go` | 15 | `*mpr.Reader`, `*mpr.Writer` |
| Security | `security.go`, `security_module.go`, `security_project.go`, `security_entity_access.go` | 10 | `*mpr.Reader`, `*mpr.Writer` |
| Java | `java_files.go`, `java_source_v2.go` | 8 | `*mpr.Reader`, `*mpr.Writer` |
| Settings | `settings.go`, `settings_legacy.go` | 6 | `*mpr.Reader` |
| Navigation | `navigation.go`, `navigation_legacy.go` | 4 | `*mpr.Reader`, `*mpr.Writer` |
| Image | `image_legacy.go` | 4 | `*mpr.Reader`, `*mpr.Writer` |
| Agent editor | `agenteditor.go` | 6 | `*mpr.Reader`, `*mpr.Writer` |
| Translation | `translation_backend.go`, `translation_microflow.go`, `translation_page_mutator.go`, `translation_writer.go` | 12 | `*mpr.Reader` 等 |

#### 第 2 步：提取模式

每个域后端接收它需要的窄依赖集。以 Module + Entity 为例：

```go
// mdl/backend/mpr/mpr_module.go
type moduleBackend struct {
    reader *modelsdkmpr.Reader
}

func (b *moduleBackend) ListModules() ([]*model.Module, error) {
    return b.reader.ListModules()
}
func (b *moduleBackend) GetModule(id model.ID) (*model.Module, error) {
    return b.reader.ListModulesByID(id)
}
// ... ModuleWriter, ModuleLister 等

// mdl/backend/mpr/mpr_entity.go
type entityBackend struct {
    reader  *modelsdkmpr.Reader
    writer  *modelsdkmpr.Writer
    mdl     *modelsdk.Model
    dmUtil  *dmUtil  // 域名共用工具（布局计算等）
}

func (b *entityBackend) ListDomainModelsGen() ([]*genDm.DomainModel, error) {
    return b.mdl.AllOfType("DomainModels$DomainModel")
}
// ... DomainModelReader, DomainModelWriter 等
```

#### 第 3 步：Facade 模式保持向后兼容

`MprBackend` 保留为组合 facade，将所有域后端组合起来：

```go
// mdl/backend/mpr/backend.go
type MprBackend struct {
    module    *moduleBackend
    entity    *entityBackend
    microflow *microflowBackend
    page      *pageBackend
    workflow  *workflowBackend
    security  *securityBackend
    // ... 更多
}

func NewMprBackend(reader *modelsdkmpr.Reader, writer *modelsdkmpr.Writer, mdl *modelsdk.Model) *MprBackend {
    return &MprBackend{
        module:    &moduleBackend{reader: reader},
        entity:    &entityBackend{reader: reader, writer: writer, mdl: mdl, dmUtil: newDMUtil(reader, mdl)},
        microflow: &microflowBackend{reader: reader, writer: writer},
        // ...
    }
}

// Facade 方法——每个只是委托给域后端：
func (b *MprBackend) ListModules() ([]*model.Module, error) {
    return b.module.ListModules()
}
func (b *MprBackend) ListDomainModelsGen() ([]*genDm.DomainModel, error) {
    return b.entity.ListDomainModelsGen()
}
// ...
```

`MprBackend` 仍然满足 `FullBackend` 接口，所有现有调用者无感知。

#### 第 4 步：BackendFactory 自动适配

`BackendFactory`（`backend/factory.go:8`）目前有 30 个工厂方法。提取后，每个工厂方法直接返回对应的域后端：

```go
// mprBackendFactory 实现 BackendFactory
func (f *mprBackendFactory) ModuleLister() backend.ModuleLister {
    return f.impl.module
}
func (f *mprBackendFactory) DomainModelReader() backend.DomainModelReader {
    return f.impl.entity
}
// ...
```

这使得 `initRoles()` 的填充变成 `deps.ModuleLister = bf.ModuleLister()` —— 每个角色接口由对应的域后端提供。

### 文件结构变更

```
Before:                          After:
mdl/backend/mpr/                 mdl/backend/mpr/
  backend.go     (MprBackend)       backend.go        (MprBackend facade + NewMprBackend)
  domainmodel.go                   mpr_module.go      (moduleBackend)
  domainmodel_v2.go                mpr_entity.go      (entityBackend)
  services_create_v2.go            mpr_microflow.go   (microflowBackend)
  page_model.go                    mpr_page.go        (pageBackend)
  page_mutator.go                  mpr_workflow.go    (workflowBackend)
  ... (100 文件)                    mpr_security.go    (securityBackend)
                                   mpr_java.go        (javaBackend)
                                   mpr_settings.go    (settingsBackend)
                                   mpr_navigation.go  (navigationBackend)
                                   mpr_image.go       (imageBackend)
                                   mpr_agent.go       (agentEditorBackend)
                                   mpr_translation.go (translationBackend)
                                   mpr_builder.go     (widgetBuilder + datagridBuilder)
                                   mpr_infra.go       (script/transaction/import buffer)
                                   ... (约 15 文件)
```

### 收益

| 指标 | 变化 |
|------|------|
| 每个域的文件数量 | 分散在 100 文件中 → 每个域 1-2 文件 |
| 构造函数可见性 | 隐式依赖 → 显式依赖注入 |
| 测试难度 | 需要 mock 整个 MprBackend → 可以只 mock 需要的域后端 |
| 编译并行度 | 单结构体 → 独立结构体，编译器可并行处理 |

---

## 4. repos/ 包激活 — 仓储层统一

### 问题

`mdl/repos/`（29 个文件）定义了干净的仓库接口，但被 `mdl/backend/` 中 30+ 个 `*Reader/Writer` 接口黯然失色。两个层共存且重叠：

| 概念 | backend/ 角色接口 | repos/ 仓库接口 |
|------|-------------------|-----------------|
| Entity 查询 | `DomainModelReader.ListDomainModelsGen()` | `DomainModelRepository.ListEntities()` |
| Microflow 查询 | `MicroflowReader.ListMicroflowsGen()` | `MicroflowRepository.List()` |
| Page 查询 | `PageReader`/`PageWriter` | `PageRepository.List()` |

`repos/` 的 `doc.go:22` 明确说："Stage 2 定义所有域接口。Microflows + Pages 在 `mdl/backend/mpr/repos` 中获得完整实现。其余是签名级别的存根，标记为 `// TODO Stage 3 cutover`。"

实际上，repos 层被设计为对 backend *Reader/Writer 的替代，但从未完全部署。Handler 代码同时使用两者：一部分通过 `deps.MicroflowRepo.List()`，一部分通过 `deps.MicroflowReader.ListMicroflowsGen()`。

### 设计

**目标：** 消除 backend/role.go 中的 `*Reader/Writer` 接口与 repos/ 接口之间的功能重叠。统一到 repos/ 作为 executor 访问数据的唯一方式。

#### 差异分析

当前 repo 接口定义在 `repos/` 目录中，但 `backend/role.go` 接口直接被 executor handler 使用。两者之间的关键区别：

| backend/ 角色接口 | repos/ 接口 | 差异 |
|-------------------|-------------|------|
| `DomainModelReader.ListDomainModelsGen()` 返回 `[]*genDm.DomainModel` | `DomainModelRepository` 定义在 `repos/domainmodels.go` | 签名不同，backend 返回 SDK gen 类型，repos 可能做适配 |
| `MicroflowReader.ListMicroflowsGen()` 返回 `[]*genMf.Microflow` | `MicroflowRepository.List() context []*genMf.Microflow` | `repos` 返回相同类型但多了一个 `context.Context` |
| `ModuleLister.ListModules()` | `ModuleRepository` | repos 侧重缓存和延迟加载 |

实际操作上，backend 接口被设计为直接对应 BSON 操作（读/写单元），而 repos 接口被设计为对应域操作（获取实体列表、按名称查找）。

#### 统一方案

**移除 backend/ 中与 repos 重叠的 *Reader/*Writer，repos/ 持有纯查询接口，backend 持有纯变异接口。**

```go
// Before:
// backend/role.go — 读+写混合
type DomainModelReader interface {
    ListDomainModelsGen() ([]*genDm.DomainModel, error)
    GetDomainModelGen(moduleID model.ID) (*genDm.DomainModel, error)
    GetEntityIDByQualifiedName(qualifiedName string) (element.ID, error)
}

// After:
// repos/domainmodels.go — 只含查询
type DomainModelRepository interface {
    ListDomainModels(ctx context.Context) ([]*genDm.DomainModel, error)
    GetDomainModelByModule(ctx context.Context, moduleID model.ID) (*genDm.DomainModel, error)
    GetEntityByQualifiedName(ctx context.Context, qn string) (*genDm.Entity, error)
}

// backend/role.go — 只含写入
type DomainModelWriter interface {
    UpdateDomainModelGen(dm *genDm.DomainModel) error
    CreateEntityGen(domainModelID model.ID, entity *genDm.Entity) error
    DeleteEntity(domainModelID model.ID, entityID model.ID) error
    // ... 只有变异
}
```

#### 迁移模式

对于使用后端角色接口进行**查询**的 handler：

```go
// Before:
entities, err := deps.DomainModelReader.ListDomainModelsGen()

// After:
entities, err := deps.DomainModels.ListDomainModels(ctx)
```

对于使用后端角色接口进行**写入**的 handler，保持不变（只保留在 backend/ 中）。

#### 具体迁移清单

| 当前 `deps.*` | 迁移目标 | 处理 |
|---------------|----------|------|
| `deps.ModuleLister` | `deps.Modules` (repos) | 查询迁移到 repos |
| `deps.DomainModelReader` | `deps.DomainModels` (repos) | 查询迁移到 repos |
| `deps.MicroflowReader` | `deps.MicroflowRepo` (repos) | 查询迁移到 repos |
| `deps.WorkflowReader` | `deps.WorkflowRepo` (repos) | 查询迁移到 repos |
| `deps.PageReader` | `deps.PageRepo` (repos) | 查询迁移到 repos |
| `deps.EnumerationReader` | repos 查询接口 | 查询迁移到 repos |
| `deps.ConstantReader` | repos 查询接口 | 查询迁移到 repos |
| `deps.ModuleWriter` | `backend.ModuleWriter` | 仅保留写入 |
| `deps.DomainModelWriter` | `backend.DomainModelWriter` | 仅保留写入 |
| `deps.MicroflowWriter` | `backend.MicroflowWriter` | 仅保留写入 (已无 Create，只有 Delete) |
| 其他 Writer | backend *Writer | 保持不变 |

#### 删除 role.go 中的重复查询接口

当所有查询被 repos 覆盖后，删除：

- `DomainModelReader`（写入部分保留）
- `MicroflowReader`（写入部分保留）
- `WorkflowReader`（写入部分保留）
- `PageReader`（写入部分保留）
- `ModuleLister`（整体删除，repos 提供）

同时从 `HandlerDeps` 中删除对应字段：

```go
type HandlerDeps struct {
    // 删除这些：
    ModuleLister         backend.ModuleLister
    DomainModelReader    backend.DomainModelReader
    MicroflowReader      backend.MicroflowReader
    WorkflowReader       backend.WorkflowReader
    PageReader           backend.PageReader
    EnumerationReader    backend.EnumerationReader
    ConstantReader       backend.ConstantReader
    SettingsReader       backend.SettingsReader
    MappingReader        backend.MappingReader
    NavigationReader     backend.NavigationReader
    // etc.

    // repos 字段已经存在，应该直接使用它们
    DomainModels         repos.DomainModelRepository
    MicroflowRepo        repos.MicroflowRepository
    // ...
}
```

### 收益

| 指标 | 变化 |
|------|------|
| 查询接口位置 | 分散在 backend/ + repos/ → 统一在 repos/ |
| 写入接口位置 | 统一在 backend/ | 
| 接口数量 | 30+ 角色接口 → ~15 写入接口 + repos |
| handler 依赖选择 | 需要在 Reader/Writer 间选择 → 读取走 repos，写入走 backend |

---

## 5. executorCache 拆分 — 按域失效

### 问题

`executorCache`（`executor.go` 中定义）是一个单结构体，持有 18 个域的缓存字段，曝露一个全量失效方法 `invalidateAllDocumentCaches()`。任何域的写入操作都会导致全量缓存重建。

具体观察：

1. `invalidateAllDocumentCaches()` 由 ~10 个写入 handler 在变更后调用
2. 一个微流写入会同时失效页面、实体、枚举、常量等缓存的无关数据
3. 缓存字段约 200+ 行，全部位于同一结构体中

### 设计

**目标：** 将 `executorCache` 拆分为按域的独立缓存结构，失效只影响受影响的域。

#### 第 1 步：定义域缓存接口

```go
// mdl/executor/domain_cache.go
type domainCache[T any] struct {
    data     []T
    loaded   bool
    loadFn   func() ([]T, error)
}

func (c *domainCache[T]) Get() ([]T, error) {
    if c.loaded && c.data != nil {
        return c.data, nil
    }
    var err error
    c.data, err = c.loadFn()
    c.loaded = true
    return c.data, err
}

func (c *domainCache[T]) Invalidate() {
    c.loaded = false
    c.data = nil
}
```

#### 第 2 步：重构 executorCache

```go
type executorCache struct {
    microflows      *domainCache[*genMf.Microflow]
    nanoflows       *domainCache[*genMf.Nanoflow]
    pages           *domainCache[*genMf.Page]    // 或者实际类型
    snippets        *domainCache[...]
    layouts         *domainCache[...]
    workflows       *domainCache[*genWf.Workflow]
    entities        *domainCache[*genDm.Entity]
    // ...
}

func (c *executorCache) Invalidate(domains ...CacheDomain) {
    for _, d := range domains {
        switch d {
        case CacheDomainMicroflows: c.microflows.Invalidate()
        case CacheDomainPages:      c.pages.Invalidate()
        // ...
        }
    }
}
```

调用者在写入后只失效受影响的域：

```go
// 写入微流后
deps.Cache.Invalidate(CacheDomainMicroflows)

// 批量关联写入后
deps.Cache.Invalidate(CacheDomainEntities, CacheDomainAssociations)
```

#### 第 3 步：移除全量失效方法

删除 `invalidateAllDocumentCaches()`。替换为 `Invalidate()` 调用。

### 收益

| 指标 | 变化 |
|------|------|
| 写入后无效数据量 | 全量 → 仅受影响域 |
| 缓存未命中率 | 高（全量失效后所有域重新加载）→ 低（仅失效域重新加载） |
| 接口复杂度 | 单一全量方法 → 显式域选择 |

---

## 6. 附录：Cobra 命令层规范化

### 问题

审计发现 `cmd/mxcli/` 中有三个不一致的模式：

1. **错误处理**：混合 `Run` + `os.Exit(1)` 与 `RunE` + `return error`
2. **注册方式**：集中式（`main.go init()`) 与分散式（`cmd_*.go init()`) 共存
3. **包级别全局变量**：`globalJSONFlag`、`globalVerboseLevel`、`version` 等

### 设计

#### 第 1 步：统一错误处理

所有命令迁移到 `RunE`：

```go
// Before:
var checkCmd = &cobra.Command{
    Run: func(cmd *cobra.Command, args []string) {
        if err := doCheck(); err != nil {
            fmt.Fprintln(os.Stderr, err)
            os.Exit(1)
        }
    },
}

// After:
var checkCmd = &cobra.Command{
    RunE: func(cmd *cobra.Command, args []string) error {
        return doCheck()
    },
}
```

#### 第 2 步：统一注册到集中式 init()

```go
// main.go init()
func init() {
    rootCmd.AddCommand(execCmd)
    rootCmd.AddCommand(showCmd)
    rootCmd.AddCommand(describeCmd)
    rootCmd.AddCommand(checkCmd)
    rootCmd.AddCommand(lintCmd)
    rootCmd.AddCommand(reportCmd)
    rootCmd.AddCommand(diffCmd)
    rootCmd.AddCommand(sqlCmd)
    rootCmd.AddCommand(explainCmd)
    rootCmd.AddCommand(evalCmd)
    // ... 所有命令都在这里
}
```

从各 `cmd_*.go` 的 `init()` 中删除 `AddCommand`。这样新贡献者只需看 `main.go` 就能知道所有命令的注册情况。

---

## 迁移总路线

```
        2026-07                    2026-08                    2026-09
        │                          │                          │
先决条件 ├── 1. FullBackend 解体 ──┤                          │
(无依赖) │   (2-3 天)              │                          │
        │                          │                          │
基础架构 ├── 1 完成后               ├── 2. 双容器合并 ────────┤
        │                          │   (1 周, 逐个 handler)  │
        │                          │                          │
域重构  └──────────────────────────┼── 3. MprBackend 分解 ──┤
                                   │   (1 周, 逐域提取)       │
                                   ├── 4. repos 激活 ────────┤
                                   │   (3-5 天, 逐 handler)  │
                                   ├── 5. executorCache 拆分 │
                                   │   (2 天)                │
                                   └── 6. Cobra 规范化 ─────┤
                                       (1 天)                │
```

每个阶段可独立实施和提交。推荐顺序：
1. 先做 FullBackend 解体（最少变更，最大收益）
2. 再做 repos 激活（决策查询/写入分离，为后续做准备）
3. 接着 MprBackend 分解（各域后端独立后，自然推动双容器合并）
4. 最后双容器合并和缓存拆分（基于前面完成的域边界）

---

*文档版本：2026-07-20*
*对应审计报告：2026-07-20 架构审计*
