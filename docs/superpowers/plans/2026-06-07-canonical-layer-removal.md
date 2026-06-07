# canonical 生命周期层删除实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 删除 `mdl/canonical/entity/`、`mdl/canonical/association/`、`mdl/canonical/registry.go`、`mdl/canonical/context.go`，消除 executor 对 canonical 生命周期基础设施的所有依赖，保留 `canonical.DataType`。

**Architecture:** 将 canonical/entity/{serialize,hydrate,lift,persist}.go 和 canonical/association/{serialize,lift,persist}.go 中的逻辑直接迁移到 executor 中，以 `entityMDLSpec`/`assocMDLSpec` 本地结构体替代中间 `EntityModel`/`AssociationModel`，从而消除 `DefaultRegistry`、`HydrateFrom`、`LiftFrom` 调用。写路径 (`persistEntityDirect`) 将 Lift+buildGenEntity 合并为一个直接从 AST 构建 gen entity 的函数，然后调用 `ctx.Backend.CreateEntityGen/UpdateEntityGen`。

**Tech Stack:** `mdl/canonical/datatype.go`（保留）、`modelsdk/gen/domainmodels`（gen types）、`mdl/ast`（AST types）、`mdl/backend`（已有方法：`CreateEntityGen`, `UpdateEntityGen`, `CreateAssociationGen`, `GetEntityIDByQualifiedName`）

---

## 文件结构

| 文件 | 操作 | 职责 |
|------|------|------|
| `mdl/executor/entity_mdl_render.go` | **新建** | `entityMDLSpec` 结构 + `renderEntityMDL()` 渲染器（MDL text 生成，替代 entity/serialize.go + entity/hydrate.go）|
| `mdl/executor/entity_mdl_render_test.go` | **新建** | `renderEntityMDL` 单元测试 |
| `mdl/executor/assoc_mdl_render.go` | **新建** | `assocMDLSpec` 结构 + `renderAssocMDL()` 渲染器（替代 association/serialize.go + association/hydrate.go）|
| `mdl/executor/assoc_mdl_render_test.go` | **新建** | `renderAssocMDL` 单元测试 |
| `mdl/executor/entity_from_ast.go` | **新建** | `buildEntityFromAST()` + helpers（替代 entity/lift.go + entity/persist.go 的 buildGenEntity；写路径用）|
| `mdl/executor/entity_from_ast_test.go` | **新建** | `buildEntityFromAST` 单元测试 |
| `mdl/executor/cmd_diff_mdl.go` | **修改** | 替换 `entityStmtToMDL`、`associationStmtToMDL`、`entityToMDLGen`、`associationToMDLGen` 的实现；删除 `ctx` 参数依赖；删除 canonical import |
| `mdl/executor/cmd_entities_gen.go` | **修改** | `describeEntityGen`：替换 `hydrateEntityModel` → `entityGenToMDL` 调用 |
| `mdl/executor/cmd_create_entity_gen.go` | **修改** | `persistEntityCanonical` 改名为 `persistEntityDirect`；删除 `LiftFrom`/`PersistContext`；使用 `buildEntityFromAST` |
| `mdl/executor/executor.go` | **修改** | 删除 `hydrateEntityModel`、`NewDefaultRegistry()`、`RegisterCodec()` 调用、`modelCodecs` 字段 |
| `mdl/executor/exec_context.go` | **修改** | 删除 `ModelCodecs` 字段 |
| `mdl/executor/executor_dispatch.go` | **修改** | 删除 `ModelCodecs: e.modelCodecs` 行 |
| `mdl/canonical/entity/` (整个目录) | **删除** | |
| `mdl/canonical/association/` (整个目录) | **删除** | |
| `mdl/canonical/registry.go` | **删除** | |
| `mdl/canonical/context.go` | **删除** | |
| `mdl/canonical/doc.go` | **修改** | 更新包说明 |
| `mdl/canonical/import_guard_test.go` | **新建** | 确保 executor 不再 import canonical 子包 |

---

### Task 1: 实现 `entityMDLSpec` 渲染器（TDD）

**Files:**
- Create: `mdl/executor/entity_mdl_render_test.go`
- Create: `mdl/executor/entity_mdl_render.go`

- [ ] **Step 1: 写失败测试**

```go
// mdl/executor/entity_mdl_render_test.go
package executor

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/canonical"
)

func TestRenderEntityMDL_simple(t *testing.T) {
	spec := entityMDLSpec{
		module: "MyModule",
		name:   "MyEntity",
		kind:   "persistent",
		attributes: []attrMDLSpec{
			{name: "Name", dataType: canonical.DataType{Kind: canonical.KindString, Length: 200}},
		},
	}
	out := renderEntityMDL(spec, false)
	if !strings.Contains(out, "create persistent entity MyModule.MyEntity") {
		t.Errorf("expected entity header, got:\n%s", out)
	}
	if !strings.Contains(out, "Name: String(200)") {
		t.Errorf("expected Name attribute, got:\n%s", out)
	}
}

func TestRenderEntityMDL_createOrModify(t *testing.T) {
	spec := entityMDLSpec{module: "M", name: "E", kind: "persistent"}
	out := renderEntityMDL(spec, true)
	if !strings.HasPrefix(out, "create or modify persistent entity M.E") {
		t.Errorf("expected 'create or modify' prefix, got:\n%s", out)
	}
}

func TestRenderEntityMDL_nonPersistent(t *testing.T) {
	spec := entityMDLSpec{module: "M", name: "E", kind: "non-persistent"}
	out := renderEntityMDL(spec, false)
	if !strings.Contains(out, "create non-persistent entity M.E") {
		t.Errorf("expected non-persistent, got:\n%s", out)
	}
}

func TestRenderEntityMDL_withDocAndPosition(t *testing.T) {
	spec := entityMDLSpec{
		module:        "M",
		name:          "E",
		kind:          "persistent",
		documentation: "hello",
		positionX:     10,
		positionY:     20,
		hasPosition:   true,
	}
	out := renderEntityMDL(spec, false)
	if !strings.Contains(out, "/**\n * hello\n */") {
		t.Errorf("expected doc, got:\n%s", out)
	}
	if !strings.Contains(out, "@Position(10, 20)") {
		t.Errorf("expected position, got:\n%s", out)
	}
}
```

- [ ] **Step 2: 确认测试失败**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./mdl/executor/... -run "TestRenderEntityMDL" -v 2>&1 | tail -10
```

预期: `FAIL` — `entityMDLSpec undefined`

- [ ] **Step 3: 实现 `entity_mdl_render.go`**

```go
// mdl/executor/entity_mdl_render.go
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/canonical"
)

// entityMDLSpec is a minimal intermediate representation for rendering an entity
// as MDL text. It replaces the canonical EntityModel for diffing and DESCRIBE.
type entityMDLSpec struct {
	module        string
	name          string
	kind          string // "persistent", "non-persistent", "view", "external"
	documentation string
	hasPosition   bool
	positionX     int
	positionY     int
	extendsQN     string // qualified name "Module.Entity", "" = no extends
	attributes    []attrMDLSpec
	indexes       []indexMDLSpec
	eventHandlers []eventHandlerMDLSpec
	systemMembers []string
	oql           string // only for view entities
}

type attrMDLSpec struct {
	name               string
	documentation      string
	dataType           canonical.DataType
	notNull            bool
	notNullError       string
	unique             bool
	uniqueError        string
	hasDefault         bool
	defaultValue       string // pre-formatted MDL literal
	calculated         bool
	calculatedMFQN     string // qualified name "Module.Microflow"
}

type indexMDLSpec struct {
	name    string // may be UUID; omitted if empty
	columns []indexColumnMDLSpec
}

