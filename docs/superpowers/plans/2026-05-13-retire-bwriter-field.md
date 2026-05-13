# Retire b.writer Write Dependency Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eliminate all write and read operations in MprBackend that depend on `b.writer` (`*sdk/mpr.Writer`), leaving `b.writer` responsible only for connection lifecycle and `Serialize*` BSON construction (which requires `ProjectVersion()` state and is deferred to a follow-up plan).

**Architecture:** Five independent refactors: (1) move two read-only `FindViewEntitySourceDocument*` methods from Writer to Reader; (2) export `ScanOqlQueryUpdates` and write OQL updates via WriteTransaction (not msdkWriteRaw); (3) use `writeDomainModel` pattern for enumeration ref updates across all domain models; (4) implement Java source file I/O in mprbackend directly via `b.path`; (5) simplify `UpdateRawUnit` to always use `msdkWriter`.

**Tech Stack:** Go 1.26, `modernc.org/sqlite`, `go.mongodb.org/mongo-driver/bson`, existing helpers `msdkWriteRaw`/`writeDomainModel`/`updateDomainModelViaModelsdk` in `mdl/backend/mpr/`.

---

## Remaining `b.writer.*` calls after this plan

Before: 10 calls. After this plan: only `b.writer.Close()` + `b.writer.Serialize*()`.

| Line | Call | Task |
|------|------|------|
| 94 | `b.writer.Close()` | Deferred — requires connection lifecycle redesign |
| 229 | `b.writer.FindViewEntitySourceDocumentID()` | Task 1 |
| 232 | `b.writer.FindAllViewEntitySourceDocumentIDs()` | Task 1 |
| 238 | `b.writer.UpdateOqlQueriesForMovedEntity()` | Task 2 |
| 241 | `b.writer.UpdateEnumerationRefsInAllDomainModels()` | Task 3 |
| 636–642 | `b.writer.Write/Delete/RenameJavaSourceFile()` | Task 4 |
| 649 | `b.writer.ReadJavaSourceFile()` | Task 4 |
| 747 | `b.writer.UpdateRawUnit()` (fallback) | Task 5 |

---

## File Map

| File | Action | Responsibility |
|------|--------|---------------|
| `sdk/mpr/reader_documents.go` | Modify | Add `FindViewEntitySourceDocumentID` and `FindAllViewEntitySourceDocumentIDs` public methods |
| `sdk/mpr/writer_domainmodel.go` | Modify | Add `ScanOqlQueryUpdates(old, new) ([]UnitPatch, int, error)` export |
| `mdl/backend/mpr/domainmodel_modelsdk.go` | Modify | Add `updateOqlQueriesForMovedEntityViaModelsdk` and `updateEnumerationRefsInAllDomainModelsViaModelsdk` |
| `mdl/backend/mpr/java_files.go` | Create | Java source file I/O using `b.path` directly |
| `mdl/backend/mpr/backend.go` | Modify | Update 8 call sites |

---

## Task 1: Move FindViewEntitySourceDocument* to Reader

These methods are read-only but live on `Writer` because they use `w.reader.listUnitsByType`. Move them to the Reader directly.

**Files:**
- Modify: `sdk/mpr/reader_documents.go`
- Modify: `mdl/backend/mpr/backend.go:229-232`

- [ ] **Step 1: Write failing test** (`sdk/mpr/reader_documents_test.go` or existing test file)

```go
// In an appropriate test file in sdk/mpr package
func TestReader_FindViewEntitySourceDocumentID_NotFound(t *testing.T) {
    r := openTestReader(t)  // use existing test helper
    id, err := r.FindViewEntitySourceDocumentID("NonExistentModule", "NonExistentDoc")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if id != "" {
        t.Fatalf("expected empty ID, got %s", id)
    }
}
```

Run: `~/go1.26/bin/go test ./sdk/mpr/... -run TestReader_FindViewEntitySourceDocument -v`
Expected: FAIL with `r.FindViewEntitySourceDocumentID undefined`

