# Association Canonical Model Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `mdl/model/association/` as the canonical model for Mendix associations, following the same Lift/Hydrate/ToMDL/Persist pattern as entity, and route `cmd_diff_mdl.go` association functions through the codec registry.

**Architecture:** `AssociationModel` carries the six MDL-expressible fields (name, from, to, type, owner, delete behavior, storage, documentation). `Lift` converts from `ast.CreateAssociationStmt`; `Hydrate` converts from `genDm.Association` using a `HydrateCtx.EntityNames` map (association BSON stores entity IDs, not names). `ToMDL` emits canonical `create association` MDL. `Persist` uses a local `assocBackend` interface (type-asserted from `PersistContext.Backend`). The codec is registered in `executor.go`; `cmd_diff_mdl.go` routes through `ctx.ModelCodecs`.

**Prerequisite:** `2026-06-02-entity-canonical-completion.md` must be complete (HydrateCtx must exist in `mdl/model/registry.go`).

**Tech Stack:** Go 1.26, `modelsdk/gen/domainmodels`, `mdl/ast`.

---

## File Map

**New files:**

| File | Responsibility |
|------|----------------|
| `mdl/model/association/model.go` | `AssociationModel` struct |
| `mdl/model/association/lift.go` | `Lift(*ast.CreateAssociationStmt) *AssociationModel` |
| `mdl/model/association/hydrate.go` | `Hydrate(ctx HydrateCtx, assoc *genDm.Association) (*AssociationModel, []Warning, error)` |
| `mdl/model/association/serialize.go` | `(*AssociationModel).ToMDL() string` |
| `mdl/model/association/persist.go` | `(*AssociationModel).Persist(ctx PersistContext) error` |
| `mdl/model/association/codec.go` | `RegisterCodec(*model.DefaultRegistry)` |
| `mdl/model/association/comply_test.go` | Compile-time interface assertions |
| `mdl/model/association/association_test.go` | Unit tests: Lift, ToMDL, Hydrate |

**Modified files:**

| File | Change |
|------|--------|
| `mdl/model/codec_guard_test.go` | Add `"DomainModels$Association"` to Required list |
| `mdl/executor/executor.go` | Call `assocmodel.RegisterCodec(mc)` |
| `mdl/executor/cmd_diff_mdl.go` | Route `associationStmtToMDL` + `associationToMDLGen` through ModelCodecs |

**Note on `HydrateCtx.EntityNames`:** `Hydrate` needs a map from entity ID (`string`) to entity name (`string`) because `genDm.Association` stores `ParentRefID` / `ChildRefID` (element IDs), not qualified names. The caller (codec or executor) must populate this map from the domain model before calling Hydrate.

---

## Task 1: `AssociationModel` data type

**Files:**
- Create: `mdl/model/association/model.go`
- Create: `mdl/model/association/comply_test.go`

- [ ] **Step 1: Create `mdl/model/association/model.go`**

```go
// SPDX-License-Identifier: Apache-2.0

// Package association implements the canonical model for Mendix associations.
package association

// AssociationModel is the canonical in-memory representation of a Mendix association.
// It captures exactly what MDL can express — not the complete BSON state.
type AssociationModel struct {
	Name           QualifiedName
	From           QualifiedName // ParentPointer — FK owner entity
	To             QualifiedName // ChildPointer — referenced entity
	Type           AssocType
	Owner          OwnerType
	Storage        StorageType
	DeleteBehavior DeleteBehaviorType
	Documentation  string
}

// QualifiedName is a module-qualified identifier.
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

// AssocType mirrors ast.AssociationType.
type AssocType int

const (
	AssocReference    AssocType = iota
	AssocReferenceSet
)

// OwnerType mirrors ast.OwnerType.
type OwnerType int

const (
	OwnerDefault OwnerType = iota
	OwnerBoth
)

// StorageType mirrors ast.StorageType.
type StorageType int

const (
	StorageTable  StorageType = iota
	StorageColumn
)

// DeleteBehaviorType mirrors ast.DeleteBehavior.
type DeleteBehaviorType int

const (
	DeleteKeepReferences      DeleteBehaviorType = iota
	DeleteCascade
	DeleteBoth
	DeleteKeepParentDeleteChild
	DeleteKeepChildDeleteParent
	DeleteIfNoReferences
)
```

