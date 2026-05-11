# MPK-Derived Widget Templates Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Any pluggable widget whose `.mpk` file is in `project/widgets/` can be used in MDL `create page` commands without a pre-built embedded template — mxcli derives the template from the MPK at runtime, transparently.

**Architecture:** New `sdk/widgets/generate.go` builds a `WidgetTemplate` (type + object shells) from a `mpk.WidgetDefinition` using existing `createPropertyPair` helpers from `augment.go`. A new internal `getOrGenerateTemplate` helper in `loader.go` adds a fallback path after the embedded-template cache miss, calling `mpk.FindMPK` → `mpk.ParseMPKForWidget` → `GenerateFromMPK`, with a session-scoped `sync.Map` to avoid re-parsing on every call.

**Tech Stack:** Go, `sdk/widgets/mpk` (existing ZIP parser), `sdk/widgets/augment.go` (existing property builders), `go.mongodb.org/mongo-driver/bson`

---

## File Map

| Action | File | What changes |
|--------|------|-------------|
| Modify | `sdk/widgets/mpk/mpk.go` | Fix `FindMPK` and add `ParseMPKForWidget` for multi-widget MPKs |
| Modify | `sdk/widgets/mpk/mpk_test.go` | Tests for multi-widget MPK handling |
| **Create** | `sdk/widgets/generate.go` | `GenerateFromMPK(def) *WidgetTemplate` |
| **Create** | `sdk/widgets/generate_test.go` | Unit tests for generate.go |
| Modify | `sdk/widgets/loader.go` | `getOrGenerateTemplate`, `generatedCache`, wire into BSON functions |
| Modify | `sdk/widgets/augment_test.go` | Add loader fallback integration test |

---

## Task 1: Fix multi-widget MPK support

**Context:** `FindMPK` calls `getWidgetIDFromMPK`, which reads only `WidgetFiles[0]` from `package.xml`. `ParseMPK` also reads only the first widget file. `CrusherWidgets.mpk` bundles 5 widgets (CavitySelector, CrusherSlider, etc.) — only the first would be discoverable. This must be fixed before anything else.

**Files:**
- Modify: `sdk/widgets/mpk/mpk.go`
- Modify: `sdk/widgets/mpk/mpk_test.go`

- [ ] **Step 1.1: Write failing test for multi-widget FindMPK**

In `sdk/widgets/mpk/mpk_test.go`, add after the existing helpers:

```go
func TestFindMPK_MultiWidget(t *testing.T) {
	mpkPath := filepath.Join("..", "..", "..", "D:/gh/posui/mendix/CrusherCopilot/widgets/CrusherWidgets.mpk")
	if _, err := os.Stat(mpkPath); err != nil {
		t.Skip("CrusherWidgets.mpk not available")
	}
	projectDir := filepath.Dir(mpkPath) // .../CrusherCopilot/widgets/ — wrong, use parent
	projectDir = filepath.Join("..", "..", "..", "D:/gh/posui/mendix/CrusherCopilot")
	if _, err := os.Stat(projectDir); err != nil {
		t.Skip("CrusherCopilot project not available")
	}

	widgets := []string{
		"com.mendix.widget.custom.CavitySelector.CavitySelector",
		"com.mendix.widget.custom.CrusherSlider.CrusherSlider",
		"com.mendix.widget.custom.PredictionBadge.PredictionBadge",
		"com.mendix.widget.custom.CrusherSimCanvas.CrusherSimCanvas",
		"com.mendix.widget.custom.HeatmapViz.HeatmapViz",
	}
	for _, wid := range widgets {
		found, err := FindMPK(projectDir, wid)
		if err != nil {
			t.Fatalf("FindMPK(%q): %v", wid, err)
		}
		if found == "" {
			t.Errorf("FindMPK(%q): expected MPK path, got empty string", wid)
		}
	}
}

func TestParseMPKForWidget_MultiWidget(t *testing.T) {
	mpkPath := filepath.Join("..", "..", "..", "D:/gh/posui/mendix/CrusherCopilot/widgets/CrusherWidgets.mpk")
	if _, err := os.Stat(mpkPath); err != nil {
		t.Skip("CrusherWidgets.mpk not available")
	}

	widgetID := "com.mendix.widget.custom.CavitySelector.CavitySelector"
	def, err := ParseMPKForWidget(mpkPath, widgetID)
	if err != nil {
		t.Fatalf("ParseMPKForWidget: %v", err)
	}
	if def == nil {
		t.Fatal("ParseMPKForWidget: got nil definition")
	}
	if def.ID != widgetID {
		t.Errorf("ID = %q, want %q", def.ID, widgetID)
	}
	if len(def.Properties) == 0 {
		t.Error("expected at least one property")
	}
}
```