type indexColumnMDLSpec struct {
	name      string
	ascending bool
}

type eventHandlerMDLSpec struct {
	moment            string
	event             string
	microflowQN       string
	raiseErrorOnFalse bool
	passEventObject   bool
}

// renderEntityMDL serialises an entityMDLSpec to deterministic MDL text.
// When createOrModify is true the statement begins with "create or modify".
func renderEntityMDL(spec entityMDLSpec, createOrModify bool) string {
	var sb strings.Builder
	if spec.documentation != "" {
		fmt.Fprintf(&sb, "/**\n * %s\n */\n", spec.documentation)
	}
	if spec.hasPosition {
		fmt.Fprintf(&sb, "@Position(%d, %d)\n", spec.positionX, spec.positionY)
	}
	prefix := "create"
	if createOrModify {
		prefix = "create or modify"
	}
	qn := spec.module + "." + spec.name
	if spec.extendsQN != "" {
		fmt.Fprintf(&sb, "%s %s entity %s extends %s (\n", prefix, spec.kind, qn, spec.extendsQN)
	} else {
		fmt.Fprintf(&sb, "%s %s entity %s (\n", prefix, spec.kind, qn)
	}
	for i, attr := range spec.attributes {
		if attr.documentation != "" {
			fmt.Fprintf(&sb, "  /** %s */\n", attr.documentation)
		}
		comma := ","
		if i == len(spec.attributes)-1 {
			comma = ""
		}
		fmt.Fprintf(&sb, "  %s: %s%s%s\n", attr.name, entityDataTypeToMDL(attr.dataType), entityAttrConstraintsToMDL(attr), comma)
	}
	sb.WriteString(")")
	for _, idx := range spec.indexes {
		cols := make([]string, 0, len(idx.columns))
		for _, col := range idx.columns {
			if col.ascending {
				cols = append(cols, col.name)
			} else {
				cols = append(cols, col.name+" desc")
			}
		}
		if idx.name != "" {
			fmt.Fprintf(&sb, "\nindex %s (%s)", idx.name, strings.Join(cols, ", "))
		} else {
			fmt.Fprintf(&sb, "\nindex (%s)", strings.Join(cols, ", "))
		}
	}
	for _, eh := range spec.eventHandlers {
		paramStr := "()"
		if eh.passEventObject {
			paramStr = "($currentObject)"
		}
		options := ""
		if eh.raiseErrorOnFalse && strings.EqualFold(eh.moment, "Before") {
			options = " raise error"
		}
		fmt.Fprintf(&sb, "\non %s %s call %s%s%s",
			strings.ToLower(eh.moment), strings.ToLower(eh.event),
			eh.microflowQN, paramStr, options)
	}
	if len(spec.systemMembers) > 0 {
		fmt.Fprintf(&sb, "\nsystem members (%s)", strings.Join(spec.systemMembers, ", "))
	}
	if spec.kind == "view" && spec.oql != "" {
		sb.WriteString(" as (\n")
		for _, line := range strings.Split(spec.oql, "\n") {
			fmt.Fprintf(&sb, "  %s\n", line)
		}
		sb.WriteString(")")
	}
	return sb.String()
}

func entityDataTypeToMDL(dt canonical.DataType) string {
	switch dt.Kind {
	case canonical.KindString:
		if dt.Length > 0 {
			return fmt.Sprintf("String(%d)", dt.Length)
		}
		return "String"
	case canonical.KindInteger:
		return "Integer"
	case canonical.KindLong:
		return "Long"
	case canonical.KindDecimal:
		if dt.Precision > 0 {
			return fmt.Sprintf("Decimal(%d, %d)", dt.Precision, dt.Scale)
		}
		return "Decimal"
	case canonical.KindBoolean:
		return "Boolean"
	case canonical.KindDateTime:
		return "DateTime"
	case canonical.KindBinary:
		return "Binary"
	case canonical.KindAutoNumber:
		return "AutoNumber"
	case canonical.KindEnumRef, canonical.KindEntityRef, canonical.KindUnresolvedRef:
		return dt.Ref
	case canonical.KindListOf:
		return "List of " + dt.Ref
	default:
		return "Unknown"
	}
}

func entityAttrConstraintsToMDL(attr attrMDLSpec) string {
	var sb strings.Builder
	if attr.notNull {
		sb.WriteString(" not null")
		if attr.notNullError != "" {
			fmt.Fprintf(&sb, " error '%s'", strings.ReplaceAll(attr.notNullError, "'", "''"))
		}
	}
	if attr.unique {
		sb.WriteString(" unique")
		if attr.uniqueError != "" {
			fmt.Fprintf(&sb, " error '%s'", strings.ReplaceAll(attr.uniqueError, "'", "''"))
		}
	}
	if attr.hasDefault {
		fmt.Fprintf(&sb, " default %s", attr.defaultValue)
	}
	if attr.calculated && attr.calculatedMFQN != "" {
		fmt.Fprintf(&sb, " calculated by %s", attr.calculatedMFQN)
	}
	return sb.String()
}
```

- [ ] **Step 4: 确认测试通过**

```bash
go test ./mdl/executor/... -run "TestRenderEntityMDL" -v 2>&1 | tail -10
```

预期: 所有 `TestRenderEntityMDL_*` — PASS

- [ ] **Step 5: 提交**

```bash
git add mdl/executor/entity_mdl_render.go mdl/executor/entity_mdl_render_test.go
git commit -m "refactor(canonical): add entity MDL renderer to replace canonical entity/serialize"
```

---

### Task 2: 实现 `assocMDLSpec` 渲染器（TDD）

**Files:**
- Create: `mdl/executor/assoc_mdl_render_test.go`
- Create: `mdl/executor/assoc_mdl_render.go`

- [ ] **Step 1: 写失败测试**

```go
// mdl/executor/assoc_mdl_render_test.go
package executor

import (
	"strings"
	"testing"
)

func TestRenderAssocMDL_basic(t *testing.T) {
	spec := assocMDLSpec{
		module:         "Mod",
		name:           "Child_Parent",
		fromQN:         "Mod.Child",
		toQN:           "Mod.Parent",
		assocType:      "Reference",
		owner:          "Default",
		deleteBehavior: "DeleteMeButKeepReferences",
	}
	out := renderAssocMDL(spec)
	if !strings.Contains(out, "create association Mod.Child_Parent") {
		t.Errorf("expected association header, got:\n%s", out)
	}
	if !strings.Contains(out, "from Mod.Child to Mod.Parent") {
		t.Errorf("expected from/to, got:\n%s", out)
	}
	if !strings.Contains(out, "type Reference") {
		t.Errorf("expected type, got:\n%s", out)
	}
}

func TestRenderAssocMDL_withDoc(t *testing.T) {
	spec := assocMDLSpec{
		module:         "M",
		name:           "A_B",
		fromQN:         "M.A",
		toQN:           "M.B",
		documentation:  "my doc",
		assocType:      "Reference",
		owner:          "Default",
		deleteBehavior: "DeleteMeButKeepReferences",
	}
	out := renderAssocMDL(spec)
	if !strings.Contains(out, "/**\n * my doc\n */") {
		t.Errorf("expected doc, got:\n%s", out)
	}
}
```

- [ ] **Step 2: 确认测试失败**

```bash
go test ./mdl/executor/... -run "TestRenderAssocMDL" -v 2>&1 | tail -5
```

预期: `FAIL` — `assocMDLSpec undefined`

- [ ] **Step 3: 实现 `assoc_mdl_render.go`**

```go
// mdl/executor/assoc_mdl_render.go
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"strings"
)