- [ ] **Step 2: Create `mdl/model/association/comply_test.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package association_test

import (
	"github.com/mendixlabs/mxcli/mdl/model"
	"github.com/mendixlabs/mxcli/mdl/model/association"
)

// Compile-time interface assertions.
var _ model.Document    = (*association.AssociationModel)(nil)
var _ model.Persistable = (*association.AssociationModel)(nil)
```

- [ ] **Step 3: Build — expect compile error (missing ToMDL and Persist)**

```bash
go build ./mdl/model/association/... 2>&1 | head -10
```

Expected: compile error because `*AssociationModel` does not implement `Document` or `Persistable`.

- [ ] **Step 4: Commit stub**

```bash
git add mdl/model/association/model.go mdl/model/association/comply_test.go
git commit -m "feat(model/association): add AssociationModel data type"
```

---

## Task 2: `Lift` — AST → AssociationModel (TDD)

**Files:**
- Create: `mdl/model/association/association_test.go`
- Create: `mdl/model/association/lift.go`

- [ ] **Step 1: Write failing tests in `association_test.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package association_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/model/association"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLift_Reference(t *testing.T) {
	stmt := &ast.CreateAssociationStmt{
		Name:           ast.QualifiedName{Module: "M", Name: "Order_Customer"},
		Parent:         ast.QualifiedName{Module: "M", Name: "Order"},
		Child:          ast.QualifiedName{Module: "M", Name: "Customer"},
		Type:           ast.AssocReference,
		Owner:          ast.OwnerDefault,
		Storage:        ast.StorageTable,
		DeleteBehavior: ast.DeleteKeepReferences,
		Documentation:  "Links orders to customers",
	}
	m := association.Lift(stmt)
	assert.Equal(t, "M.Order_Customer", m.Name.String())
	assert.Equal(t, "M.Order", m.From.String())
	assert.Equal(t, "M.Customer", m.To.String())
	assert.Equal(t, association.AssocReference, m.Type)
	assert.Equal(t, association.OwnerDefault, m.Owner)
	assert.Equal(t, association.StorageTable, m.Storage)
	assert.Equal(t, association.DeleteKeepReferences, m.DeleteBehavior)
	assert.Equal(t, "Links orders to customers", m.Documentation)
}

func TestLift_ReferenceSet(t *testing.T) {
	stmt := &ast.CreateAssociationStmt{
		Name:   ast.QualifiedName{Module: "M", Name: "Product_Tag"},
		Parent: ast.QualifiedName{Module: "M", Name: "Product"},
		Child:  ast.QualifiedName{Module: "M", Name: "Tag"},
		Type:   ast.AssocReferenceSet,
		Owner:  ast.OwnerBoth,
	}
	m := association.Lift(stmt)
	assert.Equal(t, association.AssocReferenceSet, m.Type)
	assert.Equal(t, association.OwnerBoth, m.Owner)
}

func TestLift_CascadeDelete(t *testing.T) {
	stmt := &ast.CreateAssociationStmt{
		Name:           ast.QualifiedName{Module: "M", Name: "Order_Line"},
		Parent:         ast.QualifiedName{Module: "M", Name: "Order"},
		Child:          ast.QualifiedName{Module: "M", Name: "OrderLine"},
		Type:           ast.AssocReference,
		DeleteBehavior: ast.DeleteCascade,
	}
	m := association.Lift(stmt)
	assert.Equal(t, association.DeleteCascade, m.DeleteBehavior)
}
```

- [ ] **Step 2: Run tests — expect compile failure**

```bash
go test ./mdl/model/association/... -run TestLift 2>&1 | head -10
```

Expected: `association.Lift undefined`.

