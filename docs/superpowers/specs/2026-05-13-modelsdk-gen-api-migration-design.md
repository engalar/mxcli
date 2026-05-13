# Design: Eliminate msdkWriteRaw — Migrate to gen Type-Safe API

**Date:** 2026-05-13  
**Status:** Approved  
**Driver:** Tech debt — `msdkWriteRaw` + raw BSON patches are fragile and bypass the type system

---

## Problem

The modelsdk migration (Phase 1/2) took the path of least resistance: rather than using `modelsdk/gen` type-safe APIs, most write paths produce raw BSON bytes via `sdk/mpr.Serialize*` or `codec.PatchBSONField`, then push bytes through `msdkWriteRaw`. This leaves 44 `msdkWriteRaw` calls vs only 15 `msdkWrite` calls — the type-safe pattern is the minority.

**Consequence:** Adding or changing fields requires auditing raw BSON construction; the type system provides no safety net; `PatchBSONField` calls are invisible to the compiler if field names are misspelled.

**Non-issue:** LazyDoc ensures that decode→mutate→encode via gen types preserves unknown fields. Roundtrip safety is confirmed.

---

## Write Path Taxonomy

```
msdkWrite    (15) = decode BSON → gen type → Set* → encode → WriteTransaction  [target]
msdkWriteRaw (44) = pre-built bytes → WriteTransaction                          [to eliminate]
```

`msdkWriteRaw` is kept alive by three dependency patterns:

| Pattern | Files | ~Count | Mechanism | Replace With |
|---------|-------|--------|-----------|--------------|
| P1: Serialize→raw | `update_services_modelsdk.go` | 13 | `Serialize*(sdk_obj)` → `msdkWriteRaw` | `msdkWrite` + gen `Set*` |
| P2: Patch→raw | `security_allowed_roles_modelsdk.go` | 12 | `PatchBSONField(...)` → `msdkWriteRaw` | `msdkWrite` + gen `Set*` |
| P3: DomainModel cycle + entity access | `domainmodel_modelsdk.go`, `security_entity_access_modelsdk.go` | 19 | `GetDomainModelByID` (sdk type) → mutate → `SerializeDomainModel` → `msdkWriteRaw`; `PatchAdd/Remove*EntityAccessRule` → `msdkWriteRaw` | `msdkWrite` + `gen/domainmodels Set*` |

The three Phases are independent and can be separate PRs.

---

## Phase 1: Serialize→raw Elimination (Update Services)

**Scope:** `update_services_modelsdk.go` only. Create paths (`create_services_modelsdk.go`) are left unchanged — `InsertUnit` does not call `updateTransactionID()` so the 1544 bug does not apply; Create paths will be retired with `sdk/mpr.Writer` in Phase 3.

**Pattern:**

```go
// Before
func (b *MprBackend) updateJavaActionViaModelsdk(ja *javaactions.JavaAction) error {
    contents, err := b.writer.SerializeJavaAction(ja)
    if err != nil { return err }
    return b.msdkWriteRaw(ja.ID, contents)
}

// After
func (b *MprBackend) updateJavaActionViaModelsdk(ja *javaactions.JavaAction) error {
    return b.msdkWrite(ja.ID, func(elem element.Element) error {
        typed, ok := elem.(*genja.JavaAction)
        if !ok { return fmt.Errorf("unexpected type %T", elem) }
        typed.SetName(ja.Name)
        // ... remaining fields
        return nil
    })
}
```

**Service types to migrate:**

| Function | sdk Input Type | gen Package |
|----------|---------------|-------------|
| `updateJavaActionViaModelsdk` | `*javaactions.JavaAction` | `modelsdk/gen/javaactions` |
| `updateDatabaseConnectionViaModelsdk` | `*model.DatabaseConnection` | `modelsdk/gen/databaseconnector` |
| `updateDataTransformerViaModelsdk` | `*model.DataTransformer` | `modelsdk/gen/datatransformers` |
| `updateImportMappingViaModelsdk` | `*model.ImportMapping` | `modelsdk/gen/importmappings` |
| `updateExportMappingViaModelsdk` | `*model.ExportMapping` | `modelsdk/gen/exportmappings` |
| `updateJsonStructureViaModelsdk` | `*types.JsonStructure` | `modelsdk/gen/jsonstructures` |
| `updateBusinessEventServiceViaModelsdk` | `*model.BusinessEventService` | `modelsdk/gen/businessevents` |
| `updateConsumedODataServiceViaModelsdk` | `*model.ConsumedODataService` | `modelsdk/gen/rest` (or odata) |
| `updatePublishedODataServiceViaModelsdk` | `*model.PublishedODataService` | `modelsdk/gen/odatapublish` |
| `updateConsumedRestServiceViaModelsdk` | `*model.ConsumedRestService` | `modelsdk/gen/rest` |
| `updatePublishedRestServiceViaModelsdk` | `*model.PublishedRestService` | `modelsdk/gen/rest` |
| `updateImageCollectionViaModelsdk` | `*types.ImageCollection` | `modelsdk/gen/images` (TBC) |

