# Entity Canonical Model — Completion Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete the entity canonical model so `EntityModel` is the single serialization center — eliminating the executor trailing-clause append pattern, the dual-path in CREATE ENTITY, and both red architectural guards.

**Architecture:** Four sequential tasks: (1) HydrateCtx + PersistContext.Backend fix → Guards 1 & 4 green; (2) EventHandlerModel in EntityModel; (3) Persist event handlers + eliminate dual-path; (4) OQL + system members into ToMDLStatement. After task 4, `describeEntityGen` emits nothing after `m.ToMDLStatement(true)`.

**Tech Stack:** Go 1.26, `modelsdk/gen/domainmodels`, `mdl/ast`, `mdl/backend` (behind interface in entity layer).

**Guards:** `mdl/model/import_guard_test.go` (Guard 1) and `mdl/executor/boundary_guard_test.go` (Guard 4) must be GREEN after Task 1. All other guards must stay GREEN throughout.

---

## File Map

**Modified files:**

| File | Change |
|------|--------|
| `mdl/model/registry.go` | Add `HydrateCtx`, update `Codec.HydrateFn` + `HydrateFrom` |
| `mdl/model/context.go` | `Backend any` (remove `mdl/backend` import) |
| `mdl/model/entity/persist.go` | Type-assert `ctx.Backend` to local `entityBackend` interface |
| `mdl/model/entity/codec.go` | `HydrateFn` uses `hctx.ModuleName` |
| `mdl/model/entity/model.go` | Add `EventHandlerModel`, `EventHandlers []EventHandlerModel`, `OQL string` |
| `mdl/model/entity/lift.go` | Extract `s.EventHandlers` into `[]EventHandlerModel` |
| `mdl/model/entity/hydrate.go` | Extract event handlers + OQL from gen entity |
| `mdl/model/entity/serialize.go` | Emit system members, event handlers, OQL in `ToMDLStatement` |
| `mdl/model/entity/entity_test.go` | Tests for event handler Lift/Hydrate/ToMDL |
| `internal/archtest/codec_complete_test.go` | Update `buildTestRegistry` HydrateFn signature |
| `mdl/executor/executor.go` | Add `hydrateEntityModel` helper (imports entity, allowlisted) |
| `mdl/executor/cmd_entities_gen.go` | Remove entity import; use helper; remove trailing clauses |
| `mdl/executor/cmd_diff_mdl.go` | Remove entity import; route through `ctx.ModelCodecs` |
| `mdl/executor/cmd_create_entity_gen.go` | Remove `len(s.EventHandlers) == 0` gate |

---

## Task 1: HydrateCtx + PersistContext.Backend fix

This task makes Guard 1 and Guard 4 GREEN.

**Files:**
- Modify: `mdl/model/registry.go`
- Modify: `mdl/model/context.go`
- Modify: `mdl/model/entity/persist.go`
- Modify: `mdl/model/entity/codec.go`
- Modify: `internal/archtest/codec_complete_test.go`
- Modify: `mdl/executor/executor.go`
- Modify: `mdl/executor/cmd_entities_gen.go`
- Modify: `mdl/executor/cmd_diff_mdl.go`

- [ ] **Step 1: Add `HydrateCtx` to `mdl/model/registry.go` and update `Codec` + `HydrateFrom`**

In `registry.go`, add after the `Codec` struct definition and update `HydrateFn` + `HydrateFrom`:

```go
// HydrateCtx carries contextual information not stored in the gen element
// itself (e.g. the owning module name, which entities do not store in BSON).
type HydrateCtx struct {
	ModuleName  string
	EntityNames map[string]string // entity ID → name; used by association Hydrate
}

// Codec wires Lift (AST -> canonical model) and Hydrate (gen-typed element ->
// canonical model) for a single document kind.
type Codec struct {
	LiftFn    func(stmt any) (Persistable, error)
	HydrateFn func(el any, ctx HydrateCtx) (Document, []Warning, error)
}
```

Replace the `HydrateFrom` method:

```go
// HydrateFrom dispatches a gen-typed element to the registered Hydrate codec.
func (r *DefaultRegistry) HydrateFrom(el any, ctx HydrateCtx) (Document, []Warning, error) {
	gt, ok := el.(genTyper)
	if !ok {
		return nil, nil, fmt.Errorf("model: %T does not implement TypeName()", el)
	}
	c, ok := r.byGenType[gt.TypeName()]
	if !ok {
		return nil, nil, fmt.Errorf("model: no codec for gen type %q", gt.TypeName())
	}
	return c.HydrateFn(el, ctx)
}
```