// assocMDLSpec is the minimal representation for rendering an association as MDL.
type assocMDLSpec struct {
	module         string
	name           string
	fromQN         string // qualified name of FROM (ParentPointer) entity
	toQN           string // qualified name of TO (ChildPointer) entity
	documentation  string
	assocType      string // "Reference" | "ReferenceSet"
	owner          string // "Default" | "Both"
	deleteBehavior string // MDL keyword string
}

// renderAssocMDL serialises an assocMDLSpec to deterministic MDL text.
func renderAssocMDL(spec assocMDLSpec) string {
	var sb strings.Builder
	if spec.documentation != "" {
		fmt.Fprintf(&sb, "/**\n * %s\n */\n", spec.documentation)
	}
	fmt.Fprintf(&sb, "create association %s.%s\n", spec.module, spec.name)
	fmt.Fprintf(&sb, "from %s to %s\n", spec.fromQN, spec.toQN)
	fmt.Fprintf(&sb, "type %s\n", spec.assocType)
	fmt.Fprintf(&sb, "owner %s\n", spec.owner)
	fmt.Fprintf(&sb, "delete behavior %s", spec.deleteBehavior)
	return sb.String()
}

// astAssocTypeToMDL maps ast.AssociationType to MDL string.
func astAssocTypeToMDL(t string) string {
	if t == "ReferenceSet" {
		return "ReferenceSet"
	}
	return "Reference"
}

// genAssocDeleteBehaviorToMDL converts the gen storage strings back to MDL keyword.
func genAssocDeleteBehaviorToMDL(child string) string {
	switch child {
	case "DeleteMeAndReferences":
		return "DeleteMeAndReferences"
	case "DeleteBoth":
		return "DeleteBoth"
	case "KeepParentDeleteChild":
		return "KeepParentDeleteChild"
	case "KeepChildDeleteParent":
		return "KeepChildDeleteParent"
	case "DeleteIfNoReferences":
		return "DeleteIfNoReferences"
	default:
		return "DeleteMeButKeepReferences"
	}
}
```

- [ ] **Step 4: 确认测试通过**

```bash
go test ./mdl/executor/... -run "TestRenderAssocMDL" -v 2>&1 | tail -5
```

预期: 所有 `TestRenderAssocMDL_*` — PASS

- [ ] **Step 5: 提交**

```bash
git add mdl/executor/assoc_mdl_render.go mdl/executor/assoc_mdl_render_test.go
git commit -m "refactor(canonical): add assoc MDL renderer to replace canonical association/serialize"
```

---

### Task 3: 实现 `buildEntityFromAST`（TDD，写路径）

**Files:**
- Create: `mdl/executor/entity_from_ast_test.go`
- Create: `mdl/executor/entity_from_ast.go`

- [ ] **Step 1: 写失败测试**

```go
// mdl/executor/entity_from_ast_test.go
package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/canonical"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

func TestBuildEntityFromAST_basic(t *testing.T) {
	stmt := &ast.CreateEntityStmt{
		Name: ast.QualifiedName{Module: "M", Name: "MyEntity"},
		Kind: ast.EntityPersistent,
		Attributes: []ast.Attribute{
			{Name: "Name", Type: ast.DataType{Kind: ast.TypeString, Length: 200}},
		},
	}
	gen, err := buildEntityFromAST("M", stmt)
	if err != nil {
		t.Fatalf("buildEntityFromAST: %v", err)
	}
	if gen.Name() != "MyEntity" {
		t.Errorf("expected name 'MyEntity', got %q", gen.Name())
	}
	attrs := gen.AttributesItems()
	if len(attrs) != 1 {
		t.Errorf("expected 1 attribute, got %d", len(attrs))
	}
}

func TestBuildEntityFromAST_booleanDefaultFalse(t *testing.T) {
	stmt := &ast.CreateEntityStmt{
		Name: ast.QualifiedName{Module: "M", Name: "E"},
		Kind: ast.EntityPersistent,
		Attributes: []ast.Attribute{
			{Name: "Active", Type: ast.DataType{Kind: ast.TypeBoolean}},
		},
	}
	gen, err := buildEntityFromAST("M", stmt)
	if err != nil {
		t.Fatalf("buildEntityFromAST: %v", err)
	}
	attrs := gen.AttributesItems()
	if len(attrs) != 1 {
		t.Fatalf("expected 1 attribute, got %d", len(attrs))
	}
	attr, ok := attrs[0].(*genDm.Attribute)
	if !ok {
		t.Fatalf("expected *genDm.Attribute, got %T", attrs[0])
	}
	sv, ok := attr.Value().(*genDm.StoredValue)
	if !ok {
		t.Fatalf("expected StoredValue for boolean default, got %T", attr.Value())
	}
	if sv.DefaultValue() != "false" {
		t.Errorf("expected default 'false', got %q", sv.DefaultValue())
	}
}

func TestASTDataTypeToCanonical_types(t *testing.T) {
	cases := []struct {
		ast  ast.DataType
		kind canonical.DataTypeKind
	}{
		{ast.DataType{Kind: ast.TypeString, Length: 100}, canonical.KindString},
		{ast.DataType{Kind: ast.TypeInteger}, canonical.KindInteger},
		{ast.DataType{Kind: ast.TypeBoolean}, canonical.KindBoolean},
		{ast.DataType{Kind: ast.TypeAutoNumber}, canonical.KindAutoNumber},
	}
	for _, tc := range cases {
		got := astDataTypeToCanonical(tc.ast)
		if got.Kind != tc.kind {
			t.Errorf("astDataTypeToCanonical(%v): got kind %d, want %d", tc.ast, got.Kind, tc.kind)
		}
	}
}
```

- [ ] **Step 2: 确认测试失败**

```bash
go test ./mdl/executor/... -run "TestBuildEntityFromAST|TestASTDataType" -v 2>&1 | tail -10
```

预期: `FAIL` — `buildEntityFromAST undefined`

- [ ] **Step 3: 实现 `entity_from_ast.go`**

```go
// mdl/executor/entity_from_ast.go
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"crypto/rand"
	"fmt"
	"math"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/canonical"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genTexts "github.com/mendixlabs/mxcli/modelsdk/gen/texts"
)

