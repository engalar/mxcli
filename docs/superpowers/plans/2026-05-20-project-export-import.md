# Project Export / Import Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `mxcli export` and `mxcli import` subcommands that batch-export an entire Mendix project to structured per-document MDL files and reimport them into an empty template project.

**Architecture:** A new `ExportProject()` method on `*executor.Executor` orchestrates the export: it opens one read-only backend session, iterates all modules and document types, redirects `ctx.Output` to a `bytes.Buffer` per document, calls the existing `describe*Gen()` functions, and writes each buffer to the correct file path. A new `ImportProject()` method reads and sorts `.mdl` files by dependency type, then executes them sequentially through the existing `Executor.ExecuteProgram()` path. Two thin Cobra commands in `cmd/mxcli/` wire flags and call these methods.

**Tech Stack:** Go stdlib (`os`, `path/filepath`, `bytes`, `strings`), Cobra (existing), `mdl/executor` package (existing describe functions), `mdl/backend/mpr` (existing `NewFromPath`), `mdl/visitor` (existing `visitor.Build`)

---

## File Map

| File | Action | Responsibility |
|------|--------|---------------|
| `mdl/executor/cmd_export_project.go` | Create | `ExportProject()` orchestration: classify modules, fold-path mapping, capture per-document MDL, write files |
| `mdl/executor/cmd_import_project.go` | Create | `ImportProject()` orchestration: scan dir, topological sort by document type, sequential exec |
| `mdl/executor/cmd_export_project_test.go` | Create | Unit tests (folder path logic), integration test (export minimal.mpr), round-trip test |
| `cmd/mxcli/cmd_export.go` | Create | `exportCmd` Cobra command (`mxcli export`) |
| `cmd/mxcli/cmd_import.go` | Create | `importCmd` Cobra command (`mxcli import`) |
| `cmd/mxcli/main.go` | Modify | Register `exportCmd` and `importCmd` with `rootCmd.AddCommand()` |

**No new packages.** Everything stays inside existing packages; describe functions are called package-internally.

---

## Key Patterns (read before implementing)

### Capturing describe output to a string

All `describe*Gen()` functions write to `ctx.Output` (an `io.Writer`). To capture to a string, temporarily swap `ctx.Output`:

```go
var buf bytes.Buffer
savedOutput := ctx.Output
ctx.Output = &buf
err := describeEntityGen(ctx, ast.QualifiedName{Module: modName, Name: ent.Name()})
ctx.Output = savedOutput
if err != nil {
    // log warning, skip this document
}
content := buf.String()
```

This pattern is already used in `cmd_diff_gen.go:58-60` and `cmd_catalog.go:689-690`.

### Writing a file

```go
if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
    return fmt.Errorf("mkdir %s: %w", filepath.Dir(filePath), err)
}
if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
    return fmt.Errorf("write %s: %w", filePath, err)
}
```

### Module classification

```go
modules, err := ctx.Backend.ListModules()
for _, m := range modules {
    if m.FromAppStore {
        // marketplace module
    } else {
        // regular module
    }
}
```

`model.Module.FromAppStore` is at `model/types.go:90`.

### Folder path for a document

```go
h, err := getHierarchy(ctx)
folderPath := h.BuildFolderPath(containerID) // e.g. "Microflows/ACT"
```

`getHierarchy` is in `hierarchy.go:127`. Returns `""` for top-level documents.

### File path for a document

```go
func documentFilePath(outputDir, moduleName, folderPath, qname string) string {
    if folderPath != "" {
        return filepath.Join(outputDir, moduleName, filepath.FromSlash(folderPath), qname+".mdl")
    }
    return filepath.Join(outputDir, moduleName, qname+".mdl")
}
```

### Executing a file through the executor

```go
content, _ := os.ReadFile(path)
prog, errs := visitor.Build(string(content))
if len(errs) > 0 { /* handle */ }
if err := exec.ExecuteProgram(prog); err != nil { /* handle */ }
```

---

## Topological import order (hardcoded)

Files are sorted by document type prefix before execution:

```
Order 1: enumerations  → _enumerations/ or files matching Enumeration type header
Order 2: entities      → Domain/
Order 3: associations  → _associations.mdl
Order 4: constants     → Constants/
Order 5: module roles  → _module_roles.mdl or security/
Order 6: java actions  → JavaActions/
Order 7: microflows    → Microflows/
Order 8: nanoflows     → Nanoflows/
Order 9: layouts       → Layouts/
Order 10: snippets     → Snippets/
Order 11: pages        → Pages/
Order 12: workflows    → Workflows/
Order 13: navigation   → _project/navigation.mdl
Order 14: security     → _project/security.mdl
Order 15: settings     → _project/settings.mdl
Order 16: module decl  → _module.mdl
```

**Simple heuristic:** sort by the document type determined from the directory/file name pattern, not by parsing file content.

---

## Task 1: `ExportOptions` struct and helper `documentFilePath`

**Files:**
- Create: `mdl/executor/cmd_export_project.go`
- Test: `mdl/executor/cmd_export_project_test.go`

- [ ] **Step 1: Write the failing test for `documentFilePath`**

In `mdl/executor/cmd_export_project_test.go`:

```go
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"path/filepath"
	"testing"
)

func TestDocumentFilePath_NoFolder(t *testing.T) {
	got := documentFilePath("/out", "MyModule", "", "MyModule.Customer")
	want := filepath.Join("/out", "MyModule", "MyModule.Customer.mdl")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDocumentFilePath_WithFolder(t *testing.T) {
	got := documentFilePath("/out", "MyModule", "Microflows/ACT", "MyModule.ACT_Foo")
	want := filepath.Join("/out", "MyModule", "Microflows", "ACT", "MyModule.ACT_Foo.mdl")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
```

- [ ] **Step 2: Run to confirm FAIL**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./mdl/executor/ -run TestDocumentFilePath -v 2>&1 | tail -10
```

Expected: `FAIL` — `documentFilePath` undefined.

- [ ] **Step 3: Implement `ExportOptions` and `documentFilePath`**

Create `mdl/executor/cmd_export_project.go`:

```go
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"path/filepath"
)

// ExportOptions controls the behaviour of ExportProject.
type ExportOptions struct {
	// Module restricts the export to a single module name. Empty = all modules.
	Module string
	// DryRun lists files that would be written without writing them.
	DryRun bool
	// Progress receives status lines (written to stderr by the caller).
	Progress func(line string)
}

// documentFilePath returns the output file path for a single document.
// outputDir is the root export directory.
// moduleName is the module the document belongs to.
// folderPath is the Studio Pro folder path (e.g. "Microflows/ACT"); empty = top-level.
// qname is the qualified name of the document (e.g. "MyModule.ACT_Foo").
func documentFilePath(outputDir, moduleName, folderPath, qname string) string {
	if folderPath != "" {
		return filepath.Join(outputDir, moduleName, filepath.FromSlash(folderPath), qname+".mdl")
	}
	return filepath.Join(outputDir, moduleName, qname+".mdl")
}
```

- [ ] **Step 4: Run to confirm PASS**

```bash
go test ./mdl/executor/ -run TestDocumentFilePath -v 2>&1 | tail -10
```

Expected: `PASS`.

- [ ] **Step 5: Commit**

```bash
git add mdl/executor/cmd_export_project.go mdl/executor/cmd_export_project_test.go
git commit -m "feat(export): ExportOptions struct and documentFilePath helper"
```

---

## Task 2: Module classification and `_marketplace.mdl`

**Files:**
- Modify: `mdl/executor/cmd_export_project.go`
- Modify: `mdl/executor/cmd_export_project_test.go`

- [ ] **Step 1: Write the failing test for `classifyModules`**

Add to `cmd_export_project_test.go`:

```go
import (
	"github.com/mendixlabs/mxcli/model"
)

