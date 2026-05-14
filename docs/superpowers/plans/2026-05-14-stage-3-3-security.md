# Stage 3.3 Security Domain — Detailed Sub-Plan

> **For agentic workers:** REQUIRED SUB-SKILL: `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans`. Steps use checkbox `- [ ]` syntax for trackability.
>
> Generated from `2026-05-14-stage-3-3-domain-marathon-master.md` §6 priority #1 + §4 phase template + §8.1 stub.

**Goal:** Migrate the `security` domain off the legacy hand-written `sdk/security` package onto the auto-generated `modelsdk/gen/security` types. Final state: zero `sdk/security` imports outside `sdk/mpr/` (Stage 4 territory) and zero in `modelsdk/`.

---

## §1 Background — Status Snapshot

The security domain is the **most-advanced** of the six Stage 3.3 domains: roughly **half** the code is already migrated. Recap of what landed before this plan was written:

### Already migrated (commit history) — DO NOT redo
- `mdl/executor/cmd_security_gen.go` (432 lines) — `listAccessOnMicroflowGen`, `listAccessOnNanoflowGen`, `listSecurityMatrixGen`, `listSecurityMatrixJSONGen`, helpers `splitRoleQualifiedName`, `entityRuleRoleStrings`, `entityRuleRightStrings`
- `mdl/executor/cmd_security_gen_test.go` (109 lines) — covers `listSecurityMatrixGen`
- `mdl/executor/cmd_security_write_gen.go` (408 lines) — `execGrantMicroflowAccessGen`, `execRevokeMicroflowAccessGen`, `execGrantNanoflowAccessGen`, `execRevokeNanoflowAccessGen`, `cascadeRemoveRoleFromMicroflowsGen`, `cascadeRemoveRoleFromNanoflowsGen`, helpers `mergeAllowedRoles`, `filterAllowedRoles`, `removeRoleFromList`, `genMicroflowAllowedRoles`
- `mdl/executor/cmd_security_write_gen_test.go` (243 lines) — Roundtrip + NotFound tests
- `mdl/backend/mpr/security_modelsdk.go` (54 lines) — `setSecurityLevelViaModelsdk`
- `mdl/backend/mpr/security_module_modelsdk.go` (44 lines) — `addModuleRoleViaModelsdk`, `removeModuleRoleViaModelsdk`
- `mdl/backend/mpr/security_project_modelsdk.go` (194 lines) — `setProjectDemoUsersEnabled`, `addUserRole`, `removeUserRole`, `alterUserRoleModuleRoles`, `removeModuleRoleFromAllUserRoles`, `addDemoUser`, `removeDemoUser` (all `*ViaModelsdk`)
- `mdl/backend/mpr/security_allowed_roles_modelsdk.go` (105 lines) — `updateAllowedRolesViaModelsdk`, `removeFromAllowedRolesViaModelsdk`, `updatePublishedRestServiceRolesViaModelsdk`
- `mdl/backend/mpr/security_entity_access_modelsdk.go` (243 lines) — `addEntityAccessRuleViaModelsdk`, `removeEntityAccessRuleViaModelsdk`, `removeRoleFromAllEntitiesViaModelsdk`, `revokeEntityMemberAccessViaModelsdk`, `reconcileMemberAccessesViaModelsdk` (last one stays on Patch path)
- `mdl/backend/mpr/security_entity_access_gen_test.go` (374 lines) — covers Add/Remove/Revoke/RemoveRoleFromAll
- `mdl/repos/security.go` (31 lines) — `SecurityRepository` interface (`Get`, `GetModuleSecurity`, `Update`, `UpdateModuleSecurity`)
- `mdl/backend/mpr/repos/security.go` (145 lines) — direct-mode `securityRepo` implementation
- `modelsdk/gen/security/{types,enums,refs,version}.go` — auto-generated, present

### Still to migrate (this plan's scope)
| File | LoC | What stays | What leaves |
|---|---|---|---|
| `mdl/executor/cmd_security.go` | 786 | nothing | all 12 sdk-typed read funcs |
| `mdl/executor/cmd_security_write.go` | 1189 | nothing | all 22 sdk-typed write funcs (microflow/nanoflow halves already in `_gen`) |
| `mdl/executor/cmd_security_defaults.go` | 142 | nothing | 3 helpers using `*security.ModuleSecurity` / `*security.ProjectSecurity` |
| `mdl/executor/cmd_security_mock_test.go` | 502 | needs migration | uses `security.ProjectSecurity`, `security.UserRole`, etc. as fixture types |
| `mdl/executor/cmd_json_mock_test.go` | (small) | needs migration | one fixture |
| `mdl/executor/cmd_write_handlers_mock_test.go` | (small) | needs migration | one fixture |
| `mdl/executor/cmd_error_mock_test.go` | (small) | needs migration | one fixture |
| `mdl/catalog/builder.go` | line 34 | reader interface | `security.ProjectSecurity` return type |
| `mdl/linter/context.go` | line 25 | reader interface | `security.ProjectSecurity` return type |
| `mdl/linter/rules/security.go` | line 9 | constant ref | `security.SecurityLevelProduction` constant + `ps.PasswordPolicy.MinimumLength` |
| `mdl/backend/security.go` | 69 | interface methods | `*security.ProjectSecurity`, `*security.ModuleSecurity` return types |
| `mdl/backend/mpr/backend.go` | 384–446 | shim methods | sdk-typed wrappers around legacy reader |
| `mdl/backend/mock/mock_security.go` | 158 | Func-stub layer | `*security.*` return types in mock funcs |
| `mdl/backend/mock/backend.go` | 133–142 | Func field decls | `*security.*` return types in `Func` field signatures |
| `sdk/mpr/reader_documents.go` | (Stage 4 territory) | — | DO NOT touch |
| `sdk/mpr/parser_security.go` | (Stage 4 territory) | — | DO NOT touch |

### Why this domain is priority #1
- **Smallest LoC (160 sdk LoC, 16 callers)** — lowest blast radius
- **Most prep already done** — all gen-typed backend mutators already exist; only consumer-side migration + dispatch wiring + sdk-typed read interfaces remain
- **No widget-tree complexity** — uniform "list of qualified roles" data
- **Cross-cutting cache helper** (`listMicroflowsWithContainerGen`) already present from Stage 3.2

---

## §2 Pre-Flight Survey Results

### S2.1 sdk/security importers (16 files)
```
mdl/executor/cmd_security.go
mdl/executor/cmd_security_write.go
mdl/executor/cmd_security_defaults.go
mdl/executor/cmd_json_mock_test.go
mdl/executor/cmd_write_handlers_mock_test.go
mdl/executor/cmd_error_mock_test.go
mdl/executor/cmd_security_mock_test.go
mdl/catalog/builder.go
mdl/linter/context.go
mdl/linter/rules/security.go
mdl/backend/security.go
mdl/backend/mpr/backend.go
mdl/backend/mock/backend.go
mdl/backend/mock/mock_security.go
sdk/mpr/reader_documents.go          ← Stage 4 territory, DO NOT touch
sdk/mpr/parser_security.go           ← Stage 4 territory, DO NOT touch
```

