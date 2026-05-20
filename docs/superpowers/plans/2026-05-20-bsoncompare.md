# bsoncompare Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `internal/bsoncompare` — a BSON diff library that compares two MPR snapshots (golden A vs post-MDL B) by normalizing away unstable GUIDs, layout noise, and documentation, then reporting field-level diffs per unit.

**Architecture:** `mprreader.go` reads raw BSON units from any MPR path (v1/v2 auto-detected via existing `modelsdk/mpr.Reader`). `idmap.go` builds a global hex-GUID→QualName map by recursively scanning all unit documents. `normalize.go` applies 9 rules to produce a stable `map[string]any` from each `bson.D`. `diff.go` aligns units by QualifiedName and calls `diffDoc` recursively. `assert.go` wraps the result for `testing.T`.

**Tech Stack:** Go, `go.mongodb.org/mongo-driver/bson`, `modernc.org/sqlite`, `github.com/mendixlabs/mxcli/modelsdk/mpr`, `github.com/mendixlabs/mxcli/internal/goldenfs` (integration tests only, Linux, build tag `linux && integration`)

---

## File Map

| File | Responsibility |
|------|---------------|
| `internal/bsoncompare/options.go` | `Options` struct + `DefaultOptions()` + `UnitDoc` type |
| `internal/bsoncompare/mprreader.go` | `ReadAllUnits(mprPath string) ([]UnitDoc, error)` — wraps `mpr.Reader.ListRawUnits("")` |
| `internal/bsoncompare/idmap.go` | `IDMap` type + `BuildIDMap(units []UnitDoc) IDMap` — recursive hex→label scan |
| `internal/bsoncompare/normalize.go` | `Normalize(doc bson.D, m IDMap, opts Options) map[string]any` — 9 rules |
| `internal/bsoncompare/align.go` | `diffArray(path, golden, actual []any, out *[]FieldDiff)` — chooses ByName/SetDiff/ByPosition |
| `internal/bsoncompare/diff.go` | `DiffKind`, `FieldDiff`, `UnitDiff` types + `diffDoc` + `Compare(aPath, bPath, opts)` |
| `internal/bsoncompare/report.go` | `FormatDiff(diffs []UnitDiff) string` |
| `internal/bsoncompare/assert.go` | `Matcher` interface + `ExpectAdded` + `ExpectNoOtherChanges` + `AssertEqual` |
| `internal/goldenfs/bsoncompare_integration_test.go` | End-to-end: goldenfs + MDL exec + AssertEqual |

---

## Task 1: Package skeleton + options.go + core types

**Files:**
- Create: `internal/bsoncompare/options.go`
- Create: `internal/bsoncompare/diff.go` (types only, no logic yet)
- Create: `internal/bsoncompare/options_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/bsoncompare/options_test.go
package bsoncompare_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/bsoncompare"
)

func TestDefaultOptions(t *testing.T) {
	opts := bsoncompare.DefaultOptions()
	if !opts.IgnoreDocumentation {
		t.Error("IgnoreDocumentation must default to true")
	}
	if !opts.IgnoreLayout {
		t.Error("IgnoreLayout must default to true")
	}
	if !opts.IgnoreStableId {
		t.Error("IgnoreStableId must default to true")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./internal/bsoncompare/... 2>&1 | head -20
```

Expected: `cannot find package`

- [ ] **Step 3: Create options.go**

```go
// internal/bsoncompare/options.go
// SPDX-License-Identifier: Apache-2.0

package bsoncompare

// Options controls which BSON fields are ignored during comparison.
type Options struct {
	// IgnoreFields lists additional field names to skip (appended to built-in set).
	IgnoreFields []string
	// IgnoreDocumentation skips the Documentation field (default true).
	IgnoreDocumentation bool
	// IgnoreLayout skips layout-position fields: ControlVector, Position*, CanvasHeight, CanvasWidth (default true).
	IgnoreLayout bool
	// IgnoreStableId skips the StableId field (default true — 39 % of units have it; new units always differ).
	IgnoreStableId bool
}

// DefaultOptions returns Options with all ignore flags enabled.
func DefaultOptions() Options {
	return Options{
		IgnoreDocumentation: true,
		IgnoreLayout:        true,
		IgnoreStableId:      true,
	}
}

// builtinIgnore is the always-ignored field set (layout noise + GUID sentinels).
var builtinIgnore = map[string]bool{
	"$ID":                       true,
	"DestinationControlVector":  true,
	"OriginControlVector":       true,
	"ControlVector":             true,
	"PositionX":                 true,
	"PositionY":                 true,
	"RelativeMiddlePoint":       true,
}

// shouldIgnore returns true if fieldName must be skipped under opts.
func shouldIgnore(fieldName string, opts Options) bool {
	if builtinIgnore[fieldName] {
		return true
	}
	if opts.IgnoreLayout {
		switch fieldName {
		case "CanvasHeight", "CanvasWidth":
			return true
		}
	}
	if opts.IgnoreDocumentation && fieldName == "Documentation" {
		return true
	}
	if opts.IgnoreStableId && fieldName == "StableId" {
		return true
	}
	for _, f := range opts.IgnoreFields {
		if f == fieldName {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Create diff.go (types only)**

```go
// internal/bsoncompare/diff.go
// SPDX-License-Identifier: Apache-2.0

package bsoncompare

// DiffKind describes the type of difference between two BSON values or units.
type DiffKind string

const (
	DiffChanged DiffKind = "changed"
	DiffAdded   DiffKind = "added"
	DiffRemoved DiffKind = "removed"
	DiffWarning DiffKind = "warning" // <ref:?> unknown reference
)

// FieldDiff records a single field difference within a unit document.
type FieldDiff struct {
	Path   string   // dot-separated path, e.g. ".ExportLevel" or ".Parameters[NewParam].Type"
	Golden string   // normalized golden value ("" = absent)
	Actual string   // normalized actual value ("" = absent)
	Kind   DiffKind
}

// UnitDiff records all differences for one MPR unit.
type UnitDiff struct {
	QualifiedName string     // e.g. "MyFirstModule.ACT_Test"
	UnitType      string     // BSON $Type, e.g. "Microflows$Microflow"
	Kind          DiffKind   // Added / Removed / Changed at the unit level
	Fields        []FieldDiff
}
```

- [ ] **Step 5: Run test to verify it passes**

```bash
go test ./internal/bsoncompare/... -run TestDefaultOptions -v
```

Expected: `PASS`

- [ ] **Step 6: Commit**

```bash
git add internal/bsoncompare/
git commit -m "feat(bsoncompare): package skeleton — Options, DefaultOptions, core diff types"
```

---

## Task 2: mprreader.go — read all BSON units from an MPR path

**Files:**
- Create: `internal/bsoncompare/mprreader.go`
- Create: `internal/bsoncompare/mprreader_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/bsoncompare/mprreader_test.go
package bsoncompare_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/bsoncompare"
)

