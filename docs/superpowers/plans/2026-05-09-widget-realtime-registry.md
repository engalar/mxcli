# Widget Real-Time Registry Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the `mxcli widget init` pre-extraction requirement with real-time MPK-derived widget definitions, so that `CREATE PAGE` works with project widgets without any setup step.

**Architecture:** `WidgetRegistry` gains a `SetProjectDir` method that pre-scans `widgets/*.mpk` (building a lightweight `mdlName→widgetID` map), then falls back to on-demand MPK parsing when `Get`/`GetByWidgetID` misses. Results are cached in the registry maps so the second lookup is O(1). Hand-crafted `.def.json` overrides still take priority.

**Tech Stack:** Go, `archive/zip` (stdlib), `sdk/widgets/mpk` (already imported nowhere in `mdl/executor/` — adding it is safe, no circular dependency).

---

## File Map

| File | Change |
|------|--------|
| `mdl/executor/widget_registry.go` | Add fields, methods, MPK fallback in Get/GetByWidgetID |
| `mdl/executor/widget_registry_mpk_test.go` | New test file (4 tests) |
| `mdl/executor/cmd_pages_builder.go` | Wire `SetProjectDir` into `initPluggableEngine` |
| `cmd/mxcli/cmd_widget.go` | Add `--force` flag, update help text, remove skip-if-exists guard |

---

## Task 1: Add MPK fallback to WidgetRegistry

**Files:**
- Modify: `mdl/executor/widget_registry.go`
- Create: `mdl/executor/widget_registry_mpk_test.go`

- [ ] **Step 1: Write the failing tests**

Create `mdl/executor/widget_registry_mpk_test.go`:

```go
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/sdk/widgets/mpk"
)

// writeMiniMPK creates a minimal .mpk ZIP in widgetsDir with the given widget ID.
func writeMiniMPK(t *testing.T, widgetsDir, widgetID string) {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	// last segment of ID becomes the XML file name
	parts := strings.Split(widgetID, ".")
	xmlName := parts[len(parts)-1] + ".xml"

	pkg, _ := w.Create("package.xml")
	fmt.Fprintf(pkg,
		`<package><clientModule name="Test" version="1.0.0"><widgetFiles><widgetFile path=%q/></widgetFiles></clientModule></package>`,
		xmlName,
	)

	wxml, _ := w.Create(xmlName)
	fmt.Fprintf(wxml,
		`<widget id=%q pluginWidget="true"><name>Test</name><properties><propertyGroup caption="General"><property key="caption" type="string" defaultValue="Hi"/></propertyGroup></properties></widget>`,
		widgetID,
	)

	w.Close()

	mpkPath := filepath.Join(widgetsDir, parts[len(parts)-1]+".mpk")
	if err := os.WriteFile(mpkPath, buf.Bytes(), 0644); err != nil {
		t.Fatalf("write MPK: %v", err)
	}
}

func TestRegistryMPKFallbackGet(t *testing.T) {
	mpk.ClearCache()
	dir := t.TempDir()
	widgetsDir := filepath.Join(dir, "widgets")
	os.MkdirAll(widgetsDir, 0755)
	writeMiniMPK(t, widgetsDir, "com.test.widget.Testwidget")

	reg, err := NewWidgetRegistry()
	if err != nil {
		t.Fatalf("NewWidgetRegistry: %v", err)
	}
	if err := reg.SetProjectDir(dir); err != nil {
		t.Fatalf("SetProjectDir: %v", err)
	}

	def, ok := reg.Get("TESTWIDGET")
	if !ok {
		t.Fatal("expected TESTWIDGET via MPK fallback, got not-found")
	}
	if def.WidgetID != "com.test.widget.Testwidget" {
		t.Errorf("WidgetID = %q, want com.test.widget.Testwidget", def.WidgetID)
	}
	if def.WidgetKind != "pluggable" {
		t.Errorf("WidgetKind = %q, want pluggable", def.WidgetKind)
	}
}

func TestRegistryMPKFallbackGetByWidgetID(t *testing.T) {
	mpk.ClearCache()
	dir := t.TempDir()
	widgetsDir := filepath.Join(dir, "widgets")
	os.MkdirAll(widgetsDir, 0755)
	writeMiniMPK(t, widgetsDir, "com.test.widget.Testwidget")

	reg, _ := NewWidgetRegistry()
	reg.SetProjectDir(dir)

	def, ok := reg.GetByWidgetID("com.test.widget.Testwidget")
	if !ok {
		t.Fatal("expected GetByWidgetID fallback to find widget")
	}
	if def.MDLName != "testwidget" {
		t.Errorf("MDLName = %q, want testwidget", def.MDLName)
	}
}

func TestRegistryMPKFallbackCached(t *testing.T) {
	mpk.ClearCache()
	dir := t.TempDir()
	widgetsDir := filepath.Join(dir, "widgets")
	os.MkdirAll(widgetsDir, 0755)
	writeMiniMPK(t, widgetsDir, "com.test.widget.Testwidget")

	reg, _ := NewWidgetRegistry()
	reg.SetProjectDir(dir)

	def1, ok1 := reg.Get("TESTWIDGET")
	if !ok1 {
		t.Fatal("first lookup failed")
	}
	def2, ok2 := reg.Get("TESTWIDGET")
	if !ok2 {
		t.Fatal("second lookup failed")
	}
	if def1 != def2 {
		t.Error("expected same pointer on second lookup (in-registry cache)")
	}
}

func TestRegistryMPKFallbackSkipsBuiltins(t *testing.T) {
	mpk.ClearCache()
	dir := t.TempDir()
	widgetsDir := filepath.Join(dir, "widgets")
	os.MkdirAll(widgetsDir, 0755)
	// Write a fake MPK that would derive "GALLERY" — should be ignored
	writeMiniMPK(t, widgetsDir, "com.mendix.widget.web.gallery.Gallery")

	reg, _ := NewWidgetRegistry()
	builtinDef, builtinOK := reg.Get("GALLERY")

	reg.SetProjectDir(dir)

	afterDef, afterOK := reg.Get("GALLERY")
	if builtinOK {
		// Gallery is a builtin: SetProjectDir must not replace it
		if afterDef != builtinDef {
			t.Error("SetProjectDir must not override a built-in definition")
		}
	} else {
		// Gallery not a builtin in this build: MPK fallback is fine
		if !afterOK {
			t.Error("expected GALLERY to be found after SetProjectDir")
		}
	}
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
cd D:/gh/mxcli
go test ./mdl/executor/ -run "TestRegistryMPK" -v 2>&1 | head -40
```

Expected: compilation error (`SetProjectDir` undefined) — confirms test is wired in correctly.

- [ ] **Step 3: Implement the changes in `widget_registry.go`**

**3a. Add the `mpk` import** — edit the import block at the top of the file:

```go
import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/sdk/widgets/definitions"
	"github.com/mendixlabs/mxcli/sdk/widgets/mpk"
)
```

**3b. Add two fields to `WidgetRegistry`** — replace the struct definition (lines 18-22):

```go
type WidgetRegistry struct {
	byMDLName       map[string]*WidgetDefinition // keyed by uppercase MDLName
	byWidgetID      map[string]*WidgetDefinition // keyed by widgetId
	knownOperations map[string]bool              // operations accepted during validation
	projectDir      string                       // project root for MPK fallback
	mpkNameMap      map[string]string            // uppercase MDLName → widgetID (pre-scan)
}
```

**3c. Initialise `mpkNameMap` in `NewWidgetRegistryWithOps`** — after `byWidgetID` (line 67 area):

```go
reg := &WidgetRegistry{
	byMDLName:       make(map[string]*WidgetDefinition),
	byWidgetID:      make(map[string]*WidgetDefinition),
	knownOperations: ops,
	mpkNameMap:      make(map[string]string),
}
```

