# Page Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enable the page overlay path for progressively more widget kinds so that DESCRIBE PAGE output becomes re-executable MDL. Starting point: fix the unconditional `return true` gate in `pageModelHasLossyWidget` and repair widget serialization one kind at a time, verified with `mx check` after each task.

**Architecture:** The page read/write path uses `types.PageModel` as an intermediate representation (IR). On write, `pageBuilder` (gen-typed) creates rich BSON; the IR overlay (`WritePageModel`) optionally rewrites the widget tree from the simpler IR. On read, `describePage` uses the IR for widget kinds that round-trip safely, and falls back to BSON inspection for lossy kinds. Each task fixes one widget kind's IR serialization in `mpr/page_model.go` (or equivalent), enables it in `widgetTreeHasLossyKind`, and verifies with `mx check`.

**Strategy:** Unlike entity/association, page does NOT follow Lift/Hydrate/ToMDL/Persist. The page IR overlay approach is the correct architecture. The goal is to make the overlay functional — widget by widget — not to redesign the page layer.

**Prerequisite:** `2026-06-02-entity-canonical-completion.md` complete (not blocking for page, but ensures a stable codebase to build on).

**Testing:** Every task must pass `mx check testdata/expr-checker/minimal.mpr` and `mx check testdata/corpus-b/app.mpr` with no new `StorageLoadException` errors. Run `git restore testdata/` after each integration test.

---

## File Map

**Key files (pre-existing):**

| File | Role |
|------|------|
| `mdl/executor/cmd_pages_create_v3.go` | Write gate: `pageModelHasLossyWidget`, `widgetTreeHasLossyKind` |
| `mdl/executor/cmd_pages_describe.go` | Read gate: `pageModelHasLossyWidgetReadOnly`, `describePage` |
| `mdl/backend/mpr/page_model.go` | IR ↔ BSON serialization: `widgetToBSON`, `fromBSON` |
| `mdl/types/page.go` | `PageModel`, `WidgetNode`, widget kind constants |

**Each task primarily modifies `page_model.go` and the gate functions.**

---

## Task 1: Fix the write gate — enable overlay for genuinely safe widgets

Currently `pageModelHasLossyWidget` always returns `true`, blocking all overlay. This task changes it to delegate to `widgetTreeHasLossyKind` (which already has correct per-kind logic), and adds a new guard test to prevent regression.

**Files:**
- Modify: `mdl/executor/cmd_pages_create_v3.go`

- [ ] **Step 1: Fix `pageModelHasLossyWidget`**

Replace the current always-true stub:

```go
func pageModelHasLossyWidget(pm *types.PageModel) bool {
	if pm == nil {
		return false
	}
	for _, n := range pm.Widgets {
		if widgetTreeHasLossyKind(n) {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Run build and baseline tests**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go build ./...
go test ./mdl/executor/... -count=1 2>&1 | grep -E "^--- FAIL|^FAIL|^ok" | head -20
```

Note any newly-failing tests — these are the widgets that now trigger overlay but fail.

- [ ] **Step 3: Baseline mx check**

```bash
./bin/mxcli -p testdata/expr-checker/minimal.mpr -c "create or modify persistent entity MyFirstModule.PageMigTest (Name: String);"
~/.mxcli/mxbuild/11.6.6/modeler/mx check testdata/expr-checker/minimal.mpr 2>&1 | grep -c "StorageLoadException"
git restore testdata/expr-checker/
```

Expected: 0 StorageLoadException errors. Record the baseline count.

- [ ] **Step 4: Add Guard — PageOverlayNeverTrue**

Add to `mdl/executor/cmd_pages_create_v3.go` test file (create `mdl/executor/page_overlay_guard_test.go`):

```go
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"
)

// TestPageOverlayGateNotAlwaysTrue verifies that pageModelHasLossyWidget is not
// the always-true stub. The stub (return true) prevented ALL overlay ever firing;
// this test ensures it was removed.
func TestPageOverlayGateNotAlwaysTrue(t *testing.T) {
	// A page model with NO widgets must not be considered lossy.
	pm := &types.PageModel{
		ModuleName: "M",
		Name:       "EmptyPage",
		Widgets:    nil,
	}
	if pageModelHasLossyWidget(pm) {
		t.Error("pageModelHasLossyWidget(emptyPage) = true; the always-true stub has not been removed")
	}
}
```

Add import `"github.com/mendixlabs/mxcli/mdl/types"`.

- [ ] **Step 5: Run guard test**

```bash
go test ./mdl/executor/ -run TestPageOverlayGateNotAlwaysTrue -v
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add mdl/executor/cmd_pages_create_v3.go mdl/executor/page_overlay_guard_test.go
git commit -m "fix(pages): enable overlay gate — pageModelHasLossyWidget now delegates to widgetTreeHasLossyKind"
```