- [ ] **Step 2: Add methods to reader_documents.go**

Find the end of `sdk/mpr/reader_documents.go` and append:

```go
// FindViewEntitySourceDocumentID finds a ViewEntitySourceDocument by module and document name.
// Returns the document ID if found, empty string if not found.
func (r *Reader) FindViewEntitySourceDocumentID(moduleName, docName string) (model.ID, error) {
    units, err := r.listUnitsByType("DomainModels$ViewEntitySourceDocument")
    if err != nil {
        return "", err
    }
    modules, err := r.ListModules()
    if err != nil {
        return "", err
    }
    moduleNames := make(map[string]string)
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

// FindAllViewEntitySourceDocumentIDs finds ALL ViewEntitySourceDocuments matching the
// given module and document name.
func (r *Reader) FindAllViewEntitySourceDocumentIDs(moduleName, docName string) ([]model.ID, error) {
    units, err := r.listUnitsByType("DomainModels$ViewEntitySourceDocument")
    if err != nil {
        return nil, err
    }
    modules, err := r.ListModules()
    if err != nil {
        return nil, err
    }
    moduleNames := make(map[string]string)
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

Add `"go.mongodb.org/mongo-driver/bson"` import if not already present.

- [ ] **Step 3: Run test to verify it passes**

Run: `~/go1.26/bin/go test ./sdk/mpr/... -run TestReader_FindViewEntitySourceDocument -v`
Expected: PASS

- [ ] **Step 4: Update backend.go call sites**

In `mdl/backend/mpr/backend.go`:
```
Line 229: return b.writer.FindViewEntitySourceDocumentID(moduleName, docName)
      →   return b.reader.FindViewEntitySourceDocumentID(moduleName, docName)

Line 232: return b.writer.FindAllViewEntitySourceDocumentIDs(moduleName, docName)
      →   return b.reader.FindAllViewEntitySourceDocumentIDs(moduleName, docName)
```

- [ ] **Step 5: Build and test**

```bash
~/go1.26/bin/go build ./...
~/go1.26/bin/go test ./sdk/mpr/... ./mdl/backend/mpr/... -count=1 -timeout 120s
```
Expected: all pass

- [ ] **Step 6: Commit**

```bash
git add sdk/mpr/reader_documents.go mdl/backend/mpr/backend.go
git commit -m "refactor(backend): move FindViewEntitySourceDocument* from Writer to Reader"
```

---

## Task 2: Migrate UpdateOqlQueriesForMovedEntity

Current: `b.writer.UpdateOqlQueriesForMovedEntity(old, new)` calls `w.reader.listUnitsByType` + `w.updateUnit` per hit.

Pattern: export `ScanOqlQueryUpdates(old, new) ([]UnitPatch, int, error)` (UnitPatch already defined in `sdk/mpr/writer_refs.go`), then mprbackend loops and writes each patched unit via **WriteTransaction** (not `msdkWriteRaw` — untyped raw write goes against the migration direction).

**Files:**
- Modify: `sdk/mpr/writer_domainmodel.go`
- Modify: `mdl/backend/mpr/domainmodel_modelsdk.go`
- Modify: `mdl/backend/mpr/backend.go:238`

- [ ] **Step 1: Write failing test**

In `sdk/mpr/` (find existing test file or create one):
```go
func TestWriter_ScanOqlQueryUpdates_NoHits(t *testing.T) {
    w := openTestWriter(t)  // existing test helper
    patches, count, err := w.ScanOqlQueryUpdates("OldModule.OldEntity", "NewModule.NewEntity")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if count != 0 || len(patches) != 0 {
        t.Fatalf("expected 0 hits in empty project, got %d", count)
    }
}
```

Run: `~/go1.26/bin/go test ./sdk/mpr/... -run TestWriter_ScanOqlQueryUpdates -v`
Expected: FAIL with `w.ScanOqlQueryUpdates undefined`

- [ ] **Step 2: Add ScanOqlQueryUpdates to writer_domainmodel.go**

Append to `sdk/mpr/writer_domainmodel.go`:

```go
// ScanOqlQueryUpdates scans all ViewEntitySourceDocument units and replaces
// oldQualifiedName with newQualifiedName in their Oql fields.
// Returns the patched units and the count of modified documents (not written to disk).
func (w *Writer) ScanOqlQueryUpdates(oldQualifiedName, newQualifiedName string) ([]UnitPatch, int, error) {
    units, err := w.reader.listUnitsByType("DomainModels$ViewEntitySourceDocument")
    if err != nil {
        return nil, 0, err
    }
    var patches []UnitPatch
    updated := 0
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
        updated++
    }
    return patches, updated, nil
}
```

Add `"strings"` and `"fmt"` to imports if missing. `UnitPatch` is already defined in `writer_refs.go` (same package).

- [ ] **Step 3: Run test to verify it passes**

```bash
~/go1.26/bin/go test ./sdk/mpr/... -run TestWriter_ScanOqlQueryUpdates -v
```
Expected: PASS

- [ ] **Step 4: Add updateOqlQueriesForMovedEntityViaModelsdk to domainmodel_modelsdk.go**

Append to `mdl/backend/mpr/domainmodel_modelsdk.go`:

```go
// ── UpdateOqlQueriesForMovedEntity ────────────────────

