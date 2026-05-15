# Retire b.writer Serialize* Receiver Methods Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Decouple `MprBackend` from `b.writer` for all encoding paths by converting `(w *Writer) SerializeXxx` receiver methods to standalone package-level functions, leaving `b.writer` responsible only for `Close()`.

**Architecture:** Two tiers. **Tier 1** — mechanical: convert all `(w *Writer) SerializeXxx` methods that do NOT internally read `w.reader` to standalone `func SerializeXxx` functions (most cases). **Tier 2** — parametric: for the two files that read `w.reader.ProjectVersion()` inside serialize logic (`writer_microflow.go`, `writer_domainmodel.go`), extract the version read to the call site and add `pv *ProjectVersion` as a parameter. MprBackend callers already have access to `b.reader.ProjectVersion()`. Embedded serializers (`SerializeWidget`, `SerializeClientAction`, `SerializeCustomWidgetDataSource`, `SerializeWorkflowActivity`) are already standalone — no change needed.

**Tech Stack:** Go 1.26, `sdk/mpr`, `mdl/backend/mpr`

**Current state:**
- 22 Serialize* methods have `(w *Writer)` receiver; 12 are already standalone
- Version-dependent: only `writer_microflow.go` (serializeMicroflow/Nanoflow, majorVersion checks at Mendix 9/10 boundary) and `writer_domainmodel.go` (serializeEntity, IsAtLeast(11, 0) check)
- Embedded-use serializers (Widget/ClientAction/DataSource/WorkflowActivity) already standalone — untouched by this plan
- `b.writer` currently used in MprBackend for: `Close()` (lifecycle), `Serialize*()` (encode — this plan eliminates), `Patch*()` / `Scan*()` (pure BSON helpers already standalone-ish — verified not writing via b.writer)

**Out of scope:**
- Gen-native migration of Microflow/Page/Workflow units (60+ polymorphic activity types — separate plan)
- `b.writer.Close()` retirement (connection lifecycle redesign — separate plan)
- AgentEditor `mpr.SerializeAgentEditor*` (already standalone package-level — unchanged)

---

## File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `sdk/mpr/writer_microflow.go` | Modify | Parametrize `ProjectVersion` in `serializeMicroflow` / `serializeNanoflow` and their callees |
| `sdk/mpr/writer_domainmodel.go` | Modify | Parametrize `ProjectVersion` in `serializeDomainModel` → `serializeEntity` |
| `sdk/mpr/serialize_exports.go` | Modify | Convert remaining `(w *Writer) SerializeXxx` wrappers to standalone functions |
| `sdk/mpr/writer_pages.go` | Modify | Make `serializePage` / `serializeLayout` / `serializeSnippet` callable without reader |
| `sdk/mpr/writer_services.go` (or equivalent) | Modify | Same for service-type serializers |
| `mdl/backend/mpr/mf_page_modelsdk.go` | Modify | Replace `b.writer.SerializeXxx` with standalone `mpr.SerializeXxx` + version |
| `mdl/backend/mpr/create_services_modelsdk.go` | Modify | Same |
| `mdl/backend/mpr/modules_modelsdk.go` | Modify | Same |
| `mdl/backend/mpr/domainmodel_modelsdk.go` | Modify | Same for `SerializeDomainModel` calls |
| `mdl/backend/mpr/settings_modelsdk.go` | Modify | Same for `SerializeProjectSettings` |

---

## Task 1: Audit — list all `(w *Writer) Serialize*` methods and their reader dependency

**Files:**
- Read: `sdk/mpr/serialize_exports.go`
- Read: `sdk/mpr/writer_microflow.go`
- Read: `sdk/mpr/writer_domainmodel.go`
- Read: any other `sdk/mpr/writer_*.go` containing Serialize* methods

- [ ] **Step 1: Find all receiver Serialize* methods**

```bash
grep -rn "^func (w \*Writer) Serialize" /mnt/data_sdd/gh/mxcli-wt-02/sdk/mpr/
```

- [ ] **Step 2: Find which ones reference w.reader internally**