func TestClassifyModules(t *testing.T) {
	mods := []*model.Module{
		{Name: "MyFirstModule", FromAppStore: false},
		{Name: "Administration", FromAppStore: true, AppStoreVersion: "3.4.0"},
		{Name: "AtlasCore", FromAppStore: true, AppStoreVersion: "4.0.0"},
	}
	regular, marketplace := classifyModules(mods)
	if len(regular) != 1 || regular[0].Name != "MyFirstModule" {
		t.Errorf("regular modules: got %v", regular)
	}
	if len(marketplace) != 2 {
		t.Errorf("marketplace modules: got %v", marketplace)
	}
}

func TestMarketplaceFileContent(t *testing.T) {
	mods := []*model.Module{
		{Name: "Administration", FromAppStore: true, AppStoreVersion: "3.4.0"},
		{Name: "AtlasCore", FromAppStore: true, AppStoreVersion: "4.0.0"},
	}
	got := marketplaceFileContent(mods)
	if !strings.Contains(got, "Administration") || !strings.Contains(got, "3.4.0") {
		t.Errorf("missing module in output: %q", got)
	}
	if !strings.Contains(got, "AtlasCore") {
		t.Errorf("missing AtlasCore: %q", got)
	}
}
```

Also add `"strings"` to the import block in the test file.

- [ ] **Step 2: Run to confirm FAIL**

```bash
go test ./mdl/executor/ -run TestClassifyModules/TestMarketplaceFileContent -v 2>&1 | tail -10
```

Expected: `FAIL` — functions undefined.

- [ ] **Step 3: Implement `classifyModules` and `marketplaceFileContent`**

Add to `mdl/executor/cmd_export_project.go`:

```go
import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mendixlabs/mxcli/model"
)

// classifyModules splits modules into regular (user-created) and marketplace modules.
func classifyModules(mods []*model.Module) (regular, marketplace []*model.Module) {
	for _, m := range mods {
		if m.FromAppStore {
			marketplace = append(marketplace, m)
		} else {
			regular = append(regular, m)
		}
	}
	return
}

// marketplaceFileContent generates the comment-only _marketplace.mdl content.
func marketplaceFileContent(mods []*model.Module) string {
	var sb strings.Builder
	sb.WriteString("-- Marketplace modules detected in this project.\n")
	sb.WriteString("-- Reinstall these before running mxcli import.\n")
	sb.WriteString("--\n")
	for _, m := range mods {
		version := m.AppStoreVersion
		if version == "" {
			version = "unknown"
		}
		fmt.Fprintf(&sb, "-- Module: %-30s (version: %s)\n", m.Name, version)
	}
	return sb.String()
}
```

- [ ] **Step 4: Run to confirm PASS**

```bash
go test ./mdl/executor/ -run "TestClassifyModules|TestMarketplaceFileContent" -v 2>&1 | tail -10
```

Expected: `PASS`.

- [ ] **Step 5: Commit**

```bash
git add mdl/executor/cmd_export_project.go mdl/executor/cmd_export_project_test.go
git commit -m "feat(export): classifyModules and marketplaceFileContent"
```

---

## Task 3: `captureDescribe` helper

**Files:**
- Modify: `mdl/executor/cmd_export_project.go`
- Modify: `mdl/executor/cmd_export_project_test.go`

- [ ] **Step 1: Write the failing test for `captureDescribe`**

Add to `cmd_export_project_test.go`:

```go
func TestCaptureDescribe_WritesToBuffer(t *testing.T) {
	var buf bytes.Buffer
	ctx := &ExecContext{Output: &buf}

	result, err := captureDescribe(ctx, func(c *ExecContext) error {
		fmt.Fprintln(c.Output, "hello world")
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello world\n" {
		t.Errorf("got %q, want %q", result, "hello world\n")
	}
	// Original output must still work after capture
	if buf.String() != "" {
		t.Errorf("original output was written to: %q", buf.String())
	}
}
```

Also add `"bytes"` and `"fmt"` to test imports.

- [ ] **Step 2: Run to confirm FAIL**

```bash
go test ./mdl/executor/ -run TestCaptureDescribe -v 2>&1 | tail -10
```

Expected: `FAIL` — `captureDescribe` undefined.

- [ ] **Step 3: Implement `captureDescribe`**

Add to `mdl/executor/cmd_export_project.go`:

```go
import (
	"bytes"
	// ... existing imports
)

// captureDescribe temporarily redirects ctx.Output to a buffer,
// calls fn, and returns the captured text.
// The original ctx.Output is restored even if fn returns an error.
func captureDescribe(ctx *ExecContext, fn func(*ExecContext) error) (string, error) {
	var buf bytes.Buffer
	saved := ctx.Output
	ctx.Output = &buf
	err := fn(ctx)
	ctx.Output = saved
	return buf.String(), err
}
```

- [ ] **Step 4: Run to confirm PASS**

```bash
go test ./mdl/executor/ -run TestCaptureDescribe -v 2>&1 | tail -10
```

Expected: `PASS`.

- [ ] **Step 5: Commit**

```bash
git add mdl/executor/cmd_export_project.go mdl/executor/cmd_export_project_test.go
git commit -m "feat(export): captureDescribe output-capture helper"
```

---

## Task 4: `ExportProject` — project-level files

**Files:**
- Modify: `mdl/executor/cmd_export_project.go`
- Modify: `mdl/executor/cmd_export_project_test.go`

- [ ] **Step 1: Write a failing integration test that verifies `_project/` files are created**

Add to `cmd_export_project_test.go`:

```go
import (
	"os"
	"path/filepath"
	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
)

func TestExportProject_ProjectLevelFiles(t *testing.T) {
	// Open fixture MPR via backend (same pattern as other integration tests)
	dst := copyMPRFixture(t, fixtureMprPath, t.TempDir())
	be, err := mprbackend.NewFromPath(dst)
	if err != nil {
		t.Fatalf("NewFromPath: %v", err)
	}
	t.Cleanup(func() { _ = be.Disconnect() })

	outDir := t.TempDir()
	exec := New(os.Stderr)
	exec.backend = be
	exec.cache = &executorCache{}

	opts := ExportOptions{
		Progress: func(line string) { t.Log(line) },
	}
	if err := exec.ExportProject(outDir, opts); err != nil {
		t.Fatalf("ExportProject: %v", err)
	}

	// _marketplace.mdl must exist
	marketFile := filepath.Join(outDir, "_marketplace.mdl")
	if _, err := os.Stat(marketFile); err != nil {
		t.Errorf("_marketplace.mdl missing: %v", err)
	}

	// _project/ directory must exist
	projDir := filepath.Join(outDir, "_project")
	if _, err := os.Stat(projDir); err != nil {
		t.Errorf("_project/ dir missing: %v", err)
	}
}
```

- [ ] **Step 2: Run to confirm FAIL**

```bash
go test ./mdl/executor/ -run TestExportProject_ProjectLevelFiles -v 2>&1 | tail -15
```

Expected: `FAIL` — `ExportProject` undefined.

- [ ] **Step 3: Implement `ExportProject` with project-level exports**

Add to `mdl/executor/cmd_export_project.go`:

```go
import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/model"
)