// buildEntityFromAST converts an ast.CreateEntityStmt directly to a *genDm.Entity
// ready to pass to ctx.Backend.CreateEntityGen / UpdateEntityGen.
//
// Invariants injected here (matching canonical/entity/persist.go behaviour):
//   - Boolean attributes without an explicit default get DefaultValue="false".
//   - AutoNumber attributes without a seed get DefaultValue="1".
//   - Enum default values are stripped to the trailing segment.
func buildEntityFromAST(moduleName string, s *ast.CreateEntityStmt) (*genDm.Entity, error) {
	e := genDm.NewEntity()
	e.SetName(s.Name.Name)
	if s.Documentation != "" {
		e.SetDocumentation(s.Documentation)
	}
	if s.Position != nil {
		e.SetLocation(fmt.Sprintf("%d;%d", s.Position.X, s.Position.Y))
	}

	// Generalization / NoGeneralization.
	if s.Generalization != nil {
		g := genDm.NewGeneralization()
		g.SetGeneralizationQualifiedName(s.Generalization.String())
		e.SetGeneralization(g)
	} else {
		ng := genDm.NewNoGeneralization()
		ng.SetPersistable(s.Kind != ast.EntityNonPersistent)
		applySystemMembersFromSlice(ng, s.SystemMembers)
		e.SetGeneralization(ng)
	}

	// Attributes — assign IDs eagerly so Indexes can reference them.
	attrIDs := make(map[string]element.ID, len(s.Attributes))
	entityQN := s.Name.Module + "." + s.Name.Name
	for _, a := range s.Attributes {
		dt := astDataTypeToCanonical(a.Type)
		// Inject boolean default false.
		if dt.Kind == canonical.KindBoolean && !a.HasDefault {
			a.HasDefault = true
			a.DefaultValue = false
		}
		genAttr, err := buildGenAttrFromAST(a, dt)
		if err != nil {
			return nil, fmt.Errorf("attribute %q: %w", a.Name, err)
		}
		id := newEntityElementID()
		genAttr.SetID(id)
		attrIDs[a.Name] = id
		e.AddAttributes(genAttr)

		// ValidationRules from NotNull / Unique flags.
		if a.NotNull {
			vr := genDm.NewValidationRule()
			vr.SetAttributeQualifiedName(entityQN + "." + a.Name)
			vr.SetRuleInfo(genDm.NewRequiredRuleInfo())
			if a.NotNullError != "" {
				vr.SetErrorMessage(buildENUSText(a.NotNullError))
			}
			e.AddValidationRules(vr)
		}
		if a.Unique {
			vr := genDm.NewValidationRule()
			vr.SetAttributeQualifiedName(entityQN + "." + a.Name)
			vr.SetRuleInfo(genDm.NewUniqueRuleInfo())
			if a.UniqueError != "" {
				vr.SetErrorMessage(buildENUSText(a.UniqueError))
			}
			e.AddValidationRules(vr)
		}
	}

	// Indexes.
	for _, idx := range s.Indexes {
		gidx := genDm.NewIndex()
		for _, col := range idx.Columns {
			ia := genDm.NewIndexedAttribute()
			if id, ok := attrIDs[col.Name]; ok {
				ia.SetAttributeID(id)
			}
			ia.SetAscending(!col.Descending)
			gidx.AddAttributes(ia)
		}
		e.AddIndexes(gidx)
	}

	// Event handlers.
	for _, eh := range s.EventHandlers {
		h := genDm.NewEventHandler()
		h.SetMoment(eh.Moment)
		h.SetEvent(eh.Event)
		h.SetMicroflowQualifiedName(eh.Microflow.String())
		h.SetRaiseErrorOnFalse(eh.RaiseErrorOnFalse)
		h.SetPassEventObject(eh.PassEventObject)
		e.AddEventHandlers(h)
	}

	return e, nil
}

func buildGenAttrFromAST(a ast.Attribute, dt canonical.DataType) (*genDm.Attribute, error) {
	attr := genDm.NewAttribute()
	attr.SetName(a.Name)
	if a.Documentation != "" {
		attr.SetDocumentation(a.Documentation)
	}
	at, err := canonicalDataTypeToGenAttrType(dt)
	if err != nil {
		return nil, err
	}
	attr.SetType(at)
	if a.Calculated && a.CalculatedMicroflow != nil {
		cv := genDm.NewCalculatedValue()
		cv.SetMicroflowQualifiedName(a.CalculatedMicroflow.String())
		attr.SetValue(cv)
	} else if a.HasDefault || dt.Kind == canonical.KindAutoNumber {
		sv := genDm.NewStoredValue()
		raw := stripAttrDefaultQuotes(fmt.Sprintf("%v", a.DefaultValue))
		if dt.Kind == canonical.KindAutoNumber && raw == "" {
			raw = "1"
		}
		if dt.Kind == canonical.KindUnresolvedRef || dt.Kind == canonical.KindEnumRef {
			if i := strings.LastIndex(raw, "."); i >= 0 {
				raw = raw[i+1:]
			}
		}
		sv.SetDefaultValue(raw)
		attr.SetValue(sv)
	}
	return attr, nil
}

// astDataTypeToCanonical maps an ast.DataType to a canonical.DataType.
func astDataTypeToCanonical(dt ast.DataType) canonical.DataType {
	switch dt.Kind {
	case ast.TypeString:
		return canonical.DataType{Kind: canonical.KindString, Length: dt.Length}
	case ast.TypeInteger:
		return canonical.DataType{Kind: canonical.KindInteger}
	case ast.TypeLong:
		return canonical.DataType{Kind: canonical.KindLong}
	case ast.TypeDecimal:
		return canonical.DataType{Kind: canonical.KindDecimal, Precision: dt.Precision, Scale: dt.Scale}
	case ast.TypeBoolean:
		return canonical.DataType{Kind: canonical.KindBoolean}
	case ast.TypeDateTime, ast.TypeDate:
		return canonical.DataType{Kind: canonical.KindDateTime}
	case ast.TypeBinary:
		return canonical.DataType{Kind: canonical.KindBinary}
	case ast.TypeAutoNumber:
		return canonical.DataType{Kind: canonical.KindAutoNumber}
	case ast.TypeEnumeration:
		ref := ""
		if dt.EnumRef != nil {
			ref = dt.EnumRef.String()
		}
		return canonical.DataType{Kind: canonical.KindUnresolvedRef, Ref: ref}
	case ast.TypeEntity:
		ref := ""
		if dt.EntityRef != nil {
			ref = dt.EntityRef.String()
		}
		return canonical.DataType{Kind: canonical.KindEntityRef, Ref: ref}
	case ast.TypeListOf:
		ref := ""
		if dt.EntityRef != nil {
			ref = dt.EntityRef.String()
		}
		return canonical.DataType{Kind: canonical.KindListOf, Ref: ref}
	default:
		return canonical.DataType{Kind: canonical.KindUnknown}
	}
}

// canonicalDataTypeToGenAttrType maps a canonical.DataType to its gen attribute-type constructor.
func canonicalDataTypeToGenAttrType(dt canonical.DataType) (element.Element, error) {
	switch dt.Kind {
	case canonical.KindString:
		st := genDm.NewStringAttributeType()
		if dt.Length > 0 {
			if dt.Length > math.MaxInt32 {
				return nil, fmt.Errorf("String length %d exceeds int32 max", dt.Length)
			}
			st.SetLength(int32(dt.Length))
		} else {
			st.SetLength(0)
		}
		return st, nil
	case canonical.KindInteger:
		return genDm.NewIntegerAttributeType(), nil
	case canonical.KindLong:
		return genDm.NewLongAttributeType(), nil
	case canonical.KindDecimal:
		return genDm.NewDecimalAttributeType(), nil
	case canonical.KindBoolean:
		return genDm.NewBooleanAttributeType(), nil
	case canonical.KindDateTime:
		return genDm.NewDateTimeAttributeType(), nil
	case canonical.KindBinary:
		return genDm.NewBinaryAttributeType(), nil
	case canonical.KindAutoNumber:
		return genDm.NewAutoNumberAttributeType(), nil
	case canonical.KindEnumRef, canonical.KindUnresolvedRef:
		ea := genDm.NewEnumerationAttributeType()
		ea.SetEnumerationQualifiedName(dt.Ref)
		return ea, nil
	case canonical.KindEntityRef, canonical.KindListOf:
		return nil, fmt.Errorf("entity/list-of types not allowed as entity attributes")
	default:
		return nil, fmt.Errorf("unknown DataTypeKind %d", int(dt.Kind))
	}
}

// applySystemMembersFromSlice wires system member presence bits on a NoGeneralization.
func applySystemMembersFromSlice(ng *genDm.NoGeneralization, members []string) {
	enabled := make(map[string]bool, len(members))
	for _, name := range members {
		enabled[strings.ToLower(strings.TrimSpace(name))] = true
	}
	ng.SetHasOwner(enabled["owner"])
	ng.SetHasChangedBy(enabled["changedby"])
	ng.SetHasCreatedDate(enabled["createddate"])
	ng.SetHasChangedDate(enabled["changeddate"])
}

// buildENUSText creates a single en_US text element.
func buildENUSText(msg string) *genTexts.Text {
	t := genTexts.NewText()
	tr := genTexts.NewTranslation()
	tr.SetLanguageCode("en_US")
	tr.SetText(msg)
	t.AddTranslations(tr)
	return t
}