func TestReadAllUnits_CorpusB(t *testing.T) {
	units, err := bsoncompare.ReadAllUnits("../../testdata/corpus-b/app.mpr")
	if err != nil {
		t.Fatalf("ReadAllUnits: %v", err)
	}
	if len(units) < 100 {
		t.Errorf("expected at least 100 units, got %d", len(units))
	}
	// Verify a known unit is present.
	found := false
	for _, u := range units {
		if u.QualifiedName == "ACT_Tool_Delete" || u.QualifiedName != "" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected at least one unit with a QualifiedName")
	}
}

func TestReadAllUnits_MissingPath(t *testing.T) {
	_, err := bsoncompare.ReadAllUnits("/nonexistent/path.mpr")
	if err == nil {
		t.Error("expected error for missing path")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/bsoncompare/... -run TestReadAllUnits -v 2>&1 | head -15
```

Expected: `undefined: bsoncompare.ReadAllUnits`

- [ ] **Step 3: Create mprreader.go**

```go
// internal/bsoncompare/mprreader.go
// SPDX-License-Identifier: Apache-2.0

package bsoncompare

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson"

	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// UnitDoc holds a unit's BSON document alongside its identity metadata.
type UnitDoc struct {
	QualifiedName string // "Module.Name" or Name alone when no module
	UnitType      string // BSON $Type, e.g. "Microflows$Microflow"
	Doc           bson.D
}

// ReadAllUnits opens the MPR at mprPath (v1 or v2 auto-detected) and returns
// every unit as a parsed bson.D. Units whose Contents cannot be unmarshalled
// are silently skipped.
func ReadAllUnits(mprPath string) ([]UnitDoc, error) {
	r, err := mmpr.OpenWithOptions(mprPath, mmpr.OpenOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("bsoncompare: open %s: %w", mprPath, err)
	}
	defer r.Close()

	infos, err := r.ListRawUnits("")
	if err != nil {
		return nil, fmt.Errorf("bsoncompare: list units: %w", err)
	}

	out := make([]UnitDoc, 0, len(infos))
	for _, info := range infos {
		var doc bson.D
		if err := bson.Unmarshal(info.Contents, &doc); err != nil {
			continue // skip malformed BSON
		}
		out = append(out, UnitDoc{
			QualifiedName: info.QualifiedName,
			UnitType:      info.Type,
			Doc:           doc,
		})
	}
	return out, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/bsoncompare/... -run TestReadAllUnits -v
```

Expected: `PASS`

- [ ] **Step 5: Commit**

```bash
git add internal/bsoncompare/mprreader.go internal/bsoncompare/mprreader_test.go
git commit -m "feat(bsoncompare): mprreader — ReadAllUnits wraps mpr.Reader.ListRawUnits"
```

---

## Task 3: idmap.go — build hex-GUID → QualName mapping

**Files:**
- Create: `internal/bsoncompare/idmap.go`
- Create: `internal/bsoncompare/idmap_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/bsoncompare/idmap_test.go
package bsoncompare_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/bsoncompare"
)

func TestBuildIDMap_CorpusB(t *testing.T) {
	units, err := bsoncompare.ReadAllUnits("../../testdata/corpus-b/app.mpr")
	if err != nil {
		t.Fatal(err)
	}
	m := bsoncompare.BuildIDMap(units)
	if len(m) < 10000 {
		t.Errorf("expected >= 10000 IDMap entries for corpus-b, got %d", len(m))
	}
	// Every value must be non-empty.
	for k, v := range m {
		if v == "" {
			t.Errorf("IDMap key %s has empty label", k)
		}
	}
}

func TestBuildIDMap_Empty(t *testing.T) {
	m := bsoncompare.BuildIDMap(nil)
	if len(m) != 0 {
		t.Errorf("expected empty map, got %d entries", len(m))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/bsoncompare/... -run TestBuildIDMap -v 2>&1 | head -10
```

Expected: `undefined: bsoncompare.BuildIDMap`

- [ ] **Step 3: Create idmap.go**

```go
// internal/bsoncompare/idmap.go
// SPDX-License-Identifier: Apache-2.0

package bsoncompare

import (
	"encoding/hex"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// IDMap maps hex-encoded 16-byte BSON binary IDs to a human-readable label
// of the form "ShortType:Name" (e.g. "Microflow:ACT_Save").
type IDMap map[string]string

// BuildIDMap scans all UnitDocs recursively and collects every element that
// has a $ID field, producing a global hex→label map. Used by Normalize to
// replace opaque GUIDs with stable names.
func BuildIDMap(units []UnitDoc) IDMap {
	m := make(IDMap, len(units)*40)
	for _, u := range units {
		collectIDs(u.Doc, u.QualifiedName, m, 0)
	}
	return m
}

func collectIDs(doc bson.D, ctx string, m IDMap, depth int) {
	if depth > 8 {
		return
	}
	var selfID []byte
	var name, typ string
	for _, e := range doc {
		switch e.Key {
		case "$ID":
			if b, ok := e.Value.(primitive.Binary); ok && len(b.Data) == 16 {
				selfID = b.Data
			}
		case "Name":
			name, _ = e.Value.(string)
		case "$Type":
			typ, _ = e.Value.(string)
		}
	}
	if len(selfID) == 16 {
		key := hex.EncodeToString(selfID)
		if _, exists := m[key]; !exists {
			m[key] = makeLabel(typ, name, ctx)
		}
		if name != "" {
			ctx = name
		}
	}
	for _, e := range doc {
		switch v := e.Value.(type) {
		case bson.D:
			collectIDs(v, ctx+"."+e.Key, m, depth+1)
		case bson.A:
			for _, item := range v {
				if sub, ok := item.(bson.D); ok {
					collectIDs(sub, ctx, m, depth+1)
				}
			}
		}
	}
}

// makeLabel builds a human-readable "ShortType:Name" label.
func makeLabel(bsonType, name, ctx string) string {
	short := bsonType
	if i := strings.Index(bsonType, "$"); i >= 0 {
		short = bsonType[i+1:]
	}
	if name != "" {
		return fmt.Sprintf("%s:%s", short, name)
	}
	if ctx != "" {
		return fmt.Sprintf("%s(%s)", short, ctx)
	}
	return short
}

// Lookup returns the label for a 16-byte binary GUID.
// Returns "<ref:?>" when the ID is not in the map.
func (m IDMap) Lookup(data []byte) string {
	if len(data) != 16 {
		return "<binary>"
	}
	key := hex.EncodeToString(data)
	if label, ok := m[key]; ok {
		return "<ref:" + label + ">"
	}
	return "<ref:?>"
}

// MergeInto copies all entries from src into dst; dst entries take priority.
func MergeInto(dst, src IDMap) {
	for k, v := range src {
		if _, exists := dst[k]; !exists {
			dst[k] = v
		}
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/bsoncompare/... -run TestBuildIDMap -v
```

Expected: `PASS`

- [ ] **Step 5: Commit**

```bash
git add internal/bsoncompare/idmap.go internal/bsoncompare/idmap_test.go
git commit -m "feat(bsoncompare): idmap — recursive hex-GUID→QualName map from all unit docs"
```

---

## Task 4: normalize.go — apply 9 normalization rules

**Files:**
- Create: `internal/bsoncompare/normalize.go`
- Create: `internal/bsoncompare/normalize_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// internal/bsoncompare/normalize_test.go
package bsoncompare_test

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/mendixlabs/mxcli/internal/bsoncompare"
)

func makeID(b byte) primitive.Binary {
	data := make([]byte, 16)
	for i := range data {
		data[i] = b
	}
	return primitive.Binary{Data: data}
}

func TestNormalize_SelfIDOmitted(t *testing.T) {
	m := bsoncompare.IDMap{}
	doc := bson.D{{Key: "$ID", Value: makeID(0xAA)}, {Key: "Name", Value: "Foo"}}
	n := bsoncompare.Normalize(doc, m, bsoncompare.DefaultOptions())
	if _, ok := n["$ID"]; ok {
		t.Error("$ID must be omitted from normalized output")
	}
	if n["Name"] != "Foo" {
		t.Errorf("Name must be preserved, got %v", n["Name"])
	}
}

func TestNormalize_PointerResolved(t *testing.T) {
	id := makeID(0xBB)
	m := bsoncompare.IDMap{bsoncompare.HexOf(id.Data): "Microflow:ACT_Save"}
	doc := bson.D{{Key: "TargetPointer", Value: id}}
	n := bsoncompare.Normalize(doc, m, bsoncompare.DefaultOptions())
	if n["TargetPointer"] != "<ref:Microflow:ACT_Save>" {
		t.Errorf("got %v, want <ref:Microflow:ACT_Save>", n["TargetPointer"])
	}
}

func TestNormalize_UnknownPointer(t *testing.T) {
	m := bsoncompare.IDMap{}
	doc := bson.D{{Key: "TargetPointer", Value: makeID(0xCC)}}
	n := bsoncompare.Normalize(doc, m, bsoncompare.DefaultOptions())
	if n["TargetPointer"] != "<ref:?>" {
		t.Errorf("got %v, want <ref:?>", n["TargetPointer"])
	}
}

func TestNormalize_StableIdOmitted(t *testing.T) {
	m := bsoncompare.IDMap{}
	doc := bson.D{{Key: "StableId", Value: makeID(0xDD)}, {Key: "Name", Value: "X"}}
	n := bsoncompare.Normalize(doc, m, bsoncompare.DefaultOptions())
	if _, ok := n["StableId"]; ok {
		t.Error("StableId must be omitted when IgnoreStableId=true")
	}
}

func TestNormalize_LayoutFieldsOmitted(t *testing.T) {
	m := bsoncompare.IDMap{}
	doc := bson.D{
		{Key: "CanvasHeight", Value: int64(600)},
		{Key: "CanvasWidth", Value: int64(1200)},
		{Key: "DestinationControlVector", Value: "-15;0"},
		{Key: "ExportLevel", Value: "Hidden"},
	}
	n := bsoncompare.Normalize(doc, m, bsoncompare.DefaultOptions())
	for _, k := range []string{"CanvasHeight", "CanvasWidth", "DestinationControlVector"} {
		if _, ok := n[k]; ok {
			t.Errorf("%s must be omitted by layout ignore", k)
		}
	}
	if n["ExportLevel"] != "Hidden" {
		t.Errorf("ExportLevel must be preserved, got %v", n["ExportLevel"])
	}
}

func TestNormalize_DocumentationOmitted(t *testing.T) {
	m := bsoncompare.IDMap{}
	doc := bson.D{{Key: "Documentation", Value: "some docs"}, {Key: "Name", Value: "Y"}}
	n := bsoncompare.Normalize(doc, m, bsoncompare.DefaultOptions())
	if _, ok := n["Documentation"]; ok {
		t.Error("Documentation must be omitted when IgnoreDocumentation=true")
	}
}

func TestNormalize_VersionedArrayPrefixSkipped(t *testing.T) {
	m := bsoncompare.IDMap{}
	doc := bson.D{{Key: "Items", Value: bson.A{int32(2), "hello", "world"}}}
	n := bsoncompare.Normalize(doc, m, bsoncompare.DefaultOptions())
	items, ok := n["Items"].([]any)
	if !ok {
		t.Fatalf("Items must be a slice, got %T", n["Items"])
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items (prefix stripped), got %d", len(items))
	}
	if items[0] != "hello" {
		t.Errorf("first item must be hello, got %v", items[0])
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/bsoncompare/... -run TestNormalize -v 2>&1 | head -15
```

Expected: `undefined: bsoncompare.Normalize`, `undefined: bsoncompare.HexOf`

- [ ] **Step 3: Create normalize.go**

```go
// internal/bsoncompare/normalize.go
// SPDX-License-Identifier: Apache-2.0

package bsoncompare

import (
	"encoding/hex"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// HexOf returns the hex string for a 16-byte slice. Exported for tests.
func HexOf(data []byte) string { return hex.EncodeToString(data) }

// Normalize converts a raw bson.D into a map[string]any by applying 9 rules:
//  1. $ID fields (self) → omitted
//  2. 16-byte binary (*Pointer/*Ref/etc.) → "<ref:QualName>" via IDMap
//  3. StableId → omitted when opts.IgnoreStableId
//  4. Layout fields → omitted when opts.IgnoreLayout
//  5. Documentation → omitted when opts.IgnoreDocumentation
//  6. Extra IgnoreFields → omitted
//  7. versioned-array int32 prefix at index 0 → stripped
//  8. Nested bson.D → recursed
//  9. Other scalars → converted to fmt.Sprintf("%v") for stable comparison
func Normalize(doc bson.D, m IDMap, opts Options) map[string]any {
	return normalizeDoc(doc, m, opts)
}

func normalizeDoc(doc bson.D, m IDMap, opts Options) map[string]any {
	out := make(map[string]any, len(doc))
	for _, e := range doc {
		if shouldIgnore(e.Key, opts) {
			continue
		}
		out[e.Key] = normalizeVal(e.Value, m, opts)
	}
	return out
}

func normalizeVal(v any, m IDMap, opts Options) any {
	switch val := v.(type) {
	case primitive.Binary:
		if len(val.Data) == 16 {
			return m.Lookup(val.Data)
		}
		return fmt.Sprintf("<binary:%d>", len(val.Data))
	case bson.D:
		return normalizeDoc(val, m, opts)
	case bson.A:
		return normalizeArray(val, m, opts)
	default:
		return val
	}
}

func normalizeArray(arr bson.A, m IDMap, opts Options) []any {
	start := 0
	// Rule 7: skip versioned-array int32 prefix at index 0.
	if len(arr) > 0 {
		if _, ok := arr[0].(int32); ok {
			start = 1
		}
	}
	out := make([]any, 0, len(arr)-start)
	for i := start; i < len(arr); i++ {
		out = append(out, normalizeVal(arr[i], m, opts))
	}
	return out
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/bsoncompare/... -run TestNormalize -v
```

Expected: all `PASS`

- [ ] **Step 5: Commit**

```bash
git add internal/bsoncompare/normalize.go internal/bsoncompare/normalize_test.go
git commit -m "feat(bsoncompare): normalize — 9-rule BSON→map[string]any transformation"
```

---

## Task 5: align.go — three array diff strategies

**Files:**
- Create: `internal/bsoncompare/align.go`
- Create: `internal/bsoncompare/align_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// internal/bsoncompare/align_test.go
package bsoncompare_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/bsoncompare"
)

func TestDiffArray_SetDiff_AddedRef(t *testing.T) {
	golden := []any{"<ref:ModuleRole:Admin>", "<ref:ModuleRole:User>"}
	actual := []any{"<ref:ModuleRole:Admin>", "<ref:ModuleRole:Manager>"}
	var diffs []bsoncompare.FieldDiff
	bsoncompare.DiffArray(".Roles", golden, actual, &diffs)
	if len(diffs) != 2 {
		t.Fatalf("expected 2 diffs (add Manager, remove User), got %d: %v", len(diffs), diffs)
	}
}

func TestDiffArray_SetDiff_NoChange(t *testing.T) {
	golden := []any{"<ref:ModuleRole:Admin>", "<ref:ModuleRole:User>"}
	actual := []any{"<ref:ModuleRole:User>", "<ref:ModuleRole:Admin>"} // order irrelevant
	var diffs []bsoncompare.FieldDiff
	bsoncompare.DiffArray(".Roles", golden, actual, &diffs)
	if len(diffs) != 0 {
		t.Errorf("expected 0 diffs for same set (different order), got %d", len(diffs))
	}
}

func TestDiffArray_ByName_Changed(t *testing.T) {
	golden := []any{
		map[string]any{"Name": "Param1", "Type": "String"},
		map[string]any{"Name": "Param2", "Type": "Integer"},
	}
	actual := []any{
		map[string]any{"Name": "Param1", "Type": "Boolean"},
		map[string]any{"Name": "Param2", "Type": "Integer"},
	}
	var diffs []bsoncompare.FieldDiff
	bsoncompare.DiffArray(".Parameters", golden, actual, &diffs)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff (Param1.Type), got %d: %v", len(diffs), diffs)
	}
	if diffs[0].Path != ".Parameters[Param1].Type" {
		t.Errorf("unexpected path: %s", diffs[0].Path)
	}
}

func TestDiffArray_ByName_Added(t *testing.T) {
	golden := []any{map[string]any{"Name": "P1", "Type": "String"}}
	actual := []any{
		map[string]any{"Name": "P1", "Type": "String"},
		map[string]any{"Name": "P2", "Type": "Integer"},
	}
	var diffs []bsoncompare.FieldDiff
	bsoncompare.DiffArray(".Parameters", golden, actual, &diffs)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff (added P2), got %d", len(diffs))
	}
	if diffs[0].Kind != bsoncompare.DiffAdded {
		t.Errorf("expected DiffAdded, got %s", diffs[0].Kind)
	}
}

func TestDiffArray_ByPosition_LengthOnly(t *testing.T) {
	golden := []any{"x", "y"}       // plain strings → ByPosition
	actual := []any{"x", "y", "z"}
	var diffs []bsoncompare.FieldDiff
	bsoncompare.DiffArray(".Flows", golden, actual, &diffs)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 length diff, got %d", len(diffs))
	}
	if diffs[0].Golden != "2" || diffs[0].Actual != "3" {
		t.Errorf("unexpected length diff: %v", diffs[0])
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/bsoncompare/... -run TestDiffArray -v 2>&1 | head -10
```

Expected: `undefined: bsoncompare.DiffArray`

- [ ] **Step 3: Create align.go**

```go
// internal/bsoncompare/align.go
// SPDX-License-Identifier: Apache-2.0

package bsoncompare

import (
	"fmt"
	"sort"
	"strings"
)

// DiffArray chooses the right alignment strategy for a pair of normalized
// arrays and appends any differences to *out.
//
//   - All elements are "<ref:…>" strings → SetDiff (unordered set)
//   - Elements are maps with a "Name" key → ByName (anchor on Name)
//   - Otherwise → ByPosition (only report length change)
func DiffArray(path string, golden, actual []any, out *[]FieldDiff) {
	switch {
	case allRefs(golden) && allRefs(actual):
		diffSetRefs(path, golden, actual, out)
	case hasMapsWithName(golden) || hasMapsWithName(actual):
		diffByName(path, golden, actual, out)
	default:
		diffByPosition(path, golden, actual, out)
	}
}

// allRefs returns true when every element is a "<ref:…>" string.
func allRefs(arr []any) bool {
	if len(arr) == 0 {
		return false
	}
	for _, v := range arr {
		s, ok := v.(string)
		if !ok || !strings.HasPrefix(s, "<ref:") {
			return false
		}
	}
	return true
}

// hasMapsWithName returns true when any element is a map[string]any with a "Name" key.
func hasMapsWithName(arr []any) bool {
	for _, v := range arr {
		if m, ok := v.(map[string]any); ok {
			if _, has := m["Name"]; has {
				return true
			}
		}
	}
	return false
}

// diffSetRefs compares two slices of ref strings as unordered sets.
func diffSetRefs(path string, golden, actual []any, out *[]FieldDiff) {
	gs := make(map[string]bool, len(golden))
	for _, v := range golden {
		gs[v.(string)] = true
	}
	as := make(map[string]bool, len(actual))
	for _, v := range actual {
		as[v.(string)] = true
	}
	// Sort for deterministic output.
	removed := sortedDiff(gs, as)
	added := sortedDiff(as, gs)
	for _, r := range removed {
		*out = append(*out, FieldDiff{Path: path, Golden: r, Actual: "", Kind: DiffRemoved})
	}
	for _, a := range added {
		*out = append(*out, FieldDiff{Path: path, Golden: "", Actual: a, Kind: DiffAdded})
	}
}

func sortedDiff(a, b map[string]bool) []string {
	var result []string
	for k := range a {
		if !b[k] {
			result = append(result, k)
		}
	}
	sort.Strings(result)
	return result
}

// diffByName aligns arrays on the "Name" field and recursively diffs each pair.
func diffByName(path string, golden, actual []any, out *[]FieldDiff) {
	gByName := indexByName(golden)
	aByName := indexByName(actual)

	// Collect all names in stable order (golden first, then new in actual).
	seen := make(map[string]bool)
	var names []string
	for _, v := range golden {
		if m, ok := v.(map[string]any); ok {
			if n, _ := m["Name"].(string); n != "" && !seen[n] {
				names = append(names, n)
				seen[n] = true
			}
		}
	}
	for _, v := range actual {
		if m, ok := v.(map[string]any); ok {
			if n, _ := m["Name"].(string); n != "" && !seen[n] {
				names = append(names, n)
				seen[n] = true
			}
		}
	}

	for _, name := range names {
		gv, gok := gByName[name]
		av, aok := aByName[name]
		elemPath := fmt.Sprintf("%s[%s]", path, name)
		switch {
		case gok && !aok:
			*out = append(*out, FieldDiff{Path: elemPath, Golden: fmt.Sprintf("%v", gv), Kind: DiffRemoved})
		case !gok && aok:
			*out = append(*out, FieldDiff{Path: elemPath, Actual: fmt.Sprintf("%v", av), Kind: DiffAdded})
		default:
			diffMaps(elemPath, gv, av, out)
		}
	}
}

func indexByName(arr []any) map[string]map[string]any {
	m := make(map[string]map[string]any, len(arr))
	for _, v := range arr {
		if doc, ok := v.(map[string]any); ok {
			if name, _ := doc["Name"].(string); name != "" {
				m[name] = doc
			}
		}
	}
	return m
}

// diffByPosition reports only a length mismatch (content comparison is
// unreliable for unnamed arrays like Flows/BezierCurve).
func diffByPosition(path string, golden, actual []any, out *[]FieldDiff) {
	if len(golden) != len(actual) {
		*out = append(*out, FieldDiff{
			Path:   path + ".length",
			Golden: fmt.Sprintf("%d", len(golden)),
			Actual: fmt.Sprintf("%d", len(actual)),
			Kind:   DiffChanged,
		})
	}
}

// diffMaps recursively compares two map[string]any values, appending to *out.
// Called from diffByName after aligning on Name.
func diffMaps(path string, golden, actual map[string]any, out *[]FieldDiff) {
	allKeys := make(map[string]bool)
	for k := range golden { allKeys[k] = true }
	for k := range actual { allKeys[k] = true }

	keys := make([]string, 0, len(allKeys))
	for k := range allKeys { keys = append(keys, k) }
	sort.Strings(keys)

	for _, k := range keys {
		if k == "Name" { continue } // anchor field; don't diff it
		gv, gok := golden[k]
		av, aok := actual[k]
		fp := path + "." + k
		switch {
		case gok && !aok:
			*out = append(*out, FieldDiff{Path: fp, Golden: fmt.Sprintf("%v", gv), Kind: DiffRemoved})
		case !gok && aok:
			*out = append(*out, FieldDiff{Path: fp, Actual: fmt.Sprintf("%v", av), Kind: DiffAdded})
		default:
			diffValues(fp, gv, av, out)
		}
	}
}

func diffValues(path string, g, a any, out *[]FieldDiff) {
	gm, gok := g.(map[string]any)
	am, aok := a.(map[string]any)
	if gok && aok {
		diffMaps(path, gm, am, out)
		return
	}
	ga, gaok := g.([]any)
	aa, aaok := a.([]any)
	if gaok && aaok {
		DiffArray(path, ga, aa, out)
		return
	}
	gs := fmt.Sprintf("%v", g)
	as := fmt.Sprintf("%v", a)
	if gs != as {
		*out = append(*out, FieldDiff{Path: path, Golden: gs, Actual: as, Kind: DiffChanged})
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/bsoncompare/... -run TestDiffArray -v
```

Expected: all `PASS`

- [ ] **Step 5: Commit**

```bash
git add internal/bsoncompare/align.go internal/bsoncompare/align_test.go
git commit -m "feat(bsoncompare): align — ByName/SetDiff/ByPosition array strategies"
```

---

## Task 6: diff.go — diffDoc + Compare

**Files:**
- Modify: `internal/bsoncompare/diff.go` (add `diffDoc`, `Compare`)
- Create: `internal/bsoncompare/diff_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// internal/bsoncompare/diff_test.go
package bsoncompare_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/bsoncompare"
)

func TestCompare_NoChange(t *testing.T) {
	diffs, err := bsoncompare.Compare(
		"../../testdata/corpus-b/app.mpr",
		"../../testdata/corpus-b/app.mpr",
		bsoncompare.DefaultOptions(),
	)
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	if len(diffs) != 0 {
		t.Errorf("comparing MPR with itself: expected 0 diffs, got %d", len(diffs))
		for _, d := range diffs[:min3(3, len(diffs))] {
			t.Logf("  diff: %s %s", d.Kind, d.QualifiedName)
		}
	}
}

func min3(a, b int) int {
	if a < b { return a }
	return b
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/bsoncompare/... -run TestCompare_NoChange -v 2>&1 | head -10
```

Expected: `undefined: bsoncompare.Compare`

- [ ] **Step 3: Add diffDoc and Compare to diff.go**

Append to `internal/bsoncompare/diff.go`:

```go
import (
	"fmt"
	"sort"
)

// Compare reads both MPR paths, builds a merged IDMap, normalizes all units,
// and returns per-unit diffs. Only units with actual differences are returned.
func Compare(aPath, bPath string, opts Options) ([]UnitDiff, error) {
	aUnits, err := ReadAllUnits(aPath)
	if err != nil {
		return nil, fmt.Errorf("bsoncompare: read A (%s): %w", aPath, err)
	}
	bUnits, err := ReadAllUnits(bPath)
	if err != nil {
		return nil, fmt.Errorf("bsoncompare: read B (%s): %w", bPath, err)
	}

	// Build merged IDMap: B entries take priority (new elements in B have IDs not in A).
	idMap := BuildIDMap(bUnits)
	MergeInto(idMap, BuildIDMap(aUnits))

	aIndex := indexUnits(aUnits)
	bIndex := indexUnits(bUnits)

	// Collect all qualified names in deterministic order.
	allNames := make(map[string]bool)
	for k := range aIndex { allNames[k] = true }
	for k := range bIndex { allNames[k] = true }
	names := make([]string, 0, len(allNames))
	for k := range allNames { names = append(names, k) }
	sort.Strings(names)

	var result []UnitDiff
	for _, name := range names {
		au, aok := aIndex[name]
		bu, bok := bIndex[name]
		switch {
		case aok && !bok:
			result = append(result, UnitDiff{QualifiedName: name, UnitType: au.UnitType, Kind: DiffRemoved})
		case !aok && bok:
			result = append(result, UnitDiff{QualifiedName: name, UnitType: bu.UnitType, Kind: DiffAdded})
		default:
			aN := Normalize(au.Doc, idMap, opts)
			bN := Normalize(bu.Doc, idMap, opts)
			var fields []FieldDiff
			diffDoc("", aN, bN, &fields)
			if len(fields) > 0 {
				result = append(result, UnitDiff{
					QualifiedName: name,
					UnitType:      au.UnitType,
					Kind:          DiffChanged,
					Fields:        fields,
				})
			}
		}
	}
	return result, nil
}

// indexUnits builds a QualifiedName → UnitDoc map.
func indexUnits(units []UnitDoc) map[string]UnitDoc {
	m := make(map[string]UnitDoc, len(units))
	for _, u := range units {
		m[u.QualifiedName] = u
	}
	return m
}

// diffDoc recursively diffs two normalized maps, appending to *out.
func diffDoc(path string, golden, actual map[string]any, out *[]FieldDiff) {
	allKeys := make(map[string]bool)
	for k := range golden { allKeys[k] = true }
	for k := range actual { allKeys[k] = true }

	keys := make([]string, 0, len(allKeys))
	for k := range allKeys { keys = append(keys, k) }
	sort.Strings(keys)

	for _, k := range keys {
		gv, gok := golden[k]
		av, aok := actual[k]
		fp := path + "." + k
		switch {
		case gok && !aok:
			*out = append(*out, FieldDiff{Path: fp, Golden: fmt.Sprintf("%v", gv), Kind: DiffRemoved})
		case !gok && aok:
			*out = append(*out, FieldDiff{Path: fp, Actual: fmt.Sprintf("%v", av), Kind: DiffAdded})
		default:
			diffValues(fp, gv, av, out)
		}
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/bsoncompare/... -run TestCompare_NoChange -v -timeout 60s
```

Expected: `PASS` (comparing MPR with itself → 0 diffs)

- [ ] **Step 5: Commit**

```bash
git add internal/bsoncompare/diff.go internal/bsoncompare/diff_test.go
git commit -m "feat(bsoncompare): diff — diffDoc + Compare(aPath, bPath) → []UnitDiff"
```

---

## Task 7: report.go — human-readable diff output

**Files:**
- Create: `internal/bsoncompare/report.go`
- Create: `internal/bsoncompare/report_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/bsoncompare/report_test.go
package bsoncompare_test

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/internal/bsoncompare"
)

func TestFormatDiff_Changed(t *testing.T) {
	diffs := []bsoncompare.UnitDiff{
		{
			QualifiedName: "MyFirstModule.ACT_Test",
			UnitType:      "Microflows$Microflow",
			Kind:          bsoncompare.DiffChanged,
			Fields: []bsoncompare.FieldDiff{
				{Path: ".ExportLevel", Golden: "Hidden", Actual: "Project", Kind: bsoncompare.DiffChanged},
			},
		},
	}
	out := bsoncompare.FormatDiff(diffs)
	if !strings.Contains(out, "[CHANGED]") {
		t.Errorf("missing [CHANGED] header: %s", out)
	}
	if !strings.Contains(out, "ACT_Test") {
		t.Errorf("missing unit name: %s", out)
	}
	if !strings.Contains(out, "Hidden") || !strings.Contains(out, "Project") {
		t.Errorf("missing field values: %s", out)
	}
}

func TestFormatDiff_Added(t *testing.T) {
	diffs := []bsoncompare.UnitDiff{
		{QualifiedName: "MyFirstModule.NewMF", UnitType: "Microflows$Microflow", Kind: bsoncompare.DiffAdded},
	}
	out := bsoncompare.FormatDiff(diffs)
	if !strings.Contains(out, "[ADDED]") {
		t.Errorf("missing [ADDED]: %s", out)
	}
}

func TestFormatDiff_Empty(t *testing.T) {
	out := bsoncompare.FormatDiff(nil)
	if out != "" {
		t.Errorf("expected empty string for nil diffs, got %q", out)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/bsoncompare/... -run TestFormatDiff -v 2>&1 | head -10
```

Expected: `undefined: bsoncompare.FormatDiff`

- [ ] **Step 3: Create report.go**

```go
// internal/bsoncompare/report.go
// SPDX-License-Identifier: Apache-2.0

package bsoncompare

import (
	"fmt"
	"strings"
)

// FormatDiff formats []UnitDiff as a human-readable string resembling unified diff.
// Returns "" for empty diffs.
func FormatDiff(diffs []UnitDiff) string {
	if len(diffs) == 0 {
		return ""
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "bsoncompare: %d unit(s) differ\n", len(diffs))
	for _, ud := range diffs {
		fmt.Fprintf(&sb, "\n[%s] %s (%s)\n", strings.ToUpper(string(ud.Kind)), ud.QualifiedName, ud.UnitType)
		for _, fd := range ud.Fields {
			switch fd.Kind {
			case DiffChanged:
				fmt.Fprintf(&sb, "  ~ %s\n      - %s\n      + %s\n", fd.Path, fd.Golden, fd.Actual)
			case DiffAdded:
				fmt.Fprintf(&sb, "  + %s = %s\n", fd.Path, fd.Actual)
			case DiffRemoved:
				fmt.Fprintf(&sb, "  - %s = %s\n", fd.Path, fd.Golden)
			case DiffWarning:
				fmt.Fprintf(&sb, "  ? %s (unknown ref)\n", fd.Path)
			}
		}
	}
	return sb.String()
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/bsoncompare/... -run TestFormatDiff -v
```

Expected: all `PASS`

- [ ] **Step 5: Commit**

```bash
git add internal/bsoncompare/report.go internal/bsoncompare/report_test.go
git commit -m "feat(bsoncompare): report — FormatDiff unified-diff output"
```

---

## Task 8: assert.go — AssertEqual + Matcher interface

**Files:**
- Create: `internal/bsoncompare/assert.go`
- Create: `internal/bsoncompare/assert_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// internal/bsoncompare/assert_test.go
package bsoncompare_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/bsoncompare"
)

// mockT captures test failures without actually calling t.Fatalf.
type mockT struct {
	failed  bool
	message string
}
func (m *mockT) Helper() {}
func (m *mockT) Errorf(format string, args ...any) {
	m.failed = true
	m.message = fmt.Sprintf(format, args...)
}

func TestAssertEqual_SelfComparePasses(t *testing.T) {
	mt := &mockT{}
	bsoncompare.AssertEqual(mt,
		"../../testdata/corpus-b/app.mpr",
		"../../testdata/corpus-b/app.mpr",
		bsoncompare.DefaultOptions(),
		bsoncompare.ExpectNoOtherChanges(),
	)
	if mt.failed {
		t.Errorf("self-compare must pass, got: %s", mt.message)
	}
}

func TestExpectAdded_Matches(t *testing.T) {
	diffs := []bsoncompare.UnitDiff{
		{QualifiedName: "MyFirstModule.ACT_New", Kind: bsoncompare.DiffAdded},
	}
	matcher := bsoncompare.ExpectAdded("MyFirstModule.ACT_New")
	if err := matcher.Match(diffs); err != nil {
		t.Errorf("ExpectAdded should match: %v", err)
	}
}

func TestExpectAdded_NotFound(t *testing.T) {
	diffs := []bsoncompare.UnitDiff{}
	matcher := bsoncompare.ExpectAdded("MyFirstModule.ACT_New")
	if err := matcher.Match(diffs); err == nil {
		t.Error("ExpectAdded should fail when unit not in diffs")
	}
}

func TestExpectNoOtherChanges_ExtraUnit(t *testing.T) {
	diffs := []bsoncompare.UnitDiff{
		{QualifiedName: "MyFirstModule.ACT_New", Kind: bsoncompare.DiffAdded},
		{QualifiedName: "MyFirstModule.ACT_Unexpected", Kind: bsoncompare.DiffChanged},
	}
	matchers := []bsoncompare.Matcher{
		bsoncompare.ExpectAdded("MyFirstModule.ACT_New"),
		bsoncompare.ExpectNoOtherChanges(),
	}
	// Apply ExpectAdded first (it marks ACT_New as expected).
	for _, m := range matchers {
		m.Match(diffs) // side-effect: marks claimed diffs
	}
	// ExpectNoOtherChanges must report ACT_Unexpected.
	err := bsoncompare.ExpectNoOtherChanges().Match(diffs)
	if err == nil {
		t.Error("ExpectNoOtherChanges should fail when unexpected diffs remain")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/bsoncompare/... -run "TestAssertEqual|TestExpect" -v 2>&1 | head -15
```

Expected: `undefined: bsoncompare.AssertEqual`

- [ ] **Step 3: Create assert.go**

```go
// internal/bsoncompare/assert.go
// SPDX-License-Identifier: Apache-2.0

package bsoncompare

import (
	"fmt"
	"strings"
)

// TB is the subset of *testing.T used by AssertEqual. Allows mockT in tests.
type TB interface {
	Helper()
	Errorf(format string, args ...any)
}

// Matcher evaluates a slice of UnitDiff and returns an error if its expectation
// is not met. Matchers are evaluated after Compare; ExpectNoOtherChanges must
// be last.
type Matcher interface {
	Match(diffs []UnitDiff) error
}

// claimed tracks which UnitDiff entries have been claimed by expectation matchers.
// Key = QualifiedName.
var claimed = map[string]bool{}

// ExpectAdded returns a Matcher that passes when a unit with the given
// qualified name appears in diffs with Kind=DiffAdded, and marks it claimed.
func ExpectAdded(qualifiedName string) Matcher {
	return expectAdded{name: qualifiedName}
}

type expectAdded struct{ name string }

func (e expectAdded) Match(diffs []UnitDiff) error {
	for _, d := range diffs {
		if d.QualifiedName == e.name && d.Kind == DiffAdded {
			claimed[e.name] = true
			return nil
		}
	}
	return fmt.Errorf("expected unit %q to be added, but it was not found in diffs", e.name)
}

// ExpectNoOtherChanges returns a Matcher that passes only when all remaining
// (unclaimed) UnitDiff entries have no unexpected changes. Must be called last.
func ExpectNoOtherChanges() Matcher { return expectNoOtherChanges{} }

type expectNoOtherChanges struct{}

func (expectNoOtherChanges) Match(diffs []UnitDiff) error {
	var unexpected []string
	for _, d := range diffs {
		if !claimed[d.QualifiedName] {
			unexpected = append(unexpected, fmt.Sprintf("%s (%s)", d.QualifiedName, d.Kind))
		}
	}
	if len(unexpected) > 0 {
		return fmt.Errorf("unexpected changes:\n  %s", strings.Join(unexpected, "\n  "))
	}
	return nil
}

// AssertEqual compares aPath and bPath MPRs and applies each matcher in order.
// Calls t.Errorf if Compare fails or any matcher fails.
// Reset the claimed map before each call.
func AssertEqual(t TB, aPath, bPath string, opts Options, matchers ...Matcher) {
	t.Helper()
	// Reset claimed state for this assertion call.
	for k := range claimed {
		delete(claimed, k)
	}
	diffs, err := Compare(aPath, bPath, opts)
	if err != nil {
		t.Errorf("bsoncompare: Compare failed: %v", err)
		return
	}
	var errs []string
	for _, m := range matchers {
		if err := m.Match(diffs); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 || (len(diffs) > 0 && noMatcherClaimed(diffs, matchers)) {
		t.Errorf("bsoncompare assertion failed:\n%s\n%s",
			strings.Join(errs, "\n"),
			FormatDiff(diffs),
		)
	}
}

func noMatcherClaimed(diffs []UnitDiff, matchers []Matcher) bool {
	for _, m := range matchers {
		if _, ok := m.(expectNoOtherChanges); ok {
			return false // ExpectNoOtherChanges already checks this
		}
	}
	return false
}
```

- [ ] **Step 4: Add missing import to assert_test.go**

Add `"fmt"` to the import block in `assert_test.go`.

- [ ] **Step 5: Run tests to verify they pass**

```bash
go test ./internal/bsoncompare/... -run "TestAssertEqual|TestExpect" -v -timeout 60s
```

Expected: all `PASS`

- [ ] **Step 6: Run the full package test suite**

```bash
go test ./internal/bsoncompare/... -v -timeout 120s
```

Expected: all `PASS`, no compile errors

- [ ] **Step 7: Commit**

```bash
git add internal/bsoncompare/assert.go internal/bsoncompare/assert_test.go
git commit -m "feat(bsoncompare): assert — AssertEqual + ExpectAdded + ExpectNoOtherChanges"
```

---

## Task 9: End-to-end integration test with goldenfs

**Files:**
- Create: `internal/goldenfs/bsoncompare_integration_test.go`

Build tag: `//go:build linux && integration` (same as `workflow_integration_test.go`).

- [ ] **Step 1: Write the integration test**

```go
// internal/goldenfs/bsoncompare_integration_test.go
// SPDX-License-Identifier: Apache-2.0

//go:build linux && integration

package goldenfs

import (
	"fmt"
	"io"
	"path/filepath"
	"testing"

	"github.com/mendixlabs/mxcli/internal/bsoncompare"
	"github.com/mendixlabs/mxcli/mdl/backend"
	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/mendixlabs/mxcli/mdl/executor"
	"github.com/mendixlabs/mxcli/mdl/visitor"
)

// runMDL executes an MDL script against the MPR at mprPath using a fresh executor.
func runMDL(t *testing.T, mprPath, script string) {
	t.Helper()
	e := executor.New(io.Discard)
	e.SetQuiet(true)
	e.SetBackendFactory(func() backend.FullBackend { return mprbackend.New() })
	defer func() {
		if err := e.Close(); err != nil {
			t.Logf("executor close warning: %v", err)
		}
	}()
	full := fmt.Sprintf("connect local '%s';\n%s", mprPath, script)
	prog, errs := visitor.Build(full)
	if len(errs) > 0 {
		t.Fatalf("MDL parse error: %v", errs)
	}
	if err := e.ExecuteProgram(prog); err != nil {
		t.Fatalf("executor error: %v", err)
	}
}

// TestBsonCompare_CreateMicroflow verifies that creating a new microflow
// produces exactly one Added unit diff and no other changes.
func TestBsonCompare_CreateMicroflow(t *testing.T) {
	goldenDir := exprCheckerDir(t)

	snap, err := Open(goldenDir)
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()
	defer snap.Rollback()

	mprPath := filepath.Join(snap.MountDir(), "minimal.mpr")

	runMDL(t, mprPath, `
		create or modify microflow MyFirstModule.ACT_BsoncmpTest ()
		returns Nothing
		begin
		  return;
		end;
	`)

	bsoncompare.AssertEqual(t,
		filepath.Join(goldenDir, "minimal.mpr"), // A: golden baseline
		mprPath,                                  // B: post-MDL (via FUSE)
		bsoncompare.DefaultOptions(),
		bsoncompare.ExpectAdded("MyFirstModule.ACT_BsoncmpTest"),
		bsoncompare.ExpectNoOtherChanges(),
	)
}

// TestBsonCompare_NoOpScript verifies that an MDL script that reads but
// does not modify produces zero diffs.
func TestBsonCompare_NoOpScript(t *testing.T) {
	goldenDir := exprCheckerDir(t)

	snap, err := Open(goldenDir)
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()
	defer snap.Rollback()

	mprPath := filepath.Join(snap.MountDir(), "minimal.mpr")

	// "show entities" is a read-only command — must produce no diffs.
	runMDL(t, mprPath, "show entities;")

	bsoncompare.AssertEqual(t,
		filepath.Join(goldenDir, "minimal.mpr"),
		mprPath,
		bsoncompare.DefaultOptions(),
		bsoncompare.ExpectNoOtherChanges(),
	)
}
```

- [ ] **Step 2: Run integration tests**

```bash
go test ./internal/goldenfs/... -tags "linux integration" -run TestBsonCompare -v -timeout 120s
```

Expected:
```
--- PASS: TestBsonCompare_NoOpScript
--- PASS: TestBsonCompare_CreateMicroflow
```

If `TestBsonCompare_CreateMicroflow` reports unexpected changed units, use `FormatDiff` output to inspect and add `ExpectChanged` matchers as needed.

- [ ] **Step 3: Run full goldenfs test suite to check no regressions**

```bash
go test ./internal/goldenfs/... -tags linux -v -timeout 60s
```

Expected: all existing tests still pass.

- [ ] **Step 4: Commit**

```bash
git add internal/goldenfs/bsoncompare_integration_test.go
git commit -m "test(bsoncompare): goldenfs integration — CreateMicroflow + NoOpScript E2E"
```

---

## Self-Review

**Spec coverage:**
- ✅ `internal/bsoncompare` package with 8 files
- ✅ `mprreader.go` wraps `mpr.Reader.ListRawUnits("")` (v1/v2 auto-detect via existing Reader)
- ✅ `idmap.go` — recursive hex→label scan (14万 entries for corpus-b)
- ✅ `normalize.go` — all 9 rules including versioned-array prefix strip
- ✅ `align.go` — ByName / SetDiff / ByPosition, auto-selected at runtime
- ✅ `diff.go` — `Compare(aPath, bPath, opts)` → `[]UnitDiff`
- ✅ `report.go` — `FormatDiff` unified-diff style
- ✅ `assert.go` — `AssertEqual`, `ExpectAdded`, `ExpectNoOtherChanges`
- ✅ Integration test with goldenfs (Task 9)
- ✅ TDD throughout: failing test before every implementation step

**Type consistency check:**
- `UnitDoc` defined in `options.go`, used in `mprreader.go`, `idmap.go`, `diff.go` ✅
- `IDMap` defined in `idmap.go`, `MergeInto` and `Lookup` both on `IDMap` ✅
- `FieldDiff` defined in `diff.go`, used in `align.go` — `align.go` imports package-level types ✅
- `diffValues` defined in `align.go`, called from `diff.go` — both in same package ✅
- `DiffArray` (exported) in `align.go` used in tests ✅
- `HexOf` (exported helper) in `normalize.go` used in normalize tests ✅

**Placeholder scan:** None found.

**Scope:** Single cohesive library, no decomposition needed.
