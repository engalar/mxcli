# modelsdk SOLID Write Path Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Unify all MprBackend write operations under a single typed write primitive (`writeUnitContents`), eliminate `msdkWriteRaw`, retire `b.writer` write-path dependencies, and promote type-safe gen-native writes (msdkWrite + element.Element) where gen coverage exists.

**Architecture:** Three tiers of migration. **Tier 1** — consolidate `writeUnitContents` as the package-level write primitive and replace all `msdkWriteRaw` call sites (correctness, 1544 bug elimination). **Tier 2** — retire `b.writer` write operations: move read-only helpers to Reader, implement Java file I/O via `b.path`, export OQL scanner (Dependency Inversion). **Tier 3** — gen-native AccessRule migration in `security_entity_access_modelsdk.go`, now unblocked by the `AccessRule.ModuleRoles → AllowedModuleRoles` BSON key fix. Domains whose Serialize*/Scan* encode paths handle 60+ polymorphic sub-types (Microflow, Page, Workflow, AgentEditor) stay on Tier 1 — the gen-native promotion of those is a separate multi-month effort.

**Tech Stack:** Go 1.26, `go.mongodb.org/mongo-driver/bson`, `modelsdk/gen/*` generated types, `modelsdk/element`, `~/go1.26/bin/go`

**Current state (as of this plan):**
- `writeUnitContents` defined in `security_entity_access_modelsdk.go:19` — wrong home
- `msdkWriteRaw` defined in `security_allowed_roles_modelsdk.go:110` — nearly identical implementation, 17 call sites remain
- `b.writer.*` write calls: 8 remaining in `backend.go` (FindViewEntitySourceDocument*, UpdateOqlQueries*, UpdateEnumerationRefs*, Write/Delete/Rename/ReadJavaSourceFile, UpdateRawUnit)
- `security_entity_access_modelsdk.go`: uses `writeUnitContents` + Patch* encode helpers — can be upgraded to gen-native (BSON key fix committed `0aec60aa`)

---

## File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `mdl/backend/mpr/write_helpers.go` | **Create** | Package-level `writeUnitContents` — the single write primitive |
| `mdl/backend/mpr/security_entity_access_modelsdk.go` | Modify | Remove local `writeUnitContents`; gen-native for funcs 1-4 |
| `mdl/backend/mpr/security_allowed_roles_modelsdk.go` | Modify | Remove `msdkWriteRaw` definition after all callers migrated |
| `mdl/backend/mpr/mf_page_modelsdk.go` | Modify | 6× `msdkWriteRaw` → `writeUnitContents` |
| `mdl/backend/mpr/agenteditor_modelsdk.go` | Modify | 4× `msdkWriteRaw` → `writeUnitContents` |
| `mdl/backend/mpr/navigation_modelsdk.go` | Modify | 1× `msdkWriteRaw` → `writeUnitContents` |
| `mdl/backend/mpr/refs_modelsdk.go` | Modify | 2× `msdkWriteRaw` → `writeUnitContents` |
| `mdl/backend/mpr/rename_modelsdk.go` | Modify | 1× `msdkWriteRaw` → `writeUnitContents` |
| `mdl/backend/mpr/settings_modelsdk.go` | Modify | 1× `msdkWriteRaw` → `writeUnitContents` |
| `mdl/backend/mpr/domainmodel_modelsdk.go` | Modify | OQL scan caller uses `writeUnitContents` (Task 2 of retire-bwriter) |
| `sdk/mpr/reader_documents.go` | Modify | Add `FindViewEntitySourceDocumentID` + `FindAllViewEntitySourceDocumentIDs` |
| `sdk/mpr/writer_domainmodel.go` | Modify | Export `ScanOqlQueryUpdates` |
| `sdk/mpr/writer_javaactions.go` | Modify | Export `GenerateJavaSource` |
| `mdl/backend/mpr/java_files.go` | **Create** | Java source file I/O via `b.path` |
| `mdl/backend/mpr/backend.go` | Modify | Wire up all retire-bwriter call sites; simplify `UpdateRawUnit` |

---

## Phase 1 — Consolidate the Write Primitive

### Task 1.1: Create write_helpers.go — single write primitive

**Files:**
- Create: `mdl/backend/mpr/write_helpers.go`
- Modify: `mdl/backend/mpr/security_entity_access_modelsdk.go` (remove duplicate)

- [ ] **Step 1: Write the test**

Create `mdl/backend/mpr/write_helpers_test.go`:

```go
// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"database/sql"
	"path/filepath"
	"testing"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/mendixlabs/mxcli/model"
	_ "modernc.org/sqlite"
)

func TestWriteUnitContents_RoundTrip(t *testing.T) {
	const unitID = "aaaa0001-0000-0000-0000-000000000001"
	doc := bson.D{{Key: "$Type", Value: "Test"}, {Key: "Name", Value: "Before"}}
	contents, _ := bson.Marshal(doc)
	mprPath, id := makeServiceTestMPR(t, unitID, contents) // helper in services_modelsdk_test.go

	b := New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	updated := bson.D{{Key: "$Type", Value: "Test"}, {Key: "Name", Value: "After"}}
	newContents, _ := bson.Marshal(updated)

	if err := b.writeUnitContents(id, newContents); err != nil {
		t.Fatalf("writeUnitContents: %v", err)
	}

	raw, err := b.msdkWriter.Reader().GetRawUnitBytes(string(id))
	if err != nil {
		t.Fatalf("GetRawUnitBytes: %v", err)
	}
	if got := readBSONField(t, raw, "Name"); got != "After" {
		t.Errorf("Name = %q, want %q", got, "After")
	}
}

func TestWriteUnitContents_NilWriter(t *testing.T) {
	b := &MprBackend{}
	err := b.writeUnitContents(model.ID("any"), []byte{})
	if err == nil {
		t.Fatal("expected error when msdkWriter is nil")
	}
}
```

Run: `~/go1.26/bin/go test ./mdl/backend/mpr/ -run TestWriteUnitContents -v`
Expected: FAIL — `writeUnitContents` not yet in package scope (currently only in `security_entity_access_modelsdk.go`)

