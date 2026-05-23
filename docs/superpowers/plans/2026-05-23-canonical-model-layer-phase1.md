# Canonical Model Layer — Phase 1 Implementation Plan (Framework + Entity)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Introduce `mdl/model/` as the single serialization center for the entity document type, eliminating the dual-path round-trip drift between `entityStmtToMDL` (AST→text) and `entityToMDLGen` (gen→text) in `cmd_diff_mdl.go`.

**Architecture:** A new `mdl/model/` package defines three small interfaces (`Document`, `Persistable`, `Warning`) and a `ModelRegistry` that dispatches Lift/Hydrate operations. A sub-package `mdl/model/entity/` implements all four operations (Lift, Hydrate, ToMDL, Persist) for entity documents. The executor's `ExecContext` gains a `ModelCodecs` field; both the CREATE and DESCRIBE handlers delegate to it instead of duplicating serialization logic.

**Tech Stack:** Go 1.26, `github.com/mendixlabs/mxcli` module, `modelsdk/gen/domainmodels` for gen types, `mdl/ast` for AST types, `mdl/backend` for the persistence interface.

**Spec:** `docs/superpowers/specs/2026-05-23-canonical-model-layer-design.md`

**Scope:** Phase 1 covers framework + entity only. Phase 2 (association), Phase 3 (microflow), Phase 4 (page) each follow the same task pattern (Tasks 2–9) and will have separate plans.

---

## File Map

**New files:**

| File | Responsibility |
|------|---------------|
| `mdl/model/doc.go` | `Document`, `Persistable` interfaces; `Warning` type |
| `mdl/model/datatype.go` | `DataType`, `DataTypeKind` (model-specific, no ast import) |
| `mdl/model/registry.go` | `ModelRegistry`, `Codec`, `DefaultRegistry` |
| `mdl/model/context.go` | `PersistContext` value type |
| `mdl/model/entity/model.go` | `EntityModel`, `AttributeModel`, `IndexModel`, `QualifiedName` |
| `mdl/model/entity/lift.go` | `Lift(*ast.CreateEntityStmt) → *EntityModel` |
| `mdl/model/entity/hydrate.go` | `Hydrate(*genDm.Entity) → (*EntityModel, []model.Warning)` |
| `mdl/model/entity/serialize.go` | `(*EntityModel).ToMDL() string` |
| `mdl/model/entity/persist.go` | `(*EntityModel).Persist(model.PersistContext) error` |
| `mdl/model/entity/codec.go` | `RegisterCodec(*model.DefaultRegistry)` |
| `mdl/model/entity/entity_test.go` | Unit tests: Lift, ToMDL, Hydrate |
| `mdl/model/entity/persist_test.go` | Integration test: Persist (build tag: integration) |

**Modified files:**

| File | Change |
|------|--------|
| `mdl/executor/exec_context.go` | Add `ModelCodecs *model.DefaultRegistry` field |
| `mdl/executor/executor_dispatch.go` | Populate `ModelCodecs` in `newExecContext` |
| `mdl/executor/executor.go` | Construct registry, call `entity.RegisterCodec` in `NewExecutor` |
| `mdl/executor/cmd_create_entity_gen.go` | Delegate to `Lift + Persist` via `ctx.ModelCodecs` |
| `mdl/executor/cmd_entities_gen.go` | Delegate `describeEntityGen` to `Hydrate + ToMDL` |
| `mdl/executor/cmd_diff_mdl.go` | Replace `entityStmtToMDL`/`entityToMDLGen` with `ToMDL()` |

---

## Task 1: Framework — interfaces, DataType, registry, context

**Files:**
- Create: `mdl/model/doc.go`
- Create: `mdl/model/datatype.go`
- Create: `mdl/model/registry.go`
- Create: `mdl/model/context.go`

- [ ] **Step 1: Create `mdl/model/doc.go`**

```go
// SPDX-License-Identifier: Apache-2.0

// Package model defines the canonical model layer — the single semantic
// center that both the MDL write path (parse→BSON) and read path
// (BSON→describe) must pass through. See docs/superpowers/specs/
// 2026-05-23-canonical-model-layer-design.md for rationale.
package model

// Document is the minimum contract: any canonical model can serialize
// to valid, executable MDL text. All document types implement this.
type Document interface {
	ToMDL() string
}

// Persistable extends Document for types that can be written to an MPR.
// Not all documents are writable (e.g. read-only system module entries).
type Persistable interface {
	Document
	Persist(ctx PersistContext) error
}

// Warning is a non-fatal issue encountered during Hydrate.
// Unknown BSON fields, missing optional values, etc. are returned as
// warnings so callers can log them without aborting the operation.
type Warning struct {
	Field   string
	Message string
}
```

- [ ] **Step 2: Create `mdl/model/datatype.go`**

DataType mirrors `ast.DataType` but is independent of the ast package to avoid import cycles. `Lift()` converts between the two.

```go
// SPDX-License-Identifier: Apache-2.0

package model

// DataTypeKind identifies the kind of an MDL data type.
type DataTypeKind int

const (
	KindUnknown      DataTypeKind = iota
	KindString                    // String or String(n)
	KindInteger
	KindLong
	KindDecimal                   // Decimal(precision, scale)
	KindBoolean
	KindDateTime
	KindBinary
	KindAutoNumber
	KindEnumRef                   // Enumeration(Module.EnumName) — known enum
	KindEntityRef                 // Entity reference — known entity
	KindListOf                    // List of Entity
	KindUnresolvedRef             // Module.Name — could be entity or enum (from Lift)
)

// DataType is the canonical representation of an MDL attribute/parameter type.
// It is independent of ast.DataType and genDm attribute types.
type DataType struct {
	Kind      DataTypeKind
	Length    int    // KindString: character limit, -1 = unlimited
	Precision int    // KindDecimal
	Scale     int    // KindDecimal
	Ref       string // KindEnumRef / KindEntityRef / KindListOf / KindUnresolvedRef: "Module.Name"
}
```

- [ ] **Step 3: Create `mdl/model/registry.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"fmt"
	"reflect"
)

// Codec pairs the two read-side operations for a document type.
// LiftFn converts an AST statement (passed as any) to a Persistable.
// HydrateFn converts a gen element (passed as any) to a Document.
type Codec struct {
	LiftFn    func(stmt any) (Persistable, error)
	HydrateFn func(el any) (Document, []Warning, error)
}

// DefaultRegistry maps AST statement types and gen type names to Codecs.
// It satisfies the Open/Closed principle: new document types register
// themselves via RegisterCodec; the framework never lists them by name.
type DefaultRegistry struct {
	byStmtType map[reflect.Type]Codec
	byGenType  map[string]Codec // key: gen element TypeName()
}

// NewDefaultRegistry returns an empty registry. Populate via RegisterCodec
// functions from each document sub-package (e.g. entity.RegisterCodec).
func NewDefaultRegistry() *DefaultRegistry {
	return &DefaultRegistry{
		byStmtType: make(map[reflect.Type]Codec),
		byGenType:  make(map[string]Codec),
	}
}

// Register associates a Codec with an AST statement type (for Lift) and
// a gen TypeName string (for Hydrate). Both keys are required.
func (r *DefaultRegistry) Register(stmtExample any, genTypeName string, c Codec) {
	r.byStmtType[reflect.TypeOf(stmtExample)] = c
	r.byGenType[genTypeName] = c
}

// LiftFrom converts an AST statement to a Persistable via the registered Codec.
func (r *DefaultRegistry) LiftFrom(stmt any) (Persistable, error) {
	c, ok := r.byStmtType[reflect.TypeOf(stmt)]
	if !ok {
		return nil, fmt.Errorf("model: no codec registered for %T", stmt)
	}
	return c.LiftFn(stmt)
}

// HydrateFrom converts a gen element to a Document via the registered Codec.
// The el argument must implement TypeName() string (all gen elements do).
type genTyper interface{ TypeName() string }

func (r *DefaultRegistry) HydrateFrom(el any) (Document, []Warning, error) {
	gt, ok := el.(genTyper)
	if !ok {
		return nil, nil, fmt.Errorf("model: element %T does not implement TypeName()", el)
	}
	c, ok := r.byGenType[gt.TypeName()]
	if !ok {
		return nil, nil, fmt.Errorf("model: no codec registered for gen type %q", gt.TypeName())
	}
	return c.HydrateFn(el)
}
```

