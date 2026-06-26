# bsoncompare modelsdk Migration — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rewrite bsoncompare to use `element.Element` / `property.Property` / `codec.Decoder` instead of raw `go.mongodb.org/mongo-driver/v2/bson` types.

**Architecture:** Two-layer change: (1) add `ListUnitIdentities` + `DecodeBytes` to modelsdk, (2) rewrite all bsoncompare internals to use modelsdk's typed API for comparison while keeping `ContentHash` fast path. `UnitDiff.ActualDoc` changes from `bson.D` to `element.Element` and `WithUnitCheck` callback signature changes accordingly.

**Tech Stack:** Go, mongo-driver (behind modelsdk), `modelsdk/codec`, `modelsdk/mpr`, `modelsdk/element`, `modelsdk/property`

**Design spec:** `docs/superpowers/specs/2026-06-26-bsoncompare-modelsdk-migration-design.md`

---

## File Structure

```
modelsdk/mpr/
  reader.go        — add UnitIdentity struct + ListUnitIdentities()
  types.go         — add UnitIdentity struct definition

modelsdk/codec/
  decoder.go       — add DecodeBytes([]byte) method

internal/bsoncompare/
  mprreader.go     — CHANGE: Raw from bson.Raw→[]byte, remove Doc/Decode()
  idmap.go         — REWRITE: BuildIDMap via ListUnitIdentities, delete collectIDsRaw
  compare.go       — CREATE: Element-based comparison engine
  diff.go          — CHANGE: ActualDoc from bson.D→element.Element
  diff_impl.go     — REWRITE: Compare uses codec.Decoder + compare.go
  assert.go        — CHANGE: WithUnitCheck callback takes element.Element
  align.go         — REMOVE: array alignment folded into compare.go
  options.go       — KEEP: shouldIgnore unchanged (operates on property names)
  normalize.go     — DELETE: replaced by compare.go
  normalize_test.go— DELETE
  mprreader_test.go— UPDATE: ContentHash tests
  diff_test.go     — UPDATE: remove self-compare test (still works via ContentHash)
  align_test.go    — REMOVE
  assert_test.go   — UPDATE: WithUnitCheck callback type
  options_test.go  — KEEP
  report_test.go   — KEEP
  idmap_test.go    — REWRITE: BuildIDMap test via ListUnitIdentities
```

---

### Task 1: Modelsdk — Add `ListUnitIdentities` + `DecodeBytes`

**Files:**
- Modify: `modelsdk/mpr/reader.go` — add method
- Modify: `modelsdk/mpr/types.go` — add struct
- Modify: `modelsdk/codec/decoder.go` — add method

**Interfaces:**
- Consumes: existing `mpr.Reader.ListRawUnits("")` returns `[]*types.RawUnitInfo` with `Contents []byte`
- Produces: `mpr.UnitIdentity` struct, `mpr.Reader.ListUnitIdentities()` returns `[]UnitIdentity`, `codec.Decoder.DecodeBytes([]byte)` returns `(element.Element, error)`

- [ ] **Step 1: Write tests for `ListUnitIdentities`**

```go
// modelsdk/mpr/reader_test.go (add)
func TestListUnitIdentities(t *testing.T) {
    r, err := OpenWithOptions("../../testdata/corpus-b/app.mpr", OpenOptions{ReadOnly: true})
    if err != nil {
        t.Fatal(err)
    }
    defer r.Close()
    idents, err := r.ListUnitIdentities()
    if err != nil {
        t.Fatal(err)
    }
    if len(idents) < 100 {
        t.Errorf("expected >=100 identities for corpus-b, got %d", len(idents))
    }
    for _, id := range idents {
        if id.ID == "" {
            t.Error("found identity with empty ID")
        }
    }
}
```

- [ ] **Step 2: Add `UnitIdentity` struct to `modelsdk/mpr/types.go`**

```go
// UnitIdentity holds the three metadata fields extracted from a unit's BSON header.
// Used by ListUnitIdentities to avoid full Element decode during ID-map building.
type UnitIdentity struct {
    ID   string // hex-encoded UUID from $ID
    Name string // Name field
    Type string // $Type field
}
```

- [ ] **Step 3: Write minimal `ListUnitIdentities`.**

The function reads `ListRawUnits("")`, uses `codec.decodeTypeName` and `codec.decodeID` internally (both exist in `codec/decoder.go`). Uses `bson.Raw(contents).LookupErr("Name")` to extract Name.