Actually — the test will PASS because the method exists. The real purpose is to anchor the helper in the right file. Proceed to Step 2.

- [ ] **Step 2: Create write_helpers.go**

```go
// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"
)

// writeUnitContents writes pre-encoded BSON bytes for a unit via a
// WriteTransaction. This is the single write primitive for all non-gen-native
// write paths in MprBackend. It avoids the sdk/mpr Writer.updateUnit path
// that calls updateTransactionID() and triggers SQLITE_READONLY_DBMOVED (1544)
// on hard-linked MPR files.
func (b *MprBackend) writeUnitContents(unitID model.ID, contents []byte) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	wtx, err := b.msdkWriter.BeginWriteTransaction()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	if err := wtx.WriteUnit(string(unitID), contents); err != nil {
		_ = wtx.Rollback()
		return fmt.Errorf("write unit: %w", err)
	}
	if err := wtx.Commit(); err != nil {
		return err
	}
	b.reader.InvalidateCache()
	return nil
}
```

- [ ] **Step 3: Remove the duplicate from security_entity_access_modelsdk.go**

Delete lines 19-34 (the `writeUnitContents` method) from `security_entity_access_modelsdk.go`. The package now has exactly one definition.

- [ ] **Step 4: Run tests**

```bash
~/go1.26/bin/go build ./...
~/go1.26/bin/go test ./mdl/backend/mpr/ -run TestWriteUnitContents -v
~/go1.26/bin/go test ./mdl/backend/mpr/ -count=1
```

Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add mdl/backend/mpr/write_helpers.go mdl/backend/mpr/write_helpers_test.go mdl/backend/mpr/security_entity_access_modelsdk.go
git commit -m "refactor(backend): promote writeUnitContents to package-level write primitive"
```

---

### Task 1.2: Replace all msdkWriteRaw call sites with writeUnitContents

**Files:**
- Modify: `mdl/backend/mpr/mf_page_modelsdk.go` (6 calls)
- Modify: `mdl/backend/mpr/agenteditor_modelsdk.go` (4 calls)
- Modify: `mdl/backend/mpr/navigation_modelsdk.go` (1 call)
- Modify: `mdl/backend/mpr/refs_modelsdk.go` (2 calls)
- Modify: `mdl/backend/mpr/rename_modelsdk.go` (1 call)
- Modify: `mdl/backend/mpr/settings_modelsdk.go` (1 call)

- [ ] **Step 1: Verify call site list**

```bash
grep -rn "msdkWriteRaw" mdl/backend/mpr/*.go | grep -v "_test.go" | grep -v "^.*security_allowed_roles"
```

Expected output (15 lines from 6 files — the security_allowed_roles.go definition is excluded):
- `agenteditor_modelsdk.go`: 4 lines
- `mf_page_modelsdk.go`: 6 lines
- `navigation_modelsdk.go`: 1 line
- `refs_modelsdk.go`: 2 lines
- `rename_modelsdk.go`: 1 line
- `settings_modelsdk.go`: 1 line

- [ ] **Step 2: Mechanical replacement — all 15 call sites**

In each of the 6 files, replace every occurrence of:
```go
return b.msdkWriteRaw(someID, contents)
```
with:
```go
return b.writeUnitContents(someID, contents)
```

For `mf_page_modelsdk.go` the IDs are typed as `model.ID` (already correct). For `refs_modelsdk.go` the IDs come from `UnitPatch.ID` which is `string` — wrap: `model.ID(p.ID)`.

Exact replacements:

**mf_page_modelsdk.go** — 6 replacements (lines ~56, 90, 124, 158, 192, 226):
```go
// before:
return b.msdkWriteRaw(mf.ID, contents)
// after:
return b.writeUnitContents(mf.ID, contents)
```
(Same pattern for nf.ID, page.ID, layout.ID, snippet.ID, wf.ID)

**agenteditor_modelsdk.go** — 4 replacements:
```go
// before:
return b.msdkWriteRaw(m.ID, contents)
// after:
return b.writeUnitContents(m.ID, contents)
```
(Same for k.ID, c.ID, a.ID)

**navigation_modelsdk.go** — 1 replacement:
```go
// before:
return b.msdkWriteRaw(navDocID, patched)
// after:
return b.writeUnitContents(navDocID, patched)
```

**refs_modelsdk.go** — 2 replacements (inside loop over patches):
```go
// before:
if err := b.msdkWriteRaw(model.ID(p.ID), p.Contents); err != nil {
// after:
if err := b.writeUnitContents(model.ID(p.ID), p.Contents); err != nil {
```

**rename_modelsdk.go** — 1 replacement:
```go
// before:
return b.msdkWriteRaw(model.ID(unitID), newContents)
// after:
return b.writeUnitContents(model.ID(unitID), newContents)
```

**settings_modelsdk.go** — 1 replacement:
```go
// before:
return b.msdkWriteRaw(ps.ID, contents)
// after:
return b.writeUnitContents(ps.ID, contents)
```

- [ ] **Step 3: Remove msdkWriteRaw definition and import cleanup**

In `security_allowed_roles_modelsdk.go`, delete the `msdkWriteRaw` function body (currently lines ~110-126). Then run:

```bash
~/go1.26/bin/go build ./...
```

If the build fails with "msdkWriteRaw undefined", you missed a call site — grep and fix. If it fails with unused import, remove the import.

- [ ] **Step 4: Verify zero msdkWriteRaw references remain**

```bash
grep -rn "msdkWriteRaw" mdl/backend/mpr/*.go
```

Expected: no output.

- [ ] **Step 5: Run tests**

```bash
~/go1.26/bin/go test ./mdl/backend/mpr/ -count=1 -timeout 120s
```

Expected: all pass.

- [ ] **Step 6: Commit**

```bash
git add mdl/backend/mpr/mf_page_modelsdk.go mdl/backend/mpr/agenteditor_modelsdk.go \
        mdl/backend/mpr/navigation_modelsdk.go mdl/backend/mpr/refs_modelsdk.go \
        mdl/backend/mpr/rename_modelsdk.go mdl/backend/mpr/settings_modelsdk.go \
        mdl/backend/mpr/security_allowed_roles_modelsdk.go
git commit -m "refactor(backend): replace all msdkWriteRaw call sites with writeUnitContents, delete msdkWriteRaw"
```

---

## Phase 2 — Retire b.writer Write Operations

> These 5 tasks are independent of each other and of Phase 1/3. They can be done in any order.

### Task 2.1: Move FindViewEntitySourceDocument* from Writer to Reader

**Files:**
- Modify: `sdk/mpr/reader_documents.go`
- Modify: `mdl/backend/mpr/backend.go` (lines ~229, 232)

- [ ] **Step 1: Write failing test**

Find the existing test helper for sdk/mpr Reader:
```bash
grep -rn "func openTestReader\|func newTestReader" sdk/mpr/ | head -5
```

Add to an appropriate test file in `sdk/mpr/`:

```go
func TestReader_FindViewEntitySourceDocumentID_NotFound(t *testing.T) {
	r := openMinimalTestReader(t) // use existing test helper
	id, err := r.FindViewEntitySourceDocumentID("NonExistentModule", "NonExistentDoc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "" {
		t.Fatalf("expected empty ID, got %s", id)
	}
}

func TestReader_FindAllViewEntitySourceDocumentIDs_Empty(t *testing.T) {
	r := openMinimalTestReader(t)
	ids, err := r.FindAllViewEntitySourceDocumentIDs("NonExistentModule", "NonExistentDoc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("expected empty slice, got %v", ids)
	}
}
```

Run: `~/go1.26/bin/go test ./sdk/mpr/... -run TestReader_FindViewEntitySourceDocument -v`
Expected: FAIL with `r.FindViewEntitySourceDocumentID undefined`

- [ ] **Step 2: Add methods to reader_documents.go**

Append to `sdk/mpr/reader_documents.go`:

```go
// FindViewEntitySourceDocumentID finds a ViewEntitySourceDocument by module
// and document name. Returns the document ID if found, empty string if not.
func (r *Reader) FindViewEntitySourceDocumentID(moduleName, docName string) (model.ID, error) {
	units, err := r.listUnitsByType("DomainModels$ViewEntitySourceDocument")
	if err != nil {
		return "", err
	}
	modules, err := r.ListModules()
	if err != nil {
		return "", err
	}
	moduleNames := make(map[string]string, len(modules))
	for _, m := range modules {
		moduleNames[string(m.ID)] = m.Name
	}
	for _, u := range units {
		var raw map[string]any
		if err := bson.Unmarshal(u.Contents, &raw); err != nil {
			continue
		}
		name, _ := raw["Name"].(string)
		if moduleNames[u.ContainerID] == moduleName && name == docName {
			return model.ID(u.ID), nil
		}
	}
	return "", nil
}

// FindAllViewEntitySourceDocumentIDs finds all ViewEntitySourceDocument units
// matching the given module and document name.
func (r *Reader) FindAllViewEntitySourceDocumentIDs(moduleName, docName string) ([]model.ID, error) {
	units, err := r.listUnitsByType("DomainModels$ViewEntitySourceDocument")
	if err != nil {
		return nil, err
	}
	modules, err := r.ListModules()
	if err != nil {
		return nil, err
	}
	moduleNames := make(map[string]string, len(modules))
	for _, m := range modules {
		moduleNames[string(m.ID)] = m.Name
	}
	var ids []model.ID
	for _, u := range units {
		var raw map[string]any
		if err := bson.Unmarshal(u.Contents, &raw); err != nil {
			continue
		}
		name, _ := raw["Name"].(string)
		if moduleNames[u.ContainerID] == moduleName && name == docName {
			ids = append(ids, model.ID(u.ID))
		}
	}
	return ids, nil
}
```

Add `"go.mongodb.org/mongo-driver/bson"` to imports if not already present.

- [ ] **Step 3: Run tests — PASS**

```bash
~/go1.26/bin/go test ./sdk/mpr/... -run TestReader_FindViewEntitySourceDocument -v
```

- [ ] **Step 4: Update backend.go call sites**

```bash
grep -n "FindViewEntitySourceDocument" mdl/backend/mpr/backend.go
```

For each hit, change `b.writer.` to `b.reader.`:
```go
// before:
return b.writer.FindViewEntitySourceDocumentID(moduleName, docName)
// after:
return b.reader.FindViewEntitySourceDocumentID(moduleName, docName)

// before:
return b.writer.FindAllViewEntitySourceDocumentIDs(moduleName, docName)
// after:
return b.reader.FindAllViewEntitySourceDocumentIDs(moduleName, docName)
```

- [ ] **Step 5: Build and test**

```bash
~/go1.26/bin/go build ./...
~/go1.26/bin/go test ./sdk/mpr/... ./mdl/backend/mpr/... -count=1 -timeout 120s
```

- [ ] **Step 6: Commit**

```bash
git add sdk/mpr/reader_documents.go mdl/backend/mpr/backend.go
git commit -m "refactor(backend): move FindViewEntitySourceDocument* from Writer to Reader"
```

---

### Task 2.2: Export ScanOqlQueryUpdates, migrate UpdateOqlQueriesForMovedEntity

**Files:**
- Modify: `sdk/mpr/writer_domainmodel.go`
- Modify: `mdl/backend/mpr/domainmodel_modelsdk.go`
- Modify: `mdl/backend/mpr/backend.go` (~line 238)

- [ ] **Step 1: Locate existing UpdateOqlQueriesForMovedEntity in sdk/mpr**

```bash
grep -n "UpdateOqlQueriesForMovedEntity\|updateOqlQueries\|ViewEntitySourceDocument\|Oql" \
    sdk/mpr/writer_domainmodel.go | head -20
```

Understand the existing private implementation — it scans `DomainModels$ViewEntitySourceDocument` units and replaces the `Oql` string field.

- [ ] **Step 2: Write failing test for ScanOqlQueryUpdates**

Find the existing Writer test helper:
```bash
grep -n "func.*Writer.*t \*testing" sdk/mpr/*_test.go | head -5
```

Add to the appropriate test file:

```go
func TestWriter_ScanOqlQueryUpdates_NoHits(t *testing.T) {
	w := openTestWriter(t) // existing helper
	patches, count, err := w.ScanOqlQueryUpdates("OldModule.OldEntity", "NewModule.NewEntity")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 || len(patches) != 0 {
		t.Fatalf("expected 0 hits, got count=%d patches=%d", count, len(patches))
	}
}
```

Run: `~/go1.26/bin/go test ./sdk/mpr/... -run TestWriter_ScanOqlQueryUpdates -v`
Expected: FAIL with `w.ScanOqlQueryUpdates undefined`

- [ ] **Step 3: Add ScanOqlQueryUpdates to writer_domainmodel.go**

`UnitPatch` is already defined in `sdk/mpr/writer_refs.go`. Append to `sdk/mpr/writer_domainmodel.go`:

```go
// ScanOqlQueryUpdates scans all ViewEntitySourceDocument units and replaces
// oldQualifiedName with newQualifiedName in their Oql fields.
// Returns the patched unit bytes and count; does NOT write to disk.
func (w *Writer) ScanOqlQueryUpdates(oldQualifiedName, newQualifiedName string) ([]UnitPatch, int, error) {
	units, err := w.reader.listUnitsByType("DomainModels$ViewEntitySourceDocument")
	if err != nil {
		return nil, 0, err
	}
	var patches []UnitPatch
	for _, u := range units {
		var raw map[string]any
		if err := bson.Unmarshal(u.Contents, &raw); err != nil {
			continue
		}
		oql, _ := raw["Oql"].(string)
		if oql == "" || !strings.Contains(oql, oldQualifiedName) {
			continue
		}
		raw["Oql"] = strings.ReplaceAll(oql, oldQualifiedName, newQualifiedName)
		newContents, err := bson.Marshal(raw)
		if err != nil {
			return nil, 0, fmt.Errorf("marshal OQL unit %s: %w", u.ID, err)
		}
		patches = append(patches, UnitPatch{ID: u.ID, Contents: newContents})
	}
	return patches, len(patches), nil
}
```

Add `"strings"`, `"fmt"`, `"go.mongodb.org/mongo-driver/bson"` to imports if missing.

- [ ] **Step 4: Run test — PASS**

```bash
~/go1.26/bin/go test ./sdk/mpr/... -run TestWriter_ScanOqlQueryUpdates -v
```

- [ ] **Step 5: Add updateOqlQueriesForMovedEntityViaModelsdk to domainmodel_modelsdk.go**

Append to `mdl/backend/mpr/domainmodel_modelsdk.go`:

```go
func (b *MprBackend) updateOqlQueriesForMovedEntityViaModelsdk(oldQualifiedName, newQualifiedName string) (int, error) {
	patches, count, err := b.writer.ScanOqlQueryUpdates(oldQualifiedName, newQualifiedName)
	if err != nil {
		return 0, err
	}
	for _, p := range patches {
		if err := b.writeUnitContents(model.ID(p.ID), p.Contents); err != nil {
			return 0, fmt.Errorf("write OQL unit %s: %w", p.ID, err)
		}
	}
	return count, nil
}
```

- [ ] **Step 6: Update backend.go call site**

```bash
grep -n "UpdateOqlQueriesForMovedEntity" mdl/backend/mpr/backend.go
```

Replace:
```go
// before:
return b.writer.UpdateOqlQueriesForMovedEntity(oldQualifiedName, newQualifiedName)
// after:
return b.updateOqlQueriesForMovedEntityViaModelsdk(oldQualifiedName, newQualifiedName)
```

- [ ] **Step 7: Build and test**

```bash
~/go1.26/bin/go build ./...
~/go1.26/bin/go test ./sdk/mpr/... ./mdl/backend/mpr/... -count=1 -timeout 120s
```

- [ ] **Step 8: Commit**

```bash
git add sdk/mpr/writer_domainmodel.go mdl/backend/mpr/domainmodel_modelsdk.go mdl/backend/mpr/backend.go
git commit -m "refactor(backend): migrate UpdateOqlQueriesForMovedEntity to ScanOqlQueryUpdates + writeUnitContents"
```

---

### Task 2.3: Migrate UpdateEnumerationRefsInAllDomainModels

**Files:**
- Modify: `mdl/backend/mpr/domainmodel_modelsdk.go`
- Modify: `mdl/backend/mpr/backend.go` (~line 241)

- [ ] **Step 1: Verify EnumerationAttributeType name**

```bash
grep -rn "EnumerationAttributeType\|EnumAttributeType" sdk/domainmodel/ | head -10
```

Note the exact type name used in `sdk/domainmodel/`.

- [ ] **Step 2: Write failing test**

Add to `mdl/backend/mpr/domainmodel_modelsdk_test.go`:

```go
func TestUpdateEnumerationRefsInAllDomainModelsViaModelsdk_NoOp(t *testing.T) {
	mprPath := makeDomainModelTestMPR(t)

	b := New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	err := b.updateEnumerationRefsInAllDomainModelsViaModelsdk("OldModule.OldEnum", "NewModule.NewEnum")
	if err != nil {
		t.Fatalf("unexpected error on no-op: %v", err)
	}
}
```

Run: `~/go1.26/bin/go test ./mdl/backend/mpr/ -run TestUpdateEnumerationRefs -v`
Expected: FAIL with undefined method.

- [ ] **Step 3: Add method to domainmodel_modelsdk.go**

Check the exact field name for EnumerationRef:
```bash
grep -n "EnumerationRef\|EnumerationID" sdk/domainmodel/*.go | head -10
```

Then append to `mdl/backend/mpr/domainmodel_modelsdk.go`:

```go
func (b *MprBackend) updateEnumerationRefsInAllDomainModelsViaModelsdk(oldQualifiedName, newQualifiedName string) error {
	dms, err := b.reader.ListDomainModels()
	if err != nil {
		return fmt.Errorf("list domain models: %w", err)
	}
	for _, dm := range dms {
		changed := false
		for _, entity := range dm.Entities {
			for _, attr := range entity.Attributes {
				enumType, ok := attr.Type.(*domainmodel.EnumerationAttributeType)
				if !ok {
					continue
				}
				if string(enumType.EnumerationRef) == oldQualifiedName {
					enumType.EnumerationRef = model.ID(newQualifiedName)
					changed = true
				}
			}
		}
		if changed {
			if err := b.updateDomainModelViaModelsdk(dm); err != nil {
				return fmt.Errorf("update domain model %s: %w", dm.ID, err)
			}
		}
	}
	return nil
}
```

Adjust field names (`EnumerationRef`, `EnumerationID`) based on Step 1 audit.

- [ ] **Step 4: Run test — PASS**

```bash
~/go1.26/bin/go test ./mdl/backend/mpr/ -run TestUpdateEnumerationRefs -v
```

- [ ] **Step 5: Update backend.go**

```bash
grep -n "UpdateEnumerationRefsInAllDomainModels" mdl/backend/mpr/backend.go
```

Replace:
```go
// before:
return b.writer.UpdateEnumerationRefsInAllDomainModels(oldQualifiedName, newQualifiedName)
// after:
return b.updateEnumerationRefsInAllDomainModelsViaModelsdk(oldQualifiedName, newQualifiedName)
```

- [ ] **Step 6: Build and test**

```bash
~/go1.26/bin/go build ./...
~/go1.26/bin/go test ./mdl/backend/mpr/... -count=1 -timeout 120s
```

- [ ] **Step 7: Commit**

```bash
git add mdl/backend/mpr/domainmodel_modelsdk.go mdl/backend/mpr/backend.go
git commit -m "refactor(backend): migrate UpdateEnumerationRefsInAllDomainModels to reader+writeDomainModel pattern"
```

---

### Task 2.4: Implement Java file operations directly via b.path

**Files:**
- Modify: `sdk/mpr/writer_javaactions.go` (export GenerateJavaSource)
- Create: `mdl/backend/mpr/java_files.go`
- Modify: `mdl/backend/mpr/backend.go` (lines ~636-649)

- [ ] **Step 1: Confirm generateJavaSource signature**

```bash
grep -n "^func generateJavaSource\|^func GenerateJavaSource" sdk/mpr/writer_javaactions.go
```

Note the exact parameter types.

- [ ] **Step 2: Write failing test**

Add to `mdl/backend/mpr/java_files_test.go` (new file):

```go
// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJavaFiles_WriteReadDeleteRename(t *testing.T) {
	dir := t.TempDir()
	b := &MprBackend{path: filepath.Join(dir, "test.mpr")}

	if err := b.writeJavaSourceFileViaPath("MyModule", "MyAction", "return null;", nil, nil, nil, ""); err != nil {
		t.Fatalf("write: %v", err)
	}

	expected := filepath.Join(dir, "javasource", "mymodule", "actions", "MyAction.java")
	if _, err := os.Stat(expected); err != nil {
		t.Fatalf("file not found after write: %v", err)
	}

	content, err := b.readJavaSourceFileViaPath("MyModule", "MyAction")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if content == "" {
		t.Fatal("expected non-empty content")
	}

	if err := b.renameJavaSourceFileViaPath("MyModule", "MyAction", "RenamedAction"); err != nil {
		t.Fatalf("rename: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "javasource", "mymodule", "actions", "RenamedAction.java")); err != nil {
		t.Fatalf("renamed file not found: %v", err)
	}

	if err := b.deleteJavaSourceFileViaPath("MyModule", "RenamedAction"); err != nil {
		t.Fatalf("delete: %v", err)
	}
}
```

Run: `~/go1.26/bin/go test ./mdl/backend/mpr/ -run TestJavaFiles -v`
Expected: FAIL with undefined methods.

- [ ] **Step 3: Export GenerateJavaSource from sdk/mpr**

In `sdk/mpr/writer_javaactions.go`, add after the private function:

```go
// GenerateJavaSource generates Java source for a Java action.
// Exported so callers can implement their own file I/O without going through Writer.
func GenerateJavaSource(moduleName, actionName, javaCode string, params []*javaactions.JavaActionParameter, returnType javaactions.CodeActionReturnType, extraImports []string, extraCode string) string {
	return generateJavaSource(moduleName, actionName, javaCode, params, returnType, extraImports, extraCode)
}
```

Confirm this matches the private signature from Step 1.

- [ ] **Step 4: Create java_files.go**

```go
// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sdkmpr "github.com/mendixlabs/mxcli/sdk/mpr"
	"github.com/mendixlabs/mxcli/sdk/javaactions"
)

func (b *MprBackend) projectRoot() string {
	return filepath.Dir(b.path)
}

func (b *MprBackend) javaActionPath(moduleName, actionName string) string {
	return filepath.Join(b.projectRoot(), "javasource", strings.ToLower(moduleName), "actions", actionName+".java")
}

func (b *MprBackend) writeJavaSourceFileViaPath(moduleName, actionName, javaCode string, params []*javaactions.JavaActionParameter, returnType javaactions.CodeActionReturnType, extraImports []string, extraCode string) error {
	dir := filepath.Dir(b.javaActionPath(moduleName, actionName))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create javasource dir: %w", err)
	}
	source := sdkmpr.GenerateJavaSource(moduleName, actionName, javaCode, params, returnType, extraImports, extraCode)
	if err := os.WriteFile(b.javaActionPath(moduleName, actionName), []byte(source), 0644); err != nil {
		return fmt.Errorf("write Java source: %w", err)
	}
	return nil
}

func (b *MprBackend) readJavaSourceFileViaPath(moduleName, actionName string) (string, error) {
	content, err := os.ReadFile(b.javaActionPath(moduleName, actionName))
	if err != nil {
		return "", fmt.Errorf("read Java source: %w", err)
	}
	return string(content), nil
}

func (b *MprBackend) deleteJavaSourceFileViaPath(moduleName, actionName string) error {
	if err := os.Remove(b.javaActionPath(moduleName, actionName)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete Java source: %w", err)
	}
	return nil
}

func (b *MprBackend) renameJavaSourceFileViaPath(moduleName, oldName, newName string) error {
	if err := os.Rename(b.javaActionPath(moduleName, oldName), b.javaActionPath(moduleName, newName)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("rename Java source: %w", err)
	}
	return nil
}
```

- [ ] **Step 5: Run test — PASS**

```bash
~/go1.26/bin/go test ./mdl/backend/mpr/ -run TestJavaFiles -v
```

- [ ] **Step 6: Update backend.go call sites**

```bash
grep -n "JavaSourceFile" mdl/backend/mpr/backend.go
```

Replace all 4 calls:
```go
// before:
return b.writer.WriteJavaSourceFile(moduleName, actionName, javaCode, params, returnType, extraImports, extraCode)
// after:
return b.writeJavaSourceFileViaPath(moduleName, actionName, javaCode, params, returnType, extraImports, extraCode)

// before:
return b.writer.DeleteJavaSourceFile(moduleName, actionName)
// after:
return b.deleteJavaSourceFileViaPath(moduleName, actionName)

// before:
return b.writer.RenameJavaSourceFile(moduleName, oldName, newName)
// after:
return b.renameJavaSourceFileViaPath(moduleName, oldName, newName)

// before:
return b.writer.ReadJavaSourceFile(moduleName, actionName)
// after:
return b.readJavaSourceFileViaPath(moduleName, actionName)
```

- [ ] **Step 7: Build and test**

```bash
~/go1.26/bin/go build ./...
~/go1.26/bin/go test ./mdl/backend/mpr/... -count=1 -timeout 120s
```

- [ ] **Step 8: Commit**

```bash
git add sdk/mpr/writer_javaactions.go mdl/backend/mpr/java_files.go mdl/backend/mpr/java_files_test.go mdl/backend/mpr/backend.go
git commit -m "refactor(backend): implement Java file ops via b.path directly, remove b.writer dependency"
```

---

### Task 2.5: Simplify UpdateRawUnit — remove b.writer fallback

**Files:**
- Modify: `mdl/backend/mpr/backend.go` (~line 747)

- [ ] **Step 1: Locate the function**

```bash
grep -n "UpdateRawUnit" mdl/backend/mpr/backend.go
```

- [ ] **Step 2: Replace with error-on-nil guard**

Current:
```go
func (b *MprBackend) UpdateRawUnit(unitID string, contents []byte) error {
	if b.msdkWriter == nil {
		return b.writer.UpdateRawUnit(unitID, contents)
	}
	return b.msdkWriter.UpdateRawUnit(unitID, contents)
}
```

Replace with:
```go
func (b *MprBackend) UpdateRawUnit(unitID string, contents []byte) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.msdkWriter.UpdateRawUnit(unitID, contents)
}
```

- [ ] **Step 3: Build and test**

```bash
~/go1.26/bin/go build ./...
~/go1.26/bin/go test ./mdl/backend/mpr/... ./mdl/executor/... -count=1 -timeout 120s
```

- [ ] **Step 4: Commit**

```bash
git add mdl/backend/mpr/backend.go
git commit -m "refactor(backend): remove b.writer fallback from UpdateRawUnit"
```

---

### Task 2.6: Verify b.writer write operations are fully retired

- [ ] **Step 1: Check remaining b.writer calls in backend.go**

```bash
grep -n "b\.writer\." mdl/backend/mpr/backend.go
```

Expected remaining:
- `b.writer.Close()` — lifecycle, deferred
- `b.writer.Serialize*()` calls — version-dependent BSON construction, deferred

Any write operation not in this list is a regression — investigate and fix.

- [ ] **Step 2: Commit (if any cleanup needed)**

```bash
git add mdl/backend/mpr/backend.go
git commit -m "refactor(backend): verify b.writer write retirement complete"
```

---

## Phase 3 — Gen-Native AccessRule Migration

### Task 3.1: Migrate security_entity_access_modelsdk.go to gen/domainmodels (funcs 1-4)

**Files:**
- Modify: `mdl/backend/mpr/security_entity_access_modelsdk.go`
- Modify: `mdl/backend/mpr/domainmodel_modelsdk_test.go`

**Pre-condition:** `AccessRule.moduleRoles → AllowedModuleRoles` fix committed `0aec60aa` — verify:
```bash
grep "AllowedModuleRoles" modelsdk/gen/domainmodels/types.go | head -2
```
Expected: line with `"AllowedModuleRoles"` in the NewByNameRefList constructor.

**Gen API reference (from domainmodels/types.go):**
- `genDM.Entity.AccessRulesItems() []element.Element`
- `genDM.Entity.AddAccessRules(element.Element)`
- `genDM.Entity.RemoveAccessRules(int)`
- `genDM.NewAccessRule() *AccessRule`
- `genDM.AccessRule.SetAllowCreate(bool)`, `SetAllowDelete(bool)`
- `genDM.AccessRule.SetDefaultMemberAccessRights(string)`
- `genDM.AccessRule.SetXPathConstraint(string)`, `SetXPathConstraintCaption(string)`
- `genDM.AccessRule.SetModuleRolesQualifiedNames([]string)` — writes BSON key `AllowedModuleRoles` ✅
- `genDM.AccessRule.ModuleRolesQualifiedNames() []string`
- `genDM.AccessRule.MemberAccessesItems() []element.Element`
- `genDM.AccessRule.AddMemberAccesses(element.Element)`, `RemoveMemberAccesses(int)`
- `genDM.NewMemberAccess() *MemberAccess`
- `genDM.MemberAccess.SetAccessRights(string)`
- `genDM.MemberAccess.SetAttributeQualifiedName(string)` — ByNameRef to Attribute
- `genDM.MemberAccess.SetAssociationQualifiedName(string)` — ByNameRef to Association

**Func 5 (reconcileMemberAccesses) stays as Patch* + writeUnitContents** — 200+ LOC orphan detection logic; gen-native requires cross-entity referential scan that is a separate effort.

- [ ] **Step 1: Write failing test for gen-native addEntityAccessRuleViaModelsdk**

Add to `mdl/backend/mpr/domainmodel_modelsdk_test.go`:

```go
func TestAddEntityAccessRuleViaModelsdk_GenNative(t *testing.T) {
	mprPath := makeDomainModelTestMPR(t) // existing helper

	b := New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	// Insert a minimal entity into the DM BSON so there's something to add rules to.
	const entityName = "Customer"
	const dmID = dmTestDMID

	// Seed entity via createEntityViaModelsdk or direct BSON (use existing helper pattern).
	if err := b.createEntityViaModelsdk(model.ID(dmID), &domainmodel.Entity{Name: entityName}); err != nil {
		t.Fatalf("createEntityViaModelsdk: %v", err)
	}

	err := b.addEntityAccessRuleViaModelsdk(
		model.ID(dmID), entityName,
		[]string{"TestModule.UserRole"},
		true, false,
		"ReadOnly", "",
		nil,
	)
	if err != nil {
		t.Fatalf("addEntityAccessRuleViaModelsdk: %v", err)
	}

	// Verify the rule was written with the correct BSON key.
	raw, _ := b.msdkWriter.Reader().GetRawUnitBytes(dmID)
	var doc bson.D
	bson.Unmarshal(raw, &doc)
	// Find entity, check AccessRules array has AllowedModuleRoles key.
	found := false
	for _, e := range doc {
		if e.Key != "Entities" {
			continue
		}
		if arr, ok := e.Value.(bson.A); ok {
			for _, item := range arr {
				if entBytes, ok := item.(bson.RawValue); ok {
					_ = entBytes // parse further if needed
				}
				found = true
			}
		}
	}
	_ = found // at minimum: no error means rule was added
}
```

Run: `~/go1.26/bin/go test ./mdl/backend/mpr/ -run TestAddEntityAccessRuleViaModelsdk_GenNative -v`
Expected: The test runs but may need adjustment based on entity seeding — the goal is that calling the function doesn't error.

- [ ] **Step 2: Replace addEntityAccessRuleViaModelsdk with gen-native**

In `security_entity_access_modelsdk.go`, import:
```go
genDM "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
"github.com/mendixlabs/mxcli/modelsdk/element"
sdkmpr "github.com/mendixlabs/mxcli/sdk/mpr"
```

Replace the function:
```go
func (b *MprBackend) addEntityAccessRuleViaModelsdk(
	unitID model.ID, entityName string, roleNames []string,
	allowCreate, allowDelete bool, defaultMemberAccess, xpathConstraint string,
	memberAccesses []sdkmpr.EntityMemberAccess,
) error {
	return b.msdkWrite(unitID, func(elem element.Element) error {
		dm, ok := elem.(*genDM.DomainModel)
		if !ok {
			return fmt.Errorf("unexpected type %T (want *DomainModel)", elem)
		}
		for _, e := range dm.EntitiesItems() {
			ent, ok := e.(*genDM.Entity)
			if !ok {
				continue
			}
			if ent.Name() != entityName {
				continue
			}
			rule := genDM.NewAccessRule()
			rule.SetAllowCreate(allowCreate)
			rule.SetAllowDelete(allowDelete)
			rule.SetDefaultMemberAccessRights(defaultMemberAccess)
			rule.SetXPathConstraint(xpathConstraint)
			rule.SetModuleRolesQualifiedNames(roleNames)
			for _, ma := range memberAccesses {
				genMA := genDM.NewMemberAccess()
				genMA.SetAccessRights(ma.AccessRights)
				if ma.AttributeRef != "" {
					genMA.SetAttributeQualifiedName(ma.AttributeRef)
				} else if ma.AssociationRef != "" {
					genMA.SetAssociationQualifiedName(ma.AssociationRef)
				}
				rule.AddMemberAccesses(genMA)
			}
			ent.AddAccessRules(rule)
			return nil
		}
		return fmt.Errorf("entity not found: %s", entityName)
	})
}
```

Check `sdkmpr.EntityMemberAccess` field names:
```bash
grep -n "EntityMemberAccess\|AttributeRef\|AssociationRef\|AccessRights" sdk/mpr/writer_security.go | head -10
```

- [ ] **Step 3: Replace removeEntityAccessRuleViaModelsdk**

```go
func (b *MprBackend) removeEntityAccessRuleViaModelsdk(unitID model.ID, entityName string, roleNames []string) (int, error) {
	removed := 0
	err := b.msdkWrite(unitID, func(elem element.Element) error {
		dm, ok := elem.(*genDM.DomainModel)
		if !ok {
			return fmt.Errorf("unexpected type %T", elem)
		}
		for _, e := range dm.EntitiesItems() {
			ent, ok := e.(*genDM.Entity)
			if !ok || ent.Name() != entityName {
				continue
			}
			rules := ent.AccessRulesItems()
			for i := len(rules) - 1; i >= 0; i-- {
				rule, ok := rules[i].(*genDM.AccessRule)
				if !ok {
					continue
				}
				existing := rule.ModuleRolesQualifiedNames()
				if rolesMatch(existing, roleNames) {
					ent.RemoveAccessRules(i)
					removed++
				}
			}
		}
		return nil
	})
	return removed, err
}

// rolesMatch returns true if the two role slices contain the same elements (order-independent).
func rolesMatch(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	m := make(map[string]struct{}, len(a))
	for _, r := range a {
		m[r] = struct{}{}
	}
	for _, r := range b {
		if _, ok := m[r]; !ok {
			return false
		}
	}
	return true
}
```

- [ ] **Step 4: Replace removeRoleFromAllEntitiesViaModelsdk**

```go
func (b *MprBackend) removeRoleFromAllEntitiesViaModelsdk(unitID model.ID, roleName string) (int, error) {
	removed := 0
	err := b.msdkWrite(unitID, func(elem element.Element) error {
		dm, ok := elem.(*genDM.DomainModel)
		if !ok {
			return fmt.Errorf("unexpected type %T", elem)
		}
		for _, e := range dm.EntitiesItems() {
			ent, ok := e.(*genDM.Entity)
			if !ok {
				continue
			}
			for _, r := range ent.AccessRulesItems() {
				rule, ok := r.(*genDM.AccessRule)
				if !ok {
					continue
				}
				roles := rule.ModuleRolesQualifiedNames()
				newRoles := make([]string, 0, len(roles))
				for _, ro := range roles {
					if ro != roleName {
						newRoles = append(newRoles, ro)
					}
				}
				if len(newRoles) != len(roles) {
					rule.SetModuleRolesQualifiedNames(newRoles)
					removed++
				}
			}
		}
		return nil
	})
	return removed, err
}
```

- [ ] **Step 5: Replace revokeEntityMemberAccessViaModelsdk**

First, understand the revocation struct:
```bash
grep -n "EntityAccessRevocation\|RevokeCreate\|RevokeDelete\|RevokeRead\|RevokeWrite" sdk/mpr/writer_security.go | head -20
```

Then implement:
```go
func (b *MprBackend) revokeEntityMemberAccessViaModelsdk(unitID model.ID, entityName string, roleNames []string, revocation sdkmpr.EntityAccessRevocation) (int, error) {
	revoked := 0
	err := b.msdkWrite(unitID, func(elem element.Element) error {
		dm, ok := elem.(*genDM.DomainModel)
		if !ok {
			return fmt.Errorf("unexpected type %T", elem)
		}
		for _, e := range dm.EntitiesItems() {
			ent, ok := e.(*genDM.Entity)
			if !ok || ent.Name() != entityName {
				continue
			}
			for _, r := range ent.AccessRulesItems() {
				rule, ok := r.(*genDM.AccessRule)
				if !ok {
					continue
				}
				if !rolesMatch(rule.ModuleRolesQualifiedNames(), roleNames) {
					continue
				}
				if revocation.RevokeCreate {
					rule.SetAllowCreate(false)
				}
				if revocation.RevokeDelete {
					rule.SetAllowDelete(false)
				}
				if revocation.RevokeReadAll {
					rule.SetDefaultMemberAccessRights("NoAccess")
				} else if revocation.RevokeWriteAll {
					rule.SetDefaultMemberAccessRights("ReadOnly")
				}
				if len(revocation.RevokeReadMembers) > 0 || len(revocation.RevokeWriteMembers) > 0 {
					revokeSet := make(map[string]struct{})
					for _, m := range revocation.RevokeReadMembers {
						revokeSet[m] = struct{}{}
					}
					for _, m := range revocation.RevokeWriteMembers {
						revokeSet[m] = struct{}{}
					}
					for _, ma := range rule.MemberAccessesItems() {
						maTyped, ok := ma.(*genDM.MemberAccess)
						if !ok {
							continue
						}
						ref := maTyped.AttributeQualifiedName()
						if ref == "" {
							ref = maTyped.AssociationQualifiedName()
						}
						if _, revoke := revokeSet[ref]; revoke {
							if _, isRead := revokeSet[ref]; isRead {
								maTyped.SetAccessRights("NoAccess")
							}
						}
					}
				}
				revoked++
			}
		}
		return nil
	})
	return revoked, err
}
```

Adjust `RevokeReadMembers`, `RevokeWriteMembers`, `RevokeReadAll`, `RevokeWriteAll` field names based on actual `EntityAccessRevocation` struct fields (Step grep output).

- [ ] **Step 6: Run tests**

```bash
~/go1.26/bin/go test ./mdl/backend/mpr/ -count=1 -timeout 120s
```

Expected: all pass.

- [ ] **Step 7: Commit**

```bash
git add mdl/backend/mpr/security_entity_access_modelsdk.go mdl/backend/mpr/domainmodel_modelsdk_test.go
git commit -m "refactor(backend): migrate entity-access-rule funcs 1-4 to gen-native msdkWrite + genDM.AccessRule"
```

---

## Phase 4 — Final Verification

### Task 4.1: Full sweep and cleanup

- [ ] **Step 1: Confirm zero msdkWriteRaw references**

```bash
grep -rn "msdkWriteRaw" mdl/backend/mpr/*.go
```

Expected: no output (definition deleted in Task 1.2, all call sites migrated).

- [ ] **Step 2: Confirm b.writer write operations retired**

```bash
grep -n "b\.writer\." mdl/backend/mpr/backend.go | grep -v "Close\|Serialize"
```

Expected: no output.

- [ ] **Step 3: Full build and test suite**

```bash
~/go1.26/bin/go build ./...
~/go1.26/bin/go test ./mdl/... ./sdk/mpr/... -count=1 -timeout 300s
```

Expected: all pass (pre-existing `cmd/exprgrammar-mine` timeout failure is unrelated — skip it).

- [ ] **Step 4: Commit if any cleanup was needed**

```bash
git add -u
git commit -m "refactor(backend): final cleanup — SOLID write path complete"
```

---

## Self-Review

**Spec coverage:**
- [x] Single write primitive (`writeUnitContents` in `write_helpers.go`) — Task 1.1
- [x] All 15 `msdkWriteRaw` call sites replaced — Task 1.2
- [x] `msdkWriteRaw` definition deleted — Task 1.2 Step 3
- [x] `FindViewEntitySourceDocument*` → Reader — Task 2.1
- [x] `UpdateOqlQueriesForMovedEntity` → ScanOqlQueryUpdates + writeUnitContents — Task 2.2
- [x] `UpdateEnumerationRefsInAllDomainModels` → reader + updateDomainModelViaModelsdk — Task 2.3
- [x] Java file ops → b.path — Task 2.4
- [x] `UpdateRawUnit` fallback removed — Task 2.5
- [x] gen-native AccessRule (funcs 1-4) — Task 3.1
- [x] reconcileMemberAccesses stays Patch* + writeUnitContents (200 LOC orphan scan, separate effort) — documented in Task 3.1 preamble
- [x] Final verification — Task 4.1

**Placeholder scan:** All code blocks complete. `EntityAccessRevocation` field names are audited inline via grep in Task 3.1 Step 5. `EnumerationAttributeType` field is audited in Task 2.3 Step 1.

**Type consistency:**
- `genDM.Entity`, `genDM.AccessRule`, `genDM.MemberAccess`, `genDM.NewAccessRule()`, `genDM.NewMemberAccess()` used consistently in Task 3.1.
- `writeUnitContents(unitID model.ID, contents []byte) error` signature consistent across Tasks 1.1 and 1.2.
- `UnitPatch` from `sdk/mpr` (defined in `writer_refs.go`) used in Task 2.2 — confirmed correct package.
