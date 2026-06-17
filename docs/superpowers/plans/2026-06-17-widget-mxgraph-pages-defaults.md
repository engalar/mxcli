# Widget mxgraph Index + Page Defaults Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build mxgraph WidgetAdapter to auto-index widget definitions from `.mpk` and `.mxcli/widgets/*.def.json`, wire it into `WidgetRegistry` as a fast lookup cache, and fix page builder default field gaps identified by NDSL comparison.

**Architecture:** Three independent subsystems: (A) mxgraph WidgetAdapter scanning the filesystem, (B) WidgetRegistry fast path via mxgraph index snapshot, (C) page builder default field injection. Each produces working, testable software independently.

**Tech Stack:** Go, `archive/zip` (stdlib for MPK parsing), `encoding/json` (`.def.json`), `mxgraph` (internal node/edge engine)

---

## File Map

### Created
| File | Responsibility |
|------|---------------|
| `internal/mxgraph/adapter/mpr/widget_adapter.go` | Scans `.mxcli/widgets/*.def.json` + `widgets/*.mpk`, emits mxgraph `Widget` nodes |

### Modified
| File | Change |
|------|--------|
| `mdl/graphcatalog/graph.go` | Add exported `MxGraph()` method exposing underlying `*mxgraph.Graph` |
| `mdl/executor/widget_registry.go` | Add `mxGraph` field, `SetMxGraph()` method, fast-path in `GetByWidgetID` |
| `mdl/executor/cmd_pages_builder.go` | Add `mxGraph` field to `pageBuilder`, inject in `initPluggableEngine` |
| `mdl/executor/cmd_pages_create_v3.go` | Pass `ctx.Graph.MxGraph()` into `pageBuilder.mxGraph` |
| `cmd/mxcli/cmd_bson_dump.go` | Register WidgetAdapter in `buildMxGraph()` |
| `cmd/mxcli/serve.go` | Register WidgetAdapter in `buildProjectGraph()` |
| `mdl/executor/cmd_graph.go` | Register WidgetAdapter in `buildGraph()` |
| `mdl/executor/pages_builder_v3.go` | Add page/row/column/widget/action default field injection |

---

## Task Breakdown

### Task A1: WidgetAdapter — scan `.def.json` + `.mpk`, create mxgraph nodes

**Files:**
- Create: `internal/mxgraph/adapter/mpr/widget_adapter.go`
- No test needed at this stage (integration tested via graph build)

The adapter scans two filesystem locations and creates mxgraph `Node` with `Label: "Widget"`. The `.def.json` entries take priority: if a widget ID appears in both `.def.json` and `.mpk`, only the `.def.json` version is indexed (tracked via a `seenWidgetID` set).

For `.mpk` scanning, use `modelsdk/widgets/mpk.FindMPK(projectDir)` to locate all MPK files and their widget IDs, then `mpk.ParseMPKForWidget(mpkPath, widgetID)` to extract each definition.

For `.def.json` scanning, the file format is the same `WidgetDefinition` JSON structure: `{widgetId, mdlName, templateFile, defaultEditable, propertyMappings, childSlots}`.

Props on each mxgraph node:
```json
{
  "WidgetID":   "com.helpdesk.widget.TicketStatusBadge",
  "MDLName":    "TICKETSTATUSBADGE",
  "Name":       "TicketStatusBadge",
  "WidgetKind": "pluggable",
  "Source":     "def.json" | "mpk"
}
```

- [ ] **Step A1.1: Create the WidgetAdapter struct**