```go
// modelsdk/mpr/reader.go
func (r *Reader) ListUnitIdentities() ([]UnitIdentity, error) {
    infos, err := r.ListRawUnits("")
    if err != nil {
        return nil, err
    }
    out := make([]UnitIdentity, 0, len(infos))
    for _, info := range infos {
        if len(info.Contents) == 0 {
            continue
        }
        raw := bson.Raw(info.Contents)
        typeName := codec.DecodeTypeName(raw)
        if typeName == "" {
            continue
        }
        id := string(codec.DecodeID(raw))
        name, _ := raw.LookupErr("Name")
        nameStr, _ := name.StringValueOK()
        out = append(out, UnitIdentity{
            ID:   id,
            Name: nameStr,
            Type: typeName,
        })
    }
    return out, nil
}
```

Note: `DecodeTypeName` and `DecodeID` are currently unexported (`decodeTypeName`, `decodeID`). Export them:

```go
// modelsdk/codec/decoder.go
func DecodeTypeName(raw bson.Raw) string { return decodeTypeName(raw) }
func DecodeID(raw bson.Raw) element.ID   { return decodeID(raw) }
```

- [ ] **Step 4: Add `DecodeBytes` to codec.Decoder.**

```go
// modelsdk/codec/decoder.go
// DecodeBytes is like Decode but takes a []byte to avoid callers importing bson.Raw.
func (d *Decoder) DecodeBytes(raw []byte) (element.Element, error) {
    return d.Decode(bson.Raw(raw))
}
```

- [ ] **Step 5: Run tests.**

```bash
# mpr reader tests
go test -tags integration ./modelsdk/mpr/ -run TestListUnitIdentities -v

# codec decoder tests
go test ./modelsdk/codec/ -v
```

- [ ] **Step 6: Commit.**

```bash
git add modelsdk/mpr/reader.go modelsdk/mpr/types.go modelsdk/codec/decoder.go
git commit -m "feat: add ListUnitIdentities and DecodeBytes for bsoncompare migration"
```

---

### Task 2: bsoncompare — Rewrite reading + IDMap

**Files:**
- Modify: `internal/bsoncompare/mprreader.go` — `Raw` from `bson.Raw`→`[]byte`, remove `Doc`/`Decode()`
- Rewrite: `internal/bsoncompare/idmap.go` — `BuildIDMap` from `[]UnitIdentity`
- Delete: `internal/bsoncompare/normalize.go`, `internal/bsoncompare/normalize_test.go` (replaced in Task 3)
- Update: `internal/bsoncompare/mprreader_test.go`

**Interfaces:**
- Consumes: `mpr.Reader.ListUnitIdentities()` → `[]mpr.UnitIdentity`
- Produces: `UnitDoc{Raw []byte, ContentHash uint64}`, `BuildIDMap([]UnitDoc) IDMap`

- [ ] **Step 1: Update `mprreader.go` — change `Raw` type, remove `Doc`/`Decode()`.**

```go
type UnitDoc struct {
    QualifiedName string
    UnitType      string
    Raw           []byte    // raw BSON bytes (plain Go, not bson.Raw)
    ContentHash   uint64    // FNV-1a hash of raw BSON; fast diff skip
}

func ReadAllUnits(mprPath string) ([]UnitDoc, error) {
    if cached, ok := readAllCache.Load(mprPath); ok {
        r := cached.(*cachedResult)
        return r.units, r.err
    }

    r, err := mmpr.OpenWithOptions(mprPath, mmpr.OpenOptions{ReadOnly: true})
    if err != nil {
        readAllCache.Store(mprPath, &cachedResult{err: err})
        return nil, fmt.Errorf("bsoncompare: open %s: %w", mprPath, err)
    }
    defer r.Close()

    infos, err := r.ListRawUnits("")
    if err != nil {
        readAllCache.Store(mprPath, &cachedResult{err: err})
        return nil, fmt.Errorf("bsoncompare: list units: %w", err)
    }

    out := make([]UnitDoc, 0, len(infos))
    for _, info := range infos {
        h := fnv.New64a()
        h.Write(info.Contents)
        out = append(out, UnitDoc{
            QualifiedName: info.QualifiedName,
            UnitType:      info.Type,
            Raw:           info.Contents,   // []byte, not bson.Raw
            ContentHash:   h.Sum64(),
        })
    }
    readAllCache.Store(mprPath, &cachedResult{units: out})
    return out, nil
}
```

Delete the `import "go.mongodb.org/mongo-driver/v2/bson"` line from imports (no longer used here).

- [ ] **Step 2: Rewrite `idmap.go` — `BuildIDMap` via `ListUnitIdentities`.**