- [ ] **Step 4: Create `mdl/model/context.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/model"
)

// PersistContext carries the dependencies that Persist() needs.
// The executor handler is responsible for resolving module names to IDs
// before constructing a PersistContext.
type PersistContext struct {
	// DomainModelID is the ID of the domain model document to write into.
	DomainModelID model.ID
	// ExistingEntityID is non-zero when updating an existing entity
	// (CREATE OR MODIFY when the entity already exists).
	ExistingEntityID model.ID
	// Backend provides the write operations.
	Backend backend.DomainModelBackend
}
```

- [ ] **Step 5: Verify compile**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02 && go build ./mdl/model/...
```

Expected: no errors.

- [ ] **Step 6: Commit**

```bash
git add mdl/model/doc.go mdl/model/datatype.go mdl/model/registry.go mdl/model/context.go
git commit -m "feat(model): add canonical model framework — Document, Persistable, DefaultRegistry, PersistContext"
```

---

## Task 2: EntityModel data struct

**Files:**
- Create: `mdl/model/entity/model.go`

- [ ] **Step 1: Create `mdl/model/entity/model.go`**

```go
// SPDX-License-Identifier: Apache-2.0

// Package entity implements the canonical model for MDL entity documents.
// It satisfies the mdl/model Document and Persistable interfaces.
package entity

import "github.com/mendixlabs/mxcli/mdl/model"

// QualifiedName is a module-qualified entity name.
type QualifiedName struct {
	Module string
	Name   string
}

func (q QualifiedName) String() string {
	if q.Module == "" {
		return q.Name
	}
	return q.Module + "." + q.Name
}

// EntityKind mirrors ast.EntityKind without importing ast.
type EntityKind int

const (
	EntityPersistent    EntityKind = iota
	EntityNonPersistent
	EntityView
	EntityExternal
)

// EntityModel is the canonical MDL-complete representation of an entity.
// It captures exactly what MDL can express — not the complete BSON state.
// Non-MDL fields (IDs, positions, version prefixes) live in persist.go.
type EntityModel struct {
	Name          QualifiedName
	Kind          EntityKind
	Documentation string
	Position      *Position // nil = no @Position annotation
	Extends       *QualifiedName
	Attributes    []AttributeModel
	Indexes       []IndexModel
	SystemMembers []string // "owner", "createdDate", "changedDate", "changedBy"
}

// Position holds canvas coordinates for @Position annotation.
type Position struct{ X, Y int }

// AttributeModel is the canonical representation of one entity attribute.
type AttributeModel struct {
	Name               string
	Type               model.DataType
	Documentation      string
	NotNull            bool
	NotNullError       string // empty = no custom message
	Unique             bool
	UniqueError        string
	HasDefault         bool
	DefaultValue       string // serialized form, e.g. "42", "'hello'", "Module.Enum.Value"
	Calculated         bool
	CalculatedMicroflow *QualifiedName
}

// IndexModel is the canonical representation of one entity index.
type IndexModel struct {
	Name    string
	Columns []IndexColumn
}

// IndexColumn is one column within an index.
type IndexColumn struct {
	Name      string
	Ascending bool // true = ASC (default), false = DESC
}
```

- [ ] **Step 2: Verify compile**

```bash
go build ./mdl/model/entity/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add mdl/model/entity/model.go
git commit -m "feat(model/entity): add EntityModel data struct"
```

---

## Task 3: Lift — AST → EntityModel (TDD)

**Files:**
- Create: `mdl/model/entity/lift.go`
- Create: `mdl/model/entity/entity_test.go`

- [ ] **Step 1: Write failing tests in `entity_test.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package entity_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/model"
	"github.com/mendixlabs/mxcli/mdl/model/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLift_PersistentEntity(t *testing.T) {
	stmt := &ast.CreateEntityStmt{
		Name:          ast.QualifiedName{Module: "MyModule", Name: "Customer"},
		Kind:          ast.EntityPersistent,
		Documentation: "A paying customer",
		Attributes: []ast.Attribute{
			{
				Name:         "Name",
				Type:         ast.DataType{Kind: ast.TypeString, Length: 200},
				NotNull:      true,
				NotNullError: "Name is required",
			},
			{
				Name:         "Age",
				Type:         ast.DataType{Kind: ast.TypeInteger},
				HasDefault:   true,
				DefaultValue: 0,
			},
		},
	}

	m, err := entity.Lift(stmt)
	require.NoError(t, err)

	assert.Equal(t, "MyModule", m.Name.Module)
	assert.Equal(t, "Customer", m.Name.Name)
	assert.Equal(t, entity.EntityPersistent, m.Kind)
	assert.Equal(t, "A paying customer", m.Documentation)

	require.Len(t, m.Attributes, 2)

	name := m.Attributes[0]
	assert.Equal(t, "Name", name.Name)
	assert.Equal(t, model.KindString, name.Type.Kind)
	assert.Equal(t, 200, name.Type.Length)
	assert.True(t, name.NotNull)
	assert.Equal(t, "Name is required", name.NotNullError)

	age := m.Attributes[1]
	assert.Equal(t, "Age", age.Name)
	assert.Equal(t, model.KindInteger, age.Type.Kind)
	assert.True(t, age.HasDefault)
	assert.Equal(t, "0", age.DefaultValue)
}

func TestLift_NonPersistent(t *testing.T) {
	stmt := &ast.CreateEntityStmt{
		Name: ast.QualifiedName{Module: "M", Name: "E"},
		Kind: ast.EntityNonPersistent,
	}
	m, err := entity.Lift(stmt)
	require.NoError(t, err)
	assert.Equal(t, entity.EntityNonPersistent, m.Kind)
}

func TestLift_WithExtends(t *testing.T) {
	stmt := &ast.CreateEntityStmt{
		Name:           ast.QualifiedName{Module: "M", Name: "Child"},
		Kind:           ast.EntityPersistent,
		Generalization: &ast.QualifiedName{Module: "M", Name: "Parent"},
	}
	m, err := entity.Lift(stmt)
	require.NoError(t, err)
	require.NotNil(t, m.Extends)
	assert.Equal(t, "M.Parent", m.Extends.String())
}

func TestLift_EnumAttribute(t *testing.T) {
	stmt := &ast.CreateEntityStmt{
		Name: ast.QualifiedName{Module: "M", Name: "E"},
		Kind: ast.EntityPersistent,
		Attributes: []ast.Attribute{
			{
				Name: "Status",
				// TypeEnumeration with EnumRef — from parser this is ambiguous
				Type: ast.DataType{
					Kind:    ast.TypeEnumeration,
					EnumRef: &ast.QualifiedName{Module: "M", Name: "StatusEnum"},
				},
			},
		},
	}
	m, err := entity.Lift(stmt)
	require.NoError(t, err)
	require.Len(t, m.Attributes, 1)
	attr := m.Attributes[0]
	// Lift produces KindUnresolvedRef because the parser cannot distinguish
	// entity refs from enum refs for bare qualified names.
	assert.Equal(t, model.KindUnresolvedRef, attr.Type.Kind)
	assert.Equal(t, "M.StatusEnum", attr.Type.Ref)
}

func TestLift_WithIndex(t *testing.T) {
	stmt := &ast.CreateEntityStmt{
		Name: ast.QualifiedName{Module: "M", Name: "E"},
		Kind: ast.EntityPersistent,
		Attributes: []ast.Attribute{
			{Name: "Name", Type: ast.DataType{Kind: ast.TypeString}},
		},
		Indexes: []ast.Index{
			{Name: "idx_name", Columns: []ast.IndexColumn{{Name: "Name", Ascending: true}}},
		},
	}
	m, err := entity.Lift(stmt)
	require.NoError(t, err)
	require.Len(t, m.Indexes, 1)
	assert.Equal(t, "idx_name", m.Indexes[0].Name)
	require.Len(t, m.Indexes[0].Columns, 1)
	assert.Equal(t, "Name", m.Indexes[0].Columns[0].Name)
	assert.True(t, m.Indexes[0].Columns[0].Ascending)
}
```

- [ ] **Step 2: Run tests — expect compile failure**

```bash
go test ./mdl/model/entity/... 2>&1 | head -20
```

Expected: `entity.Lift undefined` (or similar compile error).

- [ ] **Step 3: Create `mdl/model/entity/lift.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package entity

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/model"
)