```go
package mpr

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/modelsdk/widgets/mpk"
)

// WidgetAdapter emits Widget nodes from .mxcli/widgets/*.def.json and
// widgets/*.mpk files. .def.json entries have priority — duplicates by
// widget ID are resolved in favour of .def.json.
type WidgetAdapter struct {
	ProjectDir string // e.g. "D:/gh/mxcli/HelpDeskE2E"
}

func (a *WidgetAdapter) Name() string { return "widget" }

func (a *WidgetAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{
		NodeLabels: []mxgraph.Label{"Widget"},
	}
}

func (a *WidgetAdapter) Watch(ctx context.Context, sink mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}

func (a *WidgetAdapter) Build(ctx context.Context, sink mxgraph.EventSink) error {
	if a.ProjectDir == "" {
		return nil
	}
	var events []mxgraph.Event
	seen := make(map[string]bool)

	// Priority 1: .mxcli/widgets/*.def.json
	defDir := filepath.Join(a.ProjectDir, ".mxcli", "widgets")
	if entries, err := os.ReadDir(defDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".def.json") {
				continue
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			data, err := os.ReadFile(filepath.Join(defDir, entry.Name()))
			if err != nil {
				log.Printf("warning: reading %s: %v", entry.Name(), err)
				continue
			}
			var def struct {
				WidgetID    string `json:"widgetId"`
				MDLName     string `json:"mdlName"`
				WidgetKind  string `json:"widgetKind,omitempty"`
			}
			if err := json.Unmarshal(data, &def); err != nil {
				log.Printf("warning: parsing %s: %v", entry.Name(), err)
				continue
			}
			if def.WidgetID == "" {
				continue
			}
			seen[def.WidgetID] = true
			node := &mxgraph.Node{
				ID:    mxgraph.NodeID(def.WidgetID),
				Label: "Widget",
				Props: map[string]any{
					"WidgetID":   def.WidgetID,
					"MDLName":    strings.ToUpper(def.MDLName),
					"WidgetKind": def.WidgetKind,
					"Source":     "def.json",
				},
			}
			if def.MDLName != "" {
				node.Props["Name"] = def.MDLName
			}
			events = append(events, mxgraph.Event{Type: mxgraph.NodeCreated, Node: node})
		}
	}

	// Priority 2: widgets/*.mpk (only for widget IDs not already indexed)
	mpkMap, err := mpk.FindAllMPK(a.ProjectDir)
	if err != nil {
		log.Printf("warning: scanning MPK files: %v", err)
	}
	for widgetID, mpkPath := range mpkMap {
		if seen[widgetID] {
			continue // .def.json wins
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		mpkDef, err := mpk.ParseMPKForWidget(mpkPath, widgetID)
		if err != nil || mpkDef == nil {
			continue
		}
		widgetKind := "custom"
		if mpkDef.IsPluggable {
			widgetKind = "pluggable"
		}
		mdlName := lastIDSegment(widgetID)
		node := &mxgraph.Node{
			ID:    mxgraph.NodeID(widgetID),
			Label: "Widget",
			Props: map[string]any{
				"WidgetID":   widgetID,
				"MDLName":    strings.ToUpper(mdlName),
				"Name":       mpkDef.Name,
				"WidgetKind": widgetKind,
				"Source":     "mpk",
			},
		}
		events = append(events, mxgraph.Event{Type: mxgraph.NodeCreated, Node: node})
	}

	if len(events) > 0 {
		return sink.Emit(events)
	}
	return nil
}

func lastIDSegment(widgetID string) string {
	parts := strings.Split(widgetID, ".")
	return strings.ToLower(parts[len(parts)-1])
}
```

Note: `mpk.FindAllMPK` does not exist yet — see Step A1.2 below. The existing `mpk.FindMPK(projectDir, widgetID)` function only returns a path for a specific widget ID. We need a version that returns ALL widget IDs from all MPK files.

- [ ] **Step A1.2: Add `FindAllMPK` to the mpk package**

File: `modelsdk/widgets/mpk/mpk.go`

The current `FindMPK(projectDir, widgetID string) (string, error)` function scans widgets/.mpk and builds a cache, but it takes a specific widgetID. We need a function that returns the full map.

```go
// FindAllMPK scans projectDir/widgets/*.mpk and returns a map of
// widgetID → mpkFilePath for every widget discovered across all MPK files.
func FindAllMPK(projectDir string) (map[string]string, error) {
	widgetsDir := filepath.Join(projectDir, "widgets")
	entries, err := os.ReadDir(widgetsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read widgets dir: %w", err)
	}
	result := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".mpk") {
			continue
		}
		mpkPath := filepath.Join(widgetsDir, entry.Name())
		ids, err := getWidgetIDsFromMPK(mpkPath)
		if err != nil {
			continue
		}
		for _, id := range ids {
			if _, exists := result[id]; !exists {
				result[id] = mpkPath
			}
		}
	}
	return result, nil
}
```

Note: `getWidgetIDsFromMPK` already exists in `mpk.go` (line 423). Use it directly.