- [ ] **Step 3: Create `mdl/model/association/lift.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package association

import "github.com/mendixlabs/mxcli/mdl/ast"

// Lift converts a parsed CREATE ASSOCIATION AST statement to an AssociationModel.
// It is a pure function: no backend access, no side effects.
func Lift(s *ast.CreateAssociationStmt) *AssociationModel {
	return &AssociationModel{
		Name:           QualifiedName{Module: s.Name.Module, Name: s.Name.Name},
		From:           QualifiedName{Module: s.Parent.Module, Name: s.Parent.Name},
		To:             QualifiedName{Module: s.Child.Module, Name: s.Child.Name},
		Type:           liftType(s.Type),
		Owner:          liftOwner(s.Owner),
		Storage:        liftStorage(s.Storage),
		DeleteBehavior: liftDelete(s.DeleteBehavior),
		Documentation:  s.Documentation,
	}
}

func liftType(t ast.AssociationType) AssocType {
	if t == ast.AssocReferenceSet {
		return AssocReferenceSet
	}
	return AssocReference
}

func liftOwner(o ast.OwnerType) OwnerType {
	if o == ast.OwnerBoth {
		return OwnerBoth
	}
	return OwnerDefault
}

func liftStorage(s ast.StorageType) StorageType {
	if s == ast.StorageColumn {
		return StorageColumn
	}
	return StorageTable
}

func liftDelete(d ast.DeleteBehavior) DeleteBehaviorType {
	switch d {
	case ast.DeleteCascade:
		return DeleteCascade
	case ast.DeleteBoth:
		return DeleteBoth
	case ast.DeleteKeepParentDeleteChild:
		return DeleteKeepParentDeleteChild
	case ast.DeleteKeepChildDeleteParent:
		return DeleteKeepChildDeleteParent
	case ast.DeleteIfNoReferences:
		return DeleteIfNoReferences
	default:
		return DeleteKeepReferences
	}
}
```

- [ ] **Step 4: Run tests — expect pass**

```bash
go test ./mdl/model/association/... -run TestLift -v
```

Expected: all `TestLift_*` tests PASS.

- [ ] **Step 5: Commit**

```bash
git add mdl/model/association/lift.go mdl/model/association/association_test.go
git commit -m "feat(model/association): implement Lift() — AST to AssociationModel (TDD)"
```

---

## Task 3: `ToMDL` — AssociationModel → MDL text (TDD)

**Files:**
- Modify: `mdl/model/association/association_test.go`
- Create: `mdl/model/association/serialize.go`

- [ ] **Step 1: Add ToMDL tests to `association_test.go`**

Append:

```go
func TestToMDL_Reference(t *testing.T) {
	m := &association.AssociationModel{
		Name:           association.QualifiedName{Module: "M", Name: "Order_Customer"},
		From:           association.QualifiedName{Module: "M", Name: "Order"},
		To:             association.QualifiedName{Module: "M", Name: "Customer"},
		Type:           association.AssocReference,
		Owner:          association.OwnerDefault,
		Storage:        association.StorageTable,
		DeleteBehavior: association.DeleteKeepReferences,
	}
	want := "create association M.Order_Customer\nfrom M.Order to M.Customer\ntype Reference\nowner Default\ndelete behavior DeleteMeButKeepReferences"
	assert.Equal(t, want, m.ToMDL())
}

func TestToMDL_ReferenceSetWithDoc(t *testing.T) {
	m := &association.AssociationModel{
		Name:          association.QualifiedName{Module: "M", Name: "Product_Tag"},
		From:          association.QualifiedName{Module: "M", Name: "Product"},
		To:            association.QualifiedName{Module: "M", Name: "Tag"},
		Type:          association.AssocReferenceSet,
		Owner:         association.OwnerBoth,
		Storage:       association.StorageTable,
		Documentation: "Product tags",
	}
	got := m.ToMDL()
	assert.Contains(t, got, "/**\n * Product tags\n */\n")
	assert.Contains(t, got, "type ReferenceSet")
	assert.Contains(t, got, "owner Both")
}

func TestToMDL_CascadeDelete(t *testing.T) {
	m := &association.AssociationModel{
		Name:           association.QualifiedName{Module: "M", Name: "Order_Line"},
		From:           association.QualifiedName{Module: "M", Name: "Order"},
		To:             association.QualifiedName{Module: "M", Name: "OrderLine"},
		Type:           association.AssocReference,
		DeleteBehavior: association.DeleteCascade,
	}
	assert.Contains(t, m.ToMDL(), "delete behavior DeleteMeAndReferences")
}

// TestToMDL_LiftRoundTrip verifies Lift→ToMDL is deterministic.
func TestToMDL_LiftRoundTrip(t *testing.T) {
	stmt := &ast.CreateAssociationStmt{
		Name:   ast.QualifiedName{Module: "M", Name: "A_B"},
		Parent: ast.QualifiedName{Module: "M", Name: "A"},
		Child:  ast.QualifiedName{Module: "M", Name: "B"},
		Type:   ast.AssocReference,
	}
	m := association.Lift(stmt)
	got := m.ToMDL()
	assert.Contains(t, got, "create association M.A_B")
	assert.Contains(t, got, "from M.A to M.B")
}
```