// newEntityElementID returns a random RFC 4122 v4 UUID as an element.ID.
func newEntityElementID() element.ID {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return ""
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return element.ID(fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]))
}

// stripAttrDefaultQuotes removes surrounding single quotes from string default
// values and un-doubles internal escapes.
func stripAttrDefaultQuotes(v string) string {
	if len(v) >= 2 && v[0] == '\'' && v[len(v)-1] == '\'' {
		inner := v[1 : len(v)-1]
		return strings.ReplaceAll(inner, "''", "'")
	}
	return v
}
```

- [ ] **Step 4: 确认测试通过**

```bash
go test ./mdl/executor/... -run "TestBuildEntityFromAST|TestASTDataType" -v 2>&1 | tail -10
```

预期: 所有 `TestBuildEntityFromAST_*` 和 `TestASTDataTypeToCanonical_*` — PASS

- [ ] **Step 5: 编译确认**

```bash
go build ./mdl/executor/ 2>&1
```

预期: 无输出

- [ ] **Step 6: 提交**

```bash
git add mdl/executor/entity_from_ast.go mdl/executor/entity_from_ast_test.go
git commit -m "refactor(canonical): add buildEntityFromAST helper to replace canonical entity/lift+persist"
```

---

### Task 4: 迁移 `cmd_diff_mdl.go` — 四处 canonical 调用替换

**Files:**
- Modify: `mdl/executor/cmd_diff_mdl.go`

- [ ] **Step 1: 替换 `entityStmtToMDL`（AST → MDL，行 23-29）**

当前:
```go
func entityStmtToMDL(ctx *ExecContext, s *ast.CreateEntityStmt) string {
	doc, err := ctx.ModelCodecs.LiftFrom(s)
	if err != nil {
		return fmt.Sprintf("/* entity lift error: %v */", err)
	}
	return doc.ToMDL() + ";\n/"
}
```

替换为（注意：保留 `ctx *ExecContext` 参数以免改动所有调用点，但不再使用 ModelCodecs）:
```go
func entityStmtToMDL(_ *ExecContext, s *ast.CreateEntityStmt) string {
	spec := entitySpecFromAST(s)
	return renderEntityMDL(spec, false) + ";\n/"
}

// entitySpecFromAST builds an entityMDLSpec from an AST CreateEntityStmt.
func entitySpecFromAST(s *ast.CreateEntityStmt) entityMDLSpec {
	spec := entityMDLSpec{
		module:        s.Name.Module,
		name:          s.Name.Name,
		kind:          entityKindStr(s.Kind),
		documentation: s.Documentation,
	}
	if s.Position != nil {
		spec.hasPosition = true
		spec.positionX = s.Position.X
		spec.positionY = s.Position.Y
	}
	if s.Generalization != nil {
		spec.extendsQN = s.Generalization.String()
	}
	for _, a := range s.Attributes {
		dt := astDataTypeToCanonical(a.Type)
		// Do NOT inject boolean default false here — that belongs only in buildEntityFromAST
		// (write path). The diff proposed-side should reflect exactly what the user wrote.
		defaultVal := ""
		hasDefault := a.HasDefault
		if a.HasDefault {
			if a.DefaultValue == nil {
				defaultVal = ""
			} else if sv, ok := a.DefaultValue.(string); ok {
				defaultVal = sv // string literals already include quotes from the parser
			} else {
				defaultVal = fmt.Sprintf("%v", a.DefaultValue)
			}
		}
		var calcMFQN string
		if a.CalculatedMicroflow != nil {
			calcMFQN = a.CalculatedMicroflow.String()
		}
		spec.attributes = append(spec.attributes, attrMDLSpec{
			name:           a.Name,
			documentation:  a.Documentation,
			dataType:       dt,
			notNull:        a.NotNull,
			notNullError:   a.NotNullError,
			unique:         a.Unique,
			uniqueError:    a.UniqueError,
			hasDefault:     hasDefault,
			defaultValue:   defaultVal,
			calculated:     a.Calculated,
			calculatedMFQN: calcMFQN,
		})
	}
	for _, idx := range s.Indexes {
		im := indexMDLSpec{}
		for _, col := range idx.Columns {
			im.columns = append(im.columns, indexColumnMDLSpec{name: col.Name, ascending: !col.Descending})
		}
		spec.indexes = append(spec.indexes, im)
	}
	for _, eh := range s.EventHandlers {
		spec.eventHandlers = append(spec.eventHandlers, eventHandlerMDLSpec{
			moment:            eh.Moment,
			event:             eh.Event,
			microflowQN:       eh.Microflow.String(),
			raiseErrorOnFalse: eh.RaiseErrorOnFalse,
			passEventObject:   eh.PassEventObject,
		})
	}
	spec.systemMembers = append([]string(nil), s.SystemMembers...)
	return spec
}

func entityKindStr(k ast.EntityKind) string {
	switch k {
	case ast.EntityNonPersistent:
		return "non-persistent"
	case ast.EntityView:
		return "view"
	case ast.EntityExternal:
		return "external"
	default:
		return "persistent"
	}
}
```

- [ ] **Step 2: 替换 `associationStmtToMDL`（行 94-100）**

当前:
```go
func associationStmtToMDL(ctx *ExecContext, s *ast.CreateAssociationStmt) string {
	doc, err := ctx.ModelCodecs.LiftFrom(s)
	if err != nil {
		return fmt.Sprintf("/* association lift error: %v */", err)
	}
	return doc.ToMDL() + ";\n/"
}
```

替换为:
```go
func associationStmtToMDL(_ *ExecContext, s *ast.CreateAssociationStmt) string {
	spec := assocSpecFromAST(s)
	return renderAssocMDL(spec) + ";\n/"
}

func assocSpecFromAST(s *ast.CreateAssociationStmt) assocMDLSpec {
	return assocMDLSpec{
		module:         s.Name.Module,
		name:           s.Name.Name,
		fromQN:         s.Parent.String(),
		toQN:           s.Child.String(),
		documentation:  s.Documentation,
		assocType:      astAssocTypeStr(s.Type),
		owner:          astOwnerStr(s.Owner),
		deleteBehavior: astDeleteBehaviorStr(s.DeleteBehavior),
	}
}

func astAssocTypeStr(t ast.AssociationType) string {
	if t == ast.AssocReferenceSet {
		return "ReferenceSet"
	}
	return "Reference"
}

func astOwnerStr(o ast.OwnerType) string {
	if o == ast.OwnerBoth {
		return "Both"
	}
	return "Default"
}

func astDeleteBehaviorStr(d ast.DeleteBehavior) string {
	switch d {
	case ast.DeleteCascade:
		return "DeleteMeAndReferences"
	case ast.DeleteBoth:
		return "DeleteBoth"
	case ast.DeleteKeepParentDeleteChild:
		return "KeepParentDeleteChild"
	case ast.DeleteKeepChildDeleteParent:
		return "KeepChildDeleteParent"
	case ast.DeleteIfNoReferences:
		return "DeleteIfNoReferences"
	default:
		return "DeleteMeButKeepReferences"
	}
}
```

- [ ] **Step 3: 替换 `entityToMDLGen`（gen → MDL，行 439-449）**

当前:
```go
func entityToMDLGen(ctx *ExecContext, moduleName string, entity *genDm.Entity) string {
	doc, _, err := ctx.ModelCodecs.HydrateFrom(entity, canonical.HydrateCtx{ModuleName: moduleName})
	if err != nil {
		return fmt.Sprintf("/* entity hydrate error: %v */", err)
	}
	return doc.ToMDL() + ";\n/"
}
```

替换为:
```go
func entityToMDLGen(_ *ExecContext, moduleName string, entity *genDm.Entity) string {
	spec := entitySpecFromGen(moduleName, entity)
	return renderEntityMDL(spec, false) + ";\n/"
}