```go
package bsoncompare

import (
    "encoding/hex"
    "fmt"
    "strings"

    mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

type IDMap map[string]string

func BuildIDMap(units []UnitDoc) IDMap {
    m := make(IDMap, len(units)*2)
    for _, u := range units {
        id := hex.EncodeToString(u.ContentHash) // placeholder — real impl below
        _ = id
    }
    return m
}
```

Wait, that's wrong. `BuildIDMap` needs to extract IDs from the raw bytes. But we no longer have `collectIDsRaw` and we shouldn't iterate bson.

Actually, `BuildIDMap` currently takes `[]UnitDoc` and iterates them. But the new IDMap should be built from `[]UnitIdentity` obtained via `ListUnitIdentities()`.

The flow in `Compare`:
1. Open MPR
2. Call `r.ListUnitIdentities()` for IDMap
3. Call `ReadAllUnits()` for ContentHash + raw bytes
4. When comparing changed units, decode raw via `codec.Decoder.DecodeBytes()`

But we can't call `ListUnitIdentities()` from `BuildIDMap` because `BuildIDMap` takes `[]UnitDoc`. The IDMap needs to be built separately.

Better approach: add a new function `BuildIDMapFromReader(r *mmpr.Reader) IDMap`:

```go
func BuildIDMapFromReader(r *mmpr.Reader) (IDMap, error) {
    idents, err := r.ListUnitIdentities()
    if err != nil {
        return nil, err
    }
    m := make(IDMap, len(idents))
    for _, id := range idents {
        m[id.ID] = makeLabel(id.Type, id.Name, "")
    }
    return m, nil
}
```

And update `Compare` to call this instead of `BuildIDMap(units)`.

```go
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
```

Keep `MergeInto` and `Lookup` as they are.

Remove `BuildIDMap([]UnitDoc)`, `collectIDsRaw()`, the `import "go.mongodb.org/mongo-driver/v2/bson"`.

- [ ] **Step 3: Update tests — `mprreader_test.go`.**

Remove `TestReadAllUnits_ContentHashPopulated` (no longer applicable — we store `[]byte`, not `bson.Raw`).
Keep `TestReadAllUnits_ContentHashStable` (ContentHash still works).

```go
func TestReadAllUnits_ContentHashStable(t *testing.T) {
    t.Parallel()
    units1, err := ReadAllUnits("../../testdata/corpus-b/app.mpr")
    if err != nil {
        t.Fatal(err)
    }
    units2, err := ReadAllUnits("../../testdata/corpus-b/app.mpr")
    if err != nil {
        t.Fatal(err)
    }
    if len(units1) != len(units2) {
        t.Fatalf("length mismatch: %d vs %d", len(units1), len(units2))
    }
    for i := range units1 {
        if units1[i].ContentHash != units2[i].ContentHash {
            t.Errorf("unit %s: ContentHash unstable (%d vs %d)",
                units1[i].QualifiedName, units1[i].ContentHash, units2[i].ContentHash)
        }
    }
}
```

- [ ] **Step 4: Update `idmap_test.go`.**

Rewrite to test `BuildIDMapFromReader`:

```go
func TestBuildIDMapFromReader_CorpusB(t *testing.T) {
    t.Parallel()
    r, err := mmpr.OpenWithOptions("../../testdata/corpus-b/app.mpr", mmpr.OpenOptions{ReadOnly: true})
    if err != nil {
        t.Fatal(err)
    }
    defer r.Close()
    m, err := bsoncompare.BuildIDMapFromReader(r)
    if err != nil {
        t.Fatal(err)
    }
    if len(m) < 10000 {
        t.Errorf("expected >=10000 IDMap entries for corpus-b, got %d", len(m))
    }
    for k, v := range m {
        if v == "" {
            t.Errorf("key %s has empty label", k)
        }
    }
}
```

- [ ] **Step 5: Run tests to verify compilation and correctness.**

```bash
go build ./internal/bsoncompare/
go test -tags integration -count=1 -timeout 30m ./internal/bsoncompare/ -v 2>&1 | grep -E 'PASS|FAIL|ok'
```

Expected: tests that reference deleted files/functions will fail (normalize_test.go, align_test.go not yet removed).

`normalize.go` will be deleted in Task 3 when `compare.go` replaces it.

- [ ] **Step 6: Commit.**

```bash
git add internal/bsoncompare/mprreader.go internal/bsoncompare/idmap.go \
      internal/bsoncompare/mprreader_test.go internal/bsoncompare/idmap_test.go
git commit -m "refactor(bsoncompare): store []byte, build IDMap via modelsdk"
```