- [ ] **Step 2: Run — expect compile failure**

```bash
go test ./mdl/model/association/... -run TestToMDL 2>&1 | head -10
```

Expected: `(*AssociationModel).ToMDL undefined`.

- [ ] **Step 3: Create `mdl/model/association/serialize.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package association

import (
	"fmt"
	"strings"
)

// ToMDL renders the canonical association as deterministic MDL text.
func (m *AssociationModel) ToMDL() string {
	var sb strings.Builder
	if m.Documentation != "" {
		fmt.Fprintf(&sb, "/**\n * %s\n */\n", m.Documentation)
	}
	fmt.Fprintf(&sb, "create association %s\n", m.Name)
	fmt.Fprintf(&sb, "from %s to %s\n", m.From, m.To)
	fmt.Fprintf(&sb, "type %s\n", assocTypeStr(m.Type))
	fmt.Fprintf(&sb, "owner %s\n", ownerStr(m.Owner))
	fmt.Fprintf(&sb, "delete behavior %s", deleteBehaviorStr(m.DeleteBehavior))
	return sb.String()
}

func assocTypeStr(t AssocType) string {
	if t == AssocReferenceSet {
		return "ReferenceSet"
	}
	return "Reference"
}

func ownerStr(o OwnerType) string {
	if o == OwnerBoth {
		return "Both"
	}
	return "Default"
}

func deleteBehaviorStr(d DeleteBehaviorType) string {
	switch d {
	case DeleteCascade:
		return "DeleteMeAndReferences"
	case DeleteBoth:
		return "DeleteBoth"
	case DeleteKeepParentDeleteChild:
		return "KeepParentDeleteChild"
	case DeleteKeepChildDeleteParent:
		return "KeepChildDeleteParent"
	case DeleteIfNoReferences:
		return "DeleteIfNoReferences"
	default:
		return "DeleteMeButKeepReferences"
	}
}
```

- [ ] **Step 4: Run tests — expect pass**

```bash
go test ./mdl/model/association/... -run TestToMDL -v
```

- [ ] **Step 5: Commit**

```bash
git add mdl/model/association/serialize.go mdl/model/association/association_test.go
git commit -m "feat(model/association): implement ToMDL() serializer (TDD)"
```

---

## Task 4: `Hydrate` — gen.Association → AssociationModel (TDD)

**Files:**
- Modify: `mdl/model/association/association_test.go`
- Create: `mdl/model/association/hydrate.go`

- [ ] **Step 1: Add Hydrate tests to `association_test.go`**

Add to imports: `genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"` and `"github.com/mendixlabs/mxcli/mdl/model"`.

Append:

```go
func TestHydrate_Reference(t *testing.T) {
	a := genDm.NewAssociation()
	a.SetName("Order_Customer")
	a.SetParentID("entity-id-order")
	a.SetChildID("entity-id-customer")
	a.SetType("Reference")
	a.SetOwner("Default")
	a.SetStorageFormat("AssociationStorageCrossTable")
	a.SetDocumentation("Order to customer link")

	ctx := model.HydrateCtx{
		ModuleName: "M",
		EntityNames: map[string]string{
			"entity-id-order":    "Order",
			"entity-id-customer": "Customer",
		},
	}
	m, warns, err := association.Hydrate(ctx, a)
	require.NoError(t, err)
	assert.Empty(t, warns)
	assert.Equal(t, "M.Order_Customer", m.Name.String())
	assert.Equal(t, "M.Order", m.From.String())
	assert.Equal(t, "M.Customer", m.To.String())
	assert.Equal(t, association.AssocReference, m.Type)
	assert.Equal(t, "Order to customer link", m.Documentation)
}

func TestHydrate_UnknownEntityID_ReturnsWarning(t *testing.T) {
	a := genDm.NewAssociation()
	a.SetName("X_Y")
	a.SetParentID("unknown-id-1")
	a.SetChildID("unknown-id-2")
	a.SetType("Reference")

	ctx := model.HydrateCtx{ModuleName: "M", EntityNames: map[string]string{}}
	m, warns, err := association.Hydrate(ctx, a)
	require.NoError(t, err)
	assert.NotEmpty(t, warns)
	// Falls back to ID string when name not found.
	assert.Equal(t, "unknown-id-1", m.From.Name)
}
```

