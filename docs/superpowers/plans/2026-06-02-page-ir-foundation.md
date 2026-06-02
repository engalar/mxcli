# Page IR Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the `types.PageModel` / `WidgetNode` round-trip so that DataView column widths and footer widgets survive a write→describe cycle without data loss, enabling DataView to leave the lossy gate.

**Architecture:** Three targeted fixes: (1) `propInt` handles "AutoFill" → -1; (2) `WidgetNode` gains a `Footer` field, with `astWidgetToNode` and `widgetNodeFromBSON` extracting footer from DataView's `FooterWidgets` BSON array; (3) `widgetToBSON` writes `ShowFooter + FooterWidgets`. Validated end-to-end by golden rebuild. No canonical layer changes in this plan.

**Tech Stack:** Go 1.26, `go.mongodb.org/mongo-driver/v2/bson`, `modelsdk/gen/pages`.

**Prerequisite:** `2026-06-02-rename-model-to-canonical.md` complete.

**Root cause evidence:** See `docs/superpowers/specs/2026-06-02-canonical-guards-design.md` and systematic-debugging session 2026-06-02.

---

## File Map

**Modified files:**

| File | Change |
|------|--------|
| `mdl/executor/cmd_pages_ast_to_model.go` | `propInt` handles "AutoFill"→-1; DataView case re-partitions footer vs main children |
| `mdl/types/page.go` | `WidgetNode` gains `Footer []*WidgetNode` field |
| `mdl/backend/mpr/page_model.go` | `widgetNodeFromBSON`: extract `FooterWidgets`; `widgetToBSON`: write `ShowFooter` + `FooterWidgets` |
| `mdl/executor/cmd_pages_create_v3.go` | Remove DataView from `widgetTreeHasLossyKind` (re-enable overlay) |
| `mdl/executor/page_overlay_guard_test.go` | Move DataView from `stillLossy` to `enabled` list |
| `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` | Add doc comment (SOP requirement for golden rebuild) |

---

## Task 1: Fix `propInt` to handle "AutoFill" → -1

**Root cause:** `propInt` in `cmd_pages_ast_to_model.go` (line 263) has no `case string:` branch. When MDL contains `(TabletWidth: AutoFill, PhoneWidth: AutoFill)`, the parser produces the string `"AutoFill"` as the property value, `propInt` returns 0, and `widgetToBSON` writes `TabletWeight: 0`. Describe omits zero-weight column params.

**Files:**
- Modify: `mdl/executor/cmd_pages_ast_to_model.go`

- [ ] **Step 1: Write failing test**

In `mdl/executor/` find or create `cmd_pages_ast_to_model_test.go`:

```go
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

func TestPropInt_AutoFill(t *testing.T) {
	w := &ast.WidgetV3{
		Properties: map[string]any{
			"TabletWidth": "AutoFill",
			"PhoneWidth":  "autofill", // lowercase variant
		},
	}
	if got := propInt(w, "tabletwidth"); got != -1 {
		t.Errorf("propInt AutoFill: want -1, got %d", got)
	}
	if got := propInt(w, "phonewidth"); got != -1 {
		t.Errorf("propInt autofill (lowercase): want -1, got %d", got)
	}
}

func TestPropInt_NormalInt(t *testing.T) {
	w := &ast.WidgetV3{
		Properties: map[string]any{"DesktopWidth": 12},
	}
	if got := propInt(w, "desktopwidth"); got != 12 {
		t.Errorf("propInt int: want 12, got %d", got)
	}
}
```

- [ ] **Step 2: Run — expect FAIL**

```bash
go test ./mdl/executor/ -run TestPropInt -v 2>&1 | head -10
```

Expected: `TestPropInt_AutoFill` fails (propInt returns 0 for "AutoFill").

- [ ] **Step 3: Fix `propInt` in `cmd_pages_ast_to_model.go`**

Add `"strconv"` to imports. In the `propInt` function after the `case float64:` branch, add:

```go
case string:
    if strings.EqualFold(n, "AutoFill") {
        return -1
    }
    if i, err := strconv.Atoi(n); err == nil {
        return i
    }
```

- [ ] **Step 4: Run — expect PASS**

```bash
go test ./mdl/executor/ -run TestPropInt -v
```

Expected: both `TestPropInt_*` tests PASS.

- [ ] **Step 5: Verify column widths round-trip**

```bash
go test ./mdl/executor/ -run "TestRoundtrip\|TestDescribePage\|TestPage" -count=1 -v 2>&1 | grep -E "PASS|FAIL|RUN" | head -20
```

No new failures expected.

- [ ] **Step 6: Commit**

```bash
git add mdl/executor/cmd_pages_ast_to_model.go mdl/executor/cmd_pages_ast_to_model_test.go
git commit -m "fix(pages): propInt handles AutoFill→-1 — column widths survive AST→IR→BSON roundtrip"
```

---

## Task 2: Add `Footer` field to `WidgetNode`

**Files:**
- Modify: `mdl/types/page.go`

- [ ] **Step 1: Add `Footer []*WidgetNode` to `WidgetNode`**

In `mdl/types/page.go`, in the `WidgetNode` struct, add after the `Children` field:

```go
// Footer holds footer widgets (DataView only). Separate from Children
// because Mendix stores them in FooterWidgets in BSON, not Widgets.
Footer []*WidgetNode
```

- [ ] **Step 2: Build**

```bash
go build ./mdl/types/... ./mdl/executor/... ./mdl/backend/...
```

Expected: no errors (Footer is additive, zero-value is nil slice).

- [ ] **Step 3: Commit**

```bash
git add mdl/types/page.go
git commit -m "feat(types): add WidgetNode.Footer for DataView footer widgets"
```

---

## Task 3: Fix `astWidgetToNode` — separate DataView footer from main children (TDD)

**Root cause:** The general children loop (lines 62-70 in `cmd_pages_ast_to_model.go`) puts ALL DataView children — including `footer` type children — into `node.Children`. But BSON requires main widgets in `Widgets` and footer widgets in `FooterWidgets`. Re-partition in the DataView case.

**Files:**
- Modify: `mdl/executor/cmd_pages_ast_to_model.go`

- [ ] **Step 1: Write failing test**

Append to `cmd_pages_ast_to_model_test.go`:

```go
func TestAstWidgetToNode_DataView_FooterSeparated(t *testing.T) {
	footer := &ast.WidgetV3{
		Type: "footer",
		Name: "footer1",
		Children: []*ast.WidgetV3{
			{Type: "actionbutton", Name: "btnSave", Properties: map[string]any{"caption": "Save"}},
		},
	}
	dv := &ast.WidgetV3{
		Type: "dataview",
		Name: "dvMain",
		Properties: map[string]any{"datasource": (*ast.DataSourceV3)(nil)},
		Children: []*ast.WidgetV3{
			{Type: "textbox", Name: "tbName", Properties: map[string]any{"attribute": "Name"}},
			footer,
		},
	}

	node, err := astWidgetToNode(testExecCtx(), dv, "M")
	if err != nil {
		t.Fatalf("astWidgetToNode: %v", err)
	}
	if len(node.Children) != 1 {
		t.Errorf("want 1 main child (textbox), got %d: %v", len(node.Children), node.Children)
	}
	if node.Children[0].Name != "tbName" {
		t.Errorf("main child should be tbName, got %q", node.Children[0].Name)
	}
	if len(node.Footer) != 1 {
		t.Errorf("want 1 footer child, got %d", len(node.Footer))
	}
	if node.Footer[0].Name != "footer1" {
		t.Errorf("footer child should be footer1, got %q", node.Footer[0].Name)
	}
}

// testExecCtx returns a minimal *ExecContext for unit tests.
func testExecCtx() *ExecContext {
	ctx := newMinimalExecContext()
	return ctx
}
```

