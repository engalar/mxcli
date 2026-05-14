# Stage 3.3 DomainModel Domain — Detailed Sub-Plan

> **For agentic workers:** REQUIRED SUB-SKILL: `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans`. Steps use checkbox `- [ ]` syntax for trackability.
>
> Generated from `2026-05-14-stage-3-3-domain-marathon-master.md` §6 priority #4 + §7 reference template + §4 phase template. Builds on the security plan (`2026-05-14-stage-3-3-security.md`), javaactions plan (`2026-05-14-stage-3-3-javaactions.md`), and workflows plan (`2026-05-14-stage-3-3-workflows.md`); inherits Stage 3.2 + Stage 3.3.1 infrastructure (`mprread.ListUnitsByType`, `helpers_gen_container.go`, `ByNameRefList` versioned-array fix).

**Goal:** Migrate the `domainmodel` domain off the legacy hand-written `sdk/domainmodel` package onto the auto-generated `modelsdk/gen/domainmodels` types. Final state: zero `sdk/domainmodel` imports outside `sdk/mpr/` (Stage 4 territory) and `api/` (which remains public API and requires user approval).

---

## §1 Background — Status Snapshot

The `domainmodel` domain is the **largest fanout** of the six Stage 3.3 domains: **49 in-scope importers**, with cross-domain consumers in pages, security, OQL inference, ELK visualization, Mermaid rendering, and CRUD page generation. The master plan §7 picked it as the worked-example reference because its blast radius justifies the deepest decomposition.

### Master plan §7.6 special considerations (REPEATED here for emphasis)

1. **Entity access rules** — per CLAUDE.md "Association Parent/Child Pointer Semantics", MemberAccess entries must only be added to the FROM entity (the one in `ParentPointer`). Adding them to the TO entity triggers CE0066 "Entity access is out of date".
2. **AllowedModuleRoles version-prefix bug** — fixed globally by Stage 3.3.1 D6a in `modelsdk/property/reference.go::ByNameRefList.BSONValue()`. Verify by smoke test on every CREATE ENTITY ACCESS with allowed roles in this plan.
3. **Index ordering deterministic** — per CLAUDE.md "Map iteration is deterministic" rule: any map iterated for serialization output must `sort.Strings(keys)` first.
4. **System module dependency** — per memory `project_sdk_domainmodel_status`, `sdk/domainmodel` deletion has historically been blocked on the `sdk/mpr/system_module.go` BSON-native rewrite. Phase E3 of THIS plan explicitly documents the escape hatch: if Stage 4 hasn't rewritten `system_module.go`, the package stays in place with a deprecation header.

### Already migrated — DO NOT redo

- `mdl/backend/mpr/domainmodel_modelsdk.go` (345 LoC, 14 funcs) — `*ViaModelsdk` write helpers backed by `b.writeDomainModel()` callback. **Input contract is sdk-typed**: `(*domainmodel.Entity)`, `(*domainmodel.Attribute)`, `(*domainmodel.Association)`, `(*domainmodel.CrossModuleAssociation)`. The actual write goes via `b.updateDomainModelViaModelsdk`. Phase D rebuilds the input contract to gen types; the inner write path stays.
- `mdl/backend/mpr/domainmodel_modelsdk_test.go` — covers existing `*ViaModelsdk` helpers.
- `mdl/backend/mpr/security_entity_access_gen_test.go` — tests entity access rules; references domainmodel for fixture setup.
- `mdl/executor/cmd_security_gen.go` (Stage 3.3.1) — uses `*domainmodel.Entity` and `*domainmodel.AccessRule` in `entityRuleRoleStrings` / `entityRuleRightStrings` helpers. Per security plan A8b: passthrough kept until domainmodel migrates (now). Phase C of this plan migrates these helpers.
- `modelsdk/gen/domainmodels/{types,enums,refs,version}.go` — auto-generated, 5061 LoC total, 71 types. Verified in §S2.4.

### Still to migrate (this plan's scope)

| File | LoC | What stays | What leaves |
|---|---|---|---|
| `sdk/domainmodel/domainmodel.go` | 607 | nothing | full package (after E3, conditional on system_module migration) |
| `api/domainmodels.go` | 860 | builder fluent API | sdk types in 30+ method signatures **REQUIRES USER APPROVAL** (master plan §3.8); plan defaults to retire+gen-rebuild |
| `mdl/backend/domainmodel.go` | 45 | interface methods | `*domainmodel.*` types in 11 method sigs |
| `mdl/backend/mpr/backend.go` | 881 | shim wrappers | `*domainmodel.*` types in 5 method sigs |
| `mdl/backend/mpr/domainmodel_modelsdk.go` | 345 | inner write logic | sdk-typed inputs in 14 helpers |
| `mdl/backend/mpr/domainmodel_modelsdk_test.go` | (size TBD) | infra | sdk fixtures |
| `mdl/backend/mpr/delete_move_modelsdk.go` | (size TBD) | move logic | 1 ref |
| `mdl/backend/mpr/mf_page_modelsdk.go` | (size TBD) | mf/page | 1 ref |
| `mdl/backend/mpr/modules_modelsdk.go` | (size TBD) | module ops | 1 ref |
| `mdl/backend/mpr/security_entity_access_gen_test.go` | (size TBD) | infra | 1 ref |
| `mdl/backend/mock/backend.go` | 309 | Func decls | `*domainmodel.*` in 11 Func types |
| `mdl/backend/mock/mock_domainmodel.go` | 169 | shim layer | `*domainmodel.*` in 11 method sigs |
| `mdl/catalog/builder.go` | 585 | reader interface | `ListDomainModels() ([]*domainmodel.DomainModel, error)` |
| `mdl/catalog/builder_modules.go` | 877 | module ops | 1 ref |
| `mdl/catalog/builder_permissions.go` | 256 | permissions | sdk-typed access-rule walk in 11 sites |
| `mdl/catalog/builder_permissions_test.go` | (size TBD) | infra | sdk fixtures |
| `mdl/executor/cmd_entities.go` | 1011 | nothing | full file (40 sdk refs) |
| `mdl/executor/cmd_entities_describe.go` | 521 | nothing | full file |
| `mdl/executor/cmd_entities_access.go` | 218 | nothing | full file |
| `mdl/executor/cmd_associations.go` | 547 | nothing | full file |
| `mdl/executor/cmd_modules.go` | 1154 | nothing | 3 refs in module describe |
| `mdl/executor/cmd_domainmodel_elk.go` | 515 | ELK rendering | full file |
| `mdl/executor/cmd_mermaid.go` | 323 | Mermaid rendering | full file |
| `mdl/executor/cmd_contract.go` | (size TBD) | OData contract | 1 ref |
| `mdl/executor/cmd_diff_mdl.go` | (size TBD) | MDL diff | 1 ref |
| `mdl/executor/cmd_import.go` | (size TBD) | import | 1 ref |
| `mdl/executor/cmd_move.go` | (size TBD) | MOVE entity | 1 ref |
| `mdl/executor/cmd_odata.go` | (size TBD) | OData | 1 ref |
| `mdl/executor/cmd_oql_plan.go` | (size TBD) | OQL plan | 1 ref |
| `mdl/executor/cmd_pages_builder.go` | (size TBD) | pages builder | 1 ref (CRUD entity lookup) |
| `mdl/executor/cmd_pages_builder_input_filters.go` | (size TBD) | pages | 1 ref |
| `mdl/executor/cmd_pages_builder_v3.go` | (size TBD) | pages v3 | 1 ref |
| `mdl/executor/cmd_security_gen.go` | (Stage 3.3.1) | security gen | bridge helpers `entityRuleRoleStrings`, `entityRuleRightStrings` |
| `mdl/executor/cmd_structure.go` + `cmd_structure_gen.go` | 751+509 | structure | 4 refs total |
| `mdl/executor/executor.go` | (size TBD) | infra | 1 ref (likely `domainmodel` import for cache type) |
| `mdl/executor/flowbuilder_actions_retrieve_gen.go` | (size TBD) | flow builder | 1 ref |
| `mdl/executor/flowbuilder_assoc_lookup_gen.go` | (size TBD) | flow builder | 1 ref |
| `mdl/executor/helpers.go` | (size TBD) | helpers | 1 ref |
| `mdl/executor/oql_type_inference.go` | (size TBD) | OQL types | 1 ref |
| `mdl/executor/cmd_*_mock_test.go` (8 files) | varies | needs migration | sdk fixtures |
| `mdl/executor/mock_test_helpers_test.go` | (small) | helper | sdk fixtures |
| `mdl/executor/validate_duplicates_test.go` | (small) | needs migration | sdk fixtures |
| `sdk/mpr/parser_domainmodel.go` | (Stage 4 territory) | — | DO NOT touch |
| `sdk/mpr/reader_documents.go` | (Stage 4 territory) | — | DO NOT touch |
| `sdk/mpr/system_module.go` | (Stage 4 territory) | — | DO NOT touch — **blocks E3 unless Stage 4 lands first** |
| `sdk/mpr/writer_*.go` (4 files) | (Stage 4 territory) | — | DO NOT touch |