- [ ] **Step 1.2: Run tests to confirm they fail**

```bash
cd D:/gh/mxcli
go test ./sdk/widgets/mpk/ -run "TestFindMPK_MultiWidget|TestParseMPKForWidget_MultiWidget" -v
```

Expected: FAIL — `ParseMPKForWidget` is undefined; `FindMPK` returns empty for all but the first widget.

- [ ] **Step 1.3: Add `getWidgetIDsFromMPK` (plural) and `ParseMPKForWidget` to mpk.go**

In `sdk/widgets/mpk/mpk.go`, replace `getWidgetIDFromMPK` with a plural version and add `ParseMPKForWidget`.

**Replace** the existing `getWidgetIDFromMPK` function (lines ~330-406) with:

```go
// getWidgetIDsFromMPK returns ALL widget IDs declared in an .mpk package.xml.
// Multi-widget MPKs (e.g. CrusherWidgets.mpk) list multiple <widgetFile> entries.
func getWidgetIDsFromMPK(mpkPath string) ([]string, error) {
	r, err := zip.OpenReader(mpkPath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var widgetFilePaths []string
	var totalExtracted uint64
	for _, f := range r.File {
		if f.Name == "package.xml" {
			if f.UncompressedSize64 > maxFileSize {
				return nil, fmt.Errorf("package.xml exceeds max file size (%d > %d)", f.UncompressedSize64, maxFileSize)
			}
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return nil, err
			}
			totalExtracted += uint64(len(data))
			if totalExtracted > maxTotalSize {
				return nil, fmt.Errorf("total extracted size exceeds limit")
			}
			var pkg xmlPackage
			if err := xml.Unmarshal(data, &pkg); err != nil {
				return nil, err
			}
			for _, wf := range pkg.ClientModule.WidgetFiles {
				widgetFilePaths = append(widgetFilePaths, wf.Path)
			}
			break
		}
	}

	var ids []string
	for _, path := range widgetFilePaths {
		for _, f := range r.File {
			if f.Name == path {
				if f.UncompressedSize64 > maxFileSize {
					continue
				}
				rc, err := f.Open()
				if err != nil {
					continue
				}
				data, err := io.ReadAll(rc)
				rc.Close()
				if err != nil {
					continue
				}
				totalExtracted += uint64(len(data))
				if totalExtracted > maxTotalSize {
					return ids, fmt.Errorf("total extracted size exceeds limit")
				}
				var widget struct {
					ID string `xml:"id,attr"`
				}
				if err := xml.Unmarshal(data, &widget); err != nil {
					continue
				}
				if widget.ID != "" {
					ids = append(ids, widget.ID)
				}
			}
		}
	}
	return ids, nil
}

// ParseMPKForWidget parses the widget XML for a specific widgetID from an .mpk file.
// Unlike ParseMPK (which reads the first widget), this finds the widget file whose
// parsed ID matches widgetID — needed for multi-widget .mpk packages.
func ParseMPKForWidget(mpkPath string, widgetID string) (*WidgetDefinition, error) {
	// Check definition cache (ParseMPK stores by mpkPath; we use widgetID as key here)
	defCacheLock.RLock()
	if def, ok := defCache[mpkPath+"\x00"+widgetID]; ok {
		defCacheLock.RUnlock()
		return def, nil
	}
	defCacheLock.RUnlock()

	r, err := zip.OpenReader(mpkPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open mpk: %w", err)
	}
	defer r.Close()

	// Parse package.xml to get all widget file paths and version
	var pkg xmlPackage
	var version string
	var totalExtracted uint64
	for _, f := range r.File {
		if f.Name == "package.xml" {
			if f.UncompressedSize64 > maxFileSize {
				return nil, fmt.Errorf("package.xml exceeds max size")
			}
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("open package.xml: %w", err)
			}
			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return nil, fmt.Errorf("read package.xml: %w", err)
			}
			totalExtracted += uint64(len(data))
			if err := xml.Unmarshal(data, &pkg); err != nil {
				return nil, fmt.Errorf("parse package.xml: %w", err)
			}
			version = pkg.ClientModule.Version
			break
		}
	}

	// Try each widget file until we find one with a matching ID
	for _, wf := range pkg.ClientModule.WidgetFiles {
		for _, f := range r.File {
			if f.Name != wf.Path {
				continue
			}
			if f.UncompressedSize64 > maxFileSize {
				continue
			}
			rc, err := f.Open()
			if err != nil {
				continue
			}
			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				continue
			}
			totalExtracted += uint64(len(data))
			if totalExtracted > maxTotalSize {
				return nil, fmt.Errorf("total extracted size exceeds limit")
			}

			var widget xmlWidget
			if err := xml.Unmarshal(data, &widget); err != nil {
				continue
			}
			if widget.ID != widgetID {
				continue
			}

			// Found the matching widget — build definition
			def := buildDefinition(&widget, version)

			cacheKey := mpkPath + "\x00" + widgetID
			defCacheLock.Lock()
			defCache[cacheKey] = def
			defCacheLock.Unlock()
			return def, nil
		}
	}

	return nil, nil // widget not found in this MPK
}
```