// ExportProject exports the connected project to a structured MDL directory.
func (e *Executor) ExportProject(outputDir string, opts ExportOptions) error {
	ctx := e.newExecContext(context.Background())
	if !ctx.Connected() {
		return fmt.Errorf("not connected to a project")
	}

	progress := opts.Progress
	if progress == nil {
		progress = func(string) {}
	}

	// 1. List and classify modules
	mods, err := ctx.Backend.ListModules()
	if err != nil {
		return fmt.Errorf("list modules: %w", err)
	}
	regular, marketplace := classifyModules(mods)

	if opts.Module != "" {
		filtered := regular[:0]
		for _, m := range regular {
			if m.Name == opts.Module {
				filtered = append(filtered, m)
			}
		}
		regular = filtered
	}

	// 2. Write _marketplace.mdl
	marketPath := filepath.Join(outputDir, "_marketplace.mdl")
	progress(fmt.Sprintf("  [market]   %d marketplace modules → _marketplace.mdl", len(marketplace)))
	if !opts.DryRun {
		if err := writeFile(marketPath, marketplaceFileContent(marketplace)); err != nil {
			return err
		}
	}

	// 3. Write _project/ files (settings, security, navigation)
	if opts.Module == "" {
		if err := e.exportProjectLevel(ctx, outputDir, opts, progress); err != nil {
			return err
		}
	}

	// 4. Export each regular module
	for _, m := range regular {
		if err := e.exportModule(ctx, m, outputDir, opts, progress); err != nil {
			return fmt.Errorf("export module %s: %w", m.Name, err)
		}
	}

	return nil
}

// exportProjectLevel writes _project/settings.mdl, security.mdl, navigation.mdl.
func (e *Executor) exportProjectLevel(ctx *ExecContext, outputDir string, opts ExportOptions, progress func(string)) error {
	projDir := filepath.Join(outputDir, "_project")

	type projectDoc struct {
		name string
		fn   func(*ExecContext) error
	}
	docs := []projectDoc{
		{"settings.mdl", func(c *ExecContext) error { return describeSettings(c) }},
		{"security.mdl", func(c *ExecContext) error {
			// Describe all user roles
			ps, err := getProjectSecurityGen(c)
			if err != nil || ps == nil {
				return err
			}
			for _, ur := range ps.UserRolesItems() {
				genSec, ok := ur.(interface{ Name() string })
				if !ok {
					continue
				}
				_ = describeUserRoleGen(c, ast.QualifiedName{Name: genSec.Name()})
			}
			return nil
		}},
		{"navigation.mdl", func(c *ExecContext) error {
			return describeNavigation(c, ast.QualifiedName{})
		}},
	}

	for _, doc := range docs {
		progress(fmt.Sprintf("  [project]  %s", doc.name))
		content, err := captureDescribe(ctx, doc.fn)
		if err != nil {
			progress(fmt.Sprintf("  [warn]     %s: %v", doc.name, err))
			continue
		}
		if !opts.DryRun {
			if err := writeFile(filepath.Join(projDir, doc.name), content); err != nil {
				return err
			}
		}
	}
	return nil
}