If `newMinimalExecContext` doesn't exist, use `&ExecContext{}` instead (astWidgetToNode only uses ctx for module lookup which can be nil).

- [ ] **Step 2: Run — expect FAIL**

```bash
go test ./mdl/executor/ -run TestAstWidgetToNode_DataView_FooterSeparated -v 2>&1 | head -15
```

Expected: footer children end up in `Children` instead of `Footer`.

- [ ] **Step 3: Fix DataView case in `astWidgetToNode`**

In `cmd_pages_ast_to_model.go`, replace the DataView case (around line 94-98):

```go
case types.WidgetDataView:
	node.DataSource = astDataSourceToModel(propDS(w, "datasource"))
	if node.DataSource != nil {
		node.EntityCtx = node.DataSource.Entity
	}
	// Re-partition children: footer type → node.Footer, others stay in node.Children.
	// The general loop above already processed children; redo for DataView so we
	// can separate main content (→ BSON Widgets) from footer (→ BSON FooterWidgets).
	node.Children = nil
	node.Footer = nil
	for _, child := range w.Children {
		cn, err := astWidgetToNode(ctx, child, moduleName)
		if err != nil || cn == nil {
			continue
		}
		if strings.EqualFold(child.Type, "footer") {
			node.Footer = append(node.Footer, cn)
		} else {
			node.Children = append(node.Children, cn)
		}
	}
```

- [ ] **Step 4: Run — expect PASS**

```bash
go test ./mdl/executor/ -run TestAstWidgetToNode_DataView_FooterSeparated -v
```

- [ ] **Step 5: Build**

```bash
go build ./mdl/executor/...
```

- [ ] **Step 6: Commit**

```bash
git add mdl/executor/cmd_pages_ast_to_model.go mdl/executor/cmd_pages_ast_to_model_test.go
git commit -m "fix(pages): astWidgetToNode separates DataView footer children into Footer field"
```

---

## Task 4: Fix `widgetNodeFromBSON` — extract DataView `FooterWidgets` (TDD)

**Files:**
- Modify: `mdl/backend/mpr/page_model.go`

- [ ] **Step 1: Write failing test**

In `mdl/backend/mpr/` find or create `page_model_test.go`:

```go
// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/types"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func makeFooterBSON(footerName string) bson.D {
	return bson.D{
		{Key: "$ID", Value: types.UUIDToBlob(types.GenerateID())},
		{Key: "$Type", Value: "Forms$DivContainer"},
		{Key: "Name", Value: footerName},
		{Key: "Appearance", Value: bson.D{
			{Key: "$ID", Value: types.UUIDToBlob(types.GenerateID())},
			{Key: "$Type", Value: "Forms$Appearance"},
		}},
	}
}

func TestWidgetNodeFromBSON_DataView_ExtractsFooter(t *testing.T) {
	footerBSON := makeFooterBSON("footer1")
	dvBSON := bson.D{
		{Key: "$ID", Value: types.UUIDToBlob(types.GenerateID())},
		{Key: "$Type", Value: "Forms$DataView"},
		{Key: "Name", Value: "dvMain"},
		{Key: "Appearance", Value: bson.D{
			{Key: "$ID", Value: types.UUIDToBlob(types.GenerateID())},
			{Key: "$Type", Value: "Forms$Appearance"},
		}},
		{Key: "ShowFooter", Value: true},
		{Key: "FooterWidgets", Value: bsonVersionedArray([]bson.D{footerBSON})},
	}

	node := widgetNodeFromBSON(dvBSON)
	if node == nil {
		t.Fatal("widgetNodeFromBSON returned nil for DataView")
	}
	if len(node.Footer) != 1 {
		t.Errorf("want 1 footer widget, got %d", len(node.Footer))
	}
	if node.Footer[0].Name != "footer1" {
		t.Errorf("footer widget name: want footer1, got %q", node.Footer[0].Name)
	}
}
```

