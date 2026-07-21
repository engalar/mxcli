# CE0463 Golden Codec Comparison — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Find all field-level differences between a golden (mx check-passing) DataGrid2 widget and our builder's output, using `modelsdk/codec` element decoding to bypass random-ID noise.

**Architecture:** Open both golden and built MPRs via `codec.Open`. Decode the page containing the DataGrid2 widget to `*genPg.Page`. Walk to `*genCw.CustomWidget`. Recursively dump all primitive/enum field values as path→value maps. Compare maps — every difference is a potential CE0463 cause.

**Tech Stack:** Go, `modernc.org/sqlite`, `go.mongodb.org/mongo-driver/v2/bson`, `github.com/mendixlabs/mxcli/modelsdk/codec`, `github.com/mendixlabs/mxcli/modelsdk/element`, `github.com/mendixlabs/mxcli/modelsdk/gen/pages`, `github.com/mendixlabs/mxcli/modelsdk/gen/customwidgets`

## Global Constraints

- All paths relative to `/mnt/data_sdb/mxcli/`
- MxBuild at `~/.mxcli/mxbuild/11.12.1/modeler/mx`
- Golden project: `app-golden/minimal.mpr` after `mx update-widgets` (0 CE0463)
- Built project: same MPR before `mx update-widgets` (with CE0463)
- Minimal test MDL at `debug_minimal.mdl` (1 DataGrid2, 1 column)
- Comparison program at `cmd/ce0463-compare/main.go`
- Current code state: includes all prior augment.go/mpk.go/datagrid_builder.go fixes

---

## File Structure

| File | Responsibility |
|------|---------------|
| `cmd/ce0463-compare/main.go` | Load both MPRs, find DataGrid2 widgets via codec, dump element fields, compare |
| `cmd/ce0463-compare/dumper.go` | Recursive element property dumper — converts element tree to flat path→value map |
| `debug_minimal.mdl` | Minimal DataGrid2 test script (already exists — `git checkout` resets it if deleted) |
| `app-golden/minimal.mpr` | Project file (reset + exec + update-widgets to create golden state) |

---


### Task 1: Set up test environment

Reset MPR to clean state, exec the minimal MDL script, create golden version with `mx update-widgets`.

**Files:**
- Modify: `app-golden/minimal.mpr`
- Read: `debug_minimal.mdl`

- [ ] **Step 1: Reset MPR and exec minimal script**

```bash
git checkout -- app-golden/minimal.mpr
# Confirm debug_minimal.mdl exists; if not create it:
cat > /mnt/data_sdb/mxcli/debug_minimal.mdl << 'MDLEOF'
create module DebugDG;
create persistent entity DebugDG.Item (
  Name: string(200)
);
create page DebugDG.ItemList (
  title: 'Debug DG',
  layout: Atlas_Core.Atlas_Default
) {
  datagrid dgItems (datasource: database DebugDG.Item) {
    column colName (attribute: Name, caption: 'Name')
  }
};
MDLEOF

# Build latest mxcli (includes all prior fixes)
go build -o /tmp/mxcli-compare ./cmd/mxcli
/tmp/mxcli-compare -p app-golden/minimal.mpr exec debug_minimal.mdl 2>&1 | tail -3
```

Expected output:
```
Script time: ~350ms
```

- [ ] **Step 2: Verify CE0463 exists on the built version**

```bash
~/.mxcli/mxbuild/11.12.1/modeler/mx check app-golden/minimal.mpr 2>&1 | grep CE0463
```

Expected: 1 CE0463 on `Data grid 2 'dgItems'`

- [ ] **Step 3: Save built MPR as baseline**

```bash
cp app-golden/minimal.mpr /tmp/built_before.mpr
```

- [ ] **Step 4: Create golden MPR via mx update-widgets**

```bash
~/.mxcli/mxbuild/11.12.1/modeler/mx update-widgets app-golden/minimal.mpr 2>&1
```

Expected output: (no errors, just quiet success)

- [ ] **Step 5: Verify golden has 0 CE0463**

```bash
~/.mxcli/mxbuild/11.12.1/modeler/mx check app-golden/minimal.mpr 2>&1 | grep -c CE0463
```

Expected: 0

- [ ] **Step 6: Save golden MPR**

```bash
cp app-golden/minimal.mpr /tmp/golden_after.mpr
```

---


### Task 2: Write the element property dumper

**Files:**
- Create: `cmd/ce0463-compare/dumper.go`