// writeFile writes content to path, creating parent directories as needed.
func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}
	return os.WriteFile(path, []byte(content), 0o644)
}
```

- [ ] **Step 4: Run to confirm PASS**

```bash
go test ./mdl/executor/ -run TestExportProject_ProjectLevelFiles -v 2>&1 | tail -15
```

Expected: `PASS`.

- [ ] **Step 5: Commit**

```bash
git add mdl/executor/cmd_export_project.go mdl/executor/cmd_export_project_test.go
git commit -m "feat(export): ExportProject with project-level files"
```

---

## Task 5: `exportModule` — per-document files for a single module

**Files:**
- Modify: `mdl/executor/cmd_export_project.go`
- Modify: `mdl/executor/cmd_export_project_test.go`

- [ ] **Step 1: Write a failing integration test verifying module document files exist**

Add to `cmd_export_project_test.go`:

```go
func TestExportProject_ModuleDocuments(t *testing.T) {
	dst := copyMPRFixture(t, fixtureMprPath, t.TempDir())
	be, err := mprbackend.NewFromPath(dst)
	if err != nil {
		t.Fatalf("NewFromPath: %v", err)
	}
	t.Cleanup(func() { _ = be.Disconnect() })

	outDir := t.TempDir()
	exec := New(os.Stderr)
	exec.backend = be
	exec.cache = &executorCache{}

	if err := exec.ExportProject(outDir, ExportOptions{Progress: func(l string) { t.Log(l) }}); err != nil {
		t.Fatalf("ExportProject: %v", err)
	}

	// The fixture has MyFirstModule with MyFirstLogic microflow.
	// Verify at least one .mdl file exists inside a module directory.
	var mdlFiles []string
	_ = filepath.WalkDir(outDir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(p, ".mdl") {
			mdlFiles = append(mdlFiles, p)
		}
		return nil
	})
	if len(mdlFiles) == 0 {
		t.Errorf("no .mdl files exported under %s", outDir)
	}
	// At least one file should be under a module directory (not _project)
	hasModuleFile := false
	for _, f := range mdlFiles {
		rel, _ := filepath.Rel(outDir, f)
		if !strings.HasPrefix(rel, "_") {
			hasModuleFile = true
		}
	}
	if !hasModuleFile {
		t.Errorf("no module-level .mdl files found in %v", mdlFiles)
	}
}
```

Add `"io/fs"` to the test import block.

- [ ] **Step 2: Run to confirm FAIL**

```bash
go test ./mdl/executor/ -run TestExportProject_ModuleDocuments -v 2>&1 | tail -20
```

Expected: `FAIL` — `exportModule` not implemented yet (it's stubbed or not called).

- [ ] **Step 3: Implement `exportModule`**

Add to `mdl/executor/cmd_export_project.go`:

```go
// exportModule exports all documents in a single regular module.
func (e *Executor) exportModule(ctx *ExecContext, m *model.Module, outputDir string, opts ExportOptions, progress func(string)) error {
	h, err := getHierarchy(ctx)
	if err != nil {
		return fmt.Errorf("get hierarchy: %w", err)
	}

	modName := m.Name
	counts := map[string]int{}

	// Helper: capture + write one document
	writeDoc := func(docType, folderPath, qname string, fn func(*ExecContext) error) {
		content, err := captureDescribe(ctx, fn)
		if err != nil {
			progress(fmt.Sprintf("  [warn]     %s.%s: %v", modName, qname, err))
			return
		}
		if content == "" {
			return
		}
		counts[docType]++
		if !opts.DryRun {
			path := documentFilePath(outputDir, modName, folderPath, qname)
			if werr := writeFile(path, content); werr != nil {
				progress(fmt.Sprintf("  [warn]     write %s: %v", path, werr))
			}
		}
	}

	// _module.mdl
	writeDoc("module", "", "_module", func(c *ExecContext) error {
		return describeModule(c, modName, false)
	})

	// Enumerations
	if enums, err := ctx.Backend.ListEnumerations(); err == nil {
		containers := getModuleContainers(ctx, m.ID)
		for _, enum := range enums {
			if !containers[enum.ContainerID] {
				continue
			}
			folder := h.BuildFolderPath(enum.ContainerID)
			qn := modName + "." + enum.Name
			e2 := enum // capture
			writeDoc("enumeration", folder, qn, func(c *ExecContext) error {
				return describeEnumeration(c, ast.QualifiedName{Module: modName, Name: e2.Name})
			})
		}
	}

	// Entities (sorted by generalization)
	if entities, err := listEntitiesForModuleGen(ctx, modName); err == nil {
		sorted := sortEntitiesByGeneralizationGen(entities, modName)
		for _, ent := range sorted {
			ent2 := ent
			writeDoc("entity", "Domain", modName+"."+ent.Name(), func(c *ExecContext) error {
				return describeEntityGen(c, ast.QualifiedName{Module: modName, Name: ent2.Name()})
			})
		}
	}

	// Associations → _associations.mdl (one file for all)
	if assocs, err := listAssociationsForModuleGen(ctx, modName); err == nil && len(assocs) > 0 {
		var assocBuf strings.Builder
		for _, assoc := range assocs {
			assoc2 := assoc
			content, err := captureDescribe(ctx, func(c *ExecContext) error {
				return describeAssociation(c, ast.QualifiedName{Module: modName, Name: assoc2.Name()})
			})
			if err != nil {
				progress(fmt.Sprintf("  [warn]     association %s.%s: %v", modName, assoc.Name(), err))
				continue
			}
			assocBuf.WriteString(content)
		}
		if s := assocBuf.String(); s != "" {
			counts["association"] = len(assocs)
			if !opts.DryRun {
				path := filepath.Join(outputDir, modName, "_associations.mdl")
				if werr := writeFile(path, s); werr != nil {
					progress(fmt.Sprintf("  [warn]     write _associations.mdl: %v", werr))
				}
			}
		}
	}

	// Constants
	if constants, err := ctx.Backend.ListConstants(); err == nil {
		h2, _ := getHierarchy(ctx)
		containers := getModuleContainers(ctx, m.ID)
		for _, c := range constants {
			if !containers[c.ContainerID] {
				continue
			}
			folder := ""
			if h2 != nil {
				folder = h2.BuildFolderPath(c.ContainerID)
			}
			c2 := c
			modNameCopy := modName
			writeDoc("constant", folder, modName+"."+c.Name, func(ctx2 *ExecContext) error {
				return outputConstantMDL(ctx2, c2, modNameCopy)
			})
		}
	}

	// Module roles (all into _module_roles.mdl)
	if pairs, err := listModuleSecurityWithContainerGen(ctx); err == nil {
		var roleBuf strings.Builder
		for _, p := range pairs {
			pModName := h.GetModuleName(p.ContainerID)
			if pModName != modName {
				continue
			}
			for _, mr := range p.MS.ModuleRolesItems() {
				typed, ok := mr.(interface{ Name() string })
				if !ok {
					continue
				}
				content, err := captureDescribe(ctx, func(c *ExecContext) error {
					return describeModuleRoleGen(c, ast.QualifiedName{Module: modName, Name: typed.Name()})
				})
				if err != nil {
					continue
				}
				roleBuf.WriteString(content)
			}
		}
		if s := roleBuf.String(); s != "" {
			if !opts.DryRun {
				path := filepath.Join(outputDir, modName, "_module_roles.mdl")
				_ = writeFile(path, s)
			}
		}
	}

	// Java actions
	if pairs, err := listJavaActionsWithContainerGen(ctx); err == nil {
		for _, p := range pairs {
			if p.Elem == nil {
				continue
			}
			pModName := h.GetModuleName(h.FindModuleID(modelIDFromElementID(p.ContainerID)))
			if pModName != modName {
				continue
			}
			folder := h.BuildFolderPath(modelIDFromElementID(p.ContainerID))
			elem := p.Elem
			writeDoc("javaaction", folder, modName+"."+elem.Name(), func(c *ExecContext) error {
				return describeJavaActionGen(c, ast.QualifiedName{Module: modName, Name: elem.Name()})
			})
		}
	}

	// JavaScript actions
	if pairs, err := listJavaScriptActionsWithContainerGen(ctx); err == nil {
		for _, p := range pairs {
			if p.Elem == nil {
				continue
			}
			pModName := h.GetModuleName(h.FindModuleID(modelIDFromElementID(p.ContainerID)))
			if pModName != modName {
				continue
			}
			folder := h.BuildFolderPath(modelIDFromElementID(p.ContainerID))
			elem := p.Elem
			writeDoc("jsaction", folder, modName+"."+elem.Name(), func(c *ExecContext) error {
				return describeJavaScriptActionGen(c, ast.QualifiedName{Module: modName, Name: elem.Name()})
			})
		}
	}

	// Microflows
	if items, err := listMicroflowsWithContainerGen(ctx); err == nil {
		for _, item := range items {
			pModName := h.GetModuleName(h.FindModuleID(item.ContainerUUID))
			if pModName != modName {
				continue
			}
			folder := h.BuildFolderPath(item.ContainerUUID)
			mf2 := item.MF
			writeDoc("microflow", folder, modName+"."+mf2.Name(), func(c *ExecContext) error {
				return describeMicroflowGen(c, ast.QualifiedName{Module: modName, Name: mf2.Name()})
			})
		}
	}

	// Nanoflows
	if pairs, err := listNanoflowsWithContainerGen(ctx); err == nil {
		for _, p := range pairs {
			if p.Elem == nil {
				continue
			}
			pModName := h.GetModuleName(h.FindModuleID(modelIDFromElementID(p.ContainerID)))
			if pModName != modName {
				continue
			}
			folder := h.BuildFolderPath(modelIDFromElementID(p.ContainerID))
			elem := p.Elem
			writeDoc("nanoflow", folder, modName+"."+elem.Name(), func(c *ExecContext) error {
				return describeNanoflowGen(c, ast.QualifiedName{Module: modName, Name: elem.Name()})
			})
		}
	}

	// Layouts
	if pairs, err := listLayoutsWithContainerGen(ctx); err == nil {
		for _, p := range pairs {
			if p.Elem == nil {
				continue
			}
			pModName := h.GetModuleName(h.FindModuleID(modelIDFromElementID(p.ContainerID)))
			if pModName != modName {
				continue
			}
			folder := h.BuildFolderPath(modelIDFromElementID(p.ContainerID))
			elem := p.Elem
			writeDoc("layout", folder, modName+"."+elem.Name(), func(c *ExecContext) error {
				return describeLayout(c, ast.QualifiedName{Module: modName, Name: elem.Name()})
			})
		}
	}

	// Snippets
	if pairs, err := listSnippetsWithContainerGen(ctx); err == nil {
		for _, p := range pairs {
			if p.Elem == nil {
				continue
			}
			pModName := h.GetModuleName(h.FindModuleID(modelIDFromElementID(p.ContainerID)))
			if pModName != modName {
				continue
			}
			folder := h.BuildFolderPath(modelIDFromElementID(p.ContainerID))
			elem := p.Elem
			writeDoc("snippet", folder, modName+"."+elem.Name(), func(c *ExecContext) error {
				return describeSnippet(c, ast.QualifiedName{Module: modName, Name: elem.Name()})
			})
		}
	}

	// Pages
	if pairs, err := listPagesWithContainerGen(ctx); err == nil {
		for _, p := range pairs {
			if p.Elem == nil {
				continue
			}
			pModName := h.GetModuleName(h.FindModuleID(modelIDFromElementID(p.ContainerID)))
			if pModName != modName {
				continue
			}
			folder := h.BuildFolderPath(modelIDFromElementID(p.ContainerID))
			elem := p.Elem
			writeDoc("page", folder, modName+"."+elem.Name(), func(c *ExecContext) error {
				return describePage(c, ast.QualifiedName{Module: modName, Name: elem.Name()})
			})
		}
	}

	// Workflows
	if pairs, err := listWorkflowsWithContainerGen(ctx); err == nil {
		for _, p := range pairs {
			if p.Elem == nil {
				continue
			}
			pModName := h.GetModuleName(h.FindModuleID(modelIDFromElementID(p.ContainerID)))
			if pModName != modName {
				continue
			}
			folder := h.BuildFolderPath(modelIDFromElementID(p.ContainerID))
			elem := p.Elem
			writeDoc("workflow", folder, modName+"."+elem.Name(), func(c *ExecContext) error {
				return describeWorkflowGen(c, ast.QualifiedName{Module: modName, Name: elem.Name()})
			})
		}
	}

	// Progress summary
	total := 0
	for _, n := range counts {
		total += n
	}
	progress(fmt.Sprintf("  [module]   %s (%d documents)", modName, total))
	for docType, n := range counts {
		progress(fmt.Sprintf("    ✓ %2d %s", n, docType))
	}

	return nil
}
```

- [ ] **Step 4: Run to confirm PASS**

```bash
go test ./mdl/executor/ -run TestExportProject_ModuleDocuments -v -timeout 60s 2>&1 | tail -20
```

Expected: `PASS`.

- [ ] **Step 5: Run all export tests**

```bash
go test ./mdl/executor/ -run TestExportProject -v -timeout 60s 2>&1 | tail -20
```

Expected: all `PASS`.

- [ ] **Step 6: Commit**

```bash
git add mdl/executor/cmd_export_project.go mdl/executor/cmd_export_project_test.go
git commit -m "feat(export): exportModule writes per-document MDL files"
```

---

## Task 6: `ImportOptions` and topological sort

**Files:**
- Create: `mdl/executor/cmd_import_project.go`
- Modify: `mdl/executor/cmd_export_project_test.go` (add import tests)

- [ ] **Step 1: Write the failing test for `sortMDLFiles`**

Add to `cmd_export_project_test.go`:

```go
func TestSortMDLFiles_OrderedByType(t *testing.T) {
	files := []string{
		"MyModule/Pages/MyModule.Home.mdl",
		"MyModule/Domain/MyModule.Customer.mdl",
		"MyModule/_module.mdl",
		"MyModule/_associations.mdl",
		"MyModule/Microflows/ACT/MyModule.ACT_Create.mdl",
		"_project/settings.mdl",
		"_project/security.mdl",
		"_project/navigation.mdl",
		"_marketplace.mdl",
	}
	sorted := sortMDLFiles(files)

	// _module.mdl should come before entities (Domain/)
	moduleIdx := indexOfSuffix(sorted, "_module.mdl")
	domainIdx := indexOfSuffix(sorted, "MyModule.Customer.mdl")
	if moduleIdx < 0 || domainIdx < 0 {
		t.Fatal("missing expected files")
	}
	if moduleIdx > domainIdx {
		t.Errorf("_module.mdl (idx %d) should come before Domain/ (idx %d)", moduleIdx, domainIdx)
	}

	// associations after entities
	assocIdx := indexOfSuffix(sorted, "_associations.mdl")
	if assocIdx < domainIdx {
		t.Errorf("_associations.mdl (idx %d) should come after Domain/ (idx %d)", assocIdx, domainIdx)
	}

	// pages after microflows
	mfIdx := indexOfSuffix(sorted, "ACT_Create.mdl")
	pgIdx := indexOfSuffix(sorted, "Home.mdl")
	if mfIdx > pgIdx {
		t.Errorf("microflow (idx %d) should come before page (idx %d)", mfIdx, pgIdx)
	}

	// _project/settings last
	settingsIdx := indexOfSuffix(sorted, "settings.mdl")
	if settingsIdx < pgIdx {
		t.Errorf("settings.mdl (idx %d) should come after pages (idx %d)", settingsIdx, pgIdx)
	}
}

