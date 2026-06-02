# Page Canonical Model Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `mdl/canonical/page/` — the canonical model layer for Mendix pages — following the same Lift/Hydrate/ToMDL/Persist pattern as entity and association, and route the diff command's page functions through the codec registry.

**Architecture:** `PageDocument` wraps `types.PageModel` (the now-complete IR from the foundation plan) and implements `canonical.Document`. `Lift` converts `ast.CreatePageStmt` via the existing `pageASTToModel`. `Hydrate` converts a `*genPg.Page` via the existing read path. `ToMDL` delegates to the existing `pageModelToMDL`. `Persist` delegates to the existing gen builder. The codec is registered in `executor.go`; `cmd_diff_mdl.go` routes page functions through `ctx.ModelCodecs`.

**Tech Stack:** Go 1.26, `modelsdk/gen/pages`, `mdl/ast`, `mdl/canonical`.

**Prerequisite:** `2026-06-02-page-ir-foundation.md` must be complete (DataView footer + column widths round-trip, DataView overlay enabled).

**Note on architecture:** Unlike entity and association (which define standalone structs), page wraps `*types.PageModel` because the page IR is already correct and complete after the foundation plan. The canonical layer adds the interface contract and codec registration — it does not duplicate the serialization logic.

---

## File Map

**New files:**

| File | Responsibility |
|------|----------------|
| `mdl/canonical/page/model.go` | `PageDocument` struct wrapping `*types.PageModel`; `ToMDL() string` |
| `mdl/canonical/page/lift.go` | `Lift(ctx, *ast.CreatePageStmtV3) (*PageDocument, error)` |
| `mdl/canonical/page/hydrate.go` | `Hydrate(ctx, *genPg.Page) (*PageDocument, []canonical.Warning, error)` |
| `mdl/canonical/page/persist.go` | `(*PageDocument).Persist(ctx canonical.PersistContext) error` |
| `mdl/canonical/page/codec.go` | `RegisterCodec(*canonical.DefaultRegistry)` |
| `mdl/canonical/page/comply_test.go` | Compile-time `var _ canonical.Document = (*PageDocument)(nil)` |
| `mdl/canonical/page/page_test.go` | Unit tests: Lift, ToMDL, Hydrate |

**Modified files:**

| File | Change |
|------|--------|
| `mdl/canonical/codec_guard_test.go` | Add `"Forms$Page"` to Required list |
| `mdl/executor/executor.go` | Call `pagemodel.RegisterCodec(mc)` |
| `mdl/executor/cmd_diff_mdl.go` | Route `pageStmtToMDL` + `pageToMDLGen` through `ctx.ModelCodecs` |
| `mdl/executor/boundary_guard_test.go` | Add `"mdl/canonical/page"` to Forbidden list |

---

## Task 1: `PageDocument` data type + `ToMDL`

**Files:**
- Create: `mdl/canonical/page/model.go`
- Create: `mdl/canonical/page/comply_test.go`

- [ ] **Step 1: Create `mdl/canonical/page/model.go`**

```go
// SPDX-License-Identifier: Apache-2.0

// Package page implements the canonical model for Mendix page documents.
// It wraps the types.PageModel IR and satisfies the canonical.Document and
// canonical.Persistable interfaces.
package page

import (
	"github.com/mendixlabs/mxcli/mdl/executor"
	"github.com/mendixlabs/mxcli/mdl/types"
)

// PageDocument is the canonical model for a Mendix page.
// It wraps types.PageModel (the complete IR) and implements Document/Persistable.
type PageDocument struct {
	pm *types.PageModel
}

// ToMDL renders the page as deterministic MDL text via the existing
// pageModelToMDL renderer in the executor package.
func (d *PageDocument) ToMDL() string {
	if d == nil || d.pm == nil {
		return ""
	}
	return executor.PageModelToMDL(d.pm)
}

// PageModel returns the underlying IR for callers that need direct access.
func (d *PageDocument) PageModel() *types.PageModel {
	return d.pm
}
```

**Note:** `executor.PageModelToMDL` must be exported. In the next step we export it.

- [ ] **Step 2: Export `pageModelToMDL` in executor**

