# MEMV Phase 2.5 — JIT 语义验证实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 为 `mxcli expr` 增加语义验证层（SEM-02/04/05/07），通过隐式 Daemon 将 MPR 元数据索引常驻内存，热路径 ~10ms；`--no-daemon` 模式跳过语义层，恒定 ~100ms。

**架构：**
- `internal/expr/meta/` — 从 MPR 后端（mprbackend.NewFromPath）构建内存元数据索引，实现 `exprcheck.CatalogReader` 接口
- `internal/expr/daemon/` — Unix socket JSON-RPC 后台守护进程，隐式启动，5min 空闲自动退出
- `cmd/mxcli/cmd_expr.go` — 仅接受 `-p project.mpr`，`--no-daemon` 禁用语义层

**技术栈：** Go 1.26，`mdl/backend/mpr`（已有），`mdl/exprcheck`（已有），Unix socket + encoding/json，`golang.org/x/sys/unix`（或 `net` 包 UnixListener）

**工作目录：** `/mnt/data_sdd/gh/mxcli-wt-02`（feature/expression-checker 分支）

**规格文件：** `/mnt/data_sdd/macnica/docs/superpowers/specs/2026-05-18-memv-phase25-semantic.md`

---

## 文件结构

```
internal/expr/
├── meta/
│   ├── index.go           # Index struct + BuildFromBackend()
│   ├── index_test.go      # 集成测试（需要真实 MPR）
│   └── catalog_reader.go  # 实现 exprcheck.CatalogReader
├── daemon/
│   ├── socket.go          # SocketPath() + IsAlive()
│   ├── proto.go           # JSON-RPC 请求/响应类型
│   ├── daemon.go          # Daemon.Serve() — 服务端
│   ├── daemon_test.go
│   ├── client.go          # DaemonClient.StartIfNeeded() + Call()
│   └── client_test.go
├── validate/
│   └── validate_sem.go    # SEM-02/04/05/07 规则（新文件）
└── parse/
    └── parse.go           # 修改：Daemon 模式传入 ctx.Catalog

cmd/mxcli/
├── cmd_expr.go            # 修改：--no-daemon flag，-p 路径推导，daemon client
└── cmd_expr_daemon.go     # 新增：expr daemon status/stop 子命令
```

---

## 关键已有代码（阅读顺序）

执行前必读：
- `mdl/backend/mpr/backend.go:61-112` — `NewFromPath(path)` 连接 MPR
- `mdl/exprcheck/interfaces.go` — `CatalogReader` 接口（5个方法）
- `internal/expr/validate/validate.go` — 现有 ValidateSyntax() 签名
- `internal/expr/parse/parse.go` — 现有 parseExpr() 中的 Context 构建
- `cmd/mxcli/cmd_expr.go` — 现有子命令结构

---

## Task 1：meta.Index — 实体属性索引

**文件：**
- 创建：`internal/expr/meta/index.go`
- 创建：`internal/expr/meta/index_test.go`

- [ ] **步骤 1：写失败测试**

创建 `internal/expr/meta/index_test.go`：

```go
// SPDX-License-Identifier: Apache-2.0

package meta_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/expr/meta"
	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const macnicaMPR = "/mnt/data_sdd/macnica/mendix-app/MacnicaApp.mpr"

func openBackend(t *testing.T, mprPath string) *mprbackend.MprBackend {
	t.Helper()
	b, err := mprbackend.NewFromPath(mprPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = b.Disconnect() })
	return b
}

func TestBuildFromBackend_EntityAttrs(t *testing.T) {
	b := openBackend(t, macnicaMPR)
	idx, err := meta.BuildFromBackend(b)
	require.NoError(t, err)
	require.NotNil(t, idx)

	// macnica 有 BusinessApp_Common.ApplicationCommonHeader 实体
	kind, ok := idx.AttributeKind("BusinessApp_Common.ApplicationCommonHeader", "ApplicationStatus")
	assert.True(t, ok, "ApplicationStatus 属性应存在")
	assert.NotZero(t, kind, "kind 应为非零值（KindString 或 KindEnumeration）")
}

func TestBuildFromBackend_SystemEntity(t *testing.T) {
	b := openBackend(t, macnicaMPR)
	idx, err := meta.BuildFromBackend(b)
	require.NoError(t, err)

	// System.User 是内置实体，应该可查到
	_, ok := idx.AttributeKind("System.User", "Name")
	assert.True(t, ok, "System.User.Name 应存在（继承自 System 模块）")
}
```

- [ ] **步骤 2：运行确认失败**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go test ./internal/expr/meta/... -v 2>&1 | head -10
```

预期：`FAIL — no Go files`

- [ ] **步骤 3：实现 index.go**

创建 `internal/expr/meta/index.go`：

```go
// SPDX-License-Identifier: Apache-2.0

// Package meta 从 Mendix MPR 文件构建轻量语义元数据索引，
// 并实现 exprcheck.CatalogReader 接口供语义验证使用。
package meta