- [ ] **Step 2: Run — expect FAIL**

```bash
go test ./mdl/backend/mpr/ -run TestWidgetNodeFromBSON_DataView_ExtractsFooter -v 2>&1 | head -10
```

Expected: `node.Footer` is nil (not extracted).

- [ ] **Step 3: Fix `widgetNodeFromBSON` DataView case in `page_model.go`**

Replace the DataView case (around line 219-222):

```go
case types.WidgetDataView:
	node.DataSource = extractBSONDataSource(doc)
	node.EntityCtx = extractDataViewEntityCtx(doc)
	node.Children = extractChildWidgets(doc, "Widgets")
	node.Footer = extractChildWidgets(doc, "FooterWidgets")
```

- [ ] **Step 4: Run — expect PASS**

```bash
go test ./mdl/backend/mpr/ -run TestWidgetNodeFromBSON_DataView_ExtractsFooter -v
```

- [ ] **Step 5: Commit**

```bash
git add mdl/backend/mpr/page_model.go mdl/backend/mpr/page_model_test.go
git commit -m "fix(mpr): widgetNodeFromBSON extracts DataView FooterWidgets"
```

---

## Task 5: Fix `widgetToBSON` — write DataView `ShowFooter` + `FooterWidgets` (TDD)

**Files:**
- Modify: `mdl/backend/mpr/page_model.go`

- [ ] **Step 1: Write failing test**

Append to `page_model_test.go`:

```go
func TestWidgetToBSON_DataView_WritesFooter(t *testing.T) {
	footerNode := &types.WidgetNode{
		Kind: types.WidgetContainer,
		Name: "footer1",
	}
	dvNode := &types.WidgetNode{
		Kind:   types.WidgetDataView,
		Name:   "dvMain",
		Footer: []*types.WidgetNode{footerNode},
	}

	doc := widgetToBSON(dvNode)
	if doc == nil {
		t.Fatal("widgetToBSON returned nil")
	}

	// Check ShowFooter = true
	showFooter, ok := dGet(doc, "ShowFooter").(bool)
	if !ok || !showFooter {
		t.Errorf("ShowFooter should be true in BSON, got %v", dGet(doc, "ShowFooter"))
	}

	// Check FooterWidgets array contains the footer node
	footerArr := dGetArrayElements(dGet(doc, "FooterWidgets"))
	if len(footerArr) == 0 {
		t.Error("FooterWidgets array should not be empty")
	}
}

func TestWidgetToBSON_DataView_NoFooterWhenEmpty(t *testing.T) {
	dvNode := &types.WidgetNode{
		Kind:   types.WidgetDataView,
		Name:   "dvMain",
		Footer: nil,
	}
	doc := widgetToBSON(dvNode)
	if dGet(doc, "ShowFooter") != nil {
		t.Error("ShowFooter should not be written when Footer is empty")
	}
}
```

- [ ] **Step 2: Run — expect FAIL**

```bash
go test ./mdl/backend/mpr/ -run TestWidgetToBSON_DataView -v 2>&1 | head -10
```

- [ ] **Step 3: Fix `widgetToBSON` DataView case in `page_model.go`**

Replace the DataView case (around line 876-880):

```go
case types.WidgetDataView:
	if node.DataSource != nil {
		doc = append(doc, bson.E{Key: "DataSource", Value: dataSourceToBSON(node.DataSource)})
	}
	doc = append(doc, bson.E{Key: "Widgets", Value: bsonVersionedArray(widgetsToBSON(node.Children))})
	if len(node.Footer) > 0 {
		doc = append(doc, bson.E{Key: "ShowFooter", Value: true})
		doc = append(doc, bson.E{Key: "FooterWidgets", Value: bsonVersionedArray(widgetsToBSON(node.Footer))})
	}
```

- [ ] **Step 4: Run — expect PASS**

```bash
go test ./mdl/backend/mpr/ -run TestWidgetToBSON_DataView -v
```