```bash
# For each writer_*.go file that has Serialize* methods, check w.reader usage:
grep -n "w\.reader\|w\.path\|w\.db" /mnt/data_sdd/gh/mxcli-wt-02/sdk/mpr/writer_microflow.go | head -20
grep -n "w\.reader\|w\.path\|w\.db" /mnt/data_sdd/gh/mxcli-wt-02/sdk/mpr/writer_domainmodel.go | head -20
grep -n "w\.reader\|w\.path\|w\.db" /mnt/data_sdd/gh/mxcli-wt-02/sdk/mpr/writer_pages.go 2>/dev/null | head -10
grep -n "w\.reader\|w\.path\|w\.db" /mnt/data_sdd/gh/mxcli-wt-02/sdk/mpr/serialize_exports.go | head -20
```

- [ ] **Step 3: Confirm the two version-dependent files**

```bash
grep -n "ProjectVersion\|MajorVersion\|majorVersion\|IsAtLeast" \
    /mnt/data_sdd/gh/mxcli-wt-02/sdk/mpr/writer_microflow.go \
    /mnt/data_sdd/gh/mxcli-wt-02/sdk/mpr/writer_domainmodel.go
```

Expected: version reads only in these two files. No `w.reader` usage in other serialize functions.

- [ ] **Step 4: Produce the migration list**

Build two lists:
- **Group A** (no reader dependency): receiver methods that can become standalone immediately
- **Group B** (version-dependent): `serializeMicroflow`, `serializeNanoflow`, `serializeDomainModel` — need parametrization first

No commit for this task — it's research only.

---

## Task 2: Parametrize ProjectVersion in writer_microflow.go

**Files:**
- Modify: `sdk/mpr/writer_microflow.go`
- Test: `sdk/mpr/writer_microflow_test.go` (or existing test file in `sdk/mpr/`)

Context: `serializeMicroflow()` currently calls `w.reader.ProjectVersion()` at the top and threads `majorVersion int` down to:
- `serializeSequenceFlow(flow, majorVersion)` — uses `majorVersion <= 9` for old CaseValues format
- `serializeMicroflowObjectCollectionWithoutFlows(objs, majorVersion)` — uses `majorVersion >= 10` for new fields

- [ ] **Step 1: Write failing test for standalone SerializeMicroflowWithVersion**

In `sdk/mpr/` (find existing test helper with `openTestWriter` or `openTestReader`):

```go
func TestSerializeMicroflowStandalone_V10(t *testing.T) {
	mf := &microflows.Microflow{Name: "TestMF"}
	pv := &ProjectVersion{Major: 10, Minor: 18, Patch: 0}
	_, err := SerializeMicroflow(mf, pv)
	if err != nil {
		t.Fatalf("SerializeMicroflow: %v", err)
	}
}
```

Run: `~/go1.26/bin/go test ./sdk/mpr/... -run TestSerializeMicroflowStandalone -v`
Expected: FAIL with `SerializeMicroflow undefined` or wrong signature

- [ ] **Step 2: Add ProjectVersion type if not already exported**

```bash
grep -n "type ProjectVersion\|ProjectVersion struct" /mnt/data_sdd/gh/mxcli-wt-02/sdk/mpr/reader.go | head -5
```

If `ProjectVersion` already exported from `sdk/mpr`, use it directly. If not, check:
```bash
grep -n "ProjectVersion\|productVersion" /mnt/data_sdd/gh/mxcli-wt-02/sdk/mpr/reader.go | head -10
```

- [ ] **Step 3: Refactor serializeMicroflow to accept *ProjectVersion**

In `sdk/mpr/writer_microflow.go`, find the current function (search for `func (w \*Writer) serializeMicroflow` or `func (w \*Writer) SerializeMicroflow`).

Change the internal helpers to accept `pv *ProjectVersion` instead of reading from `w.reader`:

```go
// Before (private helper signature example):
func (w *Writer) serializeMicroflow(mf *microflows.Microflow) ([]byte, error) {
    pv := w.reader.ProjectVersion()
    majorVersion := 10
    if pv != nil { majorVersion = pv.MajorVersion }
    // ... uses majorVersion
}

// After — new standalone signature:
func SerializeMicroflow(mf *microflows.Microflow, pv *ProjectVersion) ([]byte, error) {
    majorVersion := 10  // safe default for unknown version
    if pv != nil { majorVersion = pv.MajorVersion }
    // same logic, now using local majorVersion
}
```