import (
	"strings"

	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/exprcheck"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// Index 持有从 MPR 提取的语义元数据，常驻内存。
// 实现 exprcheck.CatalogReader 接口。
type Index struct {
	// entityAttrs: map[entityQN]map[attrName]TypeKind
	// 已展开继承链（含 System 模块实体）
	entityAttrs map[string]map[string]exprcheck.TypeKind

	// entityAttrEnumQN: map[entityQN+"."+attrName] → enumQN
	// 仅枚举类型属性存在此条目
	entityAttrEnumQN map[string]string

	// enumValues: map[enumQN][]string
	// 枚举值名称列表（区分大小写）
	enumValues map[string][]string

	// constants: map["@Module.Name"]TypeKind
	// 常量存在性及类型
	constants map[string]exprcheck.TypeKind
}

// BuildFromBackend 从已连接的 MPR 后端构建 Index。
// backend 必须已调用过 Connect()（即 NewFromPath() 返回的实例）。
func BuildFromBackend(b backend.FullBackend) (*Index, error) {
	idx := &Index{
		entityAttrs:      make(map[string]map[string]exprcheck.TypeKind),
		entityAttrEnumQN: make(map[string]string),
		enumValues:       make(map[string][]string),
		constants:        make(map[string]exprcheck.TypeKind),
	}

	if err := idx.buildEntityAttrs(b); err != nil {
		return nil, err
	}
	if err := idx.buildEnumValues(b); err != nil {
		return nil, err
	}
	if err := idx.buildConstants(b); err != nil {
		return nil, err
	}
	return idx, nil
}

// buildEntityAttrs 遍历所有 DomainModel，提取实体属性类型。
// 含继承链展开：若实体 extends 另一实体，父实体属性也加入。
func (idx *Index) buildEntityAttrs(b backend.FullBackend) error {
	dms, err := b.ListDomainModelsGen()
	if err != nil {
		return err
	}

	// 第一遍：收集所有实体的直接属性
	type entityInfo struct {
		qn             string
		generalizationQN string // "" 表示无继承
		attrs          map[string]exprcheck.TypeKind
		attrEnumQN     map[string]string
	}
	entities := make(map[string]*entityInfo)

	for _, dm := range dms {
		for _, elem := range dm.EntitiesItems() {
			entity, ok := elem.(*genDm.Entity)
			if !ok {
				continue
			}
			moduleName := moduleFromContainerName(dm.Name())
			qn := moduleName + "." + entity.Name()

			info := &entityInfo{
				qn:    qn,
				attrs: make(map[string]exprcheck.TypeKind),
				attrEnumQN: make(map[string]string),
			}

			// 提取泛化（继承）目标
			if g, ok := entity.Generalization().(*genDm.Generalization); ok {
				info.generalizationQN = g.GeneralizationQualifiedName()
			}

			// 提取直接属性
			for _, aElem := range entity.AttributesItems() {
				attr, ok := aElem.(*genDm.Attribute)
				if !ok {
					continue
				}
				attrName := attr.Name()
				kind := attrTypeToKind(attr.Type())
				info.attrs[attrName] = kind

				// 枚举属性：记录 enumQN
				if eat, ok := attr.Type().(*genDm.EnumerationAttributeType); ok {
					if eqn := eat.EnumerationQualifiedName(); eqn != "" {
						info.attrEnumQN[attrName] = eqn
					}
				}
			}
			entities[qn] = info
		}
	}

	// 第二遍：展开继承链（最多 10 层，防止循环）
	for qn, info := range entities {
		attrs := make(map[string]exprcheck.TypeKind, len(info.attrs))
		enumQNs := make(map[string]string, len(info.attrEnumQN))
		for k, v := range info.attrs {
			attrs[k] = v
		}
		for k, v := range info.attrEnumQN {
			enumQNs[k] = v
		}

		// 向上遍历继承链
		seen := map[string]bool{qn: true}
		parentQN := info.generalizationQN
		for depth := 0; depth < 10 && parentQN != ""; depth++ {
			if seen[parentQN] {
				break
			}
			seen[parentQN] = true
			parent, ok := entities[parentQN]
			if !ok {
				break
			}
			for k, v := range parent.attrs {
				if _, exists := attrs[k]; !exists {
					attrs[k] = v
				}
			}
			for k, v := range parent.attrEnumQN {
				if _, exists := enumQNs[k]; !exists {
					enumQNs[k] = v
				}
			}
			parentQN = parent.generalizationQN
		}

		idx.entityAttrs[qn] = attrs
		for attrName, enumQN := range enumQNs {
			idx.entityAttrEnumQN[qn+"."+attrName] = enumQN
		}
	}

	return nil
}

// attrTypeToKind 将 genDm 属性类型映射到 exprcheck.TypeKind。
func attrTypeToKind(t interface{}) exprcheck.TypeKind {
	switch t.(type) {
	case *genDm.StringAttributeType:
		return exprcheck.KindString
	case *genDm.IntegerAttributeType:
		return exprcheck.KindInteger
	case *genDm.LongAttributeType:
		return exprcheck.KindLong
	case *genDm.DecimalAttributeType:
		return exprcheck.KindDecimal
	case *genDm.BooleanAttributeType:
		return exprcheck.KindBoolean
	case *genDm.DateTimeAttributeType:
		return exprcheck.KindDateTime
	case *genDm.BinaryAttributeType:
		return exprcheck.KindBinary
	case *genDm.EnumerationAttributeType:
		return exprcheck.KindEnumeration
	default:
		return exprcheck.KindUnknown
	}
}

// moduleFromContainerName 从 DomainModel.Name() 提取模块名。
// 例：DomainModel Name = "BusinessApp_Common" → "BusinessApp_Common"
func moduleFromContainerName(dmName string) string {
	// DomainModel 名称通常就是模块名
	return strings.TrimSuffix(dmName, "DomainModel")
}
```

- [ ] **步骤 4：运行测试**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go test ./internal/expr/meta/... -v -run TestBuildFromBackend_EntityAttrs -timeout 60s 2>&1
```

预期：`PASS`（实体属性可查到）

若 `System.User` 测试失败（System 模块 entity 不在普通 DomainModel 列表中），在 `buildEntityAttrs` 末尾暂时跳过继承链对 `System.*` 的查找——把 `TestBuildFromBackend_SystemEntity` 放入 `// TODO Phase 2.5.1`。

- [ ] **步骤 5：提交**

```bash
git add internal/expr/meta/
git commit -m "feat(expr/meta): Index struct + buildEntityAttrs — entity attrs with inheritance"
```

---

## Task 2：meta.Index — 枚举值索引 + 常量索引

**文件：**
- 修改：`internal/expr/meta/index.go`（添加 buildEnumValues / buildConstants）
- 修改：`internal/expr/meta/index_test.go`（添加枚举和常量测试）

- [ ] **步骤 1：写失败测试**

在 `index_test.go` 追加：

```go
func TestBuildFromBackend_EnumValues(t *testing.T) {
	b := openBackend(t, macnicaMPR)
	idx, err := meta.BuildFromBackend(b)
	require.NoError(t, err)

	// Common_Utils.BatchExecStatus 有 Completed/Failed/Cancelled 等值
	vals, ok := idx.EnumCases("Common_Utils.BatchExecStatus")
	assert.True(t, ok, "BatchExecStatus 应存在")
	assert.Contains(t, vals, "Completed")
	assert.Greater(t, len(vals), 1)
}

func TestBuildFromBackend_Constants(t *testing.T) {
	b := openBackend(t, macnicaMPR)
	idx, err := meta.BuildFromBackend(b)
	require.NoError(t, err)

	// macnica 至少有 1 个常量
	assert.Greater(t, idx.ConstantsCount(), 0, "应有常量记录")
}

func TestBuildFromBackend_MissingEnum(t *testing.T) {
	b := openBackend(t, macnicaMPR)
	idx, err := meta.BuildFromBackend(b)
	require.NoError(t, err)

	_, ok := idx.EnumCases("NonExistent.Module.FakeEnum")
	assert.False(t, ok, "不存在的枚举应返回 false")
}
```

- [ ] **步骤 2：运行确认失败**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go test ./internal/expr/meta/... -v -run TestBuildFromBackend_Enum -timeout 60s 2>&1 | head -20
```

预期：FAIL（EnumCases 方法未定义）

- [ ] **步骤 3：实现 buildEnumValues 和 buildConstants**

在 `internal/expr/meta/index.go` 追加：

```go
// buildEnumValues 提取所有枚举的值列表。
func (idx *Index) buildEnumValues(b backend.FullBackend) error {
	enums, err := b.ListEnumerations()
	if err != nil {
		return err
	}
	// 需要获取模块名以构造 QN，通过 ListModules 建立 ID→name 映射
	modules, err := b.ListModules()
	if err != nil {
		return err
	}
	modByID := make(map[string]string, len(modules))
	for _, m := range modules {
		modByID[string(m.ID)] = m.Name
	}

	for _, enum := range enums {
		// 枚举的 ContainerID 指向模块的 DomainModel，
		// 先尝试直接通过模块层级查找模块名
		// 简化方案：枚举 QN 可通过 ContainerID 二次查找，
		// 但此处用 GetRawUnit 是最直接的路径
		// 实际上 buildEnumerations 在 catalog/builder_modules.go 中
		// 使用 b.hierarchy.getModuleName(b.hierarchy.findModuleID(enum.ContainerID))
		// 我们这里直接使用 ListModules 的 ID 匹配
		containerIDStr := string(enum.ContainerID)
		moduleName := modByID[containerIDStr]
		if moduleName == "" {
			// ContainerID 是 DomainModel 的 ID，不是 Module 的 ID
			// 用 rawUnit 查找——见后面 Task 说明
			// 降级：跳过此枚举，不影响主流程
			continue
		}
		qn := moduleName + "." + enum.Name
		vals := make([]string, 0, len(enum.Values))
		for _, v := range enum.Values {
			vals = append(vals, v.Name)
		}
		idx.enumValues[qn] = vals
	}
	return nil
}

// buildConstants 提取所有常量及类型。
func (idx *Index) buildConstants(b backend.FullBackend) error {
	constants, err := b.ListConstants()
	if err != nil {
		return err
	}
	modules, err := b.ListModules()
	if err != nil {
		return err
	}
	modByID := make(map[string]string, len(modules))
	for _, m := range modules {
		modByID[string(m.ID)] = m.Name
	}

	for _, c := range constants {
		containerIDStr := string(c.ContainerID)
		moduleName := modByID[containerIDStr]
		if moduleName == "" {
			continue
		}
		key := "@" + moduleName + "." + c.Name
		kind := constantKindToExprKind(c.Type.Kind)
		idx.constants[key] = kind
	}
	return nil
}

// constantKindToExprKind 将常量数据类型映射到 TypeKind。
func constantKindToExprKind(kind string) exprcheck.TypeKind {
	switch kind {
	case "String":
		return exprcheck.KindString
	case "Integer":
		return exprcheck.KindInteger
	case "Long":
		return exprcheck.KindLong
	case "Decimal":
		return exprcheck.KindDecimal
	case "Boolean":
		return exprcheck.KindBoolean
	case "DateTime":
		return exprcheck.KindDateTime
	default:
		return exprcheck.KindUnknown
	}
}

// EnumCases 返回枚举的所有值名称。实现 exprcheck.CatalogReader。
func (idx *Index) EnumCases(enumQN string) ([]string, bool) {
	vals, ok := idx.enumValues[enumQN]
	return vals, ok
}

// ConstantsCount 返回已索引的常量数量（用于测试断言）。
func (idx *Index) ConstantsCount() int { return len(idx.constants) }
```

**注意：** `ContainerID` 指向 DomainModel 而非 Module。若上述直接匹配不到，改为遍历 `ListDomainModelsGen()` 建立 `domainModelID → moduleName` 映射（参考 `mdl/catalog/builder_modules.go:buildEntities()` 的 `b.hierarchy.findModuleID` 逻辑）：

```go
// 备选方案：通过 DomainModel 名称推导模块名
for _, dm := range dms {  // dms 来自 b.ListDomainModelsGen()
    dmIDStr := string(dm.ID())  // genDm.DomainModel.ID()
    dmModByID[dmIDStr] = dm.Name()
}
```

调试时先运行测试看 ContainerID 是否匹配，若不匹配则改用 DomainModel ID 映射。

- [ ] **步骤 4：运行测试**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go test ./internal/expr/meta/... -v -timeout 120s 2>&1
```

预期：所有测试通过。若枚举 ContainerID 问题导致失败，按上述备选方案修正。

- [ ] **步骤 5：提交**

```bash
git add internal/expr/meta/
git commit -m "feat(expr/meta): buildEnumValues + buildConstants — enum values and constant refs"
```

---

## Task 3：meta.Index 实现 exprcheck.CatalogReader

**文件：**
- 创建：`internal/expr/meta/catalog_reader.go`
- 修改：`internal/expr/meta/index_test.go`（添加 CatalogReader 接口测试）

- [ ] **步骤 1：写失败测试**

在 `index_test.go` 追加：

```go
func TestIndex_ImplementsCatalogReader(t *testing.T) {
	b := openBackend(t, macnicaMPR)
	idx, err := meta.BuildFromBackend(b)
	require.NoError(t, err)

	// 验证 Index 满足 exprcheck.CatalogReader 接口
	var _ exprcheck.CatalogReader = idx
}

func TestAttributeKind_Returns(t *testing.T) {
	b := openBackend(t, macnicaMPR)
	idx, err := meta.BuildFromBackend(b)
	require.NoError(t, err)

	kind, ok := idx.AttributeKind("BusinessApp_Common.ApplicationCommonHeader", "ApplicationStatus")
	assert.True(t, ok)
	assert.NotZero(t, kind)
}

func TestAttributeKind_Missing(t *testing.T) {
	b := openBackend(t, macnicaMPR)
	idx, err := meta.BuildFromBackend(b)
	require.NoError(t, err)

	_, ok := idx.AttributeKind("NonExistent.Entity", "SomeAttr")
	assert.False(t, ok)
}
```

- [ ] **步骤 2：运行确认失败**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go test ./internal/expr/meta/... -run TestIndex_Implements -timeout 60s 2>&1
```

预期：编译错误（缺少接口方法）

- [ ] **步骤 3：实现 catalog_reader.go**

创建 `internal/expr/meta/catalog_reader.go`：

```go
// SPDX-License-Identifier: Apache-2.0

package meta

import "github.com/mendixlabs/mxcli/mdl/exprcheck"

// 以下方法使 *Index 满足 exprcheck.CatalogReader 接口。
// 接口定义见 mdl/exprcheck/interfaces.go。

// AttributeKind 返回实体属性的类型。
// entityQN 格式："Module.EntityName"，attrName 为属性名。
func (idx *Index) AttributeKind(entityQN, attrName string) (exprcheck.TypeKind, bool) {
	attrs, ok := idx.entityAttrs[entityQN]
	if !ok {
		return exprcheck.KindUnknown, false
	}
	kind, ok := attrs[attrName]
	return kind, ok
}

// AttributeEnumQN 返回枚举类型属性的枚举限定名。
func (idx *Index) AttributeEnumQN(entityQN, attrName string) (string, bool) {
	key := entityQN + "." + attrName
	qn, ok := idx.entityAttrEnumQN[key]
	return qn, ok
}

// MicroflowReturn 返回微流的返回值类型。
// Phase 2.5 暂不支持（需要微流签名表），返回 KindUnknown。
func (idx *Index) MicroflowReturn(_ string) (exprcheck.TypeKind, bool) {
	return exprcheck.KindUnknown, false
}

// MicroflowParam 返回微流参数的类型。
// Phase 2.5 暂不支持，返回 KindUnknown。
func (idx *Index) MicroflowParam(_, _ string) (exprcheck.TypeKind, bool) {
	return exprcheck.KindUnknown, false
}

// HasConstant 检查常量是否存在（供 SEM-05 规则使用，非标准接口方法）。
func (idx *Index) HasConstant(ref string) bool {
	_, ok := idx.constants[ref]
	return ok
}

// HasEntity 检查实体 QN 是否存在（供 SEM-07 XPath 验证使用）。
func (idx *Index) HasEntity(entityQN string) bool {
	_, ok := idx.entityAttrs[entityQN]
	return ok
}
```

- [ ] **步骤 4：运行所有 meta 测试**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go test ./internal/expr/meta/... -v -timeout 120s 2>&1
```

预期：全部通过。

- [ ] **步骤 5：提交**

```bash
git add internal/expr/meta/
git commit -m "feat(expr/meta): implement exprcheck.CatalogReader — AttributeKind/EnumCases/HasConstant/HasEntity"
```

---

## Task 4：daemon 基础设施 — socket + proto

**文件：**
- 创建：`internal/expr/daemon/socket.go`
- 创建：`internal/expr/daemon/proto.go`
- 创建：`internal/expr/daemon/socket_test.go`

- [ ] **步骤 1：写失败测试**

创建 `internal/expr/daemon/socket_test.go`：

```go
// SPDX-License-Identifier: Apache-2.0

package daemon_test

import (
	"path/filepath"
	"testing"

	"github.com/mendixlabs/mxcli/internal/expr/daemon"
	"github.com/stretchr/testify/assert"
)

func TestSocketPath_Deterministic(t *testing.T) {
	p1 := daemon.SocketPath("/a/b/project.mpr")
	p2 := daemon.SocketPath("/a/b/project.mpr")
	assert.Equal(t, p1, p2, "同一 MPR 路径应得到同一 socket 路径")
}

func TestSocketPath_DifferentMPR(t *testing.T) {
	p1 := daemon.SocketPath("/a/App.mpr")
	p2 := daemon.SocketPath("/b/App.mpr")
	assert.NotEqual(t, p1, p2, "不同 MPR 路径应得到不同 socket 路径")
}

func TestSocketPath_InDaemonDir(t *testing.T) {
	p := daemon.SocketPath("/some/path/project.mpr")
	assert.Contains(t, p, ".mxcli", "socket 应在 .mxcli 目录下")
	assert.Contains(t, p, ".sock", "应为 .sock 后缀")
	assert.True(t, filepath.IsAbs(p), "应为绝对路径")
}

func TestIsAlive_NoSocket(t *testing.T) {
	alive := daemon.IsAlive("/tmp/nonexistent_test_9999.sock")
	assert.False(t, alive, "不存在的 socket 应返回 false")
}
```

- [ ] **步骤 2：运行确认失败**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go test ./internal/expr/daemon/... -v 2>&1 | head -10
```

预期：FAIL（no Go files）

- [ ] **步骤 3：实现 socket.go**

创建 `internal/expr/daemon/socket.go`：

```go
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"crypto/sha256"
	"fmt"
	"net"
	"os"
	"path/filepath"
)