Note: `buildDefinition` is a helper that contains the repeated definition-building logic. Extract it from `ParseMPK` as shown in Step 1.4.

- [ ] **Step 1.4: Refactor ParseMPK to share definition-building logic**

In `sdk/widgets/mpk/mpk.go`, extract the widget-to-definition building from `ParseMPK` into a private `buildDefinition` function. Replace the block in `ParseMPK` starting from `def := &WidgetDefinition{...}` through the end:

Find in `ParseMPK` the block that creates `def` and returns it. Extract it as:

```go
// buildDefinition constructs a WidgetDefinition from a parsed xmlWidget and version string.
func buildDefinition(widget *xmlWidget, version string) *WidgetDefinition {
	def := &WidgetDefinition{
		ID:          widget.ID,
		Name:        widget.Name,
		Version:     version,
		IsPluggable: widget.PluginWidget == "true",
	}
	for _, group := range widget.PropertyGroups {
		collectProps(def, group, "")
	}
	return def
}
```

Then in `ParseMPK`, replace the definition-building code with:
```go
def := buildDefinition(&widget, version)
```

(The existing `collectProps` function in mpk.go is unchanged.)

- [ ] **Step 1.5: Update `FindMPK` to use `getWidgetIDsFromMPK`**

In `sdk/widgets/mpk/mpk.go`, in the `FindMPK` function, find the loop body where `getWidgetIDFromMPK` is called:

```go
for _, mpkPath := range matches {
    wid, err := getWidgetIDFromMPK(mpkPath)
    if err != nil {
        continue // Skip unparseable files
    }
    if wid != "" {
        dirMap[wid] = mpkPath
    }
}
```

Replace with:

```go
for _, mpkPath := range matches {
    wids, err := getWidgetIDsFromMPK(mpkPath)
    if err != nil {
        continue // Skip unparseable files
    }
    for _, wid := range wids {
        if wid != "" {
            dirMap[wid] = mpkPath
        }
    }
}
```

- [ ] **Step 1.6: Run tests to confirm they pass**

```bash
cd D:/gh/mxcli
go test ./sdk/widgets/mpk/ -run "TestFindMPK_MultiWidget|TestParseMPKForWidget_MultiWidget" -v
```

Expected: PASS (or SKIP if CrusherWidgets.mpk not present).

Also verify existing mpk tests still pass:
```bash
go test ./sdk/widgets/mpk/ -v
```

Expected: All PASS (or SKIP for external file tests).