---

### Task 3: bsoncompare — Element comparison engine

**Files:**
- Create: `internal/bsoncompare/compare.go`
- Delete: `internal/bsoncompare/align.go`, `internal/bsoncompare/align_test.go`

**Interfaces:**
- Consumes: `element.Element`, `element.Property`, `codec.Decoder.DecodeBytes([]byte)`
- Produces: `compareElements(path, golden, actual element.Element, idMap, opts) []FieldDiff`

- [ ] **Step 1: Write comparison tests.**

Create `internal/bsoncompare/compare_test.go` (external test package).

Tests focus on the exported comparison functions: `compareElements`, `compareBareBase`, `compareStringSlice`, `compareBinary`. Property-level testing is covered by integration tests via `Compare()` on real corpus data.

```go
package bsoncompare_test

import (
    "testing"

    "github.com/mendixlabs/mxcli/internal/bsoncompare"
)

func TestCompareElements_BothNil(t *testing.T) {
    t.Parallel()
    diffs := bsoncompare.CompareElements("", nil, nil, nil, bsoncompare.DefaultOptions())
    if len(diffs) != 0 {
        t.Errorf("expected 0 diffs for nil/nil, got %d", len(diffs))
    }
}

func TestCompareElements_TypeMismatch(t *testing.T) {
    t.Parallel()
    g := newBase("Microflows$ActionActivity", "")
    a := newBase("Microflows$LoopAction", "")

    diffs := bsoncompare.CompareElements("", g, a,
        bsoncompare.IDMap{}, bsoncompare.DefaultOptions())
    if len(diffs) != 1 {
        t.Fatalf("expected 1 type diff, got %d", len(diffs))
    }
    if diffs[0].Path != ".$Type" {
        t.Errorf("path: got %q, want .$Type", diffs[0].Path)
    }
}

func TestCompareBareBase_NameDiff(t *testing.T) {
    t.Parallel()
    g := newBase("Unknown$Widget", "WidgetA")
    a := newBase("Unknown$Widget", "WidgetB")

    diffs := bsoncompare.CompareBareBase("", g, a)
    if len(diffs) != 1 {
        t.Fatalf("expected 1 name diff, got %d", len(diffs))
    }
}

// newBase creates a bare *element.Base with the given type name and name.
func newBase(typeName, name string) *element.Base {
    // Use the constructor from bsoncompare or element package
    // ...
}
```

Note: The `newBase` helper and `CompareElements`/`CompareBareBase` export status
will be determined during implementation. If kept internal, tests live in
package `bsoncompare` (not `_test`).

- [ ] **Step 2: Implement `compare.go`.**