**Interfaces:**
- Consumes: `element.Element` from modelsdk codec decode
- Produces: `map[string]any` — flat key-value map where keys are dotted paths like `"Type.ObjectType.PropertyTypes[0].ValueType.Required"` and values are the primitive Go values (string, int, bool, float64)

The dumper must:
1. Walk `elem.Properties()` recursively
2. For `element.PrimitiveProperty`: record name→value
3. For `element.EnumerationProperty`: record name→string value
4. For `element.ChildProperty`: recurse into the child, prepend path
5. For `element.ChildListProperty`: recurse into each child with `[index]` suffix
6. For `element.ByNameRefProperty`: record name→string name
7. For `element.ByIdRefProperty`: record name→UUID string (for diagnostic, filtered later)
8. Skip fields named `$ID` or `$Type` (internal BSON metadata)
9. Skip nil child properties (no ChildElement)

- [ ] **Step 1: Write dumper.go**

```go
// cmd/ce0463-compare/dumper.go
package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// dumpElement recursively walks an element's properties and returns a flat
// path→value map. Keys are dotted paths like "Type.ObjectType.PropertyTypes[0].PropertyKey".
// Binary ID fields ($ID, TypePointer, and $binary/base64/subType) are excluded.
func dumpElement(elem element.Element) map[string]any {
	result := make(map[string]any)
	dumpRecursive(elem, "", result)
	return result
}

func dumpRecursive(elem element.Element, prefix string, out map[string]any) {
	for _, prop := range elem.Properties() {
		name := prop.Name()
		// Skip BSON internal fields
		if name == "$ID" || name == "$Type" {
			continue
		}

		path := name
		if prefix != "" {
			path = prefix + "." + name
		}

		switch p := prop.(type) {
		case element.PrimitiveProperty:
			out[path] = p.PrimitiveValue()

		case element.EnumerationProperty:
			out[path] = p.EnumValue()

		case element.ByNameRefProperty:
			out[path] = p.NameValue()

		case element.ByIdRefProperty:
			// Record UUID string but mark as ID for filtering
			out[path] = "[ID:" + p.IDValue().String() + "]"

		case element.ChildProperty:
			child := p.ChildElement()
			if child != nil {
				dumpRecursive(child, path, out)
			} else {
				out[path] = nil
			}

		case element.ChildListProperty:
			children := p.ChildElements()
			if len(children) > 0 {
				for i, child := range children {
					idxPath := fmt.Sprintf("%s[%d]", path, i)
					dumpRecursive(child, idxPath, out)
				}
			} else {
				out[path] = "[]"
			}

		default:
			out[path] = fmt.Sprintf("<unknown property type: %T>", p)
		}
	}
}

// compareDumps compares two dump maps and returns differences.
// Skips keys containing binary ID markers.
// Returns: (onlyInA, onlyInB, different map[key]{a, b})
func compareDumps(a, b map[string]any) (onlyInA, onlyInB []string, different map[string][2]any) {
	different = make(map[string][2]any)

	allKeys := make(map[string]bool)
	for k := range a {
		allKeys[k] = true
	}
	for k := range b {
		allKeys[k] = true
	}

	keys := make([]string, 0, len(allKeys))
	for k := range allKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		// Skip ID-marker paths
		if strings.Contains(k, ".$ID") || strings.Contains(k, "TypePointer") ||
			strings.Contains(k, "$binary") || strings.Contains(k, "base64") ||
			strings.Contains(k, "subType") {
			continue
		}

		va, aOk := a[k]
		vb, bOk := b[k]

		if !aOk && bOk {
			onlyInB = append(onlyInB, k)
			continue
		}
		if aOk && !bOk {
			onlyInA = append(onlyInA, k)
			continue
		}

		// Compare primitives — convert both to string for comparison
		sa := fmt.Sprintf("%v", va)
		sb := fmt.Sprintf("%v", vb)
		if sa != sb {
			different[k] = [2]any{va, vb}
		}
	}

	return onlyInA, onlyInB, different
}

// printDiff outputs comparison results in a readable format.
func printDiff(onlyInA, onlyInB []string, different map[string][2]any) {
	if len(onlyInA) == 0 && len(onlyInB) == 0 && len(different) == 0 {
		fmt.Println("NO DIFFERENCES FOUND — golden and builder produce identical element trees")
		return
	}

	if len(different) > 0 {
		fmt.Printf("\n=== FIELD VALUE DIFFERENCES (%d) ===\n", len(different))
		keys := make([]string, 0, len(different))
		for k := range different {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			vals := different[k]
			fmt.Printf("  %s:\n", k)
			fmt.Printf("    GOLDEN: %v\n", vals[0])
			fmt.Printf("    BUILT:  %v\n", vals[1])
		}
	}

	if len(onlyInA) > 0 {
		fmt.Printf("\n=== ONLY IN GOLDEN (%d) ===\n", len(onlyInA))
		for _, k := range onlyInA {
			fmt.Printf("  %s\n", k)
		}
	}

	if len(onlyInB) > 0 {
		fmt.Printf("\n=== ONLY IN BUILDER (%d) ===\n", len(onlyInB))
		for _, k := range onlyInB {
			fmt.Printf("  %s\n", k)
		}
	}
}

// skipIDValues removes ID-type keys from the dump so comparison focuses on values.
func filterDump(m map[string]any, keepID bool) map[string]any {
	result := make(map[string]any, len(m))
	for k, v := range m {
		if !keepID && strings.HasPrefix(fmt.Sprintf("%v", v), "[ID:") {
			continue
		}
		result[k] = v
	}
	return result
}

// countDump returns total keys and non-ID keys.
func countDump(m map[string]any) (total, nonID int) {
	for k, v := range m {
		total++
		if !strings.HasPrefix(fmt.Sprintf("%v", v), "[ID:") {
			nonID++
		}
	}
	return
}
```

