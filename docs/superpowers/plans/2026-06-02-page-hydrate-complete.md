# Page Canonical Model — Hydrate Completion Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `Hydrate` produce a complete `PageDocument` (including widget tree), then wire `describePage` to route through the canonical model, completing the page migration.

**Architecture:** Export `PageGenToModel(moduleName, *genPg.Page) (*types.PageModel, error)` from `mdl/backend/mpr/` — it encodes the gen Page to BSON using `codec.Encoder` and passes the result to the existing `pageDocToModel`. `canonical/page.Hydrate` calls `PageGenToModel` (safe: `backend/mpr` does not import `canonical`). `describePage` calls the new `hydratePageModel` helper in `executor.go` (which type-asserts through the registry), replacing the current direct `ctx.Backend.GetPageModel` call. The legacy raw-BSON fallback path is preserved for pages with lossy widgets.

**Tech Stack:** Go 1.26, `go.mongodb.org/mongo-driver/v2/bson`, `modelsdk/codec`, `modelsdk/gen/pages`.

**Prerequisite:** Existing `mdl/canonical/page/` package (commit 9f16cf26).

---

## File Map

**New files:**

| File | Responsibility |
|------|----------------|
| `mdl/backend/mpr/page_gen_to_model.go` | `PageGenToModel` — gen Page → BSON → pageDocToModel |
| `mdl/backend/mpr/page_gen_to_model_test.go` | Unit test for PageGenToModel |

**Modified files:**

| File | Change |
|------|--------|
| `mdl/canonical/page/hydrate.go` | Call `mprbackend.PageGenToModel`; replace metadata-only stub |
| `mdl/canonical/page/page_test.go` | Add `TestHydrate_ExtractsWidgets` |
| `mdl/executor/executor.go` | Add `hydratePageModel` helper |
| `mdl/executor/cmd_pages_describe.go` | Replace `ctx.Backend.GetPageModel(pageID)` with `hydratePageModel` |

---

## Task 1: Export `PageGenToModel` from `mdl/backend/mpr` (TDD)

**Files:**
- Create: `mdl/backend/mpr/page_gen_to_model.go`
- Create: `mdl/backend/mpr/page_gen_to_model_test.go`

- [ ] **Step 1: Write failing test**

```go
// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/types"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPageGenToModel_ExtractsName(t *testing.T) {
	p := genPg.NewPage()
	p.SetName("TestPage")

	pm, err := PageGenToModel("M", p)
	require.NoError(t, err)
	require.NotNil(t, pm)
	assert.Equal(t, "M", pm.ModuleName)
	assert.Equal(t, "TestPage", pm.Name)
}

func TestPageGenToModel_NilPage(t *testing.T) {
	_, err := PageGenToModel("M", nil)
	assert.Error(t, err)
}

func TestPageGenToModel_ReturnsPageModel(t *testing.T) {
	p := genPg.NewPage()
	p.SetName("P")
	pm, err := PageGenToModel("Mod", p)
	require.NoError(t, err)
	// pm is not nil even for a minimal page
	assert.IsType(t, &types.PageModel{}, pm)
}
```

Save to `mdl/backend/mpr/page_gen_to_model_test.go`.

- [ ] **Step 2: Run tests — expect compile failure**

```bash
go test ./mdl/backend/mpr/ -run TestPageGenToModel 2>&1 | head -10
```

Expected: `PageGenToModel undefined`.

- [ ] **Step 3: Create `mdl/backend/mpr/page_gen_to_model.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/modelsdk/codec"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// PageGenToModel converts a gen-typed Page to a *types.PageModel by encoding
// it to BSON with the gen codec and passing the result through pageDocToModel.
//
// This enables canonical/page.Hydrate to produce a complete PageModel (including
// widget tree) without importing the executor package (which would create a cycle).
// mdl/backend/mpr does not import mdl/canonical, so canonical/page → backend/mpr
// is safe.
func PageGenToModel(moduleName string, p *genPg.Page) (*types.PageModel, error) {
	if p == nil {
		return nil, fmt.Errorf("PageGenToModel: nil page")
	}

	// Encode the gen Page to raw BSON bytes using the gen codec.
	enc := codec.Encoder{}
	raw, err := enc.Encode(p)
	if err != nil {
		return nil, fmt.Errorf("PageGenToModel: encode: %w", err)
	}

	// Unmarshal to bson.D so pageDocToModel can process it.
	var doc bson.D
	if err := bson.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("PageGenToModel: unmarshal: %w", err)
	}

	pm := pageDocToModel(doc)
	if pm == nil {
		pm = &types.PageModel{}
	}
	// Overlay caller-supplied metadata that the gen Page doesn't store.
	pm.ModuleName = moduleName
	pm.Name = p.Name()
	return pm, nil
}
```

- [ ] **Step 4: Run tests — expect pass**

```bash
go test ./mdl/backend/mpr/ -run TestPageGenToModel -v
```