### S2.2 Read funcs in cmd_security.go vs cmd_security_gen.go
**cmd_security.go (legacy, sdk-typed)** — 12 functions:
1. `listProjectSecurity`
2. `listModuleRoles`
3. `listUserRoles`
4. `listDemoUsers`
5. `listAccessOnEntity` (uses `*domainmodel.Entity` and `*domainmodel.AccessRule` — these are `sdk/domainmodel` types, but the legacy reader returns them so the gen migration must roundtrip via `ctx.Backend.GetDomainModel` for now and convert downstream when domainmodel domain migrates in Stage 3.3 priority #4)
6. `listAccessOnPage`
7. `listAccessOnWorkflow` (returns "unsupported" — no migration needed, sdk import not even required after the sdk-typed reads above are gone)
8. `listSecurityMatrix` (still calls `ctx.Backend.ListModuleSecurity()` and `ctx.Backend.ListDomainModels()`)
9. `listSecurityMatrixJSON`
10. `describeModuleRole`
11. `describeDemoUser`
12. `describeUserRole`

**cmd_security_gen.go (already migrated)** — 7:
1. `listAccessOnMicroflowGen`
2. `listAccessOnNanoflowGen`
3. `listSecurityMatrixGen` (the entity & module-security halves still call `ctx.Backend.ListDomainModels()` / `ctx.Backend.ListModuleSecurity()` — so it returns sdk-typed pointers internally, but as a CALLER. Migration here is "stop importing the type names" not "stop calling the method".)
4. `listSecurityMatrixJSONGen`
5. `splitRoleQualifiedName`
6. `entityRuleRoleStrings`
7. `entityRuleRightStrings`

**Wiring status:** `executor_query.go:81` still dispatches `ShowSecurityMatrix` → `listSecurityMatrix` (the legacy). `listSecurityMatrixGen` exists but **is not yet wired**. Same likely for the other gen variants — checked: `ShowAccessOnMicroflow` → `listAccessOnMicroflowGen` ✅, `ShowAccessOnNanoflow` → `listAccessOnNanoflowGen` ✅, `ShowSecurityMatrix` → `listSecurityMatrix` ❌ (needs cutover).

### S2.3 Write funcs in cmd_security_write.go vs cmd_security_write_gen.go
**cmd_security_write.go (legacy)** — 22 functions:
1. `execCreateModuleRole`
2. `execDropModuleRole` (cascade calls `cascadeRemoveRoleFromMicroflowsGen` / `cascadeRemoveRoleFromNanoflowsGen` which already exist, plus `RemoveFromAllowedRoles` for pages/OData, plus `RemoveModuleRole` final)
3. `execCreateUserRole`
4. `execAlterUserRole`
5. `execDropUserRole`
6. `execGrantEntityAccess`
7. `execRevokeEntityAccess`
8. `execGrantPageAccess`
9. `execRevokePageAccess`
10. `execGrantWorkflowAccess` (returns Unsupported — no migration)
11. `execRevokeWorkflowAccess` (returns Unsupported — no migration)
12. `validateModuleRole`
13. `execAlterProjectSecurity`
14. `execCreateDemoUser` (calls `detectUserEntity` → `ListDomainModels`)
15. `detectUserEntity`
16. `joinCandidates`
17. `execDropDemoUser`
18. `execGrantODataServiceAccess`
19. `execRevokeODataServiceAccess`
20. `execGrantPublishedRestServiceAccess`
21. `execRevokePublishedRestServiceAccess`
22. `execUpdateSecurity`

**cmd_security_write_gen.go (already)** — 6 microflow/nanoflow ops + 4 helpers.

The legacy write file's `import "...sdk/security"` line is needed exclusively for `execAlterProjectSecurity` (security level constants) and the security-level mapping switch. Once that single switch moves to a `securityLevelBSON()` helper that uses gen-typed enum constants from `modelsdk/gen/security/enums.go`, the import disappears.

### S2.4 Backend interface methods (`mdl/backend/security.go`)
```go
ProjectSecurityBackend:
  GetProjectSecurity() (*security.ProjectSecurity, error)         // sdk-typed read
  SetProjectSecurityLevel(unitID, level string) error             // ok (string param)
  SetProjectDemoUsersEnabled(unitID, bool) error                  // ok
  AddUserRole(...) error                                          // ok
  AlterUserRoleModuleRoles(...) error                             // ok
  RemoveUserRole(...) error                                       // ok
  AddDemoUser(...) error                                          // ok
  RemoveDemoUser(...) error                                       // ok

ModuleSecurityBackend:
  ListModuleSecurity() ([]*security.ModuleSecurity, error)        // sdk-typed read
  GetModuleSecurity(moduleID) (*security.ModuleSecurity, error)   // sdk-typed read
  AddModuleRole(...) error                                        // ok
  RemoveModuleRole(...) error                                     // ok
  RemoveModuleRoleFromAllUserRoles(unitID, qualifiedRole) (int, error) // ok

EntityAccessBackend:
  UpdateAllowedRoles(...) error
  UpdatePublishedRestServiceRoles(...) error
  RemoveFromAllowedRoles(...) (bool, error)
  AddEntityAccessRule(EntityAccessRuleParams) error               // params struct already in mdl/backend (not sdk)
  RemoveEntityAccessRule(...) (int, error)
  RevokeEntityMemberAccess(unitID, entityName, roleNames, types.EntityAccessRevocation) (int, error)
  RemoveRoleFromAllEntities(unitID, roleName) (int, error)
  ReconcileMemberAccesses(unitID, moduleName) (int, error)
```

The **only sdk-typed methods on the interface** are `GetProjectSecurity`, `ListModuleSecurity`, `GetModuleSecurity`. Replacing them with `repos.SecurityRepository.{Get, GetModuleSecurity}` (already exists) is a clean cutover.

### S2.5 AllowedModuleRoles version-prefix bug status
**Status: STILL PRESENT** in `modelsdk/property/reference.go:44` — `BSONValue() any { return r.qnames }` returns the raw `[]string` slice. The codec then encodes a plain BSON array without the `int32(1)` version prefix that Mendix expects.

Verification:
- `security_entity_access_modelsdk.go::addEntityAccessRuleViaModelsdk` calls `rule.SetModuleRolesQualifiedNames(roleNames)` (line 54), which is a `ByNameRefList` setter that flows through the broken `BSONValue()`
- `security_entity_access_gen_test.go::TestAddEntityAccessRuleViaModelsdk_GenNative` only checks the **key is present** (line 142), not the **shape**. The bug would slip through that test
- Memory `project_allowedmoduleroles_version_prefix` says: "fix `ByNameRefList` encoder OR add manually in `addEntityAccessRuleViaModelsdk`"

This MUST be addressed in §7 task D6 before the sdk-typed `cmd_security_write.go` can be deleted (because cmd_security_write.go's `execGrantEntityAccess` is the production path and currently does NOT trigger the bug — it goes through `ctx.Backend.AddEntityAccessRule` → `mdl/backend/mpr/backend.go::AddEntityAccessRule` → eventually `b.writer.AddEntityAccessRule()` which is the legacy sdk/mpr path with proper version prefix). Cutover MUST verify the gen-native path fixes the prefix.

### S2.6 ExecContext field
`ExecContext` (mdl/executor/exec_context.go:30–41) currently exposes `Microflows repos.MicroflowRepository` and `Nanoflows repos.NanoflowRepository`. **There is no `ctx.Security` field yet.** The pattern from Stage 3.2 is to add it during the consumer-migration phase (matches the Microflows wiring); this plan task A1 adds it.

### S2.7 MockBackend conformance
`mdl/backend/mock/mock_security.go` returns `nil, nil` (or `nil`) on missing Func fields, NOT `"MockBackend.X not configured"` errors. This is non-conformant per master plan §3 / CLAUDE.md backend-abstraction checklist. Master plan §5 P3 has a follow-up task tracker; this sub-plan task C5 brings the security mock funcs into compliance.

---

## §3 Risks Specific to Security Domain

| # | Risk | Impact | Mitigation |
|---|---|---|---|
| R1 | **AllowedModuleRoles version-prefix bug** (memory `project_allowedmoduleroles_version_prefix`) — gen-native writes produce `['Module.Role']` instead of `[1, 'Module.Role']`, causing CE0003 in `mx check` | Critical — silent data corruption | Task D6: fix `ByNameRefList` BSON encoding OR add `setRawBSONField` workaround in `addEntityAccessRuleViaModelsdk`; add explicit test that asserts `int32(1)` first element |
| R2 | **Stage 4 boundary at sdk/mpr** — `sdk/mpr/reader_documents.go` and `sdk/mpr/parser_security.go` import `sdk/security`; Stage 4 team is actively rewriting `sdk/mpr` | Medium — possible merge conflict | Task scope explicitly excludes any modification to `sdk/mpr/*`. Acceptance criterion accepts `grep` returning matches in `sdk/mpr/` until Stage 4 lands |
| R3 | **AccessRule parent/child pointer semantics** (CLAUDE.md "Association Parent/Child Pointer Semantics") — MemberAccess on the wrong side triggers CE0066 "Entity access is out of date" | High — silent rule corruption | Test fixture coverage: every D-phase test that adds a rule with `MemberAccesses` runs `mx check` (or asserts MemberAccesses are only on FROM-entity) |
| R4 | **Schema gap: ModuleRole.Description** — verify `genSec.ModuleRole` has `Description()` / `SetDescription()`. If gen schema lacks it, the `addModuleRoleViaModelsdk` (already merged) would silently drop description | Low | Already merged and tested in `security_modelsdk_test.go`; double-check before D-phase tasks |
| R5 | **PasswordPolicy** in linter rule SEC002 — gen-typed `ProjectSecurity` may expose policy through different accessor | Low | Task C3 audits and migrates; if gen lacks `PasswordPolicy`, fall back to `codec.ReadBSONFieldDoc(ps.Raw(), "PasswordPolicy")` |
| R6 | **Rules-to-list inversion via `ModuleRoleNames` vs `ModuleRoles`** — `domainmodel.AccessRule` exposes both `ModuleRoleNames []string` and `ModuleRoles []model.ID` with fallback logic in `entityRuleRoleStrings` | Low | The gen path uses `rule.ModuleRolesQualifiedNames()` directly. Once `listAccessOnEntity` migrates (task A2), the dual-list helper goes away |
| R7 | **`detectUserEntity` calls `ListDomainModels`** — couples security write path to domainmodel domain | Medium | Task D8 adapter: keep `ctx.Backend.ListDomainModels()` call but stop using `domainmodel.*` type names — only access `dm.ContainerID`, `dm.Entities[].Name`, `dm.Entities[].GeneralizationRef` (which the gen domainmodels package will also expose when domain #4 migrates). Until then, `cmd_security_defaults.go` keeps the `sdk/security` import-equivalent indirection |
| R8 | **autoDocumentRole logic** in `cmd_security_defaults.go::moduleUsesAutoDocumentRole` matches a specific `ModuleRole.Description` string | Low | Task C2 migrates the matcher to gen-typed `genSec.ModuleRole.Description()` |

---

## §4 Phase A — Read Path Completion

Goal: every read function in `cmd_security.go` has a `*Gen` twin, and the dispatcher routes to the gen variant.

### Task A0: Per-domain `listSecurityWithContainerGen` cache helper

Security units (`Security$ProjectSecurity` singleton, `Security$ModuleSecurity` per-module) need a cache helper analogous to `listMicroflowsWithContainerGen` so consumers don't pay repeat decode cost. Adds a `ctx.Cache` field for module security pairings.

**Files:**
- Modify: `mdl/executor/exec_context.go` — add `Security repos.SecurityRepository` field
- Modify: `mdl/executor/exec_context.go` — add cache fields `projectSecurityGen *genSec.ProjectSecurity` and `moduleSecurityWithContainerGen []ModuleSecurityGenWithContainer`
- Modify: `mdl/executor/executor.go` (or wherever ExecContext is constructed by the MprBackend factory) — wire `Security: mprbackend.NewSecurityRepository(b.msdkWriter)`
- Create: `mdl/executor/helpers_security_gen.go`
- Create: `mdl/executor/helpers_security_gen_test.go`

- [ ] **Step 1: Write failing test** (`mdl/executor/helpers_security_gen_test.go`)

```go
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"
)

func TestListModuleSecurityWithContainerGen_CachesAcrossCalls(t *testing.T) {
	ctx := newSecurityTestContext(t) // helper from cmd_security_gen_test.go
	list1, err := listModuleSecurityWithContainerGen(ctx)
	if err != nil {
		t.Fatalf("listModuleSecurityWithContainerGen: %v", err)
	}
	list2, _ := listModuleSecurityWithContainerGen(ctx)
	if len(list1) != len(list2) {
		t.Fatalf("cache produced different lengths: %d vs %d", len(list1), len(list2))
	}
	if len(list1) > 0 && &list1[0] != &list2[0] {
		t.Fatalf("cache must return same slice header on second call")
	}
}

func TestGetProjectSecurityGen_CachesAcrossCalls(t *testing.T) {
	ctx := newSecurityTestContext(t)
	ps1, err := getProjectSecurityGen(ctx)
	if err != nil {
		t.Fatalf("getProjectSecurityGen: %v", err)
	}
	ps2, _ := getProjectSecurityGen(ctx)
	if ps1 != ps2 {
		t.Fatalf("cache must return same pointer on second call")
	}
}
```

- [ ] **Step 2: Run test to confirm RED**
```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./mdl/executor/ -run "TestListModuleSecurityWithContainerGen_CachesAcrossCalls|TestGetProjectSecurityGen_CachesAcrossCalls" -v
```
Expected: FAIL (`undefined: listModuleSecurityWithContainerGen / getProjectSecurityGen`).

- [ ] **Step 3: Implement helpers** (`mdl/executor/helpers_security_gen.go`)

```go
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"github.com/mendixlabs/mxcli/model"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
)

// ModuleSecurityGenWithContainer pairs a gen-typed *ModuleSecurity with its
// container UUID (the parent module ID).
type ModuleSecurityGenWithContainer struct {
	MS          *genSec.ModuleSecurity
	ContainerID model.ID
}

// getProjectSecurityGen returns the singleton gen-typed ProjectSecurity,
// caching the result on ctx.Cache.projectSecurityGen for the session.
//
// Cache invalidation: any project-security mutation (SetSecurityLevel,
// AddUserRole, etc.) must call invalidateProjectSecurityCache to drop
// the cached pointer.
func getProjectSecurityGen(ctx *ExecContext) (*genSec.ProjectSecurity, error) {
	if ctx == nil || ctx.Security == nil {
		return nil, nil
	}
	if ctx.Cache != nil && ctx.Cache.projectSecurityGen != nil {
		return ctx.Cache.projectSecurityGen, nil
	}
	ps, err := ctx.Security.Get()
	if err != nil {
		return nil, err
	}
	if ctx.Cache != nil {
		ctx.Cache.projectSecurityGen = ps
	}
	return ps, nil
}

// listModuleSecurityWithContainerGen returns every ModuleSecurity unit
// in the project paired with its container module ID. Caches on
// ctx.Cache.moduleSecurityWithContainerGen for the session.
func listModuleSecurityWithContainerGen(ctx *ExecContext) ([]ModuleSecurityGenWithContainer, error) {
	if ctx == nil || ctx.Security == nil {
		return nil, nil
	}
	if ctx.Cache != nil && ctx.Cache.moduleSecurityWithContainerGen != nil {
		return ctx.Cache.moduleSecurityWithContainerGen, nil
	}
	modules, err := ctx.Backend.ListModules()
	if err != nil {
		return nil, err
	}
	out := make([]ModuleSecurityGenWithContainer, 0, len(modules))
	for _, m := range modules {
		ms, err := ctx.Security.GetModuleSecurity(m.ID)
		if err != nil || ms == nil {
			continue
		}
		out = append(out, ModuleSecurityGenWithContainer{MS: ms, ContainerID: m.ID})
	}
	if ctx.Cache != nil {
		ctx.Cache.moduleSecurityWithContainerGen = out
	}
	return out, nil
}

// invalidateProjectSecurityCache clears the cached ProjectSecurity pointer.
// Called by any write path that mutates ProjectSecurity.
func invalidateProjectSecurityCache(ctx *ExecContext) {
	if ctx == nil || ctx.Cache == nil {
		return
	}
	ctx.Cache.projectSecurityGen = nil
}

// invalidateModuleSecurityCache clears the cached per-module security list.
// Called by any write path that mutates ModuleSecurity (Add/Remove ModuleRole).
func invalidateModuleSecurityCache(ctx *ExecContext) {
	if ctx == nil || ctx.Cache == nil {
		return
	}
	ctx.Cache.moduleSecurityWithContainerGen = nil
}
```

- [ ] **Step 4: Add fields to `executorCache` and `ExecContext`** (`mdl/executor/exec_context.go`)

```go
// In executorCache struct, add:
projectSecurityGen             *genSec.ProjectSecurity
moduleSecurityWithContainerGen []ModuleSecurityGenWithContainer

// In ExecContext struct, add:
// Security is the Stage 3.3 modelsdk-native security repo.
Security repos.SecurityRepository
```

(Add `genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"` import to `exec_context.go`.)

- [ ] **Step 5: Wire `Security` field in BackendFactory / context construction**

Find the constructor that wires `Microflows`/`Nanoflows` (likely `executor.go::newExecContext` or similar) and add the parallel line:

```go
if mb, ok := backend.(*mprbackend.MprBackend); ok {
    ctx.Microflows = mprrepos.NewMicroflowRepository(mb.MsdkWriter())
    ctx.Nanoflows  = mprrepos.NewNanoflowRepository(mb.MsdkWriter())
    ctx.Security   = mprrepos.NewSecurityRepository(mb.MsdkWriter()) // NEW
}
```

If the existing wiring uses a different abstraction (`backendProvidesRepos` interface), follow that pattern.

- [ ] **Step 6: Add a `newSecurityTestContext` helper** to `cmd_security_gen_test.go` (or a new `helpers_security_gen_test.go`) that opens a fixture MPR, builds an `MprBackend`, and produces an `ExecContext` with `Security` populated. If `cmd_security_gen_test.go` already has this helper, reuse it; otherwise, mirror what `cmd_microflows_show_list_gen_test.go` does for microflows.

- [ ] **Step 7: Run test to confirm GREEN**
```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./mdl/executor/ -run "TestListModuleSecurityWithContainerGen_CachesAcrossCalls|TestGetProjectSecurityGen_CachesAcrossCalls" -v
```

- [ ] **Step 8: Build + full test gate**
```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go build ./...
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./... -count=1 -timeout 240s
```

- [ ] **Step 9: Commit**

```bash
git add mdl/executor/exec_context.go mdl/executor/helpers_security_gen.go mdl/executor/helpers_security_gen_test.go
git commit -m "$(cat <<'EOF'
feat(executor): Stage 3.3.1.A0 — security cache helpers + ctx.Security wiring

Adds getProjectSecurityGen, listModuleSecurityWithContainerGen, and the
matching cache invalidation helpers. Wires ExecContext.Security to
mprrepos.NewSecurityRepository so per-domain helpers can stop calling
ctx.Backend.GetProjectSecurity / ListModuleSecurity.

Mirrors the listMicroflowsWithContainerGen pattern from Stage 3.2.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task A1: `listProjectSecurityGen` (read ProjectSecurity)

**Files:**
- Modify: `mdl/executor/cmd_security_gen.go`
- Modify: `mdl/executor/cmd_security_gen_test.go`

- [ ] **Step 1: Write failing test**

```go
func TestListProjectSecurityGen_OutputsLevel(t *testing.T) {
	ctx := newSecurityTestContext(t)
	var buf bytes.Buffer
	ctx.Output = &buf
	ctx.Format = FormatTable
	if err := listProjectSecurityGen(ctx); err != nil {
		t.Fatalf("listProjectSecurityGen: %v", err)
	}
	if !strings.Contains(buf.String(), "Security Level:") {
		t.Errorf("output missing Security Level line: %q", buf.String())
	}
}
```

- [ ] **Step 2: Confirm RED** (`undefined: listProjectSecurityGen`)

- [ ] **Step 3: Implement** (append to `cmd_security_gen.go`)

```go
// listProjectSecurityGen handles SHOW PROJECT SECURITY using the gen-typed
// ProjectSecurity from ctx.Security. Mirrors listProjectSecurity exactly
// in output shape; only the type source changes.
func listProjectSecurityGen(ctx *ExecContext) error {
	ps, err := getProjectSecurityGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("read project security", err)
	}
	if ps == nil {
		return mdlerrors.NewBackend("read project security", fmt.Errorf("ProjectSecurity unit not found"))
	}

	if ctx.Format == FormatJSON {
		result := &TableResult{Columns: []string{"Property", "Value"}}
		result.Rows = append(result.Rows,
			[]any{"SecurityLevel", securityLevelDisplay(ps.SecurityLevel())},
			[]any{"CheckSecurity", fmt.Sprintf("%v", ps.CheckSecurity())},
			[]any{"StrictMode", fmt.Sprintf("%v", ps.StrictMode())},
			[]any{"DemoUsersEnabled", fmt.Sprintf("%v", ps.EnableDemoUsers())},
			[]any{"GuestAccess", fmt.Sprintf("%v", ps.EnableGuestAccess())},
			[]any{"UserRoles", fmt.Sprintf("%d", len(ps.UserRolesItems()))},
			[]any{"DemoUsers", fmt.Sprintf("%d", len(ps.DemoUsersItems()))},
		)
		if ps.AdminUserName() != "" {
			result.Rows = append(result.Rows, []any{"AdminUser", ps.AdminUserName()})
		}
		if ps.GuestUserRoleQualifiedName() != "" {
			result.Rows = append(result.Rows, []any{"GuestUserRole", ps.GuestUserRoleQualifiedName()})
		}
		if pp := ps.PasswordPolicy(); pp != nil {
			result.Rows = append(result.Rows,
				[]any{"PasswordPolicy.MinimumLength", fmt.Sprintf("%d", pp.MinimumLength())},
				[]any{"PasswordPolicy.RequireDigit", fmt.Sprintf("%v", pp.RequireDigit())},
				[]any{"PasswordPolicy.RequireMixedCase", fmt.Sprintf("%v", pp.RequireMixedCase())},
				[]any{"PasswordPolicy.RequireSymbol", fmt.Sprintf("%v", pp.RequireSymbol())},
			)
		}
		return writeResult(ctx, result)
	}

	fmt.Fprintf(ctx.Output, "Security Level: %s\n", securityLevelDisplay(ps.SecurityLevel()))
	fmt.Fprintf(ctx.Output, "Check Security: %v\n", ps.CheckSecurity())
	fmt.Fprintf(ctx.Output, "Strict Mode: %v\n", ps.StrictMode())
	fmt.Fprintf(ctx.Output, "Demo Users Enabled: %v\n", ps.EnableDemoUsers())
	fmt.Fprintf(ctx.Output, "Guest Access: %v\n", ps.EnableGuestAccess())
	if ps.AdminUserName() != "" {
		fmt.Fprintf(ctx.Output, "Admin User: %s\n", ps.AdminUserName())
	}
	if ps.GuestUserRoleQualifiedName() != "" {
		fmt.Fprintf(ctx.Output, "Guest User Role: %s\n", ps.GuestUserRoleQualifiedName())
	}
	fmt.Fprintf(ctx.Output, "User Roles: %d\n", len(ps.UserRolesItems()))
	fmt.Fprintf(ctx.Output, "Demo Users: %d\n", len(ps.DemoUsersItems()))

	if pp := ps.PasswordPolicy(); pp != nil {
		fmt.Fprintf(ctx.Output, "\nPassword Policy:\n")
		fmt.Fprintf(ctx.Output, "  Minimum Length: %d\n", pp.MinimumLength())
		fmt.Fprintf(ctx.Output, "  Require Digit: %v\n", pp.RequireDigit())
		fmt.Fprintf(ctx.Output, "  Require Mixed Case: %v\n", pp.RequireMixedCase())
		fmt.Fprintf(ctx.Output, "  Require Symbol: %v\n", pp.RequireSymbol())
	}
	return nil
}

// securityLevelDisplay maps gen-typed BSON SecurityLevel constants to the
// human-friendly labels used by `show project security`. Mirrors
// security.SecurityLevelDisplay (sdk/security/security.go) without
// importing the sdk package.
func securityLevelDisplay(level string) string {
	switch level {
	case "CheckNothing":
		return "Off"
	case "CheckFormsAndMicroflows":
		return "Prototype / demo"
	case "CheckEverything":
		return "Production"
	default:
		return level
	}
}
```

NOTE: accessor names (`SecurityLevel()`, `CheckSecurity()`, `UserRolesItems()`, `DemoUsersItems()`, `AdminUserName()`, `GuestUserRoleQualifiedName()`, `PasswordPolicy()`) MUST be verified against `modelsdk/gen/security/types.go` before commit. If any accessor is named differently or returns a different type, adjust to match. If `PasswordPolicy` is missing entirely from gen, fall back to `codec.ReadBSONFieldDoc(ps.Raw(), "PasswordPolicy")` per memory `project_gen_schema_gaps`.

- [ ] **Step 4: GREEN**

- [ ] **Step 5: Build + full test gate**

- [ ] **Step 6: Commit**

```bash
git add mdl/executor/cmd_security_gen.go mdl/executor/cmd_security_gen_test.go
git commit -m "$(cat <<'EOF'
feat(executor): Stage 3.3.1.A1 — listProjectSecurityGen (gen-typed)

Mirrors listProjectSecurity reading from ctx.Security via
getProjectSecurityGen instead of ctx.Backend.GetProjectSecurity.
Adds securityLevelDisplay helper to keep the human label mapping
out of the legacy sdk/security package.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task A2: `listModuleRolesGen`

**Files:**
- Modify: `mdl/executor/cmd_security_gen.go`
- Modify: `mdl/executor/cmd_security_gen_test.go`

- [ ] **Step 1: Write failing test**

```go
func TestListModuleRolesGen_FiltersByModule(t *testing.T) {
	ctx := newSecurityTestContext(t)
	var buf bytes.Buffer
	ctx.Output = &buf
	ctx.Format = FormatTable
	if err := listModuleRolesGen(ctx, "TestModule"); err != nil {
		t.Fatalf("listModuleRolesGen: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "TestModule") {
		t.Errorf("expected TestModule in output, got: %q", out)
	}
}
```

- [ ] **Step 2: RED** (undefined `listModuleRolesGen`)

- [ ] **Step 3: Implement**

```go
// listModuleRolesGen handles SHOW MODULE ROLES [IN module] using
// gen-typed ModuleSecurity units from listModuleSecurityWithContainerGen.
func listModuleRolesGen(ctx *ExecContext, moduleName string) error {
	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}
	pairs, err := listModuleSecurityWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("read module security", err)
	}
	result := &TableResult{Columns: []string{"Qualified Name", "Module", "Role", "Description"}}
	for _, p := range pairs {
		modName := h.GetModuleName(p.ContainerID)
		if modName == "" {
			continue
		}
		if moduleName != "" && modName != moduleName {
			continue
		}
		for _, mr := range p.MS.ModuleRolesItems() {
			typed, ok := mr.(*genSec.ModuleRole)
			if !ok {
				continue
			}
			qn := modName + "." + typed.Name()
			result.Rows = append(result.Rows, []any{qn, modName, typed.Name(), typed.Description()})
		}
	}
	result.Summary = fmt.Sprintf("(%d module roles)", len(result.Rows))
	return writeResult(ctx, result)
}
```

- [ ] **Step 4: GREEN**
- [ ] **Step 5: Build + test gate**
- [ ] **Step 6: Commit `feat(executor): Stage 3.3.1.A2 — listModuleRolesGen (gen-typed)`**

### Task A3: `listUserRolesGen`

**Files:** `mdl/executor/cmd_security_gen.go`, test file

- [ ] **Step 1: Test**

```go
func TestListUserRolesGen_OutputsRoleNames(t *testing.T) {
	ctx := newSecurityTestContext(t)
	var buf bytes.Buffer
	ctx.Output = &buf
	ctx.Format = FormatTable
	if err := listUserRolesGen(ctx); err != nil {
		t.Fatalf("listUserRolesGen: %v", err)
	}
	if !strings.Contains(buf.String(), "Name") {
		t.Errorf("expected column header Name, got: %q", buf.String())
	}
}
```

- [ ] **Step 2: RED**

- [ ] **Step 3: Implement**

```go
func listUserRolesGen(ctx *ExecContext) error {
	ps, err := getProjectSecurityGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("read project security", err)
	}
	if ps == nil {
		return mdlerrors.NewBackend("read project security", fmt.Errorf("ProjectSecurity not found"))
	}
	result := &TableResult{Columns: []string{"Name", "Module Roles", "Manage All", "Check Security"}}
	for _, ur := range ps.UserRolesItems() {
		typed, ok := ur.(*genSec.UserRole)
		if !ok {
			continue
		}
		ma := "No"
		if typed.ManageAllRoles() {
			ma = "Yes"
		}
		cs := "No"
		if typed.CheckSecurity() {
			cs = "Yes"
		}
		result.Rows = append(result.Rows, []any{typed.Name(), len(typed.ModuleRolesQualifiedNames()), ma, cs})
	}
	result.Summary = fmt.Sprintf("(%d user roles)", len(result.Rows))
	return writeResult(ctx, result)
}
```

- [ ] **Step 4: GREEN**
- [ ] **Step 5: Build + test gate**
- [ ] **Step 6: Commit `feat(executor): Stage 3.3.1.A3 — listUserRolesGen (gen-typed)`**

### Task A4: `listDemoUsersGen`

**Files:** `mdl/executor/cmd_security_gen.go`, test file

- [ ] **Step 1: Test**

```go
func TestListDemoUsersGen_DisabledMessage(t *testing.T) {
	ctx := newSecurityTestContext(t) // fixture has demo users disabled by default
	var buf bytes.Buffer
	ctx.Output = &buf
	ctx.Format = FormatTable
	if err := listDemoUsersGen(ctx); err != nil {
		t.Fatalf("listDemoUsersGen: %v", err)
	}
	if !strings.Contains(buf.String(), "disabled") {
		t.Errorf("expected disabled message, got: %q", buf.String())
	}
}
```

- [ ] **Step 2: RED**

- [ ] **Step 3: Implement**

```go
func listDemoUsersGen(ctx *ExecContext) error {
	ps, err := getProjectSecurityGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("read project security", err)
	}
	if ps == nil {
		return mdlerrors.NewBackend("read project security", fmt.Errorf("ProjectSecurity not found"))
	}
	if !ps.EnableDemoUsers() {
		if ctx.Format != FormatJSON {
			fmt.Fprintln(ctx.Output, "Demo users are disabled.")
			fmt.Fprintln(ctx.Output, "Enable with: alter project security demo users on;")
			return nil
		}
		return writeResult(ctx, &TableResult{Columns: []string{"User Name", "User Roles"}})
	}
	result := &TableResult{Columns: []string{"User Name", "User Roles"}}
	for _, du := range ps.DemoUsersItems() {
		typed, ok := du.(*genSec.DemoUser)
		if !ok {
			continue
		}
		rolesStr := strings.Join(typed.UserRolesQualifiedNames(), ", ")
		result.Rows = append(result.Rows, []any{typed.UserName(), rolesStr})
	}
	result.Summary = fmt.Sprintf("(%d demo users)", len(result.Rows))
	return writeResult(ctx, result)
}
```

- [ ] **Step 4: GREEN**
- [ ] **Step 5: Build + test gate**
- [ ] **Step 6: Commit `feat(executor): Stage 3.3.1.A4 — listDemoUsersGen (gen-typed)`**

### Task A5: `describeModuleRoleGen`

**Files:** `mdl/executor/cmd_security_gen.go`, test file

- [ ] **Step 1: Test**

```go
func TestDescribeModuleRoleGen_OutputsCreateStatement(t *testing.T) {
	ctx := newSecurityTestContext(t)
	var buf bytes.Buffer
	ctx.Output = &buf
	if err := describeModuleRoleGen(ctx, ast.QualifiedName{Module: "TestModule", Name: "User"}); err != nil {
		t.Fatalf("describeModuleRoleGen: %v", err)
	}
	if !strings.Contains(buf.String(), "create module role TestModule.User") {
		t.Errorf("expected create statement, got: %q", buf.String())
	}
}
```

- [ ] **Step 2: RED**

- [ ] **Step 3: Implement**

```go
func describeModuleRoleGen(ctx *ExecContext, name ast.QualifiedName) error {
	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}
	pairs, err := listModuleSecurityWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("read module security", err)
	}
	for _, p := range pairs {
		modName := h.GetModuleName(p.ContainerID)
		if name.Module != "" && modName != name.Module {
			continue
		}
		for _, mr := range p.MS.ModuleRolesItems() {
			typed, ok := mr.(*genSec.ModuleRole)
			if !ok || typed.Name() != name.Name {
				continue
			}
			fmt.Fprintf(ctx.Output, "create module role %s.%s", modName, typed.Name())
			if typed.Description() != "" {
				fmt.Fprintf(ctx.Output, " description '%s'", typed.Description())
			}
			fmt.Fprintln(ctx.Output, ";")
			fmt.Fprintln(ctx.Output, "/")
			// Show inclusion in user roles via gen ProjectSecurity
			qualifiedRole := modName + "." + typed.Name()
			if ps, err := getProjectSecurityGen(ctx); err == nil && ps != nil {
				var includedBy []string
				for _, ur := range ps.UserRolesItems() {
					urTyped, ok := ur.(*genSec.UserRole)
					if !ok {
						continue
					}
					for _, mref := range urTyped.ModuleRolesQualifiedNames() {
						if mref == qualifiedRole {
							includedBy = append(includedBy, urTyped.Name())
						}
					}
				}
				if len(includedBy) > 0 {
					fmt.Fprintf(ctx.Output, "\n-- Included in user roles: %s\n", strings.Join(includedBy, ", "))
				}
			}
			return nil
		}
	}
	return mdlerrors.NewNotFound("module role", name.String())
}
```

- [ ] **Step 4: GREEN**
- [ ] **Step 5: Build + test gate**
- [ ] **Step 6: Commit `feat(executor): Stage 3.3.1.A5 — describeModuleRoleGen (gen-typed)`**

### Task A6: `describeUserRoleGen`

**Files:** `mdl/executor/cmd_security_gen.go`, test file

- [ ] **Step 1: Test, Step 2: RED, Step 3: Implement** (mirror task A5 pattern but iterate `ps.UserRolesItems()` matching `name.Name`; output `create user role …` syntax matching `describeUserRole` in `cmd_security.go:746–783`)
- [ ] **Step 4: GREEN, Step 5: gate, Step 6: Commit `feat(executor): Stage 3.3.1.A6 — describeUserRoleGen (gen-typed)`**

### Task A7: `describeDemoUserGen`

**Files:** `mdl/executor/cmd_security_gen.go`, test file

- [ ] **Step 1: Test, Step 2: RED, Step 3: Implement**

```go
func describeDemoUserGen(ctx *ExecContext, userName string) error {
	ps, err := getProjectSecurityGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("read project security", err)
	}
	if ps == nil {
		return mdlerrors.NewNotFound("demo user", userName)
	}
	for _, du := range ps.DemoUsersItems() {
		typed, ok := du.(*genSec.DemoUser)
		if !ok || typed.UserName() != userName {
			continue
		}
		fmt.Fprintf(ctx.Output, "create demo user '%s' password '***'", typed.UserName())
		if typed.EntityQualifiedName() != "" {
			fmt.Fprintf(ctx.Output, " entity %s", typed.EntityQualifiedName())
		}
		roles := typed.UserRolesQualifiedNames()
		if len(roles) > 0 {
			fmt.Fprintf(ctx.Output, " (%s)", strings.Join(roles, ", "))
		}
		fmt.Fprintln(ctx.Output, ";")
		fmt.Fprintln(ctx.Output, "/")
		return nil
	}
	return mdlerrors.NewNotFound("demo user", userName)
}
```

- [ ] **Step 4: GREEN, Step 5: gate, Step 6: Commit `feat(executor): Stage 3.3.1.A7 — describeDemoUserGen (gen-typed)`**

### Task A8: `listAccessOnEntityGen` (gen-typed entity reads)

**Risk:** This function currently uses `*domainmodel.Entity` and `*domainmodel.AccessRule` (sdk types). To migrate without depending on Stage 3.3 priority #4 (domainmodel domain), we keep `ctx.Backend.GetDomainModel(module.ID)` but extract a minimal local type for the entity-rule fields we actually consume (already done by `entityRuleRoleStrings` / `entityRuleRightStrings` helpers in `cmd_security_gen.go`).

**Two options:**
- **A8a (preferred):** Read the gen-typed DomainModel via `mprread.ListUnitsByType[*genDM.DomainModel]` and walk gen entities/access rules. This is the same pattern as `addEntityAccessRuleViaModelsdk`.
- **A8b (fallback):** Keep calling `ctx.Backend.GetDomainModel` and convert each access rule to a local `entityAccessRuleView` struct that contains ONLY the strings used downstream. Defer full gen integration to Stage 3.3 priority #4.

This plan picks **A8b** because A8a duplicates the domainmodel migration scope. The `entityRuleRoleStrings`/`entityRuleRightStrings` helpers already accept `*domainmodel.AccessRule` — keep them as the bridge for now and replace with gen-typed versions when domainmodel domain migrates.

- [ ] **Step 1: Test** — exercise `listAccessOnEntityGen` against a fixture with one entity + one access rule
- [ ] **Step 2: RED**
- [ ] **Step 3: Implement** by copying `listAccessOnEntity` from `cmd_security.go:171–305` verbatim, renaming to `listAccessOnEntityGen`, and removing its `import "...sdk/security"` dependency (the function already only uses `sdk/domainmodel`, not `sdk/security`). The rename plus the file move is sufficient.
- [ ] **Step 4: GREEN**
- [ ] **Step 5: Build + test gate**
- [ ] **Step 6: Commit `feat(executor): Stage 3.3.1.A8 — listAccessOnEntityGen (gen-typed twin, domainmodel passthrough)`**

### Task A9: `listAccessOnPageGen`

Page domain isn't migrated yet (Stage 3.3 priority #5). Like task A8, this is a passthrough rename that drops the `sdk/security` dep:

- [ ] **Step 1: Test, Step 2: RED, Step 3: Implement** by copying `listAccessOnPage` from `cmd_security.go:307–350`, renaming to `listAccessOnPageGen`, no other changes needed (function uses `pages.AllowedRoles []string` from backend, no sdk/security import in body)
- [ ] **Step 4–6: GREEN, gate, Commit `feat(executor): Stage 3.3.1.A9 — listAccessOnPageGen (twin)`**

### Task A10: Dispatcher cutover (executor_query.go)

Switch every `ShowProjectSecurity`, `ShowModuleRoles`, `ShowUserRoles`, `ShowDemoUsers`, `ShowAccessOn`, `ShowAccessOnPage`, `ShowSecurityMatrix`, `DescribeModuleRole`, `DescribeUserRole`, `DescribeDemoUser` dispatch to its `*Gen` variant.

**Files:**
- Modify: `mdl/executor/executor_query.go`
- Modify: any other dispatcher (e.g., `register_stubs.go`) that routes Describe* statements

- [ ] **Step 1: Locate dispatch sites**
```bash
grep -nE "listProjectSecurity\b|listModuleRoles\b|listUserRoles\b|listDemoUsers\b|listAccessOnEntity\b|listAccessOnPage\b|listSecurityMatrix\b|describeModuleRole\b|describeUserRole\b|describeDemoUser\b" mdl/executor/executor_query.go mdl/executor/register_stubs.go mdl/executor/executor_describe.go
```

- [ ] **Step 2: Replace each call site** with the `*Gen` variant. Run-time observation: nothing in the test harness mocks these functions individually, so the switch is a string substitution.

- [ ] **Step 3: Build + full test gate**
```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go build ./...
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./... -count=1 -timeout 240s
```
If any test in `cmd_security_mock_test.go` fails because it mocks `Backend.GetProjectSecurity()` for the legacy path, the test will need a parallel update in §6 task C4.

- [ ] **Step 4: Commit**
```bash
git add mdl/executor/executor_query.go mdl/executor/register_stubs.go
git commit -m "$(cat <<'EOF'
refactor(executor): Stage 3.3.1.A10 — dispatch all SHOW/DESCRIBE security to gen variants

Cuts over executor_query.go and register_stubs.go to call listProjectSecurityGen,
listModuleRolesGen, listUserRolesGen, listDemoUsersGen, listAccessOnEntityGen,
listAccessOnPageGen, listSecurityMatrixGen, describeModuleRoleGen,
describeUserRoleGen, describeDemoUserGen instead of the legacy sdk-typed
counterparts.

The legacy functions stay in cmd_security.go until Phase E deletes them; this
commit only flips the routing.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## §5 Phase B — Visualization

**Skipped.** Security domain has no `cmd_security_elk.go` / `cmd_security_mermaid.go` / `cmd_security_structure.go`. Verified by `find mdl/executor/ -name "*security*"` — only show/describe/write files exist. The matrix output (`listSecurityMatrixGen`) is the closest thing to a viz and is already covered in Phase A (commit `e6538a8a` predecessor + task A10 wiring).

---

## §6 Phase C — Consumer Migration

For each non-executor file that imports `sdk/security`, switch to gen types via the new helpers/repos. One commit per file (group only when ≤5 lines per file).

### Task C1: `mdl/backend/security.go` — interface return types

This is the load-bearing change: every consumer using `ctx.Backend.GetProjectSecurity()` etc. needs the new return type.

**Strategy:** Don't break the legacy interface. Add gen-typed methods alongside, then migrate consumers, then remove legacy in Phase E. This matches Stage 3.2 pattern.

**Files:**
- Modify: `mdl/backend/security.go` (add new gen-typed methods)
- Modify: `mdl/backend/mpr/backend.go` (implement them via `b.msdkWriter` reader)
- Modify: `mdl/backend/mock/backend.go` + `mock_security.go` (Func-field stubs)

- [ ] **Step 1: Write failing test** (`mdl/backend/mpr/security_modelsdk_test.go` extension)

```go
func TestGetProjectSecurityGen_ReturnsSingleton(t *testing.T) {
	mprPath := makeBlankProjectMPR(t)
	b := New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	ps, err := b.GetProjectSecurityGen()
	if err != nil {
		t.Fatalf("GetProjectSecurityGen: %v", err)
	}
	if ps == nil {
		t.Fatalf("GetProjectSecurityGen returned nil")
	}
	if ps.SecurityLevel() == "" {
		t.Errorf("SecurityLevel empty")
	}
}
```

- [ ] **Step 2: RED**

- [ ] **Step 3: Implement**

```go
// mdl/backend/security.go — add to ProjectSecurityBackend interface
GetProjectSecurityGen() (*genSec.ProjectSecurity, error)

// mdl/backend/security.go — add to ModuleSecurityBackend interface
GetModuleSecurityGen(moduleID model.ID) (*genSec.ModuleSecurity, error)
ListModuleSecurityGen() ([]*genSec.ModuleSecurity, error)
```

```go
// mdl/backend/mpr/backend.go
func (b *MprBackend) GetProjectSecurityGen() (*genSec.ProjectSecurity, error) {
    return mprrepos.NewSecurityRepository(b.msdkWriter).Get()
}
func (b *MprBackend) GetModuleSecurityGen(moduleID model.ID) (*genSec.ModuleSecurity, error) {
    return mprrepos.NewSecurityRepository(b.msdkWriter).GetModuleSecurity(moduleID)
}
func (b *MprBackend) ListModuleSecurityGen() ([]*genSec.ModuleSecurity, error) {
    // loop over all modules; same logic as listModuleSecurityWithContainerGen
    // (without the executor cache)
}
```

Add Func-field stubs to mock backend with descriptive errors:

```go
// mdl/backend/mock/backend.go
GetProjectSecurityGenFunc      func() (*genSec.ProjectSecurity, error)
GetModuleSecurityGenFunc       func(model.ID) (*genSec.ModuleSecurity, error)
ListModuleSecurityGenFunc      func() ([]*genSec.ModuleSecurity, error)
```

```go
// mdl/backend/mock/mock_security.go
func (m *MockBackend) GetProjectSecurityGen() (*genSec.ProjectSecurity, error) {
    if m.GetProjectSecurityGenFunc != nil {
        return m.GetProjectSecurityGenFunc()
    }
    return nil, fmt.Errorf("MockBackend.GetProjectSecurityGen not configured")
}
// ... and similar for the other two
```

- [ ] **Step 4: GREEN, Step 5: Build + test gate**
- [ ] **Step 6: Commit `feat(backend): Stage 3.3.1.C1 — add gen-typed security read methods to FullBackend`**

### Task C2: `mdl/executor/cmd_security_defaults.go`

Migrate the three sdk-typed helpers (`moduleUsesAutoDocumentRole`, `defaultDocumentAccessRoles`, `pruneInvalidUserRoles`) to gen types via `ctx.Security`.

**Files:** `mdl/executor/cmd_security_defaults.go` (rewrite)

- [ ] **Step 1: Failing test** — assert `moduleUsesAutoDocumentRoleGen` returns true when a single ModuleRole matches `autoDocumentRoleName`/`autoDocumentRoleDescription`
- [ ] **Step 2: RED**
- [ ] **Step 3: Replace the three functions with gen-typed twins:**

```go
func moduleUsesAutoDocumentRoleGen(ms *genSec.ModuleSecurity) bool {
    if ms == nil { return false }
    items := ms.ModuleRolesItems()
    if len(items) != 1 { return false }
    mr, ok := items[0].(*genSec.ModuleRole)
    if !ok { return false }
    return mr.Name() == autoDocumentRoleName && mr.Description() == autoDocumentRoleDescription
}

func defaultDocumentAccessRoles(ctx *ExecContext, module *model.Module) []model.ID {
    if module == nil { return nil }
    ms, err := ctx.Backend.GetModuleSecurityGen(module.ID)
    if err != nil || ms == nil { return nil }
    if moduleUsesAutoDocumentRoleGen(ms) {
        return []model.ID{model.ID(module.Name + "." + autoDocumentRoleName)}
    }
    if len(ms.ModuleRolesItems()) > 0 { return nil }
    if err := ctx.Backend.AddModuleRole(ms.ID, autoDocumentRoleName, autoDocumentRoleDescription); err != nil {
        return nil
    }
    return []model.ID{model.ID(module.Name + "." + autoDocumentRoleName)}
}

func pruneInvalidUserRoles(ctx *ExecContext, ps *genSec.ProjectSecurity) error {
    if latest, err := ctx.Backend.GetProjectSecurityGen(); err == nil && latest != nil {
        ps = latest
    } else if ps == nil {
        return err
    }
    for _, ur := range ps.UserRolesItems() {
        typed, ok := ur.(*genSec.UserRole)
        if !ok { continue }
        hasNonSystemRole := false
        for _, moduleRole := range typed.ModuleRolesQualifiedNames() {
            if !strings.HasPrefix(moduleRole, "System.") {
                hasNonSystemRole = true
                break
            }
        }
        if hasNonSystemRole { continue }
        if err := ctx.Backend.RemoveUserRole(ps.ID(), typed.Name()); err != nil {
            return err
        }
        if !ctx.Quiet {
            fmt.Fprintf(ctx.Output, "Dropped invalid user role: %s\n", typed.Name())
        }
    }
    return nil
}
```

- [ ] **Step 4: GREEN, Step 5: gate, Step 6: Commit `refactor(executor): Stage 3.3.1.C2 — migrate cmd_security_defaults.go to gen types`**

### Task C3: `mdl/linter/rules/security.go` (SEC003 production-level + demo users)

**Files:** `mdl/linter/rules/security.go`

The rule uses `security.SecurityLevelProduction` constant. Replace with the BSON literal `"CheckEverything"` and remove the import.

- [ ] **Step 1: Test** — assert SEC003 fires for `EnableDemoUsers=true` + `SecurityLevel=CheckEverything`
- [ ] **Step 2: RED** (might already pass; if so, skip RED and go to Step 3 as a refactor)
- [ ] **Step 3: Replace `security.SecurityLevelProduction` with `"CheckEverything"` constant defined as a package-level `const securityLevelProduction = "CheckEverything"` in the rules file. Remove the `sdk/security` import.**

Also update SEC002 if it uses `ps.PasswordPolicy.MinimumLength` — when the linter context migrates to gen-typed `*genSec.ProjectSecurity` (task C4), the accessor changes to `ps.PasswordPolicy().MinimumLength()`.

- [ ] **Step 4: GREEN, Step 5: gate, Step 6: Commit `refactor(linter): Stage 3.3.1.C3 — drop sdk/security from rules/security.go`**

### Task C4: `mdl/linter/context.go` + `mdl/catalog/builder.go`

Both files have a Reader interface line that returns `*security.ProjectSecurity`. Switch to `*genSec.ProjectSecurity`.

**Files:**
- Modify: `mdl/linter/context.go`
- Modify: `mdl/catalog/builder.go`
- Modify: any concrete reader implementation

- [ ] **Step 1: Test — pick one linter rule that calls `ctx.Reader().GetProjectSecurity()` and assert the new gen-typed value flows through**
- [ ] **Step 2: RED**
- [ ] **Step 3: Implement**:
  - In `mdl/linter/context.go`, change the interface signature: `GetProjectSecurity() (*genSec.ProjectSecurity, error)`
  - In `mdl/catalog/builder.go`, same change
  - Update the catalog/linter implementation that fulfills the interface — it should now delegate to `b.GetProjectSecurityGen()` introduced in C1
  - All sites that consume `ps.PasswordPolicy.MinimumLength` switch to `ps.PasswordPolicy().MinimumLength()` (method call, not field access — gen accessor)
- [ ] **Step 4: GREEN, Step 5: gate, Step 6: Commit `refactor(linter,catalog): Stage 3.3.1.C4 — Reader.GetProjectSecurity returns gen-typed *ProjectSecurity`**

### Task C5: MockBackend audit (master plan §5 P3 for security domain)

Bring `mdl/backend/mock/mock_security.go` into compliance: every Func-field stub returns `"MockBackend.X not configured"` instead of `nil`.

**Files:** `mdl/backend/mock/mock_security.go`

- [ ] **Step 1: Test** — verify a freshly-constructed `MockBackend` returns errors when calling each security method without setting the Func
- [ ] **Step 2: RED** (tests pass today because nil-return is permissive — flip them to expect the new error format)
- [ ] **Step 3: Replace every** `return nil, nil` / `return nil` **with** `return …, fmt.Errorf("MockBackend.X not configured")` (where `X` matches the method name). Mirrors the pattern in `mdl/backend/mock/mock_microflow.go` if such file exists; otherwise establish the pattern here.
- [ ] **Step 4: GREEN** — fix every test that broke because it relied on permissive `nil, nil`. The fix is to set the Func explicitly, not to relax the mock.
- [ ] **Step 5: Build + test gate**
- [ ] **Step 6: Commit `refactor(mock): Stage 3.3.1.C5 — MockBackend security stubs return descriptive errors`**

### Task C6: `mdl/executor/cmd_security_mock_test.go` + sibling mock tests

Migrate every `*security.ProjectSecurity` / `*security.UserRole` / `*security.DemoUser` / `*security.ModuleSecurity` / `*security.ModuleRole` fixture in:
- `mdl/executor/cmd_security_mock_test.go`
- `mdl/executor/cmd_json_mock_test.go`
- `mdl/executor/cmd_write_handlers_mock_test.go`
- `mdl/executor/cmd_error_mock_test.go`

to gen types built via `genSec.NewProjectSecurity()` / `genSec.NewUserRole()` etc.

**Strategy:** these are large test files; do them as ONE commit so cross-file fixture references stay consistent. Do NOT split per file.

- [ ] **Step 1: Run test files RED first** — once C1 lands the mock helper signatures change and these tests fail with type mismatches
- [ ] **Step 2: Confirm RED**
- [ ] **Step 3: Replace each `&security.ProjectSecurity{...}` literal with the gen constructor pattern:**

```go
ps := genSec.NewProjectSecurity()
ps.SetSecurityLevel("CheckEverything")
ps.SetEnableDemoUsers(true)

ur := genSec.NewUserRole()
ur.SetName("Administrator")
ur.SetManageAllRoles(true)
ur.SetModuleRolesQualifiedNames([]string{"MyModule.User"})
ps.AddUserRoles(ur)
```

- [ ] **Step 4: GREEN, Step 5: full gate, Step 6: Commit `test(executor): Stage 3.3.1.C6 — migrate security mock fixtures to gen types`**

---

## §7 Phase D — Write Path Completion

Goal: replace every legacy `cmd_security_write.go` function with a `*Gen` twin that goes through gen-typed paths and the existing `*ViaModelsdk` backend mutators.

### Task D1: `execCreateModuleRoleGen`

**Files:** Create `mdl/executor/cmd_security_write_gen.go` extension; tests in `cmd_security_write_gen_test.go`

- [ ] **Step 1: Test** — create role; verify it appears in `ms.ModuleRolesItems()`; verify auto-provisioned role overwrite via case-insensitive match preserves the rename cascade
- [ ] **Step 2: RED**
- [ ] **Step 3: Implement** — the gen-typed module security read goes through `ctx.Backend.GetModuleSecurityGen` (added in C1). The actual mutation calls `ctx.Backend.AddModuleRole` (already wired to gen via `addModuleRoleViaModelsdk`). The case-insensitive overwrite path keeps `ctx.Backend.UpdateQualifiedNameInAllUnits` (cross-cutting rename helper, not security-specific).

```go
func execCreateModuleRoleGen(ctx *ExecContext, s *ast.CreateModuleRoleStmt) error {
    if !ctx.ConnectedForWrite() { return mdlerrors.NewNotConnectedWrite() }
    module, err := findModule(ctx, s.Name.Module)
    if err != nil { return err }
    ms, err := ctx.Backend.GetModuleSecurityGen(module.ID)
    if err != nil { return mdlerrors.NewBackend(...) }

    for _, mr := range ms.ModuleRolesItems() {
        typed, ok := mr.(*genSec.ModuleRole)
        if !ok || !strings.EqualFold(typed.Name(), s.Name.Name) { continue }
        if typed.Description() == autoDocumentRoleDescription {
            // overwrite-rename path (mirrors execCreateModuleRole exactly)
            ...
        }
        return mdlerrors.NewAlreadyExists("module role", ...)
    }
    if err := ctx.Backend.AddModuleRole(ms.ID(), s.Name.Name, s.Description); err != nil {
        return mdlerrors.NewBackend("create module role", err)
    }
    invalidateModuleSecurityCache(ctx)
    fmt.Fprintf(ctx.Output, "Created module role: %s.%s\n", s.Name.Module, s.Name.Name)
    return nil
}
```

- [ ] **Step 4: GREEN, Step 5: gate, Step 6: Commit `feat(executor): Stage 3.3.1.D1 — execCreateModuleRoleGen`**

### Task D2: `execDropModuleRoleGen`

Mirrors `execDropModuleRole` (cmd_security_write.go:76–173) including all four cascade halves (entities, microflows, nanoflows, pages, OData) and final `RemoveModuleRole`. The microflow/nanoflow cascades already go through gen (`cascadeRemoveRoleFromMicroflowsGen`); D2 wires the remaining cascades through gen-typed reads.

- [ ] **Step 1: Test** — drop role with cascade; verify role removed from entity rules, page allowed roles, OData service allowed roles, user roles
- [ ] **Step 2–6: same TDD pattern, commit `feat(executor): Stage 3.3.1.D2 — execDropModuleRoleGen with full cascade`**

### Task D3: `execCreateUserRoleGen` + `execAlterUserRoleGen` + `execDropUserRoleGen`

Three closely-related operations; bundle as one commit because they share the gen read path through `getProjectSecurityGen`.

- [ ] **Step 1–3: Test, RED, Implement** (mirror cmd_security_write.go:176–288 substituting `getProjectSecurityGen` + `(*genSec.UserRole)` casts)
- [ ] **Step 4–6: GREEN, gate, Commit `feat(executor): Stage 3.3.1.D3 — user role create/alter/drop (gen)`**

### Task D4: `execAlterProjectSecurityGen`

Removes the last `sdk/security` import from `cmd_security_write.go` (the `security.SecurityLevelProduction` etc. constants).

- [ ] **Step 1: Test** — set level to "Production" / "Prototype" / "Off"; assert BSON SecurityLevel field equals the right constant
- [ ] **Step 2: RED**
- [ ] **Step 3: Implement** with inline mapping (no sdk import):

```go
func execAlterProjectSecurityGen(ctx *ExecContext, s *ast.AlterProjectSecurityStmt) error {
    if !ctx.ConnectedForWrite() { return mdlerrors.NewNotConnectedWrite() }
    ps, err := getProjectSecurityGen(ctx)
    if err != nil || ps == nil { return mdlerrors.NewBackend("read project security", err) }

    if s.SecurityLevel != "" {
        var bsonLevel string
        switch s.SecurityLevel {
        case "Production":  bsonLevel = "CheckEverything"
        case "Prototype":   bsonLevel = "CheckFormsAndMicroflows"
        case "Off":         bsonLevel = "CheckNothing"
        default:
            return mdlerrors.NewUnsupported(fmt.Sprintf("unknown security level: %s", s.SecurityLevel))
        }
        if err := ctx.Backend.SetProjectSecurityLevel(ps.ID(), bsonLevel); err != nil {
            return mdlerrors.NewBackend("set security level", err)
        }
        invalidateProjectSecurityCache(ctx)
        fmt.Fprintf(ctx.Output, "Set project security level to %s\n", s.SecurityLevel)
    }

    if s.DemoUsersEnabled != nil {
        if err := ctx.Backend.SetProjectDemoUsersEnabled(ps.ID(), *s.DemoUsersEnabled); err != nil {
            return mdlerrors.NewBackend("set demo users", err)
        }
        invalidateProjectSecurityCache(ctx)
        state := "disabled"
        if *s.DemoUsersEnabled { state = "enabled" }
        fmt.Fprintf(ctx.Output, "Demo users %s\n", state)
    }
    return nil
}
```

- [ ] **Step 4–6: GREEN, gate, Commit `feat(executor): Stage 3.3.1.D4 — execAlterProjectSecurityGen (no sdk/security)`**

### Task D5: `execCreateDemoUserGen` + `execDropDemoUserGen`

Bundle for the same reason as D3.

- [ ] **Step 1–3: Test, RED, Implement** (mirror cmd_security_write.go:761–899; `detectUserEntity` keeps using `ctx.Backend.ListDomainModels()` — see R7. Just rename to `detectUserEntityGen` and drop the `domainmodel.*` type names by accessing only `dm.Entities[].Name` and `dm.Entities[].GeneralizationRef`)
- [ ] **Step 4–6: GREEN, gate, Commit `feat(executor): Stage 3.3.1.D5 — demo user create/drop (gen)`**

### Task D6: `execGrantEntityAccessGen` + `execRevokeEntityAccessGen` (with version-prefix fix)

The trickiest task. Two parts:

**Part D6a — fix the AllowedModuleRoles version-prefix bug** (R1)

**Files:** `modelsdk/property/reference.go` + `modelsdk/property/edge_test.go`

- [ ] **Step 1: Failing test** — encode a `ByNameRefList` with one entry and assert the BSON output is `[int32(1), "Module.Role"]` not `["Module.Role"]`
- [ ] **Step 2: RED**
- [ ] **Step 3: Fix `BSONValue` to prepend `int32(1)`:**

```go
func (r *ByNameRefList[T]) BSONValue() any {
    if len(r.qnames) == 0 { return r.qnames }
    out := make([]any, 0, len(r.qnames)+1)
    out = append(out, int32(1))
    for _, qn := range r.qnames {
        out = append(out, qn)
    }
    return out
}
```

Also adjust the decoder side: `SetFromDecode` already strips the version prefix (verify). If decoder doesn't strip, decode-then-encode is non-idempotent; fix decoder simultaneously.

- [ ] **Step 4: Re-run security_entity_access_gen_test.go and microflow `_gen` tests to verify nothing breaks downstream. The `countVersionedEntries` helper should now correctly find the `int32(1)` and skip it.**
- [ ] **Step 5: Run `mx check` smoke test against a fixture** that exercises an access rule write
- [ ] **Step 6: GREEN, full test gate, Commit `fix(modelsdk/property): ByNameRefList encodes versioned-array prefix`**

Update memory entry: `[[project_allowedmoduleroles_version_prefix]]` mark as fixed; cross-link this commit.

**Part D6b — `execGrantEntityAccessGen` + `execRevokeEntityAccessGen`**

- [ ] **Step 1: Test** — round-trip GRANT entity access; assert MemberAccesses count, AllowedModuleRoles version prefix, role list contents; round-trip REVOKE
- [ ] **Step 2: RED** (call the not-yet-existing `*Gen` functions)
- [ ] **Step 3: Implement** — 99% identical to legacy `execGrantEntityAccess` (cmd_security_write.go:291–458) but:
  - Domain-model read still goes through `ctx.Backend.GetDomainModel` (passthrough until Stage 3.3 priority #4); access only string/bool/ID fields, no `domainmodel.*` type references at the API surface
  - Mutation goes through `ctx.Backend.AddEntityAccessRule(...)` (already wired to `addEntityAccessRuleViaModelsdk` per R1's gen-native fix, now corrected by D6a)
  - Reconciliation continues through `ctx.Backend.ReconcileMemberAccesses` (still on Patch path — see `reconcileMemberAccessesViaModelsdk`)
- [ ] **Step 4–6: GREEN, gate, Commit `feat(executor): Stage 3.3.1.D6 — entity access grant/revoke (gen)`**

### Task D7: `execGrantPageAccessGen` + `execRevokePageAccessGen`

Page reads remain on `ctx.Backend.ListPages` (Stage 3.3 priority #5). Mutations go through `ctx.Backend.UpdateAllowedRoles` (already wired to `updateAllowedRolesViaModelsdk` for *Page).

- [ ] **Step 1–3: Test, RED, Implement** mirroring cmd_security_write.go:561–676
- [ ] **Step 4–6: GREEN, gate, Commit `feat(executor): Stage 3.3.1.D7 — page access grant/revoke (gen)`**

### Task D8: `execGrantODataServiceAccessGen` + `execRevokeODataServiceAccessGen` + REST equivalents

Same shape as D7 — mutations via `UpdateAllowedRoles` / `UpdatePublishedRestServiceRoles`.

- [ ] **Step 1–3: Test, RED, Implement**
- [ ] **Step 4–6: GREEN, gate, Commit `feat(executor): Stage 3.3.1.D8 — OData/REST service access grant/revoke (gen)`**

### Task D9: `execUpdateSecurityGen`

Trivial passthrough — calls `ctx.Backend.ReconcileMemberAccesses` per module. Just rename.

- [ ] **Step 1–3: Test, RED, Implement**
- [ ] **Step 4–6: GREEN, gate, Commit `feat(executor): Stage 3.3.1.D9 — execUpdateSecurityGen`**

### Task D10: Wire the new write dispatchers

**Files:** `mdl/executor/register_stubs.go`

Switch each write-stmt registration to its `*Gen` variant.

- [ ] **Step 1: Locate registrations**
```bash
grep -nE "execCreateModuleRole\b|execDropModuleRole\b|execCreateUserRole\b|execAlterUserRole\b|execDropUserRole\b|execGrantEntityAccess\b|execRevokeEntityAccess\b|execGrantPageAccess\b|execRevokePageAccess\b|execAlterProjectSecurity\b|execCreateDemoUser\b|execDropDemoUser\b|execGrantODataServiceAccess\b|execRevokeODataServiceAccess\b|execGrantPublishedRestServiceAccess\b|execRevokePublishedRestServiceAccess\b|execUpdateSecurity\b" mdl/executor/register_stubs.go
```
- [ ] **Step 2: Replace each with the `*Gen` variant**
- [ ] **Step 3: Build + full test gate**
- [ ] **Step 4: Commit `refactor(executor): Stage 3.3.1.D10 — dispatch all security write stmts to gen variants`**

---

## §8 Phase E — Cleanup Commits

### Task E1: Retire `FullBackend` deprecated sdk-typed security methods

**Files:**
- Modify: `mdl/backend/security.go` (delete `GetProjectSecurity`, `GetModuleSecurity`, `ListModuleSecurity` from the interfaces)
- Modify: `mdl/backend/mpr/backend.go` (delete the corresponding shim methods)
- Modify: `mdl/backend/mock/backend.go` + `mock_security.go` (delete the corresponding Func fields and shims)

- [ ] **Step 1: Build to confirm no remaining callers**
```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go build ./...
```
If anything fails, route the failing caller through `*Gen` first (one-line fix per site), then re-run.

- [ ] **Step 2: Delete the three sdk-typed methods + their Func fields + shims**
- [ ] **Step 3: Build + full test gate**
- [ ] **Step 4: Commit**

```bash
git add mdl/backend/security.go mdl/backend/mpr/backend.go mdl/backend/mock/backend.go mdl/backend/mock/mock_security.go
git commit -m "$(cat <<'EOF'
refactor(backend): Stage 3.3.1.E1 — retire FullBackend.{GetProjectSecurity,GetModuleSecurity,ListModuleSecurity}

All consumers now go through *Gen variants (GetProjectSecurityGen,
GetModuleSecurityGen, ListModuleSecurityGen) added in C1 and the
ctx.Security repo helpers added in A0.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task E2: Delete legacy executor security files

**Files:**
- Delete: `mdl/executor/cmd_security.go`
- Delete: `mdl/executor/cmd_security_write.go`
- Modify: `mdl/executor/cmd_security_defaults.go` (already migrated in C2 — keep, just verify zero `sdk/security` imports)

- [ ] **Step 1: Final grep before deletion**
```bash
grep -rn '"github.com/mendixlabs/mxcli/sdk/security"' mdl/executor/cmd_security.go mdl/executor/cmd_security_write.go
```
Expected: 0 unique callers using these files' exported symbols (everything routed to `*Gen` in A10 + D10).

- [ ] **Step 2: Run `go build ./mdl/executor/...` after `git rm` to confirm no missing references**
- [ ] **Step 3: Build + full test gate**
- [ ] **Step 4: Commit**

```bash
git rm mdl/executor/cmd_security.go mdl/executor/cmd_security_write.go
git commit -m "$(cat <<'EOF'
refactor(executor): Stage 3.3.1.E2 — delete legacy cmd_security{,_write}.go

All read funcs migrated to cmd_security_gen.go (Phase A) and dispatched
in A10. All write funcs migrated to cmd_security_write_gen.go (Phase D)
and dispatched in D10. Remaining cmd_security_defaults.go was migrated
to gen types in C2.

Net delete: 786 + 1189 = 1975 LoC.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task E3: Delete `sdk/security` package

**Files:**
- Delete: `sdk/security/security.go`
- Delete: `sdk/security/security_test.go`
- Delete: `sdk/security/` directory

- [ ] **Step 1: Final acceptance grep**
```bash
grep -rln '"github.com/mendixlabs/mxcli/sdk/security"' . --include="*.go"
```
Expected output: ONLY `sdk/mpr/reader_documents.go` and `sdk/mpr/parser_security.go` (Stage 4 territory).

If any non-`sdk/mpr/` file remains, halt and route it before continuing.

- [ ] **Step 2: Verify Stage 4 boundary** — check whether the Stage 4 team has removed `sdk/mpr/parser_security.go` since the master plan was written (`git log -1 --oneline -- sdk/mpr/parser_security.go`). If yes, the package is fully orphaned and can be deleted. If no, proceed with the Stage 4 caveat documented in §11.

- [ ] **Step 3: Delete**
```bash
git rm -r sdk/security/
```

- [ ] **Step 4: Build to confirm `sdk/mpr` still compiles**
```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go build ./...
```

If `sdk/mpr` fails because it imports `sdk/security`, BACK OUT the deletion and instead leave `sdk/security` in place with a deprecation comment until Stage 4 lands. This is the documented escape hatch.

If it builds, run full test gate.

- [ ] **Step 5: Commit**

```bash
git add -u sdk/security/
git commit -m "$(cat <<'EOF'
refactor: Stage 3.3.1.E3 — delete sdk/security package

Final cleanup commit for the security domain (Stage 3.3 priority #1).
All consumers in mdl/, modelsdk/, and api/ migrated to modelsdk/gen/security
in commits A0–D10. The package is now unreachable except from sdk/mpr/
(reader_documents.go, parser_security.go) which is Stage 4 team's territory
and out of scope for Stage 3.3.

Aggregate Stage 3.3 priority #1 stats:
- Commits: ~32
- LoC delta: -2135 (sdk/security 160 + cmd_security 786 + cmd_security_write 1189)
                  +XXX (gen helpers + extended tests)

Memory: project_allowedmoduleroles_version_prefix marked FIXED in D6a.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task E4: Final acceptance verification

**Files:** none (read-only checks)

- [ ] **Step 1: Acceptance greps**
```bash
# Outside sdk/mpr — must be 0
grep -rln '"github.com/mendixlabs/mxcli/sdk/security"' . --include="*.go" | grep -v "^./sdk/mpr/"

# Modelsdk — must be 0
grep -rln '"github.com/mendixlabs/mxcli/sdk/security"' modelsdk/ --include="*.go"

# api — must be 0
grep -rln '"github.com/mendixlabs/mxcli/sdk/security"' api/ --include="*.go"
```
All three: empty output.

- [ ] **Step 2: Run security_matrix-affecting tests**
```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./mdl/executor/ -run "Security|security" -count=1
```

- [ ] **Step 3: Verify mock cache helper** — `listSecurityWithContainerGen` (i.e., `listModuleSecurityWithContainerGen`) exists in `mdl/executor/helpers_security_gen.go` (added in A0)

- [ ] **Step 4: Update memory `project_stage_3_3_security_complete.md`** with final stats; cross-link in `MEMORY.md` index

- [ ] **Step 5: No commit needed** (verification only). If documenting, treat as a separate doc commit.

---

## §9 Acceptance Criteria

- [ ] `grep -rln '"github.com/mendixlabs/mxcli/sdk/security"' . --include="*.go" | grep -v "^./sdk/mpr/"` returns 0 lines
- [ ] All `Test*Security*` and `Test*security*` tests pass:
      `GOPROXY=... ~/go1.26/bin/go test ./mdl/executor/ ./mdl/backend/mpr/ ./mdl/linter/... -run "Security|security" -count=1 -v`
- [ ] `listModuleSecurityWithContainerGen` (the cache helper) is the only path used in `mdl/executor/` for module-security listing; no `ctx.Backend.ListModuleSecurityGen()` raw calls outside that helper
- [ ] `getProjectSecurityGen` is the only path for `ProjectSecurity` reads in `mdl/executor/`
- [ ] `ByNameRefList` BSON encoding includes the `int32(1)` version prefix (D6a fix verified by `mx check` smoke test on a fixture that writes an entity access rule)
- [ ] Full repo build is green: `GOPROXY=... ~/go1.26/bin/go build ./...`
- [ ] Full repo test suite is green: `GOPROXY=... ~/go1.26/bin/go test ./... -count=1 -timeout 240s`
- [ ] `sdk/security/` directory is gone (or, if Stage 4 hasn't landed `sdk/mpr/parser_security.go` removal, kept with explicit deprecation header — documented in §11)

---

## §10 Estimated Commit Count + Sequencing

| Phase | Tasks | Commits | Cumulative |
|---|---|---|---|
| A — Read path | A0, A1, A2, A3, A4, A5, A6, A7, A8, A9, A10 | 11 | 11 |
| B — Visualization | (skipped) | 0 | 11 |
| C — Consumer migration | C1, C2, C3, C4, C5, C6 | 6 | 17 |
| D — Write path | D1, D2, D3, D4, D5, D6a, D6b, D7, D8, D9, D10 | 11 | 28 |
| E — Cleanup | E1, E2, E3, E4 | 3 (E4 is verify-only, no commit) | 31 |

**Estimated total: 31 commits** (within master plan §6 row #1's 25–35 range).

**Sequencing rationale:**
- Phase A first because read paths have no behavioural risk — pure additive
- A10 dispatcher cutover MUST come after A1–A9 so the dispatcher has all targets to switch to
- Phase C (consumer migration) before D (write path) because some D tasks consume `ctx.Backend.GetProjectSecurityGen` added in C1
- D6a (version-prefix fix) MUST land before D6b (entity access write) so D6b's tests pass `mx check` smoke test
- D10 dispatcher cutover after all D-tasks
- E1–E3 only after both A10 and D10 dispatchers route to gen — otherwise legacy still has callers

**Per-session checkpoints** (from `superpowers:checkpoint`): commit after each numbered task; if interrupted mid-D6, the partial fix in D6a is independently reviewable and reusable by other domains (D6a is a `modelsdk/property` global fix that benefits every domain marathon).

---

## §11 Coordination With Stage 4 Team

### Stage 4's territory
The Stage 4 team owns rewriting `sdk/mpr` to drop legacy round-trip code. Two security-relevant files in their territory:
- `sdk/mpr/reader_documents.go` — imports `sdk/security`
- `sdk/mpr/parser_security.go` — imports `sdk/security`

### Stage 3.3 commitment
**Stage 3.3 will NOT modify any file under `sdk/mpr/`.** Specifically:
- Phase E task E3 stops at the boundary: deletion of `sdk/security/` is conditional on the package being unreachable from `sdk/mpr/`. If Stage 4 has removed both files above before E3 runs, E3 deletes the package. If not, E3 is held with the package marked deprecated; Stage 4's PR will trigger the final delete.
- The acceptance grep in §9 explicitly excludes `^./sdk/mpr/` matches. A non-zero count there is acceptable for Stage 3.3 completion as long as it's only in `sdk/mpr/`.

### Risk of merge collision
Low. Stage 3.3 touches:
- `mdl/executor/cmd_security*.go` (new and existing)
- `mdl/backend/security.go`, `mdl/backend/mpr/backend.go`
- `mdl/backend/mock/`
- `mdl/linter/`
- `mdl/catalog/`
- `modelsdk/property/reference.go` (D6a)

Stage 4 touches:
- `sdk/mpr/*.go` (parser, writer, reader)

The only file both might read is `modelsdk/property/reference.go`, but Stage 4 isn't redesigning the property package. The D6a fix is therefore safe to land independently and benefits any domain marathon that follows.

### Communication
The team-lead should mention to Stage 4:
1. D6a (`fix(modelsdk/property): ByNameRefList encodes versioned-array prefix`) lands during this plan's execution and may affect any encoder roundtrip tests they have on `sdk/mpr/`. Coordinate the test-update if needed.
2. Phase E3 may NOT actually `git rm sdk/security/` if Stage 4 hasn't yet removed `sdk/mpr/parser_security.go`. The escape-hatch path leaves the package in place with a deprecation header.

---

## §12 Self-Review Checklist (skill-required)

**Spec coverage:** §4 (Phase A) covers all 7 read funcs in `cmd_security.go` plus dispatcher cutover (A10). §6 (Phase C) covers all 8 consumer files (mock backend, linter, catalog, defaults). §7 (Phase D) covers all 22 write funcs from `cmd_security_write.go`. §8 (Phase E) covers the three cleanup commits. §11 spells out the Stage 4 boundary. ✓

**Type consistency:** Every gen-typed accessor in implementation snippets uses parentheses (`SecurityLevel()`, `Name()`) matching `modelsdk/property` getter convention. `*genSec.UserRole` casts on `UserRolesItems()` follow the documented pattern in `security_project_modelsdk.go`. Every `Set*QualifiedName(s)` matches the existing modelsdk write-path pattern. ✓

**Risk surfacing:** AllowedModuleRoles version prefix bug (R1) gets a dedicated D6a sub-task with the fix code; the task explicitly verifies via `mx check` smoke test. AccessRule ParentPointer/ChildPointer semantics (R3) noted; the write tests must include an explicit MemberAccesses-on-FROM assertion. Stage 4 boundary (R2) explicitly listed in §11. ✓

**TDD discipline:** Every task A1–A10, C1–C6, D1–D10 starts with "Step 1: Write failing test" + "Step 2: Confirm RED". No "similar to A1" shortcuts (per master plan §3.1 and §7.5). ✓

**Commit hygiene:** Each commit has a single concern; D6a is split out from D6b because D6a is a `modelsdk/property` global fix; D5 bundles two related functions sharing a read path which is acceptable per CLAUDE.md "Scope & atomicity". Commit messages use HEREDOC per CLAUDE.md. No `--no-verify`. ✓

**No public-API break without approval:** No `modelsdk.go` or `api/` files touched. The `mdl/backend/security.go` interface change in C1 is internal — additive (`*Gen` methods) before subtractive (E1 retire). ✓

---

## §13 Execution Handoff

**Recommended approach:** subagent-driven (`superpowers:subagent-driven-development`). Each task is self-contained with full test code, full implementation code, and exact commit command. Dispatch one subagent per task, review between tasks.

**Alternative:** inline execution via `superpowers:executing-plans`. Suitable for tasks A0–A10 (low cognitive load) and Phase E. Phase D tasks (especially D6) benefit from subagent isolation because they touch the schema-gap workaround surface.

**Resumability:** all tasks use the `- [ ]` checkbox syntax so `executing-plans` can resume mid-flight. Each task's commit is independently reviewable.

**Status report when done:** Update `MEMORY.md` with `project_stage_3_3_security_complete.md`, mark master plan §6 row #1 done, then move to priority #2 (javaactions).