// Lift converts a parsed CREATE ENTITY AST statement to an EntityModel.
// It is a pure function: no backend access, no side effects.
// Ambiguous type references (TypeEnumeration from the parser, which could
// be an entity or enum) are preserved as KindUnresolvedRef.
func Lift(s *ast.CreateEntityStmt) (*EntityModel, error) {
	if s == nil {
		return nil, fmt.Errorf("entity.Lift: nil statement")
	}

	m := &EntityModel{
		Name:          QualifiedName{Module: s.Name.Module, Name: s.Name.Name},
		Kind:          liftKind(s.Kind),
		Documentation: s.Documentation,
		SystemMembers: s.SystemMembers,
	}

	if s.Position != nil {
		m.Position = &Position{X: s.Position.X, Y: s.Position.Y}
	}

	if s.Generalization != nil {
		qn := QualifiedName{Module: s.Generalization.Module, Name: s.Generalization.Name}
		m.Extends = &qn
	}

	for _, a := range s.Attributes {
		m.Attributes = append(m.Attributes, liftAttribute(a))
	}

	for _, idx := range s.Indexes {
		m.Indexes = append(m.Indexes, liftIndex(idx))
	}

	return m, nil
}

func liftKind(k ast.EntityKind) EntityKind {
	switch k {
	case ast.EntityPersistent:
		return EntityPersistent
	case ast.EntityNonPersistent:
		return EntityNonPersistent
	case ast.EntityView:
		return EntityView
	case ast.EntityExternal:
		return EntityExternal
	default:
		return EntityPersistent
	}
}

func liftAttribute(a ast.Attribute) AttributeModel {
	attr := AttributeModel{
		Name:          a.Name,
		Type:          liftDataType(a.Type),
		Documentation: a.Documentation,
		NotNull:       a.NotNull,
		NotNullError:  a.NotNullError,
		Unique:        a.Unique,
		UniqueError:   a.UniqueError,
		HasDefault:    a.HasDefault,
		Calculated:    a.Calculated,
	}
	if a.HasDefault {
		attr.DefaultValue = fmt.Sprintf("%v", a.DefaultValue)
	}
	if a.CalculatedMicroflow != nil {
		qn := QualifiedName{Module: a.CalculatedMicroflow.Module, Name: a.CalculatedMicroflow.Name}
		attr.CalculatedMicroflow = &qn
	}
	return attr
}

func liftIndex(idx ast.Index) IndexModel {
	im := IndexModel{Name: idx.Name}
	for _, col := range idx.Columns {
		im.Columns = append(im.Columns, IndexColumn{Name: col.Name, Ascending: col.Ascending})
	}
	return im
}

// liftDataType converts an ast.DataType to a model.DataType.
// TypeEnumeration from the parser is ambiguous (could be entity or enum);
// it is preserved as KindUnresolvedRef for Persist() to resolve.
func liftDataType(dt ast.DataType) model.DataType {
	switch dt.Kind {
	case ast.TypeString:
		return model.DataType{Kind: model.KindString, Length: dt.Length}
	case ast.TypeInteger:
		return model.DataType{Kind: model.KindInteger}
	case ast.TypeLong:
		return model.DataType{Kind: model.KindLong}
	case ast.TypeDecimal:
		return model.DataType{Kind: model.KindDecimal, Precision: dt.Precision, Scale: dt.Scale}
	case ast.TypeBoolean:
		return model.DataType{Kind: model.KindBoolean}
	case ast.TypeDateTime:
		return model.DataType{Kind: model.KindDateTime}
	case ast.TypeBinary:
		return model.DataType{Kind: model.KindBinary}
	case ast.TypeAutoNumber:
		return model.DataType{Kind: model.KindAutoNumber}
	case ast.TypeEnumeration:
		// The parser cannot distinguish entity refs from enum refs.
		// Use EnumRef as the fallback for the qualified name.
		ref := ""
		if dt.EnumRef != nil {
			ref = dt.EnumRef.String()
		} else if dt.EntityRef != nil {
			ref = dt.EntityRef.String()
		}
		return model.DataType{Kind: model.KindUnresolvedRef, Ref: ref}
	case ast.TypeEntity:
		ref := ""
		if dt.EntityRef != nil {
			ref = dt.EntityRef.String()
		}
		return model.DataType{Kind: model.KindEntityRef, Ref: ref}
	case ast.TypeListOf:
		ref := ""
		if dt.EntityRef != nil {
			ref = dt.EntityRef.String()
		}
		return model.DataType{Kind: model.KindListOf, Ref: ref}
	default:
		return model.DataType{Kind: model.KindUnknown}
	}
}
```

- [ ] **Step 4: Run tests — expect pass**

```bash
go test ./mdl/model/entity/... -run TestLift -v
```

Expected: all `TestLift_*` tests pass.

- [ ] **Step 5: Commit**

```bash
git add mdl/model/entity/lift.go mdl/model/entity/entity_test.go
git commit -m "feat(model/entity): implement Lift() — AST to EntityModel (TDD)"
```

---

## Task 4: ToMDL — EntityModel → MDL text (TDD)

**Files:**
- Modify: `mdl/model/entity/entity_test.go` (add tests)
- Create: `mdl/model/entity/serialize.go`

- [ ] **Step 1: Add ToMDL tests to `entity_test.go`**

Append to the file:

```go
func TestToMDL_MinimalEntity(t *testing.T) {
	m := &entity.EntityModel{
		Name: entity.QualifiedName{Module: "M", Name: "E"},
		Kind: entity.EntityPersistent,
	}
	got := m.ToMDL()
	want := "create persistent entity M.E (\n)"
	assert.Equal(t, want, got)
}

func TestToMDL_WithAttributes(t *testing.T) {
	m := &entity.EntityModel{
		Name: entity.QualifiedName{Module: "MyModule", Name: "Customer"},
		Kind: entity.EntityPersistent,
		Attributes: []entity.AttributeModel{
			{
				Name:         "Name",
				Type:         model.DataType{Kind: model.KindString, Length: 200},
				NotNull:      true,
				NotNullError: "Name is required",
			},
			{
				Name:       "Age",
				Type:       model.DataType{Kind: model.KindInteger},
				HasDefault: true,
				DefaultValue: "0",
			},
		},
	}
	got := m.ToMDL()
	want := "create persistent entity MyModule.Customer (\n" +
		"  Name: String(200) not null error 'Name is required',\n" +
		"  Age: Integer default 0\n" +
		")"
	assert.Equal(t, want, got)
}

func TestToMDL_NonPersistent(t *testing.T) {
	m := &entity.EntityModel{
		Name: entity.QualifiedName{Module: "M", Name: "E"},
		Kind: entity.EntityNonPersistent,
	}
	assert.Contains(t, m.ToMDL(), "non-persistent")
}

func TestToMDL_WithExtends(t *testing.T) {
	parent := entity.QualifiedName{Module: "M", Name: "Parent"}
	m := &entity.EntityModel{
		Name:    entity.QualifiedName{Module: "M", Name: "Child"},
		Kind:    entity.EntityPersistent,
		Extends: &parent,
	}
	assert.Contains(t, m.ToMDL(), "extends M.Parent")
}

func TestToMDL_UnresolvedRefAttribute(t *testing.T) {
	m := &entity.EntityModel{
		Name: entity.QualifiedName{Module: "M", Name: "E"},
		Kind: entity.EntityPersistent,
		Attributes: []entity.AttributeModel{
			{
				Name: "Status",
				Type: model.DataType{Kind: model.KindUnresolvedRef, Ref: "M.StatusEnum"},
			},
		},
	}
	got := m.ToMDL()
	assert.Contains(t, got, "Status: M.StatusEnum")
}

func TestToMDL_WithIndex(t *testing.T) {
	m := &entity.EntityModel{
		Name: entity.QualifiedName{Module: "M", Name: "E"},
		Kind: entity.EntityPersistent,
		Attributes: []entity.AttributeModel{
			{Name: "Code", Type: model.DataType{Kind: model.KindString}},
		},
		Indexes: []entity.IndexModel{
			{Name: "idx_code", Columns: []entity.IndexColumn{{Name: "Code", Ascending: true}}},
		},
	}
	got := m.ToMDL()
	assert.Contains(t, got, "index idx_code (Code)")
}

func TestToMDL_WithDocumentation(t *testing.T) {
	m := &entity.EntityModel{
		Name:          entity.QualifiedName{Module: "M", Name: "E"},
		Kind:          entity.EntityPersistent,
		Documentation: "My entity doc",
	}
	got := m.ToMDL()
	assert.HasPrefix(t, got, "/**\n * My entity doc\n */\n")
}