Update all callee private functions to accept `majorVersion int` as parameter (they probably already do — just remove the `w.reader` read at the top):
- `serializeSequenceFlow(flow *microflows.SequenceFlow, majorVersion int)` — already has param
- `serializeMicroflowObjectCollectionWithoutFlows(objs []microflows.MicroflowObject, majorVersion int)` — already has param

Do the same for `SerializeNanoflow(nf *microflows.Nanoflow, pv *ProjectVersion) ([]byte, error)`.

- [ ] **Step 4: Keep old `(w *Writer) SerializeMicroflow` as a forwarding shim**

Add to maintain backward compatibility with any callers not yet migrated:

```go
// Deprecated: use SerializeMicroflow(mf, w.reader.ProjectVersion()) directly.
func (w *Writer) SerializeMicroflow(mf *microflows.Microflow) ([]byte, error) {
	return SerializeMicroflow(mf, w.reader.ProjectVersion())
}
func (w *Writer) SerializeNanoflow(nf *microflows.Nanoflow) ([]byte, error) {
	return SerializeNanoflow(nf, w.reader.ProjectVersion())
}
```

This lets us migrate callers one by one without a big-bang change.

- [ ] **Step 5: Run test — PASS**

```bash
~/go1.26/bin/go test ./sdk/mpr/... -run TestSerializeMicroflowStandalone -v
```

- [ ] **Step 6: Run full sdk/mpr tests**

```bash
~/go1.26/bin/go test ./sdk/mpr/... -count=1 -timeout 120s
```

- [ ] **Step 7: Commit**

```bash
git add sdk/mpr/writer_microflow.go
git commit -m "refactor(sdk/mpr): parametrize ProjectVersion in SerializeMicroflow/Nanoflow"
```

---

## Task 3: Parametrize ProjectVersion in writer_domainmodel.go

**Files:**
- Modify: `sdk/mpr/writer_domainmodel.go`

Context: `serializeDomainModel()` calls `w.reader.ProjectVersion()` and passes it down to `serializeEntity()` which checks `pv.IsAtLeast(11, 0)` for the Mendix 11 entity OQL field removal.

- [ ] **Step 1: Write failing test for standalone SerializeDomainModel**

```go
func TestSerializeDomainModelStandalone(t *testing.T) {
	dm := &domainmodel.DomainModel{}
	pv := &ProjectVersion{Major: 10, Minor: 18, Patch: 0}
	_, err := SerializeDomainModel(dm, pv)
	if err != nil {
		t.Fatalf("SerializeDomainModel: %v", err)
	}
}
```

Run: `~/go1.26/bin/go test ./sdk/mpr/... -run TestSerializeDomainModelStandalone -v`
Expected: FAIL

- [ ] **Step 2: Refactor serializeDomainModel to accept *ProjectVersion**

```go
// After:
func SerializeDomainModel(dm *domainmodel.DomainModel, pv *ProjectVersion) ([]byte, error) {
    // same logic, pv passed to serializeEntity
}

// Backward-compat shim:
func (w *Writer) SerializeDomainModel(dm *domainmodel.DomainModel) ([]byte, error) {
    return SerializeDomainModel(dm, w.reader.ProjectVersion())
}
```

The private `serializeEntity(e *domainmodel.Entity, moduleName string, pv *ProjectVersion)` already accepts `pv` — just ensure the signature is preserved.

- [ ] **Step 3: Run tests — PASS**

```bash
~/go1.26/bin/go test ./sdk/mpr/... -run TestSerializeDomainModelStandalone -v
~/go1.26/bin/go test ./sdk/mpr/... -count=1 -timeout 120s
```

- [ ] **Step 4: Commit**

```bash
git add sdk/mpr/writer_domainmodel.go
git commit -m "refactor(sdk/mpr): parametrize ProjectVersion in SerializeDomainModel"
```

---