---

## Task 2: Fix DataView overlay (most common widget, largest impact)

DataView is currently lossy because `dataSourceToBSON` cannot reproduce `Forms$DataViewSource` with `SourceVariable` + `PageVariable`. This task implements proper DataView source serialization.

**Files:**
- Modify: `mdl/backend/mpr/page_model.go` (or wherever `widgetToBSON`/`dataSourceToBSON` lives)
- Modify: `mdl/executor/cmd_pages_create_v3.go` (`widgetTreeHasLossyKind` — remove DataView case)

- [ ] **Step 1: Read `widgetToBSON` for DataView in `page_model.go`**

```bash
grep -n "DataView\|dataSource\|DataViewSource\|SourceVariable" /mnt/data_sdd/gh/mxcli-wt-02/mdl/backend/mpr/page_model.go | head -30
```

Identify the exact section that builds DataView BSON and what's missing.

- [ ] **Step 2: Write a failing roundtrip test**

In `mdl/executor/` find or create `roundtrip_page_test.go`. Add:

```go
func TestRoundtrip_DataView_EntitySource(t *testing.T) {
    // Create a page with a DataView bound to an entity, then describe it,
    // then check the description contains a valid create page statement.
    // Run with: go test ./mdl/executor/ -run TestRoundtrip_DataView_EntitySource -v -tags integration
}
```

This test should:
1. Create a test page with a DataView widget using a database entity source
2. Describe it via `describePage`
3. Verify the output parses cleanly (no `/* unknown */` fragments)

Implement the test body based on `mdl/executor/roundtrip_entity_test.go` as a pattern reference.

- [ ] **Step 3: Implement DataView source BSON serialization in `page_model.go`**

Find `dataSourceToBSON` (or equivalent). Add/fix the case for `DataSourceDatabase`/`DataSourceEntity`:

```go
case types.DataSourceDatabase, types.DataSourceEntity:
    // Forms$DataViewSource with EntityQualifiedName + XPathConstraint
    src := bson.D{
        {"$Type", "Forms$DataViewSource"},
        {"entityQualifiedName", ds.Entity},
    }
    if ds.XPathConstraint != "" {
        src = append(src, bson.E{Key: "xpathConstraint", Value: ds.XPathConstraint})
    }
    return src, nil
```

The exact BSON field names must be verified against a Studio Pro-generated MPR. Run:
```bash
./bin/mxcli -p testdata/corpus-b/app.mpr -c "describe page Administration.Account_Overview"
```
and inspect the output to confirm what field names the current implementation expects.

- [ ] **Step 4: Remove DataView from `widgetTreeHasLossyKind`**

In `cmd_pages_create_v3.go`, find the `widgetTreeHasLossyKind` function. Remove the `WidgetDataView` case:

```go
// REMOVE:
case types.WidgetDataView, types.WidgetListView:
    return true
```

Keep `WidgetListView` as lossy until Task 3.

- [ ] **Step 5: Build and mx check**

```bash
go build ./...
# Create a test page with DataView and verify mx check passes:
./bin/mxcli -p testdata/expr-checker/minimal.mpr -c "create or replace page MyFirstModule.TestDataView layout Atlas_Core.Atlas_Default () dataview db MyFirstModule.PageMigTest ();"
~/.mxcli/mxbuild/11.6.6/modeler/mx check testdata/expr-checker/minimal.mpr 2>&1 | grep -c "StorageLoadException"
git restore testdata/expr-checker/
```

Expected: 0 new StorageLoadException errors.

- [ ] **Step 6: Run full test suite**

```bash
go test ./mdl/executor/... -count=1 2>&1 | grep -E "^--- FAIL|^FAIL|^ok" | head -20
```

Fix any regressions before committing.

- [ ] **Step 7: Commit**

```bash
git add mdl/backend/mpr/page_model.go mdl/executor/cmd_pages_create_v3.go
git commit -m "fix(pages): DataView entity source overlay — remove DataView from lossy gate"
```

---

## Task 3: Fix Button overlay (second most common widget)

Buttons are lossy because action parameter mappings aren't fully captured in the IR's `OnClick` string. This task implements proper Button action serialization.

**Files:**
- Modify: `mdl/backend/mpr/page_model.go`
- Modify: `mdl/executor/cmd_pages_create_v3.go`

- [ ] **Step 1: Understand current Button BSON structure**

```bash
./bin/mxcli -p testdata/corpus-b/app.mpr -c "describe page Administration.Account_Overview" 2>&1 | grep -A5 "button\|Button"
grep -n "Button\|WidgetButton\|OnClick\|microflow" /mnt/data_sdd/gh/mxcli-wt-02/mdl/backend/mpr/page_model.go | head -30
```