- [ ] **Step 2: Fix `mdl/model/context.go` — remove backend import**

Replace the entire file:

```go
// SPDX-License-Identifier: Apache-2.0

package model

import modelID "github.com/mendixlabs/mxcli/model"

// PersistContext carries the project-scoped identifiers and backend handle a
// canonical document needs to write itself to the project.
// Backend is typed as any to keep this package free of backend imports;
// each domain's persist.go defines a local interface and type-asserts it.
type PersistContext struct {
	DomainModelID    modelID.ID
	ExistingEntityID modelID.ID
	Backend          any
}
```

- [ ] **Step 3: Fix `mdl/model/entity/persist.go` — type-assert Backend**

Replace the `Persist` method (lines 33-52) with:

```go
func (m *EntityModel) Persist(ctx model.PersistContext) error {
	if m == nil {
		return fmt.Errorf("entity.Persist: nil model")
	}
	if ctx.Backend == nil {
		return fmt.Errorf("entity.Persist: nil backend")
	}
	if ctx.DomainModelID == "" {
		return fmt.Errorf("entity.Persist: missing DomainModelID")
	}
	type entityBackend interface {
		CreateEntityGen(domainModelID mxID.ID, entity *genDm.Entity) error
		UpdateEntityGen(domainModelID mxID.ID, entity *genDm.Entity) error
	}
	b, ok := ctx.Backend.(entityBackend)
	if !ok {
		return fmt.Errorf("entity.Persist: backend %T does not implement entityBackend", ctx.Backend)
	}
	gen, err := buildGenEntity(m)
	if err != nil {
		return fmt.Errorf("entity.Persist: %w", err)
	}
	if ctx.ExistingEntityID != "" {
		gen.SetID(element.ID(ctx.ExistingEntityID))
		return b.UpdateEntityGen(ctx.DomainModelID, gen)
	}
	return b.CreateEntityGen(ctx.DomainModelID, gen)
}
```

Add to the imports block: `mxID "github.com/mendixlabs/mxcli/model"`.

- [ ] **Step 4: Fix `mdl/model/entity/codec.go` — use `HydrateCtx.ModuleName`**

Replace `HydrateFn` in `RegisterCodec`:

```go
HydrateFn: func(el any, hctx model.HydrateCtx) (model.Document, []model.Warning, error) {
    e, ok := el.(*genDm.Entity)
    if !ok {
        return nil, nil, fmt.Errorf("entity codec: expected *genDm.Entity, got %T", el)
    }
    return Hydrate(hctx.ModuleName, e)
},
```

- [ ] **Step 5: Fix `internal/archtest/codec_complete_test.go` — update HydrateFn signature**

Replace `buildTestRegistry`:

```go
func buildTestRegistry(
	liftFn func(any) (model.Persistable, error),
	hydrateFn func(any, model.HydrateCtx) (model.Document, []model.Warning, error),
) *model.DefaultRegistry {
	r := model.NewDefaultRegistry()
	r.RegisterGenType("TestType$Foo", model.Codec{
		LiftFn:    liftFn,
		HydrateFn: hydrateFn,
	})
	return r
}
```

Update each `buildTestRegistry` call to use the new signature:

```go
// TestCodecComplete_allPresent_passes:
r := buildTestRegistry(
    func(any) (model.Persistable, error) { return nil, nil },
    func(any, model.HydrateCtx) (model.Document, []model.Warning, error) { return nil, nil, nil },
)

// TestCodecComplete_nilLiftFn_fails:
r := buildTestRegistry(
    nil,
    func(any, model.HydrateCtx) (model.Document, []model.Warning, error) { return nil, nil, nil },
)

// TestCodecComplete_nilHydrateFn_fails:
r := buildTestRegistry(
    func(any) (model.Persistable, error) { return nil, nil },
    nil,
)
```

- [ ] **Step 6: Add `hydrateEntityModel` helper to `mdl/executor/executor.go`**

`executor.go` already imports `entitymodel "github.com/mendixlabs/mxcli/mdl/model/entity"`. Add this function after the existing `RegisterCodec` call site:

```go
// hydrateEntityModel hydrates a gen-typed entity through the codec registry
// and type-asserts the result to *entity.EntityModel. executor.go is the
// only executor file allowed to import mdl/model/entity directly (see Guard 4).
func hydrateEntityModel(ctx *ExecContext, moduleName string, genEnt any) (*entitymodel.EntityModel, []canonicalmodel.Warning, error) {
	doc, warns, err := ctx.ModelCodecs.HydrateFrom(genEnt, canonicalmodel.HydrateCtx{ModuleName: moduleName})
	if err != nil {
		return nil, nil, err
	}
	m, ok := doc.(*entitymodel.EntityModel)
	if !ok {
		return nil, warns, fmt.Errorf("hydrateEntityModel: unexpected type %T", doc)
	}
	return m, warns, nil
}
```

Add import `canonicalmodel "github.com/mendixlabs/mxcli/mdl/model"` if not already present.

- [ ] **Step 7: Fix `mdl/executor/cmd_entities_gen.go` — remove entity import, use helper**

Remove the import line:
```go
entityModel "github.com/mendixlabs/mxcli/mdl/model/entity"
```

In `describeEntityGen`, replace:
```go
m, warns, err := entityModel.Hydrate(modName, entity)
```
with:
```go
m, warns, err := hydrateEntityModel(ctx, modName, entity)
```

- [ ] **Step 8: Fix `mdl/executor/cmd_diff_mdl.go` — remove entity import, use ModelCodecs**

Remove:
```go
entityModel "github.com/mendixlabs/mxcli/mdl/model/entity"
```

Add if not present:
```go
canonicalmodel "github.com/mendixlabs/mxcli/mdl/model"
```

Replace `entityStmtToMDL` body (currently calls `entityModel.Lift`):
```go
func entityStmtToMDL(ctx *ExecContext, s *ast.CreateEntityStmt) string {
	doc, err := ctx.ModelCodecs.LiftFrom(s)
	if err != nil {
		return fmt.Sprintf("/* entity lift error: %v */", err)
	}
	return doc.ToMDL() + ";\n/"
}
```

Replace `entityToMDLGen` body (currently calls `entityModel.Hydrate`):
```go
func entityToMDLGen(ctx *ExecContext, moduleName string, entity *genDm.Entity) string {
	doc, _, err := ctx.ModelCodecs.HydrateFrom(entity, canonicalmodel.HydrateCtx{ModuleName: moduleName})
	if err != nil {
		return fmt.Sprintf("/* entity hydrate error: %v */", err)
	}
	return doc.ToMDL() + ";\n/"
}
```

- [ ] **Step 9: Build and verify guards GREEN**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go build ./...
go test ./mdl/model/ -run TestImportDirection -v
go test ./mdl/executor/ -run TestExecutorBoundary -v
```

Expected: both tests PASS. Also verify Guard 3 still PASS:
```bash
go test ./mdl/model/ -run TestCodecComplete -v
```

- [ ] **Step 10: Commit**

```bash
git add mdl/model/registry.go mdl/model/context.go \
        mdl/model/entity/persist.go mdl/model/entity/codec.go \
        internal/archtest/codec_complete_test.go \
        mdl/executor/executor.go mdl/executor/cmd_entities_gen.go \
        mdl/executor/cmd_diff_mdl.go
git commit -m "feat(model): HydrateCtx + PersistContext.Backend fix — Guards 1 & 4 green"
```

---

## Task 2: EventHandlerModel in EntityModel (TDD)

**Files:**
- Modify: `mdl/model/entity/model.go`
- Modify: `mdl/model/entity/entity_test.go`
- Modify: `mdl/model/entity/lift.go`
- Modify: `mdl/model/entity/hydrate.go`

- [ ] **Step 1: Add `EventHandlerModel` to `mdl/model/entity/model.go`**

In model.go, add after `IndexColumn`:

```go
// EventHandlerModel is the canonical representation of a domain event handler.
// Moment is "Before" or "After"; Event is "Create", "Commit", "Delete", or "Rollback".
type EventHandlerModel struct {
	Moment            string
	Event             string
	Microflow         QualifiedName
	RaiseErrorOnFalse bool
	PassEventObject   bool
}
```

Add `EventHandlers []EventHandlerModel` field to `EntityModel` after `SystemMembers`:

```go
EventHandlers []EventHandlerModel
```

- [ ] **Step 2: Write failing tests in `entity_test.go`**

Append to `entity_test.go`:

```go
func TestLift_WithEventHandler(t *testing.T) {
	stmt := &ast.CreateEntityStmt{
		Name: ast.QualifiedName{Module: "M", Name: "E"},
		Kind: ast.EntityPersistent,
		EventHandlers: []ast.EventHandlerDef{
			{
				Moment:            "Before",
				Event:             "Commit",
				Microflow:         ast.QualifiedName{Module: "M", Name: "BeforeCommit"},
				RaiseErrorOnFalse: true,
				PassEventObject:   true,
			},
		},
	}
	m, err := entity.Lift(stmt)
	require.NoError(t, err)
	require.Len(t, m.EventHandlers, 1)
	eh := m.EventHandlers[0]
	assert.Equal(t, "Before", eh.Moment)
	assert.Equal(t, "Commit", eh.Event)
	assert.Equal(t, "M.BeforeCommit", eh.Microflow.String())
	assert.True(t, eh.RaiseErrorOnFalse)
	assert.True(t, eh.PassEventObject)
}