- [ ] **Step 1.7: Commit**

```bash
cd D:/gh/mxcli
git add sdk/widgets/mpk/mpk.go sdk/widgets/mpk/mpk_test.go
git commit -m "fix: support multi-widget MPK files in FindMPK and ParseMPKForWidget"
```

---

## Task 2: Write generate.go (TDD)

**Context:** `GenerateFromMPK` builds a `WidgetTemplate` outer shell (`CustomWidgetType` + `WidgetObject`) and populates it by calling the existing `createPropertyPair` / `xmlTypeToBSONType` helpers from `augment.go`. All `$ID` values use `placeholderID()` — `loader.go`'s `collectIDs` will remap them to real UUIDs before BSON serialisation.

**Files:**
- Create: `sdk/widgets/generate_test.go`
- Create: `sdk/widgets/generate.go`

- [ ] **Step 2.1: Write failing tests**

Create `sdk/widgets/generate_test.go`:

```go
// SPDX-License-Identifier: Apache-2.0

package widgets

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/sdk/widgets/mpk"
)

func TestGenerateFromMPK_BasicTypes(t *testing.T) {
	ResetPlaceholderCounter()

	def := &mpk.WidgetDefinition{
		ID:      "com.example.Widget",
		Name:    "Test Widget",
		Version: "1.0.0",
		Properties: []mpk.PropertyDef{
			{Key: "label", Type: "string", Caption: "Label"},
			{Key: "enabled", Type: "boolean", Caption: "Enabled", DefaultValue: "true"},
			{Key: "count", Type: "integer", Caption: "Count", DefaultValue: "0"},
			{Key: "value", Type: "expression", Caption: "Value"},
			{Key: "attr", Type: "attribute", Caption: "Attribute"},
		},
	}

	tmpl := GenerateFromMPK(def)

	if tmpl == nil {
		t.Fatal("GenerateFromMPK returned nil")
	}
	if tmpl.WidgetID != def.ID {
		t.Errorf("WidgetID = %q, want %q", tmpl.WidgetID, def.ID)
	}
	if tmpl.Name != def.Name {
		t.Errorf("Name = %q, want %q", tmpl.Name, def.Name)
	}
	if tmpl.Version != def.Version {
		t.Errorf("Version = %q, want %q", tmpl.Version, def.Version)
	}
	if tmpl.Type == nil {
		t.Fatal("Type is nil")
	}
	if tmpl.Object == nil {
		t.Fatal("Object is nil")
	}

	// type.$Type must be CustomWidgets$CustomWidgetType
	if got := tmpl.Type["$Type"]; got != "CustomWidgets$CustomWidgetType" {
		t.Errorf("Type.$Type = %v, want CustomWidgets$CustomWidgetType", got)
	}

	// ObjectType must exist with PropertyTypes
	objType, ok := tmpl.Type["ObjectType"].(map[string]any)
	if !ok {
		t.Fatal("Type.ObjectType missing or wrong type")
	}
	propTypes, ok := objType["PropertyTypes"].([]any)
	if !ok {
		t.Fatal("ObjectType.PropertyTypes missing or wrong type")
	}
	// First element is the Mendix array version marker (float64(2))
	nonMarkerPropTypes := 0
	for _, pt := range propTypes {
		if _, isFloat := pt.(float64); !isFloat {
			nonMarkerPropTypes++
		}
	}
	if nonMarkerPropTypes != 5 {
		t.Errorf("PropertyTypes count = %d, want 5", nonMarkerPropTypes)
	}

	// Object Properties must match PropertyTypes count
	objProps, ok := tmpl.Object["Properties"].([]any)
	if !ok {
		t.Fatal("Object.Properties missing or wrong type")
	}
	nonMarkerProps := 0
	for _, p := range objProps {
		if _, isFloat := p.(float64); !isFloat {
			nonMarkerProps++
		}
	}
	if nonMarkerProps != 5 {
		t.Errorf("Properties count = %d, want 5", nonMarkerProps)
	}
}

func TestGenerateFromMPK_TypePointerCrossReference(t *testing.T) {
	ResetPlaceholderCounter()

	def := &mpk.WidgetDefinition{
		ID:   "com.example.Widget",
		Name: "Test Widget",
		Properties: []mpk.PropertyDef{
			{Key: "mode", Type: "enumeration", Caption: "Mode", DefaultValue: "fast"},
		},
	}

	tmpl := GenerateFromMPK(def)

	// Collect PropertyType $IDs
	objType := tmpl.Type["ObjectType"].(map[string]any)
	propTypes := objType["PropertyTypes"].([]any)
	var ptID string
	for _, pt := range propTypes {
		ptMap, ok := pt.(map[string]any)
		if !ok {
			continue
		}
		if ptMap["PropertyKey"] == "mode" {
			ptID = ptMap["$ID"].(string)
			break
		}
	}
	if ptID == "" {
		t.Fatal("PropertyType for 'mode' not found")
	}

	// Find Property and verify TypePointer → PropertyType.$ID
	objProps := tmpl.Object["Properties"].([]any)
	var propTypePointer string
	for _, p := range objProps {
		pMap, ok := p.(map[string]any)
		if !ok {
			continue
		}
		propTypePointer = pMap["TypePointer"].(string)
		break
	}
	if propTypePointer != ptID {
		t.Errorf("Property.TypePointer = %q, want PropertyType.$ID %q", propTypePointer, ptID)
	}
}

func TestGenerateFromMPK_NestedObject(t *testing.T) {
	ResetPlaceholderCounter()

	def := &mpk.WidgetDefinition{
		ID:   "com.example.Widget",
		Name: "Test Widget",
		Properties: []mpk.PropertyDef{
			{
				Key:     "columns",
				Type:    "object",
				Caption: "Columns",
				IsList:  true,
				Children: []mpk.PropertyDef{
					{Key: "header", Type: "string", Caption: "Header"},
					{Key: "attr", Type: "attribute", Caption: "Attribute"},
				},
			},
		},
	}

	tmpl := GenerateFromMPK(def)

	objType := tmpl.Type["ObjectType"].(map[string]any)
	propTypes := objType["PropertyTypes"].([]any)
	var columnsPT map[string]any
	for _, pt := range propTypes {
		ptMap, ok := pt.(map[string]any)
		if !ok {
			continue
		}
		if ptMap["PropertyKey"] == "columns" {
			columnsPT = ptMap
			break
		}
	}
	if columnsPT == nil {
		t.Fatal("columns PropertyType not found")
	}

	vt, ok := columnsPT["ValueType"].(map[string]any)
	if !ok {
		t.Fatal("ValueType missing on columns property")
	}
	nestedObjType, ok := vt["ObjectType"].(map[string]any)
	if !ok {
		t.Fatal("ObjectType missing on columns ValueType — nested object not built")
	}
	nestedPTs, ok := nestedObjType["PropertyTypes"].([]any)
	if !ok {
		t.Fatal("nested PropertyTypes missing")
	}
	nestedCount := 0
	for _, npt := range nestedPTs {
		if _, isFloat := npt.(float64); !isFloat {
			nestedCount++
		}
	}
	if nestedCount != 2 {
		t.Errorf("nested PropertyTypes count = %d, want 2", nestedCount)
	}
}

func TestGenerateFromMPK_UnknownTypeSkipped(t *testing.T) {
	ResetPlaceholderCounter()

	def := &mpk.WidgetDefinition{
		ID:   "com.example.Widget",
		Name: "Test Widget",
		Properties: []mpk.PropertyDef{
			{Key: "good", Type: "string", Caption: "Good"},
			{Key: "bad", Type: "unknownXmlType", Caption: "Bad"},
		},
	}

	tmpl := GenerateFromMPK(def)

	objType := tmpl.Type["ObjectType"].(map[string]any)
	propTypes := objType["PropertyTypes"].([]any)
	count := 0
	for _, pt := range propTypes {
		if _, isFloat := pt.(float64); !isFloat {
			count++
		}
	}
	if count != 1 {
		t.Errorf("PropertyTypes count = %d, want 1 (unknown type skipped)", count)
	}
}

func TestGenerateFromMPK_PlaceholderIDsRemapped(t *testing.T) {
	ResetPlaceholderCounter()

	def := &mpk.WidgetDefinition{
		ID:   "com.example.Widget",
		Name: "Test Widget",
		Properties: []mpk.PropertyDef{
			{Key: "label", Type: "string", Caption: "Label"},
		},
	}

	tmpl := GenerateFromMPK(def)

	// After GetTemplateFullBSON, no aa000000-prefix IDs should remain
	callCount := 0
	idGen := func() string {
		callCount++
		return strings.Repeat("f", 32) // deterministic non-placeholder IDs
	}

	// Temporarily insert into template cache so GetTemplateFullBSON finds it
	templateCacheLock.Lock()
	templateCache["com.example.Widget"] = tmpl
	templateCacheLock.Unlock()
	defer func() {
		templateCacheLock.Lock()
		delete(templateCache, "com.example.Widget")
		templateCacheLock.Unlock()
	}()

	bsonType, bsonObj, _, _, err := GetTemplateFullBSON("com.example.Widget", idGen, "")
	if err != nil {
		t.Fatalf("GetTemplateFullBSON: %v", err)
	}
	if containsPlaceholderID(bsonType) {
		t.Error("placeholder IDs leaked in bsonType")
	}
	if bsonObj != nil && containsPlaceholderID(bsonObj) {
		t.Error("placeholder IDs leaked in bsonObj")
	}
}
```