// entitySpecFromGen builds an entityMDLSpec from a gen *genDm.Entity (used in diff current-side).
func entitySpecFromGen(moduleName string, e *genDm.Entity) entityMDLSpec {
	spec := entityMDLSpec{
		module:        moduleName,
		name:          e.Name(),
		kind:          entityKindFromGen(e),
		documentation: e.Documentation(),
	}
	if loc := e.Location(); loc != "" {
		if x, y, ok := parseEntityLocation(loc); ok {
			spec.hasPosition = true
			spec.positionX = x
			spec.positionY = y
		}
	}
	if g, ok := e.Generalization().(*genDm.Generalization); ok {
		spec.extendsQN = g.GeneralizationQualifiedName()
	}

	// ValidationRules → notNull / unique maps (keyed by attr name).
	notNull := map[string]bool{}
	unique := map[string]bool{}
	notNullErr := map[string]string{}
	uniqueErr := map[string]string{}
	for _, item := range e.ValidationRulesItems() {
		vr, ok := item.(*genDm.ValidationRule)
		if !ok {
			continue
		}
		attrName := lastAttrSegment(vr.AttributeQualifiedName())
		if ri := vr.RuleInfo(); ri != nil {
			msg := genENUSText(vr.ErrorMessage())
			switch ri.TypeName() {
			case "DomainModels$RequiredRuleInfo":
				notNull[attrName] = true
				if msg != "" {
					notNullErr[attrName] = msg
				}
			case "DomainModels$UniqueRuleInfo":
				unique[attrName] = true
				if msg != "" {
					uniqueErr[attrName] = msg
				}
			}
		}
	}

	for _, item := range e.AttributesItems() {
		attr, ok := item.(*genDm.Attribute)
		if !ok {
			continue
		}
		am := attrSpecFromGen(attr, notNull, unique)
		am.notNullError = notNullErr[attr.Name()]
		am.uniqueError = uniqueErr[attr.Name()]
		spec.attributes = append(spec.attributes, am)
	}

	for _, item := range e.IndexesItems() {
		idx, ok := item.(*genDm.Index)
		if !ok {
			continue
		}
		attrNames := map[string]string{}
		for _, ai := range e.AttributesItems() {
			if a, ok := ai.(*genDm.Attribute); ok {
				attrNames[string(a.ID())] = a.Name()
			}
		}
		im := indexMDLSpec{name: idx.DataStorageGuid()}
		for _, ci := range idx.AttributesItems() {
			ia, ok := ci.(*genDm.IndexedAttribute)
			if !ok {
				continue
			}
			name := attrNames[string(ia.AttributeRefID())]
			if name == "" {
				name = string(ia.AttributeRefID())
			}
			im.columns = append(im.columns, indexColumnMDLSpec{name: name, ascending: ia.Ascending()})
		}
		spec.indexes = append(spec.indexes, im)
	}

	for _, item := range e.EventHandlersItems() {
		h, ok := item.(*genDm.EventHandler)
		if !ok {
			continue
		}
		if h.MicroflowQualifiedName() == "" {
			continue
		}
		spec.eventHandlers = append(spec.eventHandlers, eventHandlerMDLSpec{
			moment:            h.Moment(),
			event:             h.Event(),
			microflowQN:       h.MicroflowQualifiedName(),
			raiseErrorOnFalse: h.RaiseErrorOnFalse(),
			passEventObject:   h.PassEventObject(),
		})
	}

	if g, ok := e.Generalization().(*genDm.NoGeneralization); ok {
		if g.HasOwner() {
			spec.systemMembers = append(spec.systemMembers, "owner")
		}
		if g.HasCreatedDate() {
			spec.systemMembers = append(spec.systemMembers, "createdDate")
		}
		if g.HasChangedDate() {
			spec.systemMembers = append(spec.systemMembers, "changedDate")
		}
		if g.HasChangedBy() {
			spec.systemMembers = append(spec.systemMembers, "changedBy")
		}
	}

	if spec.kind == "view" {
		if src := e.Source(); src != nil {
			type oqlSource interface{ Oql() string }
			if oq, ok := src.(oqlSource); ok {
				spec.oql = oq.Oql()
			}
		}
	}
	return spec
}

func entityKindFromGen(e *genDm.Entity) string {
	if src := e.Source(); src != nil && strings.Contains(src.TypeName(), "OqlView") {
		return "view"
	}
	if g, ok := e.Generalization().(*genDm.NoGeneralization); ok && !g.Persistable() {
		return "non-persistent"
	}
	return "persistent"
}

func attrSpecFromGen(attr *genDm.Attribute, notNull, unique map[string]bool) attrMDLSpec {
	am := attrMDLSpec{
		name:          attr.Name(),
		documentation: attr.Documentation(),
		dataType:      genAttrTypeToCanonical(attr.Type()),
		notNull:       notNull[attr.Name()],
		unique:        unique[attr.Name()],
	}
	if cv, ok := attr.Value().(*genDm.CalculatedValue); ok {
		am.calculated = true
		am.calculatedMFQN = cv.MicroflowQualifiedName()
	} else if sv, ok := attr.Value().(*genDm.StoredValue); ok && sv.DefaultValue() != "" {
		am.hasDefault = true
		raw := sv.DefaultValue()
		if _, isStr := attr.Type().(*genDm.StringAttributeType); isStr {
			am.defaultValue = "'" + strings.ReplaceAll(raw, "'", "''") + "'"
		} else {
			am.defaultValue = raw
		}
	}
	return am
}

func genAttrTypeToCanonical(t any) canonical.DataType {
	switch v := t.(type) {
	case *genDm.StringAttributeType:
		return canonical.DataType{Kind: canonical.KindString, Length: int(v.Length())}
	case *genDm.IntegerAttributeType:
		return canonical.DataType{Kind: canonical.KindInteger}
	case *genDm.LongAttributeType:
		return canonical.DataType{Kind: canonical.KindLong}
	case *genDm.DecimalAttributeType:
		return canonical.DataType{Kind: canonical.KindDecimal}
	case *genDm.BooleanAttributeType:
		return canonical.DataType{Kind: canonical.KindBoolean}
	case *genDm.DateTimeAttributeType:
		return canonical.DataType{Kind: canonical.KindDateTime}
	case *genDm.BinaryAttributeType:
		return canonical.DataType{Kind: canonical.KindBinary}
	case *genDm.AutoNumberAttributeType:
		return canonical.DataType{Kind: canonical.KindAutoNumber}
	case *genDm.EnumerationAttributeType:
		return canonical.DataType{Kind: canonical.KindEnumRef, Ref: v.EnumerationQualifiedName()}
	default:
		return canonical.DataType{Kind: canonical.KindUnknown}
	}
}

// genENUSText extracts en_US translation from a gen text element.
func genENUSText(msg element.Element) string {
	t, ok := msg.(*genTexts.Text)
	if !ok || t == nil {
		return ""
	}
	var first string
	for _, item := range t.TranslationsItems() {
		tr, ok := item.(*genTexts.Translation)
		if !ok {
			continue
		}
		if first == "" {
			first = tr.Text()
		}
		if tr.LanguageCode() == "en_US" {
			return tr.Text()
		}
	}
	return first
}