- [ ] **Step 2: Run — expect compile failure**

```bash
go test ./mdl/model/association/... -run TestHydrate 2>&1 | head -10
```

- [ ] **Step 3: Create `mdl/model/association/hydrate.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package association

import (
	"github.com/mendixlabs/mxcli/mdl/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// Hydrate converts a gen-typed Association to an AssociationModel.
// ctx.ModuleName is the owning module (gen entity does not carry it).
// ctx.EntityNames maps element ID strings to simple entity names;
// when an ID is missing, the ID itself is used as a fallback and a Warning
// is emitted so callers can log the gap.
func Hydrate(ctx model.HydrateCtx, a *genDm.Association) (*AssociationModel, []model.Warning, error) {
	var warns []model.Warning

	fromName := resolveEntityName(ctx, string(a.ParentRefID()), &warns)
	toName := resolveEntityName(ctx, string(a.ChildRefID()), &warns)

	m := &AssociationModel{
		Name:           QualifiedName{Module: ctx.ModuleName, Name: a.Name()},
		From:           QualifiedName{Module: ctx.ModuleName, Name: fromName},
		To:             QualifiedName{Module: ctx.ModuleName, Name: toName},
		Type:           hydrateType(a.Type()),
		Owner:          hydrateOwner(a.Owner()),
		Storage:        hydrateStorage(a.StorageFormat()),
		DeleteBehavior: hydrateDelete(a),
		Documentation:  a.Documentation(),
	}
	return m, warns, nil
}

func resolveEntityName(ctx model.HydrateCtx, id string, warns *[]model.Warning) string {
	if name, ok := ctx.EntityNames[id]; ok {
		return name
	}
	*warns = append(*warns, model.Warning{
		Field:   "EntityID",
		Message: "entity ID " + id + " not found in EntityNames map; using ID as fallback",
	})
	return id
}

func hydrateType(t string) AssocType {
	if t == "ReferenceSet" {
		return AssocReferenceSet
	}
	return AssocReference
}

func hydrateOwner(o string) OwnerType {
	if o == "Both" {
		return OwnerBoth
	}
	return OwnerDefault
}

func hydrateStorage(s string) StorageType {
	if s == "AssociationStorageColumn" {
		return StorageColumn
	}
	return StorageTable
}

func hydrateDelete(a *genDm.Association) DeleteBehaviorType {
	dbe := a.DeleteBehavior()
	if dbe == nil {
		return DeleteKeepReferences
	}
	db, ok := dbe.(*genDm.AssociationDeleteBehavior)
	if !ok {
		return DeleteKeepReferences
	}
	switch db.ChildDeleteBehavior() {
	case "DeleteMeAndReferences":
		return DeleteCascade
	case "DeleteBoth":
		return DeleteBoth
	case "KeepParentDeleteChild":
		return DeleteKeepParentDeleteChild
	case "KeepChildDeleteParent":
		return DeleteKeepChildDeleteParent
	case "DeleteIfNoReferences":
		return DeleteIfNoReferences
	default:
		return DeleteKeepReferences
	}
}
```

- [ ] **Step 4: Run tests — expect pass**

```bash
go test ./mdl/model/association/... -run TestHydrate -v
```

- [ ] **Step 5: Commit**

```bash
git add mdl/model/association/hydrate.go mdl/model/association/association_test.go
git commit -m "feat(model/association): implement Hydrate() — gen Association to AssociationModel (TDD)"
```

---

## Task 5: `Persist` — AssociationModel → BSON

**Files:**
- Create: `mdl/model/association/persist.go`