// TestToMDL_LiftRoundTrip verifies that Lift(ast) then ToMDL() produces
// the expected canonical form (not necessarily byte-identical to input, but
// semantically equivalent and deterministic).
func TestToMDL_LiftRoundTrip(t *testing.T) {
	stmt := &ast.CreateEntityStmt{
		Name: ast.QualifiedName{Module: "M", Name: "Ticket"},
		Kind: ast.EntityPersistent,
		Attributes: []ast.Attribute{
			{Name: "Title", Type: ast.DataType{Kind: ast.TypeString, Length: 500}, NotNull: true},
			{Name: "Priority", Type: ast.DataType{Kind: ast.TypeInteger}, HasDefault: true, DefaultValue: 1},
		},
	}
	m, err := entity.Lift(stmt)
	require.NoError(t, err)
	got := m.ToMDL()
	assert.Contains(t, got, "M.Ticket")
	assert.Contains(t, got, "Title: String(500) not null")
	assert.Contains(t, got, "Priority: Integer default 1")
}
```

- [ ] **Step 2: Run tests — expect failures**

```bash
go test ./mdl/model/entity/... -run TestToMDL -v 2>&1 | head -30
```

Expected: `(*EntityModel).ToMDL undefined`.

- [ ] **Step 3: Create `mdl/model/entity/serialize.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package entity

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/model"
)

// ToMDL serializes the EntityModel to a canonical MDL CREATE ENTITY statement.
// Output is deterministic: attributes in declaration order, indexes sorted by name.
// This is the single serialization path — both the write path (via Lift) and
// the read path (via Hydrate) produce the same text through this method.
func (m *EntityModel) ToMDL() string {
	var sb strings.Builder

	if m.Documentation != "" {
		fmt.Fprintf(&sb, "/**\n * %s\n */\n", m.Documentation)
	}

	if m.Position != nil {
		fmt.Fprintf(&sb, "@Position(%d, %d)\n", m.Position.X, m.Position.Y)
	}

	kindStr := kindToMDL(m.Kind)
	if m.Extends != nil {
		fmt.Fprintf(&sb, "create %s entity %s extends %s (\n", kindStr, m.Name, m.Extends)
	} else {
		fmt.Fprintf(&sb, "create %s entity %s (\n", kindStr, m.Name)
	}

	for i, attr := range m.Attributes {
		if attr.Documentation != "" {
			fmt.Fprintf(&sb, "  /** %s */\n", attr.Documentation)
		}
		comma := ","
		if i == len(m.Attributes)-1 {
			comma = ""
		}
		fmt.Fprintf(&sb, "  %s: %s%s%s\n", attr.Name, dataTypeToMDL(attr.Type), constraintsToMDL(attr), comma)
	}

	sb.WriteString(")")

	// Indexes follow the closing paren.
	for _, idx := range m.Indexes {
		var cols []string
		for _, col := range idx.Columns {
			if col.Ascending {
				cols = append(cols, col.Name)
			} else {
				cols = append(cols, col.Name+" desc")
			}
		}
		fmt.Fprintf(&sb, "\nindex %s (%s)", idx.Name, strings.Join(cols, ", "))
	}

	return sb.String()
}

func kindToMDL(k EntityKind) string {
	switch k {
	case EntityNonPersistent:
		return "non-persistent"
	case EntityView:
		return "view"
	case EntityExternal:
		return "external"
	default:
		return "persistent"
	}
}

func dataTypeToMDL(dt model.DataType) string {
	switch dt.Kind {
	case model.KindString:
		if dt.Length > 0 {
			return fmt.Sprintf("String(%d)", dt.Length)
		}
		return "String"
	case model.KindInteger:
		return "Integer"
	case model.KindLong:
		return "Long"
	case model.KindDecimal:
		if dt.Precision > 0 {
			return fmt.Sprintf("Decimal(%d, %d)", dt.Precision, dt.Scale)
		}
		return "Decimal"
	case model.KindBoolean:
		return "Boolean"
	case model.KindDateTime:
		return "DateTime"
	case model.KindBinary:
		return "Binary"
	case model.KindAutoNumber:
		return "AutoNumber"
	case model.KindEnumRef, model.KindEntityRef, model.KindUnresolvedRef:
		return dt.Ref
	case model.KindListOf:
		return "List of " + dt.Ref
	default:
		return "Unknown"
	}
}

func constraintsToMDL(attr AttributeModel) string {
	var sb strings.Builder
	if attr.NotNull {
		sb.WriteString(" not null")
		if attr.NotNullError != "" {
			fmt.Fprintf(&sb, " error '%s'", strings.ReplaceAll(attr.NotNullError, "'", "''"))
		}
	}
	if attr.Unique {
		sb.WriteString(" unique")
		if attr.UniqueError != "" {
			fmt.Fprintf(&sb, " error '%s'", strings.ReplaceAll(attr.UniqueError, "'", "''"))
		}
	}
	if attr.HasDefault {
		fmt.Fprintf(&sb, " default %s", attr.DefaultValue)
	}
	return sb.String()
}
```

- [ ] **Step 4: Run tests — expect pass**

```bash
go test ./mdl/model/entity/... -run TestToMDL -v
```

Expected: all `TestToMDL_*` tests pass. Fix `assert.HasPrefix` — use `assert.True(t, strings.HasPrefix(got, ...))` if testify doesn't have that helper.

- [ ] **Step 5: Commit**

```bash
git add mdl/model/entity/serialize.go mdl/model/entity/entity_test.go
git commit -m "feat(model/entity): implement ToMDL() serializer (TDD)"
```

---

## Task 5: Hydrate — gen Entity → EntityModel (TDD)

**Files:**
- Modify: `mdl/model/entity/entity_test.go` (add tests)
- Create: `mdl/model/entity/hydrate.go`

- [ ] **Step 1: Add Hydrate tests to `entity_test.go`**

Append to the file:

```go
func TestHydrate_BasicEntity(t *testing.T) {
	e := genDm.NewEntity()
	e.SetName("Customer")
	e.SetDocumentation("A paying customer")

	noGen := genDm.NewNoGeneralization()
	noGen.SetPersistable(true)
	e.SetGeneralization(noGen)

	attr := genDm.NewAttribute()
	attr.SetName("Name")
	strType := genDm.NewStringAttributeType()
	strType.SetMaxLength(200)
	attr.SetType(strType)
	e.AddAttributes(attr)

	m, warns, err := entity.Hydrate("MyModule", e)
	require.NoError(t, err)
	assert.Empty(t, warns)
	assert.Equal(t, "MyModule", m.Name.Module)
	assert.Equal(t, "Customer", m.Name.Name)
	assert.Equal(t, "A paying customer", m.Documentation)
	assert.Equal(t, entity.EntityPersistent, m.Kind)
	require.Len(t, m.Attributes, 1)
	assert.Equal(t, "Name", m.Attributes[0].Name)
	assert.Equal(t, model.KindString, m.Attributes[0].Type.Kind)
	assert.Equal(t, 200, m.Attributes[0].Type.Length)
}

func TestHydrate_NotNullConstraint(t *testing.T) {
	e := genDm.NewEntity()
	e.SetName("E")
	noGen := genDm.NewNoGeneralization()
	noGen.SetPersistable(true)
	e.SetGeneralization(noGen)

	attr := genDm.NewAttribute()
	attr.SetName("Title")
	strType := genDm.NewStringAttributeType()
	strType.SetMaxLength(500)
	attr.SetType(strType)
	e.AddAttributes(attr)

	vr := genDm.NewValidationRule()
	vr.SetAttributeQualifiedName("M.E.Title")
	vr.SetRuleInfo(genDm.NewRequiredRuleInfo())
	e.AddValidationRules(vr)

	m, _, err := entity.Hydrate("M", e)
	require.NoError(t, err)
	require.Len(t, m.Attributes, 1)
	assert.True(t, m.Attributes[0].NotNull)
}

func TestHydrate_EnumAttribute(t *testing.T) {
	e := genDm.NewEntity()
	e.SetName("Ticket")
	noGen := genDm.NewNoGeneralization()
	noGen.SetPersistable(true)
	e.SetGeneralization(noGen)

	attr := genDm.NewAttribute()
	attr.SetName("Status")
	enumType := genDm.NewEnumerationAttributeType()
	enumType.SetEnumerationQualifiedName("M.StatusEnum")
	attr.SetType(enumType)
	e.AddAttributes(attr)

	m, _, err := entity.Hydrate("M", e)
	require.NoError(t, err)
	require.Len(t, m.Attributes, 1)
	assert.Equal(t, model.KindEnumRef, m.Attributes[0].Type.Kind)
	assert.Equal(t, "M.StatusEnum", m.Attributes[0].Type.Ref)
}
```

Add to imports block at top of entity_test.go:
```go
genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
```

- [ ] **Step 2: Run tests — expect compile failure**

```bash
go test ./mdl/model/entity/... -run TestHydrate -v 2>&1 | head -20
```

Expected: `entity.Hydrate undefined`.

- [ ] **Step 3: Create `mdl/model/entity/hydrate.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package entity