- [ ] **Step A1.3: Compile check**

```bash
cd D:/gh/mxcli && go build ./internal/mxgraph/adapter/mpr/... 2>&1
```

Expected: no errors.

- [ ] **Step A1.4: Commit**

```bash
git add internal/mxgraph/adapter/mpr/widget_adapter.go modelsdk/widgets/mpk/mpk.go
git commit -m "feat(mxgraph): add WidgetAdapter indexing .def.json and .mpk files"
```

---

### Task A2: Register WidgetAdapter in all graph build points

**Files:**
- Modify: `cmd/mxcli/cmd_bson_dump.go:419-425`
- Modify: `cmd/mxcli/serve.go:82-100`
- Modify: `mdl/executor/cmd_graph.go:40-46`

Each build point creates a `mgr := mxgraph.NewIndexManager()` then registers adapters. Add `mpradapter.WidgetAdapter` with the project directory.

In `buildMxGraph` (cmd_bson_dump.go), the project path is known but there's no reader/projectDir variable. The function takes `projectPath string`. Use `filepath.Dir(projectPath)`.

- [ ] **Step A2.1: Register in `cmd_bson_dump.go`**

```go
// Inside buildMxGraph, after other RegisterAdapter calls:
mgr.RegisterAdapter(&mpradapter.WidgetAdapter{ProjectDir: filepath.Dir(projectPath)})
```

- [ ] **Step A2.2: Register in `serve.go`**

```go
// Inside buildProjectGraph, after other RegisterAdapter calls:
mgr.RegisterAdapter(&mpradapter.WidgetAdapter{ProjectDir: filepath.Dir(projectPath)})
```

- [ ] **Step A2.3: Register in `cmd_graph.go`**

`cmd_graph.go` has access to `ctx.MprPath`. Use `filepath.Dir(ctx.MprPath)`.

```go
// Inside buildGraph, after other RegisterAdapter calls:
mgr.RegisterAdapter(&mpradapter.WidgetAdapter{ProjectDir: filepath.Dir(ctx.MprPath)})
```

- [ ] **Step A2.4: Compile and commit**

```bash
cd D:/gh/mxcli && go build ./cmd/mxcli/... ./mdl/executor/... 2>&1
git add cmd/mxcli/cmd_bson_dump.go cmd/mxcli/serve.go mdl/executor/cmd_graph.go
git commit -m "feat(mxgraph): register WidgetAdapter in all graph build points"
```

---

### Task B1: Expose Graph from ProjectGraph

**Files:**
- Modify: `mdl/graphcatalog/graph.go`

The `ProjectGraph` stores `mgr *mxgraph.IndexManager` (unexported). `WidgetRegistry` needs `*mxgraph.Graph` (from `mgr.Query()`). Add a method to expose it.

- [ ] **Step B1.1: Add `MxGraph()` method**

Find the `ProjectGraph` struct in `mdl/graphcatalog/graph.go` (around line 14). Add after the existing `g()` method:

```go
// MxGraph returns the underlying mxgraph.Graph for direct node queries.
func (pg *ProjectGraph) MxGraph() *mxgraph.Graph {
	return pg.mgr.Query()
}
```

- [ ] **Step B1.2: Compile and commit**

```bash
cd D:/gh/mxcli && go build ./mdl/graphcatalog/... 2>&1
git add mdl/graphcatalog/graph.go
git commit -m "feat(graphcatalog): add MxGraph() accessor for underlying graph"
```

---

### Task B2: Wire mxgraph into WidgetRegistry

**Files:**
- Modify: `mdl/executor/widget_registry.go`

Add an `mxGraph` field and `SetMxGraph()` method to `WidgetRegistry`. Modify `GetByWidgetID` to check the mxgraph index before falling back to `deriveFromMPK`. This makes subsequent lookups O(1) when the graph snapshot is already loaded.

- [ ] **Step B2.1: Add field and setter**

```go
// In the WidgetRegistry struct (line 28), add after existing fields:
mxGraph *mxgraph.Graph

// New method — add after SetProjectDir (around line 320):
// SetMxGraph injects an mxgraph index for fast widget definition lookup.
// Called by initPluggableEngine when a graph snapshot is available.
func (r *WidgetRegistry) SetMxGraph(g *mxgraph.Graph) {
	r.mxGraph = g
}
```