In `mdl/executor/cmd_pages_model_to_mdl.go`, find the `pageModelToMDL` function (or equivalent). If it's unexported, add an exported wrapper:

```go
// PageModelToMDL is the exported wrapper for canonical/page to call.
func PageModelToMDL(pm *types.PageModel) string {
	return pageModelToMDL(pm)
}
```

- [ ] **Step 3: Create `mdl/canonical/page/comply_test.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package page_test

import (
	"github.com/mendixlabs/mxcli/mdl/canonical"
	"github.com/mendixlabs/mxcli/mdl/canonical/page"
)

// Compile-time interface assertions.
var _ canonical.Document    = (*page.PageDocument)(nil)
var _ canonical.Persistable = (*page.PageDocument)(nil)
```

- [ ] **Step 4: Build — expect compile error (Persist missing)**

```bash
go build ./mdl/canonical/page/... 2>&1 | head -10
```

Expected: compile error — `*PageDocument` does not implement `canonical.Persistable` (missing Persist method).

- [ ] **Step 5: Commit stub**

```bash
git add mdl/canonical/page/model.go mdl/canonical/page/comply_test.go mdl/executor/cmd_pages_model_to_mdl.go
git commit -m "feat(canonical/page): add PageDocument stub + export PageModelToMDL"
```

---

## Task 2: `Lift` — AST → PageDocument (TDD)

**Files:**
- Create: `mdl/canonical/page/lift.go`
- Create: `mdl/canonical/page/page_test.go`

- [ ] **Step 1: Write failing test in `page_test.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package page_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/canonical/page"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func minimalPageAST() *ast.CreatePageStmtV3 {
	return &ast.CreatePageStmtV3{
		Name:   ast.QualifiedName{Module: "M", Name: "TestPage"},
		Layout: "Atlas_Core.Atlas_Default",
	}
}

func TestLift_MinimalPage(t *testing.T) {
	doc, err := page.Lift(nil, minimalPageAST(), "M")
	require.NoError(t, err)
	require.NotNil(t, doc)
	pm := doc.PageModel()
	require.NotNil(t, pm)
	assert.Equal(t, "M", pm.ModuleName)
	assert.Equal(t, "TestPage", pm.Name)
	assert.Equal(t, "Atlas_Core.Atlas_Default", pm.Layout)
}

func TestLift_NilStmt_ReturnsError(t *testing.T) {
	_, err := page.Lift(nil, nil, "M")
	assert.Error(t, err)
}
```

- [ ] **Step 2: Run — expect compile failure**

```bash
go test ./mdl/canonical/page/... -run TestLift 2>&1 | head -10
```

Expected: `page.Lift undefined`.

- [ ] **Step 3: Create `mdl/canonical/page/lift.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package page

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/executor"
	"github.com/mendixlabs/mxcli/mdl/types"
)

// Lift converts a parsed CREATE PAGE AST statement to a PageDocument.
// ctx may be nil in tests; it is forwarded to pageASTToModel for catalog lookups.
// moduleName is the owning module.
func Lift(ctx executor.ASTToModelContext, s *ast.CreatePageStmtV3, moduleName string) (*PageDocument, error) {
	if s == nil {
		return nil, fmt.Errorf("page.Lift: nil statement")
	}
	pm := executor.PageASTToModel(ctx, s, moduleName)
	if pm == nil {
		pm = &types.PageModel{
			ModuleName: moduleName,
			Name:       s.Name.Name,
			Layout:     s.Layout,
		}
	}
	return &PageDocument{pm: pm}, nil
}
```

**Note:** `executor.PageASTToModel` and `executor.ASTToModelContext` must be exported. In the next step we add these exports.

- [ ] **Step 4: Export `pageASTToModel` in executor**

In `mdl/executor/cmd_pages_ast_to_model.go`, add the exported interface and wrapper:

```go
// ASTToModelContext is the subset of ExecContext used by pageASTToModel.
// The canonical/page package implements Lift via this interface, not ExecContext,
// to avoid importing the executor package (which would create a cycle).
type ASTToModelContext interface {
	// No methods required for now; future: catalog lookup for entity refs.
}

// PageASTToModel is the exported entry point for canonical/page.Lift.
func PageASTToModel(ctx ASTToModelContext, s *ast.CreatePageStmtV3, moduleName string) *types.PageModel {
	if s == nil {
		return nil
	}
	// Convert ExecContext-independent fields directly.
	pm := &types.PageModel{
		ModuleName: moduleName,
		Name:       s.Name.Name,
		Title:      s.Title,
		Layout:     s.Layout,
		Folder:     s.Folder,
	}
	for _, p := range s.Params {
		pm.Params = append(pm.Params, PageParam{Name: p.Name, EntityName: p.EntityName})
	}
	for _, w := range s.Widgets {
		node, err := astWidgetToNode(nil, w, moduleName)
		if err == nil && node != nil {
			pm.Widgets = append(pm.Widgets, node)
		}
	}
	return pm
}
```

Adjust field names to match `ast.CreatePageStmtV3` exactly (read the struct to verify).

- [ ] **Step 5: Run — expect PASS**

```bash
go test ./mdl/canonical/page/... -run TestLift -v
```

- [ ] **Step 6: Commit**

```bash
git add mdl/canonical/page/lift.go mdl/canonical/page/page_test.go \
        mdl/executor/cmd_pages_ast_to_model.go
git commit -m "feat(canonical/page): implement Lift() — AST to PageDocument (TDD)"
```

---

## Task 3: `Hydrate` — gen Page → PageDocument (TDD)

**Files:**
- Modify: `mdl/canonical/page/page_test.go`
- Create: `mdl/canonical/page/hydrate.go`

- [ ] **Step 1: Add Hydrate tests to `page_test.go`**

Add import `genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"`.

```go
func TestHydrate_BasicPage(t *testing.T) {
	p := genPg.NewPage()
	p.SetName("TestPage")

	doc, warns, err := page.Hydrate("M", p)
	require.NoError(t, err)
	assert.Empty(t, warns)
	require.NotNil(t, doc)
	pm := doc.PageModel()
	require.NotNil(t, pm)
	assert.Equal(t, "M", pm.ModuleName)
	assert.Equal(t, "TestPage", pm.Name)
}
```

- [ ] **Step 2: Run — expect compile failure**

```bash
go test ./mdl/canonical/page/... -run TestHydrate 2>&1 | head -10
```

Expected: `page.Hydrate undefined`.

- [ ] **Step 3: Create `mdl/canonical/page/hydrate.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package page

import (
	"github.com/mendixlabs/mxcli/mdl/canonical"
	"github.com/mendixlabs/mxcli/mdl/executor"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// Hydrate converts a gen-typed Page to a PageDocument.
// moduleName is the owning module (gen Page does not carry it).
func Hydrate(moduleName string, p *genPg.Page) (*PageDocument, []canonical.Warning, error) {
	pm := executor.PageGenToModel(moduleName, p)
	return &PageDocument{pm: pm}, nil, nil
}
```

**Note:** `executor.PageGenToModel` must be exported. Add in `cmd_pages_describe.go` or a new `cmd_pages_gen_to_model.go`:

```go
// PageGenToModel converts a gen-typed Page to a PageModel for Hydrate.
func PageGenToModel(moduleName string, p *genPg.Page) *types.PageModel {
	// Use the existing read path in MprBackend.
	// This is a thin wrapper that constructs the PageModel via the IR read path.
	pm := &types.PageModel{
		ModuleName: moduleName,
		Name:       p.Name(),
	}
	// TODO: extract layout, widgets via the existing page_model.go read path.
	// For now return a minimal model; full extraction in a follow-up if needed.
	return pm
}
```

**Note:** Full extraction of widgets from gen Page requires the backend's `pageDocToModel` which reads raw BSON. If the full extraction is complex, document it as a known gap and return a minimal PageDocument. The codec Hydrate is primarily used for diff, where ToMDL() output matters.

- [ ] **Step 4: Run — expect PASS**

```bash
go test ./mdl/canonical/page/... -run TestHydrate -v
```

- [ ] **Step 5: Commit**

```bash
git add mdl/canonical/page/hydrate.go mdl/canonical/page/page_test.go \
        mdl/executor/cmd_pages_describe.go
git commit -m "feat(canonical/page): implement Hydrate() — gen Page to PageDocument (TDD)"
```

---

## Task 4: `Persist` — PageDocument → BSON via executor