```go
package bsoncompare

import (
    "fmt"
    "sort"
    "strings"

    "go.mongodb.org/mongo-driver/v2/bson"

    "github.com/mendixlabs/mxcli/modelsdk/element"
    "github.com/mendixlabs/mxcli/modelsdk/property"
)

// compareElements compares two element.Element trees and returns field diffs.
func compareElements(path string, golden, actual element.Element, idMap IDMap, opts Options) []FieldDiff {
    if golden == nil && actual == nil { return nil }
    if golden == nil {
        return []FieldDiff{{Path: path, Kind: DiffAdded, Actual: describe(actual)}}
    }
    if actual == nil {
        return []FieldDiff{{Path: path, Kind: DiffRemoved, Golden: describe(golden)}}
    }
    // Bare Base (unknown type) — limited comparison
    if isBareBase(golden) || isBareBase(actual) {
        return compareBareBase(path, golden, actual)
    }
    if golden.TypeName() != actual.TypeName() {
        return []FieldDiff{{Path: path + ".$Type",
            Golden: golden.TypeName(), Actual: actual.TypeName(), Kind: DiffChanged}}
    }
    return compareProperties(path, golden.Properties(), actual.Properties(), idMap, opts)
}

func isBareBase(elem element.Element) bool {
    _, ok := elem.(*element.Base)
    return ok
}

func compareBareBase(path string, golden, actual element.Element) []FieldDiff {
    var diffs []FieldDiff
    if golden.TypeName() != actual.TypeName() {
        diffs = append(diffs, FieldDiff{Path: path + ".$Type",
            Golden: golden.TypeName(), Actual: actual.TypeName(), Kind: DiffChanged})
    }
    gn := golden.NameValue()
    an := actual.NameValue()
    if gn != an {
        diffs = append(diffs, FieldDiff{Path: path + ".Name",
            Golden: gn, Actual: an, Kind: DiffChanged})
    }
    return diffs
}

func compareProperties(path string, gProps, aProps []element.Property, idMap IDMap, opts Options) []FieldDiff {
    gByKey := make(map[string]element.Property, len(gProps))
    for _, p := range gProps {
        gByKey[p.Name()] = p
    }
    aByKey := make(map[string]element.Property, len(aProps))
    for _, p := range aProps {
        aByKey[p.Name()] = p
    }

    keys := make(map[string]bool)
    for k := range gByKey { keys[k] = true }
    for k := range aByKey { keys[k] = true }
    sorted := make([]string, 0, len(keys))
    for k := range keys { sorted = append(sorted, k) }
    sort.Strings(sorted)

    var result []FieldDiff
    for _, k := range sorted {
        if shouldIgnore(k, opts) {
            continue
        }
        gp, gok := gByKey[k]
        ap, aok := aByKey[k]
        fp := path + "." + k
        switch {
        case gok && !aok:
            result = append(result, FieldDiff{Path: fp, Kind: DiffRemoved})
        case !gok && aok:
            result = append(result, FieldDiff{Path: fp, Kind: DiffAdded})
        default:
            result = append(result, compareProperty(fp, gp, ap, idMap, opts)...)
        }
    }
    return result
}

func compareProperty(path string, gp, ap element.Property, idMap IDMap, opts Options) []FieldDiff {
    // ChildProperty (Part)
    if gc, ok := gp.(element.ChildProperty); ok {
        ac := ap.(element.ChildProperty)
        return compareElements(path, gc.ChildElement(), ac.ChildElement(), idMap, opts)
    }

    // ChildListProperty (PartList)
    if gl, ok := gp.(element.ChildListProperty); ok {
        al := ap.(element.ChildListProperty)
        return compareChildList(path, gl.ChildElements(), al.ChildElements(), idMap, opts)
    }

    // WritableProperty — get the BSON values and compare
    gw := gp.(element.WritableProperty)
    aw := ap.(element.WritableProperty)
    return compareValues(path, gw.BSONValue(), aw.BSONValue(), idMap, opts)
}

func compareValues(path string, g, a any, idMap IDMap, opts Options) []FieldDiff {
    switch gv := g.(type) {
    case string:
        av, _ := a.(string)
        if gv != av { return []FieldDiff{{Path: path, Golden: gv, Actual: av, Kind: DiffChanged}} }
    case int32:
        av, _ := a.(int32)
        if gv != av { return []FieldDiff{{Path: path, Golden: fmt.Sprintf("%d", gv), Actual: fmt.Sprintf("%d", av), Kind: DiffChanged}} }
    case bool:
        av, _ := a.(bool)
        if gv != av { return []FieldDiff{{Path: path, Golden: fmt.Sprintf("%v", gv), Actual: fmt.Sprintf("%v", av), Kind: DiffChanged}} }
    case float64:
        av, _ := a.(float64)
        if gv != av { return []FieldDiff{{Path: path, Golden: fmt.Sprintf("%v", gv), Actual: fmt.Sprintf("%v", av), Kind: DiffChanged}} }
    case bson.Binary:
        return compareBinary(path, gv, a, idMap)
    case element.ID:
        av, _ := a.(element.ID)
        if gv != av { return []FieldDiff{{Path: path, Golden: string(gv), Actual: string(av), Kind: DiffChanged}} }
    case []string:
        av, _ := a.([]string)
        return compareStringSlice(path, gv, av)
    case []any:
        av, _ := a.([]any)
        return compareAnySlice(path, gv, av, idMap, opts)
    }
    return nil
}

func compareBinary(path string, gb bson.Binary, a any, idMap IDMap) []FieldDiff {
    ab, ok := a.(bson.Binary)
    if !ok { return []FieldDiff{{Path: path, Kind: DiffChanged}} }
    if len(gb.Data) == 16 && len(ab.Data) == 16 {
        gl := idMap.Lookup(gb.Data)
        al := idMap.Lookup(ab.Data)
        if gl != al {
            return []FieldDiff{{Path: path, Golden: gl, Actual: al, Kind: DiffChanged}}
        }
        return nil
    }
    if len(gb.Data) != len(ab.Data) {
        return []FieldDiff{{Path: path, Golden: fmt.Sprintf("<binary:%d>", len(gb.Data)),
            Actual: fmt.Sprintf("<binary:%d>", len(ab.Data)), Kind: DiffChanged}}
    }
    return nil
}

func compareStringSlice(path string, g, a []string) []FieldDiff {
    gs := make(map[string]int)
    for _, s := range g { gs[s]++ }
    as := make(map[string]int)
    for _, s := range a { as[s]++ }

    var diffs []FieldDiff
    for s := range gs {
        if as[s] != gs[s] { diffs = append(diffs, FieldDiff{Path: path, Golden: s, Kind: DiffRemoved}) }
    }
    for s := range as {
        if gs[s] != as[s] { diffs = append(diffs, FieldDiff{Path: path, Actual: s, Kind: DiffAdded}) }
    }
    return diffs
}

func compareAnySlice(path string, g, a []any, idMap IDMap, opts Options) []FieldDiff {
    // ByNameRefList / ByIdRefList — first element may be int32 version marker
    gs := stripVersion(g)
    as := stripVersion(a)
    // Compare as strings
    gStrs := make([]string, len(gs))
    for i, v := range gs { gStrs[i] = fmt.Sprintf("%v", v) }
    aStrs := make([]string, len(as))
    for i, v := range as { aStrs[i] = fmt.Sprintf("%v", v) }
    return compareStringSlice(path, gStrs, aStrs)
}

func stripVersion(arr []any) []any {
    if len(arr) > 0 {
        if _, ok := arr[0].(int32); ok {
            return arr[1:]
        }
    }
    return arr
}

// compareChildList compares two PartList child element slices.
func compareChildList(path string, golden, actual []element.Element, idMap IDMap, opts Options) []FieldDiff {
    switch {
    case allHaveName(golden) || allHaveName(actual):
        return compareByName(path, golden, actual, idMap, opts)
    default:
        return compareByPosition(path, golden, actual, idMap, opts)
    }
}

func allHaveName(elems []element.Element) bool {
    for _, e := range elems {
        if e == nil || e.NameValue() == "" { return false }
    }
    return len(elems) > 0
}

func compareByName(path string, golden, actual []element.Element, idMap IDMap, opts Options) []FieldDiff {
    gByName := make(map[string]element.Element, len(golden))
    for _, e := range golden {
        if n := e.NameValue(); n != "" { gByName[n] = e }
    }
    aByName := make(map[string]element.Element, len(actual))
    for _, e := range actual {
        if n := e.NameValue(); n != "" { aByName[n] = e }
    }

    names := make(map[string]bool)
    for n := range gByName { names[n] = true }
    for n := range aByName { names[n] = true }
    sorted := make([]string, 0, len(names))
    for n := range names { sorted = append(sorted, n) }
    sort.Strings(sorted)

    var diffs []FieldDiff
    for _, name := range sorted {
        ge, gok := gByName[name]
        ae, aok := aByName[name]
        elemPath := fmt.Sprintf("%s[%s]", path, name)
        switch {
        case gok && !aok:
            diffs = append(diffs, FieldDiff{Path: elemPath, Golden: name, Kind: DiffRemoved})
        case !gok && aok:
            diffs = append(diffs, FieldDiff{Path: elemPath, Actual: name, Kind: DiffAdded})
        default:
            diffs = append(diffs, compareElements(elemPath, ge, ae, idMap, opts)...)
        }
    }
    return diffs
}

func compareByPosition(path string, golden, actual []element.Element, idMap IDMap, opts Options) []FieldDiff {
    if len(golden) != len(actual) {
        return []FieldDiff{{Path: path + ".length",
            Golden: fmt.Sprintf("%d", len(golden)), Actual: fmt.Sprintf("%d", len(actual)), Kind: DiffChanged}}
    }
    return nil
}

func describe(elem element.Element) string {
    if elem == nil { return "" }
    n := elem.NameValue()
    if n != "" { return n }
    return elem.TypeName()
}
```