func (b *MprBackend) updateOqlQueriesForMovedEntityViaModelsdk(oldQualifiedName, newQualifiedName string) (int, error) {
    if b.msdkWriter == nil {
        return 0, fmt.Errorf("modelsdk writer not initialized")
    }
    patches, count, err := b.writer.ScanOqlQueryUpdates(oldQualifiedName, newQualifiedName)
    if err != nil {
        return 0, err
    }
    for _, p := range patches {
        // Use WriteTransaction, not msdkWriteRaw — maintains correct 1544-safe write path.
        wtx, err := b.msdkWriter.BeginWriteTransaction()
        if err != nil {
            return 0, err
        }
        if err := wtx.WriteUnit(p.ID, p.Contents); err != nil {
            return 0, fmt.Errorf("write OQL unit %s: %w", p.ID, err)
        }
        if err := wtx.Commit(); err != nil {
            return 0, fmt.Errorf("commit OQL unit %s: %w", p.ID, err)
        }
        b.msdkWriter.InvalidateCache(p.ID)
    }
    return count, nil
}
```

- [ ] **Step 5: Update backend.go line 238**

```go
// Before:
return b.writer.UpdateOqlQueriesForMovedEntity(oldQualifiedName, newQualifiedName)
// After:
return b.updateOqlQueriesForMovedEntityViaModelsdk(oldQualifiedName, newQualifiedName)
```

Check that the function signature matches (returns `(int, error)`):
```bash
grep -n "UpdateOqlQueriesForMovedEntity\|UpdateOqlQueries" mdl/backend/mpr/backend.go | head -5
```

- [ ] **Step 6: Build and test**

```bash
~/go1.26/bin/go build ./...
~/go1.26/bin/go test ./sdk/mpr/... ./mdl/backend/mpr/... -count=1 -timeout 120s
```

- [ ] **Step 7: Commit**

```bash
git add sdk/mpr/writer_domainmodel.go mdl/backend/mpr/domainmodel_modelsdk.go mdl/backend/mpr/backend.go
git commit -m "refactor(backend): migrate UpdateOqlQueriesForMovedEntity to WriteTransaction"
```

---

## Task 3: Migrate UpdateEnumerationRefsInAllDomainModels

Current: iterates all domain models, calls `w.updateDomainModel(dm)` for changed ones.
Migration: `b.reader.ListDomainModels()` + in-memory mutation + `b.updateDomainModelViaModelsdk(dm)`.

**Files:**
- Modify: `mdl/backend/mpr/domainmodel_modelsdk.go`
- Modify: `mdl/backend/mpr/backend.go:241`

- [ ] **Step 1: Write failing test**

```go
// In mdl/backend/mpr/ test file
func TestUpdateEnumerationRefsInAllDomainModelsViaModelsdk_NoOp(t *testing.T) {
    b := newMinimalMprBackend(t)  // create test backend with minimal MPR
    err := b.updateEnumerationRefsInAllDomainModelsViaModelsdk("OldModule.OldEnum", "NewModule.NewEnum")
    if err != nil {
        t.Fatalf("unexpected error on no-op: %v", err)
    }
}
```

Run: `~/go1.26/bin/go test ./mdl/backend/mpr/... -run TestUpdateEnumeration -v`
Expected: FAIL with `b.updateEnumerationRefsInAllDomainModelsViaModelsdk undefined`

- [ ] **Step 2: Add method to domainmodel_modelsdk.go**

Append to `mdl/backend/mpr/domainmodel_modelsdk.go`:

```go
// ── UpdateEnumerationRefsInAllDomainModels ────────────