func TestToMDL_WithEventHandler(t *testing.T) {
	m := &entity.EntityModel{
		Name: entity.QualifiedName{Module: "M", Name: "E"},
		Kind: entity.EntityPersistent,
		EventHandlers: []entity.EventHandlerModel{
			{Moment: "Before", Event: "Commit", Microflow: entity.QualifiedName{Module: "M", Name: "ACT_BeforeCommit"}, RaiseErrorOnFalse: true, PassEventObject: true},
		},
	}
	got := m.ToMDLStatement(true)
	assert.Contains(t, got, "on before commit call M.ACT_BeforeCommit($currentObject) raise error")
}

func TestToMDL_EventHandler_AfterCreate_NoPassObject(t *testing.T) {
	m := &entity.EntityModel{
		Name: entity.QualifiedName{Module: "M", Name: "E"},
		Kind: entity.EntityPersistent,
		EventHandlers: []entity.EventHandlerModel{
			{Moment: "After", Event: "Create", Microflow: entity.QualifiedName{Module: "M", Name: "ACT_After"}, PassEventObject: false},
		},
	}
	got := m.ToMDLStatement(true)
	assert.Contains(t, got, "on after create call M.ACT_After()")
	assert.NotContains(t, got, "$currentObject")
	assert.NotContains(t, got, "raise error")
}
```

- [ ] **Step 3: Run tests — expect failures**

```bash
go test ./mdl/model/entity/... -run "TestLift_WithEvent|TestToMDL_WithEvent|TestToMDL_Event" -v 2>&1 | head -20
```

Expected: failures (event handlers not Lifted or emitted).

- [ ] **Step 4: Update `entity/lift.go` — extract EventHandlers**

In the `Lift` function, after the loop that lifts Indexes, add:

```go
for _, eh := range s.EventHandlers {
    m.EventHandlers = append(m.EventHandlers, EventHandlerModel{
        Moment:            eh.Moment,
        Event:             eh.Event,
        Microflow:         QualifiedName{Module: eh.Microflow.Module, Name: eh.Microflow.Name},
        RaiseErrorOnFalse: eh.RaiseErrorOnFalse,
        PassEventObject:   eh.PassEventObject,
    })
}
```

- [ ] **Step 5: Update `entity/hydrate.go` — extract EventHandlers**

In the `Hydrate` function, after the loop that hydrates Indexes, add:

```go
for _, item := range e.EventHandlersItems() {
    h, ok := item.(*genDm.EventHandler)
    if !ok {
        warns = append(warns, model.Warning{Field: "EventHandlers", Message: "unexpected item type"})
        continue
    }
    if h.MicroflowQualifiedName() == "" {
        continue
    }
    parts := strings.SplitN(h.MicroflowQualifiedName(), ".", 2)
    mfqn := QualifiedName{}
    if len(parts) == 2 {
        mfqn = QualifiedName{Module: parts[0], Name: parts[1]}
    } else {
        mfqn = QualifiedName{Name: h.MicroflowQualifiedName()}
    }
    m.EventHandlers = append(m.EventHandlers, EventHandlerModel{
        Moment:            h.Moment(),
        Event:             h.Event(),
        Microflow:         mfqn,
        RaiseErrorOnFalse: h.RaiseErrorOnFalse(),
        PassEventObject:   h.PassEventObject(),
    })
}
```

- [ ] **Step 6: Update `entity/serialize.go` — emit EventHandlers in ToMDLStatement**

In `ToMDLStatement`, after the closing `")"` and before the index loop, add the event handler emission. Add it after the index loop (event handlers follow indexes in MDL syntax):

```go
for _, eh := range m.EventHandlers {
    paramStr := "()"
    if eh.PassEventObject {
        paramStr = "($currentObject)"
    }
    options := ""
    if eh.RaiseErrorOnFalse && strings.EqualFold(eh.Moment, "Before") {
        options = " raise error"
    }
    fmt.Fprintf(&sb, "\non %s %s call %s%s%s",
        strings.ToLower(eh.Moment), strings.ToLower(eh.Event),
        eh.Microflow, paramStr, options)
}
```

- [ ] **Step 7: Run tests — expect pass**

```bash
go test ./mdl/model/entity/... -run "TestLift_WithEvent|TestToMDL_WithEvent|TestToMDL_Event" -v
```

Expected: all pass.

- [ ] **Step 8: Commit**

```bash
git add mdl/model/entity/model.go mdl/model/entity/entity_test.go \
        mdl/model/entity/lift.go mdl/model/entity/hydrate.go \
        mdl/model/entity/serialize.go