- [ ] **Step 3: Run tests.**

```bash
go build ./internal/bsoncompare/
go test -tags integration -count=1 -timeout 30m ./internal/bsoncompare/ -run TestCompare 2>&1
```

- [ ] **Step 4: Delete old alignment files.**

```bash
git rm internal/bsoncompare/align.go internal/bsoncompare/align_test.go
```

- [ ] **Step 5: Commit.**

```bash
git add internal/bsoncompare/compare.go internal/bsoncompare/compare_test.go
git rm internal/bsoncompare/align.go internal/bsoncompare/align_test.go
git rm internal/bsoncompare/normalize.go internal/bsoncompare/normalize_test.go
git commit -m "feat(bsoncompare): add Element-based comparison engine"
```

---

### Task 4: Wire pipeline + update structs

**Files:**
- Modify: `internal/bsoncompare/diff_impl.go` — rewrite `Compare`
- Modify: `internal/bsoncompare/diff.go` — `ActualDoc` from `bson.D`→`element.Element`
- Modify: `internal/bsoncompare/assert.go` — `WithUnitCheck` callback
- Update: `internal/bsoncompare/assert_test.go`, `internal/bsoncompare/diff_test.go`

**Interfaces:**
- Consumes: `compareElements`, `codec.NewDecoder(codec.DefaultRegistry).DecodeBytes([]byte)`
- Produces: `Compare(aPath, bPath, opts) ([]UnitDiff, error)`