- [ ] **Step 2.2: Run tests to confirm they fail**

```bash
cd D:/gh/mxcli
go test ./sdk/widgets/ -run "TestGenerateFromMPK" -v
```

Expected: FAIL — `GenerateFromMPK` is undefined.

- [ ] **Step 2.3: Implement generate.go**

Create `sdk/widgets/generate.go`:

```go
// SPDX-License-Identifier: Apache-2.0

package widgets

import "github.com/mendixlabs/mxcli/sdk/widgets/mpk"

// GenerateFromMPK builds a complete WidgetTemplate from a parsed MPK WidgetDefinition.
// All $IDs are placeholder IDs (aa000000... prefix); loader.go's collectIDs remaps them
// to real UUIDs before BSON serialisation — matching the lifecycle of embedded templates.
// System properties (Label, Visibility, Editability) are not added; Studio Pro injects them.
func GenerateFromMPK(def *mpk.WidgetDefinition) *WidgetTemplate {
	typeID := placeholderID()
	objTypeID := placeholderID()

	var propTypes []any
	var objProps []any
	propTypes = append(propTypes, float64(2)) // Mendix array version marker
	objProps = append(objProps, float64(2))

	for _, p := range def.Properties {
		bsonType := xmlTypeToBSONType(p.Type)
		if bsonType == "" {
			continue // unknown XML type — skip silently
		}
		pt, prop := createPropertyPair(p, bsonType)
		if pt != nil {
			propTypes = append(propTypes, pt)
		}
		if prop != nil {
			objProps = append(objProps, prop)
		}
	}

	typeMap := map[string]any{
		"$ID":      typeID,
		"$Type":    "CustomWidgets$CustomWidgetType",
		"WidgetId": def.ID,
		"ObjectType": map[string]any{
			"$ID":           objTypeID,
			"$Type":         "CustomWidgets$WidgetObjectType",
			"PropertyTypes": propTypes,
		},
	}

	objectMap := map[string]any{
		"$ID":         placeholderID(),
		"$Type":       "CustomWidgets$WidgetObject",
		"TypePointer": typeID,
		"Properties":  objProps,
	}

	return &WidgetTemplate{
		WidgetID: def.ID,
		Name:     def.Name,
		Version:  def.Version,
		Type:     typeMap,
		Object:   objectMap,
	}
}
```