// lastAttrSegment returns the last dot-separated segment of a qualified name.
func lastAttrSegment(qn string) string {
	parts := strings.Split(qn, ".")
	if len(parts) == 0 {
		return qn
	}
	return parts[len(parts)-1]
}
```

- [ ] **Step 4: 替换 `associationToMDLGen`（行 522-542）**

当前:
```go
func associationToMDLGen(ctx *ExecContext, moduleName string, assoc *genDm.Association, dm *genDm.DomainModel) string {
	entityNames := make(map[string]string)
	for _, item := range dm.EntitiesItems() {
		if e, ok := item.(*genDm.Entity); ok {
			entityNames[string(e.ID())] = e.Name()
		}
	}
	doc, _, err := ctx.ModelCodecs.HydrateFrom(assoc, canonical.HydrateCtx{
		ModuleName:  moduleName,
		EntityNames: entityNames,
	})
	if err != nil {
		return fmt.Sprintf("/* association hydrate error: %v */", err)
	}
	return doc.ToMDL() + ";\n/"
}
```

替换为:
```go
func associationToMDLGen(_ *ExecContext, moduleName string, assoc *genDm.Association, dm *genDm.DomainModel) string {
	entityNames := make(map[string]string)
	for _, item := range dm.EntitiesItems() {
		if e, ok := item.(*genDm.Entity); ok {
			entityNames[string(e.ID())] = moduleName + "." + e.Name()
		}
	}
	spec := assocSpecFromGen(moduleName, assoc, entityNames)
	return renderAssocMDL(spec) + ";\n/"
}

func assocSpecFromGen(moduleName string, a *genDm.Association, entityNames map[string]string) assocMDLSpec {
	fromQN := entityNames[string(a.ParentID())]
	if fromQN == "" {
		fromQN = string(a.ParentID())
	}
	toQN := entityNames[string(a.ChildID())]
	if toQN == "" {
		toQN = string(a.ChildID())
	}
	db := ""
	if d, ok := a.DeleteBehavior().(*genDm.AssociationDeleteBehavior); ok {
		db = genAssocDeleteBehaviorToMDL(d.ChildDeleteBehavior())
	}
	owner := "Default"
	if a.Owner() == "Both" {
		owner = "Both"
	}
	assocType := "Reference"
	if a.Type() == "ReferenceSet" {
		assocType = "ReferenceSet"
	}
	return assocMDLSpec{
		module:         moduleName,
		name:           a.Name(),
		fromQN:         fromQN,
		toQN:           toQN,
		documentation:  a.Documentation(),
		assocType:      assocType,
		owner:          owner,
		deleteBehavior: db,
	}
}
```

- [ ] **Step 5: 删除不再需要的 import（canonical, entitymodel 等）**

在 `cmd_diff_mdl.go` 的 import 块中移除：
```go
"github.com/mendixlabs/mxcli/mdl/canonical"
```

并在 `entity_mdl_render.go` / `assoc_mdl_render.go` 中确认已 import `canonical`。同时需添加缺失的 import（`genTexts`、`element`、`strings` 等）到 `cmd_diff_mdl.go` 中。

- [ ] **Step 6: 编译确认**

```bash
go build ./mdl/executor/ 2>&1
```

预期: 无输出

- [ ] **Step 7: 提交**

```bash
git add mdl/executor/cmd_diff_mdl.go
git commit -m "refactor(canonical): migrate cmd_diff_mdl to use entityMDLSpec/assocMDLSpec without canonical registry"
```

---

### Task 5: 迁移 `cmd_entities_gen.go` — DESCRIBE ENTITY

**Files:**
- Modify: `mdl/executor/cmd_entities_gen.go`

- [ ] **Step 1: 替换 `describeEntityGen` 中的 `hydrateEntityModel` 调用（行 373-388）**

当前:
```go
m, warns, err := hydrateEntityModel(ctx, modName, entity)
if err != nil {
	return fmt.Errorf("describe entity: hydrate: %w", err)
}
for _, w := range warns {
	if ctx.Logger != nil {
		ctx.Logger.Warn("describe entity hydrate", "entity", name.String(), "field", w.Field, "msg", w.Message)
	}
}
fmt.Fprint(ctx.Output, m.ToMDLStatement(true))
fmt.Fprintln(ctx.Output, ";")
```

替换为:
```go
spec := entitySpecFromGen(modName, entity)
fmt.Fprint(ctx.Output, renderEntityMDL(spec, true))
fmt.Fprintln(ctx.Output, ";")
```

- [ ] **Step 2: 移除不再需要的 import**

从 `cmd_entities_gen.go` 的 import 块中移除所有引用 `canonical` 或 `entitymodel` 的行（如果存在）。

- [ ] **Step 3: 编译确认**

```bash
go build ./mdl/executor/ 2>&1
```

预期: 无输出

- [ ] **Step 4: 提交**

```bash
git add mdl/executor/cmd_entities_gen.go
git commit -m "refactor(canonical): migrate DESCRIBE ENTITY to entitySpecFromGen/renderEntityMDL"
```

---

### Task 6: 迁移 `cmd_create_entity_gen.go` — 写路径

**Files:**
- Modify: `mdl/executor/cmd_create_entity_gen.go`

- [ ] **Step 1: 将 `persistEntityCanonical` 重写为 `persistEntityDirect`**

找到 `persistEntityCanonical`（行 145 附近），将函数体中的 canonical 调用替换。

当前核心（行 206-224）：
```go
doc, err := ctx.ModelCodecs.LiftFrom(&stmt)
if err != nil {
	return mdlerrors.NewBackend("CREATE ENTITY: lift", err)
}

var existingID model.ID
if existing != nil {
	if elem, ok := existing.(interface{ ID() element.ID }); ok {
		existingID = model.ID(elem.ID())
	}
}

pCtx := canonical.PersistContext{
	DomainModelID:    model.ID(dm.ID()),
	ExistingEntityID: existingID,
	Backend:          ctx.Backend,
}
if err := doc.Persist(pCtx); err != nil {
	return mdlerrors.NewBackend("CREATE ENTITY: persist", err)
}
```

替换为:
```go
gen, err := buildEntityFromAST(stmt.Name.Module, &stmt)
if err != nil {
	return mdlerrors.NewBackend("CREATE ENTITY: build gen entity", err)
}

if existing != nil {
	if elem, ok := existing.(interface{ ID() element.ID }); ok {
		gen.SetID(elem.ID())
	}
	if err := ctx.Backend.UpdateEntityGen(model.ID(dm.ID()), gen); err != nil {
		return mdlerrors.NewBackend("CREATE ENTITY: update", err)
	}
} else {
	if err := ctx.Backend.CreateEntityGen(model.ID(dm.ID()), gen); err != nil {
		return mdlerrors.NewBackend("CREATE ENTITY: create", err)
	}
}
```

并将函数名从 `persistEntityCanonical` 改为 `persistEntityDirect`，更新调用处（行 80）。

- [ ] **Step 2: 删除 `if ctx.ModelCodecs != nil` 守卫和 legacy path（行 79-128）**

当前:
```go
if ctx.ModelCodecs != nil {
	if err := persistEntityCanonical(ctx, s, dm, existingEntity, module); err != nil {
		return err
	}
	return nil
}