- [ ] **Step 1: Update `diff.go` — change `ActualDoc` type.**

```go
type UnitDiff struct {
    QualifiedName string
    UnitType      string
    Kind          DiffKind
    Fields        []FieldDiff
    ActualDoc     element.Element // was bson.D
}
```

Remove `import "go.mongodb.org/mongo-driver/v2/bson"`.

- [ ] **Step 2: Rewrite `diff_impl.go` `Compare()`.**

```go
package bsoncompare

import (
    "fmt"
    "sort"

    "github.com/mendixlabs/mxcli/modelsdk/codec"
    mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

func Compare(aPath, bPath string, opts Options) ([]UnitDiff, error) {
    if aPath == bPath {
        return nil, nil
    }

    aReader, err := mmpr.OpenWithOptions(aPath, mmpr.OpenOptions{ReadOnly: true})
    if err != nil {
        return nil, fmt.Errorf("bsoncompare: open A (%s): %w", aPath, err)
    }
    defer aReader.Close()

    bReader, err := mmpr.OpenWithOptions(bPath, mmpr.OpenOptions{ReadOnly: true})
    if err != nil {
        return nil, fmt.Errorf("bsoncompare: open B (%s): %w", bPath, err)
    }
    defer bReader.Close()

    // IDMap from both sides
    idMap, err := BuildIDMapFromReader(bReader)
    if err != nil {
        return nil, fmt.Errorf("bsoncompare: IDMap B: %w", err)
    }
    aIDMap, err := BuildIDMapFromReader(aReader)
    if err != nil {
        return nil, fmt.Errorf("bsoncompare: IDMap A: %w", err)
    }
    MergeInto(idMap, aIDMap)

    aUnits, err := ReadAllUnits(aPath)
    if err != nil {
        return nil, fmt.Errorf("bsoncompare: read A: %w", err)
    }
    bUnits, err := ReadAllUnits(bPath)
    if err != nil {
        return nil, fmt.Errorf("bsoncompare: read B: %w", err)
    }

    aIndex := indexUnits(aUnits)
    bIndex := indexUnits(bUnits)

    allNames := make(map[string]bool)
    for k := range aIndex { allNames[k] = true }
    for k := range bIndex { allNames[k] = true }
    names := make([]string, 0, len(allNames))
    for k := range allNames { names = append(names, k) }
    sort.Strings(names)

    dec := codec.NewDecoder(codec.DefaultRegistry)
    var result []UnitDiff

    for _, name := range names {
        au, aok := aIndex[name]
        bu, bok := bIndex[name]
        switch {
        case aok && !bok:
            result = append(result, UnitDiff{QualifiedName: name, UnitType: au.UnitType, Kind: DiffRemoved})
        case !aok && bok:
            actual, err := dec.DecodeBytes(bu.Raw)
            if err != nil {
                continue
            }
            result = append(result, UnitDiff{
                QualifiedName: name,
                UnitType:      bu.UnitType,
                Kind:          DiffAdded,
                ActualDoc:     actual,
            })
        default:
            if au.ContentHash == bu.ContentHash {
                continue
            }
            actual, aErr := dec.DecodeBytes(bu.Raw)
            golden, gErr := dec.DecodeBytes(au.Raw)
            if aErr != nil || gErr != nil {
                continue
            }
            fields := compareElements("", golden, actual, idMap, opts)
            if len(fields) > 0 {
                result = append(result, UnitDiff{
                    QualifiedName: name,
                    UnitType:      au.UnitType,
                    Kind:          DiffChanged,
                    Fields:        fields,
                    ActualDoc:     actual,
                })
            }
        }
    }
    return result, nil
}

func indexUnits(units []UnitDoc) map[string]*UnitDoc {
    m := make(map[string]*UnitDoc, len(units))
    for i := range units {
        m[units[i].QualifiedName] = &units[i]
    }
    return m
}
```

- [ ] **Step 3: Update `assert.go` — `WithUnitCheck` callback.**