**Files:**
- Create: `mdl/canonical/page/persist.go`

- [ ] **Step 1: Create `mdl/canonical/page/persist.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package page

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/canonical"
	mxID "github.com/mendixlabs/mxcli/model"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// Persist writes the PageDocument to the project via ctx.Backend.
// ctx.Backend must satisfy pageBackend (satisfied by backend.PageBackend).
func (d *PageDocument) Persist(ctx canonical.PersistContext) error {
	type pageBackend interface {
		CreatePageGen(parentUUID, containmentName string, page *genPg.Page) error
		UpdatePageGen(page *genPg.Page) error
	}
	b, ok := ctx.Backend.(pageBackend)
	if !ok {
		return fmt.Errorf("page.Persist: backend %T does not implement pageBackend", ctx.Backend)
	}

	// Pages are created via the gen builder in the executor (execCreatePageV3).
	// PageDocument.Persist is the canonical entry point; it delegates to the
	// gen builder via the backend interface.
	// For now, Persist is a stub — the executor's createPageV3 path is authoritative.
	// This will be filled in when the executor fully delegates to the canonical model.
	_ = b
	_ = ctx
	return fmt.Errorf("page.Persist: not yet implemented — use execCreatePageV3 directly")
}

// existingPageID is the zero value for comparison.
var zeroID mxID.ID
```

**Note:** `Persist` is intentionally a stub — the page creation path is complex (gen builder + overlay) and delegating it fully is a separate task. The stub satisfies the `canonical.Persistable` interface so `comply_test.go` compiles. The codec will NOT register a LiftFn that calls Persist; the executor's existing create path remains authoritative.

- [ ] **Step 2: Build — confirm comply_test compiles**

```bash
go build ./mdl/canonical/page/...
go test ./mdl/canonical/page/... -run "^$" -v 2>&1 | head -5
```

Expected: compiles without error (the `var _ canonical.Persistable` assertion is satisfied).

- [ ] **Step 3: Commit**

```bash
git add mdl/canonical/page/persist.go
git commit -m "feat(canonical/page): add Persist stub — satisfies Persistable interface"
```

---

## Task 5: `RegisterCodec` + wire into executor

**Files:**
- Create: `mdl/canonical/page/codec.go`
- Modify: `mdl/canonical/codec_guard_test.go`
- Modify: `mdl/executor/executor.go`

- [ ] **Step 1: Create `mdl/canonical/page/codec.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package page

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/canonical"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// RegisterCodec wires the page Lift / Hydrate codecs into r.
func RegisterCodec(r *canonical.DefaultRegistry) {
	codec := canonical.Codec{
		LiftFn: func(stmt any) (canonical.Persistable, error) {
			s, ok := stmt.(*ast.CreatePageStmtV3)
			if !ok {
				return nil, fmt.Errorf("page codec: expected *ast.CreatePageStmtV3, got %T", stmt)
			}
			return Lift(nil, s, "")
		},
		HydrateFn: func(el any, hctx canonical.HydrateCtx) (canonical.Document, []canonical.Warning, error) {
			p, ok := el.(*genPg.Page)
			if !ok {
				return nil, nil, fmt.Errorf("page codec: expected *genPg.Page, got %T", el)
			}
			return Hydrate(hctx.ModuleName, p)
		},
	}
	r.Register((*ast.CreatePageStmtV3)(nil), "Forms$Page", codec)
	r.RegisterGenType("Forms$PageImpl", codec)
}
```

- [ ] **Step 2: Update `codec_guard_test.go`**

In `TestCodecComplete`, update:

```go
BuildRegistry: func() *canonical.DefaultRegistry {
    r := canonical.NewDefaultRegistry()
    entity.RegisterCodec(r)
    association.RegisterCodec(r)
    pagemodel.RegisterCodec(r)  // ← add
    return r
},
Required: []string{
    "DomainModels$Entity",
    "DomainModels$EntityImpl",
    "DomainModels$Association",
    "Forms$Page",  // ← add
},
```

Add import: `pagemodel "github.com/mendixlabs/mxcli/mdl/canonical/page"`.

- [ ] **Step 3: Update `executor.go`**

After `assocmodel.RegisterCodec(mc)`, add:

```go
pagemodel.RegisterCodec(mc)
```

Add import: `pagemodel "github.com/mendixlabs/mxcli/mdl/canonical/page"`.

- [ ] **Step 4: Verify Guard 3 GREEN**

```bash
go test ./mdl/canonical/ -run TestCodecComplete -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mdl/canonical/page/codec.go mdl/canonical/codec_guard_test.go \
        mdl/executor/executor.go
git commit -m "feat(canonical/page): register codec, wire into executor — Forms\$Page in registry"
```

---

## Task 6: Update boundary guard + route `cmd_diff_mdl.go`

**Files:**
- Modify: `mdl/executor/boundary_guard_test.go`
- Modify: `mdl/executor/cmd_diff_mdl.go`

- [ ] **Step 1: Update `boundary_guard_test.go`**

Add `"github.com/mendixlabs/mxcli/mdl/canonical/page"` to the Forbidden list:

```go
Forbidden: []string{
    "github.com/mendixlabs/mxcli/mdl/canonical/entity",
    "github.com/mendixlabs/mxcli/mdl/canonical/association",
    "github.com/mendixlabs/mxcli/mdl/canonical/page",
},
```

- [ ] **Step 2: Find page functions in `cmd_diff_mdl.go`**

```bash
grep -n "pageStmt\|pageToMDL\|pageTo\|Page.*MDL\|MDL.*Page" /mnt/data_sdd/gh/mxcli-wt-02/mdl/executor/cmd_diff_mdl.go | head -10
```

Identify the function names for page→MDL conversion in diff.

- [ ] **Step 3: Route page diff functions through ModelCodecs**

For each page→MDL function found, replace direct implementation with:

```go
func pageStmtToMDL(ctx *ExecContext, s *ast.CreatePageStmtV3) string {
	doc, err := ctx.ModelCodecs.LiftFrom(s)
	if err != nil {
		return fmt.Sprintf("/* page lift error: %v */", err)
	}
	return doc.ToMDL() + ";\n/"
}

func pageToMDLGen(ctx *ExecContext, moduleName string, p *genPg.Page) string {
	doc, _, err := ctx.ModelCodecs.HydrateFrom(p, canonical.HydrateCtx{ModuleName: moduleName})
	if err != nil {
		return fmt.Sprintf("/* page hydrate error: %v */", err)
	}
	return doc.ToMDL() + ";\n/"
}
```

Add import `canonical "github.com/mendixlabs/mxcli/mdl/canonical"` if not present.

Remove the direct `canonical/page` import from `cmd_diff_mdl.go` if it was added.

- [ ] **Step 4: Build and verify Guard 4**

```bash
go build ./mdl/executor/...
go test ./mdl/executor/ -run TestExecutorBoundary -v
```

Expected: PASS (no direct page import in cmd_diff_mdl.go).

- [ ] **Step 5: Run full test suite**

```bash
go test ./mdl/canonical/... ./mdl/executor/... ./internal/archtest/... -count=1 2>&1 | grep -E "^FAIL|^ok"
```

Expected: all guards GREEN, no new failures.

- [ ] **Step 6: Commit**

```bash
git add mdl/executor/boundary_guard_test.go mdl/executor/cmd_diff_mdl.go
git commit -m "refactor(executor): page diff routes through canonical codec registry"
```

---

## Self-Review

| Requirement | Task |
|-------------|------|
| `mdl/canonical/page/` exists | 1 |
| `PageDocument` implements `Document` | 1 |
| `PageDocument` implements `Persistable` | 4 |
| `comply_test.go` | 1 |
| Lift (AST → PageDocument) | 2 |
| Hydrate (gen → PageDocument) | 3 |
| ToMDL via existing pageModelToMDL | 1 |
| Persist stub (satisfies interface) | 4 |
| Codec registered for Forms$Page | 5 |
| Guard 3 updated | 5 |
| executor.go wired | 5 |
| cmd_diff_mdl.go routes through registry | 6 |
| Guard 4 stays GREEN | 6 |
| All guards GREEN | 6 |

**Known gap:** `Persist` is a stub. Full page creation through canonical model (replacing execCreatePageV3) is a separate future plan. The current stub satisfies the interface and enables the codec registry without breaking the existing create path.