### Why this domain is priority #4 (largest of the six)

- **Highest unblock value**: 35+ executor files unblocked
- **Cross-domain dependencies on this domain**: security, pages, OQL, mermaid, ELK all consume entities. Migrating domainmodel is a prerequisite for the cleanest pages migration (priority #5)
- **System module dependency**: see master plan §7.6 — `sdk/mpr/system_module.go` carries System.User/System.Role static entity definitions that the domainmodel system needs at startup. Phase E3 documents the escape hatch
- **`api/` public surface change** — UNIQUE to this domain (and pages); requires §14 user approval before execution

---

## §2 Pre-Flight Survey Results

### S2.1 sdk/domainmodel importers (49 in-scope, 8 Stage 4)

```
api/domainmodels.go                              ← 30+ refs in builder API (PUBLIC API)
mdl/backend/domainmodel.go                       ← interface (11 refs)
mdl/backend/mpr/backend.go                       ← shim (5 refs)
mdl/backend/mpr/delete_move_modelsdk.go
mdl/backend/mpr/domainmodel_modelsdk.go         ← write helpers (18 refs)
mdl/backend/mpr/domainmodel_modelsdk_test.go
mdl/backend/mpr/mf_page_modelsdk.go
mdl/backend/mpr/modules_modelsdk.go
mdl/backend/mpr/security_entity_access_gen_test.go
mdl/backend/mock/backend.go                      ← 11 Func types
mdl/backend/mock/mock_domainmodel.go             ← 11 method sigs
mdl/catalog/builder.go                           ← 1 reader method
mdl/catalog/builder_modules.go                   ← 1 ref
mdl/catalog/builder_permissions.go               ← 11 access-rule walk sites
mdl/catalog/builder_permissions_test.go
mdl/executor/cmd_associations.go                 ← assoc CRUD
mdl/executor/cmd_associations_mock_test.go
mdl/executor/cmd_contract.go                     ← OData contract entity lookup
mdl/executor/cmd_diff_mdl.go                     ← MDL diff
mdl/executor/cmd_domainmodel_elk.go              ← ELK rendering (Phase B)
mdl/executor/cmd_entities.go                     ← entity CRUD (40+ refs)
mdl/executor/cmd_entities_access.go              ← GRANT/REVOKE entity access
mdl/executor/cmd_entities_describe.go            ← DESCRIBE ENTITY
mdl/executor/cmd_entities_mock_test.go
mdl/executor/cmd_import.go                       ← IMPORT FROM ... INTO Module.Entity
mdl/executor/cmd_mermaid.go                      ← Mermaid rendering (Phase B)
mdl/executor/cmd_mermaid_mock_test.go
mdl/executor/cmd_modules.go                      ← module describe (3 refs)
mdl/executor/cmd_modules_mock_test.go
mdl/executor/cmd_move.go                         ← MOVE entity
mdl/executor/cmd_odata.go                        ← OData service entity refs
mdl/executor/cmd_odata_mock_test.go
mdl/executor/cmd_oql_plan.go                     ← OQL plan
mdl/executor/cmd_pages_builder.go                ← CRUD page entity lookup
mdl/executor/cmd_pages_builder_input_filters.go
mdl/executor/cmd_pages_builder_v3.go
mdl/executor/cmd_rename_mock_test.go
mdl/executor/cmd_security_gen.go                 ← entityRuleRoleStrings (Stage 3.3.1 bridge)
mdl/executor/cmd_security_mock_test.go
mdl/executor/cmd_structure.go                    ← structure deps (3 refs)
mdl/executor/cmd_structure_gen.go                ← gen structure (1 ref)
mdl/executor/cmd_write_handlers_mock_test.go
mdl/executor/executor.go                         ← cache type
mdl/executor/flowbuilder_actions_retrieve_gen.go ← retrieve activity entity ref
mdl/executor/flowbuilder_assoc_lookup_gen.go     ← assoc-lookup activity entity ref
mdl/executor/helpers.go                          ← helper
mdl/executor/mock_test_helpers_test.go
mdl/executor/oql_type_inference.go               ← OQL type inference
mdl/executor/validate_duplicates_test.go
sdk/mpr/parser_domainmodel.go            ← Stage 4 territory
sdk/mpr/reader_documents.go              ← Stage 4
sdk/mpr/system_module.go                 ← Stage 4 (BLOCKS E3)
sdk/mpr/writer_domainmodel.go            ← Stage 4
sdk/mpr/writer_domainmodel_test.go       ← Stage 4
sdk/mpr/writer_modules.go                ← Stage 4
sdk/mpr/writer_security_test.go          ← Stage 4
sdk/mpr/writer_units.go                  ← Stage 4
```

### S2.2 Read funcs in `cmd_entities*.go`, `cmd_associations.go`, `cmd_modules.go` (domain reads)

Major read functions:

`cmd_entities.go` (1011 LoC):
- `listEntities(ctx, moduleName)` — `ListDomainModels` + walk
- `entityCount`, `attributeCount` — counters
- `findEntityByQualifiedName`, `findEntityByID`
- ~30 sub-helpers for attribute formatting, type rendering, generalization walks

`cmd_entities_describe.go` (521 LoC):
- `describeEntity(ctx, name)` — `*domainmodel.Entity` walk
- `formatAttribute`, `formatAttributeType`, `formatGeneralization`, `formatIndex` etc.

`cmd_entities_access.go` (218 LoC):
- access rule reads (already partly migrated via `cmd_security_gen.go::entityRuleRoleStrings`)

`cmd_associations.go` (547 LoC):
- `listAssociations`, `describeAssociation`
- assoc-type/owner/storage formatters

`cmd_modules.go`:
- `describeModule` walks `dm.Entities` for the module's entity list

### S2.3 Write funcs in `cmd_entities.go` + `cmd_associations.go`

Major write surfaces (in `cmd_entities.go` + `cmd_associations.go`):
- `execCreateEntity` (CREATE ENTITY ... PERSISTABLE/NON-PERSISTABLE WITH GENERALIZATION ...)
- `execAlterEntity` (add/rename/drop attributes, set persistence, set documentation)
- `execDropEntity`
- `execSetEntityAccessRule` (covered by Stage 3.3.1 security plan but consumes domainmodel types)
- `execCreateAssociation` (1:1, 1:M, M:M with delete behavior + storage format)
- `execDropAssociation`
- `execMoveEntity` — covered by `mdl/backend/mpr/delete_move_modelsdk.go`

Plus `mdl/executor/cmd_move.go::execMoveEntity` for cross-module MOVE.

### S2.4 Gen accessor name map (CRITICAL — verified against `modelsdk/gen/domainmodels/types.go`)

**DomainModel (line 1497):**

| sdk field | gen accessor | Notes |
|---|---|---|
| `dm.Entities []*Entity` | `EntitiesItems() []element.Element / AddEntities / RemoveEntities` | PartList — cast each to `*Entity` |
| `dm.Associations []*Association` | `AssociationsItems()` (verify) | PartList |
| `dm.CrossAssociations []*CrossModuleAssociation` | `CrossAssociationsItems()` (verify) | PartList |
| `dm.Annotations` | likely `AnnotationsItems()` (verify) | PartList |

**Entity (line 1605):**

| sdk field | gen accessor | Notes |
|---|---|---|
| `e.Name` | `Name() / SetName` | exact |
| `e.Documentation` | `Documentation() / SetDocumentation` | exact |
| `e.Persistable` (bool) | `Persistable() / SetPersistable` (verify) | check accessor |
| `e.Generalization Generalization` | `Generalization() element.Element / SetGeneralization` | wraps `*NoGeneralization` (line 2579) or `*Generalization` (2247) |
| `e.Attributes []*Attribute` | `AttributesItems() []element.Element` | PartList |
| `e.AccessRules []*AccessRule` | `AccessRulesItems()` (verify) | PartList |
| `e.Indexes []*Index` | `IndexesItems()` (verify) | PartList |
| `e.ValidationRules []*ValidationRule` | `ValidationRulesItems()` (verify) | PartList |
| `e.EventHandlers []*EventHandler` | `EventHandlersItems()` (verify) | PartList |
| `e.Source EntitySource` | `Source() element.Element / SetSource` | wraps `*RemoteEntitySource` (2459) / `*ViewEntitySource` (2675) etc. |

**Attribute (line 825):**

| sdk field | gen accessor |
|---|---|
| `a.Name` | `Name() / SetName` |
| `a.Documentation` | `Documentation() / SetDocumentation` |
| `a.Type AttributeType` | `Type() element.Element / SetType` |
| `a.Value AttributeValue` | `Value() element.Element / SetValue` |

**Association (line 415):**

| sdk field | gen accessor |
|---|---|
| `a.Name` | `Name() / SetName` |
| `a.AssociationType` (1:1/1:M/M:M) | `AssociationType() / SetAssociationType` (verify enum mapping) |
| `a.AssociationOwner` (Default/Both) | `AssociationOwner() / SetAssociationOwner` |
| `a.AssociationStorageFormat` (DB/JSON) | `AssociationStorageFormat() / SetAssociationStorageFormat` |
| `a.Parent` (FROM entity ID — see CLAUDE.md "Association Parent/Child") | `ParentEntityRef() / SetParentEntityID` (verify) — **Note: FROM entity per the inverted naming convention** |
| `a.Child` (TO entity ID) | `ChildEntityRef() / SetChildEntityID` (verify) |
| `a.DeleteBehavior` | `DeleteBehavior() element.Element` |

**Attribute Type tree (sdk → gen):**

| sdk type | gen type (line) | Storage `$Type` |
|---|---|---|
| `StringAttributeType` | `StringAttributeType` (3138) | `DomainModels$StringAttributeType` |
| `IntegerAttributeType` | `IntegerAttributeType` (2435) | `DomainModels$IntegerAttributeType` |
| `LongAttributeType` | `LongAttributeType` (2447) | `DomainModels$LongAttributeType` |
| `DecimalAttributeType` | `DecimalAttributeType` (1445) | `DomainModels$DecimalAttributeType` |
| `BooleanAttributeType` | `BooleanAttributeType` (1069) | `DomainModels$BooleanAttributeType` |
| `DateTimeAttributeType` | `DateTimeAttributeType` (1421) | `DomainModels$DateTimeAttributeType` |
| `DateAttributeType` | (gen has no separate Date — use DateTime + bool flag) | (verify in A0.S1) |
| `EnumerationAttributeType` | `EnumerationAttributeType` (2047) | `DomainModels$EnumerationAttributeType` |
| `AutoNumberAttributeType` | `AutoNumberAttributeType` (1045) | `DomainModels$AutoNumberAttributeType` |
| `BinaryAttributeType` | `BinaryAttributeType` (1057) | `DomainModels$BinaryAttributeType` |
| `HashedStringAttributeType` | `HashedStringAttributeType` (2275) | `DomainModels$HashedStringAttributeType` |
| (none) | `CurrencyAttributeType` (1409) | gen-only |
| (none) | `FloatAttributeType` (2223) | gen-only — possibly subsumes `DecimalAttributeType`? |
| (none) | `MultiLanguageAttributeType` (2567) | gen-only |

**EntitySource tree:**

| sdk | gen (line) | Storage |
|---|---|---|
| `EntitySource` (interface) | `EntitySource` (2035) | abstract |
| (sdk doesn't expose) | `RemoteEntitySource` (2459) | `DomainModels$RemoteEntitySource` |
| (sdk doesn't expose) | `MaterializedRemoteEntitySource` (2471) | `DomainModels$MaterializedRemoteEntitySource` |
| (sdk doesn't expose) | `ViewEntitySource` (2675) + `OqlViewEntitySource` (2687) | view entities |
| (sdk doesn't expose) | `QueryBasedRemoteEntitySource` (2751) | external |

### S2.5 Backend interface methods (`mdl/backend/domainmodel.go`)

```go
DomainModelBackend:
  ListDomainModels() ([]*domainmodel.DomainModel, error)            // sdk-typed read — REPLACE
  GetDomainModel(moduleID model.ID) (*domainmodel.DomainModel, error)  // REPLACE
  GetDomainModelByID(id model.ID) (*domainmodel.DomainModel, error)    // REPLACE
  UpdateDomainModel(dm *domainmodel.DomainModel) error              // REPLACE

EntityBackend:
  CreateEntity(dmID model.ID, entity *domainmodel.Entity) error           // REPLACE input
  UpdateEntity(dmID model.ID, entity *domainmodel.Entity) error           // REPLACE
  DeleteEntity(id model.ID) error                                          // ok (just ID)
  MoveEntity(entity *domainmodel.Entity, srcDMID, tgtDMID model.ID, srcMod, tgtMod string) ([]string, error)  // REPLACE
  AddAttribute(dmID, entityID model.ID, attr *domainmodel.Attribute) error // REPLACE input
  UpdateAttribute(...) error                                               // REPLACE
  DeleteAttribute(id model.ID) error                                        // ok
  CreateAssociation(dmID model.ID, assoc *domainmodel.Association) error    // REPLACE
  CreateCrossAssociation(dmID model.ID, ca *domainmodel.CrossModuleAssociation) error  // REPLACE
  DeleteAssociation(id model.ID) error                                      // ok
```

**11 methods** with `*domainmodel.*` types — Phase C adds `*Gen` siblings (additive); Phase E retires legacy.

### S2.6 ExecContext field

No `ctx.DomainModels` field today. Phase A0 adds it.

### S2.7 MockBackend conformance — same `nil, nil` pattern violation as javaactions/workflows. Phase C5 brings into compliance.

### S2.8 Existing repos infrastructure

No `mdl/repos/domainmodels.go` or `mdl/backend/mpr/repos/domainmodels.go`. Phase A0 creates both.

### S2.9 Storage type names verified

- `DomainModels$DomainModel` — confirmed at `sdk/mpr/parser_domainmodel.go:28`
- All attribute type storage types confirmed by sdk write paths in `cmd_entities.go::astTypeToAttributeType`

### S2.10 system_module dependency (master plan §7.6)

`sdk/mpr/system_module.go` carries System.User, System.Role, System.UserRole entity definitions that domainmodel reads need at startup. Three options:

1. **Wait for Stage 4** to rewrite `system_module.go` BSON-natively before E3 deletes `sdk/domainmodel`. **Default.**
2. **Inline in this plan**: rewrite `sdk/mpr/system_module.go` as part of Phase E. **Out of scope per CLAUDE.md "no Stage 4 territory" rule.**
3. **Deprecation header on `sdk/domainmodel`** until Stage 4 lands. Same escape hatch as security E3.

Plan picks option **#1 + #3 hybrid**: complete A0–E2; E3 conditional on Stage 4. If Stage 4 hasn't landed, leave `sdk/domainmodel/domainmodel.go` with deprecation header noting "kept for sdk/mpr/system_module.go consumption only — to be deleted when Stage 4 rewrites system_module.go".

### S2.11 api/ public surface (master plan §3.8)

`api/domainmodels.go` exposes `*domainmodel.Entity`, `*domainmodel.Attribute`, `*domainmodel.Association`, etc. in 30+ method signatures. The fluent builders (`EntityBuilder`, `AssociationBuilder`, etc.) carry sdk types as state.

**Three options:**
- **Option A (preferred per Stage 3.2 task B precedent): retire and rebuild**. Delete `api/domainmodels.go` entirely; consumers route through `ctx.Backend.{CreateEntityGen,UpdateEntityGen}` directly OR through a new gen-typed `api/v2/domainmodels.go`. Higher one-time cost; cleanest end state.
- **Option B: keep API but swap state types to gen**. The fluent builders' `*EntityBuilder.entity` field becomes `*genDM.Entity`. Public method signatures change (breaking). Same end state, but in-place.
- **Option C: keep API + add v2**. New `api/v2/domainmodels.go` for gen; keep `api/domainmodels.go` as deprecated. Two parallel APIs to maintain.

**§14 open question: which option does the user choose?** Plan defaults to Option A (retire) per Stage 3.2 microflow precedent.

---

## §3 Risks Specific to DomainModel Domain

| # | Risk | Impact | Mitigation |
|---|---|---|---|
| R1 | **`api/domainmodels.go` public API change** (§S2.11) — 30+ method signatures touch sdk types | Critical — public API break | §14 user decision required BEFORE execution. Default = Option A retire. If user picks B/C, plan adjusts |
| R2 | **System module dependency** (§S2.10) — `sdk/mpr/system_module.go` consumes sdk types | High — blocks E3 | Documented escape hatch in §11; deprecation header in lieu of delete |
| R3 | **AssociationParent/ChildPointer semantics** (CLAUDE.md "Association Parent/Child Pointer Semantics") — `ParentPointer` = FROM entity (FK owner), `ChildPointer` = TO entity. MemberAccess on the wrong side triggers CE0066 | Critical — silent data corruption | Phase D7 round-trip test explicitly asserts MemberAccess only on FROM entity. Reuse Stage 3.3.1 D6 test pattern |
| R4 | **AllowedModuleRoles version-prefix** — fixed globally by Stage 3.3.1 D6a; verify CREATE ACCESS RULE round-trip emits `[int32(1), "Module.Role"]` | Critical | Phase D access-rule test asserts shape |
| R5 | **Polymorphic AttributeType discrimination** (§S2.4) — gen has 14+ concrete subtypes; need `elem.TypeName()` switch in formatters | High | Phase A formatter uses TypeName switch (mirrors javaactions/workflows pattern); document in helper file |
| R6 | **Index ordering deterministic** — per CLAUDE.md "Map iteration is deterministic"; serializing entity attributes/indexes/access rules from a map produces non-deterministic BSON | High — flaky diffs / round-trip failures | Every formatter and writer that iterates a collection MUST `sort.Slice` or `sort.Strings` first. Add lint check in Phase D |
| R7 | **`DateAttributeType` schema gap** (§S2.4) — sdk has separate `DateAttributeType`, gen may not. Date storage uses `DateTimeAttributeType` with a bool discriminator | Medium | A0.S1 verifies; if confirmed, formatter dispatches on `DateTimeAttributeType.LocalizeDate()` (or whatever the discriminator is) to render "Date" vs "DateTime" |
| R8 | **`api/` consumers**: API has external callers (any user `import` statement). Retiring breaks downstream | Critical | §14 user decision; if Option A picked, document migration in `docs/01-project/MDL_QUICK_REFERENCE.md` and `CLAUDE.md` |
| R9 | **Pages domain dependency** — `cmd_pages_builder.go`, `cmd_pages_builder_v3.go`, `cmd_pages_builder_input_filters.go` consume domainmodel for entity refs in CRUD page generation. Migrating these requires either (a) full pages migration (priority #5) or (b) shim adapters that read gen domainmodel + emit sdk-typed pages | High blast radius | Phase C7: shim adapters that take gen domainmodel as input but feed sdk-typed pages. When pages migration lands (priority #5), the shims become trivial passthroughs and are deleted in Stage 3.3.5.E2 |
| R10 | **OQL type inference** — `oql_type_inference.go` walks entity attributes to infer SQL column types. The walk is a pure-read consumer; migration is mechanical but the OQL test suite is large | Medium | Phase C8: convert + run full OQL suite as gate |
| R11 | **CrossModuleAssociation** — sdk has `CrossModuleAssociation` separate type; gen has `CrossAssociation` (line 1181). Naming difference + maybe semantic (CrossModule = always cross-module; gen `CrossAssociation` is unclear). Verify in A0 | Medium | A0.S1 inspects gen `CrossAssociation` accessors and storage `$Type` |
| R12 | **Stage 4 boundary at sdk/mpr**: 8 sdk/mpr files (more than security/javaactions/workflows). E3 strictly conditional | Medium | Same escape hatch as previous plans |
| R13 | **Cross-domain consumers in this plan**: `cmd_security_gen.go::entityRuleRoleStrings` (Stage 3.3.1 left a bridge); `cmd_workflows.go` calls `ListDomainModels` for entity validation. Both need migration here | Low | Phase C explicitly enumerates these as C9 (security bridge) and C10 (workflow bridge — already migrated by Stage 3.3.3 C7 if both plans are interleaved) |

---

## §4 Phase A — Read Path Migration

### Task A0: Repo + cache + ctx wiring

**Files:**
- Create: `mdl/repos/domainmodels.go`
- Create: `mdl/backend/mpr/repos/domainmodels.go`
- Create: `mdl/backend/mpr/repos/domainmodels_test.go`
- Create: `mdl/executor/helpers_domainmodels_gen.go`
- Create: `mdl/executor/helpers_domainmodels_gen_test.go`
- Modify: `mdl/executor/exec_context.go` — add `DomainModels repos.DomainModelRepository`
- Modify: `mdl/executor/executor.go` — add cache field
- Modify: BackendFactory wiring

#### A0.S1: Pre-flight gen verification (one-off, NO commit)

```bash
# Verify gen accessor names + missing types
grep -nE "^func \(o \*(DomainModel|Entity|Attribute|Association|Generalization|Index|AccessRule|MemberAccess)\) " modelsdk/gen/domainmodels/types.go > /tmp/dm-accessors.txt
grep -nE "^type (DateAttributeType|CurrencyAttributeType|FloatAttributeType|MultiLanguage|CrossModule|CrossAssociation)" modelsdk/gen/domainmodels/types.go > /tmp/dm-types.txt
cat /tmp/dm-accessors.txt /tmp/dm-types.txt
```

Update §S2.4 with verified accessor names. Note any gaps (especially DateAttributeType, R7).

#### A0.S2–A0.S7: same TDD pattern as workflows A0 / javaactions A0 — write failing test, RED, implement repo + helpers, wire ctx, GREEN, commit

```bash
git commit -m "feat(executor,repos): Stage 3.3.4.A0 — domainmodels repo + cache helpers + ctx wiring"
```

### Task A1: `listEntitiesGen` + entity counters

Mirror javaactions A1 / workflows A1 pattern. Walks all DomainModel units, then iterates `dm.EntitiesItems()` per module.

**Files:**
- Create: `mdl/executor/cmd_entities_gen.go`
- Create: `mdl/executor/cmd_entities_gen_test.go`

#### A1.S1: Test — `TestListEntitiesGen_OutputsAttributeCount`
#### A1.S2: RED
#### A1.S3: Implement `listEntitiesGen(ctx, moduleName)` reading via `listDomainModelsWithContainerGen` + walking `EntitiesItems`
#### A1.S4–A1.S6: GREEN, gate, Commit `feat(executor): Stage 3.3.4.A1 — listEntitiesGen (gen-typed)`

### Task A2: `formatAttributeTypeGen` polymorphic dispatch

The hardest read-path piece (parallel to javaactions A2). Replaces `formatAttributeType` with gen-typed dispatch over `attr.Type()` element by `elem.TypeName()`.

**Files:**
- Modify: `mdl/executor/cmd_entities_gen.go`
- Modify: `mdl/executor/cmd_entities_gen_test.go`

#### A2.S1: Tests — one per attribute type (StringAttributeType, IntegerAttributeType, LongAttributeType, DecimalAttributeType, BooleanAttributeType, DateTimeAttributeType, DateAttributeType (R7 gap), EnumerationAttributeType, AutoNumberAttributeType, BinaryAttributeType, HashedStringAttributeType, CurrencyAttributeType, FloatAttributeType, MultiLanguageAttributeType)

#### A2.S2: RED
#### A2.S3: Implement (mirror cmd_entities.go formatAttributeType with TypeName switch on `DomainModels$*AttributeType` keys)
#### A2.S4: GREEN per sub-test
#### A2.S5: gate
#### A2.S6: Commit `feat(executor): Stage 3.3.4.A2 — formatAttributeTypeGen polymorphic dispatch (14 attr types)`

### Task A3: `describeEntityGen` (per master plan §7.3)

Per master plan §7.3 reference. Renders ENTITY <qn> with attributes, indexes, access rules, generalization, source, validation rules, event handlers, documentation.

**Files:**
- Modify: `mdl/executor/cmd_entities_gen.go`
- Modify: `mdl/executor/cmd_entities_gen_test.go`

#### A3.S1–A3.S6: TDD per master plan §7.3 example, Commit `feat(executor): Stage 3.3.4.A3 — describeEntityGen (gen-typed, all sub-fields)`

### Task A4: `listAssociationsGen` + `describeAssociationGen`

Mirror cmd_associations.go.

**Files:**
- Create: `mdl/executor/cmd_associations_gen.go`
- Create: `mdl/executor/cmd_associations_gen_test.go`

#### A4.S1–A4.S6: TDD, Commit `feat(executor): Stage 3.3.4.A4 — listAssociationsGen + describeAssociationGen (gen-typed)`

### Task A5: `describeModuleGen` — module describe with entity walk

Migrate the 3 sdk refs in `cmd_modules.go`. They walk `dm.Entities` to render the module's entity list.

**Files:** `mdl/executor/cmd_modules.go` OR new `cmd_modules_gen.go`

#### A5.S1–A5.S6: TDD, Commit `refactor(executor): Stage 3.3.4.A5 — module describe entity walk (gen)`

### Task A6: Dispatcher cutover (executor_query.go + executor_describe.go)

Switch every `ShowEntities`, `DescribeEntity`, `ShowAssociations`, `DescribeAssociation`, `ShowAccessOnEntity` (already partially migrated by Stage 3.3.1) to `*Gen` variant.

#### A6.S1–A6.S4: locate / replace / build+test / Commit `refactor(executor): Stage 3.3.4.A6 — dispatch all SHOW/DESCRIBE entity/assoc to gen variants`

---

## §5 Phase B — Visualization (ELK + Mermaid)

The domainmodel domain has TWO dedicated visualizers: `cmd_domainmodel_elk.go` (515 LoC) and `cmd_mermaid.go` (323 LoC). Each needs full migration.

### Task B1: `cmd_domainmodel_elk_gen.go`

Mirror `cmd_domainmodel_elk.go` with gen reads. ELK output format MUST be byte-identical to legacy (consumers downstream depend on positions).

**Files:**
- Create: `mdl/executor/cmd_domainmodel_elk_gen.go`
- Create: `mdl/executor/cmd_domainmodel_elk_gen_test.go`

#### B1.S1: Snapshot test — assert ELK SVG byte-identical
#### B1.S2: RED
#### B1.S3: Implement (gen-walk over entities + associations)
#### B1.S4–B1.S6: GREEN, gate, Commit `feat(executor): Stage 3.3.4.B1 — cmd_domainmodel_elk_gen (gen-typed ELK rendering)`

### Task B2: `cmd_mermaid_gen.go`

Mirror `cmd_mermaid.go` with gen reads.

#### B2.S1–B2.S6: TDD, Commit `feat(executor): Stage 3.3.4.B2 — cmd_mermaid_gen (gen-typed Mermaid rendering)`

### Task B3: Dispatcher cutover for visualization

Switch `cmd_visualize.go` (or wherever ELK/Mermaid is dispatched) to `*Gen` variant.

#### B3.S1–B3.S4: locate / replace / build+test / Commit `refactor(executor): Stage 3.3.4.B3 — dispatch domainmodel visualizations to gen variants`

---

## §6 Phase C — Consumer Migration

### Task C1: `mdl/backend/domainmodel.go` + `mdl/backend/mpr/backend.go` — additive gen-typed methods

Add `ListDomainModelsGen`, `GetDomainModelGen`, `GetDomainModelByIDGen`, `UpdateDomainModelGen`, `CreateEntityGen`, `UpdateEntityGen`, `MoveEntityGen`, `AddAttributeGen`, `UpdateAttributeGen`, `CreateAssociationGen`, `CreateCrossAssociationGen` BEFORE retiring legacy.

**Files:**
- Modify: `mdl/backend/domainmodel.go` (add 11 gen-typed methods)
- Modify: `mdl/backend/mpr/backend.go` (impl via repo)
- Modify: `mdl/backend/mock/backend.go` + `mock_domainmodel.go` (Func-field stubs with descriptive errors)

#### C1.S1–C1.S6: TDD, Commit `feat(backend): Stage 3.3.4.C1 — add gen-typed domainmodel read/write methods to FullBackend`

### Task C2: `mdl/catalog/builder.go` + `builder_modules.go` + `builder_permissions.go`

Migrate sdk refs in 4 catalog files.

**Files:**
- Modify: `mdl/catalog/builder.go` (Reader interface change)
- Modify: `mdl/catalog/builder_modules.go` (1 ref)
- Modify: `mdl/catalog/builder_permissions.go` (11 access-rule walk sites)
- Modify: `mdl/catalog/builder_permissions_test.go` (fixtures)

#### C2.S1–C2.S6: TDD, Commit `refactor(catalog): Stage 3.3.4.C2 — domainmodel readers + permissions extractor on gen types`

### Task C3: `mdl/executor/cmd_security_gen.go` bridge migration (Stage 3.3.1 leftover)

`entityRuleRoleStrings(rule *domainmodel.AccessRule)` and `entityRuleRightStrings` were left as sdk-typed bridges by Stage 3.3.1 A8b. Migrate to gen `*genDM.AccessRule`.

**Files:** `mdl/executor/cmd_security_gen.go`

#### C3.S1–C3.S6: TDD, Commit `refactor(executor): Stage 3.3.4.C3 — security gen helpers consume gen-typed AccessRule`

### Task C4: `cmd_structure.go` + `cmd_structure_gen.go` migration

Migrate the 4 structure refs (entity/assoc counts per module).

**Files:** both files

#### C4.S1–C4.S6: TDD, Commit `refactor(executor): Stage 3.3.4.C4 — structure entity/assoc walk on gen types`

### Task C5: MockBackend audit

Bring `mdl/backend/mock/mock_domainmodel.go` into compliance.

**Files:** `mdl/backend/mock/mock_domainmodel.go`

#### C5.S1–C5.S6: TDD, Commit `refactor(mock): Stage 3.3.4.C5 — MockBackend domainmodel stubs return descriptive errors`

### Task C6: Mock test fixture migration

Migrate sdk-typed fixtures in 8 mock test files PLUS `mock_test_helpers_test.go` and `validate_duplicates_test.go`. ONE commit (cross-file consistency).

**Files:** all `cmd_*_mock_test.go` files listed in §S2.1 + helpers

#### C6.S1–C6.S6: TDD, Commit `test(executor): Stage 3.3.4.C6 — migrate domainmodel mock fixtures to gen builders`

### Task C7: Pages-builder shim adapters (R9)

`cmd_pages_builder.go`, `cmd_pages_builder_v3.go`, `cmd_pages_builder_input_filters.go` consume domainmodel for entity refs in CRUD page generation. Add a small shim function `genEntityToPagesEntityRef(*genDM.Entity) pages.EntityRef` that lets these files take gen input but emit sdk-typed `pages.*` (since pages migration is priority #5, deferred).

**Files:**
- Create: `mdl/executor/helpers_pages_dm_shim.go`
- Modify: the three pages-builder files to use the shim

#### C7.S1–C7.S6: TDD, Commit `refactor(executor): Stage 3.3.4.C7 — pages-builder consumes gen domainmodel via shim adapters`

### Task C8: Small executor consumers (each 1-ref file)

Migrate 9 small consumers in one commit per file (or grouped if ≤5 files):
- `cmd_contract.go`, `cmd_diff_mdl.go`, `cmd_import.go`, `cmd_move.go`, `cmd_odata.go`, `cmd_oql_plan.go`, `flowbuilder_actions_retrieve_gen.go`, `flowbuilder_assoc_lookup_gen.go`, `helpers.go`, `oql_type_inference.go`

#### C8.S1–C8.S6: TDD per group of ~3 files. Commits like `refactor(executor): Stage 3.3.4.C8.a — contract/diff/import on gen types` (3-4 commits total)

### Task C9: `executor.go` cache type migration

`mdl/executor/executor.go::executorCache.domainModels []*domainmodel.DomainModel` field. Migrate to gen.

**Files:** `mdl/executor/executor.go`

#### C9.S1–C9.S6: TDD, Commit `refactor(executor): Stage 3.3.4.C9 — executorCache holds gen-typed DomainModels`

### Task C10: `api/domainmodels.go` retire (§S2.11 R1, §14 Q1)

**ONLY EXECUTE IF USER PICKS OPTION A IN §14 Q1.**

Delete `api/domainmodels.go` entirely. Update consumers (likely no in-repo consumers; downstream users must migrate per release note).

**Files:**
- Delete: `api/domainmodels.go`
- Document: `docs/01-project/API_MIGRATION_3_3.md` (new, brief)

#### C10.S1: Final grep — no in-repo consumers
#### C10.S2: `git rm api/domainmodels.go`
#### C10.S3: Build + full test gate
#### C10.S4: Commit `refactor(api): Stage 3.3.4.C10 — retire api/domainmodels.go (sdk-typed)`

If Option B/C picked, replace this task with `feat(api): Stage 3.3.4.C10 — api/domainmodels.go switches to gen types` (in-place rebuild).

---

## §7 Phase D — Write Path Migration

### Task D1: AST → gen entity/attribute/association builders

**Files:**
- Create: `mdl/executor/cmd_entities_write_gen.go`
- Create: `mdl/executor/cmd_associations_write_gen.go`
- Tests in `*_test.go` siblings

#### D1.S1: Tests per AST kind (CreateEntity, AlterEntity add/rename/drop attribute, CreateAssociation per type)
#### D1.S2: RED
#### D1.S3: Implement `astToEntityGen`, `astToAttributeGen`, `astToAttributeTypeGen` (14 cases), `astToAssociationGen`, `astToGeneralizationGen`
#### D1.S4: GREEN per group, D1.S5: gate, D1.S6: Commit per group (~5 commits across attribute types, entity, association, generalization)

### Task D2: `execCreateEntityGen`

Replaces `execCreateEntity`. Uses D1 builders + `ctx.Backend.CreateEntityGen` (added in C1).

#### D2.S1: Round-trip test (CREATE persistable entity with all attribute types + generalization + index)
#### D2.S2–D2.S6: TDD, Commit `feat(executor): Stage 3.3.4.D2 — execCreateEntityGen`

### Task D3: `execAlterEntityGen` (add/rename/modify/drop attributes, set persistence, indexes, documentation)

Heaviest write task. ALTER ENTITY has many sub-operations.

**Sub-tasks (each own commit):**
- D3.a: `ALTER ENTITY ADD ATTRIBUTE`
- D3.b: `ALTER ENTITY DROP ATTRIBUTE`
- D3.c: `ALTER ENTITY RENAME ATTRIBUTE`
- D3.d: `ALTER ENTITY MODIFY ATTRIBUTE` (type/length change)
- D3.e: `ALTER ENTITY ADD INDEX`
- D3.f: `ALTER ENTITY DROP INDEX`
- D3.g: `ALTER ENTITY SET PERSISTENT/NON-PERSISTENT`
- D3.h: `ALTER ENTITY SET DOCUMENTATION`
- D3.i: `ALTER ENTITY SET GENERALIZATION`

Each sub-task: TDD, Commit `feat(executor): Stage 3.3.4.D3.<x> — ALTER ENTITY <op> (gen)`

### Task D4: `execDropEntityGen`

Mirror execDropEntity; list via `listEntitiesWithContainerGen`, delete via `ctx.Backend.DeleteEntity(id)` (unchanged).

#### D4.S1–D4.S6: TDD, Commit `feat(executor): Stage 3.3.4.D4 — execDropEntityGen`

### Task D5: `execCreateAssociationGen` + `execDropAssociationGen`

**Sub-tasks:**
- D5.a: 1:1 association
- D5.b: 1:M association
- D5.c: M:M association
- D5.d: Cross-module association (CreateCrossAssociationGen)
- D5.e: DROP

Each: TDD, Commit `feat(executor): Stage 3.3.4.D5.<x> — CREATE ASSOCIATION <kind> (gen)`

### Task D6: `execAlterEntityAccessRuleGen` (Stage 3.3.1 D6 sibling for entity side)

Builds on Stage 3.3.1 D6 + D6a (version-prefix fix). The entity-side write of the access rule was in security plan; this task ensures entity-side reads go through gen too.

#### D6.S1–D6.S6: TDD with explicit MemberAccess on FROM entity assertion (R3), Commit `feat(executor): Stage 3.3.4.D6 — entity access rule gen path (FROM-only MemberAccess)`

### Task D7: `execMoveEntityGen` (cross-module MOVE)

Migrate `mdl/backend/mpr/delete_move_modelsdk.go::moveEntityViaModelsdk` input to gen + executor `cmd_move.go` integration.

#### D7.S1–D7.S6: TDD, Commit `feat(executor,backend): Stage 3.3.4.D7 — MOVE ENTITY on gen types`

### Task D8: Backend gen-native repo writers (replace `*ViaModelsdk` input contract)

The 14 `*ViaModelsdk` helpers in `domainmodel_modelsdk.go` take sdk-typed inputs. Add gen-typed siblings (`createEntityViaModelsdkGen`, etc.) that take `*genDM.Entity` and convert to BSON via `codec.Encode`. Use these from the backend's `CreateEntityGen` etc. methods (C1).

**Files:**
- Modify: `mdl/backend/mpr/domainmodel_modelsdk.go` (add `*Gen` siblings)
- Modify: `mdl/backend/mpr/domainmodel_modelsdk_test.go` (cover both old + new)

#### D8.S1–D8.S6: TDD per helper group (~3 commits for 14 helpers grouped by entity/attribute/association)

### Task D9: Wire write dispatchers

Switch all CREATE/ALTER/DROP entity/attribute/association/index registrations in `register_stubs.go` to `*Gen` variants.

#### D9.S1–D9.S4: locate / replace / build+test / Commit `refactor(executor): Stage 3.3.4.D9 — dispatch all CREATE/ALTER/DROP entity/assoc to gen variants`

---

## §8 Phase E — Cleanup

### Task E1: Retire `FullBackend` deprecated sdk-typed domainmodel methods

**Files:**
- Modify: `mdl/backend/domainmodel.go` (delete 11 legacy methods)
- Modify: `mdl/backend/mpr/backend.go` (delete shims)
- Modify: `mdl/backend/mpr/domainmodel_modelsdk.go` (delete 14 legacy `*ViaModelsdk` helpers; keep only `*Gen` siblings)
- Modify: `mdl/backend/mock/backend.go` + `mock_domainmodel.go` (delete corresponding Func fields and shims)

#### E1.S1: Build to confirm 0 callers
#### E1.S2: Delete + commit `refactor(backend): Stage 3.3.4.E1 — retire FullBackend domainmodel sdk-typed methods + ViaModelsdk wrappers`

### Task E2: Delete legacy executor domainmodel files

**Files:**
- Delete: `mdl/executor/cmd_entities.go`
- Delete: `mdl/executor/cmd_entities_describe.go`
- Delete: `mdl/executor/cmd_entities_access.go`
- Delete: `mdl/executor/cmd_associations.go`
- Delete: `mdl/executor/cmd_domainmodel_elk.go`
- Delete: `mdl/executor/cmd_mermaid.go`

#### E2.S1: Final grep — no callers in mdl/ uses anything from those files
#### E2.S2: `git rm`
#### E2.S3: Build + full test gate
#### E2.S4: Commit `refactor(executor): Stage 3.3.4.E2 — delete legacy domainmodel executor files`

### Task E3: Delete `sdk/domainmodel` package — CONDITIONAL on Stage 4

**Files:** `sdk/domainmodel/domainmodel.go`

#### E3.S1: Final acceptance grep
```bash
grep -rln '"github.com/mendixlabs/mxcli/sdk/domainmodel"' . --include="*.go" | grep -v "^./sdk/mpr/"
```
Expected: empty (or only `sdk/mpr/system_module.go` etc.).

#### E3.S2: Verify Stage 4 boundary
```bash
git log -1 --oneline -- sdk/mpr/system_module.go sdk/mpr/parser_domainmodel.go sdk/mpr/writer_domainmodel.go
```
If Stage 4 has rewritten `system_module.go` BSON-natively + dropped the sdk import, proceed with delete.

#### E3.S3a (Stage 4 done): Delete + build + test gate + Commit `refactor: Stage 3.3.4.E3 — delete sdk/domainmodel package`

#### E3.S3b (Stage 4 NOT done): Add deprecation header

```bash
# Edit sdk/domainmodel/domainmodel.go top:
//
// DEPRECATED — Stage 3.3.4 (2026-05-XX): all in-scope consumers (mdl/, api/, modelsdk/)
// migrated to modelsdk/gen/domainmodels. The package is kept ONLY because
// sdk/mpr/system_module.go still imports it. Will be deleted after Stage 4
// rewrites system_module.go BSON-natively.
//
git add sdk/domainmodel/domainmodel.go
git commit -m "refactor: Stage 3.3.4.E3 — deprecation header on sdk/domainmodel (Stage 4 boundary)"
```

### Task E4: Final acceptance verification

#### E4.S1: Acceptance greps
```bash
grep -rln '"github.com/mendixlabs/mxcli/sdk/domainmodel"' . --include="*.go" | grep -v "^./sdk/mpr/"
grep -rln '"github.com/mendixlabs/mxcli/sdk/domainmodel"' modelsdk/ --include="*.go"
# api/ check depends on §14 Q1: if Option A (retire), must be 0
```

#### E4.S2: Run domainmodel-affecting tests
```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./mdl/executor/ ./mdl/backend/mpr/ ./mdl/backend/mpr/repos/ ./mdl/catalog/ -run "Entity|Association|DomainModel" -count=1 -v
```

#### E4.S3: Verify cache helper exists — `listDomainModelsWithContainerGen`

#### E4.S4: Update memory `project_stage_3_3_domainmodel_complete.md` with final stats

#### E4.S5: No commit needed

---

## §9 Acceptance Criteria

- [ ] `grep -rln '"github.com/mendixlabs/mxcli/sdk/domainmodel"' . --include="*.go" | grep -v "^./sdk/mpr/"` returns 0 lines (or only `api/domainmodels.go` if Option B/C picked in §14 Q1)
- [ ] All `Test*Entity*`, `Test*Association*`, `Test*DomainModel*` tests pass
- [ ] `listEntitiesWithContainerGen` (cache helper) is the only path used in `mdl/executor/` for entity listing
- [ ] `ctx.DomainModels` repo is the only path for DomainModel reads in `mdl/executor/`
- [ ] `mx check` smoke test passes on a round-tripped MPR with: CREATE ENTITY (all 14 attribute types), CREATE ASSOCIATION (1:1, 1:M, M:M, cross-module), ALTER ENTITY (all 9 sub-operations), CREATE ENTITY ACCESS RULE with allowed roles
- [ ] **Visualization parity**: `mxcli visualize domainmodel Module --format svg` AND `mxcli show structure --format mermaid` produce byte-identical output before vs after migration
- [ ] **MemberAccess on FROM entity only**: every CREATE ENTITY ACCESS RULE round-trip asserts MemberAccesses are only on ParentPointer side (R3)
- [ ] **AllowedModuleRoles version-prefix correct**: `[int32(1), "Module.Role"]` shape on every access-rule write
- [ ] Full repo build: `GOPROXY=... ~/go1.26/bin/go build ./...`
- [ ] Full repo test suite: `GOPROXY=... ~/go1.26/bin/go test ./... -count=1 -timeout 240s`
- [ ] `sdk/domainmodel/` deletion or deprecation header (per §S2.10 + §11)

---

## §10 Estimated Commit Count + Sequencing

| Phase | Tasks | Commits | Cumulative |
|---|---|---|---|
| A — Read path | A0, A1, A2 (4 commits per attr-type group of ~3-4), A3, A4, A5, A6 | 11 | 11 |
| B — Visualization | B1, B2, B3 | 3 | 14 |
| C — Consumer migration | C1, C2, C3, C4, C5, C6, C7, C8 (3 sub-commits), C9, C10 | 12 | 26 |
| D — Write path | D1 (5 sub-commits), D2, D3 (9 sub-commits), D4, D5 (5 sub-commits), D6, D7, D8 (3 sub-commits), D9 | 28 | 54 |
| E — Cleanup | E1, E2, E3, E4 | 3 (E4 verify-only) | 57 |

**Estimated total: ~57 commits** (within master plan §6 row #4's 50–70 range; on the lower end thanks to the gen schema being ~70% complete and shim adapters deferring pages migration).

**Sequencing rationale:**
- A0 first — load-bearing infra
- A1–A5 read functions; A6 dispatcher cutover after they exist
- B1/B2 visualization (independent of write path; can be parallel-dispatched)
- C1 (additive backend gen methods) BEFORE D2/D3/D5/D7 (which depend on `ctx.Backend.CreateEntityGen` etc.)
- C5 (mock audit) BEFORE C6 (fixture migration consumes new mock contract)
- C7 (pages-builder shim) prepares for priority #5 pages migration
- C10 (`api/` retire) requires §14 Q1 user decision FIRST
- D8 (backend repo writers) BEFORE D9 (dispatcher)
- E1–E3 strictly after both A6 and D9 dispatchers route to gen
- E3 conditional on Stage 4 — escape hatch documented

---

## §11 Coordination With Stage 4 Team

### Stage 4's territory (8 sdk/mpr files)
- `sdk/mpr/parser_domainmodel.go`, `sdk/mpr/writer_domainmodel.go`, `sdk/mpr/writer_domainmodel_test.go`
- `sdk/mpr/system_module.go` — **CRITICAL: blocks Stage 3.3.4 E3 deletion until Stage 4 rewrites this BSON-natively**
- `sdk/mpr/writer_modules.go`, `sdk/mpr/writer_security_test.go`, `sdk/mpr/writer_units.go`
- `sdk/mpr/reader_documents.go`

### Stage 3.3.4 commitment
**Stage 3.3.4 will NOT modify any file under `sdk/mpr/`.** E3 follows the conditional-delete pattern (deprecation header if Stage 4 hasn't landed).

### Risk of merge collision
Low. Stage 3.3.4 touches `mdl/`, `api/` (Q1-dependent), `modelsdk/property/` is unchanged. Stage 4 touches `sdk/mpr/`. **No file overlap.**

### Communication
The team-lead should mention to Stage 4:
1. **system_module.go is the critical blocker for E3 delete**. Coordinate timing.
2. The `mdl/backend/mpr/domainmodel_modelsdk.go` rebuild in D8 may surface BSON-shape differences against `sdk/mpr/writer_domainmodel.go::SerializeDomainModel`. Cross-test on shared fixture.
3. If Stage 4 rewrites `system_module.go` mid-execution, E3 unblocks and Stage 3.3.4 can run E3.S3a (delete) instead of E3.S3b (deprecation).

---

## §12 Self-Review Checklist (skill-required)

**Spec coverage:** §4 Phase A covers all read funcs in cmd_entities.go + cmd_entities_describe.go + cmd_associations.go + cmd_modules.go entity-walk + dispatcher (A6). §5 Phase B covers ELK + Mermaid (B1, B2, B3). §6 Phase C covers all 49 in-scope consumer files (catalog × 4, mock × 2 + 8 mock-test files, structure × 2, executor × 13 small + 3 pages shim, api/ × 1 conditional, security/workflows bridges × 1). §7 Phase D covers all entity/attribute/association/access-rule write paths + MOVE + 14 backend writer helpers. §8 Phase E covers cleanup with conditional E3. §11 Stage 4 boundary including system_module dependency. ✓

**Type consistency:** Gen accessor names verified against `modelsdk/gen/domainmodels/types.go` per §S2.4 with line numbers. TBD entries (DateAttributeType R7, CrossAssociation R11, EntitySource subtypes) flagged with explicit A0.S1 verification. The 14 attribute types enumerated; per-type sub-test in A2. ✓

**Risk surfacing:** R1 api/ public surface → §14 Q1 user decision required FIRST. R2 system_module dep → §11 + E3 escape hatch. R3 MemberAccess on FROM only → D6 explicit assertion. R4 AllowedModuleRoles version prefix → already-fixed Stage 3.3.1 D6a + D6 verification. R5 polymorphic AttributeType → A2 TypeName switch. R6 Index ordering deterministic → D8 lint. R7 DateAttributeType gap → A0 verify + dispatch on discriminator. R8 api/ external consumers → §14 Q1 + docs/01-project release note. R9 pages dependency → C7 shim adapters. R10 OQL inference → C8 with full OQL test suite gate. R11 CrossModuleAssociation gen mapping → A0 verify. R12 Stage 4 boundary (8 files) → §11. R13 cross-domain bridges → C3 (security) + C7 (pages). ✓

**TDD discipline:** Every task A0–A6, B1–B3, C1–C10, D1–D9, E1–E3 starts with "Step 1: Write failing test" + "Step 2: Confirm RED". A2 explicit per-attribute-type sub-tests. D3 explicit per-ALTER-op sub-tasks (9 sub-commits). D5 explicit per-association-kind sub-tasks. ✓

**Commit hygiene:** Each commit single-concern. A2 grouped attr types ~4 commits. D1 builders 5 commits. D3 ALTER ops 9 commits. D5 associations 5 commits. D8 backend writers 3 commits. Commit messages use HEREDOC. No `--no-verify`. ✓

**No public-API break without approval:** §14 Q1 explicitly flags `api/domainmodels.go` retire as REQUIRING USER APPROVAL before execution. Default = Option A retire per Stage 3.2 microflow precedent. ✓

**Cache discipline:** `listDomainModelsWithContainerGen` cache helper added in A0 BEFORE any consumer migration; every Phase D write commit pairs with `invalidateDomainModelsCache(ctx)`. ✓

---

## §13 Execution Notes (Wave Concurrency)

Lead executes the plan with the following concurrency strategy:

### Strictly serial (lead does directly)
- **A0** (repo + cache + ctx wiring) — load-bearing
- **A6, B3, D9** (dispatcher cutovers) — small but need upstream commits
- **C1** (additive FullBackend) — interface change
- **C10** (api/ retire) — requires §14 Q1 user decision
- **E1, E2, E3** (cleanup) — strictly serial

### Concurrent-safe (parallel teammates writing independent NEW files; lead serializes commits)
- **A1, A2 sub-tasks (per attr-type group), A3** — same new file `cmd_entities_gen.go`, single teammate sequential
- **A4** — `cmd_associations_gen.go`, can run in parallel with A1–A3 (different file)
- **A5** — `cmd_modules_gen.go` (new) or modify `cmd_modules.go`, parallel-safe
- **B1, B2** — `cmd_domainmodel_elk_gen.go` and `cmd_mermaid_gen.go` are independent; parallel-safe
- **C2, C3, C4, C5, C7, C8 sub-files, C9** — independent files; up to 4 parallel teammates
- **D1 sub-builders, D5 sub-associations** — same file, single teammate sequential
- **D3 sub-ALTER-ops, D8 sub-backend-writers** — same file each, single teammate sequential

### Critical / complex (single teammate + reviewer)
- **A3 (describeEntityGen)** — heavy formatter (mirrors workflow A3 / javaactions A3 complexity). Single teammate + lead `codex review`.
- **B1 (cmd_domainmodel_elk_gen)** — visualization byte-parity is critical. Single teammate + lead manual SVG diff.
- **D2 (execCreateEntityGen)** — round-trip with all 14 attr types. Single teammate + `mx check` smoke verification.
- **D6 (entity access rule gen)** — MemberAccess FROM-only assertion is load-bearing. Single teammate + lead manual hex inspection.
- **D8 (backend repo writers)** — BSON shape parity with legacy. Single teammate + cross-fixture diff.

### Safety rule
- Per memory `feedback_multi_agent_worktree_concurrency`: NEVER run two teammates concurrently with overlapping file targets.
- Per master plan §3.4: NEVER push during execution.

---

## §14 Open Questions for the User Before Execution Starts

1. **`api/domainmodels.go` retire vs in-place rebuild vs v2 sibling** (§S2.11 R1) — plan defaults to **Option A retire** per Stage 3.2 microflow precedent. Confirm BEFORE any C-phase commit. If Option B/C picked, C10 swells from 1 commit to ~5 and §9 acceptance criteria adjusts.

2. **System module dependency** (§S2.10 R2) — plan documents the **E3 escape hatch (deprecation header if Stage 4 not done)**. Confirm acceptable, or alternatively the user wants to coordinate Stage 4 to land `system_module.go` rewrite BEFORE Stage 3.3.4 E3 runs.

3. **gen schema gaps** (R7 DateAttributeType, R11 CrossAssociation) — A0.S1 produces verified gap table. Should A0.S1 produce a tracked memory `project_gen_schema_gaps_domainmodel.md`, or roll into existing `project_gen_schema_gaps.md`? Default = roll into existing.

4. **D3 ALTER ENTITY commit granularity** — plan estimates 9 sub-commits. Reviewer-friendlier to bundle small ones (set persistence + set documentation + set generalization → 1 commit; add/drop/rename/modify attribute → 4 commits; add/drop index → 1 commit) = 6 commits. Confirm preference. Default = 9 sub-commits for maximum reviewability.

5. **Visualization snapshot test medium** (B1, B2) — SVG byte snapshot vs. ELK/Mermaid intermediate-representation snapshot. SVG drifts on layout-engine upgrades; intermediate is more stable. Plan defaults to **intermediate-representation snapshot** per workflows §14 Q4.

These should be resolved BEFORE execution starts. A0–B2 can proceed without resolving (3) and (4); (1), (2), (5) are gates.