**Prerequisite per type:** confirm that the gen package's `Set*` methods cover every field written by the corresponding `Serialize*` function. Where a field is missing, add it via a supplement in `modelsdk/gen/<domain>/` before migrating that type — do not fall back to raw BSON.

**After Phase 1:** `serialize_exports.go` wrappers that were only used by `update_services_modelsdk.go` can be deleted. Those still used by `create_services_modelsdk.go` or other callers are retained.

---

## Phase 2: Patch→raw Elimination (AllowedRoles)

**Scope:** `security_allowed_roles_modelsdk.go` — three functions: `updateAllowedRolesViaModelsdk`, `updatePublishedRestServiceRolesViaModelsdk`, `removeFromAllowedRolesViaModelsdk`.

**Current mechanism:** `PatchBSONField("AllowedRoles", bson.A{int32(3), role1, ...})` — the `int32(3)` magic constant is an internal BSON array-sentinel that has no type-system representation.

**Target:**

```go
func (b *MprBackend) updateAllowedRolesViaModelsdk(unitID model.ID, roles []string) error {
    return b.msdkWrite(unitID, func(elem element.Element) error {
        type hasAllowedRoles interface {
            SetAllowedRoles([]string)
        }
        typed, ok := elem.(hasAllowedRoles)
        if !ok {
            return fmt.Errorf("unit type %T does not support AllowedRoles", elem)
        }
        typed.SetAllowedRoles(roles)
        return nil
    })
}
```

**Prerequisite:** confirm that `modelsdk/gen` types for microflow, page, workflow, published REST service, etc. implement a `SetAllowedRoles([]string)` method (or equivalent). If the interface approach doesn't work due to differing signatures, fall back to a type switch over the concrete gen types. Do not fall back to `PatchBSONField`.

**Entity access rules are NOT in this Phase** — they are deferred to Phase 3 because they operate on the same DomainModel unit and share the decode path.

---

## Phase 3: DomainModel Cycle + Entity Access Rules

**Scope:** `domainmodel_modelsdk.go` and `security_entity_access_modelsdk.go`.

### DomainModel cycle

Current flow:

```
b.reader.GetDomainModelByID() → *sdk/domainmodel.DomainModel
mutateFn(dm)                  → mutate sdk fields
b.writer.SerializeDomainModel(dm) → []byte
b.msdkWriteRaw(domainModelID, bytes)
```

Target flow:

```
msdkWrite(domainModelID, func(elem element.Element) {
    dm := elem.(*gendomainmodels.DomainModel)
    // use dm.EntitiesItems(), AddEntities(), RemoveEntities(), etc.
})
```

### Backend interface boundary

The `backend.go` method signatures accepting `*mdl/types.Entity`, `*mdl/types.Attribute`, etc. do not change — these are the executor contract. Conversion from `mdl/types` to `gen/domainmodels` occurs inside `domainmodel_modelsdk.go`:

```
backend interface (*mdl/types.Entity)
    ↓  [conversion: types → gen/domainmodels, within domainmodel_modelsdk.go]
msdkWrite + *gendomainmodels.DomainModel
```

### Entity access rules

`PatchAddEntityAccessRule`, `PatchRemoveEntityAccessRule`, `PatchRevokeEntityMemberAccess`, `PatchRemoveRoleFromAllEntities`, `PatchReconcileMemberAccesses` are replaced by the same `msdkWrite` path — the DomainModel gen type's access rule Part methods are used instead of raw BSON patching.

### Prerequisites

Confirm `modelsdk/gen/domainmodels` provides:

- `DomainModel.EntitiesItems()` / `AddEntities()` / `RemoveEntities(i)`
- `Entity.AttributesItems()` / `AddAttributes()` / `RemoveAttributes(i)`
- `DomainModel.AssociationsItems()` / `AddAssociations()` / `RemoveAssociations(i)`
- `DomainModel.CrossAssociationsItems()` / `AddCrossAssociations()` / `RemoveCrossAssociations(i)`
- Entity access rule Part operations (add, remove, query by entity name and role)

Where gen coverage is incomplete, extend via supplements. Do not fall back to raw BSON patches.

### After Phase 3

- `b.writer.SerializeDomainModel` call count drops to zero → can be deleted from `serialize_exports.go`
- All `Patch*` functions in `sdk/mpr/writer_security.go` become dead code → delete
- `b.reader.GetDomainModelByID()` in `domainmodel_modelsdk.go` is no longer needed for the write path (reads now go through `msdkWrite` decode) → the reader call is removed from that file

---

## End State

```
msdkWrite    (all write paths) — type-safe, compiler-verified
msdkWriteRaw (zero calls)      — deleted
```

`sdk/mpr.Writer` remaining responsibilities after Phase 3: Create paths (services, modules, microflows, pages), Java file system operations, AgentEditor (custom blob). These are retired in a subsequent spec.

---

## What This Is NOT

- Not changing the executor layer or backend interface types
- Not retiring `sdk/mpr.Writer` (that is a follow-on spec)
- Not migrating the read path (separate spec)
- Not touching AgentEditor (custom blob mechanism is fundamentally different)
- Not touching rename/refs cross-unit scan (separate concern)