import (
	"strings"

	"github.com/mendixlabs/mxcli/mdl/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// Hydrate converts a gen-typed Entity to an EntityModel.
// It is defensive: unknown BSON fields and sub-types are silently skipped
// and reported as Warnings. It never returns an error for missing optional
// fields — only for structural corruption.
//
// moduleName is the owning module name (not stored in the gen entity itself).
func Hydrate(moduleName string, e *genDm.Entity) (*EntityModel, []model.Warning, error) {
	var warns []model.Warning

	m := &EntityModel{
		Name:          QualifiedName{Module: moduleName, Name: e.Name()},
		Documentation: e.Documentation(),
		Kind:          hydrateKind(e),
	}

	if loc := e.Location(); loc != "" {
		if x, y, ok := parseLocationBSON(loc); ok {
			m.Position = &Position{X: x, Y: y}
		}
	}

	if ext := entityGeneralizationQN(e); ext != "" {
		parts := strings.SplitN(ext, ".", 2)
		if len(parts) == 2 {
			qn := QualifiedName{Module: parts[0], Name: parts[1]}
			m.Extends = &qn
		}
	}

	// Build validation-rule index by attribute name.
	notNullAttrs := make(map[string]bool)
	uniqueAttrs := make(map[string]bool)
	for _, item := range e.ValidationRulesItems() {
		vr, ok := item.(*genDm.ValidationRule)
		if !ok {
			warns = append(warns, model.Warning{Field: "ValidationRules", Message: "unexpected item type"})
			continue
		}
		attrName := extractAttrName(vr.AttributeQualifiedName())
		switch vr.RuleInfo().TypeName() {
		case "DomainModels$RequiredRuleInfo":
			notNullAttrs[attrName] = true
		case "DomainModels$UniqueRuleInfo":
			uniqueAttrs[attrName] = true
		}
	}

	for _, item := range e.AttributesItems() {
		attr, ok := item.(*genDm.Attribute)
		if !ok {
			warns = append(warns, model.Warning{Field: "Attributes", Message: "unexpected item type"})
			continue
		}
		am, w := hydrateAttribute(attr, notNullAttrs, uniqueAttrs)
		warns = append(warns, w...)
		m.Attributes = append(m.Attributes, am)
	}

	for _, item := range e.IndexesItems() {
		idx, ok := item.(*genDm.Index)
		if !ok {
			warns = append(warns, model.Warning{Field: "Indexes", Message: "unexpected item type"})
			continue
		}
		im, w := hydrateIndex(idx, e)
		warns = append(warns, w...)
		m.Indexes = append(m.Indexes, im)
	}

	return m, warns, nil
}

func hydrateKind(e *genDm.Entity) EntityKind {
	if src := e.Source(); src != nil && strings.Contains(src.TypeName(), "OqlView") {
		return EntityView
	}
	g, ok := e.Generalization().(*genDm.NoGeneralization)
	if ok && !g.Persistable() {
		return EntityNonPersistent
	}
	return EntityPersistent
}

func hydrateAttribute(attr *genDm.Attribute, notNull, unique map[string]bool) (AttributeModel, []model.Warning) {
	am := AttributeModel{
		Name:          attr.Name(),
		Documentation: attr.Documentation(),
		Type:          hydrateDataType(attr.Type()),
		NotNull:       notNull[attr.Name()],
		Unique:        unique[attr.Name()],
	}
	if sv, ok := attr.Value().(*genDm.StoredValue); ok && sv.DefaultValue() != "" {
		am.HasDefault = true
		am.DefaultValue = sv.DefaultValue()
		// Wrap string defaults in single quotes for MDL output.
		if _, ok := attr.Type().(*genDm.StringAttributeType); ok {
			am.DefaultValue = "'" + strings.ReplaceAll(sv.DefaultValue(), "'", "''") + "'"
		}
	}
	return am, nil
}

func hydrateDataType(t interface{ TypeName() string }) model.DataType {
	switch v := t.(type) {
	case *genDm.StringAttributeType:
		return model.DataType{Kind: model.KindString, Length: int(v.MaxLength())}
	case *genDm.IntegerAttributeType:
		return model.DataType{Kind: model.KindInteger}
	case *genDm.LongAttributeType:
		return model.DataType{Kind: model.KindLong}
	case *genDm.DecimalAttributeType:
		return model.DataType{Kind: model.KindDecimal}
	case *genDm.BooleanAttributeType:
		return model.DataType{Kind: model.KindBoolean}
	case *genDm.DateTimeAttributeType:
		return model.DataType{Kind: model.KindDateTime}
	case *genDm.BinaryAttributeType:
		return model.DataType{Kind: model.KindBinary}
	case *genDm.AutoNumberAttributeType:
		return model.DataType{Kind: model.KindAutoNumber}
	case *genDm.EnumerationAttributeType:
		return model.DataType{Kind: model.KindEnumRef, Ref: v.EnumerationQualifiedName()}
	default:
		return model.DataType{Kind: model.KindUnknown}
	}
}

func hydrateIndex(idx *genDm.Index, e *genDm.Entity) (IndexModel, []model.Warning) {
	// Build attribute ID→name map from the entity.
	attrNames := make(map[string]string)
	for _, item := range e.AttributesItems() {
		if a, ok := item.(*genDm.Attribute); ok {
			attrNames[a.ID()] = a.Name()
		}
	}
	im := IndexModel{Name: idx.Name()}
	for _, item := range idx.AttributesItems() {
		ia, ok := item.(*genDm.IndexedAttribute)
		if !ok {
			continue
		}
		col := IndexColumn{
			Name:      attrNames[ia.AttributeRefID()],
			Ascending: ia.Ascending(),
		}
		if col.Name == "" {
			col.Name = ia.AttributeRefID() // fallback: use ID if name not found
		}
		im.Columns = append(im.Columns, col)
	}
	return im, nil
}

// extractAttrName returns the simple attribute name from a qualified name
// like "Module.Entity.AttrName" → "AttrName".
func extractAttrName(qn string) string {
	parts := strings.Split(qn, ".")
	if len(parts) == 0 {
		return qn
	}
	return parts[len(parts)-1]
}

// entityGeneralizationQN returns the qualified name of the entity's
// generalization parent, or "" if none.
func entityGeneralizationQN(e *genDm.Entity) string {
	g := e.Generalization()
	if g == nil {
		return ""
	}
	// Only Generalization (not NoGeneralization) has a parent entity name.
	type namer interface{ EntityQualifiedName() string }
	if n, ok := g.(namer); ok {
		return n.EntityQualifiedName()
	}
	return ""
}

// parseLocationBSON parses a location string like "100 200" to x, y ints.
// Returns ok=false if parsing fails.
func parseLocationBSON(loc string) (x, y int, ok bool) {
	_, err := fmt.Sscanf(loc, "%d %d", &x, &y)
	return x, y, err == nil
}
```

Add `"fmt"` to the imports block.

- [ ] **Step 4: Run tests — expect pass**

```bash
go test ./mdl/model/entity/... -run TestHydrate -v
```

Expected: all `TestHydrate_*` tests pass.

- [ ] **Step 5: Commit**

```bash
git add mdl/model/entity/hydrate.go mdl/model/entity/entity_test.go
git commit -m "feat(model/entity): implement Hydrate() — gen Entity to EntityModel (TDD)"
```

---

## Task 6: Persist — EntityModel → BSON via backend (integration TDD)

**Files:**
- Create: `mdl/model/entity/persist_test.go`
- Create: `mdl/model/entity/persist.go`

- [ ] **Step 1: Write failing integration test in `persist_test.go`**

```go
// SPDX-License-Identifier: Apache-2.0

//go:build integration

package entity_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/mendixlabs/mxcli/mdl/model"
	"github.com/mendixlabs/mxcli/mdl/model/entity"
	modelID "github.com/mendixlabs/mxcli/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPersist_NewEntity writes an EntityModel to a real MPR file
// and verifies it can be hydrated back with the same field values.
// Uses testdata/expr-checker/minimal.mpr (Mendix 11.6.6).
// Run: go test -tags integration ./mdl/model/entity/... -v
func TestPersist_NewEntity(t *testing.T) {
	const mprPath = "../../../testdata/expr-checker/minimal.mpr"
	t.Cleanup(func() {
		exec.Command("git", "restore", "../../../testdata/expr-checker/").Run()
	})

	b, err := mpr.OpenForWriting(mprPath)
	require.NoError(t, err)
	defer b.Close()

	// Find the MyFirstModule domain model.
	dms, err := b.ListDomainModelsGen()
	require.NoError(t, err)
	var dmID modelID.ID
	var domainModel *genDm.DomainModel
	for _, dm := range dms {
		if dm.ContainerName == "MyFirstModule" {
			dmID = model.ID(dm.ID())
			domainModel = dm
			break
		}
	}
	require.NotEmpty(t, dmID, "MyFirstModule domain model not found")
	require.NotNil(t, domainModel)

	m := &entity.EntityModel{
		Name: entity.QualifiedName{Module: "MyFirstModule", Name: "PersistTestEntity"},
		Kind: entity.EntityPersistent,
		Attributes: []entity.AttributeModel{
			{
				Name:    "Title",
				Type:    model.DataType{Kind: model.KindString, Length: 200},
				NotNull: true,
			},
		},
	}

	ctx := model.PersistContext{
		DomainModelID: dmID,
		Backend:       b,
	}
	require.NoError(t, m.Persist(ctx))
	require.NoError(t, b.Flush())

	// Read back by re-loading the domain model and searching entities.
	dms2, err := b.ListDomainModelsGen()
	require.NoError(t, err)
	var found *entity.EntityModel
	for _, dm := range dms2 {
		if dm.ContainerName != "MyFirstModule" {
			continue
		}
		for _, item := range dm.EntitiesItems() {
			e, ok := item.(*genDm.Entity)
			if !ok || e.Name() != "PersistTestEntity" {
				continue
			}
			hydrated, _, err := entity.Hydrate("MyFirstModule", e)
			require.NoError(t, err)
			found = hydrated
			break
		}
	}
	require.NotNil(t, found, "PersistTestEntity not found after persist")
	assert.Equal(t, "MyFirstModule", found.Name.Module)
	assert.Equal(t, "PersistTestEntity", found.Name.Name)
	require.Len(t, found.Attributes, 1)
	assert.Equal(t, "Title", found.Attributes[0].Name)
	assert.True(t, found.Attributes[0].NotNull)
}
```

- [ ] **Step 2: Run test — expect compile failure**

```bash
go test -tags integration ./mdl/model/entity/... -run TestPersist -v 2>&1 | head -20
```

Expected: `entity.Persist undefined` or missing import.

- [ ] **Step 3: Create `mdl/model/entity/persist.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package entity

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
	modelID "github.com/mendixlabs/mxcli/model"
)