Add the import for `"github.com/mendixlabs/mxcli/internal/mxgraph"` in the imports block of `widget_registry.go`.

- [ ] **Step B2.2: Modify `GetByWidgetID` to check mxgraph first**

Replace lines 148-166 with:

```go
func (r *WidgetRegistry) GetByWidgetID(widgetID string) (*WidgetDefinition, bool) {
	if def, ok := r.byWidgetID[widgetID]; ok {
		return def, ok
	}

	// Fast path: check mxgraph index (populated from .mxcli/graph.gob snapshot)
	if r.mxGraph != nil {
		nodes := r.mxGraph.FindNodes("Widget", map[string]any{"WidgetID": widgetID})
		if len(nodes) > 0 {
			def := widgetDefinitionFromNode(nodes[0])
			if def != nil {
				r.byWidgetID[widgetID] = def
				r.byMDLName[strings.ToUpper(def.MDLName)] = def
				return def, true
			}
		}
	}

	if r.projectDir == "" {
		return nil, false
	}
	def, err := r.deriveFromMPK(widgetID)
	if err != nil {
		log.Printf("warning: MPK fallback for widget ID %s: %v", widgetID, err)
		return nil, false
	}
	if def == nil {
		return nil, false
	}
	r.byMDLName[strings.ToUpper(def.MDLName)] = def
	r.byWidgetID[def.WidgetID] = def
	return def, true
}
```

- [ ] **Step B2.3: Add `widgetDefinitionFromNode` helper**

Place after the `Modified code` sections (e.g., before `buildDefinitionFromMPK`):

```go
// widgetDefinitionFromNode converts an mxgraph Widget node back to a
// WidgetDefinition. Only populates fields that the index stores — the
// caller gets a minimal definition sufficient for engine lookups.
// Full property mappings require the original .def.json or MPK derivation.
func widgetDefinitionFromNode(n *mxgraph.Node) *WidgetDefinition {
	widgetID, _ := n.Props["WidgetID"].(string)
	mdlName, _ := n.Props["MDLName"].(string)
	widgetKind, _ := n.Props["WidgetKind"].(string)
	if widgetID == "" {
		return nil
	}
	return &WidgetDefinition{
		WidgetID:        widgetID,
		MDLName:         strings.ToLower(mdlName),
		WidgetKind:      widgetKind,
		TemplateFile:    strings.ToLower(mdlName) + ".json",
		DefaultEditable: "Always",
	}
}
```

- [ ] **Step B2.4: Compile and commit**

```bash
cd D:/gh/mxcli && go build ./mdl/executor/... 2>&1
git add mdl/executor/widget_registry.go
git commit -m "feat(executor): WidgetRegistry fast path via mxgraph index"
```

---

### Task B3: Inject mxgraph into pageBuilder / initPluggableEngine

**Files:**
- Modify: `mdl/executor/cmd_pages_builder.go`
- Modify: `mdl/executor/cmd_pages_create_v3.go`

- [ ] **Step B3.1: Add `mxGraph` field to `pageBuilder`**

In `cmd_pages_builder.go`, add to the `pageBuilder` struct (after line 65):

```go
mxGraph *mxgraph.Graph // Injected from ExecContext for widget registry fast path
```

Add the import for `"github.com/mendixlabs/mxcli/internal/mxgraph"` in the imports block.

- [ ] **Step B3.2: Wire it in `initPluggableEngine`**

In the `initPluggableEngine` function (line 69-90), add after `pb.pluggableEngine = ...`:

```go
if pb.mxGraph != nil {
    registry.SetMxGraph(pb.mxGraph)
}
```

- [ ] **Step B3.3: Pass `ctx.Graph.MxGraph()` in pageBuilder creation**

In `cmd_pages_create_v3.go`, find the `pageBuilder` creation (line 66-80). Add:

```go
mxGraph: ctx.Graph.MxGraph(),
```

- [ ] **Step B3.4: Also wire in `cmd_alter_page.go`**

The alter page path also creates a `pageBuilder` (line 285). Find it and add the same field:

```go
mxGraph: ctx.Graph.MxGraph(),
```