- [ ] **Step 2.4: Run tests to confirm they pass**

```bash
cd D:/gh/mxcli
go test ./sdk/widgets/ -run "TestGenerateFromMPK" -v
```

Expected: All 5 tests PASS.

- [ ] **Step 2.5: Commit**

```bash
cd D:/gh/mxcli
git add sdk/widgets/generate.go sdk/widgets/generate_test.go
git commit -m "feat: add GenerateFromMPK — build WidgetTemplate from MPK definition"
```

---

## Task 3: Wire loader.go — transparent fallback

**Context:** `GetTemplateBSON` and `GetTemplateFullBSON` both call `GetTemplate(widgetID)` which returns `nil, nil` on miss. We add a new `getOrGenerateTemplate(widgetID, projectPath)` helper that adds the MPK fallback path after the embedded-template miss. Both BSON functions switch to calling this helper instead.

**Files:**
- Modify: `sdk/widgets/loader.go`

- [ ] **Step 3.1: Write failing integration test**

In `sdk/widgets/augment_test.go`, add at the bottom:

```go
func TestGetTemplateFullBSON_FallsBackToMPK(t *testing.T) {
	crusherProjectPath := "D:/gh/posui/mendix/CrusherCopilot"
	if _, err := os.Stat(crusherProjectPath); err != nil {
		t.Skip("CrusherCopilot project not available")
	}

	widgetID := "com.mendix.widget.custom.CavitySelector.CavitySelector"

	// Must not be in embedded templates
	embedded, err := GetTemplate(widgetID)
	if err != nil {
		t.Fatalf("GetTemplate: %v", err)
	}
	if embedded != nil {
		t.Skip("CavitySelector is now an embedded template — test no longer relevant")
	}

	idGen := func() string { return strings.Repeat("a", 32) }
	bsonType, bsonObj, propTypeIDs, _, err := GetTemplateFullBSON(widgetID, idGen, crusherProjectPath)
	if err != nil {
		t.Fatalf("GetTemplateFullBSON fallback: %v", err)
	}
	if bsonType == nil {
		t.Fatal("expected non-nil bsonType from MPK fallback")
	}
	if bsonObj == nil {
		t.Fatal("expected non-nil bsonObj from MPK fallback")
	}
	if len(propTypeIDs) == 0 {
		t.Error("expected non-empty propTypeIDs")
	}
}
```