Expected: all `TestPageGenToModel_*` tests PASS.

- [ ] **Step 5: Verify build**

```bash
go build ./mdl/backend/mpr/...
```

- [ ] **Step 6: Commit**

```bash
git add mdl/backend/mpr/page_gen_to_model.go mdl/backend/mpr/page_gen_to_model_test.go
git commit -m "feat(mpr): export PageGenToModel — gen Page to PageModel via BSON encode+decode"
```

---

## Task 2: Fix `Hydrate` to call `PageGenToModel` (TDD)

**Files:**
- Modify: `mdl/canonical/page/hydrate.go`
- Modify: `mdl/canonical/page/page_test.go`

- [ ] **Step 1: Add failing test to `page_test.go`**

Append to `mdl/canonical/page/page_test.go` (add import `genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"`):

```go
func TestHydrate_ExtractsWidgets(t *testing.T) {
	// Build a gen Page with a LayoutCall containing a widget.
	lc := genPg.NewLayoutCall()
	lc.SetLayoutQualifiedName("Atlas_Core.Atlas_Default")
	arg := genPg.NewLayoutCallArgument()
	arg.SetTypeName("Forms$FormCallArgument")
	arg.SetParameterQualifiedName("Content")
	// Add a simple DivContainer widget so the widget array is non-empty.
	container := genPg.NewDivContainer()
	container.SetName("ctnMain")
	arg.AddWidgets(container)
	lc.AddArguments(arg)

	p := genPg.NewPage()
	p.SetName("WithWidgets")
	p.SetLayoutCall(lc)

	doc, warns, err := Hydrate("M", p)
	require.NoError(t, err)
	// Warnings about layout are expected if LayoutCall type-assert fails for
	// the test gen types; widget extraction via BSON path is what matters.
	_ = warns
	require.NotNil(t, doc)
	pm := doc.PageModel()
	require.NotNil(t, pm)
	assert.Equal(t, "M", pm.ModuleName)
	assert.Equal(t, "WithWidgets", pm.Name)
	// The widget tree should now be non-empty (via PageGenToModel).
	assert.NotEmpty(t, pm.Widgets, "Hydrate should produce a non-empty widget tree")
}
```

- [ ] **Step 2: Run test — expect FAIL**

```bash
go test ./mdl/canonical/page/... -run TestHydrate_ExtractsWidgets -v 2>&1 | head -15
```