// Persist writes the EntityModel to the MPR via ctx.Backend.
// If ctx.ExistingEntityID is non-zero, it updates the existing entity;
// otherwise it creates a new one. Non-MDL BSON fields (IDs, version
// prefixes) are generated by the backend — not the canonical model.
func (m *EntityModel) Persist(ctx model.PersistContext) error {
	gen, err := buildGenEntity(m, ctx.ExistingEntityID)
	if err != nil {
		return fmt.Errorf("entity.Persist: build gen: %w", err)
	}

	if ctx.ExistingEntityID == (modelID.ID{}) {
		return ctx.Backend.CreateEntityGen(ctx.DomainModelID, gen)
	}
	return ctx.Backend.UpdateEntityGen(ctx.ExistingEntityID, ctx.DomainModelID, gen)
}

// buildGenEntity constructs a gen-typed *Entity from an EntityModel.
// existingID is zero for new entities; non-zero copies the existing BSON ID.
func buildGenEntity(m *EntityModel, existingID modelID.ID) (*genDm.Entity, error) {
	e := genDm.NewEntity()
	e.SetName(m.Name.Name)
	e.SetDocumentation(m.Documentation)

	if m.Position != nil {
		e.SetLocation(layoutPos(m.Position.X, m.Position.Y))
	}

	// Generalization / kind.
	if m.Extends != nil {
		gen := genDm.NewGeneralization()
		gen.SetEntityQualifiedName(m.Extends.String())
		e.SetGeneralization(gen)
	} else {
		noGen := genDm.NewNoGeneralization()
		noGen.SetPersistable(m.Kind != EntityNonPersistent)
		// Mendix requires all 4 system flags present even when false (CE0161).
		hasOwner := contains(m.SystemMembers, "owner")
		hasChangedBy := contains(m.SystemMembers, "changedBy")
		hasCreated := contains(m.SystemMembers, "createdDate")
		hasChanged := contains(m.SystemMembers, "changedDate")
		noGen.SetHasOwner(hasOwner)
		noGen.SetHasChangedBy(hasChangedBy)
		noGen.SetHasCreatedDate(hasCreated)
		noGen.SetHasChangedDate(hasChanged)
		e.SetGeneralization(noGen)
	}

	// Attributes.
	for _, a := range m.Attributes {
		attr, err := buildGenAttribute(a, m.Name.String())
		if err != nil {
			return nil, err
		}
		e.AddAttributes(attr)
	}

	// Validation rules (not null, unique).
	for _, a := range m.Attributes {
		rules := buildValidationRules(a, m.Name.String())
		for _, r := range rules {
			e.AddValidationRules(r)
		}
	}

	// Indexes.
	for _, idx := range m.Indexes {
		genIdx, err := buildGenIndex(idx, e)
		if err != nil {
			return nil, err
		}
		e.AddIndexes(genIdx)
	}

	return e, nil
}

func buildGenAttribute(a AttributeModel, entityQN string) (*genDm.Attribute, error) {
	attr := genDm.NewAttribute()
	attr.SetName(a.Name)
	attr.SetDocumentation(a.Documentation)

	t, err := buildGenAttributeType(a.Type)
	if err != nil {
		return nil, fmt.Errorf("attribute %q: %w", a.Name, err)
	}
	attr.SetType(t)

	if a.HasDefault {
		sv := genDm.NewStoredValue()
		// Strip surrounding quotes for string defaults before storing.
		dv := a.DefaultValue
		if strings.HasPrefix(dv, "'") && strings.HasSuffix(dv, "'") {
			dv = dv[1 : len(dv)-1]
			dv = strings.ReplaceAll(dv, "''", "'")
		}
		sv.SetDefaultValue(dv)
		attr.SetValue(sv)
	}

	return attr, nil
}

func buildGenAttributeType(dt model.DataType) (interface{}, error) {
	switch dt.Kind {
	case model.KindString:
		t := genDm.NewStringAttributeType()
		if dt.Length > 0 {
			t.SetMaxLength(int32(dt.Length))
		} else {
			t.SetMaxLength(-1) // unlimited
		}
		return t, nil
	case model.KindInteger:
		return genDm.NewIntegerAttributeType(), nil
	case model.KindLong:
		return genDm.NewLongAttributeType(), nil
	case model.KindDecimal:
		t := genDm.NewDecimalAttributeType()
		return t, nil
	case model.KindBoolean:
		return genDm.NewBooleanAttributeType(), nil
	case model.KindDateTime:
		t := genDm.NewDateTimeAttributeType()
		t.SetLocalizeDate(false) // deterministic BSON
		return t, nil
	case model.KindBinary:
		return genDm.NewBinaryAttributeType(), nil
	case model.KindAutoNumber:
		return genDm.NewAutoNumberAttributeType(), nil
	case model.KindEnumRef, model.KindUnresolvedRef:
		t := genDm.NewEnumerationAttributeType()
		t.SetEnumerationQualifiedName(dt.Ref)
		return t, nil
	default:
		return nil, fmt.Errorf("unsupported DataTypeKind %d", dt.Kind)
	}
}

func buildValidationRules(a AttributeModel, entityQN string) []*genDm.ValidationRule {
	var rules []*genDm.ValidationRule
	attrQN := entityQN + "." + a.Name

	if a.NotNull {
		r := genDm.NewValidationRule()
		r.SetAttributeQualifiedName(attrQN)
		ri := genDm.NewRequiredRuleInfo()
		r.SetRuleInfo(ri)
		if a.NotNullError != "" {
			// error message goes in a Text element
		}
		rules = append(rules, r)
	}
	if a.Unique {
		r := genDm.NewValidationRule()
		r.SetAttributeQualifiedName(attrQN)
		r.SetRuleInfo(genDm.NewUniqueRuleInfo())
		rules = append(rules, r)
	}
	return rules
}