Also add the `os` and `strings` imports if missing from `augment_test.go`.

- [ ] **Step 3.2: Run test to confirm it fails**

```bash
cd D:/gh/mxcli
go test ./sdk/widgets/ -run "TestGetTemplateFullBSON_FallsBackToMPK" -v
```

Expected: FAIL — `GetTemplateFullBSON` returns `nil` for unknown widget (no fallback yet).

- [ ] **Step 3.3: Add generatedCache and getOrGenerateTemplate to loader.go**

In `sdk/widgets/loader.go`, add after the `templateCache` declaration block (around line 117):

```go
// generatedCache stores MPK-derived templates for the session lifetime.
// Key: widgetID string. Value: *WidgetTemplate (placeholder IDs, not yet remapped).
var generatedCache sync.Map
```

Then add the new helper function. Insert before `GetTemplateBSON` (around line 208):

```go
// getOrGenerateTemplate returns a WidgetTemplate for widgetID. It checks the embedded
// template cache first, then falls back to deriving a template from the project's .mpk
// widget file. Returns nil, nil when the widget is unknown and no MPK is available.
func getOrGenerateTemplate(widgetID, projectPath string) (*WidgetTemplate, error) {
	// 1. Embedded templates (existing path)
	if tmpl, err := GetTemplate(widgetID); err != nil || tmpl != nil {
		return tmpl, err
	}

	// 2. Session cache of previously generated templates
	if cached, ok := generatedCache.Load(widgetID); ok {
		return cached.(*WidgetTemplate), nil
	}

	// 3. Derive from MPK in project/widgets/
	if projectPath == "" {
		return nil, nil
	}
	mpkPath, err := mpk.FindMPK(projectPath, widgetID)
	if err != nil {
		return nil, fmt.Errorf("widget %q: scan MPK directory: %w", widgetID, err)
	}
	if mpkPath == "" {
		return nil, nil // no MPK found — caller treats nil as "widget unknown"
	}
	def, err := mpk.ParseMPKForWidget(mpkPath, widgetID)
	if err != nil {
		return nil, fmt.Errorf("widget %q: parse MPK: %w", widgetID, err)
	}
	if def == nil {
		return nil, nil
	}
	tmpl := GenerateFromMPK(def)
	generatedCache.Store(widgetID, tmpl)
	return tmpl, nil
}
```