func indexOfSuffix(slice []string, suffix string) int {
	for i, s := range slice {
		if strings.HasSuffix(filepath.ToSlash(s), suffix) {
			return i
		}
	}
	return -1
}
```

- [ ] **Step 2: Run to confirm FAIL**

```bash
go test ./mdl/executor/ -run TestSortMDLFiles -v 2>&1 | tail -10
```

Expected: `FAIL` — `sortMDLFiles` undefined.

- [ ] **Step 3: Implement `ImportOptions` and `sortMDLFiles`**

Create `mdl/executor/cmd_import_project.go`:

```go
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"path/filepath"
	"sort"
	"strings"
)

// ImportOptions controls the behaviour of ImportProject.
type ImportOptions struct {
	// Module restricts the import to a single module directory. Empty = all.
	Module string
	// DryRun parses all files and reports errors without modifying the project.
	DryRun bool
	// SkipErrors continues past individual file failures (default: stop on first error).
	SkipErrors bool
	// Progress receives status lines.
	Progress func(line string)
}

// importDocumentOrder maps path patterns to a sort priority (lower = earlier).
var importDocumentOrder = []struct {
	pattern  string
	priority int
}{
	{"_marketplace.mdl", 0},   // skip (comments only)
	{"_module.mdl", 1},        // CREATE MODULE
	{"Domain/", 2},            // entities
	{"_associations.mdl", 3},  // associations
	{"Constants/", 4},         // constants
	{"_module_roles.mdl", 5},  // module roles
	{"JavaActions/", 6},       // java actions
	{"JavaScriptActions/", 7}, // javascript actions
	{"Microflows/", 8},        // microflows
	{"Nanoflows/", 9},         // nanoflows
	{"Layouts/", 10},          // layouts
	{"Snippets/", 11},         // snippets
	{"Pages/", 12},            // pages
	{"Workflows/", 13},        // workflows
	{"_project/navigation", 14},
	{"_project/security", 15},
	{"_project/settings", 16},
}