- [ ] **Step 5: Commit**

```bash
git add mdl/backend/mpr/page_model.go mdl/backend/mpr/page_model_test.go
git commit -m "fix(mpr): widgetToBSON writes DataView ShowFooter + FooterWidgets"
```

---

## Task 6: Re-enable DataView overlay + update guard + golden rebuild

**Files:**
- Modify: `mdl/executor/cmd_pages_create_v3.go`
- Modify: `mdl/executor/page_overlay_guard_test.go`
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl`

- [ ] **Step 1: Remove DataView from `widgetTreeHasLossyKind`**

In `cmd_pages_create_v3.go`, find the DataView case in `widgetTreeHasLossyKind`:

```go
case types.WidgetDataView, types.WidgetListView:
    // DataView/ListView DataSource is a complex nested gen structure ...
    return true
```

Change to (remove DataView, keep ListView):

```go
case types.WidgetListView:
    // ListView DataSource is a complex nested gen structure the IR cannot
    // reproduce yet — gate to legacy builder.
    return true
```

- [ ] **Step 2: Update `page_overlay_guard_test.go`**

In `TestPageLossyWidgets_DocumentedGap`, remove `types.WidgetDataView` from `stillLossy` and remove the DataView rationale comment.

In `TestPageOverlayEnabledWidgets`, add `types.WidgetDataView`:

```go
enabled := []types.WidgetKind{
    types.WidgetDataView, // footer + column widths now round-trip correctly
    types.WidgetContainer,
    // ... rest unchanged
}
```

- [ ] **Step 3: Build and run all page tests**

```bash
go build ./...
go test ./mdl/executor/... -run "TestPage\|TestRoundtrip\|TestDescribe" -count=1 -v 2>&1 | grep -E "PASS|FAIL|RUN" | head -30
```

No new failures expected. The 4 pre-existing `UserRole/Workflow` failures are unrelated.

- [ ] **Step 4: Add doc comment to `helpdesk-app.mdl` (SOP)**

Append to the last `-- Note:` line in `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl`:

```
-- Note: golden rebuilt after DataView footer + column-weight AutoFill roundtrip fix (2026-06-02)
```

- [ ] **Step 5: Run golden rebuild**

```bash
make update-helpdesk-golden 2>&1 | tail -10
```

Expected: passes without errors.

- [ ] **Step 6: Inspect golden diff**

```bash
git diff --stat testdata/
git diff testdata/helpdesk-golden-11.10.0/describe-snapshot.mdl | head -60
```

Expected: column widths now appear correctly (e.g. `column col1 (DesktopWidth: 12, TabletWidth: AutoFill, PhoneWidth: AutoFill)`) and footer widgets appear (`footer footer1 { ... }`). Verify no unexpected content is lost.

If the diff shows unexpected losses, revert and investigate before committing.

- [ ] **Step 7: Commit everything together**

```bash
git add mdl/executor/cmd_pages_create_v3.go mdl/executor/page_overlay_guard_test.go \
        mdl-examples/use-cases/helpdesk/helpdesk-app.mdl \
        testdata/helpdesk-golden-11.6.6/ testdata/helpdesk-golden-11.10.0/ \
        testdata/helpdesk-clean-11.6.6/
git commit -m "fix(pages): re-enable DataView overlay — column widths and footer round-trip correctly"
```

---

## Self-Review

| Requirement | Task |
|-------------|------|
| propInt handles "AutoFill" → -1 | 1 |
| WidgetNode.Footer field | 2 |
| astWidgetToNode separates DataView footer | 3 |
| widgetNodeFromBSON extracts FooterWidgets | 4 |
| widgetToBSON writes ShowFooter + FooterWidgets | 5 |
| DataView removed from lossy gate | 6 |
| Guard test updated | 6 |
| Golden rebuild passes | 6 |
| All changes have TDD tests | 1, 3, 4, 5 |