Expected: `pm.Widgets` is empty (current stub Hydrate doesn't extract widgets).

- [ ] **Step 3: Update `hydrate.go`**

Replace the entire `hydrate.go` file:

```go
// SPDX-License-Identifier: Apache-2.0

package page

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/canonical"
	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// Hydrate converts a gen-typed Page to a PageDocument with a complete widget tree.
//
// It calls backend/mpr.PageGenToModel which encodes the gen Page to BSON and
// passes it through the existing pageDocToModel read path. This produces the same
// widget tree as ctx.Backend.GetPageModel for pages that were created by the gen
// builder. moduleName is supplied by the caller because gen Page does not carry it.
//
// canonical/page → backend/mpr is safe: backend/mpr does not import canonical.
func Hydrate(moduleName string, p *genPg.Page) (*PageDocument, []canonical.Warning, error) {
	if p == nil {
		return nil, nil, fmt.Errorf("page.Hydrate: nil page")
	}
	pm, err := mprbackend.PageGenToModel(moduleName, p)
	if err != nil {
		return nil, []canonical.Warning{{
			Field:   "widgets",
			Message: fmt.Sprintf("PageGenToModel failed: %v", err),
		}}, nil
	}
	return &PageDocument{pm: pm}, nil, nil
}
```

- [ ] **Step 4: Run tests — expect pass**

```bash
go test ./mdl/canonical/page/... -v 2>&1 | grep -E "PASS|FAIL|RUN" | head -20
```

Expected: `TestHydrate_ExtractsWidgets` PASSES, all other Hydrate tests still PASS.

- [ ] **Step 5: Build**

```bash
go build ./mdl/canonical/page/...
```

- [ ] **Step 6: Commit**

```bash
git add mdl/canonical/page/hydrate.go mdl/canonical/page/page_test.go
git commit -m "feat(canonical/page): Hydrate now extracts full widget tree via PageGenToModel"
```

---

## Task 3: Add `hydratePageModel` helper to `executor.go`

**Files:**
- Modify: `mdl/executor/executor.go`

- [ ] **Step 1: Add `hydratePageModel` function to `executor.go`**

After the existing `hydrateEntityModel` function, add:

```go
// hydratePageModel hydrates a gen-typed Page through the codec registry and
// returns the underlying *types.PageModel. executor.go is the only executor
// file allowed to import mdl/canonical/page directly (see Guard 4).
func hydratePageModel(ctx *ExecContext, moduleName string, p any) (*types.PageModel, []canonical.Warning, error) {
	doc, warns, err := ctx.ModelCodecs.HydrateFrom(p, canonical.HydrateCtx{ModuleName: moduleName})
	if err != nil {
		return nil, warns, err
	}
	pd, ok := doc.(*pagemodel.PageDocument)
	if !ok {
		return nil, warns, fmt.Errorf("hydratePageModel: unexpected type %T", doc)
	}
	return pd.PageModel(), warns, nil
}
```

Note: `pagemodel` is already imported in `executor.go` as `pagemodel "github.com/mendixlabs/mxcli/mdl/canonical/page"`. The `types` package is also already imported.

- [ ] **Step 2: Build**

```bash
go build ./mdl/executor/...
```

- [ ] **Step 3: Commit**

```bash
git add mdl/executor/executor.go
git commit -m "feat(executor): add hydratePageModel helper — routes page Hydrate through ModelCodecs"
```

---

## Task 4: Wire `describePage` to use `hydratePageModel`

**Files:**
- Modify: `mdl/executor/cmd_pages_describe.go`

The current code (around line 129) calls:
```go
pm, pmErr := ctx.Backend.GetPageModel(pageID)
```

We replace each such call with the canonical path. Run first to find all occurrences:

```bash
grep -n "GetPageModel" /mnt/data_sdd/gh/mxcli-wt-02/mdl/executor/cmd_pages_describe.go
```

- [ ] **Step 1: Replace all `ctx.Backend.GetPageModel(pageID)` calls**

Find every pattern:
```go
pm, pmErr := ctx.Backend.GetPageModel(pageID)
```

Replace with:
```go
pm, _, pmErr := hydratePageModel(ctx, modName, foundPage)
```

Note: `hydratePageModel` returns `(*types.PageModel, []canonical.Warning, error)`. The warnings are discarded (`_`) for now since the existing describe path doesn't surface them.

Also find and replace the snippet-describe and layout-describe variants if they use the same pattern. Check:

```bash
grep -n "GetPageModel\|GetSnippetModel\|GetLayoutModel" /mnt/data_sdd/gh/mxcli-wt-02/mdl/executor/cmd_pages_describe.go
```

For snippets and layouts, skip them — only route page describe through canonical in this task.

- [ ] **Step 2: Build**

```bash
go build ./mdl/executor/...
```

Fix any compile errors (e.g., `pm, pmErr` → `pm, _, pmErr` signature mismatch).

- [ ] **Step 3: Run page describe tests**

```bash
go test ./mdl/executor/... -run "TestDescribePage\|TestRoundtripPage\|TestPage" -count=1 -v 2>&1 | grep -E "PASS|FAIL|RUN" | head -30
```

Expected: same pass/fail as before — the legacy fallback path covers lossy widgets so no regressions.

- [ ] **Step 4: Smoke test with real project**

```bash
./bin/mxcli -p testdata/corpus-b/app.mpr -c "describe page Administration.Account_Overview" 2>&1 | head -20
```

Expected: valid `create or modify page Administration.Account_Overview (...)` output with widget tree.

- [ ] **Step 5: Commit**

```bash
git add mdl/executor/cmd_pages_describe.go
git commit -m "refactor(executor): describePage routes page Hydrate through canonical ModelCodecs"
```

---

## Task 5: Verify all guards stay GREEN

- [ ] **Step 1: Run all architectural guards**

```bash
go test ./mdl/canonical/ -run "TestImportDirection|TestCodecComplete|TestPackageStructure" -v
go test ./mdl/executor/ -run "TestExecutorBoundary" -v
go test ./internal/archtest/... -count=1
```

Expected: all PASS. Specifically:
- `TestImportDirection`: PASS (backend/mpr not imported by canonical root)
- `TestCodecComplete`: PASS (Forms$Page still registered)
- `TestExecutorBoundary`: PASS (cmd_pages_describe.go does not import canonical/page; it calls hydratePageModel via executor.go)

- [ ] **Step 2: Run full canonical + executor test suites**

```bash
go test ./mdl/canonical/... ./mdl/backend/mpr/... -count=1 2>&1 | grep -E "^FAIL|^ok"
```

- [ ] **Step 3: Final commit if needed**

If any guard tests needed fixes, commit them now.

---

## Self-Review

| Requirement | Task |
|-------------|------|
| `PageGenToModel` exported from backend/mpr | 1 |
| `PageGenToModel` has unit tests | 1 |
| `Hydrate` calls `PageGenToModel` — widget tree populated | 2 |
| `TestHydrate_ExtractsWidgets` passes | 2 |
| `hydratePageModel` helper in executor.go | 3 |
| `describePage` routes through canonical | 4 |
| All 5 guards GREEN | 5 |

**What remains out of scope (known gaps):**
- `Persist` is still a stub — page creation remains via gen builder
- `execCreatePageV3` is not routed through canonical (would require working Persist)
- `cmd_diff_mdl.go` has no page diff functions (pages not in diff scope currently)
- `Hydrate` via gen codec may miss Studio Pro BSON fields not modeled in gen types