- [ ] **Step 2: Verify compilation**

```bash
cd /mnt/data_sdb/mxcli && go build ./cmd/ce0463-compare/ 2>&1
```

Expected: no errors

---

### Task 3: Write the comparison tool main

**Files:**
- Create: `cmd/ce0463-compare/main.go`

**Interfaces:**
- Consumes: `dumper.go` functions; `codec.Open`, `codec.NewDecoder`, `codec.DefaultRegistry`; `genPg.Page`, `genCw.CustomWidget`, `genCw.CustomWidgetType`
- Produces: stdout report of all field-level differences

The tool must:
1. Open built MPR (`/tmp/built_before.mpr`) and golden MPR (`/tmp/golden_after.mpr`)
2. For each, iterate all units, decode to element, check if `*genPg.Page`
3. Walk each page's widget tree via `element.Walk` to find `*genCw.CustomWidget`
4. For each found widget, call `dumpElement` to get the flat field map
5. Match widgets between golden and built by `Name` field
6. Call `compareDumps` and `printDiff`

- [ ] **Step 1: Write main.go**

```go
// cmd/ce0463-compare/main.go
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/mendixlabs/mxcli/modelsdk/codec"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genCw "github.com/mendixlabs/mxcli/modelsdk/gen/customwidgets"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("Usage: %s <built.mpr> <golden.mpr>", os.Args[0])
	}

	builtPath := os.Args[1]
	goldenPath := os.Args[2]

	dec := codec.NewDecoder(codec.DefaultRegistry)

	// Extract widgets from both files
	builtWidgets := extractWidgets(builtPath, dec)
	goldenWidgets := extractWidgets(goldenPath, dec)

	fmt.Printf("Built:  %d DataGrid2 widgets\n", len(builtWidgets))
	fmt.Printf("Golden: %d DataGrid2 widgets\n", len(goldenWidgets))

	// Match by name and compare
	matched := 0
	for name, builtElem := range builtWidgets {
		goldenElem, ok := goldenWidgets[name]
		if !ok {
			fmt.Printf("\nWIDGET %q: only in built, skipping\n", name)
			continue
		}
		matched++

		fmt.Printf("\n========================================\n")
		fmt.Printf("COMPARING WIDGET: %q\n", name)
		fmt.Printf("========================================\n")

		builtDump := dumpElement(builtElem)
		goldenDump := dumpElement(goldenElem)

		bTotal, bNonID := countDump(builtDump)
		gTotal, gNonID := countDump(goldenDump)
		fmt.Printf("Built fields:  %d total, %d non-ID\n", bTotal, bNonID)
		fmt.Printf("Golden fields: %d total, %d non-ID\n", gTotal, gNonID)

		// Filter out ID values for comparison
		builtFiltered := filterDump(builtDump, false)
		goldenFiltered := filterDump(goldenDump, false)

		onlyInA, onlyInB, different := compareDumps(goldenFiltered, builtFiltered)
		printDiff(onlyInA, onlyInB, different)

		if len(onlyInA) == 0 && len(onlyInB) == 0 && len(different) == 0 {
			fmt.Println("\n✓ BUILDER MATCHES GOLDEN for this widget")
		}
	}

	if matched == 0 {
		fmt.Println("\nNo matching widgets found between built and golden")
	}

	// Also report any unmatched golden widgets
	for name := range goldenWidgets {
		if _, ok := builtWidgets[name]; !ok {
			fmt.Printf("\nWIDGET %q: only in golden (not in built)\n", name)
		}
	}
}

// extractWidgets opens an MPR, finds all DataGrid2 widgets, and returns them
// keyed by widget Name.
func extractWidgets(path string, dec *codec.Decoder) map[string]element.Element {
	result := make(map[string]element.Element)

	store, err := codec.Open(path)
	if err != nil {
		log.Printf("Warning: cannot open %s: %v", path, err)
		return result
	}
	defer store.Close()

	for _, unitID := range store.ListUnits() {
		raw, err := store.LoadUnit(unitID)
		if err != nil {
			continue
		}

		elem, err := dec.Decode(raw)
		if err != nil {
			continue
		}

		page, ok := elem.(*genPg.Page)
		if !ok {
			continue
		}

		// Walk page tree to find CustomWidgets
		element.Walk(page, func(w element.Element) bool {
			cw, ok := w.(*genCw.CustomWidget)
			if !ok {
				return true
			}
			cwType, ok := cw.Type().(*genCw.CustomWidgetType)
			if !ok {
				return true
			}
			widgetID := cwType.WidgetId()
			if widgetID != "com.mendix.widget.web.datagrid.Datagrid" &&
				widgetID != "com.mendix.widget.web.gallery.Gallery" {
				return true
			}
			name := cw.Name()
			result[name] = cw
			return true
		})
	}

	return result
}

// Ensure bson is used (for compilation with the import)
var _ = bson.MarshalExtJSON
```