git commit -m "feat(model/entity): add EventHandlerModel — Lift, Hydrate, ToMDLStatement (TDD)"
```

---

## Task 3: Persist EventHandlers + eliminate dual-path

**Files:**
- Modify: `mdl/model/entity/persist.go`
- Modify: `mdl/executor/cmd_create_entity_gen.go`

- [ ] **Step 1: Update `buildGenEntity` to write event handlers**

In `entity/persist.go`, at the end of `buildGenEntity` before `return e, nil`, add:

```go
// Event handlers.
for _, eh := range m.EventHandlers {
    h := genDm.NewEventHandler()
    h.SetMoment(eh.Moment)
    h.SetEvent(eh.Event)
    h.SetMicroflowQualifiedName(eh.Microflow.String())
    h.SetRaiseErrorOnFalse(eh.RaiseErrorOnFalse)
    h.SetPassEventObject(eh.PassEventObject)
    e.AddEventHandlers(h)
}
```

- [ ] **Step 2: Eliminate dual-path gate in `cmd_create_entity_gen.go`**

Find the gate (around line 77):
```go
if ctx.ModelCodecs != nil && len(s.EventHandlers) == 0 {
    if err := persistEntityCanonical(...); err != nil {
        return err
    }
    return nil
}
// Legacy path (event handlers, or absent codec registry).
entity := astToEntityGen(s)
```

Replace with (remove the `EventHandlers` condition):
```go
if ctx.ModelCodecs != nil {
    if err := persistEntityCanonical(ctx, s, dm, existingEntity, module); err != nil {
        return err
    }
    return nil
}
// Legacy path (absent codec registry — should not occur in production).
entity := astToEntityGen(s)
```

- [ ] **Step 3: Build and run full executor tests**

```bash
go build ./mdl/executor/...
go test ./mdl/executor/... -count=1 2>&1 | grep -E "^--- FAIL|^FAIL|^ok" | head -20
```

Expected: no new failures compared to baseline.

- [ ] **Step 4: Commit**

```bash
git add mdl/model/entity/persist.go mdl/executor/cmd_create_entity_gen.go
git commit -m "feat(model/entity): Persist event handlers; eliminate CREATE ENTITY dual-path"
```

---

## Task 4: OQL + system members into ToMDLStatement

After this task, `describeEntityGen` emits nothing after `m.ToMDLStatement(true)` except the trailing `;` and access grants.

**Files:**
- Modify: `mdl/model/entity/model.go`
- Modify: `mdl/model/entity/hydrate.go`
- Modify: `mdl/model/entity/serialize.go`
- Modify: `mdl/executor/cmd_entities_gen.go`

- [ ] **Step 1: Add `OQL` field to `EntityModel`**

In `model.go`, add to `EntityModel`:

```go
// OQL is the query body for EntityView entities. Empty for all other kinds.
OQL string
```

- [ ] **Step 2: Write tests**

Append to `entity_test.go`:

```go
func TestToMDL_WithSystemMembers(t *testing.T) {
	m := &entity.EntityModel{
		Name:          entity.QualifiedName{Module: "M", Name: "E"},
		Kind:          entity.EntityPersistent,
		SystemMembers: []string{"owner", "createdDate"},
	}
	got := m.ToMDLStatement(true)
	assert.Contains(t, got, "system members (owner, createdDate)")
}