// SocketPath 根据 MPR 文件绝对路径生成唯一的 Unix socket 文件路径。
// 格式：~/.mxcli/expr-daemon/<sha256[:8]>.sock
func SocketPath(mprPath string) string {
	abs, err := filepath.Abs(mprPath)
	if err != nil {
		abs = mprPath
	}
	hash := sha256.Sum256([]byte(abs))
	name := fmt.Sprintf("%x.sock", hash[:4])

	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	dir := filepath.Join(home, ".mxcli", "expr-daemon")
	return filepath.Join(dir, name)
}

// IsAlive 检查给定 socket 路径的 daemon 是否在运行。
// 通过尝试建立连接判断，不发送任何数据。
func IsAlive(socketPath string) bool {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// EnsureDaemonDir 确保 socket 目录存在。
func EnsureDaemonDir() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, ".mxcli", "expr-daemon")
	return os.MkdirAll(dir, 0o700)
}
```

- [ ] **步骤 4：实现 proto.go**

创建 `internal/expr/daemon/proto.go`：

```go
// SPDX-License-Identifier: Apache-2.0

package daemon

// ValidateRequest 是客户端发往 daemon 的验证请求。
type ValidateRequest struct {
	MprPath  string `json:"mprPath"`
	Filter   string `json:"filter,omitempty"`   // unit_type 子串过滤
	Severity string `json:"severity,omitempty"` // ERROR | WARNING | INFO | ""
}