```go
// WithUnitCheck returns a Matcher that runs check against the actual Element.
func WithUnitCheck(qualifiedName string, check func(element.Element) error) Matcher {
    return withUnitCheck{name: qualifiedName, check: check}
}

type withUnitCheck struct {
    name  string
    check func(element.Element) error
}

func (w withUnitCheck) Match(diffs []UnitDiff, claimed map[string]bool) error {
    for _, d := range diffs {
        if d.QualifiedName != w.name {
            continue
        }
        if d.ActualDoc == nil {
            return fmt.Errorf("WithUnitCheck %q: unit was removed (no actual doc)", w.name)
        }
        if err := w.check(d.ActualDoc); err != nil {
            return fmt.Errorf("WithUnitCheck %q: %w", w.name, err)
        }
        claimed[w.name] = true
        return nil
    }
    return fmt.Errorf("WithUnitCheck %q: unit not found in diffs", w.name)
}
```

Remove `import "go.mongodb.org/mongo-driver/v2/bson"` from `assert.go`.

- [ ] **Step 4: Update test files.**

Update `assert_test.go` — `WithUnitCheck` now takes `element.Element` instead of `bson.D`:

```go
import (
    "github.com/mendixlabs/mxcli/modelsdk/element"
)

// TestWithUnitCheck_Matches (simplified — no need for real bson.D in test)
func TestWithUnitCheck_Matches(t *testing.T) {
    diffs := []UnitDiff{
        {
            QualifiedName: "MyFirstModule.ACT_Test",
            Kind:          DiffChanged,
            ActualDoc:     &element.Base{},  // stub
        },
    }
    claimed := map[string]bool{}
    called := false
    matcher := WithUnitCheck("MyFirstModule.ACT_Test", func(e element.Element) error {
        called = true
        return nil
    })
    if err := matcher.Match(diffs, claimed); err != nil {
        t.Errorf("WithUnitCheck should match: %v", err)
    }
    if !called {
        t.Error("check function was not called")
    }
}
```

Remove `TestAssertEqual_SelfComparePasses` (the self-compare early return test — covered by `TestCompare_NoChange`).

- [ ] **Step 5: Run all tests.**

```bash
go test -tags integration -count=1 -timeout 30m ./internal/bsoncompare/ -v 2>&1 | tail -40
```

- [ ] **Step 6: Commit.**

```bash
git add internal/bsoncompare/diff.go internal/bsoncompare/diff_impl.go \
      internal/bsoncompare/assert.go internal/bsoncompare/assert_test.go \
      internal/bsoncompare/diff_test.go
git commit -m "feat(bsoncompare): wire Element comparison into Compare pipeline"
```

---

### Task 5: Clean up and validate

**Files:**
- Delete: `internal/bsoncompare/options.go` (if `shouldIgnore` moved) — actually keep
- Remove unused imports across all files
- Verify no `bson.Unmarshal`, `bson.D`, `bson.A`, `raw.Elements()` remain in bsoncompare

- [ ] **Step 1: Grep for remaining direct bson usage.**

```bash
grep -n 'bson\.' internal/bsoncompare/*.go | grep -v '_test.go' | grep -v 'import'
```

Allow only:
- `import "go.mongodb.org/mongo-driver/v2/bson"` (for type assertions in compare.go)
- `bson.Binary` type assertions
- `bson.A` type assertions

Disallow:
- `bson.Unmarshal`
- `bson.D`, `bson.E`
- `bson.Raw` construction
- `.Elements()`, `.Document()`, `.Array()` calls on bson types

- [ ] **Step 2: Validate downstream compilation.**

```bash
go vet ./mdl/executor/
go build ./cmd/mxcli/
```

Fix any downstream compilation errors (especially `WithUnitCheck` callers in `mdl/executor/roundtrip_*_test.go`).

- [ ] **Step 3: Full test run.**

```bash
go test -tags integration -count=1 -timeout 30m ./internal/bsoncompare/ 2>&1
```

Expected: all tests pass.

- [ ] **Step 4: Profile and verify improvement.**

```bash
./scripts/profile-test.sh ./internal/bsoncompare/ 2>&1 | grep -E 'Wall time|Tests found|Coverage|bson.dDecodeValue'
```

Expected: `bson.dDecodeValue` no longer in top memory consumers (was 44%).
Wall time should be comparable to or better than current (2.5s).

- [ ] **Step 5: Race check.**

```bash
go test -tags integration -count=1 -timeout 30m -race ./internal/bsoncompare/ 2>&1
```

Expected: `ok` with no race warnings.

- [ ] **Step 6: Commit.**

```bash
git add -A
git commit -m "chore: clean up obsolete files, finalize bsoncompare modelsdk migration"
```