- [ ] **Step 1: Create `mdl/model/association/persist.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package association

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	mxID "github.com/mendixlabs/mxcli/model"
)

// Persist writes the AssociationModel to the project via ctx.Backend.
// ctx.Backend must satisfy assocBackend (satisfied by backend.DomainModelBackend).
// ctx.EntityNames must map qualified entity names (e.g. "M.Order") to element IDs
// so the FROM and TO pointers can be resolved.
//
// This Persist uses a slightly extended PersistContext pattern: pass entity IDs
// via the Backend's entity lookup, or pre-populate via the caller resolving them.
func (m *AssociationModel) Persist(ctx model.PersistContext) error {
	type assocBackend interface {
		CreateAssociationGen(domainModelID mxID.ID, assoc *genDm.Association) error
		GetEntityIDByQualifiedName(qualifiedName string) (element.ID, error)
	}
	b, ok := ctx.Backend.(assocBackend)
	if !ok {
		return fmt.Errorf("association.Persist: backend %T does not implement assocBackend", ctx.Backend)
	}

	fromID, err := b.GetEntityIDByQualifiedName(m.From.String())
	if err != nil {
		return fmt.Errorf("association.Persist: resolve FROM entity %s: %w", m.From, err)
	}
	toID, err := b.GetEntityIDByQualifiedName(m.To.String())
	if err != nil {
		return fmt.Errorf("association.Persist: resolve TO entity %s: %w", m.To, err)
	}

	a := genDm.NewAssociation()
	a.SetName(m.Name.Name)
	if m.Documentation != "" {
		a.SetDocumentation(m.Documentation)
	}
	a.SetParentID(fromID)
	a.SetChildID(toID)
	a.SetType(assocTypeToGen(m.Type))
	a.SetOwner(ownerToGen(m.Owner))
	a.SetStorageFormat(storageToGen(m.Storage))
	a.SetDeleteBehavior(deleteBehaviorToGen(m.DeleteBehavior))

	return b.CreateAssociationGen(ctx.DomainModelID, a)
}

func assocTypeToGen(t AssocType) string {
	if t == AssocReferenceSet {
		return "ReferenceSet"
	}
	return "Reference"
}

func ownerToGen(o OwnerType) string {
	if o == OwnerBoth {
		return "Both"
	}
	return "Default"
}

func storageToGen(s StorageType) string {
	if s == StorageColumn {
		return "AssociationStorageColumn"
	}
	return "AssociationStorageCrossTable"
}

func deleteBehaviorToGen(d DeleteBehaviorType) element.Element {
	db := genDm.NewAssociationDeleteBehavior()
	switch d {
	case DeleteCascade:
		db.SetChildDeleteBehavior("DeleteMeAndReferences")
		db.SetParentDeleteBehavior("Nothing")
	case DeleteBoth:
		db.SetChildDeleteBehavior("DeleteBoth")
		db.SetParentDeleteBehavior("DeleteBoth")
	case DeleteKeepParentDeleteChild:
		db.SetChildDeleteBehavior("KeepParentDeleteChild")
		db.SetParentDeleteBehavior("Nothing")
	case DeleteKeepChildDeleteParent:
		db.SetChildDeleteBehavior("KeepChildDeleteParent")
		db.SetParentDeleteBehavior("Nothing")
	case DeleteIfNoReferences:
		db.SetChildDeleteBehavior("DeleteIfNoReferences")
		db.SetParentDeleteBehavior("Nothing")
	default:
		db.SetChildDeleteBehavior("DeleteMeButKeepReferences")
		db.SetParentDeleteBehavior("Nothing")
	}
	return db
}
```

**Note:** `GetEntityIDByQualifiedName` must be added to `backend.DomainModelBackend` and implemented in `mdl/backend/mpr/`. Check if it already exists; if not, add it following the backend abstraction pattern (interface method + mpr implementation + mock stub).

- [ ] **Step 2: Build**

```bash
go build ./mdl/model/association/...
```

Fix any compile errors (missing methods on genDm types, etc.).

- [ ] **Step 3: Commit**

```bash
git add mdl/model/association/persist.go
git commit -m "feat(model/association): implement Persist() — AssociationModel to BSON"
```

---

## Task 6: Codec + wire into registry

**Files:**
- Create: `mdl/model/association/codec.go`
- Modify: `mdl/model/codec_guard_test.go`
- Modify: `mdl/executor/executor.go`

- [ ] **Step 1: Create `mdl/model/association/codec.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package association

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// RegisterCodec wires the association Lift / Hydrate codecs into r.
func RegisterCodec(r *model.DefaultRegistry) {
	codec := model.Codec{
		LiftFn: func(stmt any) (model.Persistable, error) {
			s, ok := stmt.(*ast.CreateAssociationStmt)
			if !ok {
				return nil, fmt.Errorf("association codec: expected *ast.CreateAssociationStmt, got %T", stmt)
			}
			return Lift(s), nil
		},
		HydrateFn: func(el any, hctx model.HydrateCtx) (model.Document, []model.Warning, error) {
			a, ok := el.(*genDm.Association)
			if !ok {
				return nil, nil, fmt.Errorf("association codec: expected *genDm.Association, got %T", el)
			}
			return Hydrate(hctx, a)
		},
	}
	r.Register((*ast.CreateAssociationStmt)(nil), "DomainModels$Association", codec)
	r.RegisterGenType("DomainModels$AssociationImpl", codec)
}
```