## Task 4: Convert reader-independent `(w *Writer)` Serialize* methods to standalone

**Files:**
- Modify: `sdk/mpr/serialize_exports.go`
- Modify: any other `sdk/mpr/writer_*.go` files with reader-independent Serialize* methods

From the Task 1 audit, these are the Serialize* methods with no `w.reader` dependency. For each one, the conversion is:

```go
// Before:
func (w *Writer) SerializePage(page *pages.Page) ([]byte, error) {
    return serializePage(page)
}

// After (keep the receiver as a shim, add standalone):
func SerializePage(page *pages.Page) ([]byte, error) {
    return serializePage(page)
}
// Deprecated shim:
func (w *Writer) SerializePage(page *pages.Page) ([]byte, error) {
    return SerializePage(page)
}
```

For functions that are already just forwarding wrappers in `serialize_exports.go`:

```bash
# Find the pattern:
grep -A3 "^func (w \*Writer) Serialize" /mnt/data_sdd/gh/mxcli-wt-02/sdk/mpr/serialize_exports.go | head -60
```

If the body is just `return serialize*(arg)`, simply add a standalone version.

- [ ] **Step 1: Convert all reader-independent methods (one batch)**

For each method in Group A (from Task 1 audit), add the standalone version. The full list will be confirmed in Task 1 — expected to include: `SerializePage`, `SerializeLayout`, `SerializeSnippet`, `SerializeWorkflow`, `SerializeModule`, `SerializeModuleSecurity`, `SerializeModuleSettings`, `SerializeFolder`, `SerializeEnumeration`, `SerializeConstant`, `SerializeJavaAction`, `SerializeDatabaseConnection`, `SerializeImportMapping`, `SerializeExportMapping`, `SerializeBusinessEventService`, `SerializeConsumedODataService`, `SerializePublishedODataService`, `SerializeConsumedRestService`, `SerializePublishedRestService`, `SerializeProjectSettings`.

- [ ] **Step 2: Build and test**

```bash
~/go1.26/bin/go build ./...
~/go1.26/bin/go test ./sdk/mpr/... -count=1 -timeout 120s
```

- [ ] **Step 3: Commit**

```bash
git add sdk/mpr/serialize_exports.go sdk/mpr/writer_pages.go # etc.
git commit -m "refactor(sdk/mpr): expose standalone Serialize* functions alongside Writer receiver shims"
```

---

## Task 5: Update MprBackend callers — mf_page_modelsdk.go

**Files:**
- Modify: `mdl/backend/mpr/mf_page_modelsdk.go`

Replace all `b.writer.SerializeXxx` with the new standalone `mpr.SerializeXxx`. For the version-dependent ones (Microflow, Nanoflow, DomainModel), pass `b.reader.ProjectVersion()`.

- [ ] **Step 1: Confirm current b.reader is accessible**

```bash
grep -n "b\.reader\." /mnt/data_sdd/gh/mxcli-wt-02/mdl/backend/mpr/mf_page_modelsdk.go | head -5
grep -n "b\.reader\b" /mnt/data_sdd/gh/mxcli-wt-02/mdl/backend/mpr/backend.go | head -5
```

`b.reader` is `*mpr.Reader` — it has `ProjectVersion()` method.

- [ ] **Step 2: Replace b.writer.Serialize* in mf_page_modelsdk.go**

For each of the 13 calls in this file:

**Microflow (line ~35, 52):**
```go
// before:
contents, err := b.writer.SerializeMicroflow(mf)
// after:
contents, err := mpr.SerializeMicroflow(mf, b.reader.ProjectVersion())
```

**Nanoflow (line ~69, 86):**
```go
// before:
contents, err := b.writer.SerializeNanoflow(nf)
// after:
contents, err := mpr.SerializeNanoflow(nf, b.reader.ProjectVersion())
```

**Page, Layout, Snippet (lines ~103-192, 6 calls total):**
```go
// before:
contents, err := b.writer.SerializePage(page)
// after:
contents, err := mpr.SerializePage(page)
```
(No version argument — these are reader-independent)