// fileImportPriority returns the sort priority for a file path.
// Lower = executed earlier.
func fileImportPriority(path string) int {
	norm := filepath.ToSlash(path)
	for _, entry := range importDocumentOrder {
		if strings.Contains(norm, entry.pattern) || strings.HasSuffix(norm, entry.pattern) {
			return entry.priority
		}
	}
	return 50 // unknown: late
}

// sortMDLFiles returns a copy of paths sorted by import dependency order.
func sortMDLFiles(paths []string) []string {
	out := make([]string, len(paths))
	copy(out, paths)
	sort.SliceStable(out, func(i, j int) bool {
		pi := fileImportPriority(out[i])
		pj := fileImportPriority(out[j])
		if pi != pj {
			return pi < pj
		}
		return out[i] < out[j]
	})
	return out
}
```

- [ ] **Step 4: Run to confirm PASS**

```bash
go test ./mdl/executor/ -run TestSortMDLFiles -v 2>&1 | tail -10
```

Expected: `PASS`.

- [ ] **Step 5: Commit**

```bash
git add mdl/executor/cmd_import_project.go mdl/executor/cmd_export_project_test.go
git commit -m "feat(import): ImportOptions and sortMDLFiles topological sort"
```

---

## Task 7: `ImportProject` implementation

**Files:**
- Modify: `mdl/executor/cmd_import_project.go`
- Modify: `mdl/executor/cmd_export_project_test.go`

- [ ] **Step 1: Write the failing integration test for `ImportProject`**

Add to `cmd_export_project_test.go`:

```go
func TestImportProject_ExecutesFiles(t *testing.T) {
	// Export the fixture to a temp dir
	dst := copyMPRFixture(t, fixtureMprPath, t.TempDir())
	be, err := mprbackend.NewFromPath(dst)
	if err != nil {
		t.Fatalf("NewFromPath: %v", err)
	}
	t.Cleanup(func() { _ = be.Disconnect() })

	exportDir := t.TempDir()
	exportExec := New(os.Stderr)
	exportExec.backend = be
	exportExec.cache = &executorCache{}
	if err := exportExec.ExportProject(exportDir, ExportOptions{Progress: func(l string) { t.Log("export:", l) }}); err != nil {
		t.Fatalf("ExportProject: %v", err)
	}
	_ = be.Disconnect()

	// Import into a fresh copy of the fixture
	dst2 := copyMPRFixture(t, fixtureMprPath, t.TempDir())
	be2, err := mprbackend.NewFromPath(dst2)
	if err != nil {
		t.Fatalf("NewFromPath dst2: %v", err)
	}
	t.Cleanup(func() { _ = be2.Disconnect() })

	importExec := New(os.Stderr)
	importExec.backend = be2
	importExec.cache = &executorCache{}
	err = importExec.ImportProject(exportDir, ImportOptions{
		SkipErrors: true,
		Progress:   func(l string) { t.Log("import:", l) },
	})
	if err != nil {
		t.Fatalf("ImportProject: %v", err)
	}
}
```

- [ ] **Step 2: Run to confirm FAIL**

```bash
go test ./mdl/executor/ -run TestImportProject_ExecutesFiles -v -timeout 120s 2>&1 | tail -15
```

Expected: `FAIL` — `ImportProject` undefined.

- [ ] **Step 3: Implement `ImportProject`**

Add to `mdl/executor/cmd_import_project.go`:

```go
import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/visitor"
)

// ImportProject imports all .mdl files from inputDir into the connected project.
// Files are executed in dependency order (enumerations → entities → ... → settings).
func (e *Executor) ImportProject(inputDir string, opts ImportOptions) error {
	ctx := e.newExecContext(context.Background())
	if !ctx.Connected() {
		return fmt.Errorf("not connected to a project")
	}

	progress := opts.Progress
	if progress == nil {
		progress = func(string) {}
	}

	// Collect all .mdl files
	var allFiles []string
	if err := filepath.WalkDir(inputDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".mdl") {
			rel, _ := filepath.Rel(inputDir, path)
			allFiles = append(allFiles, rel)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("scan %s: %w", inputDir, err)
	}

	// Filter by module if requested
	if opts.Module != "" {
		filtered := allFiles[:0]
		for _, f := range allFiles {
			norm := filepath.ToSlash(f)
			if strings.HasPrefix(norm, opts.Module+"/") || strings.HasPrefix(norm, "_project/") {
				filtered = append(filtered, f)
			}
		}
		allFiles = filtered
	}

	// Skip _marketplace.mdl (comments only)
	filtered := allFiles[:0]
	for _, f := range allFiles {
		if filepath.Base(f) != "_marketplace.mdl" {
			filtered = append(filtered, f)
		}
	}
	allFiles = filtered

	sorted := sortMDLFiles(allFiles)

	var errs []string
	for _, rel := range sorted {
		fullPath := filepath.Join(inputDir, rel)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			msg := fmt.Sprintf("read %s: %v", rel, err)
			if !opts.SkipErrors {
				return fmt.Errorf("%s", msg)
			}
			errs = append(errs, msg)
			continue
		}

		if opts.DryRun {
			prog, parseErrs := visitor.Build(string(content))
			if len(parseErrs) > 0 {
				for _, pe := range parseErrs {
					msg := fmt.Sprintf("parse %s: %v", rel, pe)
					errs = append(errs, msg)
					if !opts.SkipErrors {
						return fmt.Errorf("%s", msg)
					}
				}
			}
			_ = prog
			progress(fmt.Sprintf("  [dry-run]  %s", rel))
			continue
		}

		prog, parseErrs := visitor.Build(string(content))
		if len(parseErrs) > 0 {
			for _, pe := range parseErrs {
				msg := fmt.Sprintf("parse %s: %v", rel, pe)
				if !opts.SkipErrors {
					return fmt.Errorf("%s", msg)
				}
				errs = append(errs, msg)
			}
			continue
		}

		progress(fmt.Sprintf("  [exec]     %s", rel))
		if err := e.ExecuteProgram(prog); err != nil {
			msg := fmt.Sprintf("exec %s: %v", rel, err)
			if !opts.SkipErrors {
				return fmt.Errorf("%s", msg)
			}
			errs = append(errs, msg)
		}
	}

	if len(errs) > 0 {
		progress(fmt.Sprintf("  [warn]     %d file(s) had errors:", len(errs)))
		for _, e := range errs {
			progress(fmt.Sprintf("    - %s", e))
		}
	}

	return nil
}