func buildGenIndex(idx IndexModel, e *genDm.Entity) (*genDm.Index, error) {
	// Build attr name→ID map from the entity's already-added attributes.
	attrIDs := make(map[string]string)
	for _, item := range e.AttributesItems() {
		if a, ok := item.(*genDm.Attribute); ok {
			attrIDs[a.Name()] = a.ID()
		}
	}
	genIdx := genDm.NewIndex()
	genIdx.SetName(idx.Name)
	for _, col := range idx.Columns {
		ia := genDm.NewIndexedAttribute()
		ia.SetAttributeRef(modelsdkmpr.BinaryIDFromString(attrIDs[col.Name]))
		ia.SetAscending(col.Ascending)
		genIdx.AddAttributes(ia)
	}
	return genIdx, nil
}

func layoutPos(x, y int) string {
	return fmt.Sprintf("%d %d", x, y)
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Fix any compile errors then run integration test**

```bash
go build ./mdl/model/entity/...
go test -tags integration ./mdl/model/entity/... -run TestPersist -v
```

Expected: `TestPersist_NewEntity` passes. If `UpdateEntityGen` doesn't exist on the backend interface, check `mdl/backend/domainmodel.go` and use the correct method name.

- [ ] **Step 5: Restore testdata**

```bash
git restore testdata/expr-checker/
```

- [ ] **Step 6: Commit**

```bash
git add mdl/model/entity/persist.go mdl/model/entity/persist_test.go
git commit -m "feat(model/entity): implement Persist() — EntityModel to BSON (integration TDD)"
```

---

## Task 7: Register codec + wire into ExecContext

**Files:**
- Create: `mdl/model/entity/codec.go`
- Modify: `mdl/executor/exec_context.go` (add `ModelCodecs` field)
- Modify: `mdl/executor/executor_dispatch.go` (populate `ModelCodecs`)
- Modify: `mdl/executor/executor.go` (construct registry and register entity codec)

- [ ] **Step 1: Create `mdl/model/entity/codec.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package entity

import (
	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// RegisterCodec registers the entity Lift and Hydrate codecs in r.
// Call this once during executor initialization.
func RegisterCodec(r *model.DefaultRegistry) {
	r.Register(
		(*ast.CreateEntityStmt)(nil),
		"DomainModels$Entity",
		model.Codec{
			LiftFn: func(stmt any) (model.Persistable, error) {
				return Lift(stmt.(*ast.CreateEntityStmt))
			},
			HydrateFn: func(el any) (model.Document, []model.Warning, error) {
				e, ok := el.(*genDm.Entity)
				if !ok {
					return nil, nil, fmt.Errorf("entity codec: expected *genDm.Entity, got %T", el)
				}
				// moduleName is not stored in the gen entity; the caller
				// must provide it via HydrateWithModule when known.
				// For registry-level hydration we use empty module name
				// and the describe handler fills it in.
				return Hydrate("", e)
			},
		},
	)
}
```

Add `"fmt"` to imports.

- [ ] **Step 2: Add `HydrateWithModule` helper to `hydrate.go`**

This lets the describe handler pass the module name without needing to call HydrateFrom:

```go
// HydrateWithModule is like Hydrate but accepts the module name explicitly.
// Use this when the caller knows the module name (e.g. from the container).
func HydrateWithModule(moduleName string, e *genDm.Entity) (*EntityModel, []model.Warning, error) {
	return Hydrate(moduleName, e)
}
```

- [ ] **Step 3: Add `ModelCodecs` field to `exec_context.go`**

In the `ExecContext` struct, after the `Backend` field, add:

```go
// ModelCodecs is the canonical model codec registry.
// Populated by newExecContext; nil in minimal test contexts.
ModelCodecs *model.DefaultRegistry
```

Add import: `"github.com/mendixlabs/mxcli/mdl/model"`.

- [ ] **Step 4: Construct registry in `executor.go`**

Find the `NewExecutor` function (or the struct initializer) and add:

```go
// In executor.go, where the Executor struct is initialized:
import (
    entityModel "github.com/mendixlabs/mxcli/mdl/model/entity"
    "github.com/mendixlabs/mxcli/mdl/model"
)

// In NewExecutor or equivalent:
modelCodecs := model.NewDefaultRegistry()
entityModel.RegisterCodec(modelCodecs)
// Phase 2+ will add: assoc.RegisterCodec(modelCodecs), etc.
```

Store `modelCodecs` in the `Executor` struct: add a field `modelCodecs *model.DefaultRegistry`.

- [ ] **Step 5: Populate in `newExecContext` in `executor_dispatch.go`**

In the `return &ExecContext{...}` block, add:

```go
ModelCodecs: e.modelCodecs,
```

- [ ] **Step 6: Verify compile**

```bash
go build ./mdl/executor/... ./cmd/mxcli/...
```

Expected: no errors.

- [ ] **Step 7: Commit**

```bash
git add mdl/model/entity/codec.go mdl/model/entity/hydrate.go \
        mdl/executor/exec_context.go mdl/executor/executor_dispatch.go \
        mdl/executor/executor.go
git commit -m "feat(model/entity): register codec, wire ModelCodecs into ExecContext"
```

---

## Task 8: Refactor CREATE ENTITY handler to use Lift + Persist

**Files:**
- Modify: `mdl/executor/cmd_create_entity_gen.go`

- [ ] **Step 1: Write a test verifying the handler still works via mock**

In `mdl/executor/`, check if there is an existing test for `execCreateEntityGen`. If so, run it first as a baseline:

```bash
go test ./mdl/executor/... -run TestExecCreate -v 2>&1 | head -30
```

- [ ] **Step 2: Update `execCreateEntityGen` to use registry**

Locate the section in `cmd_create_entity_gen.go` where it calls `astToEntityGen(s)`. Replace the body after the module/DM resolution with:

```go
// After module and DM have been resolved (keep existing resolution logic):

// Lift AST → canonical model.
doc, err := ctx.ModelCodecs.LiftFrom(s)
if err != nil {
    return fmt.Errorf("CREATE ENTITY: lift: %w", err)
}

// Resolve existing entity ID for CREATE OR MODIFY.
existingID := model.ID{}
if s.CreateOrModify {
    for _, e := range dm.EntitiesItems() {
        if ent, ok := e.(*genDm.Entity); ok && ent.Name() == s.Name.Name {
            existingID = model.ID(ent.ID())
            break
        }
    }
}

// Persist.
pCtx := mdlmodel.PersistContext{
    DomainModelID:    model.ID(dm.ID()),
    ExistingEntityID: existingID,
    Backend:          ctx.Backend,
}
if err := doc.Persist(pCtx); err != nil {
    return fmt.Errorf("CREATE ENTITY: persist: %w", err)
}

// Invalidate cache so subsequent reads see the new entity.
ctx.Cache.invalidateDomainModel(module.ID)
return nil
```

Add import aliases as needed:
```go
mdlmodel "github.com/mendixlabs/mxcli/mdl/model"
```

- [ ] **Step 3: Build and run existing tests**

```bash
go build ./mdl/executor/...
go test ./mdl/executor/... -run TestExecCreate -v
```

Expected: same pass/fail as Step 1 baseline.

- [ ] **Step 4: Commit**

```bash
git add mdl/executor/cmd_create_entity_gen.go
git commit -m "refactor(executor): CREATE ENTITY delegates to model.Lift + Persist"
```

---

## Task 9: Refactor DESCRIBE ENTITY handler to use Hydrate + ToMDL

**Files:**
- Modify: `mdl/executor/cmd_entities_gen.go`

- [ ] **Step 1: Locate `describeEntityGen` and identify the output section**

The function is in `cmd_entities_gen.go` starting around line 357. It manually builds MDL text using `switch TypeName()`. The goal is to replace the text-building section with `Hydrate + ToMDL`.

- [ ] **Step 2: Rewrite the output section of `describeEntityGen`**

Keep the `findEntityGen(ctx, name)` call at the top. Replace everything after the entity is fetched with:

```go
entity, modName, err := findEntityGen(ctx, name)
if err != nil {
    return mdlerrors.NewBackend("get entity", err)
}
if entity == nil {
    return mdlerrors.NewNotFound("entity", name.String())
}

// Hydrate gen entity → canonical model, then serialize.
m, warns, err := entityModel.HydrateWithModule(modName, entity)
if err != nil {
    return fmt.Errorf("describe entity: hydrate: %w", err)
}
for _, w := range warns {
    ctx.Logger.Debugf("describe entity %s: warning: %s.%s: %s", name, w.Field, w.Field, w.Message)
}
fmt.Fprint(ctx.Output, m.ToMDL())
fmt.Fprintln(ctx.Output)

// Access rules are delegated to outputEntityAccessGrantsGen (unchanged).
return outputEntityAccessGrantsGen(ctx, entity, modName)
```

Add import:
```go
entityModel "github.com/mendixlabs/mxcli/mdl/model/entity"
```

- [ ] **Step 3: Build**

```bash
go build ./mdl/executor/...
```

Expected: no errors.

- [ ] **Step 4: Smoke test with a real project (if available)**

```bash
./bin/mxcli -p testdata/corpus-b/app.mpr -c "describe entity Administration.Account"
```

Expected: valid MDL output. Compare with output before the change by running the old binary if available. Output must contain `create persistent entity Administration.Account`.

- [ ] **Step 5: Run full test suite**

```bash
go test ./mdl/executor/... -count=1
```

Expected: no new failures.

- [ ] **Step 6: Commit**

```bash
git add mdl/executor/cmd_entities_gen.go
git commit -m "refactor(executor): DESCRIBE ENTITY delegates to model.Hydrate + ToMDL"
```

---

## Task 10: Unify diff command — replace dual serializers with ToMDL

**Files:**
- Modify: `mdl/executor/cmd_diff_mdl.go`

- [ ] **Step 1: Replace `entityStmtToMDL` with Lift + ToMDL**

In `cmd_diff_mdl.go`, find `entityStmtToMDL` (line 20). Replace the body:

```go
func entityStmtToMDL(ctx *ExecContext, s *ast.CreateEntityStmt) string {
    m, err := entityModel.Lift(s)
    if err != nil {
        return fmt.Sprintf("/* lift error: %v */", err)
    }
    return m.ToMDL()
}
```

- [ ] **Step 2: Replace `entityToMDLGen` with Hydrate + ToMDL**

In `cmd_diff_mdl.go`, find `entityToMDLGen` (line 535). Replace the body:

```go
func entityToMDLGen(ctx *ExecContext, moduleName string, entity *genDm.Entity) string {
    m, _, err := entityModel.HydrateWithModule(moduleName, entity)
    if err != nil {
        return fmt.Sprintf("/* hydrate error: %v */", err)
    }
    return m.ToMDL()
}
```

Add import:
```go
entityModel "github.com/mendixlabs/mxcli/mdl/model/entity"
```

- [ ] **Step 3: Remove now-unused helper functions**

Check if any of these helpers are now unused: `formatAttributeTypeGen`, `parseLocationBSON` (moved to hydrate.go), `extractAttrNameFromQualified`. Remove any that are no longer referenced. Run:

```bash
go build ./mdl/executor/...
```

Fix any "declared and not used" or "imported and not used" errors.

- [ ] **Step 4: Run tests**

```bash
go test ./mdl/executor/... -count=1
```

Expected: no new failures.

- [ ] **Step 5: Commit**

```bash
git add mdl/executor/cmd_diff_mdl.go
git commit -m "refactor(executor): diff uses single ToMDL() path — eliminates entityStmtToMDL/entityToMDLGen divergence"
```

---

## Task 11: Integration — full round-trip test

**Files:**
- Create: `mdl/model/entity/roundtrip_test.go` (integration tag)

This is the proof that the canonical model layer eliminates drift: parse MDL → Lift → Persist → read back from BSON → Hydrate → ToMDL — the output must match the original MDL.

- [ ] **Step 1: Create `roundtrip_test.go`**

```go
// SPDX-License-Identifier: Apache-2.0

//go:build integration

package entity_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/mendixlabs/mxcli/mdl/model"
	"github.com/mendixlabs/mxcli/mdl/model/entity"
	modelID "github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRoundTrip_EntityCanonicalModel verifies that:
//
//	parse(mdl) → Lift → Persist → BSON → Hydrate → ToMDL == original MDL
//
// This is the primary correctness proof for the canonical model layer.
func TestRoundTrip_EntityCanonicalModel(t *testing.T) {
	cases := []struct {
		name     string
		original *entity.EntityModel
	}{
		{
			name: "persistent entity with constraints",
			original: &entity.EntityModel{
				Name: entity.QualifiedName{Module: "MyFirstModule", Name: "RTTicket"},
				Kind: entity.EntityPersistent,
				Attributes: []entity.AttributeModel{
					{
						Name:         "Title",
						Type:         model.DataType{Kind: model.KindString, Length: 500},
						NotNull:      true,
						NotNullError: "Title required",
					},
					{
						Name:       "Priority",
						Type:       model.DataType{Kind: model.KindInteger},
						HasDefault: true,
						DefaultValue: "1",
					},
				},
			},
		},
		{
			name: "non-persistent entity",
			original: &entity.EntityModel{
				Name: entity.QualifiedName{Module: "MyFirstModule", Name: "RTNonPersistent"},
				Kind: entity.EntityNonPersistent,
				Attributes: []entity.AttributeModel{
					{Name: "Value", Type: model.DataType{Kind: model.KindDecimal}},
				},
			},
		},
	}

	const mprPath = "../../../testdata/expr-checker/minimal.mpr"
	t.Cleanup(func() {
		exec.Command("git", "restore", "../../../testdata/expr-checker/").Run()
	})

	b, err := mpr.OpenForWriting(mprPath)
	require.NoError(t, err)
	defer b.Close()

	dms, err := b.ListDomainModelsGen()
	require.NoError(t, err)
	var dmID modelID.ID
	for _, dm := range dms {
		if dm.ContainerName == "MyFirstModule" {
			dmID = modelID.ID(dm.ID())
			break
		}
	}
	require.NotEmpty(t, dmID)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			originalMDL := tc.original.ToMDL()

			// Persist.
			ctx := model.PersistContext{DomainModelID: dmID, Backend: b}
			require.NoError(t, tc.original.Persist(ctx))
			require.NoError(t, b.Flush())

			// Read back by re-loading the domain model.
			dms2, err := b.ListDomainModelsGen()
			require.NoError(t, err)
			entityName := tc.original.Name.Name
			var readBack *entity.EntityModel
			for _, dm := range dms2 {
				if dm.ContainerName != "MyFirstModule" {
					continue
				}
				for _, item := range dm.EntitiesItems() {
					e, ok := item.(*genDm.Entity)
					if !ok || e.Name() != entityName {
						continue
					}
					readBack, _, err = entity.Hydrate(tc.original.Name.Module, e)
					require.NoError(t, err)
					break
				}
			}
			require.NotNil(t, readBack, "%s not found after persist", entityName)

			// The round-trip must be stable.
			roundTripMDL := readBack.ToMDL()
			assert.Equal(t, originalMDL, roundTripMDL,
				"round-trip MDL differs from original for %s", entityName)
		})
	}
}
```

- [ ] **Step 2: Run integration test**

```bash
go test -tags integration ./mdl/model/entity/... -run TestRoundTrip -v
```

Expected: both sub-tests pass. If there are field differences (e.g. default value format), fix in `Hydrate` or `Persist` until the round-trip is stable.

- [ ] **Step 3: Restore testdata**

```bash
git restore testdata/expr-checker/
```

- [ ] **Step 4: Commit**

```bash
git add mdl/model/entity/roundtrip_test.go
git commit -m "test(model/entity): add full round-trip integration test — proves canonical model eliminates drift"
```

---

## Self-Review Checklist

After completing all tasks, verify:

- [ ] `go build ./...` passes with no errors
- [ ] `go test ./mdl/model/...` (unit) passes
- [ ] `go test -tags integration ./mdl/model/entity/...` passes
- [ ] `go test ./mdl/executor/...` passes (no regressions)
- [ ] `describeEntityGen` no longer contains any `TypeName()` switch for attribute type formatting
- [ ] `entityStmtToMDL` and `entityToMDLGen` in `cmd_diff_mdl.go` are now one-liners delegating to `ToMDL()`
- [ ] `ModelCodecs` is populated in `newExecContext`
- [ ] No circular imports: `mdl/model` must not import `mdl/ast` or `mdl/executor`

---

## Phase 2+ Notes

Phases 2 (association), 3 (microflow), and 4 (page) each repeat Tasks 2–10 with different types:

- **Task 2**: Define `AssociationModel` / `MicroflowModel` / `PageModel` structs
- **Task 3**: `Lift(ast.CreateAssociationStmt)` → model
- **Task 4**: `ToMDL()` for the document type
- **Task 5**: `Hydrate(gen type)` → model
- **Task 6**: `Persist()` → BSON
- **Task 7**: `RegisterCodec` + wire into `ExecContext`
- **Task 8**: Refactor CREATE handler
- **Task 9**: Refactor DESCRIBE handler
- **Task 10**: Unify diff command for that type

Each phase has its own plan document.