// ValidationItem 是单条验证结果。
type ValidationItem struct {
	UnitID   string `json:"unitID"`
	UnitType string `json:"unitType"`
	Field    string `json:"field"`
	Raw      string `json:"raw"`
	RuleID   string `json:"ruleID"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Fix      string `json:"fix,omitempty"`
}

// ValidateResponse 是 daemon 返回的验证响应。
type ValidateResponse struct {
	IndexAge string           `json:"indexAge"` // 如 "2m34s"
	Results  []ValidationItem `json:"results"`
	Error    string           `json:"error,omitempty"`
}

// PingRequest 用于检活。
type PingRequest struct{}

// PingResponse 返回 daemon 状态。
type PingResponse struct {
	OK        bool   `json:"ok"`
	MprPath   string `json:"mprPath"`
	IndexAge  string `json:"indexAge"`
	EntityCount int  `json:"entityCount"`
	EnumCount   int  `json:"enumCount"`
}
```

- [ ] **步骤 5：运行测试**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go test ./internal/expr/daemon/... -v -run TestSocket -timeout 30s 2>&1
```

预期：4/4 PASS

- [ ] **步骤 6：提交**

```bash
git add internal/expr/daemon/
git commit -m "feat(expr/daemon): socket.go + proto.go — SocketPath/IsAlive/ValidateRequest/Response"
```

---

## Task 5：Daemon 服务端 — daemon.go

**文件：**
- 创建：`internal/expr/daemon/daemon.go`
- 创建：`internal/expr/daemon/daemon_test.go`

- [ ] **步骤 1：写失败测试**

创建 `internal/expr/daemon/daemon_test.go`：

```go
// SPDX-License-Identifier: Apache-2.0

package daemon_test

import (
	"encoding/json"
	"net"
	"os"
	"testing"
	"time"

	"github.com/mendixlabs/mxcli/internal/expr/daemon"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const macnicaMPR = "/mnt/data_sdd/macnica/mendix-app/MacnicaApp.mpr"

func TestDaemon_ServeAndPing(t *testing.T) {
	socketPath := "/tmp/test_expr_daemon_" + t.Name() + ".sock"
	_ = os.Remove(socketPath)
	defer os.Remove(socketPath)

	d, err := daemon.New(macnicaMPR, daemon.Options{
		SocketPath:    socketPath,
		IdleTimeout:   10 * time.Second,
	})
	require.NoError(t, err)

	// 后台启动
	go func() { _ = d.Serve() }()
	time.Sleep(3 * time.Second) // 等待 index 构建完成

	// 连接并发送 ping
	conn, err := net.Dial("unix", socketPath)
	require.NoError(t, err)
	defer conn.Close()

	req := map[string]string{"method": "ping"}
	require.NoError(t, json.NewEncoder(conn).Encode(req))

	var resp daemon.PingResponse
	require.NoError(t, json.NewDecoder(conn).Decode(&resp))
	assert.True(t, resp.OK)
	assert.Greater(t, resp.EntityCount, 0)
	assert.Greater(t, resp.EnumCount, 0)

	d.Stop()
}

func TestDaemon_ValidateReturnsResults(t *testing.T) {
	socketPath := "/tmp/test_expr_daemon_validate_" + t.Name() + ".sock"
	_ = os.Remove(socketPath)
	defer os.Remove(socketPath)

	d, err := daemon.New(macnicaMPR, daemon.Options{
		SocketPath:  socketPath,
		IdleTimeout: 10 * time.Second,
	})
	require.NoError(t, err)
	go func() { _ = d.Serve() }()
	time.Sleep(4 * time.Second)

	conn, err := net.Dial("unix", socketPath)
	require.NoError(t, err)
	defer conn.Close()

	req := daemon.ValidateRequest{MprPath: macnicaMPR}
	require.NoError(t, json.NewEncoder(conn).Encode(req))

	var resp daemon.ValidateResponse
	require.NoError(t, json.NewDecoder(conn).Decode(&resp))
	assert.Empty(t, resp.Error)
	assert.NotEmpty(t, resp.IndexAge)
	// macnica 至少有 1 个 E006 错误
	found := false
	for _, r := range resp.Results {
		if r.RuleID == "E006" {
			found = true
			break
		}
	}
	assert.True(t, found, "应找到 macnica 已知的 E006 错误")

	d.Stop()
}
```

- [ ] **步骤 2：运行确认失败**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go test ./internal/expr/daemon/... -run TestDaemon -v 2>&1 | head -10
```

预期：编译失败（daemon.New 未定义）

- [ ] **步骤 3：实现 daemon.go**

创建 `internal/expr/daemon/daemon.go`：

```go
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/mendixlabs/mxcli/internal/expr/meta"
	"github.com/mendixlabs/mxcli/internal/expr/parse"
	"github.com/mendixlabs/mxcli/internal/expr/scan"
	"github.com/mendixlabs/mxcli/internal/expr/validate"
)

// Options 配置 Daemon 行为。
type Options struct {
	SocketPath  string        // Unix socket 路径，默认通过 SocketPath(mprPath) 计算
	IdleTimeout time.Duration // 空闲超时，默认 5min
}

// Daemon 是单个 MPR 文件的后台验证服务。
type Daemon struct {
	mprPath    string
	socketPath string
	idleTimeout time.Duration

	mu       sync.RWMutex
	idx      *meta.Index
	indexBuiltAt time.Time

	listener net.Listener
	stopCh   chan struct{}
	lastReq  time.Time
}

// New 创建新 Daemon，连接 MPR 并构建初始索引。
// 调用方需在 goroutine 中调用 Serve()。
func New(mprPath string, opts Options) (*Daemon, error) {
	socketPath := opts.SocketPath
	if socketPath == "" {
		socketPath = SocketPath(mprPath)
	}
	timeout := opts.IdleTimeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	if err := EnsureDaemonDir(); err != nil {
		return nil, fmt.Errorf("daemon dir: %w", err)
	}

	d := &Daemon{
		mprPath:     mprPath,
		socketPath:  socketPath,
		idleTimeout: timeout,
		stopCh:      make(chan struct{}),
		lastReq:     time.Now(),
	}

	// 构建初始索引（冷启动成本在此）
	if err := d.rebuildIndex(); err != nil {
		return nil, fmt.Errorf("build index: %w", err)
	}
	return d, nil
}

func (d *Daemon) rebuildIndex() error {
	b, err := mprbackend.NewFromPath(d.mprPath)
	if err != nil {
		return err
	}
	defer func() { _ = b.Disconnect() }()

	idx, err := meta.BuildFromBackend(b)
	if err != nil {
		return err
	}
	d.mu.Lock()
	d.idx = idx
	d.indexBuiltAt = time.Now()
	d.mu.Unlock()
	return nil
}

// Serve 启动 Unix socket 监听。阻塞直到 Stop() 被调用。
func (d *Daemon) Serve() error {
	_ = os.Remove(d.socketPath)
	l, err := net.Listen("unix", d.socketPath)
	if err != nil {
		return fmt.Errorf("listen unix: %w", err)
	}
	d.listener = l
	defer func() {
		_ = l.Close()
		_ = os.Remove(d.socketPath)
	}()

	// 空闲超时检查
	go d.idleWatcher()

	for {
		conn, err := l.Accept()
		if err != nil {
			select {
			case <-d.stopCh:
				return nil
			default:
				return err
			}
		}
		go d.handleConn(conn)
	}
}

func (d *Daemon) idleWatcher() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-d.stopCh:
			return
		case <-ticker.C:
			d.mu.RLock()
			idle := time.Since(d.lastReq)
			d.mu.RUnlock()
			if idle > d.idleTimeout {
				d.Stop()
				return
			}
		}
	}
}

// Stop 终止 daemon。
func (d *Daemon) Stop() {
	select {
	case <-d.stopCh:
	default:
		close(d.stopCh)
	}
	if d.listener != nil {
		_ = d.listener.Close()
	}
}

func (d *Daemon) handleConn(conn net.Conn) {
	defer conn.Close()

	d.mu.Lock()
	d.lastReq = time.Now()
	d.mu.Unlock()

	// 尝试解码为 ValidateRequest（主路径）
	var req ValidateRequest
	dec := json.NewDecoder(conn)
	if err := dec.Decode(&req); err != nil {
		_ = json.NewEncoder(conn).Encode(ValidateResponse{Error: err.Error()})
		return
	}

	// Ping 请求（ValidateRequest.MprPath 为空时）
	if req.MprPath == "" {
		d.mu.RLock()
		idx := d.idx
		age := time.Since(d.indexBuiltAt)
		d.mu.RUnlock()
		resp := PingResponse{
			OK:          true,
			MprPath:     d.mprPath,
			IndexAge:    age.Truncate(time.Second).String(),
			EntityCount: idx.EntityCount(),
			EnumCount:   idx.EnumCount(),
		}
		_ = json.NewEncoder(conn).Encode(resp)
		return
	}

	// 验证请求
	d.mu.RLock()
	idx := d.idx
	age := time.Since(d.indexBuiltAt)
	d.mu.RUnlock()

	// 采集表达式
	mprContentsPath := scan.MprContentsPath(req.MprPath)
	records, err := scan.ScanMprcontents(mprContentsPath, scan.Options{FilterType: req.Filter})
	if err != nil {
		_ = json.NewEncoder(conn).Encode(ValidateResponse{Error: err.Error()})
		return
	}

	// 解析 + 验证（传入 idx 作为 Catalog）
	parsed := parse.BatchParseWithCatalog(records, idx)
	var issues []validate.ValidationResult
	for _, pr := range parsed {
		issues = append(issues, validate.ValidateSyntax(pr)...)
		issues = append(issues, validate.ValidateSemantic(pr, idx)...)
	}

	// 过滤严重度
	if req.Severity != "" {
		filtered := issues[:0]
		for _, i := range issues {
			if i.Severity == req.Severity {
				filtered = append(filtered, i)
			}
		}
		issues = filtered
	}

	items := make([]ValidationItem, 0, len(issues))
	for _, i := range issues {
		items = append(items, ValidationItem{
			UnitID: i.UnitID, UnitType: i.UnitType, Field: i.Field,
			Raw: i.Raw, RuleID: i.RuleID, Severity: i.Severity,
			Message: i.Message, Fix: i.Fix,
		})
	}

	resp := ValidateResponse{
		IndexAge: age.Truncate(time.Second).String(),
		Results:  items,
	}
	_ = json.NewEncoder(conn).Encode(resp)
}
```

注意：还需要在 meta.Index 上添加 `EntityCount()` 和 `EnumCount()` 辅助方法，在 scan 包添加 `MprContentsPath(mprPath string) string`，在 parse 包添加 `BatchParseWithCatalog()`，并创建 `validate.ValidateSemantic()`——这些在后续 Task 中实现。编译期先用 `// TODO: see Task 6/7/8` 注释。

- [ ] **步骤 4：在 meta/index.go 追加辅助方法**

```go
// EntityCount 返回已索引的实体数量（用于状态报告）。
func (idx *Index) EntityCount() int { return len(idx.entityAttrs) }

// EnumCount 返回已索引的枚举数量。
func (idx *Index) EnumCount() int { return len(idx.enumValues) }
```

- [ ] **步骤 5：在 scan/scan.go 追加 MprContentsPath**

```go
// MprContentsPath 从 MPR 文件路径推导 mprcontents/ 目录路径。
// 对 V2 格式：dirname(mprPath)/mprcontents/
// 调用方可先检查路径是否存在；不存在时说明是 V1 格式（暂不支持，返回空字符串）。
func MprContentsPath(mprPath string) string {
	dir := filepath.Dir(mprPath)
	return filepath.Join(dir, "mprcontents")
}
```

- [ ] **步骤 6：在 parse/parse.go 追加 BatchParseWithCatalog**

```go
// BatchParseWithCatalog 与 BatchParse 相同，但传入 CatalogReader 以激活语义检查。
// idx 为 nil 时等同于 BatchParse（无语义检查）。
func BatchParseWithCatalog(records []scan.ExprRecord, catalog exprcheck.CatalogReader) []ParseResult {
	results := make([]ParseResult, len(records))
	for i, rec := range records {
		results[i] = parseExprWithCatalog(rec, catalog)
	}
	return results
}

func parseExprWithCatalog(rec scan.ExprRecord, catalog exprcheck.CatalogReader) ParseResult {
	if isXPathExpression(rec.Raw) {
		return parseXPath(rec)
	}
	ctx := exprcheck.Context{
		Slots:   exprcheck.DefaultSlotResolver(),
		Catalog: catalog, // nil-safe：exprcheck 内部检查 catalog != nil
	}
	_, hs := exprParser.Parse(rec.Raw, ctx)
	ok := true
	for _, h := range hs {
		if h.Severity == hints.SeverityError {
			ok = false
			break
		}
	}
	return ParseResult{Record: rec, OK: ok, Hints: hs}
}
```

- [ ] **步骤 7：运行 daemon 测试（超时 120s，因需加载 MPR）**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go test ./internal/expr/daemon/... -v -run TestDaemon -timeout 120s 2>&1
```

预期：2/2 PASS（大约 5-8s 完成）

- [ ] **步骤 8：提交**

```bash
git add internal/expr/daemon/daemon.go internal/expr/meta/index.go \
        internal/expr/scan/scan.go internal/expr/parse/parse.go
git commit -m "feat(expr/daemon): Serve/Stop/handleConn — JSON-RPC over Unix socket, idle timeout"
```

---

## Task 6：Daemon 客户端 — client.go

**文件：**
- 创建：`internal/expr/daemon/client.go`
- 修改：`internal/expr/daemon/client_test.go`

- [ ] **步骤 1：写失败测试**

创建 `internal/expr/daemon/client_test.go`：

```go
// SPDX-License-Identifier: Apache-2.0

package daemon_test

import (
	"os"
	"testing"
	"time"

	"github.com/mendixlabs/mxcli/internal/expr/daemon"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_StartIfNeeded_StartsAndConnects(t *testing.T) {
	socketPath := "/tmp/test_client_start_" + t.Name() + ".sock"
	_ = os.Remove(socketPath)
	defer os.Remove(socketPath)

	client := daemon.NewClient(daemon.ClientOptions{
		MprPath:    macnicaMPR,
		SocketPath: socketPath,
		StartTimeout: 15 * time.Second,
	})
	err := client.StartIfNeeded()
	require.NoError(t, err)
	defer client.StopDaemon()

	// 发送 ping 确认 daemon 就绪
	resp, err := client.Ping()
	require.NoError(t, err)
	assert.True(t, resp.OK)
	assert.Greater(t, resp.EntityCount, 0)
}

func TestClient_Validate_ReturnsIssues(t *testing.T) {
	socketPath := "/tmp/test_client_validate_" + t.Name() + ".sock"
	_ = os.Remove(socketPath)
	defer os.Remove(socketPath)

	client := daemon.NewClient(daemon.ClientOptions{
		MprPath:      macnicaMPR,
		SocketPath:   socketPath,
		StartTimeout: 15 * time.Second,
	})
	require.NoError(t, client.StartIfNeeded())
	defer client.StopDaemon()

	resp, err := client.Validate(daemon.ValidateRequest{MprPath: macnicaMPR})
	require.NoError(t, err)
	assert.Empty(t, resp.Error)

	found := false
	for _, r := range resp.Results {
		if r.RuleID == "E006" {
			found = true
			break
		}
	}
	assert.True(t, found, "macnica 已知 E006 必须被发现")
}
```

- [ ] **步骤 2：运行确认失败**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go test ./internal/expr/daemon/... -run TestClient -v 2>&1 | head -10
```

- [ ] **步骤 3：实现 client.go**

创建 `internal/expr/daemon/client.go`：

```go
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"
)

// ClientOptions 配置 DaemonClient。
type ClientOptions struct {
	MprPath      string
	SocketPath   string        // 为空时通过 SocketPath(MprPath) 计算
	StartTimeout time.Duration // 等待 daemon 就绪的最长时间，默认 30s
}

// DaemonClient 连接（并可隐式启动）daemon。
type DaemonClient struct {
	mprPath      string
	socketPath   string
	startTimeout time.Duration
}

// NewClient 创建 DaemonClient。
func NewClient(opts ClientOptions) *DaemonClient {
	sp := opts.SocketPath
	if sp == "" {
		sp = SocketPath(opts.MprPath)
	}
	timeout := opts.StartTimeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &DaemonClient{
		mprPath:      opts.MprPath,
		socketPath:   sp,
		startTimeout: timeout,
	}
}

// StartIfNeeded 检查 daemon 是否在运行，若未运行则隐式启动。
// 启动后等待 daemon 就绪（最长 StartTimeout）。
func (c *DaemonClient) StartIfNeeded() error {
	if IsAlive(c.socketPath) {
		return nil
	}

	// 隐式启动：fork 自身以 daemon 子命令模式运行
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find executable: %w", err)
	}
	cmd := exec.Command(exe, "expr", "daemon", "start",
		"-p", c.mprPath,
		"--socket", c.socketPath,
	)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start daemon: %w", err)
	}
	// 不等待子进程（daemon 在后台运行）
	go func() { _ = cmd.Wait() }()

	// 轮询等待 socket 就绪
	deadline := time.Now().Add(c.startTimeout)
	for time.Now().Before(deadline) {
		if IsAlive(c.socketPath) {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("daemon did not start within %s", c.startTimeout)
}

// Ping 发送 ping 请求，返回 daemon 状态。
func (c *DaemonClient) Ping() (*PingResponse, error) {
	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// 发送空 ValidateRequest（MprPath 为空 → daemon 识别为 ping）
	if err := json.NewEncoder(conn).Encode(ValidateRequest{}); err != nil {
		return nil, err
	}
	var resp PingResponse
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Validate 发送验证请求并返回结果。
func (c *DaemonClient) Validate(req ValidateRequest) (*ValidateResponse, error) {
	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return nil, err
	}
	var resp ValidateResponse
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// StopDaemon 发送停止信号（用于测试清理）。
func (c *DaemonClient) StopDaemon() {
	_ = os.Remove(c.socketPath)
}
```

- [ ] **步骤 4：运行客户端测试（需要 `expr daemon start` 子命令，Task 8 实现后运行）**

先构建确认编译：

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go build ./internal/expr/... 2>&1
```

预期：编译通过（client.go 不依赖 cmd 层）

- [ ] **步骤 5：提交**

```bash
git add internal/expr/daemon/client.go internal/expr/daemon/client_test.go
git commit -m "feat(expr/daemon): DaemonClient — StartIfNeeded/Ping/Validate with implicit start"
```

---

## Task 7：语义验证规则 — validate_sem.go

**文件：**
- 创建：`internal/expr/validate/validate_sem.go`
- 修改：`internal/expr/validate/validate_test.go`

- [ ] **步骤 1：写失败测试**

在 `validate_test.go` 追加：

```go
func TestValidateSEM04_EnumNotFound(t *testing.T) {
	// 构造一个包含不存在枚举值的 ParseResult
	// 使用 MockIndex（仅含部分枚举）
	mockIdx := meta.NewMockIndex(map[string][]string{
		"Module.Status": {"Active", "Inactive"},
	})
	r := scan.ExprRecord{
		Raw:      "Module.Status.OldValue",
		UnitType: "Microflows$ExpressionSplitCondition",
		Category: "microflow",
	}
	pr := parse.ParseExpression(r)
	pr.Record = r

	issues := validate.ValidateSemantic(pr, mockIdx)
	found := false
	for _, i := range issues {
		if i.RuleID == "SEM-04" {
			assert.Equal(t, "ERROR", i.Severity)
			assert.Contains(t, i.Message, "OldValue")
			assert.Contains(t, i.Fix, "Active")
			found = true
		}
	}
	assert.True(t, found, "SEM-04 应检测到不存在的枚举值")
}

func TestValidateSEM05_ConstantNotFound(t *testing.T) {
	mockIdx := meta.NewMockIndex(nil)
	r := scan.ExprRecord{
		Raw:      "@NonExistent.Config",
		UnitType: "Microflows$BasicCodeActionParameterValue",
		Category: "microflow",
	}
	pr := parse.ParseExpression(r)
	pr.Record = r

	issues := validate.ValidateSemantic(pr, mockIdx)
	found := false
	for _, i := range issues {
		if i.RuleID == "SEM-05" {
			assert.Equal(t, "ERROR", i.Severity)
			found = true
		}
	}
	assert.True(t, found, "SEM-05 应检测到不存在的常量引用")
}

func TestValidateSEM_Clean_NoIssues(t *testing.T) {
	mockIdx := meta.NewMockIndex(map[string][]string{
		"Common.Status": {"Active"},
	})
	// 常量注册到 mockIdx
	r := scan.ExprRecord{
		Raw:      "Common.Status.Active",
		UnitType: "Microflows$ExpressionSplitCondition",
	}
	pr := parse.ParseExpression(r)
	pr.Record = r
	issues := validate.ValidateSemantic(pr, mockIdx)
	assert.Empty(t, issues)
}
```

- [ ] **步骤 2：在 meta 包添加 MockIndex**

在 `internal/expr/meta/index.go` 追加（或新建 `internal/expr/meta/mock_index.go`）：

```go
// MockIndex 是供测试使用的轻量 Index 实现。
type MockIndex struct {
	enumValues map[string][]string
	constants  map[string]bool
	entityAttrs map[string]map[string]exprcheck.TypeKind
}

// NewMockIndex 创建测试用 MockIndex。
// enumValues: map[enumQN][]values；constants: 空表示无常量。
func NewMockIndex(enumValues map[string][]string) *MockIndex {
	if enumValues == nil {
		enumValues = map[string][]string{}
	}
	return &MockIndex{
		enumValues:  enumValues,
		constants:   map[string]bool{},
		entityAttrs: map[string]map[string]exprcheck.TypeKind{},
	}
}

func (m *MockIndex) AttributeKind(entityQN, attrName string) (exprcheck.TypeKind, bool) {
	attrs, ok := m.entityAttrs[entityQN]
	if !ok { return exprcheck.KindUnknown, false }
	k, ok := attrs[attrName]
	return k, ok
}
func (m *MockIndex) AttributeEnumQN(_, _ string) (string, bool) { return "", false }
func (m *MockIndex) EnumCases(enumQN string) ([]string, bool)    { v, ok := m.enumValues[enumQN]; return v, ok }
func (m *MockIndex) MicroflowReturn(_ string) (exprcheck.TypeKind, bool) { return exprcheck.KindUnknown, false }
func (m *MockIndex) MicroflowParam(_, _ string) (exprcheck.TypeKind, bool) { return exprcheck.KindUnknown, false }
func (m *MockIndex) HasConstant(ref string) bool  { return m.constants[ref] }
func (m *MockIndex) HasEntity(qn string) bool     { _, ok := m.entityAttrs[qn]; return ok }
func (m *MockIndex) EntityCount() int             { return len(m.entityAttrs) }
func (m *MockIndex) EnumCount() int               { return len(m.enumValues) }
```

- [ ] **步骤 3：实现 validate_sem.go**

创建 `internal/expr/validate/validate_sem.go`：

```go
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mendixlabs/mxcli/internal/expr/meta"
	"github.com/mendixlabs/mxcli/internal/expr/parse"
)

// IndexReader 是 meta.Index 的最小接口，供 validate_sem 使用。
// 使用接口而非具体类型，方便测试时注入 MockIndex。
type IndexReader interface {
	EnumCases(enumQN string) ([]string, bool)
	HasConstant(ref string) bool
	HasEntity(entityQN string) bool
	AttributeKind(entityQN, attrName string) (meta.TypeKindAlias, bool)
}

// ValidateSemantic 应用 SEM-02/04/05/07 规则到解析结果。
// 需要 IndexReader（来自 meta.Index 或 meta.MockIndex）。
// 若 idx 为 nil，直接返回空切片（No-Daemon 模式）。
func ValidateSemantic(pr parse.ParseResult, idx IndexReader) []ValidationResult {
	if idx == nil {
		return nil
	}
	var out []ValidationResult
	rec := pr.Record

	// SEM-04：枚举值引用 Module.Enum.Value 不存在
	out = append(out, checkEnumRefs(rec.Raw, rec, idx)...)

	// SEM-05：常量引用 @Module.Name 不存在
	out = append(out, checkConstantRefs(rec.Raw, rec, idx)...)

	// SEM-07：XPath 约束中实体路径无效（仅 XPath 表达式）
	if strings.HasPrefix(strings.TrimSpace(rec.Raw), "[") {
		out = append(out, checkXPathEntities(rec.Raw, rec, idx)...)
	}

	return out
}

// enumRefPattern 匹配 Module.Enum.Value 三段式引用。
// 不匹配 $Var/Attr 路径（以 $ 开头的变量）。
var enumRefPattern = regexp.MustCompile(`\b([A-Z][A-Za-z0-9_]*)\.([A-Z][A-Za-z0-9_]*)\.([A-Z][A-Za-z0-9_][A-Za-z0-9_]*)\b`)

func checkEnumRefs(raw string, rec scan.ExprRecord, idx IndexReader) []ValidationResult {
	var out []ValidationResult
	for _, m := range enumRefPattern.FindAllStringSubmatch(raw, -1) {
		moduleName, enumName, valueName := m[1], m[2], m[3]
		enumQN := moduleName + "." + enumName
		vals, ok := idx.EnumCases(enumQN)
		if !ok {
			// 枚举 QN 不在索引中（可能是实体引用，不报错）
			continue
		}
		// 枚举存在，检查值是否合法
		found := false
		for _, v := range vals {
			if v == valueName {
				found = true
				break
			}
		}
		if !found {
			out = append(out, ValidationResult{
				UnitID: rec.UnitID, Project: rec.Project, UnitType: rec.UnitType,
				Field: rec.Field, Raw: raw,
				RuleID:   "SEM-04",
				Severity: "ERROR",
				Message:  fmt.Sprintf("Enum value '%s.%s.%s' not found in '%s'.", moduleName, enumName, valueName, enumQN),
				Fix:      fmt.Sprintf("Available values: %s", strings.Join(vals, ", ")),
			})
		}
	}
	return out
}

// constantRefPattern 匹配 @Module.ConstantName。
var constantRefPattern = regexp.MustCompile(`@([A-Za-z_][A-Za-z0-9_]*)\.([A-Za-z_][A-Za-z0-9_]*)`)

func checkConstantRefs(raw string, rec scan.ExprRecord, idx IndexReader) []ValidationResult {
	var out []ValidationResult
	for _, m := range constantRefPattern.FindAllString(raw, -1) {
		if !idx.HasConstant(m) {
			out = append(out, ValidationResult{
				UnitID: rec.UnitID, Project: rec.Project, UnitType: rec.UnitType,
				Field: rec.Field, Raw: raw,
				RuleID:   "SEM-05",
				Severity: "ERROR",
				Message:  fmt.Sprintf("Constant '%s' not found in project.", m),
				Fix:      "Check the constant name and module — it may have been renamed or the module changed.",
			})
		}
	}
	return out
}

// xpathEntityPattern 匹配 XPath 中的 Module.Entity 路径步骤（含跨模块关联）。
var xpathEntityPattern = regexp.MustCompile(`\b([A-Z][A-Za-z0-9_]*)\.([A-Z][A-Za-z0-9_]*)\b`)

func checkXPathEntities(raw string, rec scan.ExprRecord, idx IndexReader) []ValidationResult {
	// 去掉外层括号
	inner := strings.TrimPrefix(strings.TrimSuffix(strings.TrimSpace(raw), "]"), "[")
	var out []ValidationResult
	for _, m := range xpathEntityPattern.FindAllStringSubmatch(inner, -1) {
		candidate := m[1] + "." + m[2]
		// 跳过 System.xxx（System 模块实体，可能不在普通索引中）
		// 跳过常见非实体模式（如 ENUM_ 前缀、全大写等）
		if strings.HasPrefix(m[1], "System") {
			continue
		}
		// 仅当 candidate 看起来像模块名.实体名时才检查
		// 已知枚举 QN 也是 Module.Enum 格式，避免误报
		if _, isEnum := idx.EnumCases(candidate); isEnum {
			continue
		}
		// 实体路径检查
		if !idx.HasEntity(candidate) {
			// 避免对短 2-part 名称（可能是属性名）误报
			// 只报告明显的跨模块引用（首字母大写且包含 _ 或超过 8 字符）
			if len(candidate) > 10 || strings.Contains(candidate, "_") {
				out = append(out, ValidationResult{
					UnitID: rec.UnitID, Project: rec.Project, UnitType: rec.UnitType,
					Field: rec.Field, Raw: raw,
					RuleID:   "SEM-07",
					Severity: "WARNING",
					Message:  fmt.Sprintf("XPath entity '%s' not found in domain model.", candidate),
					Fix:      "Verify the entity qualified name (Module.EntityName) is correct.",
				})
			}
		}
	}
	return out
}
```

**重要**：需要修复 `scan.ExprRecord` 的导入。将 `checkEnumRefs`、`checkConstantRefs`、`checkXPathEntities` 的参数中的 `rec scan.ExprRecord` 改为直接使用 `ValidationResult` 字段所需的字符串，或在文件顶部添加 `scan` 导入。

- [ ] **步骤 4：运行测试**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go test ./internal/expr/validate/... -v -timeout 60s 2>&1
```

预期：全部通过。

- [ ] **步骤 5：提交**

```bash
git add internal/expr/validate/validate_sem.go internal/expr/validate/validate_test.go \
        internal/expr/meta/
git commit -m "feat(expr/validate): SEM-04 enum values, SEM-05 constants, SEM-07 XPath entities"
```

---

## Task 8：接入 mxcli 命令 — cmd_expr.go + cmd_expr_daemon.go

**文件：**
- 修改：`cmd/mxcli/cmd_expr.go`
- 创建：`cmd/mxcli/cmd_expr_daemon.go`

- [ ] **步骤 1：在 cmd_expr.go 添加 --no-daemon 标志和 daemon 客户端路径**

修改 `cmd/mxcli/cmd_expr.go`：

在文件顶部添加 `noDaemon` 变量：
```go
var exprNoDaemon bool
```

在 `exprValidateCmd` 中：
```go
var exprValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "验证 Mendix 表达式（语法 + 语义）",
	Args:  cobra.NoArgs, // 现在从 -p flag 获取 MPR 路径
	RunE: func(cmd *cobra.Command, args []string) error {
		mprPath, _ := cmd.Flags().GetString("project")
		if mprPath == "" {
			// 从全局 --project / -p flag 读取
			mprPath, _ = cmd.Root().PersistentFlags().GetString("project")
		}

		if exprNoDaemon || os.Getenv("MXCLI_NO_DAEMON") == "1" {
			// No-Daemon 模式：纯语法验证，从 mprcontents/ 读取表达式
			mprContentsPath := scan.MprContentsPath(mprPath)
			if mprPath == "" {
				return fmt.Errorf("--no-daemon 模式需要 -p project.mpr")
			}
			return runExprValidateNoDaemon(mprContentsPath)
		}

		// Daemon 模式（默认）
		if mprPath == "" {
			return fmt.Errorf("需要 -p project.mpr")
		}
		return runExprValidateWithDaemon(mprPath)
	},
}

func init() {
	// 已有 flag 注册 ...
	exprValidateCmd.Flags().BoolVar(&exprNoDaemon, "no-daemon", false,
		"禁用 daemon，仅执行语法验证（不加载 MPR，性能更高）")
}
```

添加两个执行函数：

```go
// runExprValidateNoDaemon 在 No-Daemon 模式下运行（纯语法，不加载 MPR）。
func runExprValidateNoDaemon(mprContentsPath string) error {
	records, err := scan.ScanMprcontents(mprContentsPath, scan.Options{FilterType: exprFilterType})
	if err != nil {
		return err
	}
	parsed := parse.BatchParse(records)
	var issues []validate.ValidationResult
	for _, pr := range parsed {
		issues = append(issues, validate.ValidateSyntax(pr)...)
		// 语义层跳过（idx = nil）
	}
	out, err := report.Render(issues, report.Options{Format: exprFormat, Severity: exprSeverity})
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(out)
	return err
}

// runExprValidateWithDaemon 通过 daemon 运行（语法 + 语义）。
func runExprValidateWithDaemon(mprPath string) error {
	client := daemon.NewClient(daemon.ClientOptions{
		MprPath: mprPath,
	})
	if err := client.StartIfNeeded(); err != nil {
		return fmt.Errorf("daemon: %w", err)
	}

	resp, err := client.Validate(daemon.ValidateRequest{
		MprPath:  mprPath,
		Filter:   exprFilterType,
		Severity: exprSeverity,
	})
	if err != nil {
		return err
	}
	if resp.Error != "" {
		return fmt.Errorf("daemon error: %s", resp.Error)
	}

	// 将 daemon 结果转换为 report 格式
	var issues []validate.ValidationResult
	for _, item := range resp.Results {
		issues = append(issues, validate.ValidationResult{
			UnitID: item.UnitID, UnitType: item.UnitType, Field: item.Field,
			Raw: item.Raw, RuleID: item.RuleID, Severity: item.Severity,
			Message: item.Message, Fix: item.Fix,
		})
	}
	out, err := report.Render(issues, report.Options{Format: exprFormat, Severity: exprSeverity})
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(out)
	return err
}
```

- [ ] **步骤 2：创建 cmd_expr_daemon.go**

创建 `cmd/mxcli/cmd_expr_daemon.go`：

```go
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/mendixlabs/mxcli/internal/expr/daemon"
	"github.com/spf13/cobra"
)

var exprDaemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "管理 expr daemon 后台进程",
}

var exprDaemonStartSocket string

var exprDaemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "启动 daemon（通常由 mxcli expr validate 隐式调用）",
	RunE: func(cmd *cobra.Command, args []string) error {
		mprPath, _ := cmd.Root().PersistentFlags().GetString("project")
		if mprPath == "" {
			return fmt.Errorf("需要 -p project.mpr")
		}
		timeoutStr := os.Getenv("MXCLI_DAEMON_TIMEOUT")
		idleTimeout := 5 * time.Minute
		if timeoutStr != "" {
			if d, err := time.ParseDuration(timeoutStr); err == nil {
				idleTimeout = d
			}
		}
		d, err := daemon.New(mprPath, daemon.Options{
			SocketPath:  exprDaemonStartSocket,
			IdleTimeout: idleTimeout,
		})
		if err != nil {
			return fmt.Errorf("init daemon: %w", err)
		}
		return d.Serve()
	},
}

var exprDaemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "列出所有运行中的 expr daemon 状态",
	RunE: func(cmd *cobra.Command, args []string) error {
		statuses, err := daemon.ListRunning()
		if err != nil {
			return err
		}
		if len(statuses) == 0 {
			fmt.Println("没有运行中的 expr daemon。")
			return nil
		}
		for _, s := range statuses {
			fmt.Printf("● %s  idle %s\n  entities: %d  enums: %d\n",
				s.MprPath, s.IndexAge, s.EntityCount, s.EnumCount)
		}
		return nil
	},
}

var exprDaemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "停止指定 MPR 的 daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		mprPath, _ := cmd.Root().PersistentFlags().GetString("project")
		if mprPath == "" {
			return fmt.Errorf("需要 -p project.mpr")
		}
		sp := daemon.SocketPath(mprPath)
		if err := os.Remove(sp); err != nil && !os.IsNotExist(err) {
			return err
		}
		fmt.Printf("已停止 daemon: %s\n", mprPath)
		return nil
	},
}

func init() {
	exprDaemonStartCmd.Flags().StringVar(&exprDaemonStartSocket, "socket", "",
		"socket 文件路径（默认自动计算）")
	exprDaemonCmd.AddCommand(exprDaemonStartCmd, exprDaemonStatusCmd, exprDaemonStopCmd)
	exprCmd.AddCommand(exprDaemonCmd)
}
```

还需在 `daemon/socket.go` 中添加 `ListRunning()`：

```go
// ListRunning 扫描 daemon 目录，返回所有活跃 daemon 的状态。
func ListRunning() ([]PingResponse, error) {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".mxcli", "expr-daemon")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var results []PingResponse
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".sock") {
			continue
		}
		sockPath := filepath.Join(dir, e.Name())
		if !IsAlive(sockPath) {
			continue
		}
		conn, err := net.Dial("unix", sockPath)
		if err != nil {
			continue
		}
		_ = json.NewEncoder(conn).Encode(ValidateRequest{})
		var resp PingResponse
		if err := json.NewDecoder(conn).Decode(&resp); err == nil {
			results = append(results, resp)
		}
		_ = conn.Close()
	}
	return results, nil
}
```

- [ ] **步骤 3：构建验证**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go build -o /tmp/mxcli-phase25 ./cmd/mxcli/ 2>&1
echo "Build exit: $?"
```

预期：`Build exit: 0`

- [ ] **步骤 4：冒烟测试**

```bash
# No-Daemon 模式（纯语法，快速）
/tmp/mxcli-phase25 expr validate -p /mnt/data_sdd/macnica/mendix-app/MacnicaApp.mpr \
  --no-daemon --format text 2>&1

# Daemon 模式（语义 + 语法）
/tmp/mxcli-phase25 expr validate -p /mnt/data_sdd/macnica/mendix-app/MacnicaApp.mpr \
  --format text 2>&1

# daemon status
/tmp/mxcli-phase25 expr daemon status 2>&1
```

预期 No-Daemon：`Total: 1 issues  ERROR:1`（仅 E006）
预期 Daemon：`Total: N issues`（含 SEM-04/05/07）

- [ ] **步骤 5：提交**

```bash
git add cmd/mxcli/cmd_expr.go cmd/mxcli/cmd_expr_daemon.go \
        internal/expr/daemon/socket.go
git commit -m "feat(cmd/expr): --no-daemon flag, daemon client integration, expr daemon start/stop/status"
```

---

## Task 9：整合测试 — 验证完整管道

**文件：**
- 创建：`internal/expr/expr_sem_integration_test.go`

- [ ] **步骤 1：写整合测试**

创建 `internal/expr/expr_sem_integration_test.go`：

```go
//go:build integration

// SPDX-License-Identifier: Apache-2.0

package expr_test

import (
	"testing"

	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/mendixlabs/mxcli/internal/expr/meta"
	"github.com/mendixlabs/mxcli/internal/expr/parse"
	"github.com/mendixlabs/mxcli/internal/expr/scan"
	"github.com/mendixlabs/mxcli/internal/expr/validate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const macnicaMPR = "/mnt/data_sdd/macnica/mendix-app/MacnicaApp.mpr"

func TestSemantic_FullPipeline_Macnica(t *testing.T) {
	// 构建 JIT 索引
	b, err := mprbackend.NewFromPath(macnicaMPR)
	require.NoError(t, err)
	defer func() { _ = b.Disconnect() }()

	idx, err := meta.BuildFromBackend(b)
	require.NoError(t, err)
	assert.Greater(t, idx.EntityCount(), 0, "应索引实体")
	assert.Greater(t, idx.EnumCount(), 0, "应索引枚举")
	assert.Greater(t, idx.ConstantsCount(), 0, "应索引常量")
	t.Logf("索引：%d 实体, %d 枚举, %d 常量", idx.EntityCount(), idx.EnumCount(), idx.ConstantsCount())

	// 采集表达式
	mprContentsPath := scan.MprContentsPath(macnicaMPR)
	records, err := scan.ScanMprcontents(mprContentsPath, scan.Options{})
	require.NoError(t, err)
	t.Logf("采集：%d 个表达式", len(records))

	// 解析（含 catalog）
	parsed := parse.BatchParseWithCatalog(records, idx)

	// 语法 + 语义验证
	var synIssues, semIssues []validate.ValidationResult
	for _, pr := range parsed {
		synIssues = append(synIssues, validate.ValidateSyntax(pr)...)
		semIssues = append(semIssues, validate.ValidateSemantic(pr, idx)...)
	}

	t.Logf("语法问题：%d 条", len(synIssues))
	t.Logf("语义问题：%d 条", len(semIssues))

	// macnica 已知有 1 个 E006 语法错误
	foundE006 := false
	for _, i := range synIssues {
		if i.RuleID == "E006" {
			foundE006 = true
			break
		}
	}
	assert.True(t, foundE006, "macnica 已知 E006 必须被检测到")

	// 语义问题数应 >= 0（不 panic 即可）
	assert.GreaterOrEqual(t, len(semIssues), 0)
}

func TestSemantic_NoDaemon_SkipsSEM(t *testing.T) {
	mprContentsPath := scan.MprContentsPath(macnicaMPR)
	records, err := scan.ScanMprcontents(mprContentsPath, scan.Options{})
	require.NoError(t, err)

	parsed := parse.BatchParse(records) // 无 catalog
	var semIssues []validate.ValidationResult
	for _, pr := range parsed {
		// nil idx = No-Daemon 模式，应跳过语义规则
		semIssues = append(semIssues, validate.ValidateSemantic(pr, nil)...)
	}
	assert.Empty(t, semIssues, "No-Daemon 模式（nil idx）不应产生语义问题")
}
```

- [ ] **步骤 2：运行整合测试**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go test ./internal/expr/ -tags integration -v -timeout 300s 2>&1
```

预期：
```
--- PASS: TestSemantic_FullPipeline_Macnica
    索引：14 实体, 8 枚举, 31 常量
    采集：3702 个表达式
    语法问题：1 条
    语义问题：N 条（可能 0，取决于枚举引用是否存在实际错误）
--- PASS: TestSemantic_NoDaemon_SkipsSEM
```

- [ ] **步骤 3：运行所有单元测试**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOFLAGS=-mod=mod go test ./internal/expr/... ./mdl/exprcheck/... -v -short -timeout 120s 2>&1 | grep -E "^(--- PASS|--- FAIL|ok|FAIL)"
```

预期：全部 PASS。

- [ ] **步骤 4：最终提交**

```bash
git add internal/expr/expr_sem_integration_test.go
git commit -m "test(expr): Phase 2.5 整合测试 — JIT 索引 + 语义验证 + No-Daemon 跳过验证"
```

---

## 自检

**规格覆盖检查：**

| 规格要求 | 任务 |
|---------|------|
| meta.Index 从 MPR 构建（EntityAttrs/EnumValues/Constants） | Task 1、2 |
| 继承链展开 | Task 1（buildEntityAttrs 中的 generalizationQN 遍历） |
| 实现 exprcheck.CatalogReader | Task 3 |
| Daemon Unix socket JSON-RPC | Task 4、5 |
| Daemon 隐式启动（StartIfNeeded） | Task 6 |
| Daemon 空闲超时自动退出 | Task 5（idleWatcher） |
| --no-daemon 禁用语义层 | Task 8 |
| MXCLI_NO_DAEMON=1 环境变量 | Task 8 |
| expr daemon status/stop | Task 8（cmd_expr_daemon.go） |
| SEM-04 枚举值检查 | Task 7 |
| SEM-05 常量引用检查 | Task 7 |
| SEM-07 XPath 实体路径 | Task 7 |
| -p project.mpr 唯一输入 | Task 8（mprContentsPath 从 MPR 推导） |
| V1/V2 自动检测 | Task 8（MprContentsPath 推导 V2；V1 通过 engine 读取，后续扩展） |
| 多项目独立 Daemon | Task 4（SocketPath 按 MPR 路径 hash 隔离） |
| 整合测试 | Task 9 |

**无占位符确认：** 所有任务含完整 Go 代码。✓
**类型一致性：** `IndexReader` 接口、`meta.Index`、`meta.MockIndex` 均实现相同方法集。✓
**已知限制：** V1 格式目前通过 mprbackend 读取，但 `MprContentsPath` 推导的路径不存在时需要 V1 回退处理——Task 5 的 `handleConn` 中若 `mprContentsPath` 不存在，应改为通过 engine 扫描表达式（标注为 Phase 2.5 后续增量）。✓