func TestToMDL_ViewEntityWithOQL(t *testing.T) {
	m := &entity.EntityModel{
		Name: entity.QualifiedName{Module: "M", Name: "V"},
		Kind: entity.EntityView,
		OQL:  "select MyObject/Name from MyModule.MyObject",
	}
	got := m.ToMDLStatement(true)
	assert.Contains(t, got, "as (\n  select MyObject/Name from MyModule.MyObject\n)")
}
```

- [ ] **Step 3: Run — expect failures**

```bash
go test ./mdl/model/entity/... -run "TestToMDL_WithSystem|TestToMDL_View" -v 2>&1 | head -10
```

- [ ] **Step 4: Update `entity/hydrate.go` — extract OQL**

In `Hydrate`, after the event handler loop, add:

```go
// OQL for view entities.
if m.Kind == EntityView {
    if src := e.Source(); src != nil {
        type oqlSource interface{ Oql() string }
        if oq, ok := src.(oqlSource); ok {
            m.OQL = oq.Oql()
        }
    }
}
```

- [ ] **Step 5: Update `entity/serialize.go` — emit system members + OQL**

In `ToMDLStatement`, after the event handler loop, add:

```go
// System members (after event handlers).
if len(m.SystemMembers) > 0 {
    fmt.Fprintf(&sb, "\nsystem members (%s)", strings.Join(m.SystemMembers, ", "))
}

// OQL body for view entities.
if m.Kind == EntityView && m.OQL != "" {
    sb.WriteString(" as (\n")
    for _, line := range strings.Split(m.OQL, "\n") {
        fmt.Fprintf(&sb, "  %s\n", line)
    }
    sb.WriteString(")")
}
```

- [ ] **Step 6: Run tests — expect pass**

```bash
go test ./mdl/model/entity/... -run "TestToMDL_WithSystem|TestToMDL_View" -v
```

- [ ] **Step 7: Remove trailing clauses from `describeEntityGen` in `cmd_entities_gen.go`**

Remove the three blocks that append system members, view OQL, and event handlers after `fmt.Fprint(ctx.Output, m.ToMDLStatement(true))`. The function should end with:

```go
fmt.Fprint(ctx.Output, m.ToMDLStatement(true))
fmt.Fprintln(ctx.Output, ";")
outputEntityAccessGrantsGen(ctx, entity, name.Module, name.Name)
fmt.Fprintln(ctx.Output, "/")
return nil
```

Remove the imports that were only used by those blocks (e.g. `genDm` if no longer needed; check with `go build`).

- [ ] **Step 8: Build and run full test suite**

```bash
go build ./...
go test ./mdl/model/... ./mdl/executor/... -count=1 2>&1 | grep -E "^--- FAIL|^FAIL|^ok" | head -20
```

Expected: no new failures. All five guards GREEN.

- [ ] **Step 9: Final guard verification**

```bash
go test ./mdl/model/ -run "TestImportDirection|TestCodecComplete|TestPackageStructure" -v
go test ./mdl/executor/ -run TestExecutorBoundary -v
```

Expected: TestImportDirection PASS, TestCodecComplete PASS, TestPackageStructure PASS, TestExecutorBoundary PASS.

- [ ] **Step 10: Commit**

```bash
git add mdl/model/entity/model.go mdl/model/entity/hydrate.go \
        mdl/model/entity/serialize.go mdl/model/entity/entity_test.go \
        mdl/executor/cmd_entities_gen.go
git commit -m "feat(model/entity): OQL + system members in ToMDLStatement — describeEntityGen has no trailing clauses"
```

---

## Self-Review

| Requirement | Task |
|-------------|------|
| Guard 1 green (context.go no backend import) | Task 1 |
| Guard 4 green (cmd_* no entity import) | Task 1 |
| HydrateFn passes correct module name | Task 1 |
| PersistContext.Backend type-safe in entity | Task 1 |
| EventHandlerModel Lift | Task 2 |
| EventHandlerModel Hydrate | Task 2 |
| EventHandlerModel ToMDLStatement | Task 2 |
| EventHandlerModel Persist | Task 3 |
| Dual-path eliminated | Task 3 |
| System members in ToMDLStatement | Task 4 |
| OQL in ToMDLStatement | Task 4 |
| describeEntityGen has no trailing clauses | Task 4 |