func (b *MprBackend) updateEnumerationRefsInAllDomainModelsViaModelsdk(oldQualifiedName, newQualifiedName string) error {
    if b.msdkWriter == nil {
        return fmt.Errorf("modelsdk writer not initialized")
    }
    dms, err := b.reader.ListDomainModels()
    if err != nil {
        return fmt.Errorf("list domain models: %w", err)
    }
    for _, dm := range dms {
        changed := false
        for _, entity := range dm.Entities {
            for _, attr := range entity.Attributes {
                if enumType, ok := attr.Type.(*domainmodel.EnumerationAttributeType); ok {
                    if enumType.EnumerationRef == oldQualifiedName {
                        enumType.EnumerationRef = newQualifiedName
                        enumType.EnumerationID = model.ID(newQualifiedName)
                        changed = true
                    }
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

Note: `domainmodel.EnumerationAttributeType` — confirm type name exists:
```bash
grep -rn "EnumerationAttributeType" sdk/domainmodel/ | head -5
```

- [ ] **Step 3: Run test to verify it passes**

```bash
~/go1.26/bin/go test ./mdl/backend/mpr/... -run TestUpdateEnumeration -v
```
Expected: PASS

- [ ] **Step 4: Update backend.go line 241**

```go
// Before:
return b.writer.UpdateEnumerationRefsInAllDomainModels(oldQualifiedName, newQualifiedName)
// After:
return b.updateEnumerationRefsInAllDomainModelsViaModelsdk(oldQualifiedName, newQualifiedName)
```

- [ ] **Step 5: Build and test**

```bash
~/go1.26/bin/go build ./...
~/go1.26/bin/go test ./sdk/mpr/... ./mdl/backend/mpr/... ./mdl/executor/... -count=1 -timeout 120s
```

- [ ] **Step 6: Commit**

```bash
git add mdl/backend/mpr/domainmodel_modelsdk.go mdl/backend/mpr/backend.go
git commit -m "refactor(backend): migrate UpdateEnumerationRefsInAllDomainModels to writeDomainModel pattern"
```

---

## Task 4: Implement Java File Operations in mprbackend

Current: `b.writer.WriteJavaSourceFile(...)`, `b.writer.DeleteJavaSourceFile(...)`, etc. use `filepath.Dir(w.reader.path)`. The MPR file path is available in `b.path`.

Export `GenerateJavaSource` from sdk/mpr (the internal `generateJavaSource` function), then implement file operations in mprbackend directly.

**Files:**
- Modify: `sdk/mpr/writer_javaactions.go` (export `GenerateJavaSource`)
- Create: `mdl/backend/mpr/java_files.go`
- Modify: `mdl/backend/mpr/backend.go:636-649`

- [ ] **Step 1: Confirm generateJavaSource signature**

```bash
grep -n "^func generateJavaSource" sdk/mpr/writer_javaactions.go
```

- [ ] **Step 2: Export GenerateJavaSource in sdk/mpr**

In `sdk/mpr/writer_javaactions.go` (or `serialize_exports.go`), append:

```go
// GenerateJavaSource generates the Java source code for a Java action.
// Exported for use by callers that implement their own file I/O.
func GenerateJavaSource(moduleName, actionName, javaCode string, params []*javaactions.JavaActionParameter, returnType javaactions.CodeActionReturnType, extraImports []string, extraCode string) string {
    return generateJavaSource(moduleName, actionName, javaCode, params, returnType, extraImports, extraCode)
}
```

Confirm the private function signature matches — adjust if needed.

- [ ] **Step 3: Write failing test**

```go
// In mdl/backend/mpr/ test file
func TestJavaFiles_WriteDeleteRename(t *testing.T) {
    dir := t.TempDir()
    mprPath := filepath.Join(dir, "test.mpr")
    b := &MprBackend{path: mprPath}
    // Write a Java source file
    err := b.writeJavaSourceFileViaPath("MyModule", "MyAction", "return null;", nil, "", nil, "")
    if err != nil {
        t.Fatalf("write: %v", err)
    }
    // Verify file exists
    expected := filepath.Join(dir, "javasource", "mymodule", "actions", "MyAction.java")
    if _, err := os.Stat(expected); err != nil {
        t.Fatalf("file not found: %v", err)
    }
    // Rename
    err = b.renameJavaSourceFileViaPath("MyModule", "MyAction", "RenamedAction")
    if err != nil {
        t.Fatalf("rename: %v", err)
    }
    // Delete
    err = b.deleteJavaSourceFileViaPath("MyModule", "RenamedAction")
    if err != nil {
        t.Fatalf("delete: %v", err)
    }
}
```

Run: `~/go1.26/bin/go test ./mdl/backend/mpr/... -run TestJavaFiles -v`
Expected: FAIL with method undefined

- [ ] **Step 4: Create mdl/backend/mpr/java_files.go**

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

func (b *MprBackend) writeJavaSourceFileViaPath(moduleName, actionName, javaCode string, params []*javaactions.JavaActionParameter, returnType javaactions.CodeActionReturnType, extraImports []string, extraCode string) error {
    javaDir := filepath.Join(b.projectRoot(), "javasource", strings.ToLower(moduleName), "actions")
    if err := os.MkdirAll(javaDir, 0755); err != nil {
        return fmt.Errorf("create javasource dir: %w", err)
    }
    source := sdkmpr.GenerateJavaSource(moduleName, actionName, javaCode, params, returnType, extraImports, extraCode)
    filePath := filepath.Join(javaDir, actionName+".java")
    if err := os.WriteFile(filePath, []byte(source), 0644); err != nil {
        return fmt.Errorf("write Java source: %w", err)
    }
    return nil
}

func (b *MprBackend) deleteJavaSourceFileViaPath(moduleName, actionName string) error {
    filePath := filepath.Join(b.projectRoot(), "javasource", strings.ToLower(moduleName), "actions", actionName+".java")
    if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
        return fmt.Errorf("delete Java source: %w", err)
    }
    return nil
}

func (b *MprBackend) renameJavaSourceFileViaPath(moduleName, oldName, newName string) error {
    dir := filepath.Join(b.projectRoot(), "javasource", strings.ToLower(moduleName), "actions")
    if err := os.Rename(filepath.Join(dir, oldName+".java"), filepath.Join(dir, newName+".java")); err != nil && !os.IsNotExist(err) {
        return fmt.Errorf("rename Java source: %w", err)
    }
    return nil
}

func (b *MprBackend) readJavaSourceFileViaPath(moduleName, actionName string) (string, error) {
    filePath := filepath.Join(b.projectRoot(), "javasource", strings.ToLower(moduleName), "actions", actionName+".java")
    content, err := os.ReadFile(filePath)
    if err != nil {
        return "", fmt.Errorf("read Java source: %w", err)
    }
    return string(content), nil
}
```

- [ ] **Step 5: Run test to verify it passes**

```bash
~/go1.26/bin/go test ./mdl/backend/mpr/... -run TestJavaFiles -v
```
Expected: PASS

- [ ] **Step 6: Update backend.go lines 636-649**

```go
// Line 636 — Before:
return b.writer.WriteJavaSourceFile(moduleName, actionName, javaCode, params, returnType, extraImports, extraCode)
// After:
return b.writeJavaSourceFileViaPath(moduleName, actionName, javaCode, params, returnType, extraImports, extraCode)

// Line 639:
return b.writer.DeleteJavaSourceFile(moduleName, actionName)
→ return b.deleteJavaSourceFileViaPath(moduleName, actionName)

// Line 642:
return b.writer.RenameJavaSourceFile(moduleName, oldName, newName)
→ return b.renameJavaSourceFileViaPath(moduleName, oldName, newName)

// Line 649:
return b.writer.ReadJavaSourceFile(moduleName, actionName)
→ return b.readJavaSourceFileViaPath(moduleName, actionName)
```

Confirm exact lines with:
```bash
grep -n "JavaSourceFile" mdl/backend/mpr/backend.go
```

- [ ] **Step 7: Build and test**

```bash
~/go1.26/bin/go build ./...
~/go1.26/bin/go test ./mdl/backend/mpr/... ./mdl/executor/... -count=1 -timeout 120s
```

- [ ] **Step 8: Commit**

```bash
git add sdk/mpr/writer_javaactions.go mdl/backend/mpr/java_files.go mdl/backend/mpr/backend.go
git commit -m "refactor(backend): implement Java file ops in mprbackend using b.path directly"
```

---

## Task 5: Simplify UpdateRawUnit (Remove b.writer Fallback)

Current (backend.go ~747):
```go
func (b *MprBackend) UpdateRawUnit(unitID string, contents []byte) error {
    if b.msdkWriter == nil {
        return b.writer.UpdateRawUnit(unitID, contents)
    }
    return b.msdkWriter.UpdateRawUnit(unitID, contents)
}
```

Since `msdkWriter` is always initialized in `Connect()` (returns error if it fails) and `Wrap()` (logs and leaves nil, which is a degraded mode), the fallback complicates the invariant. Simplify to require `msdkWriter`.

**Files:**
- Modify: `mdl/backend/mpr/backend.go` (~line 747)

- [ ] **Step 1: Locate the function**

```bash
grep -n "UpdateRawUnit" mdl/backend/mpr/backend.go
```

- [ ] **Step 2: Simplify to single path**

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
Expected: all pass

- [ ] **Step 4: Commit**

```bash
git add mdl/backend/mpr/backend.go
git commit -m "refactor(backend): remove b.writer fallback from UpdateRawUnit"
```

---

## Verification

After all 5 tasks:

```bash
# All write calls confirmed gone
grep -n "b\.writer\." mdl/backend/mpr/backend.go | grep -v "Close\|Serialize\|FindView\|FindAll\|UpdateOql\|UpdateEnum\|Java"
# Expected: 0 lines

# Full build + test
~/go1.26/bin/go build ./...
~/go1.26/bin/go test ./sdk/mpr/... ./mdl/backend/mpr/... ./mdl/executor/... -count=1 -timeout 120s
```

**After this plan, `b.writer` is only used for:**
1. `b.writer.Close()` — connection lifecycle; deferred pending Reader ownership redesign
2. `b.writer.Serialize*()` — version-dependent BSON construction; deferred pending "pass majorVersion as parameter" refactor

Both are explicit, documented deferral points — not forgotten work.