**Workflow (lines ~205, 222):**
```go
// before:
contents, err := b.writer.SerializeWorkflow(wf)
// after:
contents, err := mpr.SerializeWorkflow(wf)
```

**DomainModel (line ~241):**
```go
// before:
contents, err := b.writer.SerializeDomainModel(dm)
// after:
contents, err := mpr.SerializeDomainModel(dm, b.reader.ProjectVersion())
```

- [ ] **Step 3: Build and test**

```bash
~/go1.26/bin/go build ./...
~/go1.26/bin/go test ./mdl/backend/mpr/... -count=1 -timeout 120s
```

- [ ] **Step 4: Commit**

```bash
git add mdl/backend/mpr/mf_page_modelsdk.go
git commit -m "refactor(backend): use standalone mpr.Serialize* in mf_page_modelsdk (no b.writer needed)"
```

---

## Task 6: Update MprBackend callers — create_services_modelsdk.go, modules_modelsdk.go, settings_modelsdk.go

**Files:**
- Modify: `mdl/backend/mpr/create_services_modelsdk.go`
- Modify: `mdl/backend/mpr/modules_modelsdk.go`
- Modify: `mdl/backend/mpr/settings_modelsdk.go`

- [ ] **Step 1: Replace b.writer.Serialize* in create_services_modelsdk.go**

```bash
grep -n "b\.writer\.Serialize" /mnt/data_sdd/gh/mxcli-wt-02/mdl/backend/mpr/create_services_modelsdk.go
```

For each hit (expected: SerializeJavaAction, SerializeDatabaseConnection, SerializeImportMapping, SerializeExportMapping, SerializeBusinessEventService, SerializeConsumedODataService, SerializePublishedODataService, SerializeConsumedRestService, SerializePublishedRestService, SerializeEnumeration, SerializeConstant):

```go
// before:
contents, err := b.writer.SerializeEnumeration(enum)
// after:
contents, err := mpr.SerializeEnumeration(enum)
```

None of these service-type serializers are version-dependent — no `pv` argument needed.

- [ ] **Step 2: Replace b.writer.Serialize* in modules_modelsdk.go**

```bash
grep -n "b\.writer\.Serialize" /mnt/data_sdd/gh/mxcli-wt-02/mdl/backend/mpr/modules_modelsdk.go
```

Expected: SerializeModule, SerializeDomainModel, SerializeModuleSecurity, SerializeModuleSettings, SerializeFolder

```go
// before:
moduleContents, err := b.writer.SerializeModule(module)
// after:
moduleContents, err := mpr.SerializeModule(module)

// DomainModel — version-dependent:
dmContents, err := b.writer.SerializeDomainModel(dm)
// after:
dmContents, err := mpr.SerializeDomainModel(dm, b.reader.ProjectVersion())
```

- [ ] **Step 3: Replace b.writer.Serialize* in settings_modelsdk.go**

```bash
grep -n "b\.writer\.Serialize" /mnt/data_sdd/gh/mxcli-wt-02/mdl/backend/mpr/settings_modelsdk.go
```

Expected: SerializeProjectSettings

```go
// before:
contents, err := b.writer.SerializeProjectSettings(ps)
// after:
contents, err := mpr.SerializeProjectSettings(ps)
```

- [ ] **Step 4: Build and test**

```bash
~/go1.26/bin/go build ./...
~/go1.26/bin/go test ./mdl/backend/mpr/... ./mdl/executor/... -count=1 -timeout 120s
```

- [ ] **Step 5: Commit**

```bash
git add mdl/backend/mpr/create_services_modelsdk.go mdl/backend/mpr/modules_modelsdk.go mdl/backend/mpr/settings_modelsdk.go
git commit -m "refactor(backend): use standalone mpr.Serialize* in create_services, modules, settings"
```

---

## Task 7: Verify b.writer is only used for Close() — remove deprecated shims

**Files:**
- Modify: `sdk/mpr/serialize_exports.go` (delete shim methods)
- Modify: `sdk/mpr/writer_microflow.go` (delete shim methods)
- Modify: `sdk/mpr/writer_domainmodel.go` (delete shim methods)