- [ ] **Step 3.4: Update GetTemplateBSON to use getOrGenerateTemplate**

In `sdk/widgets/loader.go`, in `GetTemplateBSON` (around line 208), replace:

```go
tmpl, err := GetTemplate(widgetID)
if err != nil {
    return nil, nil, err
}
if tmpl == nil {
    return nil, nil, nil
}
```

with:

```go
tmpl, err := getOrGenerateTemplate(widgetID, projectPath)
if err != nil {
    return nil, nil, err
}
if tmpl == nil {
    return nil, nil, nil
}
```

- [ ] **Step 3.5: Update GetTemplateFullBSON to use getOrGenerateTemplate**

In `sdk/widgets/loader.go`, in `GetTemplateFullBSON` (around line 240), replace:

```go
tmpl, err := GetTemplate(widgetID)
if err != nil {
    return nil, nil, nil, "", err
}
if tmpl == nil {
    return nil, nil, nil, "", nil
}
```

with:

```go
tmpl, err := getOrGenerateTemplate(widgetID, projectPath)
if err != nil {
    return nil, nil, nil, "", err
}
if tmpl == nil {
    return nil, nil, nil, "", nil
}
```

- [ ] **Step 3.6: Run tests**

```bash
cd D:/gh/mxcli
go test ./sdk/widgets/ -run "TestGetTemplateFullBSON_FallsBackToMPK" -v
```

Expected: PASS (or SKIP if CrusherCopilot not present).

Run full widget package tests to check for regressions:
```bash
go test ./sdk/widgets/... -v 2>&1 | tail -20
```

Expected: All PASS (file-dependent tests may SKIP).

- [ ] **Step 3.7: Build and verify no compilation errors**

```bash
cd D:/gh/mxcli
make build
```

Expected: Build succeeds with no errors.

- [ ] **Step 3.8: Commit**

```bash
cd D:/gh/mxcli
git add sdk/widgets/loader.go sdk/widgets/augment_test.go
git commit -m "feat: fall back to MPK-derived template for unknown pluggable widgets"
```

---

## Task 4: End-to-end verification

**Context:** Confirm that a `create page` MDL command referencing a Crusher widget (which has no embedded template) succeeds without error and produces an MPR that Studio Pro accepts without CE0463.

**Files:** No new files — run existing CLI against CrusherCopilot.

- [ ] **Step 4.1: Run make test**

```bash
cd D:/gh/mxcli
make test
```

Expected: All tests pass.

- [ ] **Step 4.2: Smoke-test CLI against CrusherCopilot**

```bash
cd D:/gh/mxcli
./bin/mxcli -p "D:/gh/posui/mendix/CrusherCopilot/CrusherCopilot.mpr" -c \
  "show widgets"
```

Expected: Output lists widget types including CavitySelector and other Crusher widgets (resolved from MPK).

- [ ] **Step 4.3: Commit if any fixups needed, otherwise done**

If the smoke test surfaces a BSON issue, debug using `.claude/skills/debug-bson.md`.
Final commit if any fixups were made:

```bash
git add -p
git commit -m "fix: <describe the specific fix>"
```

---

## Self-Review Notes

- **Spec coverage:** Multi-widget MPK (Task 1) ✓ · generate.go (Task 2) ✓ · loader.go fallback (Task 3) ✓ · session cache (Task 3) ✓ · performance fallback deferred (out of scope per spec) ✓ · 5 named tests (Tasks 2+3) ✓ · acceptance criterion end-to-end (Task 4) ✓
- **Type consistency:** `GenerateFromMPK` signature is `(def *mpk.WidgetDefinition) *WidgetTemplate` — used identically in generate.go and loader.go · `ParseMPKForWidget` signature `(mpkPath, widgetID string) (*WidgetDefinition, error)` — consistent across mpk.go and loader.go · `FindMPK` returns `(string, error)` — error correctly handled in `getOrGenerateTemplate`
- **No placeholders:** All steps contain complete code. No TBDs.