- [ ] **Step 2: Verify compilation**

```bash
cd /mnt/data_sdb/mxcli && go build ./cmd/ce0463-compare/ 2>&1
```

Expected: no errors

If compile errors about unused imports, add the following to the bottom of main.go:
```go
// Keep genPg import alive
var _ = genPg.BuildingBlock
// Keep genCw import alive
var _ = genCw.CustomWidget{}
```

- [ ] **Step 2: Verify compilation**

```bash
cd /mnt/data_sdb/mxcli && go build ./cmd/ce0463-compare/ 2>&1
```

Expected: no errors

If compile errors about unused imports, ensure both gen packages are referenced:
```go
func init() {
	_ = genPg.Page{}
	_ = genCw.CustomWidget{}
}
```

---

### Task 4: Run comparison on minimal test case

**Files:**
- Execute: `go run ./cmd/ce0463-compare/` on `/tmp/built_before.mpr` and `/tmp/golden_after.mpr`

- [ ] **Step 1: Run the comparison tool**

```bash
cd /mnt/data_sdb/mxcli && go run ./cmd/ce0463-compare/ /tmp/built_before.mpr /tmp/golden_after.mpr 2>&1
```

Expected output: list of field-level differences between built and golden.

If the tool crashes with a panic, fix the crash (likely a nil pointer or type assertion) and re-run.

- [ ] **Step 2: Categorize each difference**

For every reported difference, classify it:

| Category | Meaning | Fix File |
|----------|---------|---------|
| **ValueType field missing** | Golden has a field our Type's ValueType doesn't have | `modelsdk/widgets/augment.go:createDefaultValueType` |
| **ValueType field wrong value** | Field exists but value differs | `modelsdk/widgets/augment.go:createDefaultValueType` |
| **WidgetValue field missing/wrong** | Object's WidgetValue has wrong field value | `mdl/backend/mpr/datagrid_builder.go:buildDefaultWidgetValueBSON` |
| **Property list count mismatch** | Different number of PropertyTypes vs Properties | `modelsdk/widgets/generate.go:GenerateFromMPK` or builder |
| **Page structure diff** | Container/surrounding page widget fields differ | `mdl/executor/pages_builder_widgets_v3.go` |
| **Category wrong** | PropertyType Category field differs | `modelsdk/widgets/augment.go:buildNestedObjectType` |
| **Translations wrong** | TextTemplate ValueType Translation entries differ | `modelsdk/widgets/mpk/mpk.go` + `augment.go` |

- [ ] **Step 3: Record findings**

Write to `docs/superpowers/plans/2026-07-20-ce0463-findings.md`:

```markdown
# CE0463 Codec Comparison Findings

## Widget: dgItems

### Differences found

| # | Path | Golden | Built | Likely CE0463 cause? | Fix |
|---|------|--------|-------|---------------------|-----|
| 1 | `Type.ObjectType.PropertyTypes[3].ValueType.Required` | `true` | `false` | Yes | `augment.go` line X: change `p.Required` to `true` |
| ... | ... | ... | ... | ... | ... |
```

Fill in each row based on the actual tool output.