// Legacy path (absent codec registry — should not occur in production).
entity := astToEntityGen(s)
...
```

替换为（直接调用，去掉守卫和 legacy path）:
```go
if err := persistEntityDirect(ctx, s, dm, existingEntity, module); err != nil {
	return err
}
return nil
```

- [ ] **Step 3: 移除 canonical import**

从 `cmd_create_entity_gen.go` 的 import 块中移除：
```go
"github.com/mendixlabs/mxcli/mdl/canonical"
```

- [ ] **Step 4: 编译确认**

```bash
go build ./mdl/executor/ 2>&1
```

预期: 无输出

- [ ] **Step 5: 提交**

```bash
git add mdl/executor/cmd_create_entity_gen.go
git commit -m "refactor(canonical): migrate CREATE ENTITY to persistEntityDirect (no canonical registry)"
```

---

### Task 7: 清理 `executor.go`、`exec_context.go`、`executor_dispatch.go`

**Files:**
- Modify: `mdl/executor/executor.go`
- Modify: `mdl/executor/exec_context.go`
- Modify: `mdl/executor/executor_dispatch.go`

- [ ] **Step 1: 从 `executor.go` 删除 canonical 相关内容**

删除以下内容（行 284-320 附近）：
- `modelCodecs *canonical.DefaultRegistry` 字段声明
- `New()` 中的 `mc := canonical.NewDefaultRegistry()`、`entitymodel.RegisterCodec(mc)`、`assocmodel.RegisterCodec(mc)`、`modelCodecs: mc`
- 整个 `hydrateEntityModel()` 函数（行 307-320）

删除 import 中：
```go
"github.com/mendixlabs/mxcli/mdl/canonical"
assocmodel "github.com/mendixlabs/mxcli/mdl/canonical/association"
entitymodel "github.com/mendixlabs/mxcli/mdl/canonical/entity"
```

- [ ] **Step 2: 从 `exec_context.go` 删除 `ModelCodecs` 字段（行 36-39）**

删除：
```go
// ModelCodecs is the canonical model codec registry for Lift/Hydrate
// operations. Populated by newExecContext from the owning Executor;
// nil for ad-hoc contexts that did not opt into the canonical pipeline.
ModelCodecs *canonical.DefaultRegistry
```

删除 import 中：
```go
"github.com/mendixlabs/mxcli/mdl/canonical"
```

- [ ] **Step 3: 从 `executor_dispatch.go` 删除 `ModelCodecs: e.modelCodecs` 行（行 73）**

从 `newExecContext` 中删除：
```go
ModelCodecs:       e.modelCodecs,
```

- [ ] **Step 4: 编译确认**

```bash
go build ./mdl/executor/ 2>&1
```

预期: 无输出

- [ ] **Step 5: 提交**

```bash
git add mdl/executor/executor.go mdl/executor/exec_context.go mdl/executor/executor_dispatch.go
git commit -m "refactor(canonical): remove ModelCodecs from ExecContext and Executor"
```

---

### Task 8: 删除 canonical 子包和基础设施

**Files:**
- Delete: `mdl/canonical/entity/` (整个目录)
- Delete: `mdl/canonical/association/` (整个目录)
- Delete: `mdl/canonical/registry.go`
- Delete: `mdl/canonical/context.go`
- Modify: `mdl/canonical/doc.go`

- [ ] **Step 1: 删除 canonical 子包**

```bash
rm -rf mdl/canonical/entity
rm -rf mdl/canonical/association
rm mdl/canonical/registry.go
rm mdl/canonical/context.go
```

- [ ] **Step 2: 更新 `mdl/canonical/doc.go`**

将 doc.go 内容替换为：
```go
// SPDX-License-Identifier: Apache-2.0

// Package canonical provides shared data types used across the MDL pipeline.
//
// This package retains only the DataType / DataTypeKind shared types that are
// used by executor commands for attribute and parameter type representation.
// The Lift / Hydrate / Persist / Codec lifecycle infrastructure that was
// previously in the entity/ and association/ sub-packages has been removed —
// those domains now go through the standard executor → backend direct path.
package canonical

// Document is kept for backward-compatible imports in test files only.
// Deprecated: do not use in new code. Executor commands use entityMDLSpec / assocMDLSpec instead.
type Document interface {
	ToMDL() string
}

// Persistable is kept for backward-compatible imports only.
// Deprecated: do not use in new code.
type Persistable interface {
	Document
	Persist(ctx PersistContext) error
}

// Warning is a non-fatal issue that was surfaced during Hydrate.
// Kept for callers that still use canonical.Warning in their API.
type Warning struct {
	Field   string
	Message string
}
```

Note: Actually, `Document`, `Persistable`, and `Warning` can be fully removed if no callers remain. Let's check:

```bash
grep -rn "canonical\.Document\|canonical\.Persistable\|canonical\.Warning" /mnt/data_sdd/gh/mxcli-wt-02/ --include="*.go" | grep -v "_test.go"
```

If zero results, remove these types too and simplify doc.go:
```go
// SPDX-License-Identifier: Apache-2.0

// Package canonical provides the DataType / DataTypeKind shared types used
// across the MDL executor pipeline for attribute and parameter type representation.
// The Lift/Hydrate/Persist/Codec lifecycle infrastructure (previously in entity/
// and association/ sub-packages) has been removed — those domains use the direct
// executor → backend path instead.
package canonical
```

- [ ] **Step 3: 编译确认**

```bash
go build ./... 2>&1
```

预期: 无输出（所有包编译通过）

- [ ] **Step 4: 提交**

```bash
git add -A mdl/canonical/
git commit -m "refactor(canonical): delete entity/ and association/ lifecycle sub-packages and registry"
```

---

### Task 9: 添加 import guard 防止回退

**Files:**
- Create: `mdl/canonical/import_guard_test.go`

- [ ] **Step 1: 写 guard 测试**

```go
// mdl/canonical/import_guard_test.go
package canonical_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoCanonicalLifecycleImportInExecutor verifies that the executor package
// does not import the canonical lifecycle sub-packages (entity, association,
// registry) that were removed in the canonical layer refactor.
func TestNoCanonicalLifecycleImportInExecutor(t *testing.T) {
	forbidden := []string{
		"github.com/mendixlabs/mxcli/mdl/canonical/entity",
		"github.com/mendixlabs/mxcli/mdl/canonical/association",
	}

	executorDir := filepath.Join("..", "executor")
	entries, err := os.ReadDir(executorDir)
	if err != nil {
		t.Fatalf("read executor dir: %v", err)
	}

	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || strings.HasSuffix(name, "_test.go") || !strings.HasSuffix(name, ".go") {
			continue
		}
		fullPath := filepath.Join(executorDir, name)
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, fullPath, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, imp := range f.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			for _, bad := range forbidden {
				if path == bad || strings.HasPrefix(path, bad+"/") {
					t.Errorf("%s: forbidden import %q (canonical lifecycle packages were removed)", name, path)
				}
			}
		}
	}
}
```

- [ ] **Step 2: 确认 guard 通过**

```bash
go test ./mdl/canonical/... -run "TestNoCanonicalLifecycleImport" -v 2>&1
```

预期: PASS

- [ ] **Step 3: 提交**

```bash
git add mdl/canonical/import_guard_test.go
git commit -m "test(canonical): add import guard to prevent canonical lifecycle re-import in executor"
```

---

### Task 10: 全量验证

- [ ] **Step 1: 全量编译**

```bash
go build ./... 2>&1
```

预期: 零输出（无 error）

- [ ] **Step 2: 全量测试**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./mdl/executor/... ./mdl/canonical/... -timeout 120s 2>&1 | tail -30
```

预期: `ok github.com/mendixlabs/mxcli/mdl/executor` 和 `ok github.com/mendixlabs/mxcli/mdl/canonical`，无 FAIL

- [ ] **Step 3: 架构守卫测试**

```bash
go test ./mdl/executor/... -run "TestNoDirectBSONImport|TestNoRawBSONType|TestNoCanonicalLifecycle" -v 2>&1
```

预期: 全部 PASS

- [ ] **Step 4: 全库测试（回归检查）**

```bash
go test ./... -timeout 180s 2>&1 | grep -E "^(ok|FAIL|---)" | tail -40
```

预期: 无 FAIL 行

- [ ] **Step 5: 最终提交（如有残余改动）**

```bash
git status
# 若有未提交改动
git add -p
git commit -m "refactor(canonical): complete canonical lifecycle layer removal"
```