- [ ] **Step 2: Extend `WidgetNode` Button properties if missing**

Check `mdl/types/page.go` for Button-specific fields. If `OnClick` is a bare `string` (microflow name only), extend to capture parameter mappings:

```go
// In types/page.go, if not already present:
type ButtonAction struct {
    Type       string // "microflow", "nanoflow", "page", "close"
    Microflow  string // qualified name
    Parameters []ButtonParam
}
type ButtonParam struct {
    Parameter string // parameter name
    Expression string // XPath expression
}
```

Update `WidgetNode` to use `*ButtonAction` instead of bare `OnClick string`.

- [ ] **Step 3: Implement Button action BSON serialization**

In `widgetToBSON` for `WidgetButton`, build the correct `Forms$MicroflowClientAction` or `Forms$NanoflowClientAction` BSON including parameter mappings.

Verify field names from a Studio Pro MPR:
```bash
./bin/mxcli -p testdata/corpus-b/app.mpr -c "describe page Administration.Account_NewEdit"
```

- [ ] **Step 4: Remove Button from `widgetTreeHasLossyKind`**

Remove `types.WidgetButton` from the lossy cases.

- [ ] **Step 5: Build + mx check + commit**

```bash
go build ./...
# Create page with button and verify:
~/.mxcli/mxbuild/11.6.6/modeler/mx check testdata/expr-checker/minimal.mpr 2>&1 | grep -c "StorageLoadException"
git restore testdata/expr-checker/
go test ./mdl/executor/... -count=1 2>&1 | grep -E "^--- FAIL|^FAIL|^ok" | head -10
git add mdl/backend/mpr/page_model.go mdl/types/page.go mdl/executor/cmd_pages_create_v3.go
git commit -m "fix(pages): Button action overlay — parameter mappings captured in IR"
```

---

## Task 4: Fix CheckBox + RadioButtons

**Files:**
- Modify: `mdl/backend/mpr/page_model.go`
- Modify: `mdl/executor/cmd_pages_create_v3.go`
- Modify: `mdl/executor/cmd_pages_describe_output.go` (add renderWidget cases)

- [ ] **Step 1: Add missing `renderWidget` cases**

In `cmd_pages_describe_output.go` (or wherever `renderWidget` is defined), add cases for `WidgetCheckBox` and `WidgetRadioButtons`:

```go
case types.WidgetCheckBox:
    fmt.Fprintf(w, "%scheckbox %s attribute %s\n", indent, n.Name, n.EntityAttr)
case types.WidgetRadioButtons:
    fmt.Fprintf(w, "%sradiobuttons %s attribute %s\n", indent, n.Name, n.EntityAttr)
```

Adjust field names to match actual `WidgetNode` structure.

- [ ] **Step 2: Implement BSON serialization for these widget kinds**

In `widgetToBSON`, add cases for `WidgetCheckBox` and `WidgetRadioButtons` following the same pattern as existing simple attribute widgets.

- [ ] **Step 3: Remove from lossy gate + build + verify**

```bash
go build ./...
~/.mxcli/mxbuild/11.6.6/modeler/mx check testdata/expr-checker/minimal.mpr 2>&1 | grep -c "StorageLoadException"
git restore testdata/expr-checker/
go test ./mdl/executor/... -count=1 2>&1 | grep -E "^--- FAIL|^FAIL|^ok" | head -10
git add mdl/backend/mpr/page_model.go mdl/executor/cmd_pages_describe_output.go mdl/executor/cmd_pages_create_v3.go
git commit -m "fix(pages): CheckBox + RadioButtons overlay enabled"
```

---

## Task 5: Fix ScrollContainer (CenterRegion children)

ScrollContainer children live in `CenterRegion.Widgets`, not in the top-level `Widgets` array. The `fromBSON` extraction reads the wrong field.

**Files:**
- Modify: `mdl/backend/mpr/page_model.go`
- Modify: `mdl/executor/cmd_pages_create_v3.go`

- [ ] **Step 1: Find the ScrollContainer fromBSON extraction**

```bash
grep -n "ScrollContainer\|CenterRegion\|scrollView\|ScrollView" /mnt/data_sdd/gh/mxcli-wt-02/mdl/backend/mpr/page_model.go | head -20
```

- [ ] **Step 2: Fix `fromBSON` to read `CenterRegion.Widgets`**

For ScrollContainer, the BSON structure is:
```
Forms$ScrollContainer {
  regions: [
    Forms$ScrollContainerRegion {
      position: "Center",
      widgets: [...]
    }
  ]
}
```

Update `fromBSON` to extract widgets from the Center region:

```go
case "Forms$ScrollContainer":
    var children []*types.WidgetNode
    for _, region := range getArray(doc, "regions") {
        if getString(region, "position") == "Center" {
            for _, w := range getArray(region, "widgets") {
                if child := fromBSON(w); child != nil {
                    children = append(children, child)
                }
            }
        }
    }
    node.Kind = types.WidgetScrollView
    node.Children = children
```

- [ ] **Step 3: Fix `widgetToBSON` for ScrollContainer**

Ensure the write path puts children in `CenterRegion.Widgets`:

```go
case types.WidgetScrollView:
    regions := bson.A{}
    childBSONs := bson.A{}
    for _, c := range n.Children {
        cb, err := widgetToBSON(c)
        if err != nil {
            return nil, err
        }
        childBSONs = append(childBSONs, cb)
    }
    regions = append(regions, bson.D{
        {"$Type", "Forms$ScrollContainerRegion"},
        {"position", "Center"},
        {"widgets", childBSONs},
    })
    return bson.D{
        {"$Type", "Forms$ScrollContainer"},
        {"name", n.Name},
        {"regions", regions},
    }, nil
```

- [ ] **Step 4: Remove ScrollView from lossy gate + build + verify**

```bash
go build ./...
~/.mxcli/mxbuild/11.6.6/modeler/mx check testdata/expr-checker/minimal.mpr 2>&1 | grep -c "StorageLoadException"
git restore testdata/expr-checker/
go test ./mdl/executor/... -count=1 2>&1 | grep -E "^--- FAIL|^FAIL|^ok" | head -10
git add mdl/backend/mpr/page_model.go mdl/executor/cmd_pages_create_v3.go
git commit -m "fix(pages): ScrollContainer reads/writes CenterRegion children correctly"
```

---

## Task 6: Verify overlay coverage — update guard

After Tasks 1-5, the only remaining lossy widget kinds should be DataGrid, Gallery, ComboBox, Image, ListView (pluggable widgets requiring template extraction). Add a coverage guard to document what's still lossy.

**Files:**
- Modify: `mdl/executor/page_overlay_guard_test.go`

- [ ] **Step 1: Add expected-lossy documentation test**

```go
// TestPageLossyWidgets_DocumentedGap documents which widget kinds remain
// lossy after Phase P Tasks 1-5. Each entry here is a known gap to be
// addressed in a future task. When a kind is fixed, remove it from this list.
func TestPageLossyWidgets_DocumentedGap(t *testing.T) {
    stillLossy := []types.WidgetKind{
        types.WidgetDataGrid,    // pluggable: needs DataGrid2 template + column schema
        types.WidgetGallery,     // pluggable: needs Gallery template
        types.WidgetComboBox,    // pluggable: needs ComboBox template
        types.WidgetImage,       // image reference resolution
        types.WidgetListView,    // ListView data source complex
        types.WidgetUnknown,     // unknown widget kinds always lossy
    }
    for _, kind := range stillLossy {
        n := &types.WidgetNode{Kind: kind}
        if !widgetTreeHasLossyKind(n) {
            t.Errorf("WidgetKind %q should still be lossy but widgetTreeHasLossyKind returned false — either the kind was fixed (remove from this list) or the gate was accidentally broken", kind)
        }
    }
}
```

- [ ] **Step 2: Run test**

```bash
go test ./mdl/executor/ -run TestPageLossyWidgets -v
```

Expected: PASS. If any kind unexpectedly returns false, either the fix was too broad or the kind was genuinely fixed (remove from the documented gap list).

- [ ] **Step 3: Run full final verification**

```bash
go build ./...
go test ./mdl/executor/... ./mdl/model/... ./internal/archtest/... -count=1 2>&1 | grep -E "^FAIL|^ok"
```

Expected: no FAIL lines except the two architectural guards (Guard 1 and Guard 4) which remain RED until the entity completion plan is also run.

- [ ] **Step 4: Commit**

```bash
git add mdl/executor/page_overlay_guard_test.go
git commit -m "test(pages): document remaining lossy widget kinds as known gaps"
```

---

## Self-Review

| Requirement | Task |
|-------------|------|
| pageModelHasLossyWidget no longer always-true | 1 |
| Guard: overlay gate not stuck | 1 |
| DataView entity source overlay working | 2 |
| Button action (incl. parameters) overlay working | 3 |
| CheckBox + RadioButtons overlay working | 4 |
| ScrollContainer CenterRegion children correct | 5 |
| Remaining lossy kinds documented | 6 |
| mx check passes after each task | all |

**Pluggable widgets (DataGrid, Gallery, ComboBox) are NOT covered by this plan.** Each requires template extraction from a Studio Pro baseline project and a separate spec. They remain in the `widgetTreeHasLossyKind` lossy list.