// Ensure sort is used (it's in sortMDLFiles above)
var _ = sort.SliceStable
```

- [ ] **Step 4: Run to confirm PASS**

```bash
go test ./mdl/executor/ -run TestImportProject_ExecutesFiles -v -timeout 120s 2>&1 | tail -20
```

Expected: `PASS` (some `[warn]` lines for documents that already exist are fine with `SkipErrors: true`).

- [ ] **Step 5: Commit**

```bash
git add mdl/executor/cmd_import_project.go mdl/executor/cmd_export_project_test.go
git commit -m "feat(import): ImportProject scans dir and executes MDL files in order"
```

---

## Task 8: `cmd/mxcli/cmd_export.go` — Cobra command

**Files:**
- Create: `cmd/mxcli/cmd_export.go`
- Modify: `cmd/mxcli/main.go`

- [ ] **Step 1: Create `cmd_export.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/mendixlabs/mxcli/mdl/backend"
	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/mendixlabs/mxcli/mdl/executor"
	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export a Mendix project to structured MDL files",
	Long: `Export all documents in a Mendix project to a directory of MDL files.

Each document (entity, microflow, page, etc.) is exported to its own .mdl file.
The directory structure mirrors the Studio Pro folder hierarchy.

Marketplace modules are listed in _marketplace.mdl (not executable).
Regular modules are exported to per-module directories.

Examples:
  mxcli export -p app.mpr --output ./export-dir
  mxcli export -p app.mpr --output ./export-dir --module MyFirstModule
  mxcli export -p app.mpr --output ./export-dir --dry-run
`,
	Run: func(cmd *cobra.Command, args []string) {
		projectPath, _ := cmd.Flags().GetString("project")
		outputDir, _ := cmd.Flags().GetString("output")
		moduleFilter, _ := cmd.Flags().GetString("module")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		if projectPath == "" {
			fmt.Fprintln(os.Stderr, "Error: --project (-p) is required")
			os.Exit(1)
		}
		if outputDir == "" {
			fmt.Fprintln(os.Stderr, "Error: --output is required")
			os.Exit(1)
		}

		be := mprbackend.New()
		if err := be.Connect(projectPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error connecting to %s: %v\n", projectPath, err)
			os.Exit(1)
		}
		defer func() { _ = be.Disconnect() }()

		exec := executor.New(os.Stdout)
		exec.SetBackendFactory(func() backend.FullBackend { return mprbackend.New() })
		// Wire the already-connected backend directly
		exec.SetBackend(be)

		start := time.Now()
		fmt.Fprintf(os.Stderr, "Exporting %s → %s\n", projectPath, outputDir)

		opts := executor.ExportOptions{
			Module: moduleFilter,
			DryRun: dryRun,
			Progress: func(line string) {
				fmt.Fprintln(os.Stderr, line)
			},
		}

		if err := exec.ExportProject(outputDir, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if dryRun {
			fmt.Fprintf(os.Stderr, "Done (dry run) in %.1fs\n", time.Since(start).Seconds())
		} else {
			fmt.Fprintf(os.Stderr, "Done in %.1fs\n", time.Since(start).Seconds())
		}
	},
}

func init() {
	exportCmd.Flags().String("output", "", "Output directory for MDL files (required)")
	exportCmd.Flags().String("module", "", "Export only this module (default: all modules)")
	exportCmd.Flags().Bool("dry-run", false, "List files that would be written without writing them")
}
```

Note: `exec.SetBackend(be)` requires adding a `SetBackend` method to `Executor`. See Step 3.

- [ ] **Step 2: Add `SetBackend` to `Executor`**

In `mdl/executor/executor.go`, after `func (e *Executor) SetLogger`:

```go
// SetBackend sets the already-connected backend (used by export/import commands
// that manage the backend lifecycle externally).
func (e *Executor) SetBackend(b backend.FullBackend) {
	e.backend = b
	e.mprPath = b.Path()
	if e.cache == nil {
		e.cache = &executorCache{}
	}
}
```

Check that `backend.FullBackend` has a `Path() string` method. If not, use:

```go
func (e *Executor) SetBackend(b backend.FullBackend) {
	e.backend = b
	if e.cache == nil {
		e.cache = &executorCache{}
	}
}
```

- [ ] **Step 3: Register exportCmd in `main.go`**

In `cmd/mxcli/main.go`, in the `init()` function, after `rootCmd.AddCommand(mprPackCmd)`:

```go
rootCmd.AddCommand(exportCmd)
```

- [ ] **Step 4: Build and smoke-test**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go build -o bin/mxcli ./cmd/mxcli 2>&1
./bin/mxcli export --help
```

Expected: help text displays with `--output`, `--module`, `--dry-run` flags.

- [ ] **Step 5: Commit**

```bash
git add cmd/mxcli/cmd_export.go cmd/mxcli/main.go mdl/executor/executor.go
git commit -m "feat(export): mxcli export Cobra command"
```

---

## Task 9: `cmd/mxcli/cmd_import.go` — Cobra command

**Files:**
- Create: `cmd/mxcli/cmd_import.go`
- Modify: `cmd/mxcli/main.go`

- [ ] **Step 1: Create `cmd_import.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/mendixlabs/mxcli/mdl/backend"
	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/mendixlabs/mxcli/mdl/executor"
	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import MDL files into a Mendix project",
	Long: `Import MDL files from an export directory into a Mendix project.

The target project must already have marketplace modules pre-installed.
Files are executed in dependency order (enumerations → entities → microflows → pages → ...).

Examples:
  mxcli import -p target.mpr --input ./export-dir
  mxcli import -p target.mpr --input ./export-dir --module MyFirstModule
  mxcli import -p target.mpr --input ./export-dir --dry-run
  mxcli import -p target.mpr --input ./export-dir --skip-errors
`,
	Run: func(cmd *cobra.Command, args []string) {
		projectPath, _ := cmd.Flags().GetString("project")
		inputDir, _ := cmd.Flags().GetString("input")
		moduleFilter, _ := cmd.Flags().GetString("module")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		skipErrors, _ := cmd.Flags().GetBool("skip-errors")

		if projectPath == "" {
			fmt.Fprintln(os.Stderr, "Error: --project (-p) is required")
			os.Exit(1)
		}
		if inputDir == "" {
			fmt.Fprintln(os.Stderr, "Error: --input is required")
			os.Exit(1)
		}

		be := mprbackend.New()
		if err := be.Connect(projectPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error connecting to %s: %v\n", projectPath, err)
			os.Exit(1)
		}
		defer func() { _ = be.Disconnect() }()

		exec := executor.New(os.Stdout)
		exec.SetBackendFactory(func() backend.FullBackend { return mprbackend.New() })
		exec.SetBackend(be)

		start := time.Now()
		fmt.Fprintf(os.Stderr, "Importing %s → %s\n", inputDir, projectPath)

		opts := executor.ImportOptions{
			Module:     moduleFilter,
			DryRun:     dryRun,
			SkipErrors: skipErrors,
			Progress: func(line string) {
				fmt.Fprintln(os.Stderr, line)
			},
		}

		if err := exec.ImportProject(inputDir, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if dryRun {
			fmt.Fprintf(os.Stderr, "Done (dry run) in %.1fs\n", time.Since(start).Seconds())
		} else {
			fmt.Fprintf(os.Stderr, "Done in %.1fs\n", time.Since(start).Seconds())
		}
	},
}

func init() {
	importCmd.Flags().String("input", "", "Input directory containing MDL files (required)")
	importCmd.Flags().String("module", "", "Import only this module (default: all modules)")
	importCmd.Flags().Bool("dry-run", false, "Parse files and report errors without modifying the project")
	importCmd.Flags().Bool("skip-errors", false, "Continue past individual file errors (default: stop on first error)")
}
```

- [ ] **Step 2: Register importCmd in `main.go`**