- [ ] **Step 2: Update `mdl/model/codec_guard_test.go` — add Required TypeName**

In `TestCodecComplete`, update `BuildRegistry` and `Required`:

```go
BuildRegistry: func() *model.DefaultRegistry {
    r := model.NewDefaultRegistry()
    entity.RegisterCodec(r)
    association.RegisterCodec(r)  // ← add
    return r
},
Required: []string{
    "DomainModels$Entity",
    "DomainModels$EntityImpl",
    "DomainModels$Association",  // ← add
},
```

Add import: `assocmodel "github.com/mendixlabs/mxcli/mdl/model/association"` and use `assocmodel.RegisterCodec(r)`.

- [ ] **Step 3: Update `mdl/executor/executor.go` — register association codec**

After `entitymodel.RegisterCodec(mc)`, add:

```go
assocmodel.RegisterCodec(mc)
```

Add import: `assocmodel "github.com/mendixlabs/mxcli/mdl/model/association"`.

- [ ] **Step 4: Verify Guard 3 still GREEN**

```bash
go test ./mdl/model/ -run TestCodecComplete -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mdl/model/association/codec.go mdl/model/codec_guard_test.go \
        mdl/executor/executor.go
git commit -m "feat(model/association): register codec, wire into executor"
```

---

## Task 7: Route `cmd_diff_mdl.go` through ModelCodecs

**Files:**
- Modify: `mdl/executor/cmd_diff_mdl.go`

- [ ] **Step 1: Replace `associationStmtToMDL` body**

Find `associationStmtToMDL` (around line 93). Replace body:

```go
func associationStmtToMDL(ctx *ExecContext, s *ast.CreateAssociationStmt) string {
	doc, err := ctx.ModelCodecs.LiftFrom(s)
	if err != nil {
		return fmt.Sprintf("/* association lift error: %v */", err)
	}
	return doc.ToMDL() + ";\n/"
}
```

- [ ] **Step 2: Replace `associationToMDLGen` body**

Find `associationToMDLGen` (around line 551). The current version builds a local entity name map. Replace:

```go
func associationToMDLGen(ctx *ExecContext, moduleName string, assoc *genDm.Association, dm *genDm.DomainModel) string {
	// Build entity ID → name map from the domain model.
	entityNames := make(map[string]string)
	for _, item := range dm.EntitiesItems() {
		if e, ok := item.(*genDm.Entity); ok {
			entityNames[string(e.ID())] = e.Name()
		}
	}
	doc, _, err := ctx.ModelCodecs.HydrateFrom(assoc, canonicalmodel.HydrateCtx{
		ModuleName:  moduleName,
		EntityNames: entityNames,
	})
	if err != nil {
		return fmt.Sprintf("/* association hydrate error: %v */", err)
	}
	return doc.ToMDL() + ";\n/"
}
```

Ensure `canonicalmodel "github.com/mendixlabs/mxcli/mdl/model"` is imported.

- [ ] **Step 3: Build and verify Guard 4**

```bash
go build ./mdl/executor/...
go test ./mdl/executor/ -run TestExecutorBoundary -v
```

Expected: PASS (no new association imports in non-allowlisted files).

- [ ] **Step 4: Run full tests**

```bash
go test ./mdl/model/... ./mdl/executor/... -count=1 2>&1 | grep -E "^FAIL|^ok" | head -20
```

- [ ] **Step 5: Commit**

```bash
git add mdl/executor/cmd_diff_mdl.go
git commit -m "refactor(executor): association diff uses canonical AssociationModel via registry"
```

---

## Self-Review

| Requirement | Task |
|-------------|------|
| AssociationModel data type | 1 |
| comply_test.go (interface assertions) | 1 |
| Lift AST → AssociationModel | 2 |
| ToMDL canonical MDL output | 3 |
| Hydrate gen → AssociationModel with entity name resolution | 4 |
| Persist via assocBackend interface | 5 |
| Codec registration | 6 |
| Guard 3 updated (DomainModels$Association required) | 6 |
| cmd_diff_mdl routes through ModelCodecs | 7 |
| Guard 4 stays GREEN | 7 |