Note: `ctx.Graph` may be nil (no graph loaded). The `MxGraph()` method on `ProjectGraph` should handle nil receiver gracefully (return nil). Update Step B1.1 if needed:

```go
func (pg *ProjectGraph) MxGraph() *mxgraph.Graph {
    if pg == nil || pg.mgr == nil {
        return nil
    }
    return pg.mgr.Query()
}
```

- [ ] **Step B3.5: Compile and commit**

```bash
cd D:/gh/mxcli && go build ./mdl/executor/... ./cmd/mxcli/... 2>&1
git add mdl/executor/cmd_pages_builder.go mdl/executor/cmd_pages_create_v3.go mdl/executor/cmd_alter_page.go
git commit -m "feat(executor): inject mxgraph into pageBuilder for widget registry fast path"
```

---

### Task C1: Page-level default fields

**Files:**
- Modify: `mdl/executor/pages_builder_v3.go` — `buildPageV3` (line ~110)

Add missing page-level default fields after line 127 (`page.SetMarkAsUsed(false)`) and before the URL block (line 128).

- [ ] **Step C1.1: Add page defaults in `buildPageV3`**

```go
// After page.SetMarkAsUsed(false) and before if s.URL != "":
page.SetExportLevel("Hidden")
page.SetAppearance(newDefaultAppearance())
page.SetCanvasWidth(800)
page.SetCanvasHeight(600)
```

- [ ] **Step C1.2: Verify that `SetExportLevel` exists on `*genPg.Page`**

Check `modelsdk/gen/pages/types.go` for the method (should exist on `Page` struct). If the method signature is different (e.g. takes an enum instead of string), use the correct signature.

- [ ] **Step C1.3: Compile and commit**

```bash
cd D:/gh/mxcli && go build ./mdl/executor/... 2>&1
git add mdl/executor/pages_builder_v3.go
git commit -m "fix(executor): add page-level default fields (ExportLevel, Appearance, Canvas)"
```

---

### Task C2: Layout grid row/column default fields

**Files:**
- Modify: `mdl/executor/pages_builder_v3.go` — `buildLayoutGridRowV3` (line ~1171) and `buildLayoutGridColumnV3` (line ~1204)

After creating the row/column element, set missing default fields.

- [ ] **Step C2.1: Add row defaults**

In `buildLayoutGridRowV3`, after creating the row (`row := genPg.NewLayoutGridRow()`):

```go
if row.Appearance() == nil {
    row.SetAppearance(newDefaultAppearance())
}
row.SetConditionalVisibilitySettings(nil)
row.SetHorizontalAlignment("None")
row.SetVerticalAlignment("None")
row.SetSpacingBetweenColumns(true)
```

- [ ] **Step C2.2: Add column defaults**

In `buildLayoutGridColumnV3`, after creating the column:

```go
if col.Appearance() == nil {
    col.SetAppearance(newDefaultAppearance())
}
col.SetVerticalAlignment("None")
```

- [ ] **Step C2.3: Compile and commit**

```bash
cd D:/gh/mxcli && go build ./mdl/executor/... 2>&1
git add mdl/executor/pages_builder_v3.go
git commit -m "fix(executor): add layout grid row/column default fields"
```

---

### Task C3: Widget-level common default fields

**Files:**
- Modify: `mdl/executor/pages_builder_v3.go` — `buildWidgetV3` (line ~342)

After each widget is built (both built-in and pluggable), apply common defaults. The function has two paths: built-in (line 358-366) and pluggable (line 369-391). Add defaults after both paths complete.

- [ ] **Step C3.1: Add defaults after built-in path**

After line 366 (`return widget, nil`), move the return to a deferred default function. Better approach: add a helper that applies defaults, called just before each return.

Add after `applyConditionalSettingsGen` call (line 365):

```go
applyWidgetDefaults(widget)
```

Create the function:

```go
// applyWidgetDefaults sets common default values on any widget.
func applyWidgetDefaults(w element.Element) {
    // ConditionalVisibilitySettings: always set to nil unless already set
    // by applyConditionalSettingsGen (which only sets it when VisibleIf is present)
    type hasCondVis interface{ ConditionalVisibilitySettings() element.Element }
    type setCondVis interface{ SetConditionalVisibilitySettings(element.Element) }
    if w, ok := w.(setCondVis); ok {
        if cv, ok2 := w.(hasCondVis); !ok2 || cv.ConditionalVisibilitySettings() == nil {
            w.SetConditionalVisibilitySettings(nil)
        }
    }
    // TabIndex defaults to 0
    type setTabIdx interface{ SetTabIndex(int32) }
    if w, ok := w.(setTabIdx); ok {
        w.SetTabIndex(0)
    }
    // Appearance: set if nil
    type hasApp interface{ Appearance() element.Element }
    type setApp interface{ SetAppearance(element.Element) }
    if w, ok := w.(setApp); ok {
        if ha, ok2 := w.(hasApp); ok2 && ha.Appearance() == nil {
            w.SetAppearance(newDefaultAppearance())
        }
    }
}
```

Apply to the pluggable path as well (after line 386 or before each return).

- [ ] **Step C3.2: Compile and commit**

```bash
cd D:/gh/mxcli && go build ./mdl/executor/... 2>&1
git add mdl/executor/pages_builder_v3.go
git commit -m "fix(executor): add widget-level common defaults (CondVis, TabIndex, Appearance)"
```

---

### Task C4: Action button metadata defaults

**Files:**
- Modify: `mdl/executor/page_action_registry.go`

For `showPage`, `microflow`, and `nanoflow` actions, add missing metadata fields that Studio Pro always sets.

- [ ] **Step C4.1: Add `DisabledDuringExecution` to showPage/microflow/nanoflow actions**

In the `showPage` builder (line 71-88), after `act.SetPageSettings(ps)`:

```go
act.SetDisabledDuringExecution(false)
```

In the `microflow` builder (line 89-118), after `act.SetMicroflowSettings(settings)`:

```go
act.SetDisabledDuringExecution(false)
```

In the `nanoflow` builder (line 119-153), after building the act:

```go
act.SetDisabledDuringExecution(false)
```

Note: Verify the method name on each action type. The genPg types may use `SetDisabledDuringExecution(bool)` or similar.

- [ ] **Step C4.2: Add `NumberOfPagesToClose2` to showPage action**

```go
act.SetNumberOfPagesToClose2(1)
```

- [ ] **Step C4.3: Compile and commit**

```bash
cd D:/gh/mxcli && go build ./mdl/executor/... 2>&1
git add mdl/executor/page_action_registry.go
git commit -m "fix(executor): add action button metadata defaults"
```

---

## Verification

After all tasks:

1. **Build the entire project:**
```bash
cd D:/gh/mxcli && go build ./... 2>&1
```

2. **Generate mxgraph snapshot with widget index:**
```bash
cd D:/gh/mxcli && go run ./cmd/mxcli bson dump -p HelpDeskE2E/HelpDeskE2E.mpr -t workflow --list 2>&1
```
Expected: widget nodes included in the graph (verify via log output or by checking `FindNodes("Widget", nil)`).

3. **Verify widget lookup via mxgraph:**
After the graph is built, create a page with `PLUGGABLEWIDGET` — the engine should find the widget via mxgraph fast path.

4. **Verify page defaults via NDSL comparison:**
```bash
go run ./cmd/mxcli bson dump -p HelpDeskE2E/HelpDeskE2E.mpr -t page -o HD.Ticket_Overview --format ndsl 2>/dev/null
```
Expected: `ExportLevel`, `Appearance`, `CanvasWidth`, `CanvasHeight`, `ConditionalVisibilitySettings`, `TabIndex` etc. now present.

5. **Run mx check:**
```bash
mx check HelpDeskE2E/HelpDeskE2E.mpr 2>&1 | grep "\[error\]" | head -10
```
Expected: CE0109, CE6686 errors resolved.

---

## Dependency Graph

```
A1 (WidgetAdapter)  →  A2 (Registration)
                               ↓
B1 (MxGraph accessor)  →  B2 (WidgetRegistry fast path)  →  B3 (pageBuilder injection)
                                                                        ↓
C1 (page defaults)  ←  C2 (row/col defaults)  ←  C3 (widget defaults)  ←  C4 (action defaults)
```

Tasks in group A are independent of B and C. Tasks in group B depend on A (for the graph to contain widgets). Tasks in group C are independent of A and B.

Suggested execution order: A1 → A2 → B1 → B2 → B3 → C1 → C2 → C3 → C4