- [ ] **Step 1: Confirm zero b.writer.Serialize* calls remain in MprBackend**

```bash
grep -rn "b\.writer\.Serialize" /mnt/data_sdd/gh/mxcli-wt-02/mdl/backend/mpr/*.go
```

Expected: no output.

- [ ] **Step 2: Check if any other callers of the Writer receiver methods exist outside MprBackend**

```bash
grep -rn "\.writer\.Serialize\|Writer\.Serialize\|\.SerializeMicroflow\|\.SerializeNanoflow\|\.SerializeDomainModel" \
    /mnt/data_sdd/gh/mxcli-wt-02/ --include="*.go" | grep -v "_test.go" | grep -v "sdk/mpr/"
```

If no external callers remain, the deprecated shim methods in `sdk/mpr/` can be deleted.

- [ ] **Step 3: Delete deprecated `(w *Writer) Serialize*` shim methods**

In each file (`serialize_exports.go`, `writer_microflow.go`, `writer_domainmodel.go`), delete the shim methods added in Tasks 2-4. The standalone functions remain.

- [ ] **Step 4: Check remaining b.writer usage in backend.go**

```bash
grep -n "b\.writer\." /mnt/data_sdd/gh/mxcli-wt-02/mdl/backend/mpr/backend.go
```

Expected: only `b.writer.Close()` remains.

- [ ] **Step 5: Build and full test**

```bash
~/go1.26/bin/go build ./...
~/go1.26/bin/go test ./sdk/mpr/... ./mdl/... -count=1 -timeout 300s
```

Expected: all pass (skip `cmd/exprgrammar-mine` — pre-existing timeout).

- [ ] **Step 6: Commit**

```bash
git add sdk/mpr/serialize_exports.go sdk/mpr/writer_microflow.go sdk/mpr/writer_domainmodel.go
git commit -m "refactor(sdk/mpr): delete deprecated Writer Serialize* receiver shims — standalone functions are canonical"
```

---

## Task 8: (Optional) Remove b.writer field if Close() can be moved

**Prerequisite:** Only if the connection lifecycle can be restructured so `b.writer.Close()` is unnecessary. This is the "connection lifecycle redesign" deferred in previous plans.

Check current Close() usage:

```bash
grep -n "b\.writer\.Close\|writer\.Close" /mnt/data_sdd/gh/mxcli-wt-02/mdl/backend/mpr/backend.go
```

If `b.writer.Close()` is the only remaining usage, evaluate whether closing the shared `*sql.DB` (via `msdkWriter.Close()`) is sufficient. If yes:
- Remove `b.writer` field from `MprBackend`
- Remove `mpr.NewWriter` call from `Connect`
- Remove `b.writer = nil` from `Disconnect`

This is a judgment call based on whether the sdk/mpr Writer's Close() does anything beyond closing the database connection that's already shared.

Do NOT implement Task 8 without first verifying that nothing breaks. If uncertain, skip and document as deferred.

---

## Self-Review

**Spec coverage:**
- [x] Version parametrization for `SerializeMicroflow/Nanoflow` — Task 2
- [x] Version parametrization for `SerializeDomainModel` — Task 3
- [x] Reader-independent methods → standalone — Task 4
- [x] MprBackend callers: mf_page_modelsdk — Task 5
- [x] MprBackend callers: create_services, modules, settings — Task 6
- [x] Shim deletion + final verification — Task 7
- [x] `b.writer` lifecycle (optional) — Task 8
- [x] Embedded serializers (Widget/Action/DataSource) — explicitly out of scope, already standalone

**Placeholder scan:** All code blocks show exact before/after. `ProjectVersion` type confirmed accessible via `b.reader.ProjectVersion()`. Shim pattern is concrete.

**Type consistency:**
- `SerializeMicroflow(mf *microflows.Microflow, pv *ProjectVersion) ([]byte, error)` — consistent across Tasks 2 and 5.
- `SerializeDomainModel(dm *domainmodel.DomainModel, pv *ProjectVersion) ([]byte, error)` — consistent across Tasks 3, 5, 6.
- All reader-independent standalone functions have no `pv` parameter — consistent across Tasks 4 and 6.