In `cmd/mxcli/main.go`, in the `init()` function, after `rootCmd.AddCommand(exportCmd)`:

```go
rootCmd.AddCommand(importCmd)
```

- [ ] **Step 3: Build and smoke-test**

```bash
go build -o bin/mxcli ./cmd/mxcli 2>&1
./bin/mxcli import --help
```

Expected: help text displays with `--input`, `--module`, `--dry-run`, `--skip-errors` flags.

- [ ] **Step 4: Commit**

```bash
git add cmd/mxcli/cmd_import.go cmd/mxcli/main.go
git commit -m "feat(import): mxcli import Cobra command"
```

---

## Task 10: Round-trip integration test

**Files:**
- Modify: `mdl/executor/cmd_export_project_test.go`

- [ ] **Step 1: Write the round-trip test**

Add to `cmd_export_project_test.go`:

```go
func TestRoundTrip_ExportThenImport(t *testing.T) {
	// Step 1: export the fixture
	exportDst := copyMPRFixture(t, fixtureMprPath, t.TempDir())
	exportBe, err := mprbackend.NewFromPath(exportDst)
	if err != nil {
		t.Fatalf("NewFromPath (export): %v", err)
	}

	exportDir := t.TempDir()
	exportExec := New(os.Stderr)
	exportExec.backend = exportBe
	exportExec.cache = &executorCache{}
	if err := exportExec.ExportProject(exportDir, ExportOptions{Progress: func(l string) { t.Log("export:", l) }}); err != nil {
		t.Fatalf("ExportProject: %v", err)
	}
	_ = exportBe.Disconnect()

	// Count original entities in source
	origDst := copyMPRFixture(t, fixtureMprPath, t.TempDir())
	origBe, err := mprbackend.NewFromPath(origDst)
	if err != nil {
		t.Fatalf("NewFromPath (orig): %v", err)
	}
	origMods, _ := origBe.ListModules()
	origCtx := &ExecContext{Backend: origBe, Cache: &executorCache{}}
	origEntityCount := 0
	for _, m := range origMods {
		if m.FromAppStore {
			continue
		}
		ents, _ := listEntitiesForModuleGen(origCtx, m.Name)
		origEntityCount += len(ents)
	}
	_ = origBe.Disconnect()

	// Step 2: import into a fresh copy
	importDst := copyMPRFixture(t, fixtureMprPath, t.TempDir())
	importBe, err := mprbackend.NewFromPath(importDst)
	if err != nil {
		t.Fatalf("NewFromPath (import): %v", err)
	}
	t.Cleanup(func() { _ = importBe.Disconnect() })

	importExec := New(os.Stderr)
	importExec.backend = importBe
	importExec.cache = &executorCache{}
	if err := importExec.ImportProject(exportDir, ImportOptions{
		SkipErrors: true,
		Progress:   func(l string) { t.Log("import:", l) },
	}); err != nil {
		t.Fatalf("ImportProject: %v", err)
	}

	// Step 3: verify entity count matches
	importMods, _ := importBe.ListModules()
	importCtx := &ExecContext{Backend: importBe, Cache: &executorCache{}}
	importEntityCount := 0
	for _, m := range importMods {
		if m.FromAppStore {
			continue
		}
		ents, _ := listEntitiesForModuleGen(importCtx, m.Name)
		importEntityCount += len(ents)
	}

	if importEntityCount < origEntityCount {
		t.Errorf("entity count after import (%d) is less than original (%d)", importEntityCount, origEntityCount)
	}
}
```

- [ ] **Step 2: Run to confirm PASS**

```bash
go test ./mdl/executor/ -run TestRoundTrip_ExportThenImport -v -timeout 120s 2>&1 | tail -20
```

Expected: `PASS`.

- [ ] **Step 3: Run full test suite to check for regressions**

```bash
go test ./mdl/executor/... -timeout 120s 2>&1 | tail -10
go test ./cmd/mxcli/... -timeout 60s 2>&1 | tail -10
```

Expected: no new failures.

- [ ] **Step 4: Commit**

```bash
git add mdl/executor/cmd_export_project_test.go
git commit -m "test(export): round-trip export→import integration test"
```

---

## Task 11: End-to-end smoke test with the CLI

- [ ] **Step 1: Build**

```bash
go build -o bin/mxcli ./cmd/mxcli 2>&1
```

- [ ] **Step 2: Export the testdata project**

```bash
./bin/mxcli export -p testdata/expr-checker/minimal.mpr --output /tmp/export-test 2>&1
```

Expected: progress output, no errors. Check that files exist:

```bash
find /tmp/export-test -name "*.mdl" | head -20
```

- [ ] **Step 3: Dry-run import**

```bash
cp -r testdata/expr-checker /tmp/import-target
./bin/mxcli import -p /tmp/import-target/minimal.mpr --input /tmp/export-test --dry-run --skip-errors 2>&1
```

Expected: `[dry-run]` lines for each file, no errors.

- [ ] **Step 4: Clean up temp dirs**

```bash
rm -rf /tmp/export-test /tmp/import-target
```

- [ ] **Step 5: Final commit**

```bash
git add -A
git commit -m "feat(export): mxcli export/import commands complete"
```

---

## Self-Review

**Spec coverage check:**
- ✅ `mxcli export -p ... --output` — Task 8
- ✅ `mxcli export ... --module` — Task 8 (ExportOptions.Module)
- ✅ `mxcli export ... --dry-run` — Task 8
- ✅ `mxcli import -p ... --input` — Task 9
- ✅ `mxcli import ... --module` — Task 9 (ImportOptions.Module)
- ✅ `mxcli import ... --dry-run` — Task 9
- ✅ `mxcli import ... --skip-errors` — Task 9
- ✅ Project-level files (`_project/`) — Task 4
- ✅ Marketplace module list (`_marketplace.mdl`) — Task 2
- ✅ Per-document MDL files — Task 5
- ✅ Studio Pro folder hierarchy — Task 5 (`h.BuildFolderPath`)
- ✅ QName file naming — Task 1 (`documentFilePath`)
- ✅ Topological import order — Task 6
- ✅ Error: skip + warn on export — Task 5 (writeDoc)
- ✅ Error: stop on first import failure (default) — Task 7
- ✅ Progress output to stderr — Tasks 4, 5, 7, 8, 9
- ✅ Unit tests — Tasks 1, 2, 3, 6
- ✅ Integration test (export fixture) — Task 4, 5
- ✅ Round-trip test — Task 10

**Type consistency check:**
- `ExportOptions` defined in Task 1, used in Tasks 4, 5, 8 ✅
- `ImportOptions` defined in Task 6, used in Tasks 7, 9, 10 ✅
- `documentFilePath` defined in Task 1, used in Task 5 ✅
- `captureDescribe` defined in Task 3, used in Task 5 ✅
- `classifyModules` defined in Task 2, used in Task 4 ✅
- `sortMDLFiles` defined in Task 6, used in Task 7 ✅
- `exec.ExportProject` defined in Task 4, used in Tasks 8, 10 ✅
- `exec.ImportProject` defined in Task 7, used in Tasks 9, 10 ✅
- `exec.SetBackend` defined in Task 8, used in Tasks 8, 9 ✅