**3d. Modify `Get`** — replace lines 107-110:

```go
// Get returns a widget definition by MDL name (case-insensitive).
// Falls back to real-time MPK derivation when SetProjectDir has been called.
func (r *WidgetRegistry) Get(mdlName string) (*WidgetDefinition, bool) {
	name := strings.ToUpper(mdlName)
	if def, ok := r.byMDLName[name]; ok {
		return def, ok
	}
	if r.projectDir == "" {
		return nil, false
	}
	widgetID, ok := r.mpkNameMap[name]
	if !ok {
		return nil, false
	}
	def, err := r.deriveFromMPK(widgetID)
	if err != nil {
		log.Printf("warning: MPK fallback for %s: %v", name, err)
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

**3e. Modify `GetByWidgetID`** — replace lines 113-116:

```go
// GetByWidgetID returns a widget definition by its full widget ID.
// Falls back to real-time MPK derivation when SetProjectDir has been called.
func (r *WidgetRegistry) GetByWidgetID(widgetID string) (*WidgetDefinition, bool) {
	if def, ok := r.byWidgetID[widgetID]; ok {
		return def, ok
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

**3f. Add new methods at the end of the file** (after `validateMappings`):

```go
// SetProjectDir enables real-time MPK fallback for this registry.
// It pre-scans widgets/*.mpk to build a lightweight mdlName→widgetID map
// (used by Get), and stores the dir for on-demand full parsing (used by both
// Get and GetByWidgetID). Safe to call multiple times; last call wins.
func (r *WidgetRegistry) SetProjectDir(projectDir string) error {
	r.projectDir = projectDir
	r.mpkNameMap = make(map[string]string)
	return r.preScanWidgets(projectDir)
}

// preScanWidgets does a full parse of all widgets/*.mpk files (results are
// cached by the mpk package) and records which MDL names are available.
// Built-in and user-override names already in byMDLName are skipped.
func (r *WidgetRegistry) preScanWidgets(projectDir string) error {
	widgetsDir := filepath.Join(projectDir, "widgets")
	matches, err := filepath.Glob(filepath.Join(widgetsDir, "*.mpk"))
	if err != nil {
		return fmt.Errorf("scan widgets dir: %w", err)
	}
	for _, mpkPath := range matches {
		defs, err := mpk.ParseAll(mpkPath)
		if err != nil {
			log.Printf("warning: widget pre-scan skipping %s: %v", filepath.Base(mpkPath), err)
			continue
		}
		for _, d := range defs {
			name := strings.ToUpper(lastIDSegment(d.ID))
			if _, exists := r.byMDLName[name]; exists {
				continue // builtin or user-override wins
			}
			r.mpkNameMap[name] = d.ID
		}
	}
	return nil
}

// deriveFromMPK parses the MPK for widgetID and returns a WidgetDefinition.
// Returns nil, nil when the widget is not found in the project's widgets/.
func (r *WidgetRegistry) deriveFromMPK(widgetID string) (*WidgetDefinition, error) {
	mpkPath, err := mpk.FindMPK(r.projectDir, widgetID)
	if err != nil {
		return nil, fmt.Errorf("find mpk for %s: %w", widgetID, err)
	}
	if mpkPath == "" {
		return nil, nil
	}
	mpkDef, err := mpk.ParseMPKForWidget(mpkPath, widgetID)
	if err != nil {
		return nil, fmt.Errorf("parse mpk for %s: %w", widgetID, err)
	}
	if mpkDef == nil {
		return nil, nil
	}
	return buildDefinitionFromMPK(mpkDef), nil
}

// lastIDSegment returns the last dot-separated segment of a widget ID, lowercased.
// e.g. "com.mendix.widget.web.gallery.Gallery" → "gallery"
func lastIDSegment(widgetID string) string {
	parts := strings.Split(widgetID, ".")
	return strings.ToLower(parts[len(parts)-1])
}

// buildDefinitionFromMPK converts an mpk.WidgetDefinition to an executor
// WidgetDefinition using the same inference logic as widget extract/init.
func buildDefinitionFromMPK(mpkDef *mpk.WidgetDefinition) *WidgetDefinition {
	mdlName := lastIDSegment(mpkDef.ID)
	widgetKind := "custom"
	if mpkDef.IsPluggable {
		widgetKind = "pluggable"
	}
	def := &WidgetDefinition{
		WidgetID:        mpkDef.ID,
		MDLName:         mdlName,
		WidgetKind:      widgetKind,
		TemplateFile:    mdlName + ".json",
		DefaultEditable: "Always",
	}

	var assocMappings []PropertyMapping
	for _, p := range mpkDef.Properties {
		switch p.Type {
		case "widgets":
			container := strings.ToUpper(p.Key)
			if p.Key == "content" {
				container = "TEMPLATE"
			}
			def.ChildSlots = append(def.ChildSlots, ChildSlotMapping{
				PropertyKey:  p.Key,
				MDLContainer: strings.ToLower(container),
				Operation:    "widgets",
			})
		case "datasource":
			def.PropertyMappings = append(def.PropertyMappings, PropertyMapping{
				PropertyKey: p.Key,
				Source:      "DataSource",
				Operation:   "datasource",
			})
		case "attribute":
			def.PropertyMappings = append(def.PropertyMappings, PropertyMapping{
				PropertyKey: p.Key,
				Source:      "Attribute",
				Operation:   "attribute",
			})
		case "association":
			assocMappings = append(assocMappings, PropertyMapping{
				PropertyKey: p.Key,
				Source:      "Association",
				Operation:   "association",
			})
		case "selection":
			def.PropertyMappings = append(def.PropertyMappings, PropertyMapping{
				PropertyKey: p.Key,
				Source:      "Selection",
				Operation:   "selection",
				Default:     p.DefaultValue,
			})
		case "boolean", "integer", "decimal", "string", "enumeration":
			m := PropertyMapping{
				PropertyKey: p.Key,
				Operation:   "primitive",
			}
			if p.DefaultValue != "" {
				m.Value = p.DefaultValue
			}
			def.PropertyMappings = append(def.PropertyMappings, m)
		}
	}
	def.PropertyMappings = append(def.PropertyMappings, assocMappings...)
	return def
}
```

- [ ] **Step 4: Run the tests**

```bash
cd D:/gh/mxcli
go test ./mdl/executor/ -run "TestRegistryMPK" -v
```

Expected output: 4 tests PASS.

- [ ] **Step 5: Run the full executor test suite**

```bash
go test ./mdl/executor/ -timeout 60s
```

Expected: all tests pass (no regressions).

- [ ] **Step 6: Commit**

```bash
git add mdl/executor/widget_registry.go mdl/executor/widget_registry_mpk_test.go
git commit -m "feat: add real-time MPK fallback to WidgetRegistry"
```

---

## Task 2: Wire SetProjectDir into pageBuilder

**Files:**
- Modify: `mdl/executor/cmd_pages_builder.go` lines 55-72 (`initPluggableEngine`)

- [ ] **Step 1: Add `filepath` import to `cmd_pages_builder.go`**

The file currently imports `context`, `fmt`, `log`, `strings` — check if `path/filepath` is already there. If not, add it to the import block:

```go
import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	// ... rest unchanged
)
```

- [ ] **Step 2: Update `initPluggableEngine` to call `SetProjectDir`**

Replace lines 55-72 with:

```go
func (pb *pageBuilder) initPluggableEngine() {
	if pb.pluggableEngine != nil || pb.pluggableEngineErr != nil {
		return
	}
	registry, err := NewWidgetRegistry()
	if err != nil {
		pb.pluggableEngineErr = mdlerrors.NewBackend("widget registry init", err)
		log.Printf("warning: %v", pb.pluggableEngineErr)
		return
	}
	if pb.backend != nil {
		if loadErr := registry.LoadUserDefinitions(pb.backend.Path()); loadErr != nil {
			log.Printf("warning: loading user widget definitions: %v", loadErr)
		}
		projectDir := filepath.Dir(pb.backend.Path())
		if scanErr := registry.SetProjectDir(projectDir); scanErr != nil {
			log.Printf("warning: widget pre-scan: %v", scanErr)
		}
	}
	pb.widgetRegistry = registry
	pb.pluggableEngine = NewPluggableWidgetEngine(pb.widgetBackend, pb)
}
```

- [ ] **Step 3: Build and test**

```bash
cd D:/gh/mxcli
make build
go test ./mdl/executor/ -timeout 60s
```

Expected: build succeeds, tests pass.

- [ ] **Step 4: Commit**

```bash
git add mdl/executor/cmd_pages_builder.go
git commit -m "feat: wire SetProjectDir into pageBuilder for runtime MPK fallback"
```

---

## Task 3: Update widget init command

**Files:**
- Modify: `cmd/mxcli/cmd_widget.go`

- [ ] **Step 1: Add `--force` flag and update help text**

Replace the `widgetInitCmd` declaration (lines 47-59) with:

```go
var widgetInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Dump widget definitions for inspection or customization",
	Long: `Scan the project's widgets/ directory and write .def.json files to
.mxcli/widgets/ for each .mpk.

Note: mxcli widget init is no longer required for CREATE PAGE to work.
Widget definitions are derived automatically at runtime from the project's
widgets/*.mpk files. Run this command only when you need to inspect or
hand-edit a widget's property mappings.

Existing .def.json files are skipped unless --force is given.

Requires --project (-p) to locate the project's widgets/ directory.`,
	Example: `  mxcli widget init -p /path/to/app.mpr
  mxcli widget init -p app.mpr --force`,
	RunE: runWidgetInit,
}
```

- [ ] **Step 2: Register the `--force` flag in `init()`**

In the `init()` function (around line 68), add after `widgetInitCmd.MarkFlagRequired("project")`:

```go
widgetInitCmd.Flags().Bool("force", false, "Overwrite existing .def.json files")
```

- [ ] **Step 3: Use the `--force` flag in `runWidgetInit`**

In `runWidgetInit` (line 220), read the flag right after reading `projectPath`:

```go
func runWidgetInit(cmd *cobra.Command, args []string) error {
	projectPath, _ := cmd.Flags().GetString("project")
	force, _ := cmd.Flags().GetBool("force")
	// ... rest of function unchanged until the skip-if-exists block
```

Then replace the skip-if-exists block (lines 265-269):

```go
			// Skip if already exists on disk (unless --force)
			if !force {
				if _, err := os.Stat(outPath); err == nil {
					skipped++
					continue
				}
			}
```

- [ ] **Step 4: Build and smoke test**

```bash
cd D:/gh/mxcli
make build
./bin/mxcli widget init --help
```

Expected output: shows new help text and `--force` flag listed.

- [ ] **Step 5: Commit**

```bash
git add cmd/mxcli/cmd_widget.go
git commit -m "feat: widget init adds --force flag, demotes to optional debug tool"
```

---

## Task 4: Full build and integration check

- [ ] **Step 1: Full build and test**

```bash
cd D:/gh/mxcli
make build && make test
```

Expected: all pass.

- [ ] **Step 2: Verify `widget list` shows no regression**

```bash
./bin/mxcli widget list
```

Expected: same built-in list as before (GALLERY, COMBOBOX, etc.).

- [ ] **Step 3: Smoke test with a real project (if available)**

If a `.mpr` project with a third-party widget is available:

```bash
./bin/mxcli -p /path/to/app.mpr -c "create page TestP (layout: Atlas_Default) { dataview dv (entity: Module.Entity) { } }"
```

Then try referencing a project widget by its derived name. No `widget init` should be needed.

- [ ] **Step 4: Final commit if any fixups were needed**

```bash
git add -p
git commit -m "fix: widget registry MPK fallback fixups"
```