---

### Task 5: Implement fixes based on findings

**Files:**
- Modify: files identified in findings

For EACH difference marked as "Likely CE0463 cause" in the findings:

- [ ] **Step N: Fix a single difference**

Example template (replace path, golden value, built value with actual finding):

```bash
# Example: Fix Required field
```
Edit `modelsdk/widgets/augment.go`:
- Find the line setting `"Required": p.Required`
- Change to: `"Required": true`
- Add override for properties with explicit `required="false"`:
```go
if p.RequiredExplicit && !p.Required {
    vt["Required"] = false
}
```
```

Fix each difference one at a time, re-building and re-checking CE0463 after each:

```bash
go build -o /tmp/mxcli-fix ./cmd/mxcli
git checkout -- app-golden/minimal.mpr
/tmp/mxcli-fix -p app-golden/minimal.mpr exec debug_minimal.mdl
~/.mxcli/mxbuild/11.12.1/modeler/mx check app-golden/minimal.mpr 2>&1 | grep CE0463
```

After each fix, check if CE0463 count drops. Once it reaches 0, move to Task 6.

- [ ] **Step: Commit each fix**

```bash
git add -A && git commit -m "fix: [description of what was fixed]"
```

---

### Task 6: Verify with full helpdesk-app.mdl

**Files:**
- Execute: full helpdesk-app.mdl exec

- [ ] **Step 1: Run full script**

```bash
git checkout -- app-golden/minimal.mpr
go build -o /tmp/mxcli-full ./cmd/mxcli
/tmp/mxcli-full -p app-golden/minimal.mpr exec helpdesk-app.mdl 2>&1 | tail -3
~/.mxcli/mxbuild/11.12.1/modeler/mx check app-golden/minimal.mpr 2>&1 | grep -c CE0463
```

Expected: 0 CE0463

- [ ] **Step 2: If CE0463 > 0, repeat diagnostic on full script**

Run the comparison tool against the full helpdesk-app.mdl output to find remaining differences:

```bash
cp app-golden/minimal.mpr /tmp/built_full.mpr
~/.mxcli/mxbuild/11.12.1/modeler/mx update-widgets app-golden/minimal.mpr
cp app-golden/minimal.mpr /tmp/golden_full.mpr
go run ./cmd/ce0463-compare/ /tmp/built_full.mpr /tmp/golden_full.mpr 2>&1 | head -50
```

Fix any additional differences found, then re-verify.

- [ ] **Step 3: Final commit**

```bash
git add -A && git commit -m "fix: resolve all CE0463 errors on DataGrid2 and Gallery widgets"
```

---

## Self-Review Checklist

**1. Spec coverage:**
- Task 1 covers test environment setup (reset MPR, exec minimal MDL, create golden via mx update-widgets) ✓
- Task 2 covers the element dumper utility (recursive element property walker) ✓
- Task 3 covers the comparison tool main (codec Open, Decode, Walk, compare) ✓
- Task 4 covers running the comparison and categorizing findings ✓
- Task 5 covers implementing fixes based on findings (template with guidance) ✓
- Task 6 covers full verification with helpdesk-app.mdl ✓

**2. Placeholder scan:**
- No "TBD", "TODO", or "implement later" patterns ✓
- All code blocks contain actual Go code ✓
- All file paths are exact ✓
- All commands are complete with expected output descriptions ✓
- Task 5 uses a template pattern for fix code (acceptable — fix code depends on comparison results that can't be known in advance) ✓
- No "Add appropriate error handling" or "Write tests for the above" without actual code ✓

**3. Type consistency:**
- `dumper.go` exports: `dumpElement(element.Element) map[string]any`, `compareDumps(...)`, `printDiff(...)`, `filterDump(...)`, `countDump(...)` ✓
- `main.go` imports: `codec` (Open, NewDecoder, DefaultRegistry), `genPg.Page`, `genCw.CustomWidget`, `genCw.CustomWidgetType`, `element.Walk` ✓
- `main.go` calls: `store.ListUnits()` → `store.LoadUnit()` → `dec.Decode()` → `elem.(*genPg.Page)` → `element.Walk(page, ...)` internally ✓
- All function signatures in main.go match definitions in dumper.go ✓

**4. Missing functionality check:**
- Tool handles case where golden has a widget not in built (and vice versa) ✓
- Tool skips binary ID fields ($ID, TypePointer) ✓
- Tool handles nil child properties ✓
- Tool counts fields with and without ID markers ✓

---

**Plan complete and saved to `docs/superpowers/plans/2026-07-20-ce0463-golden-codec-comparison.md`.**
